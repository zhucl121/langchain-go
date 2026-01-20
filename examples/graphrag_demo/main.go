package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/zhucl121/langchain-go/core/chat/providers/openai"
	"github.com/zhucl121/langchain-go/pkg/types"
	"github.com/zhucl121/langchain-go/retrieval/embeddings"
	"github.com/zhucl121/langchain-go/retrieval/graphdb"
	"github.com/zhucl121/langchain-go/retrieval/graphdb/builder"
	"github.com/zhucl121/langchain-go/retrieval/graphdb/mock"
	"github.com/zhucl121/langchain-go/retrieval/graphdb/neo4j"
	"github.com/zhucl121/langchain-go/retrieval/retrievers/graphrag"
	"github.com/zhucl121/langchain-go/retrieval/vectorstores"
)

const (
	ModeMock   = "mock"
	ModeOpenAI = "openai"
	ModeNeo4j  = "neo4j"
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

// mockRelationExtractor for testing
type mockRelationExtractor struct {
	relations []builder.Relation
}

func (m *mockRelationExtractor) Extract(ctx context.Context, text string, entities []builder.Entity) ([]builder.Relation, error) {
	return m.relations, nil
}

func (m *mockRelationExtractor) ExtractWithSchema(ctx context.Context, text string, entities []builder.Entity, schema *builder.RelationSchema) ([]builder.Relation, error) {
	return m.relations, nil
}

// mockEmbeddings implements embeddings.Embeddings interface
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

func main() {
	// 获取运行模式
	mode := os.Getenv("DEMO_MODE")
	if mode == "" {
		mode = ModeMock
	}

	fmt.Printf("🚀 GraphRAG Demo - Mode: %s\n", mode)
	fmt.Println(strings.Repeat("=", 60))

	ctx := context.Background()

	// 根据模式选择组件
	var graphDB graphdb.GraphDB
	var vectorStore vectorstores.VectorStore
	var kgBuilder *builder.StandardKGBuilder
	var retriever *graphrag.GraphRAGRetriever

	switch mode {
	case ModeMock:
		graphDB, vectorStore, kgBuilder, retriever = setupMockMode(ctx)
	case ModeOpenAI:
		graphDB, vectorStore, kgBuilder, retriever = setupOpenAIMode(ctx)
	case ModeNeo4j:
		graphDB, vectorStore, kgBuilder, retriever = setupNeo4jMode(ctx)
	default:
		log.Fatalf("Unknown mode: %s", mode)
	}

	defer graphDB.Close()

	// 演示流程
	fmt.Println("\n📚 Step 1: 准备示例文档")
	docs := prepareDocuments()
	displayDocuments(docs)

	fmt.Println("\n🔨 Step 2: 构建知识图谱")
	buildKnowledgeGraph(ctx, kgBuilder, docs, graphDB)

	fmt.Println("\n📄 Step 3: 向量化文档")
	vectorizeDocuments(ctx, vectorStore, docs)

	fmt.Println("\n🔍 Step 4: GraphRAG 检索演示")
	demoGraphRAGRetrieval(ctx, retriever)

	fmt.Println("\n🎯 Step 5: 融合策略对比")
	demoFusionStrategies(ctx, retriever)

	fmt.Println("\n🔄 Step 6: 重排序策略对比")
	demoRerankStrategies(ctx, retriever)

	fmt.Println("\n✨ Step 7: 上下文增强展示")
	demoContextAugmentation(ctx, retriever)

	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("✅ GraphRAG Demo 完成！")
}

// setupMockMode 设置 Mock 模式
func setupMockMode(ctx context.Context) (graphdb.GraphDB, vectorstores.VectorStore, *builder.StandardKGBuilder, *graphrag.GraphRAGRetriever) {
	fmt.Println("📦 使用 Mock 组件（无需外部服务）")

	// Mock GraphDB
	graphDB := mock.NewMockGraphDB()
	if err := graphDB.Connect(ctx); err != nil {
		log.Fatalf("Failed to connect to mock GraphDB: %v", err)
	}

	// Mock Embeddings
	mockEmbed := newMockEmbeddings(384)
	vectorStore := vectorstores.NewInMemoryVectorStore(mockEmbed)

	// Mock Builder Embedder
	mockBuilderEmbed := builder.NewMockEmbedder(384)

	// Mock Entity Extractor
	mockExtractor := &mockEntityExtractor{
		entities: []builder.Entity{
			{ID: "person-1", Name: "John Smith", Type: "Person"},
			{ID: "person-2", Name: "Alice Johnson", Type: "Person"},
			{ID: "org-1", Name: "TechCorp", Type: "Organization"},
			{ID: "product-1", Name: "CloudMax", Type: "Product"},
			{ID: "location-1", Name: "San Francisco", Type: "Location"},
		},
	}

	// Mock Relation Extractor
	mockRelExtractor := &mockRelationExtractor{
		relations: []builder.Relation{
			{ID: "rel-1", Source: "person-1", Target: "org-1", Type: "WORKS_FOR", Directed: true},
			{ID: "rel-2", Source: "person-2", Target: "org-1", Type: "WORKS_FOR", Directed: true},
			{ID: "rel-3", Source: "org-1", Target: "location-1", Type: "LOCATED_IN", Directed: true},
			{ID: "rel-4", Source: "org-1", Target: "product-1", Type: "LAUNCHED", Directed: true},
		},
	}

	kgBuilder, err := builder.NewKGBuilder(builder.KGBuilderConfig{
		GraphDB:           graphDB,
		EntityExtractor:   mockExtractor,
		RelationExtractor: mockRelExtractor,
		Embedder:          mockBuilderEmbed,
		EnableEmbedding:   true,
	})
	if err != nil {
		log.Fatalf("Failed to create KG builder: %v", err)
	}

	// GraphRAG Retriever
	config := graphrag.DefaultConfig(graphDB, vectorStore)
	config.EntityExtractor = mockExtractor
	config.VectorWeight = 0.6
	config.GraphWeight = 0.4

	retriever, err := graphrag.NewGraphRAGRetriever(config)
	if err != nil {
		log.Fatalf("Failed to create GraphRAG retriever: %v", err)
	}

	return graphDB, vectorStore, kgBuilder, retriever
}

// setupOpenAIMode 设置 OpenAI 模式
func setupOpenAIMode(ctx context.Context) (graphdb.GraphDB, vectorstores.VectorStore, *builder.StandardKGBuilder, *graphrag.GraphRAGRetriever) {
	fmt.Println("🤖 使用 OpenAI 组件（需要 API Key）")

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY environment variable is required")
	}

	// Mock GraphDB (OpenAI 模式下仍使用 Mock)
	graphDB := mock.NewMockGraphDB()
	if err := graphDB.Connect(ctx); err != nil {
		log.Fatalf("Failed to connect to mock GraphDB: %v", err)
	}

	// OpenAI Chat Model
	chatModel, err := openai.New(openai.Config{
		APIKey: apiKey,
		Model:  "gpt-4o-mini",
	})
	if err != nil {
		log.Fatalf("Failed to create OpenAI chat model: %v", err)
	}

	// OpenAI Embeddings
	embedConfig := embeddings.OpenAIEmbeddingsConfig{
		APIKey: apiKey,
		Model:  "text-embedding-3-small",
	}
	openaiEmbed := embeddings.NewOpenAIEmbeddings(embedConfig)

	vectorStore := vectorstores.NewInMemoryVectorStore(openaiEmbed)

	// LLM-based KG Builder
	entityExtractor := builder.NewLLMEntityExtractor(chatModel, nil)
	relationExtractor := builder.NewLLMRelationExtractor(chatModel, nil)
	embedder := builder.NewEmbeddingModelAdapter(openaiEmbed)

	kgBuilder, err := builder.NewKGBuilder(builder.KGBuilderConfig{
		GraphDB:           graphDB,
		EntityExtractor:   entityExtractor,
		RelationExtractor: relationExtractor,
		Embedder:          embedder,
		EnableEmbedding:   true,
	})
	if err != nil {
		log.Fatalf("Failed to create KG builder: %v", err)
	}

	// GraphRAG Retriever
	config := graphrag.DefaultConfig(graphDB, vectorStore)
	config.EntityExtractor = entityExtractor
	config.ChatModel = chatModel
	config.Embeddings = openaiEmbed

	retriever, err := graphrag.NewGraphRAGRetriever(config)
	if err != nil {
		log.Fatalf("Failed to create GraphRAG retriever: %v", err)
	}

	return graphDB, vectorStore, kgBuilder, retriever
}

