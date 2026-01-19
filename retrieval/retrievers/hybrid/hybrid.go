// Package hybrid 提供混合检索功能，结合向量检索和关键词检索。
//
// HybridRetriever 是核心类型，它整合了：
// - 向量检索器（VectorStore）
// - 关键词检索器（BM25）
// - 融合策略（RRF、Weighted 等）
//
// 使用示例：
//
//	retriever := hybrid.NewHybridRetriever(hybrid.Config{
//	    VectorStore: milvusStore,
//	    Documents: docs,
//	    Strategy: fusion.NewRRFStrategy(60),
//	    VectorWeight: 0.7,
//	    KeywordWeight: 0.3,
//	})
//
//	results, _ := retriever.Search(ctx, "query text", 10)
//
package hybrid

import (
	"context"
	"fmt"

	"github.com/zhucl121/langchain-go/pkg/types"
	"github.com/zhucl121/langchain-go/retrieval/loaders"
	"github.com/zhucl121/langchain-go/retrieval/retrievers/fusion"
	"github.com/zhucl121/langchain-go/retrieval/retrievers/keyword"
	"github.com/zhucl121/langchain-go/retrieval/vectorstores"
)

// Config HybridRetriever 配置
type Config struct {
	// VectorStore 向量存储（必需）
	VectorStore vectorstores.VectorStore

	// Documents 文档列表（用于构建 BM25 索引）
	Documents []types.Document

	// Strategy 融合策略（可选，默认使用 RRF）
	Strategy fusion.FusionStrategy

	// VectorWeight 向量检索权重（用于 Weighted 策略）
	VectorWeight float64

	// KeywordWeight 关键词检索权重（用于 Weighted 策略）
	KeywordWeight float64

	// BM25Config BM25 配置（可选）
	BM25Config keyword.BM25Config

	// VectorTopK 向量检索返回数量（默认与 TopK 相同）
	VectorTopK int

	// KeywordTopK 关键词检索返回数量（默认与 TopK 相同）
	KeywordTopK int

	// MinScore 最小分数阈值（可选，过滤低分结果）
	MinScore float64
}

// HybridRetriever 混合检索器
//
// 同时执行向量检索和关键词检索，然后使用融合策略合并结果。
type HybridRetriever struct {
	vectorStore     vectorstores.VectorStore
	keywordRetriever *keyword.BM25Retriever
	strategy        fusion.FusionStrategy
	config          Config
}

// NewHybridRetriever 创建混合检索器
func NewHybridRetriever(config Config) (*HybridRetriever, error) {
	if config.VectorStore == nil {
		return nil, fmt.Errorf("VectorStore is required")
	}

	if len(config.Documents) == 0 {
		return nil, fmt.Errorf("Documents is required for BM25 indexing")
	}

	// 默认使用 RRF 策略
	if config.Strategy == nil {
		config.Strategy = fusion.NewRRFStrategy(60)
	}

	// 默认权重
	if config.VectorWeight == 0 && config.KeywordWeight == 0 {
		config.VectorWeight = 0.7
		config.KeywordWeight = 0.3
	}

	// 默认 BM25 配置
	if config.BM25Config.Tokenizer == nil {
		config.BM25Config = keyword.DefaultBM25Config()
	}

	// 创建 BM25 检索器
	keywordRetriever := keyword.NewBM25Retriever(config.Documents, config.BM25Config)

	return &HybridRetriever{
		vectorStore:      config.VectorStore,
		keywordRetriever: keywordRetriever,
		strategy:         config.Strategy,
		config:           config,
	}, nil
}

// Search 执行混合检索
//
// 参数：
//   - ctx: 上下文
//   - query: 查询文本
//   - topK: 返回结果数量
//
// 返回：
//   - []SearchResult: 融合并排序后的结果
//   - error: 错误
func (h *HybridRetriever) Search(ctx context.Context, query string, topK int) ([]SearchResult, error) {
	// 确定各检索器的 TopK
	vectorTopK := h.config.VectorTopK
	if vectorTopK == 0 {
		vectorTopK = topK * 2 // 默认取 2 倍，以获得更好的融合效果
	}

	keywordTopK := h.config.KeywordTopK
	if keywordTopK == 0 {
		keywordTopK = topK * 2
	}

	// 并发执行向量检索和关键词检索
	vectorChan := make(chan vectorResult, 1)
	keywordChan := make(chan keywordResult, 1)

	// 向量检索
	go func() {
		results, err := h.vectorStore.SimilaritySearchWithScore(ctx, query, vectorTopK)
		// 转换为文档和分数列表
		docs := make([]types.Document, len(results))
		scores := make([]float64, len(results))
		for i, r := range results {
			docs[i] = convertLoaderDocToTypes(r.Document)
			scores[i] = float64(r.Score)
		}
		vectorChan <- vectorResult{docs: docs, scores: scores, err: err}
	}()

	// 关键词检索
	go func() {
		results, err := h.keywordRetriever.Search(ctx, query, keywordTopK)
		keywordChan <- keywordResult{results: results, err: err}
	}()

	// 等待结果
	vectorRes := <-vectorChan
	keywordRes := <-keywordChan

	// 检查错误
	if vectorRes.err != nil {
		return nil, fmt.Errorf("vector search failed: %w", vectorRes.err)
	}
	if keywordRes.err != nil {
		return nil, fmt.Errorf("keyword search failed: %w", keywordRes.err)
	}

	// 转换为 RankedList
	vectorList := fusion.ConvertToRankedList("vector", vectorRes.docs, vectorRes.scores)
	
	keywordList := h.convertKeywordResults(keywordRes.results)

	// 融合结果
	fusedDocs := h.strategy.Fuse([]fusion.RankedList{vectorList, keywordList})

	// 转换为 SearchResult 并应用过滤
	results := make([]SearchResult, 0, len(fusedDocs))
	for _, fusedDoc := range fusedDocs {
		// 应用最小分数阈值
		if h.config.MinScore > 0 && fusedDoc.Score < h.config.MinScore {
			continue
		}

		results = append(results, SearchResult{
			Document:     fusedDoc.Document,
			Score:        fusedDoc.Score,
			VectorScore:  fusedDoc.SourceScores["vector"],
			KeywordScore: fusedDoc.SourceScores["keyword"],
			VectorRank:   fusedDoc.SourceRanks["vector"],
			KeywordRank:  fusedDoc.SourceRanks["keyword"],
		})

		// 达到 TopK 后停止
		if len(results) >= topK {
			break
		}
	}

	return results, nil
}

