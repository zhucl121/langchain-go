// Package node 提供 LangGraph 节点系统实现。
//
// Node 是 StateGraph 中执行具体逻辑的单元。
// 节点接收状态，执行处理，返回新状态。
//
// # 节点类型
//
// node 包提供多种节点类型：
//   - FunctionNode: 基于函数的节点
//   - SubgraphNode: 嵌套图节点
//   - PrebuiltNode: 预构建节点（如 ToolNode）
//
// # 基本使用
//
// 使用函数创建节点：
//
//	fn := func(ctx context.Context, state MyState) (MyState, error) {
//	    state.Counter++
//	    return state, nil
//	}
//
//	node := node.NewFunctionNode("increment", fn)
//
// 使用节点接口：
//
//	result, err := node.Invoke(ctx, initialState)
//
// # 节点元数据
//
// 节点可以包含元数据用于日志、监控和调试：
//
//	node := node.NewFunctionNode("process", fn,
//	    node.WithDescription("处理数据"),
//	    node.WithTags("processing", "critical"),
//	)
//
// # 子图节点
//
// 子图节点允许嵌套状态图：
//
//	subgraph := state.NewStateGraph[SubState]("sub")
//	// ... 配置子图
//
//	subgraphNode := node.NewSubgraphNode("nested", subgraph,
//	    node.WithStateMapper(mapParentToSub, mapSubToParent),
//	)
//
// # 并发安全
//
// 节点实现是并发安全的，可以在多个 goroutine 中同时执行。
//
package node
