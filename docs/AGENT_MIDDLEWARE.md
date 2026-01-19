# Agent Middleware 系统

## 📖 概述

Agent Middleware 系统是 LangChain-Go v0.1.2+ 引入的统一中间件机制，对标 LangChain v1.0+ 的 Agent Middleware 设计。

### 核心特性

- ✅ **细粒度钩子** - BeforeModel, AfterModel, OnError, BeforeToolCall, AfterToolCall, OnComplete
- ✅ **中间件链** - 支持组合多个中间件
- ✅ **内置中间件** - Retry, RateLimit, ContentModeration, Caching, Logging
- ✅ **类型安全** - 强类型定义，编译时检查
- ✅ **插拔式** - 易于添加和移除
- ✅ **链式调用** - 流式 API 设计

---

## 🎯 为什么需要 Agent Middleware？

### 问题

传统的 Agent 实现缺少统一的中间件机制：
- 无法统一处理重试逻辑
- 缺少限流和内容审核
- 错误处理分散，难以维护
- 缺乏标准化的扩展点

### 解决方案

Agent Middleware 提供统一的插入点：

```go
agent := agents.CreateAgent(
    llm,
    tools,
    agents.WithMiddlewareChain(
        agents.NewRetryMiddleware(3),
        agents.NewRateLimitMiddleware(10, time.Second),
        agents.NewContentModerationMiddleware(bannedWords),
        agents.NewLoggingAgentMiddleware(),
    ),
)
```

---

## 📦 核心接口

### AgentMiddleware

主要的 Middleware 接口：

```go
type AgentMiddleware interface {
    // BeforeModel 在调用 LLM 之前执行
    BeforeModel(ctx context.Context, state *AgentState) (*AgentState, error)

    // AfterModel 在 LLM 响应后执行
    AfterModel(ctx context.Context, state *AgentState, response *types.Message) (*types.Message, error)

    // OnError 当发生错误时执行
    OnError(ctx context.Context, state *AgentState, err error) (shouldRetry bool, newErr error)

    // BeforeToolCall 在调用工具之前执行
    BeforeToolCall(ctx context.Context, toolName string, toolInput map[string]any) (map[string]any, error)

    // AfterToolCall 在工具调用后执行
    AfterToolCall(ctx context.Context, toolName string, toolInput map[string]any, toolOutput string, err error) (string, error)

    // OnComplete 当 Agent 完成执行时调用
    OnComplete(ctx context.Context, result *AgentResult) error

    // Name 返回中间件名称
    Name() string
}
```

### BaseAgentMiddleware

基础 Middleware（提供默认实现）：

```go
type BaseAgentMiddleware struct {
    name string
}

// 默认实现（不做任何修改，子类只需覆盖需要的方法）
func (b *BaseAgentMiddleware) BeforeModel(...) (*AgentState, error) {
    return state, nil
}

func (b *BaseAgentMiddleware) AfterModel(...) (*types.Message, error) {
    return response, nil
}

// ... 其他方法
```

### AgentMiddlewareChain

Middleware 链（按顺序执行多个 Middleware）：

```go
type AgentMiddlewareChain struct {
    middlewares []AgentMiddleware
}

// 依次执行所有 middleware 的钩子
func (c *AgentMiddlewareChain) BeforeModel(...) (*AgentState, error) {
    currentState := state
    for _, mw := range c.middlewares {
        newState, err := mw.BeforeModel(ctx, currentState)
        if err != nil {
            return nil, fmt.Errorf("middleware %s: %w", mw.Name(), err)
        }
        currentState = newState
    }
    return currentState, nil
}
```

---

## 🚀 使用示例

### 1. 基础 Middleware

```go
// 创建基础 middleware
middleware := agents.NewBaseAgentMiddleware("MyMiddleware")

ctx := context.Background()
state := &agents.AgentState{Input: "测试问题"}

// 使用 BeforeModel 钩子
newState, err := middleware.BeforeModel(ctx, state)
```

### 2. 重试 Middleware

```go
// 创建重试 middleware（最多重试 3 次）
retryMw := agents.NewRetryMiddleware(3).
    WithDelay(time.Second).
    WithBackoff(2.0)  // 指数退避系数

// 在发生错误时自动重试
shouldRetry, err := retryMw.OnError(ctx, state, originalErr)
if shouldRetry {
    // 会自动重试
}
```

### 3. 限流 Middleware

```go
// 创建限流 middleware（每秒最多 10 次请求）
rateLimitMw := agents.NewRateLimitMiddleware(10, time.Second)

// 在调用 LLM 前检查限流
newState, err := rateLimitMw.BeforeModel(ctx, state)
// 如果超过限流，会自动等待
```

