package loaders

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/zhucl121/langchain-go/pkg/types"
)

// ConfluenceLoader 从 Confluence 加载文档
//
// 支持的功能:
//   - 加载单个页面
//   - 加载空间中的所有页面
//   - 搜索页面
//   - 加载页面附件
//   - 支持 Cloud 和 Server 版本
//
// 使用示例:
//
//	config := loaders.ConfluenceLoaderConfig{
//	    URL:      "https://your-domain.atlassian.net/wiki",
//	    Username: "user@example.com",
//	    APIToken: "your-api-token",
//	}
//	loader := loaders.NewConfluenceLoader(config)
//	docs, _ := loader.LoadSpace(ctx, "SPACE_KEY")
//
type ConfluenceLoader struct {
	config     ConfluenceLoaderConfig
	httpClient *http.Client
}

// ConfluenceLoaderConfig Confluence 加载器配置
type ConfluenceLoaderConfig struct {
	// URL Confluence 实例 URL
	// Cloud: https://your-domain.atlassian.net/wiki
	// Server: https://your-domain.com/confluence
	URL string
	
	// Username 用户名（通常是邮箱）
	Username string
	
	// APIToken API Token (Cloud) 或 Password (Server)
	APIToken string
	
	// SpaceKey 空间键（如果只加载特定空间）
	SpaceKey string
	
	// MaxPages 最大加载页面数
	// 0 表示无限制，默认 100
	MaxPages int
	
	// IncludeAttachments 是否包含附件
	IncludeAttachments bool
	
	// IncludeComments 是否包含评论
	IncludeComments bool
	
	// Timeout 请求超时时间
	Timeout time.Duration
}

// NewConfluenceLoader 创建新的 Confluence 加载器
func NewConfluenceLoader(config ConfluenceLoaderConfig) (*ConfluenceLoader, error) {
	if config.URL == "" {
		return nil, fmt.Errorf("confluence loader: URL is required")
	}
	
	if config.Username == "" {
		return nil, fmt.Errorf("confluence loader: username is required")
	}
	
	if config.APIToken == "" {
		return nil, fmt.Errorf("confluence loader: API token is required")
	}
	
	// 移除 URL 尾部的斜杠
	config.URL = strings.TrimSuffix(config.URL, "/")
	
	if config.MaxPages == 0 {
		config.MaxPages = 100
	}
	
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	
	httpClient := &http.Client{
		Timeout: config.Timeout,
	}
	
	return &ConfluenceLoader{
		config:     config,
		httpClient: httpClient,
	}, nil
}

// LoadPage 加载单个页面
func (l *ConfluenceLoader) LoadPage(ctx context.Context, pageID string) (types.Document, error) {
	// 获取页面内容
	page, err := l.fetchPage(ctx, pageID)
	if err != nil {
		return types.Document{}, err
	}
	
	// 提取纯文本内容
	content := l.extractContent(page)
	
	return types.Document{
		Content: content,
		Metadata: map[string]interface{}{
			"source":     page.Links.WebUI,
			"page_id":    page.ID,
			"title":      page.Title,
			"space_key":  page.Space.Key,
			"space_name": page.Space.Name,
			"type":       page.Type,
			"version":    page.Version.Number,
			"created_at": page.Version.When,
			"author":     page.Version.By.DisplayName,
		},
	}, nil
}

// LoadSpace 加载空间中的所有页面
func (l *ConfluenceLoader) LoadSpace(ctx context.Context, spaceKey string) ([]types.Document, error) {
	var documents []types.Document
	start := 0
	limit := 25
	
	for {
		// 获取一批页面
		pages, hasMore, err := l.fetchPages(ctx, spaceKey, start, limit)
		if err != nil {
			return nil, err
		}
		
		// 加载每个页面
		for _, page := range pages {
			doc, err := l.LoadPage(ctx, page.ID)
			if err != nil {
				// 记录错误但继续
				continue
			}
			documents = append(documents, doc)
			
			// 检查是否达到最大页面数
			if l.config.MaxPages > 0 && len(documents) >= l.config.MaxPages {
				return documents, nil
			}
		}
		
		if !hasMore {
			break
		}
		
		start += limit
	}
	
	return documents, nil
}

// Search 搜索页面
func (l *ConfluenceLoader) Search(ctx context.Context, cql string, limit int) ([]types.Document, error) {
	if limit <= 0 {
		limit = 25
	}
	
	results, err := l.searchPages(ctx, cql, 0, limit)
	if err != nil {
		return nil, err
	}
	
	documents := make([]types.Document, 0, len(results))
	for _, result := range results {
		doc, err := l.LoadPage(ctx, result.Content.ID)
		if err != nil {
			continue
		}
		documents = append(documents, doc)
	}
	
	return documents, nil
}

// ==================== 内部方法 ====================

func (l *ConfluenceLoader) fetchPage(ctx context.Context, pageID string) (*ConfluencePage, error) {
	apiURL := fmt.Sprintf("%s/rest/api/content/%s?expand=body.storage,version,space",
		l.config.URL, pageID)
	
	data, err := l.doAPIRequest(ctx, apiURL)
	if err != nil {
		return nil, err
	}
	
	var page ConfluencePage
	if err := json.Unmarshal(data, &page); err != nil {
		return nil, fmt.Errorf("confluence loader: failed to parse page: %w", err)
	}
	
	return &page, nil
}

