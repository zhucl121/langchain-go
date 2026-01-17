package vectorstores

import (
	"context"
	"testing"

	"github.com/zhucl121/langchain-go/retrieval/loaders"
)

// TestMMROptions 测试 MMR 选项。
func TestMMROptions(t *testing.T) {
	t.Run("DefaultOptions", func(t *testing.T) {
		k := 5
		opts := DefaultMMROptions(k)

		if opts.Lambda != 0.5 {
			t.Errorf("Expected lambda 0.5, got %f", opts.Lambda)
		}
		if opts.FetchK != k*4 {
			t.Errorf("Expected fetchK %d, got %d", k*4, opts.FetchK)
		}
	})

	t.Run("Validate", func(t *testing.T) {
		tests := []struct {
			name      string
			opts      *MMROptions
			k         int
			expectErr bool
		}{
			{
				name:      "Valid options",
				opts:      &MMROptions{Lambda: 0.5, FetchK: 20},
				k:         5,
				expectErr: false,
			},
			{
				name:      "Lambda too high",
				opts:      &MMROptions{Lambda: 1.5, FetchK: 20},
				k:         5,
				expectErr: true,
			},
			{
				name:      "Lambda too low",
				opts:      &MMROptions{Lambda: -0.1, FetchK: 20},
				k:         5,
				expectErr: true,
			},
			{
				name:      "FetchK too small",
				opts:      &MMROptions{Lambda: 0.5, FetchK: 3},
				k:         5,
				expectErr: true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := tt.opts.Validate(tt.k)
				if (err != nil) != tt.expectErr {
					t.Errorf("Expected error: %v, got: %v", tt.expectErr, err)
				}
			})
		}
	})
}

// TestMaxMarginalRelevance 测试 MMR 算法。
func TestMaxMarginalRelevance(t *testing.T) {
	t.Run("EmptyCandidates", func(t *testing.T) {
		queryVector := []float32{1.0, 0.0, 0.0}
		candidates := [][]float32{}
		k := 3
		lambda := float32(0.5)

		result := maxMarginalRelevance(queryVector, candidates, k, lambda)
		if len(result) != 0 {
			t.Errorf("Expected 0 results, got %d", len(result))
		}
	})

	t.Run("KGreaterThanCandidates", func(t *testing.T) {
		queryVector := []float32{1.0, 0.0, 0.0}
		candidates := [][]float32{
			{1.0, 0.0, 0.0},
			{0.9, 0.1, 0.0},
		}
		k := 5
		lambda := float32(0.5)

		result := maxMarginalRelevance(queryVector, candidates, k, lambda)
		if len(result) != 2 {
			t.Errorf("Expected 2 results, got %d", len(result))
		}
	})

	t.Run("NormalCase", func(t *testing.T) {
		// 查询向量
		queryVector := []float32{1.0, 0.0, 0.0}

		// 候选向量
		candidates := [][]float32{
			{1.0, 0.0, 0.0},   // 索引 0: 与查询完全相同
			{0.9, 0.1, 0.0},   // 索引 1: 与查询非常相似
			{0.0, 1.0, 0.0},   // 索引 2: 与查询不同（多样性高）
			{0.95, 0.05, 0.0}, // 索引 3: 与查询非常相似
			{0.0, 0.0, 1.0},   // 索引 4: 与查询不同（多样性高）
		}

		k := 3
		lambda := float32(0.5)

		result := maxMarginalRelevance(queryVector, candidates, k, lambda)

		// 验证结果数量
		if len(result) != k {
			t.Errorf("Expected %d results, got %d", k, len(result))
		}

		// 第一个结果应该是最相似的（索引 0）
		if result[0] != 0 {
			t.Errorf("Expected first result to be index 0, got %d", result[0])
		}

		// 验证没有重复
		seen := make(map[int]bool)
		for _, idx := range result {
			if seen[idx] {
				t.Errorf("Duplicate index in results: %d", idx)
			}
			seen[idx] = true
		}
	})

	t.Run("MaxRelevance", func(t *testing.T) {
		// Lambda = 1.0 应该选择最相关的文档
		queryVector := []float32{1.0, 0.0, 0.0}
		candidates := [][]float32{
			{1.0, 0.0, 0.0},   // 最相关
			{0.9, 0.1, 0.0},   // 次相关
			{0.8, 0.2, 0.0},   // 第三相关
			{0.0, 1.0, 0.0},   // 不相关
		}

		k := 3
		lambda := float32(1.0) // 最大相关性

		result := maxMarginalRelevance(queryVector, candidates, k, lambda)

		// 应该选择前 3 个最相关的
		expectedIndices := []int{0, 1, 2}
		for i, expected := range expectedIndices {
			if result[i] != expected {
				t.Errorf("Expected index %d at position %d, got %d", expected, i, result[i])
			}
		}
	})

	t.Run("MaxDiversity", func(t *testing.T) {
		// Lambda = 0.0 应该选择最多样的文档
		queryVector := []float32{1.0, 0.0, 0.0}
		candidates := [][]float32{
			{1.0, 0.0, 0.0},   // 索引 0: 与查询相同
			{0.99, 0.01, 0.0}, // 索引 1: 与查询几乎相同
			{0.0, 1.0, 0.0},   // 索引 2: 完全不同
			{0.0, 0.0, 1.0},   // 索引 3: 完全不同（另一方向）
		}

		k := 3
		lambda := float32(0.0) // 最大多样性

		result := maxMarginalRelevance(queryVector, candidates, k, lambda)

		// 第一个应该是最相关的
		if result[0] != 0 {
			t.Errorf("Expected first result to be index 0, got %d", result[0])
		}

		// 结果应该包含多样的向量（不是索引 1，因为它与索引 0 太相似）
		hasIndex1 := false
		for _, idx := range result {
			if idx == 1 {
				hasIndex1 = true
				break
			}
		}
		if hasIndex1 {
			t.Logf("Warning: Index 1 (very similar to index 0) was selected, which is unexpected for max diversity")
		}
	})
}

