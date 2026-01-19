package retrievers

import (
	"context"
	"fmt"
	"strings"

	"github.com/zhucl121/langchain-go/core/chat"
	"github.com/zhucl121/langchain-go/pkg/types"
)

// MultiQueryRetriever 多查询生成检索器
//
// 为单个用户查询生成多个变体，然后合并所有查询的结果。
// 这种技术可以提高检索的召回率，捕获查询的不同方面。
//
// 工作原理：
//  1. 使用 LLM 为原始查询生成多个变体
//  2. 对每个变体查询独立执行检索
//  3. 合并和去重所有结果
//  4. 按相关性重新排序
//
// 使用示例:
//
//	baseRetriever := retrievers.NewVectorStoreRetriever(vectorStore)
//	multiQueryRetriever := retrievers.NewMultiQueryRetriever(
//	    baseRetriever,
//	    llm,
//	    retrievers.WithNumQueries(3),
//	)
//	docs, _ := multiQueryRetriever.GetRelevantDocuments(ctx, "What is LangChain?")
//
type MultiQueryRetriever struct {
	baseRetriever Retriever
	llm           chat.ChatModel
	config        MultiQueryConfig
}

// MultiQueryConfig 多查询检索器配置
type MultiQueryConfig struct {
	// NumQueries 要生成的查询变体数量（默认 3）
	NumQueries int
	
	// IncludeOriginal 是否包含原始查询（默认 true）
	IncludeOriginal bool
	
	// CustomPrompt 自定义查询生成提示词
	CustomPrompt string
	
	// MergeStrategy 结果合并策略
	// "union": 取并集（默认）
	// "intersection": 取交集
	// "ranked": 按相关性排序
	MergeStrategy string
	
	// MaxResults 最大返回结果数（0 表示不限制）
	MaxResults int
	
	// DeduplicateResults 是否去重（默认 true）
	DeduplicateResults bool
}

// DefaultMultiQueryConfig 返回默认配置
func DefaultMultiQueryConfig() MultiQueryConfig {
	return MultiQueryConfig{
		NumQueries:         3,
		IncludeOriginal:    true,
		MergeStrategy:      "union",
		MaxResults:         0,
		DeduplicateResults: true,
	}
}

// NewMultiQueryRetriever 创建新的多查询检索器
func NewMultiQueryRetriever(baseRetriever Retriever, llm chat.ChatModel, opts ...MultiQueryOption) *MultiQueryRetriever {
	config := DefaultMultiQueryConfig()
	
	for _, opt := range opts {
		opt(&config)
	}
	
	return &MultiQueryRetriever{
		baseRetriever: baseRetriever,
		llm:           llm,
		config:        config,
	}
}

// GetRelevantDocuments 获取相关文档
func (r *MultiQueryRetriever) GetRelevantDocuments(ctx context.Context, query string) ([]types.Document, error) {
	// 生成查询变体
	queries, err := r.generateQueries(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("multi-query retriever: failed to generate queries: %w", err)
	}
	
	// 如果包含原始查询，添加到列表
	if r.config.IncludeOriginal && !r.containsQuery(queries, query) {
		queries = append([]string{query}, queries...)
	}
	
	// 对每个查询执行检索
	allDocs := make([][]types.Document, 0, len(queries))
	for _, q := range queries {
		docs, err := r.baseRetriever.GetRelevantDocuments(ctx, q)
		if err != nil {
			// 记录错误但继续处理其他查询
			continue
		}
		allDocs = append(allDocs, docs)
	}
	
	if len(allDocs) == 0 {
		return nil, fmt.Errorf("multi-query retriever: no results from any query")
	}
	
	// 合并结果
	mergedDocs := r.mergeResults(allDocs)
	
	// 去重
	if r.config.DeduplicateResults {
		mergedDocs = r.deduplicate(mergedDocs)
	}
	
	// 限制结果数量
	if r.config.MaxResults > 0 && len(mergedDocs) > r.config.MaxResults {
		mergedDocs = mergedDocs[:r.config.MaxResults]
	}
	
	return mergedDocs, nil
}

