package bing

import (
	"context"
	"fmt"
	"net/url"

	"durl/internal/browser"
	"durl/internal/scraper"
)

func init() {
	scraper.Register(&BingScraper{})
}

// BingScraper searches cn.bing.com and returns the result list.
type BingScraper struct{}

func (s *BingScraper) Name() string { return "bing" }

func (s *BingScraper) Scrape(ctx context.Context, query string, opts scraper.Options) (scraper.Content, error) {
	if query == "" {
		return nil, fmt.Errorf("query is required for --site bing")
	}

	searchURL := "https://cn.bing.com/search?q=" + url.QueryEscape(query) + "&PC=U316&FORM=CHROMN"

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
		return nil, fmt.Errorf("failed to search bing: %w", err)
	}

	return NewBingContent(query, searchURL, results), nil
}
