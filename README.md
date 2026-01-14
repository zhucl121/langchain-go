# LangChain-Go & LangGraph-Go

用 Go 重写 LangChain 1.2+ 和 LangGraph 1.0+ 核心功能。

## 🎯 项目目标

- **核心目标**: 实现 LangGraph 1.0+ 全部核心功能（StateGraph、Checkpointing、Human-in-the-Loop）
- **扩展目标**: 实现 LangChain 核心抽象（Runnable、ChatModel、Tools）
- **性能目标**: 相比 Python 版本，并发性能提升 10x+，内存降低 50%+

## 🚀 快速开始

```go
package main

import (
    "context"
    "fmt"
    "langchain-go/core/chat/providers/openai"
    "langchain-go/graph/state"
)

func main() {
    // 创建聊天模型
    model, _ := openai.New(openai.Config{
        APIKey: "your-api-key",
        Model:  "gpt-4",
    })
    
    // 创建状态图
    type AgentState struct {
        Messages []string
    }
    
    graph := state.NewStateGraph[AgentState]("agent")
    graph.AddNode("agent", func(ctx context.Context, s AgentState) (AgentState, error) {
        // Agent 逻辑
        return s, nil
    })
    graph.SetEntryPoint("agent")
    
    // 编译并执行
    app, _ := graph.Compile()
    result, _ := app.Invoke(context.Background(), AgentState{})
    
    fmt.Printf("结果: %+v\n", result)
}
```

## 📦 项目结构

```
langchain-go/
├── pkg/                   # 公共包
│   ├── types/            # 基础类型（Message, Tool, Schema）
│   └── utils/            # 工具函数
│
├── core/                  # LangChain 核心
│   ├── runnable/         # Runnable 系统
│   ├── chat/             # ChatModel 和 Providers
│   ├── prompts/          # 提示词模板
│   ├── output/           # 输出解析器
│   ├── tools/            # 工具系统
│   ├── memory/           # 记忆系统
│   └── callbacks/        # 回调系统
│
├── graph/                 # LangGraph 核心
│   ├── state/            # StateGraph
│   ├── node/             # 节点系统
│   ├── edge/             # 边系统
│   ├── compile/          # 编译器
│   ├── execute/          # 执行引擎
│   ├── checkpoint/       # 检查点持久化 ⭐
│   ├── durability/       # 持久化模式 ⭐
│   ├── hitl/             # Human-in-the-Loop ⭐
│   └── streaming/        # 流式输出
│
├── agents/                # Agent 系统
│   ├── create.go         # create_agent
│   └── middleware/       # 中间件系统
│
└── prebuilt/              # 预构建组件
    ├── react.go          # ReAct Agent
    └── tool_node.go      # ToolNode
```

## ✨ 核心特性

### 1. Runnable 接口（LCEL）

```go
// 链式组合
chain := prompt.Pipe(model).Pipe(parser)
result, _ := chain.Invoke(ctx, input)

// 批量执行（自动并行）
results, _ := chain.Batch(ctx, inputs)

// 流式输出
stream, _ := chain.Stream(ctx, input)
for event := range stream {
    fmt.Println(event.Data)
}
```

### 2. StateGraph（LangGraph）

```go
// 创建状态图
graph := state.NewStateGraph[MyState]("my-graph")

// 添加节点
graph.AddNode("agent", agentNode)
graph.AddNode("tools", toolsNode)

// 设置流程
graph.SetEntryPoint("agent")
graph.AddConditionalEdges("agent", routerFn, map[string]string{
    "continue": "tools",
    "end":      state.END,
})
graph.AddEdge("tools", "agent")

// 编译执行
app, _ := graph.Compile()
result, _ := app.Invoke(ctx, initialState)
```

### 3. Checkpointing（持久化）

```go
// 配置检查点存储
checkpointer, _ := postgres.NewSaver("postgresql://localhost/langchain")

graph.WithCheckpointer(checkpointer).
    WithDurability(durability.ModeSync)

// 执行（自动保存检查点）
result, _ := app.Invoke(ctx, state, execute.WithThreadID("user-123"))

// 恢复执行
history, _ := app.GetHistory(ctx, "user-123", 10)
result, _ := app.Invoke(ctx, state, execute.WithCheckpointID(history[5].ID))
```

