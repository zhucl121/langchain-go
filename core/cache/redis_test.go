package cache

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 注意：这些测试需要 Redis 服务器运行在 localhost:6379
// 可以使用 docker 运行：docker run -d -p 6379:6379 redis

func TestRedisCache(t *testing.T) {
	// 跳过测试如果没有 Redis 可用
	config := DefaultRedisCacheConfig()
	config.Password = "redis123" // 设置密码
	cache, err := NewRedisCache(config)
	if err != nil {
		t.Skip("Redis not available:", err)
		return
	}
	defer cache.Close()

	ctx := context.Background()

	// 清理
	err = cache.Clear(ctx)
	require.NoError(t, err)

	t.Run("Set and Get", func(t *testing.T) {
		key := "test-key"
		value := map[string]any{
			"message": "Hello Redis",
			"count":   42,
		}

		// Set
		err := cache.Set(ctx, key, value, time.Hour)
		assert.NoError(t, err)

		// Get
		got, found, err := cache.Get(ctx, key)
		assert.NoError(t, err)
		assert.True(t, found)
		
		// 比较值
		gotMap, ok := got.(map[string]any)
		assert.True(t, ok)
		assert.Equal(t, "Hello Redis", gotMap["message"])
		assert.Equal(t, float64(42), gotMap["count"]) // JSON 数字解析为 float64
	})

	t.Run("Get Non-Existent Key", func(t *testing.T) {
		_, found, err := cache.Get(ctx, "non-existent")
		assert.NoError(t, err)
		assert.False(t, found)
	})

	t.Run("Delete", func(t *testing.T) {
		key := "delete-test"
		err := cache.Set(ctx, key, "value", time.Hour)
		assert.NoError(t, err)

		err = cache.Delete(ctx, key)
		assert.NoError(t, err)

		_, found, err := cache.Get(ctx, key)
		assert.NoError(t, err)
		assert.False(t, found)
	})

	t.Run("TTL Expiry", func(t *testing.T) {
		key := "ttl-test"
		err := cache.Set(ctx, key, "value", 1*time.Second)
		assert.NoError(t, err)

		// 立即获取应该成功
		_, found, err := cache.Get(ctx, key)
		assert.NoError(t, err)
		assert.True(t, found)

		// 等待过期
		time.Sleep(2 * time.Second)

		// 应该不存在
		_, found, err = cache.Get(ctx, key)
		assert.NoError(t, err)
		assert.False(t, found)
	})

	t.Run("Stats", func(t *testing.T) {
		// 重置缓存
		err := cache.Clear(ctx)
		require.NoError(t, err)

		// 创建新的缓存实例以重置统计
		cache2, err := NewRedisCache(config)
		require.NoError(t, err)
		defer cache2.Close()

		// 设置一些值
		cache2.Set(ctx, "stats-1", "value1", time.Hour)
		cache2.Set(ctx, "stats-2", "value2", time.Hour)

		// Hit
		cache2.Get(ctx, "stats-1")
		cache2.Get(ctx, "stats-2")

		// Miss
		cache2.Get(ctx, "stats-non-existent")

		stats := cache2.Stats()
		assert.Equal(t, int64(2), stats.Hits)
		assert.Equal(t, int64(1), stats.Misses)
		assert.InDelta(t, 0.666, stats.HitRate, 0.01)
	})

	t.Run("Ping", func(t *testing.T) {
		err := cache.Ping(ctx)
		assert.NoError(t, err)
	})

	t.Run("Keys", func(t *testing.T) {
		// 清理
		cache.Clear(ctx)

		// 设置多个键
		cache.Set(ctx, "keys-test-1", "v1", time.Hour)
		cache.Set(ctx, "keys-test-2", "v2", time.Hour)
		cache.Set(ctx, "other-key", "v3", time.Hour)

		// 列出匹配的键
		keys, err := cache.Keys(ctx, "keys-test-*")
		assert.NoError(t, err)
		assert.Len(t, keys, 2)
	})

	t.Run("Exists", func(t *testing.T) {
		key := "exists-test"
		cache.Set(ctx, key, "value", time.Hour)

		exists, err := cache.Exists(ctx, key)
		assert.NoError(t, err)
		assert.True(t, exists)

		exists, err = cache.Exists(ctx, "non-existent")
		assert.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("TTL", func(t *testing.T) {
		key := "ttl-check"
		cache.Set(ctx, key, "value", 10*time.Second)

		ttl, err := cache.TTL(ctx, key)
		assert.NoError(t, err)
		assert.Greater(t, ttl, 5*time.Second)
		assert.LessOrEqual(t, ttl, 10*time.Second)
	})

	t.Run("SetNX", func(t *testing.T) {
		key := "setnx-test"

		// 第一次应该成功
		ok, err := cache.SetNX(ctx, key, "value1", time.Hour)
		assert.NoError(t, err)
		assert.True(t, ok)

		// 第二次应该失败（键已存在）
		ok, err = cache.SetNX(ctx, key, "value2", time.Hour)
		assert.NoError(t, err)
		assert.False(t, ok)

		// 值应该是第一次设置的
		val, found, err := cache.Get(ctx, key)
		assert.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, "value1", val)
	})

	t.Run("Increment", func(t *testing.T) {
		key := "incr-test"

		// 递增
		val, err := cache.Increment(ctx, key, 1)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), val)

		val, err = cache.Increment(ctx, key, 5)
		assert.NoError(t, err)
		assert.Equal(t, int64(6), val)
	})

	t.Run("Decrement", func(t *testing.T) {
		key := "decr-test"

		// 设置初始值
		cache.Increment(ctx, key, 10)

		// 递减
		val, err := cache.Decrement(ctx, key, 3)
		assert.NoError(t, err)
		assert.Equal(t, int64(7), val)
	})
}

