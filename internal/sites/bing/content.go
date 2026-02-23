package bing

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strings"
)

// BingContent holds Bing search results and implements scraper.Content.
type BingContent struct {
	query     string
	sourceURL string
	results   []Result
}

// NewBingContent creates a new BingContent instance.
func NewBingContent(query, sourceURL string, results []Result) *BingContent {
	return &BingContent{query: query, sourceURL: sourceURL, results: results}
}

func (c *BingContent) ToMarkdown() (string, error) {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# Bing Search: %s\n\n", c.query))
	sb.WriteString(fmt.Sprintf("%d results\n\n", len(c.results)))
	for i, r := range c.results {
		sb.WriteString(fmt.Sprintf("## %d. [%s](%s)\n\n", i+1, r.Title, r.URL))
		if r.Snippet != "" {
			sb.WriteString(r.Snippet + "\n\n")
		}
	}
	return sb.String(), nil
}

func (c *BingContent) ToText() (string, error) {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Bing Search: %s\n\n", c.query))
	for i, r := range c.results {
		sb.WriteString(fmt.Sprintf("%d. %s\n   %s\n", i+1, r.Title, r.URL))
		if r.Snippet != "" {
			sb.WriteString("   " + r.Snippet + "\n")
		}
		sb.WriteString("\n")
	}
	return sb.String(), nil
}

func (c *BingContent) ToHTML() (string, error) {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("<h1>Bing Search: %s</h1>\n<ol>\n", c.query))
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

func (c *BingContent) ToJSON() ([]byte, error) {
	type jsonResult struct {
		Query   string   `json:"query"`
		Source  string   `json:"source"`
		Results []Result `json:"results"`
	}
	return json.Marshal(jsonResult{Query: c.query, Source: c.sourceURL, Results: c.results})
}

func (c *BingContent) ToCSV() (string, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	_ = w.Write([]string{"Title", "URL", "Snippet"})
	for _, r := range c.results {
		_ = w.Write([]string{r.Title, r.URL, r.Snippet})
	}
	w.Flush()
	return buf.String(), nil
}
