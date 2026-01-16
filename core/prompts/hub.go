// Package prompts 提供 Prompt Hub 集成功能。
//
// Prompt Hub 允许从远程仓库拉取、管理和版本控制 prompt 模板。
// 类似于 Python LangChain 的 LangChainHub。
//
package prompts

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// PromptHub 是 Prompt Hub 客户端。
//
// 功能：
//   - 从远程仓库拉取 prompt
//   - 本地缓存
//   - 版本管理
//
type PromptHub struct {
	baseURL string
	cache   *promptCache
	client  *http.Client
}

// PromptHubConfig 是 Prompt Hub 配置。
type PromptHubConfig struct {
	// BaseURL Hub 基础 URL
	BaseURL string
	
	// CacheEnabled 是否启用缓存
	CacheEnabled bool
	
	// CacheTTL 缓存过期时间
	CacheTTL time.Duration
	
	// Timeout HTTP 请求超时
	Timeout time.Duration
}

// DefaultPromptHubConfig 返回默认配置。
func DefaultPromptHubConfig() PromptHubConfig {
	return PromptHubConfig{
		BaseURL:      "https://smith.langchain.com/hub",
		CacheEnabled: true,
		CacheTTL:     24 * time.Hour,
		Timeout:      30 * time.Second,
	}
}

// NewPromptHub 创建 Prompt Hub 客户端。
//
// 参数：
//   - config: 配置（可选，使用默认配置传 nil）
//
// 返回：
//   - *PromptHub: Hub 客户端实例
//
// 示例：
//
//	hub := prompts.NewPromptHub(nil) // 使用默认配置
//	hub := prompts.NewPromptHub(&prompts.PromptHubConfig{
//	    BaseURL: "https://custom-hub.example.com",
//	    CacheTTL: 1 * time.Hour,
//	})
//
func NewPromptHub(config *PromptHubConfig) *PromptHub {
	var cfg PromptHubConfig
	if config != nil {
		cfg = *config
	} else {
		cfg = DefaultPromptHubConfig()
	}
	
	hub := &PromptHub{
		baseURL: cfg.BaseURL,
		client: &http.Client{
			Timeout: cfg.Timeout,
		},
	}
	
	if cfg.CacheEnabled {
		hub.cache = newPromptCache(cfg.CacheTTL)
	}
	
	return hub
}

// PullPrompt 从 Hub 拉取 prompt。
//
// 参数：
//   - ctx: 上下文
//   - name: Prompt 名称（格式: "owner/repo/prompt-name"）
//
// 返回：
//   - *PromptTemplate: Prompt 模板
//   - error: 错误
//
// 示例：
//
//	hub := prompts.NewPromptHub(nil)
//	prompt, err := hub.PullPrompt(ctx, "hwchase17/react")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
func (ph *PromptHub) PullPrompt(ctx context.Context, name string) (*PromptTemplate, error) {
	// 检查缓存
	if ph.cache != nil {
		if cached, ok := ph.cache.get(name); ok {
			return cached, nil
		}
	}
	
	// 从远程拉取
	prompt, err := ph.fetchPrompt(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("failed to pull prompt: %w", err)
	}
	
	// 缓存结果
	if ph.cache != nil {
		ph.cache.set(name, prompt)
	}
	
	return prompt, nil
}

// PullPromptVersion 从 Hub 拉取指定版本的 prompt。
//
// 参数：
//   - ctx: 上下文
//   - name: Prompt 名称
//   - version: 版本号（如 "v1.0", "latest"）
//
// 返回：
//   - *PromptTemplate: Prompt 模板
//   - error: 错误
//
// 示例：
//
//	prompt, err := hub.PullPromptVersion(ctx, "hwchase17/react", "v1.0")
//
func (ph *PromptHub) PullPromptVersion(ctx context.Context, name, version string) (*PromptTemplate, error) {
	fullName := fmt.Sprintf("%s:%s", name, version)
	
	// 检查缓存
	if ph.cache != nil {
		if cached, ok := ph.cache.get(fullName); ok {
			return cached, nil
		}
	}
	
	// 从远程拉取
	prompt, err := ph.fetchPromptVersion(ctx, name, version)
	if err != nil {
		return nil, fmt.Errorf("failed to pull prompt version: %w", err)
	}
	
	// 缓存结果
	if ph.cache != nil {
		ph.cache.set(fullName, prompt)
	}
	
	return prompt, nil
}

