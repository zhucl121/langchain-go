package retrievers

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/zhucl121/langchain-go/core/chat"
	"github.com/zhucl121/langchain-go/pkg/types"
)

// SelfQueryRetriever 自查询检索器
//
// 这个检索器能够从自然语言查询中自动提取：
//  1. 语义查询部分（用于向量搜索）
//  2. 元数据过滤条件（用于结构化过滤）
//
// 工作原理：
//  1. 使用 LLM 解析用户查询
//  2. 提取查询意图和过滤条件
//  3. 构建结构化查询
//  4. 执行向量搜索 + 元数据过滤
//
// 使用示例:
//
//	selfQueryRetriever := retrievers.NewSelfQueryRetriever(
//	    llm,
//	    vectorStore,
//	    documentContents,
//	    metadataFields,
//	)
//	docs, _ := selfQueryRetriever.GetRelevantDocuments(ctx, 
//	    "Show me sci-fi movies from 2020")
//
type SelfQueryRetriever struct {
	llm              chat.ChatModel
	vectorStore      VectorStoreWithFilter
	documentContents string
	metadataFields   []MetadataField
	config           SelfQueryConfig
}

// SelfQueryConfig 自查询检索器配置
type SelfQueryConfig struct {
	// TopK 返回的文档数量
	TopK int
	
	// CustomPrompt 自定义查询解析提示词
	CustomPrompt string
	
	// AllowEmptyQuery 是否允许空查询（仅过滤）
	AllowEmptyQuery bool
	
	// AllowEmptyFilter 是否允许空过滤（仅查询）
	AllowEmptyFilter bool
}

// MetadataField 元数据字段描述
type MetadataField struct {
	// Name 字段名称
	Name string
	
	// Type 字段类型 (string, number, boolean, date)
	Type string
	
	// Description 字段描述
	Description string
	
	// AllowedValues 允许的值（可选）
	AllowedValues []string
}

// StructuredQuery 结构化查询
type StructuredQuery struct {
	// Query 语义查询文本
	Query string `json:"query"`
	
	// Filter 过滤条件
	Filter map[string]interface{} `json:"filter"`
}

// DefaultSelfQueryConfig 返回默认配置
func DefaultSelfQueryConfig() SelfQueryConfig {
	return SelfQueryConfig{
		TopK:             4,
		AllowEmptyQuery:  true,
		AllowEmptyFilter: true,
	}
}

// NewSelfQueryRetriever 创建新的自查询检索器
//
// 参数：
//   - llm: 语言模型（用于解析查询）
//   - vectorStore: 支持过滤的向量存储
//   - documentContents: 文档内容描述
//   - metadataFields: 元数据字段定义
func NewSelfQueryRetriever(
	llm chat.ChatModel,
	vectorStore VectorStoreWithFilter,
	documentContents string,
	metadataFields []MetadataField,
	opts ...SelfQueryOption,
) *SelfQueryRetriever {
	config := DefaultSelfQueryConfig()
	
	for _, opt := range opts {
		opt(&config)
	}
	
	return &SelfQueryRetriever{
		llm:              llm,
		vectorStore:      vectorStore,
		documentContents: documentContents,
		metadataFields:   metadataFields,
		config:           config,
	}
}

// GetRelevantDocuments 获取相关文档
func (r *SelfQueryRetriever) GetRelevantDocuments(ctx context.Context, query string) ([]types.Document, error) {
	// 解析查询
	structuredQuery, err := r.parseQuery(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("self-query: failed to parse query: %w", err)
	}
	
	// 验证查询
	if !r.config.AllowEmptyQuery && structuredQuery.Query == "" {
		return nil, fmt.Errorf("self-query: empty query not allowed")
	}
	
	if !r.config.AllowEmptyFilter && len(structuredQuery.Filter) == 0 {
		return nil, fmt.Errorf("self-query: empty filter not allowed")
	}
	
	// 执行搜索
	docs, err := r.vectorStore.SimilaritySearchWithFilter(
		ctx,
		structuredQuery.Query,
		r.config.TopK,
		structuredQuery.Filter,
	)
	if err != nil {
		return nil, fmt.Errorf("self-query: search failed: %w", err)
	}
	
	return docs, nil
}

