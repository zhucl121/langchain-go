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

// TestChromaVectorStore tests Chroma vector store functionality
// Note: This test requires a running Chroma server at localhost:8000
// You can start Chroma with: docker run -p 8000:8000 chromadb/chroma
func TestChromaVectorStore(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Check if Chroma is available
	chromaURL := os.Getenv("CHROMA_URL")
	if chromaURL == "" {
		chromaURL = "http://localhost:8000"
	}

	// Skip test if Chroma is not available
	// You can check with: curl http://localhost:8000/api/v1/heartbeat
	if os.Getenv("SKIP_CHROMA_TESTS") == "true" {
		t.Skip("Skipping Chroma tests (set SKIP_CHROMA_TESTS=false to run)")
	}

	ctx := context.Background()

	// Create fake embeddings for testing
	fakeEmb := &FakeEmbeddings{
		Dimension: 384,
	}

	// Test configuration
	config := ChromaConfig{
		URL:                  chromaURL,
		CollectionName:       "test_collection",
		DistanceFunction:     "cosine",
		AutoCreateCollection: true,
	}

	t.Run("NewChromaVectorStore", func(t *testing.T) {
		store, err := NewChromaVectorStore(config, fakeEmb)
		require.NoError(t, err)
		require.NotNil(t, store)
		defer store.Close()

		// Cleanup
		err = store.Clear(ctx)
		assert.NoError(t, err)
	})

	t.Run("NewChromaVectorStore_InvalidConfig", func(t *testing.T) {
		invalidConfig := ChromaConfig{
			URL:            chromaURL,
			CollectionName: "", // Empty collection name
		}
		store, err := NewChromaVectorStore(invalidConfig, fakeEmb)
		assert.Error(t, err)
		assert.Nil(t, store)
	})

	t.Run("AddDocuments", func(t *testing.T) {
		store, err := NewChromaVectorStore(config, fakeEmb)
		require.NoError(t, err)
		defer store.Close()
		defer store.Clear(ctx)

		docs := []*loaders.Document{
			loaders.NewDocument("Machine learning is a subset of AI", map[string]any{
				"topic": "AI",
				"page":  1,
			}),
			loaders.NewDocument("Deep learning uses neural networks", map[string]any{
				"topic": "Deep Learning",
				"page":  2,
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

		// Verify count
		count, err := store.Count(ctx)
		assert.NoError(t, err)
		assert.Equal(t, int64(3), count)
	})

	t.Run("AddDocuments_Empty", func(t *testing.T) {
		store, err := NewChromaVectorStore(config, fakeEmb)
		require.NoError(t, err)
		defer store.Close()
		defer store.Clear(ctx)

		ids, err := store.AddDocuments(ctx, []*loaders.Document{})
		assert.NoError(t, err)
		assert.Empty(t, ids)
	})

	t.Run("SimilaritySearch", func(t *testing.T) {
		store, err := NewChromaVectorStore(config, fakeEmb)
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
		assert.Len(t, results, 2)

		// Verify results have similarity scores
		for _, result := range results {
			assert.NotEmpty(t, result.Content)
			assert.Contains(t, result.Metadata, "distance")
			assert.Contains(t, result.Metadata, "similarity_score")
		}
	})

	t.Run("SimilaritySearchWithScore", func(t *testing.T) {
		store, err := NewChromaVectorStore(config, fakeEmb)
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
		results, err := store.SimilaritySearchWithScore(ctx, "machine learning", 5, 0.5)
		assert.NoError(t, err)
		assert.NotEmpty(t, results)

		// All results should have distance <= 0.5
		for _, result := range results {
			distance, ok := result.Metadata["distance"].(float64)
			assert.True(t, ok)
			assert.LessOrEqual(t, distance, 0.5)
		}
	})

	t.Run("GetByIDs", func(t *testing.T) {
		store, err := NewChromaVectorStore(config, fakeEmb)
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
		assert.Len(t, retrieved, 1)
		assert.Equal(t, "First document", retrieved[0].Content)
	})

	t.Run("GetByIDs_Empty", func(t *testing.T) {
		store, err := NewChromaVectorStore(config, fakeEmb)
		require.NoError(t, err)
		defer store.Close()
		defer store.Clear(ctx)

		retrieved, err := store.GetByIDs(ctx, []string{})
		assert.NoError(t, err)
		assert.Empty(t, retrieved)
	})

	t.Run("Delete", func(t *testing.T) {
		store, err := NewChromaVectorStore(config, fakeEmb)
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

		// Verify deletion
		count, err := store.Count(ctx)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), count)
	})

	t.Run("Delete_Empty", func(t *testing.T) {
		store, err := NewChromaVectorStore(config, fakeEmb)
		require.NoError(t, err)
		defer store.Close()
		defer store.Clear(ctx)

		err = store.Delete(ctx, []string{})
		assert.NoError(t, err)
	})

	t.Run("Count", func(t *testing.T) {
		store, err := NewChromaVectorStore(config, fakeEmb)
		require.NoError(t, err)
		defer store.Close()
		defer store.Clear(ctx)

		// Initially empty
		count, err := store.Count(ctx)
		assert.NoError(t, err)
		assert.Equal(t, int64(0), count)

		// Add documents
		docs := []*loaders.Document{
			loaders.NewDocument("Doc 1", nil),
			loaders.NewDocument("Doc 2", nil),
			loaders.NewDocument("Doc 3", nil),
		}

		_, err = store.AddDocuments(ctx, docs)
		require.NoError(t, err)

		// Count after adding
		count, err = store.Count(ctx)
		assert.NoError(t, err)
		assert.Equal(t, int64(3), count)
	})

	t.Run("Clear", func(t *testing.T) {
		store, err := NewChromaVectorStore(config, fakeEmb)
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

		// Verify empty
		count, err := store.Count(ctx)
		assert.NoError(t, err)
		assert.Equal(t, int64(0), count)
	})

	t.Run("DistanceFunctions", func(t *testing.T) {
		distanceFuncs := []string{"l2", "ip", "cosine"}

		for _, distFunc := range distanceFuncs {
			t.Run(distFunc, func(t *testing.T) {
				cfg := ChromaConfig{
					URL:                  chromaURL,
					CollectionName:       "test_" + distFunc,
					DistanceFunction:     distFunc,
					AutoCreateCollection: true,
				}

				store, err := NewChromaVectorStore(cfg, fakeEmb)
				require.NoError(t, err)
				defer store.Close()
				defer store.Clear(ctx)

				// Add and search
				docs := []*loaders.Document{
					loaders.NewDocument("Test document", nil),
				}

				_, err = store.AddDocuments(ctx, docs)
				require.NoError(t, err)

				results, err := store.SimilaritySearch(ctx, "Test", 1)
				assert.NoError(t, err)
				assert.NotEmpty(t, results)
			})
		}
	})

	t.Run("MetadataPreservation", func(t *testing.T) {
		store, err := NewChromaVectorStore(config, fakeEmb)
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
		require.Len(t, retrieved, 1)

		metadata := retrieved[0].Metadata
		assert.Equal(t, "value", metadata["string_field"])
		assert.NotNil(t, metadata["int_field"])
		assert.NotNil(t, metadata["float_field"])
		assert.NotNil(t, metadata["bool_field"])
	})
}

// BenchmarkChromaVectorStore benchmarks Chroma operations
func BenchmarkChromaVectorStore(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping benchmark in short mode")
	}

	if os.Getenv("SKIP_CHROMA_TESTS") == "true" {
		b.Skip("Skipping Chroma benchmarks")
	}

	chromaURL := os.Getenv("CHROMA_URL")
	if chromaURL == "" {
		chromaURL = "http://localhost:8000"
	}

	ctx := context.Background()
	fakeEmb := &FakeEmbeddings{Dimension: 384}

	config := ChromaConfig{
		URL:                  chromaURL,
		CollectionName:       "bench_collection",
		DistanceFunction:     "cosine",
		AutoCreateCollection: true,
	}

	store, err := NewChromaVectorStore(config, fakeEmb)
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
