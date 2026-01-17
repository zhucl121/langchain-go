package vectorstores

import (
	"context"
	"testing"
	
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	
	"github.com/zhucl121/langchain-go/retrieval/embeddings"
	"github.com/zhucl121/langchain-go/retrieval/loaders"
)

// TestNewInMemoryVectorStore
func TestNewInMemoryVectorStore(t *testing.T) {
	emb := embeddings.NewFakeEmbeddings(128)
	store := NewInMemoryVectorStore(emb)
	
	assert.NotNil(t, store)
	assert.Equal(t, 0, store.GetDocumentCount())
}

// TestInMemoryVectorStore_AddDocuments
func TestInMemoryVectorStore_AddDocuments(t *testing.T) {
	emb := embeddings.NewFakeEmbeddings(128)
	store := NewInMemoryVectorStore(emb)
	
	docs := []*loaders.Document{
		loaders.NewDocument("First document", map[string]any{"id": 1}),
		loaders.NewDocument("Second document", map[string]any{"id": 2}),
	}
	
	ids, err := store.AddDocuments(context.Background(), docs)
	
	assert.NoError(t, err)
	assert.Len(t, ids, 2)
	assert.Equal(t, 2, store.GetDocumentCount())
}

// TestInMemoryVectorStore_SimilaritySearch
func TestInMemoryVectorStore_SimilaritySearch(t *testing.T) {
	emb := embeddings.NewFakeEmbeddings(128)
	store := NewInMemoryVectorStore(emb)
	
	docs := []*loaders.Document{
		loaders.NewDocument("Machine learning is a subset of AI", nil),
		loaders.NewDocument("Deep learning uses neural networks", nil),
		loaders.NewDocument("Python is a programming language", nil),
		loaders.NewDocument("Go is fast and efficient", nil),
	}
	
	_, err := store.AddDocuments(context.Background(), docs)
	require.NoError(t, err)
	
	// 搜索相似文档
	results, err := store.SimilaritySearch(context.Background(), "artificial intelligence", 2)
	
	assert.NoError(t, err)
	assert.Len(t, results, 2)
	
	// 验证返回的是文档
	for _, doc := range results {
		assert.NotEmpty(t, doc.Content)
	}
}

// TestInMemoryVectorStore_SimilaritySearchWithScore
func TestInMemoryVectorStore_SimilaritySearchWithScore(t *testing.T) {
	emb := embeddings.NewFakeEmbeddings(128)
	store := NewInMemoryVectorStore(emb)
	
	docs := []*loaders.Document{
		loaders.NewDocument("Apple is a fruit", nil),
		loaders.NewDocument("Orange is also a fruit", nil),
		loaders.NewDocument("Car is a vehicle", nil),
	}
	
	_, err := store.AddDocuments(context.Background(), docs)
	require.NoError(t, err)
	
	results, err := store.SimilaritySearchWithScore(context.Background(), "fruit", 2)
	
	assert.NoError(t, err)
	assert.Len(t, results, 2)
	
	// 验证分数存在且按降序排列
	for i, result := range results {
		assert.NotNil(t, result.Document)
		assert.GreaterOrEqual(t, result.Score, float32(0))
		assert.LessOrEqual(t, result.Score, float32(1))
		
		if i > 0 {
			// 分数应该递减
			assert.GreaterOrEqual(t, results[i-1].Score, result.Score)
		}
	}
}

// TestInMemoryVectorStore_Delete
func TestInMemoryVectorStore_Delete(t *testing.T) {
	emb := embeddings.NewFakeEmbeddings(128)
	store := NewInMemoryVectorStore(emb)
	
	docs := []*loaders.Document{
		loaders.NewDocument("Doc 1", nil),
		loaders.NewDocument("Doc 2", nil),
		loaders.NewDocument("Doc 3", nil),
	}
	
	ids, err := store.AddDocuments(context.Background(), docs)
	require.NoError(t, err)
	require.Equal(t, 3, store.GetDocumentCount())
	
	// 删除一个文档
	err = store.Delete(context.Background(), []string{ids[0]})
	
	assert.NoError(t, err)
	assert.Equal(t, 2, store.GetDocumentCount())
}

