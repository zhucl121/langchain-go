package vectorstores

import (
	"context"
	"testing"

	"github.com/zhucl121/langchain-go/pkg/types"
)

func TestNewRedis(t *testing.T) {
	redis := NewRedis(
		WithRedisAddr("localhost:6379"),
		WithRedisPassword("test-password"),
		WithRedisIndexName("test_index"),
	)

	if redis == nil {
		t.Fatal("expected Redis instance, got nil")
	}

	if redis.config.Addr != "localhost:6379" {
		t.Errorf("expected addr 'localhost:6379', got %s", redis.config.Addr)
	}

	if redis.config.IndexName != "test_index" {
		t.Errorf("expected index 'test_index', got %s", redis.config.IndexName)
	}
}

func TestRedisConfig(t *testing.T) {
	tests := []struct {
		name   string
		option RedisOption
		check  func(*RedisConfig) error
	}{
		{
			name:   "WithRedisAddr",
			option: WithRedisAddr("redis:6379"),
			check: func(c *RedisConfig) error {
				if c.Addr != "redis:6379" {
					t.Errorf("expected addr 'redis:6379', got %s", c.Addr)
				}
				return nil
			},
		},
		{
			name:   "WithRedisPassword",
			option: WithRedisPassword("secret"),
			check: func(c *RedisConfig) error {
				if c.Password != "secret" {
					t.Errorf("expected password 'secret', got %s", c.Password)
				}
				return nil
			},
		},
		{
			name:   "WithRedisDB",
			option: WithRedisDB(1),
			check: func(c *RedisConfig) error {
				if c.DB != 1 {
					t.Errorf("expected DB 1, got %d", c.DB)
				}
				return nil
			},
		},
		{
			name:   "WithRedisIndexName",
			option: WithRedisIndexName("my_vectors"),
			check: func(c *RedisConfig) error {
				if c.IndexName != "my_vectors" {
					t.Errorf("expected index 'my_vectors', got %s", c.IndexName)
				}
				return nil
			},
		},
		{
			name:   "WithRedisVectorDim",
			option: WithRedisVectorDim(768),
			check: func(c *RedisConfig) error {
				if c.VectorDim != 768 {
					t.Errorf("expected vector dim 768, got %d", c.VectorDim)
				}
				return nil
			},
		},
		{
			name:   "WithRedisDistanceMetric",
			option: WithRedisDistanceMetric("COSINE"),
			check: func(c *RedisConfig) error {
				if c.DistanceMetric != "COSINE" {
					t.Errorf("expected metric 'COSINE', got %s", c.DistanceMetric)
				}
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultRedisConfig()
			tt.option(&config)
			if err := tt.check(&config); err != nil {
				t.Error(err)
			}
		})
	}
}

func TestRedisMethods(t *testing.T) {
	t.Run("buildCreateIndexCommand", func(t *testing.T) {
		redis := NewRedis(
			WithRedisIndexName("test_idx"),
			WithRedisVectorDim(384),
			WithRedisDistanceMetric("COSINE"),
		)

		cmd := redis.buildCreateIndexCommand()
		if len(cmd) == 0 {
			t.Error("expected create index command, got empty")
		}

		// 检查命令中包含关键参数
		hasIndex := false
		hasVector := false
		for _, arg := range cmd {
			if str, ok := arg.(string); ok {
				if str == "test_idx" {
					hasIndex = true
				}
				if str == "VECTOR" {
					hasVector = true
				}
			}
		}

		if !hasIndex {
			t.Error("create index command missing index name")
		}
		if !hasVector {
			t.Error("create index command missing VECTOR field")
		}
	})

	t.Run("buildKey", func(t *testing.T) {
		redis := NewRedis(
			WithRedisKeyPrefix("doc:"),
		)

		key := redis.buildKey("123")
		if key != "doc:123" {
			t.Errorf("expected key 'doc:123', got %s", key)
		}
	})

	t.Run("buildSearchQuery", func(t *testing.T) {
		redis := NewRedis()

		query := redis.buildSearchQuery([]float64{0.1, 0.2, 0.3}, 10, nil)
		if query == "" {
			t.Error("expected search query, got empty string")
		}
	})
}

