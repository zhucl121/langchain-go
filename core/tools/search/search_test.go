package search

import (
	"context"
	"testing"
	"time"
)

// TestSearchOptions 测试搜索选项
func TestSearchOptions(t *testing.T) {
	t.Run("default options", func(t *testing.T) {
		opts := DefaultSearchOptions()
		
		if opts.MaxResults != 5 {
			t.Errorf("Expected MaxResults=5, got %d", opts.MaxResults)
		}
		
		if opts.Language != "en" {
			t.Errorf("Expected Language=en, got %s", opts.Language)
		}
		
		if opts.SafeSearch != "moderate" {
			t.Errorf("Expected SafeSearch=moderate, got %s", opts.SafeSearch)
		}
		
		if opts.Timeout != 30*time.Second {
			t.Errorf("Expected Timeout=30s, got %v", opts.Timeout)
		}
	})
	
	t.Run("validate options", func(t *testing.T) {
		tests := []struct {
			name    string
			opts    SearchOptions
			wantErr bool
		}{
			{
				name: "valid options",
				opts: SearchOptions{
					MaxResults: 10,
					SafeSearch: "moderate",
					Timeout:    10 * time.Second,
				},
				wantErr: false,
			},
			{
				name: "invalid max results - zero",
				opts: SearchOptions{
					MaxResults: 0,
				},
				wantErr: true,
			},
			{
				name: "invalid max results - too large",
				opts: SearchOptions{
					MaxResults: 101,
				},
				wantErr: true,
			},
			{
				name: "invalid safe search",
				opts: SearchOptions{
					MaxResults: 5,
					SafeSearch: "invalid",
				},
				wantErr: true,
			},
			{
				name: "auto-fix timeout",
				opts: SearchOptions{
					MaxResults: 5,
					Timeout:    0,
				},
				wantErr: false,
			},
		}
		
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := tt.opts.Validate()
				if (err != nil) != tt.wantErr {
					t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				}
			})
		}
	})
}

// TestSearchResult 测试搜索结果
func TestSearchResult(t *testing.T) {
	now := time.Now()
	result := SearchResult{
		Title:         "Test Title",
		Link:          "https://example.com",
		Snippet:       "Test snippet",
		Source:        "example.com",
		PublishedDate: &now,
		Metadata: map[string]any{
			"key": "value",
		},
	}
	
	if result.Title != "Test Title" {
		t.Errorf("Expected Title='Test Title', got %s", result.Title)
	}
	
	if result.Link != "https://example.com" {
		t.Errorf("Expected Link='https://example.com', got %s", result.Link)
	}
}

// MockSearchProvider 用于测试的 Mock 搜索提供者
type MockSearchProvider struct {
	name      SearchEngine
	available bool
	results   []SearchResult
	err       error
}

func (m *MockSearchProvider) Search(ctx context.Context, query string, options SearchOptions) (*SearchResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	
	return &SearchResponse{
		Results:      m.results,
		Query:        query,
		Engine:       m.name,
		TotalResults: len(m.results),
	}, nil
}

func (m *MockSearchProvider) GetName() SearchEngine {
	return m.name
}

func (m *MockSearchProvider) IsAvailable() bool {
	return m.available
}

