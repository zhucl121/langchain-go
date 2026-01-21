package integration_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/zhucl121/langchain-go/pkg/types"
	"github.com/zhucl121/langchain-go/retrieval/graphdb"
	"github.com/zhucl121/langchain-go/retrieval/graphdb/builder"
	"github.com/zhucl121/langchain-go/retrieval/graphdb/mock"
	"github.com/zhucl121/langchain-go/retrieval/graphdb/neo4j"
	"github.com/zhucl121/langchain-go/retrieval/retrievers/graphrag"
	"github.com/zhucl121/langchain-go/retrieval/vectorstores"
)

// TestGraphRAG_E2E_Mock 端到端测试（Mock 模式）
func TestGraphRAG_E2E_Mock(t *testing.T) {
	ctx := context.Background()

	// 1. 创建 Mock GraphDB
	graphDB := mock.NewMockGraphDB()
	if err := graphDB.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect to GraphDB: %v", err)
	}
	defer graphDB.Close()

	// 2. 创建 Mock Embeddings
	mockEmbed := newMockEmbeddings(384)
	vectorStore := vectorstores.NewInMemoryVectorStore(mockEmbed)

	// 3. 创建 KG Builder
	mockEntityExtractor := &mockEntityExtractor{
		entities: []builder.Entity{
			{ID: "person-1", Name: "John Smith", Type: "Person"},
			{ID: "org-1", Name: "TechCorp", Type: "Organization"},
		},
	}

	mockRelExtractor := &mockRelationExtractor{
		relations: []builder.Relation{
			{ID: "rel-1", Source: "person-1", Target: "org-1", Type: "WORKS_FOR", Directed: true},
		},
	}

	mockBuilderEmbed := builder.NewMockEmbedder(384)

	kgBuilder, err := builder.NewKGBuilder(builder.KGBuilderConfig{
		GraphDB:           graphDB,
		EntityExtractor:   mockEntityExtractor,
		RelationExtractor: mockRelExtractor,
		Embedder:          mockBuilderEmbed,
		EnableEmbedding:   true,
	})
	if err != nil {
		t.Fatalf("Failed to create KG builder: %v", err)
	}

	// 4. 构建知识图谱
	text := "John Smith is the CEO of TechCorp, a leading technology company."
	_, err = kgBuilder.BuildAndStore(ctx, text)
	if err != nil {
		t.Fatalf("Failed to build and store graph: %v", err)
	}

	// 5. 向量化文档
	docs := []*types.Document{
		types.NewDocument(text, map[string]any{"source": "test"}),
	}
	_, err = vectorStore.AddDocuments(ctx, docs)
	if err != nil {
		t.Fatalf("Failed to add documents: %v", err)
	}

	// 6. 创建 GraphRAG 检索器
	config := graphrag.DefaultConfig(graphDB, vectorStore)
	config.EntityExtractor = mockEntityExtractor
	config.VectorWeight = 0.6
	config.GraphWeight = 0.4

	retriever, err := graphrag.NewGraphRAGRetriever(config)
	if err != nil {
		t.Fatalf("Failed to create GraphRAG retriever: %v", err)
	}

	// 7. 执行检索
	results, err := retriever.Search(ctx, "Who is the CEO?")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	// 8. 验证结果
	if len(results) == 0 {
		t.Error("Expected some results, got 0")
	}

	// 9. 验证统计
	stats := retriever.GetStatistics()
	t.Logf("Statistics: Vector=%d, Graph=%d, Fused=%d, Time=%dms",
		stats.VectorResultsCount, stats.GraphResultsCount,
		stats.FusedResultsCount, stats.TotalTime)

	if stats.TotalTime < 0 {
		t.Error("Total time should be non-negative")
	}

	t.Logf("✅ End-to-end test passed! Found %d results", len(results))
}

