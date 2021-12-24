package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/robfig/cron/v3"
	"github.com/spf13/viper"
	"github.com/vijayyogesh/PortfolioApis/controllers"
	"github.com/vijayyogesh/PortfolioApis/processor"

	_ "github.com/lib/pq"
)

var db *sql.DB
var appC *controllers.AppController
var Logger *log.Logger

type Config struct {
	DBHost     string `mapstructure:"DB_HOST"`
	DBDriver   string `mapstructure:"DB_DRIVER"`
	DBUser     string `mapstructure:"DB_USER"`
	DBPassword string `mapstructure:"DB_PASSWORD"`
	DBName     string `mapstructure:"DB_NAME"`
	DBPort     int    `mapstructure:"DB_PORT"`
}

func main() {
	/*fmt.Println("In main - start tme: " + time.Now().String())
	processor.FetchAndUpdatePrices(db)
	fmt.Println("In main - end tme: " + time.Now().String()) */

	/*fmt.Println("In main - start tme: " + time.Now().String())
	processor.FetchAndUpdateCompaniesMasterList(db)
	fmt.Println("In main - end tme: " + time.Now().String()) */

	/*http.Handle("/PortfolioApis/refresh", *appC)
	http.ListenAndServe(":3000", nil)*/

	http.Handle("/PortfolioApis/login", *appC)
	http.Handle("/PortfolioApis/updateprices", *appC)
	http.Handle("/PortfolioApis/updatemasterlist", *appC)
	http.Handle("/PortfolioApis/adduser", *appC)
	http.Handle("/PortfolioApis/adduserholdings", *appC)
	http.Handle("/PortfolioApis/getuserholdings", *appC)
	http.Handle("/PortfolioApis/addmodelportfolio", *appC)
	http.Handle("/PortfolioApis/getmodelportfolio", *appC)
	http.Handle("/PortfolioApis/syncportfolio", *appC)
	http.Handle("/PortfolioApis/fetchnetworthoverperiod", *appC)
	http.ListenAndServe(":3000", nil)

}

func init() {
	//db = data.SetupDB()
	config, _ := LoadConfig(".")
	db = SetupDB(config)

	// If the file doesn't exist, create it or append to the file
	file, err := os.OpenFile("PortfolioApiLog.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		panic(err)
	}
	Logger = log.New(file, "PortfolioApi : ", log.Ldate|log.Ltime|log.Lshortfile)

	appC = controllers.NewAppController(db, Logger)
	appC.AppLogger.SetOutput(file)
	//startCronJobs()
	appC.AppLogger.Println("Completed init")
}

func LoadConfig(path string) (config Config, err error) {
	viper.AddConfigPath(path)
	viper.SetConfigName("app")
	viper.SetConfigType("env")

	viper.AutomaticEnv()

	err = viper.ReadInConfig()
	if err != nil {
		return
	}

	err = viper.Unmarshal(&config)
	return
}

func startCronJobs() {
	appC.AppLogger.Println("Starting Cron Jobs")
	cronJob := cron.New()
	cronJob.AddFunc("@hourly", func() {
		processor.FetchAndUpdatePrices(db)
	})
	cronJob.Start()
	appC.AppLogger.Println("Completed Cron Jobs")
}

/* Setup DB */
func SetupDB(config Config) *sql.DB {
	fmt.Println("In setupDB")
	dbinfo := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", config.DBHost, config.DBPort, config.DBUser, config.DBPassword, config.DBName)
	db, err := sql.Open(config.DBDriver, dbinfo)
	if err != nil {
		panic("Error while initializing DB")
	}
	db.SetMaxOpenConns(20)
	fmt.Println("Completed setupDB")
	return db
}
