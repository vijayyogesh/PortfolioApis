package data

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/vijayyogesh/PortfolioApis/util"
)

type Company struct {
	CompanyId   string
	CompanyName string
	LoadDate    time.Time
}

type CompaniesPriceData struct {
	CompanyId string
	OpenVal   float64
	HighVal   float64
	LowVal    float64
	CloseVal  float64
	DateVal   time.Time
}

type User struct {
	UserId       string
	StartDate    time.Time
	TargetAmount float64 `json:",string"`
	Password     string  `json:"password"`
}

type HoldingsInputJson struct {
	UserID     string               `json:"userId"`
	Holdings   []Holdings           `json:"Holdings"`
	HoldingsNT []HoldingsNonTracked `json:"HoldingsNonTracked"`
}

type HoldingsOutputJson struct {
	UserID     string               `json:"userId"`
	Holdings   []Holdings           `json:"Holdings"`
	HoldingsNT []HoldingsNonTracked `json:"HoldingsNonTracked"`
	Networth   string               `json:"Networth"`
	Allocation Allocation           `json:"Allocation"`
}

type Holdings struct {
	Companyid    string `json:"companyid"`
	CompanyName  string `json:"companyName"`
	Quantity     string `json:"quantity"`
	BuyDate      string `json:"buyDate"`
	BuyPrice     string `json:"buyPrice"`
	LTP          string `json:"ltp"`
	CurrentValue string `json:"currentValue"`
	PL           string `json:"pl"`
	NetPct       string `json:"netPct"`
}

type Allocation struct {
	Equity string `json:"equity"`
	Debt   string `json:"debt"`
}

type HoldingsNonTracked struct {
	SecurityId   string `json:"securityid"`
	BuyDate      string `json:"buyDate"`
	BuyValue     string `json:"buyValue"`
	CurrentValue string `json:"currentValue"`
	InterestRate string `json:"interestRate"`
}

type ModelPortfolio struct {
	UserID     string       `json:"userId"`
	Securities []Securities `json:"Securities"`
}

type Securities struct {
	Securityid         string `json:"securityid"`
	ReasonablePrice    string `json:"reasonablePrice"`
	ExpectedAllocation string `json:"expectedAllocation"`
}

type SyncedPortfolio struct {
	AdjustedHoldings []AdjustedHolding
}

type AdjustedHolding struct {
	Securityid                  string `json:"securityid"`
	AdjustedAmount              string `json:"adjustedAmount"`
	BelowReasonablePrice        string `json:"belowReasonablePrice"`
	PercentBelowReasonablePrice string `json:"percentBelowReasonablePrice"`
}

type NetworthOverPeriod struct {
	NetworthWithDates []NetworthOnADate
}

type NetworthOnADate struct {
	Date     string `json:"date"`
	Networth string `json:"networth"`
}

type CompaniesInput struct {
	UserID  string    `json:"userId"`
	Company []Company `json:"Company"`
}

type SIPReturnInputParam struct {
	Companyid string `json:"companyid"`
	StartDate string `json:"startdate"`
	EndDate   string `json:"enddate"`
	SIPAmount string `json:"sipamount"`
	StepUpPct string `json:"stepuppct"`
}

type SIPReturnInput struct {
	UserID              string              `json:"userId"`
	SIPReturnInputParam SIPReturnInputParam `json:"sipParams"`
}

type SIPReturnOutput struct {
	SIPReturnSubPeriod []SIPReturnSubPeriod `json:"sipReturnSubPeriod"`
	SIPReturnBracket   SIPReturnBracket     `json:"sipReturnBracket"`
}

type SIPReturnSubPeriod struct {
	Quantity        string `json:"quantity"`
	EndDate         string `json:"enddate"`
	TotalInvestment string `json:"totalInvestment"`
	TotalEndValue   string `json:"totalEndValue"`
	Xirr            string `json:"xirr"`
	BuyVal          string `json:"buyval"`
}

type SIPReturnBracket struct {
	LessThanZeroCount   string `json:"lessThanZeroCount"`
	ZeroToTwoCount      string `json:"zeroToTwoCount"`
	TwoToFiveCount      string `json:"twoToFiveCount"`
	FiveToSevenCount    string `json:"fiveToSevenCount"`
	SevenToTenCount     string `json:"sevenToTenCount"`
	GreaterThanTenCount string `json:"greaterThanTenCount"`
}