// TestGraphRAG_E2E_Neo4j 端到端测试（Neo4j 模式）
// 需要运行 Neo4j: docker compose -f docker-compose.graphdb.yml up -d neo4j
func TestGraphRAG_E2E_Neo4j(t *testing.T) {
	if os.Getenv("RUN_NEO4J_TESTS") != "true" {
		t.Skip("Skipping Neo4j test. Set RUN_NEO4J_TESTS=true to run.")
	}

	ctx := context.Background()

	// 1. 连接 Neo4j
	neo4jConfig := neo4j.DefaultConfig()
	neo4jConfig.URI = getEnvOrDefault("NEO4J_URI", "bolt://localhost:7687")
	neo4jConfig.Username = getEnvOrDefault("NEO4J_USER", "neo4j")
	neo4jConfig.Password = getEnvOrDefault("NEO4J_PASSWORD", "testpassword")

	graphDB, err := neo4j.NewNeo4jDriver(neo4jConfig)
	if err != nil {
		t.Fatalf("Failed to create Neo4j driver: %v", err)
	}

	if err := graphDB.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect to Neo4j: %v", err)
	}
	defer graphDB.Close()

	// 2. 清理测试数据
	cleanupNeo4j(t, ctx, graphDB)

	// 3. 创建 Mock Embeddings
	mockEmbed := newMockEmbeddings(384)
	vectorStore := vectorstores.NewInMemoryVectorStore(mockEmbed)

	// 4. 创建 KG Builder
	mockEntityExtractor := &mockEntityExtractor{
		entities: []builder.Entity{
			{ID: "person-1", Name: "Alice Johnson", Type: "Person"},
			{ID: "org-1", Name: "InnovateCorp", Type: "Organization"},
			{ID: "location-1", Name: "New York", Type: "Location"},
		},
	}

	mockRelExtractor := &mockRelationExtractor{
		relations: []builder.Relation{
			{ID: "rel-1", Source: "person-1", Target: "org-1", Type: "WORKS_FOR", Directed: true},
			{ID: "rel-2", Source: "org-1", Target: "location-1", Type: "LOCATED_IN", Directed: true},
		},
	}

	mockBuilderEmbed := builder.NewMockEmbedder(384)

	kgBuilder, err := builder.NewKGBuilder(builder.KGBuilderConfig{
		GraphDB:           graphDB,
		EntityExtractor:   mockEntityExtractor,
		RelationExtractor: mockRelExtractor,
		Embedder:          mockBuilderEmbed,
		EnableEmbedding:   true,
	})
	if err != nil {
		t.Fatalf("Failed to create KG builder: %v", err)
	}

	// 5. 构建知识图谱
	text := "Alice Johnson works at InnovateCorp, which is located in New York."
	_, err = kgBuilder.BuildAndStore(ctx, text)
	if err != nil {
		t.Fatalf("Failed to build and store graph: %v", err)
	}

	// 6. 验证图数据库中的节点
	node, err := graphDB.GetNode(ctx, "person-1")
	if err != nil {
		t.Fatalf("Failed to get node: %v", err)
	}
	if node == nil {
		t.Fatal("Node should exist")
	}
	t.Logf("Node: %s (%s)", node.Label, node.Type)

	// 7. 向量化文档
	docs := []*types.Document{
		types.NewDocument(text, map[string]any{"source": "neo4j_test"}),
	}
	_, err = vectorStore.AddDocuments(ctx, docs)
	if err != nil {
		t.Fatalf("Failed to add documents: %v", err)
	}

	// 8. 创建 GraphRAG 检索器
	config := graphrag.DefaultConfig(graphDB, vectorStore)
	config.EntityExtractor = mockEntityExtractor

	retriever, err := graphrag.NewGraphRAGRetriever(config)
	if err != nil {
		t.Fatalf("Failed to create GraphRAG retriever: %v", err)
	}

	// 9. 执行检索
	results, err := retriever.Search(ctx, "Where does Alice work?")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(results) == 0 {
		t.Error("Expected some results, got 0")
	}

	// 10. 验证图遍历
	stats := retriever.GetStatistics()
	t.Logf("Statistics: Vector=%d, Graph=%d, Fused=%d, Time=%dms",
		stats.VectorResultsCount, stats.GraphResultsCount,
		stats.FusedResultsCount, stats.TotalTime)

	t.Logf("✅ Neo4j end-to-end test passed! Found %d results", len(results))

	// 清理
	cleanupNeo4j(t, ctx, graphDB)
}

// TestGraphRAG_MultipleFusionStrategies 测试多种融合策略
func TestGraphRAG_MultipleFusionStrategies(t *testing.T) {
	ctx := context.Background()

	// 设置环境
	graphDB, _, retriever := setupTestEnvironment(t, ctx)
	defer graphDB.Close()

	query := "test query"

	strategies := []struct {
		name     string
		strategy graphrag.FusionStrategy
	}{
		{"Weighted", graphrag.FusionStrategyWeighted},
		{"RRF", graphrag.FusionStrategyRRF},
		{"Max", graphrag.FusionStrategyMax},
		{"Min", graphrag.FusionStrategyMin},
	}

	for _, s := range strategies {
		t.Run(s.name, func(t *testing.T) {
			opts := graphrag.SearchOptions{
				Mode:           graphrag.SearchModeHybrid,
				K:              5,
				FusionStrategy: s.strategy,
			}

			results, err := retriever.Search(ctx, query, opts)
			if err != nil {
				t.Fatalf("Search with %s failed: %v", s.name, err)
			}

			t.Logf("%s: Found %d results", s.name, len(results))
		})
	}
}

