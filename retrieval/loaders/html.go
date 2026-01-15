package loaders

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// HTMLLoader loads and parses HTML documents (from file or URL)
type HTMLLoader struct {
	*BaseLoader
	url                 string
	removeScripts       bool
	removeStyles        bool
	extractLinks        bool
	timeout             time.Duration
	customHTTPClient    *http.Client
	selector            string
	extractMetaTags     bool
}

// HTMLLoaderOptions contains configuration options for HTML loader
type HTMLLoaderOptions struct {
	// Path is the file path to load (optional if URL is provided)
	Path string

	// URL is the web URL to fetch (optional if Path is provided)
	URL string

	// RemoveScripts removes <script> tags from HTML
	RemoveScripts bool

	// RemoveStyles removes <style> tags from HTML
	RemoveStyles bool

	// ExtractLinks extracts all links from the HTML
	ExtractLinks bool

	// Timeout for HTTP requests (default: 30s)
	Timeout time.Duration

	// CustomHTTPClient allows using a custom HTTP client
	CustomHTTPClient *http.Client

	// Selector is a CSS selector to extract specific parts (e.g., "article", ".content")
	Selector string

	// ExtractMetaTags extracts meta tags (title, description, keywords)
	ExtractMetaTags bool

	// Metadata to include with the document
	Metadata map[string]any
}

// NewHTMLLoader creates a new HTML loader with the given options
func NewHTMLLoader(options HTMLLoaderOptions) (*HTMLLoader, error) {
	if options.Path == "" && options.URL == "" {
		return nil, fmt.Errorf("either Path or URL must be provided")
	}

	base := &BaseLoader{
		path:     options.Path,
		metadata: options.Metadata,
	}

	if base.metadata == nil {
		base.metadata = make(map[string]any)
	}

	if options.Timeout == 0 {
		options.Timeout = 30 * time.Second
	}

	return &HTMLLoader{
		BaseLoader:       base,
		url:              options.URL,
		removeScripts:    options.RemoveScripts,
		removeStyles:     options.RemoveStyles,
		extractLinks:     options.ExtractLinks,
		timeout:          options.Timeout,
		customHTTPClient: options.CustomHTTPClient,
		selector:         options.Selector,
		extractMetaTags:  options.ExtractMetaTags,
	}, nil
}

// Load loads the HTML document and returns parsed documents
func (loader *HTMLLoader) Load(ctx context.Context) ([]*Document, error) {
	var reader io.Reader
	var source string

	// Load from file or URL
	if loader.url != "" {
		// Fetch from URL
		content, err := loader.fetchURL(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch URL: %w", err)
		}
		reader = strings.NewReader(content)
		source = loader.url
	} else {
		// Load from file
		file, err := os.Open(loader.path)
		if err != nil {
			return nil, fmt.Errorf("failed to open file: %w", err)
		}
		defer file.Close()
		reader = file
		source = loader.path
	}

	// Parse HTML
	doc, err := goquery.NewDocumentFromReader(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	// Remove scripts and styles if requested
	if loader.removeScripts {
		doc.Find("script").Remove()
	}
	if loader.removeStyles {
		doc.Find("style").Remove()
	}

	// Extract meta tags if requested
	if loader.extractMetaTags {
		loader.extractMetadata(doc)
	}

	// Extract content
	var text string
	if loader.selector != "" {
		// Extract specific selector
		selection := doc.Find(loader.selector)
		text = strings.TrimSpace(selection.Text())
	} else {
		// Extract body text
		text = strings.TrimSpace(doc.Find("body").Text())
	}

	// Clean up whitespace
	text = cleanWhitespace(text)

	// Extract links if requested
	if loader.extractLinks {
		links := loader.extractAllLinks(doc, source)
		loader.metadata["links"] = links
		loader.metadata["link_count"] = len(links)
	}

	// Create document metadata
	metadata := make(map[string]any)
	for k, v := range loader.metadata {
		metadata[k] = v
	}
	metadata["source"] = source
	metadata["file_type"] = "html"

	// Create document
	document := NewDocument(text, metadata)

	return []*Document{document}, nil
}

// fetchURL fetches content from a URL
func (loader *HTMLLoader) fetchURL(ctx context.Context) (string, error) {
	// Create HTTP client
	client := loader.customHTTPClient
	if client == nil {
		client = &http.Client{
			Timeout: loader.timeout,
		}
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, "GET", loader.url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set user agent
	req.Header.Set("User-Agent", "LangChain-Go HTML Loader/1.0")

	// Execute request
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP request failed with status: %d", resp.StatusCode)
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	return string(body), nil
}

// extractMetadata extracts meta tags from HTML
func (loader *HTMLLoader) extractMetadata(doc *goquery.Document) {
	// Extract title
	title := doc.Find("title").Text()
	if title != "" {
		loader.metadata["title"] = strings.TrimSpace(title)
	}

	// Extract meta tags
	doc.Find("meta").Each(func(i int, s *goquery.Selection) {
		name, _ := s.Attr("name")
		property, _ := s.Attr("property")
		content, _ := s.Attr("content")

		if content == "" {
			return
		}

		// Standard meta tags
		switch strings.ToLower(name) {
		case "description":
			loader.metadata["description"] = content
		case "keywords":
			loader.metadata["keywords"] = content
		case "author":
			loader.metadata["author"] = content
		}

		// Open Graph tags
		switch strings.ToLower(property) {
		case "og:title":
			loader.metadata["og_title"] = content
		case "og:description":
			loader.metadata["og_description"] = content
		case "og:image":
			loader.metadata["og_image"] = content
		case "og:url":
			loader.metadata["og_url"] = content
		}
	})
}

// extractAllLinks extracts all links from HTML
func (loader *HTMLLoader) extractAllLinks(doc *goquery.Document, baseURL string) []string {
	var links []string
	seen := make(map[string]bool)

	doc.Find("a[href]").Each(func(i int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if !exists || href == "" {
			return
		}

		// Resolve relative URLs
		absoluteURL := resolveURL(baseURL, href)
		if absoluteURL != "" && !seen[absoluteURL] {
			links = append(links, absoluteURL)
			seen[absoluteURL] = true
		}
	})

	return links
}

// resolveURL resolves a relative URL against a base URL
func resolveURL(baseURL, href string) string {
	// Skip javascript, mailto, tel links
	if strings.HasPrefix(href, "javascript:") ||
		strings.HasPrefix(href, "mailto:") ||
		strings.HasPrefix(href, "tel:") ||
		strings.HasPrefix(href, "#") {
		return ""
	}

	// If href is already absolute, return it
	if strings.HasPrefix(href, "http://") || strings.HasPrefix(href, "https://") {
		return href
	}

	// Parse base URL
	base, err := url.Parse(baseURL)
	if err != nil {
		return ""
	}

	// Parse href
	ref, err := url.Parse(href)
	if err != nil {
		return ""
	}

	// Resolve
	absolute := base.ResolveReference(ref)
	return absolute.String()
}

// cleanWhitespace cleans up excessive whitespace in text
func cleanWhitespace(text string) string {
	// Replace multiple spaces with single space
	lines := strings.Split(text, "\n")
	var cleanLines []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			// Replace multiple spaces with single space
			words := strings.Fields(line)
			cleanLines = append(cleanLines, strings.Join(words, " "))
		}
	}

	return strings.Join(cleanLines, "\n")
}

