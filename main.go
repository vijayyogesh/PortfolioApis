package main

import (
	"database/sql"
	"fmt"
	"net/http"

	"github.com/vijayyogesh/PortfolioApis/controllers"
	"github.com/vijayyogesh/PortfolioApis/data"

	_ "github.com/lib/pq"
)

var db *sql.DB
var appC *controllers.AppController

func main() {
	/*fmt.Println("In main - start tme: " + time.Now().String())
	processor.FetchAndUpdatePrices(db)
	fmt.Println("In main - end tme: " + time.Now().String()) */

	/*fmt.Println("In main - start tme: " + time.Now().String())
	processor.FetchAndUpdateCompaniesMasterList(db)
	fmt.Println("In main - end tme: " + time.Now().String()) */

	/*http.Handle("/PortfolioApis/refresh", *appC)
	http.ListenAndServe(":3000", nil)*/

	http.Handle("/PortfolioApis/adduser", *appC)
	http.Handle("/PortfolioApis/adduserholdings", *appC)
	http.ListenAndServe(":3000", nil)

}

func init() {
	fmt.Println("In init")
	db = data.SetupDB()
	appC = controllers.NewAppController(db)
}
