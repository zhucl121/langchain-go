package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/zhucl121/langchain-go/pkg/cluster/cache"
)

func main() {
	fmt.Println("🚀 LangChain-Go 分布式缓存示例")
	fmt.Println("========================================")

	ctx := context.Background()

	// 演示各种缓存功能
	fmt.Println("\n" + strings.Repeat("=", 50))
	demoMemoryCache(ctx)

	fmt.Println("\n" + strings.Repeat("=", 50))
	demoMemoryCacheEviction(ctx)

	fmt.Println("\n" + strings.Repeat("=", 50))
	demoLayeredCache(ctx)

	fmt.Println("\n" + strings.Repeat("=", 50))
	demoCachePerformance(ctx)

	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("✅ 所有演示完成！")
}

// demoMemoryCache 演示内存缓存基础功能
func demoMemoryCache(ctx context.Context) {
	fmt.Println("💾 内存缓存 (Memory Cache)")
	fmt.Println("特点: 高速本地缓存，支持 TTL 和 LRU 驱逐")

	// 创建内存缓存
	mc := cache.NewMemoryCache(100)
	defer mc.Close()

	// 设置缓存
	fmt.Println("\n  1. 设置缓存...")
	items := map[string]string{
		"user:1001": "Alice",
		"user:1002": "Bob",
		"user:1003": "Charlie",
	}

	for key, value := range items {
		err := mc.Set(ctx, key, []byte(value), 5*time.Minute)
		if err != nil {
			fmt.Printf("    ❌ 设置 %s 失败: %v\n", key, err)
		} else {
			fmt.Printf("    ✅ 设置 %s = %s\n", key, value)
		}
	}

	// 获取缓存
	fmt.Println("\n  2. 获取缓存...")
	for key := range items {
		value, err := mc.Get(ctx, key)
		if err != nil {
			fmt.Printf("    ❌ 获取 %s 失败: %v\n", key, err)
		} else {
			fmt.Printf("    ✅ 获取 %s = %s\n", key, string(value))
		}
	}

	// 检查存在性
	fmt.Println("\n  3. 检查键是否存在...")
	exists, _ := mc.Exists(ctx, "user:1001")
	fmt.Printf("    user:1001 存在: %v\n", exists)

	exists, _ = mc.Exists(ctx, "user:9999")
	fmt.Printf("    user:9999 存在: %v\n", exists)

	// 删除缓存
	fmt.Println("\n  4. 删除缓存...")
	mc.Delete(ctx, "user:1002")
	fmt.Println("    ✅ 已删除 user:1002")

	exists, _ = mc.Exists(ctx, "user:1002")
	fmt.Printf("    user:1002 存在: %v\n", exists)

	// 显示统计
	fmt.Println("\n  5. 统计信息:")
	stats := mc.Stats()
	fmt.Printf("    总大小: %d 项\n", stats.Size)
	fmt.Printf("    命中: %d 次\n", stats.Hits)
	fmt.Printf("    未命中: %d 次\n", stats.Misses)
	fmt.Printf("    命中率: %.2f%%\n", stats.HitRate()*100)
	fmt.Printf("    设置: %d 次\n", stats.Sets)
	fmt.Printf("    删除: %d 次\n", stats.Deletes)
}

// demoMemoryCacheEviction 演示缓存驱逐策略
func demoMemoryCacheEviction(ctx context.Context) {
	fmt.Println("🔄 缓存驱逐策略 (Eviction Policy)")
	fmt.Println("特点: LRU 自动驱逐最久未使用的条目")

	// 创建小容量缓存（最多 3 项）
	config := cache.MemoryCacheConfig{
		MaxSize:         3,
		EvictionPolicy:  cache.EvictionPolicyLRU,
		CleanupInterval: 0,
	}
	mc := cache.NewMemoryCacheWithConfig(config)

	fmt.Println("\n  1. 填满缓存（最大 3 项）...")
	for i := 1; i <= 3; i++ {
		key := fmt.Sprintf("key%d", i)
		value := fmt.Sprintf("value%d", i)
		mc.Set(ctx, key, []byte(value), 1*time.Minute)
		fmt.Printf("    ✅ 设置 %s = %s\n", key, value)
	}

	fmt.Printf("\n    当前大小: %d / %d\n", mc.Stats().Size, config.MaxSize)

	// 访问 key1，使其成为最近使用
	fmt.Println("\n  2. 访问 key1（更新访问时间）...")
	mc.Get(ctx, "key1")
	time.Sleep(10 * time.Millisecond)

	// 访问 key3
	fmt.Println("  3. 访问 key3（更新访问时间）...")
	mc.Get(ctx, "key3")
	time.Sleep(10 * time.Millisecond)

	// 添加新键，应该驱逐 key2
	fmt.Println("\n  4. 添加新键 key4（触发驱逐）...")
	mc.Set(ctx, "key4", []byte("value4"), 1*time.Minute)
	fmt.Println("    ✅ 设置 key4 = value4")

	// 检查哪个键被驱逐
	fmt.Println("\n  5. 检查驱逐结果:")
	for i := 1; i <= 4; i++ {
		key := fmt.Sprintf("key%d", i)
		exists, _ := mc.Exists(ctx, key)
		if exists {
			fmt.Printf("    ✅ %s 仍在缓存中\n", key)
		} else {
			fmt.Printf("    ❌ %s 已被驱逐（LRU）\n", key)
		}
	}

	stats := mc.Stats()
	fmt.Printf("\n  6. 驱逐统计:")
	fmt.Printf("\n    驱逐次数: %d\n", stats.Evictions)
	fmt.Printf("    当前大小: %d / %d\n", stats.Size, config.MaxSize)
}

