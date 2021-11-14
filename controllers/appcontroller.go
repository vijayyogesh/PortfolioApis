package controllers

import (
	"database/sql"
	"fmt"
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
	processor.FetchAndUpdatePrices(appC.db)
	fmt.Println("Exiting Serve HTTP")
}
