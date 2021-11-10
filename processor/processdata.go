package processor

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/vijayyogesh/PortfolioApis/data"
)

func ReadCsv(filePath string, companyid string) ([]data.CompaniesPriceData, error) {
	var companiesdata []data.CompaniesPriceData

	/* Open file */
	file, err := os.Open(filePath)
	/* Return if error */
	if err != nil {
		fmt.Println(err.Error(), "Error while opening file ")
		return companiesdata, fmt.Errorf("error while opening file %s ", filePath)
	}
	fmt.Println(file.Name())

	/* Read csv */
	csvReader := csv.NewReader(file)
	records, err := csvReader.ReadAll()
	/* Return if error */
	if err != nil {
		fmt.Println(err.Error(), "Error while reading csv ")
		return companiesdata, fmt.Errorf("error while reading csv %s ", filePath)
	}
	/* Close resources */
	file.Close()

	/* Process each record */
	for k, v := range records {

		openval, dataError := strconv.ParseFloat(v[len(v)-6], 64)
		processDataErr(dataError, k)

		highval, dataError := strconv.ParseFloat(v[len(v)-5], 64)
		processDataErr(dataError, k)

		lowval, dataError := strconv.ParseFloat(v[len(v)-4], 64)
		processDataErr(dataError, k)

		closeval, dataError := strconv.ParseFloat(v[len(v)-3], 64)
		processDataErr(dataError, k)

		dateval, dataError := time.Parse("2006-01-02", v[len(v)-7])
		processDataErr(dataError, k)

		companiesdata = append(companiesdata, data.CompaniesPriceData{CompanyId: companyid, DateVal: dateval, OpenVal: openval, HighVal: highval, LowVal: lowval, CloseVal: closeval})
	}

	fmt.Println("Name - " + companyid)
	fmt.Println(len(companiesdata))
	return companiesdata, nil

}

/* Non critical record error which can be logged and ignored */
func processDataErr(dataError error, k int) {
	if dataError != nil {
		fmt.Println(dataError.Error(), "Error while processing data", k)
	}
}
