// Package state 提供 LangGraph StateGraph 状态图实现。
//
// StateGraph 是 LangGraph 的核心类型，用于定义有向图工作流。
// 它允许将复杂的 AI Agent 逻辑表达为节点和边的图结构。
//
// # 核心概念
//
// StateGraph 基于以下核心概念：
//   - State: 泛型状态类型，在节点间流转
//   - Node: 处理状态的函数节点
//   - Edge: 连接节点的边（普通边和条件边）
//   - Checkpoint: 状态持久化（支持 Time Travel）
//   - Channel: 状态通道（用于复杂状态更新）
//
// # 基本使用
//
// 创建一个简单的状态图：
//
//	type MyState struct {
//	    Counter int
//	    Message string
//	}
//
//	graph := state.NewStateGraph[MyState]("my-graph")
//
//	// 添加节点
//	graph.AddNode("start", func(ctx context.Context, s MyState) (MyState, error) {
//	    s.Counter++
//	    return s, nil
//	})
//
//	// 设置入口和边
//	graph.SetEntryPoint("start")
//	graph.AddEdge("start", state.END)
//
//	// 编译并执行
//	compiled, _ := graph.Compile()
//	result, _ := compiled.Invoke(ctx, MyState{Counter: 0})
//
// # 条件边
//
// 使用条件边实现动态路由：
//
//	graph.AddNode("agent", agentNode)
//	graph.AddNode("tools", toolsNode)
//
//	graph.AddConditionalEdges("agent",
//	    func(s MyState) string {
//	        if s.NeedTools {
//	            return "continue"
//	        }
//	        return "end"
//	    },
//	    map[string]string{
//	        "continue": "tools",
//	        "end":      state.END,
//	    },
//	)
//
// # Checkpointing (持久化)
//
// 配置检查点支持 Time Travel 和恢复：
//
//	checkpointer := checkpoint.NewMemorySaver()
//	graph.WithCheckpointer(checkpointer).
//	    WithDurability(durability.ModeSync)
//
//	compiled, _ := graph.Compile()
//	result, _ := compiled.Invoke(ctx, initialState,
//	    execute.WithThreadID("user-123"))
//
//	// 查看历史
//	history, _ := compiled.GetHistory(ctx, "user-123", 10)
//
// # 特殊常量
//
// StateGraph 定义了两个特殊的节点名称：
//   - START: 图的起始点（虚拟节点）
//   - END: 图的结束点（虚拟节点）
//
// # 参考
//
//   - Python LangGraph: https://github.com/langchain-ai/langgraph
//   - 设计文档: ../../LangChain-LangGraph-Go重写设计方案.md
//
package state
