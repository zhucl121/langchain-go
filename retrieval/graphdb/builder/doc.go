// Package builder 提供知识图谱构建功能。
//
// builder 包实现了从文本自动构建知识图谱的完整流程，包括：
//
//   - 实体提取（Entity Extraction）
//   - 关系抽取（Relation Extraction）
//   - 实体消歧（Entity Disambiguation）
//   - 向量化（Embedding）
//   - 图谱构建和存储
//
// # 核心组件
//
// ## KGBuilder - 知识图谱构建器
//
// KGBuilder 是核心接口，协调整个构建流程：
//
//	config := builder.KGBuilderConfig{
//	    GraphDB: neo4jDriver,
//	    EntityExtractor: entityExtractor,
//	    RelationExtractor: relationExtractor,
//	    Embedder: embedder,
//	    EnableEmbedding: true,
//	    EnableDisambiguation: true,
//	}
//
//	kgBuilder, _ := builder.NewKGBuilder(config)
//	kg, _ := kgBuilder.Build(ctx, "John works at TechCorp in Beijing.")
//
// ## EntityExtractor - 实体提取器
//
// EntityExtractor 从文本中识别实体：
//
//	// 基于 LLM 的实体提取
//	extractor := builder.NewLLMEntityExtractor(chatModel, nil)
//	entities, _ := extractor.Extract(ctx, text)
//
// ## RelationExtractor - 关系提取器
//
// RelationExtractor 从文本和实体中提取关系：
//
//	// 基于 LLM 的关系提取
//	extractor := builder.NewLLMRelationExtractor(chatModel, nil)
//	relations, _ := extractor.Extract(ctx, text, entities)
//
// ## Embedder - 向量化器
//
// Embedder 将实体转换为向量表示：
//
//	// 使用 OpenAI Embeddings
//	embedModel := embeddings.NewOpenAIEmbeddings(apiKey)
//	embedder := builder.NewEmbeddingModelAdapter(embedModel)
//
// # 实体和关系
//
// ## Entity - 实体
//
// Entity 表示知识图谱中的节点：
//
//	entity := builder.Entity{
//	    ID:          "person-1",
//	    Type:        builder.EntityTypePerson,
//	    Name:        "John Smith",
//	    Description: "CEO of TechCorp",
//	    Properties: map[string]interface{}{
//	        "age": 45,
//	        "role": "CEO",
//	    },
//	    Confidence: 0.95,
//	}
//
// ## Relation - 关系
//
// Relation 表示实体之间的连接：
//
//	relation := builder.Relation{
//	    ID:          "rel-1",
//	    Type:        builder.RelationTypeWorksFor,
//	    Source:      "person-1",
//	    Target:      "org-1",
//	    Description: "works as CEO",
//	    Directed:    true,
//	    Weight:      1.0,
//	    Confidence:  0.9,
//	}
//
// # Schema 定义
//
// ## EntitySchema - 实体 Schema
//
// EntitySchema 定义实体类型的结构：
//
//	schema := &builder.EntitySchema{
//	    Type:        "Person",
//	    Description: "A human being",
//	    Properties: map[string]builder.PropertySchema{
//	        "age": {
//	            Type:        "number",
//	            Description: "Age in years",
//	            Required:    false,
//	        },
//	    },
//	    Required: []string{"name"},
//	}
//
// ## RelationSchema - 关系 Schema
//
// RelationSchema 定义关系类型的结构：
//
//	schema := &builder.RelationSchema{
//	    Type:        "WORKS_FOR",
//	    Description: "Employment relationship",
//	    SourceTypes: []string{"Person"},
//	    TargetTypes: []string{"Organization"},
//	    Directed:    true,
//	}
//
// # 使用示例
//
// ## 基础用法
//
//	// 1. 创建组件
//	chatModel := openai.NewChatModel(config)
//	entityExtractor := builder.NewLLMEntityExtractor(chatModel, nil)
//	relationExtractor := builder.NewLLMRelationExtractor(chatModel, nil)
//
//	// 2. 配置 KGBuilder
//	config := builder.KGBuilderConfig{
//	    GraphDB:           neo4jDriver,
//	    EntityExtractor:   entityExtractor,
//	    RelationExtractor: relationExtractor,
//	    EnableEmbedding:   false, // 暂时禁用向量化
//	}
//
//	kgBuilder, _ := builder.NewKGBuilder(config)
//
//	// 3. 构建知识图谱
//	text := "John Smith is the CEO of TechCorp, located in Beijing."
//	kg, _ := kgBuilder.BuildAndStore(ctx, text)
//
//	fmt.Printf("Extracted %d entities and %d relations\n",
//	    len(kg.Entities), len(kg.Relations))
//
// ## 批量构建
//
//	texts := []string{
//	    "Alice works at Google.",
//	    "Bob is the founder of Meta.",
//	    "Charlie lives in New York.",
//	}
//
//	graphs, _ := kgBuilder.BuildBatch(ctx, texts)
//
//	// 合并所有图谱
//	mergedKG, _ := kgBuilder.Merge(ctx, graphs)
//
// ## 增量更新
//
//	// 首次构建
//	kg1, _ := kgBuilder.BuildAndStore(ctx, "Initial text...")
//
//	// 增量更新（添加新信息）
//	result, _ := kgBuilder.UpdateIncremental(ctx, "New information...")
//
//	fmt.Printf("Added %d new entities and %d new relations\n",
//	    len(result.Entities), len(result.Relations))
//
// ## 启用向量化
//
//	// 使用 OpenAI Embeddings
//	embedModel := embeddings.NewOpenAIEmbeddings(apiKey)
//	embedder := builder.NewEmbeddingModelAdapter(embedModel)
//
//	config.Embedder = embedder
//	config.EnableEmbedding = true
//
//	// 构建时自动为实体生成向量
//	kg, _ := kgBuilder.Build(ctx, text)
//
//	// 每个实体现在都有向量表示
//	for _, entity := range kg.Entities {
//	    fmt.Printf("Entity %s has embedding of dimension %d\n",
//	        entity.Name, len(entity.Embedding))
//	}
//
// # 最佳实践
//
// 1. **选择合适的 LLM**：实体和关系提取的质量取决于 LLM 的能力。
//    推荐使用 GPT-4、Claude-3 等高质量模型。
//
// 2. **使用 Schema 指导提取**：定义明确的 EntitySchema 和 RelationSchema
//    可以提高提取的准确性和一致性。
//
// 3. **启用实体消歧**：对于大规模知识图谱，实体消歧可以避免重复实体。
//
// 4. **批量处理**：使用 BuildBatch 处理大量文本可以提高效率。
//
// 5. **增量更新**：对于动态内容，使用 UpdateIncremental 而非完全重建。
//
// # 性能优化
//
// 1. **并发控制**：使用 MaxConcurrency 限制并发数，避免资源耗尽。
//
// 2. **批量大小**：调整 BatchSize 以平衡内存使用和吞吐量。
//
// 3. **缓存 LLM 结果**：对于相同文本，缓存提取结果避免重复调用。
//
// 4. **异步处理**：对于非实时场景，使用后台任务处理大规模文本。
//
// # 错误处理
//
//	kg, err := kgBuilder.Build(ctx, text)
//	if err != nil {
//	    switch {
//	    case errors.Is(err, context.DeadlineExceeded):
//	        // 超时处理
//	    case errors.Is(err, graphdb.ErrConnectionFailed):
//	        // 数据库连接失败
//	    default:
//	        // 其他错误
//	    }
//	}
//
// # 参考资料
//
//   - GraphDB: github.com/zhucl121/langchain-go/retrieval/graphdb
//   - ChatModel: github.com/zhucl121/langchain-go/core/chat
//   - Embeddings: github.com/zhucl121/langchain-go/retrieval/embeddings
//
package builder
