package builder

import (
	"context"

	"github.com/zhucl121/langchain-go/retrieval/graphdb"
)

// EntityExtractor 实体提取器接口。
//
// EntityExtractor 从文本中识别和提取实体。
type EntityExtractor interface {
	// Extract 从文本中提取实体。
	//
	// 参数：
	//   - ctx: 上下文
	//   - text: 输入文本
	//
	// 返回：
	//   - []Entity: 提取到的实体列表
	//   - error: 错误
	//
	Extract(ctx context.Context, text string) ([]Entity, error)

	// ExtractWithSchema 使用指定 Schema 提取实体。
	//
	// 参数：
	//   - ctx: 上下文
	//   - text: 输入文本
	//   - schema: 实体 Schema（nil 表示使用默认）
	//
	// 返回：
	//   - []Entity: 提取到的实体列表
	//   - error: 错误
	//
	ExtractWithSchema(ctx context.Context, text string, schema *EntitySchema) ([]Entity, error)
}

// RelationExtractor 关系提取器接口。
//
// RelationExtractor 从文本和实体中提取关系。
type RelationExtractor interface {
	// Extract 从文本中提取关系。
	//
	// 参数：
	//   - ctx: 上下文
	//   - text: 输入文本
	//   - entities: 已知实体列表（可选，用于辅助提取）
	//
	// 返回：
	//   - []Relation: 提取到的关系列表
	//   - error: 错误
	//
	Extract(ctx context.Context, text string, entities []Entity) ([]Relation, error)

	// ExtractWithSchema 使用指定 Schema 提取关系。
	//
	// 参数：
	//   - ctx: 上下文
	//   - text: 输入文本
	//   - entities: 已知实体列表
	//   - schema: 关系 Schema（nil 表示使用默认）
	//
	// 返回：
	//   - []Relation: 提取到的关系列表
	//   - error: 错误
	//
	ExtractWithSchema(ctx context.Context, text string, entities []Entity, schema *RelationSchema) ([]Relation, error)
}

// Embedder 向量化器接口。
//
// Embedder 将文本转换为向量表示。
type Embedder interface {
	// Embed 将文本转换为向量。
	//
	// 参数：
	//   - ctx: 上下文
	//   - text: 输入文本
	//
	// 返回：
	//   - []float32: 向量表示
	//   - error: 错误
	//
	Embed(ctx context.Context, text string) ([]float32, error)

	// EmbedBatch 批量转换文本为向量。
	//
	// 参数：
	//   - ctx: 上下文
	//   - texts: 输入文本列表
	//
	// 返回：
	//   - [][]float32: 向量列表
	//   - error: 错误
	//
	EmbedBatch(ctx context.Context, texts []string) ([][]float32, error)
}

// KGBuilder 知识图谱构建器接口。
//
// KGBuilder 协调实体提取、关系抽取和图谱构建的完整流程。
type KGBuilder interface {
	// Build 从文本构建知识图谱。
	//
	// 参数：
	//   - ctx: 上下文
	//   - text: 输入文本
	//
	// 返回：
	//   - *KnowledgeGraph: 构建的知识图谱
	//   - error: 错误
	//
	Build(ctx context.Context, text string) (*KnowledgeGraph, error)

	// BuildAndStore 构建知识图谱并存储到图数据库。
	//
	// 参数：
	//   - ctx: 上下文
	//   - text: 输入文本
	//
	// 返回：
	//   - *KnowledgeGraph: 构建的知识图谱
	//   - error: 错误
	//
	BuildAndStore(ctx context.Context, text string) (*KnowledgeGraph, error)

	// BuildBatch 批量构建知识图谱。
	//
	// 参数：
	//   - ctx: 上下文
	//   - texts: 输入文本列表
	//
	// 返回：
	//   - []*KnowledgeGraph: 构建的知识图谱列表
	//   - error: 错误
	//
	BuildBatch(ctx context.Context, texts []string) ([]*KnowledgeGraph, error)

	// Merge 合并多个知识图谱。
	//
	// 参数：
	//   - ctx: 上下文
	//   - graphs: 知识图谱列表
	//
	// 返回：
	//   - *KnowledgeGraph: 合并后的知识图谱
	//   - error: 错误
	//
	Merge(ctx context.Context, graphs []*KnowledgeGraph) (*KnowledgeGraph, error)

	// UpdateIncremental 增量更新知识图谱。
	//
	// 参数：
	//   - ctx: 上下文
	//   - text: 新文本
	//
	// 返回：
	//   - *ExtractionResult: 提取结果（新增的实体和关系）
	//   - error: 错误
	//
	UpdateIncremental(ctx context.Context, text string) (*ExtractionResult, error)
}

