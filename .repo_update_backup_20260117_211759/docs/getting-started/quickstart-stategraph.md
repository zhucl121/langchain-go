# StateGraph 快速开始

本指南将帮助您快速上手 LangChain-Go 的 StateGraph（状态图）功能。

---

## 安装

确保您的项目使用 Go 1.22+：

```bash
go get github.com/zhucl121/langchain-go
```

---

## 基础概念

### StateGraph 是什么？

StateGraph 是一个有向图工作流引擎，用于构建复杂的 AI Agent 逻辑。它允许您：

- 定义节点（处理状态的函数）
- 定义边（节点之间的连接）
- 使用条件边实现动态路由
- 实现循环和复杂的控制流

### 核心组件

1. **State（状态）**: 在节点间流转的数据
2. **Node（节点）**: 处理状态的函数
3. **Edge（边）**: 连接节点的路径
4. **Conditional Edge（条件边）**: 基于状态的动态路由

---

## 快速示例

### 示例 1: Hello World

最简单的状态图示例：

```go
package main

import (
    "context"
    "fmt"
    "github.com/zhucl121/langchain-go/graph/state"
)

type MyState struct {
    Message string
}

func main() {
    // 创建状态图
    graph := state.NewStateGraph[MyState]("hello")

    // 添加节点
    graph.AddNode("greet", func(ctx context.Context, s MyState) (MyState, error) {
        s.Message = "Hello, " + s.Message + "!"
        return s, nil
    })

    // 设置入口和结束
    graph.SetEntryPoint("greet")
    graph.AddEdge("greet", state.END)

    // 编译并执行
    compiled, _ := graph.Compile()
    result, _ := compiled.Invoke(context.Background(), MyState{Message: "World"})

    fmt.Println(result.Message) // 输出: Hello, World!
}
```

### 示例 2: 多节点链

```go
type ProcessState struct {
    Value int
    Log   []string
}

func main() {
    graph := state.NewStateGraph[ProcessState]("chain")

    // 节点 1: 加法
    graph.AddNode("add", func(ctx context.Context, s ProcessState) (ProcessState, error) {
        s.Value += 10
        s.Log = append(s.Log, "Added 10")
        return s, nil
    })

    // 节点 2: 乘法
    graph.AddNode("multiply", func(ctx context.Context, s ProcessState) (ProcessState, error) {
        s.Value *= 2
        s.Log = append(s.Log, "Multiplied by 2")
        return s, nil
    })

    // 连接节点
    graph.SetEntryPoint("add")
    graph.AddEdge("add", "multiply")
    graph.AddEdge("multiply", state.END)

    // 执行
    compiled, _ := graph.Compile()
    result, _ := compiled.Invoke(context.Background(), ProcessState{Value: 5})

    fmt.Println(result.Value) // 输出: 30 ((5+10)*2)
    fmt.Println(result.Log)   // 输出: [Added 10, Multiplied by 2]
}
```

### 示例 3: 条件分支

```go
type RouterState struct {
    Number  int
    Message string
}

func main() {
    graph := state.NewStateGraph[RouterState]("router")

    // 检查节点
    graph.AddNode("check", func(ctx context.Context, s RouterState) (RouterState, error) {
        // 只是传递状态，实际路由由条件边决定
        return s, nil
    })

    // 正数路径
    graph.AddNode("positive", func(ctx context.Context, s RouterState) (RouterState, error) {
        s.Message = "Number is positive"
        return s, nil
    })

    // 负数路径
    graph.AddNode("negative", func(ctx context.Context, s RouterState) (RouterState, error) {
        s.Message = "Number is negative"
        return s, nil
    })

    // 零路径
    graph.AddNode("zero", func(ctx context.Context, s RouterState) (RouterState, error) {
        s.Message = "Number is zero"
        return s, nil
    })

    // 设置入口
    graph.SetEntryPoint("check")

    // 添加条件边
    graph.AddConditionalEdges("check",
        func(s RouterState) string {
            if s.Number > 0 {
                return "pos"
            } else if s.Number < 0 {
                return "neg"
            }
            return "zero"
        },
        map[string]string{
            "pos":  "positive",
            "neg":  "negative",
            "zero": "zero",
        },
    )

    // 所有分支都连接到 END
    graph.AddEdge("positive", state.END)
    graph.AddEdge("negative", state.END)
    graph.AddEdge("zero", state.END)

    // 测试
    compiled, _ := graph.Compile()

    result1, _ := compiled.Invoke(context.Background(), RouterState{Number: 10})
    fmt.Println(result1.Message) // 输出: Number is positive

    result2, _ := compiled.Invoke(context.Background(), RouterState{Number: -5})
    fmt.Println(result2.Message) // 输出: Number is negative
}
```

### 示例 4: 循环（自循环）

```go
type CounterState struct {
    Count int
    Max   int
}

func main() {
    graph := state.NewStateGraph[CounterState]("loop")

    // 计数节点
    graph.AddNode("increment", func(ctx context.Context, s CounterState) (CounterState, error) {
        s.Count++
        fmt.Printf("Count: %d\n", s.Count)
        return s, nil
    })

    // 设置入口
    graph.SetEntryPoint("increment")

    // 条件边：决定是继续循环还是结束
    graph.AddConditionalEdges("increment",
        func(s CounterState) string {
            if s.Count >= s.Max {
                return "done"
            }
            return "continue"
        },
        map[string]string{
            "continue": "increment", // 自循环
            "done":     state.END,
        },
    )

    // 执行
    compiled, _ := graph.Compile()
    result, _ := compiled.Invoke(context.Background(), CounterState{Count: 0, Max: 5})

    fmt.Printf("Final count: %d\n", result.Count) // 输出: Final count: 5
}
```

