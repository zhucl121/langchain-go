package benchmark_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/zhucl121/langchain-go/pkg/types"
	"github.com/zhucl121/langchain-go/retrieval/graphdb"
	"github.com/zhucl121/langchain-go/retrieval/graphdb/builder"
	"github.com/zhucl121/langchain-go/retrieval/graphdb/mock"
	"github.com/zhucl121/langchain-go/retrieval/retrievers/graphrag"
	"github.com/zhucl121/langchain-go/retrieval/vectorstores"
)

// BenchmarkGraphDB_AddNode 基准测试：添加节点
func BenchmarkGraphDB_AddNode(b *testing.B) {
	ctx := context.Background()
	graphDB := mock.NewMockGraphDB()
	graphDB.Connect(ctx)
	defer graphDB.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		node := &graphdb.Node{
			ID:    fmt.Sprintf("node-%d", i),
			Type:  "Test",
			Label: fmt.Sprintf("Test Node %d", i),
		}
		graphDB.AddNode(ctx, node)
	}
}

// BenchmarkGraphDB_AddNodesBatch 基准测试：批量添加节点
func BenchmarkGraphDB_AddNodesBatch(b *testing.B) {
	ctx := context.Background()
	graphDB := mock.NewMockGraphDB()
	graphDB.Connect(ctx)
	defer graphDB.Close()

	// 准备批量节点
	batchSizes := []int{10, 100, 1000}

	for _, size := range batchSizes {
		b.Run(fmt.Sprintf("BatchSize-%d", size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				nodes := make([]*graphdb.Node, size)
				for j := 0; j < size; j++ {
					nodes[j] = &graphdb.Node{
						ID:    fmt.Sprintf("batch-node-%d-%d", i, j),
						Type:  "Test",
						Label: fmt.Sprintf("Batch Node %d-%d", i, j),
					}
				}
				// 批量添加节点（逐个添加）
				for _, node := range nodes {
					graphDB.AddNode(ctx, node)
				}
			}
		})
	}
}

// BenchmarkGraphDB_Traverse 基准测试：图遍历
func BenchmarkGraphDB_Traverse(b *testing.B) {
	ctx := context.Background()
	graphDB := mock.NewMockGraphDB()
	graphDB.Connect(ctx)
	defer graphDB.Close()

	// 构建测试图：100个节点，链式连接
	for i := 0; i < 100; i++ {
		node := &graphdb.Node{
			ID:    fmt.Sprintf("node-%d", i),
			Type:  "Test",
			Label: fmt.Sprintf("Node %d", i),
		}
		graphDB.AddNode(ctx, node)

		if i > 0 {
			edge := &graphdb.Edge{
				ID:       fmt.Sprintf("edge-%d", i),
				Source:   fmt.Sprintf("node-%d", i-1),
				Target:   fmt.Sprintf("node-%d", i),
				Type:     "CONNECTS",
				Directed: true,
			}
			graphDB.AddEdge(ctx, edge)
		}
	}

	depths := []int{2, 5, 10}

	for _, depth := range depths {
		b.Run(fmt.Sprintf("Depth-%d", depth), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				graphDB.Traverse(ctx, "node-0", graphdb.TraverseOptions{
					MaxDepth:  depth,
					Strategy:  graphdb.StrategyBFS,
					Direction: graphdb.DirectionBoth,
				})
			}
		})
	}
}

// BenchmarkKGBuilder_BuildBatch 基准测试：批量构建知识图谱
func BenchmarkKGBuilder_BuildBatch(b *testing.B) {
	ctx := context.Background()
	graphDB := mock.NewMockGraphDB()
	graphDB.Connect(ctx)
	defer graphDB.Close()

	mockEntityExtractor := &mockEntityExtractor{
		entities: []builder.Entity{
			{ID: "entity-1", Name: "Test Entity", Type: "Test"},
		},
	}

	mockRelExtractor := &mockRelationExtractor{
		relations: []builder.Relation{},
	}

	mockEmbed := builder.NewMockEmbedder(384)

	kgBuilder, _ := builder.NewKGBuilder(builder.KGBuilderConfig{
		GraphDB:           graphDB,
		EntityExtractor:   mockEntityExtractor,
		RelationExtractor: mockRelExtractor,
		Embedder:          mockEmbed,
		EnableEmbedding:   true,
	})

	textSizes := []int{5, 10, 50}

	for _, size := range textSizes {
		b.Run(fmt.Sprintf("Texts-%d", size), func(b *testing.B) {
			texts := make([]string, size)
			for i := 0; i < size; i++ {
				texts[i] = fmt.Sprintf("Test document %d with some content.", i)
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = kgBuilder.BuildBatch(ctx, texts)
			}
		})
	}
}

