package failover

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// DefaultFailoverManager 默认故障转移管理器
type DefaultFailoverManager struct {
	config     Config
	checker    HealthChecker
	tracker    *failureTracker
	listeners  []EventListener
	stats      *FailoverStats
	mu         sync.RWMutex
	stopCh     chan struct{}
	wg         sync.WaitGroup
}

// NewFailoverManager 创建故障转移管理器
func NewFailoverManager(config Config, checker HealthChecker) *DefaultFailoverManager {
	if config.HealthCheckInterval == 0 {
		config.HealthCheckInterval = 10 * time.Second
	}
	if config.FailureThreshold == 0 {
		config.FailureThreshold = 3
	}
	if config.RecoveryThreshold == 0 {
		config.RecoveryThreshold = 2
	}

	return &DefaultFailoverManager{
		config:  config,
		checker: checker,
		tracker: newFailureTracker(),
		stats: &FailoverStats{
			NodeStats: make(map[string]*NodeFailoverStats),
		},
		stopCh: make(chan struct{}),
	}
}

// MonitorHealth 监控节点健康
func (m *DefaultFailoverManager) MonitorHealth(ctx context.Context) error {
	ticker := time.NewTicker(m.config.HealthCheckInterval)
	defer ticker.Stop()

	// 如果启用自动重新平衡，启动重新平衡协程
	if m.config.AutoRebalance && m.config.RebalanceInterval > 0 {
		m.wg.Add(1)
		go m.autoRebalance(ctx)
	}

	for {
		select {
		case <-ctx.Done():
			m.wg.Wait()
			return ctx.Err()
		case <-m.stopCh:
			m.wg.Wait()
			return nil
		case <-ticker.C:
			// 这里简化实现，实际应该从节点管理器获取节点列表
			// 由于我们没有直接依赖 node 包，所以在实际使用时需要通过 HealthChecker 来检查
		}
	}
}

// HandleFailure 处理节点故障
func (m *DefaultFailoverManager) HandleFailure(ctx context.Context, nodeID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 更新统计
	atomic.AddInt64(&m.stats.TotalFailures, 1)
	m.stats.LastFailureTime = time.Now()
	m.stats.ActiveFailovers++

	// 初始化节点统计
	if _, exists := m.stats.NodeStats[nodeID]; !exists {
		m.stats.NodeStats[nodeID] = &NodeFailoverStats{
			NodeID: nodeID,
		}
	}

	nodeStats := m.stats.NodeStats[nodeID]
	nodeStats.Failures++
	nodeStats.LastFailureTime = time.Now()
	nodeStats.CurrentState = NodeStateFailed

	// 触发故障事件
	event := FailureEvent{
		NodeID:    nodeID,
		Type:      EventTypeNodeMarkedFailed,
		Timestamp: time.Now(),
		Metadata: map[string]interface{}{
			"failure_count": nodeStats.Failures,
		},
	}

	m.notifyListeners(event, true)

	// 发送告警
	if m.config.EnableAlerts {
		m.sendAlert(Alert{
			Type:      AlertTypeNodeFailure,
			NodeID:    nodeID,
			Message:   fmt.Sprintf("Node %s has failed", nodeID),
			Severity:  SeverityCritical,
			Timestamp: time.Now(),
			Metadata: map[string]interface{}{
				"failure_count": nodeStats.Failures,
			},
		})
	}

	// 启动故障转移
	event = FailureEvent{
		NodeID:    nodeID,
		Type:      EventTypeFailoverStarted,
		Timestamp: time.Now(),
	}
	m.notifyListeners(event, true)

	// 完成故障转移
	m.stats.ActiveFailovers--

	event = FailureEvent{
		NodeID:    nodeID,
		Type:      EventTypeFailoverCompleted,
		Timestamp: time.Now(),
	}
	m.notifyListeners(event, true)

	return nil
}

