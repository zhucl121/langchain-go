package hybrid

import (
	"context"
	"fmt"
	"testing"

	"github.com/zhucl121/langchain-go/pkg/types"
	"github.com/zhucl121/langchain-go/retrieval/loaders"
	"github.com/zhucl121/langchain-go/retrieval/retrievers/fusion"
	"github.com/zhucl121/langchain-go/retrieval/retrievers/keyword"
	"github.com/zhucl121/langchain-go/retrieval/vectorstores"
)

// MockVectorStore 模拟向量存储
type MockVectorStore struct {
	documents []types.Document
	scores    []float64
	err       error
}

func (m *MockVectorStore) SimilaritySearch(ctx context.Context, query string, k int) ([]*loaders.Document, error) {
	if m.err != nil {
		return nil, m.err
	}
	if k > len(m.documents) {
		k = len(m.documents)
	}
	
	docs := make([]*loaders.Document, k)
	for i := 0; i < k; i++ {
		docs[i] = &loaders.Document{
			Content:  m.documents[i].Content,
			Metadata: m.documents[i].Metadata,
		}
	}
	return docs, nil
}

func (m *MockVectorStore) SimilaritySearchWithScore(ctx context.Context, query string, k int) ([]vectorstores.DocumentWithScore, error) {
	if m.err != nil {
		return nil, m.err
	}
	if k > len(m.documents) {
		k = len(m.documents)
	}
	
	results := make([]vectorstores.DocumentWithScore, k)
	for i := 0; i < k; i++ {
		results[i] = vectorstores.DocumentWithScore{
			Document: &loaders.Document{
				Content:  m.documents[i].Content,
				Metadata: m.documents[i].Metadata,
			},
			Score: float32(m.scores[i]),
		}
	}
	return results, nil
}

func (m *MockVectorStore) AddDocuments(ctx context.Context, documents []*loaders.Document) ([]string, error) {
	if m.err != nil {
		return nil, m.err
	}
	
	ids := make([]string, len(documents))
	for i, doc := range documents {
		m.documents = append(m.documents, types.Document{
			Content:  doc.Content,
			Metadata: doc.Metadata,
		})
		ids[i] = fmt.Sprintf("doc-%d", i)
	}
	return ids, nil
}

func (m *MockVectorStore) Delete(ctx context.Context, ids []string) error {
	return m.err
}

func TestNewHybridRetriever_Success(t *testing.T) {
	docs := []types.Document{
		{Content: "Go programming language"},
		{Content: "Python programming language"},
	}

	mockStore := &MockVectorStore{
		documents: docs,
		scores:    []float64{0.9, 0.8},
	}

	retriever, err := NewHybridRetriever(Config{
		VectorStore: mockStore,
		Documents:   docs,
	})

	if err != nil {
		t.Fatalf("Failed to create retriever: %v", err)
	}

	if retriever == nil {
		t.Fatal("Expected non-nil retriever")
	}

	stats := retriever.GetStats()
	t.Logf("Retriever stats: %+v", stats)
}

func TestNewHybridRetriever_MissingVectorStore(t *testing.T) {
	docs := []types.Document{{Content: "test"}}

	_, err := NewHybridRetriever(Config{
		Documents: docs,
	})

	if err == nil {
		t.Error("Expected error for missing VectorStore")
	}
}

func TestNewHybridRetriever_MissingDocuments(t *testing.T) {
	mockStore := &MockVectorStore{}

	_, err := NewHybridRetriever(Config{
		VectorStore: mockStore,
	})

	if err == nil {
		t.Error("Expected error for missing Documents")
	}
}

