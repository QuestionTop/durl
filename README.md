# durl

[中文](README.zh.md)

A curl-like tool for fetching dynamic web pages with JavaScript rendering support.

## Features

- **Dynamic Rendering**: Fetches content from pages requiring JavaScript execution using Playwright
- **Multiple Content Levels**: Extract content at different levels (full, html, body, content, xpath, css)
- **Output Formats**: Support for HTML, Text, Markdown, JSON, and CSV
- **Site-Specific Modes**: Built-in scrapers for specific websites (Xueqiu comments, financial reports, Bing search, Baidu search)
- **Proxy Support**: Built-in proxy support with fallback retry mechanism
- **Flexible Wait Strategies**: Wait for page load, specific elements, or custom timeout
- **Pagination Support**: Automatic pagination handling for site-specific scrapers

## Installation

### Prerequisites

- Go 1.23.4 or higher
- Playwright browsers (automatically downloaded on first use)

### Build from Source

```bash
go build -o durl main.go
```

## Usage

### Basic Usage

Fetch a URL with default settings:

```bash
durl https://example.com
```

### Content Extraction by Selector

Extract content by CSS selector and save to markdown file:

```bash
durl -l css -s ".stock-info-content" -o out.md https://xueqiu.com/snowman/S/SZ300454/detail#/GSLRB
```

### Output Formats

Fetch content and export in different formats:

```bash
# Markdown
durl -f markdown https://example.com

# JSON
durl -f json https://example.com

# CSV (for tabular data)
durl -f csv https://xueqiu.com/snowman/S/SZ300454/detail#/GCFZB
```

### Wait Strategies

Control when content is extracted:

```bash
# Wait for page load (default)
durl -w load https://example.com

# Wait for specific element
durl -w element -T "#main-content" https://example.com

# Wait for custom time (milliseconds)
durl -w time -T 3000 https://example.com
```

### Proxy Support

Fetch content through a proxy:

```bash
# Using --proxy flag
durl -p http://127.0.0.1:7890 https://example.com

# Using environment variable
export DURL_PROXY=http://127.0.0.1:7890
durl https://example.com
```

## Site-Specific Modes

### Bing Search

Search Bing and retrieve results:

```bash
durl --site bing "golang tutorial"
durl --site bing "golang tutorial" -f json
durl --site bing "golang tutorial" -f markdown -o results.md
```

### Baidu Search

Search Baidu and retrieve results:

```bash
durl --site baidu "golang tutorial"
durl --site baidu "golang tutorial" -f json
durl --site baidu "golang tutorial" -f markdown -o results.md
```

### Xueqiu Comments

Search and extract discussions from Xueqiu:

```bash
# Search by stock name
durl --site xueqiu.comment "Yanjing Beer"

# Search by stock code
durl --site xueqiu.comment "SZ000729"

# Search with time range and sort order
durl --site xueqiu.comment --last 180d --sort new "09988"

# Time range formats: 7d, 1m, 1y, 202506, 2024
```

### Xueqiu Financial Reports

Extract financial reports and export as CSV:

```bash
# Export balance sheet
durl --site xueqiu.finreport SZ300454 -f csv -o report.csv

# Export income statement
durl --site xueqiu.finreport -o income.csv "https://xueqiu.com/snowman/S/SZ300454/detail#/GSLRB"
```

## Command Options

| Option | Short | Description | Default |
|--------|-------|-------------|---------|
| `--method` | `-X` | HTTP method (GET, POST, PUT, DELETE, etc.) | GET |
| `--header` | `-H` | HTTP headers (can be used multiple times) | - |
| `--data` | `-d` | Request body data | - |
| `--format` | `-f` | Output format (html, text, markdown, json, csv) | text |
| `--output` | `-o` | Output file path | - |
| `--wait-for` | `-w` | Wait strategy (load, element, time) | load |
| `--wait-target` | `-T` | Wait target (selector or milliseconds) | - |
| `--timeout` | `-t` | Request timeout duration | 30s |
| `--level` | `-l` | Content level (full, html, body, content, xpath, css) | body |
| `--selector` | `-s` | Selector for xpath or css level | - |
| `--site` | - | Site-specific mode (e.g. xueqiu.comment) | - |
| `--last` | - | Time range (7d, 1m, 1y, 202506, 2024) | 30d |
| `--max-pages` | - | Max pages to paginate (-1 for no limit) | -1 |
| `--sort` | - | Sort order: hot or new | hot |
| `--showui` | - | Show browser UI (disable headless mode) | false |
| `--proxy` | `-p` | Proxy URL | $DURL_PROXY |

## Content Levels

- **full**: Complete page including JS, CSS, and all resources
- **html**: HTML document only
- **body**: Page body content
- **content**: Smart extraction of main content (default)
- **xpath**: Extract content using XPath selector (requires --selector)
- **css**: Extract content using CSS selector (requires --selector)

## Architecture

The project follows a modular architecture:

```
durl/
├── main.go                 # CLI entry point using Cobra
├── internal/
│   ├── browser/           # Browser abstraction layer
│   ├── scraper/           # Scraper interface and registry
│   ├── formatter/         # Output formatting
│   └── sites/             # Site-specific scrapers
│       ├── generic/        # Generic web page scraper
│       ├── bing/           # Bing search scraper
│       ├── baidu/          # Baidu search scraper
│       └── xueqiu/         # Xueqiu-specific scrapers
│           ├── comment/   # Discussion scraper
│           └── finreport/ # Financial report scraper
```

### Extensibility

Adding a new site-specific scraper:

1. Create a new package under `internal/sites/`
2. Implement the `Scraper` interface
3. Register the scraper using `scraper.Register()`
4. Import the package in `main.go`

## Dependencies

- [go-rod](https://github.com/go-rod/rod) - Playwright-compatible browser automation
- [cobra](https://github.com/spf13/cobra) - Command-line interface framework
- [html-to-markdown](https://github.com/JohannesKaufmann/html-to-markdown) - HTML to Markdown conversion

## License

See LICENSE file for details.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
