package failover

import (
	"context"
	"errors"
	"sync"
	"time"
)

var (
	// ErrNoHealthyNodes 没有健康节点
	ErrNoHealthyNodes = errors.New("failover: no healthy nodes available")

	// ErrNodeNotFound 节点未找到
	ErrNodeNotFound = errors.New("failover: node not found")

	// ErrRebalanceFailed 重新平衡失败
	ErrRebalanceFailed = errors.New("failover: rebalance failed")

	// ErrMigrationFailed 迁移失败
	ErrMigrationFailed = errors.New("failover: migration failed")
)

// FailoverManager 故障转移管理器接口
type FailoverManager interface {
	// MonitorHealth 监控节点健康
	MonitorHealth(ctx context.Context) error

	// HandleFailure 处理节点故障
	HandleFailure(ctx context.Context, nodeID string) error

	// RecoverNode 恢复节点
	RecoverNode(ctx context.Context, nodeID string) error

	// Rebalance 重新平衡负载
	Rebalance(ctx context.Context) error

	// GetStats 获取统计信息
	GetStats() *FailoverStats

	// Close 关闭管理器
	Close() error
}

// Config 故障转移配置
type Config struct {
	// HealthCheckInterval 健康检查间隔
	HealthCheckInterval time.Duration

	// FailureThreshold 故障阈值（连续失败次数）
	FailureThreshold int

	// RecoveryThreshold 恢复阈值（连续成功次数）
	RecoveryThreshold int

	// AutoRebalance 是否自动重新平衡
	AutoRebalance bool

	// RebalanceInterval 重新平衡间隔
	RebalanceInterval time.Duration

	// EnableAlerts 是否启用告警
	EnableAlerts bool

	// AlertCallback 告警回调函数
	AlertCallback func(alert Alert)
}

// DefaultConfig 返回默认配置
func DefaultConfig() Config {
	return Config{
		HealthCheckInterval: 10 * time.Second,
		FailureThreshold:    3,
		RecoveryThreshold:   2,
		AutoRebalance:       true,
		RebalanceInterval:   5 * time.Minute,
		EnableAlerts:        true,
	}
}

// FailoverStats 故障转移统计
type FailoverStats struct {
	// TotalFailures 总故障次数
	TotalFailures int64

	// TotalRecoveries 总恢复次数
	TotalRecoveries int64

	// ActiveFailovers 活跃的故障转移数
	ActiveFailovers int

	// TotalRebalances 总重新平衡次数
	TotalRebalances int64

	// LastFailureTime 最后故障时间
	LastFailureTime time.Time

	// LastRecoveryTime 最后恢复时间
	LastRecoveryTime time.Time

	// NodeStats 节点统计
	NodeStats map[string]*NodeFailoverStats
}

// NodeFailoverStats 节点故障转移统计
type NodeFailoverStats struct {
	// NodeID 节点 ID
	NodeID string

	// Failures 故障次数
	Failures int64

	// Recoveries 恢复次数
	Recoveries int64

	// LastFailureTime 最后故障时间
	LastFailureTime time.Time

	// LastRecoveryTime 最后恢复时间
	LastRecoveryTime time.Time

	// CurrentState 当前状态
	CurrentState NodeState
}

// NodeState 节点状态
type NodeState string

const (
	// NodeStateHealthy 健康
	NodeStateHealthy NodeState = "healthy"

	// NodeStateDegraded 降级
	NodeStateDegraded NodeState = "degraded"

	// NodeStateFailed 失败
	NodeStateFailed NodeState = "failed"

	// NodeStateRecovering 恢复中
	NodeStateRecovering NodeState = "recovering"
)

// Alert 告警
type Alert struct {
	// Type 告警类型
	Type AlertType

	// NodeID 节点 ID
	NodeID string

	// Message 告警消息
	Message string

	// Severity 严重程度
	Severity AlertSeverity

	// Timestamp 时间戳
	Timestamp time.Time

	// Metadata 元数据
	Metadata map[string]interface{}
}

// AlertType 告警类型
type AlertType string

const (
	// AlertTypeNodeFailure 节点故障
	AlertTypeNodeFailure AlertType = "node_failure"

	// AlertTypeNodeRecovery 节点恢复
	AlertTypeNodeRecovery AlertType = "node_recovery"

	// AlertTypeRebalance 重新平衡
	AlertTypeRebalance AlertType = "rebalance"

	// AlertTypeHighLoad 高负载
	AlertTypeHighLoad AlertType = "high_load"

	// AlertTypeLowCapacity 低容量
	AlertTypeLowCapacity AlertType = "low_capacity"
)

// AlertSeverity 告警严重程度
type AlertSeverity string

const (
	// SeverityInfo 信息
	SeverityInfo AlertSeverity = "info"

	// SeverityWarning 警告
	SeverityWarning AlertSeverity = "warning"

	// SeverityError 错误
	SeverityError AlertSeverity = "error"

	// SeverityCritical 严重
	SeverityCritical AlertSeverity = "critical"
)

