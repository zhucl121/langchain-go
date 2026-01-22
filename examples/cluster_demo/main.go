package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/zhucl121/langchain-go/pkg/cluster/discovery"
	"github.com/zhucl121/langchain-go/pkg/cluster/health"
	"github.com/zhucl121/langchain-go/pkg/cluster/node"
)

func main() {
	fmt.Println("🚀 LangChain-Go 集群管理示例")
	fmt.Println("========================================")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 检查 Consul 是否可用
	consulAddr := os.Getenv("CONSUL_ADDR")
	if consulAddr == "" {
		consulAddr = "localhost:8500"
	}

	fmt.Printf("\n📡 连接到 Consul: %s\n", consulAddr)
	fmt.Println("提示: 如果 Consul 未运行，请执行:")
	fmt.Println("  docker run -d --name consul -p 8500:8500 consul:latest")

	// 1. 创建服务发现
	disco, err := discovery.NewConsulDiscovery(discovery.ConsulConfig{
		Address:         consulAddr,
		ServiceName:     "langchain-go-demo",
		CheckTTL:        10 * time.Second,
		DeregisterAfter: 30 * time.Second,
	})
	if err != nil {
		log.Printf("⚠️  无法连接到 Consul: %v", err)
		log.Println("运行模拟模式...")
		runSimulationMode()
		return
	}
	defer disco.Close()

	fmt.Println("✅ 成功连接到 Consul")

	// 2. 创建本地节点
	localNode := &node.Node{
		ID:      fmt.Sprintf("worker-%d", time.Now().Unix()),
		Name:    "demo-worker",
		Address: "127.0.0.1",
		Port:    8080,
		Status:  node.StatusOnline,
		Roles:   []node.NodeRole{node.RoleWorker},
		Capacity: node.Capacity{
			MaxConnections: 1000,
			MaxQPS:         500,
			MaxMemoryMB:    4096,
		},
		Load: node.Load{
			CurrentConnections: 0,
			CPUUsagePercent:    0,
			MemoryUsageMB:      512,
		},
		Metadata: map[string]string{
			"name":            "demo-worker",
			"version":         "0.5.0",
			"max_connections": "1000",
		},
		Region: "us-east-1",
		Zone:   "us-east-1a",
	}

	// 3. 注册节点
	fmt.Printf("\n📝 注册节点: %s\n", localNode.ID)
	if err := disco.RegisterNode(ctx, localNode); err != nil {
		log.Printf("❌ 注册节点失败: %v", err)
		log.Println("运行模拟模式...")
		disco.Close()
		runSimulationMode()
		return
	}
	fmt.Println("✅ 节点注册成功")

	// 4. 启动心跳
	fmt.Println("\n💓 启动心跳...")
	stopHeartbeat := make(chan struct{})
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if err := disco.Heartbeat(ctx, localNode.ID); err != nil {
					log.Printf("⚠️  心跳失败: %v", err)
				} else {
					fmt.Println("💓 心跳发送成功")
				}
			case <-stopHeartbeat:
				return
			}
		}
	}()

	// 5. 监听节点变化
	fmt.Println("\n👀 监听集群节点变化...")
	events, err := disco.Watch(ctx)
	if err != nil {
		log.Fatalf("❌ 监听失败: %v", err)
	}

	go func() {
		for event := range events {
			switch event.Type {
			case node.EventNodeJoined:
				fmt.Printf("➕ 节点加入: %s (%s)\n", event.Node.Name, event.Node.ID)
			case node.EventNodeLeft:
				fmt.Printf("➖ 节点离开: %s (%s)\n", event.Node.Name, event.Node.ID)
			case node.EventNodeUpdated:
				fmt.Printf("🔄 节点更新: %s (%s)\n", event.Node.Name, event.Node.ID)
			}
		}
	}()

	// 6. 列出所有节点
	time.Sleep(2 * time.Second)
	fmt.Println("\n📋 当前集群节点:")
	nodes, err := disco.ListNodes(ctx, nil)
	if err != nil {
		log.Printf("❌ 获取节点列表失败: %v", err)
	} else {
		for i, n := range nodes {
			fmt.Printf("  %d. %s (%s) - %s:%d - 状态: %s\n",
				i+1, n.Name, n.ID, n.Address, n.Port, n.Status)
		}
	}

	// 7. 健康检查示例
	fmt.Println("\n🏥 健康检查示例:")
	demoHealthCheck()

	// 8. 等待中断信号
	fmt.Println("\n✅ 集群管理系统运行中...")
	fmt.Println("按 Ctrl+C 退出")

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	// 9. 清理
	fmt.Println("\n🛑 正在退出...")
	close(stopHeartbeat)

	if err := disco.UnregisterNode(ctx, localNode.ID); err != nil {
		log.Printf("⚠️  注销节点失败: %v", err)
	} else {
		fmt.Println("✅ 节点已注销")
	}

	fmt.Println("👋 再见！")
}

