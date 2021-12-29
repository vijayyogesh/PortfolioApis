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

	"github.com/vijayyogesh/PortfolioApis/constants"
	"github.com/vijayyogesh/PortfolioApis/data"
	"github.com/vijayyogesh/PortfolioApis/util"
)

var dailyPriceCache map[string]map[string]data.CompaniesPriceData = make(map[string]map[string]data.CompaniesPriceData)

var dailyPriceCacheLatest map[string]data.CompaniesPriceData = make(map[string]data.CompaniesPriceData)

var companiesCache []data.Company

var usersCache map[string]data.User = make(map[string]data.User)

var appUtil *util.AppUtil

/* Initializing Processor with required config */
func InitProcessor(appUtilInput *util.AppUtil) {
	appUtil = appUtilInput
}

/* -------------------------------------- */
/* ROUTER METHODS START */

/* 1) Master method that does the following
** Download data file based on TS
** Load into DB
 */
func FetchAndUpdatePrices(db *sql.DB) string {

	/* Update only during market hours */
	hrs, _, _ := time.Now().Clock()
	if hrs >= 10 && hrs <= 16 {

		//Fetch Unique Company Details
		companiesData, err := FetchCompanies(db)
		if err == nil {
			//Download data file
			DownloadDataAsync(companiesData)

			//Read Data From File & Write into DB asynchronously
			LoadPriceData(db)
		}
		return "Prices updated successfully"
	} else {
		return "Prices not updated as current time is outside market hours"
	}
}

/* 2) Fetch/Update Master Companies List */
func FetchAndUpdateCompaniesMasterList() string {
	appUtil.AppLogger.Println("Starting FetchAndUpdateCompaniesMasterList")

	err := DownloadCompaniesMaster()
	if err != nil {
		appUtil.AppLogger.Println(err)
		return constants.AppErrMasterList
	}

	errLoad := LoadCompaniesMaster()
	if errLoad != nil {
		appUtil.AppLogger.Println(errLoad)
		return constants.AppErrMasterList
	}

	appUtil.AppLogger.Println("Completed FetchAndUpdateCompaniesMasterList")
	return constants.AppSuccessMasterList
}

/* 3) Add User */
func AddUser(userInput []byte) string {
	var user data.User

	json.Unmarshal(userInput, &user)
	user.StartDate = time.Now()

	err := data.AddUserDB(user, appUtil.Db)
	if err != nil {
		appUtil.AppLogger.Println(err)
		return constants.AppErrAddUser
	}
	/* Add to cache too */
	usersCache[user.UserId] = user
	return constants.AppSuccessAddUser
}

/* 4) Add User Holdings */
func AddUserHoldings(userInput []byte) string {
	var holdingsInput data.HoldingsInputJson

	json.Unmarshal(userInput, &holdingsInput)

	isUserPresent, err := verifyUserId(holdingsInput.UserID, appUtil.Db)
	if err != nil {
		return constants.AppErrAddUserHoldings
	}

	if isUserPresent {
		err := data.AddUserHoldingsDB(holdingsInput, appUtil.Db)
		if err != nil {
			appUtil.AppLogger.Println(err)
			return constants.AppErrAddUserHoldings
		}
		return constants.AppSuccessAddUserHoldings
	}
	return constants.AppErrAddUserHoldingsInvalid
}

/* 5) Get User Holdings */
func GetUserHoldings(userInput []byte) (data.HoldingsOutputJson, error) {
	var userHoldings data.HoldingsOutputJson

	var user data.User
	json.Unmarshal(userInput, &user)
	appUtil.AppLogger.Println(user)

	isUserPresent, err := verifyUserId(user.UserId, appUtil.Db)
	if err != nil {
		appUtil.AppLogger.Println(err)
		return userHoldings, err
	}

	if isUserPresent {
		holdings, err := data.GetUserHoldingsDB(user.UserId, appUtil.Db)
		if err != nil {
			appUtil.AppLogger.Println(err)
			return userHoldings, err
		}
		userHoldings = holdings
	}
	errCalc := calculateNetWorthAndAlloc(&userHoldings, appUtil.Db)
	if errCalc != nil {
		appUtil.AppLogger.Println(errCalc)
		return userHoldings, errCalc
	}

	return userHoldings, nil
}

/* 6) Add model Pf with allocation and Reasonable price */
func AddModelPortfolio(userInput []byte) string {
	var modelPf data.ModelPortfolio

	json.Unmarshal(userInput, &modelPf)

	isUserPresent, err := verifyUserId(modelPf.UserID, appUtil.Db)
	if err != nil {
		appUtil.AppLogger.Println(err)
		return constants.AppErrAddModelPfInvalidUser
	}

	if isUserPresent {
		err := data.AddModelPortfolioDB(modelPf, appUtil.Db)
		if err != nil {
			appUtil.AppLogger.Println(err)
			return constants.AppErrAddModelPf
		}
		return constants.AppSuccessAddModelPf
	}
	return constants.AppErrAddModelPfInvalidUser
}

