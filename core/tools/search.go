package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	
	"langchain-go/pkg/types"
)

// WikipediaSearchTool 是 Wikipedia 搜索工具。
//
// 功能：
//   - 搜索 Wikipedia 文章
//   - 获取摘要信息
//   - 支持多语言
//
type WikipediaSearchTool struct {
	config WikipediaSearchConfig
}

// WikipediaSearchConfig 是 Wikipedia 搜索配置。
type WikipediaSearchConfig struct {
	// Language Wikipedia 语言版本（如 "en", "zh" 等）
	Language string
	
	// MaxResults 最大返回结果数
	MaxResults int
	
	// SentenceCount 摘要句子数
	SentenceCount int
	
	// LoadFullArticle 是否加载完整文章（默认只加载摘要）
	LoadFullArticle bool
}

// DefaultWikipediaSearchConfig 返回默认配置。
func DefaultWikipediaSearchConfig() WikipediaSearchConfig {
	return WikipediaSearchConfig{
		Language:        "en",
		MaxResults:      3,
		SentenceCount:   5,
		LoadFullArticle: false,
	}
}

// NewWikipediaSearch 创建 Wikipedia 搜索工具。
//
// 参数：
//   - config: 配置（可选，使用默认配置传 nil）
//
// 返回：
//   - *WikipediaSearchTool: 工具实例
//
// 示例：
//
//	tool := tools.NewWikipediaSearch(nil) // 使用默认配置
//	tool := tools.NewWikipediaSearch(&tools.WikipediaSearchConfig{
//	    Language: "zh",
//	    MaxResults: 5,
//	})
//
func NewWikipediaSearch(config *WikipediaSearchConfig) *WikipediaSearchTool {
	var cfg WikipediaSearchConfig
	if config != nil {
		cfg = *config
	} else {
		cfg = DefaultWikipediaSearchConfig()
	}
	
	return &WikipediaSearchTool{
		config: cfg,
	}
}

// GetName 返回工具名称。
func (w *WikipediaSearchTool) GetName() string {
	return "wikipedia_search"
}

// GetDescription 返回工具描述。
func (w *WikipediaSearchTool) GetDescription() string {
	return "Search Wikipedia for information on a topic. Returns a summary of the most relevant articles."
}

// GetParameters 返回工具参数。
func (w *WikipediaSearchTool) GetParameters() []ToolParameter {
	return []ToolParameter{
		{
			Name:        "query",
			Type:        "string",
			Description: "The search query",
			Required:    true,
		},
	}
}

// Execute 执行 Wikipedia 搜索。
func (w *WikipediaSearchTool) Execute(ctx context.Context, input map[string]any) (any, error) {
	// 获取查询
	query, ok := input["query"].(string)
	if !ok {
		return nil, fmt.Errorf("wikipedia search: 'query' parameter is required and must be a string")
	}
	
	if query == "" {
		return nil, fmt.Errorf("wikipedia search: query cannot be empty")
	}
	
	// 搜索 Wikipedia
	results, err := w.searchWikipedia(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("wikipedia search failed: %w", err)
	}
	
	if len(results) == 0 {
		return "No results found for: " + query, nil
	}
	
	// 格式化结果
	return w.formatResults(results), nil
}

