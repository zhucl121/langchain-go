# M27-M29: Node 系统 - 实现总结

**完成日期**: 2026-01-14  
**模块**: M27 (Node 接口), M28 (FunctionNode), M29 (SubgraphNode)  
**测试覆盖率**: 89.8%

---

## 📋 实现概述

### 已完成模块

1. **M27: Node 接口**
   - Node 通用接口
   - Metadata 元数据系统
   - NodeOption 选项模式

2. **M28: FunctionNode**
   - 基于函数的节点
   - Chain（链接）
   - Retry（重试）
   - Fallback（降级）
   - Transform（转换）
   - Conditional（条件执行）

3. **M29: SubgraphNode**
   - 嵌套图节点
   - 状态映射
   - MockSubgraph 测试工具

---

## 🎯 核心功能

### 1. Node 接口 - 节点通用接口

```go
type Node[S any] interface {
    GetName() string
    GetDescription() string
    GetTags() []string
    Invoke(ctx context.Context, state S) (S, error)
    Validate() error
}
```

**特性**:
- 泛型接口，类型安全
- 元数据支持（名称、描述、标签）
- 标准化的执行和验证

### 2. Metadata - 节点元数据

```go
meta := NewMetadata("process").
    WithDescription("Process data").
    WithTags("processing", "critical").
    WithVersion("1.0.0").
    WithExtra("timeout", 30)
```

**特性**:
- 链式调用
- 可克隆
- 可验证
- 支持额外数据

### 3. FunctionNode - 函数节点

```go
node := NewFunctionNode("increment", 
    func(ctx context.Context, s State) (State, error) {
        s.Counter++
        return s, nil
    },
    WithDescription("Increment counter"),
    WithTags("math", "counter"),
)

result, err := node.Invoke(ctx, State{Counter: 0})
```

**高级功能**:

#### Chain - 链接节点

```go
add := NewFunctionNode("add", addFunc)
multiply := NewFunctionNode("multiply", multiplyFunc)
chained := add.Chain(multiplyFunc)
// chained 先执行 addFunc，再执行 multiplyFunc
```

#### Retry - 重试逻辑

```go
node := NewFunctionNode("api_call", apiCallFunc)
retryNode := node.Retry(3) // 最多重试 3 次
```

#### Fallback - 降级

```go
primaryNode := NewFunctionNode("primary", primaryFunc)
fallbackNode := primaryNode.Fallback(func(ctx context.Context, s State) (State, error) {
    s.Message = "Using fallback"
    return s, nil
})
```

#### Transform - 转换输出

```go
node := NewFunctionNode("process", processFunc)
transformed := node.Transform(func(ctx context.Context, s State) (State, error) {
    s.Value = s.Value * 2 // 将输出值翻倍
    return s, nil
})
```

#### Conditional - 条件执行

```go
node := NewFunctionNode("expensive", expensiveFunc)
conditional := node.Conditional(func(ctx context.Context, s State) bool {
    return s.NeedsProcessing
})
```

### 4. SubgraphNode - 子图节点

```go
// 定义状态类型
type ParentState struct {
    Data map[string]any
}

type ChildState struct {
    Value int
}

// 创建子图
subgraph := state.NewStateGraph[ChildState]("sub")
// ... 配置子图
compiled, _ := subgraph.Compile()

// 创建子图节点
subgraphNode := NewSubgraphNode[ParentState, ChildState](
    "nested",
    compiled,
    WithStateMapper(
        // 父状态 -> 子状态
        func(parent ParentState) (ChildState, error) {
            return ChildState{Value: parent.Data["value"].(int)}, nil
        },
        // 合并子状态 -> 父状态
        func(parent ParentState, child ChildState) (ParentState, error) {
            parent.Data["result"] = child.Value
            return parent, nil
        },
    ),
)
```

**特性**:
- 支持不同状态类型
- 灵活的状态映射
- 完整的错误处理
- Context 支持

---

## 📁 文件结构

```
graph/node/
├── doc.go              # 包文档
├── interface.go        # Node 接口和元数据 (200+ 行)
├── function.go         # FunctionNode 实现 (300+ 行)
├── function_test.go    # FunctionNode 测试 (450+ 行)
├── subgraph.go         # SubgraphNode 实现 (180+ 行)
└── subgraph_test.go    # SubgraphNode 测试 (300+ 行)
```

**代码统计**:
- 实现代码: ~680 行
- 测试代码: ~750 行
- 文档注释: ~250 行
- **总计**: ~1680 行