// TestSearchTool 测试搜索工具
func TestSearchTool(t *testing.T) {
	t.Run("create search tool", func(t *testing.T) {
		provider := &MockSearchProvider{
			name:      EngineGoogle,
			available: true,
			results: []SearchResult{
				{Title: "Result 1", Link: "https://example.com/1", Snippet: "Snippet 1"},
				{Title: "Result 2", Link: "https://example.com/2", Snippet: "Snippet 2"},
			},
		}
		
		opts := DefaultSearchOptions()
		tool, err := NewSearchTool(provider, opts)
		
		if err != nil {
			t.Fatalf("Failed to create search tool: %v", err)
		}
		
		if tool.GetName() != "google_search" {
			t.Errorf("Expected name='google_search', got %s", tool.GetName())
		}
	})
	
	t.Run("search tool with unavailable provider", func(t *testing.T) {
		provider := &MockSearchProvider{
			name:      EngineGoogle,
			available: false,
		}
		
		opts := DefaultSearchOptions()
		_, err := NewSearchTool(provider, opts)
		
		if err == nil {
			t.Error("Expected error for unavailable provider")
		}
	})
	
	t.Run("execute search", func(t *testing.T) {
		provider := &MockSearchProvider{
			name:      EngineDuckDuckGo,
			available: true,
			results: []SearchResult{
				{Title: "Test", Link: "https://test.com", Snippet: "Test snippet"},
			},
		}
		
		opts := DefaultSearchOptions()
		tool, _ := NewSearchTool(provider, opts)
		
		ctx := context.Background()
		result, err := tool.Execute(ctx, map[string]any{
			"query": "test query",
		})
		
		if err != nil {
			t.Fatalf("Execute failed: %v", err)
		}
		
		resultStr, ok := result.(string)
		if !ok {
			t.Fatal("Result is not a string")
		}
		
		if resultStr == "" {
			t.Error("Result string is empty")
		}
	})
	
	t.Run("execute with invalid args", func(t *testing.T) {
		provider := &MockSearchProvider{
			name:      EngineDuckDuckGo,
			available: true,
		}
		
		opts := DefaultSearchOptions()
		tool, _ := NewSearchTool(provider, opts)
		
		ctx := context.Background()
		
		// Missing query
		_, err := tool.Execute(ctx, map[string]any{})
		if err == nil {
			t.Error("Expected error for missing query")
		}
		
		// Empty query
		_, err = tool.Execute(ctx, map[string]any{"query": ""})
		if err == nil {
			t.Error("Expected error for empty query")
		}
	})
	
	t.Run("execute with custom max_results", func(t *testing.T) {
		provider := &MockSearchProvider{
			name:      EngineBing,
			available: true,
			results:   []SearchResult{{Title: "Test", Link: "https://test.com", Snippet: "Test"}},
		}
		
		opts := DefaultSearchOptions()
		tool, _ := NewSearchTool(provider, opts)
		
		ctx := context.Background()
		_, err := tool.Execute(ctx, map[string]any{
			"query":       "test",
			"max_results": 10,
		})
		
		if err != nil {
			t.Errorf("Execute with max_results failed: %v", err)
		}
	})
}

// TestDuckDuckGoProvider 测试 DuckDuckGo 提供者
func TestDuckDuckGoProvider(t *testing.T) {
	t.Run("create provider", func(t *testing.T) {
		provider := NewDuckDuckGoProvider(DuckDuckGoConfig{})
		
		if provider.GetName() != EngineDuckDuckGo {
			t.Errorf("Expected engine=duckduckgo, got %s", provider.GetName())
		}
		
		if !provider.IsAvailable() {
			t.Error("DuckDuckGo should always be available")
		}
	})
	
	t.Run("clean text", func(t *testing.T) {
		provider := NewDuckDuckGoProvider(DuckDuckGoConfig{})
		
		tests := []struct {
			input  string
			expect string
		}{
			{"<b>bold</b> text", "bold text"},
			{"test &amp; test", "test & test"},
			{"&lt;tag&gt;", "<tag>"},
			{"&quot;quoted&quot;", "\"quoted\""},
			{"  spaces  ", "spaces"},
		}
		
		for _, tt := range tests {
			result := provider.cleanText(tt.input)
			if result != tt.expect {
				t.Errorf("cleanText(%q) = %q, want %q", tt.input, result, tt.expect)
			}
		}
	})
	
	t.Run("get region code", func(t *testing.T) {
		provider := NewDuckDuckGoProvider(DuckDuckGoConfig{})
		
		tests := []struct {
			language string
			region   string
			expect   string
		}{
			{"en", "us", "en-us"},
			{"zh", "cn", "zh-cn"},
			{"", "uk", "en-uk"},
			{"zh-CN", "", "zh-cn"},
		}
		
		for _, tt := range tests {
			result := provider.getRegionCode(tt.language, tt.region)
			if result != tt.expect {
				t.Errorf("getRegionCode(%q, %q) = %q, want %q", 
					tt.language, tt.region, result, tt.expect)
			}
		}
	})
}

