package hybrid

import (
	"context"
	"testing"

	"github.com/zhucl121/langchain-go/pkg/types"
	"github.com/zhucl121/langchain-go/retrieval/loaders"
	"github.com/zhucl121/langchain-go/retrieval/retrievers/fusion"
	"github.com/zhucl121/langchain-go/retrieval/vectorstores"
)

// MockMilvusVectorStore 模拟 Milvus 向量存储
type MockMilvusVectorStore struct {
	vectorResults []vectorstores.DocumentWithScore
	hybridResults []vectorstores.HybridSearchResult
	err           error
}

func (m *MockMilvusVectorStore) SimilaritySearchWithScore(ctx context.Context, query string, k int) ([]vectorstores.DocumentWithScore, error) {
	if m.err != nil {
		return nil, m.err
	}
	if k > len(m.vectorResults) {
		k = len(m.vectorResults)
	}
	return m.vectorResults[:k], nil
}

func (m *MockMilvusVectorStore) HybridSearch(ctx context.Context, query string, k int, opts *vectorstores.HybridSearchOptions) ([]vectorstores.HybridSearchResult, error) {
	if m.err != nil {
		return nil, m.err
	}
	if k > len(m.hybridResults) {
		k = len(m.hybridResults)
	}
	return m.hybridResults[:k], nil
}

func (m *MockMilvusVectorStore) AddDocuments(ctx context.Context, documents []*loaders.Document) ([]string, error) {
	if m.err != nil {
		return nil, m.err
	}
	ids := make([]string, len(documents))
	for i := range documents {
		ids[i] = "mock-id"
	}
	return ids, nil
}

func TestNewMilvusHybridRetriever(t *testing.T) {
	mockStore := &MockMilvusVectorStore{}
	strategy := fusion.NewRRFStrategy(60)

	retriever := NewMilvusHybridRetriever(mockStore, strategy)

	if retriever == nil {
		t.Fatal("Expected non-nil retriever")
	}

	if retriever.config.RRFRankConstant != 60 {
		t.Errorf("Expected RRF constant 60, got %d", retriever.config.RRFRankConstant)
	}

	stats := retriever.GetStats()
	t.Logf("Retriever stats: %+v", stats)
}

func TestNewMilvusHybridRetrieverWithConfig(t *testing.T) {
	mockStore := &MockMilvusVectorStore{}

	config := MilvusHybridConfig{
		RRFRankConstant: 30,
		UseNativeRRF:    true,
		MinScore:        0.5,
	}

	retriever := NewMilvusHybridRetrieverWithConfig(mockStore, config)

	if retriever.config.RRFRankConstant != 30 {
		t.Errorf("Expected RRF constant 30, got %d", retriever.config.RRFRankConstant)
	}

	if retriever.config.MinScore != 0.5 {
		t.Errorf("Expected MinScore 0.5, got %.2f", retriever.config.MinScore)
	}
}

func TestMilvusHybridRetriever_Search(t *testing.T) {
	mockStore := &MockMilvusVectorStore{
		hybridResults: []vectorstores.HybridSearchResult{
			{
				Document: &loaders.Document{
					Content: "Go programming language",
				},
				VectorScore:  0.95,
				KeywordScore: 0.8,
				FusionScore:  0.88,
			},
			{
				Document: &loaders.Document{
					Content: "Python programming language",
				},
				VectorScore:  0.85,
				KeywordScore: 0.7,
				FusionScore:  0.78,
			},
		},
	}

	retriever := NewMilvusHybridRetriever(mockStore, fusion.NewRRFStrategy(60))

	ctx := context.Background()
	results, err := retriever.Search(ctx, "programming", 2)

	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}

	// 验证第一个结果
	if results[0].Document.Content != "Go programming language" {
		t.Errorf("Unexpected first result: %s", results[0].Document.Content)
	}

	// 验证分数
	expectedScore := 0.88
	if results[0].Score < expectedScore-0.01 || results[0].Score > expectedScore+0.01 {
		t.Errorf("Expected score ~%.2f, got %.2f", expectedScore, results[0].Score)
	}

	t.Logf("Search results:")
	for i, result := range results {
		t.Logf("  %d. %s", i+1, result.Document.Content)
		t.Logf("     Fusion: %.2f, Vector: %.2f, Keyword: %.2f",
			result.Score, result.VectorScore, result.KeywordScore)
	}
}

