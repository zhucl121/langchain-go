package graphrag

import (
	"github.com/zhucl121/langchain-go/pkg/types"
	"github.com/zhucl121/langchain-go/retrieval/graphdb"
)

// FusedResult 表示融合后的检索结果。
type FusedResult struct {
	// Document 文档
	Document *types.Document

	// VectorScore 向量检索分数 (0-1)
	VectorScore float64

	// GraphScore 图检索分数 (0-1)
	GraphScore float64

	// FusedScore 融合后的最终分数
	FusedScore float64

	// Rank 排名
	Rank int

	// RelatedNodes 相关的图节点
	RelatedNodes []*graphdb.Node

	// Metadata 额外元数据
	Metadata map[string]interface{}
}

// EntityMatch 表示实体匹配结果。
type EntityMatch struct {
	// EntityID 实体 ID
	EntityID string

	// EntityName 实体名称
	EntityName string

	// EntityType 实体类型
	EntityType string

	// MatchScore 匹配分数 (0-1)
	MatchScore float64

	// MatchedText 匹配的文本
	MatchedText string
}

// ContextInfo 表示上下文信息。
type ContextInfo struct {
	// RelatedEntities 相关实体列表
	RelatedEntities []string

	// RelationshipPaths 关系路径
	RelationshipPaths []string

	// NeighborCount 邻居数量
	NeighborCount int

	// GraphDepth 图深度
	GraphDepth int

	// AdditionalContext 额外上下文文本
	AdditionalContext string
}

// FusionStrategy 融合策略类型。
type FusionStrategy string

const (
	// FusionStrategyWeighted 加权融合
	FusionStrategyWeighted FusionStrategy = "weighted"

	// FusionStrategyRRF Reciprocal Rank Fusion
	FusionStrategyRRF FusionStrategy = "rrf"

	// FusionStrategyMax 取最大值
	FusionStrategyMax FusionStrategy = "max"

	// FusionStrategyMin 取最小值
	FusionStrategyMin FusionStrategy = "min"
)

// RerankStrategy 重排序策略类型。
type RerankStrategy string

const (
	// RerankStrategyScore 基于分数重排
	RerankStrategyScore RerankStrategy = "score"

	// RerankStrategyDiversity 基于多样性重排
	RerankStrategyDiversity RerankStrategy = "diversity"

	// RerankStrategyMMR Maximal Marginal Relevance
	RerankStrategyMMR RerankStrategy = "mmr"
)

// SearchMode 检索模式。
type SearchMode string

const (
	// SearchModeHybrid 混合检索（向量+图）
	SearchModeHybrid SearchMode = "hybrid"

	// SearchModeVector 仅向量检索
	SearchModeVector SearchMode = "vector"

	// SearchModeGraph 仅图检索
	SearchModeGraph SearchMode = "graph"
)

// SearchOptions 检索选项。
type SearchOptions struct {
	// Mode 检索模式
	Mode SearchMode

	// K 返回结果数量
	K int

	// VectorWeight 向量检索权重 (0-1)
	VectorWeight float64

	// GraphWeight 图检索权重 (0-1)
	GraphWeight float64

	// MaxTraverseDepth 最大遍历深度
	MaxTraverseDepth int

	// FusionStrategy 融合策略
	FusionStrategy FusionStrategy

	// RerankStrategy 重排序策略
	RerankStrategy RerankStrategy

	// EnableContextAugmentation 是否启用上下文增强
	EnableContextAugmentation bool

	// MinScore 最小分数阈值
	MinScore float64

	// IncludeMetadata 是否包含元数据
	IncludeMetadata bool
}

// DefaultSearchOptions 返回默认检索选项。
func DefaultSearchOptions() SearchOptions {
	return SearchOptions{
		Mode:                      SearchModeHybrid,
		K:                         10,
		VectorWeight:              0.6,
		GraphWeight:               0.4,
		MaxTraverseDepth:          2,
		FusionStrategy:            FusionStrategyWeighted,
		RerankStrategy:            RerankStrategyScore,
		EnableContextAugmentation: true,
		MinScore:                  0.0,
		IncludeMetadata:           true,
	}
}

// Statistics 检索统计信息。
type Statistics struct {
	// VectorResultsCount 向量检索结果数
	VectorResultsCount int

	// GraphResultsCount 图检索结果数
	GraphResultsCount int

	// FusedResultsCount 融合后结果数
	FusedResultsCount int

	// EntitiesExtracted 提取的实体数
	EntitiesExtracted int

	// NodesTraversed 遍历的节点数
	NodesTraversed int

	// AverageGraphDepth 平均图深度
	AverageGraphDepth float64

	// VectorSearchTime 向量检索耗时 (ms)
	VectorSearchTime int64

	// GraphSearchTime 图检索耗时 (ms)
	GraphSearchTime int64

	// FusionTime 融合耗时 (ms)
	FusionTime int64

	// RerankTime 重排序耗时 (ms)
	RerankTime int64

	// TotalTime 总耗时 (ms)
	TotalTime int64
}
