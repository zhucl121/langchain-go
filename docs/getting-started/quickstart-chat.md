# LangChain-Go ChatModel 快速开始

本指南帮助你在 5 分钟内开始使用 LangChain-Go 的 ChatModel 系统。

## 安装

```bash
# 克隆项目
cd /Users/zhuchenglong/Documents/workspace/随笔/langchain-go

# 安装依赖
go mod download

# 运行测试验证
go test ./core/chat/...
```

## 第一个示例

### 1. OpenAI 基础对话

创建文件 `examples/openai_basic.go`:

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/zhuchenglong/langchain-go/core/chat/providers/openai"
	"github.com/zhuchenglong/langchain-go/pkg/types"
)

func main() {
	// 从环境变量获取 API Key
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("请设置 OPENAI_API_KEY 环境变量")
	}

	// 创建模型
	model, err := openai.New(openai.Config{
		APIKey: apiKey,
		Model:  "gpt-4o-mini", // 使用较便宜的模型
	})
	if err != nil {
		log.Fatal(err)
	}

	// 准备消息
	messages := []types.Message{
		types.NewSystemMessage("你是一个友好的 AI 助手。"),
		types.NewUserMessage("请用一句话介绍自己。"),
	}

	// 调用模型
	ctx := context.Background()
	response, err := model.Invoke(ctx, messages)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("AI:", response.Content)
}
```

运行：

```bash
export OPENAI_API_KEY="sk-..."
go run examples/openai_basic.go
```

### 2. Anthropic (Claude) 基础对话

创建文件 `examples/anthropic_basic.go`:

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/zhuchenglong/langchain-go/core/chat/providers/anthropic"
	"github.com/zhuchenglong/langchain-go/pkg/types"
)

func main() {
	// 从环境变量获取 API Key
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		log.Fatal("请设置 ANTHROPIC_API_KEY 环境变量")
	}

	// 创建模型
	model, err := anthropic.New(anthropic.Config{
		APIKey:    apiKey,
		Model:     "claude-3-haiku-20240307", // 使用最快的模型
		MaxTokens: 500,                       // Anthropic 需要指定最大 token 数
	})
	if err != nil {
		log.Fatal(err)
	}

	// 准备消息
	messages := []types.Message{
		types.NewSystemMessage("你是一个友好的 AI 助手。"),
		types.NewUserMessage("请写一个 Python 的 Hello World 程序。"),
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

运行：

```bash
export ANTHROPIC_API_KEY="sk-ant-..."
go run examples/anthropic_basic.go
```

### 3. 流式输出示例

创建文件 `examples/streaming.go`:

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/zhuchenglong/langchain-go/core/chat/providers/openai"
	"github.com/zhuchenglong/langchain-go/core/runnable"
	"github.com/zhuchenglong/langchain-go/pkg/types"
)

func main() {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("请设置 OPENAI_API_KEY 环境变量")
	}

	model, err := openai.New(openai.Config{
		APIKey: apiKey,
		Model:  "gpt-4o-mini",
	})
	if err != nil {
		log.Fatal(err)
	}

	messages := []types.Message{
		types.NewUserMessage("请用 100 字讲一个关于机器人的故事。"),
	}

	ctx := context.Background()
	stream, err := model.Stream(ctx, messages)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Print("AI: ")
	for event := range stream {
		switch event.Type {
		case runnable.EventStream:
			// 实时打印每个字符
			fmt.Print(event.Data.Content)
		case runnable.EventError:
			log.Fatal(event.Error)
		}
	}
	fmt.Println()
}
```

### 4. 多轮对话

创建文件 `examples/conversation.go`:

```go
package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/zhuchenglong/langchain-go/core/chat/providers/openai"
	"github.com/zhuchenglong/langchain-go/pkg/types"
)

func main() {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("请设置 OPENAI_API_KEY 环境变量")
	}

	model, err := openai.New(openai.Config{
		APIKey: apiKey,
		Model:  "gpt-4o-mini",
	})
	if err != nil {
		log.Fatal(err)
	}

	// 初始化对话历史
	messages := []types.Message{
		types.NewSystemMessage("你是一个友好的 AI 助手。请简洁回答。"),
	}

	scanner := bufio.NewScanner(os.Stdin)
	ctx := context.Background()

	fmt.Println("AI 助手已就绪！输入 'quit' 退出。")
	fmt.Println()

	for {
		fmt.Print("你: ")
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}
		if input == "quit" {
			break
		}

		// 添加用户消息
		messages = append(messages, types.NewUserMessage(input))

		// 调用模型
		response, err := model.Invoke(ctx, messages)
		if err != nil {
			log.Printf("错误: %v\n", err)
			continue
		}

		// 添加 AI 响应到历史
		messages = append(messages, response)

		fmt.Printf("AI: %s\n\n", response.Content)
	}

	fmt.Println("再见！")
}
```

运行：

```bash
go run examples/conversation.go
```

## 常见问题

### Q: 如何设置代理？

```go
config := openai.Config{
    APIKey:  "sk-...",
    BaseURL: "https://your-proxy.com/v1", // 自定义 API 地址
}
```

### Q: 如何控制输出长度？

```go
// OpenAI
config := openai.Config{
    APIKey:    "sk-...",
    MaxTokens: 100, // 限制最多生成 100 tokens
}

// Anthropic
config := anthropic.Config{
    APIKey:    "sk-ant-...",
    MaxTokens: 100, // 必需参数
}
```

### Q: 如何让输出更稳定（降低随机性）？

```go
config := openai.Config{
    APIKey:      "sk-...",
    Temperature: 0.0, // 0.0 = 最确定，2.0 = 最随机
}
```

### Q: 如何处理超时？

```go
import "time"

config := openai.Config{
    APIKey:  "sk-...",
    Timeout: 30 * time.Second, // 30 秒超时
}
```

### Q: 如何批量处理多个问题？

```go
inputs := [][]types.Message{
    {types.NewUserMessage("1+1=?")},
    {types.NewUserMessage("2+2=?")},
    {types.NewUserMessage("3+3=?")},
}

// 自动并行处理
responses, err := model.Batch(ctx, inputs)
```

## 下一步

- 📖 查看 [完整示例文档](./chat-examples.md)
- 🔧 了解 [工具调用 (Function Calling)](./chat-examples.md#工具调用-function-calling)
- 📊 学习 [结构化输出](./chat-examples.md#结构化输出)
- 🧪 查看 [测试用例](../core/chat/) 了解更多用法

## 帮助

- 🐛 [报告问题](https://github.com/your-repo/issues)
- 💬 [讨论区](https://github.com/your-repo/discussions)
- 📚 [API 文档](https://pkg.go.dev/langchain-go/core/chat)

## 提示

1. **保护 API Key**: 永远不要硬编码 API Key，使用环境变量
2. **错误处理**: 生产环境务必处理所有可能的错误
3. **成本控制**: 使用 MaxTokens 限制输出长度，避免超额消费
4. **选择合适的模型**: 简单任务使用便宜的模型，复杂任务才用高级模型

祝你使用愉快！🎉
