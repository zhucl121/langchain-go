// Package hybrid 提供 Milvus 原生 Hybrid Search 的适配器。
//
// MilvusHybridRetriever 利用 Milvus 2.4+ 的原生混合检索能力，
// 同时兼容我们的统一 HybridRetriever 接口。
//
// 使用示例：
//
//	milvusStore := vectorstores.NewMilvusVectorStore(config, embeddings)
//	retriever := hybrid.NewMilvusHybridRetriever(milvusStore, fusion.NewRRFStrategy(60))
//	results, _ := retriever.Search(ctx, "query", 10)
//
package hybrid

import (
	"context"
	"fmt"

	"github.com/zhucl121/langchain-go/pkg/types"
	"github.com/zhucl121/langchain-go/retrieval/loaders"
	"github.com/zhucl121/langchain-go/retrieval/retrievers/fusion"
	"github.com/zhucl121/langchain-go/retrieval/vectorstores"
)

// MilvusHybridRetriever 利用 Milvus 原生 Hybrid Search 的检索器
//
// 相比通用 HybridRetriever，它直接使用 Milvus 的原生混合检索能力，
// 性能更好，特别是在大规模数据场景下。
type MilvusHybridRetriever struct {
	store    MilvusHybridStore
	strategy fusion.FusionStrategy
	config   MilvusHybridConfig
}

// MilvusHybridStore Milvus 混合检索存储接口
type MilvusHybridStore interface {
	SimilaritySearchWithScore(ctx context.Context, query string, k int) ([]vectorstores.DocumentWithScore, error)
	HybridSearch(ctx context.Context, query string, k int, opts *vectorstores.HybridSearchOptions) ([]vectorstores.HybridSearchResult, error)
	AddDocuments(ctx context.Context, documents []*loaders.Document) ([]string, error)
}

// MilvusHybridConfig Milvus 混合检索配置
type MilvusHybridConfig struct {
	// RRFRankConstant RRF 算法的 k 参数（默认 60）
	RRFRankConstant int

	// UseNativeRRF 是否使用 Milvus 原生 RRF（如果支持）
	UseNativeRRF bool

	// MinScore 最小分数阈值
	MinScore float64

	// VectorTopK 向量检索返回数量（默认与 TopK 相同）
	VectorTopK int

	// EnableFullText 是否启用全文检索（Milvus 2.5+）
	EnableFullText bool
}

// DefaultMilvusHybridConfig 返回默认配置
func DefaultMilvusHybridConfig() MilvusHybridConfig {
	return MilvusHybridConfig{
		RRFRankConstant: 60,
		UseNativeRRF:    true,
		MinScore:        0,
		VectorTopK:      0,
		EnableFullText:  false,
	}
}

// NewMilvusHybridRetriever 创建 Milvus 混合检索器
func NewMilvusHybridRetriever(store MilvusHybridStore, strategy fusion.FusionStrategy) *MilvusHybridRetriever {
	return &MilvusHybridRetriever{
		store:    store,
		strategy: strategy,
		config:   DefaultMilvusHybridConfig(),
	}
}

// NewMilvusHybridRetrieverWithConfig 使用自定义配置创建检索器
func NewMilvusHybridRetrieverWithConfig(store MilvusHybridStore, config MilvusHybridConfig) *MilvusHybridRetriever {
	// 如果没有指定策略，使用 RRF
	var strategy fusion.FusionStrategy
	if config.RRFRankConstant > 0 {
		strategy = fusion.NewRRFStrategy(float64(config.RRFRankConstant))
	} else {
		strategy = fusion.NewRRFStrategy(60)
	}

	return &MilvusHybridRetriever{
		store:    store,
		strategy: strategy,
		config:   config,
	}
}

// Search 执行混合检索
//
// 参数：
//   - ctx: 上下文
//   - query: 查询文本
//   - topK: 返回结果数量
//
// 返回：
//   - []SearchResult: 检索结果
//   - error: 错误
func (m *MilvusHybridRetriever) Search(ctx context.Context, query string, topK int) ([]SearchResult, error) {
	// 确定向量检索的 TopK
	vectorTopK := m.config.VectorTopK
	if vectorTopK == 0 {
		vectorTopK = topK * 2 // 默认取 2 倍用于更好的融合
	}

	// 使用 Milvus 原生 HybridSearch
	opts := &vectorstores.HybridSearchOptions{
		RRFRankConstant: m.config.RRFRankConstant,
	}

	milvusResults, err := m.store.HybridSearch(ctx, query, vectorTopK, opts)
	if err != nil {
		return nil, fmt.Errorf("milvus hybrid search failed: %w", err)
	}

	// 转换为统一的 SearchResult 格式
	results := make([]SearchResult, 0, len(milvusResults))
	for i, mr := range milvusResults {
		// 应用最小分数过滤
		if m.config.MinScore > 0 && float64(mr.FusionScore) < m.config.MinScore {
			continue
		}

		results = append(results, SearchResult{
			Document:     convertLoaderDocToTypes(mr.Document),
			Score:        float64(mr.FusionScore),
			VectorScore:  float64(mr.VectorScore),
			KeywordScore: float64(mr.KeywordScore),
			VectorRank:   i + 1, // Milvus 已经排序好
			KeywordRank:  0,     // Milvus 内部处理
		})

		// 达到 TopK 后停止
		if len(results) >= topK {
			break
		}
	}

	return results, nil
}

