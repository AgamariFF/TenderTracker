package parsergovru

import (
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"tendertracker/internal/models"
	"tendertracker/internal/urlgen"

	"github.com/PuerkitoBio/goquery"
)

type Parser struct {
	client *http.Client
}

func ParseGovRu(name string, config *models.Config, re *regexp.Regexp) []models.Tender {
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

		tenders, err := NewParser().ParseAllPages(url, re)
		if err != nil {
			fmt.Println(err)
		}
		return tenders

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

		tenders, err := NewParser().ParseAllPages(url, re)
		if err != nil {
			fmt.Println(err)
		}
		return tenders
	}

	return nil
}

func NewParser() *Parser {
	return &Parser{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (p *Parser) ParseAllPages(baseURL string, re *regexp.Regexp) ([]models.Tender, error) {
	var allTenders []models.Tender
	quantityCards := 500
	page := 1

	for {
		url := urlgen.ReplaceURLParam(urlgen.ReplaceURLParam(baseURL, "pageNumber", strconv.Itoa(page)), "recordsPerPage", "_"+strconv.Itoa(quantityCards))

		fmt.Printf("Парсинг страницы %d...\n", page)
		fmt.Println(url)

		tenders, totalCards, err := p.ParsePage(url, re)
		if err != nil {
			return nil, fmt.Errorf("ошибка на странице %d: %w", page, err)
		}

		allTenders = append(allTenders, tenders...)

		fmt.Printf("Страница %d: найдено %d карточек, распарсено %d тендеров\n",
			page, totalCards, len(tenders))

		if totalCards < quantityCards {
			fmt.Printf("Последняя страница достигнута. Всего страниц: %d\n", page)
			break
		}

		if totalCards == 0 {
			fmt.Printf("На странице %d не найдено карточек, завершаем\n", page)
			break
		}

		page++
		time.Sleep(1 * time.Second)
	}

	fmt.Printf("Всего собрано тендеров: %d\n", len(allTenders))
	return allTenders, nil
}

func (p *Parser) ParsePage(url string, re *regexp.Regexp) ([]models.Tender, int, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("ошибка создания запроса: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("ошибка выполнения запроса: %w", err)
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
		tender := p.parseTenderCard(s, re)
		if tender.Title != "" {
			tenders = append(tenders, tender)
		}
	})

	return tenders, totalCards, nil
}

func (p *Parser) parseTenderCard(s *goquery.Selection, re *regexp.Regexp) models.Tender {
	var tender models.Tender

	// Название
	tender.Title = strings.TrimSpace(s.Find(".registry-entry__body-value").Text())

	if re.MatchString(strings.ToLower(tender.Title)) {
		return models.Tender{}
	}

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

	// Цена - ищем ТОЛЬКО в пределах текущей карточки
	priceElem := s.Find(".price-block__value")
	if priceElem.Length() > 0 {
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

	// Дата окончания подачи заявок (находится отдельно)
	applicationEnd := s.Find(".data-block__title:contains('Окончание подачи заявок') + .data-block__value")
	if applicationEnd.Length() > 0 {
		tender.EndDate = strings.TrimSpace(applicationEnd.Text())
	}

	return tender
}
