package search

import (
	"context"
	"fmt"
	"time"
	
	"github.com/zhucl121/langchain-go/pkg/types"
)

// SearchEngine 定义搜索引擎类型
type SearchEngine string

const (
	// EngineGoogle Google 搜索
	EngineGoogle SearchEngine = "google"
	
	// EngineBing Bing 搜索
	EngineBing SearchEngine = "bing"
	
	// EngineDuckDuckGo DuckDuckGo 搜索
	EngineDuckDuckGo SearchEngine = "duckduckgo"
)

// SearchResult 表示单个搜索结果
type SearchResult struct {
	// Title 结果标题
	Title string
	
	// Link 结果链接
	Link string
	
	// Snippet 结果摘要
	Snippet string
	
	// Source 来源（可选）
	Source string
	
	// PublishedDate 发布日期（可选）
	PublishedDate *time.Time
	
	// Metadata 额外元数据
	Metadata map[string]any
}

// SearchResponse 表示搜索响应
type SearchResponse struct {
	// Results 搜索结果列表
	Results []SearchResult
	
	// Query 查询字符串
	Query string
	
	// Engine 使用的搜索引擎
	Engine SearchEngine
	
	// TotalResults 总结果数（如果可用）
	TotalResults int
	
	// SearchTime 搜索耗时
	SearchTime time.Duration
}

// SearchOptions 搜索选项
type SearchOptions struct {
	// MaxResults 最大结果数（默认: 5）
	MaxResults int
	
	// Language 语言代码（如: "en", "zh-CN"）
	Language string
	
	// Region 地区代码（如: "US", "CN"）
	Region string
	
	// SafeSearch 安全搜索级别（默认: "moderate"）
	// 可选值: "off", "moderate", "strict"
	SafeSearch string
	
	// Timeout 超时时间（默认: 30s）
	Timeout time.Duration
	
	// CustomParams 自定义参数
	CustomParams map[string]string
}

// DefaultSearchOptions 返回默认搜索选项
func DefaultSearchOptions() SearchOptions {
	return SearchOptions{
		MaxResults: 5,
		Language:   "en",
		SafeSearch: "moderate",
		Timeout:    30 * time.Second,
	}
}

// Validate 验证搜索选项
func (opts *SearchOptions) Validate() error {
	if opts.MaxResults <= 0 {
		return fmt.Errorf("MaxResults must be positive")
	}
	
	if opts.MaxResults > 100 {
		return fmt.Errorf("MaxResults cannot exceed 100")
	}
	
	if opts.Timeout <= 0 {
		opts.Timeout = 30 * time.Second
	}
	
	validSafeLevels := map[string]bool{
		"off":      true,
		"moderate": true,
		"strict":   true,
	}
	
	if opts.SafeSearch != "" && !validSafeLevels[opts.SafeSearch] {
		return fmt.Errorf("invalid SafeSearch level: %s", opts.SafeSearch)
	}
	
	return nil
}

// SearchProvider 定义搜索提供者接口
type SearchProvider interface {
	// Search 执行搜索
	Search(ctx context.Context, query string, options SearchOptions) (*SearchResponse, error)
	
	// GetName 获取提供者名称
	GetName() SearchEngine
	
	// IsAvailable 检查是否可用（如 API Key 是否配置）
	IsAvailable() bool
}

// SearchTool 是搜索工具的基础结构
type SearchTool struct {
	provider SearchProvider
	options  SearchOptions
}

// NewSearchTool 创建搜索工具
func NewSearchTool(provider SearchProvider, options SearchOptions) (*SearchTool, error) {
	if provider == nil {
		return nil, fmt.Errorf("provider cannot be nil")
	}
	
	if !provider.IsAvailable() {
		return nil, fmt.Errorf("search provider %s is not available (missing API key?)", provider.GetName())
	}
	
	if err := options.Validate(); err != nil {
		return nil, fmt.Errorf("invalid options: %w", err)
	}
	
	return &SearchTool{
		provider: provider,
		options:  options,
	}, nil
}

// GetName 实现 Tool 接口
func (st *SearchTool) GetName() string {
	return fmt.Sprintf("%s_search", st.provider.GetName())
}

// GetDescription 实现 Tool 接口
func (st *SearchTool) GetDescription() string {
	return fmt.Sprintf("Search the internet using %s. Returns a list of relevant web pages with titles, links, and snippets.", st.provider.GetName())
}

// GetParameters 实现 Tool 接口
func (st *SearchTool) GetParameters() types.Schema {
	return types.Schema{
		Type: "object",
		Properties: map[string]types.Schema{
			"query": {
				Type:        "string",
				Description: "The search query",
			},
			"max_results": {
				Type:        "integer",
				Description: fmt.Sprintf("Maximum number of results to return (default: %d, max: 100)", st.options.MaxResults),
			},
		},
		Required: []string{"query"},
	}
}

// Execute 实现 Tool 接口
func (st *SearchTool) Execute(ctx context.Context, args map[string]any) (any, error) {
	// 提取查询参数
	query, ok := args["query"].(string)
	if !ok {
		return nil, fmt.Errorf("query must be a string")
	}
	
	if query == "" {
		return nil, fmt.Errorf("query cannot be empty")
	}
	
	// 创建选项副本
	options := st.options
	
	// 允许覆盖 max_results
	if maxResults, ok := args["max_results"].(float64); ok {
		options.MaxResults = int(maxResults)
	} else if maxResults, ok := args["max_results"].(int); ok {
		options.MaxResults = maxResults
	}
	
	// 验证选项
	if err := options.Validate(); err != nil {
		return nil, fmt.Errorf("invalid options: %w", err)
	}
	
	// 执行搜索
	startTime := time.Now()
	response, err := st.provider.Search(ctx, query, options)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}
	
	response.SearchTime = time.Since(startTime)
	
	// 格式化结果为字符串
	return st.formatResults(response), nil
}

// formatResults 格式化搜索结果
func (st *SearchTool) formatResults(response *SearchResponse) string {
	if len(response.Results) == 0 {
		return fmt.Sprintf("No results found for query: %s", response.Query)
	}
	
	result := fmt.Sprintf("Search Results for '%s' (found %d results):\n\n", 
		response.Query, len(response.Results))
	
	for i, r := range response.Results {
		result += fmt.Sprintf("%d. %s\n", i+1, r.Title)
		result += fmt.Sprintf("   Link: %s\n", r.Link)
		result += fmt.Sprintf("   Snippet: %s\n", r.Snippet)
		
		if r.Source != "" {
			result += fmt.Sprintf("   Source: %s\n", r.Source)
		}
		
		result += "\n"
	}
	
	return result
}

// ToTypesTool 实现 Tool 接口
func (st *SearchTool) ToTypesTool() types.Tool {
	return types.Tool{
		Name:        st.GetName(),
		Description: st.GetDescription(),
		Parameters:  st.GetParameters(),
	}
}

// GetProvider 获取搜索提供者
func (st *SearchTool) GetProvider() SearchProvider {
	return st.provider
}

// UpdateOptions 更新搜索选项
func (st *SearchTool) UpdateOptions(options SearchOptions) error {
	if err := options.Validate(); err != nil {
		return err
	}
	st.options = options
	return nil
}
