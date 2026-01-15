# 示例代码

实用的代码示例和最佳实践。

---

## 📖 示例列表

### 核心功能示例
- [ChatModel 示例](./chat.md) - 对话模型使用示例
- [Prompts 示例](./prompts.md) - 提示词模板示例
- [OutputParser 示例](./output-parser.md) - 输出解析示例
- [Tools 示例](./tools.md) - 工具使用示例

### 完整应用示例
查看 `examples/` 目录了解完整应用：

- `examples/basic/` - 基础示例
- `examples/agents/` - Agent 示例
- `examples/rag/` - RAG 系统示例

---

## 💡 快速导航

### 按功能查找

- **调用 OpenAI** → [ChatModel 示例](./chat-examples.md#openai)
- **调用 Anthropic** → [ChatModel 示例](./chat-examples.md#anthropic)
- **流式输出** → [ChatModel 示例](./chat-examples.md#streaming)
- **Few-Shot 学习** → [Prompts 示例](./prompts-examples.md#few-shot)
- **JSON 解析** → [OutputParser 示例](./output-examples.md#json)
- **工具调用** → [Tools 示例](./tools-examples.md)

---

## 🚀 推荐示例

### 1. 简单对话

```go
model := openai.New(openai.Config{APIKey: "sk-..."})
response, _ := model.Invoke(ctx, []types.Message{
    types.NewUserMessage("Hello!"),
})
fmt.Println(response.Content)
```

### 2. 链式组合

```go
chain := prompt.Pipe(model).Pipe(parser)
result, _ := chain.Invoke(ctx, input)
```

### 3. StateGraph 工作流

```go
graph := state.NewStateGraph[MyState]("app")
graph.AddNode("step1", node1)
graph.AddEdge("step1", "step2")
app, _ := graph.Compile()
result, _ := app.Invoke(ctx, initialState)
```

---

## 📚 更多资源

- [快速开始](../getting-started/) - 入门教程
- [使用指南](../guides/) - 详细指南
- [高级主题](../advanced/) - 高级功能

---

<div align="center">

**[⬆ 回到文档首页](../README.md)**

</div>
