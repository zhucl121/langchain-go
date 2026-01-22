# Protocol Bridge Demo

这个示例演示 MCP 和 A2A 协议之间的双向桥接：
- MCP 工具调用 → A2A 任务
- A2A Agent → MCP 资源
- 协议互操作

## 功能演示

### 1. MCP → A2A 桥接

MCP 客户端（如 Claude Desktop）的工具调用自动转换为 A2A 任务并路由到合适的 Agent。

```
Claude Desktop (MCP Client)
        │
        │ MCP Tool Call
        ▼
   MCP Server
        │
        │ Bridge
        ▼
    A2A Router → A2A Agent → Result
```

### 2. A2A → MCP 桥接

A2A Agents 作为 MCP 资源暴露给 MCP 客户端。

```
A2A Agents
    │
    │ Expose as Resources
    ▼
MCP Server → MCP Client (Claude Desktop)
```

### 3. 双向互操作

统一 MCP 和 A2A 两个生态系统，实现无缝互操作。

## 运行示例

```bash
cd examples/protocol_bridge_demo
go run main.go
```

## 预期输出

```
=== Protocol Bridge Demo ===

Step 1: Setting up A2A agent system...
✓ A2A agents registered

Step 2: Setting up MCP server...
✓ MCP server created

Step 3: Creating bidirectional bridge...
✓ Bridge setup completed

Step 4: Testing MCP → A2A bridging...
Simulating MCP tool call...
✓ MCP tool call converted to A2A task
  Task ID: <uuid>
  Task Type: execute
  Routed to: Researcher Agent
  Result: [Researcher] Research completed...

Step 5: Testing A2A → MCP bridging...
✓ MCP resources (including A2A agents): 2
  - Researcher Agent (URI: a2a://agent/agent-researcher)
    Agent ID: agent-researcher
    Capabilities: [research search]
  - Analyst Agent (URI: a2a://agent/agent-analyst)
    Agent ID: agent-analyst
    Capabilities: [analysis statistics]

Step 6: Reading A2A agent as MCP resource...
✓ Resource content:
{
  "id": "agent-researcher",
  "name": "Researcher Agent",
  "type": "specialist",
  "status": "online",
  "capabilities": [research search]
}

Step 7: Performance test...
✓ Processed 10 tasks in 1.2s
  Average: 120.00ms per task

=== Demo Completed ===
```

## 核心概念

### MCP → A2A 转换

```go
// 创建桥接
bridge := bridge.NewMCPToA2ABridge(mcpServer, router, registry)

// MCP 工具调用
toolName := "analyze"
toolArgs := map[string]any{"data": "..."}

// 转换为 A2A 任务
task := bridge.ToolCallToTask(toolName, toolArgs)

// 路由和执行
agent, _ := router.Route(ctx, task)
response, _ := agent.SendTask(ctx, task)

// 转换回 MCP 结果
mcpResult := bridge.TaskResponseToToolResult(response)
```

### A2A → MCP 转换

```go
// 创建桥接
bridge := bridge.NewA2AToMCPBridge(registry, mcpServer)

// 暴露 A2A agents 为 MCP 资源
resources, _ := bridge.ExposeAgentsAsResources(ctx)

// 在 MCP Server 中注册
for _, resource := range resources {
    agentID := resource.Metadata["agentId"].(string)
    agent, _ := registry.FindByID(ctx, agentID)
    provider := bridge.CreateAgentResourceProvider(agent)
    mcpServer.RegisterResource(resource, provider)
}
```

### 双向桥接

```go
// 创建双向桥接
bridge := bridge.NewBidirectionalBridge(mcpServer, router, registry)

// 自动设置
bridge.Setup(ctx)

// 现在 MCP 和 A2A 可以互操作了！
```

## 使用场景

### 场景 1: Claude Desktop 使用分布式 Agent

```
用户 → Claude Desktop → MCP Server → Bridge → A2A Router → 最佳 Agent
```

Claude Desktop 用户可以透明地使用分布式部署的 A2A Agents，享受智能路由和负载均衡的好处。

### 场景 2: A2A Agent 访问 MCP 资源

```
A2A Agent → MCP Client → MCP Server → Resource Provider → 数据/工具
```

A2A Agents 可以访问 MCP 生态中的资源和工具。

### 场景 3: 统一 AI 生态

```
MCP Tools ←→ Bridge ←→ A2A Agents
```

打通 MCP 和 A2A 两个生态系统，实现真正的互操作性。

## 性能特点

- **低延迟**: 协议转换 < 2ms
- **零拷贝**: 高效的消息转换
- **透明**: 对客户端和 Agent 透明
- **可靠**: 完整的错误处理

## 扩展建议

### 1. 与现有 Agent 系统集成

```go
// 将现有 LangChain-Go Agent 桥接到 A2A
import "github.com/zhucl121/langchain-go/core/agents"

existingAgent := agents.CreateReActAgent(llm, tools)

// 适配器
adapter := &AgentAdapter{agent: existingAgent}
a2aAgent := a2a.NewA2AAgentBridge(adapter, config)

// 注册
registry.Register(ctx, a2aAgent)

// 现在可以通过 MCP 使用了！
```

### 2. 添加更多协议

桥接设计支持扩展到更多协议：
- MCP ↔ A2A（已实现）
- OpenAI Assistants API ↔ A2A
- LangChain Expression Language ↔ A2A
- 自定义协议 ↔ A2A

## 更多资源

- [MCP 规范](../../docs/V0.6.1_MCP_SPEC.md)
- [A2A 规范](../../docs/V0.6.1_A2A_SPEC.md)
- [集成指南](../../docs/V0.6.1_INTEGRATION_GUIDE.md)
- [用户指南](../../docs/V0.6.1_USER_GUIDE.md)

---

**创建日期**: 2026-01-22  
**版本**: v0.6.1
