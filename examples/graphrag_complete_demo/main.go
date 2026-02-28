package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/zhucl121/langchain-go/pkg/types"
	"github.com/zhucl121/langchain-go/retrieval/embeddings"
	"github.com/zhucl121/langchain-go/retrieval/graphdb"
	"github.com/zhucl121/langchain-go/retrieval/graphdb/builder"
	"github.com/zhucl121/langchain-go/retrieval/graphdb/mock"
	"github.com/zhucl121/langchain-go/retrieval/graphdb/nebula"
	"github.com/zhucl121/langchain-go/retrieval/graphdb/neo4j"
	"github.com/zhucl121/langchain-go/retrieval/loaders"
	"github.com/zhucl121/langchain-go/retrieval/retrievers/graphrag"
	"github.com/zhucl121/langchain-go/retrieval/vectorstores"
)

// GraphRAG 完整示例 - 演示所有功能
//
// 本示例展示：
// 1. 多种图数据库支持（Mock, Neo4j, NebulaGraph）
// 2. 知识图谱构建
// 3. GraphRAG 混合检索
// 4. 不同融合和重排序策略
// 5. 性能对比

func main() {
	fmt.Println("========================================")
	fmt.Println("  GraphRAG 完整功能演示")
	fmt.Println("========================================")

	// 检查运行模式
	mode := os.Getenv("GRAPH_MODE")
	if mode == "" {
		mode = "mock"
	}

	fmt.Printf("运行模式: %s\n\n", mode)

	ctx := context.Background()

	// 准备测试数据
	documents := prepareDocuments()

	// 根据模式选择图数据库
	var graphDB graphdb.GraphDB
	var err error

	switch mode {
	case "neo4j":
		graphDB, err = setupNeo4j()
	case "nebula":
		graphDB, err = setupNebula()
	default:
		graphDB, err = setupMock()
	}

	if err != nil {
		log.Fatalf("Failed to setup graph database: %v", err)
	}

	// 创建 embeddings
	embeddingsModel := embeddings.NewFakeEmbeddings(384)

	// 创建向量存储
	vectorStore := vectorstores.NewInMemoryVectorStore(embeddingsModel)

	// 构建知识图谱
	fmt.Println("📊 Step 1: 构建知识图谱...")
	if err := buildKnowledgeGraph(ctx, graphDB, embeddingsModel, documents); err != nil {
		log.Fatalf("Failed to build knowledge graph: %v", err)
	}

	// 向量化文档
	fmt.Println("\n📊 Step 2: 向量化文档...")
	if err := vectorizeDocuments(ctx, vectorStore, embeddingsModel, documents); err != nil {
		log.Fatalf("Failed to vectorize documents: %v", err)
	}

	// 创建 GraphRAG 检索器
	fmt.Println("\n📊 Step 3: 创建 GraphRAG 检索器...")
	retriever, err := createGraphRAGRetriever(graphDB, vectorStore, embeddingsModel)
	if err != nil {
		log.Fatalf("Failed to create retriever: %v", err)
	}

	// 测试查询
	queries := []string{
		"人工智能的发展历史",
		"机器学习的应用",
		"深度学习技术",
	}

	// 演示不同的检索模式
	fmt.Println("\n========================================")
	fmt.Println("  检索模式对比")
	fmt.Println("========================================")

	for _, query := range queries {
		fmt.Printf("\n🔍 查询: %s\n", query)
		fmt.Println(strings.Repeat("-", 60))

		// 1. 混合模式
		demoSearchMode(ctx, retriever, query, graphrag.SearchModeHybrid, "混合检索")

		// 2. 纯向量模式
		demoSearchMode(ctx, retriever, query, graphrag.SearchModeVector, "纯向量检索")

		// 3. 纯图模式
		demoSearchMode(ctx, retriever, query, graphrag.SearchModeGraph, "纯图检索")
	}

	// 演示融合策略
	fmt.Println("\n========================================")
	fmt.Println("  融合策略对比")
	fmt.Println("========================================")

	query := queries[0]
	strategies := []graphrag.FusionStrategy{
		graphrag.FusionStrategyWeighted,
		graphrag.FusionStrategyRRF,
		graphrag.FusionStrategyMax,
		graphrag.FusionStrategyMin,
	}

	for _, strategy := range strategies {
		demoFusionStrategy(ctx, retriever, query, strategy)
	}

	// 演示重排序策略
	fmt.Println("\n========================================")
	fmt.Println("  重排序策略对比")
	fmt.Println("========================================")

	rerankStrategies := []graphrag.RerankStrategy{
		graphrag.RerankStrategyScore,
		graphrag.RerankStrategyDiversity,
		graphrag.RerankStrategyMMR,
	}

	for _, strategy := range rerankStrategies {
		demoRerankStrategy(ctx, retriever, query, strategy)
	}

	// 显示统计信息
	fmt.Println("\n========================================")
	fmt.Println("  统计信息")
	fmt.Println("========================================")

	stats := retriever.GetStatistics()
	displayStatistics(stats)

	fmt.Println("\n✅ 演示完成！")
}