// searchWikipedia 执行 Wikipedia API 搜索。
func (w *WikipediaSearchTool) searchWikipedia(ctx context.Context, query string) ([]wikipediaResult, error) {
	// 构建 API URL
	baseURL := fmt.Sprintf("https://%s.wikipedia.org/w/api.php", w.config.Language)
	
	params := url.Values{}
	params.Set("action", "query")
	params.Set("format", "json")
	params.Set("list", "search")
	params.Set("srsearch", query)
	params.Set("srlimit", fmt.Sprintf("%d", w.config.MaxResults))
	params.Set("srprop", "snippet")
	
	apiURL := baseURL + "?" + params.Encode()
	
	// 发送请求
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("User-Agent", "LangChain-Go/1.0")
	
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	
	// 解析响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	
	var searchResp wikipediaSearchResponse
	if err := json.Unmarshal(body, &searchResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	
	// 获取每个结果的摘要
	results := make([]wikipediaResult, 0, len(searchResp.Query.Search))
	for _, item := range searchResp.Query.Search {
		summary, err := w.getPageSummary(ctx, item.Title)
		if err != nil {
			// 如果获取摘要失败，使用搜索片段
			results = append(results, wikipediaResult{
				Title:   item.Title,
				Summary: w.cleanHTMLSnippet(item.Snippet),
			})
			continue
		}
		
		results = append(results, wikipediaResult{
			Title:   item.Title,
			Summary: summary,
		})
	}
	
	return results, nil
}

// getPageSummary 获取页面摘要。
func (w *WikipediaSearchTool) getPageSummary(ctx context.Context, title string) (string, error) {
	baseURL := fmt.Sprintf("https://%s.wikipedia.org/w/api.php", w.config.Language)
	
	params := url.Values{}
	params.Set("action", "query")
	params.Set("format", "json")
	params.Set("prop", "extracts")
	params.Set("exintro", "true")
	params.Set("explaintext", "true")
	params.Set("exsentences", fmt.Sprintf("%d", w.config.SentenceCount))
	params.Set("titles", title)
	
	apiURL := baseURL + "?" + params.Encode()
	
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return "", err
	}
	
	req.Header.Set("User-Agent", "LangChain-Go/1.0")
	
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	
	var summaryResp wikipediaSummaryResponse
	if err := json.Unmarshal(body, &summaryResp); err != nil {
		return "", err
	}
	
	// 提取摘要
	for _, page := range summaryResp.Query.Pages {
		if page.Extract != "" {
			return page.Extract, nil
		}
	}
	
	return "", fmt.Errorf("no summary found")
}

// formatResults 格式化搜索结果。
func (w *WikipediaSearchTool) formatResults(results []wikipediaResult) string {
	var builder strings.Builder
	
	builder.WriteString(fmt.Sprintf("Found %d Wikipedia results:\n\n", len(results)))
	
	for i, result := range results {
		builder.WriteString(fmt.Sprintf("%d. %s\n", i+1, result.Title))
		builder.WriteString(fmt.Sprintf("   %s\n\n", result.Summary))
	}
	
	return builder.String()
}

// cleanHTMLSnippet 清理 HTML 片段。
func (w *WikipediaSearchTool) cleanHTMLSnippet(snippet string) string {
	// 移除 HTML 标签
	snippet = strings.ReplaceAll(snippet, "<span class=\"searchmatch\">", "")
	snippet = strings.ReplaceAll(snippet, "</span>", "")
	snippet = strings.ReplaceAll(snippet, "&quot;", "\"")
	snippet = strings.ReplaceAll(snippet, "&#039;", "'")
	
	return snippet
}

// ToTypesTool 转换为 types.Tool。
func (w *WikipediaSearchTool) ToTypesTool() types.Tool {
	return types.Tool{
		Name:        w.GetName(),
		Description: w.GetDescription(),
		Parameters:  convertToolParametersToTypes(w.GetParameters()),
	}
}

// Wikipedia API 响应结构
type wikipediaSearchResponse struct {
	Query struct {
		Search []struct {
			Title   string `json:"title"`
			Snippet string `json:"snippet"`
		} `json:"search"`
	} `json:"query"`
}

type wikipediaSummaryResponse struct {
	Query struct {
		Pages map[string]struct {
			Extract string `json:"extract"`
		} `json:"pages"`
	} `json:"query"`
}

type wikipediaResult struct {
	Title   string
	Summary string
}

// ========================
// Arxiv 搜索工具
// ========================

// ArxivSearchTool 是 Arxiv 论文搜索工具。
//
// 功能：
//   - 搜索 Arxiv 论文
//   - 获取论文摘要和元数据
//
type ArxivSearchTool struct {
	config ArxivSearchConfig
}

