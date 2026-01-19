package keyword

import (
	"context"
	"testing"

	"github.com/zhucl121/langchain-go/pkg/types"
)

func TestBM25Retriever_Basic(t *testing.T) {
	// 准备测试文档
	docs := []types.Document{
		{Content: "Go is a programming language designed for building simple reliable and efficient software"},
		{Content: "Python is a high-level programming language"},
		{Content: "JavaScript is the programming language of the Web"},
		{Content: "Rust is a systems programming language"},
		{Content: "Java is a popular programming language"},
	}

	// 创建检索器
	retriever := NewBM25Retriever(docs, DefaultBM25Config())

	// 测试搜索
	ctx := context.Background()
	results, err := retriever.Search(ctx, "programming language", 3)

	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	// 验证结果
	if len(results) == 0 {
		t.Fatal("Expected results, got none")
	}

	if len(results) > 3 {
		t.Errorf("Expected at most 3 results, got %d", len(results))
	}

	// 验证分数递减
	for i := 1; i < len(results); i++ {
		if results[i].Score > results[i-1].Score {
			t.Errorf("Results not sorted by score: %f > %f", results[i].Score, results[i-1].Score)
		}
	}

	t.Logf("Top result: %s (score: %.4f)", results[0].Document.Content, results[0].Score)
}

func TestBM25Retriever_EmptyQuery(t *testing.T) {
	docs := []types.Document{
		{Content: "test document"},
	}

	retriever := NewBM25Retriever(docs, DefaultBM25Config())
	ctx := context.Background()

	results, err := retriever.Search(ctx, "", 5)

	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("Expected 0 results for empty query, got %d", len(results))
	}
}

func TestBM25Retriever_NoMatch(t *testing.T) {
	docs := []types.Document{
		{Content: "Go programming language"},
		{Content: "Python programming language"},
	}

	retriever := NewBM25Retriever(docs, DefaultBM25Config())
	ctx := context.Background()

	results, err := retriever.Search(ctx, "javascript typescript", 5)

	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("Expected 0 results for non-matching query, got %d", len(results))
	}
}

func TestBM25Retriever_AddDocuments(t *testing.T) {
	docs := []types.Document{
		{Content: "initial document"},
	}

	retriever := NewBM25Retriever(docs, DefaultBM25Config())

	if retriever.GetDocumentCount() != 1 {
		t.Errorf("Expected 1 document, got %d", retriever.GetDocumentCount())
	}

	// 添加更多文档
	newDocs := []types.Document{
		{Content: "additional document one"},
		{Content: "additional document two"},
	}

	retriever.AddDocuments(newDocs)

	if retriever.GetDocumentCount() != 3 {
		t.Errorf("Expected 3 documents, got %d", retriever.GetDocumentCount())
	}
}

func TestBM25Retriever_IDFCalculation(t *testing.T) {
	docs := []types.Document{
		{Content: "common word document one"},
		{Content: "common word document two"},
		{Content: "rare word document three"},
	}

	retriever := NewBM25Retriever(docs, DefaultBM25Config())
	ctx := context.Background()

	// "rare" 应该比 "common" 的 IDF 更高
	results, _ := retriever.Search(ctx, "rare", 3)

	if len(results) == 0 {
		t.Fatal("Expected results for 'rare'")
	}

	rareScore := results[0].Score

	results, _ = retriever.Search(ctx, "common", 3)

	if len(results) == 0 {
		t.Fatal("Expected results for 'common'")
	}

	commonScore := results[0].Score

	// rare 的分数应该更高（因为更稀有）
	if rareScore <= commonScore {
		t.Errorf("Expected rare term to have higher score than common term: rare=%.4f, common=%.4f",
			rareScore, commonScore)
	}
}

func TestBM25Retriever_Stats(t *testing.T) {
	docs := []types.Document{
		{Content: "short doc"},
		{Content: "this is a longer document with more words"},
		{Content: "medium length document"},
	}

	retriever := NewBM25Retriever(docs, DefaultBM25Config())
	stats := retriever.GetIndexStats()

	if stats["total_docs"] != 3 {
		t.Errorf("Expected 3 total docs, got %v", stats["total_docs"])
	}

	if stats["unique_terms"].(int) == 0 {
		t.Error("Expected non-zero unique terms")
	}

	t.Logf("Index stats: %+v", stats)
}

func TestBM25Config_CustomParameters(t *testing.T) {
	docs := []types.Document{
		{Content: "test document one"},
		{Content: "test document two"},
	}

	// 自定义配置
	config := BM25Config{
		K1:        2.0,
		B:         0.5,
		Tokenizer: NewWhitespaceTokenizer(),
	}

	retriever := NewBM25Retriever(docs, config)
	ctx := context.Background()

	results, err := retriever.Search(ctx, "test", 2)

	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}
}

func BenchmarkBM25Search(b *testing.B) {
	// 准备大量文档
	docs := make([]types.Document, 1000)
	for i := 0; i < 1000; i++ {
		docs[i] = types.Document{
			Content: "This is a test document for benchmarking the BM25 search algorithm performance",
		}
	}

	retriever := NewBM25Retriever(docs, DefaultBM25Config())
	ctx := context.Background()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = retriever.Search(ctx, "test document search", 10)
	}
}

func BenchmarkBM25IndexBuilding(b *testing.B) {
	docs := make([]types.Document, 100)
	for i := 0; i < 100; i++ {
		docs[i] = types.Document{
			Content: "Sample document content for index building benchmark",
		}
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = NewBM25Retriever(docs, DefaultBM25Config())
	}
}