// RecoverNode 恢复节点
func (m *DefaultFailoverManager) RecoverNode(ctx context.Context, nodeID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 更新统计
	atomic.AddInt64(&m.stats.TotalRecoveries, 1)
	m.stats.LastRecoveryTime = time.Now()

	// 更新节点统计
	if nodeStats, exists := m.stats.NodeStats[nodeID]; exists {
		nodeStats.Recoveries++
		nodeStats.LastRecoveryTime = time.Now()
		nodeStats.CurrentState = NodeStateHealthy
	}

	// 重置故障计数
	m.tracker.reset(nodeID)

	// 触发恢复事件
	event := FailureEvent{
		NodeID:    nodeID,
		Type:      EventTypeRecoveryStarted,
		Timestamp: time.Now(),
	}
	m.notifyListeners(event, false)

	// 发送告警
	if m.config.EnableAlerts {
		m.sendAlert(Alert{
			Type:      AlertTypeNodeRecovery,
			NodeID:    nodeID,
			Message:   fmt.Sprintf("Node %s has recovered", nodeID),
			Severity:  SeverityInfo,
			Timestamp: time.Now(),
		})
	}

	event = FailureEvent{
		NodeID:    nodeID,
		Type:      EventTypeRecoveryCompleted,
		Timestamp: time.Now(),
	}
	m.notifyListeners(event, false)

	return nil
}

// Rebalance 重新平衡负载
func (m *DefaultFailoverManager) Rebalance(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 更新统计
	atomic.AddInt64(&m.stats.TotalRebalances, 1)

	// 发送告警
	if m.config.EnableAlerts {
		m.sendAlert(Alert{
			Type:      AlertTypeRebalance,
			Message:   "Cluster rebalancing started",
			Severity:  SeverityInfo,
			Timestamp: time.Now(),
		})
	}

	// 这里简化实现，实际应该计算并执行重新平衡计划
	// 实际使用时需要传入负载均衡器和节点列表

	return nil
}

// GetStats 获取统计信息
func (m *DefaultFailoverManager) GetStats() *FailoverStats {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.stats
}

// Close 关闭管理器
func (m *DefaultFailoverManager) Close() error {
	close(m.stopCh)
	m.wg.Wait()
	return nil
}

// AddListener 添加事件监听器
func (m *DefaultFailoverManager) AddListener(listener EventListener) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.listeners = append(m.listeners, listener)
}

// autoRebalance 自动重新平衡
func (m *DefaultFailoverManager) autoRebalance(ctx context.Context) {
	defer m.wg.Done()

	ticker := time.NewTicker(m.config.RebalanceInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-m.stopCh:
			return
		case <-ticker.C:
			if err := m.Rebalance(ctx); err != nil {
				// 记录错误但继续运行
				continue
			}
		}
	}
}

// notifyListeners 通知监听器
func (m *DefaultFailoverManager) notifyListeners(event FailureEvent, isFailure bool) {
	for _, listener := range m.listeners {
		if isFailure {
			listener.OnFailure(event)
		} else {
			listener.OnRecovery(event)
		}
	}
}

// sendAlert 发送告警
func (m *DefaultFailoverManager) sendAlert(alert Alert) {
	if m.config.AlertCallback != nil {
		m.config.AlertCallback(alert)
	}
}

// CheckNodeHealth 检查节点健康（辅助方法）
func (m *DefaultFailoverManager) CheckNodeHealth(ctx context.Context, nodeID string) error {
	if m.checker == nil {
		return nil
	}

	err := m.checker.CheckHealth(ctx, nodeID)

	m.mu.Lock()
	defer m.mu.Unlock()

	if err != nil {
		// 记录失败
		failureCount := m.tracker.recordFailure(nodeID)

		// 如果达到失败阈值，触发故障转移
		if failureCount >= m.config.FailureThreshold {
			go m.HandleFailure(ctx, nodeID)
		}

		return err
	}

	// 记录成功
	recoveryCount := m.tracker.recordSuccess(nodeID)

	// 如果达到恢复阈值，恢复节点
	if recoveryCount >= m.config.RecoveryThreshold {
		// 检查节点当前是否处于失败状态
		if nodeStats, exists := m.stats.NodeStats[nodeID]; exists {
			if nodeStats.CurrentState == NodeStateFailed {
				go m.RecoverNode(ctx, nodeID)
			}
		}
	}

	return nil
}