// TestGoogleProvider 测试 Google 提供者
func TestGoogleProvider(t *testing.T) {
	t.Run("create provider", func(t *testing.T) {
		provider := NewGoogleProvider(GoogleConfig{
			APIKey:   "test-key",
			EngineID: "test-engine",
		})
		
		if provider.GetName() != EngineGoogle {
			t.Errorf("Expected engine=google, got %s", provider.GetName())
		}
		
		if !provider.IsAvailable() {
			t.Error("Provider should be available with API key and engine ID")
		}
	})
	
	t.Run("unavailable without credentials", func(t *testing.T) {
		provider := NewGoogleProvider(GoogleConfig{})
		
		if provider.IsAvailable() {
			t.Error("Provider should not be available without credentials")
		}
	})
}

// TestBingProvider 测试 Bing 提供者
func TestBingProvider(t *testing.T) {
	t.Run("create provider", func(t *testing.T) {
		provider := NewBingProvider(BingConfig{
			APIKey: "test-key",
		})
		
		if provider.GetName() != EngineBing {
			t.Errorf("Expected engine=bing, got %s", provider.GetName())
		}
		
		if !provider.IsAvailable() {
			t.Error("Provider should be available with API key")
		}
	})
	
	t.Run("unavailable without API key", func(t *testing.T) {
		provider := NewBingProvider(BingConfig{})
		
		if provider.IsAvailable() {
			t.Error("Provider should not be available without API key")
		}
	})
	
	t.Run("get market code", func(t *testing.T) {
		provider := NewBingProvider(BingConfig{APIKey: "test"})
		
		tests := []struct {
			language string
			region   string
			expect   string
		}{
			{"en", "US", "en-US"},
			{"zh", "CN", "zh-CN"},
			{"zh", "TW", "zh-TW"},
			{"", "", "en-US"},
		}
		
		for _, tt := range tests {
			result := provider.getMarketCode(tt.language, tt.region)
			if result != tt.expect {
				t.Errorf("getMarketCode(%q, %q) = %q, want %q",
					tt.language, tt.region, result, tt.expect)
			}
		}
	})
}

// TestFormatResults 测试结果格式化
func TestFormatResults(t *testing.T) {
	provider := &MockSearchProvider{
		name:      EngineGoogle,
		available: true,
		results: []SearchResult{
			{
				Title:   "Test Result 1",
				Link:    "https://example.com/1",
				Snippet: "This is test result 1",
				Source:  "example.com",
			},
			{
				Title:   "Test Result 2",
				Link:    "https://example.com/2",
				Snippet: "This is test result 2",
			},
		},
	}
	
	opts := DefaultSearchOptions()
	tool, _ := NewSearchTool(provider, opts)
	
	response := &SearchResponse{
		Results:      provider.results,
		Query:        "test query",
		Engine:       EngineGoogle,
		TotalResults: 2,
	}
	
	formatted := tool.formatResults(response)
	
	if formatted == "" {
		t.Error("Formatted result is empty")
	}
	
	// 检查是否包含关键信息
	if !contains(formatted, "Test Result 1") {
		t.Error("Formatted result missing title")
	}
	
	if !contains(formatted, "https://example.com/1") {
		t.Error("Formatted result missing link")
	}
	
	if !contains(formatted, "This is test result 1") {
		t.Error("Formatted result missing snippet")
	}
}

// contains 检查字符串是否包含子串
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
