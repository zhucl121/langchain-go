package cache

import (
	"context"
	"sync"
	"time"
)

// MemoryCache 内存缓存
//
// 使用哈希表实现的内存缓存，支持 LRU 驱逐策略。
type MemoryCache struct {
	data      map[string]*CacheEntry
	maxSize   int
	eviction  EvictionPolicy
	mu        sync.RWMutex
	stats     *Stats
	ttlTicker *time.Ticker
	stopCh    chan struct{}
}

// MemoryCacheConfig 内存缓存配置
type MemoryCacheConfig struct {
	// MaxSize 最大条目数
	MaxSize int

	// EvictionPolicy 驱逐策略
	EvictionPolicy EvictionPolicy

	// CleanupInterval 清理间隔
	CleanupInterval time.Duration
}

// DefaultMemoryCacheConfig 默认配置
func DefaultMemoryCacheConfig() MemoryCacheConfig {
	return MemoryCacheConfig{
		MaxSize:         1000,
		EvictionPolicy:  EvictionPolicyLRU,
		CleanupInterval: 1 * time.Minute,
	}
}

// NewMemoryCache 创建内存缓存
func NewMemoryCache(maxSize int) *MemoryCache {
	config := DefaultMemoryCacheConfig()
	config.MaxSize = maxSize
	return NewMemoryCacheWithConfig(config)
}

// NewMemoryCacheWithConfig 使用配置创建内存缓存
func NewMemoryCacheWithConfig(config MemoryCacheConfig) *MemoryCache {
	mc := &MemoryCache{
		data:     make(map[string]*CacheEntry),
		maxSize:  config.MaxSize,
		eviction: config.EvictionPolicy,
		stats: &Stats{
			MaxSize: int64(config.MaxSize),
		},
		stopCh: make(chan struct{}),
	}

	// 启动定期清理
	if config.CleanupInterval > 0 {
		mc.ttlTicker = time.NewTicker(config.CleanupInterval)
		go mc.cleanupExpired()
	}

	return mc
}

// Get 获取缓存
func (c *MemoryCache) Get(ctx context.Context, key string) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, ok := c.data[key]
	if !ok {
		c.stats.Misses++
		return nil, ErrCacheNotFound
	}

	if entry.IsExpired() {
		delete(c.data, key)
		c.stats.Misses++
		c.stats.Size--
		return nil, ErrCacheExpired
	}

	// 更新访问信息
	entry.AccessCount++
	entry.LastAccessAt = time.Now()

	c.stats.Hits++
	return entry.Value, nil
}

// Set 设置缓存
func (c *MemoryCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 检查是否需要驱逐
	if len(c.data) >= c.maxSize {
		if _, exists := c.data[key]; !exists {
			// 新键且缓存已满，需要驱逐
			c.evict()
		}
	}

	now := time.Now()
	entry := &CacheEntry{
		Key:          key,
		Value:        value,
		CreatedAt:    now,
		LastAccessAt: now,
		AccessCount:  0,
	}

	if ttl > 0 {
		entry.ExpiresAt = now.Add(ttl)
	}

	if _, exists := c.data[key]; !exists {
		c.stats.Size++
	}

	c.data[key] = entry
	c.stats.Sets++

	return nil
}

// Delete 删除缓存
func (c *MemoryCache) Delete(ctx context.Context, key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, ok := c.data[key]; ok {
		delete(c.data, key)
		c.stats.Deletes++
		c.stats.Size--
	}

	return nil
}

// Exists 检查键是否存在
func (c *MemoryCache) Exists(ctx context.Context, key string) (bool, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.data[key]
	if !ok {
		return false, nil
	}

	if entry.IsExpired() {
		return false, nil
	}

	return true, nil
}

// Clear 清空所有缓存
func (c *MemoryCache) Clear(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.data = make(map[string]*CacheEntry)
	c.stats.Size = 0

	return nil
}

// Stats 获取统计信息
func (c *MemoryCache) Stats() *Stats {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.stats
}

// MGet 批量获取
func (c *MemoryCache) MGet(ctx context.Context, keys []string) (map[string][]byte, error) {
	result := make(map[string][]byte)
	for _, key := range keys {
		if value, err := c.Get(ctx, key); err == nil {
			result[key] = value
		}
	}
	return result, nil
}

