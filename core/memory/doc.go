// Package memory 提供 LangChain 记忆（Memory）系统的实现。
//
// memory 包实现了对话历史管理和上下文记忆功能。
// Memory 让 AI Agent 能够记住之前的对话内容，保持上下文连贯性。
//
// 核心特性：
//   - 通用的 Memory 接口
//   - BufferMemory（完整对话历史）
//   - ConversationBufferWindowMemory（滑动窗口）
//   - ConversationSummaryMemory（摘要记忆）
//   - 灵活的消息过滤和格式化
//   - 与 ChatModel 无缝集成
//
// 基本用法：
//
//	import (
//	    "context"
//	    "github.com/zhucl121/langchain-go/core/memory"
//	    "github.com/zhucl121/langchain-go/pkg/types"
//	)
//
//	// 创建缓冲记忆
//	mem := memory.NewBufferMemory()
//
//	// 保存对话
//	mem.SaveContext(context.Background(), map[string]any{
//	    "input": "Hello",
//	    "output": "Hi there!",
//	})
//
//	// 加载记忆
//	memVars, _ := mem.LoadMemoryVariables(context.Background(), map[string]any{})
//	history := memVars["history"].([]types.Message)
//
// 使用滑动窗口：
//
//	// 只保留最近的 K 轮对话
//	mem := memory.NewConversationBufferWindowMemory(memory.WindowMemoryConfig{
//	    K: 5, // 保留最近 5 轮对话
//	})
//
//	// 使用方式相同
//	mem.SaveContext(ctx, map[string]any{
//	    "input": "What's the weather?",
//	    "output": "It's sunny.",
//	})
//
// 与 ChatModel 集成：
//
//	// 创建记忆
//	mem := memory.NewBufferMemory()
//
//	// 创建模型
//	model, _ := openai.New(openai.Config{...})
//
//	// 对话循环
//	for {
//	    userInput := getUserInput()
//
//	    // 加载历史记忆
//	    memVars, _ := mem.LoadMemoryVariables(ctx, nil)
//	    history := memVars["history"].([]types.Message)
//
//	    // 添加新的用户消息
//	    messages := append(history, types.NewUserMessage(userInput))
//
//	    // 调用模型
//	    response, _ := model.Invoke(ctx, messages)
//
//	    // 保存对话到记忆
//	    mem.SaveContext(ctx, map[string]any{
//	        "input": userInput,
//	        "output": response.Content,
//	    })
//
//	    fmt.Println("AI:", response.Content)
//	}
//
// 自定义消息格式：
//
//	mem := memory.NewBufferMemory()
//
//	// 自定义输入/输出键名
//	mem.SetInputKey("user_message")
//	mem.SetOutputKey("ai_response")
//
//	// 自定义返回键名
//	mem.SetMemoryKey("chat_history")
//
// 清空记忆：
//
//	mem.Clear(context.Background())
//
// 摘要记忆（需要 LLM）：
//
//	// 当对话历史过长时，使用 LLM 生成摘要
//	summaryMem := memory.NewConversationSummaryMemory(memory.SummaryMemoryConfig{
//	    LLM:        model,
//	    MaxTokens:  2000, // 超过此限制时触发摘要
//	})
//
//	// 使用方式与 BufferMemory 相同
//	summaryMem.SaveContext(ctx, inputs)
//
package memory
