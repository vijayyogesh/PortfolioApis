package controllers

import (
	"database/sql"
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
	if r.URL.Path == "/PortfolioApis/adduser" {
		reqBody, _ := ioutil.ReadAll(r.Body)
		//fmt.Fprintf(w, "%+v", string(reqBody))
		fmt.Println(string(reqBody))
		processor.AddUser()
	}
	fmt.Println("Exiting Serve HTTP")
}
