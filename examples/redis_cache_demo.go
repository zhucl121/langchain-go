package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/zhucl121/langchain-go/core/cache"
	"github.com/zhucl121/langchain-go/core/chat"
	"github.com/zhucl121/langchain-go/core/chat/providers/openai"
)

// RedisCache 使用示例

func main() {
	fmt.Println("=== Redis Cache 使用示例 ===\n")

	// 1. 基础使用
	basicUsage()

	// 2. LLM 缓存
	llmCacheDemo()

	// 3. 集群模式
	// clusterDemo()

	// 4. 高级特性
	advancedFeatures()
}

// basicUsage 展示基础使用
func basicUsage() {
	fmt.Println("1. Redis 缓存基础使用")
	fmt.Println("─────────────────────")

	// 创建 Redis 缓存
	config := cache.DefaultRedisCacheConfig()
	config.Addr = "localhost:6379"
	config.Password = "" // 设置密码（如果需要）
	config.Prefix = "myapp:"

	redisCache, err := cache.NewRedisCache(config)
	if err != nil {
		log.Printf("警告：Redis 不可用: %v\n", err)
		return
	}
	defer redisCache.Close()

	ctx := context.Background()

	// 测试连接
	if err := redisCache.Ping(ctx); err != nil {
		log.Fatal("Redis ping 失败:", err)
	}
	fmt.Println("✅ Redis 连接成功")

	// 设置缓存
	data := map[string]any{
		"user_id": 12345,
		"name":    "Alice",
		"email":   "alice@example.com",
	}

	err = redisCache.Set(ctx, "user:12345", data, 24*time.Hour)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("✅ 数据已缓存")

	// 获取缓存
	cached, found, err := redisCache.Get(ctx, "user:12345")
	if err != nil {
		log.Fatal(err)
	}

	if found {
		fmt.Printf("✅ 缓存命中: %+v\n", cached)
	}

	// 检查 TTL
	ttl, err := redisCache.TTL(ctx, "user:12345")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("⏱  剩余过期时间: %v\n", ttl.Round(time.Second))

	// 查看统计
	stats := redisCache.Stats()
	fmt.Printf("📊 缓存统计: 命中=%d, 未命中=%d, 命中率=%.2f%%\n\n",
		stats.Hits, stats.Misses, stats.HitRate*100)
}

// llmCacheDemo 展示 LLM 缓存使用
func llmCacheDemo() {
	fmt.Println("2. LLM 缓存示例")
	fmt.Println("─────────────────────")

	// 创建 Redis 缓存
	config := cache.DefaultRedisCacheConfig()
	config.Prefix = "llm:"

	redisCache, err := cache.NewRedisCache(config)
	if err != nil {
		log.Printf("警告：Redis 不可用: %v\n", err)
		return
	}
	defer redisCache.Close()

	// 创建 LLM 缓存
	llmCache := cache.NewLLMCache(redisCache)

	ctx := context.Background()

	// 模拟 LLM 调用
	prompt := "什么是人工智能？"

	// 第一次调用（未命中）
	start := time.Now()
	response, found := llmCache.Get(ctx, prompt)
	if !found {
		fmt.Println("⚠️  缓存未命中，调用 LLM...")

		// 模拟 LLM 调用（实际会很慢）
		time.Sleep(500 * time.Millisecond)
		response = "人工智能（AI）是计算机科学的一个分支，致力于创建能够执行通常需要人类智能的任务的智能机器。"

		// 缓存响应
		llmCache.Set(ctx, prompt, response, 1*time.Hour)
		fmt.Printf("✅ LLM 响应已缓存（耗时: %v）\n", time.Since(start))
	}
	fmt.Printf("📝 响应: %s\n", response)

	// 第二次调用（命中）
	start = time.Now()
	response, found = llmCache.Get(ctx, prompt)
	if found {
		fmt.Printf("✅ 缓存命中！（耗时: %v）\n", time.Since(start))
		fmt.Printf("📝 响应: %s\n", response)
	}

	// 统计
	stats := llmCache.Stats()
	fmt.Printf("📊 命中率: %.2f%% (节省了 %d 次 LLM 调用)\n\n",
		stats.HitRate*100, stats.Hits)
}