// setupNeo4jMode 设置 Neo4j 模式
func setupNeo4jMode(ctx context.Context) (graphdb.GraphDB, vectorstores.VectorStore, *builder.StandardKGBuilder, *graphrag.GraphRAGRetriever) {
	fmt.Println("🗄️  使用 Neo4j + OpenAI 组件")

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY environment variable is required")
	}

	neo4jURI := os.Getenv("NEO4J_URI")
	if neo4jURI == "" {
		neo4jURI = "bolt://localhost:7687"
	}

	neo4jUser := os.Getenv("NEO4J_USER")
	if neo4jUser == "" {
		neo4jUser = "neo4j"
	}

	neo4jPassword := os.Getenv("NEO4J_PASSWORD")
	if neo4jPassword == "" {
		neo4jPassword = "testpassword"
	}

	// Neo4j GraphDB
	neo4jConfig := neo4j.DefaultConfig()
	neo4jConfig.URI = neo4jURI
	neo4jConfig.Username = neo4jUser
	neo4jConfig.Password = neo4jPassword
	graphDB, err := neo4j.NewNeo4jDriver(neo4jConfig)
	if err != nil {
		log.Fatalf("Failed to create Neo4j driver: %v", err)
	}

	if err := graphDB.Connect(ctx); err != nil {
		log.Fatalf("Failed to connect to Neo4j: %v", err)
	}

	// OpenAI Chat Model
	chatModel, err := openai.New(openai.Config{
		APIKey: apiKey,
		Model:  "gpt-4o-mini",
	})
	if err != nil {
		log.Fatalf("Failed to create OpenAI chat model: %v", err)
	}

	// OpenAI Embeddings
	embedConfig := embeddings.OpenAIEmbeddingsConfig{
		APIKey: apiKey,
		Model:  "text-embedding-3-small",
	}
	openaiEmbed := embeddings.NewOpenAIEmbeddings(embedConfig)

	vectorStore := vectorstores.NewInMemoryVectorStore(openaiEmbed)

	// LLM-based KG Builder
	entityExtractor := builder.NewLLMEntityExtractor(chatModel, nil)
	relationExtractor := builder.NewLLMRelationExtractor(chatModel, nil)
	embedder := builder.NewEmbeddingModelAdapter(openaiEmbed)

	kgBuilder, err := builder.NewKGBuilder(builder.KGBuilderConfig{
		GraphDB:           graphDB,
		EntityExtractor:   entityExtractor,
		RelationExtractor: relationExtractor,
		Embedder:          embedder,
		EnableEmbedding:   true,
	})
	if err != nil {
		log.Fatalf("Failed to create KG builder: %v", err)
	}

	// GraphRAG Retriever
	config := graphrag.DefaultConfig(graphDB, vectorStore)
	config.EntityExtractor = entityExtractor
	config.ChatModel = chatModel
	config.Embeddings = openaiEmbed

	retriever, err := graphrag.NewGraphRAGRetriever(config)
	if err != nil {
		log.Fatalf("Failed to create GraphRAG retriever: %v", err)
	}

	return graphDB, vectorStore, kgBuilder, retriever
}

