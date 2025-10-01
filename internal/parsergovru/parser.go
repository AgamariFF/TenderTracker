package parsergovru

import (
	"fmt"
	"net/http"
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

func NewParser() *Parser {
	return &Parser{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (p *Parser) ParseAllPages(baseURL string) ([]models.Tender, error) {
	var allTenders []models.Tender
	quantityCards := 500
	page := 1

	for {
		url := urlgen.ReplaceURLParam(urlgen.ReplaceURLParam(baseURL, "pageNumber", strconv.Itoa(page)), "recordsPerPage", "_"+strconv.Itoa(quantityCards))

		fmt.Printf("Парсинг страницы %d...\n", page)

		tenders, totalCards, err := p.ParsePage(url)
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

func (p *Parser) ParsePage(url string) ([]models.Tender, int, error) {
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
		tender := p.parseTenderCard(s)
		if tender.Title != "" {
			tenders = append(tenders, tender)
		}
	})

	return tenders, totalCards, nil
}

// parseTenderCard парсит отдельную карточку закупки
func (p *Parser) parseTenderCard(s *goquery.Selection) models.Tender {
	var tender models.Tender

	// Ссылка
	link, exists := s.Find(".registry-entry__body-value a").Attr("href")
	if exists {
		if !strings.HasPrefix(link, "http") {
			tender.Link = "https://zakupki.gov.ru" + link
		} else {
			tender.Link = link
		}
	}

	// Название (может быть в другом селекторе)
	tender.Title = strings.TrimSpace(s.Find(".registry-entry__body-title").Text())

	// Заказчик
	tender.Customer = strings.TrimSpace(s.Find(".registry-entry__body-href").Text())

	// Цена
	tender.Price = strings.TrimSpace(s.Find(".price-block__value").Text())

	// Дата
	tender.Date = strings.TrimSpace(s.Find(".data-block__value").Text())

	// Статус
	tender.Status = strings.TrimSpace(s.Find(".registry-entry__header-mid__title").Text())

	return tender
}
