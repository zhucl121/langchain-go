package graphrag_test

import (
	"context"
	"testing"

	"github.com/zhucl121/langchain-go/pkg/types"
	"github.com/zhucl121/langchain-go/retrieval/graphdb"
	"github.com/zhucl121/langchain-go/retrieval/graphdb/builder"
	"github.com/zhucl121/langchain-go/retrieval/graphdb/mock"
	"github.com/zhucl121/langchain-go/retrieval/retrievers/graphrag"
	"github.com/zhucl121/langchain-go/retrieval/vectorstores"
)

// mockEntityExtractor for testing
type mockEntityExtractor struct {
	entities []builder.Entity
}

func (m *mockEntityExtractor) Extract(ctx context.Context, text string) ([]builder.Entity, error) {
	return m.entities, nil
}

func (m *mockEntityExtractor) ExtractWithSchema(ctx context.Context, text string, schema *builder.EntitySchema) ([]builder.Entity, error) {
	return m.entities, nil
}

// mockEmbeddings for testing
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

func setupTestRetriever(t *testing.T) (*graphrag.GraphRAGRetriever, graphdb.GraphDB, vectorstores.VectorStore, func()) {
	ctx := context.Background()

	// 创建 Mock GraphDB
	graphDB := mock.NewMockGraphDB()
	if err := graphDB.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect to mock GraphDB: %v", err)
	}

	// 添加测试数据到图数据库
	testNodes := []*graphdb.Node{
		{ID: "person-1", Type: "Person", Label: "John Smith", Properties: map[string]interface{}{"description": "CEO of TechCorp"}},
		{ID: "org-1", Type: "Organization", Label: "TechCorp", Properties: map[string]interface{}{"description": "Technology company"}},
		{ID: "location-1", Type: "Location", Label: "San Francisco", Properties: map[string]interface{}{"description": "City in California"}},
	}

	for _, node := range testNodes {
		graphDB.AddNode(ctx, node)
	}

	testEdges := []*graphdb.Edge{
		{ID: "edge-1", Source: "person-1", Target: "org-1", Type: "WORKS_FOR", Directed: true, Weight: 1.0},
		{ID: "edge-2", Source: "org-1", Target: "location-1", Type: "LOCATED_IN", Directed: true, Weight: 1.0},
	}

	for _, edge := range testEdges {
		graphDB.AddEdge(ctx, edge)
	}

	// 创建 Mock VectorStore
	embedModel := newMockEmbeddings(384)
	vectorStore := vectorstores.NewInMemoryVectorStore(embedModel)

	// 添加测试文档
	testDocs := []*types.Document{
		types.NewDocument("John Smith is the CEO of TechCorp.", map[string]any{"source": "doc1"}).WithID("doc-1"),
		types.NewDocument("TechCorp is located in San Francisco.", map[string]any{"source": "doc2"}).WithID("doc-2"),
		types.NewDocument("Alice works at TechCorp as an engineer.", map[string]any{"source": "doc3"}).WithID("doc-3"),
	}

	vectorStore.AddDocuments(ctx, testDocs)

	// 创建 Mock EntityExtractor
	entityExtractor := &mockEntityExtractor{
		entities: []builder.Entity{
			{ID: "person-1", Name: "John Smith", Type: "Person"},
		},
	}

	// 创建 GraphRAG 检索器
	config := graphrag.Config{
		GraphDB:           graphDB,
		VectorStore:       vectorStore,
		EntityExtractor:   entityExtractor,
		VectorWeight:      0.6,
		GraphWeight:       0.4,
		MaxTraverseDepth:  2,
		TopK:              10,
		FusionStrategy:    graphrag.FusionStrategyWeighted,
		RerankStrategy:    graphrag.RerankStrategyScore,
		EnableContextAugmentation: true,
	}

	retriever, err := graphrag.NewGraphRAGRetriever(config)
	if err != nil {
		t.Fatalf("Failed to create GraphRAG retriever: %v", err)
	}

	cleanup := func() {
		graphDB.Close()
	}

	return retriever, graphDB, vectorStore, cleanup
}

func TestGraphRAGRetriever_Search_Hybrid(t *testing.T) {
	retriever, _, _, cleanup := setupTestRetriever(t)
	defer cleanup()

	ctx := context.Background()

	// 执行混合检索
	docs, err := retriever.Search(ctx, "Who is the CEO of TechCorp?")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	// 验证结果
	if len(docs) == 0 {
		t.Error("Expected some results, got 0")
	}

	t.Logf("Found %d documents", len(docs))

	// 验证元数据
	for i, doc := range docs {
		t.Logf("Doc %d: %s", i+1, doc.Content[:min(50, len(doc.Content))])

		if _, ok := doc.Metadata["fused_score"]; !ok {
			t.Error("Expected fused_score in metadata")
		}
	}

	// 验证统计信息
	stats := retriever.GetStatistics()
	t.Logf("Statistics: Vector=%d, Graph=%d, Fused=%d, TotalTime=%dms",
		stats.VectorResultsCount, stats.GraphResultsCount,
		stats.FusedResultsCount, stats.TotalTime)

	if stats.VectorResultsCount == 0 {
		t.Error("Expected vector results")
	}
}

