# A2A Collaboration Demo

这个示例演示 A2A 协议的高级协作功能：
- 智能任务路由
- 多 Agent 协作
- 复杂任务分解
- 结果聚合

## 功能演示

### 1. 智能任务路由

基于 Agent 能力、负载、性能等多个维度，智能选择最合适的 Agent。

**路由策略**:
- **Capability** - 能力匹配
- **Load** - 负载均衡
- **Performance** - 性能优先
- **Hybrid** - 混合策略（推荐）

### 2. 多 Agent 协作

协作协调器自动：
1. 分解复杂任务为子任务
2. 为每个子任务路由合适的 Agent
3. 并行执行子任务
4. 聚合结果生成最终输出

### 3. 专家 Agent 系统

- **Researcher** - 研究和信息收集
- **Analyst** - 数据分析和洞察
- **Writer** - 内容创作和报告撰写

## 运行示例

```bash
cd examples/a2a_collaboration_demo
go run main.go
```

## 预期输出

```
=== A2A Collaboration Demo ===

Step 1: Setting up agent registry and router...
✓ Registry and router created

Step 2: Registering specialized agents...
✓ Registered: Researcher Agent
✓ Registered: Analyst Agent
✓ Registered: Writer Agent

Step 3: Testing simple task routing...
Task routed to: Researcher Agent
Status: completed
Result: [Researcher] Completed research on...

Step 4: Testing complex task coordination...
Complex task: Research AI trends in 2026, analyze the data, and write a comprehensive report
Decomposing and coordinating...

✓ Coordination completed in 650ms

=== Final Result ===
[Subtask <uuid>]: [Researcher] Completed research...

[Subtask <uuid>]: [Analyst] Analyzed...

[Subtask <uuid>]: [Writer] Written report...

=== Collaboration Session Details ===
Session ID: <uuid>
Status: completed
Participants: 3 agent(s)
Subtasks: 3
Created: 2026-01-22T...
Duration: 650ms

Participating Agents:
  - agent-researcher
  - agent-analyst
  - agent-writer

=== Demo Completed ===
```

## 核心概念

### 智能路由

```go
// 创建路由器
router := a2a.NewSmartTaskRouter(registry, a2a.RouterConfig{
    Strategy: a2a.StrategyHybrid,
    Scorer: &a2a.AgentScorer{
        Weights: &a2a.ScoringWeights{
            CapabilityMatch: 0.4,
            Load:            0.3,
            Performance:     0.2,
            Reputation:      0.1,
        },
    },
})

// 路由任务
agent, _ := router.Route(ctx, task)
```

### 协作协调

```go
// 创建协调器
coordinator := a2a.NewCollaborationCoordinator(registry, router)

// 提交复杂任务
complexTask := &a2a.Task{
    Type: a2a.TaskTypeComplex,
    Input: &a2a.TaskInput{
        Content: "Multi-step task...",
    },
}

// 自动协调完成
result, _ := coordinator.Coordinate(ctx, complexTask)
```

## 工作流程

```
复杂任务
    │
    ▼
协调器分解
    │
    ├─► 子任务 1 → 路由器 → Researcher → 结果 1
    │
    ├─► 子任务 2 → 路由器 → Analyst   → 结果 2
    │
    └─► 子任务 3 → 路由器 → Writer    → 结果 3
    │
    ▼
聚合结果 → 最终输出
```

## 性能特点

- **并行执行**: 子任务并行处理，提升效率
- **智能路由**: 自动选择最合适的 Agent
- **负载均衡**: 考虑 Agent 当前负载
- **容错处理**: 单个子任务失败不影响整体

## 扩展建议

### 1. 使用分布式注册中心

```go
// 替换为 Consul 注册中心
import "github.com/hashicorp/consul/api"

consulConfig := &api.Config{
    Address: "localhost:8500",
}

registry := a2a.NewConsulRegistry(consulConfig)
```

### 2. 自定义评分权重

```go
// 根据业务需求调整权重
weights := &a2a.ScoringWeights{
    CapabilityMatch: 0.6,  // 更注重能力匹配
    Load:            0.2,
    Performance:     0.15,
    Reputation:      0.05,
}

router := a2a.NewSmartTaskRouter(registry, a2a.RouterConfig{
    Strategy: a2a.StrategyHybrid,
    Scorer:   &a2a.AgentScorer{Weights: weights},
})
```

### 3. 监控指标

```go
// 获取 Agent 性能指标
metrics := router.GetMetrics("agent-researcher")
fmt.Printf("Success Rate: %.2f%%\n", metrics.SuccessRate*100)
fmt.Printf("Avg Response Time: %.2fs\n", metrics.AvgResponseTime)
fmt.Printf("Current Load: %d\n", metrics.CurrentLoad)
```

## 更多资源

- [A2A 规范](../../docs/V0.6.1_A2A_SPEC.md)
- [用户指南](../../docs/V0.6.1_USER_GUIDE.md)
- [基础示例](../a2a_basic_demo/)
- [分布式示例](../a2a_distributed_demo/)

---

**创建日期**: 2026-01-22  
**版本**: v0.6.1
