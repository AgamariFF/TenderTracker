package parsersber

import "encoding/xml"

type ElasticRequest struct {
	XMLName      xml.Name     `xml:"elasticrequest"`
	PersonID     int          `xml:"personid"`
	BUID         int          `xml:"buid"`
	Filters      Filters      `xml:"filters"`
	Fields       Fields       `xml:"fields"`
	Sort         Sort         `xml:"sort"`
	Aggregations Aggregations `xml:"aggregations"`
	Size         int          `xml:"size"`
	From         int          `xml:"from"`
}

type Filters struct {
	MainSearchBar     MainSearchBar     `xml:"mainSearchBar"`
	PurchAmount       PurchAmount       `xml:"purchAmount"`
	PublicDate        DateRange         `xml:"PublicDate"`
	PurchaseStageTerm PurchaseStageTerm `xml:"PurchaseStageTerm"`
	RegionNameTerm    RegionNameTerm    `xml:"RegionNameTerm"`
	RequestStartDate  DateRange         `xml:"RequestStartDate"`
	RequestDate       DateRange         `xml:"RequestDate"`
	AuctionBeginDate  DateRange         `xml:"AuctionBeginDate"`
}

type MainSearchBar struct {
	Value              string `xml:"value"`
	Type               string `xml:"type"`
	MinimumShouldMatch string `xml:"minimum_should_match"`
}

type PurchAmount struct {
	MinValue string `xml:"minvalue"`
	MaxValue string `xml:"maxvalue"`
}

type DateRange struct {
	MinValue string `xml:"minvalue"`
	MaxValue string `xml:"maxvalue"`
}

type PurchaseStageTerm struct {
	Value       string `xml:"value"`
	VisiblePart string `xml:"visiblepart"`
}

type RegionNameTerm struct {
	Value       string `xml:"value"`
	VisiblePart string `xml:"visiblepart"`
}

type Fields struct {
	Field []string `xml:"field"`
}

type Sort struct {
	Value     string `xml:"value"`
	Direction string `xml:"direction"`
}

type Aggregations struct {
	Empty EmptyAggregation `xml:"empty"`
}

type EmptyAggregation struct {
	FilterType string `xml:"filterType"`
	Field      string `xml:"field"`
}
