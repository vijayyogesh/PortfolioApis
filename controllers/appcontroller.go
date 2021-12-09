package controllers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

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

func (appC AppController) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Println("In Serve HTTP")
	//processor.FetchAndUpdatePrices(appC.db)

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

	fmt.Println("Exiting Serve HTTP")
}