// generateQueries 生成查询变体
func (r *MultiQueryRetriever) generateQueries(ctx context.Context, query string) ([]string, error) {
	// 构建提示词
	prompt := r.buildPrompt(query)
	
	// 调用 LLM 生成查询
	messages := []types.Message{
		types.NewUserMessage(prompt),
	}
	
	response, err := r.llm.Invoke(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("failed to invoke LLM: %w", err)
	}
	
	// 解析响应
	queries := r.parseQueries(response.Content)
	
	return queries, nil
}

// buildPrompt 构建查询生成提示词
func (r *MultiQueryRetriever) buildPrompt(query string) string {
	if r.config.CustomPrompt != "" {
		return strings.ReplaceAll(r.config.CustomPrompt, "{query}", query)
	}
	
	// 默认提示词
	return fmt.Sprintf(`You are an AI assistant helping to generate alternative queries.

Given the original query, generate %d alternative queries that:
1. Capture different aspects or perspectives of the original query
2. Use different wording and phrasing
3. Maintain the same intent and topic

Original Query: %s

Generate %d alternative queries (one per line, without numbering):`, 
		r.config.NumQueries, query, r.config.NumQueries)
}

// parseQueries 解析 LLM 响应中的查询列表
func (r *MultiQueryRetriever) parseQueries(response string) []string {
	lines := strings.Split(response, "\n")
	queries := make([]string, 0, len(lines))
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		// 跳过空行
		if line == "" {
			continue
		}
		
		// 移除编号（如果有）
		line = r.removeNumbering(line)
		
		// 移除引号（如果有）
		line = strings.Trim(line, `"'`)
		
		if line != "" {
			queries = append(queries, line)
		}
		
		// 达到所需数量后停止
		if len(queries) >= r.config.NumQueries {
			break
		}
	}
	
	return queries
}

// removeNumbering 移除行首的编号
func (r *MultiQueryRetriever) removeNumbering(line string) string {
	// 移除 "1. ", "1) ", "- " 等格式
	line = strings.TrimSpace(line)
	
	// 尝试匹配常见的编号格式
	for i := 0; i < 10; i++ {
		prefixes := []string{
			fmt.Sprintf("%d. ", i+1),
			fmt.Sprintf("%d) ", i+1),
			fmt.Sprintf("%d.", i+1),
			"- ",
			"* ",
		}
		
		for _, prefix := range prefixes {
			if strings.HasPrefix(line, prefix) {
				line = strings.TrimPrefix(line, prefix)
				line = strings.TrimSpace(line)
				break
			}
		}
	}
	
	return line
}

// containsQuery 检查查询列表是否包含特定查询
func (r *MultiQueryRetriever) containsQuery(queries []string, query string) bool {
	query = strings.ToLower(strings.TrimSpace(query))
	
	for _, q := range queries {
		if strings.ToLower(strings.TrimSpace(q)) == query {
			return true
		}
	}
	
	return false
}

// mergeResults 合并多个查询的结果
func (r *MultiQueryRetriever) mergeResults(allDocs [][]types.Document) []types.Document {
	switch r.config.MergeStrategy {
	case "intersection":
		return r.mergeIntersection(allDocs)
	case "ranked":
		return r.mergeRanked(allDocs)
	default: // "union"
		return r.mergeUnion(allDocs)
	}
}

// mergeUnion 取所有结果的并集
func (r *MultiQueryRetriever) mergeUnion(allDocs [][]types.Document) []types.Document {
	var result []types.Document
	
	for _, docs := range allDocs {
		result = append(result, docs...)
	}
	
	return result
}