// SearchVectorOnly 仅执行向量检索（用于对比测试）
func (h *HybridRetriever) SearchVectorOnly(ctx context.Context, query string, topK int) ([]SearchResult, error) {
	results, err := h.vectorStore.SimilaritySearchWithScore(ctx, query, topK)
	if err != nil {
		return nil, fmt.Errorf("vector search failed: %w", err)
	}

	searchResults := make([]SearchResult, len(results))
	for i, r := range results {
		searchResults[i] = SearchResult{
			Document:    convertLoaderDocToTypes(r.Document),
			Score:       float64(r.Score),
			VectorScore: float64(r.Score),
			VectorRank:  i + 1,
		}
	}

	return searchResults, nil
}

// SearchKeywordOnly 仅执行关键词检索（用于对比测试）
func (h *HybridRetriever) SearchKeywordOnly(ctx context.Context, query string, topK int) ([]SearchResult, error) {
	keywordResults, err := h.keywordRetriever.Search(ctx, query, topK)
	if err != nil {
		return nil, fmt.Errorf("keyword search failed: %w", err)
	}

	results := make([]SearchResult, len(keywordResults))
	for i, kr := range keywordResults {
		results[i] = SearchResult{
			Document:     kr.Document,
			Score:        kr.Score,
			KeywordScore: kr.Score,
			KeywordRank:  i + 1,
		}
	}

	return results, nil
}

// AddDocuments 添加新文档
//
// 同时更新向量存储和 BM25 索引。
func (h *HybridRetriever) AddDocuments(ctx context.Context, documents []types.Document) error {
	// 转换为 loaders.Document
	loaderDocs := make([]*loaders.Document, len(documents))
	for i, doc := range documents {
		loaderDocs[i] = convertTypesToLoaderDoc(doc)
	}

	// 添加到向量存储
	if _, err := h.vectorStore.AddDocuments(ctx, loaderDocs); err != nil {
		return fmt.Errorf("failed to add documents to vector store: %w", err)
	}

	// 添加到关键词索引
	h.keywordRetriever.AddDocuments(documents)

	return nil
}

// GetStats 获取统计信息
func (h *HybridRetriever) GetStats() map[string]any {
	return map[string]any{
		"strategy":      fmt.Sprintf("%T", h.strategy),
		"vector_weight": h.config.VectorWeight,
		"keyword_weight": h.config.KeywordWeight,
		"bm25_stats":    h.keywordRetriever.GetIndexStats(),
		"min_score":     h.config.MinScore,
	}
}

// convertKeywordResults 转换关键词检索结果为 RankedList
func (h *HybridRetriever) convertKeywordResults(results []keyword.ScoredDocument) fusion.RankedList {
	rankedDocs := make([]fusion.RankedDocument, len(results))

	for i, result := range results {
		rankedDocs[i] = fusion.RankedDocument{
			Document: result.Document,
			Score:    result.Score,
			Rank:     i + 1,
		}
	}

	return fusion.RankedList{
		Source:    "keyword",
		Documents: rankedDocs,
	}
}

// SearchResult 混合检索结果
type SearchResult struct {
	Document types.Document
	Score    float64 // 融合后的分数

	// 各来源的原始分数
	VectorScore  float64
	KeywordScore float64

	// 各来源的原始排名
	VectorRank  int
	KeywordRank int
}

// 内部类型，用于并发通信
type vectorResult struct {
	docs   []types.Document
	scores []float64
	err    error
}

type keywordResult struct {
	results []keyword.ScoredDocument
	err     error
}

// 转换函数：loaders.Document -> types.Document
func convertLoaderDocToTypes(doc *loaders.Document) types.Document {
	return types.Document{
		Content:  doc.Content,
		Metadata: doc.Metadata,
	}
}

// 转换函数：types.Document -> loaders.Document
func convertTypesToLoaderDoc(doc types.Document) *loaders.Document {
	return &loaders.Document{
		Content:  doc.Content,
		Metadata: doc.Metadata,
	}
}
