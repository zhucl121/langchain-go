# Tools 系统使用指南

本指南详细介绍如何使用 LangChain-Go 的 Tools 系统。

## 目录

1. [基础概念](#基础概念)
2. [创建自定义工具](#创建自定义工具)
3. [内置工具](#内置工具)
4. [工具执行器](#工具执行器)
5. [与 ChatModel 集成](#与-chatmodel-集成)
6. [实战示例](#实战示例)
7. [最佳实践](#最佳实践)

---

## 基础概念

**Tool** 是 AI Agent 与外部世界交互的桥梁。工具允许 LLM：
- 执行计算（如计算器）
- 访问外部 API
- 查询数据库
- 操作文件系统
- 调用其他服务

在 LangChain-Go 中，Tool 的核心是 `Tool` 接口：

```go
type Tool interface {
    GetName() string
    GetDescription() string
    GetParameters() types.Schema
    Execute(ctx context.Context, args map[string]any) (any, error)
    ToTypesTool() types.Tool
}
```

---

## 创建自定义工具

### 基础方法：使用 FunctionTool

最简单的方法是使用 `FunctionTool`：

```go
package main

import (
    "context"
    "fmt"
    "strings"

    "github.com/zhuchenglong/langchain-go/core/tools"
    "github.com/zhuchenglong/langchain-go/pkg/types"
)

func main() {
    // 创建字符串转换工具
    uppercaseTool := tools.NewFunctionTool(tools.FunctionToolConfig{
        Name:        "uppercase",
        Description: "Convert text to uppercase",
        Parameters: types.Schema{
            Type: "object",
            Properties: map[string]types.Schema{
                "text": {
                    Type:        "string",
                    Description: "Text to convert",
                },
            },
            Required: []string{"text"},
        },
        Fn: func(ctx context.Context, args map[string]any) (any, error) {
            text := args["text"].(string)
            return strings.ToUpper(text), nil
        },
    })

    // 执行工具
    result, err := uppercaseTool.Execute(context.Background(), map[string]any{
        "text": "hello world",
    })

    if err != nil {
        panic(err)
    }

    fmt.Println(result) // "HELLO WORLD"
}
```

### 高级方法：实现 Tool 接口

对于复杂工具，可以实现自定义类型：

```go
package main

import (
    "context"
    "database/sql"
    "fmt"

    "github.com/zhuchenglong/langchain-go/core/tools"
    "github.com/zhuchenglong/langchain-go/pkg/types"
)

// DatabaseTool 数据库查询工具
type DatabaseTool struct {
    db *sql.DB
}

func NewDatabaseTool(db *sql.DB) *DatabaseTool {
    return &DatabaseTool{db: db}
}

func (t *DatabaseTool) GetName() string {
    return "database_query"
}

func (t *DatabaseTool) GetDescription() string {
    return "Execute SQL queries on the database"
}

func (t *DatabaseTool) GetParameters() types.Schema {
    return types.Schema{
        Type: "object",
        Properties: map[string]types.Schema{
            "query": {
                Type:        "string",
                Description: "SQL query to execute",
            },
        },
        Required: []string{"query"},
    }
}

func (t *DatabaseTool) Execute(ctx context.Context, args map[string]any) (any, error) {
    query := args["query"].(string)

    rows, err := t.db.QueryContext(ctx, query)
    if err != nil {
        return nil, fmt.Errorf("query failed: %w", err)
    }
    defer rows.Close()

    // 解析结果
    var results []map[string]any
    // ... 实现查询结果解析

    return results, nil
}

func (t *DatabaseTool) ToTypesTool() types.Tool {
    return types.Tool{
        Name:        t.GetName(),
        Description: t.GetDescription(),
        Parameters:  t.GetParameters(),
    }
}
```

---

## 内置工具

### 1. Calculator Tool - 计算器

```go
package main

import (
    "context"
    "fmt"

    "github.com/zhuchenglong/langchain-go/core/tools"
)

func main() {
    // 创建计算器工具
    calc := tools.NewCalculatorTool()

    // 基本运算
    result, _ := calc.Execute(context.Background(), map[string]any{
        "expression": "2 + 3 * 4",
    })
    fmt.Println(result) // 14

    // 复杂表达式
    result, _ = calc.Execute(context.Background(), map[string]any{
        "expression": "(10 + 5) * 2 / 3",
    })
    fmt.Println(result) // 10

    // 幂运算
    result, _ = calc.Execute(context.Background(), map[string]any{
        "expression": "2^8",
    })
    fmt.Println(result) // 256
}
```

支持的运算符：
- 加法：`+`
- 减法：`-`
- 乘法：`*`
- 除法：`/`
- 取模：`%`
- 幂运算：`^`
- 括号：`()`

### 2. HTTP Request Tool - HTTP 请求

```go
package main

import (
    "context"
    "fmt"
    "time"

    "github.com/zhuchenglong/langchain-go/core/tools"
)

func main() {
    // 创建 HTTP 工具（带安全限制）
    httpTool := tools.NewHTTPRequestTool(tools.HTTPRequestConfig{
        Timeout:        10 * time.Second,
        AllowedMethods: []string{"GET", "POST"},
        AllowedDomains: []string{"api.example.com", "jsonplaceholder.typicode.com"},
    })

    // GET 请求
    result, err := httpTool.Execute(context.Background(), map[string]any{
        "url":    "https://jsonplaceholder.typicode.com/posts/1",
        "method": "GET",
    })

    if err != nil {
        panic(err)
    }

    // 解析结果
    resp := result.(map[string]any)
    fmt.Printf("Status: %d\n", resp["status_code"])
    fmt.Printf("Body: %s\n", resp["body"])

    // POST 请求
    result, _ = httpTool.Execute(context.Background(), map[string]any{
        "url":    "https://api.example.com/data",
        "method": "POST",
        "headers": map[string]any{
            "Content-Type":  "application/json",
            "Authorization": "Bearer token",
        },
        "body": `{"key": "value"}`,
    })
}
```

### 3. JSONPlaceholder Tool - 测试 API

```go
package main

import (
    "context"
    "fmt"

    "github.com/zhuchenglong/langchain-go/core/tools"
)

func main() {
    // 用于测试的 API 工具
    jsonTool := tools.NewJSONPlaceholderTool()

    // 获取所有 posts
    result, _ := jsonTool.Execute(context.Background(), map[string]any{
        "resource": "posts",
    })

    // 获取特定 post
    result, _ = jsonTool.Execute(context.Background(), map[string]any{
        "resource": "posts",
        "id":       1.0,
    })

    fmt.Println(result)
}
```

---

## 工具执行器

`ToolExecutor` 管理多个工具并提供统一的执行接口：

### 基本用法

```go
package main

import (
    "context"
    "fmt"
    "time"

    "github.com/zhuchenglong/langchain-go/core/tools"
)

func main() {
    // 创建多个工具
    calc := tools.NewCalculatorTool()
    http := tools.NewHTTPRequestTool(tools.HTTPRequestConfig{
        Timeout: 5 * time.Second,
    })

    // 创建执行器
    executor := tools.NewToolExecutor(tools.ToolExecutorConfig{
        Tools:   []tools.Tool{calc, http},
        Timeout: 30 * time.Second, // 全局超时
    })

    // 执行工具
    result, err := executor.Execute(context.Background(), "calculator", map[string]any{
        "expression": "100 * 50",
    })

    if err != nil {
        panic(err)
    }

    fmt.Printf("Result: %v\n", result)
}
```

### 动态管理工具

```go
// 检查工具是否存在
if executor.HasTool("calculator") {
    fmt.Println("Calculator tool available")
}

// 获取工具
tool, exists := executor.GetTool("calculator")
if exists {
    result, _ := tool.Execute(ctx, args)
}

// 添加工具
newTool := tools.NewFunctionTool(...)
executor.AddTool(newTool)

// 移除工具
executor.RemoveTool("calculator")

// 获取所有工具
allTools := executor.GetAllTools()
fmt.Printf("Total tools: %d\n", len(allTools))

// 获取 types.Tool 列表（用于绑定到 ChatModel）
typesTools := executor.GetTypesTools()
```

---

## 与 ChatModel 集成

### 完整的 Agent 流程

```go
package main

import (
    "context"
    "fmt"
    "os"

    "github.com/zhuchenglong/langchain-go/core/chat/providers/openai"
    "github.com/zhuchenglong/langchain-go/core/tools"
    "github.com/zhuchenglong/langchain-go/pkg/types"
)

func main() {
    // 1. 创建工具
    calc := tools.NewCalculatorTool()
    http := tools.NewHTTPRequestTool(tools.HTTPRequestConfig{})

    // 2. 创建工具执行器
    executor := tools.NewToolExecutor(tools.ToolExecutorConfig{
        Tools: []tools.Tool{calc, http},
    })

    // 3. 创建 ChatModel
    model, _ := openai.New(openai.Config{
        APIKey: os.Getenv("OPENAI_API_KEY"),
        Model:  "gpt-4",
    })

    // 4. 绑定工具到模型
    modelWithTools := model.BindTools(executor.GetTypesTools())

    // 5. 发送用户请求
    messages := []types.Message{
        types.NewUserMessage("What is 234 * 567?"),
    }

    ctx := context.Background()
    response, _ := modelWithTools.Invoke(ctx, messages)

    // 6. 检查是否有工具调用
    if len(response.ToolCalls) > 0 {
        fmt.Println("Model wants to call tools:")

        for _, toolCall := range response.ToolCalls {
            fmt.Printf("- Tool: %s\n", toolCall.Function.Name)
            fmt.Printf("  Args: %s\n", toolCall.Function.Arguments)

            // 7. 执行工具
            result, err := executor.ExecuteToolCall(ctx, toolCall)
            if err != nil {
                fmt.Printf("  Error: %v\n", err)
                continue
            }

            fmt.Printf("  Result: %v\n", result)

            // 8. 添加工具结果到消息历史
            messages = append(messages, response)
            messages = append(messages, types.NewToolMessage(
                fmt.Sprintf("%v", result),
                toolCall.ID,
            ))
        }

        // 9. 再次调用模型获取最终答案
        finalResponse, _ := modelWithTools.Invoke(ctx, messages)
        fmt.Println("\nFinal answer:", finalResponse.Content)
    } else {
        fmt.Println("Direct response:", response.Content)
    }
}
```

输出示例：

```
Model wants to call tools:
- Tool: calculator
  Args: {"expression": "234 * 567"}
  Result: 132678

Final answer: 234 × 567 = 132,678
```

### Agent 循环

```go
func agentLoop(model *openai.ChatModel, executor *tools.ToolExecutor, userQuery string) string {
    messages := []types.Message{
        types.NewUserMessage(userQuery),
    }

    maxIterations := 5
    ctx := context.Background()

    for i := 0; i < maxIterations; i++ {
        // 调用模型
        response, err := model.Invoke(ctx, messages)
        if err != nil {
            return fmt.Sprintf("Error: %v", err)
        }

        // 将模型响应加入历史
        messages = append(messages, response)

        // 如果没有工具调用，返回最终答案
        if len(response.ToolCalls) == 0 {
            return response.Content
        }

        // 执行所有工具调用
        for _, toolCall := range response.ToolCalls {
            result, err := executor.ExecuteToolCall(ctx, toolCall)

            var content string
            if err != nil {
                content = fmt.Sprintf("Error: %v", err)
            } else {
                content = fmt.Sprintf("%v", result)
            }

            // 添加工具结果
            messages = append(messages, types.NewToolMessage(content, toolCall.ID))
        }
    }

    return "Max iterations reached"
}

// 使用
result := agentLoop(modelWithTools, executor, "Calculate 15% of 500 and tell me if it's greater than 70")
fmt.Println(result)
```

---

## 实战示例

### 示例 1: 数学助手

```go
package main

import (
    "context"
    "fmt"
    "os"

    "github.com/zhuchenglong/langchain-go/core/chat/providers/openai"
    "github.com/zhuchenglong/langchain-go/core/prompts"
    "github.com/zhuchenglong/langchain-go/core/tools"
    "github.com/zhuchenglong/langchain-go/pkg/types"
)

func main() {
    // 创建工具
    calc := tools.NewCalculatorTool()
    executor := tools.NewToolExecutor(tools.ToolExecutorConfig{
        Tools: []tools.Tool{calc},
    })

    // 创建模型
    model, _ := openai.New(openai.Config{
        APIKey: os.Getenv("OPENAI_API_KEY"),
        Model:  "gpt-4",
    })
    modelWithTools := model.BindTools(executor.GetTypesTools())

    // 创建提示词
    template := prompts.NewChatPromptTemplate(
        prompts.SystemMessagePromptTemplate("You are a math assistant. Use the calculator tool for computations."),
        prompts.HumanMessagePromptTemplate("{query}"),
    )

    // 处理用户查询
    messages, _ := template.FormatMessages(map[string]any{
        "query": "If I invest $10,000 at 7% annual interest for 5 years, how much will I have?",
    })

    response, _ := modelWithTools.Invoke(context.Background(), messages)

    // 执行工具调用
    if len(response.ToolCalls) > 0 {
        for _, toolCall := range response.ToolCalls {
            result, _ := executor.ExecuteToolCall(context.Background(), toolCall)
            fmt.Printf("%s: %v\n", toolCall.Function.Name, result)
        }
    }
}
```

### 示例 2: API 集成助手

```go
package main

import (
    "context"
    "fmt"
    "time"

    "github.com/zhuchenglong/langchain-go/core/tools"
    "github.com/zhuchenglong/langchain-go/pkg/types"
)

// 创建天气查询工具
func NewWeatherTool() tools.Tool {
    return tools.NewFunctionTool(tools.FunctionToolConfig{
        Name:        "get_weather",
        Description: "Get current weather for a city",
        Parameters: types.Schema{
            Type: "object",
            Properties: map[string]types.Schema{
                "city": {Type: "string", Description: "City name"},
            },
            Required: []string{"city"},
        },
        Fn: func(ctx context.Context, args map[string]any) (any, error) {
            city := args["city"].(string)
            // 实际实现应该调用天气 API
            return map[string]any{
                "city":        city,
                "temperature": 22,
                "condition":   "Sunny",
            }, nil
        },
    })
}

// 创建新闻搜索工具
func NewNewsSearchTool() tools.Tool {
    return tools.NewFunctionTool(tools.FunctionToolConfig{
        Name:        "search_news",
        Description: "Search for recent news articles",
        Parameters: types.Schema{
            Type: "object",
            Properties: map[string]types.Schema{
                "query": {Type: "string", Description: "Search query"},
            },
            Required: []string{"query"},
        },
        Fn: func(ctx context.Context, args map[string]any) (any, error) {
            query := args["query"].(string)
            // 实际实现应该调用新闻 API
            return []map[string]any{
                {"title": "Breaking news about " + query, "source": "News API"},
            }, nil
        },
    })
}

func main() {
    executor := tools.NewToolExecutor(tools.ToolExecutorConfig{
        Tools: []tools.Tool{
            NewWeatherTool(),
            NewNewsSearchTool(),
        },
        Timeout: 10 * time.Second,
    })

    // 执行工具
    weather, _ := executor.Execute(context.Background(), "get_weather", map[string]any{
        "city": "Beijing",
    })
    fmt.Printf("Weather: %v\n", weather)

    news, _ := executor.Execute(context.Background(), "search_news", map[string]any{
        "query": "AI technology",
    })
    fmt.Printf("News: %v\n", news)
}
```

### 示例 3: 数据分析助手

```go
package main

import (
    "context"
    "fmt"

    "github.com/zhuchenglong/langchain-go/core/tools"
    "github.com/zhuchenglong/langchain-go/pkg/types"
)

// CSV 读取工具
func NewCSVReaderTool() tools.Tool {
    return tools.NewFunctionTool(tools.FunctionToolConfig{
        Name:        "read_csv",
        Description: "Read data from a CSV file",
        Parameters: types.Schema{
            Type: "object",
            Properties: map[string]types.Schema{
                "file_path": {Type: "string", Description: "Path to CSV file"},
            },
            Required: []string{"file_path"},
        },
        Fn: func(ctx context.Context, args map[string]any) (any, error) {
            // 实现 CSV 读取逻辑
            return [][]string{
                {"Name", "Age", "City"},
                {"Alice", "30", "NYC"},
                {"Bob", "25", "LA"},
            }, nil
        },
    })
}

// 数据统计工具
func NewDataStatsTool() tools.Tool {
    return tools.NewFunctionTool(tools.FunctionToolConfig{
        Name:        "calculate_stats",
        Description: "Calculate statistics for a list of numbers",
        Parameters: types.Schema{
            Type: "object",
            Properties: map[string]types.Schema{
                "numbers": {Type: "array", Description: "List of numbers"},
            },
            Required: []string{"numbers"},
        },
        Fn: func(ctx context.Context, args map[string]any) (any, error) {
            nums := args["numbers"].([]any)
            sum := 0.0
            for _, n := range nums {
                sum += n.(float64)
            }
            avg := sum / float64(len(nums))

            return map[string]any{
                "count":   len(nums),
                "sum":     sum,
                "average": avg,
            }, nil
        },
    })
}

func main() {
    executor := tools.NewToolExecutor(tools.ToolExecutorConfig{
        Tools: []tools.Tool{
            NewCSVReaderTool(),
            NewDataStatsTool(),
            tools.NewCalculatorTool(),
        },
    })

    // Agent 可以使用这些工具进行数据分析
    fmt.Println("Data analysis tools ready:", len(executor.GetAllTools()))
}
```

---

## 最佳实践

### 1. 工具设计原则

```go
// ✅ 好的工具设计
func NewGoodTool() tools.Tool {
    return tools.NewFunctionTool(tools.FunctionToolConfig{
        Name:        "search_documents",
        Description: "Search for documents in the knowledge base by keyword",
        Parameters: types.Schema{
            Type: "object",
            Properties: map[string]types.Schema{
                "query": {
                    Type:        "string",
                    Description: "Search query (keywords or phrase)",
                },
                "limit": {
                    Type:        "integer",
                    Description: "Maximum number of results (default: 10)",
                },
            },
            Required: []string{"query"},
        },
        Fn: func(ctx context.Context, args map[string]any) (any, error) {
            // 实现
            return results, nil
        },
    })
}

// ❌ 避免：描述不清晰
func NewBadTool() tools.Tool {
    return tools.NewFunctionTool(tools.FunctionToolConfig{
        Name:        "search",
        Description: "Search stuff",  // 太模糊！
        // ...
    })
}
```

### 2. 错误处理

```go
func (t *MyTool) Execute(ctx context.Context, args map[string]any) (any, error) {
    // 验证参数
    query, ok := args["query"].(string)
    if !ok || query == "" {
        return nil, fmt.Errorf("%w: missing or invalid 'query'", tools.ErrInvalidArguments)
    }

    // 检查上下文取消
    select {
    case <-ctx.Done():
        return nil, ctx.Err()
    default:
    }

    // 执行逻辑
    result, err := performSearch(query)
    if err != nil {
        return nil, fmt.Errorf("%w: %v", tools.ErrExecutionFailed, err)
    }

    return result, nil
}
```

### 3. 超时控制

```go
// 方式 1: 使用 ToolExecutor 的全局超时
executor := tools.NewToolExecutor(tools.ToolExecutorConfig{
    Tools:   []tools.Tool{...},
    Timeout: 30 * time.Second,
})

// 方式 2: 为单个调用设置超时
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

result, err := executor.Execute(ctx, "slow_tool", args)
```

### 4. 安全考虑

```go
// ✅ 限制允许的操作
httpTool := tools.NewHTTPRequestTool(tools.HTTPRequestConfig{
    AllowedMethods: []string{"GET"},  // 只允许 GET
    AllowedDomains: []string{          // 白名单域名
        "api.trusted-service.com",
        "data.company.com",
    },
    Timeout: 10 * time.Second,
})

// ✅ 验证输入
func (t *FileTool) Execute(ctx context.Context, args map[string]any) (any, error) {
    path := args["path"].(string)

    // 防止路径遍历攻击
    if strings.Contains(path, "..") {
        return nil, fmt.Errorf("%w: invalid path", tools.ErrInvalidArguments)
    }

    // 检查路径是否在允许的目录内
    if !strings.HasPrefix(path, "/safe/directory") {
        return nil, fmt.Errorf("%w: path not allowed", tools.ErrInvalidArguments)
    }

    // 继续执行
}
```

### 5. 可测试性

```go
// 使用依赖注入使工具可测试
type APITool struct {
    client HTTPClient  // 接口而不是具体类型
}

type HTTPClient interface {
    Get(url string) ([]byte, error)
}

// 测试时可以使用 mock
func TestAPITool(t *testing.T) {
    mockClient := &MockHTTPClient{
        response: []byte(`{"result": "test"}`),
    }

    tool := &APITool{client: mockClient}
    result, err := tool.Execute(ctx, args)

    assert.NoError(t, err)
    // ...
}
```

### 6. 日志和监控

```go
func (t *MyTool) Execute(ctx context.Context, args map[string]any) (any, error) {
    start := time.Now()

    // 记录开始
    log.Printf("Tool %s started with args: %v", t.GetName(), args)

    result, err := t.doExecute(ctx, args)

    // 记录完成
    duration := time.Since(start)
    if err != nil {
        log.Printf("Tool %s failed after %v: %v", t.GetName(), duration, err)
    } else {
        log.Printf("Tool %s completed in %v", t.GetName(), duration)
    }

    return result, err
}
```

---

## 常见问题

### Q: 如何处理工具执行失败？

```go
result, err := executor.Execute(ctx, "my_tool", args)
if err != nil {
    if errors.Is(err, tools.ErrToolNotFound) {
        // 工具不存在
    } else if errors.Is(err, tools.ErrInvalidArguments) {
        // 参数无效
    } else if errors.Is(err, tools.ErrTimeout) {
        // 超时
    } else {
        // 其他错误
    }
}
```

### Q: 工具可以调用其他工具吗？

可以！但要小心避免无限循环：

```go
type ComposeTool struct {
    executor *tools.ToolExecutor
    maxDepth int
}

func (t *ComposeTool) Execute(ctx context.Context, args map[string]any) (any, error) {
    depth := getDepthFromContext(ctx)
    if depth >= t.maxDepth {
        return nil, fmt.Errorf("max tool call depth reached")
    }

    ctx = setDepthInContext(ctx, depth+1)

    // 调用其他工具
    result, err := t.executor.Execute(ctx, "other_tool", otherArgs)
    // ...
}
```

### Q: 如何让工具返回流式结果？

当前 Tool 接口不直接支持流式输出，但可以通过返回 channel 实现：

```go
Fn: func(ctx context.Context, args map[string]any) (any, error) {
    resultChan := make(chan string, 10)

    go func() {
        defer close(resultChan)
        // 生成流式数据
        for i := 0; i < 10; i++ {
            resultChan <- fmt.Sprintf("chunk %d", i)
        }
    }()

    return resultChan, nil
}
```

---

## 参考

- [API 文档](https://pkg.go.dev/langchain-go/core/tools)
- [ChatModel 集成](./chat-examples.md)
- [Prompts 使用](./prompts-examples.md)

---

**最后更新**: 2026-01-14
