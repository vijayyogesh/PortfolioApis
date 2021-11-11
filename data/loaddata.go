package data

import (
	"database/sql"
	"fmt"
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

const (
	DB_USER     = "postgres"
	DB_PASSWORD = "phorrj"
	DB_NAME     = "PortfolioApis"
)

/* Setup DB */
func SetupDB() *sql.DB {
	dbinfo := fmt.Sprintf("user=%s password=%s dbname=%s sslmode=disable", DB_USER, DB_PASSWORD, DB_NAME)
	db, err := sql.Open("postgres", dbinfo)
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

	fmt.Println(len(dailyPriceRecords))
	/* Loop and Insert Records */
	for k, v := range dailyPriceRecords {
		_, err := db.Exec("INSERT INTO COMPANIES_PRICE_DATA(COMPANY_ID, OPEN_VAL,HIGH_VAL, LOW_VAL, CLOSE_VAL, DATE_VAL) VALUES($1, $2, $3, $4, $5, $6)",
			v.CompanyId, v.OpenVal, v.HighVal, v.LowVal, v.CloseVal, v.DateVal)

		/* Ignoring data errors for now */
		if err != nil {
			fmt.Println(err.Error(), " Error while inserting Record : ", k)
		}
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
	records, err := db.Query("SELECT COMPANY_ID FROM COMPANIES ")
	if err != nil {
		panic(err.Error())
	}
	for records.Next() {
		var company Company
		err := records.Scan(&company.CompanyId)
		if err != nil {
			fmt.Println(err.Error(), "Error scanning record ")
		}
		companies = append(companies, company)
	}
	return companies
}
