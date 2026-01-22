package cache

import (
	"context"
	"time"
)

// LayeredCache 分层缓存
//
// 结合本地缓存和远程缓存，提供两层缓存策略。
// 优先从本地缓存读取，未命中时从远程缓存读取并回写本地。
type LayeredCache struct {
	local  Cache
	remote DistributedCache
	config LayeredCacheConfig
	stats  *Stats
}

// LayeredCacheConfig 分层缓存配置
type LayeredCacheConfig struct {
	// LocalTTL 本地缓存 TTL
	LocalTTL time.Duration

	// RemoteTTL 远程缓存 TTL
	RemoteTTL time.Duration

	// WriteThrough 是否写穿（同时写入本地和远程）
	WriteThrough bool

	// WriteBack 是否写回（只写本地，异步写远程）
	WriteBack bool

	// ReadThrough 是否读穿（本地未命中时从远程读取）
	ReadThrough bool
}

// DefaultLayeredCacheConfig 返回默认配置
func DefaultLayeredCacheConfig() LayeredCacheConfig {
	return LayeredCacheConfig{
		LocalTTL:     5 * time.Minute,
		RemoteTTL:    30 * time.Minute,
		WriteThrough: true,
		WriteBack:    false,
		ReadThrough:  true,
	}
}

// NewLayeredCache 创建分层缓存
func NewLayeredCache(local Cache, remote DistributedCache) *LayeredCache {
	return NewLayeredCacheWithConfig(local, remote, DefaultLayeredCacheConfig())
}

// NewLayeredCacheWithConfig 使用配置创建分层缓存
func NewLayeredCacheWithConfig(local Cache, remote DistributedCache, config LayeredCacheConfig) *LayeredCache {
	return &LayeredCache{
		local:  local,
		remote: remote,
		config: config,
		stats:  &Stats{},
	}
}

// Get 获取缓存
func (c *LayeredCache) Get(ctx context.Context, key string) ([]byte, error) {
	// 1. 先尝试从本地缓存读取
	if data, err := c.local.Get(ctx, key); err == nil {
		c.stats.Hits++
		return data, nil
	}

	// 2. 本地未命中，如果启用读穿，从远程读取
	if c.config.ReadThrough {
		data, err := c.remote.Get(ctx, key)
		if err != nil {
			c.stats.Misses++
			return nil, err
		}

		// 3. 回写到本地缓存
		if err := c.local.Set(ctx, key, data, c.config.LocalTTL); err == nil {
			c.stats.Hits++
		}

		return data, nil
	}

	c.stats.Misses++
	return nil, ErrCacheNotFound
}

// Set 设置缓存
func (c *LayeredCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	c.stats.Sets++

	if c.config.WriteThrough {
		// 写穿：同时写入本地和远程
		errLocal := c.local.Set(ctx, key, value, c.config.LocalTTL)
		errRemote := c.remote.Set(ctx, key, value, c.config.RemoteTTL)

		// 只要有一个成功就算成功
		if errLocal != nil && errRemote != nil {
			return errRemote
		}

		return nil
	}

	if c.config.WriteBack {
		// 写回：先写本地，异步写远程
		if err := c.local.Set(ctx, key, value, c.config.LocalTTL); err != nil {
			return err
		}

		// 异步写远程
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			c.remote.Set(ctx, key, value, c.config.RemoteTTL)
		}()

		return nil
	}

	// 默认：只写本地
	return c.local.Set(ctx, key, value, c.config.LocalTTL)
}

// Delete 删除缓存
func (c *LayeredCache) Delete(ctx context.Context, key string) error {
	c.stats.Deletes++

	// 同时删除本地和远程
	errLocal := c.local.Delete(ctx, key)
	errRemote := c.remote.Delete(ctx, key)

	// 只要有一个成功就算成功
	if errLocal != nil && errRemote != nil {
		return errRemote
	}

	return nil
}

