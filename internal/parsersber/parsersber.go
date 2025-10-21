package parsersber

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"tendertracker/internal/logger"
	"tendertracker/internal/models"
)

type Parser struct {
	client *http.Client
}

func NewParser() *Parser {
	return &Parser{
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

func ParseSberAst(name string, config *models.Config, re *regexp.Regexp) ([]models.Tender, error) {
	switch name {
	case "vent":
		return parseSingleCategory(config, "вент", config.MinPriceVent, name, re)

	case "doors":
		return parseMultipleCategories(config, []string{
			"двер",
			"дверны",
		}, config.MinPriceDoors, name, re)

	case "build":
		return parseMultipleCategories(config, []string{
			"реконструкция",
			"строительство",
			"капитальный ремонт",
		}, config.MinPriceBuild, name, re)

	case "metal":
		return parseSingleCategory(config, "металлоконструкц", config.MinPriceMetal, name, re)
	}

	return nil, fmt.Errorf("incorrect parameters")
}

func parseSingleCategory(config *models.Config, searchString string, minPrice int, name string, re *regexp.Regexp) ([]models.Tender, error) {
	searchRequest := createSearchRequest(searchString, minPrice, config, 0, 20)
	return NewParser().ParseAllPages(name, searchRequest, re, config, minPrice)
}

func parseMultipleCategories(config *models.Config, searchStrings []string, minPrice int, name string, re *regexp.Regexp) ([]models.Tender, error) {
	var wg sync.WaitGroup
	var mu sync.Mutex

	var allErrors []string
	var allTenders []models.Tender

	parseInGoroutine := func(searchString string, suffix string) {
		defer wg.Done()

		searchRequest := createSearchRequest(searchString, minPrice, config, 0, 20)
		tenders, err := NewParser().ParseAllPages(name+suffix, searchRequest, re, config, minPrice)

		mu.Lock()
		if err != nil {
			allErrors = append(allErrors, fmt.Sprintf("%s: %v", searchString, err))
		} else {
			allTenders = append(allTenders, tenders...)
		}
		mu.Unlock()
	}

	wg.Add(len(searchStrings))
	for i, searchString := range searchStrings {
		go parseInGoroutine(searchString, strconv.Itoa(i))
	}
	wg.Wait()

	if len(allErrors) > 0 {
		return allTenders, fmt.Errorf("%s search failed: %s", name, strings.Join(allErrors, "; "))
	}

	return mergeTendersWithoutDuplicates(allTenders), nil
}

func (p *Parser) ParseAllPages(name string, searchRequest ElasticRequest, re *regexp.Regexp, config *models.Config, minPrice int) ([]models.Tender, error) {
	var allTenders []models.Tender
	pageSize := 20
	from := 0
	maxPages := 50 // Ограничиваем 50 страницами (1000 тендеров)

	for page := 1; page <= maxPages; page++ {
		searchRequest.From = from
		searchRequest.Size = pageSize

		logger.SugaredLogger.Infof("%s: Парсинг страницы %d (from: %d, size: %d)...", name, page, from, pageSize)

		tenders, totalHits, err := p.ParsePage(name, searchRequest, re, config, minPrice)
		if err != nil {
			return nil, fmt.Errorf("%s: ошибка на странице %d: %w", name, page, err)
		}

		// Если страница пустая - останавливаемся
		if len(tenders) == 0 {
			logger.SugaredLogger.Infof("%s: Пустая страница %d, завершаем парсинг", name, page)
			break
		}

		allTenders = append(allTenders, tenders...)

		logger.SugaredLogger.Infof("%s: Страница %d: найдено %d тендеров, распарсено %d, всего: %d",
			name, page, totalHits, len(tenders), len(allTenders))

		// Останавливаемся если достигли лимита страниц
		if page >= maxPages {
			logger.SugaredLogger.Infof("%s: Достигнут лимит в %d страниц", name, maxPages)
			break
		}

		// Останавливаемся если достигли общего количества (когда оно реальное)
		if totalHits < 10000 && from+pageSize >= totalHits {
			logger.SugaredLogger.Infof("%s: Достигнут конец данных", name)
			break
		}

		from += pageSize
		time.Sleep(1 * time.Second)
	}

	logger.SugaredLogger.Infof("%s: Завершено. Всего собрано тендеров: %d", name, len(allTenders))
	return allTenders, nil
}

// Функция для определения условий остановки парсинга
func shouldStopParsing(from, pageSize, totalHits, currentPageTenders, currentPage, maxPages int) bool {
	// Если достигли максимального количества страниц
	if currentPage >= maxPages {
		logger.SugaredLogger.Infof("Остановка: достигнут лимит в %d страниц", maxPages)
		return true
	}

	// Если на текущей странице меньше половины ожидаемых тендеров (возможно, конец данных)
	if currentPageTenders < pageSize/2 && currentPage > 5 {
		logger.SugaredLogger.Infof("Остановка: на странице только %d тендеров из %d", currentPageTenders, pageSize)
		return true
	}

	// Если API возвращает 10000 (максимум), но мы уже далеко прошли
	if totalHits == 10000 && from > 5000 && currentPageTenders == 0 {
		logger.SugaredLogger.Infof("Остановка: достигнут предел данных API при from=%d", from)
		return true
	}

	// Стандартное условие - если достигли общего количества
	if from+pageSize >= totalHits && totalHits < 10000 {
		logger.SugaredLogger.Infof("Остановка: достигнут конец данных (from=%d, totalHits=%d)", from, totalHits)
		return true
	}

	return false
}

func (p *Parser) ParsePage(name string, searchRequest ElasticRequest, re *regexp.Regexp, config *models.Config, minPrice int) ([]models.Tender, int, error) {
	var resp *http.Response
	var err error

	// Конвертируем XML запрос в строку
	xmlData, err := xml.Marshal(searchRequest)
	if err != nil {
		return nil, 0, fmt.Errorf("ошибка маршалинга XML: %w", err)
	}

	// Создаем форму данных как в ручном запросе
	formData := url.Values{}
	formData.Add("xmlData", string(xmlData))
	formData.Add("orgId", "0")
	formData.Add("targetPageCode", "UnitedPurchaseList")
	formData.Add("PID", "0")

	// Кодируем данные для тела запроса
	body := strings.NewReader(formData.Encode())

	// Правильный URL из вашего ручного запроса
	baseURL := "https://sberbank-ast.ru/SearchQuery.aspx"

	for attempt := 1; attempt <= 3; attempt++ {
		req, err := http.NewRequest("POST", baseURL, body)
		if err != nil {
			return nil, 0, fmt.Errorf("ошибка создания запроса: %w", err)
		}

		// Устанавливаем правильные заголовки для формы
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
		req.Header.Set("Accept", "application/json, text/plain, */*")
		req.Header.Set("Accept-Language", "ru-RU,ru;q=0.9,en;q=0.8")
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
		req.Header.Set("X-Requested-With", "XMLHttpRequest")

		// Добавляем параметр name в URL как в ручном запросе
		q := req.URL.Query()
		q.Add("name", "Main")
		req.URL.RawQuery = q.Encode()

		logger.SugaredLogger.Debugf("URL: %s", req.URL.String())
		logger.SugaredLogger.Debugf("Form Data: %s", formData.Encode())

		resp, err = p.client.Do(req)
		if err == nil {
			break
		}

		if attempt < 3 {
			waitTime := time.Duration(attempt*attempt) * 2 * time.Second
			logger.SugaredLogger.Warnf("%s Попытка %d не удалась, повтор через %v: %v", name, attempt, waitTime, err)
			time.Sleep(waitTime)
			continue
		}
		return nil, 0, fmt.Errorf("ошибка выполнения запроса после 3 попыток: %w", err)
	}

	if err != nil {
		return nil, 0, fmt.Errorf("ошибка выполнения запроса после 3 попыток: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, 0, fmt.Errorf("статус код ошибки: %d %s, тело ответа: %s", resp.StatusCode, resp.Status, string(bodyBytes))
	}

	var apiResponse SberAstResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		return nil, 0, fmt.Errorf("ошибка декодирования JSON: %w", err)
	}

	if apiResponse.Result != "success" {
		return nil, 0, fmt.Errorf("API вернуло ошибку: %s", apiResponse.Result)
	}

	var dataResponse DataResponse
	if err := json.Unmarshal([]byte(apiResponse.Data), &dataResponse); err != nil {
		return nil, 0, fmt.Errorf("ошибка декодирования data JSON: %w", err)
	}

	var elasticResponse ElasticResponse
	if err := json.Unmarshal([]byte(dataResponse.Data), &elasticResponse); err != nil {
		return nil, 0, fmt.Errorf("ошибка декодирования elastic JSON: %w", err)
	}

	totalHits := elasticResponse.Hits.Total.Value
	var tenders []models.Tender

	for _, hit := range elasticResponse.Hits.Hits {
		tender := p.parseTenderHit(name, hit, re, config)
		if tender.Title != "" {
			tenders = append(tenders, tender)
		}
	}

	return tenders, totalHits, nil
}

func (p *Parser) parseTenderHit(name string, hit Hit, re *regexp.Regexp, config *models.Config) models.Tender {
	var tender models.Tender

	if hit.Source.PurchName != "" {
		tender.Title = hit.Source.PurchName
	} else {
		tender.Title = hit.Source.BidName
	}

	if re.MatchString(strings.ToLower(tender.Title)) {
		return models.Tender{}
	}

	tender.Price = formatPrice(hit.Source.PurchAmount)
	tender.PublishDate = hit.Source.PublicDate
	tender.Customer = hit.Source.OrgName
	tender.EndDate = hit.Source.EndDate

	if hit.Source.ObjectHrefTerm != "" {
		tender.Link = hit.Source.ObjectHrefTerm
	} else if hit.Source.SourceHrefTerm != "" {
		tender.Link = hit.Source.SourceHrefTerm
	}

	return tender
}

func formatPrice(amount float64) string {
	if amount == 0 {
		return "Не указана"
	}
	return fmt.Sprintf("%.2f руб.", amount)
}

func createSearchRequest(searchText string, minPrice int, config *models.Config, from, size int) ElasticRequest {
	searchRequest := ElasticRequest{
		PersonID: 0,
		BUID:     0,
		Filters: Filters{
			MainSearchBar: SearchFilter{
				Value:              searchText,
				Type:               "best_fields",
				MinimumShouldMatch: "1%",
			},
			PurchAmount: PriceFilter{
				MinValue: strconv.Itoa(minPrice),
			},
			PublicDate: DateFilter{
				MinValue: "",
				MaxValue: "",
			},
			// ДОБАВЛЕНО: Фильтр по стадии закупки - только "Опубликовано" и "Подача заявок"
			PurchaseStageTerm: PurchaseStageTerm{
				Value:       "Опубликовано|;|Подача заявок",
				VisiblePart: "Опубликовано,Подача заявок",
			},
			SourceTerm: SourceTerm{
				Value:       "",
				VisiblePart: "",
			},
			RegionNameTerm: RegionNameTerm{
				Value:       "",
				VisiblePart: "",
			},
			RequestStartDate: DateFilter{
				MinValue: "",
				MaxValue: "",
			},
			RequestDate: DateFilter{
				MinValue: "",
				MaxValue: "",
			},
			AuctionBeginDate: DateFilter{
				MinValue: "",
				MaxValue: "",
			},
			Okdp2MultiMatch: Okdp2MultiMatch{
				Value: "",
			},
			Okdp2Tree: Okdp2Tree{
				Value:        "",
				ProductField: "",
				BranchField:  "",
			},
			Classifier: Classifier{
				VisiblePart: "",
			},
			OrgCondition: OrgCondition{
				Value: "",
			},
			OrgDictionary: OrgDictionary{
				Value: "",
			},
			Organizator: Organizator{
				VisiblePart: "",
			},
			CustomerCondition: CustomerCondition{
				Value: "",
			},
			CustomerDictionary: CustomerDictionary{
				Value: "",
			},
			Customer: Customer{
				VisiblePart: "",
			},
			PurchaseWayTerm: PurchaseWayTerm{
				Value:       "",
				VisiblePart: "",
			},
			PurchaseTypeNameTerm: PurchaseTypeNameTerm{
				Value:       "",
				VisiblePart: "",
			},
			BranchNameTerm: BranchNameTerm{
				Value:       "",
				VisiblePart: "",
			},
			IsSharedTerm: IsSharedTerm{
				Value:       "",
				VisiblePart: "",
			},
			IsHasComplaint: IsHasComplaint{
				Value: "",
			},
			IsPurchCostDetails: IsPurchCostDetails{
				Value: "",
			},
			NotificationFeatures: NotificationFeatures{
				Value:       "",
				VisiblePart: "",
			},
		},
		Fields: []string{
			"TradeSectionId", "purchAmount", "purchCurrency", "purchCodeTerm",
			"PurchaseTypeName", "purchStateName", "BidStatusName", "OrgName",
			"SourceTerm", "PublicDate", "RequestDate", "RequestStartDate",
			"RequestAcceptDate", "EndDate", "CreateRequestHrefTerm",
			"CreateRequestAlowed", "purchName", "BidName", "SourceHrefTerm",
			"objectHrefTerm", "needPayment", "IsSMP", "isIncrease",
			"isHasComplaint", "isPurchCostDetails", "purchType",
		},
		Sort: Sort{
			Value: "default",
		},
		Aggregations: Aggregations{
			Empty: EmptyAggregation{
				FilterType: "filter_aggregation",
			},
		},
		Size: size,
		From: from,
	}

	if config.ProcurementType == "active" {
		twoYearsAgo := time.Now().AddDate(-2, 0, 0)
		searchRequest.Filters.PublicDate.MinValue = twoYearsAgo.Format("02.01.2006")
	}

	return searchRequest
}

func mergeTendersWithoutDuplicates(tenderSlices ...[]models.Tender) []models.Tender {
	seen := make(map[string]bool)
	var result []models.Tender

	for _, slice := range tenderSlices {
		for _, tender := range slice {
			if tender.Title != "" && !seen[tender.Title] {
				seen[tender.Title] = true
				result = append(result, tender)
			}
		}
	}

	return result
}

type SberAstResponse struct {
	Result string `json:"result"`
	Data   string `json:"data"`
}

type DataResponse struct {
	Data string `json:"data"`
}

type ElasticResponse struct {
	Hits struct {
		Total struct {
			Value int `json:"value"`
		} `json:"total"`
		Hits []Hit `json:"hits"`
	} `json:"hits"`
}

type Hit struct {
	Source struct {
		PurchName        string  `json:"purchName"`
		BidName          string  `json:"BidName"`
		PurchAmount      float64 `json:"purchAmount"`
		PublicDate       string  `json:"PublicDate"`
		EndDate          string  `json:"EndDate"`
		OrgName          string  `json:"OrgName"`
		ObjectHrefTerm   string  `json:"objectHrefTerm"`
		SourceHrefTerm   string  `json:"SourceHrefTerm"`
		PurchaseTypeName string  `json:"PurchaseTypeName"`
		PurchStateName   string  `json:"purchStateName"`
	} `json:"_source"`
}

type ElasticRequest struct {
	XMLName      xml.Name     `xml:"elasticrequest"`
	PersonID     int          `xml:"personid"`
	BUID         int          `xml:"buid"`
	Filters      Filters      `xml:"filters"`
	Fields       []string     `xml:"fields>field"`
	Sort         Sort         `xml:"sort"`
	Aggregations Aggregations `xml:"aggregations"`
	Size         int          `xml:"size"`
	From         int          `xml:"from"`
}

type Filters struct {
	MainSearchBar        SearchFilter         `xml:"mainSearchBar"`
	PurchAmount          PriceFilter          `xml:"purchAmount"`
	PublicDate           DateFilter           `xml:"PublicDate"`
	PurchaseStageTerm    PurchaseStageTerm    `xml:"PurchaseStageTerm"`
	SourceTerm           SourceTerm           `xml:"SourceTerm"`
	RegionNameTerm       RegionNameTerm       `xml:"RegionNameTerm"`
	RequestStartDate     DateFilter           `xml:"RequestStartDate"`
	RequestDate          DateFilter           `xml:"RequestDate"`
	AuctionBeginDate     DateFilter           `xml:"AuctionBeginDate"`
	Okdp2MultiMatch      Okdp2MultiMatch      `xml:"okdp2MultiMatch"`
	Okdp2Tree            Okdp2Tree            `xml:"okdp2tree"`
	Classifier           Classifier           `xml:"classifier"`
	OrgCondition         OrgCondition         `xml:"orgCondition"`
	OrgDictionary        OrgDictionary        `xml:"orgDictionary"`
	Organizator          Organizator          `xml:"organizator"`
	CustomerCondition    CustomerCondition    `xml:"CustomerCondition"`
	CustomerDictionary   CustomerDictionary   `xml:"CustomerDictionary"`
	Customer             Customer             `xml:"customer"`
	PurchaseWayTerm      PurchaseWayTerm      `xml:"PurchaseWayTerm"`
	PurchaseTypeNameTerm PurchaseTypeNameTerm `xml:"PurchaseTypeNameTerm"`
	BranchNameTerm       BranchNameTerm       `xml:"BranchNameTerm"`
	IsSharedTerm         IsSharedTerm         `xml:"isSharedTerm"`
	IsHasComplaint       IsHasComplaint       `xml:"isHasComplaint"`
	IsPurchCostDetails   IsPurchCostDetails   `xml:"isPurchCostDetails"`
	NotificationFeatures NotificationFeatures `xml:"notificationFeatures"`
}

type SearchFilter struct {
	Value              string `xml:"value"`
	Type               string `xml:"type"`
	MinimumShouldMatch string `xml:"minimum_should_match"`
}

type PriceFilter struct {
	MinValue string `xml:"minvalue"`
	MaxValue string `xml:"maxvalue"`
}

type DateFilter struct {
	MinValue string `xml:"minvalue"`
	MaxValue string `xml:"maxvalue"`
}

type PurchaseStageTerm struct {
	Value       string `xml:"value"`
	VisiblePart string `xml:"visiblepart"`
}

type SourceTerm struct {
	Value       string `xml:"value"`
	VisiblePart string `xml:"visiblepart"`
}

type RegionNameTerm struct {
	Value       string `xml:"value"`
	VisiblePart string `xml:"visiblepart"`
}

type Okdp2MultiMatch struct {
	Value string `xml:"value"`
}

type Okdp2Tree struct {
	Value        string `xml:"value"`
	ProductField string `xml:"productField"`
	BranchField  string `xml:"branchField"`
}

type Classifier struct {
	VisiblePart string `xml:"visiblepart"`
}

type OrgCondition struct {
	Value string `xml:"value"`
}

type OrgDictionary struct {
	Value string `xml:"value"`
}

type Organizator struct {
	VisiblePart string `xml:"visiblepart"`
}

type CustomerCondition struct {
	Value string `xml:"value"`
}

type CustomerDictionary struct {
	Value string `xml:"value"`
}

type Customer struct {
	VisiblePart string `xml:"visiblepart"`
}

type PurchaseWayTerm struct {
	Value       string `xml:"value"`
	VisiblePart string `xml:"visiblepart"`
}

type PurchaseTypeNameTerm struct {
	Value       string `xml:"value"`
	VisiblePart string `xml:"visiblepart"`
}

type BranchNameTerm struct {
	Value       string `xml:"value"`
	VisiblePart string `xml:"visiblepart"`
}

type IsSharedTerm struct {
	Value       string `xml:"value"`
	VisiblePart string `xml:"visiblepart"`
}

type IsHasComplaint struct {
	Value string `xml:"value"`
}

type IsPurchCostDetails struct {
	Value string `xml:"value"`
}

type NotificationFeatures struct {
	Value       string `xml:"value"`
	VisiblePart string `xml:"visiblepart"`
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
