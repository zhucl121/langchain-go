package search

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"
)

// GoogleProvider 实现 Google 自定义搜索
//
// 需要配置：
// - API Key: 从 Google Cloud Console 获取
// - Search Engine ID (CX): 从 Programmable Search Engine 获取
//
// 官方文档: https://developers.google.com/custom-search/v1/overview
//
type GoogleProvider struct {
	apiKey   string
	engineID string
	client   *http.Client
	baseURL  string
}

// GoogleConfig Google 搜索配置
type GoogleConfig struct {
	// APIKey Google API Key (必需)
	// 可以从环境变量 GOOGLE_API_KEY 读取
	APIKey string
	
	// EngineID 自定义搜索引擎 ID (必需)
	// 可以从环境变量 GOOGLE_SEARCH_ENGINE_ID 读取
	EngineID string
	
	// HTTPClient 自定义 HTTP 客户端（可选）
	HTTPClient *http.Client
	
	// BaseURL 基础 URL（可选，用于测试）
	BaseURL string
}

// googleSearchResponse Google API 响应结构
type googleSearchResponse struct {
	Items []struct {
		Title       string `json:"title"`
		Link        string `json:"link"`
		Snippet     string `json:"snippet"`
		DisplayLink string `json:"displayLink"`
		Pagemap     struct {
			Metatags []map[string]string `json:"metatags"`
		} `json:"pagemap"`
	} `json:"items"`
	SearchInformation struct {
		TotalResults string `json:"totalResults"`
		SearchTime   float64 `json:"searchTime"`
	} `json:"searchInformation"`
	Queries struct {
		Request []struct {
			Count int `json:"count"`
		} `json:"request"`
	} `json:"queries"`
}

// NewGoogleProvider 创建 Google 搜索提供者
func NewGoogleProvider(config GoogleConfig) *GoogleProvider {
	// 尝试从环境变量读取配置
	apiKey := config.APIKey
	if apiKey == "" {
		apiKey = os.Getenv("GOOGLE_API_KEY")
	}
	
	engineID := config.EngineID
	if engineID == "" {
		engineID = os.Getenv("GOOGLE_SEARCH_ENGINE_ID")
	}
	
	client := config.HTTPClient
	if client == nil {
		client = &http.Client{
			Timeout: 30 * time.Second,
		}
	}
	
	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = "https://www.googleapis.com/customsearch/v1"
	}
	
	return &GoogleProvider{
		apiKey:   apiKey,
		engineID: engineID,
		client:   client,
		baseURL:  baseURL,
	}
}

// GetName 实现 SearchProvider 接口
func (p *GoogleProvider) GetName() SearchEngine {
	return EngineGoogle
}

// IsAvailable 实现 SearchProvider 接口
func (p *GoogleProvider) IsAvailable() bool {
	return p.apiKey != "" && p.engineID != ""
}

// Search 实现 SearchProvider 接口
func (p *GoogleProvider) Search(ctx context.Context, query string, options SearchOptions) (*SearchResponse, error) {
	if !p.IsAvailable() {
		return nil, fmt.Errorf("Google search is not available: missing API key or engine ID")
	}
	
	// 创建带超时的上下文
	if options.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, options.Timeout)
		defer cancel()
	}
	
	// 构建请求参数
	params := url.Values{}
	params.Set("key", p.apiKey)
	params.Set("cx", p.engineID)
	params.Set("q", query)
	params.Set("num", fmt.Sprintf("%d", options.MaxResults))
	
	// 语言设置
	if options.Language != "" {
		params.Set("lr", "lang_"+options.Language)
		params.Set("hl", options.Language)
	}
	
	// 地区设置
	if options.Region != "" {
		params.Set("gl", options.Region)
	}
	
	// 安全搜索
	switch options.SafeSearch {
	case "strict":
		params.Set("safe", "active")
	case "off":
		params.Set("safe", "off")
	// moderate 是默认值
	}
	
	// 自定义参数
	for k, v := range options.CustomParams {
		params.Set(k, v)
	}
	
	reqURL := p.baseURL + "?" + params.Encode()
	
	// 创建请求
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	// 执行请求
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	
	// 检查状态码
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}
	
	// 解析响应
	var apiResp googleSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	
	// 转换为标准格式
	results := make([]SearchResult, 0, len(apiResp.Items))
	for _, item := range apiResp.Items {
		result := SearchResult{
			Title:   item.Title,
			Link:    item.Link,
			Snippet: item.Snippet,
			Source:  item.DisplayLink,
			Metadata: map[string]any{
				"pagemap": item.Pagemap,
			},
		}
		
		// 尝试提取发布日期
		if len(item.Pagemap.Metatags) > 0 {
			metatags := item.Pagemap.Metatags[0]
			if pubDate, ok := metatags["article:published_time"]; ok {
				if t, err := time.Parse(time.RFC3339, pubDate); err == nil {
					result.PublishedDate = &t
				}
			}
		}
		
		results = append(results, result)
	}
	
	// 解析总结果数
	totalResults := 0
	if apiResp.SearchInformation.TotalResults != "" {
		fmt.Sscanf(apiResp.SearchInformation.TotalResults, "%d", &totalResults)
	}
	
	return &SearchResponse{
		Results:      results,
		Query:        query,
		Engine:       EngineGoogle,
		TotalResults: totalResults,
	}, nil
}

// GetAPIUsage 获取 API 使用信息
// Google Custom Search 有每日配额限制
func (p *GoogleProvider) GetAPIUsage() string {
	if !p.IsAvailable() {
		return "Not configured"
	}
	return "Google Custom Search API - 100 queries/day (free tier)"
}