// Exists 检查键是否存在
func (c *LayeredCache) Exists(ctx context.Context, key string) (bool, error) {
	// 先检查本地
	if exists, err := c.local.Exists(ctx, key); err == nil && exists {
		return true, nil
	}

	// 再检查远程
	return c.remote.Exists(ctx, key)
}

// Clear 清空所有缓存
func (c *LayeredCache) Clear(ctx context.Context) error {
	errLocal := c.local.Clear(ctx)
	errRemote := c.remote.Clear(ctx)

	if errLocal != nil && errRemote != nil {
		return errRemote
	}

	return nil
}

// MGet 批量获取
func (c *LayeredCache) MGet(ctx context.Context, keys []string) (map[string][]byte, error) {
	result := make(map[string][]byte)
	missedKeys := make([]string, 0)

	// 1. 从本地缓存批量获取
	for _, key := range keys {
		if data, err := c.local.Get(ctx, key); err == nil {
			result[key] = data
			c.stats.Hits++
		} else {
			missedKeys = append(missedKeys, key)
		}
	}

	// 2. 如果有未命中的键，从远程获取
	if len(missedKeys) > 0 && c.config.ReadThrough {
		remoteData, err := c.remote.MGet(ctx, missedKeys)
		if err != nil {
			return result, err
		}

		// 3. 回写到本地缓存
		for key, data := range remoteData {
			result[key] = data
			c.local.Set(ctx, key, data, c.config.LocalTTL)
			c.stats.Hits++
		}
	}

	c.stats.Misses += int64(len(missedKeys))

	return result, nil
}

// MSet 批量设置
func (c *LayeredCache) MSet(ctx context.Context, items map[string][]byte, ttl time.Duration) error {
	c.stats.Sets += int64(len(items))

	if c.config.WriteThrough {
		// 写穿：同时写入本地和远程
		for key, value := range items {
			c.local.Set(ctx, key, value, c.config.LocalTTL)
		}
		return c.remote.MSet(ctx, items, c.config.RemoteTTL)
	}

	if c.config.WriteBack {
		// 写回：先写本地，异步写远程
		for key, value := range items {
			if err := c.local.Set(ctx, key, value, c.config.LocalTTL); err != nil {
				return err
			}
		}

		// 异步写远程
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			c.remote.MSet(ctx, items, c.config.RemoteTTL)
		}()

		return nil
	}

	// 默认：只写本地
	for key, value := range items {
		if err := c.local.Set(ctx, key, value, c.config.LocalTTL); err != nil {
			return err
		}
	}

	return nil
}

// MDelete 批量删除
func (c *LayeredCache) MDelete(ctx context.Context, keys []string) error {
	c.stats.Deletes += int64(len(keys))

	// 同时删除本地和远程
	for _, key := range keys {
		c.local.Delete(ctx, key)
	}

	return c.remote.MDelete(ctx, keys)
}

// Expire 设置过期时间
func (c *LayeredCache) Expire(ctx context.Context, key string, ttl time.Duration) error {
	return c.remote.Expire(ctx, key, ttl)
}

// TTL 获取剩余过期时间
func (c *LayeredCache) TTL(ctx context.Context, key string) (time.Duration, error) {
	return c.remote.TTL(ctx, key)
}

// Keys 获取匹配的键列表
func (c *LayeredCache) Keys(ctx context.Context, pattern string) ([]string, error) {
	return c.remote.Keys(ctx, pattern)
}

// Stats 获取统计信息
func (c *LayeredCache) Stats() *Stats {
	return c.stats
}

// Close 关闭缓存
func (c *LayeredCache) Close() error {
	if closer, ok := c.local.(interface{ Close() error }); ok {
		closer.Close()
	}
	return c.remote.Close()
}

// InvalidateLocal 失效本地缓存
func (c *LayeredCache) InvalidateLocal(ctx context.Context, key string) error {
	return c.local.Delete(ctx, key)
}

// InvalidateLocalPattern 按模式失效本地缓存
func (c *LayeredCache) InvalidateLocalPattern(ctx context.Context, pattern string) error {
	// 本地缓存通常不支持模式匹配，直接清空
	return c.local.Clear(ctx)
}
