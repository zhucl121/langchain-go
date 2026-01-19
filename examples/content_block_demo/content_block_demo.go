package main

import (
	"fmt"
	"log"

	"github.com/zhucl121/langchain-go/pkg/types"
)

func main() {
	fmt.Println("=== LangChain-Go 标准内容块示例 ===\n")

	// 示例 1: 基础文本内容块
	basicExample()

	// 示例 2: 带推理过程的内容块
	reasoningExample()

	// 示例 3: 带引用来源的内容块（RAG 场景）
	citationExample()

	// 示例 4: 完整 RAG 场景
	fullRAGExample()

	// 示例 5: 错误处理
	errorExample()

	// 示例 6: 内容块列表
	listExample()
}

// basicExample 基础文本内容块示例
func basicExample() {
	fmt.Println("1. 基础文本内容块")
	fmt.Println("---")

	block := types.NewTextContentBlock("这是一个简单的文本响应").
		WithID("block_1").
		WithConfidence(0.95).
		WithMetadata("model", "gpt-4").
		WithMetadata("tokens", 120)

	fmt.Printf("内容块类型: %s\n", block.Type)
	fmt.Printf("内容: %s\n", block.Content)
	fmt.Printf("置信度: %.2f\n", *block.Confidence)
	fmt.Printf("元数据: %v\n\n", block.Metadata)
}

// reasoningExample 带推理过程的示例
func reasoningExample() {
	fmt.Println("2. 带推理过程的内容块")
	fmt.Println("---")

	block := types.NewTextContentBlock("答案是 42").
		WithReasoning([]string{
			"步骤1: 分析问题 - 什么是宇宙的终极答案？",
			"步骤2: 查阅《银河系漫游指南》",
			"步骤3: 深度思考号计算了 750 万年",
			"步骤4: 得出结论 - 答案是 42",
		}).
		WithConfidence(1.0).
		WithMetadata("source", "Deep Thought Computer")

	fmt.Printf("内容: %s\n", block.Content)
	fmt.Println("推理过程:")
	for i, step := range block.Reasoning {
		fmt.Printf("  %d. %s\n", i+1, step)
	}
	fmt.Println()
}

// citationExample 带引用来源的示例（RAG）
func citationExample() {
	fmt.Println("3. 带引用来源的内容块 (RAG)")
	fmt.Println("---")

	page1 := 15
	page2 := 42

	block := types.NewTextContentBlock("机器学习是人工智能的一个重要分支，它使计算机能够从数据中学习而无需显式编程。").
		AddCitation(types.Citation{
			Source:  "ml_textbook.pdf",
			Excerpt: "机器学习使计算机能够从经验中学习...",
			Score:   0.95,
			Page:    &page1,
			Title:   "机器学习导论",
		}).
		AddCitation(types.Citation{
			Source:  "ai_handbook.pdf",
			Excerpt: "AI 的核心是让机器具有学习能力...",
			Score:   0.88,
			Page:    &page2,
			Title:   "人工智能手册",
			URL:     "https://example.com/ai-handbook",
		}).
		WithConfidence(0.92)

	fmt.Printf("回答: %s\n\n", block.Content)
	fmt.Println("引用来源:")
	for i, citation := range block.Citations {
		fmt.Printf("%d. %s", i+1, citation.Source)
		if citation.Page != nil {
			fmt.Printf(" (第 %d 页)", *citation.Page)
		}
		fmt.Printf(" - 相似度: %.2f\n", citation.Score)
		fmt.Printf("   片段: %s\n", citation.Excerpt)
	}
	fmt.Println()
}

