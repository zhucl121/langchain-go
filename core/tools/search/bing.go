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

// BingProvider 实现 Bing 搜索
//
// 需要配置：
// - API Key: 从 Azure Portal 获取（Bing Search v7 API）
//
// 官方文档: https://docs.microsoft.com/en-us/bing/search-apis/
//
type BingProvider struct {
	apiKey  string
	client  *http.Client
	baseURL string
}

// BingConfig Bing 搜索配置
type BingConfig struct {
	// APIKey Bing Search API Key (必需)
	// 可以从环境变量 BING_API_KEY 读取
	APIKey string
	
	// HTTPClient 自定义 HTTP 客户端（可选）
	HTTPClient *http.Client
	
	// BaseURL 基础 URL（可选，用于测试）
	BaseURL string
}

// bingSearchResponse Bing API 响应结构
type bingSearchResponse struct {
	WebPages struct {
		Value []struct {
			Name         string    `json:"name"`
			URL          string    `json:"url"`
			Snippet      string    `json:"snippet"`
			DisplayURL   string    `json:"displayUrl"`
			DatePublished string   `json:"datePublished,omitempty"`
		} `json:"value"`
		TotalEstimatedMatches int `json:"totalEstimatedMatches"`
	} `json:"webPages"`
	RankingResponse struct {
		MainLine struct {
			Items []struct {
				AnswerType string `json:"answerType"`
			} `json:"items"`
		} `json:"mainline"`
	} `json:"rankingResponse"`
}

// NewBingProvider 创建 Bing 搜索提供者
func NewBingProvider(config BingConfig) *BingProvider {
	// 尝试从环境变量读取 API Key
	apiKey := config.APIKey
	if apiKey == "" {
		apiKey = os.Getenv("BING_API_KEY")
	}
	
	client := config.HTTPClient
	if client == nil {
		client = &http.Client{
			Timeout: 30 * time.Second,
		}
	}
	
	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = "https://api.bing.microsoft.com/v7.0/search"
	}
	
	return &BingProvider{
		apiKey:  apiKey,
		client:  client,
		baseURL: baseURL,
	}
}

// GetName 实现 SearchProvider 接口
func (p *BingProvider) GetName() SearchEngine {
	return EngineBing
}

// IsAvailable 实现 SearchProvider 接口
func (p *BingProvider) IsAvailable() bool {
	return p.apiKey != ""
}

// Search 实现 SearchProvider 接口
func (p *BingProvider) Search(ctx context.Context, query string, options SearchOptions) (*SearchResponse, error) {
	if !p.IsAvailable() {
		return nil, fmt.Errorf("Bing search is not available: missing API key")
	}
	
	// 创建带超时的上下文
	if options.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, options.Timeout)
		defer cancel()
	}
	
	// 构建请求参数
	params := url.Values{}
	params.Set("q", query)
	params.Set("count", fmt.Sprintf("%d", options.MaxResults))
	params.Set("responseFilter", "Webpages") // 只返回网页结果
	
	// 语言设置
	// Bing 使用 mkt 参数，格式: {language}-{region}
	if options.Language != "" || options.Region != "" {
		mkt := p.getMarketCode(options.Language, options.Region)
		params.Set("mkt", mkt)
	}
	
	// 安全搜索
	switch options.SafeSearch {
	case "strict":
		params.Set("safeSearch", "Strict")
	case "moderate":
		params.Set("safeSearch", "Moderate")
	case "off":
		params.Set("safeSearch", "Off")
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
	
	// 设置认证头
	req.Header.Set("Ocp-Apim-Subscription-Key", p.apiKey)
	
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
	var apiResp bingSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	
	// 转换为标准格式
	results := make([]SearchResult, 0, len(apiResp.WebPages.Value))
	for _, item := range apiResp.WebPages.Value {
		result := SearchResult{
			Title:   item.Name,
			Link:    item.URL,
			Snippet: item.Snippet,
			Source:  item.DisplayURL,
		}
		
		// 解析发布日期
		if item.DatePublished != "" {
			if t, err := time.Parse(time.RFC3339, item.DatePublished); err == nil {
				result.PublishedDate = &t
			}
		}
		
		results = append(results, result)
	}
	
	return &SearchResponse{
		Results:      results,
		Query:        query,
		Engine:       EngineBing,
		TotalResults: apiResp.WebPages.TotalEstimatedMatches,
	}, nil
}

// getMarketCode 获取市场代码
func (p *BingProvider) getMarketCode(language, region string) string {
	// Bing 使用标准的 language-REGION 格式
	// 例如: en-US, zh-CN, ja-JP
	
	if language == "" {
		language = "en"
	}
	
	if region == "" {
		region = "US"
	}
	
	// 处理特殊情况
	if language == "zh" {
		if region == "CN" || region == "cn" {
			return "zh-CN"
		}
		if region == "TW" || region == "tw" {
			return "zh-TW"
		}
		if region == "HK" || region == "hk" {
			return "zh-HK"
		}
	}
	
	return fmt.Sprintf("%s-%s", language, region)
}

// GetAPIUsage 获取 API 使用信息
func (p *BingProvider) GetAPIUsage() string {
	if !p.IsAvailable() {
		return "Not configured"
	}
	return "Bing Search API v7 - Check your Azure subscription for quota limits"
}

// GetSupportedMarkets 获取支持的市场列表
// Bing 支持多种市场和语言
func (p *BingProvider) GetSupportedMarkets() []string {
	return []string{
		"en-US", "en-GB", "en-CA", "en-AU", "en-IN",
		"zh-CN", "zh-TW", "zh-HK",
		"ja-JP", "ko-KR",
		"fr-FR", "de-DE", "es-ES", "it-IT",
		"pt-BR", "ru-RU", "ar-SA",
		// ... 更多市场
	}
}