// parseQuery 解析用户查询为结构化查询
func (r *SelfQueryRetriever) parseQuery(ctx context.Context, query string) (*StructuredQuery, error) {
	// 构建提示词
	prompt := r.buildPrompt(query)
	
	// 调用 LLM
	messages := []types.Message{
		types.NewUserMessage(prompt),
	}
	
	response, err := r.llm.Invoke(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("failed to invoke LLM: %w", err)
	}
	
	// 解析响应
	structuredQuery, err := r.parseResponse(response.Content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	
	return structuredQuery, nil
}

// buildPrompt 构建查询解析提示词
func (r *SelfQueryRetriever) buildPrompt(query string) string {
	if r.config.CustomPrompt != "" {
		return strings.ReplaceAll(r.config.CustomPrompt, "{query}", query)
	}
	
	// 构建元数据字段说明
	var fieldsDesc strings.Builder
	for _, field := range r.metadataFields {
		fieldsDesc.WriteString(fmt.Sprintf("\n- %s (%s): %s", 
			field.Name, field.Type, field.Description))
		
		if len(field.AllowedValues) > 0 {
			fieldsDesc.WriteString(fmt.Sprintf("\n  Allowed values: %v", field.AllowedValues))
		}
	}
	
	// 默认提示词
	return fmt.Sprintf(`You are a query parser that extracts structured information from natural language queries.

Document Contents: %s

Available Metadata Fields:%s

User Query: %s

Please analyze the query and extract:
1. The semantic query (what to search for)
2. Any metadata filters (field conditions)

Respond with a JSON object in this exact format:
{
  "query": "semantic search query",
  "filter": {
    "field_name": "value",
    "another_field": 123
  }
}

If there's no semantic query, use an empty string for "query".
If there are no filters, use an empty object for "filter".

JSON Response:`,
		r.documentContents,
		fieldsDesc.String(),
		query)
}

// parseResponse 解析 LLM 响应
func (r *SelfQueryRetriever) parseResponse(response string) (*StructuredQuery, error) {
	// 提取 JSON 部分
	jsonStr := r.extractJSON(response)
	
	// 解析 JSON
	var structuredQuery StructuredQuery
	if err := json.Unmarshal([]byte(jsonStr), &structuredQuery); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}
	
	return &structuredQuery, nil
}

// extractJSON 从响应中提取 JSON
func (r *SelfQueryRetriever) extractJSON(response string) string {
	// 尝试找到 JSON 对象
	start := strings.Index(response, "{")
	if start == -1 {
		return "{}"
	}
	
	// 查找匹配的结束括号
	count := 0
	for i := start; i < len(response); i++ {
		if response[i] == '{' {
			count++
		} else if response[i] == '}' {
			count--
			if count == 0 {
				return response[start : i+1]
			}
		}
	}
	
	return "{}"
}

// ==================== 辅助接口 ====================

// VectorStoreWithFilter 支持过滤的向量存储接口
type VectorStoreWithFilter interface {
	// SimilaritySearchWithFilter 带过滤条件的相似度搜索
	SimilaritySearchWithFilter(
		ctx context.Context,
		query string,
		k int,
		filter map[string]interface{},
	) ([]types.Document, error)
}

// ==================== 选项模式 ====================

// SelfQueryOption 配置选项
type SelfQueryOption func(*SelfQueryConfig)

// WithSelfQueryTopK 设置返回文档数量
func WithSelfQueryTopK(k int) SelfQueryOption {
	return func(c *SelfQueryConfig) {
		c.TopK = k
	}
}

// WithSelfQueryPrompt 设置自定义提示词
func WithSelfQueryPrompt(prompt string) SelfQueryOption {
	return func(c *SelfQueryConfig) {
		c.CustomPrompt = prompt
	}
}

// WithAllowEmptyQuery 设置是否允许空查询
func WithAllowEmptyQuery(allow bool) SelfQueryOption {
	return func(c *SelfQueryConfig) {
		c.AllowEmptyQuery = allow
	}
}

// WithAllowEmptyFilter 设置是否允许空过滤
func WithAllowEmptyFilter(allow bool) SelfQueryOption {
	return func(c *SelfQueryConfig) {
		c.AllowEmptyFilter = allow
	}
}

// ==================== 辅助函数 ====================

// NewMetadataField 创建元数据字段定义
func NewMetadataField(name, fieldType, description string, allowedValues ...string) MetadataField {
	return MetadataField{
		Name:          name,
		Type:          fieldType,
		Description:   description,
		AllowedValues: allowedValues,
	}
}
