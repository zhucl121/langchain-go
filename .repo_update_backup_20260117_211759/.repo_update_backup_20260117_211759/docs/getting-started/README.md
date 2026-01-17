# 快速开始

欢迎使用 LangChain-Go！本指南将帮助你在几分钟内开始使用。

---

## 📖 快速导航

1. [安装指南](./installation.md) - 环境准备和依赖安装
2. [5分钟快速开始](./quickstart.md) - 最快上手教程
3. [ChatModel 快速开始](./quickstart-chat.md) - 对话模型使用
4. [Prompts 快速开始](./quickstart-prompts.md) - 提示词模板
5. [OutputParser 快速开始](./quickstart-output.md) - 输出解析
6. [Tools 快速开始](./quickstart-tools.md) - 工具系统
7. [Memory 快速开始](./quickstart-memory.md) - 记忆系统
8. [StateGraph 快速开始](./quickstart-stategraph.md) - 状态图工作流

---

## 🚀 推荐学习顺序

### 第一步：安装（5分钟）
从[安装指南](./installation.md)开始，设置开发环境。

```bash
go get github.com/yourusername/langchain-go
```

### 第二步：基础使用（10分钟）
跟随[快速开始](./quickstart.md)，学习基本概念：
- 调用 LLM
- 使用提示词模板
- 解析 LLM 输出

### 第三步：核心组件（30分钟）
根据需求选择学习：
- 构建对话系统 → [ChatModel](./quickstart-chat.md)
- 设计提示词 → [Prompts](./quickstart-prompts.md)
- 解析结构化输出 → [OutputParser](./quickstart-output.md)
- 使用工具 → [Tools](./quickstart-tools.md)

### 第四步：高级功能（1小时）
- 管理对话历史 → [Memory](./quickstart-memory.md)
- 编排复杂工作流 → [StateGraph](./quickstart-stategraph.md)

---

## 💡 快速示例

### Hello World

```go
package main

import (
    "context"
    "fmt"
    "github.com/zhucl121/langchain-go/core/chat/providers/openai"
    "github.com/zhucl121/langchain-go/pkg/types"
)

func main() {
    model := openai.New(openai.Config{
        APIKey: "your-api-key",
        Model:  "gpt-4",
    })
    
    response, _ := model.Invoke(context.Background(), []types.Message{
        types.NewUserMessage("你好！"),
    })
    
    fmt.Println(response.Content)
}
```

### 链式组合（LCEL）

```go
chain := prompt.Pipe(model).Pipe(parser)
result, _ := chain.Invoke(ctx, input)
```

### StateGraph 工作流

```go
graph := state.NewStateGraph[MyState]("workflow")
graph.AddNode("step1", node1)
graph.AddEdge("step1", "step2")
app, _ := graph.Compile()
result, _ := app.Invoke(ctx, initialState)
```

---

## 📚 相关资源

- [使用指南](../guides/) - 详细的功能文档
- [示例代码](../examples/) - 更多代码示例
- [API 文档](https://pkg.go.dev/langchain-go) - 完整 API 参考

---

## 🆘 遇到问题？

- 查看 [常见问题](../reference/faq.md)
- 搜索 [GitHub Issues](https://github.com/yourusername/langchain-go/issues)
- 提问 [Discussions](https://github.com/yourusername/langchain-go/discussions)

---

## ➡️ 下一步

完成快速开始后，推荐：

1. 深入学习 [核心功能指南](../guides/core/)
2. 探索 [LangGraph 工作流](../guides/langgraph/)
3. 构建 [Agent 应用](../guides/agents/)
4. 实现 [RAG 系统](../guides/rag/)

---

<div align="center">

**[⬆ 回到文档首页](../README.md)**

</div>
