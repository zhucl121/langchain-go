# Redis 缓存后端文档

## 概述

Redis 缓存后端提供了分布式缓存能力，适用于多实例部署和高并发场景。

## 特性

✅ **分布式缓存** - 多个应用实例共享缓存  
✅ **高性能** - 亚毫秒级响应时间  
✅ **持久化** - 支持 RDB 和 AOF 持久化  
✅ **集群模式** - 支持 Redis Cluster  
✅ **原子操作** - 支持 SetNX、Incr/Decr 等  
✅ **统一接口** - 与内存缓存相同的 API

## 快速开始

### 1. 安装 Redis

```bash
# Docker
docker run -d -p 6379:6379 redis

# macOS
brew install redis
redis-server

# Ubuntu
apt-get install redis-server
systemctl start redis
```

### 2. 基础使用

```go
package main

import (
    "context"
    "log"
    "time"
    
    "langchain-go/core/cache"
)

func main() {
    // 创建 Redis 缓存
    config := cache.DefaultRedisCacheConfig()
    config.Addr = "localhost:6379"
    config.Password = "" // 如果需要
    
    redisCache, err := cache.NewRedisCache(config)
    if err != nil {
        log.Fatal(err)
    }
    defer redisCache.Close()
    
    ctx := context.Background()
    
    // 设置缓存
    err = redisCache.Set(ctx, "key", "value", time.Hour)
    if err != nil {
        log.Fatal(err)
    }
    
    // 获取缓存
    value, found, err := redisCache.Get(ctx, "key")
    if err != nil {
        log.Fatal(err)
    }
    
    if found {
        log.Printf("Value: %v", value)
    }
}
```

### 3. LLM 缓存

```go
// 创建 LLM 缓存
llmCache := cache.NewLLMCache(redisCache)

// 检查缓存
if cached, found := llmCache.Get(ctx, prompt); found {
    return cached, nil
}

// 调用 LLM
response, err := llm.Call(ctx, messages)
if err != nil {
    return "", err
}

// 缓存结果
llmCache.Set(ctx, prompt, response.Content, 24*time.Hour)
```

### 4. 工具缓存

```go
// 创建工具缓存
toolCache := cache.NewToolCache(redisCache)

// 检查缓存
if cached, found := toolCache.Get(ctx, toolName, input); found {
    return cached, nil
}

// 执行工具
result, err := tool.Run(ctx, input)
if err != nil {
    return "", err
}

// 缓存结果
toolCache.Set(ctx, toolName, input, result, time.Hour)
```

## 配置选项

### RedisCacheConfig

```go
type RedisCacheConfig struct {
    // Addr Redis 地址 (默认: "localhost:6379")
    Addr string
    
    // Password Redis 密码 (可选)
    Password string
    
    // DB Redis 数据库编号 (默认: 0)
    DB int
    
    // Prefix 键前缀 (默认: "langchain:")
    Prefix string
    
    // PoolSize 连接池大小 (默认: 10)
    PoolSize int
    
    // MinIdleConns 最小空闲连接数 (默认: 5)
    MinIdleConns int
    
    // MaxRetries 最大重试次数 (默认: 3)
    MaxRetries int
    
    // DialTimeout 连接超时 (默认: 5s)
    DialTimeout time.Duration
    
    // ReadTimeout 读取超时 (默认: 3s)
    ReadTimeout time.Duration
    
    // WriteTimeout 写入超时 (默认: 3s)
    WriteTimeout time.Duration
}
```

### 生产环境推荐配置

```go
config := cache.RedisCacheConfig{
    Addr:         "redis.prod.example.com:6379",
    Password:     os.Getenv("REDIS_PASSWORD"),
    DB:           0,
    Prefix:       "prod:langchain:",
    PoolSize:     20,                    // 增加连接池
    MinIdleConns: 10,                    // 保持足够的空闲连接
    MaxRetries:   3,                     // 重试失败的操作
    DialTimeout:  5 * time.Second,
    ReadTimeout:  3 * time.Second,
    WriteTimeout: 3 * time.Second,
}
```

## Redis 集群模式

```go
// 创建 Redis 集群缓存
config := cache.RedisClusterConfig{
    Addrs: []string{
        "redis-1:7000",
        "redis-2:7001",
        "redis-3:7002",
    },
    Password:     os.Getenv("REDIS_PASSWORD"),
    Prefix:       "cluster:",
    DialTimeout:  5 * time.Second,
    ReadTimeout:  3 * time.Second,
    WriteTimeout: 3 * time.Second,
}

clusterCache, err := cache.NewRedisClusterCache(config)
if err != nil {
    log.Fatal(err)
}
defer clusterCache.Close()

// 使用与单机版相同的 API
clusterCache.Set(ctx, "key", "value", time.Hour)
value, found, _ := clusterCache.Get(ctx, "key")
```

## 高级特性

### 1. 分布式锁 (SetNX)

```go
// 获取锁
acquired, err := redisCache.SetNX(ctx, "lock:resource", "locked", 10*time.Second)
if err != nil {
    return err
}

if acquired {
    // 执行关键操作
    defer redisCache.Delete(ctx, "lock:resource")
    
    // ... 业务逻辑 ...
} else {
    return errors.New("resource locked")
}
```

### 2. 原子计数器

```go
// 递增
count, err := redisCache.Increment(ctx, "page:views", 1)
if err != nil {
    return err
}
log.Printf("Page views: %d", count)

// 递减
count, err = redisCache.Decrement(ctx, "quota:remaining", 1)
if err != nil {
    return err
}
```

### 3. 键管理