// demoHealthCheck 演示健康检查功能
func demoHealthCheck() {
	// 创建测试节点
	testNode := &node.Node{
		ID:      "test-node",
		Name:    "test",
		Address: "127.0.0.1",
		Port:    8080,
		Status:  node.StatusOnline,
		Roles:   []node.NodeRole{node.RoleWorker},
	}

	// TCP 健康检查
	tcpChecker := health.NewTCPChecker(health.TCPConfig{
		Timeout:    3 * time.Second,
		RetryCount: 1,
	})

	ctx := context.Background()
	result, _ := tcpChecker.Check(ctx, testNode)
	if result.Healthy {
		fmt.Printf("  ✅ TCP 检查通过 (延迟: %v)\n", result.Latency)
	} else {
		fmt.Printf("  ⚠️  TCP 检查失败: %s\n", result.Message)
	}

	// HTTP 健康检查
	httpChecker := health.NewHTTPChecker(health.HTTPConfig{
		Endpoint: "/health",
		Timeout:  5 * time.Second,
		Scheme:   "http",
	})

	result, _ = httpChecker.Check(ctx, testNode)
	if result.Healthy {
		fmt.Printf("  ✅ HTTP 检查通过 (延迟: %v)\n", result.Latency)
	} else {
		fmt.Printf("  ⚠️  HTTP 检查失败: %s\n", result.Message)
	}
}

// runSimulationMode 运行模拟模式（不需要 Consul）
func runSimulationMode() {
	fmt.Println("\n🎭 模拟模式")
	fmt.Println("========================================")

	// 创建模拟节点
	nodes := []*node.Node{
		{
			ID:      "worker-1",
			Name:    "worker-1",
			Address: "192.168.1.10",
			Port:    8080,
			Status:  node.StatusOnline,
			Roles:   []node.NodeRole{node.RoleWorker},
			Capacity: node.Capacity{
				MaxConnections: 1000,
				MaxQPS:         500,
				MaxMemoryMB:    4096,
			},
			Load: node.Load{
				CurrentConnections: 500,
				CPUUsagePercent:    45.0,
				MemoryUsageMB:      2048,
			},
		},
		{
			ID:      "worker-2",
			Name:    "worker-2",
			Address: "192.168.1.11",
			Port:    8080,
			Status:  node.StatusBusy,
			Roles:   []node.NodeRole{node.RoleWorker},
			Capacity: node.Capacity{
				MaxConnections: 1000,
				MaxQPS:         500,
				MaxMemoryMB:    4096,
			},
			Load: node.Load{
				CurrentConnections: 850,
				CPUUsagePercent:    80.0,
				MemoryUsageMB:      3500,
			},
		},
		{
			ID:      "cache-1",
			Name:    "cache-1",
			Address: "192.168.1.20",
			Port:    6379,
			Status:  node.StatusOnline,
			Roles:   []node.NodeRole{node.RoleCache},
			Capacity: node.Capacity{
				MaxConnections: 10000,
				MaxMemoryMB:    8192,
			},
			Load: node.Load{
				CurrentConnections: 2000,
				MemoryUsageMB:      4096,
			},
		},
	}

	// 显示节点信息
	fmt.Println("\n📋 模拟集群节点:")
	for i, n := range nodes {
		fmt.Printf("\n%d. %s (%s)\n", i+1, n.Name, n.ID)
		fmt.Printf("   地址: %s:%d\n", n.Address, n.Port)
		fmt.Printf("   状态: %s\n", n.Status)
		fmt.Printf("   角色: %v\n", n.Roles)
		fmt.Printf("   容量: %d 连接, %d QPS, %d MB 内存\n",
			n.Capacity.MaxConnections, n.Capacity.MaxQPS, n.Capacity.MaxMemoryMB)
		fmt.Printf("   负载: %d 连接 (%.1f%%), CPU %.1f%%, 内存 %d MB\n",
			n.Load.CurrentConnections,
			n.GetLoadPercent(),
			n.Load.CPUUsagePercent,
			n.Load.MemoryUsageMB)
		if n.IsHealthy() {
			fmt.Println("   健康: ✅ 健康")
		} else {
			fmt.Println("   健康: ⚠️  不健康")
		}
	}

	// 演示节点过滤
	fmt.Println("\n🔍 节点过滤示例:")

	// 只显示在线节点
	filter := node.NewNodeFilter().WithStatus(node.StatusOnline)
	onlineNodes := filter.MatchAny(nodes)
	fmt.Printf("\n在线节点 (%d 个):\n", len(onlineNodes))
	for _, n := range onlineNodes {
		fmt.Printf("  - %s (%s)\n", n.Name, n.ID)
	}

	// 只显示健康的工作节点
	filter = node.NewNodeFilter().
		WithRoles(node.RoleWorker).
		WithHealthyOnly()
	healthyWorkers := filter.MatchAny(nodes)
	fmt.Printf("\n健康的工作节点 (%d 个):\n", len(healthyWorkers))
	for _, n := range healthyWorkers {
		fmt.Printf("  - %s (负载: %.1f%%)\n", n.Name, n.GetLoadPercent())
	}

	fmt.Println("\n✅ 模拟完成")
}
