# Phase 2: LangGraph 核心实现规划

**开始日期**: 2026-01-14  
**预计完成**: 2026-02-15 (4-5 周)  
**当前状态**: 🚀 启动中

---

## 一、Phase 2 概述

### 1.1 目标

实现 LangGraph 1.0+ 全部核心功能，包括：
- ✅ StateGraph 状态图系统
- ✅ Checkpointing 持久化 ⭐
- ✅ Durability 持久化模式 ⭐
- ✅ Human-in-the-Loop 人工干预 ⭐
- ✅ Streaming 流式输出

### 1.2 模块清单

| 分组 | 模块 | 优先级 | 预估 Token | 说明 |
|------|------|--------|-----------|------|
| **剩余 Phase 1** |
| Memory | M19-M21 | P1 | 26K | 记忆系统（可选但建议） |
| **StateGraph 核心** |
| State | M24-M26 | P0 | 47K | StateGraph、Channel、Reducer |
| Node | M27-M29 | P0 | 33K | 节点接口和实现 |
| Edge | M30-M32 | P0 | 30K | 边系统 |
| Compile | M33-M34 | P0 | 32K | 编译和验证 |
| Execute | M35-M37 | P0 | 50K | 执行引擎 |
| **LangGraph 1.0 核心特性** ⭐ |
| Checkpoint | M38-M42 | P0 | 61K | 检查点系统 |
| Durability | M43-M45 | P0 | 35K | 持久化模式 |
| HITL | M46-M49 | P0 | 45K | 人工干预 |
| Streaming | M50-M52 | P0 | 28K | 流式输出 |
| **总计** | **29 个模块** | - | **~387K** | - |

### 1.3 依赖关系

```
Phase 1 (已完成)
    ├── M01-M04: 基础类型 ✅
    ├── M05-M08: Runnable 系统 ✅
    ├── M09-M12: ChatModel ✅
    ├── M13-M14: Prompts ✅
    ├── M15-M16: OutputParser ✅
    └── M17-M18: Tools ✅

M19-M21: Memory (可选先完成)
    └── 为 Agent 系统准备

M24-M26: StateGraph 核心
    └── 依赖: M04 (config)

M27-M29: Node 系统
    └── 依赖: M24

M30-M32: Edge 系统
    └── 依赖: M24, M27

M33-M34: Compile 系统
    └── 依赖: M24-M32

M38-M42: Checkpoint ⭐
    └── 依赖: 无（独立）

M43-M45: Durability ⭐
    └── 依赖: M38

M35-M37: Execute 引擎
    └── 依赖: M33, M38, M43

M46-M49: HITL ⭐
    └── 依赖: M35, M38

M50-M52: Streaming
    └── 依赖: M35
```

---

## 二、实施计划

### 2.1 Week 1: StateGraph 核心 + Memory (可选)

**目标**: 建立状态图基础架构

#### 选项 A: 先完成 Memory（推荐）
```
Day 1-2: M19-M21 Memory 系统
├── M19: memory/interface.go
├── M20: memory/buffer.go
└── M21: memory/summary.go

Day 3-4: M24-M26 StateGraph 核心
├── M24: state/graph.go (核心)
├── M25: state/channel.go
└── M26: state/reducer.go

Day 5: M27-M28 Node 基础
├── M27: node/interface.go
└── M28: node/function.go
```

#### 选项 B: 直接进入 LangGraph
```
Day 1-3: M24-M26 StateGraph 核心
├── M24: state/graph.go (核心)
├── M25: state/channel.go
└── M26: state/reducer.go

Day 4-5: M27-M28 Node 系统
├── M27: node/interface.go
└── M28: node/function.go
```

**交付物**:
- [ ] StateGraph 基础实现
- [ ] Node 接口和函数节点
- [ ] (可选) Memory 系统
- [ ] 单元测试
- [ ] 示例代码

### 2.2 Week 2: Edge + Compile

**目标**: 完善图定义和编译能力

