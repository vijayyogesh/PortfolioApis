package main

import (
	"fmt"

	"github.com/vijayyogesh/PortfolioApis/data"
	"github.com/vijayyogesh/PortfolioApis/processor"

	_ "github.com/lib/pq"
)

func main() {
	fmt.Println("In main")
	companiesdata, err := processor.ReadCsv("C:\\Users\\Ajay\\Downloads\\HINDUNILVR.NS.csv")
	if err != nil {
		panic(err)
	}
	data.AddPriceData(companiesdata)
}
