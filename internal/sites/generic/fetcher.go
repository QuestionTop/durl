package generic

import (
	"encoding/json"
	"fmt"
	"time"

	"durl/internal/browser"

	"github.com/go-rod/rod"
)

// WaitStrategy wait strategy type
type WaitStrategy string

const (
	WaitStrategyLoad    WaitStrategy = "load"    // Wait for page to fully load
	WaitStrategyElement WaitStrategy = "element" // Wait for specific element to appear
	WaitStrategyTime    WaitStrategy = "time"    // Wait for fixed time
)

// FetchResult fetch result
type FetchResult struct {
	Page     *rod.Page     // Page object
	Title    string        // Page title
	URL      string        // Final URL
	LoadTime time.Duration // Load time
}

// Fetcher page fetcher
type Fetcher struct {
	browser *browser.Browser
}

// NewFetcher creates a new Fetcher instance
func NewFetcher(browser *browser.Browser) *Fetcher {
	return &Fetcher{
		browser: browser,
	}
}

// SetBrowser sets the browser instance used by Fetcher
func (f *Fetcher) SetBrowser(browser *browser.Browser) {
	f.browser = browser
}

// Fetch executes page fetching
// url: target URL
// method: HTTP method (GET, POST, PUT, DELETE, etc.)
// headers: request header map
// body: request body (for POST/PUT methods)
// waitStrategy: wait strategy (load/element/time)
// waitTarget: wait target (selector for element strategy or milliseconds for time strategy)
// timeout: timeout duration
func (f *Fetcher) Fetch(url, method string, headers map[string]string, body string, waitStrategy WaitStrategy, waitTarget string, timeout time.Duration) (*FetchResult, error) {
	startTime := time.Now()

	// Create new page (no timeout to avoid affecting subsequent operations)
	page, err := f.browser.NewPage()
	if err != nil {
		return nil, fmt.Errorf("failed to create page: %w", err)
	}

	// Set request headers
	if len(headers) > 0 {
		// Convert map[string]string to []string format
		headerList := make([]string, 0, len(headers)*2)
		for k, v := range headers {
			headerList = append(headerList, k, v)
		}
		cleanup, err := page.SetExtraHeaders(headerList)
		if err != nil {
			page.Close()
			return nil, fmt.Errorf("failed to set headers: %w", err)
		}
		defer cleanup()
	}

	// Execute HTTP request
	switch method {
	case "GET":
		if err := page.Timeout(timeout).Navigate(url); err != nil {
			page.Close()
			return nil, fmt.Errorf("failed to navigate: %w", err)
		}
	case "POST", "PUT", "DELETE", "PATCH":
		// For methods requiring request body, use JavaScript fetch API
		// Then write response content to page
		var responseText string
		if body != "" {
			// Case with request body
			result, err := page.Timeout(timeout).Eval(fmt.Sprintf(`() => {
				return fetch('%s', {
					method: '%s',
					headers: %s,
					body: %s
				}).then(r => r.text());
			}`, url, method, headersToJS(headers), stringToJS(body)))
			if err != nil {
				page.Close()
				return nil, fmt.Errorf("failed to execute %s request: %w", method, err)
			}
			responseText = page.MustObjectToJSON(result).String()
		} else {
			// Case without request body
			result, err := page.Timeout(timeout).Eval(fmt.Sprintf(`() => {
				return fetch('%s', {
					method: '%s',
					headers: %s
				}).then(r => r.text());
			}`, url, method, headersToJS(headers)))
			if err != nil {
				page.Close()
				return nil, fmt.Errorf("failed to execute %s request: %w", method, err)
			}
			responseText = page.MustObjectToJSON(result).String()
		}

		// Write response content to page
		_, err := page.Eval(fmt.Sprintf(`() => {
			document.open();
			document.write(%s);
			document.close();
		}`, stringToJS(responseText)))
		if err != nil {
			page.Close()
			return nil, fmt.Errorf("failed to write response to page: %w", err)
		}
	case "HEAD", "OPTIONS":
		// For HEAD and OPTIONS, use JavaScript fetch API
		// These methods don't return response body, only response headers
		result, err := page.Timeout(timeout).Eval(fmt.Sprintf(`() => {
			return fetch('%s', {
				method: '%s',
				headers: %s
			}).then(r => {
				// Convert response header information to text
				let headersText = '';
				r.headers.forEach((value, key) => {
					headersText += key + ': ' + value + '\\n';
				});
				return headersText || 'No headers returned';
			});
		}`, url, method, headersToJS(headers)))
		if err != nil {
			page.Close()
			return nil, fmt.Errorf("failed to execute %s request: %w", method, err)
		}

		// Write response header information to page
		headersText := page.MustObjectToJSON(result).String()
		_, err = page.Eval(fmt.Sprintf(`() => {
			document.open();
			document.write(%s);
			document.close();
		}`, stringToJS(headersText)))
		if err != nil {
			page.Close()
			return nil, fmt.Errorf("failed to write response to page: %w", err)
		}
	default:
		// Default to GET
		if err := page.Timeout(timeout).Navigate(url); err != nil {
			page.Close()
			return nil, fmt.Errorf("failed to navigate: %w", err)
		}
	}

	// Apply wait strategy
	if err := f.applyWaitStrategy(page, waitStrategy, waitTarget); err != nil {
		page.Close()
		return nil, fmt.Errorf("wait strategy failed: %w", err)
	}

	// Get page metadata
	title, err := page.Eval(`() => document.title`)
	if err != nil {
		page.Close()
		return nil, fmt.Errorf("failed to get page title: %w", err)
	}

	// Get final URL
	finalURL := page.MustInfo().URL

	// Calculate load time
	loadTime := time.Since(startTime)

	result := &FetchResult{
		Page:     page,
		Title:    page.MustObjectToJSON(title).String(),
		URL:      finalURL,
		LoadTime: loadTime,
	}

	return result, nil
}

// applyWaitStrategy applies wait strategy
func (f *Fetcher) applyWaitStrategy(page *rod.Page, strategy WaitStrategy, target string) error {
	switch strategy {
	case WaitStrategyLoad:
		// Wait for page to fully load
		if err := page.WaitLoad(); err != nil {
			return fmt.Errorf("failed to wait for page load: %w", err)
		}

	case WaitStrategyElement:
		// Wait for specific element to appear
		if target == "" {
			return fmt.Errorf("wait target is required for element strategy")
		}
		if _, err := page.Element(target); err != nil {
			return fmt.Errorf("failed to wait for element '%s': %w", target, err)
		}

	case WaitStrategyTime:
		// Wait for fixed time
		if target == "" {
			return fmt.Errorf("wait target is required for time strategy")
		}
		duration, err := time.ParseDuration(target + "ms")
		if err != nil {
			return fmt.Errorf("invalid wait time '%s': %w", target, err)
		}
		time.Sleep(duration)

	default:
		// Default to wait for page load
		if err := page.WaitLoad(); err != nil {
			return fmt.Errorf("failed to wait for page load: %w", err)
		}
	}

	return nil
}

// headersToJS converts request header map to JavaScript object string
func headersToJS(headers map[string]string) string {
	if len(headers) == 0 {
		return "{}"
	}
	result := "{"
	for k, v := range headers {
		result += fmt.Sprintf(`"%s": "%s",`, k, v)
	}
	result = result[:len(result)-1] + "}"
	return result
}

// stringToJS converts a Go string to a JavaScript string literal using JSON encoding
// to correctly escape all special characters.
func stringToJS(s string) string {
	quoted, _ := json.Marshal(s)
	return string(quoted)
}