// ListVersions 列出 prompt 的所有版本。
//
// 参数：
//   - ctx: 上下文
//   - name: Prompt 名称
//
// 返回：
//   - []PromptVersion: 版本列表
//   - error: 错误
//
func (ph *PromptHub) ListVersions(ctx context.Context, name string) ([]PromptVersion, error) {
	url := fmt.Sprintf("%s/%s/versions", ph.baseURL, name)
	
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("Accept", "application/json")
	
	resp, err := ph.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	
	var versions []PromptVersion
	if err := json.Unmarshal(body, &versions); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	
	return versions, nil
}

// SearchPrompts 搜索 prompts。
//
// 参数：
//   - ctx: 上下文
//   - query: 搜索查询
//
// 返回：
//   - []PromptInfo: Prompt 列表
//   - error: 错误
//
func (ph *PromptHub) SearchPrompts(ctx context.Context, query string) ([]PromptInfo, error) {
	url := fmt.Sprintf("%s/search?q=%s", ph.baseURL, query)
	
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("Accept", "application/json")
	
	resp, err := ph.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	
	var results []PromptInfo
	if err := json.Unmarshal(body, &results); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	
	return results, nil
}

// ClearCache 清除缓存。
func (ph *PromptHub) ClearCache() {
	if ph.cache != nil {
		ph.cache.clear()
	}
}

// fetchPrompt 从远程获取 prompt。
func (ph *PromptHub) fetchPrompt(ctx context.Context, name string) (*PromptTemplate, error) {
	url := fmt.Sprintf("%s/%s", ph.baseURL, name)
	
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("Accept", "application/json")
	
	resp, err := ph.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	
	// 解析响应
	var promptData promptHubResponse
	if err := json.Unmarshal(body, &promptData); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	
	// 转换为 PromptTemplate
	return ph.convertToPromptTemplate(promptData)
}

// fetchPromptVersion 从远程获取指定版本的 prompt。
func (ph *PromptHub) fetchPromptVersion(ctx context.Context, name, version string) (*PromptTemplate, error) {
	url := fmt.Sprintf("%s/%s/%s", ph.baseURL, name, version)
	return ph.fetchPrompt(ctx, url)
}

// convertToPromptTemplate 转换为 PromptTemplate。
func (ph *PromptHub) convertToPromptTemplate(data promptHubResponse) (*PromptTemplate, error) {
	config := PromptTemplateConfig{
		Template:       data.Template,
		InputVariables: data.InputVariables,
	}
	
	return NewPromptTemplate(config)
}

// PromptVersion 是 Prompt 版本信息。
type PromptVersion struct {
	Version     string    `json:"version"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// PromptInfo 是 Prompt 信息。
type PromptInfo struct {
	Name        string   `json:"name"`
	Owner       string   `json:"owner"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
	Stars       int      `json:"stars"`
	Downloads   int      `json:"downloads"`
}

// promptHubResponse 是 Hub API 响应。
type promptHubResponse struct {
	Name           string   `json:"name"`
	Template       string   `json:"template"`
	InputVariables []string `json:"input_variables"`
	Version        string   `json:"version"`
	Description    string   `json:"description"`
}

// ========================
// Prompt 缓存
// ========================

// promptCache 是 Prompt 缓存。
type promptCache struct {
	mu    sync.RWMutex
	items map[string]*cachedPrompt
	ttl   time.Duration
}

// cachedPrompt 是缓存的 Prompt。
type cachedPrompt struct {
	prompt    *PromptTemplate
	expiresAt time.Time
}

// newPromptCache 创建 Prompt 缓存。
func newPromptCache(ttl time.Duration) *promptCache {
	return &promptCache{
		items: make(map[string]*cachedPrompt),
		ttl:   ttl,
	}
}

