package cache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// Cache 缓存接口。
//
// 用于缓存 LLM 响应和工具结果。
type Cache interface {
	// Get 获取缓存
	Get(ctx context.Context, key string) (any, bool, error)

	// Set 设置缓存
	Set(ctx context.Context, key string, value any, ttl time.Duration) error

	// Delete 删除缓存
	Delete(ctx context.Context, key string) error

	// Clear 清空所有缓存
	Clear(ctx context.Context) error

	// Stats 获取缓存统计
	Stats() *CacheStats
}

// CacheStats 缓存统计。
type CacheStats struct {
	// Hits 命中次数
	Hits int64

	// Misses 未命中次数
	Misses int64

	// Size 当前缓存大小
	Size int

	// HitRate 命中率
	HitRate float64
}

// MemoryCache 内存缓存。
//
// 使用 sync.Map 实现的线程安全内存缓存。
type MemoryCache struct {
	data    sync.Map
	ttls    sync.Map
	hits    int64
	misses  int64
	maxSize int
	size    int
	mu      sync.RWMutex
}

// cacheEntry 缓存条目。
type cacheEntry struct {
	Value      any
	ExpireAt   time.Time
	CreateTime time.Time
}

// NewMemoryCache 创建内存缓存。
//
// 参数：
//   - maxSize: 最大缓存条目数 (0 表示无限制)
//
// 返回：
//   - *MemoryCache: 内存缓存实例
//
func NewMemoryCache(maxSize int) *MemoryCache {
	cache := &MemoryCache{
		maxSize: maxSize,
	}

	// 启动过期清理 goroutine
	go cache.cleanupExpired()

	return cache
}

// Get 实现 Cache 接口。
func (m *MemoryCache) Get(ctx context.Context, key string) (any, bool, error) {
	value, ok := m.data.Load(key)
	if !ok {
		m.mu.Lock()
		m.misses++
		m.mu.Unlock()
		return nil, false, nil
	}

	entry := value.(*cacheEntry)

	// 检查是否过期
	if !entry.ExpireAt.IsZero() && time.Now().After(entry.ExpireAt) {
		m.data.Delete(key)
		m.mu.Lock()
		m.size--
		m.misses++
		m.mu.Unlock()
		return nil, false, nil
	}

	m.mu.Lock()
	m.hits++
	m.mu.Unlock()

	return entry.Value, true, nil
}

// Set 实现 Cache 接口。
func (m *MemoryCache) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 检查是否超过最大大小
	if m.maxSize > 0 && m.size >= m.maxSize {
		// 简单的 LRU: 删除最旧的条目
		m.evictOldest()
	}

	entry := &cacheEntry{
		Value:      value,
		CreateTime: time.Now(),
	}

	if ttl > 0 {
		entry.ExpireAt = time.Now().Add(ttl)
	}

	m.data.Store(key, entry)
	m.size++

	return nil
}

// Delete 实现 Cache 接口。
func (m *MemoryCache) Delete(ctx context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.data.Load(key); ok {
		m.data.Delete(key)
		m.size--
	}

	return nil
}

// Clear 实现 Cache 接口。
func (m *MemoryCache) Clear(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.data = sync.Map{}
	m.size = 0
	m.hits = 0
	m.misses = 0

	return nil
}

// Stats 实现 Cache 接口。
func (m *MemoryCache) Stats() *CacheStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := &CacheStats{
		Hits:   m.hits,
		Misses: m.misses,
		Size:   m.size,
	}

	total := m.hits + m.misses
	if total > 0 {
		stats.HitRate = float64(m.hits) / float64(total)
	}

	return stats
}

// evictOldest 驱逐最旧的条目。
func (m *MemoryCache) evictOldest() {
	var oldestKey string
	var oldestTime time.Time

	m.data.Range(func(key, value any) bool {
		entry := value.(*cacheEntry)
		if oldestTime.IsZero() || entry.CreateTime.Before(oldestTime) {
			oldestTime = entry.CreateTime
			oldestKey = key.(string)
		}
		return true
	})

	if oldestKey != "" {
		m.data.Delete(oldestKey)
		m.size--
	}
}

// cleanupExpired 清理过期条目。
func (m *MemoryCache) cleanupExpired() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		m.data.Range(func(key, value any) bool {
			entry := value.(*cacheEntry)
			if !entry.ExpireAt.IsZero() && now.After(entry.ExpireAt) {
				m.Delete(context.Background(), key.(string))
			}
			return true
		})
	}
}

