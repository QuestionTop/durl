package generic

import (
	"context"
	"fmt"
	"time"

	"durl/internal/browser"
	"durl/internal/scraper"
	"github.com/go-rod/rod/lib/proto"
)

// GenericScraper generic scraper
type GenericScraper struct {
	cfg browser.Config
}

// NewGenericScraper creates generic scraper instance
func NewGenericScraper(cfg browser.Config) *GenericScraper {
	return &GenericScraper{cfg: cfg}
}

// Name returns scraper name
func (g *GenericScraper) Name() string {
	return "generic"
}

// Scrape fetches the page and extracts all content before closing the browser.
// PageContent is returned with pre-extracted strings so it does not require
// a live browser connection during formatting.
func (g *GenericScraper) Scrape(ctx context.Context, target string, opts scraper.Options) (scraper.Content, error) {
	b, err := browser.New(g.cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create browser: %w", err)
	}
	defer b.Close()

	f := NewFetcher(b)
	result, err := f.Fetch(target, opts.Method, opts.Headers, opts.Body, WaitStrategy(opts.WaitFor), opts.WaitTarget, opts.Timeout)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch: %w", err)
	}
	defer result.Page.Close()

	// When the default "load" wait strategy is used, additionally wait for
	// network idle so that JS-driven pages (e.g. Bing/Google SPA search) finish
	// populating dynamic content before we extract.
	if opts.WaitFor == "load" {
		wait := result.Page.Timeout(opts.Timeout).WaitRequestIdle(
			500*time.Millisecond, nil, nil,
			[]proto.NetworkResourceType{proto.NetworkResourceTypeImage, proto.NetworkResourceTypeMedia},
		)
		wait()
	}

	// Extract all content while the browser is still open.
	// PageContent must not hold a live page reference because the browser is
	// closed (via defer b.Close()) before the formatter calls ToHTML/ToMarkdown/etc.
	extractor := NewExtractor(result.Page)

	var htmlContent, mainContent, textContent string

	if opts.Level == "body" {
		// ToHTML() needs body innerHTML; ToText() needs body innerText
		htmlContent, err = extractor.Extract("html", "")
		if err != nil {
			return nil, fmt.Errorf("failed to extract HTML content: %w", err)
		}
		textContent, err = extractor.Extract("body", "")
		if err != nil {
			return nil, fmt.Errorf("failed to extract text content: %w", err)
		}
		// ToMarkdown/ToCSV/ToJSON use the same HTML as ToHTML for body level
		mainContent = htmlContent
	} else {
		// For all other levels (full, html, content, xpath, css),
		// the same extraction serves HTML, markdown, CSV, JSON and text (after conversion).
		mainContent, err = extractor.Extract(opts.Level, opts.Selector)
		if err != nil {
			return nil, fmt.Errorf("failed to extract content: %w", err)
		}
		htmlContent = mainContent
		textContent = mainContent
	}

	return NewPageContent(htmlContent, mainContent, textContent, opts.Level, result.Title, result.URL, result.LoadTime), nil
}
