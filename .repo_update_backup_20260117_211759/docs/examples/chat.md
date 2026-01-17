# ChatModel 使用示例

本文档展示如何使用 `langchain-go` 的 ChatModel 系统。

## 基础用法

### OpenAI

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/zhucl121/langchain-go/core/chat/providers/openai"
	"github.com/zhucl121/langchain-go/pkg/types"
)

func main() {
	// 创建 OpenAI 模型
	model, err := openai.New(openai.Config{
		APIKey: "sk-...", // 你的 OpenAI API Key
		Model:  "gpt-4",
	})
	if err != nil {
		log.Fatal(err)
	}

	// 准备消息
	messages := []types.Message{
		types.NewSystemMessage("You are a helpful assistant."),
		types.NewUserMessage("Hello! What's 2+2?"),
	}

	// 调用模型
	ctx := context.Background()
	response, err := model.Invoke(ctx, messages)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Assistant:", response.Content)
}
```

### Anthropic (Claude)

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/zhucl121/langchain-go/core/chat/providers/anthropic"
	"github.com/zhucl121/langchain-go/pkg/types"
)

func main() {
	// 创建 Anthropic 模型
	model, err := anthropic.New(anthropic.Config{
		APIKey:    "sk-ant-...", // 你的 Anthropic API Key
		Model:     "claude-3-5-sonnet-20241022",
		MaxTokens: 1000,
	})
	if err != nil {
		log.Fatal(err)
	}

	// 准备消息
	messages := []types.Message{
		types.NewSystemMessage("You are a helpful assistant."),
		types.NewUserMessage("Hello! Tell me a short joke."),
	}

	// 调用模型
	ctx := context.Background()
	response, err := model.Invoke(ctx, messages)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Claude:", response.Content)
}
```

## 流式输出

流式输出可以逐步接收模型的响应，提升用户体验：

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/zhucl121/langchain-go/core/chat/providers/openai"
	"github.com/zhucl121/langchain-go/core/runnable"
	"github.com/zhucl121/langchain-go/pkg/types"
)

func main() {
	model, err := openai.New(openai.Config{
		APIKey: "sk-...",
		Model:  "gpt-4",
	})
	if err != nil {
		log.Fatal(err)
	}

	messages := []types.Message{
		types.NewUserMessage("Tell me a story about a robot."),
	}

	// 流式调用
	ctx := context.Background()
	stream, err := model.Stream(ctx, messages)
	if err != nil {
		log.Fatal(err)
	}

	// 处理流式事件
	for event := range stream {
		switch event.Type {
		case runnable.EventStart:
			fmt.Println("开始生成...")

		case runnable.EventStream:
			// 逐步打印内容
			fmt.Print(event.Data.Content)

		case runnable.EventEnd:
			fmt.Println("\n\n完成！")

		case runnable.EventError:
			log.Println("错误:", event.Error)
		}
	}
}
```

## 工具调用 (Function Calling)

工具调用允许模型请求执行外部函数：

```go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/zhucl121/langchain-go/core/chat/providers/openai"
	"github.com/zhucl121/langchain-go/pkg/types"
)

func main() {
	model, err := openai.New(openai.Config{
		APIKey: "sk-...",
		Model:  "gpt-4",
	})
	if err != nil {
		log.Fatal(err)
	}

	// 定义工具
	weatherTool := types.Tool{
		Name:        "get_weather",
		Description: "Get the current weather for a location",
		Parameters: types.Schema{
			Type: "object",
			Properties: map[string]types.Schema{
				"location": {
					Type:        "string",
					Description: "City name, e.g., 'San Francisco'",
				},
				"unit": {
					Type:        "string",
					Description: "Temperature unit (celsius or fahrenheit)",
					Enum:        []string{"celsius", "fahrenheit"},
				},
			},
			Required: []string{"location"},
		},
	}

	// 绑定工具到模型
	modelWithTools := model.BindTools([]types.Tool{weatherTool})

	// 发送请求
	messages := []types.Message{
		types.NewUserMessage("What's the weather like in Tokyo?"),
	}

	ctx := context.Background()
	response, err := modelWithTools.Invoke(ctx, messages)
	if err != nil {
		log.Fatal(err)
	}

	// 检查是否有工具调用
	if len(response.ToolCalls) > 0 {
		for _, tc := range response.ToolCalls {
			fmt.Printf("模型请求调用工具: %s\n", tc.Function.Name)
			
			// 解析参数
			var args map[string]any
			json.Unmarshal([]byte(tc.Function.Arguments), &args)
			fmt.Printf("参数: %+v\n", args)

			// 这里你应该实际执行工具并返回结果
			// 然后将结果作为新消息发送回模型
			toolResult := types.NewToolMessage(
				tc.ID,
				`{"temperature": 22, "condition": "sunny"}`,
			)

			// 继续对话
			messages = append(messages, response, toolResult)
			finalResponse, _ := modelWithTools.Invoke(ctx, messages)
			fmt.Println("最终回复:", finalResponse.Content)
		}
	} else {
		fmt.Println("回复:", response.Content)
	}
}
```

## 结构化输出

强制模型返回特定格式的 JSON：

```go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/zhucl121/langchain-go/core/chat/providers/openai"
	"github.com/zhucl121/langchain-go/pkg/types"
)

