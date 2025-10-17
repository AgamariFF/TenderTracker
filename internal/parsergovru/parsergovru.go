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
			Timeout: 45 * time.Second,
		},
	}
}

func ParseGovRu(name string, config *models.Config, re *regexp.Regexp) ([]models.Tender, error) {
	switch name {
	case "vent":
		encoder := urlgen.NewURLEncoder("https://zakupki.gov.ru/epz/order/extendedsearch/results.html")

		url := encoder.
			AddParam("searchString", "вентиляции").
			AddParam("morphology", "on").
			AddParam("search-filter", "Дате размещения").
			AddParam("fz44", "on").
			AddParam("fz223", "on").
			AddParam("ppRf615", "on").
			AddArrayParam("customerPlace", config.VentCustomerPlace).
			AddArrayParam("delKladrIds", config.VentDelKladrIds).
			AddParam("gws", "Выберите тип закупки").
			// AddParam("publishDateFrom", "01.10.2025").
			// AddParam("applSubmissionCloseDateTo", "02.10.2025").
			AddParam("af", "on").
			Build()

		tenders, err := NewParser().ParseAllPages(name, url, re, config)
		if err != nil {
			return tenders, err
		}
		return tenders, nil

	case "doors":
		encoder := urlgen.NewURLEncoder("https://zakupki.gov.ru/epz/order/extendedsearch/results.html")

		url := encoder.
			AddParam("searchString", "монтаж двер").
			AddParam("morphology", "on").
			AddParam("search-filter", "Дате размещения").
			AddParam("fz44", "on").
			AddParam("fz223", "on").
			AddParam("ppRf615", "on").
			AddArrayParam("customerPlace", config.VentCustomerPlace).
			AddArrayParam("delKladrIds", config.VentDelKladrIds).
			AddParam("gws", "Выберите тип закупки").
			// AddParam("publishDateFrom", "01.10.2025").
			// AddParam("applSubmissionCloseDateTo", "02.10.2025").
			AddParam("af", "on").
			Build()

		tenders, err := NewParser().ParseAllPages(name, url, re, config)
		if err != nil {
			return tenders, err
		}
		return tenders, nil

	case "build":
		var wg sync.WaitGroup
		var mu sync.Mutex

		var allErrors []string
		var allTenders []models.Tender

		parseInGoroutine := func(searchString string, suffix string) {
			defer wg.Done()

			encoder := urlgen.NewURLEncoder("https://zakupki.gov.ru/epz/order/extendedsearch/results.html")
			url := encoder.
				AddParam("searchString", searchString).
				AddParam("morphology", "on").
				AddParam("search-filter", "Дате размещения").
				AddParam("fz44", "on").
				AddParam("fz223", "on").
				AddParam("ppRf615", "on").
				AddArrayParam("customerPlace", config.VentCustomerPlace).
				AddArrayParam("delKladrIds", config.VentDelKladrIds).
				AddParam("gws", "Выберите тип закупки").
				// AddParam("publishDateFrom", "01.10.2025").
				// AddParam("applSubmissionCloseDateTo", "02.10.2025").
				AddParam("af", "on").
				Build()

			tenders, err := NewParser().ParseAllPages(name+suffix, url, re, config)

			mu.Lock()
			if err != nil {
				allErrors = append(allErrors, fmt.Sprintf("%s: %v", searchString, err))
			} else {
				allTenders = append(allTenders, tenders...)
			}
			mu.Unlock()
		}

		wg.Add(3)
		go parseInGoroutine("реконструкция здания", "0")
		go parseInGoroutine("строительство здания", "1")
		go parseInGoroutine("капитальный ремонт здания", "2")
		wg.Wait()

		if len(allErrors) > 0 {
			return allTenders, fmt.Errorf("build search failed: %s", strings.Join(allErrors, "; "))
		}

		tenders := mergeTendersWithoutDuplicates(allTenders)
		return tenders, nil

	case "metal":
		encoder := urlgen.NewURLEncoder("https://zakupki.gov.ru/epz/order/extendedsearch/results.html")

		url := encoder.
			AddParam("searchString", "изготовление металлоконструкц").
			AddParam("morphology", "on").
			AddParam("search-filter", "Дате размещения").
			AddParam("fz44", "on").
			AddParam("fz223", "on").
			AddParam("ppRf615", "on").
			AddArrayParam("customerPlace", config.VentCustomerPlace).
			AddArrayParam("delKladrIds", config.VentDelKladrIds).
			AddParam("gws", "Выберите тип закупки").
			// AddParam("publishDateFrom", "01.10.2025").
			// AddParam("applSubmissionCloseDateTo", "02.10.2025").
			AddParam("af", "on").
			Build()

		tenders, err := NewParser().ParseAllPages(name, url, re, config)
		if err != nil {
			return tenders, err
		}
		return tenders, nil
	}

	return nil, fmt.Errorf("Incorrect parametrs")
}

func (p *Parser) ParseAllPages(name, baseURL string, re *regexp.Regexp, config *models.Config) ([]models.Tender, error) {
	var allTenders []models.Tender
	quantityCards := 50
	page := 1

	for {
		url := urlgen.ReplaceURLParam(urlgen.ReplaceURLParam(baseURL, "pageNumber", strconv.Itoa(page)), "recordsPerPage", "_"+strconv.Itoa(quantityCards))

		logger.SugaredLogger.Infof("%s: Парсинг страницы %d...\n", name, page)
		logger.SugaredLogger.Info(url)

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
			logger.SugaredLogger.Warnf("Попытка %d не удалась, повтор через %v: %v", attempt, waitTime, err)
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

	doc.Find(".search-registry-entry-block").Each(func(i int, s *goquery.Selection) {
		tender := p.parseTenderCard(name, s, re, config)
		if tender.Title != "" {
			tenders = append(tenders, tender)
		}
	})

	return tenders, totalCards, nil
}

func (p *Parser) parseTenderCard(name string, s *goquery.Selection, re *regexp.Regexp, config *models.Config) models.Tender {
	var tender models.Tender

	// Название
	tender.Title = strings.TrimSpace(s.Find(".registry-entry__body-value").Text())

	if re.MatchString(strings.ToLower(tender.Title)) {
		return models.Tender{}
	}

	// Цена - ищем ТОЛЬКО в пределах текущей карточки
	var minPrice int

	switch name {
	case ("vent"):
		minPrice = config.MinPriceVent
	case ("doors"):
		minPrice = config.MinPriceDoors
	case ("build"):
		minPrice = config.MinPriceBuild
	case ("metal"):
		minPrice = config.MinPriceMetal
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

	return tender
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
