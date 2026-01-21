package cache

import (
	"context"
	"testing"
	"time"
)

func TestMemoryCache_SetGet(t *testing.T) {
	cache := NewMemoryCache(100)
	defer cache.Close()

	ctx := context.Background()
	key := "test-key"
	value := []byte("test-value")

	// 设置缓存
	err := cache.Set(ctx, key, value, 1*time.Minute)
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	// 获取缓存
	got, err := cache.Get(ctx, key)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if string(got) != string(value) {
		t.Errorf("Get() = %s, want %s", got, value)
	}

	// 验证统计
	stats := cache.Stats()
	if stats.Sets != 1 {
		t.Errorf("Stats.Sets = %d, want 1", stats.Sets)
	}
	if stats.Hits != 1 {
		t.Errorf("Stats.Hits = %d, want 1", stats.Hits)
	}
}

func TestMemoryCache_GetNotFound(t *testing.T) {
	cache := NewMemoryCache(100)
	defer cache.Close()

	ctx := context.Background()

	_, err := cache.Get(ctx, "non-existent")
	if err != ErrCacheNotFound {
		t.Errorf("Get() error = %v, want ErrCacheNotFound", err)
	}

	stats := cache.Stats()
	if stats.Misses != 1 {
		t.Errorf("Stats.Misses = %d, want 1", stats.Misses)
	}
}

func TestMemoryCache_Delete(t *testing.T) {
	cache := NewMemoryCache(100)
	defer cache.Close()

	ctx := context.Background()
	key := "test-key"
	value := []byte("test-value")

	cache.Set(ctx, key, value, 1*time.Minute)

	// 删除缓存
	err := cache.Delete(ctx, key)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// 验证已删除
	_, err = cache.Get(ctx, key)
	if err != ErrCacheNotFound {
		t.Errorf("Get() after Delete() error = %v, want ErrCacheNotFound", err)
	}

	stats := cache.Stats()
	if stats.Deletes != 1 {
		t.Errorf("Stats.Deletes = %d, want 1", stats.Deletes)
	}
}

func TestMemoryCache_Exists(t *testing.T) {
	cache := NewMemoryCache(100)
	defer cache.Close()

	ctx := context.Background()
	key := "test-key"
	value := []byte("test-value")

	// 键不存在
	exists, _ := cache.Exists(ctx, key)
	if exists {
		t.Error("Exists() = true, want false for non-existent key")
	}

	// 设置键
	cache.Set(ctx, key, value, 1*time.Minute)

	// 键存在
	exists, _ = cache.Exists(ctx, key)
	if !exists {
		t.Error("Exists() = false, want true for existing key")
	}
}

func TestMemoryCache_TTL(t *testing.T) {
	cache := NewMemoryCache(100)
	defer cache.Close()

	ctx := context.Background()
	key := "test-key"
	value := []byte("test-value")

	// 设置短 TTL
	cache.Set(ctx, key, value, 50*time.Millisecond)

	// 立即获取应该成功
	_, err := cache.Get(ctx, key)
	if err != nil {
		t.Errorf("Get() error = %v, want nil", err)
	}

	// 等待过期
	time.Sleep(100 * time.Millisecond)

	// 过期后获取应该失败
	_, err = cache.Get(ctx, key)
	if err != ErrCacheExpired {
		t.Errorf("Get() after TTL error = %v, want ErrCacheExpired", err)
	}
}

func TestMemoryCache_Eviction_LRU(t *testing.T) {
	// 创建最大容量为 3 的缓存
	config := MemoryCacheConfig{
		MaxSize:         3,
		EvictionPolicy:  EvictionPolicyLRU,
		CleanupInterval: 0, // 禁用自动清理
	}
	cache := NewMemoryCacheWithConfig(config)

	ctx := context.Background()

	// 填满缓存
	cache.Set(ctx, "key1", []byte("value1"), 1*time.Minute)
	cache.Set(ctx, "key2", []byte("value2"), 1*time.Minute)
	cache.Set(ctx, "key3", []byte("value3"), 1*time.Minute)

	// 访问 key1，使其成为最近使用
	cache.Get(ctx, "key1")
	time.Sleep(10 * time.Millisecond)

	// 访问 key3
	cache.Get(ctx, "key3")
	time.Sleep(10 * time.Millisecond)

	// 添加新键，应该驱逐 key2（最久未使用）
	cache.Set(ctx, "key4", []byte("value4"), 1*time.Minute)

	// key2 应该被驱逐
	_, err := cache.Get(ctx, "key2")
	if err != ErrCacheNotFound {
		t.Errorf("Get(key2) error = %v, want ErrCacheNotFound", err)
	}

	// key1 应该还在
	_, err = cache.Get(ctx, "key1")
	if err != nil {
		t.Errorf("Get(key1) error = %v, want nil", err)
	}

	stats := cache.Stats()
	if stats.Evictions != 1 {
		t.Errorf("Stats.Evictions = %d, want 1", stats.Evictions)
	}
}

func TestMemoryCache_Clear(t *testing.T) {
	cache := NewMemoryCache(100)
	defer cache.Close()

	ctx := context.Background()

	// 设置多个键
	cache.Set(ctx, "key1", []byte("value1"), 1*time.Minute)
	cache.Set(ctx, "key2", []byte("value2"), 1*time.Minute)
	cache.Set(ctx, "key3", []byte("value3"), 1*time.Minute)

	// 清空缓存
	err := cache.Clear(ctx)
	if err != nil {
		t.Fatalf("Clear() error = %v", err)
	}

	// 验证所有键都被清除
	stats := cache.Stats()
	if stats.Size != 0 {
		t.Errorf("Stats.Size = %d, want 0 after Clear()", stats.Size)
	}

	// 尝试获取键，应该失败
	_, err = cache.Get(ctx, "key1")
	if err != ErrCacheNotFound {
		t.Errorf("Get() after Clear() error = %v, want ErrCacheNotFound", err)
	}
}

func TestMemoryCache_HitRate(t *testing.T) {
	cache := NewMemoryCache(100)
	defer cache.Close()

	ctx := context.Background()

	// 设置一个键
	cache.Set(ctx, "key1", []byte("value1"), 1*time.Minute)

	// 5 次命中
	for i := 0; i < 5; i++ {
		cache.Get(ctx, "key1")
	}

	// 5 次未命中
	for i := 0; i < 5; i++ {
		cache.Get(ctx, "non-existent")
	}

	stats := cache.Stats()
	hitRate := stats.HitRate()

	if hitRate != 0.5 {
		t.Errorf("HitRate() = %.2f, want 0.50", hitRate)
	}
}

func TestMemoryCache_Concurrent(t *testing.T) {
	cache := NewMemoryCache(1000)
	defer cache.Close()

	ctx := context.Background()

	// 并发写入
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				key := string(rune('a' + id))
				cache.Set(ctx, key, []byte("value"), 1*time.Minute)
				cache.Get(ctx, key)
			}
			done <- true
		}(i)
	}

	// 等待所有 goroutine 完成
	for i := 0; i < 10; i++ {
		<-done
	}

	stats := cache.Stats()
	t.Logf("Concurrent test stats: Sets=%d, Hits=%d, Misses=%d",
		stats.Sets, stats.Hits, stats.Misses)
}
