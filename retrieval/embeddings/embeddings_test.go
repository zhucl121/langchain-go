package embeddings

import (
	"context"
	"testing"
	
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewBaseEmbeddings
func TestNewBaseEmbeddings(t *testing.T) {
	embeddings := NewBaseEmbeddings("test-model", 1536)
	
	assert.Equal(t, 1536, embeddings.GetDimension())
	assert.Equal(t, "test-model", embeddings.GetModelName())
}

// TestFakeEmbeddings_EmbedDocuments
func TestFakeEmbeddings_EmbedDocuments(t *testing.T) {
	embeddings := NewFakeEmbeddings(128)
	
	texts := []string{"hello", "worlds", "testing"} // 不同长度
	vectors, err := embeddings.EmbedDocuments(context.Background(), texts)
	
	assert.NoError(t, err)
	assert.Len(t, vectors, 3)
	
	// 检查维度
	for _, vector := range vectors {
		assert.Len(t, vector, 128)
	}
	
	// 检查向量不同（基于文本长度）
	assert.NotEqual(t, vectors[0], vectors[1])
	assert.NotEqual(t, vectors[1], vectors[2])
}

// TestFakeEmbeddings_EmbedQuery
func TestFakeEmbeddings_EmbedQuery(t *testing.T) {
	embeddings := NewFakeEmbeddings(64)
	
	vector, err := embeddings.EmbedQuery(context.Background(), "test query")
	
	assert.NoError(t, err)
	assert.Len(t, vector, 64)
}

// TestFakeEmbeddings_Deterministic
func TestFakeEmbeddings_Deterministic(t *testing.T) {
	embeddings := NewFakeEmbeddings(128)
	
	text := "same text"
	
	// 多次嵌入相同文本应该得到相同结果
	vector1, err := embeddings.EmbedQuery(context.Background(), text)
	require.NoError(t, err)
	
	vector2, err := embeddings.EmbedQuery(context.Background(), text)
	require.NoError(t, err)
	
	assert.Equal(t, vector1, vector2)
}

// TestCachedEmbeddings
func TestCachedEmbeddings(t *testing.T) {
	underlying := NewFakeEmbeddings(128)
	cached := NewCachedEmbeddings(underlying)
	
	texts := []string{"text1", "text2", "text1"} // text1 重复
	
	vectors, err := cached.EmbedDocuments(context.Background(), texts)
	
	assert.NoError(t, err)
	assert.Len(t, vectors, 3)
	
	// text1 的两次结果应该相同（来自缓存）
	assert.Equal(t, vectors[0], vectors[2])
}

// TestCachedEmbeddings_Query
func TestCachedEmbeddings_Query(t *testing.T) {
	underlying := NewFakeEmbeddings(128)
	cached := NewCachedEmbeddings(underlying)
	
	text := "cached query"
	
	// 第一次调用
	vector1, err := cached.EmbedQuery(context.Background(), text)
	require.NoError(t, err)
	
	// 第二次调用（应该从缓存返回）
	vector2, err := cached.EmbedQuery(context.Background(), text)
	require.NoError(t, err)
	
	assert.Equal(t, vector1, vector2)
}

// TestCachedEmbeddings_ClearCache
func TestCachedEmbeddings_ClearCache(t *testing.T) {
	underlying := NewFakeEmbeddings(128)
	cached := NewCachedEmbeddings(underlying)
	
	text := "test"
	
	// 嵌入文本
	_, err := cached.EmbedQuery(context.Background(), text)
	require.NoError(t, err)
	
	// 清空缓存
	cached.ClearCache()
	
	// 缓存应该为空
	assert.Len(t, cached.cache, 0)
}

// TestCachedEmbeddings_GetDimension
func TestCachedEmbeddings_GetDimension(t *testing.T) {
	underlying := NewFakeEmbeddings(256)
	cached := NewCachedEmbeddings(underlying)
	
	assert.Equal(t, 256, cached.GetDimension())
}

// TestOpenAIEmbeddings_Configuration
func TestOpenAIEmbeddings_Configuration(t *testing.T) {
	// 测试默认配置
	embeddings1 := NewOpenAIEmbeddings(OpenAIEmbeddingsConfig{
		APIKey: "test-key",
	})
	
	assert.Equal(t, "text-embedding-ada-002", embeddings1.model)
	assert.Equal(t, 1536, embeddings1.GetDimension())
	assert.Equal(t, "https://api.openai.com/v1", embeddings1.baseURL)
	
	// 测试自定义配置
	embeddings2 := NewOpenAIEmbeddings(OpenAIEmbeddingsConfig{
		APIKey:  "test-key",
		Model:   "text-embedding-3-large",
		BaseURL: "https://custom.api.com",
	})
	
	assert.Equal(t, "text-embedding-3-large", embeddings2.model)
	assert.Equal(t, 3072, embeddings2.GetDimension())
	assert.Equal(t, "https://custom.api.com", embeddings2.baseURL)
}

// TestFakeEmbeddings_EmptyInput
func TestFakeEmbeddings_EmptyInput(t *testing.T) {
	embeddings := NewFakeEmbeddings(128)
	
	vectors, err := embeddings.EmbedDocuments(context.Background(), []string{})
	
	assert.NoError(t, err)
	assert.Empty(t, vectors)
}

// Benchmark 测试
func BenchmarkFakeEmbeddings_EmbedDocuments(b *testing.B) {
	embeddings := NewFakeEmbeddings(1536)
	texts := []string{"text1", "text2", "text3", "text4", "text5"}
	ctx := context.Background()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = embeddings.EmbedDocuments(ctx, texts)
	}
}

func BenchmarkCachedEmbeddings_HitRate(b *testing.B) {
	underlying := NewFakeEmbeddings(1536)
	cached := NewCachedEmbeddings(underlying)
	texts := []string{"text1", "text2", "text3"} // 重复使用相同文本
	ctx := context.Background()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = cached.EmbedDocuments(ctx, texts)
	}
}