---

## ✅ 测试结果

### 测试覆盖率: 89.8%

```bash
$ go test -v ./graph/node -cover

27 个测试全部通过：

Metadata:
- TestNewMetadata
- TestMetadata_WithMethods
- TestMetadata_Clone
- TestMetadata_Validate

FunctionNode:
- TestNewFunctionNode (基础和选项)
- TestFunctionNode_Invoke (执行和错误)
- TestFunctionNode_Validate
- TestFunctionNode_Chain (链接)
- TestFunctionNode_Retry (重试成功和失败)
- TestFunctionNode_Fallback (降级)
- TestFunctionNode_Transform (转换)
- TestFunctionNode_Conditional (条件执行)
- TestFunctionNode_WithFunc (替换函数)

SubgraphNode:
- TestNewSubgraphNode
- TestSubgraphNode_Invoke (执行和选项)
- TestSubgraphNode_MapToChild_Error
- TestSubgraphNode_Subgraph_Error
- TestSubgraphNode_MapFromChild_Error
- TestSubgraphNode_Validate
- TestSubgraphNode_ContextCancellation
- TestSubgraphNode_ComplexMapping

coverage: 89.8% of statements
ok  	langchain-go/graph/node	0.540s
```

---

## 🎓 使用示例

### 示例 1: 基础函数节点

```go
package main

import (
    "context"
    "fmt"
    "langchain-go/graph/node"
)

type MyState struct {
    Counter int
    Message string
}

func main() {
    // 创建节点
    incrementNode := node.NewFunctionNode("increment",
        func(ctx context.Context, s MyState) (MyState, error) {
            s.Counter++
            s.Message = fmt.Sprintf("Counter is now %d", s.Counter)
            return s, nil
        },
        node.WithDescription("Increment counter"),
        node.WithTags("math"),
    )

    // 执行节点
    result, _ := incrementNode.Invoke(context.Background(), MyState{Counter: 5})
    fmt.Println(result.Counter)  // 输出: 6
    fmt.Println(result.Message)  // 输出: Counter is now 6
}
```

### 示例 2: 重试逻辑

```go
// 模拟不稳定的 API 调用
attempts := 0
apiNode := node.NewFunctionNode("api_call",
    func(ctx context.Context, s MyState) (MyState, error) {
        attempts++
        if attempts < 3 {
            return s, errors.New("temporary network error")
        }
        s.Message = "Success!"
        return s, nil
    },
)

// 添加重试
retryNode := apiNode.Retry(5)

result, err := retryNode.Invoke(context.Background(), MyState{})
// 成功（经过 3 次尝试）
fmt.Println(result.Message) // 输出: Success!
fmt.Println(attempts)        // 输出: 3
```

### 示例 3: 降级策略

```go
// 主节点（可能失败）
primaryNode := node.NewFunctionNode("fetch_from_db",
    func(ctx context.Context, s MyState) (MyState, error) {
        // 假设数据库不可用
        return s, errors.New("database connection failed")
    },
)

// 降级节点
fallbackNode := primaryNode.Fallback(
    func(ctx context.Context, s MyState) (MyState, error) {
        s.Message = "Using cached data"
        s.Counter = 100 // 从缓存获取
        return s, nil
    },
)

result, _ := fallbackNode.Invoke(context.Background(), MyState{})
fmt.Println(result.Message) // 输出: Using cached data
fmt.Println(result.Counter) // 输出: 100
```

### 示例 4: 链接节点

```go
// 第一步：验证
validateNode := node.NewFunctionNode("validate",
    func(ctx context.Context, s MyState) (MyState, error) {
        if s.Counter < 0 {
            return s, errors.New("counter cannot be negative")
        }
        return s, nil
    },
)

// 第二步：处理
processFunc := func(ctx context.Context, s MyState) (MyState, error) {
    s.Counter *= 2
    return s, nil
}

// 链接
pipeline := validateNode.Chain(processFunc)

result, _ := pipeline.Invoke(context.Background(), MyState{Counter: 10})
fmt.Println(result.Counter) // 输出: 20
```

### 示例 5: 条件执行

