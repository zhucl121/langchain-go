package agents

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// CircuitState 电路状态。
type CircuitState string

const (
	// CircuitClosed 闭合状态（正常工作）
	CircuitClosed CircuitState = "closed"

	// CircuitOpen 断开状态（熔断，拒绝请求）
	CircuitOpen CircuitState = "open"

	// CircuitHalfOpen 半开状态（试探性允许少量请求）
	CircuitHalfOpen CircuitState = "half-open"
)

// 错误定义
var (
	ErrCircuitOpen       = errors.New("circuit breaker: circuit is open, rejecting request")
	ErrCircuitHalfOpen   = errors.New("circuit breaker: circuit is half-open, max probe requests reached")
)

// CircuitBreakerConfig 电路断路器配置。
type CircuitBreakerConfig struct {
	// Name 断路器名称（用于日志）
	Name string

	// FailureThreshold 触发熔断的连续失败次数（默认 5）
	FailureThreshold int

	// SuccessThreshold 半开状态转为闭合所需的连续成功次数（默认 2）
	SuccessThreshold int

	// Timeout 熔断持续时间（超过后进入半开）（默认 30s）
	Timeout time.Duration

	// HalfOpenMaxProbes 半开状态允许的最大探测请求数（默认 1）
	HalfOpenMaxProbes int

	// OnStateChange 状态变化回调
	OnStateChange func(name string, from, to CircuitState)
}

// DefaultCircuitBreakerConfig 返回默认配置。
func DefaultCircuitBreakerConfig(name string) CircuitBreakerConfig {
	return CircuitBreakerConfig{
		Name:              name,
		FailureThreshold:  5,
		SuccessThreshold:  2,
		Timeout:           30 * time.Second,
		HalfOpenMaxProbes: 1,
	}
}

// CircuitBreaker 电路断路器。
//
// 对标业界最佳实践，实现三态（Closed/Open/HalfOpen）电路断路器，
// 防止级联失败，在依赖服务不稳定时自动熔断。
//
// 使用示例：
//
//	cb := agents.NewCircuitBreaker(agents.DefaultCircuitBreakerConfig("openai-api"))
//
//	result, err := cb.Execute(ctx, func(ctx context.Context) (any, error) {
//	    return callOpenAI(ctx, prompt)
//	})
type CircuitBreaker struct {
	config CircuitBreakerConfig

	mu              sync.RWMutex
	state           CircuitState
	failureCount    int
	successCount    int
	lastFailureTime time.Time
	probeCount      int
}

// NewCircuitBreaker 创建电路断路器。
func NewCircuitBreaker(config CircuitBreakerConfig) *CircuitBreaker {
	if config.FailureThreshold <= 0 {
		config.FailureThreshold = 5
	}
	if config.SuccessThreshold <= 0 {
		config.SuccessThreshold = 2
	}
	if config.Timeout <= 0 {
		config.Timeout = 30 * time.Second
	}
	if config.HalfOpenMaxProbes <= 0 {
		config.HalfOpenMaxProbes = 1
	}

	return &CircuitBreaker{
		config: config,
		state:  CircuitClosed,
	}
}

// Execute 通过断路器执行函数。
//
// 根据当前状态决定是否允许执行：
//   - Closed：正常执行
//   - Open：直接返回 ErrCircuitOpen
//   - HalfOpen：允许有限探测请求
func (cb *CircuitBreaker) Execute(ctx context.Context, fn func(context.Context) (any, error)) (any, error) {
	// 检查是否可以执行
	if err := cb.allowRequest(); err != nil {
		return nil, err
	}

	// 执行
	result, err := fn(ctx)

	// 记录结果
	cb.recordResult(err)

	return result, err
}