// fullRAGExample 完整 RAG 场景示例
func fullRAGExample() {
	fmt.Println("4. 完整 RAG 场景")
	fmt.Println("---")

	// 创建内容块列表
	list := types.NewContentBlockList()

	// 1. 思考过程块
	thinkingBlock := types.NewThinkingContentBlock("用户问了一个关于 Go 语言的问题，我需要搜索相关文档").
		WithID("thinking_1").
		WithMetadata("model", "gpt-4")
	list.Add(thinkingBlock)

	// 2. 工具调用块
	toolUseBlock := types.NewToolUseContentBlock([]types.ToolCall{
		{
			ID:   "call_search_1",
			Type: "function",
			Function: types.FunctionCall{
				Name:      "vector_search",
				Arguments: `{"query": "Go语言并发特性", "k": 3}`,
			},
		},
	}).WithID("tool_use_1")
	list.Add(toolUseBlock)

	// 3. 工具结果块
	toolResultBlock := types.NewToolResultContentBlock("找到 3 个相关文档").
		WithID("tool_result_1").
		WithParentID("tool_use_1")
	list.Add(toolResultBlock)

	// 4. 最终答案块（带推理和引用）
	page := 127
	answerBlock := types.NewTextContentBlock("Go 语言的并发特性主要基于 goroutine 和 channel。Goroutine 是轻量级线程，而 channel 用于 goroutine 之间的通信。").
		WithID("answer_1").
		WithReasoning([]string{
			"分析用户问题：Go 语言的并发特性",
			"执行向量搜索查找相关文档",
			"综合 3 个文档的信息",
			"生成准确且有引用的答案",
		}).
		AddCitation(types.Citation{
			Source:  "go_concurrency.pdf",
			Excerpt: "Goroutine 是 Go 的轻量级线程...",
			Score:   0.96,
			Page:    &page,
			Title:   "Go 并发编程",
		}).
		WithConfidence(0.94).
		WithMetadata("tokens", 185).
		WithMetadata("latency_ms", 1450)
	list.Add(answerBlock)

	// 输出整个流程
	fmt.Printf("总共生成了 %d 个内容块\n\n", len(list.Blocks))

	// 提取最终文本内容
	textContent := list.GetTextContent()
	fmt.Printf("最终文本输出:\n%s\n\n", textContent)

	// 提取所有引用
	citations := list.GetAllCitations()
	fmt.Printf("引用来源数量: %d\n", len(citations))
	for i, citation := range citations {
		fmt.Printf("%d. %s (相似度: %.2f)\n", i+1, citation.Title, citation.Score)
	}

	// 序列化为 JSON
	jsonStr, err := list.ToJSON()
	if err != nil {
		log.Fatalf("序列化失败: %v", err)
	}
	fmt.Printf("\nJSON 输出 (%d 字节):\n", len(jsonStr))
	// 只显示前 200 个字符
	if len(jsonStr) > 200 {
		fmt.Printf("%s...\n\n", jsonStr[:200])
	} else {
		fmt.Printf("%s\n\n", jsonStr)
	}
}

// errorExample 错误处理示例
func errorExample() {
	fmt.Println("5. 错误处理")
	fmt.Println("---")

	errorBlock := types.NewErrorContentBlock("RATE_LIMIT_EXCEEDED", "API 调用频率超限").
		WithMetadata("retry_after", 60).
		WithMetadata("quota_remaining", 0)

	// 设置错误详情
	errorBlock.Error.Details = map[string]any{
		"current_rate":  150,
		"max_rate":      100,
		"reset_time":    "2026-01-20T00:30:00Z",
	}
	errorBlock.Error.Recoverable = true

	fmt.Printf("错误类型: %s\n", errorBlock.Type)
	fmt.Printf("错误码: %s\n", errorBlock.Error.Code)
	fmt.Printf("错误消息: %s\n", errorBlock.Error.Message)
	fmt.Printf("可恢复: %v\n", errorBlock.Error.Recoverable)
	fmt.Printf("详情: %v\n\n", errorBlock.Error.Details)
}

// listExample 内容块列表操作示例
func listExample() {
	fmt.Println("6. 内容块列表操作")
	fmt.Println("---")

	list := types.NewContentBlockList()

	// 添加多个不同类型的块
	list.Add(types.NewTextContentBlock("第一段内容").WithID("text_1"))
	list.Add(types.NewThinkingContentBlock("思考中...").WithID("think_1"))
	list.Add(types.NewTextContentBlock("第二段内容").WithID("text_2"))
	list.Add(types.NewErrorContentBlock("TIMEOUT", "请求超时").WithID("error_1"))

	// 按类型过滤
	textBlocks := list.GetByType(types.ContentBlockText)
	fmt.Printf("文本块数量: %d\n", len(textBlocks))

	thinkingBlocks := list.GetByType(types.ContentBlockThinking)
	fmt.Printf("思考块数量: %d\n", len(thinkingBlocks))

	errorBlocks := list.GetByType(types.ContentBlockError)
	fmt.Printf("错误块数量: %d\n", len(errorBlocks))

	// 按 ID 查找
	block := list.GetByID("text_1")
	if block != nil {
		fmt.Printf("\n找到块 'text_1': %s\n", block.Content)
	}

	// 获取文本内容
	textContent := list.GetTextContent()
	fmt.Printf("\n拼接的文本内容:\n%s\n", textContent)
}
