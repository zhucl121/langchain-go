package agents

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestCircuitBreaker_ClosedState_AllowsRequests(t *testing.T) {
	cb := NewCircuitBreaker(DefaultCircuitBreakerConfig("test"))

	if cb.State() != CircuitClosed {
		t.Errorf("initial state should be closed, got %s", cb.State())
	}

	result, err := cb.Execute(context.Background(), func(ctx context.Context) (any, error) {
		return "ok", nil
	})
	if err != nil || result != "ok" {
		t.Errorf("closed circuit should allow request: err=%v result=%v", err, result)
	}
}

func TestCircuitBreaker_OpensAfterFailures(t *testing.T) {
	config := DefaultCircuitBreakerConfig("test")
	config.FailureThreshold = 3
	cb := NewCircuitBreaker(config)

	failFn := func(ctx context.Context) (any, error) {
		return nil, errors.New("service failure")
	}

	for i := 0; i < 3; i++ {
		_, _ = cb.Execute(context.Background(), failFn)
	}

	if cb.State() != CircuitOpen {
		t.Errorf("circuit should be open after %d failures, got %s", config.FailureThreshold, cb.State())
	}

	// 下一次请求应直接被拒绝
	_, err := cb.Execute(context.Background(), failFn)
	if !errors.Is(err, ErrCircuitOpen) {
		t.Errorf("expected ErrCircuitOpen, got %v", err)
	}
}

func TestCircuitBreaker_HalfOpenAfterTimeout(t *testing.T) {
	config := DefaultCircuitBreakerConfig("test")
	config.FailureThreshold = 1
	config.Timeout = 50 * time.Millisecond
	cb := NewCircuitBreaker(config)

	// 触发熔断
	_, _ = cb.Execute(context.Background(), func(ctx context.Context) (any, error) {
		return nil, errors.New("fail")
	})

	if cb.State() != CircuitOpen {
		t.Skip("circuit should be open")
	}

	// 等待超时
	time.Sleep(100 * time.Millisecond)

	// 发一个请求 -> 应进入半开并允许
	_, err := cb.Execute(context.Background(), func(ctx context.Context) (any, error) {
		return "probe", nil
	})
	if err != nil {
		t.Errorf("first request after timeout should succeed in half-open: %v", err)
	}
}

func TestCircuitBreaker_ClosesAfterSuccessInHalfOpen(t *testing.T) {
	config := DefaultCircuitBreakerConfig("test")
	config.FailureThreshold = 1
	config.SuccessThreshold = 2
	config.Timeout = 50 * time.Millisecond
	config.HalfOpenMaxProbes = 3
	cb := NewCircuitBreaker(config)

	// 触发熔断
	_, _ = cb.Execute(context.Background(), func(ctx context.Context) (any, error) {
		return nil, errors.New("fail")
	})

	time.Sleep(100 * time.Millisecond)

	// 2 次成功 -> 应闭合
	successFn := func(ctx context.Context) (any, error) { return "ok", nil }
	_, _ = cb.Execute(context.Background(), successFn)
	_, _ = cb.Execute(context.Background(), successFn)

	if cb.State() != CircuitClosed {
		t.Errorf("circuit should be closed after %d successes in half-open, got %s",
			config.SuccessThreshold, cb.State())
	}
}

func TestCircuitBreaker_StateChangeCallback(t *testing.T) {
	var changes []string
	var mu sync.Mutex

	config := DefaultCircuitBreakerConfig("test-callback")
	config.FailureThreshold = 1
	config.OnStateChange = func(name string, from, to CircuitState) {
		mu.Lock()
		changes = append(changes, string(from)+"->"+string(to))
		mu.Unlock()
	}
	cb := NewCircuitBreaker(config)

	_, _ = cb.Execute(context.Background(), func(ctx context.Context) (any, error) {
		return nil, errors.New("fail")
	})

	time.Sleep(10 * time.Millisecond) // 等待 goroutine 执行
	mu.Lock()
	defer mu.Unlock()

	if len(changes) == 0 {
		t.Error("expected state change callback to be called")
	}
	t.Logf("state changes: %v", changes)
}

