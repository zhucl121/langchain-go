package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisCache Redis 缓存实现。
//
// 使用 Redis 作为缓存后端，支持分布式部署。
type RedisCache struct {
	client *redis.Client
	prefix string
	hits   int64
	misses int64
}

// RedisCacheConfig Redis 缓存配置。
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

// DefaultRedisCacheConfig 返回默认 Redis 配置。
func DefaultRedisCacheConfig() RedisCacheConfig {
	return RedisCacheConfig{
		Addr:         "localhost:6379",
		Password:     "",
		DB:           0,
		Prefix:       "langchain:",
		PoolSize:     10,
		MinIdleConns: 5,
		MaxRetries:   3,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	}
}

// NewRedisCache 创建 Redis 缓存。
//
// 参数：
//   - config: Redis 配置
//
// 返回：
//   - *RedisCache: Redis 缓存实例
//   - error: 错误
//
// 示例：
//
//	config := cache.DefaultRedisCacheConfig()
//	config.Addr = "localhost:6379"
//	config.Password = "your-password"
//	
//	redisCache, err := cache.NewRedisCache(config)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
func NewRedisCache(config RedisCacheConfig) (*RedisCache, error) {
	// 创建 Redis 客户端
	client := redis.NewClient(&redis.Options{
		Addr:         config.Addr,
		Password:     config.Password,
		DB:           config.DB,
		PoolSize:     config.PoolSize,
		MinIdleConns: config.MinIdleConns,
		MaxRetries:   config.MaxRetries,
		DialTimeout:  config.DialTimeout,
		ReadTimeout:  config.ReadTimeout,
		WriteTimeout: config.WriteTimeout,
	})

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisCache{
		client: client,
		prefix: config.Prefix,
	}, nil
}

// Get 实现 Cache 接口。
func (r *RedisCache) Get(ctx context.Context, key string) (any, bool, error) {
	fullKey := r.prefix + key

	// 从 Redis 获取
	val, err := r.client.Get(ctx, fullKey).Result()
	if err == redis.Nil {
		// 键不存在
		r.misses++
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("redis get error: %w", err)
	}

	// 反序列化
	var value any
	if err := json.Unmarshal([]byte(val), &value); err != nil {
		return nil, false, fmt.Errorf("failed to unmarshal value: %w", err)
	}

	r.hits++
	return value, true, nil
}

// Set 实现 Cache 接口。
func (r *RedisCache) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	fullKey := r.prefix + key

	// 序列化
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	// 设置到 Redis
	if ttl > 0 {
		err = r.client.Set(ctx, fullKey, data, ttl).Err()
	} else {
		err = r.client.Set(ctx, fullKey, data, 0).Err()
	}

	if err != nil {
		return fmt.Errorf("redis set error: %w", err)
	}

	return nil
}

// Delete 实现 Cache 接口。
func (r *RedisCache) Delete(ctx context.Context, key string) error {
	fullKey := r.prefix + key

	err := r.client.Del(ctx, fullKey).Err()
	if err != nil {
		return fmt.Errorf("redis delete error: %w", err)
	}

	return nil
}

// Clear 实现 Cache 接口。
func (r *RedisCache) Clear(ctx context.Context) error {
	// 使用 SCAN 命令遍历所有匹配的键
	pattern := r.prefix + "*"
	
	iter := r.client.Scan(ctx, 0, pattern, 0).Iterator()
	for iter.Next(ctx) {
		if err := r.client.Del(ctx, iter.Val()).Err(); err != nil {
			return fmt.Errorf("redis clear error: %w", err)
		}
	}
	
	if err := iter.Err(); err != nil {
		return fmt.Errorf("redis scan error: %w", err)
	}

	return nil
}

// Stats 实现 Cache 接口。
func (r *RedisCache) Stats() *CacheStats {
	stats := &CacheStats{
		Hits:   r.hits,
		Misses: r.misses,
		Size:   0, // Redis 不容易获取准确大小
	}

	total := r.hits + r.misses
	if total > 0 {
		stats.HitRate = float64(r.hits) / float64(total)
	}

	return stats
}

// Close 关闭 Redis 连接。
func (r *RedisCache) Close() error {
	return r.client.Close()
}

// GetClient 获取底层 Redis 客户端。
//
// 用于高级操作。
func (r *RedisCache) GetClient() *redis.Client {
	return r.client
}

