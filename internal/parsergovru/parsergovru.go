package parsergovru

import (
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"tendertracker/internal/logger"
	"tendertracker/internal/models"
	"tendertracker/internal/urlgen"

	"github.com/PuerkitoBio/goquery"
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

func ParseGovRu(name string, config *models.Config, re *regexp.Regexp) ([]models.Tender, error) {
	switch name {
	case "vent":
		return parseSingleCategory(config, "вентиляции", config.MinPriceVent, name, re)

	case "doors":
		return parseMultipleCategories(config, []string{
			"монтаж двер",
			"дверны блок",
			"установ двер",
			"замен двер",
		}, config.MinPriceDoors, name, re)

	case "build":
		return parseMultipleCategories(config, []string{
			"реконструкция здания",
			"строительство здания",
			"капитальный ремонт здания",
		}, config.MinPriceBuild, name, re)

	case "metal":
		return parseSingleCategory(config, "изготовление металлоконструкц", config.MinPriceMetal, name, re)
	}

	return nil, fmt.Errorf("incorrect parameters")
}

func parseSingleCategory(config *models.Config, searchString string, minPrice int, name string, re *regexp.Regexp) ([]models.Tender, error) {
	url := createUrl(*config, searchString, minPrice)
	return NewParser().ParseAllPages(name, url, re, config)
}

func parseMultipleCategories(config *models.Config, searchStrings []string, minPrice int, name string, re *regexp.Regexp) ([]models.Tender, error) {
	var wg sync.WaitGroup
	var mu sync.Mutex

	var allErrors []string
	var allTenders []models.Tender

	parseInGoroutine := func(searchString string, suffix string) {
		defer wg.Done()

		url := createUrl(*config, searchString, minPrice)
		tenders, err := NewParser().ParseAllPages(name+suffix, url, re, config)

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

func (p *Parser) ParseAllPages(name, baseURL string, re *regexp.Regexp, config *models.Config) ([]models.Tender, error) {
	var allTenders []models.Tender
	quantityCards := 100
	page := 1

	for {
		url := urlgen.ReplaceURLParam(urlgen.ReplaceURLParam(baseURL, "pageNumber", strconv.Itoa(page)), "recordsPerPage", "_"+strconv.Itoa(quantityCards))

		logger.SugaredLogger.Infof("%s: Парсинг страницы %d...\n", name, page)
		logger.SugaredLogger.Debug(url)

		tenders, totalCards, err := p.ParsePage(name, url, re, config)
		if err != nil {
			return nil, fmt.Errorf("%s: ошибка на странице %d: %w", name, page, err)
		}

		allTenders = append(allTenders, tenders...)

		logger.SugaredLogger.Infof("%s: Страница %d: найдено %d карточек, распарсено %d тендеров\n",
			name, page, totalCards, len(tenders))

		if totalCards < quantityCards {
			logger.SugaredLogger.Infof("%s: Последняя страница достигнута. Всего страниц: %d\n", name, page)
			break
		}

		if totalCards == 0 {
			logger.SugaredLogger.Infof("%s: На странице %d не найдено карточек, завершаем\n", name, page)
			break
		}

		page++
		time.Sleep(1 * time.Second)
	}

	fmt.Printf("%s: Всего собрано тендеров: %d\n", name, len(allTenders))
	return allTenders, nil
}

func (p *Parser) ParsePage(name, url string, re *regexp.Regexp, config *models.Config) ([]models.Tender, int, error) {
	var resp *http.Response
	var err error

	for attempt := 1; attempt <= 3; attempt++ {
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, 0, fmt.Errorf("ошибка создания запроса: %w", err)
		}

		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
		req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
		req.Header.Set("Accept-Language", "ru-RU,ru;q=0.9,en;q=0.8")

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
		return nil, 0, fmt.Errorf("статус код ошибки: %d %s", resp.StatusCode, resp.Status)
	}

	// Парсим HTML
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, 0, fmt.Errorf("ошибка парсинга HTML: %w", err)
	}

	var tenders []models.Tender
	totalCards := doc.Find(".search-registry-entry-block").Length()

	var cards []*goquery.Selection
	doc.Find(".search-registry-entry-block").Each(func(i int, s *goquery.Selection) {
		cards = append(cards, s)
	})

	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, card := range cards {
		wg.Add(1)

		go func(s *goquery.Selection) {
			defer wg.Done()

			tender := p.parseTenderCard(name, s, re, config)
			if tender.Title != "" {
				mu.Lock()
				tenders = append(tenders, tender)
				mu.Unlock()
			}
		}(card)
	}

	wg.Wait()

	return tenders, totalCards, nil
}
func (p *Parser) parseTenderCard(name string, s *goquery.Selection, re *regexp.Regexp, config *models.Config) models.Tender {
	var tender models.Tender

	// Название
	tender.Title = strings.TrimSpace(s.Find(".registry-entry__body-value").Text())

	// logger.SugaredLogger.Debugf(s.Text())

	if re.MatchString(strings.ToLower(tender.Title)) {
		logger.SugaredLogger.Debugf("%s: отменено: %s", name, tender.Title)
		return models.Tender{}
	}

	// Цена - ищем ТОЛЬКО в пределах текущей карточки
	var minPrice int

	switch name {
	case ("vent"):
		minPrice = config.MinPriceVent
	case ("doors"):
		minPrice = config.MinPriceDoors
	case ("metal"):
		minPrice = config.MinPriceMetal
	default:
		if strings.Contains(name, "build") {
			minPrice = config.MinPriceBuild
		} else {
			minPrice = 0
		}
	}

	priceElem := s.Find(".price-block__value")
	if priceElem.Length() > 0 {
		priceText := strings.TrimSpace(priceElem.First().Text())

		priceText = strings.ReplaceAll(priceText, "	", "")
		priceText = strings.ReplaceAll(priceText, "	", "")      // табуляция
		priceText = strings.ReplaceAll(priceText, "\u00A0", "") // неразрывный пробел

		// Находим индекс первого нецифрового символа
		firstNonDigit := strings.IndexFunc(priceText, func(r rune) bool {
			return !unicode.IsDigit(r)
		})

		if firstNonDigit != -1 {
			priceText = priceText[:firstNonDigit]

			priceInt, err := strconv.Atoi(priceText)
			if err != nil {
				logger.SugaredLogger.Warnf("Incorrect trimed price (not number)")
			}
			if priceInt < minPrice {
				return models.Tender{}
			}

		} else {
			logger.SugaredLogger.Warnln("Incorrect price in tender card")
		}

		tender.Price = strings.TrimSpace(priceElem.First().Text())
	} else {
		tender.Price = "Не указана" // или пустая строка
	}

	dateBlocks := s.Find(".data-block .row .col-6")
	dateBlocks.Each(func(i int, dateBlock *goquery.Selection) {
		title := strings.TrimSpace(dateBlock.Find(".data-block__title").Text())
		value := strings.TrimSpace(dateBlock.Find(".data-block__value").Text())

		if title == "Размещено" {
			tender.PublishDate = value
		}
	})

	// Ссылка
	link, exists := s.Find(".registry-entry__header-mid__number a").Attr("href")
	if exists {
		if !strings.HasPrefix(link, "http") {
			tender.Link = "https://zakupki.gov.ru" + link
		} else {
			tender.Link = link
		}
	}

	// Заказчик
	tender.Customer = strings.TrimSpace(s.Find(".registry-entry__body-href").Text())

	// Дата окончания подачи заявок (находится отдельно)
	applicationEnd := s.Find(".data-block__title:contains('Окончание подачи заявок') + .data-block__value")
	if applicationEnd.Length() > 0 {
		tender.EndDate = strings.TrimSpace(applicationEnd.Text())
	}

	//Адрес
	tender.Region = NewParser().parsePlace(tender.Link)

	return tender
}

func (p *Parser) parsePlace(url string) string {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		logger.SugaredLogger.Errorf("ошибка создания запроса: %v", err)
		return ""
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "ru-RU,ru;q=0.9,en;q=0.8")

	resp, err := p.client.Do(req)
	if err != nil {
		logger.SugaredLogger.Errorf("ошибка отправки запроса: %v", err)
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		logger.SugaredLogger.Errorf("неверный статус код: %d", resp.StatusCode)
		return ""
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		logger.SugaredLogger.Errorf("ошибка парсинга HTML: %v", err)
		return ""
	}

	place := doc.Find(".blockInfo__section .section__info").FilterFunction(func(i int, s *goquery.Selection) bool {
		// Проверяем, что предыдущий элемент содержит заголовок "Место нахождения"
		title := s.Prev().Find(".section__title").Text()
		return strings.Contains(title, "Место нахождения")
	}).First()

	if place.Length() == 0 {
		// Альтернативный поиск, если структура немного отличается
		place = doc.Find("section:contains('Место нахождения') .section__info").First()
	}

	if place.Length() > 0 {
		return strings.TrimSpace(place.Text())
	}

	return ""
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

func createUrl(config models.Config, searchText string, minPrice int) string {
	encoder := urlgen.NewURLEncoder("https://zakupki.gov.ru/epz/order/extendedsearch/results.html")

	now := time.Now()
	twoYearsAgo := now.AddDate(-2, 0, 0)
	dateString := twoYearsAgo.Format("02.01.2006")

	url := encoder.
		AddParam("morphology", "on").
		AddParam("search-filter", "Дате размещения").
		AddParam("fz44", "on").
		AddParam("fz223", "on").
		AddParam("ppRf615", "on").
		AddArrayParam("customerPlace", config.VentCustomerPlace).
		AddParam("gws", "Выберите тип закупки").
		// AddParam("publishDateFrom", "01.10.2025").
		AddParam("applSubmissionCloseDateFrom", dateString).
		AddParam("searchString", searchText).
		AddParam("priceFromGeneral", strconv.Itoa(minPrice))

	switch config.ProcurementType {
	case "completed":
		url.AddParam("pc", "on")
	case "active":
		url.AddParam("af", "on")
	}

	return url.Build()
}
