package chains_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"langchain-go/core/chat/ollama"
	"langchain-go/retrieval/chains"
	"langchain-go/retrieval/embeddings"
	"langchain-go/retrieval/loaders"
	"langchain-go/retrieval/vectorstores"
)

// MockRetriever 模拟检索器用于测试
type MockRetriever struct {
	docs []*loaders.Document
}

func (m *MockRetriever) GetRelevantDocuments(ctx context.Context, query string) ([]*loaders.Document, error) {
	// 返回模拟文档
	return m.docs, nil
}

func NewMockRetriever(docs []*loaders.Document) *MockRetriever {
	return &MockRetriever{docs: docs}
}

// TestRAGChain_Basic 测试基本 RAG 功能
func TestRAGChain_Basic(t *testing.T) {
	// 创建模拟文档
	docs := []*loaders.Document{
		{
			Content: "LangChain 是一个用于构建 LLM 应用的框架。",
			Metadata: map[string]interface{}{
				"source": "doc1.txt",
				"score":  float32(0.95),
			},
		},
		{
			Content: "LangChain 提供了 RAG、Agent 等高层抽象。",
			Metadata: map[string]interface{}{
				"source": "doc2.txt",
				"score":  float32(0.85),
			},
		},
	}

	// 创建模拟检索器
	retriever := NewMockRetriever(docs)

	// 创建 LLM (需要实际运行的 Ollama)
	llm := ollama.NewChatOllama("qwen2.5:7b")

	// 创建 RAG Chain
	ragChain := chains.NewRAGChain(retriever, llm)

	// 执行查询
	result, err := ragChain.Run(context.Background(), "什么是 LangChain?")

	// 验证结果
	if err != nil {
		t.Logf("Warning: RAG execution failed (需要运行 Ollama): %v", err)
		return
	}

	if result.Question == "" {
		t.Error("Question should not be empty")
	}

	if result.Answer == "" {
		t.Error("Answer should not be empty")
	}

	if len(result.Context) != 2 {
		t.Errorf("Expected 2 context documents, got %d", len(result.Context))
	}

	if result.Confidence <= 0 {
		t.Error("Confidence should be greater than 0")
	}

	t.Logf("Question: %s", result.Question)
	t.Logf("Answer: %s", result.Answer)
	t.Logf("Confidence: %.2f", result.Confidence)
	t.Logf("Time: %v", result.TimeElapsed)
}

// TestRAGChain_WithScoreThreshold 测试分数过滤
func TestRAGChain_WithScoreThreshold(t *testing.T) {
	// 创建不同分数的文档
	docs := []*loaders.Document{
		{
			Content: "高分文档",
			Metadata: map[string]interface{}{
				"score": float32(0.9),
			},
		},
		{
			Content: "低分文档",
			Metadata: map[string]interface{}{
				"score": float32(0.3),
			},
		},
	}

	retriever := NewMockRetriever(docs)
	llm := ollama.NewChatOllama("qwen2.5:7b")

	// 设置阈值为 0.7
	ragChain := chains.NewRAGChain(retriever, llm,
		chains.WithScoreThreshold(0.7),
	)

	result, err := ragChain.Run(context.Background(), "测试问题")

	if err != nil {
		t.Logf("Warning: RAG execution failed (需要运行 Ollama): %v", err)
		return
	}

	// 应该只有 1 个文档通过过滤
	if len(result.Context) != 1 {
		t.Errorf("Expected 1 document after filtering, got %d", len(result.Context))
	}

	if result.Context[0].Content != "高分文档" {
		t.Error("Wrong document after filtering")
	}
}

// TestRAGChain_EmptyDocuments 测试空文档处理
func TestRAGChain_EmptyDocuments(t *testing.T) {
	retriever := NewMockRetriever([]*loaders.Document{})
	llm := ollama.NewChatOllama("qwen2.5:7b")
	ragChain := chains.NewRAGChain(retriever, llm)

	result, err := ragChain.Run(context.Background(), "测试问题")

	if err != nil {
		t.Fatalf("Should not error on empty documents: %v", err)
	}

	if !strings.Contains(result.Answer, "没有找到") {
		t.Error("Should indicate no documents found")
	}

	if result.Confidence != 0.0 {
		t.Error("Confidence should be 0 for empty documents")
	}
}