// Ping 测试 Redis 连接。
func (r *RedisCache) Ping(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

// Keys 列出所有匹配的键。
//
// 参数：
//   - ctx: 上下文
//   - pattern: 匹配模式 (例如: "*")
//
// 返回：
//   - []string: 键列表
//   - error: 错误
//
func (r *RedisCache) Keys(ctx context.Context, pattern string) ([]string, error) {
	fullPattern := r.prefix + pattern
	
	var keys []string
	iter := r.client.Scan(ctx, 0, fullPattern, 0).Iterator()
	for iter.Next(ctx) {
		// 移除前缀
		key := iter.Val()
		if len(key) > len(r.prefix) {
			keys = append(keys, key[len(r.prefix):])
		}
	}
	
	if err := iter.Err(); err != nil {
		return nil, err
	}

	return keys, nil
}

// Exists 检查键是否存在。
func (r *RedisCache) Exists(ctx context.Context, key string) (bool, error) {
	fullKey := r.prefix + key
	
	n, err := r.client.Exists(ctx, fullKey).Result()
	if err != nil {
		return false, err
	}
	
	return n > 0, nil
}

// TTL 获取键的剩余过期时间。
func (r *RedisCache) TTL(ctx context.Context, key string) (time.Duration, error) {
	fullKey := r.prefix + key
	
	ttl, err := r.client.TTL(ctx, fullKey).Result()
	if err != nil {
		return 0, err
	}
	
	return ttl, nil
}

// SetNX 仅当键不存在时设置。
//
// 返回：
//   - bool: 是否设置成功
//   - error: 错误
//
func (r *RedisCache) SetNX(ctx context.Context, key string, value any, ttl time.Duration) (bool, error) {
	fullKey := r.prefix + key

	// 序列化
	data, err := json.Marshal(value)
	if err != nil {
		return false, fmt.Errorf("failed to marshal value: %w", err)
	}

	// 使用 SetNX
	ok, err := r.client.SetNX(ctx, fullKey, data, ttl).Result()
	if err != nil {
		return false, fmt.Errorf("redis setnx error: %w", err)
	}

	return ok, nil
}

// Increment 原子递增。
func (r *RedisCache) Increment(ctx context.Context, key string, delta int64) (int64, error) {
	fullKey := r.prefix + key
	
	val, err := r.client.IncrBy(ctx, fullKey, delta).Result()
	if err != nil {
		return 0, err
	}
	
	return val, nil
}

// Decrement 原子递减。
func (r *RedisCache) Decrement(ctx context.Context, key string, delta int64) (int64, error) {
	fullKey := r.prefix + key
	
	val, err := r.client.DecrBy(ctx, fullKey, delta).Result()
	if err != nil {
		return 0, err
	}
	
	return val, nil
}

// RedisClusterCache Redis 集群缓存实现。
//
// 用于 Redis Cluster 模式。
type RedisClusterCache struct {
	client *redis.ClusterClient
	prefix string
	hits   int64
	misses int64
}

// RedisClusterConfig Redis 集群配置。
type RedisClusterConfig struct {
	// Addrs Redis 集群地址列表
	Addrs []string

	// Password Redis 密码 (可选)
	Password string

	// Prefix 键前缀 (默认: "langchain:")
	Prefix string

	// MaxRetries 最大重试次数 (默认: 3)
	MaxRetries int

	// DialTimeout 连接超时 (默认: 5s)
	DialTimeout time.Duration

	// ReadTimeout 读取超时 (默认: 3s)
	ReadTimeout time.Duration

	// WriteTimeout 写入超时 (默认: 3s)
	WriteTimeout time.Duration
}

// NewRedisClusterCache 创建 Redis 集群缓存。
//
// 参数：
//   - config: Redis 集群配置
//
// 返回：
//   - *RedisClusterCache: Redis 集群缓存实例
//   - error: 错误
//
func NewRedisClusterCache(config RedisClusterConfig) (*RedisClusterCache, error) {
	// 创建 Redis 集群客户端
	client := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:        config.Addrs,
		Password:     config.Password,
		MaxRetries:   config.MaxRetries,
		DialTimeout:  config.DialTimeout,
		ReadTimeout:  config.ReadTimeout,
		WriteTimeout: config.WriteTimeout,
	})

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis Cluster: %w", err)
	}

	return &RedisClusterCache{
		client: client,
		prefix: config.Prefix,
	}, nil
}

// Get 实现 Cache 接口。
func (r *RedisClusterCache) Get(ctx context.Context, key string) (any, bool, error) {
	fullKey := r.prefix + key

	val, err := r.client.Get(ctx, fullKey).Result()
	if err == redis.Nil {
		r.misses++
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("redis cluster get error: %w", err)
	}

	var value any
	if err := json.Unmarshal([]byte(val), &value); err != nil {
		return nil, false, fmt.Errorf("failed to unmarshal value: %w", err)
	}

	r.hits++
	return value, true, nil
}

// Set 实现 Cache 接口。
func (r *RedisClusterCache) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	fullKey := r.prefix + key

	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	if ttl > 0 {
		err = r.client.Set(ctx, fullKey, data, ttl).Err()
	} else {
		err = r.client.Set(ctx, fullKey, data, 0).Err()
	}

	if err != nil {
		return fmt.Errorf("redis cluster set error: %w", err)
	}

	return nil
}

// Delete 实现 Cache 接口。
func (r *RedisClusterCache) Delete(ctx context.Context, key string) error {
	fullKey := r.prefix + key
	return r.client.Del(ctx, fullKey).Err()
}

// Clear 实现 Cache 接口。
func (r *RedisClusterCache) Clear(ctx context.Context) error {
	// Redis Cluster 的 SCAN 需要遍历所有节点
	// 这里简化实现
	pattern := r.prefix + "*"
	
	err := r.client.ForEachMaster(ctx, func(ctx context.Context, client *redis.Client) error {
		iter := client.Scan(ctx, 0, pattern, 0).Iterator()
		for iter.Next(ctx) {
			if err := client.Del(ctx, iter.Val()).Err(); err != nil {
				return err
			}
		}
		return iter.Err()
	})

	return err
}

// Stats 实现 Cache 接口。
func (r *RedisClusterCache) Stats() *CacheStats {
	stats := &CacheStats{
		Hits:   r.hits,
		Misses: r.misses,
		Size:   0,
	}

	total := r.hits + r.misses
	if total > 0 {
		stats.HitRate = float64(r.hits) / float64(total)
	}

	return stats
}

// Close 关闭 Redis 集群连接。
func (r *RedisClusterCache) Close() error {
	return r.client.Close()
}
