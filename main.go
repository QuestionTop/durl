package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"durl/internal/browser"
	"durl/internal/formatter"
	"durl/internal/scraper"
	_ "durl/internal/sites/baidu"
	_ "durl/internal/sites/bing"
	generic "durl/internal/sites/generic"
	_ "durl/internal/sites/xueqiu"

	"github.com/spf13/cobra"
)

var version = "dev"

var (
	method       string
	headers      []string
	data         string
	outputFormat string
	outputFile   string
	waitFor      string
	waitTarget   string
	timeout      time.Duration
	level        string
	selector     string
	site         string
	last         string
	maxPages     int
	sort         string
	showUI       bool
	proxyURL     string
)

func main() {
	var rootCmd = &cobra.Command{
		Use:     "durl [URL]",
		Short:   "A curl-like tool with dynamic rendering support",
		Version: version,
		Long: `durl is a command-line tool similar to curl, but with support for
dynamic web page rendering using Playwright. It can fetch and display
content from pages that require JavaScript execution.`,
		Example: `  # Extract content by CSS selector and save to file
  durl -l css -s ".stock-info-content" -o out.md https://xueqiu.com/snowman/S/SZ300454/detail#/GSLRB

  # Fetch content from search page
  durl -f markdown -l css -s "#b_results" "https://cn.bing.com/search?q=your-keyword&PC=U316&FORM=CHROMN"

  # Search xueqiu discussions by stock name or code
  durl --site xueqiu.comment "Yanjing Beer"
  durl --site xueqiu.comment "SZ000729"
  durl --site xueqiu.comment --last 180d --sort new "09988"

  # Fetch xueqiu financial reports and export as CSV
  durl --site xueqiu.finreport SZ300454 -f csv -o report.csv

  # Search Bing and get results
  durl --site bing "keyword"
  durl --site bing "durl" -f markdown

  # Search Baidu and get results
  durl --site baidu "golang tutorial" -f json`,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				cmd.Help()
				os.Exit(0)
			}
			return cobra.ExactArgs(1)(cmd, args)
		},
		RunE:         run,
		SilenceUsage: true,
	}

	rootCmd.Flags().StringVarP(&method, "method", "X", "GET", "HTTP method (GET, POST, PUT, DELETE, etc.)")
	rootCmd.Flags().StringSliceVarP(&headers, "header", "H", []string{}, "HTTP headers (can be used multiple times)")
	rootCmd.Flags().StringVarP(&data, "data", "d", "", "Request body data")
	rootCmd.Flags().StringVarP(&outputFormat, "format", "f", "text", "Output format (html, text, markdown, json, csv)")
	rootCmd.Flags().StringVarP(&outputFile, "output", "o", "", "Output file path (format inferred from extension if -f not specified)")
	rootCmd.Flags().StringVarP(&waitFor, "wait-for", "w", "load", "Wait strategy (load, element, time)")
	rootCmd.Flags().StringVarP(&waitTarget, "wait-target", "T", "", "Wait target (selector for 'element' strategy, milliseconds for 'time' strategy)")
	rootCmd.Flags().DurationVarP(&timeout, "timeout", "t", 30*time.Second, "Request timeout duration")
	rootCmd.Flags().StringVarP(&level, "level", "l", "body", "Content extraction level (full, html, body, content, xpath, css)")
	rootCmd.Flags().StringVarP(&selector, "selector", "s", "", "Selector for xpath or css level")
	rootCmd.Flags().StringVar(&site, "site", "", "Site-specific mode (e.g. xueqiu)")
	rootCmd.Flags().StringVar(&last, "last", "30d", "time range: 7d, 1m, 1y, 202506, 2024")
	rootCmd.Flags().IntVar(&maxPages, "max-pages", -1, "Max pages to paginate (-1 for no limit)")
	rootCmd.Flags().StringVar(&sort, "sort", "hot", "Sort order: hot or new")
	rootCmd.Flags().BoolVar(&showUI, "showui", false, "Show browser UI (disable headless mode)")
	rootCmd.Flags().StringVarP(&proxyURL, "proxy", "p", os.Getenv("DURL_PROXY"), "Proxy URL (e.g. http://127.0.0.1:7890), defaults to DURL_PROXY env var")

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) error {
	target := args[0]

	// If output file is specified but format is not, infer format from file extension
	if outputFile != "" && outputFormat == "text" {
		inferredFormat := inferFormatFromExtension(outputFile)
		if inferredFormat != "" {
			outputFormat = inferredFormat
		}
	}

	if err := validateFlags(); err != nil {
		return err
	}

	// Build scraper.Options
	opts := scraper.Options{
		Method:     method,
		Headers:    parseHeaders(headers),
		Body:       data,
		WaitFor:    waitFor,
		WaitTarget: waitTarget,
		Timeout:    timeout,
		Level:      level,
		Selector:   selector,
		ShowUI:     showUI,
		ProxyURL:   proxyURL,
		Extra: map[string]string{
			"last":      last,
			"max-pages": strconv.Itoa(maxPages),
			"sort":      sort,
		},
	}

	ctx := context.Background()

	var content scraper.Content
	var err error

	if site != "" {
		// Site-specific mode: Get Scraper from registry
		s, ok := scraper.Get(site)
		if !ok {
			return fmt.Errorf("unknown site: %s", site)
		}
		content, err = s.Scrape(ctx, target, opts)
		if err != nil {
			return fmt.Errorf("failed to scrape: %w", err)
		}
	} else {
		// Generic mode: Use GenericScraper with proxy retry
		target = normalizeURL(target)
		gs := generic.NewGenericScraper(browser.Config{
			Headless: !showUI,
		})
		content, err = gs.Scrape(ctx, target, opts)
		if err != nil {
			// If failed and proxy is available, retry with proxy
			if proxyURL != "" {
				fmt.Fprintf(os.Stderr, "Warning: First attempt failed: %v\n", err)
				fmt.Fprintf(os.Stderr, "Retrying with proxy: %s\n", proxyURL)
				gs2 := generic.NewGenericScraper(browser.Config{
					ProxyURL: proxyURL,
					Headless: !showUI,
				})
				content, err = gs2.Scrape(ctx, target, opts)
				if err != nil {
					return fmt.Errorf("failed to fetch page (even with proxy): %w", err)
				}
				fmt.Fprintf(os.Stderr, "Fetched successfully (with proxy: %s)\n", proxyURL)
			} else {
				return fmt.Errorf("failed to fetch page: %w", err)
			}
		}
	}

	// Format output
	outputContent, err := formatter.Format(content, outputFormat)
	if err != nil {
		return fmt.Errorf("failed to format output: %w", err)
	}

	// Output result
	if outputFile != "" {
		if err := os.WriteFile(outputFile, []byte(outputContent), 0644); err != nil {
			return fmt.Errorf("failed to write to file: %w", err)
		}
		fmt.Fprintf(os.Stderr, "Output written to: %s\n", outputFile)
	} else {
		fmt.Println(outputContent)
	}

	// For non-JSON format and stdout output, output metadata to stderr (generic mode only)
	if site == "" && outputFormat != "json" && outputFile == "" {
		// In generic mode, content is *generic.PageContent which contains metadata
		// Get metadata via ToJSON (simplified: output separator directly)
		fmt.Fprintf(os.Stderr, "\n---\n")
	}

	return nil
}

