package retrievers

import (
	"context"
	"fmt"
	"strings"

	"github.com/zhucl121/langchain-go/core/chat"
	"github.com/zhucl121/langchain-go/retrieval/embeddings"
	"github.com/zhucl121/langchain-go/pkg/types"
)

// HyDERetriever HyDE (Hypothetical Document Embeddings) 检索器
//
// HyDE 是一种创新的检索技术，不直接嵌入查询，而是：
//  1. 使用 LLM 生成假设性的文档（回答查询的文档）
//  2. 嵌入这个假设性文档
//  3. 使用假设文档的嵌入进行相似度搜索
//
// 优势：
//   - 假设文档与真实文档在嵌入空间中更接近
//   - 克服了查询和文档之间的语义鸿沟
//   - 特别适合需要专业知识的领域
//
// 使用示例:
//
//	hydeRetriever := retrievers.NewHyDERetriever(
//	    llm,
//	    embedder,
//	    vectorStore,
//	    retrievers.WithNumHypothetical(2),
//	)
//	docs, _ := hydeRetriever.GetRelevantDocuments(ctx, "What is machine learning?")
//
type HyDERetriever struct {
	llm         chat.ChatModel
	embedder    embeddings.Embeddings
	vectorStore VectorStore
	config      HyDEConfig
}

// HyDEConfig HyDE 检索器配置
type HyDEConfig struct {
	// NumHypothetical 要生成的假设文档数量（默认 1）
	NumHypothetical int
	
	// CustomPrompt 自定义假设文档生成提示词
	CustomPrompt string
	
	// CombineStrategy 多个假设文档的组合策略
	// "average": 平均嵌入（默认）
	// "first": 只使用第一个
	// "separate": 分别搜索后合并
	CombineStrategy string
	
	// TopK 每个查询返回的文档数量
	TopK int
	
	// IncludeQueryEmbedding 是否也包含原始查询的嵌入
	IncludeQueryEmbedding bool
	
	// QueryWeight 原始查询嵌入的权重（如果启用）
	QueryWeight float64
}

// DefaultHyDEConfig 返回默认配置
func DefaultHyDEConfig() HyDEConfig {
	return HyDEConfig{
		NumHypothetical:       1,
		CombineStrategy:       "average",
		TopK:                  4,
		IncludeQueryEmbedding: false,
		QueryWeight:           0.3,
	}
}

// NewHyDERetriever 创建新的 HyDE 检索器
func NewHyDERetriever(llm chat.ChatModel, embedder embeddings.Embeddings, vectorStore VectorStore, opts ...HyDEOption) *HyDERetriever {
	config := DefaultHyDEConfig()
	
	for _, opt := range opts {
		opt(&config)
	}
	
	return &HyDERetriever{
		llm:         llm,
		embedder:    embedder,
		vectorStore: vectorStore,
		config:      config,
	}
}

// GetRelevantDocuments 获取相关文档
func (r *HyDERetriever) GetRelevantDocuments(ctx context.Context, query string) ([]types.Document, error) {
	// 生成假设文档
	hypotheticalDocs, err := r.generateHypotheticalDocuments(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("hyde: failed to generate hypothetical documents: %w", err)
	}
	
	// 根据策略处理
	switch r.config.CombineStrategy {
	case "first":
		return r.searchWithFirst(ctx, query, hypotheticalDocs)
	case "separate":
		return r.searchSeparately(ctx, query, hypotheticalDocs)
	default: // "average"
		return r.searchWithAverage(ctx, query, hypotheticalDocs)
	}
}

// generateHypotheticalDocuments 生成假设文档
func (r *HyDERetriever) generateHypotheticalDocuments(ctx context.Context, query string) ([]string, error) {
	// 构建提示词
	prompt := r.buildPrompt(query)
	
	// 调用 LLM 生成假设文档
	messages := []types.Message{
		types.NewUserMessage(prompt),
	}
	
	response, err := r.llm.Invoke(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("failed to invoke LLM: %w", err)
	}
	
	// 解析响应
	docs := r.parseHypotheticalDocuments(response.Content)
	
	// 如果需要多个但只生成了一个，重复使用
	if len(docs) < r.config.NumHypothetical && len(docs) > 0 {
		for len(docs) < r.config.NumHypothetical {
			docs = append(docs, docs[0])
		}
	}
	
	return docs, nil
}