func setupMock() (graphdb.GraphDB, error) {
	return mock.NewMockGraphDB(), nil
}

func setupNeo4j() (graphdb.GraphDB, error) {
	config := neo4j.DefaultConfig()
	config.URI = os.Getenv("NEO4J_URI")
	if config.URI == "" {
		config.URI = "bolt://localhost:7687"
	}

	driver, err := neo4j.NewNeo4jDriver(config)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	if err := driver.Connect(ctx); err != nil {
		return nil, err
	}

	return driver, nil
}

func setupNebula() (graphdb.GraphDB, error) {
	config := nebula.DefaultConfig().
		WithSpace("langchain_demo").
		WithTimeout(30 * time.Second)

	addresses := os.Getenv("NEBULA_ADDRESSES")
	if addresses != "" {
		config = config.WithAddresses([]string{addresses})
	}

	driver, err := nebula.NewNebulaDriver(config)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	if err := driver.Connect(ctx); err != nil {
		return nil, err
	}

	return driver, nil
}

func prepareDocuments() []types.Document {
	return []types.Document{
		{
			Content: "人工智能（AI）是计算机科学的一个分支，致力于创建能够执行通常需要人类智能的任务的系统。",
			Metadata: map[string]interface{}{
				"source": "ai_intro",
				"topic":  "artificial_intelligence",
			},
		},
		{
			Content: "机器学习是人工智能的一个子领域，使计算机系统能够从数据中学习和改进，而无需明确编程。",
			Metadata: map[string]interface{}{
				"source": "ml_basics",
				"topic":  "machine_learning",
			},
		},
		{
			Content: "深度学习是机器学习的一个分支，使用多层神经网络来处理复杂的模式识别任务。",
			Metadata: map[string]interface{}{
				"source": "dl_intro",
				"topic":  "deep_learning",
			},
		},
		{
			Content: "自然语言处理（NLP）是人工智能的一个重要应用领域，专注于使计算机能够理解和生成人类语言。",
			Metadata: map[string]interface{}{
				"source": "nlp_overview",
				"topic":  "natural_language_processing",
			},
		},
		{
			Content: "计算机视觉是人工智能的另一个关键领域，使机器能够解释和理解视觉信息。",
			Metadata: map[string]interface{}{
				"source": "cv_basics",
				"topic":  "computer_vision",
			},
		},
	}
}

