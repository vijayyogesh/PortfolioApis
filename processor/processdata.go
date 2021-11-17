package processor

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/vijayyogesh/PortfolioApis/data"
)

var dailyPriceCache map[string][]data.CompaniesPriceData = make(map[string][]data.CompaniesPriceData)

var companiesCache []data.Company

/* Master method that does the following
1) Download data file based on TS
2) Load into DB
*/
func FetchAndUpdatePrices(db *sql.DB) {
	//Fetch Unique Company Details
	companiesData := FetchCompanies(db)

	//Download data file in parallel
	DownloadDataAsync(companiesData)

	//Read Data From File & Write into DB asynchronously
	LoadPriceData(db)
}

/* Fetch Unique Company Details */
func FetchCompanies(db *sql.DB) []data.Company {
	var companies []data.Company
	if companiesCache != nil {
		fmt.Println("Fetching Companies Master List From Cache")
		return companiesCache
	} else {
		fmt.Println("Fetching Companies Master List From DB")
		companies = data.FetchCompaniesDB(db)
		companiesCache = companies
		return companies
	}
}

/* Call DownloadDataFile from go routine */
func DownloadDataAsync(companiesData []data.Company) {
	var wg sync.WaitGroup
	for _, company := range companiesData {
		wg.Add(1)
		go func(companyId string, fromTime time.Time) {
			defer wg.Done()
			if fromTime.IsZero() {
				fromTime = time.Date(1996, 1, 1, 0, 0, 0, 0, time.UTC)
			}
			//fmt.Println("fromTime -- ", fromTime)
			DownloadDataFile(companyId, fromTime)
		}(company.CompanyId, company.LoadDate)
	}
	wg.Wait()
}

/* Download data file from online */
func DownloadDataFile(companyId string, fromTime time.Time) error {
	//fmt.Println("Loading file for company " + companyId)
	filePath := "C:\\Users\\Ajay\\Downloads\\" + companyId + ".NS.csv"
	url := "https://query1.finance.yahoo.com/v7/finance/download/" + companyId +
		".NS?period1=%s&period2=%s&interval=1d&events=history&includeAdjustedClose=true"

	out, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer out.Close()

	/* Form start and end time */
	//startTime := strconv.Itoa(int(time.Date(1996, 1, 1, 0, 0, 0, 0, time.UTC).Unix()))
	startTime := strconv.Itoa(int(fromTime.Unix()))
	endTime := strconv.Itoa(int(time.Now().Unix()))
	//fmt.Println("Start Time " + startTime + " End Time " + endTime)
	url = fmt.Sprintf(url, startTime, endTime)
	//fmt.Println("filePath " + filePath)
	//fmt.Println("url " + url)

	/* Get the data from Yahoo Finance */
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	/* Check server response */
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	/* Writer the body to file */
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	fmt.Println("Finished loading file for company " + companyId)
	return nil
}

/* Read Data From File & Write into DB asynchronously */
func LoadPriceData(db *sql.DB) {
	companies := FetchCompanies(db)
	//fmt.Println(companies)
	var totRecordsCount int64
	var wg sync.WaitGroup

	for _, company := range companies {
		wg.Add(1)
		filePath := "C:\\Users\\Ajay\\Downloads\\" + company.CompanyId + ".NS.csv"
		fmt.Println(filePath)

		go func(companyid string) {
			defer wg.Done()
			var err error
			companiesdata, err, recordsCount := ReadDailyPriceCsv(filePath, companyid)
			if err != nil {
				panic(err)
			}
			fmt.Println(companyid, " - ", recordsCount)
			atomic.AddInt64(&totRecordsCount, int64(recordsCount))
			data.LoadPriceDataDB(companiesdata, db)
			data.UpdateLoadDate(db, companyid, time.Now())
		}(company.CompanyId)
	}
	wg.Wait()
	fmt.Println(totRecordsCount)
}

