// Package chat 提供 LangChain 聊天模型（ChatModel）的核心实现。
//
// chat 包实现了与各种 LLM 提供商（OpenAI、Anthropic、Ollama 等）交互的统一接口。
// 所有 ChatModel 都实现了 runnable.Runnable 接口，支持单次调用、批量调用和流式输出。
//
// 核心特性：
//   - 统一的 ChatModel 接口，支持多个 LLM 提供商
//   - 完整的工具调用（Function Calling）支持
//   - 流式响应处理
//   - 结构化输出（Structured Output）
//   - 重试和降级策略
//   - 可扩展的 Provider 架构
//
// 基本用法：
//
//	import (
//	    "context"
//	    "langchain-go/core/chat/providers/openai"
//	    "langchain-go/pkg/types"
//	)
//
//	// 创建 OpenAI 模型
//	model, err := openai.New(openai.Config{
//	    APIKey: "your-api-key",
//	    Model:  "gpt-4",
//	})
//	if err != nil {
//	    panic(err)
//	}
//
//	// 调用模型
//	messages := []types.Message{
//	    types.NewUserMessage("Hello, how are you?"),
//	}
//	response, err := model.Invoke(context.Background(), messages)
//	if err != nil {
//	    panic(err)
//	}
//	fmt.Println(response.Content)
//
// 工具调用示例：
//
//	// 定义工具
//	tool := types.Tool{
//	    Name:        "get_weather",
//	    Description: "Get current weather for a location",
//	    Parameters: types.Schema{
//	        Type: "object",
//	        Properties: map[string]types.Schema{
//	            "location": {Type: "string", Description: "City name"},
//	        },
//	        Required: []string{"location"},
//	    },
//	}
//
//	// 绑定工具到模型
//	modelWithTools := model.BindTools([]types.Tool{tool})
//
//	// 调用模型
//	response, err := modelWithTools.Invoke(ctx, messages)
//
// 流式输出示例：
//
//	stream, err := model.Stream(context.Background(), messages)
//	if err != nil {
//	    panic(err)
//	}
//
//	for event := range stream {
//	    switch event.Type {
//	    case runnable.EventStream:
//	        fmt.Print(event.Data.Content)
//	    case runnable.EventError:
//	        fmt.Println("Error:", event.Error)
//	    }
//	}
//
// 结构化输出示例：
//
//	schema := types.Schema{
//	    Type: "object",
//	    Properties: map[string]types.Schema{
//	        "name": {Type: "string"},
//	        "age":  {Type: "integer"},
//	    },
//	    Required: []string{"name", "age"},
//	}
//
//	modelWithSchema := model.WithStructuredOutput(schema)
//	response, err := modelWithSchema.Invoke(ctx, messages)
//
// 支持的提供商：
//   - OpenAI (GPT-3.5, GPT-4, GPT-4o 等)
//   - Anthropic (Claude 3 系列)
//   - Ollama (本地模型)
//
// 提供商特定功能：
//   - OpenAI: 支持 Vision、JSON Mode、Function Calling
//   - Anthropic: 支持 Computer Use、Vision、Tool Use
//   - Ollama: 支持本地运行开源模型
//
package chat