func buildKnowledgeGraph(ctx context.Context, graphDB graphdb.GraphDB, emb embeddings.Embeddings, docs []types.Document) error {
	// 创建简单的实体提取器（mock）
	extractor := &mockEntityExtractor{}
	relationExtractor := &mockRelationExtractor{}

	kgBuilder, err := builder.NewKGBuilder(builder.KGBuilderConfig{
		GraphDB:           graphDB,
		EntityExtractor:   extractor,
		RelationExtractor: relationExtractor,
	})
	if err != nil {
		return fmt.Errorf("failed to create kg builder: %w", err)
	}

	// 转换为文本列表
	texts := make([]string, len(docs))
	for i, doc := range docs {
		texts[i] = doc.Content
	}
	
	// 批量构建
	graphs, err := kgBuilder.BuildBatch(ctx, texts)
	if err != nil {
		return fmt.Errorf("failed to build batch: %w", err)
	}

	fmt.Printf("  ✓ 构建了 %d 个子图\n", len(graphs))

	// 合并图
	merged, err := kgBuilder.Merge(ctx, graphs)
	if err != nil {
		return fmt.Errorf("failed to merge graphs: %w", err)
	}

	fmt.Printf("  ✓ 合并后: %d 实体, %d 关系\n", len(merged.Entities), len(merged.Relations))

	// 知识图谱已构建完成（BuildBatch 内部已存储）
	fmt.Println("  ✓ 知识图谱已构建和存储")

	return nil
}

func vectorizeDocuments(ctx context.Context, store vectorstores.VectorStore, emb embeddings.Embeddings, docs []types.Document) error {
	// 转换为 loaders.Document
	loaderDocs := make([]*loaders.Document, len(docs))
	for i, doc := range docs {
		loaderDocs[i] = &loaders.Document{
			Content:  doc.Content,
			Metadata: doc.Metadata,
		}
	}
	
	_, err := store.AddDocuments(ctx, loaderDocs)
	if err != nil {
		return err
	}

	fmt.Printf("  ✓ 向量化了 %d 个文档\n", len(docs))
	return nil
}

func createGraphRAGRetriever(graphDB graphdb.GraphDB, vectorStore vectorstores.VectorStore, emb embeddings.Embeddings) (*graphrag.GraphRAGRetriever, error) {
	extractor := &mockEntityExtractor{}

	config := graphrag.Config{
		GraphDB:          graphDB,
		VectorStore:      vectorStore,
		EntityExtractor:  extractor,
		Embeddings:       emb,
		VectorWeight:     0.6,
		GraphWeight:      0.4,
		MaxTraverseDepth: 2,
		TopK:             5,
	}

	retriever, err := graphrag.NewGraphRAGRetriever(config)
	if err != nil {
		return nil, err
	}

	fmt.Println("  ✓ GraphRAG 检索器已创建")
	return retriever, nil
}

func demoSearchMode(ctx context.Context, retriever *graphrag.GraphRAGRetriever, query string, mode graphrag.SearchMode, label string) {
	start := time.Now()

	results, err := retriever.Search(ctx, query, graphrag.SearchOptions{
		Mode: mode,
		K:    5,
	})

	duration := time.Since(start)

	if err != nil {
		fmt.Printf("  ❌ %s 失败: %v\n", label, err)
		return
	}

	fmt.Printf("\n  📌 %s (耗时: %v)\n", label, duration)
	fmt.Printf("     结果数: %d\n", len(results))

	for i, doc := range results {
		if i >= 3 {
			break
		}
		score := 0.0
		if scoreVal, ok := doc.Metadata["score"].(float64); ok {
			score = scoreVal
		}
		fmt.Printf("     [%d] Score: %.3f | %s\n", i+1, score, truncate(doc.Content, 50))
	}
}

func demoFusionStrategy(ctx context.Context, retriever *graphrag.GraphRAGRetriever, query string, strategy graphrag.FusionStrategy) {
	strategyNames := map[graphrag.FusionStrategy]string{
		graphrag.FusionStrategyWeighted: "加权融合",
		graphrag.FusionStrategyRRF:      "RRF 融合",
		graphrag.FusionStrategyMax:      "最大值融合",
		graphrag.FusionStrategyMin:      "最小值融合",
	}

	start := time.Now()

	results, err := retriever.Search(ctx, query, graphrag.SearchOptions{
		Mode:           graphrag.SearchModeHybrid,
		K:              5,
		FusionStrategy: strategy,
	})

	duration := time.Since(start)

	if err != nil {
		fmt.Printf("  ❌ %s 失败: %v\n", strategyNames[strategy], err)
		return
	}

	fmt.Printf("\n  📌 %s (耗时: %v)\n", strategyNames[strategy], duration)
	fmt.Printf("     Top 3 结果:\n")

	for i, doc := range results {
		if i >= 3 {
			break
		}
		score := 0.0
		if scoreVal, ok := doc.Metadata["score"].(float64); ok {
			score = scoreVal
		}
		fmt.Printf("     [%d] Score: %.3f | %s\n", i+1, score, truncate(doc.Content, 50))
	}
}

