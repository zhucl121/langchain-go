# 统一 Agent 创建接口演示

🌍 **Language**: 中文 | [English](#english)

## 功能演示

本示例展示 LangChain-Go v0.7.0 新增的**统一 Agent 创建接口**，对标 LangChain 1.0 的 `create_agent` 抽象。

核心优势：**一套 API，五种策略**，修改一个字段即可切换 Agent 类型。

## 运行示例

```bash
cd examples/create_agent_v2_demo
go run main.go
```

## 核心概念

### Agent 预设类型对比

| 预设 | 适用场景 | 底层实现 |
|------|----------|----------|
| `PresetReAct` | 通用推理，逐步思考 | ReActAgent |
| `PresetToolCalling` | 工具密集型任务，原生函数调用 | OpenAIFunctionsAgent |
| `PresetPlanExecute` | 复杂多步任务，先规划后执行 | PlanAndExecuteAgent |
| `PresetSelfAsk` | 复合查询，递归分解问题 | SelfAskAgent |
| `PresetConversationalV2` | 多轮对话，带记忆管理 | ReActAgent + Memory |

### 使用示例

**最简用法**

```go
ca, err := agents.CreateUnifiedAgent(agents.UnifiedAgentConfig{
    Preset: agents.PresetReAct,   // 一行切换策略
    Model:  llm,
    Tools:  myTools,
})
result, _ := ca.Run(ctx, "帮我搜索 Go 最新版本")
fmt.Println(result.Output)
```

**函数式 API（链式选项）**

```go
ca, err := agents.NewReActAgentV2(llm, tools,
    agents.WithUnifiedSystemPrompt("你是一个 Go 专家"),
    agents.WithUnifiedMaxSteps(20),
    agents.WithUnifiedVerbose(true),
)
```

**带记忆的多轮对话**

```go
memory := agents.NewInMemoryConversationMemory()

ca, _ := agents.NewMemoryAgentV2(llm, nil,
    agents.WithUnifiedMemory(memory),
)

// 第一轮：保存对话到记忆
result1, _ := ca.RunWithMemory(ctx, "Go 有什么特点？")

// 第二轮：自动感知历史上下文
result2, _ := ca.RunWithMemory(ctx, "继续详细解释")
```

**带 Pre/Post Model Hooks**

```go
// 创建 Hook 链（PII 脱敏 + 内容过滤）
hookChain := middleware.NewHookChain(
    middleware.NewPIIRedactHook(middleware.DefaultPIIConfig()),
    middleware.NewGuardrailHook(middleware.GuardrailConfig{...}),
)

ca, _ := agents.CreateUnifiedAgent(agents.UnifiedAgentConfig{
    Preset:     agents.PresetToolCalling,
    Model:      llm,
    ModelHooks: hookChain,   // 在每次 LLM 调用前后自动执行
})
```

**带弹性配置（Circuit Breaker）**

```go
ca, _ := agents.CreateUnifiedAgent(agents.UnifiedAgentConfig{
    Preset: agents.PresetReAct,
    Model:  llm,
    Resilience: &agents.ResilienceConfig{
        CircuitBreaker: agents.DefaultCircuitBreakerConfig("llm-api"),
        Bulkhead: agents.BulkheadConfig{
            Name:          "llm-api",
            MaxConcurrent: 10,
            MaxWaitTime:   5 * time.Second,
        },
    },
})
```

### 与旧版 API 对比

```go
// ❌ 旧版：每种 Agent 有独立的构造函数和配置
agent1 := agents.CreateReActAgent(llm, tools, agents.WithMaxSteps(10))
agent2 := agents.CreateToolCallingAgent(llm, tools, agents.WithSystemPrompt("..."))
agent3 := agents.NewPlanAndExecuteAgent(agents.PlanAndExecuteConfig{LLM: llm, ...})

// ✅ v0.7.0：统一入口，只改 Preset
ca, _ := agents.CreateUnifiedAgent(agents.UnifiedAgentConfig{
    Preset: agents.PresetReAct,       // 改这一行即可切换策略
    Model:  llm,
    Tools:  tools,
})
```

## 相关文档

- 📚 [v0.7.0 用户指南](../../docs/V0.7.0_USER_GUIDE.md)
- 📋 [设计方案](../../docs/V0.7.0_DESIGN_PROPOSAL.md)
- 🔗 [create_agent.go](../../core/agents/create_agent.go)
- 🔗 [Pre/Post Hooks](../../core/middleware/hooks.go)
- 🔗 [Circuit Breaker](../../core/agents/circuit_breaker.go)

---

## English

### Unified Agent Creation API Demo

This example demonstrates the **Unified Agent API** introduced in LangChain-Go v0.7.0, equivalent to LangChain 1.0's `create_agent` abstraction.

**One API, Five Strategies** — change a single `Preset` field to switch agent behavior.

### Run

```bash
cd examples/create_agent_v2_demo
go run main.go
```

### Quick Reference

```go
// Minimal usage
ca, _ := agents.CreateUnifiedAgent(agents.UnifiedAgentConfig{
    Preset: agents.PresetToolCalling,  // switch: PresetReAct / PresetPlanExecute / PresetSelfAsk
    Model:  llm,
    Tools:  myTools,
})
result, _ := ca.Run(ctx, "your question")

// Multi-turn with memory
memory := agents.NewInMemoryConversationMemory()
ca, _ := agents.NewMemoryAgentV2(llm, nil, agents.WithUnifiedMemory(memory))
result, _ := ca.RunWithMemory(ctx, "follow-up question")

// Functional options style
ca, _ := agents.NewReActAgentV2(llm, tools,
    agents.WithUnifiedSystemPrompt("You are an expert"),
    agents.WithUnifiedMaxSteps(20),
)
```
