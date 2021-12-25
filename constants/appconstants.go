package constants

const (
	/* Logger Constants */
	AppLoggerFile   string = "PortfolioApiLog.txt"
	AppLoggerPrefix string = "Portfolio Api : "

	/* DB Constants */
	AppDBFmtString string = "host=%s port=%d user=%s password=%s dbname=%s sslmode=disable"
	AppDBMaxConn   int    = 20

	/* Env/Config Constants */
	AppEnvName string = "app"
	AppEnvType string = "env"
	AppEnvPath string = "."

	/* EndPoint/Route Paths */
	AppRouteLogin            string = "/PortfolioApis/login"
	AppRouteUpdatePrices     string = "/PortfolioApis/updateprices"
	AppRouteUpdateMasterList string = "/PortfolioApis/updatemasterlist"
	AppRouteAddUser          string = "/PortfolioApis/adduser"
	AppRouteAddUserHoldings  string = "/PortfolioApis/adduserholdings"
	AppRouteGetUserHoldings  string = "/PortfolioApis/getuserholdings"
	AppRouteAddModelPf       string = "/PortfolioApis/addmodelportfolio"
	AppRouteGetModelPf       string = "/PortfolioApis/getmodelportfolio"
	AppRouteSyncPf           string = "/PortfolioApis/syncportfolio"
	AppRouteNWPeriod         string = "/PortfolioApis/fetchnetworthoverperiod"
)