// get 获取缓存的 Prompt。
func (pc *promptCache) get(key string) (*PromptTemplate, bool) {
	pc.mu.RLock()
	defer pc.mu.RUnlock()
	
	cached, ok := pc.items[key]
	if !ok {
		return nil, false
	}
	
	// 检查是否过期
	if time.Now().After(cached.expiresAt) {
		return nil, false
	}
	
	return cached.prompt, true
}

// set 设置缓存。
func (pc *promptCache) set(key string, prompt *PromptTemplate) {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	
	pc.items[key] = &cachedPrompt{
		prompt:    prompt,
		expiresAt: time.Now().Add(pc.ttl),
	}
}

// clear 清除所有缓存。
func (pc *promptCache) clear() {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	
	pc.items = make(map[string]*cachedPrompt)
}

// ========================
// 便捷函数
// ========================

// defaultHub 是默认的 Hub 实例。
var defaultHub *PromptHub
var hubOnce sync.Once

// getDefaultHub 获取默认 Hub 实例。
func getDefaultHub() *PromptHub {
	hubOnce.Do(func() {
		defaultHub = NewPromptHub(nil)
	})
	return defaultHub
}

// PullPrompt 从默认 Hub 拉取 prompt（便捷函数）。
//
// 参数：
//   - name: Prompt 名称（格式: "owner/repo/prompt-name"）
//
// 返回：
//   - *PromptTemplate: Prompt 模板
//   - error: 错误
//
// 示例：
//
//	prompt, err := prompts.PullPrompt("hwchase17/react")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// 使用 prompt
//	result, _ := prompt.Format(map[string]any{
//	    "input": "What is the weather?",
//	})
//
func PullPrompt(name string) (*PromptTemplate, error) {
	return getDefaultHub().PullPrompt(context.Background(), name)
}

// PullPromptWithContext 从默认 Hub 拉取 prompt（带上下文）。
func PullPromptWithContext(ctx context.Context, name string) (*PromptTemplate, error) {
	return getDefaultHub().PullPrompt(ctx, name)
}

// GetPromptVersions 获取 prompt 的所有版本（便捷函数）。
//
// 参数：
//   - name: Prompt 名称
//
// 返回：
//   - []PromptVersion: 版本列表
//   - error: 错误
//
func GetPromptVersions(name string) ([]PromptVersion, error) {
	return getDefaultHub().ListVersions(context.Background(), name)
}

// ========================
// Prompt 生成器
// ========================

// GeneratePrompt 根据任务和示例生成 prompt。
//
// 这是一个实验性功能，使用 LLM 自动生成适合任务的 prompt。
//
// 参数：
//   - task: 任务描述
//   - examples: 示例列表
//
// 返回：
//   - *PromptTemplate: 生成的 Prompt 模板
//   - error: 错误
//
// 示例：
//
//	prompt, err := prompts.GeneratePrompt(
//	    "Classify movie reviews as positive or negative",
//	    []string{
//	        "Input: This movie was great! Output: positive",
//	        "Input: Terrible film. Output: negative",
//	    },
//	)
//
func GeneratePrompt(task string, examples []string) (*PromptTemplate, error) {
	// 这是一个简化实现
	// 实际使用时应该调用 LLM 来生成更好的 prompt
	
	var templateBuilder strings.Builder
	
	templateBuilder.WriteString(fmt.Sprintf("Task: %s\n\n", task))
	
	if len(examples) > 0 {
		templateBuilder.WriteString("Examples:\n")
		for _, example := range examples {
			templateBuilder.WriteString(fmt.Sprintf("- %s\n", example))
		}
		templateBuilder.WriteString("\n")
	}
	
	templateBuilder.WriteString("Input: {{.input}}\n")
	templateBuilder.WriteString("Output:")
	
	return NewPromptTemplate(PromptTemplateConfig{
		Template:       templateBuilder.String(),
		InputVariables: []string{"input"},
	})
}
