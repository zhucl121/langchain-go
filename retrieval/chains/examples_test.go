// Package chains 示例文档
//
// 这个文件展示了如何使用 chains 包。
//
package chains_test

import (
	"context"
	"fmt"

	"langchain-go/core/chat/ollama"
	"langchain-go/retrieval/chains"
	"langchain-go/retrieval/embeddings"
	"langchain-go/retrieval/loaders"
	"langchain-go/retrieval/vectorstores"
)

// Example_completeRAG 完整的 RAG 示例
func Example_completeRAG() {
	ctx := context.Background()

	// 步骤 1: 准备文档
	docs := []*loaders.Document{
		{
			Content: "LangChain 是一个用于构建大语言模型应用的开源框架。",
			Metadata: map[string]interface{}{
				"source": "langchain_intro.txt",
			},
		},
		{
			Content: "RAG (检索增强生成) 是一种结合检索和生成的技术，可以让 LLM 访问外部知识库。",
			Metadata: map[string]interface{}{
				"source": "rag_explanation.txt",
			},
		},
		{
			Content: "向量数据库用于存储和检索文档的向量表示，支持语义搜索。",
			Metadata: map[string]interface{}{
				"source": "vector_db.txt",
			},
		},
	}

	// 步骤 2: 创建向量存储
	embedder := embeddings.NewOllamaEmbeddings("nomic-embed-text")
	vectorStore := vectorstores.NewInMemoryVectorStore(embedder)

	// 添加文档
	_, err := vectorStore.AddDocuments(ctx, docs)
	if err != nil {
		fmt.Printf("添加文档失败: %v\n", err)
		return
	}

	// 步骤 3: 创建 RAG Chain (只需 3 行!)
	// 注意：实际使用需要先实现 retrievers 包
	// retriever := retrievers.NewVectorStoreRetriever(vectorStore)
	llm := ollama.NewChatOllama("qwen2.5:7b")
	// ragChain := chains.NewRAGChain(retriever, llm)

	// 步骤 4: 执行查询
	// result, err := ragChain.Run(ctx, "什么是 RAG?")
	// if err != nil {
	// 	fmt.Printf("查询失败: %v\n", err)
	// 	return
	// }

	// 步骤 5: 显示结果
	// fmt.Println("问题:", result.Question)
	// fmt.Println("答案:", result.Answer)
	// fmt.Printf("置信度: %.2f\n", result.Confidence)
	// fmt.Printf("耗时: %v\n", result.TimeElapsed)
	// fmt.Println("\n来源文档:")
	// for i, doc := range result.Context {
	// 	fmt.Printf("[%d] %s\n", i+1, doc.Metadata["source"])
	// }

	fmt.Println("示例代码 (需要实现 retrievers 包)")
	_ = llm
	_ = vectorStore

	// Output:
	// 示例代码 (需要实现 retrievers 包)
}

// Example_streamingRAG 流式 RAG 示例
func Example_streamingRAG() {
	// ctx := context.Background()

	// // 创建 RAG Chain
	// retriever := retrievers.NewVectorStoreRetriever(vectorStore)
	// llm := ollama.NewChatOllama("qwen2.5:7b")
	// ragChain := chains.NewRAGChain(retriever, llm)

	// // 流式执行
	// stream, err := ragChain.Stream(ctx, "解释一下 LangChain 的核心概念")
	// if err != nil {
	// 	fmt.Printf("流式执行失败: %v\n", err)
	// 	return
	// }

	// fmt.Println("🤖 AI 助手正在思考...\n")

	// for chunk := range stream {
	// 	switch chunk.Type {
	// 	case "start":
	// 		fmt.Println("✓ 开始处理")

	// 	case "retrieval":
	// 		data := chunk.Data.(map[string]interface{})
	// 		count := data["count"].(int)
	// 		fmt.Printf("✓ 检索到 %d 个相关文档\n\n", count)
	// 		fmt.Println("回答:")

	// 	case "llm_token":
	// 		// 实时打印 token
	// 		fmt.Print(chunk.Data.(string))

	// 	case "done":
	// 		result := chunk.Data.(chains.RAGResult)
	// 		fmt.Printf("\n\n✓ 完成 (耗时: %v)\n", result.TimeElapsed)
	// 		fmt.Printf("置信度: %.2f\n", result.Confidence)

	// 	case "error":
	// 		fmt.Printf("❌ 错误: %v\n", chunk.Data)
	// 	}
	// }

	fmt.Println("示例代码 (需要实现 retrievers 包)")

	// Output:
	// 示例代码 (需要实现 retrievers 包)
}

