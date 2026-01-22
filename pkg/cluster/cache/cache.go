package cache

import (
	"context"
	"errors"
	"time"
)

var (
	// ErrCacheNotFound 缓存未找到
	ErrCacheNotFound = errors.New("cache: key not found")

	// ErrCacheExpired 缓存已过期
	ErrCacheExpired = errors.New("cache: key expired")

	// ErrInvalidTTL 无效的 TTL
	ErrInvalidTTL = errors.New("cache: invalid TTL")

	// ErrCacheFull 缓存已满
	ErrCacheFull = errors.New("cache: cache is full")
)

// Cache 缓存接口
//
// Cache 定义了基础的缓存操作。
type Cache interface {
	// Get 获取缓存
	Get(ctx context.Context, key string) ([]byte, error)

	// Set 设置缓存
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error

	// Delete 删除缓存
	Delete(ctx context.Context, key string) error

	// Exists 检查键是否存在
	Exists(ctx context.Context, key string) (bool, error)

	// Clear 清空所有缓存
	Clear(ctx context.Context) error

	// Stats 获取统计信息
	Stats() *Stats
}

// DistributedCache 分布式缓存接口
//
// DistributedCache 扩展了 Cache 接口，添加了分布式特性。
type DistributedCache interface {
	Cache

	// MGet 批量获取
	MGet(ctx context.Context, keys []string) (map[string][]byte, error)

	// MSet 批量设置
	MSet(ctx context.Context, items map[string][]byte, ttl time.Duration) error

	// MDelete 批量删除
	MDelete(ctx context.Context, keys []string) error

	// Expire 设置过期时间
	Expire(ctx context.Context, key string, ttl time.Duration) error

	// TTL 获取剩余过期时间
	TTL(ctx context.Context, key string) (time.Duration, error)

	// Keys 获取匹配的键列表
	Keys(ctx context.Context, pattern string) ([]string, error)

	// Close 关闭缓存连接
	Close() error
}

// Stats 缓存统计信息
type Stats struct {
	// Hits 命中次数
	Hits int64

	// Misses 未命中次数
	Misses int64

	// Sets 设置次数
	Sets int64

	// Deletes 删除次数
	Deletes int64

	// Evictions 驱逐次数
	Evictions int64

	// Size 当前大小
	Size int64

	// MaxSize 最大大小
	MaxSize int64
}

// HitRate 计算命中率
func (s *Stats) HitRate() float64 {
	total := s.Hits + s.Misses
	if total == 0 {
		return 0
	}
	return float64(s.Hits) / float64(total)
}

// CacheEntry 缓存条目
type CacheEntry struct {
	// Key 键
	Key string

	// Value 值
	Value []byte

	// ExpiresAt 过期时间
	ExpiresAt time.Time

	// CreatedAt 创建时间
	CreatedAt time.Time

	// AccessCount 访问次数
	AccessCount int64

	// LastAccessAt 最后访问时间
	LastAccessAt time.Time
}

// IsExpired 检查是否过期
func (e *CacheEntry) IsExpired() bool {
	if e.ExpiresAt.IsZero() {
		return false
	}
	return time.Now().After(e.ExpiresAt)
}

// EvictionPolicy 驱逐策略
type EvictionPolicy string

const (
	// EvictionPolicyLRU 最近最少使用
	EvictionPolicyLRU EvictionPolicy = "lru"

	// EvictionPolicyLFU 最不经常使用
	EvictionPolicyLFU EvictionPolicy = "lfu"

	// EvictionPolicyFIFO 先进先出
	EvictionPolicyFIFO EvictionPolicy = "fifo"

	// EvictionPolicyTTL 按 TTL 驱逐
	EvictionPolicyTTL EvictionPolicy = "ttl"
)

// WarmupStrategy 预热策略
type WarmupStrategy interface {
	// ShouldWarmup 判断是否需要预热
	ShouldWarmup(key string) bool

	// GetWarmupKeys 获取需要预热的键列表
	GetWarmupKeys() []string

	// LoadData 加载预热数据
	LoadData(ctx context.Context, key string) ([]byte, error)
}

// InvalidationStrategy 失效策略
type InvalidationStrategy interface {
	// ShouldInvalidate 判断是否应该失效
	ShouldInvalidate(entry *CacheEntry) bool

	// OnInvalidate 失效时的回调
	OnInvalidate(key string, value []byte)
}
