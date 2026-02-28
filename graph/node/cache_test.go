package node

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

type cacheTestState struct {
	Query  string
	Result string
}

func TestInMemoryNodeCache_GetSet(t *testing.T) {
	cache := NewInMemoryNodeCache(100)
	ctx := context.Background()

	err := cache.Set(ctx, "key1", "value1", time.Minute)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	val, found, err := cache.Get(ctx, "key1")
	if err != nil || !found {
		t.Fatalf("Get failed: err=%v found=%v", err, found)
	}
	if val != "value1" {
		t.Errorf("want value1, got %v", val)
	}
}

func TestInMemoryNodeCache_TTLExpiry(t *testing.T) {
	cache := NewInMemoryNodeCache(100)
	ctx := context.Background()

	_ = cache.Set(ctx, "expiry-key", "data", 50*time.Millisecond)

	// 应该命中
	_, found, _ := cache.Get(ctx, "expiry-key")
	if !found {
		t.Fatal("should hit cache before TTL")
	}

	time.Sleep(100 * time.Millisecond)

	// TTL 后应该 miss
	_, found, _ = cache.Get(ctx, "expiry-key")
	if found {
		t.Fatal("should miss cache after TTL")
	}
}

func TestInMemoryNodeCache_Eviction(t *testing.T) {
	cache := NewInMemoryNodeCache(3)
	ctx := context.Background()

	_ = cache.Set(ctx, "k1", "v1", 0)
	_ = cache.Set(ctx, "k2", "v2", 0)
	_ = cache.Set(ctx, "k3", "v3", 0)
	// 添加第 4 个触发驱逐
	_ = cache.Set(ctx, "k4", "v4", 0)

	stats := cache.Stats()
	if stats.Evictions == 0 {
		t.Error("expected at least one eviction")
	}
	if stats.Size > 3 {
		t.Errorf("cache size should not exceed maxEntries, got %d", stats.Size)
	}
}

func TestInMemoryNodeCache_Stats(t *testing.T) {
	cache := NewInMemoryNodeCache(100)
	ctx := context.Background()

	_ = cache.Set(ctx, "s1", "v1", 0)
	cache.Get(ctx, "s1")  // hit
	cache.Get(ctx, "s1")  // hit
	cache.Get(ctx, "nope") // miss

	stats := cache.Stats()
	if stats.Hits != 2 {
		t.Errorf("want 2 hits, got %d", stats.Hits)
	}
	if stats.Misses != 1 {
		t.Errorf("want 1 miss, got %d", stats.Misses)
	}
	if stats.HitRate < 0.66 || stats.HitRate > 0.68 {
		t.Errorf("unexpected hit rate: %f", stats.HitRate)
	}
}

func TestCachedFunctionNode_CacheHit(t *testing.T) {
	var callCount atomic.Int32

	fn := func(ctx context.Context, s cacheTestState) (cacheTestState, error) {
		callCount.Add(1)
		s.Result = "computed:" + s.Query
		return s, nil
	}

	node := NewCachedFunctionNode("test", fn, NodeCacheConfig{
		Enabled: true,
		TTL:     time.Minute,
	})

	ctx := context.Background()
	input := cacheTestState{Query: "hello"}

	// 第一次调用 - 应执行函数
	r1, err := node.Invoke(ctx, input)
	if err != nil {
		t.Fatalf("first invoke failed: %v", err)
	}
	if callCount.Load() != 1 {
		t.Errorf("want 1 call, got %d", callCount.Load())
	}

	// 第二次相同输入 - 应命中缓存
	r2, err := node.Invoke(ctx, input)
	if err != nil {
		t.Fatalf("second invoke failed: %v", err)
	}
	if callCount.Load() != 1 {
		t.Errorf("want still 1 call (cache hit), got %d", callCount.Load())
	}
	if r1.Result != r2.Result {
		t.Errorf("cached result mismatch: %s != %s", r1.Result, r2.Result)
	}

	// 不同输入 - 应再次执行函数
	_, err = node.Invoke(ctx, cacheTestState{Query: "world"})
	if err != nil {
		t.Fatalf("third invoke failed: %v", err)
	}
	if callCount.Load() != 2 {
		t.Errorf("want 2 calls after different input, got %d", callCount.Load())
	}
}

func TestCachedFunctionNode_Disabled(t *testing.T) {
	var callCount atomic.Int32

	fn := func(ctx context.Context, s cacheTestState) (cacheTestState, error) {
		callCount.Add(1)
		s.Result = "computed"
		return s, nil
	}

	node := NewCachedFunctionNode("test", fn, NodeCacheConfig{
		Enabled: false,
	})

	ctx := context.Background()
	input := cacheTestState{Query: "q"}

	for i := 0; i < 3; i++ {
		_, _ = node.Invoke(ctx, input)
	}

	if callCount.Load() != 3 {
		t.Errorf("disabled cache should call fn every time, got %d calls", callCount.Load())
	}
}