### 4. 内容审核 Middleware

```go
// 创建内容审核 middleware
moderationMw := agents.NewContentModerationMiddleware([]string{
    "敏感词1",
    "敏感词2",
    "禁用词",
}).WithCaseSensitive(false).
   OnViolation(func(ctx context.Context, violationType string, content string) error {
    log.Printf("检测到违规内容: %s", violationType)
    return nil
})

// 检查输入
newState, err := moderationMw.BeforeModel(ctx, state)
if err != nil {
    // 输入包含敏感词
}

// 检查输出
newResponse, err := moderationMw.AfterModel(ctx, state, response)
if err != nil {
    // 输出包含敏感词
}
```

### 5. 缓存 Middleware

```go
// 创建缓存 middleware
cacheMw := agents.NewCachingMiddleware().
    WithTTL(5 * time.Minute).
    WithMaxSize(1000)

// 第一次调用 - 缓存未命中
newState, _ := cacheMw.BeforeModel(ctx, state)
response := callLLM(...)
cacheMw.AfterModel(ctx, state, response)

// 第二次调用 - 缓存命中
newState2, _ := cacheMw.BeforeModel(ctx, state)
if newState2.Extra["cache_hit"] == true {
    // 直接使用缓存的响应
    cached := newState2.Extra["cached_response"].(*types.Message)
}

// 获取统计
hits, misses, hitRate := cacheMw.GetStats()
fmt.Printf("命中率: %.2f%%\n", hitRate)
```

### 6. 日志 Middleware

```go
// 创建日志 middleware
loggingMw := agents.NewLoggingAgentMiddleware().
    WithLogger(func(level, message string, fields map[string]any) {
        log.Printf("[%s] %s %v", level, message, fields)
    }).
    WithVerbose(true)

// 自动记录所有关键操作
// - LLM 调用
// - 工具调用
// - 错误
// - 完成事件
```

### 7. 自定义 Middleware

```go
type MyMiddleware struct {
    *agents.BaseAgentMiddleware
    // 自定义字段
}

func NewMyMiddleware() *MyMiddleware {
    return &MyMiddleware{
        BaseAgentMiddleware: agents.NewBaseAgentMiddleware("MyMiddleware"),
    }
}

// 覆盖需要的方法
func (m *MyMiddleware) BeforeModel(ctx context.Context, state *agents.AgentState) (*agents.AgentState, error) {
    // 自定义逻辑
    log.Printf("Before model: %s", state.Input)
    return state, nil
}

func (m *MyMiddleware) AfterModel(ctx context.Context, state *agents.AgentState, response *types.Message) (*types.Message, error) {
    // 自定义逻辑
    log.Printf("After model: %s", response.Content)
    return response, nil
}
```

### 8. Middleware 链

```go
// 组合多个 middleware
chain := agents.NewAgentMiddlewareChain(
    agents.NewLoggingAgentMiddleware(),
    agents.NewRetryMiddleware(3),
    agents.NewRateLimitMiddleware(10, time.Second),
    agents.NewContentModerationMiddleware(bannedWords),
    agents.NewCachingMiddleware(),
)

// 按顺序执行所有 middleware
newState, err := chain.BeforeModel(ctx, state)
```

### 9. 与 Agent 集成

```go
// 方式 1: 使用 WithMiddleware 选项
agent := agents.CreateReActAgent(
    llm,
    tools,
    agents.WithMiddleware(agents.NewRetryMiddleware(3)),
    agents.WithMiddleware(agents.NewLoggingAgentMiddleware()),
)

// 方式 2: 使用 WithMiddlewareChain 选项
agent := agents.CreateToolCallingAgent(
    llm,
    tools,
    agents.WithMiddlewareChain(
        agents.NewRetryMiddleware(3),
        agents.NewRateLimitMiddleware(10, time.Second),
        agents.NewLoggingAgentMiddleware(),
    ),
)

// 方式 3: 手动构建配置
config := &agents.AgentConfig{
    LLM:   llm,
    Tools: tools,
    Extra: map[string]any{
        "middleware_chain": agents.NewAgentMiddlewareChain(
            agents.NewRetryMiddleware(3),
        ),
    },
}
```

---

## 📋 内置 Middleware

### RetryMiddleware

自动重试失败的操作。

**配置：**
- `maxRetries`: 最大重试次数
- `delay`: 初始延迟时间
- `backoff`: 退避系数（指数退避）

**特性：**
- 指数退避
- 自动延迟
- 重试计数管理

**使用场景：**
- 网络不稳定
- API 限流（配合重试）
- 临时故障

### RateLimitMiddleware

限制 LLM 调用频率。

**配置：**
- `maxRequests`: 时间窗口内的最大请求数
- `window`: 时间窗口

