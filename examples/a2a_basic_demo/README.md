# A2A Basic Demo

这个示例演示 A2A (Agent-to-Agent) 协议的基础功能：
- Agent 注册和发现
- 能力匹配
- 任务发送和执行
- 健康检查

## 功能演示

### 1. Agent 注册

注册 3 个专家 Agent：
- **Research Agent** - 研究和信息收集
- **Analysis Agent** - 数据分析
- **Writing Agent** - 内容写作

### 2. Agent 发现

通过能力标签发现 Agent：
- 查找具有 "research" 能力的 Agent
- 查找具有 "analysis" 能力的 Agent

### 3. 任务执行

发送任务到特定 Agent 并获取结果。

### 4. 健康检查

检查 Agent 的健康状态和运行时间。

## 运行示例

```bash
cd examples/a2a_basic_demo
go run main.go
```

## 预期输出

```
=== A2A Basic Demo ===

Step 1: Creating local agent registry...
Step 2: Creating and registering agents...

✓ Registered: Research Agent (ID: agent-research)
✓ Registered: Analysis Agent (ID: agent-analysis)
✓ Registered: Writing Agent (ID: agent-writing)

Step 3: Listing all registered agents...
Total agents: 3

- Research Agent (ID: agent-research)
  Type: specialist, Status: online
  Capabilities: [research search analysis]

- Analysis Agent (ID: agent-analysis)
  Type: specialist, Status: online
  Capabilities: [analysis statistics visualization]

- Writing Agent (ID: agent-writing)
  Type: specialist, Status: online
  Capabilities: [writing editing summarization]

Step 4: Discovering agents by capability...
Found 1 agent(s) with 'research' capability
Found 1 agent(s) with 'analysis' capability

Step 5: Sending task to Research Agent...
Task ID: <uuid>
Task Type: query
Task Content: Research the latest AI developments in 2026

Task Status: completed
Task Progress: 100%
Result: [Research Agent] Researched: '...'. Found relevant information...

Step 6: Sending task to Analysis Agent...
Task Status: completed
Result: [Analysis Agent] Analyzed: '...'. Key insights: ...

Step 7: Checking agent health...
Agent ID: agent-research
Status: healthy
Last Heartbeat: 2026-01-22T...
Uptime: 0s

=== Demo Completed ===
```

## 核心概念

### Agent 注册

```go
// 桥接现有 Agent
a2aAgent := a2a.NewA2AAgentBridge(myAgent, &a2a.BridgeConfig{
    Info: &a2a.AgentInfo{
        ID:   "agent-1",
        Name: "My Agent",
    },
    Capabilities: &a2a.AgentCapabilities{
        Capabilities: []string{"research", "analysis"},
    },
})

// 注册到本地注册中心
registry := a2a.NewLocalRegistry()
registry.Register(ctx, a2aAgent)
```

### Agent 发现

```go
// 按 ID 查找
agent, _ := registry.FindByID(ctx, "agent-1")

// 按能力查找
agents, _ := registry.FindByCapability(ctx, "research")

// 列出所有
allAgents, _ := registry.ListAll(ctx)
```

### 任务执行

```go
// 创建任务
task := &a2a.Task{
    ID:   uuid.New().String(),
    Type: a2a.TaskTypeQuery,
    Input: &a2a.TaskInput{
        Type:    "text",
        Content: "Research AI developments",
    },
}

// 发送任务
response, _ := agent.SendTask(ctx, task)
fmt.Printf("Status: %s\n", response.Status)
fmt.Printf("Result: %s\n", response.Result.Content)
```

## 扩展示例

### 使用 Consul 注册中心

```go
// 需要先启动 Consul
// docker run -d -p 8500:8500 consul

import "github.com/hashicorp/consul/api"

consulConfig := &api.Config{
    Address: "localhost:8500",
}

registry := a2a.NewConsulRegistry(consulConfig)
registry.Register(ctx, a2aAgent)
```

### 分布式部署

参见 `examples/a2a_distributed_demo/` 了解如何部署分布式 Agent 系统。

## 更多资源

- [A2A 规范](../../docs/V0.6.1_A2A_SPEC.md)
- [用户指南](../../docs/V0.6.1_USER_GUIDE.md)
- [协作示例](../a2a_collaboration_demo/)

---

**创建日期**: 2026-01-22  
**版本**: v0.6.1