// FailureEvent 故障事件
type FailureEvent struct {
	// NodeID 节点 ID
	NodeID string

	// Type 事件类型
	Type FailureEventType

	// Timestamp 时间戳
	Timestamp time.Time

	// Error 错误信息
	Error error

	// Metadata 元数据
	Metadata map[string]interface{}
}

// FailureEventType 故障事件类型
type FailureEventType string

const (
	// EventTypeHealthCheckFailed 健康检查失败
	EventTypeHealthCheckFailed FailureEventType = "health_check_failed"

	// EventTypeNodeMarkedFailed 节点标记为失败
	EventTypeNodeMarkedFailed FailureEventType = "node_marked_failed"

	// EventTypeFailoverStarted 故障转移开始
	EventTypeFailoverStarted FailureEventType = "failover_started"

	// EventTypeFailoverCompleted 故障转移完成
	EventTypeFailoverCompleted FailureEventType = "failover_completed"

	// EventTypeRecoveryStarted 恢复开始
	EventTypeRecoveryStarted FailureEventType = "recovery_started"

	// EventTypeRecoveryCompleted 恢复完成
	EventTypeRecoveryCompleted FailureEventType = "recovery_completed"
)

// HealthChecker 健康检查器
type HealthChecker interface {
	// CheckHealth 检查节点健康
	CheckHealth(ctx context.Context, nodeID string) error
}

// HealthCheckerFunc 健康检查函数
type HealthCheckerFunc func(ctx context.Context, nodeID string) error

// CheckHealth 实现 HealthChecker 接口
func (f HealthCheckerFunc) CheckHealth(ctx context.Context, nodeID string) error {
	return f(ctx, nodeID)
}

// RebalanceStrategy 重新平衡策略
type RebalanceStrategy interface {
	// ShouldRebalance 判断是否需要重新平衡
	ShouldRebalance(ctx context.Context) (bool, error)

	// CalculateRebalance 计算重新平衡方案
	CalculateRebalance(ctx context.Context) (*RebalancePlan, error)
}

// RebalancePlan 重新平衡计划
type RebalancePlan struct {
	// Migrations 迁移列表
	Migrations []Migration

	// EstimatedTime 预估时间
	EstimatedTime time.Duration

	// ExpectedImprovement 预期改进
	ExpectedImprovement float64
}

// Migration 迁移
type Migration struct {
	// FromNodeID 源节点 ID
	FromNodeID string

	// ToNodeID 目标节点 ID
	ToNodeID string

	// ResourceType 资源类型
	ResourceType string

	// ResourceIDs 资源 ID 列表
	ResourceIDs []string

	// Priority 优先级
	Priority int
}

// EventListener 事件监听器
type EventListener interface {
	// OnFailure 故障事件
	OnFailure(event FailureEvent)

	// OnRecovery 恢复事件
	OnRecovery(event FailureEvent)
}

// EventListenerFunc 事件监听函数
type EventListenerFunc struct {
	OnFailureFunc  func(event FailureEvent)
	OnRecoveryFunc func(event FailureEvent)
}

// OnFailure 实现 EventListener 接口
func (f *EventListenerFunc) OnFailure(event FailureEvent) {
	if f.OnFailureFunc != nil {
		f.OnFailureFunc(event)
	}
}

// OnRecovery 实现 EventListener 接口
func (f *EventListenerFunc) OnRecovery(event FailureEvent) {
	if f.OnRecoveryFunc != nil {
		f.OnRecoveryFunc(event)
	}
}

// failureTracker 故障追踪器
type failureTracker struct {
	failureCounts  map[string]int
	recoveryCounts map[string]int
	lastFailure    map[string]time.Time
	lastRecovery   map[string]time.Time
	mu             sync.RWMutex
}

// newFailureTracker 创建故障追踪器
func newFailureTracker() *failureTracker {
	return &failureTracker{
		failureCounts:  make(map[string]int),
		recoveryCounts: make(map[string]int),
		lastFailure:    make(map[string]time.Time),
		lastRecovery:   make(map[string]time.Time),
	}
}

// recordFailure 记录故障
func (t *failureTracker) recordFailure(nodeID string) int {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.failureCounts[nodeID]++
	t.recoveryCounts[nodeID] = 0
	t.lastFailure[nodeID] = time.Now()

	return t.failureCounts[nodeID]
}

// recordSuccess 记录成功
func (t *failureTracker) recordSuccess(nodeID string) int {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.recoveryCounts[nodeID]++
	if t.recoveryCounts[nodeID] >= 1 {
		t.failureCounts[nodeID] = 0
	}
	t.lastRecovery[nodeID] = time.Now()

	return t.recoveryCounts[nodeID]
}

// reset 重置节点计数
func (t *failureTracker) reset(nodeID string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	delete(t.failureCounts, nodeID)
	delete(t.recoveryCounts, nodeID)
	delete(t.lastFailure, nodeID)
	delete(t.lastRecovery, nodeID)
}

// getFailureCount 获取故障次数
func (t *failureTracker) getFailureCount(nodeID string) int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.failureCounts[nodeID]
}

// getRecoveryCount 获取恢复次数
func (t *failureTracker) getRecoveryCount(nodeID string) int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.recoveryCounts[nodeID]
}