**特性：**
- 滑动窗口算法
- 自动等待
- 并发安全

**使用场景：**
- API 速率限制
- 成本控制
- 防止滥用

### ContentModerationMiddleware

检查输入和输出是否包含敏感内容。

**配置：**
- `bannedWords`: 禁用词列表
- `checkInput`: 是否检查输入
- `checkOutput`: 是否检查输出
- `caseSensitive`: 是否区分大小写

**特性：**
- 输入输出双向检查
- 自定义违规回调
- 大小写控制

**使用场景：**
- 内容合规
- 敏感词过滤
- 用户输入验证

### CachingMiddleware

缓存 LLM 响应。

**配置：**
- `ttl`: 缓存过期时间
- `maxSize`: 最大缓存数量

**特性：**
- 自动缓存管理
- TTL 过期
- 统计信息

**使用场景：**
- 减少重复调用
- 降低延迟
- 成本优化

### LoggingAgentMiddleware

记录 Agent 执行的详细日志。

**配置：**
- `verbose`: 是否详细输出
- `logModelCalls`: 是否记录 LLM 调用
- `logToolCalls`: 是否记录工具调用
- `logErrors`: 是否记录错误

**特性：**
- 全生命周期日志
- 自定义日志函数
- 详细字段

**使用场景：**
- 调试
- 性能分析
- 审计

---

## 🎨 设计模式

### 1. 责任链模式

Middleware 链按顺序执行，每个 Middleware 负责一部分职责。

```go
chain := NewAgentMiddlewareChain(
    mw1,  // 第一个处理
    mw2,  // 第二个处理
    mw3,  // 第三个处理
)
```

### 2. 装饰器模式

每个 Middleware 装饰原有的 Agent 行为，添加额外功能。

```go
// 基础 Agent
agent := NewReActAgent(llm, tools)

// 装饰：添加重试
agent = WithRetry(agent, 3)

// 装饰：添加限流
agent = WithRateLimit(agent, 10, time.Second)
```

### 3. 钩子模式

在关键节点插入自定义逻辑。

```go
type MyMiddleware struct {
    *BaseAgentMiddleware
}

// 在 LLM 调用前插入
func (m *MyMiddleware) BeforeModel(...) {
    // 自定义逻辑
}

// 在 LLM 调用后插入
func (m *MyMiddleware) AfterModel(...) {
    // 自定义逻辑
}
```

---

## 🔄 执行流程

```
User Input
    ↓
[BeforeModel Hook - Middleware 1]
    ↓
[BeforeModel Hook - Middleware 2]
    ↓
[BeforeModel Hook - Middleware 3]
    ↓
LLM Call
    ↓
[AfterModel Hook - Middleware 1]
    ↓
[AfterModel Hook - Middleware 2]
    ↓
[AfterModel Hook - Middleware 3]
    ↓
[BeforeToolCall Hook]
    ↓
Tool Execution
    ↓
[AfterToolCall Hook]
    ↓
[OnComplete Hook]
    ↓
Final Result
```

如果发生错误：

```
Error
    ↓
[OnError Hook - Middleware 1]
    ↓
[OnError Hook - Middleware 2]
    ↓
[OnError Hook - Middleware 3]
    ↓
决定：重试 / 返回错误
```

---

## ⚡ 性能优化

### 1. 避免不必要的 Middleware

只添加真正需要的 Middleware：

```go
// Bad: 添加过多 middleware
chain := NewAgentMiddlewareChain(
    mw1, mw2, mw3, mw4, mw5, mw6, mw7, mw8,
)

// Good: 只添加必要的
chain := NewAgentMiddlewareChain(
    NewRetryMiddleware(3),
    NewLoggingAgentMiddleware(),
)
```

### 2. 优化 Middleware 顺序

将轻量级 Middleware 放在前面：

```go
// Good: 轻量级的日志在前，重量级的缓存在后
chain := NewAgentMiddlewareChain(
    NewLoggingAgentMiddleware(),  // 轻量
    NewCachingMiddleware(),        // 重量（查询缓存）
    NewRetryMiddleware(3),         // 中等
)
```

### 3. 使用缓存 Middleware

减少重复的 LLM 调用：

```go
cacheMw := NewCachingMiddleware().
    WithTTL(5 * time.Minute).
    WithMaxSize(1000)
```

### 4. 合理设置重试和限流

避免过度重试和过严限流：

```go
// Good: 合理的配置
retryMw := NewRetryMiddleware(3).  // 最多 3 次
    WithDelay(time.Second)         // 延迟 1 秒

rateLimitMw := NewRateLimitMiddleware(10, time.Second)  // 每秒 10 次
```