// demoLayeredCache 演示分层缓存
func demoLayeredCache(ctx context.Context) {
	fmt.Println("🔗 分层缓存 (Layered Cache)")
	fmt.Println("特点: 本地 + 远程两层缓存，自动回写")

	// 创建本地和远程缓存
	local := cache.NewMemoryCache(100)
	remote := cache.NewMemoryCache(1000) // 模拟远程缓存

	// 创建分层缓存
	layered := cache.NewLayeredCache(local, remote)
	defer layered.Close()

	fmt.Println("\n  1. 写入数据（写穿模式）...")
	data := map[string]string{
		"product:1001": "iPhone 15 Pro",
		"product:1002": "MacBook Pro",
		"product:1003": "AirPods Pro",
	}

	for key, value := range data {
		err := layered.Set(ctx, key, []byte(value), 10*time.Minute)
		if err != nil {
			fmt.Printf("    ❌ 设置 %s 失败\n", key)
		} else {
			fmt.Printf("    ✅ 设置 %s = %s (本地+远程)\n", key, value)
		}
	}

	// 清空本地缓存，模拟本地缓存失效
	fmt.Println("\n  2. 清空本地缓存（模拟失效）...")
	local.Clear(ctx)
	fmt.Println("    ✅ 本地缓存已清空")

	// 从分层缓存读取，会自动从远程回写到本地
	fmt.Println("\n  3. 读取数据（自动回写）...")
	for key := range data {
		value, err := layered.Get(ctx, key)
		if err != nil {
			fmt.Printf("    ❌ 获取 %s 失败: %v\n", key, err)
		} else {
			fmt.Printf("    ✅ 获取 %s = %s (从远程回写到本地)\n", key, string(value))
		}
	}

	// 验证本地缓存已回写
	fmt.Println("\n  4. 验证本地缓存回写...")
	for key := range data {
		exists, _ := local.Exists(ctx, key)
		if exists {
			fmt.Printf("    ✅ %s 已回写到本地缓存\n", key)
		} else {
			fmt.Printf("    ❌ %s 未在本地缓存中\n", key)
		}
	}

	// 批量操作
	fmt.Println("\n  5. 批量操作...")
	keys := []string{"product:1001", "product:1002", "product:1003"}
	results, _ := layered.MGet(ctx, keys)
	fmt.Printf("    ✅ 批量获取 %d 个键\n", len(results))
	for key, value := range results {
		fmt.Printf("       %s = %s\n", key, string(value))
	}

	// 统计信息
	fmt.Println("\n  6. 统计信息:")
	stats := layered.Stats()
	fmt.Printf("    命中: %d 次\n", stats.Hits)
	fmt.Printf("    未命中: %d 次\n", stats.Misses)
	fmt.Printf("    命中率: %.2f%%\n", stats.HitRate()*100)
}

// demoCachePerformance 演示缓存性能
func demoCachePerformance(ctx context.Context) {
	fmt.Println("⚡ 缓存性能测试")
	fmt.Println("特点: 高并发读写性能")

	mc := cache.NewMemoryCache(10000)
	defer mc.Close()

	// 写入测试
	fmt.Println("\n  1. 写入性能测试（1000 个键）...")
	start := time.Now()
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("perf:key:%d", i)
		value := fmt.Sprintf("value-%d", i)
		mc.Set(ctx, key, []byte(value), 10*time.Minute)
	}
	writeTime := time.Since(start)
	fmt.Printf("    ✅ 完成 1000 次写入，耗时: %v\n", writeTime)
	fmt.Printf("    写入速度: %.0f ops/s\n", 1000.0/writeTime.Seconds())

	// 读取测试
	fmt.Println("\n  2. 读取性能测试（10000 次）...")
	start = time.Now()
	hits := 0
	for i := 0; i < 10000; i++ {
		key := fmt.Sprintf("perf:key:%d", i%1000)
		if _, err := mc.Get(ctx, key); err == nil {
			hits++
		}
	}
	readTime := time.Since(start)
	fmt.Printf("    ✅ 完成 10000 次读取，耗时: %v\n", readTime)
	fmt.Printf("    读取速度: %.0f ops/s\n", 10000.0/readTime.Seconds())
	fmt.Printf("    命中率: %.2f%%\n", float64(hits)/100.0)

	// 混合操作测试
	fmt.Println("\n  3. 混合操作测试（80% 读 + 20% 写）...")
	start = time.Now()
	for i := 0; i < 5000; i++ {
		key := fmt.Sprintf("perf:key:%d", i%1000)
		if i%5 == 0 {
			// 20% 写
			mc.Set(ctx, key, []byte("updated"), 10*time.Minute)
		} else {
			// 80% 读
			mc.Get(ctx, key)
		}
	}
	mixedTime := time.Since(start)
	fmt.Printf("    ✅ 完成 5000 次混合操作，耗时: %v\n", mixedTime)
	fmt.Printf("    操作速度: %.0f ops/s\n", 5000.0/mixedTime.Seconds())

	// 最终统计
	fmt.Println("\n  4. 最终统计:")
	stats := mc.Stats()
	fmt.Printf("    总大小: %d 项\n", stats.Size)
	fmt.Printf("    总命中: %d 次\n", stats.Hits)
	fmt.Printf("    总未命中: %d 次\n", stats.Misses)
	fmt.Printf("    总设置: %d 次\n", stats.Sets)
	fmt.Printf("    命中率: %.2f%%\n", stats.HitRate()*100)
}
