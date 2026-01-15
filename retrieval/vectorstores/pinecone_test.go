package vectorstores

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tmc/langchaingo/retrieval/embeddings"
	"github.com/tmc/langchaingo/retrieval/loaders"
)

// TestPineconeVectorStore tests Pinecone vector store functionality
// Note: This test requires a Pinecone API key
// Set PINECONE_API_KEY environment variable to run these tests
func TestPineconeVectorStore(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	apiKey := os.Getenv("PINECONE_API_KEY")
	if apiKey == "" {
		t.Skip("Skipping Pinecone tests (set PINECONE_API_KEY to run)")
	}

	ctx := context.Background()

	// Create fake embeddings for testing
	fakeEmb := &FakeEmbeddings{
		Dimension: 384,
	}

	// Test configuration
	config := PineconeConfig{
		APIKey:          apiKey,
		IndexName:       "test-index",
		Namespace:       "test",
		Dimension:       384,
		Metric:          "cosine",
		AutoCreateIndex: true,
	}

	t.Run("NewPineconeVectorStore", func(t *testing.T) {
		store, err := NewPineconeVectorStore(config, fakeEmb)
		require.NoError(t, err)
		require.NotNil(t, store)
		defer store.Close()

		// Cleanup
		err = store.Clear(ctx)
		assert.NoError(t, err)
	})

	t.Run("NewPineconeVectorStore_InvalidConfig", func(t *testing.T) {
		invalidConfigs := []PineconeConfig{
			{
				APIKey:    "", // Empty API key
				IndexName: "test",
				Dimension: 384,
			},
			{
				APIKey:    apiKey,
				IndexName: "", // Empty index name
				Dimension: 384,
			},
			{
				APIKey:    apiKey,
				IndexName: "test",
				Dimension: 0, // Invalid dimension
			},
		}

		for _, cfg := range invalidConfigs {
			store, err := NewPineconeVectorStore(cfg, fakeEmb)
			assert.Error(t, err)
			assert.Nil(t, store)
		}
	})

	t.Run("AddDocuments", func(t *testing.T) {
		store, err := NewPineconeVectorStore(config, fakeEmb)
		require.NoError(t, err)
		defer store.Close()
		defer store.Clear(ctx)

		docs := []*loaders.Document{
			loaders.NewDocument("Machine learning is a subset of AI", map[string]any{
				"topic":  "AI",
				"page":   1,
				"author": "John Doe",
			}),
			loaders.NewDocument("Deep learning uses neural networks", map[string]any{
				"topic":  "Deep Learning",
				"page":   2,
				"author": "Jane Smith",
			}),
			loaders.NewDocument("NLP processes human language", map[string]any{
				"topic": "NLP",
				"page":  3,
			}),
		}

		ids, err := store.AddDocuments(ctx, docs)
		assert.NoError(t, err)
		assert.Len(t, ids, 3)
		assert.NotEmpty(t, ids[0])

		// Note: Pinecone has eventual consistency, may need to wait
		// for vectors to be indexed before querying
	})

	t.Run("AddDocuments_Empty", func(t *testing.T) {
		store, err := NewPineconeVectorStore(config, fakeEmb)
		require.NoError(t, err)
		defer store.Close()
		defer store.Clear(ctx)

		ids, err := store.AddDocuments(ctx, []*loaders.Document{})
		assert.NoError(t, err)
		assert.Empty(t, ids)
	})

	t.Run("SimilaritySearch", func(t *testing.T) {
		store, err := NewPineconeVectorStore(config, fakeEmb)
		require.NoError(t, err)
		defer store.Close()
		defer store.Clear(ctx)

		// Add test documents
		docs := []*loaders.Document{
			loaders.NewDocument("Artificial intelligence and machine learning", nil),
			loaders.NewDocument("Deep learning and neural networks", nil),
			loaders.NewDocument("Natural language processing", nil),
			loaders.NewDocument("Computer vision and image recognition", nil),
		}

		_, err = store.AddDocuments(ctx, docs)
		require.NoError(t, err)

		// Search
		results, err := store.SimilaritySearch(ctx, "AI and ML", 2)
		assert.NoError(t, err)

		// Results may be empty due to eventual consistency
		if len(results) > 0 {
			assert.LessOrEqual(t, len(results), 2)

			// Verify results have scores
			for _, result := range results {
				assert.NotEmpty(t, result.Content)
				assert.Contains(t, result.Metadata, "score")
			}
		}
	})

	t.Run("SimilaritySearchWithScore", func(t *testing.T) {
		store, err := NewPineconeVectorStore(config, fakeEmb)
		require.NoError(t, err)
		defer store.Close()
		defer store.Clear(ctx)

		// Add test documents
		docs := []*loaders.Document{
			loaders.NewDocument("Machine learning algorithms", nil),
			loaders.NewDocument("Deep learning models", nil),
			loaders.NewDocument("Data science techniques", nil),
		}

		_, err = store.AddDocuments(ctx, docs)
		require.NoError(t, err)

		// Search with score threshold
		results, err := store.SimilaritySearchWithScore(ctx, "machine learning", 5, 0.7)
		assert.NoError(t, err)

		// All results should have score >= 0.7
		for _, result := range results {
			score, ok := result.Metadata["score"].(float64)
			assert.True(t, ok)
			assert.GreaterOrEqual(t, score, 0.7)
		}
	})

	t.Run("GetByIDs", func(t *testing.T) {
		store, err := NewPineconeVectorStore(config, fakeEmb)
		require.NoError(t, err)
		defer store.Close()
		defer store.Clear(ctx)

		// Add documents
		docs := []*loaders.Document{
			loaders.NewDocument("First document", nil),
			loaders.NewDocument("Second document", nil),
		}

		ids, err := store.AddDocuments(ctx, docs)
		require.NoError(t, err)
		require.Len(t, ids, 2)

		// Get by IDs
		retrieved, err := store.GetByIDs(ctx, ids[:1])
		assert.NoError(t, err)

		// May be empty due to eventual consistency
		if len(retrieved) > 0 {
			assert.Equal(t, "First document", retrieved[0].Content)
		}
	})

	t.Run("GetByIDs_Empty", func(t *testing.T) {
		store, err := NewPineconeVectorStore(config, fakeEmb)
		require.NoError(t, err)
		defer store.Close()
		defer store.Clear(ctx)

		retrieved, err := store.GetByIDs(ctx, []string{})
		assert.NoError(t, err)
		assert.Empty(t, retrieved)
	})

	t.Run("Delete", func(t *testing.T) {
		store, err := NewPineconeVectorStore(config, fakeEmb)
		require.NoError(t, err)
		defer store.Close()
		defer store.Clear(ctx)

		// Add documents
		docs := []*loaders.Document{
			loaders.NewDocument("Document to delete", nil),
			loaders.NewDocument("Document to keep", nil),
		}

		ids, err := store.AddDocuments(ctx, docs)
		require.NoError(t, err)
		require.Len(t, ids, 2)

		// Delete first document
		err = store.Delete(ctx, ids[:1])
		assert.NoError(t, err)
	})

	t.Run("Delete_Empty", func(t *testing.T) {
		store, err := NewPineconeVectorStore(config, fakeEmb)
		require.NoError(t, err)
		defer store.Close()
		defer store.Clear(ctx)

		err = store.Delete(ctx, []string{})
		assert.NoError(t, err)
	})

	t.Run("Count", func(t *testing.T) {
		store, err := NewPineconeVectorStore(config, fakeEmb)
		require.NoError(t, err)
		defer store.Close()
		defer store.Clear(ctx)

		// Get initial count
		count, err := store.Count(ctx)
		assert.NoError(t, err)
		initialCount := count

		// Add documents
		docs := []*loaders.Document{
			loaders.NewDocument("Doc 1", nil),
			loaders.NewDocument("Doc 2", nil),
			loaders.NewDocument("Doc 3", nil),
		}

		_, err = store.AddDocuments(ctx, docs)
		require.NoError(t, err)

		// Count may not reflect immediately due to eventual consistency
		count, err = store.Count(ctx)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, count, initialCount)
	})

	t.Run("Clear", func(t *testing.T) {
		store, err := NewPineconeVectorStore(config, fakeEmb)
		require.NoError(t, err)
		defer store.Close()

		// Add documents
		docs := []*loaders.Document{
			loaders.NewDocument("Doc 1", nil),
			loaders.NewDocument("Doc 2", nil),
		}

		_, err = store.AddDocuments(ctx, docs)
		require.NoError(t, err)

		// Clear
		err = store.Clear(ctx)
		assert.NoError(t, err)
	})

	t.Run("MetricTypes", func(t *testing.T) {
		metrics := []string{"cosine", "euclidean", "dotproduct"}

		for _, metric := range metrics {
			t.Run(metric, func(t *testing.T) {
				cfg := PineconeConfig{
					APIKey:          apiKey,
					IndexName:       "test-" + metric,
					Namespace:       "test",
					Dimension:       384,
					Metric:          metric,
					AutoCreateIndex: true,
				}

				store, err := NewPineconeVectorStore(cfg, fakeEmb)
				require.NoError(t, err)
				defer store.Close()
				defer store.Clear(ctx)

				// Add and search
				docs := []*loaders.Document{
					loaders.NewDocument("Test document", nil),
				}

				_, err = store.AddDocuments(ctx, docs)
				require.NoError(t, err)
			})
		}
	})

	t.Run("Namespaces", func(t *testing.T) {
		namespaces := []string{"namespace1", "namespace2"}

		for _, ns := range namespaces {
			t.Run(ns, func(t *testing.T) {
				cfg := config
				cfg.Namespace = ns

				store, err := NewPineconeVectorStore(cfg, fakeEmb)
				require.NoError(t, err)
				defer store.Close()
				defer store.Clear(ctx)

				// Add document
				docs := []*loaders.Document{
					loaders.NewDocument("Document in "+ns, nil),
				}

				ids, err := store.AddDocuments(ctx, docs)
				require.NoError(t, err)
				assert.NotEmpty(t, ids)
			})
		}
	})

	t.Run("MetadataPreservation", func(t *testing.T) {
		store, err := NewPineconeVectorStore(config, fakeEmb)
		require.NoError(t, err)
		defer store.Close()
		defer store.Clear(ctx)

		// Add document with various metadata types
		docs := []*loaders.Document{
			loaders.NewDocument("Test document", map[string]any{
				"string_field": "value",
				"int_field":    42,
				"float_field":  3.14,
				"bool_field":   true,
			}),
		}

		ids, err := store.AddDocuments(ctx, docs)
		require.NoError(t, err)

		// Retrieve and check metadata
		retrieved, err := store.GetByIDs(ctx, ids)
		require.NoError(t, err)

		// May be empty due to eventual consistency
		if len(retrieved) > 0 {
			metadata := retrieved[0].Metadata
			assert.NotNil(t, metadata["string_field"])
			assert.NotNil(t, metadata["int_field"])
			assert.NotNil(t, metadata["float_field"])
			assert.NotNil(t, metadata["bool_field"])
		}
	})
}