// EntityDisambiguator 实体消歧器接口。
//
// EntityDisambiguator 用于解决实体引用的歧义，将多个引用合并为同一实体。
type EntityDisambiguator interface {
	// Disambiguate 消歧实体列表。
	//
	// 参数：
	//   - ctx: 上下文
	//   - entities: 实体列表
	//
	// 返回：
	//   - []Entity: 消歧后的实体列表（合并了重复实体）
	//   - map[string]string: 实体 ID 映射（旧 ID -> 新 ID）
	//   - error: 错误
	//
	Disambiguate(ctx context.Context, entities []Entity) ([]Entity, map[string]string, error)

	// ResolveEntity 解析实体引用。
	//
	// 参数：
	//   - ctx: 上下文
	//   - entity: 待解析的实体
	//   - candidates: 候选实体列表（从图数据库中查询）
	//
	// 返回：
	//   - *Entity: 匹配的实体（nil 表示无匹配）
	//   - float64: 匹配置信度
	//   - error: 错误
	//
	ResolveEntity(ctx context.Context, entity Entity, candidates []Entity) (*Entity, float64, error)
}

// KGValidator 知识图谱验证器接口。
//
// KGValidator 验证知识图谱的质量和一致性。
type KGValidator interface {
	// ValidateGraph 验证知识图谱。
	//
	// 参数：
	//   - ctx: 上下文
	//   - kg: 知识图谱
	//
	// 返回：
	//   - []ValidationError: 验证错误列表
	//   - error: 错误
	//
	ValidateGraph(ctx context.Context, kg *KnowledgeGraph) ([]ValidationError, error)

	// ValidateEntity 验证实体。
	//
	// 参数：
	//   - entity: 实体
	//   - schema: 实体 Schema（nil 表示跳过 Schema 验证）
	//
	// 返回：
	//   - []ValidationError: 验证错误列表
	//
	ValidateEntity(entity Entity, schema *EntitySchema) []ValidationError

	// ValidateRelation 验证关系。
	//
	// 参数：
	//   - relation: 关系
	//   - schema: 关系 Schema（nil 表示跳过 Schema 验证）
	//
	// 返回：
	//   - []ValidationError: 验证错误列表
	//
	ValidateRelation(relation Relation, schema *RelationSchema) []ValidationError
}

// ValidationError 验证错误。
type ValidationError struct {
	// Type 错误类型
	Type string

	// EntityID 相关实体 ID
	EntityID string

	// RelationID 相关关系 ID
	RelationID string

	// Field 错误字段
	Field string

	// Message 错误消息
	Message string

	// Severity 严重程度（error, warning, info）
	Severity string
}

// KGBuilderConfig KGBuilder 配置。
type KGBuilderConfig struct {
	// GraphDB 图数据库
	GraphDB graphdb.GraphDB

	// EntityExtractor 实体提取器
	EntityExtractor EntityExtractor

	// RelationExtractor 关系提取器
	RelationExtractor RelationExtractor

	// Embedder 向量化器（可选）
	Embedder Embedder

	// Disambiguator 实体消歧器（可选）
	Disambiguator EntityDisambiguator

	// Validator 验证器（可选）
	Validator KGValidator

	// EntitySchemas 实体 Schema 映射（可选）
	EntitySchemas map[string]*EntitySchema

	// RelationSchemas 关系 Schema 映射（可选）
	RelationSchemas map[string]*RelationSchema

	// EnableEmbedding 是否启用实体向量化
	EnableEmbedding bool

	// EnableDisambiguation 是否启用实体消歧
	EnableDisambiguation bool

	// EnableValidation 是否启用验证
	EnableValidation bool

	// BatchSize 批量处理大小
	BatchSize int

	// MaxConcurrency 最大并发数
	MaxConcurrency int
}

// DefaultKGBuilderConfig 返回默认配置。
func DefaultKGBuilderConfig() KGBuilderConfig {
	return KGBuilderConfig{
		EnableEmbedding:      true,
		EnableDisambiguation: true,
		EnableValidation:     true,
		BatchSize:            10,
		MaxConcurrency:       5,
	}
}