// buildPrompt 构建假设文档生成提示词
func (r *HyDERetriever) buildPrompt(query string) string {
	if r.config.CustomPrompt != "" {
		return strings.ReplaceAll(r.config.CustomPrompt, "{query}", query)
	}
	
	// 默认提示词
	if r.config.NumHypothetical == 1 {
		return fmt.Sprintf(`Please write a passage to answer the question.

Question: %s

Answer:`, query)
	}
	
	return fmt.Sprintf(`Please write %d passages to answer the question from different perspectives.

Question: %s

Write %d separate passages (separated by "---"):`, 
		r.config.NumHypothetical, query, r.config.NumHypothetical)
}

// parseHypotheticalDocuments 解析假设文档
func (r *HyDERetriever) parseHypotheticalDocuments(response string) []string {
	if r.config.NumHypothetical == 1 {
		return []string{strings.TrimSpace(response)}
	}
	
	// 按分隔符分割
	parts := strings.Split(response, "---")
	docs := make([]string, 0, len(parts))
	
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			docs = append(docs, part)
		}
	}
	
	return docs
}

// searchWithFirst 使用第一个假设文档搜索
func (r *HyDERetriever) searchWithFirst(ctx context.Context, query string, hypotheticalDocs []string) ([]types.Document, error) {
	if len(hypotheticalDocs) == 0 {
		return nil, fmt.Errorf("hyde: no hypothetical documents generated")
	}
	
	// 嵌入第一个假设文档
	embedding, err := r.embedder.EmbedQuery(ctx, hypotheticalDocs[0])
	if err != nil {
		return nil, fmt.Errorf("hyde: failed to embed hypothetical document: %w", err)
	}
	
	// 如果包含查询嵌入，进行加权
	if r.config.IncludeQueryEmbedding {
		queryEmbedding, err := r.embedder.EmbedQuery(ctx, query)
		if err != nil {
			return nil, fmt.Errorf("hyde: failed to embed query: %w", err)
		}
		
		embedding64 := r.weightedAverage(
			[][]float64{float32ToFloat64(embedding), float32ToFloat64(queryEmbedding)}, 
			[]float64{1 - r.config.QueryWeight, r.config.QueryWeight})
		embedding = float64ToFloat32(embedding64)
	}
	
	// 使用嵌入搜索
	docs, err := r.vectorStore.SimilaritySearchByVector(ctx, float32ToFloat64(embedding), r.config.TopK)
	if err != nil {
		return nil, fmt.Errorf("hyde: failed to search: %w", err)
	}
	
	return docs, nil
}

// searchWithAverage 使用平均嵌入搜索
func (r *HyDERetriever) searchWithAverage(ctx context.Context, query string, hypotheticalDocs []string) ([]types.Document, error) {
	if len(hypotheticalDocs) == 0 {
		return nil, fmt.Errorf("hyde: no hypothetical documents generated")
	}
	
	// 嵌入所有假设文档
	embeddings := make([][]float64, 0, len(hypotheticalDocs))
	weights := make([]float64, 0, len(hypotheticalDocs))
	
	for _, doc := range hypotheticalDocs {
		embedding, err := r.embedder.EmbedQuery(ctx, doc)
		if err != nil {
			continue
		}
		embeddings = append(embeddings, float32ToFloat64(embedding))
		weights = append(weights, 1.0)
	}
	
	if len(embeddings) == 0 {
		return nil, fmt.Errorf("hyde: failed to embed any hypothetical document")
	}
	
	// 如果包含查询嵌入
	if r.config.IncludeQueryEmbedding {
		queryEmbedding, err := r.embedder.EmbedQuery(ctx, query)
		if err != nil {
			return nil, fmt.Errorf("hyde: failed to embed query: %w", err)
		}
		
		embeddings = append(embeddings, float32ToFloat64(queryEmbedding))
		weights = append(weights, r.config.QueryWeight*float64(len(hypotheticalDocs)))
	}
	
	// 计算加权平均
	avgEmbedding := r.weightedAverage(embeddings, weights)
	
	// 使用平均嵌入搜索
	docs, err := r.vectorStore.SimilaritySearchByVector(ctx, avgEmbedding, r.config.TopK)
	if err != nil {
		return nil, fmt.Errorf("hyde: failed to search: %w", err)
	}
	
	return docs, nil
}

