# LangGraph 指南

LangGraph 状态图工作流系统的详细使用指南。

---

## 📖 指南列表

- [StateGraph 状态图](./stategraph.md) - 工作流编排和执行
- [Checkpoint 检查点](./checkpoint.md) - 状态持久化系统
- [Durability 持久性](./durability.md) - 故障恢复和重试
- HITL 人机协作 - Human-in-the-Loop（即将添加）

---

## 🎯 学习路径

### 第一步：理解 StateGraph
[StateGraph](./stategraph.md) 是 LangGraph 的核心，学习：
- 如何定义状态结构
- 如何添加节点和边
- 如何编译和执行图

### 第二步：持久化状态
[Checkpoint](./checkpoint.md) 让你的应用可以：
- 保存执行状态
- 从任意点恢复
- 实现时间旅行
- 支持多用户会话

### 第三步：容错处理
[Durability](./durability.md) 提供三种持久性保证：
- AtMostOnce - 最多一次
- AtLeastOnce - 至少一次
- ExactlyOnce - 恰好一次

### 第四步：人机协作
HITL（即将添加）让你的工作流可以：
- 在关键点暂停
- 等待人工审批
- 根据反馈继续执行

---

## 💡 核心概念

### StateGraph
状态图是 LangGraph 的核心抽象，用于定义复杂的工作流。

```go
type MyState struct {
    Messages []string
    Step     int
}

graph := state.NewStateGraph[MyState]("workflow")
```

### 节点 (Nodes)
节点是执行逻辑的基本单元。

```go
graph.AddNode("process", func(ctx context.Context, state MyState) (MyState, error) {
    // 处理逻辑
    state.Step++
    return state, nil
})
```

### 边 (Edges)
边定义了节点之间的流转。

```go
// 普通边
graph.AddEdge("step1", "step2")

// 条件边
graph.AddConditionalEdges("router", routerFunc, map[string]string{
    "left":  "leftNode",
    "right": "rightNode",
})
```

### Checkpoint
检查点用于持久化状态。

```go
checkpointer := postgres.NewSaver("postgresql://...")
app := graph.WithCheckpointer(checkpointer).Compile()
```

---

## 🚀 快速示例

### 简单工作流

```go
graph := state.NewStateGraph[MyState]("app")
graph.AddNode("start", startNode)
graph.AddNode("process", processNode)
graph.AddEdge("start", "process")
graph.SetEntryPoint("start")

app, _ := graph.Compile()
result, _ := app.Invoke(ctx, initialState)
```

### 带持久化的工作流

```go
checkpointer, _ := sqlite.NewSaver("checkpoints.db")
app := graph.WithCheckpointer(checkpointer).Compile()

// 自动保存状态
result, _ := app.Invoke(ctx, state, execute.WithThreadID("user-123"))

// 恢复执行
result, _ := app.Invoke(ctx, state, execute.WithThreadID("user-123"))
```

### 条件分支工作流

```go
graph.AddConditionalEdges("decision", func(ctx context.Context, state MyState) (string, error) {
    if state.Score > 0.8 {
        return "success", nil
    }
    return "retry", nil
}, map[string]string{
    "success": state.END,
    "retry":   "process",
})
```

---

## 📚 相关资源

- [快速开始](../../getting-started/quickstart-stategraph.md) - StateGraph 快速入门
- [核心功能指南](../core/) - 核心组件文档
- [Agent 指南](../agents/) - Agent 系统
- [示例代码](../../examples/) - 实用示例

---

<div align="center">

**[⬆ 回到指南首页](../README.md)** | **[回到文档首页](../../README.md)**

</div>