func TestHybridRetriever_Search(t *testing.T) {
	docs := []types.Document{
		{Content: "Go is a programming language designed for simplicity"},
		{Content: "Python is a high-level programming language"},
		{Content: "JavaScript is the language of the web"},
		{Content: "Rust is a systems programming language"},
	}

	// Mock 向量存储返回前2个文档
	mockStore := &MockVectorStore{
		documents: []types.Document{docs[0], docs[1]},
		scores:    []float64{0.95, 0.85},
	}

	retriever, err := NewHybridRetriever(Config{
		VectorStore: mockStore,
		Documents:   docs,
		Strategy:    fusion.NewRRFStrategy(60),
	})

	if err != nil {
		t.Fatalf("Failed to create retriever: %v", err)
	}

	ctx := context.Background()
	results, err := retriever.Search(ctx, "programming language", 3)

	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("Expected non-empty results")
	}

	t.Logf("Found %d results:", len(results))
	for i, result := range results {
		t.Logf("  %d. %s", i+1, result.Document.Content)
		t.Logf("     Score: %.4f (Vector: %.4f, Keyword: %.4f)",
			result.Score, result.VectorScore, result.KeywordScore)
		t.Logf("     Ranks: Vector=%d, Keyword=%d", result.VectorRank, result.KeywordRank)
	}

	// 验证分数递减
	for i := 1; i < len(results); i++ {
		if results[i].Score > results[i-1].Score {
			t.Errorf("Results not sorted by score")
		}
	}
}