---

## 高级特性

### Context 和取消

```go
func main() {
    graph := state.NewStateGraph[MyState]("cancelable")

    graph.AddNode("work", func(ctx context.Context, s MyState) (MyState, error) {
        // 检查 context 是否被取消
        select {
        case <-ctx.Done():
            return s, ctx.Err()
        default:
            // 执行工作
            s.Value++
            return s, nil
        }
    })

    graph.SetEntryPoint("work")
    graph.AddEdge("work", state.END)

    // 使用带超时的 context
    ctx, cancel := context.WithTimeout(context.Background(), time.Second)
    defer cancel()

    compiled, _ := graph.Compile()
    result, err := compiled.Invoke(ctx, MyState{})

    if err != nil {
        fmt.Println("Error:", err)
    }
}
```

### 错误处理

```go
func main() {
    graph := state.NewStateGraph[MyState]("error-handling")

    graph.AddNode("risky", func(ctx context.Context, s MyState) (MyState, error) {
        if s.Value < 0 {
            return s, errors.New("value cannot be negative")
        }
        s.Value++
        return s, nil
    })

    graph.SetEntryPoint("risky")
    graph.AddEdge("risky", state.END)

    compiled, _ := graph.Compile()

    // 正常执行
    result1, err1 := compiled.Invoke(context.Background(), MyState{Value: 10})
    fmt.Println("Result:", result1.Value, "Error:", err1)

    // 错误情况
    result2, err2 := compiled.Invoke(context.Background(), MyState{Value: -5})
    fmt.Println("Result:", result2.Value, "Error:", err2) // Error: value cannot be negative
}
```

---

## 实战案例

### 案例: 简单的聊天 Agent

```go
type AgentState struct {
    Messages []string
    Done     bool
}

func main() {
    graph := state.NewStateGraph[AgentState]("chat-agent")

    // Agent 节点：处理消息
    graph.AddNode("agent", func(ctx context.Context, s AgentState) (AgentState, error) {
        lastMessage := s.Messages[len(s.Messages)-1]

        // 简单的响应逻辑
        if strings.Contains(lastMessage, "bye") {
            s.Messages = append(s.Messages, "Goodbye!")
            s.Done = true
        } else {
            s.Messages = append(s.Messages, "I received: "+lastMessage)
        }

        return s, nil
    })

    graph.SetEntryPoint("agent")
    graph.AddConditionalEdges("agent",
        func(s AgentState) string {
            if s.Done {
                return "end"
            }
            return "continue"
        },
        map[string]string{
            "continue": "agent",
            "end":      state.END,
        },
    )

    compiled, _ := graph.Compile()

    // 执行对话
    result, _ := compiled.Invoke(context.Background(), AgentState{
        Messages: []string{"Hello", "How are you?", "bye"},
    })

    for _, msg := range result.Messages {
        fmt.Println(msg)
    }
}
```

---

## 常见问题

### Q: 如何避免无限循环？

**A**: 使用条件边和终止条件：

```go
graph.AddConditionalEdges("loop_node",
    func(s State) string {
        if s.Iteration >= maxIterations {
            return "end"  // 达到最大迭代次数，结束
        }
        return "continue"
    },
    map[string]string{
        "continue": "loop_node",
        "end":      state.END,
    },
)
```

### Q: 如何在节点间共享数据？

**A**: 使用状态：

```go
type SharedState struct {
    SharedData map[string]any
    CurrentStep string
}

// 节点 1 写入数据
func node1(ctx context.Context, s SharedState) (SharedState, error) {
    s.SharedData["key"] = "value"
    return s, nil
}

// 节点 2 读取数据
func node2(ctx context.Context, s SharedState) (SharedState, error) {
    value := s.SharedData["key"]
    fmt.Println(value) // 输出: value
    return s, nil
}
```

### Q: 如何处理复杂的状态更新？

**A**: 使用 Channel 和 Reducer（高级特性）：

```go
// 追加通道（用于列表）
appendChannel := state.NewAppendChannel("messages")
result, _ := appendChannel.Update([]any{"a", "b"}, "c")
// result == []any{"a", "b", "c"}

// 合并 Reducer（用于 map）
mergeReducer := state.MergeReducer()
result := mergeReducer(
    map[string]any{"a": 1},
    map[string]any{"b": 2},
)
// result == {"a": 1, "b": 2}
```

---

## 下一步

- 学习 [Node 系统](./node-examples.md)（即将推出）
- 学习 [Checkpoint 持久化](./checkpoint-examples.md)（即将推出）
- 学习 [Human-in-the-Loop](./hitl-examples.md)（即将推出）
- 查看 [完整 API 文档](./M24-M26-StateGraph-Summary.md)

---

## 更多资源

- [项目进度](../PROJECT-PROGRESS.md)
- [设计方案](../../LangChain-LangGraph-Go重写设计方案.md)
- [Phase 2 规划](./Phase2-Planning.md)

---

**最后更新**: 2026-01-14