// ArxivSearchConfig 是 Arxiv 搜索配置。
type ArxivSearchConfig struct {
	// MaxResults 最大返回结果数
	MaxResults int
	
	// SortBy 排序方式（relevance, lastUpdatedDate, submittedDate）
	SortBy string
	
	// SortOrder 排序顺序（ascending, descending）
	SortOrder string
}

// DefaultArxivSearchConfig 返回默认配置。
func DefaultArxivSearchConfig() ArxivSearchConfig {
	return ArxivSearchConfig{
		MaxResults: 3,
		SortBy:     "relevance",
		SortOrder:  "descending",
	}
}

// NewArxivSearch 创建 Arxiv 搜索工具。
//
// 参数：
//   - config: 配置（可选，使用默认配置传 nil）
//
// 返回：
//   - *ArxivSearchTool: 工具实例
//
// 示例：
//
//	tool := tools.NewArxivSearch(nil) // 使用默认配置
//	tool := tools.NewArxivSearch(&tools.ArxivSearchConfig{
//	    MaxResults: 5,
//	    SortBy: "submittedDate",
//	})
//
func NewArxivSearch(config *ArxivSearchConfig) *ArxivSearchTool {
	var cfg ArxivSearchConfig
	if config != nil {
		cfg = *config
	} else {
		cfg = DefaultArxivSearchConfig()
	}
	
	return &ArxivSearchTool{
		config: cfg,
	}
}

// GetName 返回工具名称。
func (a *ArxivSearchTool) GetName() string {
	return "arxiv_search"
}

// GetDescription 返回工具描述。
func (a *ArxivSearchTool) GetDescription() string {
	return "Search Arxiv for academic papers. Returns paper titles, authors, abstracts, and URLs."
}

// GetParameters 返回工具参数。
func (a *ArxivSearchTool) GetParameters() []ToolParameter {
	return []ToolParameter{
		{
			Name:        "query",
			Type:        "string",
			Description: "The search query (keywords, author name, etc.)",
			Required:    true,
		},
	}
}

// Execute 执行 Arxiv 搜索。
func (a *ArxivSearchTool) Execute(ctx context.Context, input map[string]any) (any, error) {
	// 获取查询
	query, ok := input["query"].(string)
	if !ok {
		return nil, fmt.Errorf("arxiv search: 'query' parameter is required and must be a string")
	}
	
	if query == "" {
		return nil, fmt.Errorf("arxiv search: query cannot be empty")
	}
	
	// 搜索 Arxiv
	results, err := a.searchArxiv(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("arxiv search failed: %w", err)
	}
	
	if len(results) == 0 {
		return "No papers found for: " + query, nil
	}
	
	// 格式化结果
	return a.formatResults(results), nil
}

// searchArxiv 执行 Arxiv API 搜索。
func (a *ArxivSearchTool) searchArxiv(ctx context.Context, query string) ([]arxivResult, error) {
	// 构建 API URL
	baseURL := "http://export.arxiv.org/api/query"
	
	params := url.Values{}
	params.Set("search_query", fmt.Sprintf("all:%s", query))
	params.Set("start", "0")
	params.Set("max_results", fmt.Sprintf("%d", a.config.MaxResults))
	params.Set("sortBy", a.config.SortBy)
	params.Set("sortOrder", a.config.SortOrder)
	
	apiURL := baseURL + "?" + params.Encode()
	
	// 发送请求
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
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
	
	// 解析 XML（Arxiv 返回 XML）
	results, err := parseArxivXML(string(body))
	if err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	
	return results, nil
}