func TestRedisCache_WithLLMCache(t *testing.T) {
	// 跳过测试如果没有 Redis 可用
	config := DefaultRedisCacheConfig()
	config.Password = "redis123" // 设置密码
	redisCache, err := NewRedisCache(config)
	if err != nil {
		t.Skip("Redis not available:", err)
		return
	}
	defer redisCache.Close()

	ctx := context.Background()
	redisCache.Clear(ctx)

	// 创建 LLM 缓存
	cacheConfig := CacheConfig{
		Enabled: true,
		TTL:     time.Hour,
		Backend: redisCache,
	}
	llmCache := NewLLMCache(cacheConfig)

	t.Run("LLM Cache Hit", func(t *testing.T) {
		prompt := "What is AI?"
		model := "gpt-3.5-turbo"
		response := "AI is Artificial Intelligence"

		// 第一次调用 - 应该是 miss
		cached, found, err := llmCache.Get(ctx, prompt, model)
		assert.NoError(t, err)
		assert.False(t, found)
		assert.Empty(t, cached)

		// 设置缓存
		err = llmCache.Set(ctx, prompt, model, response)
		assert.NoError(t, err)

		// 第二次调用 - 应该是 hit
		cached, found, err = llmCache.Get(ctx, prompt, model)
		assert.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, response, cached)
	})
}

func TestRedisCache_WithToolCache(t *testing.T) {
	// 跳过测试如果没有 Redis 可用
	redisConfig := DefaultRedisCacheConfig()
	redisConfig.Password = "redis123" // 设置密码
	redisCache, err := NewRedisCache(redisConfig)
	if err != nil {
		t.Skip("Redis not available:", err)
		return
	}
	defer redisCache.Close()

	ctx := context.Background()
	redisCache.Clear(ctx)

	// 创建工具缓存
	cacheConfig := CacheConfig{
		Enabled: true,
		TTL:     time.Hour,
		Backend: redisCache,
	}
	toolCache := NewToolCache(cacheConfig)

	t.Run("Tool Cache Hit", func(t *testing.T) {
		toolName := "calculator"
		input := map[string]any{"expression": "2 + 2"}
		result := "4"

		// 第一次调用 - 应该是 miss
		cached, found, err := toolCache.Get(ctx, toolName, input)
		assert.NoError(t, err)
		assert.False(t, found)
		assert.Empty(t, cached)

		// 设置缓存
		err = toolCache.Set(ctx, toolName, input, result)
		assert.NoError(t, err)

		// 第二次调用 - 应该是 hit
		cached, found, err = toolCache.Get(ctx, toolName, input)
		assert.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, result, cached)
	})
}

func BenchmarkRedisCache_Set(b *testing.B) {
	config := DefaultRedisCacheConfig()
	config.Password = "redis123" // 设置密码
	cache, err := NewRedisCache(config)
	if err != nil {
		b.Skip("Redis not available:", err)
		return
	}
	defer cache.Close()

	ctx := context.Background()
	value := map[string]any{"data": "test"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Set(ctx, "bench-key", value, time.Hour)
	}
}

func BenchmarkRedisCache_Get(b *testing.B) {
	config := DefaultRedisCacheConfig()
	config.Password = "redis123" // 设置密码
	cache, err := NewRedisCache(config)
	if err != nil {
		b.Skip("Redis not available:", err)
		return
	}
	defer cache.Close()

	ctx := context.Background()
	cache.Set(ctx, "bench-key", map[string]any{"data": "test"}, time.Hour)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Get(ctx, "bench-key")
	}
}

func BenchmarkRedisCache_SetGet(b *testing.B) {
	config := DefaultRedisCacheConfig()
	config.Password = "redis123" // 设置密码
	cache, err := NewRedisCache(config)
	if err != nil {
		b.Skip("Redis not available:", err)
		return
	}
	defer cache.Close()

	ctx := context.Background()
	value := map[string]any{"data": "test"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := "bench-key"
		cache.Set(ctx, key, value, time.Hour)
		cache.Get(ctx, key)
	}
}
