package formatter

import (
	"fmt"

	"durl/internal/scraper"
)

func Format(content scraper.Content, format string) (string, error) {
	switch format {
	case "html":
		return content.ToHTML()
	case "text":
		return content.ToText()
	case "markdown":
		return content.ToMarkdown()
	case "csv":
		return content.ToCSV()
	case "json":
		b, err := content.ToJSON()
		if err != nil {
			return "", err
		}
		return string(b), nil
	default:
		return "", fmt.Errorf("unsupported output format: %s", format)
	}
}