// BenchmarkVectorStore_AddDocuments 基准测试：向量存储添加文档
func BenchmarkVectorStore_AddDocuments(b *testing.B) {
	ctx := context.Background()

	docSizes := []int{10, 100, 1000}

	for _, size := range docSizes {
		b.Run(fmt.Sprintf("Docs-%d", size), func(b *testing.B) {
			docs := make([]*types.Document, size)
			for i := 0; i < size; i++ {
				docs[i] = types.NewDocument(
					fmt.Sprintf("Document %d content", i),
					map[string]any{"id": i},
				)
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				// 创建新的 store 避免累积
				mockEmbed := newMockEmbeddings(384)
				store := vectorstores.NewInMemoryVectorStore(mockEmbed)
				_, _ = store.AddDocuments(ctx, docs)
			}
		})
	}
}

// BenchmarkVectorStore_SimilaritySearch 基准测试：相似度搜索
func BenchmarkVectorStore_SimilaritySearch(b *testing.B) {
	ctx := context.Background()
	mockEmbed := newMockEmbeddings(384)
	vectorStore := vectorstores.NewInMemoryVectorStore(mockEmbed)

	// 准备1000个文档
	docs := make([]*types.Document, 1000)
	for i := 0; i < 1000; i++ {
		docs[i] = types.NewDocument(
			fmt.Sprintf("Document %d with various content for testing", i),
			map[string]any{"id": i},
		)
	}
	vectorStore.AddDocuments(ctx, docs)

	topKs := []int{5, 10, 50}

	for _, k := range topKs {
		b.Run(fmt.Sprintf("TopK-%d", k), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				vectorStore.SimilaritySearch(ctx, "test query", k)
			}
		})
	}
}

// BenchmarkGraphRAG_Search 基准测试：GraphRAG 混合检索
func BenchmarkGraphRAG_Search(b *testing.B) {
	ctx := context.Background()

	// 设置环境
	graphDB := mock.NewMockGraphDB()
	graphDB.Connect(ctx)
	defer graphDB.Close()

	// 添加100个节点
	for i := 0; i < 100; i++ {
		node := &graphdb.Node{
			ID:    fmt.Sprintf("node-%d", i),
			Type:  "Test",
			Label: fmt.Sprintf("Node %d", i),
		}
		graphDB.AddNode(ctx, node)
	}

	mockEmbed := newMockEmbeddings(384)
	vectorStore := vectorstores.NewInMemoryVectorStore(mockEmbed)

	// 添加100个文档
	docs := make([]*types.Document, 100)
	for i := 0; i < 100; i++ {
		docs[i] = types.NewDocument(
			fmt.Sprintf("Document %d content", i),
			map[string]any{"id": i},
		)
	}
	vectorStore.AddDocuments(ctx, docs)

	mockEntityExtractor := &mockEntityExtractor{
		entities: []builder.Entity{
			{ID: "node-0", Name: "Test", Type: "Test"},
		},
	}

	config := graphrag.DefaultConfig(graphDB, vectorStore)
	config.EntityExtractor = mockEntityExtractor

	retriever, _ := graphrag.NewGraphRAGRetriever(config)

	modes := []graphrag.SearchMode{
		graphrag.SearchModeHybrid,
		graphrag.SearchModeVector,
		graphrag.SearchModeGraph,
	}

	for _, mode := range modes {
		b.Run(fmt.Sprintf("Mode-%s", mode), func(b *testing.B) {
			opts := graphrag.SearchOptions{
				Mode: mode,
				K:    10,
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				retriever.Search(ctx, "test query", opts)
			}
		})
	}
}

// BenchmarkGraphRAG_FusionStrategies 基准测试：融合策略
func BenchmarkGraphRAG_FusionStrategies(b *testing.B) {
	ctx := context.Background()

	graphDB, _, retriever := setupBenchmarkEnvironment(b, ctx)
	defer graphDB.Close()

	strategies := []graphrag.FusionStrategy{
		graphrag.FusionStrategyWeighted,
		graphrag.FusionStrategyRRF,
		graphrag.FusionStrategyMax,
		graphrag.FusionStrategyMin,
	}

	for _, strategy := range strategies {
		b.Run(fmt.Sprintf("Strategy-%s", strategy), func(b *testing.B) {
			opts := graphrag.SearchOptions{
				Mode:           graphrag.SearchModeHybrid,
				K:              10,
				FusionStrategy: strategy,
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = retriever.Search(ctx, "test query", opts)
			}
		})
	}
}

