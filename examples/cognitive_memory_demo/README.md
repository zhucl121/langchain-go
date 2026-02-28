# 三层认知记忆系统演示

🌍 **Language**: 中文 | [English](#english)

## 功能演示

本示例展示 LangChain-Go v0.7.0 新增的**三层认知记忆系统**，对标 LangMem SDK 的完整记忆架构：

| 记忆层 | 存储内容 | 适用场景 |
|--------|----------|----------|
| **语义记忆**（Semantic） | 事实、知识、用户偏好 | "Alice 是 Go 开发者" |
| **情节记忆**（Episodic） | 对话片段、Few-Shot 示例 | "上次讨论了并发爬虫" |
| **程序记忆**（Procedural） | 行为规则、学习到的技能 | "始终提供完整代码示例" |

## 运行示例

```bash
cd examples/cognitive_memory_demo
go run main.go
```

## 核心概念

### 三层记忆架构

```
用户对话
    │
    ▼ AutoConsolidate()
┌───────────────────────────────────────┐
│            认知记忆管理器              │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐│
│  │ 语义记忆  │ │ 情节记忆  │ │ 程序记忆  ││
│  │ 事实知识  │ │ 对话历史  │ │ 行为规则  ││
│  └──────────┘ └──────────┘ └──────────┘│
└───────────────────────────────────────┘
    │
    ▼ Recall(query)
增强上下文 → 注入 System Prompt → LLM 生成个性化回答
```

### 使用示例

**初始化**

```go
manager := cogmem.NewMemoryManager(cogmem.ManagerConfig{
    UserID:            "user-alice",
    SemanticStorage:   cogmem.NewInMemorySemanticStorage(),
    EpisodicStorage:   cogmem.NewInMemoryEpisodicStorage(),
    ProceduralStorage: cogmem.NewInMemoryProceduralStorage(),
    Extractor:         cogmem.NewRuleBasedExtractor(),
})
```

**存储语义记忆**

```go
manager.StoreSemanticMemory(ctx, &cogmem.SemanticMemory{
    Content:    "用户偏好简洁、高性能的代码风格",
    Category:   "user-preference",
    Confidence: 0.9,
})
```

**从对话自动提取**

```go
// 传入对话消息，自动提取并存储到三层记忆
manager.AutoConsolidate(ctx, messages)
```

**跨层并行检索**

```go
recalled, _ := manager.Recall(ctx, "Alice 的编程风格", cogmem.DefaultRecallOptions())

// AugmentedContext 可直接注入 System Prompt
systemPrompt := basePrompt + "\n\n" + recalled.AugmentedContext
```

**更新程序记忆（行为学习）**

```go
// 正面反馈 → 提取成功规则
manager.UpdateProceduralMemory(ctx, &cogmem.Feedback{
    Score:   1.0,
    Comment: "用户满意完整代码示例",
})

// 负面反馈 → 提取改进规则
manager.UpdateProceduralMemory(ctx, &cogmem.Feedback{
    Score:         -0.5,
    FailureReason: "回答过于冗长",
})
```

## 存储后端扩展

内置内存存储适合开发/测试，生产环境可替换：

```go
// 实现 SemanticStorage 接口即可替换后端
type PostgresSemanticStorage struct { /* ... */ }
func (s *PostgresSemanticStorage) Save(ctx context.Context, mem *SemanticMemory) error { /* ... */ }
func (s *PostgresSemanticStorage) Search(ctx context.Context, query string, k int, userID string) ([]*SemanticMemory, error) { /* ... */ }
```

## 相关文档

- 📚 [v0.7.0 用户指南](../../docs/V0.7.0_USER_GUIDE.md)
- 📋 [设计方案](../../docs/V0.7.0_DESIGN_PROPOSAL.md)
- 🔗 [接口定义](../../core/memory/cognitive/interface.go)

---

## English

### 3-Layer Cognitive Memory System Demo

This example demonstrates the **3-Layer Cognitive Memory System** introduced in LangChain-Go v0.7.0:

- **Semantic Memory**: Facts and knowledge (user preferences, knowledge triplets)
- **Episodic Memory**: Experiences (conversation history, Few-Shot examples)
- **Procedural Memory**: Behaviors and skills (behavior rules learned from feedback)

### Run

```bash
cd examples/cognitive_memory_demo
go run main.go
```

### Key API

```go
// Initialize
manager := cogmem.NewMemoryManager(cogmem.ManagerConfig{UserID: "user-123"})

// Store
manager.StoreSemanticMemory(ctx, &cogmem.SemanticMemory{Content: "User prefers Go"})

// Auto-extract from conversation
manager.AutoConsolidate(ctx, messages)

// Cross-layer retrieval → augmented context
recalled, _ := manager.Recall(ctx, "user coding style", cogmem.DefaultRecallOptions())
systemPrompt += recalled.AugmentedContext
```
