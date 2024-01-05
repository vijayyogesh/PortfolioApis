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
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/alpeb/go-finance/fin"

	"github.com/vijayyogesh/PortfolioApis/constants"
	"github.com/vijayyogesh/PortfolioApis/data"
	"github.com/vijayyogesh/PortfolioApis/util"
)

var dailyPriceCache map[string]map[string]data.CompaniesPriceData = make(map[string]map[string]data.CompaniesPriceData)

var dailyPriceCacheLatest map[string]data.CompaniesPriceData = make(map[string]data.CompaniesPriceData)

var companiesATHPriceCache map[string]data.CompaniesPriceData

var companiesCache []data.Company

var usersCache map[string]data.User = make(map[string]data.User)

var appUtil *util.AppUtil

var benchmark = "BSE-500"

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
	if (hrs >= 9 && hrs <= 16) && (time.Now().Weekday() != time.Saturday) && (time.Now().Weekday() != time.Sunday) {

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

/* 1b) Update Prices for Company */
func UpdateSelectedCompanies(userInput []byte) string {
	var CompaniesInput data.CompaniesInput
	err := json.Unmarshal(userInput, &CompaniesInput)

	if err == nil {
		//Download data file
		DownloadDataAsync(CompaniesInput.Company)

		//Read Data From File & Write into DB asynchronously
		LoadPriceData(appUtil.Db)
		return constants.AppSuccessUpdateSelectedCompaniesPrice
	} else {
		return constants.AppErrUpdateSelectedCompaniesPrice
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
func AddUser(user data.User) string {
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
	appUtil.AppLogger.Println("Starting AddUserHoldings")
	appUtil.AppLogger.Println(userInput)
	var holdingsInput data.HoldingsInputJson

	json.Unmarshal(userInput, &holdingsInput)

	isUserPresent, err := verifyUserId(holdingsInput.UserID, appUtil.Db)
	if err != nil {
		return constants.AppErrAddUserHoldings
	}

	if isUserPresent {
		appUtil.AppLogger.Println(holdingsInput)

		/* When it is a Sell transaction, move it to cash by default */
		for _, company := range holdingsInput.Holdings {
			qty, errQty := strconv.ParseFloat(company.Quantity, 64)
			sellPrice, errSellPrice := strconv.ParseFloat(company.BuyPrice, 64)
			if errQty != nil {
				appUtil.AppLogger.Println(errQty)
				return constants.AppErrAddUserHoldings
			}
			if errSellPrice != nil {
				appUtil.AppLogger.Println(errSellPrice)
				return constants.AppErrAddUserHoldings
			}

			if qty < 0 {
				value := -qty * sellPrice
				holdingsNTCash := data.HoldingsNonTracked{
					SecurityId:   "CASH",
					BuyDate:      company.BuyDate,
					BuyValue:     fmt.Sprintf("%f", value),
					CurrentValue: fmt.Sprintf("%f", value),
					InterestRate: "0",
				}
				holdingsInput.HoldingsNT = append(holdingsInput.HoldingsNT, holdingsNTCash)
			}
		}

		/* Push data to DB */
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
func GetUserHoldings(userInput []byte, aggregateHoldings bool) (data.HoldingsOutputJson, error) {
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

	if aggregateHoldings {
		aggregatedHoldings, err := AggregateHoldings(userHoldings)
		if err != nil {
			return userHoldings, err
		}
		userHoldings.Holdings = aggregatedHoldings
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
		/* Below block is added for formatting */
		for key := range modelPf.Securities {
			secReasonablePrice, _ := strconv.ParseFloat(modelPf.Securities[key].ReasonablePrice, 64)
			modelPf.Securities[key].ReasonablePrice = fmt.Sprintf("%.2f", secReasonablePrice)
		}
		modelPortfolio = modelPf
	}

	return modelPortfolio, nil
}

/* 8) Sync Model Portfolio with actual for given User */
func GetPortfolioModelSync(userInput []byte) (data.SyncedPortfolio, error) {
	var syncedPf data.SyncedPortfolio

	var user data.User
	json.Unmarshal(userInput, &user)

	/* Get Target Amount */
	targetAmount, err := data.GetTargetAmountDB(user.UserId, appUtil.Db)
	if err != nil {
		appUtil.AppLogger.Println(err)
		return syncedPf, err
	}

	/* Get Current Holdings */
	holdingsOutputJson, err := GetUserHoldings(userInput, true)
	if err != nil {
		appUtil.AppLogger.Println(err)
		return syncedPf, err
	}

	/* Get Model Pf */
	modelPf, errModelPf := GetModelPortfolio(userInput)
	if errModelPf != nil {
		appUtil.AppLogger.Println(errModelPf)
		return syncedPf, errModelPf
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
			appUtil.AppLogger.Println(parseErr)
			return syncedPf, parseErr
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
		adjustedHolding.AdjustedAmount = fmt.Sprintf("%.2f", amountToBeAllocated)

		/* Check if current price is below reasonable price */
		LoadLatestCompaniesCompletePrice(security.Securityid, appUtil.Db)
		latestPriceData := dailyPriceCacheLatest[security.Securityid]
		secReasonablePrice, _ := strconv.ParseFloat(security.ReasonablePrice, 64)
		percentBRP := (secReasonablePrice - latestPriceData.CloseVal) / secReasonablePrice * 100.0
		adjustedHolding.PercentBelowReasonablePrice = fmt.Sprintf("%.2f", percentBRP)
		if latestPriceData.CloseVal < secReasonablePrice {
			adjustedHolding.BelowReasonablePrice = "Y"
		} else {
			adjustedHolding.BelowReasonablePrice = "N"
		}

		syncedPf.AdjustedHoldings = append(syncedPf.AdjustedHoldings, adjustedHolding)
	}

	return syncedPf, nil
}

/* 9) Fetch NW Periods for given User */
func FetchNetWorthOverPeriods(userInput []byte) (map[string]map[string]float64, error) {
	appUtil.AppLogger.Println("Starting FetchNetWorthOverPeriods")

	var combinedOutputMap map[string]map[string]float64 = make(map[string]map[string]float64)

	var networthMap map[string]float64 = make(map[string]float64)
	var trackedHoldingsMap map[string]float64 = make(map[string]float64)
	var nonTrackedHoldingsMap map[string]float64 = make(map[string]float64)
	var allocationMap map[string]float64 = make(map[string]float64)
	var benchMarkMap map[string]float64 = make(map[string]float64)
	var amountInvestedMap map[string]float64 = make(map[string]float64)

	userHoldings, err := GetUserHoldings(userInput, false)
	appUtil.AppLogger.Println(userHoldings)
	if err != nil {
		appUtil.AppLogger.Println(err)
		return combinedOutputMap, err
	}
	for _, holdings := range userHoldings.Holdings {
		dailyPriceRecordsMap := FetchCompaniesCompletePrice(holdings.Companyid, appUtil.Db)

		/* Benchmark changes */
		benchMarkRecordsMap := FetchCompaniesCompletePrice(benchmark, appUtil.Db)
		holdingsQty, _ := strconv.ParseFloat(holdings.Quantity, 64)
		holdingsBuyPrice, _ := strconv.ParseFloat(holdings.BuyPrice, 64)
		holdingsBuyValue := holdingsBuyPrice * holdingsQty

		buyDate, err := time.Parse("2006-01-02T15:04:05Z", holdings.BuyDate)

		/* Benchmark changes */
		bmDateStr := buyDate.Format("2006-01-02")
		bmQty := holdingsBuyValue / benchMarkRecordsMap[bmDateStr].CloseVal

		if err != nil {
			appUtil.AppLogger.Println(err)
			return combinedOutputMap, err
		} else {
			qty, parseErr := strconv.ParseFloat(holdings.Quantity, 64)
			if parseErr != nil {
				appUtil.AppLogger.Println(parseErr)
				return combinedOutputMap, parseErr
			}

			/* Loop all dates from Buy Date and calc NW */
			for buyDate.Before(time.Now()) {
				dateStr := buyDate.Format("2006-01-02")
				dailyData, ok := dailyPriceRecordsMap[dateStr]

				/* Benchmark changes */
				benchMarkData, bmDataExists := benchMarkRecordsMap[dateStr]

				/* Amount Invested */
				investedVal := float64(int((amountInvestedMap[dateStr]+(holdingsBuyValue))*100)) / 100
				amountInvestedMap[dateStr] = investedVal

				if ok && dailyData.CloseVal != 0 {
					networthVal := float64(int((networthMap[dateStr]+(dailyData.CloseVal*qty))*100)) / 100
					networthMap[dateStr] = networthVal
					trackedHoldingsMap[dateStr] = networthVal

					/* Benchmark changes */
					if bmDataExists && benchMarkData.CloseVal != 0 {
						benchMarkVal := float64(int((benchMarkMap[dateStr]+(benchMarkData.CloseVal*bmQty))*100)) / 100
						benchMarkMap[dateStr] = benchMarkVal
					} else {
						previousDate := buyDate.AddDate(0, 0, -1)
						previousDateStr := previousDate.Format("2006-01-02")
						benchMarkMap[dateStr] = benchMarkMap[previousDateStr]
					}
				} else {
					/* As prices are zero on Holidays */
					isZero := true
					buyDateCopy := buyDate
					for isZero {
						previousDate := buyDateCopy.AddDate(0, 0, -1)
						previousDateStr := previousDate.Format("2006-01-02")
						dailyDataCopy, ok := dailyPriceRecordsMap[previousDateStr]

						benchMarkDataCopy, bmDataExists := benchMarkRecordsMap[previousDateStr]

						if ok && dailyDataCopy.CloseVal != 0 {
							networthVal := float64(int((networthMap[dateStr]+(dailyDataCopy.CloseVal*qty))*100)) / 100
							networthMap[dateStr] = networthVal
							trackedHoldingsMap[dateStr] = networthVal
							isZero = false

							/* Benchmark changes */
							if bmDataExists && benchMarkDataCopy.CloseVal != 0 {
								benchMarkVal := float64(int((benchMarkMap[dateStr]+(benchMarkDataCopy.CloseVal*bmQty))*100)) / 100
								benchMarkMap[dateStr] = benchMarkVal
							} else {
								benchMarkMap[dateStr] = benchMarkMap[previousDateStr]
							}

							break
						}
						buyDateCopy = previousDate
					}
				}
				buyDate = buyDate.AddDate(0, 0, 1)
				allocationMap[dateStr] = (trackedHoldingsMap[dateStr] / networthMap[dateStr]) * 100
			}
		}
	}

	/* Add NT Holdings to NW */
	for _, holdingsNt := range userHoldings.HoldingsNT {
		buyDate, err := time.Parse("2006-01-02T15:04:05Z", holdingsNt.BuyDate)
		if err != nil {
			appUtil.AppLogger.Println(err)
			return combinedOutputMap, err
		}

		for buyDate.Before(time.Now()) {
			dateStr := buyDate.Format("2006-01-02")
			currVal, parseErr := strconv.ParseFloat(holdingsNt.CurrentValue, 64)
			if parseErr != nil {
				appUtil.AppLogger.Println(parseErr)
				return combinedOutputMap, parseErr
			}

			networthVal := float64(int((networthMap[dateStr]+currVal)*100)) / 100
			networthMap[dateStr] = networthVal
			nonTrackedHoldingsMap[dateStr] = float64(int((nonTrackedHoldingsMap[dateStr]+currVal)*100)) / 100

			buyDate = buyDate.AddDate(0, 0, 1)

			allocationMap[dateStr] = (trackedHoldingsMap[dateStr] / networthMap[dateStr]) * 100
		}
	}

	combinedOutputMap["networth"] = networthMap
	combinedOutputMap["equity"] = trackedHoldingsMap
	combinedOutputMap["benchmark"] = benchMarkMap
	combinedOutputMap["debt"] = nonTrackedHoldingsMap
	combinedOutputMap["invested"] = amountInvestedMap

	appUtil.AppLogger.Println("Completed FetchNetWorthOverPeriods")
	return combinedOutputMap, nil
}

/* 10) Fetch All Company Names */
func FetchAllCompanies(userInput []byte) ([]data.Company, error) {
	return FetchCompanies(appUtil.Db)
}

/* 11) Calculate Return */
func CalculateReturn(userInput []byte) (string, error) {
	/* Get Current Holdings */
	holdingsOutputJson, err := GetUserHoldings(userInput, false)
	if err != nil {
		appUtil.AppLogger.Println(err)
		return "", err
	}

	var totalUnits float64

	/* No of units * 10 => Total Buy Value
	   No of units * NAV => Total Current Value */

	var holdingsBuyDateMap = make(map[string][]data.Holdings)
	for _, holding := range holdingsOutputJson.Holdings {
		buyDate, _ := time.Parse("2006-01-02T15:04:05Z", holding.BuyDate)
		dateStr := buyDate.Format("2006-01-02")
		holdingsBuyDateMap[dateStr] = append(holdingsBuyDateMap[dateStr], holding)
	}
	appUtil.AppLogger.Println("holdingsBuyDateMap ")
	appUtil.AppLogger.Println(holdingsBuyDateMap)

	for _, holding := range holdingsOutputJson.Holdings {
		err := LoadLatestCompaniesCompletePrice(holding.Companyid, appUtil.Db)
		if err != nil {
			return "", err
		}
		//latestPriceData := dailyPriceCacheLatest[holding.Companyid]
		qty, errQty := strconv.ParseFloat(holding.Quantity, 64)
		if errQty != nil {
			appUtil.AppLogger.Println(errQty)
			return "", errQty
		}

		buyPrice, errBuyPrice := strconv.ParseFloat(holding.BuyPrice, 64)
		if errBuyPrice != nil {
			appUtil.AppLogger.Println(errBuyPrice)
			return "", errBuyPrice
		}
		totalUnits = totalUnits + ((buyPrice * qty) / 10)
		//appUtil.AppLogger.Println("totalUnits ")
		//appUtil.AppLogger.Println(totalUnits)
	}

	return constants.AppSuccessCalculateReturn, nil
}

/* 12) Calculate Index SIP Return */
func CalculateIndexSIPReturn(userInput []byte) (data.SIPReturnOutput, error) {
	var sipReturnInput data.SIPReturnInput
	err := json.Unmarshal(userInput, &sipReturnInput)

	if err == nil {
		appUtil.AppLogger.Println("SIPReturnInput")
		appUtil.AppLogger.Println(sipReturnInput)
	}

	var sipReturnOutput data.SIPReturnOutput
	var sipReturnSubPeriodArr []data.SIPReturnSubPeriod
	var sipReturnSubPeriod data.SIPReturnSubPeriod
	var dates []time.Time
	var values []float64
	var periodCount int64
	startDateStr := sipReturnInput.SIPReturnInputParam.StartDate
	endDateStr := sipReturnInput.SIPReturnInputParam.EndDate
	sipAmountStr := sipReturnInput.SIPReturnInputParam.SIPAmount
	companyId := sipReturnInput.SIPReturnInputParam.Companyid
	stepUpPct, _ := strconv.ParseFloat(sipReturnInput.SIPReturnInputParam.StepUpPct, 64)

	FetchCompaniesCompletePrice(companyId, appUtil.Db)
	dailyPriceRecordsMap := dailyPriceCache[companyId]

	startDate, _ := time.Parse("2006/01/02", startDateStr)
	appUtil.AppLogger.Println(startDate)

	endDate, _ := time.Parse("2006/01/02", endDateStr)
	appUtil.AppLogger.Println(endDate)

	sipAmount, _ := strconv.ParseFloat(sipAmountStr, 64)
	qty := 0.0
	finalCloseVal := 0.0

	for startDate.Before(endDate) || startDate.Equal(endDate) {
		dates = append(dates, startDate)

		closeVal := dailyPriceRecordsMap[startDate.Format("2006-01-02")].CloseVal
		startDateUpdated := startDate

		for closeVal < 1 {
			startDateUpdated = startDateUpdated.AddDate(0, 0, 1)
			closeVal = dailyPriceRecordsMap[startDateUpdated.Format("2006-01-02")].CloseVal
		}

		qty = qty + (sipAmount / closeVal)
		values = append(values, -sipAmount)

		finalCloseVal = closeVal
		xirrSubPeriod, errXirr := fin.ScheduledInternalRateOfReturn(append(values, qty*finalCloseVal), append(dates, startDate), 0.0)
		if errXirr != nil {
			appUtil.AppLogger.Println(errXirr)
		}

		periodCount++

		sipReturnSubPeriod.Quantity = fmt.Sprintf("%.2f", qty)
		sipReturnSubPeriod.EndDate = startDate.Format("2006-01-02")
		sipReturnSubPeriod.TotalEndValue = fmt.Sprintf("%.2f", qty*finalCloseVal)
		sipReturnSubPeriod.Xirr = fmt.Sprintf("%.2f", xirrSubPeriod*100)
		sipReturnSubPeriod.TotalInvestment = fmt.Sprintf("%.2f", float64(periodCount*int64(sipAmount)))
		sipReturnSubPeriod.BuyVal = fmt.Sprintf("%.2f", finalCloseVal)
		sipReturnSubPeriodArr = append(sipReturnSubPeriodArr, sipReturnSubPeriod)

		startDate = startDate.AddDate(0, 1, 0)

		if periodCount%12 == 0 {
			sipAmount = sipAmount + (stepUpPct / 100 * sipAmount)
			appUtil.AppLogger.Println("updated Sip Amount ")
			appUtil.AppLogger.Println(sipAmount)
		}
	}

	appUtil.AppLogger.Println("sipReturnSubPeriodArr - ")
	for _, sipReturnSubPeriod := range sipReturnSubPeriodArr {
		appUtil.AppLogger.Println(sipReturnSubPeriod)
	}

	var lessThanZeroCount float64
	var zeroToTwoCount float64
	var twoToFiveCount float64
	var fiveToSevenCount float64
	var sevenToTenCount float64
	var greaterThanTenCount float64
	for _, sipReturnSubPeriod := range sipReturnSubPeriodArr {
		xirrVal, _ := strconv.ParseFloat(sipReturnSubPeriod.Xirr, 64)
		if xirrVal <= 0 {
			lessThanZeroCount++
		} else if xirrVal > 0 && xirrVal <= 2.5 {
			zeroToTwoCount++
		} else if xirrVal > 2.5 && xirrVal <= 5 {
			twoToFiveCount++
		} else if xirrVal > 5 && xirrVal <= 7.5 {
			fiveToSevenCount++
		} else if xirrVal > 7.5 && xirrVal <= 10 {
			sevenToTenCount++
		} else if xirrVal > 10 {
			greaterThanTenCount++
		}
	}

	var sipReturnBracket data.SIPReturnBracket
	sipReturnBracket.LessThanZeroCount = fmt.Sprintf("%.2f", lessThanZeroCount/float64(periodCount)*100)
	sipReturnBracket.ZeroToTwoCount = fmt.Sprintf("%.2f", zeroToTwoCount/float64(periodCount)*100)
	sipReturnBracket.TwoToFiveCount = fmt.Sprintf("%.2f", twoToFiveCount/float64(periodCount)*100)
	sipReturnBracket.FiveToSevenCount = fmt.Sprintf("%.2f", fiveToSevenCount/float64(periodCount)*100)
	sipReturnBracket.SevenToTenCount = fmt.Sprintf("%.2f", sevenToTenCount/float64(periodCount)*100)
	sipReturnBracket.GreaterThanTenCount = fmt.Sprintf("%.2f", greaterThanTenCount/float64(periodCount)*100)

	sipReturnOutput.SIPReturnBracket = sipReturnBracket
	sipReturnOutput.SIPReturnSubPeriod = sipReturnSubPeriodArr

	appUtil.AppLogger.Println("sipReturnOutput - ")
	appUtil.AppLogger.Println(sipReturnOutput)

	appUtil.AppLogger.Println("sipReturnBracket - ")
	appUtil.AppLogger.Println(sipReturnBracket)

	dates = append(dates, endDate)
	values = append(values, qty*finalCloseVal)

	xirr, err := fin.ScheduledInternalRateOfReturn(values, dates, 0.0)
	if err != nil {
		appUtil.AppLogger.Println(err)
	}
	appUtil.AppLogger.Println("XIRR - ")
	appUtil.AppLogger.Println(xirr)

	return sipReturnOutput, nil
}

/* 13) Calculate All Time High for Portfolio */
func CalculateATHforPF(userInput []byte) (data.HoldingsOutputJson, error) {
	var holdingsATHOutputJson data.HoldingsOutputJson
	var netWorth float64

	/* Get Current Holdings */
	holdingsOutputJson, err := GetUserHoldings(userInput, true)
	if err != nil {
		appUtil.AppLogger.Println(err)
	}

	GetATHforCompanies()

	if companiesATHPriceCache != nil {
		for _, holding := range holdingsOutputJson.Holdings {
			companyId := holding.Companyid
			athPrice := companiesATHPriceCache[companyId].CloseVal
			holding.LTP = fmt.Sprintf("%.2f", athPrice)

			qty, _ := strconv.ParseFloat(holding.Quantity, 64)
			holding.CurrentValue = fmt.Sprintf("%.2f", athPrice*qty)

			holdingBuyPrice, _ := strconv.ParseFloat(holding.BuyPrice, 64)
			holdingBuyVal := holdingBuyPrice * qty
			holdingPL := ((athPrice * qty) - (holdingBuyPrice * qty))
			holding.PL = fmt.Sprintf("%.2f", holdingPL)

			holdingReturn := (holdingPL / holdingBuyVal) * 100
			holding.NetPct = fmt.Sprintf("%.2f", holdingReturn)

			netWorth = netWorth + (athPrice * qty)

			holdingsATHOutputJson.Holdings = append(holdingsATHOutputJson.Holdings, holding)
		}
		holdingsATHOutputJson.Networth = fmt.Sprintf("%.2f", netWorth)
	}
	return holdingsATHOutputJson, nil
}

/* 14) Calculate xirr values for Portfolio from start date to Now */
func CalculateXirrReturn(userInput []byte) (map[string]map[string]float64, error) {

	/* Output map with xirr values */
	var combinedOutputMap map[string]map[string]float64 = make(map[string]map[string]float64)
	var xirrDateMap map[string]float64 = make(map[string]float64)
	var bmXirrDateMap map[string]float64 = make(map[string]float64)

	/* Fetch Holdings grouped by Buy Date */
	holdingsDateMap, startDateTime := GetHoldingsDateWiseMapForUser(userInput)
	var holdingsDataAsOfDate []data.Holdings

	startDate, _ := time.Parse("2006-01-02", startDateTime)
	endDate := time.Now()

	/* Exclude first 6 months in output for return */
	cutOffDate := startDate.AddDate(0, 6, 0)

	var dates []time.Time
	var values []float64
	var bmValues []float64
	finalCloseVal := 0.0
	bmFinalCloseVal := 0.0
	skipDate := false

	/* Added for cases where prices are zero */
	var latestDates []time.Time
	var latestValues []float64
	latestCloseVal := 0.0
	var bmLatestValues []float64
	bmLatestCloseVal := 0.0

	/* Loop all dates from PF start date */
	for startDate.Before(endDate) || startDate.Equal(endDate) {
		startDateStr := startDate.Format("2006-01-02")

		/* Append to user holdings when new buydate is available */
		if _, ok := holdingsDateMap[startDateStr]; ok {
			holdingsDataAsOfDate = append(holdingsDataAsOfDate, holdingsDateMap[startDateStr]...)
		}

		/* Loop Holdings and calculate value/portfolio value with prices of a particular day  */
		for _, holding := range holdingsDataAsOfDate {
			dailyPriceRecordsMap := FetchCompaniesCompletePrice(holding.Companyid, appUtil.Db)

			closeVal := dailyPriceRecordsMap[startDate.Format("2006-01-02")].CloseVal
			holdingBuyPrice, _ := strconv.ParseFloat(holding.BuyPrice, 64)
			qty, _ := strconv.ParseFloat(holding.Quantity, 64)
			buyDate, _ := time.Parse("2006-01-02T15:04:05Z", holding.BuyDate)
			finalCloseVal = finalCloseVal + (qty * closeVal)

			if closeVal < 1 {
				skipDate = true
				break
			}

			values = append(values, qty*-holdingBuyPrice)
			dates = append(dates, buyDate)

			/* Benchmark changes */
			bmDailyPriceRecordsMap := FetchCompaniesCompletePrice(benchmark, appUtil.Db)
			bmCloseVal := bmDailyPriceRecordsMap[startDate.Format("2006-01-02")].CloseVal
			bmBuyDateVal := bmDailyPriceRecordsMap[buyDate.Format("2006-01-02")].CloseVal
			bmQty := (qty * holdingBuyPrice) / bmBuyDateVal
			bmFinalCloseVal = bmFinalCloseVal + (bmQty * bmCloseVal)
			bmValues = append(bmValues, bmQty*-bmBuyDateVal)
		}

		if skipDate {
			/* When prices are zero/holidays, use latest available values with start date */
			xirrSubPeriod, errXirr := fin.ScheduledInternalRateOfReturn(append(latestValues, latestCloseVal), append(latestDates, startDate), 0.0)
			if errXirr != nil {
				appUtil.AppLogger.Println(errXirr)
				xirrSubPeriod = 0.0
			}
			xirrFloat, _ := strconv.ParseFloat(fmt.Sprintf("%.2f", xirrSubPeriod*100), 64)

			if startDate.After(cutOffDate) {
				xirrDateMap[startDateStr] = xirrFloat
			}

			/* Benchmark changes */
			bmXirrSubPeriod, bmErrXirr := fin.ScheduledInternalRateOfReturn(append(bmLatestValues, bmLatestCloseVal), append(latestDates, startDate), 0.0)
			if bmErrXirr != nil {
				appUtil.AppLogger.Println(bmErrXirr)
				bmXirrSubPeriod = 0.0
			}
			bmXirrFloat, _ := strconv.ParseFloat(fmt.Sprintf("%.2f", bmXirrSubPeriod*100), 64)
			if startDate.After(cutOffDate) {
				bmXirrDateMap[startDateStr] = bmXirrFloat
			}
		} else {
			xirrSubPeriod, errXirr := fin.ScheduledInternalRateOfReturn(append(values, finalCloseVal), append(dates, startDate), 0.0)
			if errXirr != nil {
				appUtil.AppLogger.Println(errXirr)
				xirrSubPeriod = 0.0
			}
			xirrFloat, _ := strconv.ParseFloat(fmt.Sprintf("%.2f", xirrSubPeriod*100), 64)
			if startDate.After(cutOffDate) {
				xirrDateMap[startDateStr] = xirrFloat
			}

			/* Benchmark changes */
			bmXirrSubPeriod, bmErrXirr := fin.ScheduledInternalRateOfReturn(append(bmValues, bmFinalCloseVal), append(dates, startDate), 0.0)
			if bmErrXirr != nil {
				appUtil.AppLogger.Println(bmErrXirr)
				xirrSubPeriod = 0.0
			}
			bmXirrFloat, _ := strconv.ParseFloat(fmt.Sprintf("%.2f", bmXirrSubPeriod*100), 64)
			if startDate.After(cutOffDate) {
				bmXirrDateMap[startDateStr] = bmXirrFloat
			}
		}

		/* Set to latest available values */
		if !skipDate {
			latestDates = dates
			latestValues = values
			latestCloseVal = finalCloseVal

			/* Benchmark changes */
			bmLatestCloseVal = bmFinalCloseVal
			bmLatestValues = bmValues
		}

		/* Reset loop variables */
		finalCloseVal = 0.0
		values = values[:0]
		dates = dates[:0]
		startDate = startDate.AddDate(0, 0, 1)
		skipDate = false

		/* Benchmark changes */
		bmFinalCloseVal = 0.0
		bmValues = bmValues[:0]
	}

	combinedOutputMap["portfolioReturn"] = xirrDateMap
	combinedOutputMap["benchmarkReturn"] = bmXirrDateMap

	return combinedOutputMap, nil
}

/* 14) Calculate nav style returns for Portfolio from start date to Now */
func CalculateNavReturn(userInput []byte) (map[string]float64, error) {
	/* Output map with NAV values */
	var navDateMap map[string]float64 = make(map[string]float64)
	return navDateMap, nil
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
	filePath := ""
	url := ""
	if strings.Contains(companyId, constants.AppDataPrefixMF) {
		filePath = appUtil.Config.AppDataDir + companyId + constants.AppDataPricesFileSuffixMF
		url = constants.AppDataPricesUrl + companyId + constants.AppDataPricesUrlSuffixMF
	} else if strings.Compare(companyId, constants.AppDataBenchmarkNSE) == 0 {
		filePath = appUtil.Config.AppDataDir + companyId + constants.AppDataCsv
		url = constants.AppDataPricesUrl + constants.AppDataBenchmarkAppenderText + companyId + constants.AppDataBenchmarkPricesUrlSuffix
	} else if strings.Contains(companyId, constants.AppDataBenchmarkBSE) {
		filePath = appUtil.Config.AppDataDir + companyId + constants.AppDataPricesFileSuffixMF
		url = constants.AppDataPricesUrl + companyId + constants.AppDataPricesUrlSuffixMF
	} else {
		filePath = appUtil.Config.AppDataDir + companyId + constants.AppDataPricesFileSuffix
		url = constants.AppDataPricesUrl + companyId + constants.AppDataPricesUrlSuffix
	}

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
			filePath := ""
			if strings.Contains(company.CompanyId, constants.AppDataPrefixMF) {
				filePath = appUtil.Config.AppDataDir + company.CompanyId + constants.AppDataPricesFileSuffixMF
			} else if strings.Compare(company.CompanyId, constants.AppDataBenchmarkNSE) == 0 {
				filePath = appUtil.Config.AppDataDir + company.CompanyId + constants.AppDataCsv
			} else if strings.Contains(company.CompanyId, constants.AppDataBenchmarkBSE) {
				filePath = appUtil.Config.AppDataDir + company.CompanyId + constants.AppDataPricesFileSuffixMF
			} else {
				filePath = appUtil.Config.AppDataDir + company.CompanyId + constants.AppDataPricesFileSuffix
			}

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
	dailyPriceCache = make(map[string]map[string]data.CompaniesPriceData)
	dailyPriceCacheLatest = make(map[string]data.CompaniesPriceData)
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
		//appUtil.AppLogger.Println("FetchCompaniesCompletePrice - From Cache")
		dailyPriceRecordsMap = dailyPriceCache[companyid]
	} else {
		//appUtil.AppLogger.Println("FetchCompaniesCompletePrice - From DB")
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
		//appUtil.AppLogger.Println("LoadLatestCompaniesCompletePrice - Price data already in Cache")
	} else {
		//appUtil.AppLogger.Println("LoadLatestCompaniesCompletePrice - Loading Price Data From DB to Cache")
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

/* Compare userInput password with DB */
func IsValidPassword(user data.User) bool {
	password, err := data.GetPassword(user.UserId, appUtil.Db)
	if err != nil {
		appUtil.AppLogger.Println(err)
		return false
	} else if user.Password == password {
		return true
	}
	return false
}

/* Prepare data for Holdings table */
func AggregateHoldings(userHoldings data.HoldingsOutputJson) ([]data.Holdings, error) {

	var holdingsAggregated []data.Holdings

	var holdingsMap map[string]data.Holdings = make(map[string]data.Holdings)

	var aggregatedBuyVal float64
	var aggregatedBuyPrice float64

	/* Loop transaction data and aggregate as needed */
	for _, holding := range userHoldings.Holdings {
		if holdingMapVal, isPresent := holdingsMap[holding.Companyid]; isPresent {

			holdingMapQty, err := strconv.ParseFloat(holdingMapVal.Quantity, 64)
			if err != nil {
				appUtil.AppLogger.Println(err)
				return holdingsAggregated, err
			}

			holdingMapBuyPrice, err := strconv.ParseFloat(holdingMapVal.BuyPrice, 64)
			if err != nil {
				appUtil.AppLogger.Println(err)
				return holdingsAggregated, err
			}

			holdingQty, err := strconv.ParseFloat(holding.Quantity, 64)
			if err != nil {
				appUtil.AppLogger.Println(err)
				return holdingsAggregated, err
			}

			holdingBuyPrice, err := strconv.ParseFloat(holding.BuyPrice, 64)
			if err != nil {
				appUtil.AppLogger.Println(err)
				return holdingsAggregated, err
			}

			aggregatedQty := holdingMapQty + holdingQty

			if holdingQty > 0 {
				aggregatedBuyVal = (holdingMapQty * holdingMapBuyPrice) + (holdingQty * holdingBuyPrice)
				aggregatedBuyPrice = aggregatedBuyVal / aggregatedQty
			} else {
				/* Use same buy price & buy val for a sell transaction */
				aggregatedBuyVal = aggregatedQty * holdingMapBuyPrice
				aggregatedBuyPrice = holdingMapBuyPrice
			}

			latestPriceData := dailyPriceCacheLatest[holding.Companyid]

			/* Set aggregated Qty/Buy Price/Current Val/PL/return */
			var updatedHolding data.Holdings
			updatedHolding.Companyid = holding.Companyid
			updatedHolding.CompanyName = holding.CompanyName
			updatedHolding.Quantity = fmt.Sprintf("%.0f", aggregatedQty)
			updatedHolding.BuyPrice = fmt.Sprintf("%.2f", aggregatedBuyPrice)
			updatedHolding.LTP = fmt.Sprintf("%.2f", latestPriceData.CloseVal)
			updatedHolding.BuyDate = holding.BuyDate

			aggregatedCurrentVal := latestPriceData.CloseVal * aggregatedQty
			updatedHolding.CurrentValue = fmt.Sprintf("%.2f", aggregatedCurrentVal)

			aggregatedPL := aggregatedCurrentVal - aggregatedBuyVal
			updatedHolding.PL = fmt.Sprintf("%.2f", aggregatedPL)

			aggregatedReturn := (aggregatedPL / aggregatedBuyVal) * 100
			updatedHolding.NetPct = fmt.Sprintf("%.2f", aggregatedReturn)

			holdingsMap[holding.Companyid] = updatedHolding

		} else {
			latestPriceData := dailyPriceCacheLatest[holding.Companyid]

			qty, err := strconv.ParseFloat(holding.Quantity, 64)
			if err != nil {
				appUtil.AppLogger.Println(err)
				return holdingsAggregated, err
			}

			buyPrice, err := strconv.ParseFloat(holding.BuyPrice, 64)
			if err != nil {
				appUtil.AppLogger.Println(err)
				return holdingsAggregated, err
			}
			holding.Quantity = fmt.Sprintf("%.0f", qty)
			holding.BuyPrice = fmt.Sprintf("%.2f", buyPrice)

			holding.LTP = fmt.Sprintf("%.2f", latestPriceData.CloseVal)
			holding.CurrentValue = fmt.Sprintf("%.2f", latestPriceData.CloseVal*qty)

			holdingBuyVal := buyPrice * qty
			holdingPL := ((latestPriceData.CloseVal * qty) - holdingBuyVal)
			holding.PL = fmt.Sprintf("%.2f", holdingPL)

			holdingReturn := (holdingPL / holdingBuyVal) * 100
			holding.NetPct = fmt.Sprintf("%.2f", holdingReturn)

			holdingsMap[holding.Companyid] = holding
		}
	}

	/* Flatten Holdings aggregated map to slice */

	for _, holding := range holdingsMap {
		holdingsAggregated = append(holdingsAggregated, holding)
	}

	return holdingsAggregated, nil

}

/* Fetch ATH of companies and store in cache map */
func GetATHforCompanies() (map[string]data.CompaniesPriceData, error) {
	var companiesATHMap map[string]data.CompaniesPriceData = make(map[string]data.CompaniesPriceData)

	if companiesATHPriceCache != nil {
		appUtil.AppLogger.Println("Fetching ATH for Companies From Cache")
		return companiesATHPriceCache, nil
	} else {
		appUtil.AppLogger.Println("Fetching ATH for Companies From DB")
		athRecords, err := data.FetchATHForCompaniesDB(appUtil.Db)
		if err != nil {
			return companiesATHMap, err
		}

		for _, companyData := range athRecords {
			companiesATHMap[companyData.CompanyId] = companyData
		}
		companiesATHPriceCache = companiesATHMap
		return companiesATHPriceCache, nil
	}
}

/* Fetch Holdings grouped by Buy Date */
func GetHoldingsDateWiseMapForUser(userInput []byte) (map[string][]data.Holdings, string) {

	var startDate string

	userHoldings, err := GetUserHoldings(userInput, false)
	if err != nil {
		appUtil.AppLogger.Println(err)
	}

	/* Form Map to hold buyDate - key, transactions as Value*/
	holdingsMap := make(map[string][]data.Holdings)
	for _, holding := range userHoldings.Holdings {
		buyDate, _ := time.Parse("2006-01-02T15:04:05Z", holding.BuyDate)
		buyDateStr := buyDate.Format("2006-01-02")
		holdingsMap[buyDateStr] = append(holdingsMap[buyDateStr], holding)
		if startDate == "" {
			startDate = buyDateStr
		}
	}

	return holdingsMap, startDate
}
