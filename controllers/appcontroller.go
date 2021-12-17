package controllers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/vijayyogesh/PortfolioApis/processor"
)

type AppController struct {
	db        *sql.DB
	AppLogger *log.Logger
}

func NewAppController(dbRef *sql.DB, logger *log.Logger) *AppController {
	return &AppController{
		db:        dbRef,
		AppLogger: logger,
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
	claims["exp"] = time.Now().Add(time.Minute * 10).Unix()

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
	appC.AppLogger.Println("In Serve HTTP")

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

	appC.AppLogger.Println("Exiting Serve HTTP")
}

func ProcessAppRequests(w http.ResponseWriter, r *http.Request, appC AppController) {
	if (r.URL.Path == "/PortfolioApis/updateprices") && (r.Method == http.MethodPost) {
		msg := processor.FetchAndUpdatePrices(appC.db)
		json.NewEncoder(w).Encode(msg)
	}
	if (r.URL.Path == "/PortfolioApis/updatemasterlist") && (r.Method == http.MethodPost) {
		msg := processor.FetchAndUpdateCompaniesMasterList(appC.db)
		json.NewEncoder(w).Encode(msg)
	}
	if (r.URL.Path == "/PortfolioApis/adduser") && (r.Method == http.MethodPost) {
		reqBody, _ := ioutil.ReadAll(r.Body)
		msg := processor.AddUser(reqBody, appC.db)
		json.NewEncoder(w).Encode(msg)
	} else if (r.URL.Path == "/PortfolioApis/adduserholdings") && (r.Method == http.MethodPost) {
		reqBody, _ := ioutil.ReadAll(r.Body)
		msg := processor.AddUserHoldings(reqBody, appC.db)
		json.NewEncoder(w).Encode(msg)
	} else if (r.URL.Path == "/PortfolioApis/getuserholdings") && (r.Method == http.MethodPost) {
		reqBody, _ := ioutil.ReadAll(r.Body)
		msg := processor.GetUserHoldings(reqBody, appC.db)
		json.NewEncoder(w).Encode(msg)
	} else if (r.URL.Path == "/PortfolioApis/addmodelportfolio") && (r.Method == http.MethodPost) {
		reqBody, _ := ioutil.ReadAll(r.Body)
		msg := processor.AddModelPortfolio(reqBody, appC.db)
		json.NewEncoder(w).Encode(msg)
	} else if (r.URL.Path == "/PortfolioApis/getmodelportfolio") && (r.Method == http.MethodPost) {
		reqBody, _ := ioutil.ReadAll(r.Body)
		msg := processor.GetModelPortfolio(reqBody, appC.db)
		json.NewEncoder(w).Encode(msg)
	} else if (r.URL.Path == "/PortfolioApis/syncportfolio") && (r.Method == http.MethodPost) {
		reqBody, _ := ioutil.ReadAll(r.Body)
		msg := processor.GetPortfolioModelSync(reqBody, appC.db)
		json.NewEncoder(w).Encode(msg)
	}
}
