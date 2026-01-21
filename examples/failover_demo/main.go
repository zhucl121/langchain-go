package main

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/zhucl121/langchain-go/pkg/cluster/failover"
)

func main() {
	fmt.Println("🚀 LangChain-Go 故障转移与高可用示例")
	fmt.Println("========================================")

	// 演示各种故障转移功能
	fmt.Println("\n" + strings.Repeat("=", 50))
	demoCircuitBreaker()

	fmt.Println("\n" + strings.Repeat("=", 50))
	demoCircuitBreakerWithRecovery()

	fmt.Println("\n" + strings.Repeat("=", 50))
	demoFailoverManager()

	fmt.Println("\n" + strings.Repeat("=", 50))
	demoFailoverManagerWithEvents()

	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("✅ 所有演示完成！")
}

// demoCircuitBreaker 演示熔断器基础功能
func demoCircuitBreaker() {
	fmt.Println("⚡ 熔断器 (Circuit Breaker)")
	fmt.Println("特点: 自动熔断，保护服务免受故障影响")

	// 创建熔断器
	config := failover.DefaultCircuitBreakerConfig()
	config.FailureThreshold = 3
	config.Timeout = 2 * time.Second

	cb := failover.NewCircuitBreaker(config)

	fmt.Println("\n  1. 正常请求...")
	for i := 0; i < 3; i++ {
		err := cb.Execute(func() error {
			// 模拟成功的服务调用
			return nil
		})

		if err != nil {
			fmt.Printf("    ❌ 请求 #%d 失败: %v\n", i+1, err)
		} else {
			fmt.Printf("    ✅ 请求 #%d 成功\n", i+1)
		}
	}

	fmt.Printf("\n    当前状态: %s\n", cb.GetState())

	fmt.Println("\n  2. 触发失败...")
	for i := 0; i < 3; i++ {
		err := cb.Execute(func() error {
			// 模拟失败的服务调用
			return errors.New("service unavailable")
		})

		if err != nil {
			fmt.Printf("    ❌ 请求 #%d 失败: %v\n", i+1, err)
		}
	}

	fmt.Printf("\n    当前状态: %s (熔断器已打开)\n", cb.GetState())

	fmt.Println("\n  3. 尝试新请求（被拒绝）...")
	err := cb.Execute(func() error {
		return nil
	})

	if err == failover.ErrCircuitOpen {
		fmt.Println("    ⚠️  请求被拒绝 - 熔断器处于打开状态")
	}

	// 显示统计
	stats := cb.GetStats()
	fmt.Println("\n  4. 统计信息:")
	fmt.Printf("    总请求: %d 次\n", stats.TotalRequests)
	fmt.Printf("    成功请求: %d 次\n", stats.SuccessRequests)
	fmt.Printf("    失败请求: %d 次\n", stats.FailedRequests)
	fmt.Printf("    被拒绝: %d 次\n", stats.RejectedRequests)
}

// demoCircuitBreakerWithRecovery 演示熔断器恢复
func demoCircuitBreakerWithRecovery() {
	fmt.Println("🔄 熔断器恢复 (Circuit Breaker Recovery)")
	fmt.Println("特点: 自动探测恢复，逐步放行流量")

	config := failover.DefaultCircuitBreakerConfig()
	config.FailureThreshold = 2
	config.SuccessThreshold = 2
	config.Timeout = 1 * time.Second

	cb := failover.NewCircuitBreaker(config)

	fmt.Println("\n  1. 触发熔断...")
	for i := 0; i < 2; i++ {
		cb.Execute(func() error {
			return errors.New("service error")
		})
		fmt.Printf("    ❌ 失败 #%d\n", i+1)
	}

	fmt.Printf("\n    当前状态: %s\n", cb.GetState())

	fmt.Println("\n  2. 等待超时（1秒）...")
	time.Sleep(1100 * time.Millisecond)

	fmt.Println("\n  3. 尝试恢复（半开状态）...")
	err := cb.Execute(func() error {
		// 模拟服务恢复
		return nil
	})

	if err != nil {
		fmt.Printf("    ❌ 请求失败: %v\n", err)
	} else {
		fmt.Println("    ✅ 请求成功")
	}

	fmt.Printf("\n    当前状态: %s (半开状态)\n", cb.GetState())

	fmt.Println("\n  4. 继续成功请求...")
	err = cb.Execute(func() error {
		return nil
	})

	if err != nil {
		fmt.Printf("    ❌ 请求失败: %v\n", err)
	} else {
		fmt.Println("    ✅ 请求成功")
	}

	fmt.Printf("\n    当前状态: %s (已恢复)\n", cb.GetState())

	stats := cb.GetStats()
	fmt.Println("\n  5. 统计信息:")
	fmt.Printf("    总请求: %d 次\n", stats.TotalRequests)
	fmt.Printf("    成功请求: %d 次\n", stats.SuccessRequests)
	fmt.Printf("    失败请求: %d 次\n", stats.FailedRequests)
}

