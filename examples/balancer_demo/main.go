package main

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/zhucl121/langchain-go/pkg/cluster/balancer"
	"github.com/zhucl121/langchain-go/pkg/cluster/node"
)

func main() {
	fmt.Println("🚀 LangChain-Go 负载均衡示例")
	fmt.Println("========================================")

	// 创建测试节点
	nodes := createTestNodes()
	fmt.Printf("\n📋 集群节点列表 (%d 个):\n", len(nodes))
	for i, n := range nodes {
		fmt.Printf("  %d. %s - %s:%d (负载: %.1f%%)\n",
			i+1, n.Name, n.Address, n.Port, n.GetLoadPercent())
	}

	ctx := context.Background()

	// 演示各种负载均衡策略
	fmt.Println("\n" + strings.Repeat("=", 50))
	demoRoundRobin(ctx, nodes)

	fmt.Println("\n" + strings.Repeat("=", 50))
	demoLeastConnection(ctx, nodes)

	fmt.Println("\n" + strings.Repeat("=", 50))
	demoWeighted(ctx, nodes)

	fmt.Println("\n" + strings.Repeat("=", 50))
	demoConsistentHash(ctx, nodes)

	fmt.Println("\n" + strings.Repeat("=", 50))
	demoAdaptive(ctx, nodes)

	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("✅ 所有演示完成！")
}

// createTestNodes 创建测试节点
func createTestNodes() []*node.Node {
	return []*node.Node{
		{
			ID:      "node-1",
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
				CurrentConnections: 200,
				CPUUsagePercent:    30,
				MemoryUsageMB:      1024,
			},
		},
		{
			ID:      "node-2",
			Name:    "worker-2",
			Address: "192.168.1.11",
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
				CPUUsagePercent:    60,
				MemoryUsageMB:      2560,
			},
		},
		{
			ID:      "node-3",
			Name:    "worker-3",
			Address: "192.168.1.12",
			Port:    8080,
			Status:  node.StatusOnline,
			Roles:   []node.NodeRole{node.RoleWorker},
			Capacity: node.Capacity{
				MaxConnections: 2000,
				MaxQPS:         1000,
				MaxMemoryMB:    8192,
			},
			Load: node.Load{
				CurrentConnections: 300,
				CPUUsagePercent:    25,
				MemoryUsageMB:      2048,
			},
		},
	}
}

// demoRoundRobin 演示轮询负载均衡
func demoRoundRobin(ctx context.Context, nodes []*node.Node) {
	fmt.Println("🔄 轮询负载均衡 (Round Robin)")
	fmt.Println("特点: 按顺序依次选择节点，确保请求均匀分布")

	lb := balancer.NewRoundRobinBalancer(nodes)

	// 发送 12 个请求
	selectedCounts := make(map[string]int)
	for i := 0; i < 12; i++ {
		req := &balancer.Request{
			ID:   fmt.Sprintf("req-%d", i+1),
			Type: balancer.RequestTypeLLM,
		}

		selected, _ := lb.SelectNode(ctx, req)
		selectedCounts[selected.Name]++
		fmt.Printf("  请求 #%2d → %s\n", i+1, selected.Name)

		// 模拟请求完成
		lb.RecordResult(selected.ID, true, randomLatency())
	}

	printDistribution(selectedCounts)
}

// demoLeastConnection 演示最少连接负载均衡
func demoLeastConnection(ctx context.Context, nodes []*node.Node) {
	fmt.Println("📊 最少连接负载均衡 (Least Connection)")
	fmt.Println("特点: 选择当前连接数最少的节点，适合长连接场景")

	lb := balancer.NewLeastConnectionBalancer(nodes)

	// 模拟并发请求
	fmt.Println("\n  模拟 10 个并发长连接...")
	for i := 0; i < 10; i++ {
		req := &balancer.Request{
			ID:   fmt.Sprintf("req-%d", i+1),
			Type: balancer.RequestTypeLLM,
		}

		selected, _ := lb.SelectNode(ctx, req)
		fmt.Printf("  连接 #%2d → %s (当前连接: %d)\n",
			i+1, selected.Name, lb.GetConnectionCount(selected.ID))

		// 模拟一些连接完成
		if i%3 == 0 {
			lb.RecordResult(selected.ID, true, 100*time.Millisecond)
		}
	}
}

// demoWeighted 演示加权负载均衡
func demoWeighted(ctx context.Context, nodes []*node.Node) {
	fmt.Println("⚖️  加权负载均衡 (Weighted)")
	fmt.Println("特点: 根据节点权重分配请求，权重越高分配越多")

	// 设置权重: node-1:1, node-2:2, node-3:3
	weights := []int{1, 2, 3}
	lb := balancer.NewWeightedBalancer(nodes, weights)

	fmt.Println("\n  权重配置:")
	for i, n := range nodes {
		fmt.Printf("    %s: 权重 %d\n", n.Name, weights[i])
	}

	// 发送 60 个请求，统计分布
	selectedCounts := make(map[string]int)
	for i := 0; i < 60; i++ {
		req := &balancer.Request{
			ID:   fmt.Sprintf("req-%d", i+1),
			Type: balancer.RequestTypeLLM,
		}

		selected, _ := lb.SelectNode(ctx, req)
		selectedCounts[selected.Name]++
		lb.RecordResult(selected.ID, true, randomLatency())
	}

	printDistribution(selectedCounts)
}

