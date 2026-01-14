// Package embeddings 提供嵌入模型接口和实现。
//
// Embeddings 将文本转换为向量表示，用于语义搜索和相似度计算。
//
// 支持的嵌入模型：
//   - OpenAI Embeddings (text-embedding-ada-002, text-embedding-3-small, etc.)
//   - 本地嵌入模型（通过接口扩展）
//
// 使用示例：
//
//	embeddings := embeddings.NewOpenAIEmbeddings("sk-...")
//	vectors, err := embeddings.EmbedDocuments(ctx, []string{"text1", "text2"})
//
package embeddings

import (
	"context"
)

// Embeddings 是嵌入模型接口。
//
// 所有嵌入模型都必须实现此接口。
//
type Embeddings interface {
	// EmbedDocuments 嵌入多个文档
	//
	// 参数：
	//   - ctx: 上下文
	//   - texts: 文本列表
	//
	// 返回：
	//   - [][]float32: 向量列表
	//   - error: 错误
	//
	EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error)
	
	// EmbedQuery 嵌入单个查询
	//
	// 某些模型对查询和文档使用不同的嵌入策略。
	//
	// 参数：
	//   - ctx: 上下文
	//   - text: 查询文本
	//
	// 返回：
	//   - []float32: 向量
	//   - error: 错误
	//
	EmbedQuery(ctx context.Context, text string) ([]float32, error)
	
	// GetDimension 获取向量维度
	//
	// 返回：
	//   - int: 向量维度
	//
	GetDimension() int
}

// BaseEmbeddings 提供嵌入模型的基础实现。
type BaseEmbeddings struct {
	dimension int
	modelName string
}

// NewBaseEmbeddings 创建基础嵌入模型。
func NewBaseEmbeddings(modelName string, dimension int) *BaseEmbeddings {
	return &BaseEmbeddings{
		dimension: dimension,
		modelName: modelName,
	}
}

// GetDimension 实现 Embeddings 接口。
func (be *BaseEmbeddings) GetDimension() int {
	return be.dimension
}

// GetModelName 获取模型名称。
func (be *BaseEmbeddings) GetModelName() string {
	return be.modelName
}
