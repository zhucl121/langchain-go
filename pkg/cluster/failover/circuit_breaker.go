package failover

import (
	"context"
	"errors"
	"sync"
	"time"
)

var (
	// ErrCircuitOpen 熔断器打开
	ErrCircuitOpen = errors.New("circuit breaker: circuit is open")

	// ErrTooManyRequests 请求过多
	ErrTooManyRequests = errors.New("circuit breaker: too many requests")
)

// CircuitState 熔断器状态
type CircuitState string

const (
	// StateClosed 关闭状态（正常）
	StateClosed CircuitState = "closed"

	// StateHalfOpen 半开状态（测试）
	StateHalfOpen CircuitState = "half_open"

	// StateOpen 打开状态（熔断）
	StateOpen CircuitState = "open"
)

// CircuitBreaker 熔断器
type CircuitBreaker struct {
	config       CircuitBreakerConfig
	state        CircuitState
	failureCount int64
	successCount int64
	lastFailure  time.Time
	openTime     time.Time
	mu           sync.RWMutex
	stats        *CircuitBreakerStats
}

// CircuitBreakerConfig 熔断器配置
type CircuitBreakerConfig struct {
	// FailureThreshold 失败阈值
	FailureThreshold int64

	// SuccessThreshold 成功阈值（半开状态）
	SuccessThreshold int64

	// Timeout 超时时间（打开到半开的等待时间）
	Timeout time.Duration

	// MaxRequests 半开状态下最大请求数
	MaxRequests int

	// OnStateChange 状态变化回调
	OnStateChange func(from, to CircuitState)
}

// DefaultCircuitBreakerConfig 返回默认配置
func DefaultCircuitBreakerConfig() CircuitBreakerConfig {
	return CircuitBreakerConfig{
		FailureThreshold: 5,
		SuccessThreshold: 2,
		Timeout:          30 * time.Second,
		MaxRequests:      1,
	}
}

// CircuitBreakerStats 熔断器统计
type CircuitBreakerStats struct {
	// State 当前状态
	State CircuitState

	// TotalRequests 总请求数
	TotalRequests int64

	// SuccessRequests 成功请求数
	SuccessRequests int64

	// FailedRequests 失败请求数
	FailedRequests int64

	// RejectedRequests 被拒绝的请求数
	RejectedRequests int64

	// LastStateChange 最后状态变化时间
	LastStateChange time.Time

	// OpenTime 打开时间
	OpenTime time.Time
}

// NewCircuitBreaker 创建熔断器
func NewCircuitBreaker(config CircuitBreakerConfig) *CircuitBreaker {
	if config.FailureThreshold == 0 {
		config.FailureThreshold = 5
	}
	if config.SuccessThreshold == 0 {
		config.SuccessThreshold = 2
	}
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	if config.MaxRequests == 0 {
		config.MaxRequests = 1
	}

	return &CircuitBreaker{
		config: config,
		state:  StateClosed,
		stats: &CircuitBreakerStats{
			State: StateClosed,
		},
	}
}

// Execute 执行操作
func (cb *CircuitBreaker) Execute(fn func() error) error {
	return cb.ExecuteContext(context.Background(), fn)
}

// ExecuteContext 执行操作（带上下文）
func (cb *CircuitBreaker) ExecuteContext(ctx context.Context, fn func() error) error {
	// 检查是否允许执行
	if err := cb.beforeRequest(); err != nil {
		return err
	}

	// 执行操作
	err := fn()

	// 记录结果
	cb.afterRequest(err)

	return err
}

// beforeRequest 请求前检查
func (cb *CircuitBreaker) beforeRequest() error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.stats.TotalRequests++

	switch cb.state {
	case StateClosed:
		// 关闭状态，允许请求
		return nil

	case StateOpen:
		// 检查是否可以转为半开
		if time.Since(cb.openTime) >= cb.config.Timeout {
			cb.setState(StateHalfOpen)
			return nil
		}

		// 仍处于打开状态，拒绝请求
		cb.stats.RejectedRequests++
		return ErrCircuitOpen

	case StateHalfOpen:
		// 半开状态，限制并发请求数
		if cb.successCount >= int64(cb.config.MaxRequests) {
			cb.stats.RejectedRequests++
			return ErrTooManyRequests
		}
		return nil

	default:
		return nil
	}
}

// afterRequest 请求后处理
func (cb *CircuitBreaker) afterRequest(err error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err != nil {
		// 请求失败
		cb.onFailure()
	} else {
		// 请求成功
		cb.onSuccess()
	}
}

// onSuccess 成功处理
func (cb *CircuitBreaker) onSuccess() {
	cb.stats.SuccessRequests++

	switch cb.state {
	case StateClosed:
		// 关闭状态，重置失败计数
		cb.failureCount = 0

	case StateHalfOpen:
		// 半开状态，增加成功计数
		cb.successCount++

		// 如果达到成功阈值，转为关闭状态
		if cb.successCount >= cb.config.SuccessThreshold {
			cb.setState(StateClosed)
			cb.failureCount = 0
			cb.successCount = 0
		}
	}
}

// onFailure 失败处理
func (cb *CircuitBreaker) onFailure() {
	cb.stats.FailedRequests++
	cb.lastFailure = time.Now()

	switch cb.state {
	case StateClosed:
		// 关闭状态，增加失败计数
		cb.failureCount++

		// 如果达到失败阈值，转为打开状态
		if cb.failureCount >= cb.config.FailureThreshold {
			cb.setState(StateOpen)
			cb.openTime = time.Now()
		}

	case StateHalfOpen:
		// 半开状态遇到失败，立即转为打开状态
		cb.setState(StateOpen)
		cb.openTime = time.Now()
		cb.successCount = 0
	}
}

// setState 设置状态
func (cb *CircuitBreaker) setState(newState CircuitState) {
	if cb.state == newState {
		return
	}

	oldState := cb.state
	cb.state = newState
	cb.stats.State = newState
	cb.stats.LastStateChange = time.Now()

	if newState == StateOpen {
		cb.stats.OpenTime = time.Now()
	}

	// 回调通知
	if cb.config.OnStateChange != nil {
		cb.config.OnStateChange(oldState, newState)
	}
}

// GetState 获取当前状态
func (cb *CircuitBreaker) GetState() CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// GetStats 获取统计信息
func (cb *CircuitBreaker) GetStats() *CircuitBreakerStats {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.stats
}

// Reset 重置熔断器
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.setState(StateClosed)
	cb.failureCount = 0
	cb.successCount = 0
	cb.stats = &CircuitBreakerStats{
		State: StateClosed,
	}
}

// ForceOpen 强制打开
func (cb *CircuitBreaker) ForceOpen() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.setState(StateOpen)
	cb.openTime = time.Now()
}

// ForceClose 强制关闭
func (cb *CircuitBreaker) ForceClose() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.setState(StateClosed)
	cb.failureCount = 0
	cb.successCount = 0
}
