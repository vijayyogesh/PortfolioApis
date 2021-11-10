package main

import (
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/vijayyogesh/PortfolioApis/data"
	"github.com/vijayyogesh/PortfolioApis/processor"

	_ "github.com/lib/pq"
)

var db *sql.DB

func main() {
	fmt.Println("In main - start tme: " + time.Now().String())
	companies := []string{"HINDUNILVR", "NESTLEIND", "HDFCBANK", "ITC", "RELIANCE"}

	var wg sync.WaitGroup

	for _, companyid := range companies {
		wg.Add(1)

		filePath := "C:\\Users\\Ajay\\Downloads\\" + companyid + ".NS.csv"
		fmt.Println(filePath)

		go func(companyid string) {
			defer wg.Done()
			var err error
			companiesdata, err := processor.ReadCsv(filePath, companyid)
			if err != nil {
				panic(err)
			}
			data.AddPriceData(companiesdata, db)
		}(companyid)
	}

	wg.Wait()

	fmt.Println("In main - end tme: " + time.Now().String())
}

func init() {
	fmt.Println("In init")
	db = data.SetupDB()
}
