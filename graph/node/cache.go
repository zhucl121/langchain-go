package node

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// NodeCacheStorage 节点缓存存储后端接口。
type NodeCacheStorage interface {
	Get(ctx context.Context, key string) (any, bool, error)
	Set(ctx context.Context, key string, value any, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	Stats() NodeCacheStats
}

// NodeCacheStats 节点缓存统计。
type NodeCacheStats struct {
	Hits      int64
	Misses    int64
	Evictions int64
	Size      int
	HitRate   float64
}

// NodeCacheConfig 节点级缓存配置。
//
// 对标 LangGraph 1.0 Node-Level Caching，允许对每个节点独立配置缓存策略。
//
// 示例：
//
//	config := NodeCacheConfig{
//	    Enabled: true,
//	    TTL:     5 * time.Minute,
//	    KeyFunc: func(state any) (string, error) {
//	        s := state.(MyState)
//	        return fmt.Sprintf("query:%s", s.Query), nil
//	    },
//	}
type NodeCacheConfig struct {
	// Enabled 是否启用节点缓存
	Enabled bool

	// TTL 缓存过期时间（0 表示永不过期）
	TTL time.Duration

	// KeyFunc 自定义缓存 key 生成函数（nil 时使用默认 SHA256 哈希）
	KeyFunc func(state any) (string, error)

	// Storage 缓存存储后端（nil 时使用内存缓存）
	Storage NodeCacheStorage

	// MaxEntries 最大缓存条目数（0 表示无限制，仅内存存储有效）
	MaxEntries int
}

// nodeCacheEntry 内部缓存条目。
type nodeCacheEntry struct {
	Value      any
	CreatedAt  time.Time
	ExpiresAt  time.Time
	AccessedAt time.Time
}

// isExpired 判断是否过期。
func (e *nodeCacheEntry) isExpired() bool {
	return !e.ExpiresAt.IsZero() && time.Now().After(e.ExpiresAt)
}

// InMemoryNodeCache 基于内存的节点缓存。
type InMemoryNodeCache struct {
	mu         sync.RWMutex
	entries    map[string]*nodeCacheEntry
	maxEntries int
	hits       atomic.Int64
	misses     atomic.Int64
	evictions  atomic.Int64
}

// NewInMemoryNodeCache 创建内存节点缓存。
func NewInMemoryNodeCache(maxEntries int) *InMemoryNodeCache {
	c := &InMemoryNodeCache{
		entries:    make(map[string]*nodeCacheEntry),
		maxEntries: maxEntries,
	}
	go c.cleanupLoop()
	return c
}

// Get 实现 NodeCacheStorage 接口。
func (c *InMemoryNodeCache) Get(ctx context.Context, key string) (any, bool, error) {
	c.mu.RLock()
	entry, ok := c.entries[key]
	c.mu.RUnlock()

	if !ok {
		c.misses.Add(1)
		return nil, false, nil
	}

	if entry.isExpired() {
		c.mu.Lock()
		delete(c.entries, key)
		c.mu.Unlock()
		c.misses.Add(1)
		return nil, false, nil
	}

	c.mu.Lock()
	entry.AccessedAt = time.Now()
	c.mu.Unlock()

	c.hits.Add(1)
	return entry.Value, true, nil
}

// Set 实现 NodeCacheStorage 接口。
func (c *InMemoryNodeCache) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.maxEntries > 0 && len(c.entries) >= c.maxEntries {
		c.evictOldest()
	}

	entry := &nodeCacheEntry{
		Value:      value,
		CreatedAt:  time.Now(),
		AccessedAt: time.Now(),
	}
	if ttl > 0 {
		entry.ExpiresAt = time.Now().Add(ttl)
	}

	c.entries[key] = entry
	return nil
}

// Delete 实现 NodeCacheStorage 接口。
func (c *InMemoryNodeCache) Delete(ctx context.Context, key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, key)
	return nil
}

// Stats 实现 NodeCacheStorage 接口。
func (c *InMemoryNodeCache) Stats() NodeCacheStats {
	c.mu.RLock()
	size := len(c.entries)
	c.mu.RUnlock()

	hits := c.hits.Load()
	misses := c.misses.Load()
	total := hits + misses

	var hitRate float64
	if total > 0 {
		hitRate = float64(hits) / float64(total)
	}

	return NodeCacheStats{
		Hits:      hits,
		Misses:    misses,
		Evictions: c.evictions.Load(),
		Size:      size,
		HitRate:   hitRate,
	}
}

// evictOldest 驱逐访问时间最旧的条目（需持有写锁）。
func (c *InMemoryNodeCache) evictOldest() {
	var oldestKey string
	var oldestTime time.Time

	for k, v := range c.entries {
		if oldestTime.IsZero() || v.AccessedAt.Before(oldestTime) {
			oldestTime = v.AccessedAt
			oldestKey = k
		}
	}

	if oldestKey != "" {
		delete(c.entries, oldestKey)
		c.evictions.Add(1)
	}
}

// cleanupLoop 定期清理过期条目。
func (c *InMemoryNodeCache) cleanupLoop() {
	ticker := time.NewTicker(2 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for k, v := range c.entries {
			if !v.ExpiresAt.IsZero() && now.After(v.ExpiresAt) {
				delete(c.entries, k)
			}
		}
		c.mu.Unlock()
	}
}

