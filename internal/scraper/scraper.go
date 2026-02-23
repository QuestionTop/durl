package scraper

import (
	"context"
	"time"
)

type Scraper interface {
	Name() string
	Scrape(ctx context.Context, target string, opts Options) (Content, error)
}

type Content interface {
	ToHTML() (string, error)
	ToText() (string, error)
	ToMarkdown() (string, error)
	ToJSON() ([]byte, error)
	ToCSV() (string, error)
}

type Options struct {
	Method     string
	Headers    map[string]string
	Body       string
	WaitFor    string
	WaitTarget string
	Timeout    time.Duration
	Level      string // full/html/body/content/xpath/css
	Selector   string
	ShowUI     bool
	ProxyURL   string            // --proxy flag or DURL_PROXY env var
	Extra      map[string]string // Site-specific parameters (last-days/max-pages/sort, etc.)
}
