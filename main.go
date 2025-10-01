package main

import (
	"fmt"
	"tendertracker/internal/parsergovru"
	"tendertracker/internal/urlgen"
)

func main() {
	encoder := urlgen.NewURLEncoder("https://zakupki.gov.ru/epz/order/extendedsearch/results.html")

	url := encoder.
		AddParam("searchString", "вентиляци").
		AddParam("morphology", "on").
		AddParam("search-filter", "Дате размещения").
		AddParam("recordsPerPage", "_500").
		AddParam("fz44", "on").
		AddParam("fz223", "on").
		AddParam("ppRf615", "on").
		// AddArrayParam("customerPlace", []string{urlgen.UFO}).
		// AddArrayParam("delKladrIds", []string{"OKER36", "OKER33"}).
		AddParam("gws", "Выберите тип закупки").
		AddParam("applSubmissionCloseDateTo", "02.10.2025").Build()

	tenders, err := parsergovru.NewParser().ParseAllPages(url)
	if err != nil {
		fmt.Print(err)
	}
	fmt.Print(tenders)
}
