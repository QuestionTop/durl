package xueqiu

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// FinReportContent financial report content, implements scraper.Content interface
type FinReportContent struct {
	stockCode string
	stockName string
	tables    []FinReportTable
}

// NewFinReportContent creates a new FinReportContent instance
func NewFinReportContent(stockCode, stockName string, tables []FinReportTable) *FinReportContent {
	return &FinReportContent{
		stockCode: stockCode,
		stockName: stockName,
		tables:    tables,
	}
}

// cellValueRe matches the YoY (year-over-year) part in a cell: last +/- followed by number and %
// Example: "5.41B-32.56%"  →  value="5.41B"  YoY="-32.56%"
//
//	"4.70B-"          →  value="4.70B"  YoY="--" (single - means no data)
//	"--"              →  value="--"     YoY="--"
var cellValueRe = regexp.MustCompile(`^(.*?)([+-]\d[\d.]*%)$`)

// splitCell splits "5.41B-32.56%" into ("5.41B", "-32.56%")
// If no YoY data, returns (original value, "--")
func splitCell(s string) (value, ratio string) {
	s = strings.TrimSpace(s)
	if s == "" || s == "--" {
		return "--", "--"
	}
	m := cellValueRe.FindStringSubmatch(s)
	if m != nil {
		return strings.TrimSpace(m[1]), m[2]
	}
	// Cannot split (e.g. value with trailing "-" or pure number), set YoY to "--"
	// Remove trailing isolated "-"
	v := strings.TrimRight(s, "-")
	if v == "" {
		return "--", "--"
	}
	return strings.TrimSpace(v), "--"
}

// expandHeaders expands period column headers to ["2025Q3", "2025Q3(YoY)", ...]
func expandHeaders(headers []string) []string {
	out := make([]string, 0, len(headers)*2)
	for _, h := range headers {
		out = append(out, h, h+"(YoY)")
	}
	return out
}

// expandValues expands data values list, each value split into value+YoY two columns
func expandValues(values []string) []string {
	out := make([]string, 0, len(values)*2)
	for _, v := range values {
		val, ratio := splitCell(v)
		out = append(out, val, ratio)
	}
	return out
}

// ToCSV returns CSV format content
// Each period is split into two columns: value column and YoY column (e.g. "2025Q3" / "2025Q3(YoY)")
func (f *FinReportContent) ToCSV() (string, error) {
	var buf bytes.Buffer
	for i, table := range f.tables {
		if i > 0 {
			buf.WriteString("\n")
		}
		buf.WriteString(fmt.Sprintf("# %s (%s %s)\n", table.Type, f.stockName, f.stockCode))

		w := csv.NewWriter(&buf)
		// Header: Indicator + each period expanded to two columns
		header := append([]string{"Indicator"}, expandHeaders(table.Headers)...)
		_ = w.Write(header)
		// Data rows
		for _, row := range table.Rows {
			record := append([]string{row.Name}, expandValues(row.Values)...)
			_ = w.Write(record)
		}
		w.Flush()
	}
	return buf.String(), nil
}

// ToMarkdown returns Markdown format content
// Also splits each period into value and YoY two columns
func (f *FinReportContent) ToMarkdown() (string, error) {
	var sb strings.Builder
	for i, table := range f.tables {
		if i > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(fmt.Sprintf("## %s\n\n", table.Type))

		expandedHeaders := expandHeaders(table.Headers)

		// Header row
		sb.WriteString("| Indicator |")
		for _, h := range expandedHeaders {
			sb.WriteString(fmt.Sprintf(" %s |", h))
		}
		sb.WriteString("\n")

		// Separator row
		sb.WriteString("|---|")
		for range expandedHeaders {
			sb.WriteString("---|")
		}
		sb.WriteString("\n")

		// Data rows
		for _, row := range table.Rows {
			sb.WriteString(fmt.Sprintf("| %s |", row.Name))
			for _, v := range expandValues(row.Values) {
				sb.WriteString(fmt.Sprintf(" %s |", v))
			}
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}
	return sb.String(), nil
}

// ToText returns plain text content (delegates to ToMarkdown)
func (f *FinReportContent) ToText() (string, error) {
	return f.ToMarkdown()
}

// ToHTML returns HTML format content
func (f *FinReportContent) ToHTML() (string, error) {
	md, err := f.ToMarkdown()
	if err != nil {
		return "", err
	}
	return "<pre>" + md + "</pre>", nil
}

// ToJSON returns JSON format content (preserves original merged value, no split)
func (f *FinReportContent) ToJSON() ([]byte, error) {
	return json.Marshal(struct {
		StockCode string           `json:"stockCode"`
		StockName string           `json:"stockName"`
		Tables    []FinReportTable `json:"tables"`
	}{
		StockCode: f.stockCode,
		StockName: f.stockName,
		Tables:    f.tables,
	})
}
