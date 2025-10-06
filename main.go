package main

import (
	"fmt"
	"tendertracker/internal/excel"
	"tendertracker/internal/parsergovru"
	"tendertracker/internal/urlgen"
)

func main() {
	encoder := urlgen.NewURLEncoder("https://zakupki.gov.ru/epz/order/extendedsearch/results.html")

	url := encoder.
		AddParam("searchString", "вентиляции легких").
		AddParam("morphology", "on").
		AddParam("search-filter", "Дате размещения").
		AddParam("fz44", "on").
		AddParam("fz223", "on").
		AddParam("ppRf615", "on").
		AddArrayParam("customerPlace", []string{urlgen.SZFO}).
		// AddArrayParam("delKladrIds", []string{"OKER36", "OKER33"}).
		AddParam("gws", "Выберите тип закупки").
		// AddParam("publishDateFrom", "01.10.2025").
		// AddParam("applSubmissionCloseDateTo", "02.10.2025").
		AddParam("af", "on").
		Build()

	tenders, err := parsergovru.NewParser().ParseAllPages(url)
	if err != nil {
		fmt.Print(err)
	}

	excel.ToExcel(&tenders)
}
