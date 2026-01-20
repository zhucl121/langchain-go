package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/zhucl121/langchain-go/core/chat/providers/openai"
	"github.com/zhucl121/langchain-go/retrieval/embeddings"
	"github.com/zhucl121/langchain-go/retrieval/graphdb"
	"github.com/zhucl121/langchain-go/retrieval/graphdb/builder"
	"github.com/zhucl121/langchain-go/retrieval/graphdb/mock"
	"github.com/zhucl121/langchain-go/retrieval/graphdb/neo4j"
)

func main() {
	ctx := context.Background()

	// 选择运行模式
	mode := os.Getenv("MODE")
	if mode == "" {
		mode = "mock" // 默认使用 mock
	}

	fmt.Printf("Running in %s mode...\n\n", mode)

	// 1. 创建图数据库实例
	var graphDB graphdb.GraphDB
	var err error

	switch mode {
	case "neo4j":
		// Neo4j 模式
		config := neo4j.DefaultConfig()
		config.URI = getEnv("NEO4J_URI", "bolt://localhost:7687")
		config.Username = getEnv("NEO4J_USERNAME", "neo4j")
		config.Password = getEnv("NEO4J_PASSWORD", "password123")
		config.Database = getEnv("NEO4J_DATABASE", "neo4j")

		neo4jDriver, err := neo4j.NewNeo4jDriver(config)
		if err != nil {
			log.Fatalf("Failed to create Neo4j driver: %v", err)
		}

		if err := neo4jDriver.Connect(ctx); err != nil {
			log.Fatalf("Failed to connect to Neo4j: %v", err)
		}
		defer neo4jDriver.Close()

		graphDB = neo4jDriver
		fmt.Println("Connected to Neo4j!")

	default:
		// Mock 模式（默认）
		mockDB := mock.NewMockGraphDB()
		if err := mockDB.Connect(ctx); err != nil {
			log.Fatalf("Failed to connect to mock database: %v", err)
		}
		defer mockDB.Close()

		graphDB = mockDB
		fmt.Println("Using Mock GraphDB!")
	}

	// 2. 创建提取器（需要配置 OpenAI API Key 或使用 Mock）
	var entityExtractor builder.EntityExtractor
	var relationExtractor builder.RelationExtractor
	var embedder builder.Embedder

	apiKey := os.Getenv("OPENAI_API_KEY")

	if apiKey != "" {
		// 使用真实的 OpenAI 模型
		chatModel, err := openai.New(openai.Config{
			APIKey: apiKey,
			Model:  "gpt-4",
		})
		if err != nil {
			log.Fatalf("Failed to create OpenAI chat model: %v", err)
		}

		entityExtractor = builder.NewLLMEntityExtractor(chatModel, nil)
		relationExtractor = builder.NewLLMRelationExtractor(chatModel, nil)

		// 使用 OpenAI Embeddings
		embedModel := embeddings.NewOpenAIEmbeddings(embeddings.OpenAIEmbeddingsConfig{
			APIKey: apiKey,
			Model:  "text-embedding-3-small",
		})
		embedder = builder.NewEmbeddingModelAdapter(embedModel)

		fmt.Println("Using OpenAI for extraction and embedding!")
	} else {
		// 使用 Mock（演示用）
		fmt.Println("Using Mock extractors (set OPENAI_API_KEY for real extraction)!")

		entityExtractor = &mockEntityExtractor{}
		relationExtractor = &mockRelationExtractor{}
		embedder = builder.NewMockEmbedder(384)
	}

	// 3. 创建 KGBuilder
	config := builder.KGBuilderConfig{
		GraphDB:           graphDB,
		EntityExtractor:   entityExtractor,
		RelationExtractor: relationExtractor,
		Embedder:          embedder,
		EnableEmbedding:   true,
		EnableDisambiguation: false,
		EnableValidation:  false,
		BatchSize:         10,
		MaxConcurrency:    5,
	}

	kgBuilder, err := builder.NewKGBuilder(config)
	if err != nil {
		log.Fatalf("Failed to create KGBuilder: %v", err)
	}

	// 4. 示例文本
	texts := []string{
		"John Smith is the CEO of TechCorp, a leading technology company based in San Francisco.",
		"Alice Johnson works as a senior engineer at TechCorp. She specializes in AI and machine learning.",
		"Bob Chen founded DataFlow in 2020. The company focuses on big data analytics.",
		"TechCorp acquired DataFlow in 2023 for $500 million.",
	}

	fmt.Println("\n=== Processing Texts ===\n")

	// 5. 批量构建知识图谱
	graphs, err := kgBuilder.BuildBatch(ctx, texts)
	if err != nil {
		log.Fatalf("BuildBatch failed: %v", err)
	}

	// 6. 打印每个图谱的结果
	for i, kg := range graphs {
		fmt.Printf("Graph %d (from text %d):\n", i+1, i+1)
		fmt.Printf("  Text: %s\n", texts[i])
		fmt.Printf("  Entities: %d\n", len(kg.Entities))
		for _, entity := range kg.Entities {
			fmt.Printf("    - [%s] %s (%s) - confidence: %.2f\n",
				entity.ID, entity.Name, entity.Type, entity.Confidence)
			if len(entity.Embedding) > 0 {
				fmt.Printf("      Embedding: [%.3f, %.3f, ...] (dim=%d)\n",
					entity.Embedding[0], entity.Embedding[1], len(entity.Embedding))
			}
		}

		fmt.Printf("  Relations: %d\n", len(kg.Relations))
		for _, relation := range kg.Relations {
			fmt.Printf("    - %s -[%s]-> %s (weight: %.2f, confidence: %.2f)\n",
				relation.Source, relation.Type, relation.Target, relation.Weight, relation.Confidence)
		}
		fmt.Println()
	}

	// 7. 合并所有图谱
	fmt.Println("=== Merging All Graphs ===\n")
	mergedKG, err := kgBuilder.Merge(ctx, graphs)
	if err != nil {
		log.Fatalf("Merge failed: %v", err)
	}

	fmt.Printf("Merged Knowledge Graph:\n")
	fmt.Printf("  Total Entities: %d\n", len(mergedKG.Entities))
	fmt.Printf("  Total Relations: %d\n", len(mergedKG.Relations))
	fmt.Println()

	// 8. 存储到图数据库
	fmt.Println("=== Storing to Graph Database ===\n")

	// 为每个实体和关系存储
	nodes := make([]*graphdb.Node, len(mergedKG.Entities))
	for i, entity := range mergedKG.Entities {
		nodes[i] = entity.ToNode()
	}

	if err := graphDB.BatchAddNodes(ctx, nodes); err != nil {
		log.Printf("Warning: failed to add nodes: %v", err)
	} else {
		fmt.Printf("✓ Stored %d nodes\n", len(nodes))
	}

	edges := make([]*graphdb.Edge, len(mergedKG.Relations))
	for i, relation := range mergedKG.Relations {
		edges[i] = relation.ToEdge()
	}

	if err := graphDB.BatchAddEdges(ctx, edges); err != nil {
		log.Printf("Warning: failed to add edges: %v", err)
	} else {
		fmt.Printf("✓ Stored %d edges\n", len(edges))
	}

	// 9. 查询验证
	if mode == "neo4j" || mode == "mock" {
		fmt.Println("\n=== Verification ===\n")

		// 查询一个实体
		if len(mergedKG.Entities) > 0 {
			firstEntity := mergedKG.Entities[0]
			node, err := graphDB.GetNode(ctx, firstEntity.ID)
			if err != nil {
				log.Printf("Warning: failed to get node: %v", err)
			} else {
				fmt.Printf("✓ Verified node '%s' exists in database\n", node.Label)
			}
		}
	}

	fmt.Println("\n=== Demo Complete ===")
}

