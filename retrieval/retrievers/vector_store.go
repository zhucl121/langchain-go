package retrievers

import (
	"context"
	"fmt"

	"github.com/zhucl121/langchain-go/retrieval/loaders"
	"github.com/zhucl121/langchain-go/retrieval/vectorstores"
)

// VectorStoreRetriever 向量存储检索器
//
// 封装向量存储，提供统一的检索接口。
//
type VectorStoreRetriever struct {
	*BaseRetriever
	vectorStore    vectorstores.VectorStore
	searchType     SearchType
	k              int
	scoreThreshold float32
	filter         map[string]interface{}
}

// VectorStoreOption 向量存储检索器配置选项
type VectorStoreOption func(*VectorStoreRetriever)

// NewVectorStoreRetriever 创建向量存储检索器
//
// 参数：
//   - store: 向量存储
//   - opts: 可选配置项
//
// 返回：
//   - *VectorStoreRetriever: 检索器实例
//
// 使用示例：
//
//	retriever := retrievers.NewVectorStoreRetriever(vectorStore,
//	    retrievers.WithSearchType(retrievers.SearchSimilarity),
//	    retrievers.WithTopK(5),
//	    retrievers.WithScoreThreshold(0.7),
//	)
//
func NewVectorStoreRetriever(
	store vectorstores.VectorStore,
	opts ...VectorStoreOption,
) *VectorStoreRetriever {
	retriever := &VectorStoreRetriever{
		BaseRetriever:  NewBaseRetriever(),
		vectorStore:    store,
		searchType:     SearchSimilarity,
		k:              5,
		scoreThreshold: 0.0,
		filter:         nil,
	}

	for _, opt := range opts {
		opt(retriever)
	}

	return retriever
}

// 配置选项函数

// WithSearchType 设置搜索类型
func WithSearchType(searchType SearchType) VectorStoreOption {
	return func(r *VectorStoreRetriever) {
		r.searchType = searchType
	}
}

// WithVectorStoreTopK 设置返回文档数量
func WithVectorStoreTopK(k int) VectorStoreOption {
	return func(r *VectorStoreRetriever) {
		r.k = k
	}
}

// WithScoreThreshold 设置分数阈值
func WithScoreThreshold(threshold float32) VectorStoreOption {
	return func(r *VectorStoreRetriever) {
		r.scoreThreshold = threshold
	}
}

// WithFilter 设置过滤条件
func WithFilter(filter map[string]interface{}) VectorStoreOption {
	return func(r *VectorStoreRetriever) {
		r.filter = filter
	}
}

// GetRelevantDocuments 实现 Retriever 接口
//
// 根据配置的搜索类型执行检索。
//
func (r *VectorStoreRetriever) GetRelevantDocuments(ctx context.Context, query string) ([]*loaders.Document, error) {
	// 触发开始回调
	r.triggerStart(ctx, query)

	var docs []*loaders.Document
	var err error

	// 根据搜索类型执行不同的检索策略
	switch r.searchType {
	case SearchSimilarity:
		docs, err = r.vectorStore.SimilaritySearch(ctx, query, r.k)

	case SearchMMR:
		// MMR 搜索需要向量存储支持
		// 如果 vectorStore 实现了 MMR 接口，使用 MMR
		if mmrStore, ok := r.vectorStore.(interface {
			MMRSearch(ctx context.Context, query string, k int) ([]*loaders.Document, error)
		}); ok {
			docs, err = mmrStore.MMRSearch(ctx, query, r.k)
		} else {
			// 降级到相似度搜索
			docs, err = r.vectorStore.SimilaritySearch(ctx, query, r.k)
		}

	case SearchHybrid:
		// 混合搜索需要向量存储支持
		if hybridStore, ok := r.vectorStore.(interface {
			HybridSearch(ctx context.Context, query string, k int, filter map[string]interface{}) ([]vectorstores.DocumentWithScore, error)
		}); ok {
			results, err := hybridStore.HybridSearch(ctx, query, r.k, r.filter)
			if err != nil {
				r.triggerError(ctx, err)
				return nil, err
			}

			docs = make([]*loaders.Document, len(results))
			for i, result := range results {
				// 添加分数到元数据
				if result.Document.Metadata == nil {
					result.Document.Metadata = make(map[string]interface{})
				}
				result.Document.Metadata["score"] = result.Score
				docs[i] = result.Document
			}
		} else {
			// 降级到相似度搜索
			docs, err = r.vectorStore.SimilaritySearch(ctx, query, r.k)
		}

	default:
		docs, err = r.vectorStore.SimilaritySearch(ctx, query, r.k)
	}

	if err != nil {
		r.triggerError(ctx, err)
		return nil, fmt.Errorf("search failed: %w", err)
	}

	// 过滤低分文档
	if r.scoreThreshold > 0 {
		docs = r.filterByScore(docs)
	}

	// 触发结束回调
	r.triggerEnd(ctx, docs)

	return docs, nil
}