func LoadPriceDataDB(dailyPriceRecords []CompaniesPriceData, db *sql.DB) error {
	valueStrings := make([]string, 0, len(dailyPriceRecords))
	valueArgs := make([]interface{}, 0, len(dailyPriceRecords)*6)

	/* Loop and Bulk Insert Records */
	for k, v := range dailyPriceRecords {
		valueStrings = append(valueStrings, fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, $%d)", k*6+1, k*6+2, k*6+3, k*6+4, k*6+5, k*6+6))
		valueArgs = append(valueArgs, v.CompanyId, v.OpenVal, v.HighVal, v.LowVal, v.CloseVal, v.DateVal)
	}
	stmt := fmt.Sprintf("INSERT INTO COMPANIES_PRICE_DATA(COMPANY_ID, OPEN_VAL,HIGH_VAL, LOW_VAL, CLOSE_VAL, DATE_VAL) VALUES %s "+
		" ON CONFLICT(COMPANY_ID, DATE_VAL) DO UPDATE SET CLOSE_VAL = excluded.CLOSE_VAL ", strings.Join(valueStrings, ","))

	_, err := db.Exec(stmt, valueArgs...)
	if err != nil {
		return err
	}

	return nil
}

/* Fetch All Price Data for a given company */
func FetchCompaniesCompletePriceDataDB(companyid string, db *sql.DB) []CompaniesPriceData {
	var dailyPriceRecords []CompaniesPriceData
	records, err := db.Query("SELECT DATE_VAL, CLOSE_VAL FROM COMPANIES_PRICE_DATA WHERE COMPANY_ID = $1 ", companyid)
	if err != nil {
		panic(err.Error())
	}
	defer records.Close()
	for records.Next() {
		var dailyRecord CompaniesPriceData
		err := records.Scan(&dailyRecord.DateVal, &dailyRecord.CloseVal)
		if err != nil {
			fmt.Println(err.Error(), "Error scanning record ")
		}
		dailyPriceRecords = append(dailyPriceRecords, dailyRecord)
	}
	return dailyPriceRecords
}

/* Fetch All Price Data for a given company */
func FetchATHForCompaniesDB(db *sql.DB) ([]CompaniesPriceData, error) {
	var companyATHRecords []CompaniesPriceData
	records, err := db.Query("SELECT COMPANY_ID, MAX(CLOSE_VAL) AS ATH FROM COMPANIES_PRICE_DATA GROUP BY COMPANY_ID ")
	if err != nil {
		return companyATHRecords, err
	}
	defer records.Close()
	for records.Next() {
		var dailyRecord CompaniesPriceData
		err := records.Scan(&dailyRecord.CompanyId, &dailyRecord.CloseVal)
		if err != nil {
			return companyATHRecords, err
		}
		companyATHRecords = append(companyATHRecords, dailyRecord)
	}
	return companyATHRecords, nil
}

/* Fetch Latest Price Data for a given company */
func FetchCompaniesLatestPriceDataDB(companyid string, db *sql.DB) (CompaniesPriceData, error) {
	var dailyPriceRecords CompaniesPriceData
	records, err := db.Query("SELECT DATE_VAL, CLOSE_VAL FROM COMPANIES_PRICE_DATA WHERE COMPANY_ID = $1 AND CLOSE_VAL != 0 ORDER BY DATE_VAL DESC LIMIT 1", companyid)
	if err != nil {
		return dailyPriceRecords, err
	}
	defer records.Close()
	for records.Next() {
		errScan := records.Scan(&dailyPriceRecords.DateVal, &dailyPriceRecords.CloseVal)
		if errScan != nil {
			return dailyPriceRecords, errScan
		}
	}

	return dailyPriceRecords, nil
}

/* Fetch Unique Company Ids */
func FetchCompaniesDB(db *sql.DB) ([]Company, error) {
	var companies []Company
	records, err := db.Query("SELECT COMPANY_ID, COMPANY_NAME, LOAD_DATE FROM COMPANIES ")
	if err != nil {
		return companies, err
	}
	defer records.Close()
	for records.Next() {
		var company Company
		err := records.Scan(&company.CompanyId, &company.CompanyName, &company.LoadDate)
		if err != nil {
			return companies, err
		}
		companies = append(companies, company)
	}
	return companies, nil
}