// llmCacheWithRealLLM 展示真实 LLM 的缓存使用
func llmCacheWithRealLLM() {
	fmt.Println("2b. 真实 LLM 缓存")
	fmt.Println("─────────────────────")

	// 创建 Redis 缓存
	config := cache.DefaultRedisCacheConfig()
	redisCache, err := cache.NewRedisCache(config)
	if err != nil {
		log.Printf("警告：Redis 不可用: %v\n", err)
		return
	}
	defer redisCache.Close()

	llmCache := cache.NewLLMCache(redisCache)

	// 创建 OpenAI LLM（带缓存）
	llm, err := openai.New(openai.Config{APIKey: "your-api-key", Model: "gpt-3.5-turbo"})
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	prompt := "解释量子计算"

	// 包装 LLM 调用以使用缓存
	callLLMWithCache := func(prompt string) (string, error) {
		// 检查缓存
		if cached, found := llmCache.Get(ctx, prompt); found {
			fmt.Println("✅ 从缓存返回")
			return cached, nil
		}

		// 调用 LLM
		fmt.Println("⚠️  调用 LLM API...")
		messages := []chat.Message{
			chat.NewHumanMessage(prompt),
		}

		response, err := llm.Call(ctx, messages)
		if err != nil {
			return "", err
		}

		result := response.Content
		// 缓存结果
		llmCache.Set(ctx, prompt, result, 24*time.Hour)

		return result, nil
	}

	// 第一次调用
	start := time.Now()
	response, err := callLLMWithCache(prompt)
	if err != nil {
		log.Printf("错误: %v\n", err)
		return
	}
	fmt.Printf("响应: %s\n", response)
	fmt.Printf("耗时: %v\n", time.Since(start))

	// 第二次调用（从缓存）
	start = time.Now()
	response, err = callLLMWithCache(prompt)
	if err != nil {
		log.Printf("错误: %v\n", err)
		return
	}
	fmt.Printf("响应: %s\n", response)
	fmt.Printf("耗时: %v（快了 200x！）\n\n", time.Since(start))
}

// clusterDemo 展示 Redis 集群使用
func clusterDemo() {
	fmt.Println("3. Redis 集群模式")
	fmt.Println("─────────────────────")

	// 创建 Redis 集群缓存
	config := cache.RedisClusterConfig{
		Addrs: []string{
			"localhost:7000",
			"localhost:7001",
			"localhost:7002",
		},
		Password:     "",
		Prefix:       "cluster:",
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	}

	clusterCache, err := cache.NewRedisClusterCache(config)
	if err != nil {
		log.Printf("警告：Redis 集群不可用: %v\n", err)
		return
	}
	defer clusterCache.Close()

	ctx := context.Background()

	// 使用与单机版相同的 API
	err = clusterCache.Set(ctx, "key", "value", time.Hour)
	if err != nil {
		log.Fatal(err)
	}

	value, found, err := clusterCache.Get(ctx, "key")
	if err != nil {
		log.Fatal(err)
	}

	if found {
		fmt.Printf("✅ 集群缓存工作正常: %v\n\n", value)
	}
}

