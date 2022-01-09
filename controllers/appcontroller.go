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
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Accept, X-Requested-With, remember-me, Authorization, type, token")

	reqBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		handlePayloadError(err, appC, w)
	} else {
		/* Check UserId in Payload */
		user, err := getUser(reqBody, appC)
		userId := user.UserId

		if userId != "" && err == nil {
			/* Handle Register */
			if (r.URL.Path == constants.AppRouteRegister) && (r.Method == http.MethodPost) {
				if user.Password != "" {
					appC.AppUtil.AppLogger.Println("Registering New User")
					shaPasswd := auth.GenerateSHA(user.Password)
					user.Password = shaPasswd
					msg := processor.AddUser(user)
					json.NewEncoder(w).Encode(msg)
				} else {
					json.NewEncoder(w).Encode(constants.AppErrInvalidPassword)
				}
			} else if (r.URL.Path == constants.AppRouteLogin) && (r.Method == http.MethodPost) {
				/* Handle Login */

				/* Validate password */
				shaPassword := auth.GenerateSHA(user.Password)
				user.Password = shaPassword
				isValidPassword := processor.IsValidPassword(user)
				if isValidPassword {
					appC.AppUtil.AppLogger.Println("Password Validated ")
					/* Generate JWT when password is validated */
					msg, err := auth.GetJWT(userId)
					if err != nil {
						appC.AppUtil.AppLogger.Println("Error encountered while generating JWT")
						appC.AppUtil.AppLogger.Println(err)
						msg = constants.AppErrJWTAuth
					}
					appC.AppUtil.AppLogger.Println("Generated JWT for user - " + userId)
					json.NewEncoder(w).Encode(msg)
				} else {
					appC.AppUtil.AppLogger.Println("Invalid password provided ")
					json.NewEncoder(w).Encode(constants.AppErrIncorrectPassword)
				}

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

/* Get User from request Payload */
func getUser(reqBody []byte, appC AppController) (data.User, error) {
	var user data.User
	err := json.Unmarshal(reqBody, &user)
	if err != nil {
		appC.AppUtil.AppLogger.Println(err)
		return user, err
	}
	return user, err
}

/* Process App routes post JWT authentication */
func ProcessAppRequests(w http.ResponseWriter, r *http.Request, appC AppController, payload []byte) {

	processor.InitProcessor(appC.AppUtil)

	/* Commented as updatePrices is taken care by Cron Job */
	/*if (r.URL.Path == constants.AppRouteUpdatePrices) && (r.Method == http.MethodPost) {
		msg := processor.FetchAndUpdatePrices(appC.AppUtil.Db)
		json.NewEncoder(w).Encode(msg)
	} */

	if (r.URL.Path == constants.AppRouteUpdateSelectedCompanies) && (r.Method == http.MethodPost) {
		msg := processor.UpdateSelectedCompanies(payload)
		json.NewEncoder(w).Encode(msg)
	} else if (r.URL.Path == constants.AppRouteUpdateMasterList) && (r.Method == http.MethodPost) {
		/* Route to update/refresh master list of companies */
		msg := processor.FetchAndUpdateCompaniesMasterList()
		json.NewEncoder(w).Encode(msg)
	} else if (r.URL.Path == constants.AppRouteAddUser) && (r.Method == http.MethodPost) {
		/* Route to add new user into system */
		var user data.User
		json.Unmarshal(payload, &user)
		msg := processor.AddUser(user)
		json.NewEncoder(w).Encode(msg)
	} else if (r.URL.Path == constants.AppRouteAddUserHoldings) && (r.Method == http.MethodPost) {
		/* Route to add user holdings */
		msg := processor.AddUserHoldings(payload)
		json.NewEncoder(w).Encode(msg)
	} else if (r.URL.Path == constants.AppRouteGetUserHoldings) && (r.Method == http.MethodPost) {
		/* Route to fetch User Holdings */
		resp, err := processor.GetUserHoldings(payload)
		if err != nil {
			json.NewEncoder(w).Encode(constants.AppErrGetUserHoldings)
		} else {
			json.NewEncoder(w).Encode(resp)
		}
	} else if (r.URL.Path == constants.AppRouteAddModelPf) && (r.Method == http.MethodPost) {
		/* Route to Add Model Portfolio */
		msg := processor.AddModelPortfolio(payload)
		json.NewEncoder(w).Encode(msg)
	} else if (r.URL.Path == constants.AppRouteGetModelPf) && (r.Method == http.MethodPost) {
		/* Route to fetch Model Portfolio */
		resp, err := processor.GetModelPortfolio(payload)
		if err != nil {
			json.NewEncoder(w).Encode(constants.AppErrGetModelPf)
		} else {
			json.NewEncoder(w).Encode(resp)
		}
	} else if (r.URL.Path == constants.AppRouteSyncPf) && (r.Method == http.MethodPost) {
		/* Route to sync Model Pf with actual Pf */
		resp, err := processor.GetPortfolioModelSync(payload)
		if err != nil {
			json.NewEncoder(w).Encode(constants.AppErrGetModelPfSync)
		} else {
			json.NewEncoder(w).Encode(resp)
		}
	} else if (r.URL.Path == constants.AppRouteNWPeriod) && (r.Method == http.MethodPost) {
		/* Route to display NetWorth over a timeframe */
		resp, err := processor.FetchNetWorthOverPeriods(payload)
		if err != nil {
			json.NewEncoder(w).Encode(constants.AppErrFetchNWOverPeriods)
		} else {
			json.NewEncoder(w).Encode(resp)
		}
	}
}

func handlePayloadError(err error, appC AppController, w http.ResponseWriter) {
	appC.AppUtil.AppLogger.Println(err)
	json.NewEncoder(w).Encode("Error in Payload Data. Please check !!")
}
