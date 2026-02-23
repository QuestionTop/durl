package xueqiu

import (
	"fmt"
	"os"
	"strings"
	"time"

	"durl/internal/browser"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
)

// Discussion discussion structure
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

// Client Xueqiu browser client
type Client struct {
	browser *browser.Browser
	page    *rod.Page
}

// NewClient creates a new Client instance
func NewClient(b *browser.Browser) *Client {
	return &Client{browser: b}
}

// Close closes page
func (c *Client) Close() {
	if c.page != nil {
		c.page.Close()
	}
}

// Page returns current page (for ResolveStockCode in scraper.go)
func (c *Client) Page() *rod.Page {
	return c.page
}

// ResolveStockCode resolves stock code (delegates to package-level function in parser.go)
func (c *Client) ResolveStockCode(query string) (code, name string, err error) {
	return ResolveStockCode(query, c.page)
}

// Init initializes Xueqiu session
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

// FetchDiscussions gets discussions by stock code
func (c *Client) FetchDiscussions(code string, cutoff time.Time, sort string, maxPages int) ([]Discussion, error) {
	return c.FetchByURL("https://xueqiu.com/S/"+code, cutoff, sort, maxPages)
}

// FetchByURL gets discussions by URL
func (c *Client) FetchByURL(targetURL string, cutoff time.Time, sort string, maxPages int) ([]Discussion, error) {
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

	// Close login popup (if appears)
	_, _ = c.page.Eval(`() => {
        const mask = document.querySelector('.modal-mask, .login-modal, [class*=modal]');
        if (mask) mask.remove();
        const overlay = document.querySelector('.overlay');
        if (overlay) overlay.remove();
    }`)

	// Switch sort order: new=new posts, hot=hot posts (default hot)
	// Labels must match the Chinese text rendered by Xueqiu's UI.
	sortLabel := "最热"
	if strings.ToLower(sort) == "new" {
		sortLabel = "最新"
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

		if !cutoff.IsZero() && reachedCutoff {
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
		time.Sleep(1500 * time.Millisecond)
		if c.firstVisibleItemID() == firstIDBefore {
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