/* 7) Fetch Model Portfolio for given User */
func GetModelPortfolio(userInput []byte) (data.ModelPortfolio, error) {
	var modelPortfolio data.ModelPortfolio

	var user data.User
	json.Unmarshal(userInput, &user)

	isUserPresent, err := verifyUserId(user.UserId, appUtil.Db)
	if err != nil {
		appUtil.AppLogger.Println(err)
		return modelPortfolio, err
	}

	if isUserPresent {
		modelPf, err := data.GetModelPortfolioDB(user.UserId, appUtil.Db)
		if err != nil {
			appUtil.AppLogger.Println(err)
			return modelPortfolio, err
		}
		modelPortfolio = modelPf
	}

	return modelPortfolio, nil
}

/* 8) Sync Model Portfolio with actual for given User */
func GetPortfolioModelSync(userInput []byte, db *sql.DB) data.SyncedPortfolio {
	var syncedPf data.SyncedPortfolio

	var user data.User
	json.Unmarshal(userInput, &user)

	/* Get Target Amount */
	targetAmount := data.GetTargetAmountDB(user.UserId, db)

	/* Get Current Holdings */
	holdingsOutputJson, err := GetUserHoldings(userInput)
	if err != nil {
		appUtil.AppLogger.Println(err)
	}

	/* Get Model Pf */
	modelPf, errModelPf := GetModelPortfolio(userInput)
	if errModelPf != nil {
		appUtil.AppLogger.Println(errModelPf)
	}

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
			appUtil.AppLogger.Println("Error while parsing allocation ")
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
		LoadLatestCompaniesCompletePrice(security.Securityid, db)
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

/* 9) Fetch NW Periods for given User */
func FetchNetWorthOverPeriods(userInput []byte, db *sql.DB) map[string]float64 {
	appUtil.AppLogger.Println("Starting FetchNetWorthOverPeriods")

	var networthMap map[string]float64 = make(map[string]float64)

	userHoldings, err := GetUserHoldings(userInput)
	if err != nil {
		appUtil.AppLogger.Println(err)
	}
	for _, holdings := range userHoldings.Holdings {
		dailyPriceRecordsMap := FetchCompaniesCompletePrice(holdings.Companyid, db)
		buyDate, err := time.Parse("2006-01-02T15:04:05Z", holdings.BuyDate)
		if err != nil {
			appUtil.AppLogger.Println(err)
		} else {
			qty, parseErr := strconv.ParseFloat(holdings.Quantity, 64)
			if parseErr != nil {
				appUtil.AppLogger.Println(err)
			}

			/* Loop all dates from Buy Date and calc NW */
			for buyDate.Before(time.Now()) {
				dateStr := buyDate.Format("2006-01-02")
				dailyData, ok := dailyPriceRecordsMap[dateStr]
				if ok {
					networthMap[dateStr] = networthMap[dateStr] + (dailyData.CloseVal * qty)
				}
				buyDate = buyDate.AddDate(0, 0, 1)
			}
		}
	}

	appUtil.AppLogger.Println("Completed FetchNetWorthOverPeriods")
	return networthMap
}

/* ROUTER METHODS END */
/* -------------------------------------- */

/* Fetch Unique Company Details */
func FetchCompanies(db *sql.DB) ([]data.Company, error) {
	if companiesCache != nil {
		appUtil.AppLogger.Println("Fetching Companies Master List From Cache")
		return companiesCache, nil
	} else {
		appUtil.AppLogger.Println("Fetching Companies Master List From DB")
		companies, err := data.FetchCompaniesDB(db)
		if err != nil {
			appUtil.AppLogger.Println(err)
			return companies, err
		}
		companiesCache = companies
		return companies, nil
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

	/* Avoided Go routine as Yahoo Finance blocks more than 5 hits per second */
	for _, company := range companiesData {
		companyId := company.CompanyId
		fromTime := company.LoadDate
		if fromTime.IsZero() {
			fromTime = time.Date(1996, 1, 1, 0, 0, 0, 0, time.UTC)
		}
		time.Sleep(2 * time.Second)
		err := DownloadDataFile(companyId, fromTime)
		if err != nil {
			appUtil.AppLogger.Println(err)
		}
	}
}

/* Download data file from online */
func DownloadDataFile(companyId string, fromTime time.Time) error {
	filePath := appUtil.Config.AppDataDir + companyId + constants.AppDataPricesFileSuffix
	url := constants.AppDataPricesUrl + companyId + constants.AppDataPricesUrlSuffix

	out, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer out.Close()

	/* Form start and end time */
	startTime := strconv.Itoa(int(fromTime.Unix()))
	endTime := strconv.Itoa(int(time.Now().Unix()))

	url = fmt.Sprintf(url, startTime, endTime)

	appUtil.AppLogger.Println("Hitting url " + url + " for company - " + companyId)

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

	appUtil.AppLogger.Println("Completed loading file for company " + companyId)
	return nil
}

/* Read Data From File & Write into DB asynchronously */
func LoadPriceData(db *sql.DB) {
	companies, err := FetchCompanies(db)
	if err == nil {
		var totRecordsCount int64
		var wg sync.WaitGroup

		for _, company := range companies {
			wg.Add(1)
			filePath := appUtil.Config.AppDataDir + company.CompanyId + constants.AppDataPricesFileSuffix

			go func(companyid string) {
				defer wg.Done()
				var err error
				companiesdata, recordsCount, err := ReadDailyPriceCsv(filePath, companyid)
				if err != nil {
					appUtil.AppLogger.Println(err.Error(), "Error returned from ReadDailyPriceCsv ")
				}
				appUtil.AppLogger.Printf("CompanyId - %s RecordsCount - %d ", companyid, recordsCount)
				if recordsCount != 0 {
					atomic.AddInt64(&totRecordsCount, int64(recordsCount))
					appUtil.AppLogger.Printf("Inserting %d records for company %s ", recordsCount, companyid)
					err := data.LoadPriceDataDB(companiesdata, db)
					/* Ignoring data errors for now */
					if err != nil {
						appUtil.AppLogger.Println(err.Error(), " Error while inserting Records for CompanyId: "+companyid)
					}
					data.UpdateLoadDate(db, companyid, time.Now())
				} else {
					appUtil.AppLogger.Println("Skipping DB Insert as file record count is zero for - " + companyid)
				}
			}(company.CompanyId)
		}
		wg.Wait()
	} else {
		appUtil.AppLogger.Println(err)
	}
}

func ReadDailyPriceCsv(filePath string, companyid string) ([]data.CompaniesPriceData, int, error) {
	var companiesdata []data.CompaniesPriceData

	/* Open file */
	file, err := os.Open(filePath)
	/* Return if error */
	if err != nil {
		appUtil.AppLogger.Println(err.Error(), "Error while opening file ")
		return companiesdata, 0, fmt.Errorf("error while opening file %s ", filePath)
	}
	appUtil.AppLogger.Println("Reading from File - " + file.Name() + " for company - " + companyid)

	/* Read csv */
	csvReader := csv.NewReader(file)
	records, err := csvReader.ReadAll()
	/* Return if error */
	if err != nil {
		appUtil.AppLogger.Println(err.Error(), "Error while reading csv ")
		return companiesdata, 0, fmt.Errorf("error while reading csv %s ", filePath)
	}
	/* Close resources */
	file.Close()

	/* Process each record */
	for k, v := range records {
		if k != 0 {

			openval, dataError := strconv.ParseFloat(v[len(v)-6], 64)
			processDataErr(dataError, k, companyid)

			highval, dataError := strconv.ParseFloat(v[len(v)-5], 64)
			processDataErr(dataError, k, companyid)

			lowval, dataError := strconv.ParseFloat(v[len(v)-4], 64)
			processDataErr(dataError, k, companyid)

			closeval, dataError := strconv.ParseFloat(v[len(v)-3], 64)
			processDataErr(dataError, k, companyid)

			dateval, dataError := time.Parse("2006-01-02", v[len(v)-7])
			processDataErr(dataError, k, companyid)

			companiesdata = append(companiesdata, data.CompaniesPriceData{CompanyId: companyid, DateVal: dateval, OpenVal: openval, HighVal: highval, LowVal: lowval, CloseVal: closeval})
		}
	}

	appUtil.AppLogger.Println("Completed Reading from File for company - " + companyid)
	return companiesdata, len(companiesdata), nil

}

/* Non critical record error which can be logged and ignored */
func processDataErr(dataError error, k int, companyid string) {
	if dataError != nil {
		appUtil.AppLogger.Println(dataError)
		appUtil.AppLogger.Printf("Error while processing/reading data for Company - %s record - %d ", companyid, k)
	}
}

/* Fetch All Price Data initially from DB and use cache for subsequent requests */
func FetchCompaniesCompletePrice(companyid string, db *sql.DB) map[string]data.CompaniesPriceData {
	var dailyPriceRecordsMap map[string]data.CompaniesPriceData = make(map[string]data.CompaniesPriceData)
	if dailyPriceCache[companyid] != nil {
		appUtil.AppLogger.Println("FetchCompaniesCompletePrice - From Cache")
		dailyPriceRecordsMap = dailyPriceCache[companyid]
	} else {
		appUtil.AppLogger.Println("FetchCompaniesCompletePrice - From DB")
		dailyPriceRecords := data.FetchCompaniesCompletePriceDataDB(companyid, db)
		for _, priceData := range dailyPriceRecords {
			dateStr := priceData.DateVal.Format("2006-01-02")
			dailyPriceRecordsMap[dateStr] = priceData
		}
		dailyPriceCache[companyid] = dailyPriceRecordsMap
	}
	return dailyPriceRecordsMap
}

/* Load Latest Price Data from DB and use cache for subsequent requests */
func LoadLatestCompaniesCompletePrice(companyid string, db *sql.DB) error {
	if _, ok := dailyPriceCacheLatest[companyid]; ok {
		appUtil.AppLogger.Println("LoadLatestCompaniesCompletePrice - Price data already in Cache")
	} else {
		appUtil.AppLogger.Println("LoadLatestCompaniesCompletePrice - Loading Price Data From DB to Cache")
		dailyPriceRecordsLatest, err := data.FetchCompaniesLatestPriceDataDB(companyid, db)
		if err != nil {
			appUtil.AppLogger.Println(err)
			return err
		}
		dailyPriceCacheLatest[companyid] = dailyPriceRecordsLatest
	}
	return nil
}

/* Download data file from online */
func DownloadCompaniesMaster() error {

	filePath := appUtil.Config.AppDataDir + constants.AppDataMasterFile
	url := constants.AppDataMasterUrl

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

	return nil
}

/* Read Companies Master Data From File & Write into DB  */
func LoadCompaniesMaster() error {
	companiesMasterList, errRead := ReadCompaniesMasterCsv(appUtil.Config.AppDataDir + constants.AppDataMasterFile)
	if errRead != nil {
		return errRead
	}
	errLoad := LoadCompaniesMasterList(companiesMasterList)
	return errLoad
}

/* Write Companies Master List into DB */
func LoadCompaniesMasterList(companiesMasterList []data.Company) error {
	err := data.LoadCompaniesMasterListDB(companiesMasterList, appUtil)
	return err
}

func ReadCompaniesMasterCsv(filePath string) ([]data.Company, error) {
	var companiesMasterList []data.Company

	/* Open file */
	file, err := os.Open(filePath)
	/* Return if error opening file */
	if err != nil {
		appUtil.AppLogger.Println(err.Error(), "Error while opening file ")
		return companiesMasterList, fmt.Errorf("error while opening file %s ", filePath)
	}
	appUtil.AppLogger.Println("Reading from File - " + file.Name())

	/* Read csv */
	csvReader := csv.NewReader(file)
	records, err := csvReader.ReadAll()
	/* Return if error */
	if err != nil {
		appUtil.AppLogger.Println(err.Error(), "Error while reading csv ")
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
			companiesMasterList = append(companiesMasterList, data.Company{CompanyId: companyid, CompanyName: companyname, LoadDate: time.Date(1996, 1, 1, 0, 0, 0, 0, time.UTC)})
		}
	}

	appUtil.AppLogger.Println("Records Read From File - ", len(companiesMasterList))
	return companiesMasterList, nil

}

func verifyUserId(userid string, db *sql.DB) (bool, error) {
	appUtil.AppLogger.Println("Verifying UserId - " + userid)
	/* Populate cache first time */
	if len(usersCache) == 0 {
		users, err := data.FetchUniqueUsersDB(db)
		if err != nil {
			appUtil.AppLogger.Println(err)
			return false, err
		}
		for _, user := range users {
			usersCache[user.UserId] = user
		}
		appUtil.AppLogger.Println("Loaded User Data In Cache")
	}

	if _, isPresent := usersCache[userid]; isPresent {
		appUtil.AppLogger.Println("User data available in Cache")
		return true, nil
	}
	return false, nil
}

func calculateNetWorthAndAlloc(userHoldings *data.HoldingsOutputJson, db *sql.DB) error {
	var NW float64
	var eqTotal float64
	var debtTotal float64

	for _, holding := range userHoldings.Holdings {
		err := LoadLatestCompaniesCompletePrice(holding.Companyid, db)
		if err != nil {
			return err
		}
		latestPriceData := dailyPriceCacheLatest[holding.Companyid]
		qty, errQty := strconv.ParseFloat(holding.Quantity, 64)
		if errQty != nil {
			appUtil.AppLogger.Println(errQty)
			return errQty
		}

		NW = NW + (latestPriceData.CloseVal * qty)
		eqTotal = eqTotal + (latestPriceData.CloseVal * qty)
	}

	for _, holdingNT := range userHoldings.HoldingsNT {
		cv, errCV := strconv.ParseFloat(holdingNT.CurrentValue, 64)
		if errCV != nil {
			appUtil.AppLogger.Println(errCV)
			return errCV
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
	return nil
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
