package browser

import (
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
)

// Browser 封装 rod.Browser 实例
type Browser struct {
	browser  *rod.Browser
	launcher *launcher.Launcher
	proxyURL string // 代理URL
}

// NewBrowser 创建并初始化新的浏览器实例（headless 模式）
func NewBrowser() (*Browser, error) {
	return newBrowser("", true)
}

// NewBrowserWithProxy 创建带代理的 headless 浏览器实例
func NewBrowserWithProxy(proxyURL string) (*Browser, error) {
	return newBrowser(proxyURL, true)
}

// NewBrowserHeaded 创建有界面的浏览器实例
func NewBrowserHeaded(proxyURL string) (*Browser, error) {
	return newBrowser(proxyURL, false)
}

// NewBrowserWithOptions 创建浏览器实例，showUI=true 时显示界面
func NewBrowserWithOptions(proxyURL string, showUI bool) (*Browser, error) {
	return newBrowser(proxyURL, !showUI)
}

func newBrowser(proxyURL string, headless bool) (*Browser, error) {
	l := launcher.New().Headless(headless)

	if proxyURL != "" {
		l = l.Proxy(proxyURL)
	}

	url := l.MustLaunch()
	browser := rod.New().ControlURL(url).MustConnect()

	b := &Browser{
		browser:  browser,
		launcher: l,
		proxyURL: proxyURL,
	}

	return b, nil
}

// GetProxyURL 获取当前使用的代理URL
func (b *Browser) GetProxyURL() string {
	return b.proxyURL
}

// NewPage 创建新的浏览器页面
func (b *Browser) NewPage() (*rod.Page, error) {
	page, err := b.browser.Page(proto.TargetCreateTarget{})
	if err != nil {
		return nil, err
	}
	return page, nil
}

// Close 关闭浏览器并清理资源
func (b *Browser) Close() error {
	if b.browser != nil {
		if err := b.browser.Close(); err != nil {
			return err
		}
	}
	if b.launcher != nil {
		b.launcher.Kill()
	}
	return nil
}