func validateFlags() error {
	validMethods := map[string]bool{
		"GET":     true,
		"POST":    true,
		"PUT":     true,
		"DELETE":  true,
		"PATCH":   true,
		"HEAD":    true,
		"OPTIONS": true,
	}
	if !validMethods[method] {
		return fmt.Errorf("invalid HTTP method: %s", method)
	}

	validFormats := map[string]bool{
		"html":     true,
		"text":     true,
		"markdown": true,
		"json":     true,
		"csv":      true,
	}
	if !validFormats[outputFormat] {
		return fmt.Errorf("invalid output format: %s", outputFormat)
	}

	validStrategies := map[string]bool{
		"load":    true,
		"element": true,
		"time":    true,
	}
	if !validStrategies[waitFor] {
		return fmt.Errorf("invalid wait strategy: %s", waitFor)
	}

	if waitFor == "element" && waitTarget == "" {
		return fmt.Errorf("--wait-target is required when using 'element' wait strategy")
	}

	if waitFor == "time" && waitTarget == "" {
		return fmt.Errorf("--wait-target is required when using 'time' wait strategy")
	}

	validLevels := map[string]bool{
		"full":    true,
		"html":    true,
		"body":    true,
		"content": true,
		"xpath":   true,
		"css":     true,
	}
	if !validLevels[level] {
		return fmt.Errorf("invalid content level: %s", level)
	}

	if (level == "xpath" || level == "css") && selector == "" {
		return fmt.Errorf("--selector is required when using '%s' level", level)
	}

	if level != "xpath" && level != "css" && selector != "" {
		return fmt.Errorf("--selector is only valid with 'xpath' or 'css' level")
	}

	return nil
}

// inferFormatFromExtension infers output format from file extension
func inferFormatFromExtension(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".md", ".markdown":
		return "markdown"
	case ".json":
		return "json"
	case ".html", ".htm":
		return "html"
	case ".txt":
		return "text"
	case ".csv":
		return "csv"
	default:
		return ""
	}
}

// parseHeaders parses request header parameters
func parseHeaders(headerSlice []string) map[string]string {
	headersMap := make(map[string]string)
	for _, h := range headerSlice {
		parts := strings.SplitN(h, ":", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			if key != "" {
				headersMap[key] = value
			}
		}
	}
	return headersMap
}

// normalizeURL normalizes URL, adds http:// if no protocol prefix
func normalizeURL(rawURL string) string {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return rawURL
	}
	if !strings.HasPrefix(strings.ToLower(rawURL), "http://") && !strings.HasPrefix(strings.ToLower(rawURL), "https://") {
		return "http://" + rawURL
	}
	return rawURL
}
