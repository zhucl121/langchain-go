// Package cache 提供分布式缓存功能。
//
// 本包实现了多种缓存策略，用于跨节点共享数据：
// - Redis Cluster 分布式缓存
// - 分层缓存（本地 + 远程）
// - 缓存预热和失效策略
//
// 示例:
//
//	// 创建 Redis 缓存
//	redisCache := cache.NewRedisCache(cache.RedisCacheConfig{
//	    Addrs:    []string{"localhost:6379"},
//	    Password: "",
//	    Prefix:   "langchain:",
//	})
//
//	// 设置缓存
//	ctx := context.Background()
//	err := redisCache.Set(ctx, "key1", []byte("value1"), 5*time.Minute)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// 获取缓存
//	value, err := redisCache.Get(ctx, "key1")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// 使用分层缓存
//	layeredCache := cache.NewLayeredCache(
//	    cache.NewMemoryCache(1000),
//	    redisCache,
//	)
//
// 支持的功能：
// - 基础 CRUD 操作
// - 批量操作
// - TTL 管理
// - 缓存预热
// - 自动失效
// - 分片策略
//
package cache
