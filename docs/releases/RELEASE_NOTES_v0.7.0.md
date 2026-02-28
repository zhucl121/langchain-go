# v0.7.0 发布说明 - 认知增强 + 生产就绪

**发布日期**: 2026-02-28  
**标签**: v0.7.0  
**类型**: Minor Release（向后兼容功能新增）

---

## 🌟 重大更新

v0.7.0 是 LangChain-Go 迄今为止最大的能力跃升之一，聚焦两个核心方向：

1. **认知智能** — 三层记忆系统让 Agent 真正"记住"用户，实现个性化、连贯的多轮对话
2. **生产就绪** — Circuit Breaker、Bulkhead、节点缓存、统一 API 让 Agent 在生产环境更稳定

---

## ✨ 新功能（10个核心模块）

### A. LangGraph 图能力增强

#### A1. 节点级缓存（Node-Level Caching）
- 对标 LangGraph 1.0 Node-Level Caching
- `CachedFunctionNode`：对相同输入直接返回缓存结果，跳过 LLM 调用
- 支持 TTL 过期、LRU 淘汰、自定义 KeyFunc
- 内置 `InMemoryNodeCache`，可扩展 Redis 等后端
- 22 个测试用例

#### A2. 延迟节点（Deferred Nodes）
- 对标 LangGraph Deferred Nodes，实现 Map-Reduce 模式
- `DeferredFunctionNode`：等待所有上游并行分支完成后聚合执行
- 支持 Timeout、FailFast、MinBranches 配置
- `MapReduceRunner`：三行代码实现 Map-Reduce

#### A3. Pre/Post Model Hooks
- 对标 LangGraph Pre/Post Model Hooks
- `ModelHook` 接口 + `HookChain` 链式执行
- 内置 5 种 Hook：`SummaryHook`（历史摘要）、`PIIRedactHook`（PII 脱敏）、`GuardrailHook`（内容护栏）、`RateLimitHook`（速率限制）、`ContentFilterHook`（关键词过滤）

#### A4. 可恢复流（Resumable Streams）
- 对标 LangGraph JS `reconnectOnMount`
- 网络中断后从断点续传，不丢失任何 token
- `InMemoryStreamStorage`：带 TTL 的内存存储
- 可扩展 Redis 等持久化后端

### B. 三层认知记忆系统
- **语义记忆**（Semantic Memory）：事实、知识、用户偏好，支持置信度和知识三元组
- **情节记忆**（Episodic Memory）：对话历史、Few-Shot 示例，支持质量评分筛选
- **程序记忆**（Procedural Memory）：从反馈中学习的行为规则，自动优化 Agent 响应
- **统一 Recall**：并行检索三层，生成可直接注入 System Prompt 的 `AugmentedContext`
- **AutoConsolidate**：从对话自动提取记忆（支持同步/异步模式）
- 可扩展存储接口：接入 PostgreSQL、Redis、向量数据库

### C. 可靠结构化输出（Trustcall）
- **JSON Patch 修复**：提取失败时只传差异，而非重新生成完整 JSON（节省 50-80% Token）
- `ReliableExtractor[T]`：泛型接口，类型安全，自动从 Markdown/混合文本提取 JSON
- `ReasoningTracer`：结构化记录推理步骤、工具调用、Token 用量
- 与 `types.ContentBlock` 互操作（`ToContentBlock()`）

### D. 生产级容错机制
- **CircuitBreaker**：三态状态机（Closed/Open/HalfOpen），防止 LLM API 级联失败
- **Bulkhead**：信号量并发控制，防止某个 Agent 耗尽所有资源
- **ResilienceWrapper**：Bulkhead + CircuitBreaker 一体化组合

### E. Standard Content Blocks（已有，增强集成）
- `types.ContentBlock`：标准化文本/思考/工具调用/错误内容块
- 与 v0.7.0 新增的 `ReasoningTracer` 深度集成

### F. LangGraph Store（跨会话存储）
- 命名空间 + 键值存储，实现 StateGraph 跨会话数据共享
- `InMemoryStore`：开发/测试零配置
- `WithStore(ctx)` / `GetStore(ctx)`：context 注入，在节点内直接访问
- 支持 Search、List、BatchPut、BatchGet、ListNamespaces

### G. 统一 Agent 创建接口
- `CreateUnifiedAgent`：一个入口，支持 5 种预设策略，修改一个字段即可切换
- `ManagedAgent`：封装 Agent + Executor，`Run` / `RunWithMemory` 便捷方法
- `InMemoryConversationMemory`：多轮对话记忆，自动截断（最多 20 条）
- ModelHooks + Resilience 一体化配置