// demoFailoverManager 演示故障转移管理器
func demoFailoverManager() {
	fmt.Println("🔧 故障转移管理器 (Failover Manager)")
	fmt.Println("特点: 自动故障检测和转移")

	// 创建健康检查器
	nodeHealth := make(map[string]bool)
	nodeHealth["node-1"] = true
	nodeHealth["node-2"] = true
	nodeHealth["node-3"] = true

	checker := failover.HealthCheckerFunc(func(ctx context.Context, nodeID string) error {
		if !nodeHealth[nodeID] {
			return errors.New("node unhealthy")
		}
		return nil
	})

	// 创建管理器
	config := failover.DefaultConfig()
	config.HealthCheckInterval = 1 * time.Second
	config.FailureThreshold = 2
	config.RecoveryThreshold = 2
	config.EnableAlerts = false

	manager := failover.NewFailoverManager(config, checker)
	defer manager.Close()

	ctx := context.Background()

	fmt.Println("\n  1. 初始状态 - 所有节点健康")
	fmt.Println("    ✅ node-1: 健康")
	fmt.Println("    ✅ node-2: 健康")
	fmt.Println("    ✅ node-3: 健康")

	fmt.Println("\n  2. 模拟 node-2 故障...")
	nodeHealth["node-2"] = false

	// 检查节点健康
	for i := 0; i < 2; i++ {
		err := manager.CheckNodeHealth(ctx, "node-2")
		if err != nil {
			fmt.Printf("    ❌ node-2 健康检查失败 (%d/%d)\n", i+1, config.FailureThreshold)
		}
		time.Sleep(100 * time.Millisecond)
	}

	// 等待故障转移完成
	time.Sleep(100 * time.Millisecond)

	stats := manager.GetStats()
	fmt.Printf("\n  3. 故障转移完成:")
	fmt.Printf("\n    总故障次数: %d\n", stats.TotalFailures)

	if nodeStats, ok := stats.NodeStats["node-2"]; ok {
		fmt.Printf("    node-2 状态: %s\n", nodeStats.CurrentState)
		fmt.Printf("    node-2 故障次数: %d\n", nodeStats.Failures)
	}

	fmt.Println("\n  4. 模拟 node-2 恢复...")
	nodeHealth["node-2"] = true

	// 检查恢复
	for i := 0; i < 2; i++ {
		err := manager.CheckNodeHealth(ctx, "node-2")
		if err == nil {
			fmt.Printf("    ✅ node-2 健康检查成功 (%d/%d)\n", i+1, config.RecoveryThreshold)
		}
		time.Sleep(100 * time.Millisecond)
	}

	// 等待恢复完成
	time.Sleep(100 * time.Millisecond)

	stats = manager.GetStats()
	fmt.Printf("\n  5. 恢复完成:")
	fmt.Printf("\n    总恢复次数: %d\n", stats.TotalRecoveries)

	if nodeStats, ok := stats.NodeStats["node-2"]; ok {
		fmt.Printf("    node-2 状态: %s\n", nodeStats.CurrentState)
		fmt.Printf("    node-2 恢复次数: %d\n", nodeStats.Recoveries)
	}
}

// demoFailoverManagerWithEvents 演示带事件监听的故障转移
func demoFailoverManagerWithEvents() {
	fmt.Println("📡 故障转移事件监听")
	fmt.Println("特点: 实时监听故障和恢复事件")

	checker := failover.HealthCheckerFunc(func(ctx context.Context, nodeID string) error {
		return nil
	})

	config := failover.DefaultConfig()
	config.EnableAlerts = false

	manager := failover.NewFailoverManager(config, checker)
	defer manager.Close()

	// 添加事件监听器
	eventLog := []string{}
	listener := &failover.EventListenerFunc{
		OnFailureFunc: func(event failover.FailureEvent) {
			msg := fmt.Sprintf("[%s] 故障事件: %s - %s",
				event.Timestamp.Format("15:04:05"),
				event.NodeID,
				event.Type)
			eventLog = append(eventLog, msg)
		},
		OnRecoveryFunc: func(event failover.FailureEvent) {
			msg := fmt.Sprintf("[%s] 恢复事件: %s - %s",
				event.Timestamp.Format("15:04:05"),
				event.NodeID,
				event.Type)
			eventLog = append(eventLog, msg)
		},
	}

	manager.AddListener(listener)

	ctx := context.Background()

	fmt.Println("\n  1. 触发故障...")
	manager.HandleFailure(ctx, "node-1")

	fmt.Println("\n  2. 触发恢复...")
	time.Sleep(50 * time.Millisecond)
	manager.RecoverNode(ctx, "node-1")

	time.Sleep(50 * time.Millisecond)

	fmt.Println("\n  3. 事件日志:")
	for _, log := range eventLog {
		fmt.Printf("    %s\n", log)
	}

	stats := manager.GetStats()
	fmt.Println("\n  4. 最终统计:")
	fmt.Printf("    总故障: %d 次\n", stats.TotalFailures)
	fmt.Printf("    总恢复: %d 次\n", stats.TotalRecoveries)
}

// 模拟不稳定的服务
func unstableService() error {
	// 30% 概率失败
	if rand.Float64() < 0.3 {
		return errors.New("service temporarily unavailable")
	}
	return nil
}
