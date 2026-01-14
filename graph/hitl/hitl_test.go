package hitl

import (
	"context"
	"errors"
	"testing"
	"time"
)

// TestState 测试状态
type TestState struct {
	Counter int
	Message string
}

// TestInterruptPoint 测试中断点
func TestInterruptPoint(t *testing.T) {
	point := NewInterruptPoint("node1", InterruptBefore)

	if point.NodeName != "node1" {
		t.Errorf("expected NodeName 'node1', got %s", point.NodeName)
	}

	if point.Type != InterruptBefore {
		t.Errorf("expected Type InterruptBefore, got %s", point.Type)
	}

	// 无条件应该总是中断
	if !point.ShouldInterrupt(nil) {
		t.Error("should interrupt without condition")
	}
}

// TestInterruptPoint_WithCondition 测试条件中断
func TestInterruptPoint_WithCondition(t *testing.T) {
	point := NewInterruptPoint("node1", InterruptBefore).
		WithCondition(func(state any) bool {
			if s, ok := state.(TestState); ok {
				return s.Counter > 5
			}
			return false
		})

	// Counter > 5 才中断
	if point.ShouldInterrupt(TestState{Counter: 3}) {
		t.Error("should not interrupt when Counter = 3")
	}

	if !point.ShouldInterrupt(TestState{Counter: 10}) {
		t.Error("should interrupt when Counter = 10")
	}
}

// TestInterrupt 测试中断实例
func TestInterrupt(t *testing.T) {
	point := NewInterruptPoint("node1", InterruptBefore)
	interrupt := NewInterrupt("int-1", point, "thread-1")

	if interrupt.GetID() != "int-1" {
		t.Errorf("expected ID 'int-1', got %s", interrupt.GetID())
	}

	if interrupt.GetNodeName() != "node1" {
		t.Errorf("expected NodeName 'node1', got %s", interrupt.GetNodeName())
	}

	if interrupt.IsResolved() {
		t.Error("should not be resolved initially")
	}
}

// TestInterrupt_Resolve 测试解决中断
func TestInterrupt_Resolve(t *testing.T) {
	point := NewInterruptPoint("node1", InterruptBefore)
	interrupt := NewInterrupt("int-1", point, "thread-1")

	resolution := NewResolution(ActionContinue)
	err := interrupt.Resolve(resolution)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	if !interrupt.IsResolved() {
		t.Error("should be resolved")
	}

	// 重复解决应该失败
	err = interrupt.Resolve(resolution)
	if !errors.Is(err, ErrAlreadyResolved) {
		t.Error("expected ErrAlreadyResolved")
	}
}

// TestInterruptManager 测试中断管理器
func TestInterruptManager(t *testing.T) {
	manager := NewInterruptManager()

	// 添加中断点
	point := NewInterruptPoint("node1", InterruptBefore)
	manager.AddInterruptPoint(point)

	// 检查是否应该中断
	if !manager.ShouldInterrupt("node1", InterruptBefore, nil) {
		t.Error("should interrupt at node1")
	}

	if manager.ShouldInterrupt("node2", InterruptBefore, nil) {
		t.Error("should not interrupt at node2")
	}
}

// TestInterruptManager_CreateInterrupt 测试创建中断
func TestInterruptManager_CreateInterrupt(t *testing.T) {
	manager := NewInterruptManager()

	point := NewInterruptPoint("node1", InterruptBefore)
	state := TestState{Counter: 5}

	_ = manager.CreateInterrupt("int-1", point, "thread-1", state)

	// 获取中断
	retrieved, err := manager.GetInterrupt("int-1")
	if err != nil {
		t.Fatalf("GetInterrupt failed: %v", err)
	}

	if retrieved.GetID() != "int-1" {
		t.Error("retrieved interrupt mismatch")
	}
}

// TestInterruptManager_ResolveInterrupt 测试解决中断
func TestInterruptManager_ResolveInterrupt(t *testing.T) {
	manager := NewInterruptManager()

	point := NewInterruptPoint("node1", InterruptBefore)
	_ = manager.CreateInterrupt("int-1", point, "thread-1", nil)

	// 解决中断
	resolution := NewResolution(ActionContinue)
	err := manager.ResolveInterrupt("int-1", resolution)
	if err != nil {
		t.Fatalf("ResolveInterrupt failed: %v", err)
	}

	// 应该从活跃列表中移除
	_, err = manager.GetInterrupt("int-1")
	if !errors.Is(err, ErrNoInterrupt) {
		t.Error("expected ErrNoInterrupt after resolution")
	}

	// 但应该在历史中
	history := manager.GetHistory()
	if len(history) != 1 {
		t.Errorf("expected 1 history entry, got %d", len(history))
	}
}

