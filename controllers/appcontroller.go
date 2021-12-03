package controllers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/vijayyogesh/PortfolioApis/processor"
)

type AppController struct {
	db *sql.DB
}

func NewAppController(dbRef *sql.DB) *AppController {
	return &AppController{
		db: dbRef,
	}
}

func (appC AppController) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Println("In Serve HTTP")
	//processor.FetchAndUpdatePrices(appC.db)

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
