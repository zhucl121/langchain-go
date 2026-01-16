# 🤝 Multi-Agent 系统快速开始

## 5 分钟上手 Multi-Agent

### 安装

```bash
# 已包含在 langchain-go 中，无需额外安装
go get langchain-go/core/agents
```

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
    
    // 1. 创建协调器
    strategy := agents.NewSequentialStrategy(llm)
    coordinator := agents.NewCoordinatorAgent("coordinator", llm, strategy)
    
    // 2. 创建系统
    system := agents.NewMultiAgentSystem(coordinator, nil)
    
    // 3. 添加 Agent
    researcher := agents.NewResearcherAgent("researcher", llm, 
        tools.NewDuckDuckGoSearch())
    system.AddAgent("researcher", researcher)
    coordinator.RegisterAgent(researcher)
    
    writer := agents.NewWriterAgent("writer", llm, "technical")
    system.AddAgent("writer", writer)
    coordinator.RegisterAgent(writer)
    
    // 4. 执行任务
    result, _ := system.Run(ctx, "Research AI trends and write a summary")
    
    // 5. 输出结果
    fmt.Println("结果:", result.FinalResult)
    fmt.Printf("消息数: %d, 耗时: %v\n", result.MessageCount, result.Duration)
}
```

---

## 6 个专用 Agent

### 1. Researcher Agent - 研究员 🔍

```go
researcher := agents.NewResearcherAgent("researcher", llm, searchTool)
```

**擅长**: 搜索、调研、信息收集

### 2. Writer Agent - 写作者 ✍️

```go
writer := agents.NewWriterAgent("writer", llm, "creative")
// 风格: "creative", "technical", "formal"
```

**擅长**: 内容创作、编辑、摘要

### 3. Reviewer Agent - 审核者 ✅

```go
reviewer := agents.NewReviewerAgent("reviewer", llm, 
    []string{"accuracy", "clarity", "grammar"})
```

**擅长**: 质量检查、内容评估

### 4. Analyst Agent - 分析师 📊

```go
analyst := agents.NewAnalystAgent("analyst", llm)
```

**擅长**: 数据分析、模式识别

### 5. Planner Agent - 规划者 📋

```go
planner := agents.NewPlannerAgent("planner", llm)
```

**擅长**: 任务规划、策略制定

### 6. Coordinator Agent - 协调器 🎯

```go
coordinator := agents.NewCoordinatorAgent("coordinator", llm, strategy)
```

**擅长**: 任务分解、Agent 选择、结果聚合

---

## 3 个典型场景

### 场景 1: 内容创作流水线

```go
// 团队: Planner → Researcher → Writer → Reviewer
strategy := agents.NewSequentialStrategy(llm)
coordinator := agents.NewCoordinatorAgent("coordinator", llm, strategy)
system := agents.NewMultiAgentSystem(coordinator, nil)

// 添加团队成员
system.AddAgent("planner", agents.NewPlannerAgent("planner", llm))
system.AddAgent("researcher", agents.NewResearcherAgent("researcher", llm, searchTool))
system.AddAgent("writer", agents.NewWriterAgent("writer", llm, "creative"))
system.AddAgent("reviewer", agents.NewReviewerAgent("reviewer", llm, nil))

// 执行创作
result, _ := system.Run(ctx, "Create a blog post about AI")
```

### 场景 2: 数据分析管道

```go
// 团队: Collector → Analyst → Writer
system.AddAgent("collector", agents.NewResearcherAgent("collector", llm, nil))
system.AddAgent("analyst", agents.NewAnalystAgent("analyst", llm))
system.AddAgent("writer", agents.NewWriterAgent("writer", llm, "technical"))

// 执行分析
result, _ := system.Run(ctx, "Analyze market trends for EVs")
```

### 场景 3: 客户支持系统

```go
// 团队: 技术支持 + 账单支持
system.AddAgent("tech_support", agents.NewResearcherAgent("tech", llm, nil))
system.AddAgent("billing", agents.NewAnalystAgent("billing", llm))

// 处理问题
result, _ := system.Run(ctx, "Why was I charged twice?")
```

---

## 自定义 Agent

```go
type MyAgent struct {
    agents.BaseMultiAgent
    // 自定义字段
}

func (ma *MyAgent) ReceiveMessage(ctx context.Context, msg *agents.AgentMessage) error {
    // 处理消息
    if msg.Type == agents.MessageTypeTask {
        result := ma.process(msg.Content)
        return ma.SendResult(ctx, msg, result)
    }
    return nil
}

func (ma *MyAgent) CanHandle(task string) (bool, float64) {
    // 判断能否处理
    if strings.Contains(task, "my_keyword") {
        return true, 0.9
    }
    return false, 0.0
}
```

---

## 监控和调试

```go
// 获取指标
metrics := system.GetMetrics()
stats := metrics.GetStats()
fmt.Printf("成功率: %.1f%%\n", stats["success_rate"])

// 查看历史
history := system.GetHistory()
for _, record := range history.GetAllRecords() {
    fmt.Printf("%s: %s\n", record.MessageID, record.Status)
}

// 共享状态
system.GetSharedState().Set("key", "value")
```

---

## 配置选项

```go
config := &agents.MultiAgentConfig{
    MaxConcurrentAgents: 5,              // 最大并行数
    MessageTimeout:      30 * time.Second, // 消息超时
    TaskTimeout:         5 * time.Minute,  // 任务超时
    MaxRetries:          3,                // 最大重试
    EnableSharedState:   true,             // 共享状态
    EnableHistory:       true,             // 历史记录
    MessageQueueSize:    1000,             // 队列大小
}

system := agents.NewMultiAgentSystem(coordinator, config)
```

---

## 常见问题

### Q: 如何选择协调策略？

**A**: 
- `Sequential` - 有依赖关系的任务（内容创作）
- `Parallel` - 独立并行的任务（数据收集）
- `Hierarchical` - 复杂层次化任务（大型项目）

### Q: 如何处理错误？

**A**:
```go
result, err := system.Run(ctx, task)
if err != nil {
    log.Printf("错误: %v", err)
    // 检查历史记录
    history := system.GetHistory()
    for _, record := range history.GetAllRecords() {
        if record.Error != nil {
            log.Printf("Agent %s 失败: %v", record.MessageID, record.Error)
        }
    }
}
```

### Q: 如何优化性能？

**A**:
1. 根据 CPU 核数设置并发数
2. 使用 Redis 缓存 LLM 响应
3. 调整消息队列大小
4. 合理设置超时时间

---

## 下一步

- 📖 阅读 [完整使用指南](./docs/guides/multi-agent-guide.md)
- 🏗️ 查看 [架构设计文档](./MULTI_AGENT_DESIGN.md)
- 🎮 运行 [示例代码](./examples/multi_agent_demo.go)
- 📝 查看 [发布说明](./V1.7.0_RELEASE_NOTES.md)

---

## 核心优势

✅ **简单易用** - 3 步创建 Multi-Agent 系统  
✅ **功能完整** - 6 个专用 Agent 开箱即用  
✅ **灵活扩展** - 易于创建自定义 Agent  
✅ **生产就绪** - 完善的错误处理和监控  
✅ **高性能** - 充分利用 Go 的并发优势

---

**版本**: v1.7.0  
**状态**: ✅ 生产就绪

🚀 **开始使用 Multi-Agent 系统吧！**
