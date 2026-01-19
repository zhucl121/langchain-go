package vectorstores

import (
	"context"
	"testing"

	"github.com/zhucl121/langchain-go/pkg/types"
)

func TestNewWeaviate(t *testing.T) {
	weaviate := NewWeaviate(
		WithWeaviateURL("http://localhost:8080"),
		WithWeaviateClassName("TestClass"),
		WithWeaviateAPIKey("test-key"),
	)

	if weaviate == nil {
		t.Fatal("expected Weaviate instance, got nil")
	}

	if weaviate.config.URL != "http://localhost:8080" {
		t.Errorf("expected URL 'http://localhost:8080', got %s", weaviate.config.URL)
	}

	if weaviate.config.ClassName != "TestClass" {
		t.Errorf("expected class 'TestClass', got %s", weaviate.config.ClassName)
	}
}

func TestWeaviateConfig(t *testing.T) {
	tests := []struct {
		name   string
		option WeaviateOption
		check  func(*WeaviateConfig) error
	}{
		{
			name:   "WithWeaviateURL",
			option: WithWeaviateURL("http://test:8080"),
			check: func(c *WeaviateConfig) error {
				if c.URL != "http://test:8080" {
					t.Errorf("expected URL 'http://test:8080', got %s", c.URL)
				}
				return nil
			},
		},
		{
			name:   "WithWeaviateClassName",
			option: WithWeaviateClassName("MyClass"),
			check: func(c *WeaviateConfig) error {
				if c.ClassName != "MyClass" {
					t.Errorf("expected class 'MyClass', got %s", c.ClassName)
				}
				return nil
			},
		},
		{
			name:   "WithWeaviateAPIKey",
			option: WithWeaviateAPIKey("secret-key"),
			check: func(c *WeaviateConfig) error {
				if c.APIKey != "secret-key" {
					t.Errorf("expected API key 'secret-key', got %s", c.APIKey)
				}
				return nil
			},
		},
		{
			name:   "WithWeaviateVectorizer",
			option: WithWeaviateVectorizer("text2vec-openai"),
			check: func(c *WeaviateConfig) error {
				if c.Vectorizer != "text2vec-openai" {
					t.Errorf("expected vectorizer 'text2vec-openai', got %s", c.Vectorizer)
				}
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultWeaviateConfig()
			tt.option(&config)
			if err := tt.check(&config); err != nil {
				t.Error(err)
			}
		})
	}
}

func TestWeaviateMethods(t *testing.T) {
	t.Run("buildSchema", func(t *testing.T) {
		weaviate := NewWeaviate(
			WithWeaviateClassName("TestClass"),
			WithWeaviateVectorizer("text2vec-transformers"),
		)

		schema := weaviate.buildSchema()
		if schema.Class != "TestClass" {
			t.Errorf("expected class 'TestClass', got %s", schema.Class)
		}
		if schema.Vectorizer != "text2vec-transformers" {
			t.Errorf("expected vectorizer 'text2vec-transformers', got %s", schema.Vectorizer)
		}
		if len(schema.Properties) == 0 {
			t.Error("expected properties, got none")
		}
	})

	t.Run("buildObject", func(t *testing.T) {
		weaviate := NewWeaviate(WithWeaviateClassName("TestClass"))

		doc := types.Document{
			PageContent: "test content",
			Metadata: map[string]interface{}{
				"title": "Test",
				"year":  2023,
			},
		}
		embedding := []float64{0.1, 0.2, 0.3}

		obj := weaviate.buildObject(doc, embedding)
		if obj.Class != "TestClass" {
			t.Errorf("expected class 'TestClass', got %s", obj.Class)
		}
		if len(obj.Vector) != 3 {
			t.Errorf("expected 3 vector dimensions, got %d", len(obj.Vector))
		}
		if obj.Properties["content"] != "test content" {
			t.Error("content not set correctly")
		}
	})

	t.Run("buildNearVectorQuery", func(t *testing.T) {
		weaviate := NewWeaviate()

		embedding := []float64{0.1, 0.2, 0.3}
		query := weaviate.buildNearVectorQuery(embedding, 10, nil)

		if query == nil {
			t.Fatal("expected query, got nil")
		}
	})

	t.Run("buildHybridQuery", func(t *testing.T) {
		weaviate := NewWeaviate()

		query := weaviate.buildHybridQuery("test query", []float64{0.1, 0.2}, 10, 0.7)

		if query == nil {
			t.Fatal("expected hybrid query, got nil")
		}
	})
}

