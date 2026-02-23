package xueqiu

import (
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"durl/internal/browser"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
)

type Discussion struct {
	ID           string
	Author       string
	Content      string
	CreatedAt    time.Time
	RelativeTime string
	ReplyCount   string
	LikeCount    string
	URL          string
}

type Client struct {
	browser *browser.Browser
	page    *rod.Page
}

func NewClient(b *browser.Browser) *Client {
	return &Client{browser: b}
}

func (c *Client) Close() {
	if c.page != nil {
		c.page.Close()
	}
}

func (c *Client) Init(timeout time.Duration) error {
	page, err := c.browser.NewPage()
	if err != nil {
		return fmt.Errorf("failed to create page: %w", err)
	}
	c.page = page

	_ = page.SetUserAgent(&proto.NetworkSetUserAgentOverride{
		UserAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	})
	_, _ = page.EvalOnNewDocument(`Object.defineProperty(navigator, 'webdriver', {get: () => undefined});`)

	_ = page.Timeout(timeout).Navigate("https://xueqiu.com")
	time.Sleep(4 * time.Second)

	if _, err := page.Timeout(10 * time.Second).Eval(`() => document.title`); err != nil {
		return fmt.Errorf("page not available: %w", err)
	}
	return nil
}

func (c *Client) ResolveStockCode(query string) (code string, name string, err error) {
	query = strings.TrimSpace(query)
	upper := strings.ToUpper(query)

	if regexp.MustCompile(`^(SZ|SH|HK)\d+$`).MatchString(upper) {
		return upper, "", nil
	}

	if regexp.MustCompile(`^\d{6}$`).MatchString(query) {
		prefix := "SH"
		ch := query[0]
		if ch == '0' || ch == '3' || ch == '1' {
			prefix = "SZ"
		}
		return prefix + query, "", nil
	}

	if regexp.MustCompile(`^\d{2,5}$`).MatchString(query) {
		padded := fmt.Sprintf("%05s", query)
		return padded, "", nil
	}

	return c.searchStockByPage(query)
}

func (c *Client) searchStockByPage(query string) (string, string, error) {
	searchURL := "https://xueqiu.com/k?q=" + url.QueryEscape(query)

	if err := c.page.Timeout(20 * time.Second).Navigate(searchURL); err != nil {
		return "", "", fmt.Errorf("failed to navigate to search: %w", err)
	}
	time.Sleep(4 * time.Second)

	result, err := c.page.Timeout(10 * time.Second).Eval(`() => {
		const items = [];
		document.querySelectorAll('table tr a, .stock-item a').forEach(function(a) {
			const href = a.href || '';
			const match = href.match(/\/S\/([A-Z0-9]+)$/);
			if (match) {
				const row = a.closest('tr') || a.closest('.stock-item');
				items.push({code: match[1], name: row ? row.innerText.substring(0,50) : a.innerText});
			}
		});
		return items;
	}`)
	if err != nil {
		return "", "", fmt.Errorf("failed to extract search results: %w", err)
	}

	type stockItem struct {
		Code string `json:"code"`
		Name string `json:"name"`
	}
	var items []stockItem
	if err := c.page.MustObjectToJSON(result).Unmarshal(&items); err != nil || len(items) == 0 {
		return "", "", fmt.Errorf("no stock found for: %s", query)
	}

	first := items[0]
	name := strings.Fields(first.Name)
	stockName := ""
	if len(name) > 0 {
		stockName = name[0]
	}
	return first.Code, stockName, nil
}

func (c *Client) FetchDiscussions(code string, days int, sort string, maxPages int) ([]Discussion, error) {
	return c.FetchByURL("https://xueqiu.com/S/"+code, days, sort, maxPages)
}

