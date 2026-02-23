package xueqiu

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// XueqiuContent Xueqiu discussion content
type XueqiuContent struct {
	discussions []Discussion
	title       string
	cutoff      time.Time
}

// NewXueqiuContent creates a new XueqiuContent instance
func NewXueqiuContent(discussions []Discussion, title string, cutoff time.Time) *XueqiuContent {
	return &XueqiuContent{
		discussions: discussions,
		title:       title,
		cutoff:      cutoff,
	}
}

// ToMarkdown returns Markdown format content
func (x *XueqiuContent) ToMarkdown() (string, error) {
	var sb strings.Builder

	header := fmt.Sprintf("# %s Discussions", x.title)
	if !x.cutoff.IsZero() {
		if x.cutoff.Hour() == 0 && x.cutoff.Minute() == 0 && x.cutoff.Second() == 0 {
			days := int(time.Since(x.cutoff).Hours()/24) + 1
			header = fmt.Sprintf("# %s Discussions in Last %d Days", x.title, days)
		} else {
			header = fmt.Sprintf("# %s Discussions until %s", x.title, x.cutoff.Format("2006-01-02"))
		}
	}
	sb.WriteString(header + "\n\n")
	sb.WriteString(fmt.Sprintf("Total %d discussions\n\n", len(x.discussions)))
	sb.WriteString("---\n\n")

	for _, d := range x.discussions {
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
			sb.WriteString(fmt.Sprintf("%s  [Link](%s)\n\n", strings.TrimSpace(stats), d.URL))
		}
		sb.WriteString("---\n\n")
	}

	return sb.String(), nil
}

// ToText returns plain text content
func (x *XueqiuContent) ToText() (string, error) {
	return x.ToMarkdown()
}

// ToHTML returns HTML format content
func (x *XueqiuContent) ToHTML() (string, error) {
	md, err := x.ToMarkdown()
	if err != nil {
		return "", err
	}
	return "<pre>" + md + "</pre>", nil
}

// ToJSON returns JSON format content
func (x *XueqiuContent) ToJSON() ([]byte, error) {
	return json.Marshal(x.discussions)
}

// ToCSV returns CSV format content
func (x *XueqiuContent) ToCSV() (string, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	_ = w.Write([]string{"Time", "Author", "Content", "Replies", "Likes", "URL"})
	for _, d := range x.discussions {
		_ = w.Write([]string{
			d.CreatedAt.Format("2006-01-02 15:04"),
			d.Author,
			d.Content,
			d.ReplyCount,
			d.LikeCount,
			d.URL,
		})
	}
	w.Flush()
	return buf.String(), nil
}
