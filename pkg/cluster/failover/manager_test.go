package failover

import (
	"context"
	"testing"
	"time"
)

// mockHealthChecker 模拟健康检查器
type mockHealthChecker struct {
	shouldFail bool
}

func (m *mockHealthChecker) CheckHealth(ctx context.Context, nodeID string) error {
	if m.shouldFail {
		return ErrNodeNotFound
	}
	return nil
}

func TestNewFailoverManager(t *testing.T) {
	config := DefaultConfig()
	checker := &mockHealthChecker{}

	manager := NewFailoverManager(config, checker)

	if manager == nil {
		t.Fatal("NewFailoverManager() returned nil")
	}

	if manager.config.HealthCheckInterval != config.HealthCheckInterval {
		t.Errorf("HealthCheckInterval = %v, want %v", manager.config.HealthCheckInterval, config.HealthCheckInterval)
	}
}

func TestHandleFailure(t *testing.T) {
	config := DefaultConfig()
	config.EnableAlerts = false // 禁用告警避免回调问题
	checker := &mockHealthChecker{}

	manager := NewFailoverManager(config, checker)

	ctx := context.Background()
	nodeID := "test-node-1"

	// 处理故障
	err := manager.HandleFailure(ctx, nodeID)
	if err != nil {
		t.Fatalf("HandleFailure() error = %v", err)
	}

	// 验证统计
	stats := manager.GetStats()
	if stats.TotalFailures != 1 {
		t.Errorf("TotalFailures = %d, want 1", stats.TotalFailures)
	}

	// 验证节点统计
	nodeStats, exists := stats.NodeStats[nodeID]
	if !exists {
		t.Fatal("Node stats not found")
	}

	if nodeStats.Failures != 1 {
		t.Errorf("Node failures = %d, want 1", nodeStats.Failures)
	}

	if nodeStats.CurrentState != NodeStateFailed {
		t.Errorf("Node state = %s, want %s", nodeStats.CurrentState, NodeStateFailed)
	}
}

func TestRecoverNode(t *testing.T) {
	config := DefaultConfig()
	config.EnableAlerts = false
	checker := &mockHealthChecker{}

	manager := NewFailoverManager(config, checker)

	ctx := context.Background()
	nodeID := "test-node-1"

	// 先触发故障
	manager.HandleFailure(ctx, nodeID)

	// 恢复节点
	err := manager.RecoverNode(ctx, nodeID)
	if err != nil {
		t.Fatalf("RecoverNode() error = %v", err)
	}

	// 验证统计
	stats := manager.GetStats()
	if stats.TotalRecoveries != 1 {
		t.Errorf("TotalRecoveries = %d, want 1", stats.TotalRecoveries)
	}

	// 验证节点统计
	nodeStats := stats.NodeStats[nodeID]
	if nodeStats.Recoveries != 1 {
		t.Errorf("Node recoveries = %d, want 1", nodeStats.Recoveries)
	}

	if nodeStats.CurrentState != NodeStateHealthy {
		t.Errorf("Node state = %s, want %s", nodeStats.CurrentState, NodeStateHealthy)
	}
}

func TestCheckNodeHealth_Failure(t *testing.T) {
	config := DefaultConfig()
	config.FailureThreshold = 2
	config.EnableAlerts = false

	checker := &mockHealthChecker{shouldFail: true}
	manager := NewFailoverManager(config, checker)

	ctx := context.Background()
	nodeID := "test-node-1"

	// 第一次检查失败
	err := manager.CheckNodeHealth(ctx, nodeID)
	if err == nil {
		t.Error("Expected error, got nil")
	}

	failureCount := manager.tracker.getFailureCount(nodeID)
	if failureCount != 1 {
		t.Errorf("Failure count = %d, want 1", failureCount)
	}

	// 第二次检查失败，应该触发故障转移
	time.Sleep(10 * time.Millisecond) // 等待异步处理
	err = manager.CheckNodeHealth(ctx, nodeID)
	if err == nil {
		t.Error("Expected error, got nil")
	}

	failureCount = manager.tracker.getFailureCount(nodeID)
	if failureCount != 2 {
		t.Errorf("Failure count = %d, want 2", failureCount)
	}

	// 等待故障转移完成
	time.Sleep(50 * time.Millisecond)

	// 验证故障被记录
	stats := manager.GetStats()
	if stats.TotalFailures == 0 {
		t.Error("Expected at least one failure")
	}
}

