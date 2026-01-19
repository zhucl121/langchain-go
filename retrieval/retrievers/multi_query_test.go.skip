package retrievers

import (
	"context"
	"testing"

	"github.com/zhucl121/langchain-go/pkg/types"
)

// Mock Retriever for testing
type mockRetriever struct {
	docs []types.Document
}

func (m *mockRetriever) GetRelevantDocuments(ctx context.Context, query string) ([]types.Document, error) {
	return m.docs, nil
}

// Mock LLM for testing
type mockLLM struct {
	response string
}

func (m *mockLLM) Invoke(ctx context.Context, messages []types.Message, opts ...interface{}) (types.Message, error) {
	return types.Message{
		Role:    types.RoleAssistant,
		Content: m.response,
	}, nil
}

func (m *mockLLM) Stream(ctx context.Context, messages []types.Message, opts ...interface{}) (<-chan types.StreamEvent, error) {
	return nil, nil
}

func (m *mockLLM) Batch(ctx context.Context, messagesList [][]types.Message, opts ...interface{}) ([]types.Message, error) {
	return nil, nil
}

func TestNewMultiQueryRetriever(t *testing.T) {
	baseRetriever := &mockRetriever{
		docs: []types.Document{
			{PageContent: "test doc 1"},
		},
	}
	
	llm := &mockLLM{
		response: "alternative query 1\nalternative query 2",
	}
	
	retriever := NewMultiQueryRetriever(baseRetriever, llm)
	
	if retriever == nil {
		t.Fatal("expected retriever, got nil")
	}
	
	if retriever.config.NumQueries != 3 {
		t.Errorf("expected 3 queries, got %d", retriever.config.NumQueries)
	}
}

func TestParseQueries(t *testing.T) {
	retriever := &MultiQueryRetriever{
		config: DefaultMultiQueryConfig(),
	}
	
	tests := []struct {
		name     string
		response string
		expected []string
	}{
		{
			name:     "simple lines",
			response: "query 1\nquery 2\nquery 3",
			expected: []string{"query 1", "query 2", "query 3"},
		},
		{
			name:     "numbered lines",
			response: "1. query 1\n2. query 2\n3. query 3",
			expected: []string{"query 1", "query 2", "query 3"},
		},
		{
			name:     "bullet points",
			response: "- query 1\n- query 2\n- query 3",
			expected: []string{"query 1", "query 2", "query 3"},
		},
		{
			name:     "with quotes",
			response: "\"query 1\"\n\"query 2\"\n\"query 3\"",
			expected: []string{"query 1", "query 2", "query 3"},
		},
		{
			name:     "mixed format",
			response: "1) query 1\n- query 2\n* query 3",
			expected: []string{"query 1", "query 2", "query 3"},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			queries := retriever.parseQueries(tt.response)
			
			if len(queries) != len(tt.expected) {
				t.Errorf("expected %d queries, got %d", len(tt.expected), len(queries))
				return
			}
			
			for i, expected := range tt.expected {
				if queries[i] != expected {
					t.Errorf("query %d: expected %q, got %q", i, expected, queries[i])
				}
			}
		})
	}
}

func TestMergeStrategies(t *testing.T) {
	retriever := &MultiQueryRetriever{
		config: DefaultMultiQueryConfig(),
	}
	
	allDocs := [][]types.Document{
		{
			{PageContent: "doc1"},
			{PageContent: "doc2"},
		},
		{
			{PageContent: "doc2"},
			{PageContent: "doc3"},
		},
		{
			{PageContent: "doc1"},
			{PageContent: "doc3"},
		},
	}
	
	t.Run("union strategy", func(t *testing.T) {
		retriever.config.MergeStrategy = "union"
		result := retriever.mergeResults(allDocs)
		
		// Union should contain all documents (with duplicates)
		if len(result) != 6 {
			t.Errorf("expected 6 documents, got %d", len(result))
		}
	})
	
	t.Run("ranked strategy", func(t *testing.T) {
		retriever.config.MergeStrategy = "ranked"
		result := retriever.mergeResults(allDocs)
		
		// Should deduplicate and rank by frequency
		if len(result) < 3 {
			t.Errorf("expected at least 3 unique documents, got %d", len(result))
		}
	})
}

func TestDeduplicate(t *testing.T) {
	retriever := &MultiQueryRetriever{
		config: DefaultMultiQueryConfig(),
	}
	
	docs := []types.Document{
		{PageContent: "doc1"},
		{PageContent: "doc2"},
		{PageContent: "doc1"}, // duplicate
		{PageContent: "doc3"},
		{PageContent: "doc2"}, // duplicate
	}
	
	result := retriever.deduplicate(docs)
	
	if len(result) != 3 {
		t.Errorf("expected 3 unique documents, got %d", len(result))
	}
}

func TestOptions(t *testing.T) {
	baseRetriever := &mockRetriever{}
	llm := &mockLLM{}
	
	t.Run("WithNumQueries", func(t *testing.T) {
		retriever := NewMultiQueryRetriever(baseRetriever, llm, WithNumQueries(5))
		if retriever.config.NumQueries != 5 {
			t.Errorf("expected 5 queries, got %d", retriever.config.NumQueries)
		}
	})
	
	t.Run("WithIncludeOriginal", func(t *testing.T) {
		retriever := NewMultiQueryRetriever(baseRetriever, llm, WithIncludeOriginal(false))
		if retriever.config.IncludeOriginal != false {
			t.Error("expected IncludeOriginal to be false")
		}
	})
	
	t.Run("WithMergeStrategy", func(t *testing.T) {
		retriever := NewMultiQueryRetriever(baseRetriever, llm, WithMergeStrategy("ranked"))
		if retriever.config.MergeStrategy != "ranked" {
			t.Errorf("expected ranked strategy, got %s", retriever.config.MergeStrategy)
		}
	})
	
	t.Run("WithMaxResults", func(t *testing.T) {
		retriever := NewMultiQueryRetriever(baseRetriever, llm, WithMaxResults(10))
		if retriever.config.MaxResults != 10 {
			t.Errorf("expected 10 max results, got %d", retriever.config.MaxResults)
		}
	})
}