// TestRAGChain_Batch 测试批量处理
func TestRAGChain_Batch(t *testing.T) {
	docs := []*loaders.Document{
		{
			Content: "LangChain 是一个框架。",
			Metadata: map[string]interface{}{
				"score": float32(0.9),
			},
		},
	}

	retriever := NewMockRetriever(docs)
	llm := ollama.NewChatOllama("qwen2.5:7b")
	ragChain := chains.NewRAGChain(retriever, llm)

	questions := []string{
		"什么是 LangChain?",
		"LangChain 的作用是什么?",
	}

	results, err := ragChain.Batch(context.Background(), questions)

	if err != nil {
		t.Logf("Warning: Batch execution failed (需要运行 Ollama): %v", err)
		return
	}

	if len(results) != len(questions) {
		t.Errorf("Expected %d results, got %d", len(questions), len(results))
	}

	for i, result := range results {
		if result.Question != questions[i] {
			t.Errorf("Question mismatch at index %d", i)
		}
		if result.Answer == "" {
			t.Errorf("Empty answer at index %d", i)
		}
	}
}

// TestRAGChain_Stream 测试流式处理
func TestRAGChain_Stream(t *testing.T) {
	docs := []*loaders.Document{
		{
			Content: "LangChain 测试文档。",
			Metadata: map[string]interface{}{
				"score": float32(0.9),
			},
		},
	}

	retriever := NewMockRetriever(docs)
	llm := ollama.NewChatOllama("qwen2.5:7b")
	ragChain := chains.NewRAGChain(retriever, llm)

	stream, err := ragChain.Stream(context.Background(), "什么是 LangChain?")
	if err != nil {
		t.Fatalf("Failed to create stream: %v", err)
	}

	var hasRetrieval, hasToken, hasDone bool
	var fullAnswer strings.Builder

	for chunk := range stream {
		switch chunk.Type {
		case "start":
			t.Log("Stream started")
		case "retrieval":
			hasRetrieval = true
			t.Log("Retrieval completed")
		case "llm_token":
			hasToken = true
			fullAnswer.WriteString(chunk.Data.(string))
		case "done":
			hasDone = true
			t.Log("Stream done")
		case "error":
			t.Logf("Warning: Stream error (需要运行 Ollama): %v", chunk.Data)
			return
		}
	}

	if !hasRetrieval {
		t.Error("Should have retrieval event")
	}

	// 如果有 token，说明 LLM 正常工作
	if hasToken && !hasDone {
		t.Error("Should have done event")
	}

	t.Logf("Full answer: %s", fullAnswer.String())
}

// TestContextFormatters 测试不同的上下文格式化器
func TestContextFormatters(t *testing.T) {
	docs := []*loaders.Document{
		{
			Content: "文档1内容",
			Metadata: map[string]interface{}{
				"source": "doc1.txt",
				"score":  float32(0.9),
			},
		},
		{
			Content: "文档2内容",
			Metadata: map[string]interface{}{
				"source": "doc2.txt",
				"score":  float32(0.8),
			},
		},
	}

	// 测试默认格式化器
	t.Run("DefaultFormatter", func(t *testing.T) {
		result := chains.DefaultContextFormatter(docs)
		if !strings.Contains(result, "[文档 1]") {
			t.Error("Should contain document numbering")
		}
		if !strings.Contains(result, "来源: doc1.txt") {
			t.Error("Should contain source")
		}
		if !strings.Contains(result, "相关度:") {
			t.Error("Should contain score")
		}
	})

	// 测试简洁格式化器
	t.Run("SimpleFormatter", func(t *testing.T) {
		result := chains.SimpleContextFormatter(docs)
		if !strings.Contains(result, "文档1内容") {
			t.Error("Should contain document content")
		}
		if strings.Contains(result, "来源:") {
			t.Error("Should not contain metadata")
		}
	})

	// 测试结构化格式化器
	t.Run("StructuredFormatter", func(t *testing.T) {
		result := chains.StructuredContextFormatter(docs)
		if !strings.HasPrefix(result, "[") {
			t.Error("Should be JSON array")
		}
		if !strings.Contains(result, `"content"`) {
			t.Error("Should contain JSON keys")
		}
	})
}

