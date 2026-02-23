package generic

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/PuerkitoBio/goquery"
)

// PageContent holds pre-extracted page content as strings.
// All content is extracted while the browser is still open (in Scrape),
// so the formatters do not need a live browser connection.
type PageContent struct {
	htmlContent string // for ToHTML(): body innerHTML (or selector HTML)
	mainContent string // for ToMarkdown(), ToCSV(), ToJSON(): content at the requested level/selector
	textContent string // for ToText(): plain text (body innerText when level=body, else same as mainContent)
	level       string // original level flag, used to decide ToText conversion
	title       string
	url         string
	loadTime    time.Duration
}

// NewPageContent creates a PageContent from pre-extracted strings.
// htmlContent: HTML for ToHTML()
// mainContent: HTML (or text) for ToMarkdown/ToCSV/ToJSON
// textContent: plain text for ToText() when level=="body", else HTML to be converted
// level: the original --level flag value
func NewPageContent(htmlContent, mainContent, textContent, level, title, url string, loadTime time.Duration) *PageContent {
	return &PageContent{
		htmlContent: htmlContent,
		mainContent: mainContent,
		textContent: textContent,
		level:       level,
		title:       title,
		url:         url,
		loadTime:    loadTime,
	}
}

// ToHTML returns HTML format content
func (p *PageContent) ToHTML() (string, error) {
	return p.htmlContent, nil
}

// ToText returns plain text content
func (p *PageContent) ToText() (string, error) {
	if p.level == "body" {
		// textContent is already plain text (body innerText)
		return p.textContent, nil
	}

	converter := md.NewConverter("", true, nil)
	text, err := converter.ConvertString(p.textContent)
	if err != nil {
		return "", fmt.Errorf("failed to convert HTML to text: %w", err)
	}
	return text, nil
}

// ToMarkdown returns Markdown format content
func (p *PageContent) ToMarkdown() (string, error) {
	htmlWithMarkdownTables := convertTablesInHTML(p.mainContent)

	converter := md.NewConverter("", true, nil)
	markdown, err := converter.ConvertString(htmlWithMarkdownTables)
	if err != nil {
		return "", fmt.Errorf("failed to convert HTML to Markdown: %w", err)
	}

	markdown = strings.ReplaceAll(markdown, "<!-- MARKDOWN_TABLE -->", "")
	markdown = strings.ReplaceAll(markdown, "<!-- END_MARKDOWN_TABLE -->", "")

	return markdown, nil
}

// ToJSON returns JSON format content
func (p *PageContent) ToJSON() ([]byte, error) {
	html, err := p.ToHTML()
	if err != nil {
		return nil, fmt.Errorf("failed to get page HTML: %w", err)
	}

	text, err := p.ToText()
	if err != nil {
		return nil, fmt.Errorf("failed to get page text: %w", err)
	}

	markdown, err := p.ToMarkdown()
	if err != nil {
		return nil, fmt.Errorf("failed to get page markdown: %w", err)
	}

	type jsonOutput struct {
		HTML     string `json:"html"`
		Text     string `json:"text"`
		Markdown string `json:"markdown"`
		Title    string `json:"title"`
		URL      string `json:"url"`
		LoadTime int64  `json:"load_time"`
	}

	output := jsonOutput{
		HTML:     html,
		Text:     text,
		Markdown: markdown,
		Title:    p.title,
		URL:      p.url,
		LoadTime: p.loadTime.Milliseconds(),
	}

	return json.MarshalIndent(output, "", "  ")
}

// ToCSV returns CSV format content (extracts all HTML tables from page)
func (p *PageContent) ToCSV() (string, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(p.mainContent))
	if err != nil {
		return "", fmt.Errorf("failed to parse HTML: %w", err)
	}

	var buf bytes.Buffer
	tableIndex := 0
	doc.Find("table").Each(func(i int, table *goquery.Selection) {
		tableIndex++
		if tableIndex > 1 {
			buf.WriteString("\n")
		}
		buf.WriteString(fmt.Sprintf("# Table %d\n", tableIndex))

		w := csv.NewWriter(&buf)
		table.Find("tr").Each(func(j int, row *goquery.Selection) {
			var record []string
			row.Find("th, td").Each(func(k int, cell *goquery.Selection) {
				record = append(record, strings.TrimSpace(cell.Text()))
			})
			if len(record) > 0 {
				_ = w.Write(record)
			}
		})
		w.Flush()
	})

	return buf.String(), nil
}

// convertTablesInHTML converts all tables in HTML to Markdown table format
func convertTablesInHTML(htmlContent string) string {
	re := regexp.MustCompile(`(?is)<table\b[^>]*>.*?</table>`)
	return re.ReplaceAllStringFunc(htmlContent, convertHTMLTableToMarkdown)
}

// convertHTMLTableToMarkdown converts HTML table to Markdown table
func convertHTMLTableToMarkdown(tableHTML string) string {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(tableHTML))
	if err != nil {
		return tableHTML
	}

	var builder strings.Builder
	doc.Find("table").Each(func(i int, table *goquery.Selection) {
		headers := []string{}
		headerRow := table.Find("thead tr").First()
		if headerRow.Length() == 0 {
			headerRow = table.Find("tr").First()
		}
		headerRow.Find("th, td").Each(func(j int, header *goquery.Selection) {
			text := strings.TrimSpace(header.Text())
			headers = append(headers, text)
		})

		if len(headers) < 1 {
			return
		}

		builder.WriteString("| ")
		for j, hdr := range headers {
			if j > 0 {
				builder.WriteString(" | ")
			}
			builder.WriteString(strings.TrimSpace(hdr))
		}
		builder.WriteString(" |\n")

		builder.WriteString("| ")
		for j := range headers {
			if j > 0 {
				builder.WriteString(" | ")
			}
			builder.WriteString("---")
		}
		builder.WriteString(" |\n")

		dataRows := table.Find("tbody tr")
		if dataRows.Length() == 0 {
			dataRows = table.Find("tr").Slice(1, goquery.ToEnd)
		}

		dataRows.Each(func(j int, row *goquery.Selection) {
			cells := []string{}
			row.Find("td, th").Each(func(k int, cell *goquery.Selection) {
				text := strings.TrimSpace(cell.Text())
				cells = append(cells, text)
			})

			if len(cells) >= 1 {
				builder.WriteString("| ")
				for k, cell := range cells {
					if k > 0 {
						builder.WriteString(" | ")
					}
					builder.WriteString(strings.TrimSpace(cell))
				}
				builder.WriteString(" |\n")
			}
		})

		builder.WriteString("\n")
	})

	return builder.String()
}