func TestHybridRetriever_SearchVectorOnly(t *testing.T) {
	docs := []types.Document{
		{Content: "test document one"},
		{Content: "test document two"},
	}

	mockStore := &MockVectorStore{
		documents: docs,
		scores:    []float64{0.9, 0.8},
	}

	retriever, err := NewHybridRetriever(Config{
		VectorStore: mockStore,
		Documents:   docs,
	})

	if err != nil {
		t.Fatalf("Failed to create retriever: %v", err)
	}

	ctx := context.Background()
	results, err := retriever.SearchVectorOnly(ctx, "test", 2)

	if err != nil {
		t.Fatalf("Vector search failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}

	// 验证向量分数（允许微小误差）
	for i, result := range results {
		expected := mockStore.scores[i]
		if result.VectorScore < expected-0.01 || result.VectorScore > expected+0.01 {
			t.Errorf("Expected vector score ~%.2f, got %.2f",
				expected, result.VectorScore)
		}
	}
}

func TestHybridRetriever_SearchKeywordOnly(t *testing.T) {
	docs := []types.Document{
		{Content: "Go programming language"},
		{Content: "Python programming language"},
		{Content: "JavaScript web development"},
	}

	mockStore := &MockVectorStore{}

	retriever, err := NewHybridRetriever(Config{
		VectorStore: mockStore,
		Documents:   docs,
	})

	if err != nil {
		t.Fatalf("Failed to create retriever: %v", err)
	}

	ctx := context.Background()
	results, err := retriever.SearchKeywordOnly(ctx, "programming", 2)

	if err != nil {
		t.Fatalf("Keyword search failed: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("Expected non-empty results")
	}

	t.Logf("Keyword-only results:")
	for i, result := range results {
		t.Logf("  %d. %s (score: %.4f)", i+1, result.Document.Content, result.KeywordScore)
	}
}

func TestHybridRetriever_AddDocuments(t *testing.T) {
	initialDocs := []types.Document{
		{Content: "initial document"},
	}

	mockStore := &MockVectorStore{
		documents: initialDocs,
		scores:    []float64{0.9},
	}

	retriever, err := NewHybridRetriever(Config{
		VectorStore: mockStore,
		Documents:   initialDocs,
	})

	if err != nil {
		t.Fatalf("Failed to create retriever: %v", err)
	}

	// 添加新文档
	newDocs := []types.Document{
		{Content: "new document one"},
		{Content: "new document two"},
	}

	ctx := context.Background()
	err = retriever.AddDocuments(ctx, newDocs)

	if err != nil {
		t.Fatalf("AddDocuments failed: %v", err)
	}

	// 验证文档数量
	stats := retriever.GetStats()
	bm25Stats := stats["bm25_stats"].(map[string]any)
	totalDocs := bm25Stats["total_docs"].(int)

	if totalDocs != 3 {
		t.Errorf("Expected 3 documents, got %d", totalDocs)
	}
}

func TestHybridRetriever_MinScore(t *testing.T) {
	docs := []types.Document{
		{Content: "relevant document"},
		{Content: "less relevant document"},
	}

	mockStore := &MockVectorStore{
		documents: docs,
		scores:    []float64{0.9, 0.3},
	}

	retriever, err := NewHybridRetriever(Config{
		VectorStore: mockStore,
		Documents:   docs,
		MinScore:    0.5, // 设置最小分数阈值
	})

	if err != nil {
		t.Fatalf("Failed to create retriever: %v", err)
	}

	ctx := context.Background()
	results, err := retriever.Search(ctx, "relevant", 10)

	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	// 验证所有结果的分数都高于阈值
	for _, result := range results {
		if result.Score < 0.5 {
			t.Errorf("Result score %.4f is below threshold 0.5", result.Score)
		}
	}

	t.Logf("Filtered results: %d documents above threshold", len(results))
}

func TestHybridRetriever_CustomStrategy(t *testing.T) {
	docs := []types.Document{
		{Content: "test document"},
	}

	mockStore := &MockVectorStore{
		documents: docs,
		scores:    []float64{0.9},
	}

	// 使用加权策略
	weightedStrategy := fusion.NewWeightedStrategy(map[string]float64{
		"vector":  0.8,
		"keyword": 0.2,
	})

	retriever, err := NewHybridRetriever(Config{
		VectorStore: mockStore,
		Documents:   docs,
		Strategy:    weightedStrategy,
	})

	if err != nil {
		t.Fatalf("Failed to create retriever: %v", err)
	}

	stats := retriever.GetStats()
	strategyType := stats["strategy"].(string)

	if strategyType != "*fusion.WeightedStrategy" {
		t.Errorf("Expected WeightedStrategy, got %s", strategyType)
	}
}

func TestHybridRetriever_CustomBM25Config(t *testing.T) {
	docs := []types.Document{
		{Content: "test document"},
	}

	mockStore := &MockVectorStore{
		documents: docs,
		scores:    []float64{0.9},
	}

	// 自定义 BM25 配置
	bm25Config := keyword.BM25Config{
		K1:        2.0,
		B:         0.5,
		Tokenizer: keyword.NewWhitespaceTokenizer(),
	}

	retriever, err := NewHybridRetriever(Config{
		VectorStore: mockStore,
		Documents:   docs,
		BM25Config:  bm25Config,
	})

	if err != nil {
		t.Fatalf("Failed to create retriever: %v", err)
	}

	if retriever == nil {
		t.Fatal("Expected non-nil retriever")
	}
}

func TestHybridRetriever_VectorError(t *testing.T) {
	docs := []types.Document{{Content: "test"}}

	mockStore := &MockVectorStore{
		err: fmt.Errorf("vector store error"),
	}

	retriever, err := NewHybridRetriever(Config{
		VectorStore: mockStore,
		Documents:   docs,
	})

	if err != nil {
		t.Fatalf("Failed to create retriever: %v", err)
	}

	ctx := context.Background()
	_, err = retriever.Search(ctx, "test", 5)

	if err == nil {
		t.Error("Expected error from vector store")
	}

	t.Logf("Got expected error: %v", err)
}

func BenchmarkHybridRetriever_Search(b *testing.B) {
	docs := make([]types.Document, 100)
	for i := 0; i < 100; i++ {
		docs[i] = types.Document{
			Content: fmt.Sprintf("This is document number %d about programming languages", i),
		}
	}

	mockStore := &MockVectorStore{
		documents: docs[:10],
		scores:    []float64{0.95, 0.9, 0.85, 0.8, 0.75, 0.7, 0.65, 0.6, 0.55, 0.5},
	}

	retriever, err := NewHybridRetriever(Config{
		VectorStore: mockStore,
		Documents:   docs,
	})

	if err != nil {
		b.Fatalf("Failed to create retriever: %v", err)
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = retriever.Search(ctx, "programming language", 10)
	}
}
