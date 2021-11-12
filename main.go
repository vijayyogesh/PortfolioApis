package main

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/vijayyogesh/PortfolioApis/data"
	"github.com/vijayyogesh/PortfolioApis/processor"

	_ "github.com/lib/pq"
)

var db *sql.DB

func main() {
	/*fmt.Println("In main - start tme: " + time.Now().String())
	processor.LoadPriceData(db)
	fmt.Println("In main - end tme: " + time.Now().String()) */

	/*fmt.Println("In main - start tme: " + time.Now().String())
	processor.FetchCompaniesPrice("HINDUNILVR", db)
	fmt.Println("In main - end tme: " + time.Now().String())
	fmt.Println("In main - start tme: " + time.Now().String())
	processor.FetchCompaniesPrice("HINDUNILVR", db)
	fmt.Println("In main - end tme: " + time.Now().String()) */

	fmt.Println("In main - start tme: " + time.Now().String())
	err := processor.DownloadDataFile("TIMKEN")
	if nil != err {
		panic(err)
	}
	fmt.Println("In main - end tme: " + time.Now().String())
}

func init() {
	fmt.Println("In init")
	db = data.SetupDB()
}