// GetRelevantDocumentsWithScore 实现 Retriever 接口
//
// 返回带分数的文档。
//
func (r *VectorStoreRetriever) GetRelevantDocumentsWithScore(ctx context.Context, query string) ([]DocumentWithScore, error) {
	// 触发开始回调
	r.triggerStart(ctx, query)

	// 使用带分数的搜索
	results, err := r.vectorStore.SimilaritySearchWithScore(ctx, query, r.k)
	if err != nil {
		r.triggerError(ctx, err)
		return nil, fmt.Errorf("search with score failed: %w", err)
	}

	// 过滤低分文档
	if r.scoreThreshold > 0 {
		var filtered []vectorstores.DocumentWithScore
		for _, result := range results {
			if result.Score >= r.scoreThreshold {
				filtered = append(filtered, result)
			}
		}
		results = filtered
	}

	// 转换为 retriever 的 DocumentWithScore 类型
	docs := make([]DocumentWithScore, len(results))
	for i, result := range results {
		// 确保分数在元数据中
		if result.Document.Metadata == nil {
			result.Document.Metadata = make(map[string]interface{})
		}
		result.Document.Metadata["score"] = result.Score

		docs[i] = DocumentWithScore{
			Document: result.Document,
			Score:    result.Score,
		}
	}

	// 触发结束回调
	plainDocs := make([]*loaders.Document, len(docs))
	for i, d := range docs {
		plainDocs[i] = d.Document
	}
	r.triggerEnd(ctx, plainDocs)

	return docs, nil
}

// filterByScore 按分数过滤文档
func (r *VectorStoreRetriever) filterByScore(docs []*loaders.Document) []*loaders.Document {
	var filtered []*loaders.Document

	for _, doc := range docs {
		// 尝试从元数据中获取分数
		if score, ok := doc.Metadata["score"].(float32); ok {
			if score >= r.scoreThreshold {
				filtered = append(filtered, doc)
			}
		} else if score, ok := doc.Metadata["score"].(float64); ok {
			if float32(score) >= r.scoreThreshold {
				filtered = append(filtered, doc)
			}
		} else {
			// 没有分数信息，保留
			filtered = append(filtered, doc)
		}
	}

	return filtered
}

// SetK 设置返回文档数量
func (r *VectorStoreRetriever) SetK(k int) {
	r.k = k
}

// SetScoreThreshold 设置分数阈值
func (r *VectorStoreRetriever) SetScoreThreshold(threshold float32) {
	r.scoreThreshold = threshold
}

// SetSearchType 设置搜索类型
func (r *VectorStoreRetriever) SetSearchType(searchType SearchType) {
	r.searchType = searchType
}

// GetVectorStore 获取底层向量存储
func (r *VectorStoreRetriever) GetVectorStore() vectorstores.VectorStore {
	return r.vectorStore
}
