package rbac

import (
	"sync"
	"time"
)

// PermissionCache 权限缓存
type PermissionCache struct {
	mu      sync.RWMutex
	entries map[string]*cacheEntry
	maxSize int
	ttl     time.Duration
}

type cacheEntry struct {
	key       string
	createdAt time.Time
}

// NewPermissionCache 创建权限缓存
func NewPermissionCache(maxSize int, ttl time.Duration) *PermissionCache {
	cache := &PermissionCache{
		entries: make(map[string]*cacheEntry),
		maxSize: maxSize,
		ttl:     ttl,
	}

	// 启动清理goroutine
	go cache.cleanupLoop()

	return cache
}

// cacheKey 生成缓存键
func (c *PermissionCache) cacheKey(req *PermissionRequest) string {
	return req.UserID + ":" + req.Resource + ":" + req.Action + ":" + req.ResourceID
}

// Has 检查缓存中是否存在
func (c *PermissionCache) Has(req *PermissionRequest) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	key := c.cacheKey(req)
	entry, exists := c.entries[key]
	if !exists {
		return false
	}

	// 检查是否过期
	if time.Since(entry.createdAt) > c.ttl {
		return false
	}

	return true
}

// Set 设置缓存
func (c *PermissionCache) Set(req *PermissionRequest) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 检查大小限制
	if len(c.entries) >= c.maxSize {
		// 清除最旧的条目
		c.evictOldest()
	}

	key := c.cacheKey(req)
	c.entries[key] = &cacheEntry{
		key:       key,
		createdAt: time.Now(),
	}
}

// InvalidateUser 清除用户的所有缓存
func (c *PermissionCache) InvalidateUser(userID string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for key := range c.entries {
		// 简单的前缀匹配
		if len(key) > len(userID) && key[:len(userID)] == userID {
			delete(c.entries, key)
		}
	}
}

// InvalidateAll 清除所有缓存
func (c *PermissionCache) InvalidateAll() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries = make(map[string]*cacheEntry)
}

// evictOldest 清除最旧的条目
func (c *PermissionCache) evictOldest() {
	var oldest *cacheEntry
	for _, entry := range c.entries {
		if oldest == nil || entry.createdAt.Before(oldest.createdAt) {
			oldest = entry
		}
	}

	if oldest != nil {
		delete(c.entries, oldest.key)
	}
}

// cleanupLoop 定期清理过期条目
func (c *PermissionCache) cleanupLoop() {
	ticker := time.NewTicker(c.ttl / 2)
	defer ticker.Stop()

	for range ticker.C {
		c.cleanup()
	}
}

// cleanup 清理过期条目
func (c *PermissionCache) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for key, entry := range c.entries {
		if now.Sub(entry.createdAt) > c.ttl {
			delete(c.entries, key)
		}
	}
}