// GenerateCacheKey 生成缓存键。
//
// 参数：
//   - prefix: 前缀
//   - parts: 键的各个部分
//
// 返回：
//   - string: 缓存键
//
func GenerateCacheKey(prefix string, parts ...any) string {
	// 序列化所有部分
	data, err := json.Marshal(parts)
	if err != nil {
		// 如果序列化失败，使用简单的字符串拼接
		return fmt.Sprintf("%s:%v", prefix, parts)
	}

	// 使用 SHA256 生成哈希
	hash := sha256.Sum256(data)
	hashStr := hex.EncodeToString(hash[:])

	return fmt.Sprintf("%s:%s", prefix, hashStr[:16])
}

// CacheConfig 缓存配置。
type CacheConfig struct {
	// Enabled 是否启用缓存
	Enabled bool

	// TTL 缓存过期时间 (0 表示永不过期)
	TTL time.Duration

	// MaxSize 最大缓存条目数 (0 表示无限制)
	MaxSize int

	// Backend 缓存后端 (默认使用内存缓存)
	Backend Cache
}

// DefaultCacheConfig 返回默认缓存配置。
func DefaultCacheConfig() CacheConfig {
	return CacheConfig{
		Enabled: true,
		TTL:     1 * time.Hour,
		MaxSize: 1000,
		Backend: NewMemoryCache(1000),
	}
}

// CachedValue 缓存值包装器。
type CachedValue struct {
	Value      any
	CachedAt   time.Time
	FromCache  bool
	CacheKey   string
}

// LLMCache LLM 响应缓存。
type LLMCache struct {
	cache  Cache
	config CacheConfig
}

// NewLLMCache 创建 LLM 缓存。
//
// 参数：
//   - config: 缓存配置
//
// 返回：
//   - *LLMCache: LLM 缓存实例
//
func NewLLMCache(config CacheConfig) *LLMCache {
	if config.Backend == nil {
		config.Backend = NewMemoryCache(config.MaxSize)
	}

	return &LLMCache{
		cache:  config.Backend,
		config: config,
	}
}

// Get 获取缓存的 LLM 响应。
//
// 参数：
//   - ctx: 上下文
//   - prompt: 提示词
//   - model: 模型名称
//
// 返回：
//   - string: 响应内容
//   - bool: 是否命中缓存
//   - error: 错误
//
func (lc *LLMCache) Get(ctx context.Context, prompt, model string) (string, bool, error) {
	if !lc.config.Enabled {
		return "", false, nil
	}

	key := GenerateCacheKey("llm", model, prompt)
	value, found, err := lc.cache.Get(ctx, key)
	if err != nil || !found {
		return "", false, err
	}

	return value.(string), true, nil
}

// Set 缓存 LLM 响应。
//
// 参数：
//   - ctx: 上下文
//   - prompt: 提示词
//   - model: 模型名称
//   - response: 响应内容
//
// 返回：
//   - error: 错误
//
func (lc *LLMCache) Set(ctx context.Context, prompt, model, response string) error {
	if !lc.config.Enabled {
		return nil
	}

	key := GenerateCacheKey("llm", model, prompt)
	return lc.cache.Set(ctx, key, response, lc.config.TTL)
}

// Stats 获取缓存统计。
func (lc *LLMCache) Stats() *CacheStats {
	return lc.cache.Stats()
}

// ToolCache 工具结果缓存。
type ToolCache struct {
	cache  Cache
	config CacheConfig
}

// NewToolCache 创建工具缓存。
//
// 参数：
//   - config: 缓存配置
//
// 返回：
//   - *ToolCache: 工具缓存实例
//
func NewToolCache(config CacheConfig) *ToolCache {
	if config.Backend == nil {
		config.Backend = NewMemoryCache(config.MaxSize)
	}

	return &ToolCache{
		cache:  config.Backend,
		config: config,
	}
}

// Get 获取缓存的工具结果。
//
// 参数：
//   - ctx: 上下文
//   - toolName: 工具名称
//   - args: 工具参数
//
// 返回：
//   - any: 工具结果
//   - bool: 是否命中缓存
//   - error: 错误
//
func (tc *ToolCache) Get(ctx context.Context, toolName string, args map[string]any) (any, bool, error) {
	if !tc.config.Enabled {
		return nil, false, nil
	}

	key := GenerateCacheKey("tool", toolName, args)
	return tc.cache.Get(ctx, key)
}

// Set 缓存工具结果。
//
// 参数：
//   - ctx: 上下文
//   - toolName: 工具名称
//   - args: 工具参数
//   - result: 工具结果
//
// 返回：
//   - error: 错误
//
func (tc *ToolCache) Set(ctx context.Context, toolName string, args map[string]any, result any) error {
	if !tc.config.Enabled {
		return nil
	}

	key := GenerateCacheKey("tool", toolName, args)
	return tc.cache.Set(ctx, key, result, tc.config.TTL)
}

// Stats 获取缓存统计。
func (tc *ToolCache) Stats() *CacheStats {
	return tc.cache.Stats()
}
