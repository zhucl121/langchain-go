package retrievers

import (
	"context"
	"testing"

	"github.com/zhucl121/langchain-go/pkg/types"
)

// Mock VectorStore for testing
type mockVectorStore struct {
	docs []types.Document
}

func (m *mockVectorStore) SimilaritySearchByVector(ctx context.Context, embedding []float64, k int) ([]types.Document, error) {
	if k > len(m.docs) {
		k = len(m.docs)
	}
	return m.docs[:k], nil
}

// Mock Embedder for testing
type mockEmbedder struct{}

func (m *mockEmbedder) EmbedQuery(ctx context.Context, text string) ([]float64, error) {
	// Return a simple mock embedding
	return []float64{0.1, 0.2, 0.3, 0.4}, nil
}

func (m *mockEmbedder) EmbedDocuments(ctx context.Context, texts []string) ([][]float64, error) {
	embeddings := make([][]float64, len(texts))
	for i := range texts {
		embeddings[i] = []float64{0.1, 0.2, 0.3, 0.4}
	}
	return embeddings, nil
}

func TestNewHyDERetriever(t *testing.T) {
	llm := &mockLLM{
		response: "This is a hypothetical document answering the question.",
	}
	
	embedder := &mockEmbedder{}
	
	vectorStore := &mockVectorStore{
		docs: []types.Document{
			{PageContent: "test doc 1"},
			{PageContent: "test doc 2"},
		},
	}
	
	retriever := NewHyDERetriever(llm, embedder, vectorStore)
	
	if retriever == nil {
		t.Fatal("expected retriever, got nil")
	}
	
	if retriever.config.NumHypothetical != 1 {
		t.Errorf("expected 1 hypothetical document, got %d", retriever.config.NumHypothetical)
	}
}

func TestParseHypotheticalDocuments(t *testing.T) {
	retriever := &HyDERetriever{
		config: DefaultHyDEConfig(),
	}
	
	t.Run("single document", func(t *testing.T) {
		retriever.config.NumHypothetical = 1
		response := "This is a single hypothetical document."
		
		docs := retriever.parseHypotheticalDocuments(response)
		
		if len(docs) != 1 {
			t.Errorf("expected 1 document, got %d", len(docs))
		}
		
		if docs[0] != response {
			t.Errorf("expected %q, got %q", response, docs[0])
		}
	})
	
	t.Run("multiple documents", func(t *testing.T) {
		retriever.config.NumHypothetical = 3
		response := "Document 1 content\n---\nDocument 2 content\n---\nDocument 3 content"
		
		docs := retriever.parseHypotheticalDocuments(response)
		
		if len(docs) != 3 {
			t.Errorf("expected 3 documents, got %d", len(docs))
		}
	})
}

func TestWeightedAverage(t *testing.T) {
	retriever := &HyDERetriever{
		config: DefaultHyDEConfig(),
	}
	
	embeddings := [][]float64{
		{1.0, 2.0, 3.0},
		{2.0, 4.0, 6.0},
	}
	
	weights := []float64{1.0, 1.0}
	
	result := retriever.weightedAverage(embeddings, weights)
	
	expected := []float64{1.5, 3.0, 4.5}
	
	if len(result) != len(expected) {
		t.Fatalf("expected length %d, got %d", len(expected), len(result))
	}
	
	for i, v := range result {
		if v != expected[i] {
			t.Errorf("index %d: expected %f, got %f", i, expected[i], v)
		}
	}
}

func TestCombineStrategies(t *testing.T) {
	llm := &mockLLM{
		response: "Hypothetical document",
	}
	
	embedder := &mockEmbedder{}
	
	vectorStore := &mockVectorStore{
		docs: []types.Document{
			{PageContent: "doc1"},
			{PageContent: "doc2"},
			{PageContent: "doc3"},
		},
	}
	
	t.Run("first strategy", func(t *testing.T) {
		retriever := NewHyDERetriever(llm, embedder, vectorStore,
			WithCombineStrategy("first"),
			WithTopK(2),
		)
		
		ctx := context.Background()
		docs, err := retriever.GetRelevantDocuments(ctx, "test query")
		
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		
		if len(docs) > 2 {
			t.Errorf("expected at most 2 documents, got %d", len(docs))
		}
	})
	
	t.Run("average strategy", func(t *testing.T) {
		retriever := NewHyDERetriever(llm, embedder, vectorStore,
			WithCombineStrategy("average"),
			WithNumHypothetical(2),
			WithTopK(2),
		)
		
		ctx := context.Background()
		docs, err := retriever.GetRelevantDocuments(ctx, "test query")
		
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		
		if len(docs) > 2 {
			t.Errorf("expected at most 2 documents, got %d", len(docs))
		}
	})
}

func TestHyDEOptions(t *testing.T) {
	llm := &mockLLM{}
	embedder := &mockEmbedder{}
	vectorStore := &mockVectorStore{}
	
	t.Run("WithNumHypothetical", func(t *testing.T) {
		retriever := NewHyDERetriever(llm, embedder, vectorStore,
			WithNumHypothetical(3),
		)
		
		if retriever.config.NumHypothetical != 3 {
			t.Errorf("expected 3, got %d", retriever.config.NumHypothetical)
		}
	})
	
	t.Run("WithCombineStrategy", func(t *testing.T) {
		retriever := NewHyDERetriever(llm, embedder, vectorStore,
			WithCombineStrategy("separate"),
		)
		
		if retriever.config.CombineStrategy != "separate" {
			t.Errorf("expected 'separate', got %s", retriever.config.CombineStrategy)
		}
	})
	
	t.Run("WithTopK", func(t *testing.T) {
		retriever := NewHyDERetriever(llm, embedder, vectorStore,
			WithTopK(10),
		)
		
		if retriever.config.TopK != 10 {
			t.Errorf("expected 10, got %d", retriever.config.TopK)
		}
	})
	
	t.Run("WithQueryEmbedding", func(t *testing.T) {
		retriever := NewHyDERetriever(llm, embedder, vectorStore,
			WithQueryEmbedding(true, 0.5),
		)
		
		if !retriever.config.IncludeQueryEmbedding {
			t.Error("expected IncludeQueryEmbedding to be true")
		}
		
		if retriever.config.QueryWeight != 0.5 {
			t.Errorf("expected 0.5, got %f", retriever.config.QueryWeight)
		}
	})
}
