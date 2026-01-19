package retrievers

import (
	"context"
	"testing"

	"github.com/zhucl121/langchain-go/pkg/types"
)

// Mock vector store with filter
type mockVectorStoreWithFilter struct {
	docs []types.Document
}

func (m *mockVectorStoreWithFilter) SimilaritySearchWithFilter(
	ctx context.Context,
	query string,
	k int,
	filter map[string]interface{},
) ([]types.Document, error) {
	// 简单过滤实现
	var filtered []types.Document
	
	for _, doc := range m.docs {
		matches := true
		
		// 检查过滤条件
		for key, value := range filter {
			if doc.Metadata == nil {
				matches = false
				break
			}
			
			if docValue, ok := doc.Metadata[key]; !ok || docValue != value {
				matches = false
				break
			}
		}
		
		if matches {
			filtered = append(filtered, doc)
		}
	}
	
	// 限制返回数量
	if k > len(filtered) {
		k = len(filtered)
	}
	
	return filtered[:k], nil
}

func TestNewSelfQueryRetriever(t *testing.T) {
	llm := &mockLLM{
		response: `{"query": "test query", "filter": {"category": "tech"}}`,
	}
	
	vectorStore := &mockVectorStoreWithFilter{
		docs: []types.Document{
			{
				PageContent: "test doc",
				Metadata:    map[string]interface{}{"category": "tech"},
			},
		},
	}
	
	metadataFields := []MetadataField{
		NewMetadataField("category", "string", "Document category", "tech", "science", "art"),
		NewMetadataField("year", "number", "Publication year"),
	}
	
	retriever := NewSelfQueryRetriever(
		llm,
		vectorStore,
		"Technical documents",
		metadataFields,
	)
	
	if retriever == nil {
		t.Fatal("expected retriever, got nil")
	}
	
	if retriever.config.TopK != 4 {
		t.Errorf("expected TopK 4, got %d", retriever.config.TopK)
	}
}

func TestParseQuery(t *testing.T) {
	llm := &mockLLM{
		response: `{"query": "machine learning", "filter": {"category": "tech", "year": 2023}}`,
	}
	
	vectorStore := &mockVectorStoreWithFilter{}
	metadataFields := []MetadataField{
		NewMetadataField("category", "string", "Category"),
		NewMetadataField("year", "number", "Year"),
	}
	
	retriever := NewSelfQueryRetriever(
		llm,
		vectorStore,
		"Documents",
		metadataFields,
	)
	
	ctx := context.Background()
	structuredQuery, err := retriever.parseQuery(ctx, "Show me tech articles about machine learning from 2023")
	
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	
	if structuredQuery.Query != "machine learning" {
		t.Errorf("expected 'machine learning', got %q", structuredQuery.Query)
	}
	
	if structuredQuery.Filter["category"] != "tech" {
		t.Errorf("expected category 'tech', got %v", structuredQuery.Filter["category"])
	}
}

func TestExtractJSON(t *testing.T) {
	retriever := &SelfQueryRetriever{}
	
	tests := []struct {
		name     string
		response string
		expected string
	}{
		{
			name:     "simple json",
			response: `{"query": "test", "filter": {}}`,
			expected: `{"query": "test", "filter": {}}`,
		},
		{
			name:     "json with text before",
			response: `Here is the result: {"query": "test", "filter": {}}`,
			expected: `{"query": "test", "filter": {}}`,
		},
		{
			name:     "nested json",
			response: `{"query": "test", "filter": {"nested": {"key": "value"}}}`,
			expected: `{"query": "test", "filter": {"nested": {"key": "value"}}}`,
		},
		{
			name:     "no json",
			response: `No JSON here`,
			expected: `{}`,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := retriever.extractJSON(tt.response)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestGetRelevantDocuments(t *testing.T) {
	llm := &mockLLM{
		response: `{"query": "test", "filter": {"category": "tech"}}`,
	}
	
	vectorStore := &mockVectorStoreWithFilter{
		docs: []types.Document{
			{
				PageContent: "tech doc 1",
				Metadata:    map[string]interface{}{"category": "tech"},
			},
			{
				PageContent: "science doc 1",
				Metadata:    map[string]interface{}{"category": "science"},
			},
			{
				PageContent: "tech doc 2",
				Metadata:    map[string]interface{}{"category": "tech"},
			},
		},
	}
	
	metadataFields := []MetadataField{
		NewMetadataField("category", "string", "Category"),
	}
	
	retriever := NewSelfQueryRetriever(
		llm,
		vectorStore,
		"Documents",
		metadataFields,
	)
	
	ctx := context.Background()
	docs, err := retriever.GetRelevantDocuments(ctx, "Show me tech documents")
	
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	
	// 应该只返回 tech 类别的文档
	if len(docs) > 2 {
		t.Errorf("expected at most 2 tech documents, got %d", len(docs))
	}
	
	for _, doc := range docs {
		if doc.Metadata["category"] != "tech" {
			t.Errorf("expected only tech documents, got category: %v", doc.Metadata["category"])
		}
	}
}

func TestMetadataField(t *testing.T) {
	field := NewMetadataField(
		"genre",
		"string",
		"Movie genre",
		"action",
		"comedy",
		"drama",
	)
	
	if field.Name != "genre" {
		t.Errorf("expected 'genre', got %s", field.Name)
	}
	
	if field.Type != "string" {
		t.Errorf("expected 'string', got %s", field.Type)
	}
	
	if len(field.AllowedValues) != 3 {
		t.Errorf("expected 3 allowed values, got %d", len(field.AllowedValues))
	}
}

func TestSelfQueryOptions(t *testing.T) {
	llm := &mockLLM{}
	vectorStore := &mockVectorStoreWithFilter{}
	metadataFields := []MetadataField{}
	
	t.Run("WithSelfQueryTopK", func(t *testing.T) {
		retriever := NewSelfQueryRetriever(
			llm, vectorStore, "docs", metadataFields,
			WithSelfQueryTopK(10),
		)
		
		if retriever.config.TopK != 10 {
			t.Errorf("expected 10, got %d", retriever.config.TopK)
		}
	})
	
	t.Run("WithAllowEmptyQuery", func(t *testing.T) {
		retriever := NewSelfQueryRetriever(
			llm, vectorStore, "docs", metadataFields,
			WithAllowEmptyQuery(false),
		)
		
		if retriever.config.AllowEmptyQuery != false {
			t.Error("expected AllowEmptyQuery to be false")
		}
	})
	
	t.Run("WithAllowEmptyFilter", func(t *testing.T) {
		retriever := NewSelfQueryRetriever(
			llm, vectorStore, "docs", metadataFields,
			WithAllowEmptyFilter(false),
		)
		
		if retriever.config.AllowEmptyFilter != false {
			t.Error("expected AllowEmptyFilter to be false")
		}
	})
}
