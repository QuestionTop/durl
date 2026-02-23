package xueqiu

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"durl/internal/browser"
	"durl/internal/scraper"
)

func init() {
	scraper.Register(&XueqiuScraper{})
}

// XueqiuScraper implements scraper.Scraper interface
type XueqiuScraper struct{}

// Name returns site name
func (x *XueqiuScraper) Name() string {
	return "xueqiu.comment"
}

// Scrape executes Xueqiu scraping
func (x *XueqiuScraper) Scrape(ctx context.Context, target string, opts scraper.Options) (scraper.Content, error) {
	// Read parameters from opts.Extra
	lastStr := "30d"
	if v, ok := opts.Extra["last"]; ok && v != "" {
		lastStr = v
	}
	cutoff, err := ParseLast(lastStr)
	if err != nil {
		return nil, fmt.Errorf("invalid --last: %w", err)
	}

	maxPages := -1
	if v, ok := opts.Extra["max-pages"]; ok {
		if n, err := strconv.Atoi(v); err == nil {
			maxPages = n
		}
	}

	sort := "hot"
	if v, ok := opts.Extra["sort"]; ok && v != "" {
		sort = v
	}

	// Create browser
	b, err := browser.New(browser.Config{
		ProxyURL: opts.ProxyURL,
		Headless: !opts.ShowUI,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create browser: %w", err)
	}
	defer b.Close()

	// Create Xueqiu client
	client := NewClient(b)
	defer client.Close()

	// Initialize Xueqiu session
	if err := client.Init(opts.Timeout); err != nil {
		return nil, fmt.Errorf("failed to init xueqiu client: %w", err)
	}

	var discussions []Discussion
	var title string

	// Check if target is Xueqiu URL
	if strings.Contains(target, "xueqiu.com") {
		discussions, err = client.FetchByURL(target, cutoff, sort, maxPages)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch discussions: %w", err)
		}
		title = target
	} else {
		// Resolve stock code
		code, name, err := client.ResolveStockCode(target)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve stock code: %w", err)
		}
		discussions, err = client.FetchDiscussions(code, cutoff, sort, maxPages)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch discussions: %w", err)
		}
		if name != "" {
			title = name + " (" + code + ")"
		} else {
			title = code
		}
	}

	return NewXueqiuContent(discussions, title, cutoff), nil
}