---

## 📦 完整交付

### 新增文件

| 文件 | 说明 |
|------|------|
| `graph/node/cache.go` + `cache_test.go` | 节点级缓存 |
| `graph/node/deferred.go` + `deferred_test.go` | 延迟节点 |
| `core/middleware/hooks.go` + `hooks_test.go` | Pre/Post Model Hooks |
| `graph/streaming/resumable.go` + `resumable_test.go` | 可恢复流 |
| `core/memory/cognitive/interface.go` | 认知记忆接口定义 |
| `core/memory/cognitive/manager.go` + `manager_test.go` | 认知记忆管理器 |
| `core/output/trustcall.go` + `trustcall_test.go` | 可靠结构化输出 |
| `core/agents/circuit_breaker.go` + `circuit_breaker_test.go` | 容错机制 |
| `core/agents/create_agent.go` + `create_agent_test.go` | 统一 Agent API |
| `graph/store/interface.go` + `memory_store.go` + `store_test.go` | LangGraph Store |
| `examples/cognitive_memory_demo/` | 认知记忆示例 |
| `examples/create_agent_v2_demo/` | 统一 Agent 示例 |
| `docs/V0.7.0_USER_GUIDE.md` | 用户指南 |
| `docs/V0.7.0_DESIGN_PROPOSAL.md` | 设计方案 |

### 统计数据

| 指标 | 数值 |
|------|------|
| 新增代码 | ~10,000 行 |
| 新增测试 | 200+ 用例 |
| 新增示例 | 2 个 |
| 新增文档 | 4 个 |
| 破坏性变更 | **0** |

---

## 🚀 快速开始

### 安装

```bash
go get github.com/zhucl121/langchain-go@v0.7.0
```

### 5 分钟体验认知记忆

```go
package main

import (
    "context"
    "fmt"
    cogmem "github.com/zhucl121/langchain-go/core/memory/cognitive"
)

func main() {
    ctx := context.Background()
    
    manager := cogmem.NewMemoryManager(cogmem.ManagerConfig{
        UserID:            "user-alice",
        SemanticStorage:   cogmem.NewInMemorySemanticStorage(),
        EpisodicStorage:   cogmem.NewInMemoryEpisodicStorage(),
        ProceduralStorage: cogmem.NewInMemoryProceduralStorage(),
    })
    
    // 存储用户偏好
    manager.StoreSemanticMemory(ctx, &cogmem.SemanticMemory{
        Content: "用户是 Go 开发者，喜欢简洁代码", Confidence: 0.9,
    })
    
    // 检索并增强上下文
    recalled, _ := manager.Recall(ctx, "Go 编程", cogmem.DefaultRecallOptions())
    fmt.Println(recalled.AugmentedContext)
}
```

### 5 分钟体验统一 Agent API

```go
package main

import (
    "context"
    "fmt"
    "github.com/zhucl121/langchain-go/core/agents"
)

func main() {
    ctx := context.Background()
    
    // 一个 API，切换 Preset 即可换策略
    ca, _ := agents.CreateUnifiedAgent(agents.UnifiedAgentConfig{
        Preset: agents.PresetToolCalling,
        Model:  yourLLM,
        Tools:  yourTools,
    })
    
    result, _ := ca.Run(ctx, "你的问题")
    fmt.Println(result.Output)
}
```

---

## 🔄 升级指南

v0.7.0 完全向后兼容，无需任何代码修改即可升级。

可选的新特性采用方式：

1. **认知记忆**：在现有 Agent 中添加 `manager.Recall()` 调用，增强 System Prompt
2. **统一 API**：新建 Agent 使用 `CreateUnifiedAgent`，旧代码无需修改
3. **节点缓存**：对高频/昂贵节点调用 `.WithNodeCache()`，透明加速
4. **容错**：在 `CreateUnifiedAgent` 配置中添加 `Resilience` 字段

---

## 📚 相关文档

- 📘 [完整用户指南](../V0.7.0_USER_GUIDE.md)
- 📋 [设计方案](../V0.7.0_DESIGN_PROPOSAL.md)
- 💡 [认知记忆示例](../../examples/cognitive_memory_demo/)
- 💡 [统一 Agent 示例](../../examples/create_agent_v2_demo/)
- 📋 [CHANGELOG](../../CHANGELOG.md)
