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

	/* Auth/JWT */
	AppJWTAudience = "ApiUsers"
	AppJWTIssuer   = "PortfolioApisApp"

	/* AppFile */
	AppDataDir              = "C:\\Users\\vijay\\root\\development\\data\\"
	AppDataMasterUrl        = "https://www1.nseindia.com/content/indices/ind_nifty500list.csv"
	AppDataMasterFile       = "TOP500.csv"
	AppDataPricesFileSuffix = ".NS.csv"
	AppDataPricesUrl        = "https://query1.finance.yahoo.com/v7/finance/download/"
	AppDataPricesUrlSuffix  = ".NS?period1=%s&period2=%s&interval=1d&events=history&includeAdjustedClose=true"

	/* Error Codes */
	AppErrUserUnauthorized = "E100: User is Unauthorized!!. Please check Token value."
	AppErrJWTAuth          = "E101: Error encountered while authenticating user"
	AppErrUserIdInvalid    = "E102: Please provide a valid UserId."

	AppErrMasterList     = "E200: Error encountered while loading companies master list"
	AppSuccessMasterList = "Master companies list loaded successfully"
)