// BenchmarkRAGChain_Run 性能测试
func BenchmarkRAGChain_Run(b *testing.B) {
	docs := []*loaders.Document{
		{
			Content: "性能测试文档",
			Metadata: map[string]interface{}{
				"score": float32(0.9),
			},
		},
	}

	retriever := NewMockRetriever(docs)
	llm := ollama.NewChatOllama("qwen2.5:7b")
	ragChain := chains.NewRAGChain(retriever, llm)

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ragChain.Run(ctx, "测试问题")
		if err != nil {
			b.Logf("Benchmark skipped (需要运行 Ollama): %v", err)
			b.SkipNow()
		}
	}
}

// ExampleRAGChain_basic 基础使用示例
func ExampleRAGChain_basic() {
	// 1. 准备向量存储
	embedder := embeddings.NewOllamaEmbeddings("nomic-embed-text")
	vectorStore := vectorstores.NewInMemoryVectorStore(embedder)

	// 2. 添加文档
	docs := []*loaders.Document{
		{Content: "LangChain 是一个用于构建 LLM 应用的框架。"},
		{Content: "RAG 结合了检索和生成两个步骤。"},
	}
	vectorStore.AddDocuments(context.Background(), docs)

	// 3. 创建 RAG Chain (核心只需 3 行!)
	// retriever := retrievers.NewVectorStoreRetriever(vectorStore)
	llm := ollama.NewChatOllama("qwen2.5:7b")
	// ragChain := chains.NewRAGChain(retriever, llm)

	// 4. 执行查询
	// result, _ := ragChain.Run(context.Background(), "什么是 RAG?")
	// fmt.Println("答案:", result.Answer)

	fmt.Println("示例代码 (需要 retrievers 包)")
	_ = llm
}

// ExampleRAGChain_withOptions 带选项的示例
func ExampleRAGChain_withOptions() {
	// 带配置选项的 RAG Chain
	// retriever := retrievers.NewVectorStoreRetriever(vectorStore)
	// llm := ollama.NewChatOllama("qwen2.5:7b")

	// ragChain := chains.NewRAGChain(retriever, llm,
	// 	chains.WithScoreThreshold(0.7),    // 设置相似度阈值
	// 	chains.WithMaxContextLen(2000),    // 限制上下文长度
	// 	chains.WithTopK(3),                // 返回 top 3 文档
	// 	chains.WithReturnSources(true),    // 返回来源文档
	// )

	// result, _ := ragChain.Run(context.Background(), "问题")
	// fmt.Printf("置信度: %.2f\n", result.Confidence)
	// fmt.Printf("来源数量: %d\n", len(result.Context))

	fmt.Println("示例代码 (需要 retrievers 包)")
}

// ExampleRAGChain_streaming 流式输出示例
func ExampleRAGChain_streaming() {
	// retriever := retrievers.NewVectorStoreRetriever(vectorStore)
	// llm := ollama.NewChatOllama("qwen2.5:7b")
	// ragChain := chains.NewRAGChain(retriever, llm)

	// // 流式输出
	// stream, _ := ragChain.Stream(context.Background(), "什么是 LangChain?")

	// for chunk := range stream {
	// 	switch chunk.Type {
	// 	case "retrieval":
	// 		fmt.Println("✓ 检索完成")
	// 	case "llm_token":
	// 		fmt.Print(chunk.Data) // 实时打印 token
	// 	case "done":
	// 		fmt.Println("\n✓ 完成")
	// 	}
	// }

	fmt.Println("示例代码 (需要 retrievers 包)")
}
