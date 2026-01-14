// Package anthropic 提供 Anthropic (Claude) ChatModel 的实现。
//
// 支持特性：
//   - Claude 3 系列模型（Opus, Sonnet, Haiku）
//   - 完整的 Tool Use 支持
//   - 流式响应（SSE）
//   - Vision 能力
//   - Computer Use（部分模型支持）
//   - 长上下文（最高 200K tokens）
//
// 基本用法：
//
//	import "langchain-go/core/chat/providers/anthropic"
//
//	model, err := anthropic.New(anthropic.Config{
//	    APIKey: "sk-ant-...",
//	    Model:  "claude-3-opus-20240229",
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
package anthropic