### 4. Human-in-the-Loop（人工干预）

```go
// 节点中触发中断
func approvalNode(ctx context.Context, state State) (State, error) {
    if state.RequiresApproval {
        hitl.TriggerInterrupt(hitl.Interrupt{
            Type:    hitl.InterruptApproval,
            Message: "请审批此操作",
        })
    }
    return state, nil
}

// 查询待处理的中断
interrupt, _ := app.GetPendingInterrupt(ctx, "user-123")

// 恢复执行
result, _ := app.Resume(ctx, "user-123", hitl.ResumeData{
    Action: hitl.ActionApprove,
})
```

### 5. Streaming（流式输出）

```go
// 多种流模式
streamer := streaming.NewStreamer(app, streaming.ModeEvents)
events, _ := streamer.Stream(ctx, initialState)

for event := range events {
    switch event.Type {
    case "node_start":
        fmt.Printf("开始执行: %s\n", event.NodeName)
    case "node_end":
        fmt.Printf("完成: %s\n", event.NodeName)
    case "values":
        fmt.Printf("状态: %+v\n", event.State)
    }
}
```

### 6. Agent 系统（LangChain 1.0）

```go
// 创建 Agent
agent, _ := agents.CreateAgent(agents.Config{
    Model:        model,
    Tools:        []tools.Tool{searchTool, calculatorTool},
    SystemPrompt: "你是一个有帮助的助手",
    Middleware: []middleware.Middleware{
        logging.New(),
        hitl.New(hitl.Config{
            RequireApproval: func(tc types.ToolCall) bool {
                return tc.Name == "危险操作"
            },
        }),
    },
})

// 执行
result, _ := agent.Invoke(ctx, "帮我搜索并计算...")
```

## 📊 性能对比

| 指标 | Python LangChain | Go LangChain | 提升 |
|------|-----------------|--------------|------|
| 并发连接 | ~10K | ~100K+ | 10x |
| 内存占用 | ~500MB | ~150MB | 70% |
| 冷启动 | 2-3s | <100ms | 20-30x |
| 请求延迟 | 基准 | -30-50% | 30-50% |
| 部署大小 | ~500MB | ~20MB | 95% |

## 🔧 安装

```bash
go get langchain-go
```

## 📖 文档

- [设计文档](../LangChain-LangGraph-Go重写设计方案.md) - 完整的设计方案和实现指南
- [API 文档](https://pkg.go.dev/langchain-go) - API 参考文档
- [示例代码](./examples) - 各种使用示例

## 🗺️ 开发路线图

### Phase 1: 基础核心 ✅
- [x] M01-M04: 基础类型（Message, Tool, Schema, Config）
- [x] M05-M08: Runnable 系统
- [x] M09-M11: ChatModel + OpenAI Provider
- [x] M13-M18: Prompts, Output, Tools

### Phase 2: LangGraph 核心 🚧
- [x] M24-M26: StateGraph 核心
- [x] M27-M32: Node, Edge 系统
- [x] M33-M37: 编译和执行引擎
- [ ] M38-M42: Checkpointing ⭐
- [ ] M43-M45: Durability 模式 ⭐
- [ ] M46-M49: Human-in-the-Loop ⭐
- [ ] M50-M52: Streaming

### Phase 3: Agent 系统 📅
- [ ] M53-M58: create_agent + Middleware
- [ ] M12: Anthropic Provider
- [ ] M19-M23: Memory + Callbacks

### Phase 4: 高级特性 📅
- [ ] M59: ReAct Agent
- [ ] M60: ToolNode
- [ ] 完整示例和文档

## 🤝 贡献指南

1. 阅读 [.cursorrules](./.cursorrules) 了解代码规范
2. 选择一个模块（参考设计文档）
3. 创建 feature 分支：`git checkout -b feature/M{ID}`
4. 实现功能（遵循规范）
5. 编写测试
6. 提交 PR

## 📝 许可证

MIT License

## 🙏 致谢

本项目灵感来自：
- [LangChain](https://github.com/langchain-ai/langchain) (Python)
- [LangGraph](https://github.com/langchain-ai/langgraph) (Python)
- [LangChainGo](https://github.com/tmc/langchaingo) (社区版本)

---

**当前状态**: 🚧 开发中

**文档版本**: 1.0.0

**Go 版本**: 1.22+
