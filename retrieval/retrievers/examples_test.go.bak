package retrievers_test

import (
	"context"
	"fmt"

	"langchain-go/core/chat/ollama"
	"langchain-go/retrieval/embeddings"
	"langchain-go/retrieval/loaders"
	"langchain-go/retrieval/retrievers"
	"langchain-go/retrieval/vectorstores"
)

// Example_vectorStoreRetriever 向量存储检索器示例
func Example_vectorStoreRetriever() {
	ctx := context.Background()

	// 1. 创建向量存储
	embedder := embeddings.NewOllamaEmbeddings("nomic-embed-text")
	vectorStore := vectorstores.NewInMemoryVectorStore(embedder)

	// 2. 添加文档
	docs := []*loaders.Document{
		{Content: "LangChain 是一个 LLM 应用框架。"},
		{Content: "RAG 结合了检索和生成。"},
		{Content: "向量数据库用于语义搜索。"},
	}
	vectorStore.AddDocuments(ctx, docs)

	// 3. 创建检索器 (简单!)
	retriever := retrievers.NewVectorStoreRetriever(vectorStore,
		retrievers.WithTopK(2),
		retrievers.WithScoreThreshold(0.5),
	)

	// 4. 执行检索
	results, _ := retriever.GetRelevantDocuments(ctx, "什么是 RAG?")

	fmt.Printf("找到 %d 个相关文档\n", len(results))

	// Output:
	// 找到 2 个相关文档
}

// Example_multiQueryRetriever 多查询检索器示例
func Example_multiQueryRetriever() {
	// ctx := context.Background()

	// // 基础检索器
	// vectorRetriever := retrievers.NewVectorStoreRetriever(vectorStore)

	// // LLM 用于生成查询变体
	// llm := ollama.NewChatOllama("qwen2.5:7b")

	// // 多查询检索器
	// multiRetriever := retrievers.NewMultiQueryRetriever(
	// 	vectorRetriever,
	// 	llm,
	// 	retrievers.WithNumQueries(3),          // 生成 3 个查询变体
	// 	retrievers.WithIncludeOriginal(true),  // 包含原始查询
	// )

	// // 自动生成多个查询并检索
	// results, _ := multiRetriever.GetRelevantDocuments(ctx, "如何使用 LangChain?")

	// fmt.Printf("找到 %d 个相关文档\n", len(results))

	fmt.Println("示例代码 (需要运行 Ollama)")

	// Output:
	// 示例代码 (需要运行 Ollama)
}

// Example_ensembleRetriever 集成检索器示例
func Example_ensembleRetriever() {
	// ctx := context.Background()

	// // 向量检索器
	// vectorRetriever := retrievers.NewVectorStoreRetriever(vectorStore)

	// // BM25 检索器 (假设已实现)
	// // bm25Retriever := retrievers.NewBM25Retriever(documents)

	// // 集成检索器 (混合检索)
	// ensemble := retrievers.NewEnsembleRetriever(
	// 	[]retrievers.Retriever{vectorRetriever /*, bm25Retriever*/},
	// 	retrievers.WithWeights([]float64{0.5, 0.5}), // 等权重
	// 	retrievers.WithRRFK(60),                     // RRF 常数
	// )

	// // 自动融合多个检索器的结果
	// results, _ := ensemble.GetRelevantDocuments(ctx, "LangChain 教程")

	// fmt.Printf("融合后找到 %d 个文档\n", len(results))

	fmt.Println("示例代码 (需要实现 BM25)")

	// Output:
	// 示例代码 (需要实现 BM25)
}

// Example_completeWorkflow 完整的检索工作流
func Example_completeWorkflow() {
	ctx := context.Background()

	// 步骤 1: 准备数据
	docs := []*loaders.Document{
		{Content: "LangChain 是一个开源框架，用于构建大语言模型应用。"},
		{Content: "RAG (检索增强生成) 通过检索相关文档来增强 LLM 的回答。"},
		{Content: "向量数据库可以高效地存储和检索文档的向量表示。"},
		{Content: "Milvus 是一个流行的开源向量数据库。"},
	}

	// 步骤 2: 创建向量存储
	embedder := embeddings.NewOllamaEmbeddings("nomic-embed-text")
	vectorStore := vectorstores.NewInMemoryVectorStore(embedder)
	vectorStore.AddDocuments(ctx, docs)

	// 步骤 3: 创建基础检索器
	baseRetriever := retrievers.NewVectorStoreRetriever(vectorStore,
		retrievers.WithTopK(3),
	)

	// 步骤 4: 包装为多查询检索器 (可选)
	llm := ollama.NewChatOllama("qwen2.5:7b")
	multiRetriever := retrievers.NewMultiQueryRetriever(
		baseRetriever,
		llm,
		retrievers.WithNumQueries(2),
	)

	// 步骤 5: 执行检索
	results, err := multiRetriever.GetRelevantDocuments(ctx, "什么是 RAG?")
	if err != nil {
		fmt.Printf("检索失败: %v\n", err)
		return
	}

	fmt.Printf("找到 %d 个相关文档\n", len(results))
	for i, doc := range results {
		fmt.Printf("[%d] %s\n", i+1, doc.Content[:30]+"...")
	}

	// Output:
	// 找到 3 个相关文档
	// [1] LangChain 是一个开源框架，用于构建大语言模型应...
	// [2] RAG (检索增强生成) 通过检索相关文档来增强 LLM...
	// [3] 向量数据库可以高效地存储和检索文档的向量表示。...
}