```go
// 列出所有键
keys, err := redisCache.Keys(ctx, "*")

// 检查键是否存在
exists, err := redisCache.Exists(ctx, "my-key")

// 获取 TTL
ttl, err := redisCache.TTL(ctx, "my-key")

// 删除键
err = redisCache.Delete(ctx, "my-key")

// 清空所有键
err = redisCache.Clear(ctx)
```

### 4. 统计信息

```go
stats := redisCache.Stats()
log.Printf("Hits: %d", stats.Hits)
log.Printf("Misses: %d", stats.Misses)
log.Printf("Hit Rate: %.2f%%", stats.HitRate*100)
```

## 性能对比

| 操作 | 内存缓存 | Redis | 差异 |
|------|----------|-------|------|
| Set  | 50ns     | 500µs | 10,000x |
| Get  | 30ns     | 300µs | 10,000x |
| 命中率 | 本地    | 分布式 | - |
| 扩展性 | 单机    | 集群   | ✅ |
| 持久化 | 否      | 是     | ✅ |
| 多进程共享 | 否 | 是 | ✅ |

## 使用场景

### 开发/测试环境
```go
// 使用内存缓存（快速、简单）
cache := cache.NewInMemoryCache()
```

### 单机部署
```go
// 内存缓存 + 定期持久化
cache := cache.NewInMemoryCache()
```

### 分布式部署
```go
// Redis 缓存（必需）
cache, _ := cache.NewRedisCache(config)
```

### 高并发场景
```go
// Redis 集群
cache, _ := cache.NewRedisClusterCache(config)
```

## 成本优化

### LLM 成本节省

假设：
- 10,000 次 LLM 调用/天
- 平均 1K tokens/次
- LLM 成本: $0.002/1K tokens

**无缓存**:
- 成本: 10,000 × $0.002 = $20/天 = $600/月

**有缓存（50% 命中率）**:
- LLM 成本: $10/天
- Redis 成本: $5/月
- 总成本: $305/月
- **节省: $295/月 (49%)**

**有缓存（90% 命中率）**:
- LLM 成本: $2/天
- Redis 成本: $5/月
- 总成本: $65/月
- **节省: $535/月 (89%)**

### 响应时间优化

| 场景 | 无缓存 | 有缓存 | 提升 |
|------|--------|--------|------|
| LLM 调用 | 2000ms | 10ms | 200x |
| 工具调用 | 500ms | 5ms | 100x |
| 用户体验 | 慢 | 快 | ⭐⭐⭐⭐⭐ |

## 运维建议

### 1. Redis 配置

```conf
# redis.conf

# 内存策略
maxmemory 2gb
maxmemory-policy allkeys-lru

# 持久化
save 900 1
save 300 10
save 60 10000
appendonly yes

# 网络
timeout 300
tcp-keepalive 60

# 安全
requirepass your-secure-password
```

### 2. 监控指标

- 命中率 (Hit Rate)
- 内存使用 (Used Memory)
- 连接数 (Connected Clients)
- 操作延迟 (Latency)
- 键数量 (Keys)

### 3. 高可用

**Redis Sentinel**:
```go
// 使用 Sentinel 模式
client := redis.NewFailoverClient(&redis.FailoverOptions{
    MasterName:    "mymaster",
    SentinelAddrs: []string{"sentinel:26379"},
})
```

**Redis Cluster**:
```go
// 使用集群模式
cache, _ := cache.NewRedisClusterCache(config)
```

### 4. 备份策略

- RDB: 每天全量备份
- AOF: 实时增量备份
- 定期测试恢复

## 故障排查

### 连接失败

```go
// 检查连接
err := redisCache.Ping(ctx)
if err != nil {
    log.Printf("Redis connection failed: %v", err)
}
```

### 性能问题

```go
// 检查统计
stats := redisCache.Stats()
if stats.HitRate < 0.5 {
    log.Printf("Low hit rate: %.2f%%", stats.HitRate*100)
}
```

### 内存不足

```bash
# 检查内存使用
redis-cli INFO memory

# 清理过期键
redis-cli --scan --pattern "prefix:*" | xargs redis-cli DEL
```

## API 参考

### RedisCache 方法

| 方法 | 说明 |
|------|------|
| `Get(ctx, key)` | 获取缓存值 |
| `Set(ctx, key, value, ttl)` | 设置缓存值 |
| `Delete(ctx, key)` | 删除缓存值 |
| `Clear(ctx)` | 清空所有缓存 |
| `Stats()` | 获取统计信息 |
| `Ping(ctx)` | 测试连接 |
| `Keys(ctx, pattern)` | 列出匹配的键 |
| `Exists(ctx, key)` | 检查键是否存在 |
| `TTL(ctx, key)` | 获取 TTL |
| `SetNX(ctx, key, value, ttl)` | 仅当键不存在时设置 |
| `Increment(ctx, key, delta)` | 原子递增 |
| `Decrement(ctx, key, delta)` | 原子递减 |
| `Close()` | 关闭连接 |

## 最佳实践

1. **使用密码认证** - 生产环境必须
2. **设置合理的 TTL** - 避免内存溢出
3. **使用键前缀** - 隔离不同应用
4. **监控连接池** - 调整 PoolSize
5. **启用持久化** - 防止数据丢失
6. **定期备份** - 重要数据
7. **配置高可用** - Sentinel 或 Cluster

## 示例代码

完整示例请参考：
- `examples/redis_cache_demo.go` - Redis 缓存基础使用
- `core/cache/redis_test.go` - 单元测试

## 相关文档

- [缓存架构设计](./CACHE_ARCHITECTURE.md)
- [性能优化指南](../advanced/performance.md)
- [生产环境部署](../deployment/production.md)
