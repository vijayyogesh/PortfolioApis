package constants

const (
	/* Logger Constants */
	AppLoggerFile   string = "PortfolioApiLog.txt"
	AppLoggerPrefix string = "Portfolio Api : "

	/* DB Constants */
	AppDBFmtString string = "host=%s port=%d user=%s password=%s dbname=%s sslmode=disable"
	AppDBMaxConn   int    = 20

	/* Return constants */
	ReturnBaseValue = 10

	/* Env/Config Constants */
	AppEnvName string = "app"
	AppEnvType string = "env"
	AppEnvPath string = "."

	/* EndPoint/Route Paths */
	AppRouteRegister                string = "/PortfolioApis/register"
	AppRouteLogin                   string = "/PortfolioApis/login"
	AppRouteUpdatePrices            string = "/PortfolioApis/updateprices"
	AppRouteUpdateMasterList        string = "/PortfolioApis/updatemasterlist"
	AppRouteAddUser                 string = "/PortfolioApis/adduser"
	AppRouteAddUserHoldings         string = "/PortfolioApis/adduserholdings"
	AppRouteGetUserHoldings         string = "/PortfolioApis/getuserholdings"
	AppRouteAddModelPf              string = "/PortfolioApis/addmodelportfolio"
	AppRouteGetModelPf              string = "/PortfolioApis/getmodelportfolio"
	AppRouteSyncPf                  string = "/PortfolioApis/syncportfolio"
	AppRouteNWPeriod                string = "/PortfolioApis/fetchnetworthoverperiod"
	AppRouteUpdateSelectedCompanies string = "/PortfolioApis/updateselectedcompanies"
	AppRouteFetchAllCompanies       string = "/PortfolioApis/fetchallcompanies"
	AppRouteCalculateReturn         string = "/PortfolioApis/calculatereturn"

	/* Auth/JWT */
	AppJWTAudience = "ApiUsers"
	AppJWTIssuer   = "PortfolioApisApp"

	/* AppFile */
	AppDataMasterUrl        = "https://www1.nseindia.com/content/indices/ind_nifty500list.csv"
	AppDataMasterFile       = "TOP500.csv"
	AppDataPricesFileSuffix = ".NS.csv"
	AppDataPricesUrl        = "https://query1.finance.yahoo.com/v7/finance/download/"
	AppDataPricesUrlSuffix  = ".NS?period1=%s&period2=%s&interval=1d&events=history&includeAdjustedClose=true"

	/* Below 4 constants added to support Index data from Yahoo finance*/
	AppDataCsv                      = ".csv"
	AppDataBenchmarkNSE             = "NSEI"
	AppDataBenchmarkPricesUrlSuffix = "?period1=%s&period2=%s&interval=1d&events=history&includeAdjustedClose=true"
	AppDataBenchmarkAppenderText    = "^"

	AppDataPrefixMF           = "0P00"
	AppDataPricesFileSuffixMF = ".BO.csv"
	AppDataPricesUrlSuffixMF  = ".BO?period1=%s&period2=%s&interval=1d&events=history&includeAdjustedClose=true"

	/* Error Codes */
	AppErrUserUnauthorized  = "E100: User is Unauthorized!!. Please check Token value."
	AppErrJWTAuth           = "E101: Error encountered while authenticating user"
	AppErrUserIdInvalid     = "E102: Please provide a valid UserId."
	AppErrInvalidPassword   = "E103: Please provide a valid Password."
	AppErrIncorrectPassword = "E104: Incorrect credentials provided."

	AppErrMasterList     = "E200: Error encountered while loading companies master list"
	AppSuccessMasterList = "Master companies list loaded successfully!!"

	AppErrAddUser     = "E201: Error while adding new User"
	AppSuccessAddUser = "User Added successfully!!"

	AppErrAddUserHoldings        = "E202: Error while adding User Holdings"
	AppErrAddUserHoldingsInvalid = "E203: Invalid UserId provided"
	AppSuccessAddUserHoldings    = "User Holdings Added successfully!!"

	AppErrGetUserHoldings = "E204: Error while fetching User Holdings"

	AppErrAddModelPf            = "E205: Error while adding Model Portfolio"
	AppErrAddModelPfInvalidUser = "E206: Invalid UserId provided"
	AppSuccessAddModelPf        = "Model Portfolio Added successfully!!"

	AppErrGetModelPf = "E207: Error while fetching Model Portfolio"

	AppErrGetModelPfSync     = "E208: Error while syncing Model Portfolio"
	AppErrFetchNWOverPeriods = "E209: Error while calculating Networth over periods"

	AppErrUpdateSelectedCompaniesPrice     = "E210: Error while Updating Prices for selected companies"
	AppSuccessUpdateSelectedCompaniesPrice = "Prices updated successfully for selected companies !!"

	AppErrFetchAllCompanies     = "E211: Error while fetching all companies"
	AppSuccessFetchAllCompanies = "Fetched all companies !!"

	AppErrCalculateReturn     = "E211: Error while calculating return"
	AppSuccessCalculateReturn = "Calculated Return Successfuly !!"
)
