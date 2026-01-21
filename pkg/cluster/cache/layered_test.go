package cache

import (
	"context"
	"testing"
	"time"
)

func TestLayeredCache_Get_LocalHit(t *testing.T) {
	local := NewMemoryCache(100)
	remote := NewMemoryCache(100) // 用内存缓存模拟远程缓存

	cache := NewLayeredCache(local, remote)

	ctx := context.Background()
	key := "test-key"
	value := []byte("test-value")

	// 先设置到本地
	local.Set(ctx, key, value, 1*time.Minute)

	// 从分层缓存获取，应该命中本地
	got, err := cache.Get(ctx, key)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if string(got) != string(value) {
		t.Errorf("Get() = %s, want %s", got, value)
	}

	// 验证统计
	stats := cache.Stats()
	if stats.Hits != 1 {
		t.Errorf("Stats.Hits = %d, want 1", stats.Hits)
	}
}

func TestLayeredCache_Get_RemoteHit(t *testing.T) {
	local := NewMemoryCache(100)
	remote := NewMemoryCache(100)

	cache := NewLayeredCache(local, remote)

	ctx := context.Background()
	key := "test-key"
	value := []byte("test-value")

	// 只设置到远程
	remote.Set(ctx, key, value, 1*time.Minute)

	// 从分层缓存获取，应该从远程获取并回写本地
	got, err := cache.Get(ctx, key)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if string(got) != string(value) {
		t.Errorf("Get() = %s, want %s", got, value)
	}

	// 验证本地缓存已回写
	localValue, err := local.Get(ctx, key)
	if err != nil {
		t.Error("Local cache should have the value after remote hit")
	}
	if string(localValue) != string(value) {
		t.Errorf("Local cache value = %s, want %s", localValue, value)
	}
}

func TestLayeredCache_Set_WriteThrough(t *testing.T) {
	local := NewMemoryCache(100)
	remote := NewMemoryCache(100)

	config := DefaultLayeredCacheConfig()
	config.WriteThrough = true
	cache := NewLayeredCacheWithConfig(local, remote, config)

	ctx := context.Background()
	key := "test-key"
	value := []byte("test-value")

	// 写入分层缓存
	err := cache.Set(ctx, key, value, 1*time.Minute)
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	// 验证本地和远程都有
	localValue, err := local.Get(ctx, key)
	if err != nil {
		t.Error("Local cache should have the value")
	}
	if string(localValue) != string(value) {
		t.Errorf("Local cache value = %s, want %s", localValue, value)
	}

	remoteValue, err := remote.Get(ctx, key)
	if err != nil {
		t.Error("Remote cache should have the value")
	}
	if string(remoteValue) != string(value) {
		t.Errorf("Remote cache value = %s, want %s", remoteValue, value)
	}
}

func TestLayeredCache_Set_WriteBack(t *testing.T) {
	local := NewMemoryCache(100)
	remote := NewMemoryCache(100)

	config := DefaultLayeredCacheConfig()
	config.WriteThrough = false
	config.WriteBack = true
	cache := NewLayeredCacheWithConfig(local, remote, config)

	ctx := context.Background()
	key := "test-key"
	value := []byte("test-value")

	// 写入分层缓存
	err := cache.Set(ctx, key, value, 1*time.Minute)
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	// 验证本地立即有值
	localValue, err := local.Get(ctx, key)
	if err != nil {
		t.Error("Local cache should have the value immediately")
	}
	if string(localValue) != string(value) {
		t.Errorf("Local cache value = %s, want %s", localValue, value)
	}

	// 等待异步写入远程
	time.Sleep(100 * time.Millisecond)

	// 验证远程也有值
	remoteValue, err := remote.Get(ctx, key)
	if err != nil {
		t.Error("Remote cache should have the value after async write")
	}
	if string(remoteValue) != string(value) {
		t.Errorf("Remote cache value = %s, want %s", remoteValue, value)
	}
}

