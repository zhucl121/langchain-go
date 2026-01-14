package vectorstores

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"langchain-go/retrieval/embeddings"
	"langchain-go/retrieval/loaders"
)

// 注意: 这些测试需要运行 Milvus 实例
// 可以使用 Docker 启动: docker run -d --name milvus -p 19530:19530 milvusdb/milvus:latest
// 或者跳过这些测试: go test -short

func skipIfShort(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping Milvus integration test in short mode")
	}
}

// TestNewMilvusVectorStore 测试创建 Milvus 存储
func TestNewMilvusVectorStore(t *testing.T) {
	skipIfShort(t)

	emb := embeddings.NewFakeEmbeddings(128)
	config := MilvusConfig{
		Address:              "localhost:19530",
		CollectionName:       "test_collection_" + time.Now().Format("20060102150405"),
		Dimension:            128,
		AutoCreateCollection: true,
	}

	store, err := NewMilvusVectorStore(config, emb)
	require.NoError(t, err)
	require.NotNil(t, store)
	defer store.Close()

	assert.Equal(t, config.CollectionName, store.collectionName)
	assert.Equal(t, 128, store.dimension)
}

// TestMilvusVectorStore_AddDocuments 测试添加文档
func TestMilvusVectorStore_AddDocuments(t *testing.T) {
	skipIfShort(t)

	emb := embeddings.NewFakeEmbeddings(128)
	config := MilvusConfig{
		Address:              "localhost:19530",
		CollectionName:       "test_add_" + time.Now().Format("20060102150405"),
		Dimension:            128,
		AutoCreateCollection: true,
	}

	store, err := NewMilvusVectorStore(config, emb)
	require.NoError(t, err)
	defer store.Close()
	defer store.DropCollection(context.Background())

	docs := []*loaders.Document{
		loaders.NewDocument("First document about AI", map[string]any{"topic": "AI"}),
		loaders.NewDocument("Second document about ML", map[string]any{"topic": "ML"}),
	}

	ids, err := store.AddDocuments(context.Background(), docs)

	assert.NoError(t, err)
	assert.Len(t, ids, 2)

	// 等待索引完成
	time.Sleep(time.Second)

	// 验证文档数量
	count, err := store.GetDocumentCount(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, int64(2), count)
}

// TestMilvusVectorStore_SimilaritySearch 测试相似度搜索
func TestMilvusVectorStore_SimilaritySearch(t *testing.T) {
	skipIfShort(t)

	emb := embeddings.NewFakeEmbeddings(128)
	config := MilvusConfig{
		Address:              "localhost:19530",
		CollectionName:       "test_search_" + time.Now().Format("20060102150405"),
		Dimension:            128,
		AutoCreateCollection: true,
	}

	store, err := NewMilvusVectorStore(config, emb)
	require.NoError(t, err)
	defer store.Close()
	defer store.DropCollection(context.Background())

	docs := []*loaders.Document{
		loaders.NewDocument("Machine learning is a subset of AI", nil),
		loaders.NewDocument("Deep learning uses neural networks", nil),
		loaders.NewDocument("Python is a programming language", nil),
		loaders.NewDocument("Go is fast and efficient for backends", nil),
	}

	_, err = store.AddDocuments(context.Background(), docs)
	require.NoError(t, err)

	// 等待索引完成
	time.Sleep(2 * time.Second)

	// 搜索相似文档
	results, err := store.SimilaritySearch(context.Background(), "artificial intelligence", 2)

	assert.NoError(t, err)
	assert.Len(t, results, 2)

	// 验证返回的是文档
	for _, doc := range results {
		assert.NotEmpty(t, doc.Content)
	}
}

