package prompts

import (
	"context"
	"testing"
	"time"
)

// TestPromptHub 测试 Prompt Hub 基本功能。
func TestPromptHub(t *testing.T) {
	config := DefaultPromptHubConfig()
	hub := NewPromptHub(&config)
	
	if hub == nil {
		t.Error("Expected hub to be created")
	}
	
	if hub.baseURL == "" {
		t.Error("Expected baseURL to be set")
	}
}

// TestPromptHubConfig 测试配置。
func TestPromptHubConfig(t *testing.T) {
	config := DefaultPromptHubConfig()
	
	if config.CacheEnabled != true {
		t.Error("Expected CacheEnabled to be true")
	}
	
	if config.CacheTTL != 24*time.Hour {
		t.Errorf("Expected CacheTTL 24h, got %v", config.CacheTTL)
	}
	
	if config.Timeout != 30*time.Second {
		t.Errorf("Expected Timeout 30s, got %v", config.Timeout)
	}
}

// TestPromptCache 测试 Prompt 缓存。
func TestPromptCache(t *testing.T) {
	cache := newPromptCache(1 * time.Second)
	
	// 创建测试 prompt
	prompt, _ := NewPromptTemplate(PromptTemplateConfig{
		Template:       "Hello {{.name}}",
		InputVariables: []string{"name"},
	})
	
	// 设置缓存
	cache.set("test-key", prompt)
	
	// 获取缓存
	cached, ok := cache.get("test-key")
	if !ok {
		t.Error("Expected to get cached prompt")
	}
	
	if cached == nil {
		t.Error("Expected non-nil cached prompt")
	}
	
	// 测试过期
	time.Sleep(1500 * time.Millisecond)
	_, ok = cache.get("test-key")
	if ok {
		t.Error("Expected cache to be expired")
	}
}

// TestPromptCacheClear 测试清除缓存。
func TestPromptCacheClear(t *testing.T) {
	cache := newPromptCache(1 * time.Hour)
	
	prompt, _ := NewPromptTemplate(PromptTemplateConfig{
		Template:       "Test",
		InputVariables: []string{},
	})
	
	cache.set("key1", prompt)
	cache.set("key2", prompt)
	
	// 清除缓存
	cache.clear()
	
	_, ok := cache.get("key1")
	if ok {
		t.Error("Expected cache to be cleared")
	}
}

// TestPullPrompt 测试拉取 Prompt。
func TestPullPrompt(t *testing.T) {
	// 这需要实际的网络连接和有效的 Hub
	t.Skip("Skipping prompt pull test - requires network and valid hub")
	
	hub := NewPromptHub(nil)
	ctx := context.Background()
	
	prompt, err := hub.PullPrompt(ctx, "hwchase17/react")
	if err != nil {
		t.Errorf("Failed to pull prompt: %v", err)
	}
	
	if prompt == nil {
		t.Error("Expected non-nil prompt")
	}
}

// TestGeneratePrompt 测试生成 Prompt。
func TestGeneratePrompt(t *testing.T) {
	task := "Classify sentiment"
	examples := []string{
		"Input: Great movie! Output: positive",
		"Input: Terrible film. Output: negative",
	}
	
	prompt, err := GeneratePrompt(task, examples)
	if err != nil {
		t.Errorf("Failed to generate prompt: %v", err)
	}
	
	if prompt == nil {
		t.Error("Expected non-nil prompt")
	}
	
	// 测试格式化
	result, err := prompt.Format(map[string]any{
		"input": "This movie is amazing!",
	})
	
	if err != nil {
		t.Errorf("Failed to format prompt: %v", err)
	}
	
	if len(result) == 0 {
		t.Error("Expected non-empty result")
	}
}

// TestGetDefaultHub 测试默认 Hub。
func TestGetDefaultHub(t *testing.T) {
	hub1 := getDefaultHub()
	hub2 := getDefaultHub()
	
	// 应该返回同一个实例
	if hub1 != hub2 {
		t.Error("Expected same hub instance")
	}
}

// TestPromptHubClearCache 测试清除 Hub 缓存。
func TestPromptHubClearCache(t *testing.T) {
	config := DefaultPromptHubConfig()
	config.CacheEnabled = true
	hub := NewPromptHub(&config)
	
	// 应该不会 panic
	hub.ClearCache()
}

// TestPromptVersion 测试 Prompt 版本。
func TestPromptVersion(t *testing.T) {
	version := PromptVersion{
		Version:     "v1.0",
		Description: "Initial version",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	
	if version.Version != "v1.0" {
		t.Errorf("Expected version 'v1.0', got %s", version.Version)
	}
}

// TestPromptInfo 测试 Prompt 信息。
func TestPromptInfo(t *testing.T) {
	info := PromptInfo{
		Name:        "test-prompt",
		Owner:       "test-owner",
		Description: "Test description",
		Tags:        []string{"test", "example"},
		Stars:       100,
		Downloads:   1000,
	}
	
	if info.Name != "test-prompt" {
		t.Errorf("Expected name 'test-prompt', got %s", info.Name)
	}
	
	if len(info.Tags) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(info.Tags))
	}
}