// TestApprovalRequest 测试审批请求
func TestApprovalRequest(t *testing.T) {
	request := NewApprovalRequest("req-1", "Test Approval").
		WithOptions("approve", "reject")

	if request.ID != "req-1" {
		t.Errorf("expected ID 'req-1', got %s", request.ID)
	}

	if len(request.Options) != 2 {
		t.Errorf("expected 2 options, got %d", len(request.Options))
	}
}

// TestApprovalManager 测试审批管理器
func TestApprovalManager(t *testing.T) {
	manager := NewApprovalManager()

	request := NewApprovalRequest("req-1", "Test Approval")
	request.Timeout = time.Second

	// 在另一个 goroutine 中提交决策
	go func() {
		time.Sleep(100 * time.Millisecond)
		decision := NewApprovalDecision("req-1", ApprovalApproved)
		decision.Decision = "approve"
		manager.SubmitDecision(decision)
	}()

	// 请求审批（会等待）
	decision, err := manager.RequestApproval(context.Background(), request)
	if err != nil {
		t.Fatalf("RequestApproval failed: %v", err)
	}

	if decision.Status != ApprovalApproved {
		t.Errorf("expected Approved, got %s", decision.Status)
	}
}

// TestApprovalManager_Timeout 测试审批超时
func TestApprovalManager_Timeout(t *testing.T) {
	manager := NewApprovalManager()

	request := NewApprovalRequest("req-1", "Test Approval")
	request.Timeout = 100 * time.Millisecond

	// 不提交决策，等待超时
	decision, err := manager.RequestApproval(context.Background(), request)
	
	if !errors.Is(err, ErrTimeout) {
		t.Errorf("expected ErrTimeout, got %v", err)
	}

	if decision.Status != ApprovalTimeout {
		t.Errorf("expected Timeout status, got %s", decision.Status)
	}
}

// TestResumeManager 测试恢复管理器
func TestResumeManager(t *testing.T) {
	interruptManager := NewInterruptManager()
	resumeManager := NewResumeManager[TestState](interruptManager)

	// 创建中断
	point := NewInterruptPoint("node1", InterruptBefore)
	interrupt := interruptManager.CreateInterrupt("int-1", point, "thread-1", nil)

	if interrupt.IsResolved() {
		t.Error("should not be resolved initially")
	}

	// 恢复
	err := resumeManager.ResumeWithInput(context.Background(), "int-1", "user input")
	if err != nil {
		t.Fatalf("Resume failed: %v", err)
	}

	// 检查已解决
	if !interrupt.IsResolved() {
		t.Error("should be resolved after resume")
	}
}

// TestCallbackHandler 测试回调处理器
func TestCallbackHandler(t *testing.T) {
	interruptCalled := false
	resumeCalled := false

	handler := NewCallbackHandler().
		WithOnInterrupt(func(ctx context.Context, interrupt *Interrupt) error {
			interruptCalled = true
			return nil
		}).
		WithOnResume(func(ctx context.Context, interrupt *Interrupt, resolution *InterruptResolution) error {
			resumeCalled = true
			return nil
		})

	// 调用回调
	point := NewInterruptPoint("node1", InterruptBefore)
	interrupt := NewInterrupt("int-1", point, "thread-1")

	handler.OnInterrupt(context.Background(), interrupt)
	if !interruptCalled {
		t.Error("OnInterrupt callback not called")
	}

	resolution := NewResolution(ActionContinue)
	handler.OnResume(context.Background(), interrupt, resolution)
	if !resumeCalled {
		t.Error("OnResume callback not called")
	}
}

// TestResolutionAction 测试解决操作
func TestResolutionAction(t *testing.T) {
	actions := []ResolutionAction{
		ActionContinue,
		ActionModify,
		ActionSkip,
		ActionAbort,
		ActionRetry,
	}

	for _, action := range actions {
		resolution := NewResolution(action)
		if resolution.Action != action {
			t.Errorf("action mismatch for %s", action)
		}
	}
}

// TestInterruptManager_MultiplePoints 测试多个中断点
func TestInterruptManager_MultiplePoints(t *testing.T) {
	manager := NewInterruptManager()

	// 为同一个节点添加多个中断点
	manager.AddInterruptPoint(NewInterruptPoint("node1", InterruptBefore))
	manager.AddInterruptPoint(NewInterruptPoint("node1", InterruptAfter))

	// 检查 Before 中断
	if !manager.ShouldInterrupt("node1", InterruptBefore, nil) {
		t.Error("should interrupt Before")
	}

	// 检查 After 中断
	if !manager.ShouldInterrupt("node1", InterruptAfter, nil) {
		t.Error("should interrupt After")
	}

	// 移除 Before 中断
	manager.RemoveInterruptPoint("node1", InterruptBefore)

	// Before 应该不再中断
	if manager.ShouldInterrupt("node1", InterruptBefore, nil) {
		t.Error("should not interrupt Before after removal")
	}

	// After 仍应该中断
	if !manager.ShouldInterrupt("node1", InterruptAfter, nil) {
		t.Error("should still interrupt After")
	}
}