// TestMilvusVectorStore_SimilaritySearchWithScore 测试带分数搜索
func TestMilvusVectorStore_SimilaritySearchWithScore(t *testing.T) {
	skipIfShort(t)

	emb := embeddings.NewFakeEmbeddings(128)
	config := MilvusConfig{
		Address:              "localhost:19530",
		CollectionName:       "test_score_" + time.Now().Format("20060102150405"),
		Dimension:            128,
		AutoCreateCollection: true,
	}

	store, err := NewMilvusVectorStore(config, emb)
	require.NoError(t, err)
	defer store.Close()
	defer store.DropCollection(context.Background())

	docs := []*loaders.Document{
		loaders.NewDocument("Apple is a fruit that grows on trees", nil),
		loaders.NewDocument("Orange is also a citrus fruit", nil),
		loaders.NewDocument("Car is a vehicle for transportation", nil),
	}

	_, err = store.AddDocuments(context.Background(), docs)
	require.NoError(t, err)

	// 等待索引完成
	time.Sleep(2 * time.Second)

	results, err := store.SimilaritySearchWithScore(context.Background(), "fruit", 2)

	assert.NoError(t, err)
	assert.Len(t, results, 2)

	// 验证分数存在
	for i, result := range results {
		assert.NotNil(t, result.Document)
		assert.GreaterOrEqual(t, result.Score, float32(0))

		if i > 0 {
			// 分数应该递减（相似度从高到低）
			assert.GreaterOrEqual(t, results[i-1].Score, result.Score)
		}
	}
}

// TestMilvusVectorStore_Delete 测试删除文档
func TestMilvusVectorStore_Delete(t *testing.T) {
	skipIfShort(t)

	emb := embeddings.NewFakeEmbeddings(128)
	config := MilvusConfig{
		Address:              "localhost:19530",
		CollectionName:       "test_delete_" + time.Now().Format("20060102150405"),
		Dimension:            128,
		AutoCreateCollection: true,
	}

	store, err := NewMilvusVectorStore(config, emb)
	require.NoError(t, err)
	defer store.Close()
	defer store.DropCollection(context.Background())

	docs := []*loaders.Document{
		loaders.NewDocument("Doc 1", nil),
		loaders.NewDocument("Doc 2", nil),
		loaders.NewDocument("Doc 3", nil),
	}

	ids, err := store.AddDocuments(context.Background(), docs)
	require.NoError(t, err)

	// 等待索引完成
	time.Sleep(time.Second)

	count, _ := store.GetDocumentCount(context.Background())
	require.Equal(t, int64(3), count)

	// 删除一个文档
	err = store.Delete(context.Background(), []string{ids[0]})
	assert.NoError(t, err)

	// 等待删除完成
	time.Sleep(time.Second)

	count, _ = store.GetDocumentCount(context.Background())
	assert.Equal(t, int64(2), count)
}

// TestMilvusVectorStore_EmptySearch 测试空存储搜索
func TestMilvusVectorStore_EmptySearch(t *testing.T) {
	skipIfShort(t)

	emb := embeddings.NewFakeEmbeddings(128)
	config := MilvusConfig{
		Address:              "localhost:19530",
		CollectionName:       "test_empty_" + time.Now().Format("20060102150405"),
		Dimension:            128,
		AutoCreateCollection: true,
	}

	store, err := NewMilvusVectorStore(config, emb)
	require.NoError(t, err)
	defer store.Close()
	defer store.DropCollection(context.Background())

	// 在空存储中搜索
	results, err := store.SimilaritySearch(context.Background(), "query", 5)

	assert.NoError(t, err)
	assert.Empty(t, results)
}

// TestMilvusVectorStore_CustomFields 测试自定义字段名
func TestMilvusVectorStore_CustomFields(t *testing.T) {
	skipIfShort(t)

	emb := embeddings.NewFakeEmbeddings(128)
	config := MilvusConfig{
		Address:              "localhost:19530",
		CollectionName:       "test_custom_" + time.Now().Format("20060102150405"),
		Dimension:            128,
		IDField:              "doc_id",
		VectorField:          "embedding",
		ContentField:         "text",
		MetadataField:        "meta",
		AutoCreateCollection: true,
	}

	store, err := NewMilvusVectorStore(config, emb)
	require.NoError(t, err)
	defer store.Close()
	defer store.DropCollection(context.Background())

	assert.Equal(t, "doc_id", store.idField)
	assert.Equal(t, "embedding", store.vectorField)
	assert.Equal(t, "text", store.contentField)
	assert.Equal(t, "meta", store.metadataField)
}

