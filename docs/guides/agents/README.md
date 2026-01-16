# Agent 系统指南

智能 Agent 的构建和使用指南。

---

## 📖 指南列表

- [Agent 概述](./overview.md) - Agent 系统介绍和架构
- [Plan-Execute Agent](./plan-execute.md) - 计划执行 Agent
- ReAct Agent - 推理和行动 Agent（即将添加）
- 自定义 Agent - 创建自定义 Agent（即将添加）

---

## 🎯 什么是 Agent？

Agent 是一个可以使用工具、推理和采取行动的智能系统。LangChain-Go 提供了多种 Agent 类型：

### Agent 类型

1. **ReAct Agent** - 推理和行动
   - 思考下一步要做什么
   - 选择合适的工具
   - 根据结果继续推理

2. **ToolCalling Agent** - 工具调用
   - 使用 LLM 的原生 Tool Calling 功能
   - 更高效的工具使用
   - 支持并行工具调用

3. **Conversational Agent** - 对话型
   - 保持对话上下文
   - 自然的对话交互
   - 记忆管理

4. **Plan-Execute Agent** - 计划执行
   - 将复杂任务分解为步骤
   - 制定执行计划
   - 逐步执行并追踪进度

---

## 🚀 快速示例

### 基础 Agent

```go
import "github.com/zhuchenglong/langchain-go/core/agents"

// 创建 Agent
agent, _ := agents.CreateAgent(agents.Config{
    Model: model,
    Tools: []tools.Tool{
        searchTool,
        calculatorTool,
    },
    SystemPrompt: "你是一个有帮助的助手",
})

// 执行任务
result, _ := agent.Invoke(ctx, "帮我搜索...")
```

### Plan-Execute Agent

```go
import "github.com/zhuchenglong/langchain-go/core/agents/planexecute"

// 创建 Plan-Execute Agent
agent, _ := planexecute.NewPlanExecuteAgent(planexecute.Config{
    Planner:  llm,
    Tools:    tools,
    Executor: executor,
})

// 执行复杂任务
result, _ := agent.Invoke(ctx, "分析市场趋势并生成报告")
```

---

## 💡 选择合适的 Agent

| Agent 类型 | 适用场景 | 优势 | 劣势 |
|-----------|---------|------|------|
| ReAct | 需要推理的任务 | 透明的思考过程 | Token 消耗较多 |
| ToolCalling | 多工具协作 | 高效、原生支持 | 需要模型支持 |
| Conversational | 对话应用 | 自然交互 | 需要记忆管理 |
| Plan-Execute | 复杂多步骤任务 | 清晰的计划 | 执行时间较长 |

---

## 🔧 Agent 组件

### 1. Tools（工具）
Agent 可以调用的工具：

```go
searchTool := search.NewDuckDuckGoSearchTool(search.DuckDuckGoConfig{
    MaxResults: 5,
})

calcTool := tools.NewCalculatorTool()
```

### 2. Memory（记忆）
保存对话历史：

```go
memory := memory.NewBufferMemory()
agent.WithMemory(memory)
```

### 3. Middleware（中间件）
添加额外功能：

```go
agent.WithMiddleware(
    logging.New(),
    metrics.New(),
    hitl.New(hitlConfig),
)
```

---

## 📚 相关资源

- [快速开始](../../getting-started/) - 新手入门
- [核心功能指南](../core/) - 核心组件
- [Tools 工具指南](../core/tools.md) - 工具系统
- [示例代码](../../examples/) - Agent 示例

---

<div align="center">

**[⬆ 回到指南首页](../README.md)** | **[回到文档首页](../../README.md)**

</div>