func UpdateLoadDate(db *sql.DB, companyId string, loadDate time.Time) {
	db.Exec("UPDATE COMPANIES SET LOAD_DATE = $1 WHERE COMPANY_ID = $2 ", loadDate, companyId)
}

func LoadCompaniesMasterListDB(companiesMasterList []Company, appUtil *util.AppUtil) error {

	/* Loop and Insert Records */
	for k, v := range companiesMasterList {
		_, err := appUtil.Db.Exec("INSERT INTO COMPANIES(COMPANY_ID, COMPANY_NAME, LOAD_DATE) VALUES($1, $2, $3) "+
			" ON CONFLICT(COMPANY_ID) DO NOTHING ",
			v.CompanyId, v.CompanyName, v.LoadDate)

		/* Ignoring data errors for now */
		if err != nil {
			appUtil.AppLogger.Println(err.Error(), " Error while inserting Record : ", k)
			return err
		}
	}
	appUtil.AppLogger.Println("Inserted Companies Master List")
	return nil
}

func AddUserDB(user User, db *sql.DB) error {
	_, err := db.Exec("INSERT INTO USERS(USER_ID, START_DATE, TARGET_AMOUNT, PASSWORD) VALUES($1, $2, $3, $4) ",
		user.UserId, user.StartDate, user.TargetAmount, user.Password)
	if err != nil {
		return err
	}
	return nil
}

func AddUserHoldingsDB(userHoldings HoldingsInputJson, db *sql.DB) error {
	userId := userHoldings.UserID
	/* Add Tracked assets */
	for _, company := range userHoldings.Holdings {
		buyPrice, parseErr := strconv.ParseFloat(company.BuyPrice, 64)
		if parseErr != nil {
			return parseErr
		}

		_, err := db.Exec("INSERT INTO USER_HOLDINGS(USER_ID, COMPANY_ID, QUANTITY, BUY_DATE, BUY_PRICE) VALUES($1, $2, $3, $4, $5) ",
			userId, company.Companyid, company.Quantity, company.BuyDate, buyPrice)
		if err != nil {
			return err
		}
	}

	/* Add Non Tracked assets */
	for _, security := range userHoldings.HoldingsNT {
		buyValue, bvParseErr := strconv.ParseFloat(security.BuyValue, 64)
		if bvParseErr != nil {
			return bvParseErr
		}
		currentValue, cvParseErr := strconv.ParseFloat(security.CurrentValue, 64)
		if cvParseErr != nil {
			return cvParseErr
		}
		interestRate, irParseErr := strconv.ParseFloat(security.InterestRate, 64)
		if irParseErr != nil {
			return irParseErr
		}

		_, err := db.Exec("INSERT INTO USER_HOLDINGS_NT(USER_ID, SECURITY_ID, BUY_DATE, BUY_VALUE, CURRENT_VALUE, INTEREST_RATE) VALUES($1, $2, $3, $4, $5, $6) ",
			userId, security.SecurityId, security.BuyDate, buyValue, currentValue, interestRate)
		if err != nil {
			return err
		}
	}
	return nil
}

func FetchUniqueUsersDB(db *sql.DB) ([]User, error) {
	var users []User
	records, err := db.Query("SELECT USER_ID FROM USERS ")
	if err != nil {
		return users, err
	}
	defer records.Close()
	for records.Next() {
		var user User
		err := records.Scan(&user.UserId)
		if err != nil {
			return users, err
		}
		users = append(users, user)
	}
	return users, nil
}

