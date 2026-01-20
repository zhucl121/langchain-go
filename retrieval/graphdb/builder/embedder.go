package builder

import (
	"context"
	"fmt"

	"github.com/zhucl121/langchain-go/retrieval/embeddings"
)

// EmbeddingModelAdapter Embedding 模型适配器。
//
// 将 retrieval/embeddings 包的 Embeddings 接口适配为 builder 包的 Embedder 接口。
type EmbeddingModelAdapter struct {
	model embeddings.Embeddings
}

// NewEmbeddingModelAdapter 创建 Embedding 模型适配器。
//
// 参数：
//   - model: Embeddings 模型实例
//
// 返回：
//   - *EmbeddingModelAdapter: 适配器实例
//
func NewEmbeddingModelAdapter(model embeddings.Embeddings) *EmbeddingModelAdapter {
	return &EmbeddingModelAdapter{
		model: model,
	}
}

// Embed 将文本转换为向量。
func (e *EmbeddingModelAdapter) Embed(ctx context.Context, text string) ([]float32, error) {
	// 使用 EmbedQuery 方法（针对单个查询）
	embedding, err := e.model.EmbedQuery(ctx, text)
	if err != nil {
		return nil, fmt.Errorf("embed failed: %w", err)
	}

	return embedding, nil
}

// EmbedBatch 批量转换文本为向量。
func (e *EmbeddingModelAdapter) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return [][]float32{}, nil
	}

	// 使用 EmbedDocuments 方法（针对多个文档）
	results, err := e.model.EmbedDocuments(ctx, texts)
	if err != nil {
		return nil, fmt.Errorf("batch embed failed: %w", err)
	}

	return results, nil
}

// MockEmbedder Mock 向量化器（用于测试）。
type MockEmbedder struct {
	dimension int
}

// NewMockEmbedder 创建 Mock 向量化器。
//
// 参数：
//   - dimension: 向量维度
//
// 返回：
//   - *MockEmbedder: Mock 向量化器实例
//
func NewMockEmbedder(dimension int) *MockEmbedder {
	return &MockEmbedder{
		dimension: dimension,
	}
}

// Embed 生成固定维度的随机向量（用于测试）。
func (m *MockEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	// 生成简单的模拟向量（基于文本长度）
	embedding := make([]float32, m.dimension)
	for i := range embedding {
		embedding[i] = float32(len(text)+i) / float32(m.dimension*100)
	}
	return embedding, nil
}

// EmbedBatch 批量生成向量。
func (m *MockEmbedder) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	results := make([][]float32, len(texts))
	for i, text := range texts {
		embedding, err := m.Embed(ctx, text)
		if err != nil {
			return nil, err
		}
		results[i] = embedding
	}
	return results, nil
}