func (c *Client) FetchByURL(targetURL string, days int, sort string, maxPages int) ([]Discussion, error) {
	if !strings.HasPrefix(targetURL, "http") {
		targetURL = "https://" + targetURL
	}

	fmt.Fprintf(os.Stderr, "[xueqiu] navigating to %s\n", targetURL)
	if err := c.page.Timeout(30 * time.Second).Navigate(targetURL); err != nil {
		return nil, fmt.Errorf("failed to navigate to stock page: %w", err)
	}
	time.Sleep(5 * time.Second)

	if _, err := c.page.Timeout(15 * time.Second).Element(".timeline__item"); err != nil {
		return nil, fmt.Errorf("timeline not found on page: %w", err)
	}

	// 关闭登录弹窗（如果出现）
	_, _ = c.page.Eval(`() => {
		const mask = document.querySelector('.modal-mask, .login-modal, [class*=modal]');
		if (mask) mask.remove();
		const overlay = document.querySelector('.overlay');
		if (overlay) overlay.remove();
	}`)

	// 切换排序方式：new=新帖，hot=热帖（默认热帖）
	sortLabel := "热帖"
	if strings.ToLower(sort) == "new" {
		sortLabel = "新帖"
	}
	switched, _ := c.page.Timeout(5 * time.Second).Eval(fmt.Sprintf(`() => {
		const links = document.querySelectorAll('.sort-types a');
		for (let i = 0; i < links.length; i++) {
			if (links[i].innerText.trim() === %q) {
				links[i].click();
				return true;
			}
		}
		return false;
	}`, sortLabel))
	if switched != nil && c.page.MustObjectToJSON(switched).Bool() {
		fmt.Fprintf(os.Stderr, "[xueqiu] sort switched to: %s\n", sortLabel)
		time.Sleep(1500 * time.Millisecond)
	}

	unlimitedDays := days < 0
	var cutoff time.Time
	if !unlimitedDays {
		cutoff = time.Now().AddDate(0, 0, -days)
	}
	unlimitedPages := maxPages < 0
	var allDiscussions []Discussion
	seenIDs := make(map[string]bool)

	for pageNum := 1; unlimitedPages || pageNum <= maxPages; pageNum++ {
		fmt.Fprintf(os.Stderr, "[xueqiu] extracting page %d (total so far: %d)\n", pageNum, len(allDiscussions))

		items, reachedCutoff, err := c.extractVisibleItems(cutoff, seenIDs)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[xueqiu] extract err on page %d: %v\n", pageNum, err)
			break
		}
		allDiscussions = append(allDiscussions, items...)

		if !unlimitedDays && reachedCutoff {
			break
		}

		firstIDBefore := c.firstVisibleItemID()
		countBefore := c.countVisibleItems()
		action, err := c.advancePage()
		if err != nil || action == "none" {
			break
		}
		if action == "paginate" {
			if !c.waitForPageChange(firstIDBefore, 15*time.Second) {
				break
			}
		} else {
			if !c.waitForMoreItems(countBefore, 10*time.Second) {
				fmt.Fprintf(os.Stderr, "[xueqiu] no more items loaded after scroll (may require login)\n")
				break
			}
		}
	}

	return allDiscussions, nil
}

func (c *Client) advancePage() (string, error) {
	// 先尝试点击分页按钮，同时记录点击前的第一个 item ID 用于验证
	firstIDBefore := c.firstVisibleItemID()

	result, err := c.page.Timeout(5 * time.Second).Eval(`() => {
		const nextBtn = document.querySelector('.pagination__next');
		if (nextBtn && nextBtn.style.display !== 'none' && !nextBtn.classList.contains('disabled')) {
			nextBtn.click();
			return 'paginate';
		}
		window.scrollTo(0, document.body.scrollHeight);
		return 'scroll';
	}`)
	if err != nil {
		return "none", err
	}
	action := c.page.MustObjectToJSON(result).String()

	if action == "paginate" {
		// 等1.5秒看分页是否真的生效（firstID 变了）
		time.Sleep(1500 * time.Millisecond)
		if c.firstVisibleItemID() == firstIDBefore {
			// 分页按钮点击无效，降级为滚动
			fmt.Fprintf(os.Stderr, "[xueqiu] paginate had no effect, falling back to scroll\n")
			_, _ = c.page.Timeout(5 * time.Second).Eval(`() => window.scrollTo(0, document.body.scrollHeight)`)
			action = "scroll"
		}
	}

	fmt.Fprintf(os.Stderr, "[xueqiu] advance: %s\n", action)
	return action, nil
}

func (c *Client) firstVisibleItemID() string {
	result, err := c.page.Timeout(5 * time.Second).Eval(`() => {
		const el = document.querySelector('.timeline__item a.date-and-source[data-id]');
		return el ? el.getAttribute('data-id') : '';
	}`)
	if err != nil {
		return ""
	}
	return c.page.MustObjectToJSON(result).String()
}

func (c *Client) countVisibleItems() int {
	result, err := c.page.Timeout(5 * time.Second).Eval(`() => document.querySelectorAll('.timeline__item').length`)
	if err != nil {
		return 0
	}
	return int(c.page.MustObjectToJSON(result).Int())
}

func (c *Client) waitForPageChange(firstIDBefore string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		time.Sleep(500 * time.Millisecond)
		if c.firstVisibleItemID() != firstIDBefore {
			return true
		}
	}
	return false
}

func (c *Client) waitForMoreItems(countBefore int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		time.Sleep(500 * time.Millisecond)
		if c.countVisibleItems() > countBefore {
			return true
		}
	}
	return false
}

