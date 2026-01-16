# 🤝 Multi-Agent 系统使用指南

## 📚 目录

1. [快速开始](#快速开始)
2. [核心概念](#核心概念)
3. [系统架构](#系统架构)
4. [创建 Agent](#创建-agent)
5. [协调策略](#协调策略)
6. [实战案例](#实战案例)
7. [最佳实践](#最佳实践)
8. [性能优化](#性能优化)
9. [故障排查](#故障排查)

---

## 快速开始

### 最简单的例子

```go
package main

import (
    "context"
    "fmt"
    "langchain-go/core/agents"
    "langchain-go/core/chat/ollama"
    "langchain-go/core/tools"
)

func main() {
    ctx := context.Background()
    llm := ollama.NewChatOllama("qwen2.5:7b")
    
    // 1. 创建协调策略和协调器
    strategy := agents.NewSequentialStrategy(llm)
    coordinator := agents.NewCoordinatorAgent("coordinator", llm, strategy)
    
    // 2. 创建 Multi-Agent 系统
    system := agents.NewMultiAgentSystem(coordinator, nil)
    
    // 3. 添加专用 Agent
    researcher := agents.NewResearcherAgent("researcher", llm, tools.NewDuckDuckGoSearch())
    system.AddAgent("researcher", researcher)
    coordinator.RegisterAgent(researcher)
    
    writer := agents.NewWriterAgent("writer", llm, "technical")
    system.AddAgent("writer", writer)
    coordinator.RegisterAgent(writer)
    
    // 4. 执行任务
    result, _ := system.Run(ctx, "Research AI trends and write a summary")
    
    fmt.Println(result.FinalResult)
}
```

---

## 核心概念

### 1. Multi-Agent 系统

Multi-Agent 系统是多个智能 Agent 协作完成复杂任务的框架。

**核心组件**:
- **Coordinator**: 协调器，负责任务分配和结果聚合
- **Specialized Agents**: 专用 Agent，各有专长
- **Message Bus**: 消息总线，负责 Agent 间通信
- **Shared State**: 共享状态，Agent 间共享数据
- **Execution History**: 执行历史，记录系统运行轨迹

### 2. Agent 类型

#### 内置 Agent

| Agent 类型 | 能力 | 适用场景 |
|-----------|------|---------|
| **CoordinatorAgent** | 协调、任务分解、结果聚合 | 系统核心 |
| **ResearcherAgent** | 研究、搜索、信息收集 | 资料收集 |
| **WriterAgent** | 写作、编辑、内容创作 | 内容生成 |
| **ReviewerAgent** | 审核、评估、质量检查 | 质量保证 |
| **AnalystAgent** | 分析、数据处理、洞察 | 数据分析 |
| **PlannerAgent** | 规划、策略、任务分解 | 任务规划 |

#### 自定义 Agent

```go
type CustomAgent struct {
    agents.BaseMultiAgent
    // 自定义字段
    domain string
}

func (ca *CustomAgent) ReceiveMessage(ctx context.Context, msg *agents.AgentMessage) error {
    // 实现消息处理逻辑
    return nil
}

func (ca *CustomAgent) CanHandle(task string) (bool, float64) {
    // 实现任务匹配逻辑
    return true, 0.8
}
```

### 3. 消息类型

```go
MessageTypeRequest    // 请求
MessageTypeResponse   // 响应
MessageTypeTask       // 任务分配
MessageTypeResult     // 任务结果
MessageTypeQuery      // 查询
MessageTypeBroadcast  // 广播
MessageTypeError      // 错误
MessageTypeAck        // 确认
```

---

## 系统架构

### 架构图

```
┌─────────────────────────────────────────┐
│        Multi-Agent System               │
├─────────────────────────────────────────┤
│                                          │
│  Coordinator Agent                      │
│       ↓                                  │
│  Message Bus (Router)                   │
│       ↓                                  │
│  ┌──────┬──────┬──────┬──────┐         │
│  │Agent1│Agent2│Agent3│Agent4│         │
│  └──────┴──────┴──────┴──────┘         │
│       ↓                                  │
│  Shared State & History                 │
└─────────────────────────────────────────┘
```

### 工作流程

1. **任务接收**: 系统接收用户任务
2. **任务分解**: Coordinator 将任务分解为子任务
3. **Agent 选择**: 为每个子任务选择合适的 Agent
4. **并行执行**: Agent 并行处理各自的子任务
5. **结果聚合**: Coordinator 聚合所有结果
6. **返回结果**: 系统返回最终结果

---

## 创建 Agent

### 使用内置 Agent

```go
// Researcher Agent - 研究和搜索
researcher := agents.NewResearcherAgent(
    "researcher",
    llm,
    tools.NewDuckDuckGoSearch(),
)

// Writer Agent - 内容创作
writer := agents.NewWriterAgent(
    "writer",
    llm,
    "creative", // 写作风格: technical, creative, formal
)

// Reviewer Agent - 质量审核
reviewer := agents.NewReviewerAgent(
    "reviewer",
    llm,
    []string{"accuracy", "clarity", "grammar"},
)

// Analyst Agent - 数据分析
analyst := agents.NewAnalystAgent("analyst", llm)

// Planner Agent - 任务规划
planner := agents.NewPlannerAgent("planner", llm)
```

### 创建自定义 Agent

```go
type DataScientistAgent struct {
    agents.BaseMultiAgent
    tools []tools.Tool
}

func NewDataScientistAgent(id string, llm chat.ChatModel) *DataScientistAgent {
    return &DataScientistAgent{
        BaseMultiAgent: agents.BaseMultiAgent{
            ID:           id,
            LLM:          llm,
            Capabilities: []string{"data_science", "ml", "statistics"},
        },
    }
}

func (dsa *DataScientistAgent) ReceiveMessage(ctx context.Context, msg *agents.AgentMessage) error {
    if msg.Type != agents.MessageTypeTask {
        return nil
    }
    
    // 处理数据科学任务
    result, err := dsa.processTask(ctx, msg.Content)
    if err != nil {
        return dsa.SendError(ctx, msg, err)
    }
    
    return dsa.SendResult(ctx, msg, result)
}

func (dsa *DataScientistAgent) processTask(ctx context.Context, task string) (string, error) {
    // 实现数据科学处理逻辑
    prompt := fmt.Sprintf("As a data scientist, analyze: %s", task)
    messages := []chat.Message{chat.NewHumanMessage(prompt)}
    response, err := dsa.LLM.Generate(ctx, messages)
    if err != nil {
        return "", err
    }
    return response.Content, nil
}

func (dsa *DataScientistAgent) CanHandle(task string) (bool, float64) {
    keywords := []string{"data", "ml", "model", "train", "predict"}
    taskLower := strings.ToLower(task)
    
    for _, keyword := range keywords {
        if strings.Contains(taskLower, keyword) {
            return true, 0.9
        }
    }
    return false, 0.0
}
```

---

## 协调策略

### Sequential Strategy (顺序执行)

```go
strategy := agents.NewSequentialStrategy(llm)
```

**特点**:
- 按顺序执行子任务
- 简单可靠
- 适合有依赖关系的任务

**使用场景**:
- 内容创作流水线 (研究 → 写作 → 审核)
- 数据处理管道 (收集 → 清洗 → 分析)

### Parallel Strategy (并行执行)

```go
strategy := agents.NewParallelStrategy(llm, maxConcurrency)
```

**特点**:
- 并行执行子任务
- 高效快速
- 适合独立任务

**使用场景**:
- 多源数据收集
- 批量内容生成

### Hierarchical Strategy (层次化执行)

```go
strategy := agents.NewHierarchicalStrategy(llm)
```

**特点**:
- 层次化任务分配
- 支持复杂依赖
- 适合大型项目

**使用场景**:
- 软件开发项目
- 复杂研究任务

---

## 实战案例

### 案例 1: 内容创作流水线

```go
func ContentCreationPipeline() {
    ctx := context.Background()
    llm := ollama.NewChatOllama("qwen2.5:7b")
    
    strategy := agents.NewSequentialStrategy(llm)
    coordinator := agents.NewCoordinatorAgent("coordinator", llm, strategy)
    
    system := agents.NewMultiAgentSystem(coordinator, nil)
    
    // 组建团队
    planner := agents.NewPlannerAgent("planner", llm)
    system.AddAgent("planner", planner)
    coordinator.RegisterAgent(planner)
    
    researcher := agents.NewResearcherAgent("researcher", llm, tools.NewDuckDuckGoSearch())
    system.AddAgent("researcher", researcher)
    coordinator.RegisterAgent(researcher)
    
    writer := agents.NewWriterAgent("writer", llm, "creative")
    system.AddAgent("writer", writer)
    coordinator.RegisterAgent(writer)
    
    reviewer := agents.NewReviewerAgent("reviewer", llm, 
        []string{"grammar", "clarity", "engagement"})
    system.AddAgent("reviewer", reviewer)
    coordinator.RegisterAgent(reviewer)
    
    // 执行任务
    task := "Create a blog post about sustainable technology"
    result, _ := system.Run(ctx, task)
    
    fmt.Println(result.FinalResult)
}
```

**流程**:
1. Planner 制定内容计划
2. Researcher 收集相关资料
3. Writer 撰写文章
4. Reviewer 审核并提出修改建议
5. Coordinator 聚合最终版本

### 案例 2: 数据分析管道

```go
func DataAnalysisPipeline() {
    ctx := context.Background()
    llm := ollama.NewChatOllama("qwen2.5:7b")
    
    strategy := agents.NewSequentialStrategy(llm)
    coordinator := agents.NewCoordinatorAgent("coordinator", llm, strategy)
    
    system := agents.NewMultiAgentSystem(coordinator, nil)
    
    // 组建分析团队
    collector := agents.NewResearcherAgent("collector", llm, nil)
    system.AddAgent("collector", collector)
    coordinator.RegisterAgent(collector)
    
    analyst := agents.NewAnalystAgent("analyst", llm)
    system.AddAgent("analyst", analyst)
    coordinator.RegisterAgent(analyst)
    
    writer := agents.NewWriterAgent("writer", llm, "technical")
    system.AddAgent("writer", writer)
    coordinator.RegisterAgent(writer)
    
    // 执行分析
    task := "Analyze market trends for electric vehicles in 2024"
    result, _ := system.Run(ctx, task)
    
    fmt.Println(result.FinalResult)
}
```

### 案例 3: 客户支持系统

```go
func CustomerSupportSystem() {
    ctx := context.Background()
    llm := ollama.NewChatOllama("qwen2.5:7b")
    
    strategy := agents.NewSequentialStrategy(llm)
    coordinator := agents.NewCoordinatorAgent("coordinator", llm, strategy)
    
    system := agents.NewMultiAgentSystem(coordinator, nil)
    
    // 专业客服 Agent
    techSupport := agents.NewResearcherAgent("tech_support", llm, nil)
    system.AddAgent("tech_support", techSupport)
    coordinator.RegisterAgent(techSupport)
    
    billing := agents.NewAnalystAgent("billing", llm)
    system.AddAgent("billing", billing)
    coordinator.RegisterAgent(billing)
    
    // 处理客户问题
    question := "Why was I charged twice this month?"
    result, _ := system.Run(ctx, question)
    
    fmt.Println("Response:", result.FinalResult)
}
```

### 案例 4: 软件开发助手

```go
type CodeReviewAgent struct {
    agents.BaseMultiAgent
}

func SoftwareDevelopmentAssistant() {
    ctx := context.Background()
    llm := ollama.NewChatOllama("qwen2.5:7b")
    
    strategy := agents.NewSequentialStrategy(llm)
    coordinator := agents.NewCoordinatorAgent("coordinator", llm, strategy)
    
    system := agents.NewMultiAgentSystem(coordinator, nil)
    
    // 开发团队
    architect := agents.NewPlannerAgent("architect", llm)
    system.AddAgent("architect", architect)
    coordinator.RegisterAgent(architect)
    
    developer := agents.NewWriterAgent("developer", llm, "technical")
    system.AddAgent("developer", developer)
    coordinator.RegisterAgent(developer)
    
    reviewer := agents.NewReviewerAgent("code_reviewer", llm,
        []string{"code_quality", "security", "performance"})
    system.AddAgent("code_reviewer", reviewer)
    coordinator.RegisterAgent(reviewer)
    
    // 开发任务
    task := "Design and implement a user authentication system"
    result, _ := system.Run(ctx, task)
    
    fmt.Println(result.FinalResult)
}
```

---

## 最佳实践

### 1. Agent 职责分离

**✅ 好的做法**:
```go
// 每个 Agent 专注于单一职责
researcher := agents.NewResearcherAgent("researcher", llm, searchTool)
writer := agents.NewWriterAgent("writer", llm, "technical")
reviewer := agents.NewReviewerAgent("reviewer", llm, criteria)
```

**❌ 不好的做法**:
```go
// Agent 职责过多，难以维护
multiPurposeAgent := NewAgent("multi", llm) // 既研究又写作又审核
```

### 2. 合理设置超时

```go
config := &agents.MultiAgentConfig{
    MessageTimeout: 30 * time.Second,  // 消息超时
    TaskTimeout:    5 * time.Minute,   // 任务超时
    MaxRetries:     3,                 // 最大重试
}
```

### 3. 使用共享状态

```go
// 在 Agent 间共享数据
system.GetSharedState().Set("research_data", data)

// 其他 Agent 可以访问
data, _ := system.GetSharedState().Get("research_data")
```

### 4. 监控和日志

```go
// 获取系统指标
metrics := system.GetMetrics()
stats := metrics.GetStats()

fmt.Printf("成功率: %.1f%%\n", stats["success_rate"])
fmt.Printf("平均时间: %v\n", stats["average_time"])

// 查看执行历史
history := system.GetHistory()
records := history.GetAllRecords()
```

### 5. 错误处理

```go
result, err := system.Run(ctx, task)
if err != nil {
    log.Printf("任务失败: %v", err)
    
    // 检查是否超时
    if errors.Is(err, context.DeadlineExceeded) {
        log.Println("任务超时，请增加 TaskTimeout")
    }
    
    return
}
```

---

## 性能优化

### 1. 并发控制

```go
config := &agents.MultiAgentConfig{
    MaxConcurrentAgents: runtime.NumCPU(), // 根据 CPU 核数
    MessageQueueSize:    1000,             // 足够的队列大小
}
```

### 2. 消息队列大小

```go
// 小型系统
config.MessageQueueSize = 100

// 中型系统
config.MessageQueueSize = 1000

// 大型系统
config.MessageQueueSize = 10000
```

### 3. Agent 复用

```go
// 创建 Agent 池
type AgentPool struct {
    agents []agents.MultiAgent
    mu     sync.Mutex
}

func (ap *AgentPool) GetAgent() agents.MultiAgent {
    ap.mu.Lock()
    defer ap.mu.Unlock()
    
    if len(ap.agents) > 0 {
        agent := ap.agents[0]
        ap.agents = ap.agents[1:]
        return agent
    }
    
    return agents.NewResearcherAgent("new", llm, nil)
}
```

### 4. 缓存 LLM 响应

```go
import "langchain-go/core/cache"

// 使用 Redis 缓存
redisCache, _ := cache.NewRedisCache(cache.DefaultRedisCacheConfig())
llmCache := cache.NewLLMCache(cache.CacheConfig{
    Enabled: true,
    TTL:     24 * time.Hour,
    Backend: redisCache,
})

// 在 Agent 中使用缓存的 LLM
```

---

## 故障排查

### 常见问题

#### 1. 任务超时

**症状**: `context deadline exceeded`

**解决方案**:
```go
config.TaskTimeout = 10 * time.Minute  // 增加超时时间
config.MessageTimeout = 60 * time.Second
```

#### 2. Agent 未找到

**症状**: `agent not found`

**解决方案**:
```go
// 确保 Agent 已添加到系统
system.AddAgent(agent.ID(), agent)

// 确保 Agent 已注册到 Coordinator
coordinator.RegisterAgent(agent)
```

#### 3. 消息队列满

**症状**: 系统hang住，无响应

**解决方案**:
```go
config.MessageQueueSize = 10000 // 增加队列大小
```

#### 4. 内存占用高

**解决方案**:
```go
// 禁用不需要的功能
config.EnableHistory = false  // 不记录历史
config.EnableSharedState = false  // 不使用共享状态
```

### 调试技巧

```go
// 1. 启用详细日志
config.Verbose = true

// 2. 查看执行历史
history := system.GetHistory()
for _, record := range history.GetAllRecords() {
    fmt.Printf("%s: %s -> %s\n", 
        record.MessageID, 
        record.Status,
        record.Error)
}

// 3. 监控指标
metrics := system.GetMetrics()
fmt.Printf("失败率: %.1f%%\n", 
    float64(metrics.FailedRuns)/float64(metrics.TotalRuns)*100)

// 4. 检查 Agent 状态
agents := system.ListAgents()
fmt.Printf("活跃 Agent 数: %d\n", len(agents))
```

---

## 高级主题

### 1. 动态 Agent 创建

```go
func (mas *MultiAgentSystem) CreateAgentOnDemand(
    agentType string,
    capabilities []string,
) (agents.MultiAgent, error) {
    switch agentType {
    case "researcher":
        return agents.NewResearcherAgent(
            fmt.Sprintf("researcher_%d", time.Now().Unix()),
            mas.llm,
            tools.NewDuckDuckGoSearch(),
        ), nil
    case "writer":
        return agents.NewWriterAgent(
            fmt.Sprintf("writer_%d", time.Now().Unix()),
            mas.llm,
            "technical",
        ), nil
    default:
        return nil, fmt.Errorf("unknown agent type: %s", agentType)
    }
}
```

### 2. Agent 学习和优化

```go
type LearningAgent interface {
    agents.MultiAgent
    Learn(feedback *Feedback) error
    GetPerformance() *PerformanceMetrics
}

type Feedback struct {
    TaskID     string
    Success    bool
    Rating     float64
    Comments   string
}
```

### 3. 分布式 Multi-Agent

```go
type DistributedMultiAgentSystem struct {
    localSystem  *agents.MultiAgentSystem
    remoteNodes  []string
    coordinator  *DistributedCoordinator
}
```

---

## 总结

Multi-Agent 系统是处理复杂任务的强大工具。通过合理的 Agent 设计和协调策略，可以实现高效的任务分解和并行处理。

**关键要点**:
1. ✅ 明确的 Agent 职责分离
2. ✅ 合理的协调策略选择
3. ✅ 完善的错误处理和监控
4. ✅ 性能优化和资源管理

**下一步**:
- 查看 [MULTI_AGENT_DESIGN.md](./MULTI_AGENT_DESIGN.md) 了解架构设计
- 运行 [multi_agent_demo.go](../examples/multi_agent_demo.go) 体验示例
- 创建自己的专用 Agent

---

**文档版本**: v1.0  
**更新日期**: 2026-01-16  
**状态**: ✅ 完整
