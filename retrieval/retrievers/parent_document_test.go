package retrievers

import (
	"context"
	"testing"

	"github.com/zhucl121/langchain-go/pkg/types"
)

// Mock text splitter
type mockTextSplitter struct {
	chunkSize int
}

func (m *mockTextSplitter) SplitDocuments(documents []types.Document) []types.Document {
	var result []types.Document
	
	for _, doc := range documents {
		content := doc.PageContent
		// 简单地按固定大小分割
		for i := 0; i < len(content); i += m.chunkSize {
			end := i + m.chunkSize
			if end > len(content) {
				end = len(content)
			}
			
			chunk := types.Document{
				PageContent: content[i:end],
				Metadata:    make(map[string]interface{}),
			}
			
			// 复制原始元数据
			for k, v := range doc.Metadata {
				chunk.Metadata[k] = v
			}
			
			result = append(result, chunk)
		}
	}
	
	return result
}

// Mock vector store with add
type mockVectorStoreWithAdd struct {
	docs []types.Document
}

func (m *mockVectorStoreWithAdd) AddDocuments(ctx context.Context, documents []types.Document) error {
	m.docs = append(m.docs, documents...)
	return nil
}

func (m *mockVectorStoreWithAdd) SimilaritySearch(ctx context.Context, query string, k int) ([]types.Document, error) {
	if k > len(m.docs) {
		k = len(m.docs)
	}
	return m.docs[:k], nil
}

func (m *mockVectorStoreWithAdd) SimilaritySearchByVector(ctx context.Context, embedding []float64, k int) ([]types.Document, error) {
	return m.SimilaritySearch(ctx, "", k)
}

func TestNewParentDocumentRetriever(t *testing.T) {
	vectorStore := &mockVectorStoreWithAdd{}
	docStore := NewMemoryDocumentStore()
	childSplitter := &mockTextSplitter{chunkSize: 50}
	
	retriever := NewParentDocumentRetriever(vectorStore, docStore, childSplitter)
	
	if retriever == nil {
		t.Fatal("expected retriever, got nil")
	}
	
	if retriever.config.TopK != 4 {
		t.Errorf("expected TopK 4, got %d", retriever.config.TopK)
	}
}

func TestAddDocuments(t *testing.T) {
	vectorStore := &mockVectorStoreWithAdd{}
	docStore := NewMemoryDocumentStore()
	childSplitter := &mockTextSplitter{chunkSize: 10}
	
	retriever := NewParentDocumentRetriever(vectorStore, docStore, childSplitter)
	
	docs := []types.Document{
		{
			PageContent: "This is a long document that will be split into smaller chunks for indexing.",
			Metadata:    map[string]interface{}{"source": "test"},
		},
	}
	
	ctx := context.Background()
	err := retriever.AddDocuments(ctx, docs)
	
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	
	// 验证子文档被添加到向量存储
	if len(vectorStore.docs) == 0 {
		t.Error("expected child documents in vector store")
	}
	
	// 验证父文档被添加到文档存储
	if len(docStore.docs) == 0 {
		t.Error("expected parent documents in doc store")
	}
}

func TestGetRelevantDocuments(t *testing.T) {
	vectorStore := &mockVectorStoreWithAdd{}
	docStore := NewMemoryDocumentStore()
	childSplitter := &mockTextSplitter{chunkSize: 20}
	
	retriever := NewParentDocumentRetriever(vectorStore, docStore, childSplitter)
	
	// 添加文档
	docs := []types.Document{
		{
			PageContent: "This is a test document with some content.",
			Metadata:    map[string]interface{}{"source": "test1"},
		},
	}
	
	ctx := context.Background()
	err := retriever.AddDocuments(ctx, docs)
	if err != nil {
		t.Fatalf("failed to add documents: %v", err)
	}
	
	// 检索文档
	results, err := retriever.GetRelevantDocuments(ctx, "test")
	if err != nil {
		t.Fatalf("failed to retrieve documents: %v", err)
	}
	
	// 验证返回了父文档
	if len(results) == 0 {
		t.Error("expected at least one document")
	}
	
	// 验证返回的是完整父文档
	if len(results) > 0 && len(results[0].PageContent) < len(docs[0].PageContent) {
		t.Error("expected full parent document, got child document")
	}
}