```
Day 1-2: M30-M31 Edge 系统
├── M30: edge/edge.go
└── M31: edge/conditional.go

Day 3: M32 Router
└── M32: edge/router.go

Day 4-5: M33-M34 Compile 系统
├── M33: compile/compiler.go
└── M34: compile/validator.go
```

**交付物**:
- [ ] Edge 和条件边实现
- [ ] 图编译器
- [ ] 图验证器
- [ ] 单元测试
- [ ] 简单图示例

### 2.3 Week 3: Checkpoint 系统 ⭐

**目标**: 实现完整的检查点持久化

```
Day 1: M38-M39 接口和类型
├── M38: checkpoint/interface.go
└── M39: checkpoint/checkpoint.go

Day 2: M40 内存存储
└── M40: checkpoint/memory.go

Day 3-4: M41 SQLite 存储
└── M41: checkpoint/sqlite.go

Day 5: M42 PostgreSQL 存储 (可选延后)
└── M42: checkpoint/postgres.go
```

**交付物**:
- [ ] Saver 接口
- [ ] MemorySaver 实现
- [ ] SQLiteSaver 实现
- [ ] (可选) PostgresSaver 实现
- [ ] 检查点测试
- [ ] Time Travel 支持

### 2.4 Week 4: Durability + Execute

**目标**: 实现持久化模式和执行引擎

```
Day 1-2: M43-M45 Durability
├── M43: durability/mode.go
├── M44: durability/task.go
└── M45: durability/recovery.go

Day 3-5: M35-M37 Execute 引擎
├── M35: execute/executor.go (核心)
├── M36: execute/context.go
└── M37: execute/scheduler.go
```

**交付物**:
- [ ] 三种持久化模式 (exit/async/sync)
- [ ] 任务包装和去重
- [ ] 图执行引擎
- [ ] 执行上下文
- [ ] 并行调度器
- [ ] 集成测试

### 2.5 Week 5: HITL + Streaming

**目标**: 完成 LangGraph 1.0 核心特性

```
Day 1-2: M46-M47 中断和恢复
├── M46: hitl/interrupt.go
└── M47: hitl/resume.go

Day 3: M48-M49 审批和处理器
├── M48: hitl/approval.go
└── M49: hitl/handler.go

Day 4: M50-M52 Streaming
├── M50: streaming/stream.go
├── M51: streaming/modes.go
└── M52: streaming/event.go

Day 5: 集成测试和文档
```

**交付物**:
- [ ] 中断机制
- [ ] 恢复机制
- [ ] 审批模式
- [ ] 流式输出（3种模式）
- [ ] 完整集成测试
- [ ] Phase 2 总结文档
- [ ] 使用示例

---

## 三、关键技术点

### 3.1 StateGraph 设计

```go
type StateGraph[S any] struct {
    name         string
    nodes        map[string]Node[S]
    edges        []Edge
    conditionals []ConditionalEdge[S]
    entryPoint   string
    
    // LangGraph 1.0 新增
    checkpointer checkpoint.Saver
    durability   durability.Mode
    channels     map[string]Channel
}
```

**关键特性**:
- 泛型状态类型
- 声明式 API
- 链式调用
- 图验证

### 3.2 Checkpointing 设计

```go
type Checkpoint struct {
    ID          string
    ThreadID    string
    ParentID    *string
    State       []byte         // 序列化状态
    Metadata    map[string]any
    CreatedAt   time.Time
    CurrentNode string
    Status      CheckpointStatus
}

type Saver interface {
    Put(ctx context.Context, config Config, cp Checkpoint) error
    Get(ctx context.Context, config Config) (*Checkpoint, error)
    List(ctx context.Context, config Config, opts ListOptions) ([]Checkpoint, error)
}
```

**关键特性**:
- Time Travel (历史查询)
- 多存储后端（Memory/SQLite/Postgres）
- 状态序列化
- 父子关系追踪

### 3.3 Durability 模式

```go
type Mode string

const (
    ModeExit  Mode = "exit"  // 退出时持久化
    ModeAsync Mode = "async" // 异步批量
    ModeSync  Mode = "sync"  // 同步持久化
)
```

