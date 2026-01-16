package tools

import (
	"context"
	"testing"
)

// TestTavilySearchTool 测试 Tavily 搜索工具。
func TestTavilySearchTool(t *testing.T) {
	// 需要 API key 才能运行真实测试
	t.Skip("Skipping Tavily search test - requires API key")
	
	apiKey := "test-api-key"
	tool := NewTavilySearch(apiKey, nil)
	
	if tool.GetName() != "tavily_search" {
		t.Errorf("Expected name 'tavily_search', got %s", tool.GetName())
	}
	
	desc := tool.GetDescription()
	if len(desc) == 0 {
		t.Error("Expected non-empty description")
	}
	
	params := tool.GetParameters()
	if len(params.Properties) == 0 {
		t.Error("Expected at least one parameter")
	}
}

// TestGoogleSearchTool 测试 Google 搜索工具。
func TestGoogleSearchTool(t *testing.T) {
	// 需要 API key 才能运行真实测试
	t.Skip("Skipping Google search test - requires API key")
	
	apiKey := "test-api-key"
	engineID := "test-engine-id"
	tool := NewGoogleSearch(apiKey, engineID, nil)
	
	if tool.GetName() != "google_search" {
		t.Errorf("Expected name 'google_search', got %s", tool.GetName())
	}
	
	desc := tool.GetDescription()
	if len(desc) == 0 {
		t.Error("Expected non-empty description")
	}
}

// TestTavilySearchConfig 测试 Tavily 搜索配置。
func TestTavilySearchConfig(t *testing.T) {
	config := DefaultTavilySearchConfig()
	
	if config.MaxResults != 5 {
		t.Errorf("Expected MaxResults 5, got %d", config.MaxResults)
	}
	
	if config.SearchDepth != "basic" {
		t.Errorf("Expected SearchDepth 'basic', got %s", config.SearchDepth)
	}
	
	if !config.IncludeAnswer {
		t.Error("Expected IncludeAnswer to be true")
	}
}

// TestGoogleSearchConfig 测试 Google 搜索配置。
func TestGoogleSearchConfig(t *testing.T) {
	config := DefaultGoogleSearchConfig()
	
	if config.MaxResults != 5 {
		t.Errorf("Expected MaxResults 5, got %d", config.MaxResults)
	}
	
	if config.Language != "en" {
		t.Errorf("Expected Language 'en', got %s", config.Language)
	}
	
	if config.SafeSearch != "medium" {
		t.Errorf("Expected SafeSearch 'medium', got %s", config.SafeSearch)
	}
}

// TestTavilySearchExecuteWithoutAPIKey 测试没有 API key 的错误。
func TestTavilySearchExecuteWithoutAPIKey(t *testing.T) {
	tool := NewTavilySearch("", nil)
	
	ctx := context.Background()
	input := map[string]any{
		"query": "test query",
	}
	
	_, err := tool.Execute(ctx, input)
	if err == nil {
		t.Error("Expected error when API key is missing")
	}
}

// TestGoogleSearchExecuteWithoutAPIKey 测试没有 API key 的错误。
func TestGoogleSearchExecuteWithoutAPIKey(t *testing.T) {
	tool := NewGoogleSearch("", "", nil)
	
	ctx := context.Background()
	input := map[string]any{
		"query": "test query",
	}
	
	_, err := tool.Execute(ctx, input)
	if err == nil {
		t.Error("Expected error when API key is missing")
	}
}

// TestSearchToolsToTypesTool 测试转换为 types.Tool。
func TestSearchToolsToTypesTool(t *testing.T) {
	// Tavily
	tavilyTool := NewTavilySearch("test-key", nil)
	typesTool := tavilyTool.ToTypesTool()
	
	if typesTool.Name != "tavily_search" {
		t.Errorf("Expected name 'tavily_search', got %s", typesTool.Name)
	}
	
	// Google
	googleTool := NewGoogleSearch("test-key", "test-engine", nil)
	typesTool = googleTool.ToTypesTool()
	
	if typesTool.Name != "google_search" {
		t.Errorf("Expected name 'google_search', got %s", typesTool.Name)
	}
}
