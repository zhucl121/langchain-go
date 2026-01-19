package vectorstores

import (
	"context"
	"testing"

	"github.com/zhucl121/langchain-go/retrieval/loaders"
)

// MockEmbedder 用于测试的模拟嵌入器
type MockChromaEmbedder struct{}

func (m *MockChromaEmbedder) EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error) {
	// 返回模拟的嵌入向量
	embeddings := make([][]float32, len(texts))
	for i := range texts {
		// 简单的模拟向量
		embeddings[i] = []float32{0.1, 0.2, 0.3, 0.4}
	}
	return embeddings, nil
}

func (m *MockChromaEmbedder) EmbedQuery(ctx context.Context, text string) ([]float32, error) {
	return []float32{0.1, 0.2, 0.3, 0.4}, nil
}

func (m *MockChromaEmbedder) GetDimension() int {
	return 4
}

func (m *MockChromaEmbedder) EmbedQuery(ctx context.Context, text string) ([]float32, error) {
	return []float32{0.1, 0.2, 0.3, 0.4}, nil
}

func TestChromaVectorStore_NewChromaVectorStore(t *testing.T) {
	tests := []struct {
		name      string
		config    ChromaConfig
		embedder  embeddings.Embeddings
		wantError bool
	}{
		{
			name: "valid config",
			config: ChromaConfig{
				URL:            "http://localhost:8000",
				CollectionName: "test_collection",
			},
			embedder:  &MockChromaEmbedder{},
			wantError: false,
		},
		{
			name: "missing URL",
			config: ChromaConfig{
				CollectionName: "test_collection",
			},
			embedder:  &MockChromaEmbedder{},
			wantError: true,
		},
		{
			name: "missing collection name",
			config: ChromaConfig{
				URL: "http://localhost:8000",
			},
			embedder:  &MockChromaEmbedder{},
			wantError: true,
		},
		{
			name: "missing embedder",
			config: ChromaConfig{
				URL:            "http://localhost:8000",
				CollectionName: "test_collection",
			},
			embedder:  nil,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, err := NewChromaVectorStore(tt.config, tt.embedder)
			if tt.wantError {
				if err == nil {
					t.Error("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if store == nil {
					t.Error("expected store, got nil")
				}
			}
		})
	}
}

func TestChromaVectorStore_DefaultConfig(t *testing.T) {
	config := DefaultChromaConfig()

	if config.URL == "" {
		t.Error("expected default URL")
	}

	if config.CollectionName == "" {
		t.Error("expected default collection name")
	}

	if config.DistanceMetric == "" {
		t.Error("expected default distance metric")
	}

	if config.Timeout == 0 {
		t.Error("expected default timeout")
	}
}

func TestChromaVectorStore_ConvertDistance(t *testing.T) {
	tests := []struct {
		name           string
		distanceMetric string
		distance       float32
		wantMin        float32
		wantMax        float32
	}{
		{
			name:           "L2 distance",
			distanceMetric: "l2",
			distance:       1.0,
			wantMin:        0.0,
			wantMax:        1.0,
		},
		{
			name:           "cosine distance",
			distanceMetric: "cosine",
			distance:       0.5,
			wantMin:        0.0,
			wantMax:        1.0,
		},
		{
			name:           "inner product",
			distanceMetric: "ip",
			distance:       0.8,
			wantMin:        0.0,
			wantMax:        1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := ChromaConfig{
				URL:            "http://localhost:8000",
				CollectionName: "test",
				DistanceMetric: tt.distanceMetric,
			}
			store := &ChromaVectorStore{config: config}

			score := store.convertDistance(tt.distance)
			
			if score < tt.wantMin || score > tt.wantMax {
				t.Errorf("score %f out of range [%f, %f]", score, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestChromaVectorStore_GenerateID(t *testing.T) {
	id1 := generateChromaID()
	id2 := generateChromaID()

	if id1 == "" {
		t.Error("expected non-empty ID")
	}

	if id1 == id2 {
		t.Error("expected unique IDs")
	}
}

// 集成测试（需要运行 Chroma 服务器）
// 跳过这些测试，除非设置了环境变量 CHROMA_TEST_ENABLED=1
func TestChromaVectorStore_Integration(t *testing.T) {
	// 检查是否启用集成测试
	// if os.Getenv("CHROMA_TEST_ENABLED") != "1" {
	// 	t.Skip("Skipping integration test. Set CHROMA_TEST_ENABLED=1 to run.")
	// }

	t.Skip("Integration test - requires running Chroma server")

	ctx := context.Background()
	config := ChromaConfig{
		URL:            "http://localhost:8000",
		CollectionName: "test_collection",
	}
	embedder := &MockChromaEmbedder{}

	store, err := NewChromaVectorStore(config, embedder)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	// 初始化
	if err := store.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}

	// 添加文档
	docs := []*loaders.Document{
		{
			Content: "Hello world",
			Metadata: map[string]interface{}{
				"source": "test1",
			},
		},
		{
			Content: "Goodbye world",
			Metadata: map[string]interface{}{
				"source": "test2",
			},
		},
	}

	ids, err := store.AddDocuments(ctx, docs)
	if err != nil {
		t.Fatalf("Failed to add documents: %v", err)
	}

	if len(ids) != len(docs) {
		t.Errorf("Expected %d IDs, got %d", len(docs), len(ids))
	}

	// 相似度搜索
	results, err := store.SimilaritySearch(ctx, "Hello", 2)
	if err != nil {
		t.Fatalf("Failed to search: %v", err)
	}

	if len(results) == 0 {
		t.Error("Expected search results, got none")
	}

	// 清理 - 删除文档
	if err := store.DeleteDocuments(ctx, ids); err != nil {
		t.Fatalf("Failed to delete documents: %v", err)
	}
}