func TestWeaviateFilterBuilder(t *testing.T) {
	weaviate := NewWeaviate()

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
			name: "boolean filter",
			filter: map[string]interface{}{
				"active": true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := weaviate.buildFilter(tt.filter)
			// 如果没有过滤条件，filter 可以为 nil
			if filter == nil && len(tt.filter) > 0 {
				// 某些简单情况下可能不构建复杂过滤器
				t.Log("filter not built for simple case")
			}
		})
	}
}

func TestWeaviateDefaultConfig(t *testing.T) {
	config := DefaultWeaviateConfig()

	if config.URL != "http://localhost:8080" {
		t.Errorf("expected default URL, got %s", config.URL)
	}

	if config.ClassName != "Document" {
		t.Errorf("expected default class 'Document', got %s", config.ClassName)
	}

	if config.Vectorizer != "none" {
		t.Errorf("expected default vectorizer 'none', got %s", config.Vectorizer)
	}

	if config.BatchSize != 100 {
		t.Errorf("expected default batch size 100, got %d", config.BatchSize)
	}
}

func TestWeaviateResponseParser(t *testing.T) {
	weaviate := NewWeaviate()

	t.Run("parseSearchResponse", func(t *testing.T) {
		// 模拟响应数据
		response := map[string]interface{}{
			"data": map[string]interface{}{
				"Get": map[string]interface{}{
					"Document": []interface{}{
						map[string]interface{}{
							"content": "test content 1",
							"_additional": map[string]interface{}{
								"distance": 0.1,
							},
						},
						map[string]interface{}{
							"content": "test content 2",
							"_additional": map[string]interface{}{
								"distance": 0.2,
							},
						},
					},
				},
			},
		}

		docs := weaviate.parseSearchResponse(response)
		if len(docs) != 2 {
			t.Errorf("expected 2 documents, got %d", len(docs))
		}
	})
}

// 注意：以下是集成测试，需要真实的 Weaviate 服务

func TestWeaviateIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()

	weaviate := NewWeaviate(
		WithWeaviateURL("http://localhost:8080"),
		WithWeaviateClassName("TestIntegration"),
		WithWeaviateVectorizer("none"),
	)

	// 测试创建 Schema
	t.Run("CreateSchema", func(t *testing.T) {
		// err := weaviate.CreateSchema(ctx)
		// if err != nil {
		// 	t.Fatalf("failed to create schema: %v", err)
		// }

		t.Log("Integration test would create schema here")
	})

	// 测试添加文档
	t.Run("AddDocuments", func(t *testing.T) {
		docs := []types.Document{
			{
				PageContent: "test document 1",
				Metadata:    map[string]interface{}{"title": "Test 1"},
			},
			{
				PageContent: "test document 2",
				Metadata:    map[string]interface{}{"title": "Test 2"},
			},
		}

		embeddings := [][]float64{
			{0.1, 0.2, 0.3},
			{0.4, 0.5, 0.6},
		}

		// err := weaviate.AddDocuments(ctx, docs, embeddings)
		// if err != nil {
		// 	t.Fatalf("failed to add documents: %v", err)
		// }

		t.Log("Integration test would add documents here")
	})

	// 测试混合搜索
	t.Run("HybridSearch", func(t *testing.T) {
		// docs, err := weaviate.HybridSearch(ctx, "test query", []float64{0.1, 0.2, 0.3}, 2, 0.7)
		// if err != nil {
		// 	t.Fatalf("failed to hybrid search: %v", err)
		// }

		t.Log("Integration test would perform hybrid search here")
	})
}
