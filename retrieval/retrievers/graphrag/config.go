package graphrag

import (
	"fmt"

	"github.com/zhucl121/langchain-go/core/chat"
	"github.com/zhucl121/langchain-go/retrieval/embeddings"
	"github.com/zhucl121/langchain-go/retrieval/graphdb"
	"github.com/zhucl121/langchain-go/retrieval/graphdb/builder"
	"github.com/zhucl121/langchain-go/retrieval/vectorstores"
)

// Config GraphRAG 检索器配置。
type Config struct {
	// GraphDB 图数据库
	GraphDB graphdb.GraphDB

	// VectorStore 向量存储
	VectorStore vectorstores.VectorStore

	// EntityExtractor 实体提取器（用于查询实体识别）
	EntityExtractor builder.EntityExtractor

	// Embeddings 嵌入模型（用于向量化）
	Embeddings embeddings.Embeddings

	// ChatModel 聊天模型（用于实体提取）
	ChatModel chat.ChatModel

	// VectorWeight 向量检索权重 (0-1，默认 0.6)
	VectorWeight float64

	// GraphWeight 图检索权重 (0-1，默认 0.4)
	GraphWeight float64

	// MaxTraverseDepth 最大遍历深度（默认 2)
	MaxTraverseDepth int

	// TopK 默认返回的结果数量（默认 10）
	TopK int

	// FusionStrategy 融合策略（默认 weighted）
	FusionStrategy FusionStrategy

	// RerankStrategy 重排序策略（默认 score）
	RerankStrategy RerankStrategy

	// EnableContextAugmentation 是否启用上下文增强（默认 true）
	EnableContextAugmentation bool

	// MinScore 最小分数阈值（默认 0.0）
	MinScore float64

	// RRFConstant RRF 融合常数（默认 60）
	RRFConstant float64

	// MMRLambda MMR 多样性参数（0-1，默认 0.5）
	MMRLambda float64
}

// Validate 验证配置。
func (c *Config) Validate() error {
	if c.GraphDB == nil {
		return fmt.Errorf("GraphDB is required")
	}

	if c.VectorStore == nil {
		return fmt.Errorf("VectorStore is required")
	}

	if c.VectorWeight < 0 || c.VectorWeight > 1 {
		return fmt.Errorf("VectorWeight must be between 0 and 1")
	}

	if c.GraphWeight < 0 || c.GraphWeight > 1 {
		return fmt.Errorf("GraphWeight must be between 0 and 1")
	}

	if c.VectorWeight+c.GraphWeight <= 0 {
		return fmt.Errorf("VectorWeight + GraphWeight must be greater than 0")
	}

	if c.MaxTraverseDepth < 1 {
		return fmt.Errorf("MaxTraverseDepth must be at least 1")
	}

	if c.TopK < 1 {
		return fmt.Errorf("TopK must be at least 1")
	}

	if c.MMRLambda < 0 || c.MMRLambda > 1 {
		return fmt.Errorf("MMRLambda must be between 0 and 1")
	}

	return nil
}

// DefaultConfig 返回默认配置。
func DefaultConfig(graphDB graphdb.GraphDB, vectorStore vectorstores.VectorStore) Config {
	return Config{
		GraphDB:                   graphDB,
		VectorStore:               vectorStore,
		VectorWeight:              0.6,
		GraphWeight:               0.4,
		MaxTraverseDepth:          2,
		TopK:                      10,
		FusionStrategy:            FusionStrategyWeighted,
		RerankStrategy:            RerankStrategyScore,
		EnableContextAugmentation: true,
		MinScore:                  0.0,
		RRFConstant:               60.0,
		MMRLambda:                 0.5,
	}
}