// parseArxivXML 解析 Arxiv XML 响应。
// 注意：这是简化实现，实际使用可能需要使用 encoding/xml 包
func parseArxivXML(xmlData string) ([]arxivResult, error) {
	results := []arxivResult{}
	
	// 简单的字符串解析（实际应使用 XML parser）
	entries := strings.Split(xmlData, "<entry>")
	
	for i, entry := range entries {
		if i == 0 {
			continue // 跳过第一个分割结果
		}
		
		result := arxivResult{}
		
		// 提取标题
		if titleStart := strings.Index(entry, "<title>"); titleStart != -1 {
			titleEnd := strings.Index(entry[titleStart:], "</title>")
			if titleEnd != -1 {
				result.Title = strings.TrimSpace(entry[titleStart+7 : titleStart+titleEnd])
			}
		}
		
		// 提取摘要
		if summaryStart := strings.Index(entry, "<summary>"); summaryStart != -1 {
			summaryEnd := strings.Index(entry[summaryStart:], "</summary>")
			if summaryEnd != -1 {
				result.Abstract = strings.TrimSpace(entry[summaryStart+9 : summaryStart+summaryEnd])
			}
		}
		
		// 提取作者
		authorNames := []string{}
		authorEntries := strings.Split(entry, "<author>")
		for j, authorEntry := range authorEntries {
			if j == 0 {
				continue
			}
			if nameStart := strings.Index(authorEntry, "<name>"); nameStart != -1 {
				nameEnd := strings.Index(authorEntry[nameStart:], "</name>")
				if nameEnd != -1 {
					authorNames = append(authorNames, strings.TrimSpace(authorEntry[nameStart+6:nameStart+nameEnd]))
				}
			}
		}
		result.Authors = strings.Join(authorNames, ", ")
		
		// 提取链接
		if idStart := strings.Index(entry, "<id>"); idStart != -1 {
			idEnd := strings.Index(entry[idStart:], "</id>")
			if idEnd != -1 {
				result.URL = strings.TrimSpace(entry[idStart+4 : idStart+idEnd])
			}
		}
		
		// 提取发布日期
		if publishedStart := strings.Index(entry, "<published>"); publishedStart != -1 {
			publishedEnd := strings.Index(entry[publishedStart:], "</published>")
			if publishedEnd != -1 {
				result.Published = strings.TrimSpace(entry[publishedStart+11 : publishedStart+publishedEnd])
			}
		}
		
		if result.Title != "" {
			results = append(results, result)
		}
	}
	
	return results, nil
}

// formatResults 格式化搜索结果。
func (a *ArxivSearchTool) formatResults(results []arxivResult) string {
	var builder strings.Builder
	
	builder.WriteString(fmt.Sprintf("Found %d Arxiv papers:\n\n", len(results)))
	
	for i, result := range results {
		builder.WriteString(fmt.Sprintf("%d. %s\n", i+1, result.Title))
		builder.WriteString(fmt.Sprintf("   Authors: %s\n", result.Authors))
		builder.WriteString(fmt.Sprintf("   Published: %s\n", result.Published))
		builder.WriteString(fmt.Sprintf("   URL: %s\n", result.URL))
		
		// 截断摘要
		abstract := result.Abstract
		if len(abstract) > 300 {
			abstract = abstract[:300] + "..."
		}
		builder.WriteString(fmt.Sprintf("   Abstract: %s\n\n", abstract))
	}
	
	return builder.String()
}

// ToTypesTool 转换为 types.Tool。
func (a *ArxivSearchTool) ToTypesTool() types.Tool {
	return types.Tool{
		Name:        a.GetName(),
		Description: a.GetDescription(),
		Parameters:  convertToolParametersToTypes(a.GetParameters()),
	}
}

// Arxiv 结果结构
type arxivResult struct {
	Title     string
	Authors   string
	Abstract  string
	URL       string
	Published string
}

// convertToolParametersToTypes 转换工具参数为 types 参数。
func convertToolParametersToTypes(params []ToolParameter) []types.ToolParameter {
	result := make([]types.ToolParameter, len(params))
	for i, p := range params {
		result[i] = types.ToolParameter{
			Name:        p.Name,
			Type:        p.Type,
			Description: p.Description,
			Required:    p.Required,
		}
	}
	return result
}

// ========================
// Tavily 搜索工具
// ========================