---

## 🧪 测试

### 单元测试

```go
func TestMyMiddleware(t *testing.T) {
    mw := NewMyMiddleware()
    
    ctx := context.Background()
    state := &agents.AgentState{Input: "test"}
    
    newState, err := mw.BeforeModel(ctx, state)
    if err != nil {
        t.Errorf("BeforeModel failed: %v", err)
    }
    
    // 验证行为
    if newState.Input != "modified test" {
        t.Error("middleware should modify input")
    }
}
```

### 集成测试

```go
func TestMiddlewareChain(t *testing.T) {
    chain := agents.NewAgentMiddlewareChain(
        agents.NewRetryMiddleware(2),
        agents.NewLoggingAgentMiddleware(),
    )
    
    ctx := context.Background()
    state := &agents.AgentState{Input: "test"}
    
    newState, err := chain.BeforeModel(ctx, state)
    if err != nil {
        t.Fatalf("chain failed: %v", err)
    }
    
    // 验证链式执行
}
```

---

## 🐛 故障排查

### 问题1: Middleware 不生效

**检查：**
- Middleware 是否正确添加到 Agent 配置
- Middleware 方法是否正确覆盖
- Middleware 是否返回错误

```go
// 确保正确添加
agent := CreateAgent(
    llm,
    tools,
    WithMiddleware(myMiddleware),  // ✅ 正确
)
```

### 问题2: Middleware 链顺序错误

**检查：**
- Middleware 链的顺序是否正确
- 每个 Middleware 是否正确传递状态

```go
// 顺序很重要！
chain := NewAgentMiddlewareChain(
    loggingMw,      // 第一个：记录原始输入
    moderationMw,   // 第二个：检查输入
    retryMw,        // 第三个：错误重试
)
```

### 问题3: 缓存未命中

**检查：**
- 缓存键生成是否正确
- TTL 是否过短
- 缓存大小是否过小

```go
// 增加 TTL 和缓存大小
cache := NewCachingMiddleware().
    WithTTL(10 * time.Minute).  // 增加 TTL
    WithMaxSize(10000)           // 增加容量
```

---

## 🔗 相关资源

- **源码**: `core/agents/middleware.go`
- **内置 Middleware**: `core/agents/middleware_builtin.go`
- **测试**: `core/agents/middleware_test.go`
- **示例**: `examples/agent_middleware_demo/`
- **设计文档**: [LangChain v1.0 Agent Middleware](https://blog.langchain.com/langchain-langgraph-1dot0/)

---

## 🎓 最佳实践

### 1. 始终使用 WithMiddleware 或 WithMiddlewareChain

```go
// Good
agent := CreateAgent(
    llm,
    tools,
    WithMiddlewareChain(mw1, mw2, mw3),
)

// Bad: 手动修改配置
config.Extra["middlewares"] = []AgentMiddleware{mw1, mw2}
```

### 2. 日志 Middleware 应该放在最外层

```go
// Good: 日志记录所有操作
chain := NewAgentMiddlewareChain(
    NewLoggingAgentMiddleware(),  // 最外层
    NewRetryMiddleware(3),
    NewCachingMiddleware(),
)
```

### 3. 内容审核应该在最前面

```go
// Good: 先审核再处理
chain := NewAgentMiddlewareChain(
    NewContentModerationMiddleware(bannedWords),  // 最前面
    NewCachingMiddleware(),
    NewRetryMiddleware(3),
)
```

### 4. 缓存应该在重试之后

```go
// Good: 先重试再缓存
chain := NewAgentMiddlewareChain(
    NewRetryMiddleware(3),      // 先重试
    NewCachingMiddleware(),      // 再缓存成功的结果
)
```

### 5. 自定义 Middleware 应该继承 BaseAgentMiddleware

```go
// Good
type MyMiddleware struct {
    *agents.BaseAgentMiddleware
}

func NewMyMiddleware() *MyMiddleware {
    return &MyMiddleware{
        BaseAgentMiddleware: agents.NewBaseAgentMiddleware("MyMiddleware"),
    }
}
```

### 6. 错误处理要清晰

```go
func (m *MyMiddleware) BeforeModel(ctx context.Context, state *agents.AgentState) (*agents.AgentState, error) {
    if state.Input == "" {
        return nil, fmt.Errorf("middleware %s: input is empty", m.Name())
    }
    return state, nil
}
```

---

## 🚀 下一步

- [Streaming 支持](./STREAMING.md)
- [Hybrid Search](./HYBRID_SEARCH.md)
- [Content Block](./CONTENT_BLOCK.md)

---

**版本**: v0.1.2  
**状态**: ✅ 已实现  
**最后更新**: 2026-01-20
