// Package openai 提供 OpenAI ChatModel 的实现。
//
// 支持特性：
//   - GPT-3.5, GPT-4, GPT-4o 等所有 OpenAI 模型
//   - 完整的 Function Calling 支持
//   - 流式响应（SSE）
//   - JSON Mode 和结构化输出
//   - Vision 能力（GPT-4 Vision）
//   - 自定义 BaseURL（支持代理和兼容 API）
//
// 基本用法：
//
//	import "github.com/zhucl121/langchain-go/core/chat/providers/openai"
//
//	model, err := openai.New(openai.Config{
//	    APIKey: "sk-...",
//	    Model:  "gpt-4",
//	})
//
//	messages := []types.Message{
//	    types.NewUserMessage("Hello!"),
//	}
//	response, err := model.Invoke(context.Background(), messages)
//
// 流式输出：
//
//	stream, err := model.Stream(ctx, messages)
//	for event := range stream {
//	    if event.Type == runnable.EventStream {
//	        fmt.Print(event.Data.Content)
//	    }
//	}
//
// 工具调用：
//
//	tool := types.Tool{
//	    Name:        "get_weather",
//	    Description: "Get weather for a location",
//	    Parameters:  weatherSchema,
//	}
//	modelWithTools := model.BindTools([]types.Tool{tool})
//	response, err := modelWithTools.Invoke(ctx, messages)
//
package openai