func TestCheckNodeHealth_Recovery(t *testing.T) {
	config := DefaultConfig()
	config.RecoveryThreshold = 2
	config.EnableAlerts = false

	checker := &mockHealthChecker{shouldFail: false}
	manager := NewFailoverManager(config, checker)

	ctx := context.Background()
	nodeID := "test-node-1"

	// 先标记为失败
	manager.HandleFailure(ctx, nodeID)

	// 第一次健康检查成功
	err := manager.CheckNodeHealth(ctx, nodeID)
	if err != nil {
		t.Errorf("CheckNodeHealth() error = %v", err)
	}

	recoveryCount := manager.tracker.getRecoveryCount(nodeID)
	if recoveryCount != 1 {
		t.Errorf("Recovery count = %d, want 1", recoveryCount)
	}

	// 第二次健康检查成功，应该触发恢复
	time.Sleep(10 * time.Millisecond)
	err = manager.CheckNodeHealth(ctx, nodeID)
	if err != nil {
		t.Errorf("CheckNodeHealth() error = %v", err)
	}

	recoveryCount = manager.tracker.getRecoveryCount(nodeID)
	if recoveryCount != 2 {
		t.Errorf("Recovery count = %d, want 2", recoveryCount)
	}

	// 等待恢复完成
	time.Sleep(50 * time.Millisecond)

	// 验证恢复被记录
	stats := manager.GetStats()
	if stats.TotalRecoveries == 0 {
		t.Error("Expected at least one recovery")
	}
}

func TestEventListener(t *testing.T) {
	config := DefaultConfig()
	config.EnableAlerts = false
	checker := &mockHealthChecker{}

	manager := NewFailoverManager(config, checker)

	ctx := context.Background()
	nodeID := "test-node-1"

	// 添加事件监听器
	failureReceived := false
	recoveryReceived := false

	listener := &EventListenerFunc{
		OnFailureFunc: func(event FailureEvent) {
			if event.Type == EventTypeNodeMarkedFailed {
				failureReceived = true
			}
		},
		OnRecoveryFunc: func(event FailureEvent) {
			if event.Type == EventTypeRecoveryStarted {
				recoveryReceived = true
			}
		},
	}

	manager.AddListener(listener)

	// 触发故障
	manager.HandleFailure(ctx, nodeID)
	if !failureReceived {
		t.Error("Failure event not received")
	}

	// 触发恢复
	manager.RecoverNode(ctx, nodeID)
	if !recoveryReceived {
		t.Error("Recovery event not received")
	}
}

func TestRebalance(t *testing.T) {
	config := DefaultConfig()
	config.EnableAlerts = false
	checker := &mockHealthChecker{}

	manager := NewFailoverManager(config, checker)

	ctx := context.Background()

	// 执行重新平衡
	err := manager.Rebalance(ctx)
	if err != nil {
		t.Fatalf("Rebalance() error = %v", err)
	}

	// 验证统计
	stats := manager.GetStats()
	if stats.TotalRebalances != 1 {
		t.Errorf("TotalRebalances = %d, want 1", stats.TotalRebalances)
	}
}

func TestFailureTracker(t *testing.T) {
	tracker := newFailureTracker()
	nodeID := "test-node"

	// 记录失败
	count := tracker.recordFailure(nodeID)
	if count != 1 {
		t.Errorf("Failure count = %d, want 1", count)
	}

	count = tracker.recordFailure(nodeID)
	if count != 2 {
		t.Errorf("Failure count = %d, want 2", count)
	}

	// 记录成功
	count = tracker.recordSuccess(nodeID)
	if count != 1 {
		t.Errorf("Recovery count = %d, want 1", count)
	}

	// 验证失败计数被重置
	failureCount := tracker.getFailureCount(nodeID)
	if failureCount != 0 {
		t.Errorf("Failure count = %d, want 0 after success", failureCount)
	}

	// 重置
	tracker.reset(nodeID)
	count = tracker.getRecoveryCount(nodeID)
	if count != 0 {
		t.Errorf("Recovery count = %d, want 0 after reset", count)
	}
}