func ReadDailyPriceCsv(filePath string, companyid string) ([]data.CompaniesPriceData, error, int) {
	var companiesdata []data.CompaniesPriceData

	/* Open file */
	file, err := os.Open(filePath)
	/* Return if error */
	if err != nil {
		fmt.Println(err.Error(), "Error while opening file ")
		return companiesdata, fmt.Errorf("error while opening file %s ", filePath), 0
	}
	fmt.Println(file.Name())

	/* Read csv */
	csvReader := csv.NewReader(file)
	records, err := csvReader.ReadAll()
	/* Return if error */
	if err != nil {
		fmt.Println(err.Error(), "Error while reading csv ")
		return companiesdata, fmt.Errorf("error while reading csv %s ", filePath), 0
	}
	/* Close resources */
	file.Close()

	/* Process each record */
	for k, v := range records {
		if k != 0 {

			openval, dataError := strconv.ParseFloat(v[len(v)-6], 64)
			processDataErr(dataError, k)

			highval, dataError := strconv.ParseFloat(v[len(v)-5], 64)
			processDataErr(dataError, k)

			lowval, dataError := strconv.ParseFloat(v[len(v)-4], 64)
			processDataErr(dataError, k)

			closeval, dataError := strconv.ParseFloat(v[len(v)-3], 64)
			processDataErr(dataError, k)

			dateval, dataError := time.Parse("2006-01-02", v[len(v)-7])
			processDataErr(dataError, k)

			companiesdata = append(companiesdata, data.CompaniesPriceData{CompanyId: companyid, DateVal: dateval, OpenVal: openval, HighVal: highval, LowVal: lowval, CloseVal: closeval})
		}
	}

	fmt.Println("Name - " + companyid)
	//fmt.Println(len(companiesdata))
	return companiesdata, nil, len(companiesdata)

}

/* Non critical record error which can be logged and ignored */
func processDataErr(dataError error, k int) {
	if dataError != nil {
		fmt.Println(dataError.Error(), "Error while processing data", k)
	}
}

/* Fetch Price Data initially from DB and use cache for subsequent requests */
func FetchCompaniesPrice(companyid string, db *sql.DB) {
	var dailyPriceRecords []data.CompaniesPriceData
	if dailyPriceCache[companyid] != nil {
		fmt.Println("From Cache")
		dailyPriceRecords = dailyPriceCache[companyid]
	} else {
		fmt.Println("From DB")
		dailyPriceRecords = data.FetchCompaniesPriceDataDB(companyid, db)
		dailyPriceCache[companyid] = dailyPriceRecords
	}
	fmt.Println(len(dailyPriceRecords))
}

func FetchAndUpdateCompaniesMasterList(db *sql.DB) {

	//Download data file in parallel
	DownloadCompaniesMaster()

	//Read Data From File & Write into DB asynchronously
	LoadCompaniesMaster(db)
}

/* Download data file from online */
func DownloadCompaniesMaster() error {
	fmt.Println("Loading Companies Master List ")
	filePath := "C:\\Users\\Ajay\\Downloads\\" + "TOP200" + ".csv"
	url := "https://www1.nseindia.com/content/indices/ind_nifty200list.csv"

	out, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer out.Close()

	/* Get the data from NSE INDIA */
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	/* Check server response */
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	/* Writer the body to file */
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	fmt.Println("Finished downloading Companies Master List File")
	return nil
}

/* Read Companies Master Data From File & Write into DB  */
func LoadCompaniesMaster(db *sql.DB) {
	companiesMasterList, err := ReadCompaniesMasterCsv("C:\\Users\\Ajay\\Downloads\\" + "TOP200" + ".csv")
	if err == nil {
		LoadCompaniesMasterList(db, companiesMasterList)
	}
}

/* Write Companies Master List into DB */
func LoadCompaniesMasterList(db *sql.DB, companiesMasterList []data.Company) {
	data.LoadCompaniesMasterListDB(companiesMasterList, db)
}

func ReadCompaniesMasterCsv(filePath string) ([]data.Company, error) {
	var companiesMasterList []data.Company

	/* Open file */
	file, err := os.Open(filePath)
	/* Return if error opening file */
	if err != nil {
		fmt.Println(err.Error(), "Error while opening file ")
		return companiesMasterList, fmt.Errorf("error while opening file %s ", filePath)
	}
	fmt.Println(file.Name())

	/* Read csv */
	csvReader := csv.NewReader(file)
	records, err := csvReader.ReadAll()
	/* Return if error */
	if err != nil {
		fmt.Println(err.Error(), "Error while reading csv ")
		return companiesMasterList, fmt.Errorf("error while reading csv %s ", filePath)
	}
	/* Close resources */
	file.Close()

	/* Process each record */
	for k, v := range records {
		/* Ignore first record */
		if k != 0 {
			companyid := v[len(v)-3]
			companyname := v[len(v)-5]
			companiesMasterList = append(companiesMasterList, data.Company{CompanyId: companyid, CompanyName: companyname})
		}
	}

	fmt.Println(len(companiesMasterList))
	fmt.Println((companiesMasterList))
	return companiesMasterList, nil

}