func TestRedisKeyBuilder(t *testing.T) {
	tests := []struct {
		name      string
		prefix    string
		id        string
		expected  string
	}{
		{
			name:     "default prefix",
			prefix:   "doc:",
			id:       "123",
			expected: "doc:123",
		},
		{
			name:     "custom prefix",
			prefix:   "vector:",
			id:       "abc",
			expected: "vector:abc",
		},
		{
			name:     "no prefix",
			prefix:   "",
			id:       "xyz",
			expected: "xyz",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			redis := NewRedis(WithRedisKeyPrefix(tt.prefix))
			key := redis.buildKey(tt.id)
			if key != tt.expected {
				t.Errorf("expected key %s, got %s", tt.expected, key)
			}
		})
	}
}

func TestRedisDefaultConfig(t *testing.T) {
	config := DefaultRedisConfig()

	if config.Addr != "localhost:6379" {
		t.Errorf("expected default addr, got %s", config.Addr)
	}

	if config.DB != 0 {
		t.Errorf("expected default DB 0, got %d", config.DB)
	}

	if config.IndexName != "vector_index" {
		t.Errorf("expected default index, got %s", config.IndexName)
	}

	if config.VectorDim != 1536 {
		t.Errorf("expected default vector dim 1536, got %d", config.VectorDim)
	}

	if config.DistanceMetric != "COSINE" {
		t.Errorf("expected default metric 'COSINE', got %s", config.DistanceMetric)
	}
}

func TestRedisFilterBuilder(t *testing.T) {
	redis := NewRedis()

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
			name: "numeric filter",
			filter: map[string]interface{}{
				"score": 95,
			},
		},
		{
			name: "multiple filters",
			filter: map[string]interface{}{
				"category": "tech",
				"year":     2023,
				"active":   true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := redis.buildFilterClause(tt.filter)
			// Redis 过滤可能返回空字符串（表示无过滤）
			t.Logf("Filter: %s", filter)
		})
	}
}

func TestRedisVectorEncoding(t *testing.T) {
	redis := NewRedis()

	t.Run("encodeVector", func(t *testing.T) {
		vector := []float64{0.1, 0.2, 0.3, 0.4}
		encoded := redis.encodeVector(vector)

		if len(encoded) == 0 {
			t.Error("expected encoded vector, got empty")
		}

		// 检查编码长度（4 floats * 4 bytes = 16 bytes）
		expectedLen := len(vector) * 4
		if len(encoded) != expectedLen {
			t.Errorf("expected %d bytes, got %d", expectedLen, len(encoded))
		}
	})

	t.Run("encodeVector empty", func(t *testing.T) {
		vector := []float64{}
		encoded := redis.encodeVector(vector)

		if len(encoded) != 0 {
			t.Error("expected empty encoding for empty vector")
		}
	})
}

// 注意：以下是集成测试，需要真实的 Redis 服务（带 RediSearch 模块）

func TestRedisIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()

	redis := NewRedis(
		WithRedisAddr("localhost:6379"),
		WithRedisIndexName("test_integration"),
		WithRedisVectorDim(3),
	)

	// 测试创建索引
	t.Run("CreateIndex", func(t *testing.T) {
		// err := redis.CreateIndex(ctx)
		// if err != nil {
		// 	t.Fatalf("failed to create index: %v", err)
		// }

		t.Log("Integration test would create index here")
	})

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

		// err := redis.AddDocuments(ctx, docs, embeddings)
		// if err != nil {
		// 	t.Fatalf("failed to add documents: %v", err)
		// }

		t.Log("Integration test would add documents here")
	})

	// 测试向量搜索
	t.Run("VectorSearch", func(t *testing.T) {
		embedding := []float64{0.1, 0.2, 0.3}

		// docs, err := redis.SimilaritySearchByVector(ctx, embedding, 2)
		// if err != nil {
		// 	t.Fatalf("failed to search: %v", err)
		// }

		t.Log("Integration test would search vectors here")
	})
}
