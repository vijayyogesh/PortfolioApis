package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"

	"github.com/robfig/cron/v3"
	"github.com/vijayyogesh/PortfolioApis/controllers"
	"github.com/vijayyogesh/PortfolioApis/data"
	"github.com/vijayyogesh/PortfolioApis/processor"

	_ "github.com/lib/pq"
)

var db *sql.DB
var appC *controllers.AppController
var Logger *log.Logger

func main() {
	/*fmt.Println("In main - start tme: " + time.Now().String())
	processor.FetchAndUpdatePrices(db)
	fmt.Println("In main - end tme: " + time.Now().String()) */

	/*fmt.Println("In main - start tme: " + time.Now().String())
	processor.FetchAndUpdateCompaniesMasterList(db)
	fmt.Println("In main - end tme: " + time.Now().String()) */

	/*http.Handle("/PortfolioApis/refresh", *appC)
	http.ListenAndServe(":3000", nil)*/

	http.Handle("/PortfolioApis/updateprices", *appC)
	http.Handle("/PortfolioApis/updatemasterlist", *appC)
	http.Handle("/PortfolioApis/adduser", *appC)
	http.Handle("/PortfolioApis/adduserholdings", *appC)
	http.Handle("/PortfolioApis/getuserholdings", *appC)
	http.Handle("/PortfolioApis/addmodelportfolio", *appC)
	http.Handle("/PortfolioApis/getmodelportfolio", *appC)
	http.Handle("/PortfolioApis/syncportfolio", *appC)
	http.ListenAndServe(":3000", nil)

}

func init() {
	db = data.SetupDB()

	// If the file doesn't exist, create it or append to the file
	file, err := os.OpenFile("PortfolioApiLog.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		panic(err)
	}
	Logger = log.New(file, "PortfolioApi : ", log.Ldate|log.Ltime|log.Lshortfile)

	appC = controllers.NewAppController(db, Logger)
	appC.AppLogger.SetOutput(file)
	startCronJobs()
	appC.AppLogger.Println("Completed init")
}

func startCronJobs() {
	appC.AppLogger.Println("Starting Cron Jobs")
	cronJob := cron.New()
	cronJob.AddFunc("*/1 * * * *", processor.TestCron)
	cronJob.Start()
	//fmt.Println(c.Entries())
}
