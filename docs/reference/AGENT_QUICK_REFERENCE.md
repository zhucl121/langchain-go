# 🚀 LangChain-Go Agent 快速参考

## 📦 安装

```bash
go get github.com/zhuchenglong/langchain-go
```

---

## ⚡ 3 行创建 Agent

```go
agent := agents.CreateReActAgent(llm, tools)
executor := agents.NewSimplifiedAgentExecutor(agent, tools)
result, _ := executor.Run(ctx, "question")
```

---

## 🛠️ 内置工具 (16个)

### 快速获取

```go
// 所有工具 (16个)
tools.GetBuiltinTools()

// 基础工具 (4个: Calculator, GetTime, GetDate, HTTPGet)
tools.GetBasicTools()

// 按分类
tools.GetTimeTools()    // 5个时间工具
tools.GetHTTPTools()    // 3个HTTP工具  
tools.GetJSONTools()    // 3个JSON工具
tools.GetStringTools()  // 3个字符串工具
```

### 工具列表

| 工具名 | 功能 | 使用 |
|--------|------|------|
| `calculator` | 数学计算 | `NewCalculator()` |
| `get_time` | 当前时间 | `NewGetTimeTool(nil)` |
| `get_date` | 当前日期 | `NewGetDateTool(nil)` |
| `get_datetime` | 日期时间 | `NewGetDateTimeTool(nil)` |
| `format_time` | 时间格式化 | `NewFormatTimeTool()` |
| `get_day_of_week` | 星期几 | `NewGetDayOfWeekTool()` |
| `http_get` | HTTP GET | `NewHTTPGetTool(nil)` |
| `http_post` | HTTP POST | `NewHTTPPostTool(nil)` |
| `http_request` | 通用HTTP | `NewHTTPRequestTool(nil)` |
| `json_parse` | JSON解析 | `NewJSONParseTool()` |
| `json_stringify` | JSON序列化 | `NewJSONStringifyTool()` |
| `json_extract` | JSON提取 | `NewJSONExtractTool()` |
| `string_length` | 字符串长度 | `NewStringLengthTool()` |
| `string_split` | 字符串分割 | `NewStringSplitTool()` |
| `string_join` | 字符串连接 | `NewStringJoinTool()` |

---

## 🎯 Agent 类型

### 1. ReAct Agent

```go
agent := agents.CreateReActAgent(llm, tools,
    agents.WithMaxSteps(10),
    agents.WithVerbose(true),
)
```

### 2. Tool Calling Agent

```go
agent := agents.CreateToolCallingAgent(llm, tools,
    agents.WithSystemPrompt("You are helpful"),
)
```

### 3. Conversational Agent

```go
agent := agents.CreateConversationalAgent(llm)
```

---

## ⚙️ 配置选项

```go
agents.WithMaxSteps(10)              // 最大步数
agents.WithSystemPrompt("...")       // 系统提示词
agents.WithVerbose(true)             // 详细日志
agents.WithExtra("key", value)       // 额外配置
```

---

## 🔄 执行模式

### 同步执行

```go
result, err := executor.Run(ctx, "question")
fmt.Println(result.Output)
```

### 流式执行

```go
eventChan := executor.Stream(ctx, "question")

for event := range eventChan {
    switch event.Type {
    case agents.EventTypeStart:
        // 开始
    case agents.EventTypeToolCall:
        // 工具调用
    case agents.EventTypeToolResult:
        // 工具结果
    case agents.EventTypeFinish:
        // 完成
    }
}
```

---

## 📚 完整示例

### 示例 1: 最简单

```go
package main

import (
    "context"
    "github.com/zhuchenglong/langchain-go/core/agents"
    "github.com/zhuchenglong/langchain-go/core/chat/ollama"
    "github.com/zhuchenglong/langchain-go/core/tools"
)

func main() {
    ctx := context.Background()
    llm := ollama.NewChatOllama("qwen2.5:7b")
    
    // 1 行获取工具
    agentTools := tools.GetBasicTools()
    
    // 1 行创建 Agent
    agent := agents.CreateReActAgent(llm, agentTools)
    
    // 1 行创建执行器
    executor := agents.NewSimplifiedAgentExecutor(agent, agentTools)
    
    // 1 行执行
    result, _ := executor.Run(ctx, "What time is it?")
    
    println(result.Output)
}
```

### 示例 2: 流式执行