// SearchVectorOnly 仅执行向量检索
func (m *MilvusHybridRetriever) SearchVectorOnly(ctx context.Context, query string, topK int) ([]SearchResult, error) {
	milvusResults, err := m.store.SimilaritySearchWithScore(ctx, query, topK)
	if err != nil {
		return nil, fmt.Errorf("vector search failed: %w", err)
	}

	results := make([]SearchResult, len(milvusResults))
	for i, mr := range milvusResults {
		results[i] = SearchResult{
			Document:    convertLoaderDocToTypes(mr.Document),
			Score:       float64(mr.Score),
			VectorScore: float64(mr.Score),
			VectorRank:  i + 1,
		}
	}

	return results, nil
}

// SearchWithCustomStrategy 使用自定义融合策略的检索
//
// 注意：这个方法会先获取向量检索结果，然后在客户端使用自定义策略融合。
// 如果需要最佳性能，应该使用 Search() 方法让 Milvus 服务端处理。
func (m *MilvusHybridRetriever) SearchWithCustomStrategy(ctx context.Context, query string, topK int, strategy fusion.FusionStrategy) ([]SearchResult, error) {
	// 获取向量检索结果
	vectorResults, err := m.store.SimilaritySearchWithScore(ctx, query, topK*2)
	if err != nil {
		return nil, fmt.Errorf("vector search failed: %w", err)
	}

	// 转换为 RankedList
	docs := make([]types.Document, len(vectorResults))
	scores := make([]float64, len(vectorResults))
	for i, vr := range vectorResults {
		docs[i] = convertLoaderDocToTypes(vr.Document)
		scores[i] = float64(vr.Score)
	}

	vectorList := fusion.ConvertToRankedList("vector", docs, scores)

	// 目前只有向量检索，未来可以添加关键词检索
	// TODO: 添加 BM25 或全文检索支持

	// 融合（目前只有一个列表）
	fusedDocs := strategy.Fuse([]fusion.RankedList{vectorList})

	// 转换为 SearchResult
	results := make([]SearchResult, 0, len(fusedDocs))
	for _, fd := range fusedDocs {
		if m.config.MinScore > 0 && fd.Score < m.config.MinScore {
			continue
		}

		results = append(results, SearchResult{
			Document:    fd.Document,
			Score:       fd.Score,
			VectorScore: fd.SourceScores["vector"],
			VectorRank:  fd.SourceRanks["vector"],
		})

		if len(results) >= topK {
			break
		}
	}

	return results, nil
}

// AddDocuments 添加文档到 Milvus
func (m *MilvusHybridRetriever) AddDocuments(ctx context.Context, documents []types.Document) error {
	loaderDocs := make([]*loaders.Document, len(documents))
	for i, doc := range documents {
		loaderDocs[i] = convertTypesToLoaderDoc(doc)
	}

	_, err := m.store.AddDocuments(ctx, loaderDocs)
	if err != nil {
		return fmt.Errorf("failed to add documents: %w", err)
	}

	return nil
}

// GetStats 获取统计信息
func (m *MilvusHybridRetriever) GetStats() map[string]any {
	return map[string]any{
		"type":              "MilvusHybridRetriever",
		"strategy":          fmt.Sprintf("%T", m.strategy),
		"rrf_k":             m.config.RRFRankConstant,
		"use_native_rrf":    m.config.UseNativeRRF,
		"min_score":         m.config.MinScore,
		"enable_full_text":  m.config.EnableFullText,
	}
}

// WithConfig 设置配置（链式调用）
func (m *MilvusHybridRetriever) WithConfig(config MilvusHybridConfig) *MilvusHybridRetriever {
	m.config = config
	return m
}

// WithMinScore 设置最小分数阈值（链式调用）
func (m *MilvusHybridRetriever) WithMinScore(minScore float64) *MilvusHybridRetriever {
	m.config.MinScore = minScore
	return m
}

// WithRRFConstant 设置 RRF 常量（链式调用）
func (m *MilvusHybridRetriever) WithRRFConstant(k int) *MilvusHybridRetriever {
	m.config.RRFRankConstant = k
	// 更新策略
	m.strategy = fusion.NewRRFStrategy(float64(k))
	return m
}

// MilvusNativeHybridSearch 直接调用 Milvus 原生混合检索（便捷方法）
//
// 这是一个便捷函数，不需要创建 Retriever 实例。
func MilvusNativeHybridSearch(ctx context.Context, store MilvusHybridStore, query string, topK int) ([]SearchResult, error) {
	retriever := NewMilvusHybridRetriever(store, fusion.NewRRFStrategy(60))
	return retriever.Search(ctx, query, topK)
}
