package bing

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"durl/internal/browser"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
)

// Result holds a single Bing search result.
type Result struct {
	Title   string
	URL     string
	Snippet string
}

// Client is a browser client for Bing search.
type Client struct {
	browser *browser.Browser
	page    *rod.Page
}

// NewClient creates a new Client instance.
func NewClient(b *browser.Browser) *Client {
	return &Client{browser: b}
}

// Close closes the page.
func (c *Client) Close() {
	if c.page != nil {
		c.page.Close()
	}
}

// Search navigates to searchURL and extracts results from #b_results.
func (c *Client) Search(ctx context.Context, searchURL string, timeout time.Duration) ([]Result, error) {
	page, err := c.browser.NewPage()
	if err != nil {
		return nil, fmt.Errorf("failed to create page: %w", err)
	}
	c.page = page

	_ = page.SetUserAgent(&proto.NetworkSetUserAgentOverride{
		UserAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	})
	_, _ = page.EvalOnNewDocument(`Object.defineProperty(navigator, 'webdriver', {get: () => undefined});`)

	if err := page.Timeout(timeout).Navigate(searchURL); err != nil {
		return nil, fmt.Errorf("failed to navigate: %w", err)
	}

	if err := page.Timeout(timeout).WaitLoad(); err != nil {
		return nil, fmt.Errorf("failed to wait for page load: %w", err)
	}

	// Wait for network idle so JS-rendered results are populated.
	wait := page.Timeout(timeout).WaitRequestIdle(
		500*time.Millisecond, nil, nil,
		[]proto.NetworkResourceType{proto.NetworkResourceTypeImage, proto.NetworkResourceTypeMedia},
	)
	wait()

	// Extract all results via a single JS evaluation to avoid per-element
	// visibility issues that cause rod's Text() / Property() to return empty.
	val, err := page.Timeout(10 * time.Second).Eval(`() => {
		const items = document.querySelectorAll('#b_results li.b_algo');
		return Array.from(items).map(li => {
			const a = li.querySelector('h2 a');
			const p = li.querySelector('.b_caption p') || li.querySelector('p');
			return {
				title:   a ? a.textContent.trim() : '',
				url:     a ? (a.getAttribute('href') || '') : '',
				snippet: p ? p.textContent.trim() : '',
			};
		}).filter(r => r.title !== '');
	}`)
	if err != nil {
		return nil, fmt.Errorf("failed to extract results: %w", err)
	}

	type jsResult struct {
		Title   string `json:"title"`
		URL     string `json:"url"`
		Snippet string `json:"snippet"`
	}
	var jsResults []jsResult
	raw, _ := val.Value.MarshalJSON()
	if err := json.Unmarshal(raw, &jsResults); err != nil {
		return nil, fmt.Errorf("failed to parse results: %w", err)
	}

	results := make([]Result, 0, len(jsResults))
	for _, r := range jsResults {
		results = append(results, Result{Title: r.Title, URL: r.URL, Snippet: r.Snippet})
	}

	return results, nil
}
