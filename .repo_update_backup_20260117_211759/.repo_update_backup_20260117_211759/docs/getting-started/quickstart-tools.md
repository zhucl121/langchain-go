# LangChain-Go 快速开始 - Tools 系统

本指南帮助你快速上手 LangChain-Go 的 Tools 系统。

## 什么是 Tools？

Tools（工具）让 AI Agent 能够与外部世界交互：
- 执行计算
- 访问 API
- 查询数据
- 调用函数

## 5 分钟快速上手

### 1. 使用内置工具 - 计算器

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/zhucl121/langchain-go/core/tools"
)

func main() {
	// 创建计算器工具
	calc := tools.NewCalculatorTool()

	// 执行计算
	result, err := calc.Execute(context.Background(), map[string]any{
		"expression": "2 + 3 * 4",
	})

	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Result: %v\n", result) // 14
}
```

### 2. 创建自定义工具

```go
package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/zhucl121/langchain-go/core/tools"
	"github.com/zhucl121/langchain-go/pkg/types"
)

func main() {
	// 创建自定义工具
	uppercaseTool := tools.NewFunctionTool(tools.FunctionToolConfig{
		Name:        "uppercase",
		Description: "Convert text to uppercase",
		Parameters: types.Schema{
			Type: "object",
			Properties: map[string]types.Schema{
				"text": {Type: "string", Description: "Text to convert"},
			},
			Required: []string{"text"},
		},
		Fn: func(ctx context.Context, args map[string]any) (any, error) {
			text := args["text"].(string)
			return strings.ToUpper(text), nil
		},
	})

	// 使用工具
	result, _ := uppercaseTool.Execute(context.Background(), map[string]any{
		"text": "hello world",
	})

	fmt.Println(result) // HELLO WORLD
}
```

### 3. 工具执行器 - 管理多个工具

```go
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/zhucl121/langchain-go/core/tools"
)

func main() {
	// 创建多个工具
	calc := tools.NewCalculatorTool()
	http := tools.NewHTTPRequestTool(tools.HTTPRequestConfig{
		Timeout: 10 * time.Second,
	})

	// 创建执行器
	executor := tools.NewToolExecutor(tools.ToolExecutorConfig{
		Tools:   []tools.Tool{calc, http},
		Timeout: 30 * time.Second,
	})

	// 执行工具
	result, _ := executor.Execute(context.Background(), "calculator", map[string]any{
		"expression": "100 * 50",
	})

	fmt.Printf("Calculator result: %v\n", result)
}
```

### 4. 与 ChatModel 集成

```go
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/zhucl121/langchain-go/core/chat/providers/openai"
	"github.com/zhucl121/langchain-go/core/tools"
	"github.com/zhucl121/langchain-go/pkg/types"
)

func main() {
	// 1. 创建工具
	calc := tools.NewCalculatorTool()
	executor := tools.NewToolExecutor(tools.ToolExecutorConfig{
		Tools: []tools.Tool{calc},
	})

	// 2. 创建模型并绑定工具
	model, _ := openai.New(openai.Config{
		APIKey: os.Getenv("OPENAI_API_KEY"),
		Model:  "gpt-4",
	})
	modelWithTools := model.BindTools(executor.GetTypesTools())

	// 3. 发送请求
	messages := []types.Message{
		types.NewUserMessage("What is 234 * 567?"),
	}

	response, _ := modelWithTools.Invoke(context.Background(), messages)

	// 4. 执行工具调用
	if len(response.ToolCalls) > 0 {
		for _, toolCall := range response.ToolCalls {
			result, _ := executor.ExecuteToolCall(context.Background(), toolCall)
			fmt.Printf("%s: %v\n", toolCall.Function.Name, result)
		}
	}
}
```

## 常用内置工具

### Calculator Tool
```go
calc := tools.NewCalculatorTool()
result, _ := calc.Execute(ctx, map[string]any{
    "expression": "2^8 + 10",
})
```

### HTTP Request Tool
```go
httpTool := tools.NewHTTPRequestTool(tools.HTTPRequestConfig{
    Timeout:        10 * time.Second,
    AllowedMethods: []string{"GET", "POST"},
})

result, _ := httpTool.Execute(ctx, map[string]any{
    "url":    "https://api.example.com/data",
    "method": "GET",
})
```

### JSONPlaceholder Tool (测试用)
```go
jsonTool := tools.NewJSONPlaceholderTool()
result, _ := jsonTool.Execute(ctx, map[string]any{
    "resource": "posts",
    "id":       1.0,
})
```

## 提示

1. **清晰的描述**: 工具描述要准确，帮助 LLM 理解何时使用
2. **参数验证**: 始终验证输入参数
3. **错误处理**: 提供有意义的错误信息
4. **超时控制**: 为长时间运行的工具设置超时
5. **安全考虑**: 限制允许的操作和访问范围

## 下一步

- 📖 查看 [完整示例文档](./tools-examples.md)
- 🔗 学习 [ChatModel 集成](./chat-examples.md)
- 🤖 了解 [Agent 模式](./agent-examples.md)

祝你使用愉快！🎉