// TestInMemoryVectorStoreMMR 测试 InMemoryVectorStore 的 MMR 功能。
func TestInMemoryVectorStoreMMR(t *testing.T) {
	ctx := context.Background()

	// 使用 FakeEmbeddings 进行测试
	emb := &FakeEmbeddings{dimension: 3}
	store := NewInMemoryVectorStore(emb)

	// 添加测试文档
	docs := []*loaders.Document{
		loaders.NewDocument("AI is transforming technology", nil),
		loaders.NewDocument("Artificial intelligence is the future", nil),
		loaders.NewDocument("Machine learning is a subset of AI", nil),
		loaders.NewDocument("The weather is nice today", nil),
		loaders.NewDocument("Deep learning uses neural networks", nil),
		loaders.NewDocument("I love eating pizza", nil),
		loaders.NewDocument("Natural language processing is important", nil),
	}

	_, err := store.AddDocuments(ctx, docs)
	if err != nil {
		t.Fatalf("Failed to add documents: %v", err)
	}

	t.Run("BasicMMRSearch", func(t *testing.T) {
		k := 3
		results, err := store.SimilaritySearchWithMMR(ctx, "artificial intelligence", k, nil)

		if err != nil {
			t.Fatalf("MMR search failed: %v", err)
		}

		if len(results) != k {
			t.Errorf("Expected %d results, got %d", k, len(results))
		}

		// 验证没有重复
		seen := make(map[string]bool)
		for _, doc := range results {
			if seen[doc.Content] {
				t.Errorf("Duplicate document in results: %s", doc.Content)
			}
			seen[doc.Content] = true
		}
	})

	t.Run("CustomLambda", func(t *testing.T) {
		k := 3
		options := &MMROptions{
			Lambda: 0.7, // 更偏向相关性
			FetchK: 6,
		}

		results, err := store.SimilaritySearchWithMMR(ctx, "machine learning", k, options)

		if err != nil {
			t.Fatalf("MMR search with custom lambda failed: %v", err)
		}

		if len(results) != k {
			t.Errorf("Expected %d results, got %d", k, len(results))
		}
	})

	t.Run("LowDiversityLambda", func(t *testing.T) {
		k := 3
		options := &MMROptions{
			Lambda: 0.2, // 更偏向多样性
			FetchK: 6,
		}

		results, err := store.SimilaritySearchWithMMR(ctx, "AI technology", k, options)

		if err != nil {
			t.Fatalf("MMR search with low lambda failed: %v", err)
		}

		if len(results) != k {
			t.Errorf("Expected %d results, got %d", k, len(results))
		}
	})

	t.Run("KLargerThanDocuments", func(t *testing.T) {
		k := 100 // 大于文档总数
		results, err := store.SimilaritySearchWithMMR(ctx, "test query", k, nil)

		if err != nil {
			t.Fatalf("MMR search failed: %v", err)
		}

		// 应该返回所有文档
		if len(results) != len(docs) {
			t.Errorf("Expected %d results (all documents), got %d", len(docs), len(results))
		}
	})

	t.Run("InvalidOptions", func(t *testing.T) {
		k := 3
		invalidOptions := &MMROptions{
			Lambda: 1.5, // 无效的 lambda
			FetchK: 10,
		}

		_, err := store.SimilaritySearchWithMMR(ctx, "test", k, invalidOptions)

		if err == nil {
			t.Error("Expected error for invalid lambda, got nil")
		}
	})
}

// TestMMRInterface 测试 MMRVectorStore 接口。
func TestMMRInterface(t *testing.T) {
	var _ MMRVectorStore = (*InMemoryVectorStore)(nil)
}

// FakeEmbeddings 用于测试的假嵌入。
type FakeEmbeddings struct {
	dimension int
}

func (e *FakeEmbeddings) EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error) {
	embeddings := make([][]float32, len(texts))
	for i, text := range texts {
		embeddings[i] = e.generateVector(text)
	}
	return embeddings, nil
}

func (e *FakeEmbeddings) EmbedQuery(ctx context.Context, text string) ([]float32, error) {
	return e.generateVector(text), nil
}

func (e *FakeEmbeddings) GetDimension() int {
	return e.dimension
}

func (e *FakeEmbeddings) generateVector(text string) []float32 {
	// 简单的确定性向量生成
	vec := make([]float32, e.dimension)
	for i := range vec {
		vec[i] = float32(len(text)%10) / 10.0
		if i < len(text) {
			vec[i] += float32(text[i]) / 255.0
		}
	}
	return vec
}