```go
expensiveNode := node.NewFunctionNode("expensive_operation",
    func(ctx context.Context, s MyState) (MyState, error) {
        // 假设这是一个很昂贵的操作
        time.Sleep(time.Second)
        s.Counter += 1000
        return s, nil
    },
)

// 只有当 Counter > 100 时才执行
conditionalNode := expensiveNode.Conditional(
    func(ctx context.Context, s MyState) bool {
        return s.Counter > 100
    },
)

// Counter = 50, 不执行
result1, _ := conditionalNode.Invoke(context.Background(), MyState{Counter: 50})
fmt.Println(result1.Counter) // 输出: 50 (未改变)

// Counter = 150, 执行
result2, _ := conditionalNode.Invoke(context.Background(), MyState{Counter: 150})
fmt.Println(result2.Counter) // 输出: 1150 (150 + 1000)
```

### 示例 6: 子图节点

```go
type ParentState struct {
    Input  int
    Output int
}

type ChildState struct {
    Value int
}

// 创建子图（假设已实现）
subgraph := createProcessingSubgraph() // 返回 SubgraphExecutor[ChildState]

// 创建子图节点
subgraphNode := node.NewSubgraphNode[ParentState, ChildState](
    "process_subgraph",
    subgraph,
    node.WithStateMapper(
        // 父 -> 子
        func(parent ParentState) (ChildState, error) {
            return ChildState{Value: parent.Input}, nil
        },
        // 子 -> 父
        func(parent ParentState, child ChildState) (ParentState, error) {
            parent.Output = child.Value
            return parent, nil
        },
    ),
)

result, _ := subgraphNode.Invoke(context.Background(), ParentState{Input: 10})
fmt.Println(result.Output) // 子图处理后的结果
```

---

## 🔧 技术特点

### 1. 泛型设计

所有节点类型都使用泛型，提供类型安全：

```go
type Node[S any] interface {
    Invoke(ctx context.Context, state S) (S, error)
}

type FunctionNode[S any] struct {
    fn NodeFunc[S]
}

type SubgraphNode[ParentState, ChildState any] struct {
    // 支持不同的状态类型
}
```

### 2. 不可变风格

节点函数返回新状态而不是修改原状态：

```go
func processNode(ctx context.Context, s State) (State, error) {
    // 不修改 s，返回新状态
    newState := s
    newState.Counter++
    return newState, nil
}
```

### 3. 组合优于继承

使用函数组合实现高级功能：

```go
// 不是继承，而是包装
node.Retry(3).Fallback(fallbackFunc).Transform(transformFunc)
```

### 4. Context 支持

所有节点都支持 Context：

- 超时控制
- 取消传播
- 请求级别数据

### 5. 元数据系统

完整的元数据支持：

- 描述
- 标签
- 版本
- 自定义数据

---

## 📊 性能特性

### 内存使用

- 节点结构轻量级
- 元数据按需克隆
- 无不必要的分配

### 执行效率

- 直接函数调用
- 最小的运行时开销
- 支持并发执行（线程安全）

---

## 🔮 后续工作

### M30-M32: Edge 系统

- [ ] Edge 定义标准化
- [ ] Conditional Edge 实现
- [ ] Router 路由器

### 集成 StateGraph

- [ ] 将 Node 系统集成到 StateGraph
- [ ] 替换当前的简单 NodeFunc
- [ ] 支持节点元数据在图中的传播

---

## 🎯 设计决策

### 1. 接口 vs 具体类型

**选择**: 定义 Node 接口

**理由**:
- 支持多种节点类型
- 便于测试（Mock）
- 扩展性好

### 2. 包装器模式

**选择**: Retry, Fallback 等返回新节点

**理由**:
- 不可变性
- 可组合
- 清晰的依赖关系

### 3. 状态映射

**选择**: SubgraphNode 需要显式的状态映射

**理由**:
- 类型安全
- 灵活性
- 清晰的接口边界

---

## 📚 参考资源

- [Python LangGraph Nodes](https://github.com/langchain-ai/langgraph)
- [设计方案](../../LangChain-LangGraph-Go重写设计方案.md)
- [StateGraph 总结](./M24-M26-StateGraph-Summary.md)

---

## 🎉 里程碑

- ✅ Node 系统完成
- ✅ 89.8% 测试覆盖率
- ✅ 27 个测试全部通过
- ✅ 1680+ 行高质量代码
- ✅ 完整的高级功能（Retry、Fallback、Chain等）

**Phase 2 进度**: 6/29 模块完成 (21%)

**下一步**: M30-M32 Edge 系统 🚀

---

**完成日期**: 2026-01-14  
**创建者**: AI Assistant  
**状态**: ✅ 已完成
