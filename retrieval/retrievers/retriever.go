// Package retrievers 提供统一的文档检索抽象。
//
// Retriever 是连接向量存储和应用的桥梁，提供统一的检索接口。
//
// 支持的 Retriever 类型：
//   - VectorStoreRetriever: 向量存储检索器
//   - MultiQueryRetriever: 多查询检索器
//   - EnsembleRetriever: 集成检索器 (混合检索)
//
// 使用示例：
//
//	retriever := retrievers.NewVectorStoreRetriever(vectorStore,
//	    retrievers.WithSearchType(retrievers.SearchSimilarity),
//	    retrievers.WithTopK(5),
//	)
//	docs, _ := retriever.GetRelevantDocuments(ctx, "query")
//
package retrievers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"

	"github.com/zhucl121/langchain-go/retrieval/loaders"
)

// Retriever 检索器接口
//
// 所有检索器都实现此接口，提供统一的文档检索功能。
//
type Retriever interface {
	// GetRelevantDocuments 获取相关文档
	//
	// 参数：
	//   - ctx: 上下文
	//   - query: 查询文本
	//
	// 返回：
	//   - []*loaders.Document: 相关文档列表
	//   - error: 错误
	//
	GetRelevantDocuments(ctx context.Context, query string) ([]*loaders.Document, error)

	// GetRelevantDocumentsWithScore 带分数获取文档
	//
	// 参数：
	//   - ctx: 上下文
	//   - query: 查询文本
	//
	// 返回：
	//   - []DocumentWithScore: 带分数的文档列表
	//   - error: 错误
	//
	GetRelevantDocumentsWithScore(ctx context.Context, query string) ([]DocumentWithScore, error)
}

// DocumentWithScore 带分数的文档
type DocumentWithScore struct {
	Document *loaders.Document
	Score    float32 // 相似度分数（越高越相似）
}

// BaseRetriever 基础检索器
//
// 提供通用功能，可以被具体实现继承。
//
type BaseRetriever struct {
	callbacks []Callback
	metadata  map[string]interface{}
}

// Callback 回调接口
//
// 用于监控检索过程。
//
type Callback interface {
	// OnRetrieverStart 检索开始时调用
	OnRetrieverStart(ctx context.Context, query string)

	// OnRetrieverEnd 检索结束时调用
	OnRetrieverEnd(ctx context.Context, docs []*loaders.Document)

	// OnRetrieverError 检索错误时调用
	OnRetrieverError(ctx context.Context, err error)
}

// NewBaseRetriever 创建基础检索器
func NewBaseRetriever() *BaseRetriever {
	return &BaseRetriever{
		callbacks: make([]Callback, 0),
		metadata:  make(map[string]interface{}),
	}
}

// AddCallback 添加回调
func (b *BaseRetriever) AddCallback(callback Callback) {
	b.callbacks = append(b.callbacks, callback)
}

// SetMetadata 设置元数据
func (b *BaseRetriever) SetMetadata(key string, value interface{}) {
	b.metadata[key] = value
}

// GetMetadata 获取元数据
func (b *BaseRetriever) GetMetadata(key string) interface{} {
	return b.metadata[key]
}

// triggerStart 触发开始回调
func (b *BaseRetriever) triggerStart(ctx context.Context, query string) {
	for _, cb := range b.callbacks {
		cb.OnRetrieverStart(ctx, query)
	}
}

// triggerEnd 触发结束回调
func (b *BaseRetriever) triggerEnd(ctx context.Context, docs []*loaders.Document) {
	for _, cb := range b.callbacks {
		cb.OnRetrieverEnd(ctx, docs)
	}
}

// triggerError 触发错误回调
func (b *BaseRetriever) triggerError(ctx context.Context, err error) {
	for _, cb := range b.callbacks {
		cb.OnRetrieverError(ctx, err)
	}
}

// SearchType 搜索类型
type SearchType string

const (
	// SearchSimilarity 相似度搜索
	SearchSimilarity SearchType = "similarity"

	// SearchMMR 最大边际相关性搜索
	// 平衡相关性和多样性
	SearchMMR SearchType = "mmr"

	// SearchHybrid 混合搜索
	// 结合向量搜索和关键词搜索
	SearchHybrid SearchType = "hybrid"
)

// RetrieverOption 检索器配置选项
type RetrieverOption func(interface{})

// MultiQueryOption 多查询检索器配置选项
type MultiQueryOption func(*MultiQueryRetriever)

// EnsembleOption 集成检索器配置选项
type EnsembleOption func(*EnsembleRetriever)

// hashContent 计算内容哈希
//
// 用于文档去重。
//
func hashContent(content string) string {
	hash := sha256.Sum256([]byte(content))
	return hex.EncodeToString(hash[:])
}

// min 返回较小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
