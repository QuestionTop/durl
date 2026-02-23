package generic

import (
	"fmt"
	"strings"
	"time"

	"github.com/go-rod/rod"
)

// Extractor content extractor
type Extractor struct {
	page *rod.Page
}

// NewExtractor creates a new Extractor instance
func NewExtractor(page *rod.Page) *Extractor {
	return &Extractor{
		page: page,
	}
}

// Extract extracts content based on level
// level: extraction level (full/html/body/content/xpath/css)
// selector: selector (only for xpath and css levels)
func (e *Extractor) Extract(level, selector string) (string, error) {
	switch level {
	case "full":
		return e.extractFull()
	case "html":
		return e.extractHTML()
	case "body":
		return e.extractBody()
	case "content":
		return e.extractContent()
	case "xpath":
		return e.extractByXPath(selector)
	case "css":
		return e.extractByCSS(selector)
	default:
		return "", fmt.Errorf("unsupported level: %s", level)
	}
}

// extractFull extracts complete HTML document (including head)
func (e *Extractor) extractFull() (string, error) {
	// Use JavaScript to get complete HTML document, including DOCTYPE
	result, err := e.page.Timeout(10 * time.Second).Eval(`() => {
		return document.documentElement.outerHTML;
	}`)
	if err != nil {
		return "", fmt.Errorf("failed to get full HTML: %w", err)
	}

	html := e.page.MustObjectToJSON(result).String()
	// Add DOCTYPE declaration (if not exists)
	if !strings.Contains(html, "<!DOCTYPE") {
		html = "<!DOCTYPE html>\n" + html
	}
	return html, nil
}

// extractHTML extracts all HTML tags within body (including script and style tags)
func (e *Extractor) extractHTML() (string, error) {
	// Use JavaScript to extract body content
	result, err := e.page.Timeout(10 * time.Second).Eval(`() => {
		const body = document.body;
		return body ? body.innerHTML : '';
	}`)
	if err != nil {
		return "", fmt.Errorf("failed to extract body HTML: %w", err)
	}

	bodyHTML := e.page.MustObjectToJSON(result).String()
	return bodyHTML, nil
}

// extractBody extracts plain text content within body
func (e *Extractor) extractBody() (string, error) {
	result, err := e.page.Eval(`() => document.body.innerText`)
	if err != nil {
		return "", fmt.Errorf("failed to get body text: %w", err)
	}

	text := e.page.MustObjectToJSON(result).String()
	return text, nil
}

// extractContent intelligently identifies main content (Readability+selector+fallback strategy)
func (e *Extractor) extractContent() (string, error) {
	// Strategy 1: Try to use Readability algorithm
	hasReadability, err := e.page.Timeout(5 * time.Second).Eval(`() => typeof window.readability !== 'undefined'`)
	if err == nil && e.page.MustObjectToJSON(hasReadability).Bool() {
		result, err := e.page.Timeout(5 * time.Second).Eval(`() => {
			const article = new Readability(document).parse();
			return article ? article.content : '';
		}`)
		if err == nil {
			content := e.page.MustObjectToJSON(result).String()
			if content != "" {
				return content, nil
			}
		}
	}

	// Strategy 2: Try common content selectors
	selectors := []string{"article", "main", ".content", ".article", ".post", ".entry-content"}
	for _, sel := range selectors {
		element, err := e.page.Timeout(1 * time.Second).Element(sel)
		if err == nil {
			html, err := element.HTML()
			if err == nil && html != "" {
				return html, nil
			}
		}
	}

	// Strategy 3: Fallback to body content
	html, err := e.extractHTML()
	if err != nil {
		return "", fmt.Errorf("failed to extract content: %w", err)
	}

	return html, nil
}

// extractByXPath extracts content using XPath selector
func (e *Extractor) extractByXPath(xpath string) (string, error) {
	elements, err := e.page.Timeout(10 * time.Second).ElementsX(xpath)
	if err != nil {
		return "", fmt.Errorf("failed to query XPath: %w", err)
	}

	if len(elements) == 0 {
		return "", nil
	}

	var content string
	for i, element := range elements {
		html, err := element.HTML()
		if err != nil {
			return "", fmt.Errorf("failed to get element HTML: %w", err)
		}
		if i > 0 {
			content += "\n"
		}
		content += html
	}

	return content, nil
}

// extractByCSS extracts content using CSS selector
func (e *Extractor) extractByCSS(selector string) (string, error) {
	elements, err := e.page.Timeout(10 * time.Second).Elements(selector)
	if err != nil {
		return "", fmt.Errorf("failed to query CSS selector: %w", err)
	}

	if len(elements) == 0 {
		return "", nil
	}

	var content string
	for i, element := range elements {
		html, err := element.HTML()
		if err != nil {
			return "", fmt.Errorf("failed to get element HTML: %w", err)
		}
		if i > 0 {
			content += "\n"
		}
		content += html
	}

	return content, nil
}
