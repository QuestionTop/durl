package xueqiu

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-rod/rod"
)

// ResolveStockCode resolves stock code (package-level function, accepts page parameter)
// Rules:
// 1. SZ/SH/HK prefix + number → return directly in uppercase
// 2. 6-digit pure number → add SH/SZ prefix based on first digit
// 3. 2-5 digit pure number → pad to 5 digits
// 4. Others → call searchStockByPage to search
func ResolveStockCode(query string, page *rod.Page) (code, name string, err error) {
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

	return searchStockByPage(page, query)
}

// searchStockByPage searches stock code by page (private function)
func searchStockByPage(page *rod.Page, query string) (string, string, error) {
	searchURL := "https://xueqiu.com/k?q=" + url.QueryEscape(query)

	if err := page.Timeout(20 * time.Second).Navigate(searchURL); err != nil {
		return "", "", fmt.Errorf("failed to navigate to search: %w", err)
	}
	time.Sleep(4 * time.Second)

	result, err := page.Timeout(10 * time.Second).Eval(`() => {
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
	if err := page.MustObjectToJSON(result).Unmarshal(&items); err != nil || len(items) == 0 {
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

// parseRelativeTime parses relative time string to time.Time
func parseRelativeTime(relTime string) time.Time {
	now := time.Now()
	relTime = strings.TrimSpace(relTime)

	if relTime == "" {
		return now
	}

	// Remove suffix like "· from Android"
	if idx := strings.Index(relTime, "·"); idx > 0 {
		relTime = strings.TrimSpace(relTime[:idx])
	}

	numRe := regexp.MustCompile(`(\d+)`)
	match := numRe.FindString(relTime)
	n := 0
	if match != "" {
		fmt.Sscanf(match, "%d", &n)
	}

	// Xueqiu renders relative times in Chinese (UI strings must match), e.g. "5分钟前"=5min ago, "2小时前"=2h ago, "3天前"=3d ago.
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
		// Try to parse date format "01-02" or "2024-01-02"
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

// cleanStatText removes "·" prefix from statistics text
func cleanStatText(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "·")
	s = strings.TrimSpace(s)
	return s
}

// ParseLast parses --last string to cutoff time
// Supports formats: 7d (N days ago), 1m (N months ago), 1y (N years ago), 202506 (to June 1, 2025), 2024 (to January 1, 2024)
func ParseLast(s string) (time.Time, error) {
	s = strings.TrimSpace(s)

	// Match Nd (N days ago)
	if m := regexp.MustCompile(`^(\d+)d$`).FindStringSubmatch(s); m != nil {
		n, err := strconv.Atoi(m[1])
		if err != nil {
			return time.Time{}, err
		}
		return time.Now().AddDate(0, 0, -n), nil
	}

	// Match Nm (N months ago)
	if m := regexp.MustCompile(`^(\d+)m$`).FindStringSubmatch(s); m != nil {
		n, err := strconv.Atoi(m[1])
		if err != nil {
			return time.Time{}, err
		}
		return time.Now().AddDate(0, -n, 0), nil
	}

	// Match Ny (N years ago)
	if m := regexp.MustCompile(`^(\d+)y$`).FindStringSubmatch(s); m != nil {
		n, err := strconv.Atoi(m[1])
		if err != nil {
			return time.Time{}, err
		}
		return time.Now().AddDate(-n, 0, 0), nil
	}

	// Match 6-digit pure number (YYYYMM)
	if regexp.MustCompile(`^\d{6}$`).MatchString(s) {
		year, err := strconv.Atoi(s[:4])
		if err != nil {
			return time.Time{}, err
		}
		month, err := strconv.Atoi(s[4:])
		if err != nil {
			return time.Time{}, err
		}
		return time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.Local), nil
	}

	// Match 4-digit pure number (YYYY)
	if regexp.MustCompile(`^\d{4}$`).MatchString(s) {
		year, err := strconv.Atoi(s)
		if err != nil {
			return time.Time{}, err
		}
		return time.Date(year, 1, 1, 0, 0, 0, 0, time.Local), nil
	}

	return time.Time{}, fmt.Errorf("invalid --last value: %q, supported: 7d, 1m, 1y, 202506, 2024", s)
}