// mergeIntersection 取所有结果的交集
func (r *MultiQueryRetriever) mergeIntersection(allDocs [][]types.Document) []types.Document {
	if len(allDocs) == 0 {
		return nil
	}
	
	if len(allDocs) == 1 {
		return allDocs[0]
	}
	
	// 使用第一个查询的结果作为基础
	result := make([]types.Document, 0)
	
	for _, doc := range allDocs[0] {
		// 检查文档是否在所有其他结果中出现
		inAll := true
		for i := 1; i < len(allDocs); i++ {
			if !r.containsDocument(allDocs[i], doc) {
				inAll = false
				break
			}
		}
		
		if inAll {
			result = append(result, doc)
		}
	}
	
	return result
}

// mergeRanked 按相关性排序合并
func (r *MultiQueryRetriever) mergeRanked(allDocs [][]types.Document) []types.Document {
	// 计算每个文档在多少个查询中出现
	docCount := make(map[string]int)
	docMap := make(map[string]types.Document)
	
	for _, docs := range allDocs {
		for _, doc := range docs {
			key := r.getDocumentKey(doc)
			docCount[key]++
			if _, exists := docMap[key]; !exists {
				docMap[key] = doc
			}
		}
	}
	
	// 按出现次数排序
	type docScore struct {
		doc   types.Document
		score int
	}
	
	scores := make([]docScore, 0, len(docMap))
	for key, doc := range docMap {
		scores = append(scores, docScore{
			doc:   doc,
			score: docCount[key],
		})
	}
	
	// 简单的冒泡排序（按分数降序）
	for i := 0; i < len(scores); i++ {
		for j := i + 1; j < len(scores); j++ {
			if scores[j].score > scores[i].score {
				scores[i], scores[j] = scores[j], scores[i]
			}
		}
	}
	
	// 提取文档
	result := make([]types.Document, len(scores))
	for i, s := range scores {
		result[i] = s.doc
	}
	
	return result
}

// deduplicate 去重文档
func (r *MultiQueryRetriever) deduplicate(docs []types.Document) []types.Document {
	seen := make(map[string]bool)
	result := make([]types.Document, 0, len(docs))
	
	for _, doc := range docs {
		key := r.getDocumentKey(doc)
		if !seen[key] {
			seen[key] = true
			result = append(result, doc)
		}
	}
	
	return result
}

// containsDocument 检查文档列表是否包含特定文档
func (r *MultiQueryRetriever) containsDocument(docs []types.Document, doc types.Document) bool {
	key := r.getDocumentKey(doc)
	
	for _, d := range docs {
		if r.getDocumentKey(d) == key {
			return true
		}
	}
	
	return false
}

// getDocumentKey 获取文档的唯一键
func (r *MultiQueryRetriever) getDocumentKey(doc types.Document) string {
	// 使用内容前100个字符作为键
	content := doc.PageContent
	if len(content) > 100 {
		content = content[:100]
	}
	return content
}

// ==================== 选项模式 ====================

// MultiQueryOption 配置选项
type MultiQueryOption func(*MultiQueryConfig)

// WithNumQueries 设置生成的查询数量
func WithNumQueries(num int) MultiQueryOption {
	return func(c *MultiQueryConfig) {
		c.NumQueries = num
	}
}

// WithIncludeOriginal 设置是否包含原始查询
func WithIncludeOriginal(include bool) MultiQueryOption {
	return func(c *MultiQueryConfig) {
		c.IncludeOriginal = include
	}
}

// WithCustomPrompt 设置自定义提示词
func WithCustomPrompt(prompt string) MultiQueryOption {
	return func(c *MultiQueryConfig) {
		c.CustomPrompt = prompt
	}
}

// WithMergeStrategy 设置合并策略
func WithMergeStrategy(strategy string) MultiQueryOption {
	return func(c *MultiQueryConfig) {
		c.MergeStrategy = strategy
	}
}

// WithMaxResults 设置最大结果数
func WithMaxResults(max int) MultiQueryOption {
	return func(c *MultiQueryConfig) {
		c.MaxResults = max
	}
}

// WithDeduplication 设置是否去重
func WithDeduplication(dedupe bool) MultiQueryOption {
	return func(c *MultiQueryConfig) {
		c.DeduplicateResults = dedupe
	}
}
