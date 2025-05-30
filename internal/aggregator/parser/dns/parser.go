package dns

import (
	"StartupPCConfigurator/internal/aggregator/usecase"
	"context"
	"fmt"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/page"
	"log"
	"math/rand"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/chromedp"
)

// DNSParser содержит логику парсинга конкретного магазина DNS
type DNSParser struct {
	logger *log.Logger
}

func NewStealthContext(parent context.Context, logger *log.Logger) (context.Context, context.CancelFunc) {
	// 1. Создаём allocator (браузер + флаги)
	allocCtx, cancelAlloc := chromedp.NewExecAllocator(
		parent,
		chromedp.ExecPath(`C:\Program Files\Google\Chrome\Application\chrome.exe`),
		chromedp.Flag("headless", false),
		chromedp.Flag("disable-gpu", true),
		chromedp.UserAgent(
			"Mozilla/5.0 (Windows NT 10.0; Win64; x64) "+
				"AppleWebKit/537.36 (KHTML, like Gecko) "+
				"Chrome/135.0.0.0 Safari/537.36",
		),
	)

	// 2. Новый контекст с CDP‑слушателями
	ctx, cancelCtx := chromedp.NewContext(
		allocCtx,
		chromedp.WithLogf(logger.Printf),
	)

	// 3. Устанавливаем таймаут на всё
	ctx, cancelTimeout := context.WithTimeout(ctx, 90*time.Second)

	// Вот здесь — ActionFunc, который внутри вызывает Do и игнорирует ScriptIdentifier:
	stealth := chromedp.ActionFunc(func(ctx context.Context) error {
		// включаем сеть, ставим доп. заголовки
		if err := network.Enable().Do(ctx); err != nil {
			return err
		}
		if err := network.SetExtraHTTPHeaders(network.Headers{
			"Accept-Language": "ru-RU,ru;q=0.9",
			"Accept":          "text/html,application/xhtml+xml,application/xml;q=0.9",
		}).Do(ctx); err != nil {
			return err
		}
		// инжектим скрипт перед любой загрузкой страницы
		js := `(() => {
            Object.defineProperty(navigator, 'webdriver', { get: () => false });
            window.chrome = { runtime: {} };
            Object.defineProperty(navigator, 'languages', {
              get: () => ['ru-RU', 'ru', 'en-US', 'en']
            });
            Object.defineProperty(navigator, 'plugins', {
              get: () => [1, 2, 3, 4, 5]
            });
        })();`
		// AddScriptToEvaluateOnNewDocumentParams.Do возвращает (ScriptIdentifier, error)
		// мы используем ActionFunc, чтобы вернуть только ошибку
		if _, err := page.AddScriptToEvaluateOnNewDocument(js).Do(ctx); err != nil {
			return err
		}
		return nil
	})

	// Прогоним готовый stealth‑action один раз, чтобы он установился
	if err := chromedp.Run(ctx, stealth); err != nil {
		logger.Fatalf("failed to install stealth scripts: %v", err)
	}

	// Объединяем все cancel’ы в один
	cancel := func() {
		cancelTimeout()
		cancelCtx()
		cancelAlloc()
	}
	return ctx, cancel
}

// NewDNSParser конструктор (может принимать иные параметры)
func NewDNSParser(logger *log.Logger) *DNSParser {
	return &DNSParser{logger: logger}
}

func (p *DNSParser) ParseProductPage(_ctx context.Context, url string) (*ParsedItem, error) {
	logger := p.logger

	// Вместо chromedp.NewContext делаем
	ctx, cancel := NewStealthContext(_ctx, logger)
	defer cancel()

	// 2. Случайная задержка (имитация человека, если нужно)
	pause := time.Duration(rand.Intn(5)+5) * time.Second // от 5 до 10 сек
	time.Sleep(pause)

	// 3. Выполняем сценарий: зайти на url, дождаться загрузки, взять HTML
	var html string
	err := chromedp.Run(ctx,
		chromedp.Navigate(url),
		chromedp.WaitVisible(`div.product-buy__price`, chromedp.ByQuery),
		chromedp.OuterHTML("html", &html),
	)
	if err != nil {
		return nil, fmt.Errorf("chromedp run error: %w", err)
	}

	// 4. Парсим HTML через goquery
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, fmt.Errorf("goquery parse error: %w", err)
	}

	// или короче:
	// doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))

	// 5. Извлекаем нужные данные
	// Примерно по тем же селекторам, что в Python
	item := &ParsedItem{}

	title := doc.Find("div.product-card-description__title").First().Text()
	item.Name = title

	price := doc.Find("div.product-buy__price").First().Text()
	item.Price = price

	desc := doc.Find("div.product-card-description-text").First().Text()
	item.Description = desc

	availability := doc.Find("a.order-avail-wrap__link.ui-link.ui-link_blue").First().Text()
	if availability == "" {
		availability = "Товара нет в наличии"
	}
	item.Availability = availability

	// Пример получения главной картинки:
	mainPic, _ := doc.Find("img.product-images-slider__main-img").Attr("src")
	item.MainImage = mainPic

	// Пример парсинга списка картинок
	var pictures []string
	doc.Find("img.product-images-slider__img.loaded.tns-complete").Each(func(i int, s *goquery.Selection) {
		if src, ok := s.Attr("data-src"); ok {
			pictures = append(pictures, src)
		}
	})
	item.Images = pictures

	// Категорию можно искать, например:
	category := "Категория не найдена"
	doc.Find("span").Each(func(i int, s *goquery.Selection) {
		// Логика поиска
		if goquery.NodeName(s) == "span" && s.AttrOr("data-go-back-catalog", "") != "" {
			category = s.Text()
		}
	})
	item.Category = category

	// Пример характеристики:
	var specs []KV
	doc.Find("div.product-characteristics__spec-title").Each(func(i int, s *goquery.Selection) {
		specTitle := s.Text()
		specValue := doc.Find("div.product-characteristics__spec-value").Eq(i).Text()
		specs = append(specs, KV{Key: specTitle, Value: specValue})
	})
	item.Characteristics = specs

	// вернуть результат
	return item, nil
}

// После уже существующего ParseProductPage(...)
func (p *DNSParser) Parse(ctx context.Context, url string) (*usecase.ParsedItem, error) {
	// просто делегируем, но возвращаем нужный usecase.ParsedItem
	prod, err := p.ParseProductPage(ctx, url)
	if err != nil {
		return nil, err
	}
	return &usecase.ParsedItem{
		Price:        prod.Price,
		Availability: prod.Availability,
		URL:          url,
	}, nil
}

// ParsedItem структура для хранения результата
type ParsedItem struct {
	Name            string
	Price           string
	Description     string
	Availability    string
	Category        string
	MainImage       string
	Images          []string
	Characteristics []KV
}

// Пример для хранения ключ-значение:
type KV struct {
	Key   string
	Value string
}