// allowRequest 判断当前是否允许请求（需持有锁）。
func (cb *CircuitBreaker) allowRequest() error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case CircuitClosed:
		return nil

	case CircuitOpen:
		// 检查是否超过熔断超时，可以转为半开
		if time.Since(cb.lastFailureTime) >= cb.config.Timeout {
			cb.transitionTo(CircuitHalfOpen)
			cb.probeCount = 1
			return nil
		}
		return ErrCircuitOpen

	case CircuitHalfOpen:
		if cb.probeCount >= cb.config.HalfOpenMaxProbes {
			return ErrCircuitHalfOpen
		}
		cb.probeCount++
		return nil
	}

	return nil
}

// recordResult 记录执行结果，更新状态。
func (cb *CircuitBreaker) recordResult(err error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err != nil {
		cb.failureCount++
		cb.successCount = 0
		cb.lastFailureTime = time.Now()

		switch cb.state {
		case CircuitClosed:
			if cb.failureCount >= cb.config.FailureThreshold {
				cb.transitionTo(CircuitOpen)
			}
		case CircuitHalfOpen:
			// 半开状态下失败 -> 重新断开
			cb.transitionTo(CircuitOpen)
			cb.probeCount = 0
		}
	} else {
		cb.successCount++
		cb.failureCount = 0

		switch cb.state {
		case CircuitHalfOpen:
			if cb.successCount >= cb.config.SuccessThreshold {
				cb.transitionTo(CircuitClosed)
				cb.probeCount = 0
			}
		}
	}
}

// transitionTo 状态转移（需持有锁）。
func (cb *CircuitBreaker) transitionTo(newState CircuitState) {
	oldState := cb.state
	cb.state = newState

	if oldState != newState && cb.config.OnStateChange != nil {
		go cb.config.OnStateChange(cb.config.Name, oldState, newState)
	}
}

// State 返回当前状态。
func (cb *CircuitBreaker) State() CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// Stats 返回断路器统计信息。
func (cb *CircuitBreaker) Stats() CircuitBreakerStats {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return CircuitBreakerStats{
		Name:            cb.config.Name,
		State:           cb.state,
		FailureCount:    cb.failureCount,
		SuccessCount:    cb.successCount,
		LastFailureTime: cb.lastFailureTime,
	}
}

// Reset 手动重置断路器为闭合状态。
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.state = CircuitClosed
	cb.failureCount = 0
	cb.successCount = 0
	cb.probeCount = 0
}

// CircuitBreakerStats 断路器统计信息。
type CircuitBreakerStats struct {
	Name            string
	State           CircuitState
	FailureCount    int
	SuccessCount    int
	LastFailureTime time.Time
}

// ─── Bulkhead 舱壁隔离 ─────────────────────────────────────────

// BulkheadConfig 舱壁配置。
type BulkheadConfig struct {
	// Name 舱壁名称
	Name string

	// MaxConcurrent 最大并发数
	MaxConcurrent int

	// MaxWaitTime 最大等待时间（超过则拒绝）
	MaxWaitTime time.Duration
}

// Bulkhead 舱壁隔离。
//
// 通过限制并发数隔离不同 Agent/服务的资源消耗，防止某个组件的故障
// 耗尽所有资源，影响其他正常组件（级联失败防护）。
//
// 使用示例：
//
//	bh := agents.NewBulkhead(agents.BulkheadConfig{
//	    Name:          "llm-api",
//	    MaxConcurrent: 10,
//	    MaxWaitTime:   5 * time.Second,
//	})
//
//	result, err := bh.Execute(ctx, fn)
type Bulkhead struct {
	config  BulkheadConfig
	sem     chan struct{}
	mu      sync.RWMutex
	active  int
	rejected int64
	total   int64
}

// NewBulkhead 创建舱壁隔离。
func NewBulkhead(config BulkheadConfig) *Bulkhead {
	if config.MaxConcurrent <= 0 {
		config.MaxConcurrent = 10
	}
	return &Bulkhead{
		config: config,
		sem:    make(chan struct{}, config.MaxConcurrent),
	}
}

