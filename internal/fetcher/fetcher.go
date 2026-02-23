package fetcher

import (
	"encoding/json"
	"fmt"
	"time"

	"durl/internal/browser"

	"github.com/go-rod/rod"
)

// WaitStrategy 等待策略类型
type WaitStrategy string

const (
	WaitStrategyLoad    WaitStrategy = "load"    // 等待页面完全加载
	WaitStrategyElement WaitStrategy = "element" // 等待特定元素出现
	WaitStrategyTime    WaitStrategy = "time"    // 等待固定时间
)

// FetchResult 抓取结果
type FetchResult struct {
	Page     *rod.Page     // 页面对象
	Title    string        // 页面标题
	URL      string        // 最终URL
	LoadTime time.Duration // 加载时间
}

// Fetcher 页面抓取器
type Fetcher struct {
	browser *browser.Browser
}

// NewFetcher 创建新的 Fetcher 实例
func NewFetcher(browser *browser.Browser) *Fetcher {
	return &Fetcher{
		browser: browser,
	}
}

// SetBrowser 设置 Fetcher 使用的浏览器实例
func (f *Fetcher) SetBrowser(browser *browser.Browser) {
	f.browser = browser
}

// Fetch 执行页面抓取
// url: 目标URL
// method: HTTP方法 (GET, POST, PUT, DELETE等)
// headers: 请求头映射
// body: 请求体 (对于POST/PUT等方法)
// waitStrategy: 等待策略 (load/element/time)
// waitTarget: 等待目标 (element策略的选择器或time策略的毫秒数)
// timeout: 超时时间
func (f *Fetcher) Fetch(url, method string, headers map[string]string, body string, waitStrategy WaitStrategy, waitTarget string, timeout time.Duration) (*FetchResult, error) {
	startTime := time.Now()

	// 创建新页面（不设置超时，避免影响后续操作）
	page, err := f.browser.NewPage()
	if err != nil {
		return nil, fmt.Errorf("failed to create page: %w", err)
	}

	// 设置请求头
	if len(headers) > 0 {
		// 将 map[string]string 转换为 []string 格式
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

	// 执行HTTP请求
	switch method {
	case "GET":
		if err := page.Timeout(timeout).Navigate(url); err != nil {
			page.Close()
			return nil, fmt.Errorf("failed to navigate: %w", err)
		}
	case "POST", "PUT", "DELETE", "PATCH":
		// 对于需要请求体的方法，使用 JavaScript fetch API
		// 然后将响应内容写入页面
		var responseText string
		if body != "" {
			// 有请求体的情况
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
			// 无请求体的情况
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

		// 将响应内容写入页面
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
		// 对于 HEAD 和 OPTIONS，使用 JavaScript fetch API
		// 这些方法不返回响应体，只返回响应头
		result, err := page.Timeout(timeout).Eval(fmt.Sprintf(`() => {
			return fetch('%s', {
				method: '%s',
				headers: %s
			}).then(r => {
				// 将响应头信息转换为文本
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

		// 将响应头信息写入页面
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
		// 默认使用 GET
		if err := page.Timeout(timeout).Navigate(url); err != nil {
			page.Close()
			return nil, fmt.Errorf("failed to navigate: %w", err)
		}
	}

	// 应用等待策略
	if err := f.applyWaitStrategy(page, waitStrategy, waitTarget); err != nil {
		page.Close()
		return nil, fmt.Errorf("wait strategy failed: %w", err)
	}

	// 获取页面元数据
	title, err := page.Eval(`() => document.title`)
	if err != nil {
		page.Close()
		return nil, fmt.Errorf("failed to get page title: %w", err)
	}

	// 获取最终URL
	finalURL := page.MustInfo().URL

	// 计算加载时间
	loadTime := time.Since(startTime)

	result := &FetchResult{
		Page:     page,
		Title:    page.MustObjectToJSON(title).String(),
		URL:      finalURL,
		LoadTime: loadTime,
	}

	return result, nil
}

// applyWaitStrategy 应用等待策略
func (f *Fetcher) applyWaitStrategy(page *rod.Page, strategy WaitStrategy, target string) error {
	switch strategy {
	case WaitStrategyLoad:
		// 等待页面完全加载
		if err := page.WaitLoad(); err != nil {
			return fmt.Errorf("failed to wait for page load: %w", err)
		}

	case WaitStrategyElement:
		// 等待特定元素出现
		if target == "" {
			return fmt.Errorf("wait target is required for element strategy")
		}
		if _, err := page.Element(target); err != nil {
			return fmt.Errorf("failed to wait for element '%s': %w", target, err)
		}

	case WaitStrategyTime:
		// 等待固定时间
		if target == "" {
			return fmt.Errorf("wait target is required for time strategy")
		}
		duration, err := time.ParseDuration(target + "ms")
		if err != nil {
			return fmt.Errorf("invalid wait time '%s': %w", target, err)
		}
		time.Sleep(duration)

	default:
		// 默认等待页面加载
		if err := page.WaitLoad(); err != nil {
			return fmt.Errorf("failed to wait for page load: %w", err)
		}
	}

	return nil
}

// headersToJS 将请求头映射转换为 JavaScript 对象字符串
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

// stringToJS 将字符串转换为 JavaScript 字符串字面量
func stringToJS(s string) string {
	// 使用 JSON 编码来正确转义字符串
	// 将单引号替换为双引号，然后使用 JSON.Marshal 来转义
	quoted, _ := json.Marshal(s)
	return string(quoted)
}
