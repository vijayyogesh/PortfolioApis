package data

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
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
	UserId    string
	StartDate time.Time
}

type HoldingsInputJson struct {
	UserID   string `json:"userId"`
	Holdings []struct {
		Companyid string `json:"companyid"`
		Quantity  string `json:"quantity"`
		BuyDate   string `json:"buyDate"`
	} `json:"Holdings"`
}

const (
	DB_USER     = "postgres"
	DB_PASSWORD = "phorrj"
	DB_NAME     = "PortfolioApis"
)

/* Setup DB */
func SetupDB() *sql.DB {
	dbinfo := fmt.Sprintf("user=%s password=%s dbname=%s sslmode=disable", DB_USER, DB_PASSWORD, DB_NAME)
	db, err := sql.Open("postgres", dbinfo)
	db.SetMaxOpenConns(20)
	checkErr(err)
	return db
}

/* Check critcal errors */
func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}

func LoadPriceDataDB(dailyPriceRecords []CompaniesPriceData, db *sql.DB) {
	valueStrings := make([]string, 0, len(dailyPriceRecords))
	valueArgs := make([]interface{}, 0, len(dailyPriceRecords)*6)
	fmt.Println(len(dailyPriceRecords))

	/* Loop and Bulk Insert Records */
	for k, v := range dailyPriceRecords {
		valueStrings = append(valueStrings, fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, $%d)", k*6+1, k*6+2, k*6+3, k*6+4, k*6+5, k*6+6))
		valueArgs = append(valueArgs, v.CompanyId, v.OpenVal, v.HighVal, v.LowVal, v.CloseVal, v.DateVal)
	}
	stmt := fmt.Sprintf("INSERT INTO COMPANIES_PRICE_DATA(COMPANY_ID, OPEN_VAL,HIGH_VAL, LOW_VAL, CLOSE_VAL, DATE_VAL) VALUES %s "+
		" ON CONFLICT(COMPANY_ID, DATE_VAL) DO UPDATE SET CLOSE_VAL = excluded.CLOSE_VAL ", strings.Join(valueStrings, ","))

	_, err := db.Exec(stmt, valueArgs...)

	/* Ignoring data errors for now */
	if err != nil {
		fmt.Println(err.Error(), " Error while inserting Record : ")
	}

	fmt.Println("Inserted")
}

/* Fetch Price Data for a given company */
func FetchCompaniesPriceDataDB(companyid string, db *sql.DB) []CompaniesPriceData {
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

/* Fetch Unique Company Ids */
func FetchCompaniesDB(db *sql.DB) []Company {
	var companies []Company
	records, err := db.Query("SELECT COMPANY_ID, LOAD_DATE FROM COMPANIES ")
	if err != nil {
		panic(err.Error())
	}
	defer records.Close()
	for records.Next() {
		var company Company
		err := records.Scan(&company.CompanyId, &company.LoadDate)
		if err != nil {
			fmt.Println(err.Error(), "Error scanning record ")
		}
		companies = append(companies, company)
	}
	return companies
}

func UpdateLoadDate(db *sql.DB, companyId string, loadDate time.Time) {
	fmt.Println("comp id - ", companyId)
	db.Exec("UPDATE COMPANIES SET LOAD_DATE = $1 WHERE COMPANY_ID = $2 ", loadDate, companyId)
}

func LoadCompaniesMasterListDB(companiesMasterList []Company, db *sql.DB) {

	/* Loop and Insert Records */
	for k, v := range companiesMasterList {
		_, err := db.Exec("INSERT INTO COMPANIES(COMPANY_ID, COMPANY_NAME) VALUES($1, $2) "+
			" ON CONFLICT(COMPANY_ID) DO NOTHING ",
			v.CompanyId, v.CompanyName)

		/* Ignoring data errors for now */
		if err != nil {
			fmt.Println(err.Error(), " Error while inserting Record : ", k)
		}
	}
	fmt.Println("Inserted Companies Master List")
}

func AddUserDB(user User, db *sql.DB) error {
	_, err := db.Exec("INSERT INTO USERS(USER_ID, START_DATE) VALUES($1, $2) ", user.UserId, user.StartDate)
	if err != nil {
		return err
	}
	return nil
}
