package health

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/zhucl121/langchain-go/pkg/cluster/node"
)

var (
	// ErrCheckTimeout 健康检查超时
	ErrCheckTimeout = errors.New("health: check timeout")

	// ErrCheckFailed 健康检查失败
	ErrCheckFailed = errors.New("health: check failed")

	// ErrInvalidConfig 无效的配置
	ErrInvalidConfig = errors.New("health: invalid config")
)

// CheckResult 健康检查结果
type CheckResult struct {
	// Healthy 是否健康
	Healthy bool

	// Status 健康状态
	Status HealthStatus

	// Message 状态消息
	Message string

	// Latency 检查延迟
	Latency time.Duration

	// Timestamp 检查时间
	Timestamp time.Time

	// Metadata 额外元数据
	Metadata map[string]interface{}
}

// HealthStatus 健康状态
type HealthStatus string

const (
	// HealthStatusHealthy 健康
	HealthStatusHealthy HealthStatus = "healthy"

	// HealthStatusUnhealthy 不健康
	HealthStatusUnhealthy HealthStatus = "unhealthy"

	// HealthStatusDegraded 降级
	HealthStatusDegraded HealthStatus = "degraded"

	// HealthStatusUnknown 未知
	HealthStatusUnknown HealthStatus = "unknown"
)

// Checker 健康检查器接口
type Checker interface {
	// Check 执行健康检查
	Check(ctx context.Context, n *node.Node) (*CheckResult, error)

	// Type 返回检查器类型
	Type() string
}

// CheckerFunc 函数类型的健康检查器
type CheckerFunc func(ctx context.Context, n *node.Node) (*CheckResult, error)

// Check 实现 Checker 接口
func (f CheckerFunc) Check(ctx context.Context, n *node.Node) (*CheckResult, error) {
	return f(ctx, n)
}

// Type 返回类型
func (f CheckerFunc) Type() string {
	return "custom"
}

// CompositeChecker 组合健康检查器
//
// 可以组合多个检查器，所有检查器都通过才认为节点健康。
type CompositeChecker struct {
	checkers []Checker
	strategy AggregationStrategy
}

// AggregationStrategy 聚合策略
type AggregationStrategy string

const (
	// StrategyAll 所有检查器都通过才健康
	StrategyAll AggregationStrategy = "all"

	// StrategyAny 任意一个检查器通过就健康
	StrategyAny AggregationStrategy = "any"

	// StrategyMajority 大多数检查器通过就健康
	StrategyMajority AggregationStrategy = "majority"
)

// NewCompositeChecker 创建组合检查器
func NewCompositeChecker(strategy AggregationStrategy, checkers ...Checker) *CompositeChecker {
	return &CompositeChecker{
		checkers: checkers,
		strategy: strategy,
	}
}

// Check 执行所有健康检查
func (c *CompositeChecker) Check(ctx context.Context, n *node.Node) (*CheckResult, error) {
	if len(c.checkers) == 0 {
		return &CheckResult{
			Healthy:   true,
			Status:    HealthStatusHealthy,
			Message:   "No checkers configured",
			Timestamp: time.Now(),
		}, nil
	}

	results := make([]*CheckResult, len(c.checkers))
	healthyCount := 0
	totalLatency := time.Duration(0)

	// 执行所有检查
	for i, checker := range c.checkers {
		result, err := checker.Check(ctx, n)
		if err != nil {
			results[i] = &CheckResult{
				Healthy:   false,
				Status:    HealthStatusUnhealthy,
				Message:   err.Error(),
				Timestamp: time.Now(),
			}
			continue
		}

		results[i] = result
		if result.Healthy {
			healthyCount++
		}
		totalLatency += result.Latency
	}

	// 根据策略聚合结果
	healthy := false
	switch c.strategy {
	case StrategyAll:
		healthy = healthyCount == len(c.checkers)
	case StrategyAny:
		healthy = healthyCount > 0
	case StrategyMajority:
		healthy = healthyCount > len(c.checkers)/2
	}

	status := HealthStatusHealthy
	if !healthy {
		status = HealthStatusUnhealthy
	} else if healthyCount < len(c.checkers) {
		status = HealthStatusDegraded
	}

	return &CheckResult{
		Healthy:   healthy,
		Status:    status,
		Message:   c.buildMessage(results),
		Latency:   totalLatency / time.Duration(len(c.checkers)),
		Timestamp: time.Now(),
		Metadata: map[string]interface{}{
			"total_checks":   len(c.checkers),
			"healthy_checks": healthyCount,
			"results":        results,
		},
	}, nil
}

// Type 返回类型
func (c *CompositeChecker) Type() string {
	return "composite"
}

// buildMessage 构建聚合消息
func (c *CompositeChecker) buildMessage(results []*CheckResult) string {
	if len(results) == 0 {
		return "No checks performed"
	}

	healthyCount := 0
	for _, result := range results {
		if result.Healthy {
			healthyCount++
		}
	}

	return fmt.Sprintf("%d/%d checks passed", healthyCount, len(results))
}

// PeriodicChecker 定期健康检查器
//
// 包装一个检查器，定期执行健康检查并缓存结果。
type PeriodicChecker struct {
	checker   Checker
	interval  time.Duration
	lastCheck *CheckResult
	mu        sync.RWMutex
	stopCh    chan struct{}
	started   bool
}

// NewPeriodicChecker 创建定期检查器
func NewPeriodicChecker(checker Checker, interval time.Duration) *PeriodicChecker {
	return &PeriodicChecker{
		checker:  checker,
		interval: interval,
		stopCh:   make(chan struct{}),
	}
}

// Start 开始定期检查
func (p *PeriodicChecker) Start(ctx context.Context, n *node.Node) {
	p.mu.Lock()
	if p.started {
		p.mu.Unlock()
		return
	}
	p.started = true
	p.mu.Unlock()

	go func() {
		ticker := time.NewTicker(p.interval)
		defer ticker.Stop()

		// 立即执行一次检查
		p.runCheck(ctx, n)

		for {
			select {
			case <-ticker.C:
				p.runCheck(ctx, n)
			case <-p.stopCh:
				return
			case <-ctx.Done():
				return
			}
		}
	}()
}

// Stop 停止定期检查
func (p *PeriodicChecker) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.started {
		return
	}

	close(p.stopCh)
	p.started = false
}

// Check 返回最近的检查结果
func (p *PeriodicChecker) Check(ctx context.Context, n *node.Node) (*CheckResult, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.lastCheck == nil {
		return &CheckResult{
			Healthy:   false,
			Status:    HealthStatusUnknown,
			Message:   "No health check performed yet",
			Timestamp: time.Now(),
		}, nil
	}

	return p.lastCheck, nil
}

// Type 返回类型
func (p *PeriodicChecker) Type() string {
	return "periodic_" + p.checker.Type()
}

// runCheck 执行检查并更新结果
func (p *PeriodicChecker) runCheck(ctx context.Context, n *node.Node) {
	result, err := p.checker.Check(ctx, n)
	if err != nil {
		result = &CheckResult{
			Healthy:   false,
			Status:    HealthStatusUnhealthy,
			Message:   err.Error(),
			Timestamp: time.Now(),
		}
	}

	p.mu.Lock()
	p.lastCheck = result
	p.mu.Unlock()
}