**特性**:
- 非确定性操作包装
- 任务去重
- 结果缓存
- 异步批量写入

### 3.4 Human-in-the-Loop 设计

```go
// 节点中触发中断
func approvalNode(ctx context.Context, state State) (State, error) {
    if state.RequiresApproval {
        hitl.TriggerInterrupt(hitl.Interrupt{
            Type:    hitl.InterruptApproval,
            Message: "请审批此操作",
        })
    }
    return state, nil
}

// 恢复执行
result, err := executor.Resume(ctx, threadID, hitl.ResumeData{
    Action: hitl.ActionApprove,
})
```

**关键特性**:
- panic/recover 模式
- 中断检查点
- 多种恢复动作
- 审批流程

### 3.5 Streaming 模式

```go
type Mode string

const (
    ModeValues  Mode = "values"  // 状态更新
    ModeUpdates Mode = "updates" // 增量更新
    ModeEvents  Mode = "events"  // 详细事件
)
```

**特性**:
- 三种流模式
- Channel 实现
- 事件类型化
- 取消支持

---

## 四、测试策略

### 4.1 单元测试

每个模块必须包含：
- ✅ 正常路径测试
- ✅ 错误处理测试
- ✅ 边界条件测试
- ✅ 并发安全测试

### 4.2 集成测试

关键场景：
- [ ] 简单状态图执行
- [ ] 条件边和循环
- [ ] 检查点保存和恢复
- [ ] Time Travel
- [ ] 中断和恢复
- [ ] 流式输出
- [ ] 并发执行

### 4.3 性能测试

- [ ] 大状态序列化性能
- [ ] 检查点写入性能
- [ ] 并发执行性能
- [ ] 内存占用测试

---

## 五、风险和挑战

### 5.1 技术风险

| 风险 | 影响 | 缓解措施 |
|------|------|---------|
| 状态序列化复杂 | 高 | 先支持 JSON，后续优化 gob |
| 并发安全问题 | 高 | 充分的并发测试 |
| 检查点性能 | 中 | 支持多种持久化模式 |
| HITL 实现复杂 | 中 | 参考 Python 实现 |

### 5.2 进度风险

| 风险 | 概率 | 应对方案 |
|------|------|---------|
| 模块依赖阻塞 | 低 | 按依赖顺序实施 |
| 测试覆盖不足 | 中 | 边开发边测试 |
| 集成问题 | 中 | 每周集成测试 |

---

## 六、成功标准

### 6.1 Phase 2 完成标准

- [ ] 全部 29 个模块实现
- [ ] 单元测试覆盖率 > 60%
- [ ] 集成测试通过
- [ ] 文档完整（总结 + 示例）
- [ ] 至少 2 个端到端示例

### 6.2 质量标准

- [ ] 所有模块通过 `go vet`
- [ ] 所有测试通过 `go test`
- [ ] 遵循 Go 编码规范
- [ ] 完整的 GoDoc 注释
- [ ] 错误处理完善

---

## 七、下一步行动

### 立即开始

根据项目进度，建议采用 **选项 A（先完成 Memory）**：

1. ✅ **M19-M21: Memory 系统** (本周完成)
   - 为 Phase 3 Agent 系统做准备
   - 相对独立，不阻塞 LangGraph 核心
   - 可以先完成，增加项目完整性

2. ✅ **M24-M26: StateGraph 核心** (下周开始)
   - LangGraph 的基础
   - 后续模块的依赖

### 启动命令

```bash
# 创建目录结构
mkdir -p graph/{state,node,edge,compile,execute,checkpoint,durability,hitl,streaming}

# 开始 M19: Memory 接口
# (或者直接开始 M24: StateGraph)
```

---

## 八、参考资源

- [设计方案](../LangChain-LangGraph-Go重写设计方案.md)
- [项目进度](../PROJECT-PROGRESS.md)
- [.cursorrules](../.cursorrules)
- [Python LangGraph](https://github.com/langchain-ai/langgraph)

---

**创建日期**: 2026-01-14  
**创建者**: AI Assistant  
**状态**: ✅ 已完成规划，等待启动
