package processor

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
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

var dailyPriceCacheLatest map[string]data.CompaniesPriceData = make(map[string]data.CompaniesPriceData)

var companiesCache []data.Company

var usersCache map[string]data.User = make(map[string]data.User)

/* Master method that does the following
1) Download data file based on TS
2) Load into DB
*/
func FetchAndUpdatePrices(db *sql.DB) string {
	//Fetch Unique Company Details
	companiesData := FetchCompanies(db)

	//Download data file in parallel
	DownloadDataAsync(companiesData)

	//Read Data From File & Write into DB asynchronously
	LoadPriceData(db)

	return "Prices updated successfully"
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
	/*var wg sync.WaitGroup
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
	*/

	for _, company := range companiesData {
		companyId := company.CompanyId
		fromTime := company.LoadDate
		if fromTime.IsZero() {
			fromTime = time.Date(1996, 1, 1, 0, 0, 0, 0, time.UTC)
		}
		time.Sleep(2 * time.Second)
		DownloadDataFile(companyId, fromTime)
	}
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
	fmt.Println("url " + url)

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

/* Fetch All Price Data initially from DB and use cache for subsequent requests */
func FetchAllCompaniesCompletePrice(companyid string, db *sql.DB) {
	var dailyPriceRecords []data.CompaniesPriceData
	if dailyPriceCache[companyid] != nil {
		fmt.Println("From Cache")
		dailyPriceRecords = dailyPriceCache[companyid]
	} else {
		fmt.Println("From DB")
		dailyPriceRecords = data.FetchCompaniesCompletePriceDataDB(companyid, db)
		dailyPriceCache[companyid] = dailyPriceRecords
	}
	fmt.Println(len(dailyPriceRecords))
}

/* Fetch Latest Price Data from DB and use cache for subsequent requests */
func FetchLatestCompaniesCompletePrice(companyid string, db *sql.DB) {
	var dailyPriceRecordsLatest data.CompaniesPriceData
	if dailyPriceCache[companyid] != nil {
		fmt.Println("From Cache")
		dailyPriceRecordsLatest = dailyPriceCacheLatest[companyid]
	} else {
		fmt.Println("From DB")
		dailyPriceRecordsLatest = data.FetchCompaniesLatestPriceDataDB(companyid, db)
		dailyPriceCacheLatest[companyid] = dailyPriceRecordsLatest
	}
	fmt.Println(dailyPriceRecordsLatest)
}

func FetchAndUpdateCompaniesMasterList(db *sql.DB) string {
	DownloadCompaniesMaster()

	LoadCompaniesMaster(db)

	return "Master companies list loaded successfully"
}

/* Download data file from online */
func DownloadCompaniesMaster() error {
	fmt.Println("Loading Companies Master List ")
	filePath := "C:\\Users\\Ajay\\Downloads\\" + "TOP500" + ".csv"
	url := "https://www1.nseindia.com/content/indices/ind_nifty500list.csv"

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
	companiesMasterList, err := ReadCompaniesMasterCsv("C:\\Users\\Ajay\\Downloads\\" + "TOP500" + ".csv")
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

/* User Profiles */
func AddUser(userInput []byte, db *sql.DB) string {
	fmt.Println("In Add User")
	var user data.User

	json.Unmarshal(userInput, &user)
	user.StartDate = time.Now()

	err := data.AddUserDB(user, db)
	if err != nil {
		fmt.Println(err)
		return "Error while adding new User"
	}
	/* Add to cache too */
	usersCache[user.UserId] = user
	return "Added successfully"
}

func AddUserHoldings(userInput []byte, db *sql.DB) string {
	fmt.Println("In Add User Holdings")
	var holdingsInput data.HoldingsInputJson

	json.Unmarshal(userInput, &holdingsInput)
	fmt.Println(holdingsInput)

	isUserPresent := verifyUserId(holdingsInput.UserID, db)
	if isUserPresent {
		err := data.AddUserHoldingsDB(holdingsInput, db)
		if err != nil {
			fmt.Println(err)
			return "Error while adding new User"
		}
		return "Added User Holdings successfully"
	}
	return "Invalid User"
}

func verifyUserId(userid string, db *sql.DB) bool {
	fmt.Println(userid)
	/* Populate cahce first time */
	if len(usersCache) == 0 {
		users := data.FetchUniqueUsersDB(db)
		for _, user := range users {
			usersCache[user.UserId] = user
		}
		fmt.Println("Loaded User Data In Cache")
	}

	if _, isPresent := usersCache[userid]; isPresent {
		fmt.Println("User data available in Cache")
		return true
	}
	return false

}

func GetUserHoldings(userInput []byte, db *sql.DB) data.HoldingsOutputJson {
	var userHoldings data.HoldingsOutputJson

	var user data.User
	json.Unmarshal(userInput, &user)
	fmt.Println(user)

	isUserPresent := verifyUserId(user.UserId, db)
	if isUserPresent {
		holdings, err := data.GetUserHoldingsDB(user.UserId, db)
		if err != nil {
			fmt.Println(err)
		}
		userHoldings = holdings
	}
	calculateNetWorthAndAlloc(&userHoldings, db)

	return userHoldings
}

func calculateNetWorthAndAlloc(userHoldings *data.HoldingsOutputJson, db *sql.DB) {
	var NW float64
	var eqTotal float64
	var debtTotal float64

	for _, holding := range userHoldings.Holdings {
		FetchLatestCompaniesCompletePrice(holding.Companyid, db)
		latestPriceData := dailyPriceCacheLatest[holding.Companyid]
		qty, errQty := strconv.ParseFloat(holding.Quantity, 64)
		if errQty != nil {
			fmt.Println(errQty)
		}

		NW = NW + (latestPriceData.CloseVal * qty)
		eqTotal = eqTotal + (latestPriceData.CloseVal * qty)
	}

	for _, holdingNT := range userHoldings.HoldingsNT {
		cv, errCV := strconv.ParseFloat(holdingNT.CurrentValue, 64)
		if errCV != nil {
			fmt.Println(errCV)
		}
		NW = NW + cv
		debtTotal = debtTotal + cv
	}

	/* Calculate EQ and Debt % */
	eqPercent := fmt.Sprintf("%f", eqTotal/NW*100)
	debtPercent := fmt.Sprintf("%f", debtTotal/NW*100)

	userHoldings.Allocation.Equity = eqPercent
	userHoldings.Allocation.Debt = debtPercent

	userHoldings.Networth = fmt.Sprintf("%f", NW)
}

/* Add model Pf with allocation and Reasonable price */
func AddModelPortfolio(userInput []byte, db *sql.DB) string {
	fmt.Println("In AddModelPortfolio")
	var modelPf data.ModelPortfolio

	json.Unmarshal(userInput, &modelPf)
	fmt.Println(modelPf)

	isUserPresent := verifyUserId(modelPf.UserID, db)
	if isUserPresent {
		err := data.AddModelPortfolioDB(modelPf, db)
		if err != nil {
			fmt.Println(err)
			return "Error while createing Model Portfolio"
		}
		return "Added Model Portfolio successfully"
	}
	return "Invalid User"
}

/* Fetch Model Pf for given User */
func GetModelPortfolio(userInput []byte, db *sql.DB) data.ModelPortfolio {
	var modelPortfolio data.ModelPortfolio

	var user data.User
	json.Unmarshal(userInput, &user)
	fmt.Println(user)

	isUserPresent := verifyUserId(user.UserId, db)
	if isUserPresent {
		modelPf, err := data.GetModelPortfolioDB(user.UserId, db)
		if err != nil {
			fmt.Println(err)
		}
		modelPortfolio = modelPf
	}

	return modelPortfolio
}

func GetPortfolioModelSync(userInput []byte, db *sql.DB) data.SyncedPortfolio {
	var syncedPf data.SyncedPortfolio

	var user data.User
	json.Unmarshal(userInput, &user)

	/* Get Target Amount */
	targetAmount := data.GetTargetAmountDB(user.UserId, db)
	fmt.Println(targetAmount)

	/* Get Current Holdings */
	holdingsOutputJson := GetUserHoldings(userInput, db)
	fmt.Println(holdingsOutputJson)

	/* Get Model Pf */
	modelPf := GetModelPortfolio(userInput, db)
	fmt.Println(modelPf)

	/* For each Model Pf holding
	1) Check if exists in actual Pf (Tracked Holdings)
	2) Calculate amount to be invested/sold
	3) Check if current price is below reasonable price. */

	/* Form Map to hold company id - key, transactions as Value*/
	holdingsMap := make(map[string][]data.Holdings)
	for _, holding := range holdingsOutputJson.Holdings {
		companyid := holding.Companyid
		holdingsMap[companyid] = append(holdingsMap[companyid], holding)
	}

	for _, security := range modelPf.Securities {

		/* For each Model security, Calculate Amount to Be allocated based on expected allocation & target Amount */
		var amountToBeAllocated float64
		allocation, parseErr := strconv.ParseFloat(security.ExpectedAllocation, 64)
		if parseErr != nil {
			fmt.Println("Error while parsing allocation ")
		}
		amountToBeAllocated = allocation / 100.0 * targetAmount

		/* If already present, check if over/under invested */
		holdings, ok := holdingsMap[security.Securityid]
		if ok {
			cumulativeAmount := CalculateCumulativeInvestedAmount(holdings)
			amountToBeAllocated = (allocation / 100.0 * targetAmount) - cumulativeAmount
		}

		/* Form Structure stating how much to invest/prune in each model security */
		var adjustedHolding data.AdjustedHolding
		adjustedHolding.Securityid = security.Securityid
		adjustedHolding.AdjustedAmount = fmt.Sprintf("%f", amountToBeAllocated)

		/* Check if current price is below reasonable price */
		FetchLatestCompaniesCompletePrice(security.Securityid, db)
		latestPriceData := dailyPriceCacheLatest[security.Securityid]
		secReasonablePrice, _ := strconv.ParseFloat(security.ReasonablePrice, 64)
		if latestPriceData.CloseVal < secReasonablePrice {
			adjustedHolding.BelowReasonablePrice = "Y"
			percentBRP := (secReasonablePrice - latestPriceData.CloseVal) / secReasonablePrice * 100.0
			adjustedHolding.PercentBelowReasonablePrice = fmt.Sprintf("%f", percentBRP)
		} else {
			adjustedHolding.BelowReasonablePrice = "N"
		}

		syncedPf.AdjustedHoldings = append(syncedPf.AdjustedHoldings, adjustedHolding)
	}

	return syncedPf
}

func CalculateCumulativeInvestedAmount(holdings []data.Holdings) float64 {
	var cumulativeAmount float64
	for _, holding := range holdings {
		buyPrice, _ := strconv.ParseFloat(holding.BuyPrice, 64)
		qty, _ := strconv.ParseFloat(holding.Quantity, 64)
		cumulativeAmount = cumulativeAmount + (buyPrice * qty)
	}
	return cumulativeAmount
}