// LoadAndSplit loads the HTML document and splits it into chunks
func (loader *HTMLLoader) LoadAndSplit(
	ctx context.Context,
	splitter TextSplitter,
) ([]*Document, error) {
	docs, err := loader.Load(ctx)
	if err != nil {
		return nil, err
	}

	return SplitDocuments(docs, splitter)
}

// WebCrawler crawls multiple web pages and loads their content
type WebCrawler struct {
	startURL    string
	maxDepth    int
	maxPages    int
	loader      *HTMLLoader
	visited     map[string]bool
	sameDomain  bool
	baseURL     *url.URL
}

// WebCrawlerOptions contains configuration options for web crawler
type WebCrawlerOptions struct {
	// StartURL is the starting URL to crawl
	StartURL string

	// MaxDepth is the maximum crawl depth (default: 1)
	MaxDepth int

	// MaxPages is the maximum number of pages to crawl (default: 10)
	MaxPages int

	// SameDomain restricts crawling to the same domain (default: true)
	SameDomain bool

	// HTMLLoaderOptions for loading individual pages
	HTMLLoaderOptions HTMLLoaderOptions
}

// NewWebCrawler creates a new web crawler
func NewWebCrawler(options WebCrawlerOptions) (*WebCrawler, error) {
	if options.StartURL == "" {
		return nil, fmt.Errorf("start URL is required")
	}

	if options.MaxDepth == 0 {
		options.MaxDepth = 1
	}

	if options.MaxPages == 0 {
		options.MaxPages = 10
	}

	// Parse base URL
	baseURL, err := url.Parse(options.StartURL)
	if err != nil {
		return nil, fmt.Errorf("invalid start URL: %w", err)
	}

	// Create HTML loader
	options.HTMLLoaderOptions.URL = options.StartURL
	options.HTMLLoaderOptions.ExtractLinks = true
	loader, err := NewHTMLLoader(options.HTMLLoaderOptions)
	if err != nil {
		return nil, err
	}

	return &WebCrawler{
		startURL:   options.StartURL,
		maxDepth:   options.MaxDepth,
		maxPages:   options.MaxPages,
		loader:     loader,
		visited:    make(map[string]bool),
		sameDomain: options.SameDomain,
		baseURL:    baseURL,
	}, nil
}

// Crawl crawls web pages starting from the start URL
func (crawler *WebCrawler) Crawl(ctx context.Context) ([]*Document, error) {
	var allDocs []*Document

	err := crawler.crawlRecursive(ctx, crawler.startURL, 0, &allDocs)
	if err != nil {
		return nil, err
	}

	return allDocs, nil
}

// crawlRecursive recursively crawls pages
func (crawler *WebCrawler) crawlRecursive(
	ctx context.Context,
	currentURL string,
	depth int,
	allDocs *[]*Document,
) error {
	// Check limits
	if depth > crawler.maxDepth || len(*allDocs) >= crawler.maxPages {
		return nil
	}

	// Check if already visited
	if crawler.visited[currentURL] {
		return nil
	}
	crawler.visited[currentURL] = true

	// Check same domain restriction
	if crawler.sameDomain {
		parsedURL, err := url.Parse(currentURL)
		if err != nil || parsedURL.Host != crawler.baseURL.Host {
			return nil
		}
	}

	// Load page
	crawler.loader.url = currentURL
	docs, err := crawler.loader.Load(ctx)
	if err != nil {
		// Log error but continue crawling
		return nil
	}

	// Add documents
	*allDocs = append(*allDocs, docs...)

	// Check if we've reached max pages
	if len(*allDocs) >= crawler.maxPages {
		return nil
	}

	// Extract links and crawl recursively
	if len(docs) > 0 && docs[0].Metadata["links"] != nil {
		links, ok := docs[0].Metadata["links"].([]string)
		if ok {
			for _, link := range links {
				if len(*allDocs) >= crawler.maxPages {
					break
				}

				err := crawler.crawlRecursive(ctx, link, depth+1, allDocs)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}