func TestMilvusHybridRetriever_SearchVectorOnly(t *testing.T) {
	mockStore := &MockMilvusVectorStore{
		vectorResults: []vectorstores.DocumentWithScore{
			{
				Document: &loaders.Document{
					Content: "test document",
				},
				Score: 0.9,
			},
		},
	}

	retriever := NewMilvusHybridRetriever(mockStore, fusion.NewRRFStrategy(60))

	ctx := context.Background()
	results, err := retriever.SearchVectorOnly(ctx, "test", 1)

	if err != nil {
		t.Fatalf("SearchVectorOnly failed: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}

	expectedScore := 0.9
	if results[0].VectorScore < expectedScore-0.01 || results[0].VectorScore > expectedScore+0.01 {
		t.Errorf("Expected vector score ~%.2f, got %.2f", expectedScore, results[0].VectorScore)
	}
}

func TestMilvusHybridRetriever_MinScoreFilter(t *testing.T) {
	mockStore := &MockMilvusVectorStore{
		hybridResults: []vectorstores.HybridSearchResult{
			{
				Document: &loaders.Document{
					Content: "high score doc",
				},
				FusionScore: 0.9,
			},
			{
				Document: &loaders.Document{
					Content: "low score doc",
				},
				FusionScore: 0.3,
			},
		},
	}

	config := MilvusHybridConfig{
		RRFRankConstant: 60,
		MinScore:        0.5, // 过滤低分结果
	}

	retriever := NewMilvusHybridRetrieverWithConfig(mockStore, config)

	ctx := context.Background()
	results, err := retriever.Search(ctx, "test", 10)

	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	// 应该只有一个结果（高分文档）
	if len(results) != 1 {
		t.Errorf("Expected 1 result after filtering, got %d", len(results))
	}

	if results[0].Document.Content != "high score doc" {
		t.Errorf("Unexpected result: %s", results[0].Document.Content)
	}
}

func TestMilvusHybridRetriever_AddDocuments(t *testing.T) {
	mockStore := &MockMilvusVectorStore{}

	retriever := NewMilvusHybridRetriever(mockStore, fusion.NewRRFStrategy(60))

	docs := []types.Document{
		{Content: "new document 1"},
		{Content: "new document 2"},
	}

	ctx := context.Background()
	err := retriever.AddDocuments(ctx, docs)

	if err != nil {
		t.Errorf("AddDocuments failed: %v", err)
	}
}

func TestMilvusHybridRetriever_ChainedConfig(t *testing.T) {
	mockStore := &MockMilvusVectorStore{
		hybridResults: []vectorstores.HybridSearchResult{
			{
				Document: &loaders.Document{
					Content: "test",
				},
				FusionScore: 0.8,
			},
		},
	}

	retriever := NewMilvusHybridRetriever(mockStore, fusion.NewRRFStrategy(60)).
		WithMinScore(0.5).
		WithRRFConstant(30)

	if retriever.config.MinScore != 0.5 {
		t.Errorf("Expected MinScore 0.5, got %.2f", retriever.config.MinScore)
	}

	if retriever.config.RRFRankConstant != 30 {
		t.Errorf("Expected RRF constant 30, got %d", retriever.config.RRFRankConstant)
	}

	ctx := context.Background()
	results, err := retriever.Search(ctx, "test", 1)

	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}
}

func TestMilvusHybridRetriever_SearchWithCustomStrategy(t *testing.T) {
	mockStore := &MockMilvusVectorStore{
		vectorResults: []vectorstores.DocumentWithScore{
			{
				Document: &loaders.Document{
					Content: "test document",
				},
				Score: 0.9,
			},
		},
	}

	retriever := NewMilvusHybridRetriever(mockStore, fusion.NewRRFStrategy(60))

	// 使用自定义加权策略
	customStrategy := fusion.NewWeightedStrategy(map[string]float64{
		"vector": 1.0,
	})

	ctx := context.Background()
	results, err := retriever.SearchWithCustomStrategy(ctx, "test", 1, customStrategy)

	if err != nil {
		t.Fatalf("SearchWithCustomStrategy failed: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}
}

func TestMilvusNativeHybridSearch(t *testing.T) {
	mockStore := &MockMilvusVectorStore{
		hybridResults: []vectorstores.HybridSearchResult{
			{
				Document: &loaders.Document{
					Content: "test document",
				},
				FusionScore: 0.85,
			},
		},
	}

	ctx := context.Background()
	results, err := MilvusNativeHybridSearch(ctx, mockStore, "test", 1)

	if err != nil {
		t.Fatalf("MilvusNativeHybridSearch failed: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}

	expectedScore := 0.85
	if results[0].Score < expectedScore-0.01 || results[0].Score > expectedScore+0.01 {
		t.Errorf("Expected score ~%.2f, got %.2f", expectedScore, results[0].Score)
	}
}

func TestDefaultMilvusHybridConfig(t *testing.T) {
	config := DefaultMilvusHybridConfig()

	if config.RRFRankConstant != 60 {
		t.Errorf("Expected default RRF constant 60, got %d", config.RRFRankConstant)
	}

	if !config.UseNativeRRF {
		t.Error("Expected UseNativeRRF to be true by default")
	}

	if config.MinScore != 0 {
		t.Errorf("Expected default MinScore 0, got %.2f", config.MinScore)
	}
}

func BenchmarkMilvusHybridRetriever_Search(b *testing.B) {
	// 准备测试数据
	hybridResults := make([]vectorstores.HybridSearchResult, 100)
	for i := 0; i < 100; i++ {
		hybridResults[i] = vectorstores.HybridSearchResult{
			Document: &loaders.Document{
				Content: "test document",
			},
			VectorScore:  float32(0.9 - float64(i)*0.001),
			KeywordScore: float32(0.8 - float64(i)*0.001),
			FusionScore:  float32(0.85 - float64(i)*0.001),
		}
	}

	mockStore := &MockMilvusVectorStore{
		hybridResults: hybridResults,
	}

	retriever := NewMilvusHybridRetriever(mockStore, fusion.NewRRFStrategy(60))
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = retriever.Search(ctx, "test", 10)
	}
}