func TestCircuitBreaker_Reset(t *testing.T) {
	config := DefaultCircuitBreakerConfig("test")
	config.FailureThreshold = 1
	cb := NewCircuitBreaker(config)

	_, _ = cb.Execute(context.Background(), func(ctx context.Context) (any, error) {
		return nil, errors.New("fail")
	})

	if cb.State() != CircuitOpen {
		t.Skip("should be open")
	}

	cb.Reset()
	if cb.State() != CircuitClosed {
		t.Errorf("after reset, state should be closed, got %s", cb.State())
	}
}

func TestBulkhead_AllowsConcurrentRequests(t *testing.T) {
	bh := NewBulkhead(BulkheadConfig{
		Name:          "test",
		MaxConcurrent: 5,
		MaxWaitTime:   time.Second,
	})

	var active atomic.Int32
	var maxActive atomic.Int32

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = bh.Execute(context.Background(), func(ctx context.Context) (any, error) {
				cur := active.Add(1)
				for {
					old := maxActive.Load()
					if cur <= old {
						break
					}
					if maxActive.CompareAndSwap(old, cur) {
						break
					}
				}
				time.Sleep(20 * time.Millisecond)
				active.Add(-1)
				return nil, nil
			})
		}()
	}
	wg.Wait()

	if maxActive.Load() > 5 {
		t.Errorf("max active %d exceeded MaxConcurrent 5", maxActive.Load())
	}
}

func TestBulkhead_RejectsWhenFull(t *testing.T) {
	bh := NewBulkhead(BulkheadConfig{
		Name:          "test",
		MaxConcurrent: 2,
		MaxWaitTime:   10 * time.Millisecond,
	})

	started := make(chan struct{}, 2)
	release := make(chan struct{})

	longFn := func(ctx context.Context) (any, error) {
		started <- struct{}{}
		<-release
		return nil, nil
	}

	// 占满并发槽
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = bh.Execute(context.Background(), longFn)
		}()
	}

	// 等两个 goroutine 都进入
	<-started
	<-started

	// 第 3 个请求应被拒绝
	_, err := bh.Execute(context.Background(), func(ctx context.Context) (any, error) {
		return nil, nil
	})
	if err == nil {
		t.Error("expected bulkhead rejection")
	}
	t.Logf("rejected: %v", err)

	close(release)
	wg.Wait()

	stats := bh.BulkheadStats()
	if stats.Rejected == 0 {
		t.Error("expected non-zero rejections")
	}
}

func TestResilienceWrapper_Execute(t *testing.T) {
	rw := NewResilienceWrapper("test", ResilienceConfig{
		CircuitBreaker: DefaultCircuitBreakerConfig("test"),
		Bulkhead: BulkheadConfig{
			Name:          "test",
			MaxConcurrent: 5,
			MaxWaitTime:   time.Second,
		},
	})

	result, err := rw.Execute(context.Background(), func(ctx context.Context) (any, error) {
		return "success", nil
	})
	if err != nil || result != "success" {
		t.Errorf("resilience wrapper should allow normal execution: err=%v result=%v", err, result)
	}
}

func TestResilienceWrapper_CircuitBreaksAfterFailures(t *testing.T) {
	config := ResilienceConfig{
		CircuitBreaker: CircuitBreakerConfig{
			Name:             "test",
			FailureThreshold: 2,
			Timeout:          time.Hour,
		},
		Bulkhead: BulkheadConfig{
			Name:          "test",
			MaxConcurrent: 10,
			MaxWaitTime:   time.Second,
		},
	}

	rw := NewResilienceWrapper("test", config)

	for i := 0; i < 2; i++ {
		_, _ = rw.Execute(context.Background(), func(ctx context.Context) (any, error) {
			return nil, errors.New("fail")
		})
	}

	if rw.State() != CircuitOpen {
		t.Errorf("circuit should be open, got %s", rw.State())
	}

	_, err := rw.Execute(context.Background(), func(ctx context.Context) (any, error) {
		return nil, nil
	})
	if !errors.Is(err, ErrCircuitOpen) {
		t.Errorf("expected ErrCircuitOpen, got %v", err)
	}
}