// Example_batchRAG 批量 RAG 示例
func Example_batchRAG() {
	// ctx := context.Background()

	// // 创建 RAG Chain
	// retriever := retrievers.NewVectorStoreRetriever(vectorStore)
	// llm := ollama.NewChatOllama("qwen2.5:7b")
	// ragChain := chains.NewRAGChain(retriever, llm)

	// // 批量查询
	// questions := []string{
	// 	"什么是 LangChain?",
	// 	"什么是 RAG?",
	// 	"什么是向量数据库?",
	// }

	// fmt.Println("批量处理 3 个问题...\n")

	// results, err := ragChain.Batch(ctx, questions)
	// if err != nil {
	// 	fmt.Printf("批量执行失败: %v\n", err)
	// 	return
	// }

	// // 显示结果
	// for i, result := range results {
	// 	fmt.Printf("问题 %d: %s\n", i+1, result.Question)
	// 	fmt.Printf("答案: %s\n", result.Answer)
	// 	fmt.Printf("置信度: %.2f | 耗时: %v\n\n", result.Confidence, result.TimeElapsed)
	// }

	fmt.Println("示例代码 (需要实现 retrievers 包)")

	// Output:
	// 示例代码 (需要实现 retrievers 包)
}

// Example_customPrompt 自定义 Prompt 示例
func Example_customPrompt() {
	// import "langchain-go/core/prompts"
	// import "langchain-go/core/prompts/templates"

	// // 使用预定义模板
	// ragChain := chains.NewRAGChain(retriever, llm,
	// 	chains.WithPrompt(templates.DetailedRAGPrompt),
	// )

	// // 或者自定义模板
	// customPrompt := prompts.NewPromptTemplate(`
	// 你是一个专业的技术顾问。

	// 参考资料:
	// {{.context}}

	// 用户问题: {{.question}}

	// 请提供详细的技术解答，并给出实际应用建议:
	// `, []string{"context", "question"})

	// ragChain := chains.NewRAGChain(retriever, llm,
	// 	chains.WithPrompt(customPrompt),
	// )

	// result, _ := ragChain.Run(ctx, "如何设计一个 RAG 系统?")
	// fmt.Println(result.Answer)

	fmt.Println("示例代码 (需要实现 retrievers 包)")

	// Output:
	// 示例代码 (需要实现 retrievers 包)
}

// Example_advancedConfiguration 高级配置示例
func Example_advancedConfiguration() {
	// import "langchain-go/core/prompts/templates"

	// // 高级配置
	// ragChain := chains.NewRAGChain(retriever, llm,
	// 	// 设置相似度阈值，过滤低质量文档
	// 	chains.WithScoreThreshold(0.7),

	// 	// 限制上下文长度，避免超过 LLM 上下文窗口
	// 	chains.WithMaxContextLen(2000),

	// 	// 只返回 top 3 文档
	// 	chains.WithTopK(3),

	// 	// 返回来源文档
	// 	chains.WithReturnSources(true),

	// 	// 使用详细的 prompt 模板
	// 	chains.WithPrompt(templates.DetailedRAGPrompt),

	// 	// 自定义上下文格式化器
	// 	chains.WithContextFormatter(chains.SimpleContextFormatter),
	// )

	// result, err := ragChain.Run(ctx, "LangChain 的优势是什么?")
	// if err != nil {
	// 	fmt.Printf("查询失败: %v\n", err)
	// 	return
	// }

	// // 访问详细结果
	// fmt.Printf("置信度: %.2f\n", result.Confidence)
	// fmt.Printf("检索文档数: %d\n", len(result.Context))
	// fmt.Printf("上下文长度: %d\n", result.Metadata["context_length"])

	// // 查看来源
	// fmt.Println("\n来源:")
	// for _, doc := range result.Context {
	// 	source := doc.Metadata["source"].(string)
	// 	score := doc.Metadata["score"].(float32)
	// 	fmt.Printf("- %s (相关度: %.2f)\n", source, score)
	// }

	fmt.Println("示例代码 (需要实现 retrievers 包)")

	// Output:
	// 示例代码 (需要实现 retrievers 包)
}

// Example_errorHandling 错误处理示例
func Example_errorHandling() {
	// ctx := context.Background()

	// ragChain := chains.NewRAGChain(retriever, llm,
	// 	chains.WithScoreThreshold(0.9), // 高阈值
	// )

	// result, err := ragChain.Run(ctx, "一个不相关的问题")
	// if err != nil {
	// 	fmt.Printf("执行错误: %v\n", err)
	// 	return
	// }

	// // 检查置信度
	// if result.Confidence < 0.5 {
	// 	fmt.Println("⚠️ 警告: 低置信度回答")
	// 	fmt.Printf("置信度: %.2f\n", result.Confidence)
	// }

	// // 检查是否有文档
	// if len(result.Context) == 0 {
	// 	fmt.Println("⚠️ 警告: 未找到相关文档")
	// }

	// // 检查元数据
	// if metadata, ok := result.Metadata["filtered_docs"].(int); ok {
	// 	if metadata == 0 {
	// 		fmt.Println("所有文档都被过滤")
	// 	}
	// }

	// fmt.Println(result.Answer)

	fmt.Println("示例代码 (需要实现 retrievers 包)")

	// Output:
	// 示例代码 (需要实现 retrievers 包)
}