// MSet 批量设置
func (c *MemoryCache) MSet(ctx context.Context, items map[string][]byte, ttl time.Duration) error {
	for key, value := range items {
		if err := c.Set(ctx, key, value, ttl); err != nil {
			return err
		}
	}
	return nil
}

// MDelete 批量删除
func (c *MemoryCache) MDelete(ctx context.Context, keys []string) error {
	for _, key := range keys {
		c.Delete(ctx, key)
	}
	return nil
}

// Expire 设置过期时间
func (c *MemoryCache) Expire(ctx context.Context, key string, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, ok := c.data[key]
	if !ok {
		return ErrCacheNotFound
	}

	entry.ExpiresAt = time.Now().Add(ttl)
	return nil
}

// TTL 获取剩余过期时间
func (c *MemoryCache) TTL(ctx context.Context, key string) (time.Duration, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.data[key]
	if !ok {
		return 0, ErrCacheNotFound
	}

	if entry.ExpiresAt.IsZero() {
		return 0, nil // 永不过期
	}

	remaining := time.Until(entry.ExpiresAt)
	if remaining < 0 {
		return 0, ErrCacheExpired
	}

	return remaining, nil
}

// Keys 获取匹配的键列表
func (c *MemoryCache) Keys(ctx context.Context, pattern string) ([]string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// 简单实现：返回所有键（内存缓存不支持复杂模式匹配）
	keys := make([]string, 0, len(c.data))
	for key := range c.data {
		keys = append(keys, key)
	}

	return keys, nil
}

// Close 关闭缓存
func (c *MemoryCache) Close() error {
	if c.ttlTicker != nil {
		c.ttlTicker.Stop()
	}
	close(c.stopCh)
	return nil
}

// evict 驱逐一个条目
func (c *MemoryCache) evict() {
	switch c.eviction {
	case EvictionPolicyLRU:
		c.evictLRU()
	case EvictionPolicyLFU:
		c.evictLFU()
	case EvictionPolicyFIFO:
		c.evictFIFO()
	case EvictionPolicyTTL:
		c.evictTTL()
	default:
		c.evictLRU()
	}
}

// evictLRU 驱逐最近最少使用的条目
func (c *MemoryCache) evictLRU() {
	var oldestKey string
	var oldestTime time.Time

	for key, entry := range c.data {
		if oldestTime.IsZero() || entry.LastAccessAt.Before(oldestTime) {
			oldestTime = entry.LastAccessAt
			oldestKey = key
		}
	}

	if oldestKey != "" {
		delete(c.data, oldestKey)
		c.stats.Evictions++
		c.stats.Size--
	}
}

// evictLFU 驱逐最不经常使用的条目
func (c *MemoryCache) evictLFU() {
	var leastKey string
	var leastCount int64 = -1

	for key, entry := range c.data {
		if leastCount == -1 || entry.AccessCount < leastCount {
			leastCount = entry.AccessCount
			leastKey = key
		}
	}

	if leastKey != "" {
		delete(c.data, leastKey)
		c.stats.Evictions++
		c.stats.Size--
	}
}

// evictFIFO 驱逐最早创建的条目
func (c *MemoryCache) evictFIFO() {
	var oldestKey string
	var oldestTime time.Time

	for key, entry := range c.data {
		if oldestTime.IsZero() || entry.CreatedAt.Before(oldestTime) {
			oldestTime = entry.CreatedAt
			oldestKey = key
		}
	}

	if oldestKey != "" {
		delete(c.data, oldestKey)
		c.stats.Evictions++
		c.stats.Size--
	}
}

// evictTTL 驱逐最先过期的条目
func (c *MemoryCache) evictTTL() {
	var earliestKey string
	var earliestTime time.Time

	for key, entry := range c.data {
		if !entry.ExpiresAt.IsZero() {
			if earliestTime.IsZero() || entry.ExpiresAt.Before(earliestTime) {
				earliestTime = entry.ExpiresAt
				earliestKey = key
			}
		}
	}

	if earliestKey != "" {
		delete(c.data, earliestKey)
		c.stats.Evictions++
		c.stats.Size--
	}
}

// cleanupExpired 清理过期条目
func (c *MemoryCache) cleanupExpired() {
	for {
		select {
		case <-c.stopCh:
			return
		case <-c.ttlTicker.C:
			c.mu.Lock()
			for key, entry := range c.data {
				if entry.IsExpired() {
					delete(c.data, key)
					c.stats.Size--
					c.stats.Evictions++
				}
			}
			c.mu.Unlock()
		}
	}
}
