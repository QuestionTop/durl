package extractor

import (
	"fmt"
	"strings"
	"time"

	"github.com/go-rod/rod"
)

// Extractor 内容提取器
type Extractor struct {
	page *rod.Page
}

// NewExtractor 创建新的 Extractor 实例
func NewExtractor(page *rod.Page) *Extractor {
	return &Extractor{
		page: page,
	}
}

// Extract 根据级别提取内容
// level: 提取级别 (full/html/body/content/xpath/css)
// selector: 选择器 (仅用于 xpath 和 css 级别)
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

// extractFull 提取完整HTML文档（包含head）
func (e *Extractor) extractFull() (string, error) {
	// 使用 JavaScript 获取完整的 HTML 文档，包括 DOCTYPE
	result, err := e.page.Timeout(10 * time.Second).Eval(`() => {
		return document.documentElement.outerHTML;
	}`)
	if err != nil {
		return "", fmt.Errorf("failed to get full HTML: %w", err)
	}

	html := e.page.MustObjectToJSON(result).String()
	// 添加 DOCTYPE 声明（如果不存在）
	if !strings.Contains(html, "<!DOCTYPE") {
		html = "<!DOCTYPE html>\n" + html
	}
	return html, nil
}

// extractHTML 提取body内的所有HTML标签（包含script和style标签）
func (e *Extractor) extractHTML() (string, error) {
	// 使用 JavaScript 提取 body 内容
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

// extractBody 提取body内的纯文本内容
func (e *Extractor) extractBody() (string, error) {
	result, err := e.page.Eval(`() => document.body.innerText`)
	if err != nil {
		return "", fmt.Errorf("failed to get body text: %w", err)
	}

	text := e.page.MustObjectToJSON(result).String()
	return text, nil
}

// extractContent 智能识别正文内容（Readability+选择器+降级策略）
func (e *Extractor) extractContent() (string, error) {
	// 策略1：尝试使用 Readability 算法
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

	// 策略2：尝试常见内容选择器
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

	// 策略3：降级到 body 内容
	html, err := e.extractHTML()
	if err != nil {
		return "", fmt.Errorf("failed to extract content: %w", err)
	}

	return html, nil
}

// extractByXPath 使用 XPath 选择器提取内容
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

// extractByCSS 使用 CSS 选择器提取内容
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
