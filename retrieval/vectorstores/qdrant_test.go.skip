package vectorstores

import (
	"context"
	"testing"

	"github.com/zhucl121/langchain-go/pkg/types"
)

func TestNewQdrant(t *testing.T) {
	qdrant := NewQdrant(
		WithQdrantURL("http://localhost:6333"),
		WithQdrantCollection("test_collection"),
		WithQdrantAPIKey("test-key"),
	)

	if qdrant == nil {
		t.Fatal("expected Qdrant instance, got nil")
	}

	if qdrant.config.URL != "http://localhost:6333" {
		t.Errorf("expected URL 'http://localhost:6333', got %s", qdrant.config.URL)
	}

	if qdrant.config.CollectionName != "test_collection" {
		t.Errorf("expected collection 'test_collection', got %s", qdrant.config.CollectionName)
	}
}

func TestQdrantConfig(t *testing.T) {
	tests := []struct {
		name   string
		option QdrantOption
		check  func(*QdrantConfig) error
	}{
		{
			name:   "WithQdrantURL",
			option: WithQdrantURL("http://test:6333"),
			check: func(c *QdrantConfig) error {
				if c.URL != "http://test:6333" {
					t.Errorf("expected URL 'http://test:6333', got %s", c.URL)
				}
				return nil
			},
		},
		{
			name:   "WithQdrantCollection",
			option: WithQdrantCollection("my_collection"),
			check: func(c *QdrantConfig) error {
				if c.CollectionName != "my_collection" {
					t.Errorf("expected collection 'my_collection', got %s", c.CollectionName)
				}
				return nil
			},
		},
		{
			name:   "WithQdrantAPIKey",
			option: WithQdrantAPIKey("secret-key"),
			check: func(c *QdrantConfig) error {
				if c.APIKey != "secret-key" {
					t.Errorf("expected API key 'secret-key', got %s", c.APIKey)
				}
				return nil
			},
		},
		{
			name:   "WithQdrantVectorSize",
			option: WithQdrantVectorSize(768),
			check: func(c *QdrantConfig) error {
				if c.VectorSize != 768 {
					t.Errorf("expected vector size 768, got %d", c.VectorSize)
				}
				return nil
			},
		},
		{
			name:   "WithQdrantDistance",
			option: WithQdrantDistance("Cosine"),
			check: func(c *QdrantConfig) error {
				if c.Distance != "Cosine" {
					t.Errorf("expected distance 'Cosine', got %s", c.Distance)
				}
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultQdrantConfig()
			tt.option(&config)
			if err := tt.check(&config); err != nil {
				t.Error(err)
			}
		})
	}
}

func TestQdrantMethods(t *testing.T) {
	// 这些是单元测试，不需要真实的 Qdrant 连接
	t.Run("buildCreateCollectionRequest", func(t *testing.T) {
		qdrant := NewQdrant(
			WithQdrantVectorSize(384),
			WithQdrantDistance("Cosine"),
		)

		req := qdrant.buildCreateCollectionRequest()
		if req == nil {
			t.Fatal("expected create collection request, got nil")
		}
	})

	t.Run("buildPoint", func(t *testing.T) {
		qdrant := NewQdrant()

		doc := types.Document{
			PageContent: "test content",
			Metadata: map[string]interface{}{
				"key": "value",
			},
		}
		embedding := []float64{0.1, 0.2, 0.3}

		point := qdrant.buildPoint("test-id", doc, embedding)
		if point.ID == "" {
			t.Error("expected point ID, got empty string")
		}
		if len(point.Vector) != 3 {
			t.Errorf("expected 3 vector dimensions, got %d", len(point.Vector))
		}
	})

	t.Run("buildSearchRequest", func(t *testing.T) {
		qdrant := NewQdrant(WithQdrantCollection("test"))

		embedding := []float64{0.1, 0.2, 0.3}
		req := qdrant.buildSearchRequest(embedding, 10, nil)

		if req == nil {
			t.Fatal("expected search request, got nil")
		}
		if req.Limit != 10 {
			t.Errorf("expected limit 10, got %d", req.Limit)
		}
		if len(req.Vector) != 3 {
			t.Errorf("expected 3 vector dimensions, got %d", len(req.Vector))
		}
	})
}

func TestQdrantFilterBuilder(t *testing.T) {
	qdrant := NewQdrant()

	tests := []struct {
		name   string
		filter map[string]interface{}
	}{
		{
			name: "simple filter",
			filter: map[string]interface{}{
				"category": "tech",
			},
		},
		{
			name: "multiple filters",
			filter: map[string]interface{}{
				"category": "tech",
				"year":     2023,
			},
		},
		{
			name: "complex filter",
			filter: map[string]interface{}{
				"category": "tech",
				"score":    95.5,
				"active":   true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := qdrant.buildFilter(tt.filter)
			if filter == nil && len(tt.filter) > 0 {
				t.Error("expected filter, got nil")
			}
		})
	}
}

func TestQdrantDefaultConfig(t *testing.T) {
	config := DefaultQdrantConfig()

	if config.URL != "http://localhost:6333" {
		t.Errorf("expected default URL, got %s", config.URL)
	}

	if config.VectorSize != 1536 {
		t.Errorf("expected default vector size 1536, got %d", config.VectorSize)
	}

	if config.Distance != "Cosine" {
		t.Errorf("expected default distance 'Cosine', got %s", config.Distance)
	}

	if config.BatchSize != 100 {
		t.Errorf("expected default batch size 100, got %d", config.BatchSize)
	}
}

// 注意：以下是集成测试，需要真实的 Qdrant 服务
// 在 CI/CD 环境中可以使用 Docker 启动 Qdrant 进行测试

func TestQdrantIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()

	qdrant := NewQdrant(
		WithQdrantURL("http://localhost:6333"),
		WithQdrantCollection("test_integration"),
		WithQdrantVectorSize(3),
	)

	// 测试添加文档
	t.Run("AddDocuments", func(t *testing.T) {
		docs := []types.Document{
			{
				PageContent: "test document 1",
				Metadata:    map[string]interface{}{"id": "1"},
			},
			{
				PageContent: "test document 2",
				Metadata:    map[string]interface{}{"id": "2"},
			},
		}

		embeddings := [][]float64{
			{0.1, 0.2, 0.3},
			{0.4, 0.5, 0.6},
		}

		// 注意：这需要真实的 Qdrant 连接
		// err := qdrant.AddDocuments(ctx, docs, embeddings)
		// if err != nil {
		// 	t.Fatalf("failed to add documents: %v", err)
		// }

		t.Log("Integration test would add documents here")
	})

	// 测试相似度搜索
	t.Run("SimilaritySearch", func(t *testing.T) {
		embedding := []float64{0.1, 0.2, 0.3}

		// 注意：这需要真实的 Qdrant 连接
		// docs, err := qdrant.SimilaritySearchByVector(ctx, embedding, 2)
		// if err != nil {
		// 	t.Fatalf("failed to search: %v", err)
		// }

		t.Log("Integration test would search documents here")
	})
}