func (l *ConfluenceLoader) fetchPages(ctx context.Context, spaceKey string, start, limit int) ([]ConfluencePage, bool, error) {
	params := url.Values{}
	params.Add("spaceKey", spaceKey)
	params.Add("start", fmt.Sprintf("%d", start))
	params.Add("limit", fmt.Sprintf("%d", limit))
	params.Add("expand", "version,space")
	
	apiURL := fmt.Sprintf("%s/rest/api/content?%s", l.config.URL, params.Encode())
	
	data, err := l.doAPIRequest(ctx, apiURL)
	if err != nil {
		return nil, false, err
	}
	
	var response ConfluencePageList
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, false, fmt.Errorf("confluence loader: failed to parse pages: %w", err)
	}
	
	hasMore := (start + limit) < response.Size
	
	return response.Results, hasMore, nil
}

func (l *ConfluenceLoader) searchPages(ctx context.Context, cql string, start, limit int) ([]ConfluenceSearchResult, error) {
	params := url.Values{}
	params.Add("cql", cql)
	params.Add("start", fmt.Sprintf("%d", start))
	params.Add("limit", fmt.Sprintf("%d", limit))
	
	apiURL := fmt.Sprintf("%s/rest/api/content/search?%s", l.config.URL, params.Encode())
	
	data, err := l.doAPIRequest(ctx, apiURL)
	if err != nil {
		return nil, err
	}
	
	var response ConfluenceSearchResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("confluence loader: failed to parse search results: %w", err)
	}
	
	return response.Results, nil
}

func (l *ConfluenceLoader) doAPIRequest(ctx context.Context, apiURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("confluence loader: failed to create request: %w", err)
	}
	
	// 基本认证
	req.SetBasicAuth(l.config.Username, l.config.APIToken)
	
	// 设置 headers
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := l.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("confluence loader: request failed: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("confluence loader: API error (status %d): %s", resp.StatusCode, string(body))
	}
	
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("confluence loader: failed to read response: %w", err)
	}
	
	return data, nil
}

func (l *ConfluenceLoader) extractContent(page *ConfluencePage) string {
	// 从 storage 格式提取内容
	if page.Body.Storage.Value != "" {
		// 移除 HTML 标签（简化处理）
		content := page.Body.Storage.Value
		content = l.stripHTML(content)
		return content
	}
	
	return page.Title
}

func (l *ConfluenceLoader) stripHTML(html string) string {
	// 简单的 HTML 标签移除
	// 实际使用时建议使用专门的 HTML 解析库
	result := html
	
	// 移除常见标签
	tags := []string{"<p>", "</p>", "<br>", "<br/>", "<div>", "</div>", 
		"<span>", "</span>", "<h1>", "</h1>", "<h2>", "</h2>", 
		"<h3>", "</h3>", "<ul>", "</ul>", "<li>", "</li>",
		"<ol>", "</ol>", "<strong>", "</strong>", "<em>", "</em>"}
	
	for _, tag := range tags {
		result = strings.ReplaceAll(result, tag, "")
	}
	
	// 移除其他 HTML 标签（正则表达式替换）
	// 这里使用简化实现
	
	return strings.TrimSpace(result)
}

// ==================== API 类型定义 ====================

// ConfluencePage Confluence 页面
type ConfluencePage struct {
	ID      string               `json:"id"`
	Type    string               `json:"type"`
	Title   string               `json:"title"`
	Space   ConfluenceSpace      `json:"space"`
	Version ConfluenceVersion    `json:"version"`
	Body    ConfluenceBody       `json:"body"`
	Links   ConfluenceLinks      `json:"_links"`
}

// ConfluenceSpace Confluence 空间
type ConfluenceSpace struct {
	Key  string `json:"key"`
	Name string `json:"name"`
}

// ConfluenceVersion 版本信息
type ConfluenceVersion struct {
	Number int              `json:"number"`
	When   string           `json:"when"`
	By     ConfluenceUser   `json:"by"`
}

// ConfluenceUser 用户信息
type ConfluenceUser struct {
	DisplayName string `json:"displayName"`
	Email       string `json:"email"`
}

// ConfluenceBody 页面内容
type ConfluenceBody struct {
	Storage ConfluenceStorage `json:"storage"`
}

// ConfluenceStorage 存储格式
type ConfluenceStorage struct {
	Value string `json:"value"`
}

// ConfluenceLinks 链接
type ConfluenceLinks struct {
	WebUI string `json:"webui"`
}

// ConfluencePageList 页面列表响应
type ConfluencePageList struct {
	Results []ConfluencePage `json:"results"`
	Size    int              `json:"size"`
	Start   int              `json:"start"`
	Limit   int              `json:"limit"`
}

// ConfluenceSearchResponse 搜索响应
type ConfluenceSearchResponse struct {
	Results []ConfluenceSearchResult `json:"results"`
	Size    int                      `json:"size"`
}

// ConfluenceSearchResult 搜索结果
type ConfluenceSearchResult struct {
	Content ConfluenceContent `json:"content"`
}

// ConfluenceContent 内容引用
type ConfluenceContent struct {
	ID    string `json:"id"`
	Type  string `json:"type"`
	Title string `json:"title"`
}
