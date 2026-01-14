// Package vectorstores 提供向量存储接口和实现。
//
// Vector Stores 用于存储和检索文档向量，支持语义搜索。
//
// 支持的向量存储：
//   - InMemoryVectorStore: 内存向量存储（适合开发和小规模应用）
//   - 可扩展支持 Chroma、Pinecone、Weaviate 等
//
// 使用示例：
//
//	store := vectorstores.NewInMemoryVectorStore(embeddings)
//	store.AddDocuments(ctx, docs)
//	results, _ := store.SimilaritySearch(ctx, "query", 5)
//
package vectorstores

import (
	"context"
	"fmt"
	"math"
	"sort"
	"sync"
	
	"langchain-go/retrieval/embeddings"
	"langchain-go/retrieval/loaders"
)

// VectorStore 是向量存储接口。
type VectorStore interface {
	// AddDocuments 添加文档
	//
	// 参数：
	//   - ctx: 上下文
	//   - docs: 文档列表
	//
	// 返回：
	//   - []string: 文档 ID 列表
	//   - error: 错误
	//
	AddDocuments(ctx context.Context, docs []*loaders.Document) ([]string, error)
	
	// SimilaritySearch 相似度搜索
	//
	// 参数：
	//   - ctx: 上下文
	//   - query: 查询文本
	//   - k: 返回结果数量
	//
	// 返回：
	//   - []*loaders.Document: 相似文档列表
	//   - error: 错误
	//
	SimilaritySearch(ctx context.Context, query string, k int) ([]*loaders.Document, error)
	
	// SimilaritySearchWithScore 带分数的相似度搜索
	//
	// 参数：
	//   - ctx: 上下文
	//   - query: 查询文本
	//   - k: 返回结果数量
	//
	// 返回：
	//   - []DocumentWithScore: 带分数的文档列表
	//   - error: 错误
	//
	SimilaritySearchWithScore(ctx context.Context, query string, k int) ([]DocumentWithScore, error)
	
	// Delete 删除文档
	//
	// 参数：
	//   - ctx: 上下文
	//   - ids: 文档 ID 列表
	//
	// 返回：
	//   - error: 错误
	//
	Delete(ctx context.Context, ids []string) error
}

// DocumentWithScore 表示带相似度分数的文档。
type DocumentWithScore struct {
	Document *loaders.Document
	Score    float32 // 相似度分数（越高越相似）
}

// InMemoryVectorStore 是内存向量存储。
//
// 适合开发、测试和小规模应用。
//
type InMemoryVectorStore struct {
	embeddings embeddings.Embeddings
	
	mu        sync.RWMutex
	documents map[string]*loaders.Document // ID -> Document
	vectors   map[string][]float32         // ID -> Vector
	idCounter int
}

// NewInMemoryVectorStore 创建内存向量存储。
//
// 参数：
//   - embeddings: 嵌入模型
//
// 返回：
//   - *InMemoryVectorStore: 向量存储实例
//
func NewInMemoryVectorStore(embeddings embeddings.Embeddings) *InMemoryVectorStore {
	return &InMemoryVectorStore{
		embeddings: embeddings,
		documents:  make(map[string]*loaders.Document),
		vectors:    make(map[string][]float32),
		idCounter:  0,
	}
}

// AddDocuments 实现 VectorStore 接口。
func (store *InMemoryVectorStore) AddDocuments(ctx context.Context, docs []*loaders.Document) ([]string, error) {
	if len(docs) == 0 {
		return []string{}, nil
	}
	
	// 提取文本
	texts := make([]string, len(docs))
	for i, doc := range docs {
		texts[i] = doc.Content
	}
	
	// 生成嵌入
	vectors, err := store.embeddings.EmbedDocuments(ctx, texts)
	if err != nil {
		return nil, err
	}
	
	// 存储
	store.mu.Lock()
	defer store.mu.Unlock()
	
	ids := make([]string, len(docs))
	for i, doc := range docs {
		store.idCounter++
		id := generateID(store.idCounter)
		ids[i] = id
		
		store.documents[id] = doc
		store.vectors[id] = vectors[i]
	}
	
	return ids, nil
}

// SimilaritySearch 实现 VectorStore 接口。
func (store *InMemoryVectorStore) SimilaritySearch(ctx context.Context, query string, k int) ([]*loaders.Document, error) {
	results, err := store.SimilaritySearchWithScore(ctx, query, k)
	if err != nil {
		return nil, err
	}
	
	docs := make([]*loaders.Document, len(results))
	for i, result := range results {
		docs[i] = result.Document
	}
	
	return docs, nil
}

// SimilaritySearchWithScore 实现 VectorStore 接口。
func (store *InMemoryVectorStore) SimilaritySearchWithScore(ctx context.Context, query string, k int) ([]DocumentWithScore, error) {
	// 生成查询向量
	queryVector, err := store.embeddings.EmbedQuery(ctx, query)
	if err != nil {
		return nil, err
	}
	
	// 计算所有文档的相似度
	store.mu.RLock()
	defer store.mu.RUnlock()
	
	type scoredDoc struct {
		id    string
		score float32
	}
	
	scores := make([]scoredDoc, 0, len(store.documents))
	for id, vector := range store.vectors {
		score := cosineSimilarity(queryVector, vector)
		scores = append(scores, scoredDoc{id: id, score: score})
	}
	
	// 按相似度排序（降序）
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].score > scores[j].score
	})
	
	// 取前 k 个
	if k > len(scores) {
		k = len(scores)
	}
	
	results := make([]DocumentWithScore, k)
	for i := 0; i < k; i++ {
		results[i] = DocumentWithScore{
			Document: store.documents[scores[i].id],
			Score:    scores[i].score,
		}
	}
	
	return results, nil
}

// Delete 实现 VectorStore 接口。
func (store *InMemoryVectorStore) Delete(ctx context.Context, ids []string) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	
	for _, id := range ids {
		delete(store.documents, id)
		delete(store.vectors, id)
	}
	
	return nil
}

// GetDocumentCount 获取文档数量。
func (store *InMemoryVectorStore) GetDocumentCount() int {
	store.mu.RLock()
	defer store.mu.RUnlock()
	return len(store.documents)
}

// Clear 清空所有文档。
func (store *InMemoryVectorStore) Clear() {
	store.mu.Lock()
	defer store.mu.Unlock()
	
	store.documents = make(map[string]*loaders.Document)
	store.vectors = make(map[string][]float32)
	store.idCounter = 0
}

// cosineSimilarity 计算余弦相似度。
func cosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0
	}
	
	var dotProduct float32
	var normA float32
	var normB float32
	
	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}
	
	if normA == 0 || normB == 0 {
		return 0
	}
	
	return dotProduct / (float32(math.Sqrt(float64(normA))) * float32(math.Sqrt(float64(normB))))
}

// generateID 生成文档 ID。
func generateID(counter int) string {
	return fmt.Sprintf("doc_%d", counter)
}