// TestMilvusVectorStore_LargeDataset 测试大规模数据
func TestMilvusVectorStore_LargeDataset(t *testing.T) {
	skipIfShort(t)

	emb := embeddings.NewFakeEmbeddings(128)
	config := MilvusConfig{
		Address:              "localhost:19530",
		CollectionName:       "test_large_" + time.Now().Format("20060102150405"),
		Dimension:            128,
		AutoCreateCollection: true,
	}

	store, err := NewMilvusVectorStore(config, emb)
	require.NoError(t, err)
	defer store.Close()
	defer store.DropCollection(context.Background())

	// 创建 100 个文档
	docs := make([]*loaders.Document, 100)
	for i := 0; i < 100; i++ {
		docs[i] = loaders.NewDocument(
			"Document content number "+string(rune(i)),
			map[string]any{"index": i},
		)
	}

	ids, err := store.AddDocuments(context.Background(), docs)
	require.NoError(t, err)
	assert.Len(t, ids, 100)

	// 等待索引完成
	time.Sleep(3 * time.Second)

	// 搜索
	results, err := store.SimilaritySearch(context.Background(), "content", 10)
	assert.NoError(t, err)
	assert.Len(t, results, 10)
}

// TestDistanceToSimilarity 测试距离转相似度
func TestDistanceToSimilarity(t *testing.T) {
	tests := []struct {
		name       string
		distance   float32
		metricType entity.MetricType
		expected   float32
	}{
		{
			name:       "L2 distance",
			distance:   1.0,
			metricType: entity.L2,
			expected:   0.5, // 1/(1+1) = 0.5
		},
		{
			name:       "Inner Product",
			distance:   0.8,
			metricType: entity.IP,
			expected:   0.8,
		},
		{
			name:       "Cosine",
			distance:   0.95,
			metricType: entity.COSINE,
			expected:   0.95,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := distanceToSimilarity(tt.distance, tt.metricType)
			assert.InDelta(t, tt.expected, result, 0.01)
		})
	}
}

// TestMilvusVectorStore_HybridSearch 测试混合搜索（Milvus 2.6+）
func TestMilvusVectorStore_HybridSearch(t *testing.T) {
	skipIfShort(t)

	emb := embeddings.NewFakeEmbeddings(128)
	config := MilvusConfig{
		Address:              "localhost:19530",
		CollectionName:       "test_hybrid_" + time.Now().Format("20060102150405"),
		Dimension:            128,
		AutoCreateCollection: true,
	}

	store, err := NewMilvusVectorStore(config, emb)
	require.NoError(t, err)
	defer store.Close()
	defer store.DropCollection(context.Background())

	docs := []*loaders.Document{
		loaders.NewDocument("Machine learning is artificial intelligence", nil),
		loaders.NewDocument("Deep learning uses neural networks for AI", nil),
		loaders.NewDocument("Python programming language for data science", nil),
		loaders.NewDocument("Go language is fast and efficient", nil),
	}

	_, err = store.AddDocuments(context.Background(), docs)
	require.NoError(t, err)

	// 等待索引完成
	time.Sleep(2 * time.Second)

	// 执行混合搜索
	options := &HybridSearchOptions{
		VectorWeight:   0.7,
		KeywordWeight:  0.3,
		RerankStrategy: "rrf",
		RRFParam:       60,
	}

	results, err := store.HybridSearch(context.Background(), "artificial intelligence machine learning", 3, options)

	assert.NoError(t, err)
	assert.LessOrEqual(t, len(results), 3)

	// 验证结果包含文档和分数
	for _, result := range results {
		assert.NotNil(t, result.Document)
		assert.Greater(t, result.Score, float32(0))
	}
}

// TestMilvusVectorStore_HybridSearch_Weighted 测试加权融合
func TestMilvusVectorStore_HybridSearch_Weighted(t *testing.T) {
	skipIfShort(t)

	emb := embeddings.NewFakeEmbeddings(128)
	config := MilvusConfig{
		Address:              "localhost:19530",
		CollectionName:       "test_weighted_" + time.Now().Format("20060102150405"),
		Dimension:            128,
		AutoCreateCollection: true,
	}

	store, err := NewMilvusVectorStore(config, emb)
	require.NoError(t, err)
	defer store.Close()
	defer store.DropCollection(context.Background())

	docs := []*loaders.Document{
		loaders.NewDocument("Vector search with embeddings", nil),
		loaders.NewDocument("Keyword search with BM25 algorithm", nil),
		loaders.NewDocument("Hybrid search combines both approaches", nil),
	}

	_, err = store.AddDocuments(context.Background(), docs)
	require.NoError(t, err)
	time.Sleep(2 * time.Second)

	// 加权融合策略
	options := &HybridSearchOptions{
		VectorWeight:   0.8,
		KeywordWeight:  0.2,
		RerankStrategy: "weighted",
	}

	results, err := store.HybridSearch(context.Background(), "search embeddings", 2, options)

	assert.NoError(t, err)
	assert.LessOrEqual(t, len(results), 2)
}