func main() {
	model, err := openai.New(openai.Config{
		APIKey: "sk-...",
		Model:  "gpt-4o", // 需要支持结构化输出的模型
	})
	if err != nil {
		log.Fatal(err)
	}

	// 定义输出 Schema
	schema := types.Schema{
		Type: "object",
		Properties: map[string]types.Schema{
			"name": {
				Type:        "string",
				Description: "Person's name",
			},
			"age": {
				Type:        "integer",
				Description: "Person's age",
			},
			"occupation": {
				Type:        "string",
				Description: "Person's job",
			},
		},
		Required: []string{"name", "age", "occupation"},
	}

	// 配置结构化输出
	modelWithSchema := model.WithStructuredOutput(schema)

	// 发送请求
	messages := []types.Message{
		types.NewUserMessage("Extract information: John Doe is a 30-year-old software engineer."),
	}

	ctx := context.Background()
	response, err := modelWithSchema.Invoke(ctx, messages)
	if err != nil {
		log.Fatal(err)
	}

	// 解析结构化响应
	var person map[string]any
	if err := json.Unmarshal([]byte(response.Content), &person); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("提取的信息:\n")
	fmt.Printf("  姓名: %s\n", person["name"])
	fmt.Printf("  年龄: %.0f\n", person["age"])
	fmt.Printf("  职业: %s\n", person["occupation"])
}
```

## 批量处理

并行处理多个对话：

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/zhucl121/langchain-go/core/chat/providers/openai"
	"github.com/zhucl121/langchain-go/pkg/types"
)

func main() {
	model, err := openai.New(openai.Config{
		APIKey: "sk-...",
		Model:  "gpt-3.5-turbo",
	})
	if err != nil {
		log.Fatal(err)
	}

	// 准备多组对话
	batchInputs := [][]types.Message{
		{types.NewUserMessage("What is 2+2?")},
		{types.NewUserMessage("What is the capital of France?")},
		{types.NewUserMessage("Who wrote Hamlet?")},
	}

	// 批量调用（自动并行）
	ctx := context.Background()
	responses, err := model.Batch(ctx, batchInputs)
	if err != nil {
		log.Fatal(err)
	}

	// 打印所有响应
	for i, response := range responses {
		fmt.Printf("问题 %d: %s\n", i+1, batchInputs[i][0].Content)
		fmt.Printf("回答 %d: %s\n\n", i+1, response.Content)
	}
}
```

## 配置选项

### OpenAI 配置

```go
config := openai.Config{
	APIKey:           "sk-...",
	BaseURL:          "https://api.openai.com/v1", // 可自定义（如使用代理）
	Model:            "gpt-4",
	Temperature:      0.7,  // 0.0-2.0，控制随机性
	MaxTokens:        4096, // 最大生成 token 数
	TopP:             0.9,  // 核采样参数
	FrequencyPenalty: 0.0,  // 频率惩罚 (-2.0 到 2.0)
	PresencePenalty:  0.0,  // 存在惩罚 (-2.0 到 2.0)
	Timeout:          60 * time.Second,
	User:             "user-123", // 用户标识
	Organization:     "org-...",  // 组织 ID
}
```

### Anthropic 配置

```go
config := anthropic.Config{
	APIKey:      "sk-ant-...",
	BaseURL:     "https://api.anthropic.com", // 可自定义
	Model:       "claude-3-5-sonnet-20241022",
	Temperature: 1.0,   // 0.0-1.0
	MaxTokens:   4096,  // 必需参数！
	TopP:        0.9,
	TopK:        50,
	Timeout:     60 * time.Second,
}
```

## 错误处理

```go
response, err := model.Invoke(ctx, messages)
if err != nil {
	// 处理各种错误
	switch {
	case strings.Contains(err.Error(), "HTTP 401"):
		log.Fatal("API Key 无效")
	case strings.Contains(err.Error(), "HTTP 429"):
		log.Fatal("请求频率过高，请稍后重试")
	case strings.Contains(err.Error(), "context deadline exceeded"):
		log.Fatal("请求超时")
	default:
		log.Fatalf("未知错误: %v", err)
	}
}
```

## 与 Runnable 系统集成

ChatModel 实现了 `Runnable` 接口，可以与其他 Runnable 组合：

```go
import (
	"github.com/zhucl121/langchain-go/core/runnable"
	"github.com/zhucl121/langchain-go/core/chat/providers/openai"
)

// 创建模型
model, _ := openai.New(openai.Config{APIKey: "sk-..."})

// 添加重试
modelWithRetry := model.WithRetry(types.RetryPolicy{
	MaxAttempts: 3,
	InitialDelay: time.Second,
})

// 添加降级方案
fallbackModel, _ := openai.New(openai.Config{
	APIKey: "sk-...",
	Model:  "gpt-3.5-turbo", // 更便宜的模型作为备选
})
modelWithFallback := model.WithFallbacks(fallbackModel)

// 组合使用
response, _ := modelWithRetry.Invoke(ctx, messages)
```

## 最佳实践

1. **API Key 管理**: 不要硬编码 API Key，使用环境变量：
   ```go
   apiKey := os.Getenv("OPENAI_API_KEY")
   ```

2. **错误处理**: 始终检查错误并适当处理

3. **超时控制**: 为长时间运行的请求设置合适的超时

4. **流式输出**: 对于长响应，使用流式输出提升体验

5. **批量处理**: 需要处理多个独立请求时，使用 `Batch` 方法提高效率

6. **工具调用**: 正确处理工具调用循环，避免无限递归

7. **成本控制**: 监控 token 使用，选择合适的模型

## 支持的模型

### OpenAI
- `gpt-4` - 最强大的模型
- `gpt-4-turbo` - 更快、更便宜的 GPT-4
- `gpt-4o` - 多模态模型
- `gpt-4o-mini` - 轻量级多模态模型
- `gpt-3.5-turbo` - 快速、经济的模型

### Anthropic
- `claude-3-opus-20240229` - 最强大的 Claude 模型
- `claude-3-sonnet-20240229` - 平衡性能和速度
- `claude-3-haiku-20240307` - 最快、最经济
- `claude-3-5-sonnet-20241022` - 最新的 Sonnet 版本
