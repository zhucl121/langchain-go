package search

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// DuckDuckGoProvider 实现 DuckDuckGo 搜索
//
// DuckDuckGo 提供免费的 Instant Answer API，无需 API Key。
// 注意：这是一个简化的实现，使用 HTML 搜索而非官方 API。
//
type DuckDuckGoProvider struct {
	client  *http.Client
	baseURL string
}

// DuckDuckGoConfig 配置
type DuckDuckGoConfig struct {
	// HTTPClient 自定义 HTTP 客户端（可选）
	HTTPClient *http.Client
	
	// BaseURL 基础 URL（可选，用于测试）
	BaseURL string
}

// NewDuckDuckGoProvider 创建 DuckDuckGo 搜索提供者
func NewDuckDuckGoProvider(config DuckDuckGoConfig) *DuckDuckGoProvider {
	client := config.HTTPClient
	if client == nil {
		client = &http.Client{
			Timeout: 30 * time.Second,
		}
	}
	
	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = "https://html.duckduckgo.com/html/"
	}
	
	return &DuckDuckGoProvider{
		client:  client,
		baseURL: baseURL,
	}
}

// GetName 实现 SearchProvider 接口
func (p *DuckDuckGoProvider) GetName() SearchEngine {
	return EngineDuckDuckGo
}

// IsAvailable 实现 SearchProvider 接口
// DuckDuckGo 不需要 API Key，总是可用
func (p *DuckDuckGoProvider) IsAvailable() bool {
	return true
}

// Search 实现 SearchProvider 接口
func (p *DuckDuckGoProvider) Search(ctx context.Context, query string, options SearchOptions) (*SearchResponse, error) {
	// 创建带超时的上下文
	if options.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, options.Timeout)
		defer cancel()
	}
	
	// 构建请求
	params := url.Values{}
	params.Set("q", query)
	params.Set("kl", p.getRegionCode(options.Language, options.Region))
	
	// 安全搜索
	if options.SafeSearch == "strict" {
		params.Set("kp", "1")  // Strict
	} else if options.SafeSearch == "off" {
		params.Set("kp", "-2") // Off
	}
	// moderate 是默认值，不需要设置
	
	reqURL := p.baseURL + "?" + params.Encode()
	
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	// 设置 User-Agent
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; LangChain-Go/1.0)")
	
	// 执行请求
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	
	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	
	// 解析 HTML 结果
	results, err := p.parseHTML(string(body), options.MaxResults)
	if err != nil {
		return nil, fmt.Errorf("failed to parse results: %w", err)
	}
	
	return &SearchResponse{
		Results:      results,
		Query:        query,
		Engine:       EngineDuckDuckGo,
		TotalResults: len(results),
	}, nil
}

// parseHTML 解析 DuckDuckGo HTML 响应
// 注意：这是一个简化的解析器，生产环境建议使用专门的 HTML 解析库
func (p *DuckDuckGoProvider) parseHTML(html string, maxResults int) ([]SearchResult, error) {
	results := []SearchResult{}
	
	// 简化的解析逻辑
	// 在实际应用中，应该使用 golang.org/x/net/html 或类似库
	
	// 查找结果块
	lines := strings.Split(html, "\n")
	currentResult := &SearchResult{}
	inResult := false
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		// 检测结果块开始
		if strings.Contains(line, `class="result__a"`) {
			if inResult && currentResult.Title != "" {
				results = append(results, *currentResult)
				if len(results) >= maxResults {
					break
				}
			}
			currentResult = &SearchResult{}
			inResult = true
			
			// 提取标题和链接
			if titleStart := strings.Index(line, ">"); titleStart != -1 {
				if titleEnd := strings.Index(line[titleStart+1:], "<"); titleEnd != -1 {
					currentResult.Title = p.cleanText(line[titleStart+1 : titleStart+1+titleEnd])
				}
			}
			
			if hrefStart := strings.Index(line, `href="`); hrefStart != -1 {
				hrefStart += 6
				if hrefEnd := strings.Index(line[hrefStart:], `"`); hrefEnd != -1 {
					href := line[hrefStart : hrefStart+hrefEnd]
					// DuckDuckGo 使用重定向，需要解码
					if strings.HasPrefix(href, "//duckduckgo.com/l/?") {
						if u, err := url.Parse("https:" + href); err == nil {
							if uddg := u.Query().Get("uddg"); uddg != "" {
								href = uddg
							}
						}
					}
					currentResult.Link = href
				}
			}
		}
		
		// 提取摘要
		if inResult && strings.Contains(line, `class="result__snippet"`) {
			if snippetStart := strings.Index(line, ">"); snippetStart != -1 {
				if snippetEnd := strings.Index(line[snippetStart+1:], "<"); snippetEnd != -1 {
					currentResult.Snippet = p.cleanText(line[snippetStart+1 : snippetStart+1+snippetEnd])
				}
			}
		}
	}
	
	// 添加最后一个结果
	if inResult && currentResult.Title != "" {
		results = append(results, *currentResult)
	}
	
	// 如果HTML解析失败，返回空结果而不是错误
	// 这样至少工具不会崩溃
	return results, nil
}

