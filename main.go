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

	appUtil.AppLogger.Println("Listening and Serving In Port = ", appUtil.Config.APPPort)

	http.ListenAndServe(fmt.Sprintf(":%d", appUtil.Config.APPPort), nil)
}

func init() {
	/* Initialize all global members */
	appUtil = util.NewAppUtil()

	/* Initialize Controller */
	appC = controllers.NewAppController(appUtil)
}

func startCronJobs() {
	appUtil.AppLogger.Println("Starting Cron Jobs")
	cronJob := cron.New()
	cronJob.AddFunc("@hourly", func() {
		processor.FetchAndUpdatePrices(appUtil.Db)
	})
	cronJob.Start()
	appUtil.AppLogger.Println("Completed Cron Jobs")
}