```go
agent := agents.CreateReActAgent(llm, tools)
executor := agents.NewSimplifiedAgentExecutor(agent, tools)

eventChan := executor.Stream(ctx, "Calculate 10+20")

for event := range eventChan {
    switch event.Type {
    case agents.EventTypeStart:
        fmt.Println("🚀 Started")
    case agents.EventTypeToolCall:
        fmt.Printf("🔧 Tool: %s\n", event.Action.Tool)
    case agents.EventTypeToolResult:
        fmt.Printf("📊 Result: %s\n", event.Observation)
    case agents.EventTypeFinish:
        fmt.Printf("✅ Done: %s\n", event.Observation)
    }
}
```

### 示例 3: 自定义工具

```go
// 创建自定义工具
customTool := tools.NewFunctionTool(tools.FunctionToolConfig{
    Name:        "greet",
    Description: "Greet someone",
    Parameters: tools.Schema{
        Type: "object",
        Properties: map[string]tools.Schema{
            "name": {Type: "string"},
        },
    },
    Fn: func(ctx context.Context, args map[string]any) (any, error) {
        name := args["name"].(string)
        return fmt.Sprintf("Hello, %s!", name), nil
    },
})

// 组合工具
agentTools := []tools.Tool{
    customTool,
    tools.NewCalculator(),
}

agent := agents.CreateReActAgent(llm, agentTools)
```

### 示例 4: 工具注册表

```go
// 创建注册表
registry := tools.NewToolRegistry()

// 注册工具
registry.RegisterAll(tools.GetBasicTools())
registry.Register(customTool)

// 使用注册表
if tool, exists := registry.Get("calculator"); exists {
    result, _ := tool.Execute(ctx, args)
}

// 获取所有工具
allTools := registry.GetAll()
agent := agents.CreateReActAgent(llm, allTools)
```

---

## 🧪 测试

```bash
# 运行测试
go test ./core/agents/...
go test ./core/tools/...

# 性能测试
go test -bench=. ./core/tools/...

# 运行示例
go run examples/agent_simple_demo.go
```

---

## 📊 性能

| 操作 | 耗时 | 备注 |
|------|------|------|
| 创建 Agent | <1ms | 极快 |
| 获取工具 | <1ms | 预加载 |
| 工具执行 | <1ms | 本地工具 |
| LLM 调用 | 1-5s | 取决于模型 |

---

## 🎓 最佳实践

### 1. 选择合适的工具集

```go
// 基础任务
tools.GetBasicTools()

// 需要时间功能
tools.GetTimeTools()

// 需要HTTP
tools.GetHTTPTools()

// 需要所有功能
tools.GetBuiltinTools()
```

### 2. 设置合理的 MaxSteps

```go
// 简单任务
agents.WithMaxSteps(5)

// 复杂任务
agents.WithMaxSteps(15)

// 默认
agents.WithMaxSteps(10)
```

### 3. 使用流式执行获得更好的用户体验

```go
eventChan := executor.Stream(ctx, question)
// 实时反馈给用户
```

### 4. 使用 Verbose 调试

```go
agent := agents.CreateReActAgent(llm, tools,
    agents.WithVerbose(true),  // 输出详细日志
)
```

---

## 🔗 相关文档

- [OPTIMIZATION_COMPLETE.md](./OPTIMIZATION_COMPLETE.md) - 完整优化报告
- [FEATURE_COMPLETION_STATUS.md](./FEATURE_COMPLETION_STATUS.md) - 功能完成状况
- [examples/agent_simple_demo.go](./examples/agent_simple_demo.go) - 完整示例

---

## ❓ 常见问题

### Q: 如何添加自定义工具？

```go
customTool := tools.NewFunctionTool(tools.FunctionToolConfig{
    Name:        "my_tool",
    Description: "My custom tool",
    Parameters:  schema,
    Fn:          myFunction,
})
```

### Q: 如何限制执行时间？

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

result, err := executor.Run(ctx, question)
```

### Q: 如何处理错误？

```go
result, err := executor.Run(ctx, question)
if err != nil {
    if errors.Is(err, agents.ErrAgentMaxSteps) {
        // 达到最大步数
    }
    // 其他错误处理
}
```

### Q: 支持哪些 LLM？

```go
// Ollama (本地)
ollama.NewChatOllama("qwen2.5:7b")

// OpenAI
openai.NewChatOpenAI(apiKey, "gpt-4")

// 其他实现 chat.ChatModel 接口的 LLM
```

---

**版本**: v1.1.0  
**更新日期**: 2026-01-16

🎉 **现在开始使用 LangChain-Go 构建智能 Agent!**