// demoConsistentHash 演示一致性哈希负载均衡
func demoConsistentHash(ctx context.Context, nodes []*node.Node) {
	fmt.Println("🔗 一致性哈希负载均衡 (Consistent Hash)")
	fmt.Println("特点: 相同的请求总是路由到相同的节点，适合缓存场景")

	lb := balancer.NewConsistentHashBalancer(nodes, 150)

	// 测试相同用户的多次请求
	users := []string{"alice", "bob", "charlie", "david", "eve"}
	userNodes := make(map[string]string)

	fmt.Println("\n  用户路由测试:")
	for _, user := range users {
		req := &balancer.Request{
			ID:     "req-1",
			UserID: user,
			Type:   balancer.RequestTypeLLM,
		}

		selected, _ := lb.SelectNode(ctx, req)
		userNodes[user] = selected.Name
		fmt.Printf("    用户 %8s → %s\n", user, selected.Name)

		// 验证一致性：相同用户再次请求
		req2 := &balancer.Request{
			ID:     "req-2",
			UserID: user,
			Type:   balancer.RequestTypeLLM,
		}
		selected2, _ := lb.SelectNode(ctx, req2)
		if selected.ID != selected2.ID {
			fmt.Printf("      ⚠️  警告: 一致性失败！\n")
		} else {
			fmt.Printf("      ✅ 一致性验证通过\n")
		}
	}
}

// demoAdaptive 演示自适应负载均衡
func demoAdaptive(ctx context.Context, nodes []*node.Node) {
	fmt.Println("🧠 自适应负载均衡 (Adaptive)")
	fmt.Println("特点: 根据节点实时性能动态调整，优先选择表现好的节点")

	lb := balancer.NewAdaptiveBalancer(nodes, 10)

	// 模拟节点性能差异
	fmt.Println("\n  模拟节点性能:")
	fmt.Println("    node-1: 快速响应 (50ms), 100% 成功率")
	fmt.Println("    node-2: 中等响应 (150ms), 50% 成功率")
	fmt.Println("    node-3: 慢响应 (300ms), 100% 成功率")

	// 先记录一些历史数据
	for i := 0; i < 5; i++ {
		lb.RecordResult("node-1", true, 50*time.Millisecond)
		lb.RecordResult("node-2", i < 2, 150*time.Millisecond)
		lb.RecordResult("node-3", true, 300*time.Millisecond)
	}

	// 显示初始得分
	fmt.Println("\n  节点得分 (0-1, 越高越好):")
	for _, n := range nodes {
		score := lb.GetScore(n.ID)
		fmt.Printf("    %s: %.3f\n", n.Name, score)
	}

	// 发送 20 个请求
	fmt.Println("\n  请求分配:")
	selectedCounts := make(map[string]int)
	for i := 0; i < 20; i++ {
		req := &balancer.Request{
			ID:   fmt.Sprintf("req-%d", i+1),
			Type: balancer.RequestTypeLLM,
		}

		selected, _ := lb.SelectNode(ctx, req)
		selectedCounts[selected.Name]++

		// 根据节点特性模拟响应
		var success bool
		var latency time.Duration
		switch selected.ID {
		case "node-1":
			success = true
			latency = 50 * time.Millisecond
		case "node-2":
			success = rand.Float64() < 0.5
			latency = 150 * time.Millisecond
		case "node-3":
			success = true
			latency = 300 * time.Millisecond
		}

		lb.RecordResult(selected.ID, success, latency)
	}

	printDistribution(selectedCounts)

	// 显示最终得分
	fmt.Println("\n  最终节点得分:")
	for _, n := range nodes {
		score := lb.GetScore(n.ID)
		stats := lb.GetStats().NodeStats[n.ID]
		fmt.Printf("    %s: %.3f (请求: %d, 成功: %d)\n",
			n.Name, score, stats.Requests, stats.SuccessRequests)
	}
}

// printDistribution 打印分布统计
func printDistribution(counts map[string]int) {
	total := 0
	for _, count := range counts {
		total += count
	}

	fmt.Println("\n  分布统计:")
	for name, count := range counts {
		percentage := float64(count) / float64(total) * 100
		fmt.Printf("    %s: %3d 次 (%.1f%%)\n", name, count, percentage)
	}
}

// randomLatency 生成随机延迟
func randomLatency() time.Duration {
	return time.Duration(50+rand.Intn(150)) * time.Millisecond
}