// defaultKeyFunc 默认缓存 key 生成函数：对状态 JSON 做 SHA256 哈希。
func defaultKeyFunc(state any) (string, error) {
	data, err := json.Marshal(state)
	if err != nil {
		return "", fmt.Errorf("node cache: failed to marshal state for key generation: %w", err)
	}
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:]), nil
}

// CachedFunctionNode 带缓存能力的函数节点。
//
// CachedFunctionNode 在执行节点函数前先查询缓存，命中时直接返回缓存结果，
// 未命中时执行函数并将结果写入缓存，显著减少重复计算开销。
//
// 对标 LangGraph 1.0 Node-Level Caching 特性。
//
// 示例：
//
//	node := NewCachedFunctionNode("llm-call", expensiveFunc,
//	    NodeCacheConfig{
//	        Enabled:    true,
//	        TTL:        5 * time.Minute,
//	        MaxEntries: 500,
//	    },
//	)
type CachedFunctionNode[S any] struct {
	inner   *FunctionNode[S]
	config  NodeCacheConfig
	storage NodeCacheStorage
	keyFunc func(state any) (string, error)
}

// NewCachedFunctionNode 创建带节点级缓存的函数节点。
//
// 参数：
//   - name: 节点名称
//   - fn: 节点函数
//   - config: 缓存配置
//   - opts: 节点选项
func NewCachedFunctionNode[S any](name string, fn NodeFunc[S], config NodeCacheConfig, opts ...NodeOption) *CachedFunctionNode[S] {
	storage := config.Storage
	if storage == nil {
		maxEntries := config.MaxEntries
		if maxEntries <= 0 {
			maxEntries = 1000
		}
		storage = NewInMemoryNodeCache(maxEntries)
	}

	keyFunc := config.KeyFunc
	if keyFunc == nil {
		keyFunc = defaultKeyFunc
	}

	return &CachedFunctionNode[S]{
		inner:   NewFunctionNode(name, fn, opts...),
		config:  config,
		storage: storage,
		keyFunc: keyFunc,
	}
}

// GetName 实现 Node 接口。
func (n *CachedFunctionNode[S]) GetName() string {
	return n.inner.GetName()
}

// GetDescription 实现 Node 接口。
func (n *CachedFunctionNode[S]) GetDescription() string {
	return n.inner.GetDescription()
}

// GetTags 实现 Node 接口。
func (n *CachedFunctionNode[S]) GetTags() []string {
	return n.inner.GetTags()
}

// Validate 实现 Node 接口。
func (n *CachedFunctionNode[S]) Validate() error {
	return n.inner.Validate()
}

// Invoke 实现 Node 接口。
//
// 执行顺序：
//  1. 如缓存未启用，直接执行节点函数
//  2. 生成缓存 key
//  3. 查询缓存，命中时直接返回
//  4. 执行节点函数
//  5. 将结果写入缓存
func (n *CachedFunctionNode[S]) Invoke(ctx context.Context, state S) (S, error) {
	select {
	case <-ctx.Done():
		return state, ctx.Err()
	default:
	}

	if !n.config.Enabled {
		return n.inner.Invoke(ctx, state)
	}

	// 生成缓存 key
	key, err := n.keyFunc(state)
	if err != nil {
		// key 生成失败时退化为直接执行（不缓存）
		return n.inner.Invoke(ctx, state)
	}

	cacheKey := fmt.Sprintf("node:%s:%s", n.GetName(), key)

	// 查询缓存
	cached, found, err := n.storage.Get(ctx, cacheKey)
	if err == nil && found {
		if result, ok := cached.(S); ok {
			return result, nil
		}
	}

	// 执行节点函数
	result, err := n.inner.Invoke(ctx, state)
	if err != nil {
		return result, err
	}

	// 写入缓存（忽略写入错误，不影响正常流程）
	_ = n.storage.Set(ctx, cacheKey, result, n.config.TTL)

	return result, nil
}

// Invalidate 手动使指定状态对应的缓存失效。
func (n *CachedFunctionNode[S]) Invalidate(ctx context.Context, state S) error {
	key, err := n.keyFunc(state)
	if err != nil {
		return fmt.Errorf("node cache invalidate: %w", err)
	}
	cacheKey := fmt.Sprintf("node:%s:%s", n.GetName(), key)
	return n.storage.Delete(ctx, cacheKey)
}

// CacheStats 返回节点缓存统计信息。
func (n *CachedFunctionNode[S]) CacheStats() NodeCacheStats {
	return n.storage.Stats()
}

// WithNodeCache 为 FunctionNode 添加节点级缓存，返回 CachedFunctionNode。
//
// 用于链式调用风格的节点构建。
//
// 示例：
//
//	node := NewFunctionNode("agent", agentFunc).WithNodeCache(NodeCacheConfig{
//	    Enabled: true,
//	    TTL:     10 * time.Minute,
//	})
func (n *FunctionNode[S]) WithNodeCache(config NodeCacheConfig) *CachedFunctionNode[S] {
	return NewCachedFunctionNode(n.metadata.Name, n.fn, config,
		WithDescription(n.metadata.Description),
		WithTags(n.metadata.Tags...),
	)
}
