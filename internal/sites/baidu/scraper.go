package baidu

import (
	"context"
	"fmt"
	"net/url"

	"durl/internal/browser"
	"durl/internal/scraper"
)

func init() {
	scraper.Register(&BaiduScraper{})
}

// BaiduScraper searches www.baidu.com and returns the result list.
type BaiduScraper struct{}

func (s *BaiduScraper) Name() string { return "baidu" }

func (s *BaiduScraper) Scrape(ctx context.Context, query string, opts scraper.Options) (scraper.Content, error) {
	if query == "" {
		return nil, fmt.Errorf("query is required for --site baidu")
	}

	searchURL := "https://www.baidu.com/s?wd=" + url.QueryEscape(query)

	b, err := browser.New(browser.Config{
		ProxyURL: opts.ProxyURL,
		Headless: !opts.ShowUI,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create browser: %w", err)
	}
	defer b.Close()

	client := NewClient(b)
	defer client.Close()

	results, err := client.Search(ctx, searchURL, opts.Timeout)
	if err != nil {
		return nil, fmt.Errorf("failed to search baidu: %w", err)
	}

	return NewBaiduContent(query, searchURL, results), nil
}