// TestInMemoryVectorStore_Clear
func TestInMemoryVectorStore_Clear(t *testing.T) {
	emb := embeddings.NewFakeEmbeddings(128)
	store := NewInMemoryVectorStore(emb)
	
	docs := []*loaders.Document{
		loaders.NewDocument("Doc 1", nil),
		loaders.NewDocument("Doc 2", nil),
	}
	
	_, err := store.AddDocuments(context.Background(), docs)
	require.NoError(t, err)
	require.Equal(t, 2, store.GetDocumentCount())
	
	store.Clear()
	
	assert.Equal(t, 0, store.GetDocumentCount())
}

// TestInMemoryVectorStore_EmptySearch
func TestInMemoryVectorStore_EmptySearch(t *testing.T) {
	emb := embeddings.NewFakeEmbeddings(128)
	store := NewInMemoryVectorStore(emb)
	
	// 在空存储中搜索
	results, err := store.SimilaritySearch(context.Background(), "query", 5)
	
	assert.NoError(t, err)
	assert.Empty(t, results)
}

// TestInMemoryVectorStore_AddEmptyDocuments
func TestInMemoryVectorStore_AddEmptyDocuments(t *testing.T) {
	emb := embeddings.NewFakeEmbeddings(128)
	store := NewInMemoryVectorStore(emb)
	
	ids, err := store.AddDocuments(context.Background(), []*loaders.Document{})
	
	assert.NoError(t, err)
	assert.Empty(t, ids)
}

// TestCosineSimilarity
func TestCosineSimilarity(t *testing.T) {
	tests := []struct {
		name     string
		a        []float32
		b        []float32
		expected float32
	}{
		{
			name:     "Identical vectors",
			a:        []float32{1, 0, 0},
			b:        []float32{1, 0, 0},
			expected: 1.0,
		},
		{
			name:     "Orthogonal vectors",
			a:        []float32{1, 0, 0},
			b:        []float32{0, 1, 0},
			expected: 0.0,
		},
		{
			name:     "Similar vectors",
			a:        []float32{1, 1, 0},
			b:        []float32{1, 0.5, 0},
			expected: 0.9486833, // 约等于
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cosineSimilarity(tt.a, tt.b)
			assert.InDelta(t, tt.expected, result, 0.01)
		})
	}
}

// TestCosineSimilarity_DifferentLengths
func TestCosineSimilarity_DifferentLengths(t *testing.T) {
	a := []float32{1, 2, 3}
	b := []float32{1, 2}
	
	result := cosineSimilarity(a, b)
	assert.Equal(t, float32(0), result)
}

// TestInMemoryVectorStore_ConcurrentAccess
func TestInMemoryVectorStore_ConcurrentAccess(t *testing.T) {
	emb := embeddings.NewFakeEmbeddings(128)
	store := NewInMemoryVectorStore(emb)
	
	// 并发添加文档
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(idx int) {
			docs := []*loaders.Document{
				loaders.NewDocument("Concurrent doc", map[string]any{"idx": idx}),
			}
			_, _ = store.AddDocuments(context.Background(), docs)
			done <- true
		}(i)
	}
	
	// 等待所有 goroutine 完成
	for i := 0; i < 10; i++ {
		<-done
	}
	
	assert.Equal(t, 10, store.GetDocumentCount())
}

// Benchmark 测试
func BenchmarkInMemoryVectorStore_AddDocuments(b *testing.B) {
	emb := embeddings.NewFakeEmbeddings(1536)
	store := NewInMemoryVectorStore(emb)
	
	docs := []*loaders.Document{
		loaders.NewDocument("Test document 1", nil),
		loaders.NewDocument("Test document 2", nil),
		loaders.NewDocument("Test document 3", nil),
	}
	ctx := context.Background()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store.Clear()
		_, _ = store.AddDocuments(ctx, docs)
	}
}

func BenchmarkInMemoryVectorStore_SimilaritySearch(b *testing.B) {
	emb := embeddings.NewFakeEmbeddings(1536)
	store := NewInMemoryVectorStore(emb)
	
	// 添加 100 个文档
	docs := make([]*loaders.Document, 100)
	for i := 0; i < 100; i++ {
		docs[i] = loaders.NewDocument("Document content", map[string]any{"id": i})
	}
	ctx := context.Background()
	_, _ = store.AddDocuments(ctx, docs)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = store.SimilaritySearch(ctx, "query", 10)
	}
}
