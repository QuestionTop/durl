package xueqiu

import (
	"context"
	"fmt"
	"regexp"

	"durl/internal/browser"
	"durl/internal/scraper"
)

func init() {
	scraper.Register(&XueqiuFinReportScraper{})
}

// XueqiuFinReportScraper financial report scraper
type XueqiuFinReportScraper struct{}

// Name returns site name
func (x *XueqiuFinReportScraper) Name() string {
	return "xueqiu.finreport"
}

// Scrape executes financial report scraping
func (x *XueqiuFinReportScraper) Scrape(ctx context.Context, target string, opts scraper.Options) (scraper.Content, error) {
	b, err := browser.New(browser.Config{
		ProxyURL: opts.ProxyURL,
		Headless: !opts.ShowUI,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create browser: %w", err)
	}
	defer b.Close()

	client := NewFinReportClient(b)

	var code, stockName string

	if regexp.MustCompile(`xueqiu\.com`).MatchString(target) {
		// Extract stock code from URL
		m := regexp.MustCompile(`/S/([A-Z0-9]+)/`).FindStringSubmatch(target)
		if m == nil {
			return nil, fmt.Errorf("failed to extract stock code from URL: %s", target)
		}
		code = m[1]
	} else {
		// Need a page to search stock code, borrow initPage to initialize a temporary page
		worker, err := client.initPage(opts.Timeout)
		if err != nil {
			return nil, fmt.Errorf("failed to init page for stock code resolution: %w", err)
		}
		defer worker.page.Close()

		code, stockName, err = ResolveStockCode(target, worker.page)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve stock code: %w", err)
		}
	}

	tables, err := client.FetchAllReports(code, opts.Timeout)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch reports: %w", err)
	}

	return NewFinReportContent(code, stockName, tables), nil
}
