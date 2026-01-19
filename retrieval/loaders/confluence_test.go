package loaders

import (
	"testing"
)

func TestNewConfluenceLoader(t *testing.T) {
	tests := []struct {
		name      string
		config    ConfluenceLoaderConfig
		wantError bool
	}{
		{
			name: "valid config",
			config: ConfluenceLoaderConfig{
				URL:      "https://test.atlassian.net/wiki",
				Username: "user@example.com",
				APIToken: "test-token",
			},
			wantError: false,
		},
		{
			name: "missing URL",
			config: ConfluenceLoaderConfig{
				Username: "user@example.com",
				APIToken: "test-token",
			},
			wantError: true,
		},
		{
			name: "missing username",
			config: ConfluenceLoaderConfig{
				URL:      "https://test.atlassian.net/wiki",
				APIToken: "test-token",
			},
			wantError: true,
		},
		{
			name: "missing API token",
			config: ConfluenceLoaderConfig{
				URL:      "https://test.atlassian.net/wiki",
				Username: "user@example.com",
			},
			wantError: true,
		},
		{
			name: "URL with trailing slash",
			config: ConfluenceLoaderConfig{
				URL:      "https://test.atlassian.net/wiki/",
				Username: "user@example.com",
				APIToken: "test-token",
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loader, err := NewConfluenceLoader(tt.config)
			if tt.wantError {
				if err == nil {
					t.Error("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if loader == nil {
					t.Error("expected loader, got nil")
				}
				// 验证 URL 处理
				if loader != nil && loader.config.URL[len(loader.config.URL)-1] == '/' {
					t.Error("URL should not end with slash")
				}
			}
		})
	}
}

func TestStripHTML(t *testing.T) {
	loader := &ConfluenceLoader{}

	tests := []struct {
		name     string
		html     string
		expected string
	}{
		{
			name:     "simple paragraph",
			html:     "<p>Hello World</p>",
			expected: "Hello World",
		},
		{
			name:     "with br tags",
			html:     "Line 1<br/>Line 2",
			expected: "Line 1Line 2",
		},
		{
			name:     "with multiple tags",
			html:     "<div><h1>Title</h1><p>Content</p></div>",
			expected: "TitleContent",
		},
		{
			name:     "with list",
			html:     "<ul><li>Item 1</li><li>Item 2</li></ul>",
			expected: "Item 1Item 2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := loader.stripHTML(tt.html)
			if result != tt.expected {
				t.Errorf("stripHTML() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestExtractContent(t *testing.T) {
	loader := &ConfluenceLoader{}

	t.Run("with storage content", func(t *testing.T) {
		page := &ConfluencePage{
			Title: "Test Page",
			Body: ConfluenceBody{
				Storage: ConfluenceStorage{
					Value: "<p>Test content</p>",
				},
			},
		}

		content := loader.extractContent(page)
		if content == "" {
			t.Error("expected non-empty content")
		}
	})

	t.Run("without storage content", func(t *testing.T) {
		page := &ConfluencePage{
			Title: "Test Page",
			Body:  ConfluenceBody{},
		}

		content := loader.extractContent(page)
		if content != "Test Page" {
			t.Errorf("expected title as content, got %q", content)
		}
	})
}

// 集成测试（需要真实 Confluence 实例）
func TestConfluenceLoaderIntegration(t *testing.T) {
	t.Skip("Integration test - requires Confluence instance")

	// 取消注释以运行集成测试
	// config := ConfluenceLoaderConfig{
	// 	URL:      os.Getenv("CONFLUENCE_URL"),
	// 	Username: os.Getenv("CONFLUENCE_USERNAME"),
	// 	APIToken: os.Getenv("CONFLUENCE_API_TOKEN"),
	// 	SpaceKey: "TEST",
	// 	MaxPages: 5,
	// }
	//
	// if config.URL == "" || config.Username == "" || config.APIToken == "" {
	// 	t.Skip("Confluence configuration not set")
	// }
	//
	// loader, err := NewConfluenceLoader(config)
	// if err != nil {
	// 	t.Fatalf("Failed to create loader: %v", err)
	// }
	//
	// ctx := context.Background()
	//
	// // 测试加载空间
	// docs, err := loader.LoadSpace(ctx, config.SpaceKey)
	// if err != nil {
	// 	t.Fatalf("Failed to load space: %v", err)
	// }
	//
	// t.Logf("Loaded %d pages from space %s", len(docs), config.SpaceKey)
}
