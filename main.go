package main

import (
	"fmt"
	"net/http"

	"github.com/robfig/cron/v3"
	"github.com/vijayyogesh/PortfolioApis/constants"
	"github.com/vijayyogesh/PortfolioApis/controllers"
	"github.com/vijayyogesh/PortfolioApis/processor"
	"github.com/vijayyogesh/PortfolioApis/util"

	_ "github.com/lib/pq"
)

var appC *controllers.AppController
var appUtil *util.AppUtil

type Config struct {
	DBHost     string `mapstructure:"DB_HOST"`
	DBDriver   string `mapstructure:"DB_DRIVER"`
	DBUser     string `mapstructure:"DB_USER"`
	DBPassword string `mapstructure:"DB_PASSWORD"`
	DBName     string `mapstructure:"DB_NAME"`
	DBPort     int    `mapstructure:"DB_PORT"`
}

func main() {
	http.Handle(constants.AppRouteRegister, *appC)
	http.Handle(constants.AppRouteLogin, *appC)
	http.Handle(constants.AppRouteUpdateMasterList, *appC)
	http.Handle(constants.AppRouteUpdatePrices, *appC)
	http.Handle(constants.AppRouteAddUser, *appC)
	http.Handle(constants.AppRouteAddUserHoldings, *appC)
	http.Handle(constants.AppRouteGetUserHoldings, *appC)
	http.Handle(constants.AppRouteAddModelPf, *appC)
	http.Handle(constants.AppRouteGetModelPf, *appC)
	http.Handle(constants.AppRouteSyncPf, *appC)
	http.Handle(constants.AppRouteNWPeriod, *appC)
	http.Handle(constants.AppRouteUpdateSelectedCompanies, *appC)
	http.Handle(constants.AppRouteFetchAllCompanies, *appC)
	http.Handle(constants.AppRouteCalculateReturn, *appC)
	http.Handle(constants.AppRouteCalculateIndexSIPReturn, *appC)
	http.Handle(constants.AppRouteCalculateATHforPF, *appC)

	appUtil.AppLogger.Println("----- STARTED PORTFOLIO APIS -----")

	appUtil.AppLogger.Println("Listening and Serving In Port = ", appUtil.Config.APPPort)
	http.ListenAndServe(fmt.Sprintf(":%d", appUtil.Config.APPPort), nil)
}

func init() {
	/* Initialize all global members */
	appUtil = util.NewAppUtil()

	/* Initialize Controller */
	appC = controllers.NewAppController(appUtil)

	processor.InitProcessor(appC.AppUtil)

	/* Start scheduled jobs */
	startCronJobs()
}

func startCronJobs() {
	cronJob := cron.New()
	cronJob.AddFunc("@hourly", func() {
		msg := processor.FetchAndUpdatePrices(appUtil.Db)
		appUtil.AppLogger.Println(msg)
	})
	cronJob.Start()
	appUtil.AppLogger.Println("Scheduled Cron Jobs")
}