// TestMilvusVectorStore_RerankRRF 测试 RRF 重排序算法
func TestMilvusVectorStore_RerankRRF(t *testing.T) {
	emb := embeddings.NewFakeEmbeddings(128)
	config := MilvusConfig{
		Address:        "localhost:19530",
		CollectionName: "test",
		Dimension:      128,
	}
	store, _ := NewMilvusVectorStore(config, emb)

	vectorResults := []DocumentWithScore{
		{Document: loaders.NewDocument("doc1", nil), Score: 0.9},
		{Document: loaders.NewDocument("doc2", nil), Score: 0.8},
		{Document: loaders.NewDocument("doc3", nil), Score: 0.7},
	}

	keywordResults := []DocumentWithScore{
		{Document: loaders.NewDocument("doc2", nil), Score: 0.95},
		{Document: loaders.NewDocument("doc1", nil), Score: 0.85},
		{Document: loaders.NewDocument("doc4", nil), Score: 0.75},
	}

	results := store.rerankRRF(vectorResults, keywordResults, 60)

	assert.Greater(t, len(results), 0)
	// doc1 和 doc2 应该排名靠前（两个结果集都包含）
	assert.Contains(t, []string{results[0].Document.Content, results[1].Document.Content}, "doc1")
	assert.Contains(t, []string{results[0].Document.Content, results[1].Document.Content}, "doc2")
}

// TestMilvusVectorStore_RerankWeighted 测试加权重排序算法
func TestMilvusVectorStore_RerankWeighted(t *testing.T) {
	emb := embeddings.NewFakeEmbeddings(128)
	config := MilvusConfig{
		Address:        "localhost:19530",
		CollectionName: "test",
		Dimension:      128,
	}
	store, _ := NewMilvusVectorStore(config, emb)

	vectorResults := []DocumentWithScore{
		{Document: loaders.NewDocument("doc1", nil), Score: 0.9},
		{Document: loaders.NewDocument("doc2", nil), Score: 0.5},
	}

	keywordResults := []DocumentWithScore{
		{Document: loaders.NewDocument("doc2", nil), Score: 0.8},
		{Document: loaders.NewDocument("doc3", nil), Score: 0.6},
	}

	results := store.rerankWeighted(vectorResults, keywordResults, 0.7, 0.3)

	assert.Greater(t, len(results), 0)
	// doc1 向量分数高，应该排名靠前
	assert.Equal(t, "doc1", results[0].Document.Content)
}

// Benchmark 测试

func BenchmarkMilvusVectorStore_AddDocuments(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping benchmark in short mode")
	}

	emb := embeddings.NewFakeEmbeddings(1536)
	config := MilvusConfig{
		Address:              "localhost:19530",
		CollectionName:       "bench_add_" + time.Now().Format("20060102150405"),
		Dimension:            1536,
		AutoCreateCollection: true,
	}

	store, err := NewMilvusVectorStore(config, emb)
	if err != nil {
		b.Fatal(err)
	}
	defer store.Close()
	defer store.DropCollection(context.Background())

	docs := []*loaders.Document{
		loaders.NewDocument("Test document 1", nil),
		loaders.NewDocument("Test document 2", nil),
		loaders.NewDocument("Test document 3", nil),
	}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = store.AddDocuments(ctx, docs)
	}
}

func BenchmarkMilvusVectorStore_SimilaritySearch(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping benchmark in short mode")
	}

	emb := embeddings.NewFakeEmbeddings(1536)
	config := MilvusConfig{
		Address:              "localhost:19530",
		CollectionName:       "bench_search_" + time.Now().Format("20060102150405"),
		Dimension:            1536,
		AutoCreateCollection: true,
	}

	store, err := NewMilvusVectorStore(config, emb)
	if err != nil {
		b.Fatal(err)
	}
	defer store.Close()
	defer store.DropCollection(context.Background())

	// 添加 100 个文档
	docs := make([]*loaders.Document, 100)
	for i := 0; i < 100; i++ {
		docs[i] = loaders.NewDocument("Document content", map[string]any{"id": i})
	}
	ctx := context.Background()
	_, _ = store.AddDocuments(ctx, docs)
	time.Sleep(2 * time.Second) // 等待索引

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = store.SimilaritySearch(ctx, "query", 10)
	}
}