// TavilySearchTool 是 Tavily AI 搜索工具。
//
// Tavily 是一个专为 AI Agent 设计的搜索 API，提供高质量的搜索结果。
//
// 功能：
//   - AI 优化的搜索结果
//   - 支持深度搜索
//   - 包含相关性评分
//
// 使用前需要在 https://tavily.com 注册获取 API Key。
//
type TavilySearchTool struct {
	apiKey string
	config TavilySearchConfig
}

// TavilySearchConfig 是 Tavily 搜索配置。
type TavilySearchConfig struct {
	// MaxResults 最大返回结果数 (1-10)
	MaxResults int
	
	// SearchDepth 搜索深度 ("basic" 或 "advanced")
	SearchDepth string
	
	// IncludeAnswer 是否包含 AI 生成的答案
	IncludeAnswer bool
	
	// IncludeRawContent 是否包含原始内容
	IncludeRawContent bool
	
	// IncludeDomains 限制搜索的域名列表
	IncludeDomains []string
	
	// ExcludeDomains 排除的域名列表
	ExcludeDomains []string
}

// DefaultTavilySearchConfig 返回默认配置。
func DefaultTavilySearchConfig() TavilySearchConfig {
	return TavilySearchConfig{
		MaxResults:        5,
		SearchDepth:       "basic",
		IncludeAnswer:     true,
		IncludeRawContent: false,
		IncludeDomains:    []string{},
		ExcludeDomains:    []string{},
	}
}

// NewTavilySearch 创建 Tavily 搜索工具。
//
// 参数：
//   - apiKey: Tavily API Key (在 https://tavily.com 获取)
//   - config: 配置（可选，使用默认配置传 nil）
//
// 返回：
//   - *TavilySearchTool: 工具实例
//
// 示例：
//
//	tool := tools.NewTavilySearch("your-api-key", nil)
//	tool := tools.NewTavilySearch("your-api-key", &tools.TavilySearchConfig{
//	    MaxResults: 10,
//	    SearchDepth: "advanced",
//	})
//
func NewTavilySearch(apiKey string, config *TavilySearchConfig) *TavilySearchTool {
	var cfg TavilySearchConfig
	if config != nil {
		cfg = *config
	} else {
		cfg = DefaultTavilySearchConfig()
	}
	
	return &TavilySearchTool{
		apiKey: apiKey,
		config: cfg,
	}
}

// GetName 返回工具名称。
func (t *TavilySearchTool) GetName() string {
	return "tavily_search"
}

// GetDescription 返回工具描述。
func (t *TavilySearchTool) GetDescription() string {
	return "Search the internet using Tavily AI. Returns high-quality, AI-optimized search results with relevance scores."
}

// GetParameters 返回工具参数。
func (t *TavilySearchTool) GetParameters() []ToolParameter {
	return []ToolParameter{
		{
			Name:        "query",
			Type:        "string",
			Description: "The search query",
			Required:    true,
		},
	}
}

// Execute 执行 Tavily 搜索。
func (t *TavilySearchTool) Execute(ctx context.Context, input map[string]any) (any, error) {
	// 获取查询
	query, ok := input["query"].(string)
	if !ok {
		return nil, fmt.Errorf("tavily search: 'query' parameter is required and must be a string")
	}
	
	if query == "" {
		return nil, fmt.Errorf("tavily search: query cannot be empty")
	}
	
	if t.apiKey == "" {
		return nil, fmt.Errorf("tavily search: API key is required")
	}
	
	// 搜索 Tavily
	results, err := t.searchTavily(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("tavily search failed: %w", err)
	}
	
	// 格式化结果
	return t.formatResults(results), nil
}