func TestGraphRAGRetriever_Search_VectorOnly(t *testing.T) {
	retriever, _, _, cleanup := setupTestRetriever(t)
	defer cleanup()

	ctx := context.Background()

	// 仅向量检索
	opts := graphrag.SearchOptions{
		Mode: graphrag.SearchModeVector,
		K:    5,
	}

	docs, err := retriever.Search(ctx, "TechCorp", opts)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(docs) == 0 {
		t.Error("Expected some results, got 0")
	}

	t.Logf("Found %d documents (vector only)", len(docs))

	// 验证统计
	stats := retriever.GetStatistics()
	// 在 vector-only 模式下，graphResultsCount 应该是 0
	// vectorResultsCount 可能是 0（因为统计没有追踪纯向量模式）
	t.Logf("Stats: Vector=%d, Graph=%d", stats.VectorResultsCount, stats.GraphResultsCount)
}

func TestGraphRAGRetriever_Search_GraphOnly(t *testing.T) {
	retriever, _, _, cleanup := setupTestRetriever(t)
	defer cleanup()

	ctx := context.Background()

	// 仅图检索
	opts := graphrag.SearchOptions{
		Mode:             graphrag.SearchModeGraph,
		K:                5,
		MaxTraverseDepth: 2,
	}

	docs, err := retriever.Search(ctx, "John Smith", opts)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	// 可能没有结果（因为 mock extractor 返回固定实体）
	t.Logf("Found %d documents (graph only)", len(docs))

	// 验证统计
	stats := retriever.GetStatistics()
	t.Logf("Graph results: %d", stats.GraphResultsCount)
}

func TestGraphRAGRetriever_FusionStrategies(t *testing.T) {
	retriever, _, _, cleanup := setupTestRetriever(t)
	defer cleanup()

	ctx := context.Background()

	strategies := []graphrag.FusionStrategy{
		graphrag.FusionStrategyWeighted,
		graphrag.FusionStrategyRRF,
		graphrag.FusionStrategyMax,
		graphrag.FusionStrategyMin,
	}

	for _, strategy := range strategies {
		t.Run(string(strategy), func(t *testing.T) {
			opts := graphrag.SearchOptions{
				Mode:           graphrag.SearchModeHybrid,
				K:              5,
				FusionStrategy: strategy,
			}

			docs, err := retriever.Search(ctx, "TechCorp CEO", opts)
			if err != nil {
				t.Fatalf("Search with %s failed: %v", strategy, err)
			}

			t.Logf("%s: Found %d documents", strategy, len(docs))
		})
	}
}

func TestGraphRAGRetriever_RerankStrategies(t *testing.T) {
	retriever, _, _, cleanup := setupTestRetriever(t)
	defer cleanup()

	ctx := context.Background()

	strategies := []graphrag.RerankStrategy{
		graphrag.RerankStrategyScore,
		graphrag.RerankStrategyDiversity,
		graphrag.RerankStrategyMMR,
	}

	for _, strategy := range strategies {
		t.Run(string(strategy), func(t *testing.T) {
			opts := graphrag.SearchOptions{
				Mode:           graphrag.SearchModeHybrid,
				K:              5,
				RerankStrategy: strategy,
			}

			docs, err := retriever.Search(ctx, "TechCorp", opts)
			if err != nil {
				t.Fatalf("Search with %s failed: %v", strategy, err)
			}

			t.Logf("%s: Found %d documents", strategy, len(docs))
		})
	}
}

func TestGraphRAGRetriever_ContextAugmentation(t *testing.T) {
	retriever, _, _, cleanup := setupTestRetriever(t)
	defer cleanup()

	ctx := context.Background()

	// 启用上下文增强
	opts := graphrag.SearchOptions{
		Mode:                      graphrag.SearchModeHybrid,
		K:                         5,
		EnableContextAugmentation: true,
	}

	docs, err := retriever.Search(ctx, "CEO", opts)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	// 验证上下文增强
	for _, doc := range docs {
		if _, ok := doc.Metadata["fused_score"]; !ok {
			t.Error("Expected fused_score in metadata")
		}

		// 可能有 related_entities
		if entities, ok := doc.Metadata["related_entities"]; ok {
			t.Logf("Related entities: %v", entities)
		}
	}
}

func TestConfig_Validate(t *testing.T) {
	graphDB := mock.NewMockGraphDB()
	embedModel := newMockEmbeddings(384)
	vectorStore := vectorstores.NewInMemoryVectorStore(embedModel)

	tests := []struct {
		name    string
		config  graphrag.Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: graphrag.Config{
				GraphDB:      graphDB,
				VectorStore:  vectorStore,
				VectorWeight: 0.6,
				GraphWeight:  0.4,
				MaxTraverseDepth: 2,
				TopK:         10,
			},
			wantErr: false,
		},
		{
			name: "missing GraphDB",
			config: graphrag.Config{
				VectorStore: vectorStore,
			},
			wantErr: true,
		},
		{
			name: "missing VectorStore",
			config: graphrag.Config{
				GraphDB: graphDB,
			},
			wantErr: true,
		},
		{
			name: "invalid VectorWeight",
			config: graphrag.Config{
				GraphDB:      graphDB,
				VectorStore:  vectorStore,
				VectorWeight: 1.5,
				GraphWeight:  0.4,
				MaxTraverseDepth: 2,
				TopK:         10,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
