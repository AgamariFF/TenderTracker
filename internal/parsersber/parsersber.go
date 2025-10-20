package parsersber

import (
	"encoding/xml"
	"fmt"
	"net/url"
	"strings"
)

type SearchConfig struct {
	Query            string
	MinPrice         int
	MaxPrice         int
	FederalDistricts []string // ["Центральный ФО", "Северо-Западный ФО"]
	PurchaseStages   []string // ["Опубликовано", "Подача заявок"]
	Page             int
	PageSize         int
}

func BuildSearchRequest(config SearchConfig) (*url.URL, error) {
	// Получаем регионы для выбранных федеральных округов
	selectedRegions := getRegionsByFederalDistricts(config.FederalDistricts)

	// Формируем XML запрос
	request := ElasticRequest{
		PersonID: 0,
		BUID:     0,
		Filters: Filters{
			MainSearchBar: MainSearchBar{
				Value:              config.Query,
				Type:               "phrase_prefix",
				MinimumShouldMatch: "100%",
			},
			PurchAmount: PurchAmount{
				MinValue: fmt.Sprintf("%d", config.MinPrice),
				MaxValue: fmt.Sprintf("%d", config.MaxPrice),
			},
			PurchaseStageTerm: PurchaseStageTerm{
				Value:       strings.Join(config.PurchaseStages, "|;|"),
				VisiblePart: strings.Join(config.PurchaseStages, ","),
			},
			RegionNameTerm: RegionNameTerm{
				Value:       strings.Join(selectedRegions, "|;|"),
				VisiblePart: truncateVisiblePart(strings.Join(selectedRegions, ",")),
			},
		},
		Fields: Fields{
			Field: []string{
				"TradeSectionId", "purchAmount", "purchCurrency", "purchCodeTerm",
				"PurchaseTypeName", "purchStateName", "BidStatusName", "OrgName",
				"SourceTerm", "PublicDate", "RequestDate", "RequestStartDate",
				"RequestAcceptDate", "EndDate", "CreateRequestHrefTerm",
				"CreateRequestAlowed", "purchName", "BidName", "SourceHrefTerm",
				"objectHrefTerm", "needPayment", "IsSMP", "isIncrease",
				"isHasComplaint", "isPurchCostDetails", "purchType",
			},
		},
		Sort: Sort{
			Value:     "default",
			Direction: "",
		},
		Aggregations: Aggregations{
			Empty: EmptyAggregation{
				FilterType: "filter_aggregation",
				Field:      "",
			},
		},
		Size: config.PageSize,
		From: config.Page * config.PageSize,
	}

	// Сериализуем XML
	xmlData, err := xml.Marshal(request)
	if err != nil {
		return nil, err
	}

	encodedXML := url.QueryEscape(string(xmlData))

	// Формируем полный URL
	baseURL := "https://sberbank-ast.ru"
	params := url.Values{}
	params.Add("xmlData", encodedXML)
	params.Add("orgId", "0")
	params.Add("targetPageCode", "UnitedPurchaseList")
	params.Add("PID", "0")

	fullURL := baseURL + "?" + params.Encode()
	return url.Parse(fullURL)
}

func getRegionsByFederalDistricts(districts []string) []string {
	var regionsList []string
	for _, district := range districts {
		if regions, exists := FederalDistricts[district]; exists {
			regionsList = append(regionsList, regions...)
		}
	}
	return regionsList
}

func truncateVisiblePart(text string) string {
	if len(text) > 50 {
		return text[:47] + "..."
	}
	return text
}