// prepareDocuments 准备示例文档
func prepareDocuments() []*types.Document {
	return []*types.Document{
		types.NewDocument(
			"John Smith is the CEO of TechCorp, a leading technology company founded in 2010. He has over 20 years of experience in the tech industry.",
			map[string]any{"source": "company_profile", "category": "leadership"},
		),
		types.NewDocument(
			"TechCorp is headquartered in San Francisco, California. The company specializes in cloud computing and artificial intelligence solutions.",
			map[string]any{"source": "company_profile", "category": "company"},
		),
		types.NewDocument(
			"Alice Johnson works at TechCorp as the Chief Technology Officer. She leads the engineering team of over 500 engineers.",
			map[string]any{"source": "company_profile", "category": "leadership"},
		),
		types.NewDocument(
			"TechCorp recently launched CloudMax, a new cloud infrastructure platform that competes with AWS and Azure. The platform has gained 10,000 customers in its first year.",
			map[string]any{"source": "product_news", "category": "product"},
		),
		types.NewDocument(
			"San Francisco is known as a major hub for technology companies. Many startups and established tech giants have offices in the city.",
			map[string]any{"source": "general_info", "category": "location"},
		),
	}
}

// displayDocuments 显示文档
func displayDocuments(docs []*types.Document) {
	for i, doc := range docs {
		content := doc.Content
		if len(content) > 80 {
			content = content[:80] + "..."
		}
		fmt.Printf("  %d. %s\n", i+1, content)
	}
	fmt.Printf("  总计: %d 个文档\n", len(docs))
}