// TestGraphRAG_MultipleRerankStrategies 测试多种重排序策略
func TestGraphRAG_MultipleRerankStrategies(t *testing.T) {
	ctx := context.Background()

	graphDB, _, retriever := setupTestEnvironment(t, ctx)
	defer graphDB.Close()

	query := "test query"

	strategies := []struct {
		name     string
		strategy graphrag.RerankStrategy
	}{
		{"Score", graphrag.RerankStrategyScore},
		{"Diversity", graphrag.RerankStrategyDiversity},
		{"MMR", graphrag.RerankStrategyMMR},
	}

	for _, s := range strategies {
		t.Run(s.name, func(t *testing.T) {
			opts := graphrag.SearchOptions{
				Mode:           graphrag.SearchModeHybrid,
				K:              5,
				RerankStrategy: s.strategy,
			}

			results, err := retriever.Search(ctx, query, opts)
			if err != nil {
				t.Fatalf("Search with %s failed: %v", s.name, err)
			}

			t.Logf("%s: Found %d results", s.name, len(results))
		})
	}
}

// TestGraphRAG_BatchBuilding 测试批量构建
func TestGraphRAG_BatchBuilding(t *testing.T) {
	ctx := context.Background()

	graphDB := mock.NewMockGraphDB()
	if err := graphDB.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer graphDB.Close()

	mockEntityExtractor := &mockEntityExtractor{
		entities: []builder.Entity{
			{ID: "person-1", Name: "Bob", Type: "Person"},
			{ID: "person-2", Name: "Carol", Type: "Person"},
		},
	}

	mockRelExtractor := &mockRelationExtractor{
		relations: []builder.Relation{
			{ID: "rel-1", Source: "person-1", Target: "person-2", Type: "KNOWS", Directed: false},
		},
	}

	mockBuilderEmbed := builder.NewMockEmbedder(384)

	kgBuilder, err := builder.NewKGBuilder(builder.KGBuilderConfig{
		GraphDB:           graphDB,
		EntityExtractor:   mockEntityExtractor,
		RelationExtractor: mockRelExtractor,
		Embedder:          mockBuilderEmbed,
		EnableEmbedding:   true,
	})
	if err != nil {
		t.Fatalf("Failed to create builder: %v", err)
	}

	// 批量构建
	texts := []string{
		"Bob is a software engineer.",
		"Carol is a data scientist.",
		"Bob and Carol are colleagues.",
	}

	start := time.Now()
	graphs, err := kgBuilder.BuildBatch(ctx, texts)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Batch build failed: %v", err)
	}

	if len(graphs) != len(texts) {
		t.Errorf("Expected %d graphs, got %d", len(texts), len(graphs))
	}

	t.Logf("✅ Batch built %d graphs in %v", len(graphs), elapsed)

	// 合并图
	mergedGraph, err := kgBuilder.Merge(ctx, graphs)
	if err != nil {
		t.Fatalf("Merge failed: %v", err)
	}

	t.Logf("Merged graph: %d entities, %d relations",
		len(mergedGraph.Entities), len(mergedGraph.Relations))
}

// Helper functions

func setupTestEnvironment(t *testing.T, ctx context.Context) (graphdb.GraphDB, vectorstores.VectorStore, *graphrag.GraphRAGRetriever) {
	graphDB := mock.NewMockGraphDB()
	if err := graphDB.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	mockEmbed := newMockEmbeddings(384)
	vectorStore := vectorstores.NewInMemoryVectorStore(mockEmbed)

	// 添加测试文档
	docs := []*types.Document{
		types.NewDocument("Test document 1", map[string]any{"id": "1"}),
		types.NewDocument("Test document 2", map[string]any{"id": "2"}),
	}
	vectorStore.AddDocuments(ctx, docs)

	// 添加测试节点
	graphDB.AddNode(ctx, &graphdb.Node{ID: "node-1", Type: "Test", Label: "Test Node"})

	mockEntityExtractor := &mockEntityExtractor{
		entities: []builder.Entity{
			{ID: "node-1", Name: "Test Node", Type: "Test"},
		},
	}

	config := graphrag.DefaultConfig(graphDB, vectorStore)
	config.EntityExtractor = mockEntityExtractor

	retriever, err := graphrag.NewGraphRAGRetriever(config)
	if err != nil {
		t.Fatalf("Failed to create retriever: %v", err)
	}

	return graphDB, vectorStore, retriever
}

func cleanupNeo4j(t *testing.T, ctx context.Context, graphDB graphdb.GraphDB) {
	// 删除测试节点
	testIDs := []string{"person-1", "org-1", "location-1"}
	for _, id := range testIDs {
		graphDB.DeleteNode(ctx, id)
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
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