// searchTavily 执行 Tavily API 搜索。
func (t *TavilySearchTool) searchTavily(ctx context.Context, query string) (*tavilySearchResponse, error) {
	apiURL := "https://api.tavily.com/search"
	
	// 构建请求体
	requestBody := map[string]any{
		"api_key":             t.apiKey,
		"query":               query,
		"max_results":         t.config.MaxResults,
		"search_depth":        t.config.SearchDepth,
		"include_answer":      t.config.IncludeAnswer,
		"include_raw_content": t.config.IncludeRawContent,
	}
	
	if len(t.config.IncludeDomains) > 0 {
		requestBody["include_domains"] = t.config.IncludeDomains
	}
	
	if len(t.config.ExcludeDomains) > 0 {
		requestBody["exclude_domains"] = t.config.ExcludeDomains
	}
	
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	
	// 发送请求
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, strings.NewReader(string(jsonData)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}
	
	// 解析响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	
	var searchResp tavilySearchResponse
	if err := json.Unmarshal(body, &searchResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	
	return &searchResp, nil
}

// formatResults 格式化搜索结果。
func (t *TavilySearchTool) formatResults(results *tavilySearchResponse) string {
	var builder strings.Builder
	
	// 如果有 AI 答案，先显示
	if results.Answer != "" {
		builder.WriteString("AI Answer:\n")
		builder.WriteString(results.Answer)
		builder.WriteString("\n\n")
	}
	
	builder.WriteString(fmt.Sprintf("Found %d results:\n\n", len(results.Results)))
	
	for i, result := range results.Results {
		builder.WriteString(fmt.Sprintf("%d. %s\n", i+1, result.Title))
		builder.WriteString(fmt.Sprintf("   URL: %s\n", result.URL))
		builder.WriteString(fmt.Sprintf("   Score: %.2f\n", result.Score))
		
		// 截断内容
		content := result.Content
		if len(content) > 300 {
			content = content[:300] + "..."
		}
		builder.WriteString(fmt.Sprintf("   Content: %s\n\n", content))
	}
	
	return builder.String()
}

// ToTypesTool 转换为 types.Tool。
func (t *TavilySearchTool) ToTypesTool() types.Tool {
	return types.Tool{
		Name:        t.GetName(),
		Description: t.GetDescription(),
		Parameters:  convertToolParametersToTypes(t.GetParameters()),
	}
}

// Tavily API 响应结构
type tavilySearchResponse struct {
	Answer  string               `json:"answer"`
	Query   string               `json:"query"`
	Results []tavilySearchResult `json:"results"`
}

type tavilySearchResult struct {
	Title      string  `json:"title"`
	URL        string  `json:"url"`
	Content    string  `json:"content"`
	Score      float64 `json:"score"`
	RawContent string  `json:"raw_content,omitempty"`
}

// ========================
// Google 搜索工具
// ========================

// GoogleSearchTool 是 Google Custom Search API 工具。
//
// 功能：
//   - 使用 Google Custom Search API 进行搜索
//   - 高质量的搜索结果
//   - 支持自定义搜索引擎
//
// 使用前需要：
// 1. 在 https://console.cloud.google.com/ 创建项目并启用 Custom Search API
// 2. 在 https://cse.google.com/cse/ 创建自定义搜索引擎
//
type GoogleSearchTool struct {
	apiKey   string
	engineID string
	config   GoogleSearchConfig
}

// GoogleSearchConfig 是 Google 搜索配置。
type GoogleSearchConfig struct {
	// MaxResults 最大返回结果数 (1-10)
	MaxResults int
	
	// Language 搜索语言
	Language string
	
	// Country 搜索国家
	Country string
	
	// SafeSearch 安全搜索级别 ("off", "medium", "high")
	SafeSearch string
}

// DefaultGoogleSearchConfig 返回默认配置。
func DefaultGoogleSearchConfig() GoogleSearchConfig {
	return GoogleSearchConfig{
		MaxResults: 5,
		Language:   "en",
		Country:    "",
		SafeSearch: "medium",
	}
}

// NewGoogleSearch 创建 Google 搜索工具。
//
// 参数：
//   - apiKey: Google Custom Search API Key
//   - engineID: Custom Search Engine ID
//   - config: 配置（可选，使用默认配置传 nil）
//
// 返回：
//   - *GoogleSearchTool: 工具实例
//
// 示例：
//
//	tool := tools.NewGoogleSearch("your-api-key", "your-engine-id", nil)
//	tool := tools.NewGoogleSearch("your-api-key", "your-engine-id", &tools.GoogleSearchConfig{
//	    MaxResults: 10,
//	    Language: "zh-CN",
//	})
//
func NewGoogleSearch(apiKey, engineID string, config *GoogleSearchConfig) *GoogleSearchTool {
	var cfg GoogleSearchConfig
	if config != nil {
		cfg = *config
	} else {
		cfg = DefaultGoogleSearchConfig()
	}
	
	return &GoogleSearchTool{
		apiKey:   apiKey,
		engineID: engineID,
		config:   cfg,
	}
}

// GetName 返回工具名称。
func (g *GoogleSearchTool) GetName() string {
	return "google_search"
}

// GetDescription 返回工具描述。
func (g *GoogleSearchTool) GetDescription() string {
	return "Search the internet using Google Custom Search API. Returns high-quality search results from Google."
}

// GetParameters 返回工具参数。
func (g *GoogleSearchTool) GetParameters() []ToolParameter {
	return []ToolParameter{
		{
			Name:        "query",
			Type:        "string",
			Description: "The search query",
			Required:    true,
		},
	}
}

// Execute 执行 Google 搜索。
func (g *GoogleSearchTool) Execute(ctx context.Context, input map[string]any) (any, error) {
	// 获取查询
	query, ok := input["query"].(string)
	if !ok {
		return nil, fmt.Errorf("google search: 'query' parameter is required and must be a string")
	}
	
	if query == "" {
		return nil, fmt.Errorf("google search: query cannot be empty")
	}
	
	if g.apiKey == "" {
		return nil, fmt.Errorf("google search: API key is required")
	}
	
	if g.engineID == "" {
		return nil, fmt.Errorf("google search: engine ID is required")
	}
	
	// 搜索 Google
	results, err := g.searchGoogle(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("google search failed: %w", err)
	}
	
	// 格式化结果
	return g.formatResults(results), nil
}

// searchGoogle 执行 Google Custom Search API 搜索。
func (g *GoogleSearchTool) searchGoogle(ctx context.Context, query string) (*googleSearchResponse, error) {
	baseURL := "https://www.googleapis.com/customsearch/v1"
	
	params := url.Values{}
	params.Set("key", g.apiKey)
	params.Set("cx", g.engineID)
	params.Set("q", query)
	params.Set("num", fmt.Sprintf("%d", g.config.MaxResults))
	
	if g.config.Language != "" {
		params.Set("lr", fmt.Sprintf("lang_%s", g.config.Language))
	}
	
	if g.config.Country != "" {
		params.Set("gl", g.config.Country)
	}
	
	if g.config.SafeSearch != "" {
		params.Set("safe", g.config.SafeSearch)
	}
	
	apiURL := baseURL + "?" + params.Encode()
	
	// 发送请求
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}
	
	// 解析响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	
	var searchResp googleSearchResponse
	if err := json.Unmarshal(body, &searchResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	
	return &searchResp, nil
}

// formatResults 格式化搜索结果。
func (g *GoogleSearchTool) formatResults(results *googleSearchResponse) string {
	var builder strings.Builder
	
	builder.WriteString(fmt.Sprintf("Found %d Google results:\n\n", len(results.Items)))
	
	for i, result := range results.Items {
		builder.WriteString(fmt.Sprintf("%d. %s\n", i+1, result.Title))
		builder.WriteString(fmt.Sprintf("   URL: %s\n", result.Link))
		builder.WriteString(fmt.Sprintf("   Snippet: %s\n\n", result.Snippet))
	}
	
	return builder.String()
}

// ToTypesTool 转换为 types.Tool。
func (g *GoogleSearchTool) ToTypesTool() types.Tool {
	return types.Tool{
		Name:        g.GetName(),
		Description: g.GetDescription(),
		Parameters:  convertToolParametersToTypes(g.GetParameters()),
	}
}

// Google API 响应结构
type googleSearchResponse struct {
	Items []googleSearchItem `json:"items"`
}

type googleSearchItem struct {
	Title   string `json:"title"`
	Link    string `json:"link"`
	Snippet string `json:"snippet"`
}
