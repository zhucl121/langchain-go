package cache

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisCache Redis 分布式缓存
//
// 基于 Redis Cluster 的分布式缓存实现。
type RedisCache struct {
	client *redis.ClusterClient
	prefix string
	stats  *Stats
}

// RedisCacheConfig Redis 缓存配置
type RedisCacheConfig struct {
	// Addrs Redis 集群地址
	Addrs []string

	// Password 密码
	Password string

	// Prefix 键前缀
	Prefix string

	// PoolSize 连接池大小
	PoolSize int

	// MinIdleConns 最小空闲连接数
	MinIdleConns int

	// MaxRetries 最大重试次数
	MaxRetries int

	// DialTimeout 连接超时
	DialTimeout time.Duration

	// ReadTimeout 读超时
	ReadTimeout time.Duration

	// WriteTimeout 写超时
	WriteTimeout time.Duration
}

// DefaultRedisCacheConfig 返回默认配置
func DefaultRedisCacheConfig() RedisCacheConfig {
	return RedisCacheConfig{
		Addrs:        []string{"localhost:6379"},
		Password:     "",
		Prefix:       "langchain:",
		PoolSize:     10,
		MinIdleConns: 2,
		MaxRetries:   3,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	}
}

// NewRedisCache 创建 Redis 缓存
func NewRedisCache(config RedisCacheConfig) (*RedisCache, error) {
	client := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:        config.Addrs,
		Password:     config.Password,
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
		return nil, err
	}

	return &RedisCache{
		client: client,
		prefix: config.Prefix,
		stats:  &Stats{},
	}, nil
}

// Get 获取缓存
func (c *RedisCache) Get(ctx context.Context, key string) ([]byte, error) {
	fullKey := c.prefix + key
	data, err := c.client.Get(ctx, fullKey).Bytes()

	if err == redis.Nil {
		c.stats.Misses++
		return nil, ErrCacheNotFound
	}

	if err != nil {
		return nil, err
	}

	c.stats.Hits++
	return data, nil
}

// Set 设置缓存
func (c *RedisCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	fullKey := c.prefix + key
	err := c.client.Set(ctx, fullKey, value, ttl).Err()

	if err == nil {
		c.stats.Sets++
	}

	return err
}

// Delete 删除缓存
func (c *RedisCache) Delete(ctx context.Context, key string) error {
	fullKey := c.prefix + key
	err := c.client.Del(ctx, fullKey).Err()

	if err == nil {
		c.stats.Deletes++
	}

	return err
}

// Exists 检查键是否存在
func (c *RedisCache) Exists(ctx context.Context, key string) (bool, error) {
	fullKey := c.prefix + key
	count, err := c.client.Exists(ctx, fullKey).Result()

	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// Clear 清空所有缓存
func (c *RedisCache) Clear(ctx context.Context) error {
	// 使用 SCAN 命令遍历所有匹配的键
	pattern := c.prefix + "*"
	iter := c.client.Scan(ctx, 0, pattern, 0).Iterator()

	for iter.Next(ctx) {
		if err := c.client.Del(ctx, iter.Val()).Err(); err != nil {
			return err
		}
	}

	return iter.Err()
}

// MGet 批量获取
func (c *RedisCache) MGet(ctx context.Context, keys []string) (map[string][]byte, error) {
	if len(keys) == 0 {
		return make(map[string][]byte), nil
	}

	fullKeys := make([]string, len(keys))
	for i, key := range keys {
		fullKeys[i] = c.prefix + key
	}

	values, err := c.client.MGet(ctx, fullKeys...).Result()
	if err != nil {
		return nil, err
	}

	result := make(map[string][]byte)
	for i, val := range values {
		if val != nil {
			if str, ok := val.(string); ok {
				result[keys[i]] = []byte(str)
				c.stats.Hits++
			}
		} else {
			c.stats.Misses++
		}
	}

	return result, nil
}

// MSet 批量设置
func (c *RedisCache) MSet(ctx context.Context, items map[string][]byte, ttl time.Duration) error {
	pipe := c.client.Pipeline()

	for key, value := range items {
		fullKey := c.prefix + key
		pipe.Set(ctx, fullKey, value, ttl)
	}

	_, err := pipe.Exec(ctx)
	if err == nil {
		c.stats.Sets += int64(len(items))
	}

	return err
}

// MDelete 批量删除
func (c *RedisCache) MDelete(ctx context.Context, keys []string) error {
	if len(keys) == 0 {
		return nil
	}

	fullKeys := make([]string, len(keys))
	for i, key := range keys {
		fullKeys[i] = c.prefix + key
	}

	err := c.client.Del(ctx, fullKeys...).Err()
	if err == nil {
		c.stats.Deletes += int64(len(keys))
	}

	return err
}

// Expire 设置过期时间
func (c *RedisCache) Expire(ctx context.Context, key string, ttl time.Duration) error {
	fullKey := c.prefix + key
	return c.client.Expire(ctx, fullKey, ttl).Err()
}

// TTL 获取剩余过期时间
func (c *RedisCache) TTL(ctx context.Context, key string) (time.Duration, error) {
	fullKey := c.prefix + key
	return c.client.TTL(ctx, fullKey).Result()
}

// Keys 获取匹配的键列表
func (c *RedisCache) Keys(ctx context.Context, pattern string) ([]string, error) {
	fullPattern := c.prefix + pattern
	keys, err := c.client.Keys(ctx, fullPattern).Result()

	if err != nil {
		return nil, err
	}

	// 移除前缀
	result := make([]string, len(keys))
	prefixLen := len(c.prefix)
	for i, key := range keys {
		if len(key) >= prefixLen {
			result[i] = key[prefixLen:]
		} else {
			result[i] = key
		}
	}

	return result, nil
}

// Stats 获取统计信息
func (c *RedisCache) Stats() *Stats {
	return c.stats
}

// Close 关闭连接
func (c *RedisCache) Close() error {
	return c.client.Close()
}