// buildKnowledgeGraph 构建知识图谱
func buildKnowledgeGraph(ctx context.Context, kgBuilder *builder.StandardKGBuilder, docs []*types.Document, graphDB graphdb.GraphDB) {
	texts := make([]string, len(docs))
	for i, doc := range docs {
		texts[i] = doc.Content
	}

	// 批量构建
	graphs, err := kgBuilder.BuildBatch(ctx, texts)
	if err != nil {
		log.Printf("Warning: Failed to build knowledge graph: %v", err)
		return
	}

	// 合并图
	mergedGraph, err := kgBuilder.Merge(ctx, graphs)
	if err != nil {
		log.Printf("Warning: Failed to merge graphs: %v", err)
		return
	}

	fmt.Printf("  提取实体: %d 个\n", len(mergedGraph.Entities))
	fmt.Printf("  提取关系: %d 个\n", len(mergedGraph.Relations))

	// 显示部分实体
	if len(mergedGraph.Entities) > 0 {
		fmt.Println("  示例实体:")
		for i, entity := range mergedGraph.Entities {
			if i >= 5 {
				break
			}
			fmt.Printf("    - %s (%s)\n", entity.Name, entity.Type)
		}
	}

	// 显示部分关系
	if len(mergedGraph.Relations) > 0 {
		fmt.Println("  示例关系:")
		for i, rel := range mergedGraph.Relations {
			if i >= 5 {
				break
			}
			// 注意：Relation 只有 Source 和 Target ID，没有名称
			fmt.Printf("    - %s -[%s]-> %s\n", rel.Source, rel.Type, rel.Target)
		}
	}

	// 存储到图数据库
	fmt.Println("  存储到图数据库...")
	if _, err := kgBuilder.BuildAndStore(ctx, strings.Join(texts, "\n")); err != nil {
		log.Printf("Warning: Failed to store graph: %v", err)
	}
}

// vectorizeDocuments 向量化文档
func vectorizeDocuments(ctx context.Context, vectorStore vectorstores.VectorStore, docs []*types.Document) {
	ids, err := vectorStore.AddDocuments(ctx, docs)
	if err != nil {
		log.Fatalf("Failed to add documents to vector store: %v", err)
	}

	fmt.Printf("  成功向量化 %d 个文档\n", len(ids))
}

// demoGraphRAGRetrieval 演示 GraphRAG 检索
func demoGraphRAGRetrieval(ctx context.Context, retriever *graphrag.GraphRAGRetriever) {
	queries := []string{
		"Who is the CEO of TechCorp?",
		"What products does TechCorp offer?",
		"Where is TechCorp located?",
	}

	for i, query := range queries {
		fmt.Printf("\n  查询 %d: %s\n", i+1, query)

		docs, err := retriever.Search(ctx, query)
		if err != nil {
			log.Printf("  ❌ 检索失败: %v\n", err)
			continue
		}

		fmt.Printf("  ✅ 找到 %d 个结果\n", len(docs))

		// 显示前3个结果
		for j, doc := range docs {
			if j >= 3 {
				break
			}

			content := doc.Content
			if len(content) > 100 {
				content = content[:100] + "..."
			}

			score := 0.0
			if s, ok := doc.Metadata["fused_score"].(float64); ok {
				score = s
			}

			fmt.Printf("    %d. [%.3f] %s\n", j+1, score, content)

			// 显示相关实体
			if entities, ok := doc.Metadata["related_entities"].([]string); ok && len(entities) > 0 {
				fmt.Printf("       相关实体: %s\n", strings.Join(entities, ", "))
			}
		}

		// 显示统计
		stats := retriever.GetStatistics()
		fmt.Printf("  📊 统计: 向量=%d, 图=%d, 融合=%d, 耗时=%dms\n",
			stats.VectorResultsCount, stats.GraphResultsCount,
			stats.FusedResultsCount, stats.TotalTime)
	}
}

