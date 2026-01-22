# 分布式缓存示例

这个示例展示了 LangChain-Go 的分布式缓存功能，包括内存缓存、分层缓存和缓存驱逐策略。

## 功能演示

- ✅ 内存缓存 (Memory Cache)
- ✅ 缓存驱逐策略 (LRU, LFU, FIFO, TTL)
- ✅ 分层缓存 (Local + Remote)
- ✅ 缓存性能测试

## 运行示例

```bash
cd examples/cache_demo
go run main.go
```

## 示例输出

```
🚀 LangChain-Go 分布式缓存示例
========================================

==================================================
💾 内存缓存 (Memory Cache)
特点: 高速本地缓存，支持 TTL 和 LRU 驱逐

  1. 设置缓存...
    ✅ 设置 user:1001 = Alice
    ✅ 设置 user:1002 = Bob
    ✅ 设置 user:1003 = Charlie

  2. 获取缓存...
    ✅ 获取 user:1001 = Alice
    ✅ 获取 user:1002 = Bob
    ✅ 获取 user:1003 = Charlie

  3. 检查键是否存在...
    user:1001 存在: true
    user:9999 存在: false

  4. 删除缓存...
    ✅ 已删除 user:1002
    user:1002 存在: false

  5. 统计信息:
    总大小: 2 项
    命中: 3 次
    未命中: 1 次
    命中率: 75.00%
    设置: 3 次
    删除: 1 次

==================================================
🔄 缓存驱逐策略 (Eviction Policy)
特点: LRU 自动驱逐最久未使用的条目

  1. 填满缓存（最大 3 项）...
    ✅ 设置 key1 = value1
    ✅ 设置 key2 = value2
    ✅ 设置 key3 = value3

    当前大小: 3 / 3

  2. 访问 key1（更新访问时间）...
  3. 访问 key3（更新访问时间）...

  4. 添加新键 key4（触发驱逐）...
    ✅ 设置 key4 = value4

  5. 检查驱逐结果:
    ✅ key1 仍在缓存中
    ❌ key2 已被驱逐（LRU）
    ✅ key3 仍在缓存中
    ✅ key4 仍在缓存中

  6. 驱逐统计:
    驱逐次数: 1
    当前大小: 3 / 3

==================================================
🔗 分层缓存 (Layered Cache)
特点: 本地 + 远程两层缓存，自动回写

  1. 写入数据（写穿模式）...
    ✅ 设置 product:1001 = iPhone 15 Pro (本地+远程)
    ✅ 设置 product:1002 = MacBook Pro (本地+远程)
    ✅ 设置 product:1003 = AirPods Pro (本地+远程)

  2. 清空本地缓存（模拟失效）...
    ✅ 本地缓存已清空

  3. 读取数据（自动回写）...
    ✅ 获取 product:1001 = iPhone 15 Pro (从远程回写到本地)
    ✅ 获取 product:1002 = MacBook Pro (从远程回写到本地)
    ✅ 获取 product:1003 = AirPods Pro (从远程回写到本地)

  4. 验证本地缓存回写...
    ✅ product:1001 已回写到本地缓存
    ✅ product:1002 已回写到本地缓存
    ✅ product:1003 已回写到本地缓存

  5. 批量操作...
    ✅ 批量获取 3 个键
       product:1001 = iPhone 15 Pro
       product:1002 = MacBook Pro
       product:1003 = AirPods Pro

  6. 统计信息:
    命中: 9 次
    未命中: 0 次
    命中率: 100.00%

==================================================
⚡ 缓存性能测试
特点: 高并发读写性能

  1. 写入性能测试（1000 个键）...
    ✅ 完成 1000 次写入，耗时: 2.5ms
    写入速度: 400000 ops/s

  2. 读取性能测试（10000 次）...
    ✅ 完成 10000 次读取，耗时: 8.2ms
    读取速度: 1219512 ops/s
    命中率: 100.00%

  3. 混合操作测试（80% 读 + 20% 写）...
    ✅ 完成 5000 次混合操作，耗时: 4.8ms
    操作速度: 1041666 ops/s

  4. 最终统计:
    总大小: 1000 项
    总命中: 14000 次
    总未命中: 0 次
    总设置: 2000 次
    命中率: 100.00%

==================================================
✅ 所有演示完成！
```

## 缓存类型说明

### 1. Memory Cache（内存缓存）

**特点**:
- 高速本地缓存
- 支持 TTL 自动过期
- 支持多种驱逐策略
- 线程安全

**适用场景**:
- 热点数据缓存
- 会话数据
- 临时计算结果

### 2. Redis Cache（Redis 缓存）

**特点**:
- 分布式共享
- 持久化支持
- 集群支持
- 跨节点访问

**适用场景**:
- 跨节点共享数据
- 持久化缓存
- 分布式锁
- 消息队列

### 3. Layered Cache（分层缓存）

**特点**:
- 本地 + 远程两层
- 自动回写
- 写穿/写回模式
- 智能降级

**适用场景**:
- 高性能要求
- 热点数据自动提升
- 网络波动容忍
- 成本优化

## 驱逐策略对比

| 策略 | 特点 | 适用场景 |
|------|------|----------|
| LRU | 驱逐最久未使用 | 通用场景 |
| LFU | 驱逐访问频率最低 | 热点数据 |
| FIFO | 驱逐最早加入 | 简单队列 |
| TTL | 按过期时间驱逐 | 临时数据 |

## 性能特点

### 内存缓存
- **写入速度**: 400K+ ops/s
- **读取速度**: 1.2M+ ops/s
- **延迟**: < 1μs

### Redis 缓存
- **写入速度**: 50K+ ops/s
- **读取速度**: 100K+ ops/s
- **延迟**: < 1ms

### 分层缓存
- **本地命中**: < 1μs
- **远程命中**: < 1ms
- **自动回写**: 异步

## 配置建议

### 开发环境
```go
// 使用纯内存缓存
cache := cache.NewMemoryCache(1000)
```

### 生产环境
```go
// 使用分层缓存
local := cache.NewMemoryCache(1000)
remote := cache.NewRedisCache(cache.RedisCacheConfig{
    Addrs: []string{"redis-1:6379", "redis-2:6379"},
})
layered := cache.NewLayeredCache(local, remote)
```

### 高性能场景
```go
// 配置大容量本地缓存 + 写回模式
config := cache.LayeredCacheConfig{
    LocalTTL:     5 * time.Minute,
    RemoteTTL:    30 * time.Minute,
    WriteBack:    true,  // 异步写远程
    ReadThrough:  true,
}
layered := cache.NewLayeredCacheWithConfig(local, remote, config)
```

## 下一步

查看更多示例：
- 负载均衡示例：`examples/balancer_demo/`
- 集群管理示例：`examples/cluster_demo/`

## 相关文档

- [V0.5.0 实施计划](../../docs/V0.5.0_IMPLEMENTATION_PLAN.md)
- [V0.5.0 进度跟踪](../../docs/V0.5.0_PROGRESS.md)