func GetUserHoldingsDB(userid string, db *sql.DB) (HoldingsOutputJson, error) {
	var holdingsOutputJson HoldingsOutputJson
	holdingsOutputJson.UserID = userid

	/* Tracked Data */
	records, err := db.Query("SELECT HOLDINGS.USER_ID, HOLDINGS.COMPANY_ID, HOLDINGS.QUANTITY, HOLDINGS.BUY_DATE, HOLDINGS.BUY_PRICE, COMPANIES.COMPANY_NAME "+
		"FROM USERS USERS, USER_HOLDINGS HOLDINGS, COMPANIES COMPANIES "+
		"WHERE USERS.USER_ID = HOLDINGS.USER_ID AND HOLDINGS.COMPANY_ID = COMPANIES.COMPANY_ID AND USERS.USER_ID = $1 ORDER BY BUY_DATE", userid)
	if err != nil {
		return holdingsOutputJson, err
	}
	defer records.Close()

	for records.Next() {
		var holdings Holdings
		var userid string
		err := records.Scan(&userid, &holdings.Companyid, &holdings.Quantity, &holdings.BuyDate, &holdings.BuyPrice, &holdings.CompanyName)
		if err != nil {
			return holdingsOutputJson, err
		}
		holdingsOutputJson.Holdings = append(holdingsOutputJson.Holdings, holdings)
	}

	/* Non Tracked Data*/
	recordsNT, errNT := db.Query("SELECT HOLDINGS_NT.USER_ID, HOLDINGS_NT.SECURITY_ID, HOLDINGS_NT.BUY_VALUE, HOLDINGS_NT.BUY_DATE, HOLDINGS_NT.CURRENT_VALUE FROM USERS USERS, USER_HOLDINGS_NT HOLDINGS_NT "+
		"WHERE USERS.USER_ID = HOLDINGS_NT.USER_ID AND USERS.USER_ID = $1", userid)
	if errNT != nil {
		return holdingsOutputJson, errNT
	}
	defer recordsNT.Close()

	for recordsNT.Next() {
		var holdingsNT HoldingsNonTracked
		var userid string
		err := recordsNT.Scan(&userid, &holdingsNT.SecurityId, &holdingsNT.BuyValue, &holdingsNT.BuyDate, &holdingsNT.CurrentValue)
		if err != nil {
			return holdingsOutputJson, err
		}
		holdingsOutputJson.HoldingsNT = append(holdingsOutputJson.HoldingsNT, holdingsNT)
	}

	return holdingsOutputJson, nil
}

func AddModelPortfolioDB(userHoldings ModelPortfolio, db *sql.DB) error {
	userId := userHoldings.UserID
	for _, security := range userHoldings.Securities {
		reasonablePrice, parseErr := strconv.ParseFloat(security.ReasonablePrice, 64)
		if parseErr != nil {
			return parseErr
		}

		expAlloc, parseErr := strconv.ParseFloat(security.ExpectedAllocation, 64)
		if parseErr != nil {
			return parseErr
		}

		_, err := db.Exec("INSERT INTO USER_MODEL_PF(USER_ID, SECURITY_ID, REASONABLE_PRICE, EXP_ALLOC) VALUES($1, $2, $3, $4) "+
			" ON CONFLICT(USER_ID, SECURITY_ID) DO UPDATE SET REASONABLE_PRICE = excluded.REASONABLE_PRICE, EXP_ALLOC =  excluded.EXP_ALLOC ",
			userId, security.Securityid, reasonablePrice, expAlloc)
		if err != nil {
			return err
		}
	}
	return nil
}

func GetModelPortfolioDB(userid string, db *sql.DB) (ModelPortfolio, error) {
	var modelPf ModelPortfolio
	modelPf.UserID = userid

	records, err := db.Query("SELECT SECURITY_ID, REASONABLE_PRICE, EXP_ALLOC FROM USER_MODEL_PF  "+
		"WHERE USER_ID = $1", userid)
	if err != nil {
		return modelPf, err
	}
	defer records.Close()

	for records.Next() {
		var security Securities
		err := records.Scan(&security.Securityid, &security.ReasonablePrice, &security.ExpectedAllocation)
		if err != nil {
			return modelPf, err
		}
		modelPf.Securities = append(modelPf.Securities, security)
	}

	return modelPf, nil
}

func GetTargetAmountDB(userid string, db *sql.DB) (float64, error) {
	var targetAmount float64
	records, err := db.Query("SELECT TARGET_AMOUNT FROM USERS WHERE USER_ID = $1 ", userid)
	if err != nil {
		return targetAmount, err
	}
	defer records.Close()
	for records.Next() {
		errRead := records.Scan(&targetAmount)
		if errRead != nil {
			return targetAmount, errRead
		}
	}
	return targetAmount, nil
}

func GetPassword(userid string, db *sql.DB) (string, error) {
	var password string
	records, err := db.Query("SELECT PASSWORD FROM USERS WHERE USER_ID = $1 ", userid)
	if err != nil {
		return password, err
	}
	defer records.Close()
	for records.Next() {
		errRead := records.Scan(&password)
		if errRead != nil {
			return password, errRead
		}
	}
	return password, nil
}
