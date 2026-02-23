package baidu

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strings"
)

// BaiduContent holds Baidu search results and implements scraper.Content.
type BaiduContent struct {
	query     string
	sourceURL string
	results   []Result
}

// NewBaiduContent creates a new BaiduContent instance.
func NewBaiduContent(query, sourceURL string, results []Result) *BaiduContent {
	return &BaiduContent{query: query, sourceURL: sourceURL, results: results}
}

func (c *BaiduContent) ToMarkdown() (string, error) {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# Baidu Search: %s\n\n", c.query))
	sb.WriteString(fmt.Sprintf("%d results\n\n", len(c.results)))
	for i, r := range c.results {
		sb.WriteString(fmt.Sprintf("## %d. [%s](%s)\n\n", i+1, r.Title, r.URL))
		if r.Snippet != "" {
			sb.WriteString(r.Snippet + "\n\n")
		}
	}
	return sb.String(), nil
}

func (c *BaiduContent) ToText() (string, error) {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Baidu Search: %s\n\n", c.query))
	for i, r := range c.results {
		sb.WriteString(fmt.Sprintf("%d. %s\n   %s\n", i+1, r.Title, r.URL))
		if r.Snippet != "" {
			sb.WriteString("   " + r.Snippet + "\n")
		}
		sb.WriteString("\n")
	}
	return sb.String(), nil
}

func (c *BaiduContent) ToHTML() (string, error) {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("<h1>Baidu Search: %s</h1>\n<ol>\n", c.query))
	for _, r := range c.results {
		sb.WriteString(fmt.Sprintf("  <li><a href=%q>%s</a>", r.URL, r.Title))
		if r.Snippet != "" {
			sb.WriteString("<p>" + r.Snippet + "</p>")
		}
		sb.WriteString("</li>\n")
	}
	sb.WriteString("</ol>\n")
	return sb.String(), nil
}

func (c *BaiduContent) ToJSON() ([]byte, error) {
	type jsonResult struct {
		Query   string   `json:"query"`
		Source  string   `json:"source"`
		Results []Result `json:"results"`
	}
	return json.Marshal(jsonResult{Query: c.query, Source: c.sourceURL, Results: c.results})
}

func (c *BaiduContent) ToCSV() (string, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	_ = w.Write([]string{"Title", "URL", "Snippet"})
	for _, r := range c.results {
		_ = w.Write([]string{r.Title, r.URL, r.Snippet})
	}
	w.Flush()
	return buf.String(), nil
}
