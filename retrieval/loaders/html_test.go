package loaders

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHTMLLoader tests HTML document loader
func TestHTMLLoader(t *testing.T) {
	ctx := context.Background()

	t.Run("NewHTMLLoader_FromFile", func(t *testing.T) {
		loader, err := NewHTMLLoader(HTMLLoaderOptions{
			Path:          "test.html",
			RemoveScripts: true,
			RemoveStyles:  true,
		})

		require.NoError(t, err)
		assert.NotNil(t, loader)
		assert.Equal(t, "test.html", loader.path)
		assert.True(t, loader.removeScripts)
		assert.True(t, loader.removeStyles)
	})

	t.Run("NewHTMLLoader_FromURL", func(t *testing.T) {
		loader, err := NewHTMLLoader(HTMLLoaderOptions{
			URL:             "https://example.com",
			ExtractLinks:    true,
			ExtractMetaTags: true,
		})

		require.NoError(t, err)
		assert.NotNil(t, loader)
		assert.Equal(t, "https://example.com", loader.url)
		assert.True(t, loader.extractLinks)
		assert.True(t, loader.extractMetaTags)
	})

	t.Run("NewHTMLLoader_NoPathOrURL", func(t *testing.T) {
		loader, err := NewHTMLLoader(HTMLLoaderOptions{})

		assert.Error(t, err)
		assert.Nil(t, loader)
	})

	t.Run("Load_SimpleHTML", func(t *testing.T) {
		// Create a test HTML file
		testFile := filepath.Join(t.TempDir(), "test.html")
		htmlContent := `
<!DOCTYPE html>
<html>
<head>
    <title>Test Page</title>
    <meta name="description" content="Test description">
</head>
<body>
    <h1>Hello World</h1>
    <p>This is a test paragraph.</p>
</body>
</html>
`
		err := os.WriteFile(testFile, []byte(htmlContent), 0644)
		require.NoError(t, err)

		loader, err := NewHTMLLoader(HTMLLoaderOptions{
			Path:            testFile,
			ExtractMetaTags: true,
		})
		require.NoError(t, err)

		docs, err := loader.Load(ctx)
		require.NoError(t, err)
		require.Len(t, docs, 1)

		assert.Contains(t, docs[0].Content, "Hello World")
		assert.Contains(t, docs[0].Content, "test paragraph")
		assert.Equal(t, testFile, docs[0].Metadata["source"])
		assert.Equal(t, "html", docs[0].Metadata["file_type"])
		assert.Equal(t, "Test Page", docs[0].Metadata["title"])
		assert.Equal(t, "Test description", docs[0].Metadata["description"])
	})

	t.Run("Load_RemoveScriptsAndStyles", func(t *testing.T) {
		testFile := filepath.Join(t.TempDir(), "test.html")
		htmlContent := `
<!DOCTYPE html>
<html>
<head>
    <style>body { color: red; }</style>
</head>
<body>
    <p>Visible content</p>
    <script>console.log('hidden');</script>
</body>
</html>
`
		err := os.WriteFile(testFile, []byte(htmlContent), 0644)
		require.NoError(t, err)

		loader, err := NewHTMLLoader(HTMLLoaderOptions{
			Path:          testFile,
			RemoveScripts: true,
			RemoveStyles:  true,
		})
		require.NoError(t, err)

		docs, err := loader.Load(ctx)
		require.NoError(t, err)
		require.Len(t, docs, 1)

		assert.Contains(t, docs[0].Content, "Visible content")
		assert.NotContains(t, docs[0].Content, "color: red")
		assert.NotContains(t, docs[0].Content, "console.log")
	})

	t.Run("Load_WithSelector", func(t *testing.T) {
		testFile := filepath.Join(t.TempDir(), "test.html")
		htmlContent := `
<!DOCTYPE html>
<html>
<body>
    <div class="header">Header content</div>
    <article class="content">Article content</article>
    <div class="footer">Footer content</div>
</body>
</html>
`
		err := os.WriteFile(testFile, []byte(htmlContent), 0644)
		require.NoError(t, err)

		loader, err := NewHTMLLoader(HTMLLoaderOptions{
			Path:     testFile,
			Selector: "article.content",
		})
		require.NoError(t, err)

		docs, err := loader.Load(ctx)
		require.NoError(t, err)
		require.Len(t, docs, 1)

		assert.Contains(t, docs[0].Content, "Article content")
		assert.NotContains(t, docs[0].Content, "Header content")
		assert.NotContains(t, docs[0].Content, "Footer content")
	})

	t.Run("Load_ExtractLinks", func(t *testing.T) {
		testFile := filepath.Join(t.TempDir(), "test.html")
		htmlContent := `
<!DOCTYPE html>
<html>
<body>
    <a href="https://example.com/page1">Link 1</a>
    <a href="/page2">Link 2</a>
    <a href="javascript:void(0)">JS Link</a>
</body>
</html>
`
		err := os.WriteFile(testFile, []byte(htmlContent), 0644)
		require.NoError(t, err)

		loader, err := NewHTMLLoader(HTMLLoaderOptions{
			Path:         testFile,
			ExtractLinks: true,
		})
		require.NoError(t, err)

		docs, err := loader.Load(ctx)
		require.NoError(t, err)
		require.Len(t, docs, 1)

		links, ok := docs[0].Metadata["links"].([]string)
		assert.True(t, ok)
		assert.Contains(t, links, "https://example.com/page1")
		assert.NotContains(t, links, "javascript:void(0)")
	})

	t.Run("Load_FromURL", func(t *testing.T) {
		// Create a test HTTP server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			html := `
<!DOCTYPE html>
<html>
<head><title>Test Server</title></head>
<body><h1>Hello from server</h1></body>
</html>
`
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(html))
		}))
		defer server.Close()

		loader, err := NewHTMLLoader(HTMLLoaderOptions{
			URL:             server.URL,
			ExtractMetaTags: true,
		})
		require.NoError(t, err)

		docs, err := loader.Load(ctx)
		require.NoError(t, err)
		require.Len(t, docs, 1)

		assert.Contains(t, docs[0].Content, "Hello from server")
		assert.Equal(t, "Test Server", docs[0].Metadata["title"])
	})

	t.Run("Load_FromURL_NotFound", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		loader, err := NewHTMLLoader(HTMLLoaderOptions{
			URL: server.URL,
		})
		require.NoError(t, err)

		docs, err := loader.Load(ctx)
		assert.Error(t, err)
		assert.Nil(t, docs)
	})

	t.Run("Load_FromURL_Timeout", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(2 * time.Second)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		loader, err := NewHTMLLoader(HTMLLoaderOptions{
			URL:     server.URL,
			Timeout: 100 * time.Millisecond,
		})
		require.NoError(t, err)

		docs, err := loader.Load(ctx)
		assert.Error(t, err)
		assert.Nil(t, docs)
	})

	t.Run("Load_FileNotFound", func(t *testing.T) {
		loader, err := NewHTMLLoader(HTMLLoaderOptions{
			Path: "nonexistent.html",
		})
		require.NoError(t, err)

		docs, err := loader.Load(ctx)
		assert.Error(t, err)
		assert.Nil(t, docs)
	})

	t.Run("LoadAndSplit", func(t *testing.T) {
		testFile := filepath.Join(t.TempDir(), "test.html")
		htmlContent := `
<!DOCTYPE html>
<html>
<body>` + strings.Repeat("<p>This is a long paragraph. </p>", 50) + `</body>
</html>
`
		err := os.WriteFile(testFile, []byte(htmlContent), 0644)
		require.NoError(t, err)

		loader, err := NewHTMLLoader(HTMLLoaderOptions{
			Path: testFile,
		})
		require.NoError(t, err)

		splitter := NewCharacterTextSplitter(CharacterTextSplitterOptions{
			ChunkSize:    200,
			ChunkOverlap: 20,
		})

		docs, err := loader.LoadAndSplit(ctx, splitter)
		require.NoError(t, err)
		assert.Greater(t, len(docs), 1)
	})

	t.Run("resolveURL", func(t *testing.T) {
		tests := []struct {
			base     string
			href     string
			expected string
		}{
			{
				base:     "https://example.com/page",
				href:     "https://other.com/link",
				expected: "https://other.com/link",
			},
			{
				base:     "https://example.com/page",
				href:     "/about",
				expected: "https://example.com/about",
			},
			{
				base:     "https://example.com/dir/page",
				href:     "other.html",
				expected: "https://example.com/dir/other.html",
			},
			{
				base:     "https://example.com",
				href:     "javascript:void(0)",
				expected: "",
			},
			{
				base:     "https://example.com",
				href:     "mailto:test@example.com",
				expected: "",
			},
		}

		for _, tt := range tests {
			result := resolveURL(tt.base, tt.href)
			assert.Equal(t, tt.expected, result, "base: %s, href: %s", tt.base, tt.href)
		}
	})

	t.Run("cleanWhitespace", func(t *testing.T) {
		tests := []struct {
			input    string
			expected string
		}{
			{
				input:    "Hello    World",
				expected: "Hello World",
			},
			{
				input:    "Line 1\n\n\nLine 2",
				expected: "Line 1\nLine 2",
			},
			{
				input:    "  Spaces  \n  Everywhere  ",
				expected: "Spaces\nEverywhere",
			},
		}

		for _, tt := range tests {
			result := cleanWhitespace(tt.input)
			assert.Equal(t, tt.expected, result)
		}
	})
}

// TestWebCrawler tests web crawler functionality
func TestWebCrawler(t *testing.T) {
	ctx := context.Background()

	t.Run("NewWebCrawler", func(t *testing.T) {
		crawler, err := NewWebCrawler(WebCrawlerOptions{
			StartURL:   "https://example.com",
			MaxDepth:   2,
			MaxPages:   5,
			SameDomain: true,
		})

		require.NoError(t, err)
		assert.NotNil(t, crawler)
		assert.Equal(t, "https://example.com", crawler.startURL)
		assert.Equal(t, 2, crawler.maxDepth)
		assert.Equal(t, 5, crawler.maxPages)
	})

	t.Run("NewWebCrawler_NoStartURL", func(t *testing.T) {
		crawler, err := NewWebCrawler(WebCrawlerOptions{})

		assert.Error(t, err)
		assert.Nil(t, crawler)
	})

	t.Run("Crawl_SinglePage", func(t *testing.T) {
		// Create test server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			html := `
<!DOCTYPE html>
<html>
<body>
    <h1>Page</h1>
    <a href="/page2">Link to page 2</a>
</body>
</html>
`
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(html))
		}))
		defer server.Close()

		crawler, err := NewWebCrawler(WebCrawlerOptions{
			StartURL: server.URL,
			MaxDepth: 0, // Only start page
			MaxPages: 1,
		})
		require.NoError(t, err)

		docs, err := crawler.Crawl(ctx)
		require.NoError(t, err)
		assert.Len(t, docs, 1)
		assert.Contains(t, docs[0].Content, "Page")
	})

	t.Run("Crawl_MultiplePages", func(t *testing.T) {
		pageCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			pageCount++
			html := `
<!DOCTYPE html>
<html>
<body>
    <h1>Page ` + r.URL.Path + `</h1>
    <a href="/page2">Link 2</a>
    <a href="/page3">Link 3</a>
</body>
</html>
`
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(html))
		}))
		defer server.Close()

		crawler, err := NewWebCrawler(WebCrawlerOptions{
			StartURL: server.URL,
			MaxDepth: 1,
			MaxPages: 3,
		})
		require.NoError(t, err)

		docs, err := crawler.Crawl(ctx)
		require.NoError(t, err)
		assert.LessOrEqual(t, len(docs), 3)
	})

	t.Run("Crawl_SameDomainRestriction", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			html := `
<!DOCTYPE html>
<html>
<body>
    <a href="` + server.URL + `/page2">Internal</a>
    <a href="https://external.com">External</a>
</body>
</html>
`
			w.Write([]byte(html))
		}))
		defer server.Close()

		crawler, err := NewWebCrawler(WebCrawlerOptions{
			StartURL:   server.URL,
			MaxDepth:   2,
			MaxPages:   10,
			SameDomain: true,
		})
		require.NoError(t, err)

		docs, err := crawler.Crawl(ctx)
		require.NoError(t, err)

		// Should only crawl pages from same domain
		for _, doc := range docs {
			source := doc.Metadata["source"].(string)
			assert.True(t, strings.HasPrefix(source, server.URL))
		}
	})
}