func TestLayeredCache_Delete(t *testing.T) {
	local := NewMemoryCache(100)
	remote := NewMemoryCache(100)

	cache := NewLayeredCache(local, remote)

	ctx := context.Background()
	key := "test-key"
	value := []byte("test-value")

	// 设置到两层
	local.Set(ctx, key, value, 1*time.Minute)
	remote.Set(ctx, key, value, 1*time.Minute)

	// 删除
	err := cache.Delete(ctx, key)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// 验证两层都被删除
	_, err = local.Get(ctx, key)
	if err != ErrCacheNotFound {
		t.Error("Local cache should not have the key after delete")
	}

	_, err = remote.Get(ctx, key)
	if err != ErrCacheNotFound {
		t.Error("Remote cache should not have the key after delete")
	}
}

func TestLayeredCache_MGet(t *testing.T) {
	local := NewMemoryCache(100)
	remote := NewMemoryCache(100)

	cache := NewLayeredCache(local, remote)

	ctx := context.Background()

	// 设置一些值到本地和远程
	local.Set(ctx, "key1", []byte("value1"), 1*time.Minute)
	remote.Set(ctx, "key2", []byte("value2"), 1*time.Minute)
	remote.Set(ctx, "key3", []byte("value3"), 1*time.Minute)

	// 批量获取
	keys := []string{"key1", "key2", "key3"}
	result, err := cache.MGet(ctx, keys)
	if err != nil {
		t.Fatalf("MGet() error = %v", err)
	}

	// 验证结果
	if len(result) != 3 {
		t.Errorf("MGet() returned %d items, want 3", len(result))
	}

	if string(result["key1"]) != "value1" {
		t.Errorf("result[key1] = %s, want value1", result["key1"])
	}
	if string(result["key2"]) != "value2" {
		t.Errorf("result[key2] = %s, want value2", result["key2"])
	}
	if string(result["key3"]) != "value3" {
		t.Errorf("result[key3] = %s, want value3", result["key3"])
	}

	// 验证 key2 和 key3 已回写到本地
	localValue2, _ := local.Get(ctx, "key2")
	if string(localValue2) != "value2" {
		t.Error("key2 should be written back to local cache")
	}

	localValue3, _ := local.Get(ctx, "key3")
	if string(localValue3) != "value3" {
		t.Error("key3 should be written back to local cache")
	}
}

func TestLayeredCache_MSet(t *testing.T) {
	local := NewMemoryCache(100)
	remote := NewMemoryCache(100)

	config := DefaultLayeredCacheConfig()
	config.WriteThrough = true
	cache := NewLayeredCacheWithConfig(local, remote, config)

	ctx := context.Background()

	items := map[string][]byte{
		"key1": []byte("value1"),
		"key2": []byte("value2"),
		"key3": []byte("value3"),
	}

	// 批量设置
	err := cache.MSet(ctx, items, 1*time.Minute)
	if err != nil {
		t.Fatalf("MSet() error = %v", err)
	}

	// 验证所有键都在本地和远程
	for key, expectedValue := range items {
		localValue, err := local.Get(ctx, key)
		if err != nil {
			t.Errorf("Local cache should have key %s", key)
		}
		if string(localValue) != string(expectedValue) {
			t.Errorf("Local cache[%s] = %s, want %s", key, localValue, expectedValue)
		}

		remoteValue, err := remote.Get(ctx, key)
		if err != nil {
			t.Errorf("Remote cache should have key %s", key)
		}
		if string(remoteValue) != string(expectedValue) {
			t.Errorf("Remote cache[%s] = %s, want %s", key, remoteValue, expectedValue)
		}
	}
}

func TestLayeredCache_InvalidateLocal(t *testing.T) {
	local := NewMemoryCache(100)
	remote := NewMemoryCache(100)

	cache := NewLayeredCache(local, remote)

	ctx := context.Background()
	key := "test-key"
	value := []byte("test-value")

	// 设置到两层
	cache.Set(ctx, key, value, 1*time.Minute)

	// 失效本地缓存
	err := cache.InvalidateLocal(ctx, key)
	if err != nil {
		t.Fatalf("InvalidateLocal() error = %v", err)
	}

	// 验证本地已失效
	_, err = local.Get(ctx, key)
	if err != ErrCacheNotFound {
		t.Error("Local cache should not have the key after invalidation")
	}

	// 验证远程还有
	remoteValue, err := remote.Get(ctx, key)
	if err != nil {
		t.Error("Remote cache should still have the value")
	}
	if string(remoteValue) != string(value) {
		t.Errorf("Remote cache value = %s, want %s", remoteValue, value)
	}
}