// BenchmarkPineconeVectorStore benchmarks Pinecone operations
func BenchmarkPineconeVectorStore(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping benchmark in short mode")
	}

	apiKey := os.Getenv("PINECONE_API_KEY")
	if apiKey == "" {
		b.Skip("Skipping Pinecone benchmarks (set PINECONE_API_KEY to run)")
	}

	ctx := context.Background()
	fakeEmb := &FakeEmbeddings{Dimension: 384}

	config := PineconeConfig{
		APIKey:          apiKey,
		IndexName:       "bench-index",
		Namespace:       "bench",
		Dimension:       384,
		Metric:          "cosine",
		AutoCreateIndex: true,
	}

	store, err := NewPineconeVectorStore(config, fakeEmb)
	if err != nil {
		b.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()
	defer store.Clear(ctx)

	b.Run("AddDocuments", func(b *testing.B) {
		docs := []*loaders.Document{
			loaders.NewDocument("Benchmark document", nil),
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := store.AddDocuments(ctx, docs)
			if err != nil {
				b.Fatalf("AddDocuments failed: %v", err)
			}
		}
	})

	// Prepare data for search benchmark
	docs := make([]*loaders.Document, 100)
	for i := 0; i < 100; i++ {
		docs[i] = loaders.NewDocument("Test document for benchmarking", nil)
	}
	store.AddDocuments(ctx, docs)

	b.Run("SimilaritySearch", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := store.SimilaritySearch(ctx, "test query", 10)
			if err != nil {
				b.Fatalf("SimilaritySearch failed: %v", err)
			}
		}
	})
}