// demoFusionStrategies 演示融合策略
func demoFusionStrategies(ctx context.Context, retriever *graphrag.GraphRAGRetriever) {
	query := "Tell me about TechCorp"

	strategies := []struct {
		name     string
		strategy graphrag.FusionStrategy
	}{
		{"加权融合 (Weighted)", graphrag.FusionStrategyWeighted},
		{"RRF 融合", graphrag.FusionStrategyRRF},
		{"最大值融合 (Max)", graphrag.FusionStrategyMax},
		{"最小值融合 (Min)", graphrag.FusionStrategyMin},
	}

	for _, s := range strategies {
		fmt.Printf("\n  策略: %s\n", s.name)

		opts := graphrag.SearchOptions{
			Mode:           graphrag.SearchModeHybrid,
			K:              5,
			FusionStrategy: s.strategy,
		}

		docs, err := retriever.Search(ctx, query, opts)
		if err != nil {
			log.Printf("  ❌ 检索失败: %v\n", err)
			continue
		}

		fmt.Printf("  找到 %d 个结果\n", len(docs))

		// 显示前3个结果的分数
		for i, doc := range docs {
			if i >= 3 {
				break
			}

			score := 0.0
			if s, ok := doc.Metadata["fused_score"].(float64); ok {
				score = s
			}

			content := doc.Content
			if len(content) > 60 {
				content = content[:60] + "..."
			}

			fmt.Printf("    %d. [%.3f] %s\n", i+1, score, content)
		}
	}
}

// demoRerankStrategies 演示重排序策略
func demoRerankStrategies(ctx context.Context, retriever *graphrag.GraphRAGRetriever) {
	query := "TechCorp company information"

	strategies := []struct {
		name     string
		strategy graphrag.RerankStrategy
	}{
		{"分数排序 (Score)", graphrag.RerankStrategyScore},
		{"多样性排序 (Diversity)", graphrag.RerankStrategyDiversity},
		{"MMR 排序", graphrag.RerankStrategyMMR},
	}

	for _, s := range strategies {
		fmt.Printf("\n  策略: %s\n", s.name)

		opts := graphrag.SearchOptions{
			Mode:           graphrag.SearchModeHybrid,
			K:              5,
			RerankStrategy: s.strategy,
		}

		docs, err := retriever.Search(ctx, query, opts)
		if err != nil {
			log.Printf("  ❌ 检索失败: %v\n", err)
			continue
		}

		fmt.Printf("  找到 %d 个结果\n", len(docs))

		// 显示结果的多样性
		categories := make(map[string]int)
		for _, doc := range docs {
			if cat, ok := doc.Metadata["category"].(string); ok {
				categories[cat]++
			}
		}

		fmt.Printf("  类别分布: ")
		for cat, count := range categories {
			fmt.Printf("%s=%d ", cat, count)
		}
		fmt.Println()
	}
}

// demoContextAugmentation 演示上下文增强
func demoContextAugmentation(ctx context.Context, retriever *graphrag.GraphRAGRetriever) {
	query := "Who leads TechCorp?"

	// 不启用上下文增强
	fmt.Println("\n  不启用上下文增强:")
	opts1 := graphrag.SearchOptions{
		Mode:                      graphrag.SearchModeHybrid,
		K:                         3,
		EnableContextAugmentation: false,
	}

	docs1, err := retriever.Search(ctx, query, opts1)
	if err != nil {
		log.Printf("  ❌ 检索失败: %v\n", err)
		return
	}

	if len(docs1) > 0 {
		doc := docs1[0]
		fmt.Printf("  内容长度: %d 字符\n", len(doc.Content))
		fmt.Printf("  元数据键数: %d\n", len(doc.Metadata))
	}

	// 启用上下文增强
	fmt.Println("\n  启用上下文增强:")
	opts2 := graphrag.SearchOptions{
		Mode:                      graphrag.SearchModeHybrid,
		K:                         3,
		EnableContextAugmentation: true,
	}

	docs2, err := retriever.Search(ctx, query, opts2)
	if err != nil {
		log.Printf("  ❌ 检索失败: %v\n", err)
		return
	}

	if len(docs2) > 0 {
		doc := docs2[0]
		fmt.Printf("  内容长度: %d 字符\n", len(doc.Content))
		fmt.Printf("  元数据键数: %d\n", len(doc.Metadata))

		// 显示增强的元数据
		fmt.Println("  增强的元数据:")
		for key := range doc.Metadata {
			if strings.HasPrefix(key, "related_") || strings.HasPrefix(key, "neighbor_") || strings.HasPrefix(key, "graph_") {
				fmt.Printf("    - %s: %v\n", key, doc.Metadata[key])
			}
		}

		// 显示相关实体
		if entities, ok := doc.Metadata["related_entities"].([]string); ok && len(entities) > 0 {
			fmt.Printf("  相关实体: %s\n", strings.Join(entities, ", "))
		}
	}
}
