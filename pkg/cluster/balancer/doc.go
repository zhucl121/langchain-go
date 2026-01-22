// Package balancer 提供负载均衡功能。
//
// 本包实现了多种负载均衡策略，用于在集群节点间分发请求：
// - Round Robin（轮询）
// - Least Connection（最少连接）
// - Weighted（加权）
// - Consistent Hash（一致性哈希）
// - Adaptive（自适应）
//
// 示例:
//
//	// 创建轮询负载均衡器
//	lb := balancer.NewRoundRobinBalancer()
//
//	// 更新节点列表
//	nodes := []*node.Node{node1, node2, node3}
//	lb.UpdateNodes(nodes)
//
//	// 选择节点处理请求
//	req := &balancer.Request{
//	    ID:   "req-123",
//	    Type: balancer.RequestTypeLLM,
//	}
//	selectedNode, err := lb.SelectNode(ctx, req)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// 记录请求结果
//	lb.RecordResult(selectedNode.ID, true, 100*time.Millisecond)
package balancer