// BenchmarkGraphRAG_RerankStrategies 基准测试：重排序策略
func BenchmarkGraphRAG_RerankStrategies(b *testing.B) {
	ctx := context.Background()

	graphDB, _, retriever := setupBenchmarkEnvironment(b, ctx)
	defer graphDB.Close()

	strategies := []graphrag.RerankStrategy{
		graphrag.RerankStrategyScore,
		graphrag.RerankStrategyDiversity,
		graphrag.RerankStrategyMMR,
	}

	for _, strategy := range strategies {
		b.Run(fmt.Sprintf("Strategy-%s", strategy), func(b *testing.B) {
			opts := graphrag.SearchOptions{
				Mode:           graphrag.SearchModeHybrid,
				K:              20,
				RerankStrategy: strategy,
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = retriever.Search(ctx, "test query", opts)
			}
		})
	}
}

// BenchmarkGraphRAG_ContextAugmentation 基准测试：上下文增强
func BenchmarkGraphRAG_ContextAugmentation(b *testing.B) {
	ctx := context.Background()

	graphDB, _, retriever := setupBenchmarkEnvironment(b, ctx)
	defer graphDB.Close()

	b.Run("WithAugmentation", func(b *testing.B) {
		opts := graphrag.SearchOptions{
			Mode:                      graphrag.SearchModeHybrid,
			K:                         10,
			EnableContextAugmentation: true,
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = retriever.Search(ctx, "test query", opts)
		}
	})

	b.Run("WithoutAugmentation", func(b *testing.B) {
		opts := graphrag.SearchOptions{
			Mode:                      graphrag.SearchModeHybrid,
			K:                         10,
			EnableContextAugmentation: false,
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = retriever.Search(ctx, "test query", opts)
		}
	})
}

// Helper functions

func setupBenchmarkEnvironment(b *testing.B, ctx context.Context) (graphdb.GraphDB, vectorstores.VectorStore, *graphrag.GraphRAGRetriever) {
	graphDB := mock.NewMockGraphDB()
	graphDB.Connect(ctx)

	// 添加50个节点
	for i := 0; i < 50; i++ {
		node := &graphdb.Node{
			ID:    fmt.Sprintf("node-%d", i),
			Type:  "Test",
			Label: fmt.Sprintf("Node %d", i),
		}
		graphDB.AddNode(ctx, node)
	}

	mockEmbed := newMockEmbeddings(384)
	vectorStore := vectorstores.NewInMemoryVectorStore(mockEmbed)

	// 添加50个文档
	docs := make([]*types.Document, 50)
	for i := 0; i < 50; i++ {
		docs[i] = types.NewDocument(
			fmt.Sprintf("Document %d content for benchmarking", i),
			map[string]any{"id": i},
		)
	}
	vectorStore.AddDocuments(ctx, docs)

	mockEntityExtractor := &mockEntityExtractor{
		entities: []builder.Entity{
			{ID: "node-0", Name: "Test", Type: "Test"},
		},
	}

	config := graphrag.DefaultConfig(graphDB, vectorStore)
	config.EntityExtractor = mockEntityExtractor

	retriever, _ := graphrag.NewGraphRAGRetriever(config)

	return graphDB, vectorStore, retriever
}

// Mock types

type mockEntityExtractor struct {
	entities []builder.Entity
}

func (m *mockEntityExtractor) Extract(ctx context.Context, text string) ([]builder.Entity, error) {
	return m.entities, nil
}

func (m *mockEntityExtractor) ExtractWithSchema(ctx context.Context, text string, schema *builder.EntitySchema) ([]builder.Entity, error) {
	return m.entities, nil
}

type mockRelationExtractor struct {
	relations []builder.Relation
}

func (m *mockRelationExtractor) Extract(ctx context.Context, text string, entities []builder.Entity) ([]builder.Relation, error) {
	return m.relations, nil
}

func (m *mockRelationExtractor) ExtractWithSchema(ctx context.Context, text string, entities []builder.Entity, schema *builder.RelationSchema) ([]builder.Relation, error) {
	return m.relations, nil
}

type mockEmbeddings struct {
	dimension int
}

func newMockEmbeddings(dimension int) *mockEmbeddings {
	return &mockEmbeddings{dimension: dimension}
}

func (m *mockEmbeddings) EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error) {
	results := make([][]float32, len(texts))
	for i := range texts {
		results[i] = make([]float32, m.dimension)
		for j := range results[i] {
			results[i][j] = float32(i+j) / 100.0
		}
	}
	return results, nil
}

func (m *mockEmbeddings) EmbedQuery(ctx context.Context, text string) ([]float32, error) {
	result := make([]float32, m.dimension)
	for i := range result {
		result[i] = float32(i) / 100.0
	}
	return result, nil
}

func (m *mockEmbeddings) GetDimension() int {
	return m.dimension
}