func demoRerankStrategy(ctx context.Context, retriever *graphrag.GraphRAGRetriever, query string, strategy graphrag.RerankStrategy) {
	strategyNames := map[graphrag.RerankStrategy]string{
		graphrag.RerankStrategyScore:     "分数重排",
		graphrag.RerankStrategyDiversity: "多样性重排",
		graphrag.RerankStrategyMMR:       "MMR 重排",
	}

	start := time.Now()

	results, err := retriever.Search(ctx, query, graphrag.SearchOptions{
		Mode:           graphrag.SearchModeHybrid,
		K:              5,
		RerankStrategy: strategy,
	})

	duration := time.Since(start)

	if err != nil {
		fmt.Printf("  ❌ %s 失败: %v\n", strategyNames[strategy], err)
		return
	}

	fmt.Printf("\n  📌 %s (耗时: %v)\n", strategyNames[strategy], duration)
	fmt.Printf("     Top 3 结果:\n")

	for i, doc := range results {
		if i >= 3 {
			break
		}
		score := 0.0
		if scoreVal, ok := doc.Metadata["score"].(float64); ok {
			score = scoreVal
		}
		fmt.Printf("     [%d] Score: %.3f | %s\n", i+1, score, truncate(doc.Content, 50))
	}
}

func displayStatistics(stats graphrag.Statistics) {
	fmt.Printf("向量检索结果数: %d\n", stats.VectorResultsCount)
	fmt.Printf("图检索结果数: %d\n", stats.GraphResultsCount)
	fmt.Printf("融合后结果数: %d\n", stats.FusedResultsCount)
	fmt.Printf("提取的实体数: %d\n", stats.EntitiesExtracted)
	fmt.Printf("遍历的节点数: %d\n", stats.NodesTraversed)
	fmt.Printf("平均图深度: %.3f\n", stats.AverageGraphDepth)
	fmt.Printf("向量检索耗时: %d ms\n", stats.VectorSearchTime)
	fmt.Printf("图检索耗时: %d ms\n", stats.GraphSearchTime)
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// Mock 实现

type mockEntityExtractor struct{}

func (m *mockEntityExtractor) Extract(ctx context.Context, text string) ([]builder.Entity, error) {
	// 简单的关键词提取
	keywords := []string{"人工智能", "机器学习", "深度学习", "自然语言处理", "计算机视觉"}

	entities := []builder.Entity{}
	for _, keyword := range keywords {
		if contains(text, keyword) {
			entities = append(entities, builder.Entity{
				ID:   fmt.Sprintf("entity_%s", keyword),
				Name: keyword,
				Type: "Concept",
			})
		}
	}

	return entities, nil
}

func (m *mockEntityExtractor) ExtractWithSchema(ctx context.Context, text string, schema *builder.EntitySchema) ([]builder.Entity, error) {
	// 简化实现，忽略 schema
	return m.Extract(ctx, text)
}

type mockRelationExtractor struct{}

func (m *mockRelationExtractor) Extract(ctx context.Context, text string, entities []builder.Entity) ([]builder.Relation, error) {
	relations := []builder.Relation{}

	// 简单的关系提取逻辑
	if len(entities) >= 2 {
		for i := 0; i < len(entities)-1; i++ {
			relations = append(relations, builder.Relation{
				Source: entities[i].ID,
				Target: entities[i+1].ID,
				Type:   "RELATED_TO",
			})
		}
	}

	return relations, nil
}

func (m *mockRelationExtractor) ExtractWithSchema(ctx context.Context, text string, entities []builder.Entity, schema *builder.RelationSchema) ([]builder.Relation, error) {
	// 简化实现，忽略 schema
	return m.Extract(ctx, text, entities)
}

func contains(text, keyword string) bool {
	return strings.Contains(text, keyword)
}
