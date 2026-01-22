// Package discovery 提供服务发现功能。
//
// 本包实现了基于不同后端的服务发现机制，支持：
// - Consul 服务发现
// - Etcd 服务发现（可选）
// - 节点自动注册和注销
// - 节点变化监听
// - 健康检查集成
//
// 示例:
//
//	// 创建 Consul 服务发现
//	config := discovery.ConsulConfig{
//	    Address:     "localhost:8500",
//	    ServiceName: "langchain-go",
//	    CheckTTL:    10 * time.Second,
//	}
//	disco, err := discovery.NewConsulDiscovery(config)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer disco.Close()
//
//	// 注册节点
//	node := &node.Node{
//	    ID:      "node-1",
//	    Name:    "worker-1",
//	    Address: "192.168.1.10",
//	    Port:    8080,
//	    Status:  node.StatusOnline,
//	    Roles:   []node.NodeRole{node.RoleWorker},
//	}
//	err = disco.RegisterNode(ctx, node)
//
//	// 监听节点变化
//	events, err := disco.Watch(ctx)
//	for event := range events {
//	    fmt.Printf("Node %s: %s\n", event.Type, event.Node.ID)
//	}
package discovery