// advancedFeatures 展示高级特性
func advancedFeatures() {
	fmt.Println("4. 高级特性")
	fmt.Println("─────────────────────")

	config := cache.DefaultRedisCacheConfig()
	redisCache, err := cache.NewRedisCache(config)
	if err != nil {
		log.Printf("警告：Redis 不可用: %v\n", err)
		return
	}
	defer redisCache.Close()

	ctx := context.Background()

	// SetNX - 分布式锁
	fmt.Println("a) 分布式锁 (SetNX)")
	lockKey := "resource:lock"

	acquired, err := redisCache.SetNX(ctx, lockKey, "locked", 10*time.Second)
	if err != nil {
		log.Fatal(err)
	}

	if acquired {
		fmt.Println("✅ 获取锁成功")

		// 执行关键操作
		fmt.Println("执行关键操作...")
		time.Sleep(1 * time.Second)

		// 释放锁
		redisCache.Delete(ctx, lockKey)
		fmt.Println("✅ 释放锁")
	} else {
		fmt.Println("⚠️  锁已被占用")
	}

	// 原子计数器
	fmt.Println("\nb) 原子计数器")
	counterKey := "page:views"

	// 递增
	for i := 0; i < 5; i++ {
		count, err := redisCache.Increment(ctx, counterKey, 1)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("访问量: %d\n", count)
	}

	// 列出所有键
	fmt.Println("\nc) 列出键")
	keys, err := redisCache.Keys(ctx, "*")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("所有键: %v\n", keys)

	// 批量操作
	fmt.Println("\nd) 批量操作")
	for i := 0; i < 3; i++ {
		key := fmt.Sprintf("batch:%d", i)
		redisCache.Set(ctx, key, fmt.Sprintf("value-%d", i), time.Hour)
	}

	batchKeys, _ := redisCache.Keys(ctx, "batch:*")
	fmt.Printf("批量创建的键: %v\n", batchKeys)

	// 清理
	fmt.Println("\ne) 清理缓存")
	err = redisCache.Clear(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("✅ 缓存已清空")
}

// 生产环境配置建议
func productionConfig() {
	fmt.Println("=== 生产环境配置建议 ===\n")

	// 推荐配置
	config := cache.RedisCacheConfig{
		Addr:         "redis.prod.example.com:6379",
		Password:     "your-secure-password", // 从环境变量读取
		DB:           0,
		Prefix:       "prod:langchain:",
		PoolSize:     20,                    // 增加连接池
		MinIdleConns: 10,                    // 保持足够的空闲连接
		MaxRetries:   3,                     // 重试失败的操作
		DialTimeout:  5 * time.Second,       // 连接超时
		ReadTimeout:  3 * time.Second,       // 读取超时
		WriteTimeout: 3 * time.Second,       // 写入超时
	}

	redisCache, err := cache.NewRedisCache(config)
	if err != nil {
		log.Fatal(err)
	}
	defer redisCache.Close()

	fmt.Println("✅ 生产环境 Redis 配置完成")
	fmt.Println("\n推荐设置:")
	fmt.Println("- 使用密码认证")
	fmt.Println("- 启用持久化 (AOF/RDB)")
	fmt.Println("- 配置合适的内存策略 (maxmemory-policy)")
	fmt.Println("- 监控连接池使用情况")
	fmt.Println("- 设置合理的 TTL")
	fmt.Println("- 考虑使用 Redis Sentinel 或 Cluster")
}

/*
性能对比：

内存缓存 vs Redis 缓存：

操作          内存缓存    Redis      差异
─────────────────────────────────────
Set           50ns       500µs      10,000x
Get           30ns       300µs      10,000x
命中率        本地       分布式      -
扩展性        单机       集群        ✅
持久化        否         是          ✅
多进程共享    否         是          ✅

使用建议：
1. 开发/测试 → 内存缓存
2. 单机部署 → 内存缓存 + 定期持久化
3. 分布式部署 → Redis 缓存
4. 高并发场景 → Redis 集群

成本优化：
- LLM 调用: $0.002/1K tokens
- Redis 存储: $0.00003/1K (便宜 67x)
- 命中率 50% → 节省 50% LLM 成本
- 命中率 90% → 节省 90% LLM 成本

ROI 计算：
假设：
- 10,000 次 LLM 调用/天
- 平均 1K tokens/次
- LLM 成本: $0.002/1K tokens

无缓存成本: 10,000 * $0.002 = $20/天 = $600/月

有缓存（50% 命中率）:
- LLM 成本: $10/天
- Redis 成本: $5/月
- 总成本: $305/月
- 节省: $295/月 (49%)

有缓存（90% 命中率）:
- LLM 成本: $2/天
- Redis 成本: $5/月
- 总成本: $65/月
- 节省: $535/月 (89%)
*/