// cleanText 清理HTML文本
func (p *DuckDuckGoProvider) cleanText(text string) string {
	// 移除HTML标签
	text = strings.ReplaceAll(text, "<b>", "")
	text = strings.ReplaceAll(text, "</b>", "")
	text = strings.ReplaceAll(text, "<i>", "")
	text = strings.ReplaceAll(text, "</i>", "")
	text = strings.ReplaceAll(text, "<em>", "")
	text = strings.ReplaceAll(text, "</em>", "")
	
	// 解码HTML实体
	text = strings.ReplaceAll(text, "&amp;", "&")
	text = strings.ReplaceAll(text, "&lt;", "<")
	text = strings.ReplaceAll(text, "&gt;", ">")
	text = strings.ReplaceAll(text, "&quot;", "\"")
	text = strings.ReplaceAll(text, "&#39;", "'")
	text = strings.ReplaceAll(text, "&nbsp;", " ")
	
	return strings.TrimSpace(text)
}

// getRegionCode 获取地区代码
func (p *DuckDuckGoProvider) getRegionCode(language, region string) string {
	// DuckDuckGo 使用 kl 参数表示地区
	// 格式: {language}-{region} 例如: en-us, zh-cn
	
	if region != "" {
		region = strings.ToLower(region)
	} else {
		region = "us" // 默认
	}
	
	if language != "" {
		language = strings.ToLower(language)
		// 处理 "zh-CN" 这样的格式
		if strings.Contains(language, "-") {
			parts := strings.Split(language, "-")
			if len(parts) == 2 {
				return parts[0] + "-" + strings.ToLower(parts[1])
			}
		}
		return language + "-" + region
	}
	
	return "en-" + region
}

// duckDuckGoInstantAnswer 使用 DuckDuckGo Instant Answer API
// 这是官方 API，但返回的是结构化数据而非网页搜索结果
type duckDuckGoInstantAnswer struct {
	Abstract       string `json:"Abstract"`
	AbstractText   string `json:"AbstractText"`
	AbstractSource string `json:"AbstractSource"`
	AbstractURL    string `json:"AbstractURL"`
	Heading        string `json:"Heading"`
	RelatedTopics  []struct {
		Text     string `json:"Text"`
		FirstURL string `json:"FirstURL"`
	} `json:"RelatedTopics"`
}

// SearchInstantAnswer 搜索 Instant Answer
// 这是一个补充方法，用于获取快速答案
func (p *DuckDuckGoProvider) SearchInstantAnswer(ctx context.Context, query string) (*duckDuckGoInstantAnswer, error) {
	params := url.Values{}
	params.Set("q", query)
	params.Set("format", "json")
	params.Set("no_html", "1")
	
	reqURL := "https://api.duckduckgo.com/?" + params.Encode()
	
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, err
	}
	
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	var result duckDuckGoInstantAnswer
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	
	return &result, nil
}
