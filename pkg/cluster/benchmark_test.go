package cluster_test

import (
	"context"
	"testing"
	"time"

	"github.com/zhucl121/langchain-go/pkg/cluster/balancer"
	"github.com/zhucl121/langchain-go/pkg/cluster/cache"
	"github.com/zhucl121/langchain-go/pkg/cluster/failover"
	"github.com/zhucl121/langchain-go/pkg/cluster/node"
)

// BenchmarkLoadBalancer 负载均衡器性能基准测试
func BenchmarkRoundRobinBalancer_SelectNode(b *testing.B) {
	nodes := createBenchmarkNodes(10)
	lb := balancer.NewRoundRobinBalancer(nodes)
	ctx := context.Background()
	req := &balancer.Request{ID: "test", Type: balancer.RequestTypeLLM}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = lb.SelectNode(ctx, req)
	}
}

func BenchmarkConsistentHashBalancer_SelectNode(b *testing.B) {
	nodes := createBenchmarkNodes(10)
	lb := balancer.NewConsistentHashBalancer(nodes, 150)
	ctx := context.Background()
	req := &balancer.Request{ID: "test", Type: balancer.RequestTypeLLM}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = lb.SelectNode(ctx, req)
	}
}

func BenchmarkAdaptiveBalancer_SelectNode(b *testing.B) {
	nodes := createBenchmarkNodes(10)
	lb := balancer.NewAdaptiveBalancer(nodes, 100)
	ctx := context.Background()
	req := &balancer.Request{ID: "test", Type: balancer.RequestTypeLLM}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = lb.SelectNode(ctx, req)
	}
}

// BenchmarkCache 缓存性能基准测试
func BenchmarkMemoryCache_Set(b *testing.B) {
	cache := cache.NewMemoryCache(10000)
	defer cache.Close()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := string(rune(i % 1000))
		cache.Set(ctx, key, []byte("value"), 1*time.Minute)
	}
}

func BenchmarkMemoryCache_Get(b *testing.B) {
	mc := cache.NewMemoryCache(10000)
	defer mc.Close()
	ctx := context.Background()

	// 预填充数据
	for i := 0; i < 1000; i++ {
		key := string(rune(i))
		mc.Set(ctx, key, []byte("value"), 1*time.Minute)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := string(rune(i % 1000))
		_, _ = mc.Get(ctx, key)
	}
}

func BenchmarkMemoryCache_Concurrent(b *testing.B) {
	mc := cache.NewMemoryCache(10000)
	defer mc.Close()
	ctx := context.Background()

	// 预填充数据
	for i := 0; i < 1000; i++ {
		key := string(rune(i))
		mc.Set(ctx, key, []byte("value"), 1*time.Minute)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := string(rune(i % 1000))
			if i%2 == 0 {
				mc.Get(ctx, key)
			} else {
				mc.Set(ctx, key, []byte("value"), 1*time.Minute)
			}
			i++
		}
	})
}

func BenchmarkLayeredCache_Get_LocalHit(b *testing.B) {
	local := cache.NewMemoryCache(1000)
	remote := cache.NewMemoryCache(10000)
	layered := cache.NewLayeredCache(local, remote)
	defer layered.Close()

	ctx := context.Background()

	// 预填充本地缓存
	for i := 0; i < 100; i++ {
		key := string(rune(i))
		local.Set(ctx, key, []byte("value"), 1*time.Minute)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := string(rune(i % 100))
		_, _ = layered.Get(ctx, key)
	}
}

// BenchmarkCircuitBreaker 熔断器性能基准测试
func BenchmarkCircuitBreaker_Execute_Closed(b *testing.B) {
	config := failover.DefaultCircuitBreakerConfig()
	cb := failover.NewCircuitBreaker(config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cb.Execute(func() error {
			return nil
		})
	}
}

func BenchmarkCircuitBreaker_GetState(b *testing.B) {
	config := failover.DefaultCircuitBreakerConfig()
	cb := failover.NewCircuitBreaker(config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cb.GetState()
	}
}

// BenchmarkFailoverManager 故障转移管理器性能基准测试
func BenchmarkFailoverManager_HandleFailure(b *testing.B) {
	checker := failover.HealthCheckerFunc(func(ctx context.Context, nodeID string) error {
		return nil
	})
	config := failover.DefaultConfig()
	config.EnableAlerts = false
	manager := failover.NewFailoverManager(config, checker)
	defer manager.Close()

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		nodeID := string(rune(i % 10))
		_ = manager.HandleFailure(ctx, nodeID)
	}
}

// 辅助函数
func createBenchmarkNodes(count int) []*node.Node {
	nodes := make([]*node.Node, count)
	for i := 0; i < count; i++ {
		nodes[i] = &node.Node{
			ID:      string(rune('a' + i)),
			Name:    string(rune('a' + i)),
			Address: "192.168.1." + string(rune('0'+i)),
			Port:    8080,
			Status:  node.StatusOnline,
			Roles:   []node.NodeRole{node.RoleWorker},
			Capacity: node.Capacity{
				MaxConnections: 1000,
				MaxQPS:         500,
				MaxMemoryMB:    4096,
			},
			Load: node.Load{
				CurrentConnections: 100,
				CPUUsagePercent:    30,
				MemoryUsageMB:      1024,
			},
		}
	}
	return nodes
}