func TestCachedFunctionNode_ErrorNotCached(t *testing.T) {
	var callCount atomic.Int32

	fn := func(ctx context.Context, s cacheTestState) (cacheTestState, error) {
		callCount.Add(1)
		if callCount.Load() == 1 {
			return s, errors.New("first call error")
		}
		s.Result = "ok"
		return s, nil
	}

	node := NewCachedFunctionNode("test", fn, NodeCacheConfig{
		Enabled: true,
		TTL:     time.Minute,
	})

	ctx := context.Background()
	input := cacheTestState{Query: "err-test"}

	// 第一次调用返回错误
	_, err := node.Invoke(ctx, input)
	if err == nil {
		t.Fatal("expected error on first call")
	}

	// 第二次调用应再次执行（错误结果不缓存）
	r2, err := node.Invoke(ctx, input)
	if err != nil {
		t.Fatalf("second call should succeed: %v", err)
	}
	if r2.Result != "ok" {
		t.Errorf("want ok, got %s", r2.Result)
	}
	if callCount.Load() != 2 {
		t.Errorf("want 2 calls, got %d", callCount.Load())
	}
}

func TestCachedFunctionNode_CustomKeyFunc(t *testing.T) {
	var callCount atomic.Int32

	fn := func(ctx context.Context, s cacheTestState) (cacheTestState, error) {
		callCount.Add(1)
		s.Result = "computed"
		return s, nil
	}

	// 自定义 key 函数：只用 Query 字段（忽略 Result）
	node := NewCachedFunctionNode("test", fn, NodeCacheConfig{
		Enabled: true,
		TTL:     time.Minute,
		KeyFunc: func(state any) (string, error) {
			s := state.(cacheTestState)
			return "query:" + s.Query, nil
		},
	})

	ctx := context.Background()

	// 两个 Query 相同但 Result 不同的状态应该共享缓存
	s1 := cacheTestState{Query: "shared", Result: "A"}
	s2 := cacheTestState{Query: "shared", Result: "B"}

	_, _ = node.Invoke(ctx, s1)
	_, _ = node.Invoke(ctx, s2)

	if callCount.Load() != 1 {
		t.Errorf("custom key func: want 1 call (same query), got %d", callCount.Load())
	}
}

func TestCachedFunctionNode_Invalidate(t *testing.T) {
	var callCount atomic.Int32

	fn := func(ctx context.Context, s cacheTestState) (cacheTestState, error) {
		callCount.Add(1)
		s.Result = "computed"
		return s, nil
	}

	node := NewCachedFunctionNode("test", fn, NodeCacheConfig{
		Enabled: true,
		TTL:     time.Minute,
	})

	ctx := context.Background()
	input := cacheTestState{Query: "inv"}

	_, _ = node.Invoke(ctx, input) // 写入缓存

	err := node.Invalidate(ctx, input)
	if err != nil {
		t.Fatalf("Invalidate failed: %v", err)
	}

	_, _ = node.Invoke(ctx, input) // 应重新执行
	if callCount.Load() != 2 {
		t.Errorf("after invalidate, want 2 calls, got %d", callCount.Load())
	}
}

func TestCachedFunctionNode_ContextCancel(t *testing.T) {
	fn := func(ctx context.Context, s cacheTestState) (cacheTestState, error) {
		return s, nil
	}

	node := NewCachedFunctionNode("test", fn, NodeCacheConfig{Enabled: true, TTL: time.Minute})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := node.Invoke(ctx, cacheTestState{Query: "q"})
	if err == nil {
		t.Error("expected context cancellation error")
	}
}

func TestFunctionNode_WithNodeCache(t *testing.T) {
	fn := func(ctx context.Context, s cacheTestState) (cacheTestState, error) {
		s.Result = "done"
		return s, nil
	}

	node := NewFunctionNode("fn", fn).WithNodeCache(NodeCacheConfig{
		Enabled: true,
		TTL:     time.Minute,
	})

	if node.GetName() != "fn" {
		t.Errorf("want name fn, got %s", node.GetName())
	}

	ctx := context.Background()
	r, err := node.Invoke(ctx, cacheTestState{Query: "test"})
	if err != nil || r.Result != "done" {
		t.Errorf("invoke failed: err=%v result=%s", err, r.Result)
	}
}

func TestCachedFunctionNode_ConcurrentAccess(t *testing.T) {
	var callCount atomic.Int32

	fn := func(ctx context.Context, s cacheTestState) (cacheTestState, error) {
		callCount.Add(1)
		time.Sleep(5 * time.Millisecond)
		s.Result = "done"
		return s, nil
	}

	node := NewCachedFunctionNode("concurrent", fn, NodeCacheConfig{
		Enabled: true,
		TTL:     time.Minute,
	})

	ctx := context.Background()
	input := cacheTestState{Query: "concurrent-key"}

	done := make(chan struct{}, 20)
	for i := 0; i < 20; i++ {
		go func() {
			defer func() { done <- struct{}{} }()
			_, _ = node.Invoke(ctx, input)
		}()
	}

	for i := 0; i < 20; i++ {
		<-done
	}

	stats := node.CacheStats()
	t.Logf("concurrent test: calls=%d hits=%d misses=%d", callCount.Load(), stats.Hits, stats.Misses)
}
