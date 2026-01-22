// Package node 提供集群节点管理功能。
//
// 本包定义了集群中节点的核心数据结构和管理接口，支持：
// - 节点注册和注销
// - 节点状态管理
// - 节点容量和负载监控
// - 节点事件监听
//
// 示例:
//
//	// 创建节点
//	node := &node.Node{
//	    ID:      "node-1",
//	    Name:    "worker-1",
//	    Address: "192.168.1.10",
//	    Port:    8080,
//	    Status:  node.StatusOnline,
//	    Roles:   []node.NodeRole{node.RoleWorker},
//	    Capacity: node.Capacity{
//	        MaxConnections: 1000,
//	        MaxQPS:        500,
//	        MaxMemoryMB:   4096,
//	    },
//	}
//
//	// 注册节点
//	err := manager.RegisterNode(ctx, node)
//
//	// 监听节点变化
//	events, err := manager.Watch(ctx)
//	for event := range events {
//	    fmt.Printf("Node event: %s\n", event.Type)
//	}
package node