// searchSeparately 分别搜索后合并
func (r *HyDERetriever) searchSeparately(ctx context.Context, query string, hypotheticalDocs []string) ([]types.Document, error) {
	if len(hypotheticalDocs) == 0 {
		return nil, fmt.Errorf("hyde: no hypothetical documents generated")
	}
	
	allDocs := make([]types.Document, 0)
	seen := make(map[string]bool)
	
	// 对每个假设文档单独搜索
	for _, doc := range hypotheticalDocs {
		embedding, err := r.embedder.EmbedQuery(ctx, doc)
		if err != nil {
			continue
		}
		
		docs, err := r.vectorStore.SimilaritySearchByVector(ctx, float32ToFloat64(embedding), r.config.TopK)
		if err != nil {
			continue
		}
		
		// 合并结果并去重
		for _, d := range docs {
			key := r.getDocumentKey(d)
			if !seen[key] {
				seen[key] = true
				allDocs = append(allDocs, d)
			}
		}
	}
	
	// 限制结果数量
	if len(allDocs) > r.config.TopK {
		allDocs = allDocs[:r.config.TopK]
	}
	
	return allDocs, nil
}

// weightedAverage 计算加权平均嵌入
func (r *HyDERetriever) weightedAverage(embeddings [][]float64, weights []float64) []float64 {
	if len(embeddings) == 0 {
		return nil
	}
	
	// 确保权重数量匹配
	if len(weights) != len(embeddings) {
		// 使用均匀权重
		weights = make([]float64, len(embeddings))
		for i := range weights {
			weights[i] = 1.0
		}
	}
	
	// 归一化权重
	totalWeight := 0.0
	for _, w := range weights {
		totalWeight += w
	}
	
	for i := range weights {
		weights[i] /= totalWeight
	}
	
	// 计算加权平均
	dim := len(embeddings[0])
	result := make([]float64, dim)
	
	for i, emb := range embeddings {
		for j := range emb {
			result[j] += emb[j] * weights[i]
		}
	}
	
	return result
}

// getDocumentKey 获取文档的唯一键
func (r *HyDERetriever) getDocumentKey(doc types.Document) string {
	content := doc.Content
	if len(content) > 100 {
		content = content[:100]
	}
	return content
}

// ==================== 选项模式 ====================

// HyDEOption 配置选项
type HyDEOption func(*HyDEConfig)

// WithNumHypothetical 设置假设文档数量
func WithNumHypothetical(num int) HyDEOption {
	return func(c *HyDEConfig) {
		c.NumHypothetical = num
	}
}

// WithHyDEPrompt 设置自定义提示词
func WithHyDEPrompt(prompt string) HyDEOption {
	return func(c *HyDEConfig) {
		c.CustomPrompt = prompt
	}
}

// WithCombineStrategy 设置组合策略
func WithCombineStrategy(strategy string) HyDEOption {
	return func(c *HyDEConfig) {
		c.CombineStrategy = strategy
	}
}

// WithTopK 设置返回文档数量
func WithTopK(k int) HyDEOption {
	return func(c *HyDEConfig) {
		c.TopK = k
	}
}

// float32ToFloat64 转换 float32 slice 为 float64 slice
func float32ToFloat64(v []float32) []float64 {
	result := make([]float64, len(v))
	for i, val := range v {
		result[i] = float64(val)
	}
	return result
}

// float64ToFloat32 转换 float64 slice 为 float32 slice  
func float64ToFloat32(v []float64) []float32 {
	result := make([]float32, len(v))
	for i, val := range v {
		result[i] = float32(val)
	}
	return result
}

// WithQueryEmbedding 设置是否包含查询嵌入
func WithQueryEmbedding(include bool, weight float64) HyDEOption {
	return func(c *HyDEConfig) {
		c.IncludeQueryEmbedding = include
		c.QueryWeight = weight
	}
}

// ==================== 辅助接口 ====================

// VectorStore 向量存储接口（简化版）
type VectorStore interface {
	SimilaritySearchByVector(ctx context.Context, embedding []float64, k int) ([]types.Document, error)
}