func TestExtractParentIDs(t *testing.T) {
	retriever := &ParentDocumentRetriever{
		config: DefaultParentDocumentConfig(),
	}
	
	docs := []types.Document{
		{
			PageContent: "chunk 1",
			Metadata: map[string]interface{}{
				"parent_id": "parent1",
			},
		},
		{
			PageContent: "chunk 2",
			Metadata: map[string]interface{}{
				"parent_id": "parent1",
			},
		},
		{
			PageContent: "chunk 3",
			Metadata: map[string]interface{}{
				"parent_id": "parent2",
			},
		},
	}
	
	ids := retriever.extractParentIDs(docs)
	
	// 应该去重，只返回唯一的父 ID
	if len(ids) != 2 {
		t.Errorf("expected 2 unique parent IDs, got %d", len(ids))
	}
}

func TestMemoryDocumentStore(t *testing.T) {
	store := NewMemoryDocumentStore()
	ctx := context.Background()
	
	// 添加文档
	docs := []types.Document{
		{
			PageContent: "doc 1",
			Metadata: map[string]interface{}{
				"doc_id": "id1",
			},
		},
		{
			PageContent: "doc 2",
			Metadata: map[string]interface{}{
				"doc_id": "id2",
			},
		},
	}
	
	err := store.AddDocuments(ctx, docs)
	if err != nil {
		t.Fatalf("failed to add documents: %v", err)
	}
	
	// 获取文档
	retrieved, err := store.GetDocuments(ctx, []string{"id1", "id2"})
	if err != nil {
		t.Fatalf("failed to get documents: %v", err)
	}
	
	if len(retrieved) != 2 {
		t.Errorf("expected 2 documents, got %d", len(retrieved))
	}
	
	// 删除文档
	err = store.DeleteDocuments(ctx, []string{"id1"})
	if err != nil {
		t.Fatalf("failed to delete documents: %v", err)
	}
	
	// 验证删除
	retrieved, err = store.GetDocuments(ctx, []string{"id1"})
	if err != nil {
		t.Fatalf("failed to get documents: %v", err)
	}
	
	if len(retrieved) != 0 {
		t.Errorf("expected 0 documents after deletion, got %d", len(retrieved))
	}
}

func TestParentDocumentOptions(t *testing.T) {
	vectorStore := &mockVectorStoreWithAdd{}
	docStore := NewMemoryDocumentStore()
	childSplitter := &mockTextSplitter{chunkSize: 50}
	parentSplitter := &mockTextSplitter{chunkSize: 200}
	
	t.Run("WithParentSplitter", func(t *testing.T) {
		retriever := NewParentDocumentRetriever(
			vectorStore, docStore, childSplitter,
			WithParentSplitter(parentSplitter),
		)
		
		if retriever.parentSplitter == nil {
			t.Error("expected parent splitter to be set")
		}
	})
	
	t.Run("WithIDKey", func(t *testing.T) {
		retriever := NewParentDocumentRetriever(
			vectorStore, docStore, childSplitter,
			WithIDKey("custom_id"),
		)
		
		if retriever.config.IDKey != "custom_id" {
			t.Errorf("expected 'custom_id', got %s", retriever.config.IDKey)
		}
	})
	
	t.Run("WithParentTopK", func(t *testing.T) {
		retriever := NewParentDocumentRetriever(
			vectorStore, docStore, childSplitter,
			WithParentTopK(10),
		)
		
		if retriever.config.TopK != 10 {
			t.Errorf("expected 10, got %d", retriever.config.TopK)
		}
	})
}