func (c *Client) extractVisibleItems(cutoff time.Time, seenIDs map[string]bool) ([]Discussion, bool, error) {
	result, err := c.page.Timeout(10 * time.Second).Eval(`() => {
		const items = [];
		document.querySelectorAll('.timeline__item').forEach(function(item) {
			const linkEl = item.querySelector('a.date-and-source[data-id]');
			const id = linkEl ? linkEl.getAttribute('data-id') : '';
			const relTime = linkEl ? linkEl.innerText.trim() : '';
			const authorEl = item.querySelector('.user-name');
			const author = authorEl ? authorEl.innerText.trim() : '';
			const contentEl = item.querySelector('.timeline__item__content .content');
			const content = contentEl ? contentEl.innerText.trim() : '';
			const replyEl = item.querySelector('.replay-count');
			const likeEl = item.querySelector('.like-count');
			const reply = replyEl ? replyEl.innerText.trim() : '';
			const like = likeEl ? likeEl.innerText.trim() : '';
			const postURL = linkEl ? linkEl.href : '';
			items.push({id: id, relTime: relTime, author: author, content: content, reply: reply, like: like, url: postURL});
		});
		return items;
	}`)
	if err != nil {
		return nil, false, err
	}

	type rawItem struct {
		ID      string `json:"id"`
		RelTime string `json:"relTime"`
		Author  string `json:"author"`
		Content string `json:"content"`
		Reply   string `json:"reply"`
		Like    string `json:"like"`
		URL     string `json:"url"`
	}
	var rawItems []rawItem
	if err := c.page.MustObjectToJSON(result).Unmarshal(&rawItems); err != nil {
		return nil, false, err
	}

	var discussions []Discussion
	reachedCutoff := false

	for _, item := range rawItems {
		if item.ID == "" || seenIDs[item.ID] {
			continue
		}

		createdAt := parseRelativeTime(item.RelTime)
		if createdAt.Before(cutoff) {
			reachedCutoff = true
			break
		}

		seenIDs[item.ID] = true
		discussions = append(discussions, Discussion{
			ID:           item.ID,
			Author:       item.Author,
			Content:      item.Content,
			CreatedAt:    createdAt,
			RelativeTime: item.RelTime,
			ReplyCount:   cleanStatText(item.Reply),
			LikeCount:    cleanStatText(item.Like),
			URL:          item.URL,
		})
	}

	return discussions, reachedCutoff, nil
}

func parseRelativeTime(relTime string) time.Time {
	now := time.Now()
	relTime = strings.TrimSpace(relTime)

	if relTime == "" {
		return now
	}

	// "· 来自Android" 等后缀去掉
	if idx := strings.Index(relTime, "·"); idx > 0 {
		relTime = strings.TrimSpace(relTime[:idx])
	}

	numRe := regexp.MustCompile(`(\d+)`)
	match := numRe.FindString(relTime)
	n := 0
	if match != "" {
		fmt.Sscanf(match, "%d", &n)
	}

	switch {
	case strings.Contains(relTime, "秒"):
		return now.Add(-time.Duration(n) * time.Second)
	case strings.Contains(relTime, "分钟"):
		return now.Add(-time.Duration(n) * time.Minute)
	case strings.Contains(relTime, "小时"):
		return now.Add(-time.Duration(n) * time.Hour)
	case strings.Contains(relTime, "天"):
		return now.AddDate(0, 0, -n)
	case strings.Contains(relTime, "周"):
		return now.AddDate(0, 0, -n*7)
	case strings.Contains(relTime, "月"):
		return now.AddDate(0, -n, 0)
	case strings.Contains(relTime, "年"):
		return now.AddDate(-n, 0, 0)
	default:
		// 尝试解析日期格式 "01-02" 或 "2024-01-02"
		layouts := []string{"2006-01-02 15:04", "2006-01-02", "01-02 15:04", "01-02"}
		for _, layout := range layouts {
			if t, err := time.ParseInLocation(layout, relTime, now.Location()); err == nil {
				if t.Year() <= 1 {
					t = time.Date(now.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), 0, now.Location())
					if t.After(now) {
						t = t.AddDate(-1, 0, 0)
					}
				}
				return t
			}
		}
		return now
	}
}

func cleanStatText(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "·")
	s = strings.TrimSpace(s)
	return s
}

func FormatDiscussions(discussions []Discussion, stockName, code string, days int) string {
	title := code
	if stockName != "" {
		title = stockName + " (" + code + ")"
	}
	return FormatDiscussionsWithTitle(discussions, title, days)
}

func FormatDiscussionsWithTitle(discussions []Discussion, title string, days int) string {
	var sb strings.Builder

	header := fmt.Sprintf("# %s 讨论", title)
	if days >= 0 {
		header = fmt.Sprintf("# %s 近%d天讨论", title, days)
	}
	sb.WriteString(header + "\n\n")
	sb.WriteString(fmt.Sprintf("共 %d 条讨论\n\n", len(discussions)))
	sb.WriteString(fmt.Sprintf("共 %d 条讨论\n\n", len(discussions)))
	sb.WriteString("---\n\n")

	for _, d := range discussions {
		timeStr := d.CreatedAt.Format("2006-01-02 15:04")
		if d.RelativeTime != "" {
			timeStr = d.RelativeTime
		}
		sb.WriteString(fmt.Sprintf("## %s - %s\n\n", timeStr, d.Author))
		sb.WriteString(d.Content)
		sb.WriteString("\n\n")
		stats := ""
		if d.ReplyCount != "" {
			stats += d.ReplyCount + "  "
		}
		if d.LikeCount != "" {
			stats += d.LikeCount
		}
		if stats != "" || d.URL != "" {
			sb.WriteString(fmt.Sprintf("%s  [链接](%s)\n\n", strings.TrimSpace(stats), d.URL))
		}
		sb.WriteString("---\n\n")
	}

	return sb.String()
}