// getEnv 获取环境变量，如果不存在则返回默认值
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// mockEntityExtractor Mock 实体提取器（用于演示）
type mockEntityExtractor struct{}

func (m *mockEntityExtractor) Extract(ctx context.Context, text string) ([]builder.Entity, error) {
	// 简单的 mock：创建一些虚拟实体
	entities := []builder.Entity{
		{
			ID:          fmt.Sprintf("entity-mock-%d", len(text)),
			Type:        builder.EntityTypePerson,
			Name:        "Mock Person",
			Description: "A mock entity for demo",
			Properties:  make(map[string]interface{}),
			Metadata:    make(map[string]interface{}),
			SourceText:  text,
			Confidence:  0.8,
		},
	}
	return entities, nil
}

func (m *mockEntityExtractor) ExtractWithSchema(ctx context.Context, text string, schema *builder.EntitySchema) ([]builder.Entity, error) {
	return m.Extract(ctx, text)
}

// mockRelationExtractor Mock 关系提取器（用于演示）
type mockRelationExtractor struct{}

func (m *mockRelationExtractor) Extract(ctx context.Context, text string, entities []builder.Entity) ([]builder.Relation, error) {
	// 简单的 mock：不创建关系
	return []builder.Relation{}, nil
}

func (m *mockRelationExtractor) ExtractWithSchema(ctx context.Context, text string, entities []builder.Entity, schema *builder.RelationSchema) ([]builder.Relation, error) {
	return m.Extract(ctx, text, entities)
}