// Execute 通过舱壁执行函数。
func (bh *Bulkhead) Execute(ctx context.Context, fn func(context.Context) (any, error)) (any, error) {
	bh.mu.Lock()
	bh.total++
	bh.mu.Unlock()

	// 尝试获取信号量
	if bh.config.MaxWaitTime > 0 {
		waitCtx, cancel := context.WithTimeout(ctx, bh.config.MaxWaitTime)
		defer cancel()

		select {
		case bh.sem <- struct{}{}:
		case <-waitCtx.Done():
			bh.mu.Lock()
			bh.rejected++
			bh.mu.Unlock()
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			return nil, fmt.Errorf("bulkhead %s: max concurrent %d reached, request rejected after %s",
				bh.config.Name, bh.config.MaxConcurrent, bh.config.MaxWaitTime)
		}
	} else {
		select {
		case bh.sem <- struct{}{}:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	bh.mu.Lock()
	bh.active++
	bh.mu.Unlock()

	defer func() {
		<-bh.sem
		bh.mu.Lock()
		bh.active--
		bh.mu.Unlock()
	}()

	return fn(ctx)
}

// BulkheadStats 舱壁统计。
func (bh *Bulkhead) BulkheadStats() BulkheadStats {
	bh.mu.RLock()
	defer bh.mu.RUnlock()
	return BulkheadStats{
		Name:          bh.config.Name,
		MaxConcurrent: bh.config.MaxConcurrent,
		Active:        bh.active,
		Rejected:      bh.rejected,
		Total:         bh.total,
	}
}

// BulkheadStats 舱壁统计信息。
type BulkheadStats struct {
	Name          string
	MaxConcurrent int
	Active        int
	Rejected      int64
	Total         int64
}

// ─── ResilienceWrapper 弹性包装器 ─────────────────────────────

// ResilienceWrapper 将 CircuitBreaker + Bulkhead 组合为弹性执行器。
//
// 执行顺序：先通过 Bulkhead 限流，再通过 CircuitBreaker 熔断保护。
//
// 使用示例：
//
//	rw := agents.NewResilienceWrapper("openai", agents.ResilienceConfig{
//	    CircuitBreaker: agents.DefaultCircuitBreakerConfig("openai"),
//	    Bulkhead: agents.BulkheadConfig{
//	        Name: "openai", MaxConcurrent: 20, MaxWaitTime: 3*time.Second,
//	    },
//	})
//
//	result, err := rw.Execute(ctx, fn)
type ResilienceWrapper struct {
	name           string
	circuitBreaker *CircuitBreaker
	bulkhead       *Bulkhead
}

// ResilienceConfig 弹性配置。
type ResilienceConfig struct {
	CircuitBreaker CircuitBreakerConfig
	Bulkhead       BulkheadConfig
}

// NewResilienceWrapper 创建弹性执行包装器。
func NewResilienceWrapper(name string, config ResilienceConfig) *ResilienceWrapper {
	return &ResilienceWrapper{
		name:           name,
		circuitBreaker: NewCircuitBreaker(config.CircuitBreaker),
		bulkhead:       NewBulkhead(config.Bulkhead),
	}
}

// Execute 弹性执行（Bulkhead -> CircuitBreaker -> fn）。
func (rw *ResilienceWrapper) Execute(ctx context.Context, fn func(context.Context) (any, error)) (any, error) {
	return rw.bulkhead.Execute(ctx, func(ctx context.Context) (any, error) {
		return rw.circuitBreaker.Execute(ctx, fn)
	})
}

// State 返回断路器状态。
func (rw *ResilienceWrapper) State() CircuitState {
	return rw.circuitBreaker.State()
}

// CircuitBreakerStats 返回断路器统计。
func (rw *ResilienceWrapper) CircuitBreakerStats() CircuitBreakerStats {
	return rw.circuitBreaker.Stats()
}

// BulkheadStats 返回舱壁统计。
func (rw *ResilienceWrapper) BulkheadStats() BulkheadStats {
	return rw.bulkhead.BulkheadStats()
}
