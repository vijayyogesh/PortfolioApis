package controllers

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/vijayyogesh/PortfolioApis/auth"
	"github.com/vijayyogesh/PortfolioApis/constants"
	"github.com/vijayyogesh/PortfolioApis/data"
	"github.com/vijayyogesh/PortfolioApis/processor"
	"github.com/vijayyogesh/PortfolioApis/util"
)

type AppController struct {
	AppUtil *util.AppUtil
}

func NewAppController(apputil *util.AppUtil) *AppController {
	return &AppController{
		AppUtil: apputil,
	}
}

/* Initial Handler for all routes/endpoints */
func (appC AppController) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	appC.AppUtil.AppLogger.Println("Starting ServeHTTP")

	reqBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		handlePayloadError(err, appC, w)
	} else {
		/* Check UserId in Payload */
		userId, err := getUserId(reqBody, appC)

		if userId != "" && err == nil {
			/* Handle Login */
			if (r.URL.Path == constants.AppRouteLogin) && (r.Method == http.MethodPost) {
				msg, err := auth.GetJWT(userId)
				if err != nil {
					appC.AppUtil.AppLogger.Println("Error encountered while generating JWT")
					appC.AppUtil.AppLogger.Println(err)
					msg = constants.AppErrJWTAuth
				}
				json.NewEncoder(w).Encode(msg)

			} else {
				/* Authenticate Token when already logged In */
				if auth.AuthenticateToken(r, userId) {
					ProcessAppRequests(w, r, appC, reqBody)
				} else {
					json.NewEncoder(w).Encode(constants.AppErrUserUnauthorized)
				}
			}
		} else {
			appC.AppUtil.AppLogger.Println("Invalid request")
			json.NewEncoder(w).Encode(constants.AppErrUserIdInvalid)
		}
	}

	appC.AppUtil.AppLogger.Println("Completed ServeHTTP")
}

/* Get UserId from request Payload */
func getUserId(reqBody []byte, appC AppController) (string, error) {
	var user data.User
	err := json.Unmarshal(reqBody, &user)
	if err != nil {
		appC.AppUtil.AppLogger.Println(err)
		return "", err
	}
	return user.UserId, err
}

/* Process App routes post JWT authentication */
func ProcessAppRequests(w http.ResponseWriter, r *http.Request, appC AppController, payload []byte) {

	/* Commented as updatePrices is taken care by Cron Job */
	/*if (r.URL.Path == constants.AppRouteUpdatePrices) && (r.Method == http.MethodPost) {
		msg := processor.FetchAndUpdatePrices(appC.AppUtil.Db)
		json.NewEncoder(w).Encode(msg)
	} */

	processor.InitProcessor(appC.AppUtil)

	if (r.URL.Path == constants.AppRouteUpdateMasterList) && (r.Method == http.MethodPost) {
		/* Route to update/refresh master list of companies */
		msg := processor.FetchAndUpdateCompaniesMasterList()
		json.NewEncoder(w).Encode(msg)
	} else if (r.URL.Path == constants.AppRouteAddUser) && (r.Method == http.MethodPost) {
		/* Route to add new user into system */
		msg := processor.AddUser(payload, appC.AppUtil.Db)
		json.NewEncoder(w).Encode(msg)
	} else if (r.URL.Path == constants.AppRouteAddUserHoldings) && (r.Method == http.MethodPost) {
		/* Route to add user holdings */
		msg := processor.AddUserHoldings(payload, appC.AppUtil.Db)
		json.NewEncoder(w).Encode(msg)
	} else if (r.URL.Path == constants.AppRouteGetUserHoldings) && (r.Method == http.MethodPost) {
		/* Route to fetch User Holdings */
		msg := processor.GetUserHoldings(payload, appC.AppUtil.Db)
		json.NewEncoder(w).Encode(msg)
	} else if (r.URL.Path == constants.AppRouteAddModelPf) && (r.Method == http.MethodPost) {
		/* Route to Add Model Portfolio */
		msg := processor.AddModelPortfolio(payload, appC.AppUtil.Db)
		json.NewEncoder(w).Encode(msg)
	} else if (r.URL.Path == constants.AppRouteGetModelPf) && (r.Method == http.MethodPost) {
		/* Route to fetch Model Portfolio */
		msg := processor.GetModelPortfolio(payload, appC.AppUtil.Db)
		json.NewEncoder(w).Encode(msg)
	} else if (r.URL.Path == constants.AppRouteSyncPf) && (r.Method == http.MethodPost) {
		/* Route to sync Model Pf with actual Pf */
		msg := processor.GetPortfolioModelSync(payload, appC.AppUtil.Db)
		json.NewEncoder(w).Encode(msg)
	} else if (r.URL.Path == constants.AppRouteNWPeriod) && (r.Method == http.MethodPost) {
		/* Route to display NetWorth over a timeframe */
		msg := processor.FetchNetWorthOverPeriods(payload, appC.AppUtil.Db)
		json.NewEncoder(w).Encode(msg)
	}
}

func handlePayloadError(err error, appC AppController, w http.ResponseWriter) {
	appC.AppUtil.AppLogger.Println(err)
	json.NewEncoder(w).Encode("Error in Paylod Data. Please check !!")
}
