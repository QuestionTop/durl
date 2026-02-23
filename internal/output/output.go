package output

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"durl/internal/extractor"
	"durl/internal/fetcher"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/PuerkitoBio/goquery"
)

// Output 输出格式化器
type Output struct {
	result    *fetcher.FetchResult
	extractor *extractor.Extractor
	level     string
	selector  string
}

// NewOutput 创建新的 Output 实例
func NewOutput(result *fetcher.FetchResult, level, selector string) *Output {
	var ext *extractor.Extractor
	if result.Page != nil {
		ext = extractor.NewExtractor(result.Page)
	}
	return &Output{
		result:    result,
		extractor: ext,
		level:     level,
		selector:  selector,
	}
}

// OutputHTML 输出HTML内容
func (o *Output) OutputHTML() (string, error) {
	if o.result.Page == nil {
		return "", fmt.Errorf("page is nil")
	}
	if o.extractor == nil {
		return "", fmt.Errorf("extractor is nil")
	}

	// 如果level为body，使用html级别以获取HTML内容而非纯文本
	level := o.level
	if o.level == "body" {
		level = "html"
	}

	content, err := o.extractor.Extract(level, o.selector)
	if err != nil {
		return "", fmt.Errorf("failed to extract content: %w", err)
	}
	return content, nil
}

// OutputText 输出纯文本内容
func (o *Output) OutputText() (string, error) {
	if o.result.Page == nil {
		return "", fmt.Errorf("page is nil")
	}
	if o.extractor == nil {
		return "", fmt.Errorf("extractor is nil")
	}

	// 如果level为body，直接返回提取的内容（已经是纯文本）
	if o.level == "body" {
		content, err := o.extractor.Extract(o.level, o.selector)
		if err != nil {
			return "", fmt.Errorf("failed to extract content: %w", err)
		}
		return content, nil
	}

	// 对于其他level，获取HTML内容并转换为纯文本
	html, err := o.extractor.Extract(o.level, o.selector)
	if err != nil {
		return "", fmt.Errorf("failed to extract content: %w", err)
	}

	// 创建 Converter
	converter := md.NewConverter("", true, nil)

	// 将HTML转换为Markdown（作为纯文本）
	text, err := converter.ConvertString(html)
	if err != nil {
		return "", fmt.Errorf("failed to convert HTML to text: %w", err)
	}

	return text, nil
}

// OutputMarkdown 输出Markdown格式内容
func (o *Output) OutputMarkdown() (string, error) {
	if o.result.Page == nil {
		return "", fmt.Errorf("page is nil")
	}
	if o.extractor == nil {
		return "", fmt.Errorf("extractor is nil")
	}

	// 获取HTML内容
	html, err := o.extractor.Extract(o.level, o.selector)
	if err != nil {
		return "", fmt.Errorf("failed to extract content: %w", err)
	}

	// 预处理 HTML，将表格转换为 Markdown 表格
	htmlWithMarkdownTables := convertTablesInHTML(html)

	// 创建 Converter
	converter := md.NewConverter("", true, nil)

	// 将处理后的HTML转换为Markdown
	markdown, err := converter.ConvertString(htmlWithMarkdownTables)
	if err != nil {
		return "", fmt.Errorf("failed to convert HTML to Markdown: %w", err)
	}

	// 清理 markdown 中的注释标记
	markdown = strings.ReplaceAll(markdown, "<!-- MARKDOWN_TABLE -->", "")
	markdown = strings.ReplaceAll(markdown, "<!-- END_MARKDOWN_TABLE -->", "")

	return markdown, nil
}

// convertTablesInHTML 将HTML中的所有表格转换为Markdown表格格式
func convertTablesInHTML(htmlContent string) string {
	// 使用正则表达式找到所有<table>标签并替换
	re := regexp.MustCompile(`(?is)<table\b[^>]*>.*?</table>`)
	return re.ReplaceAllStringFunc(htmlContent, convertHTMLTableToMarkdown)
}

// convertHTMLTableToMarkdown 将HTML表格转换为Markdown表格
func convertHTMLTableToMarkdown(tableHTML string) string {
	// 使用 goquery 解析 HTML
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(tableHTML))
	if err != nil {
		// 解析失败，返回预处理HTML
		return tableHTML
	}

	// 查找所有表格
	var builder strings.Builder
	doc.Find("table").Each(func(i int, table *goquery.Selection) {
		// 提取表头行（优先从thead找，其次从第一个tr找）
		headers := []string{}
		headerRow := table.Find("thead tr").First()
		if headerRow.Length() == 0 {
			headerRow = table.Find("tr").First()
		}
		headerRow.Find("th, td").Each(func(j int, header *goquery.Selection) {
			text := strings.TrimSpace(header.Text())
			headers = append(headers, text)
		})

		// 只有有效的表格数据才转换
		if len(headers) < 1 {
			return
		}

		// 处理表头行 - 标准Markdown表格格式
		builder.WriteString("| ")
		for j, hdr := range headers {
			if j > 0 {
				builder.WriteString(" | ")
			}
			builder.WriteString(strings.TrimSpace(hdr))
		}
		builder.WriteString(" |\n")

		// 添加分隔线
		builder.WriteString("| ")
		for j := range headers {
			if j > 0 {
				builder.WriteString(" | ")
			}
			builder.WriteString("---")
		}
		builder.WriteString(" |\n")

		// 提取表格数据行（排除表头行）
		dataRows := table.Find("tbody tr")
		if dataRows.Length() == 0 {
			// 如果没有tbody，获取所有tr并排除第一个
			dataRows = table.Find("tr").Slice(1, goquery.ToEnd)
		}

		dataRows.Each(func(j int, row *goquery.Selection) {
			cells := []string{}
			row.Find("td, th").Each(func(k int, cell *goquery.Selection) {
				text := strings.TrimSpace(cell.Text())
				cells = append(cells, text)
			})

			// 处理数据行
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

// JSONOutput JSON输出结构
type JSONOutput struct {
	HTML     string `json:"html"`
	Text     string `json:"text"`
	Markdown string `json:"markdown"`
	Title    string `json:"title"`
	URL      string `json:"url"`
	LoadTime int64  `json:"load_time"`
}

// OutputJSON 输出JSON格式数据
func (o *Output) OutputJSON() (string, error) {
	if o.result.Page == nil {
		return "", fmt.Errorf("page is nil")
	}

	// 获取各种格式的内容
	html, err := o.OutputHTML()
	if err != nil {
		return "", fmt.Errorf("failed to get page HTML: %w", err)
	}

	text, err := o.OutputText()
	if err != nil {
		return "", fmt.Errorf("failed to get page text: %w", err)
	}

	markdown, err := o.OutputMarkdown()
	if err != nil {
		return "", fmt.Errorf("failed to get page markdown: %w", err)
	}

	// 构建JSON输出结构
	output := JSONOutput{
		HTML:     html,
		Text:     text,
		Markdown: markdown,
		Title:    o.result.Title,
		URL:      o.result.URL,
		LoadTime: o.result.LoadTime.Milliseconds(),
	}

	// 转换为JSON字符串
	jsonData, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %w", err)
	}

	return string(jsonData), nil
}

// Format 根据用户选择的格式调用相应输出方法
func (o *Output) Format(format string) (string, error) {
	switch strings.ToLower(format) {
	case "html":
		return o.OutputHTML()
	case "text":
		return o.OutputText()
	case "markdown":
		return o.OutputMarkdown()
	case "json":
		return o.OutputJSON()
	default:
		return "", fmt.Errorf("unsupported output format: %s", format)
	}
}
