package controllers

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/dgrijalva/jwt-go"
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

var mySigningKey = []byte("UG9ydGZvbGlvQXBpLUtleSM=")

func GetJWT() (string, error) {
	token := jwt.New(jwt.SigningMethodHS256)

	claims := token.Claims.(jwt.MapClaims)

	claims["authorized"] = true
	claims["client"] = "testuser"
	claims["aud"] = "ApiUsers"
	claims["iss"] = "PortfolioApisApp"
	claims["exp"] = time.Now().Add(time.Minute * 60000).Unix()

	tokenString, err := token.SignedString(mySigningKey)

	if err != nil {
		fmt.Println(err.Error())
		return "", err
	}

	return tokenString, nil
}

func AuthenticateToken(r *http.Request) bool {
	if r.Header["Token"] != nil {
		token, err := jwt.Parse(r.Header["Token"][0], func(token *jwt.Token) (interface{}, error) {
			return mySigningKey, nil
		})
		if err != nil {
			fmt.Println("Error while parsing Token")
			fmt.Println(err)
		}

		if token.Valid {
			fmt.Println("Token Authenticated")
			return true
		} else {
			fmt.Println("Invalid Token")
			return false
		}
	}
	fmt.Println("Token Not Found")
	return false
}

func (appC AppController) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	appC.AppUtil.AppLogger.Println("In Serve HTTP")

	if (r.URL.Path == "/PortfolioApis/login") && (r.Method == http.MethodPost) {
		msg, _ := GetJWT()
		json.NewEncoder(w).Encode(msg)
	} else {
		if AuthenticateToken(r) {
			ProcessAppRequests(w, r, appC)
		} else {
			json.NewEncoder(w).Encode("Unauthorized!!")
		}
	}

	appC.AppUtil.AppLogger.Println("Exiting Serve HTTP")
}

func ProcessAppRequests(w http.ResponseWriter, r *http.Request, appC AppController) {
	if (r.URL.Path == "/PortfolioApis/updateprices") && (r.Method == http.MethodPost) {
		msg := processor.FetchAndUpdatePrices(appC.AppUtil.Db)
		json.NewEncoder(w).Encode(msg)
	}
	if (r.URL.Path == "/PortfolioApis/updatemasterlist") && (r.Method == http.MethodPost) {
		msg := processor.FetchAndUpdateCompaniesMasterList(appC.AppUtil.Db)
		json.NewEncoder(w).Encode(msg)
	}
	if (r.URL.Path == "/PortfolioApis/adduser") && (r.Method == http.MethodPost) {
		reqBody, err := ioutil.ReadAll(r.Body)
		if err != nil {
			fmt.Println(err)
		}
		msg := processor.AddUser(reqBody, appC.AppUtil.Db)
		json.NewEncoder(w).Encode(msg)
	} else if (r.URL.Path == "/PortfolioApis/adduserholdings") && (r.Method == http.MethodPost) {
		reqBody, _ := ioutil.ReadAll(r.Body)
		msg := processor.AddUserHoldings(reqBody, appC.AppUtil.Db)
		json.NewEncoder(w).Encode(msg)
	} else if (r.URL.Path == "/PortfolioApis/getuserholdings") && (r.Method == http.MethodPost) {
		reqBody, _ := ioutil.ReadAll(r.Body)
		msg := processor.GetUserHoldings(reqBody, appC.AppUtil.Db)
		json.NewEncoder(w).Encode(msg)
	} else if (r.URL.Path == "/PortfolioApis/addmodelportfolio") && (r.Method == http.MethodPost) {
		reqBody, _ := ioutil.ReadAll(r.Body)
		msg := processor.AddModelPortfolio(reqBody, appC.AppUtil.Db)
		json.NewEncoder(w).Encode(msg)
	} else if (r.URL.Path == "/PortfolioApis/getmodelportfolio") && (r.Method == http.MethodPost) {
		reqBody, _ := ioutil.ReadAll(r.Body)
		msg := processor.GetModelPortfolio(reqBody, appC.AppUtil.Db)
		json.NewEncoder(w).Encode(msg)
	} else if (r.URL.Path == "/PortfolioApis/syncportfolio") && (r.Method == http.MethodPost) {
		reqBody, _ := ioutil.ReadAll(r.Body)
		msg := processor.GetPortfolioModelSync(reqBody, appC.AppUtil.Db)
		json.NewEncoder(w).Encode(msg)
	} else if (r.URL.Path == "/PortfolioApis/fetchnetworthoverperiod") && (r.Method == http.MethodPost) {
		reqBody, _ := ioutil.ReadAll(r.Body)
		msg := processor.FetchNetWorthOverPeriods(reqBody, appC.AppUtil.Db)
		json.NewEncoder(w).Encode(msg)
	}
}
