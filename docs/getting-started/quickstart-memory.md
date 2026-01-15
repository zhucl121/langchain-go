# Memory 系统快速开始

> LangChain-Go Memory 系统使用指南

## 快速导航

- [基础概念](#基础概念)
- [Buffer Memory](#buffer-memory-完整历史)
- [Window Memory](#window-memory-滑动窗口)
- [Summary Memory](#summary-memory-智能摘要)
- [与 ChatModel 集成](#与-chatmodel-集成)

---

## 基础概念

Memory 让 AI Agent 能够记住对话历史，保持上下文连贯性。

### Memory 接口

```go
type Memory interface {
    LoadMemoryVariables(ctx context.Context, inputs map[string]any) (map[string]any, error)
    SaveContext(ctx context.Context, inputs map[string]any, outputs map[string]any) error
    Clear(ctx context.Context) error
}
```

---

## Buffer Memory (完整历史)

保存所有对话历史，适合短对话场景。

### 基本使用

```go
package main

import (
    "context"
    "fmt"
    "langchain-go/core/memory"
)

func main() {
    ctx := context.Background()
    
    // 创建 BufferMemory
    mem := memory.NewBufferMemory()
    
    // 保存对话
    mem.SaveContext(ctx,
        map[string]any{"input": "你好"},
        map[string]any{"output": "你好！我是 AI 助手。"},
    )
    
    mem.SaveContext(ctx,
        map[string]any{"input": "今天天气怎么样？"},
        map[string]any{"output": "今天天气很好，阳光明媚。"},
    )
    
    // 加载历史
    vars, _ := mem.LoadMemoryVariables(ctx, nil)
    history := vars["history"].([]types.Message)
    
    for _, msg := range history {
        fmt.Printf("%s: %s\n", msg.Role, msg.Content)
    }
}
```

### 输出

```
user: 你好
assistant: 你好！我是 AI 助手。
user: 今天天气怎么样？
assistant: 今天天气很好，阳光明媚。
```

---

## Window Memory (滑动窗口)

只保留最近的 K 轮对话，自动丢弃旧对话。

### 基本使用

```go
mem := memory.NewConversationBufferWindowMemory(memory.WindowMemoryConfig{
    K: 3, // 只保留最近 3 轮对话（6 条消息）
})

// 添加多轮对话
for i := 1; i <= 5; i++ {
    mem.SaveContext(ctx,
        map[string]any{"input": fmt.Sprintf("问题 %d", i)},
        map[string]any{"output": fmt.Sprintf("回答 %d", i)},
    )
}

// 只会保留最近 3 轮
vars, _ := mem.LoadMemoryVariables(ctx, nil)
history := vars["history"].([]types.Message)
fmt.Println("保留的对话数:", len(history)) // 输出: 6
```

---

## Summary Memory (智能摘要)

使用 LLM 将长对话压缩为摘要，节省 Token。

### 基本使用

```go
import (
    "langchain-go/core/chat/providers/openai"
    "langchain-go/core/memory"
)

// 创建 LLM
model, _ := openai.New(openai.Config{
    APIKey: "your-api-key",
    Model:  "gpt-4",
})

// 创建 SummaryMemory
mem := memory.NewConversationSummaryMemory(memory.SummaryMemoryConfig{
    LLM:       model,
    MaxTokens: 2000, // 超过此限制时触发摘要
})

// 添加多轮对话
for i := 1; i <= 10; i++ {
    mem.SaveContext(ctx,
        map[string]any{"input": fmt.Sprintf("长问题 %d...", i)},
        map[string]any{"output": fmt.Sprintf("长回答 %d...", i)},
    )
}

// 会自动生成摘要
summary := mem.GetSummary()
fmt.Println("对话摘要:", summary)
```

---

## 与 ChatModel 集成

Memory 与 ChatModel 无缝集成，构建有记忆的对话系统。

### 完整示例

```go
package main

import (
    "context"
    "fmt"
    "langchain-go/core/chat/providers/openai"
    "langchain-go/core/memory"
    "langchain-go/pkg/types"
)

func main() {
    ctx := context.Background()
    
    // 创建模型
    model, _ := openai.New(openai.Config{
        APIKey: "your-api-key",
        Model:  "gpt-4",
    })
    
    // 创建记忆
    mem := memory.NewBufferMemory()
    
    // 对话循环
    questions := []string{
        "你好，我叫张三",
        "我叫什么名字？", // 测试记忆
        "今天天气不错",
        "我们之前聊了什么？", // 测试记忆
    }
    
    for _, question := range questions {
        fmt.Println("\n用户:", question)
        
        // 1. 加载历史记忆
        memVars, _ := mem.LoadMemoryVariables(ctx, nil)
        history := memVars["history"].([]types.Message)
        
        // 2. 添加新的用户消息
        messages := append(history, types.NewUserMessage(question))
        
        // 3. 调用模型
        response, _ := model.Invoke(ctx, messages)
        fmt.Println("AI:", response.Content)
        
        // 4. 保存对话到记忆
        mem.SaveContext(ctx,
            map[string]any{"input": question},
            map[string]any{"output": response.Content},
        )
    }
}
```

### 输出示例

```
用户: 你好，我叫张三
AI: 你好，张三！很高兴认识你。

用户: 我叫什么名字？
AI: 你叫张三。

用户: 今天天气不错
AI: 是的，天气很好！

用户: 我们之前聊了什么？
AI: 我们聊了你的名字（张三）和今天的天气。
```

---

## 自定义键名

```go
mem := memory.NewBufferMemory()

// 自定义输入/输出键名
mem.SetInputKey("user_message")
mem.SetOutputKey("ai_response")

// 自定义返回键名
mem.SetMemoryKey("chat_history")

// 保存时使用自定义键名
mem.SaveContext(ctx,
    map[string]any{"user_message": "Hello"},
    map[string]any{"ai_response": "Hi!"},
)

// 加载时也使用自定义键名
vars, _ := mem.LoadMemoryVariables(ctx, nil)
history := vars["chat_history"].([]types.Message)
```

---

## 字符串格式

```go
mem := memory.NewBufferMemory()
mem.SetReturnMessages(false) // 返回字符串而非消息列表

mem.SaveContext(ctx,
    map[string]any{"input": "Hello"},
    map[string]any{"output": "Hi!"},
)

vars, _ := mem.LoadMemoryVariables(ctx, nil)
historyStr := vars["history"].(string)

fmt.Println(historyStr)
// 输出: Human: Hello\nAI: Hi!
```

---

## 清空记忆

```go
// 清空所有历史
mem.Clear(ctx)
```

---

## 最佳实践

### 1. 选择合适的 Memory 类型

- **短对话**: BufferMemory
- **长对话**: WindowMemory 或 SummaryMemory
- **成本敏感**: WindowMemory (节省 Token)
- **保留完整上下文**: SummaryMemory

### 2. 合理设置窗口大小

```go
// 根据场景设置 K 值
mem := memory.NewConversationBufferWindowMemory(memory.WindowMemoryConfig{
    K: 5, // 一般对话
    // K: 10, // 复杂任务
    // K: 2, // 简单查询
})
```

### 3. 监控 Token 使用

```go
// SummaryMemory 自动控制
mem := memory.NewConversationSummaryMemory(memory.SummaryMemoryConfig{
    LLM:       model,
    MaxTokens: 2000, // 根据模型上下文限制调整
})
```

### 4. 错误处理

```go
if err := mem.SaveContext(ctx, inputs, outputs); err != nil {
    log.Printf("保存对话失败: %v", err)
    // 处理错误
}
```

---

## 下一步

- 查看 [Memory 详细示例](./docs/memory-examples.md)
- 查看 [Memory 系统总结](./docs/M19-M21-Memory-Summary.md)
- 查看 [ChatModel 集成](./QUICKSTART-CHAT.md)
- 查看 [完整项目文档](./README.md)
