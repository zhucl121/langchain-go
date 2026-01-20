// Package graphrag 提供 GraphRAG (Graph Retrieval Augmented Generation) 检索器。
//
// GraphRAG 结合向量检索和图遍历，实现更智能、更全面的信息检索。
//
// # 核心概念
//
// ## GraphRAG 架构
//
// GraphRAG 检索流程分为以下几个步骤：
//
//  1. **向量检索** - 使用向量数据库进行语义搜索
//  2. **实体识别** - 从查询中提取实体
//  3. **图遍历** - 在知识图谱中遍历相关节点
//  4. **结果融合** - 融合向量和图检索结果
//  5. **重排序** - 基于相关性或多样性重排
//  6. **上下文增强** - 添加图结构信息
//
// ## 主要组件
//
// ### GraphRAGRetriever - 核心检索器
//
//	config := graphrag.Config{
//	    GraphDB:      neo4jDriver,
//	    VectorStore:  vectorStore,
//	    VectorWeight: 0.6,
//	    GraphWeight:  0.4,
//	}
//
//	retriever, _ := graphrag.NewGraphRAGRetriever(config)
//	docs, _ := retriever.Search(ctx, "query text")
//
// ### 融合策略 (FusionStrategy)
//
//   - **Weighted** - 加权融合（默认）
//   - **RRF** - Reciprocal Rank Fusion
//   - **Max** - 取最大值
//   - **Min** - 取最小值
//
// ### 重排序策略 (RerankStrategy)
//
//   - **Score** - 基于分数重排（默认）
//   - **Diversity** - 基于多样性重排
//   - **MMR** - Maximal Marginal Relevance
//
// # 使用示例
//
// ## 基础用法
//
//	// 1. 准备组件
//	graphDB := neo4j.NewNeo4jDriver(config)
//	vectorStore := vectorstores.NewInMemoryVectorStore(embeddings)
//	entityExtractor := builder.NewLLMEntityExtractor(chatModel, nil)
//
//	// 2. 创建检索器
//	config := graphrag.Config{
//	    GraphDB:          graphDB,
//	    VectorStore:      vectorStore,
//	    EntityExtractor:  entityExtractor,
//	    VectorWeight:     0.6,
//	    GraphWeight:      0.4,
//	    MaxTraverseDepth: 2,
//	    TopK:             10,
//	}
//
//	retriever, _ := graphrag.NewGraphRAGRetriever(config)
//
//	// 3. 执行检索
//	docs, _ := retriever.Search(ctx, "Who is the CEO of TechCorp?")
//
//	for _, doc := range docs {
//	    fmt.Println(doc.Content)
//	    fmt.Println("Related entities:", doc.Metadata["related_entities"])
//	}
//
// ## 自定义检索选项
//
//	// 使用 RRF 融合和 MMR 重排序
//	opts := graphrag.SearchOptions{
//	    Mode:                      graphrag.SearchModeHybrid,
//	    K:                         20,
//	    VectorWeight:              0.7,
//	    GraphWeight:               0.3,
//	    MaxTraverseDepth:          3,
//	    FusionStrategy:            graphrag.FusionStrategyRRF,
//	    RerankStrategy:            graphrag.RerankStrategyMMR,
//	    EnableContextAugmentation: true,
//	    MinScore:                  0.5,
//	}
//
//	docs, _ := retriever.Search(ctx, query, opts)
//
// ## 仅向量检索
//
//	opts := graphrag.SearchOptions{
//	    Mode: graphrag.SearchModeVector,
//	    K:    10,
//	}
//
//	docs, _ := retriever.Search(ctx, query, opts)
//
// ## 仅图检索
//
//	opts := graphrag.SearchOptions{
//	    Mode:             graphrag.SearchModeGraph,
//	    K:                10,
//	    MaxTraverseDepth: 2,
//	}
//
//	docs, _ := retriever.Search(ctx, query, opts)
//
// ## 查看统计信息
//
//	docs, _ := retriever.Search(ctx, query)
//
//	stats := retriever.GetStatistics()
//	fmt.Printf("Vector results: %d\n", stats.VectorResultsCount)
//	fmt.Printf("Graph results: %d\n", stats.GraphResultsCount)
//	fmt.Printf("Total time: %dms\n", stats.TotalTime)
//
// # 融合策略详解
//
// ## Weighted Fusion (加权融合)
//
// 最简单直观的融合方式：
//
//	FusedScore = VectorWeight * VectorScore + GraphWeight * GraphScore
//
// 适合大多数场景。
//
// ## RRF (Reciprocal Rank Fusion)
//
// 基于排名的融合方式，对不同scale的分数更鲁棒：
//
//	RRF_Score = sum(1 / (k + rank_i))
//
// 适合结合多个不同类型的检索器。
//
// ## Max Fusion (最大值融合)
//
// 取向量和图分数的最大值：
//
//	FusedScore = max(VectorScore, GraphScore)
//
// 适合任一来源的高分结果都重要的场景。
//
// ## Min Fusion (最小值融合)
//
// 取向量和图分数的最小值：
//
//	FusedScore = min(VectorScore, GraphScore)
//
// 适合需要同时在两个来源中得分都高的结果。
//
// # 重排序策略详解
//
// ## Score Reranking (分数重排)
//
// 直接按融合分数排序，最简单直接。
//
// ## Diversity Reranking (多样性重排)
//
// 在保持相关性的同时增加结果多样性：
//
//   - 选择第一个最高分结果
//   - 迭代选择与已选结果最不相似的
//
// 适合需要展示不同角度信息的场景。
//
// ## MMR (Maximal Marginal Relevance)
//
// 平衡相关性和多样性：
//
//	MMR = λ * Sim(D, Q) - (1-λ) * max Sim(D, D_i)
//
// 其中 λ 控制平衡：
//   - λ = 1: 完全基于相关性
//   - λ = 0: 完全基于多样性
//   - λ = 0.5: 平衡（默认）
//
// # 上下文增强
//
// GraphRAG 为每个检索结果添加图结构信息：
//
//	doc.Metadata["related_entities"]  // 相关实体列表
//	doc.Metadata["relationship_paths"] // 关系路径
//	doc.Metadata["neighbor_count"]     // 邻居数量
//	doc.Metadata["graph_depth"]        // 图深度
//
// 这些信息可以帮助 LLM 更好地理解上下文。
//
// # 性能优化
//
// ## 1. 调整 TopK
//
//	// 向量检索获取更多候选，再融合筛选
//	config.TopK = 20  // 最终返回
//	// 内部会获取 TopK * 2 的向量结果
//
// ## 2. 控制遍历深度
//
//	// 深度越大，图遍历越慢
//	config.MaxTraverseDepth = 2  // 推荐值
//
// ## 3. 权重调优
//
//	// 根据数据特点调整
//	config.VectorWeight = 0.7  // 文本相似度更重要
//	config.GraphWeight = 0.3   // 图结构辅助
//
// ## 4. 选择合适的融合策略
//
//	// RRF 对分数 scale 更鲁棒
//	config.FusionStrategy = graphrag.FusionStrategyRRF
//
// # 最佳实践
//
// 1. **构建高质量知识图谱** - 使用 builder 包自动构建
// 2. **实体识别准确** - 使用好的 LLM 进行实体提取
// 3. **向量化一致** - 查询和文档使用相同的 Embedding 模型
// 4. **权重调优** - 根据实际数据调整向量和图权重
// 5. **监控统计** - 使用 GetStatistics() 监控性能
//
// # 与 LangChain/LangGraph Python 对比
//
// ## 相似之处
//
//   - 混合检索理念相同
//   - 支持多种融合策略
//   - 上下文增强机制
//
// ## 优势
//
//   - Go 原生性能优势
//   - 类型安全
//   - 更简洁的 API
//   - 内置统计和监控
//
// # 参考资料
//
//   - GraphDB: github.com/zhucl121/langchain-go/retrieval/graphdb
//   - VectorStores: github.com/zhucl121/langchain-go/retrieval/vectorstores
//   - KGBuilder: github.com/zhucl121/langchain-go/retrieval/graphdb/builder
//
package graphrag
