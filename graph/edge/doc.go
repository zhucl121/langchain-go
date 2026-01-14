// Package edge 提供 LangGraph 边系统实现。
//
// Edge 定义了状态图中节点之间的连接和转换逻辑。
// 边控制着执行流程，决定下一个执行哪个节点。
//
// # 边类型
//
// edge 包提供多种边类型：
//   - NormalEdge: 普通边（静态连接）
//   - ConditionalEdge: 条件边（动态路由）
//   - BranchEdge: 分支边（多路分支）
//
// # 基本使用
//
// 普通边：
//
//	edge := edge.NewNormalEdge("node1", "node2")
//	nextNode := edge.GetTarget() // "node2"
//
// 条件边：
//
//	condEdge := edge.NewConditionalEdge("router",
//	    func(state MyState) string {
//	        if state.Done {
//	            return "end"
//	        }
//	        return "continue"
//	    },
//	    map[string]string{
//	        "continue": "process",
//	        "end":      state.END,
//	    },
//	)
//
//	nextNode := condEdge.Route(state) // 根据状态动态决定
//
// # Router
//
// Router 提供更高级的路由逻辑：
//
//	router := edge.NewRouter[MyState]()
//	router.AddRoute("path1", "node1", func(s MyState) bool {
//	    return s.Counter > 0
//	})
//	router.AddRoute("path2", "node2", func(s MyState) bool {
//	    return s.Counter <= 0
//	})
//
//	nextNode := router.Route(state)
//
// # 并发安全
//
// 边实现是并发安全的，可以在多个 goroutine 中同时使用。
//
package edge
