package executor

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

// TestState 测试用状态
type TestState struct {
	Counter int
	Message string
	Done    bool
}

// TestNewExecutionContext 测试创建执行上下文
func TestNewExecutionContext(t *testing.T) {
	state := TestState{Counter: 0}
	ctx := NewExecutionContext(state)

	if ctx == nil {
		t.Fatal("NewExecutionContext returned nil")
	}

	if ctx.GetState().Counter != 0 {
		t.Error("state not initialized correctly")
	}

	if ctx.GetCurrentStep() != 0 {
		t.Error("step should be 0")
	}
}

// TestExecutionContext_UpdateState 测试更新状态
func TestExecutionContext_UpdateState(t *testing.T) {
	ctx := NewExecutionContext(TestState{Counter: 0})

	newState := TestState{Counter: 5}
	ctx.UpdateState(newState)

	if ctx.GetState().Counter != 5 {
		t.Errorf("expected counter 5, got %d", ctx.GetState().Counter)
	}
}

// TestExecutionContext_IncrementStep 测试增加步数
func TestExecutionContext_IncrementStep(t *testing.T) {
	ctx := NewExecutionContext(TestState{})

	if err := ctx.IncrementStep(); err != nil {
		t.Fatalf("IncrementStep failed: %v", err)
	}

	if ctx.GetCurrentStep() != 1 {
		t.Errorf("expected step 1, got %d", ctx.GetCurrentStep())
	}
}

// TestExecutionContext_MaxSteps 测试最大步数限制
func TestExecutionContext_MaxSteps(t *testing.T) {
	ctx := NewExecutionContext(TestState{}).WithMaxSteps(2)

	// 第一步
	if err := ctx.IncrementStep(); err != nil {
		t.Fatalf("step 1 failed: %v", err)
	}

	// 第二步
	if err := ctx.IncrementStep(); err != nil {
		t.Fatalf("step 2 failed: %v", err)
	}

	// 第三步应该失败
	err := ctx.IncrementStep()
	if !errors.Is(err, ErrMaxStepsExceeded) {
		t.Errorf("expected ErrMaxStepsExceeded, got %v", err)
	}
}

// TestExecutionContext_InterruptPoint 测试中断点
func TestExecutionContext_InterruptPoint(t *testing.T) {
	ctx := NewExecutionContext(TestState{})

	ctx.AddInterruptPoint("node1")

	if !ctx.ShouldInterrupt("node1") {
		t.Error("expected interrupt at node1")
	}

	if ctx.ShouldInterrupt("node2") {
		t.Error("should not interrupt at node2")
	}
}

// TestExecutionContext_Interrupt 测试中断
func TestExecutionContext_Interrupt(t *testing.T) {
	ctx := NewExecutionContext(TestState{})

	if ctx.IsInterrupted() {
		t.Error("should not be interrupted initially")
	}

	ctx.Interrupt("node1")

	if !ctx.IsInterrupted() {
		t.Error("should be interrupted after Interrupt()")
	}

	if ctx.GetInterruptNode() != "node1" {
		t.Errorf("expected interrupt node 'node1', got %s", ctx.GetInterruptNode())
	}
}

// TestExecutionContext_History 测试执行历史
func TestExecutionContext_History(t *testing.T) {
	ctx := NewExecutionContext(TestState{})

	history1 := ExecutionHistory{
		NodeName:  "node1",
		Timestamp: time.Now(),
		Duration:  100 * time.Millisecond,
	}

	ctx.AddHistory(history1)

	history := ctx.GetHistory()
	if len(history) != 1 {
		t.Errorf("expected 1 history entry, got %d", len(history))
	}

	if history[0].NodeName != "node1" {
		t.Errorf("expected node1, got %s", history[0].NodeName)
	}
}

// TestExecutionContext_Callback 测试事件回调
func TestExecutionContext_Callback(t *testing.T) {
	ctx := NewExecutionContext(TestState{})

	callbackCalled := false
	ctx.AddCallback(func(event Event) {
		if event.Type == EventStateUpdate {
			callbackCalled = true
		}
	})

	ctx.UpdateState(TestState{Counter: 1})

	// 给异步回调一点时间
	time.Sleep(50 * time.Millisecond)

	if !callbackCalled {
		t.Error("callback was not called")
	}
}

// TestExecutionContext_Clone 测试克隆
func TestExecutionContext_Clone(t *testing.T) {
	original := NewExecutionContext(TestState{Counter: 5})
	original.AddInterruptPoint("node1")
	original.IncrementStep()

	clone := original.Clone()

	if clone.GetState().Counter != 5 {
		t.Error("cloned state mismatch")
	}

	if clone.GetCurrentStep() != 1 {
		t.Error("cloned step mismatch")
	}

	if !clone.ShouldInterrupt("node1") {
		t.Error("cloned interrupt points mismatch")
	}

	// 修改克隆不应影响原始
	clone.UpdateState(TestState{Counter: 10})
	if original.GetState().Counter != 5 {
		t.Error("modifying clone affected original")
	}
}

// TestNewScheduler 测试创建调度器
func TestNewScheduler(t *testing.T) {
	scheduler := NewScheduler[TestState]()

	if scheduler == nil {
		t.Fatal("NewScheduler returned nil")
	}

	if scheduler.GetStrategy() != StrategySequential {
		t.Error("default strategy should be sequential")
	}

	if scheduler.GetMaxConcurrent() != 1 {
		t.Error("default max concurrent should be 1")
	}
}

// TestScheduler_WithStrategy 测试设置策略
func TestScheduler_WithStrategy(t *testing.T) {
	scheduler := NewScheduler[TestState]()

	result := scheduler.WithStrategy(StrategyParallel)

	if result != scheduler {
		t.Error("WithStrategy should return self")
	}

	if scheduler.GetStrategy() != StrategyParallel {
		t.Error("strategy not set")
	}
}

// TestScheduler_WithMaxConcurrent 测试设置并发数
func TestScheduler_WithMaxConcurrent(t *testing.T) {
	scheduler := NewScheduler[TestState]()

	scheduler.WithMaxConcurrent(5)

	if scheduler.GetMaxConcurrent() != 5 {
		t.Errorf("expected max concurrent 5, got %d", scheduler.GetMaxConcurrent())
	}
}

// MockNodeExecutor 模拟节点执行器
type MockNodeExecutor struct {
	executeFunc func(ctx context.Context, state any) (any, error)
}

func (m *MockNodeExecutor) Execute(ctx context.Context, state any) (any, error) {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, state)
	}
	return state, nil
}

// TestScheduler_ScheduleNode 测试调度单个节点
func TestScheduler_ScheduleNode(t *testing.T) {
	scheduler := NewScheduler[TestState]()

	executor := &MockNodeExecutor{
		executeFunc: func(ctx context.Context, state any) (any, error) {
			s := state.(TestState)
			s.Counter++
			return s, nil
		},
	}

	state := TestState{Counter: 0}
	newState, err := scheduler.ScheduleNode(context.Background(), "test", executor, state)

	if err != nil {
		t.Fatalf("ScheduleNode failed: %v", err)
	}

	if newState.Counter != 1 {
		t.Errorf("expected counter 1, got %d", newState.Counter)
	}
}

// TestScheduler_ScheduleNode_ContextCancelled 测试取消
func TestScheduler_ScheduleNode_ContextCancelled(t *testing.T) {
	scheduler := NewScheduler[TestState]()

	// 阻塞的执行器
	executor := &MockNodeExecutor{
		executeFunc: func(ctx context.Context, state any) (any, error) {
			<-ctx.Done()
			return nil, ctx.Err()
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // 立即取消

	_, err := scheduler.ScheduleNode(ctx, "test", executor, TestState{})

	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

// MockGraph 模拟编译后的图
type MockGraph struct {
	name       string
	entryPoint string
	adjacency  map[string][]string
}

func (m *MockGraph) GetName() string {
	return m.name
}

func (m *MockGraph) GetEntryPoint() string {
	return m.entryPoint
}

func (m *MockGraph) GetAdjacency() map[string][]string {
	return m.adjacency
}

// TestNewExecutor 测试创建执行器
func TestNewExecutor(t *testing.T) {
	executor := NewExecutor[TestState]()

	if executor == nil {
		t.Fatal("NewExecutor returned nil")
	}

	if executor.scheduler == nil {
		t.Error("scheduler should be initialized")
	}
}

// TestExecutor_RegisterNode 测试注册节点
func TestExecutor_RegisterNode(t *testing.T) {
	executor := NewExecutor[TestState]()

	nodeFunc := func(ctx context.Context, state TestState) (TestState, error) {
		state.Counter++
		return state, nil
	}

	result := executor.RegisterNode("test", nodeFunc)

	if result != executor {
		t.Error("RegisterNode should return self")
	}

	if len(executor.nodes) != 1 {
		t.Error("node not registered")
	}
}

// TestExecutor_Execute_Simple 测试简单执行
func TestExecutor_Execute_Simple(t *testing.T) {
	executor := NewExecutor[TestState]()

	// 注册节点
	executor.RegisterNode("node1", func(ctx context.Context, state TestState) (TestState, error) {
		state.Counter++
		return state, nil
	})

	executor.RegisterNode("node2", func(ctx context.Context, state TestState) (TestState, error) {
		state.Counter *= 2
		return state, nil
	})

	// 构建图
	graph := &MockGraph{
		name:       "test",
		entryPoint: "node1",
		adjacency: map[string][]string{
			"node1": {"node2"},
			"node2": {"__end__"},
		},
	}

	// 执行
	state := TestState{Counter: 0}
	result, err := executor.Execute(context.Background(), graph, state)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// (0 + 1) * 2 = 2
	if result.Counter != 2 {
		t.Errorf("expected counter 2, got %d", result.Counter)
	}
}

// TestExecutor_Execute_NoEntry 测试无入口点
func TestExecutor_Execute_NoEntry(t *testing.T) {
	executor := NewExecutor[TestState]()

	graph := &MockGraph{
		name:       "test",
		entryPoint: "",
		adjacency:  map[string][]string{},
	}

	_, err := executor.Execute(context.Background(), graph, TestState{})

	if err == nil {
		t.Fatal("expected error for no entry point")
	}
}

// TestExecutor_Execute_NodeNotFound 测试节点未找到
func TestExecutor_Execute_NodeNotFound(t *testing.T) {
	executor := NewExecutor[TestState]()

	graph := &MockGraph{
		name:       "test",
		entryPoint: "nonexistent",
		adjacency:  map[string][]string{},
	}

	_, err := executor.Execute(context.Background(), graph, TestState{})

	if !errors.Is(err, ErrNodeNotFound) {
		t.Errorf("expected ErrNodeNotFound, got %v", err)
	}
}

// TestExecutor_Execute_WithInterrupt 测试中断
func TestExecutor_Execute_WithInterrupt(t *testing.T) {
	executor := NewExecutor[TestState]()

	executor.RegisterNode("node1", func(ctx context.Context, state TestState) (TestState, error) {
		state.Counter++
		return state, nil
	})

	graph := &MockGraph{
		name:       "test",
		entryPoint: "node1",
		adjacency: map[string][]string{
			"node1": {"__end__"},
		},
	}

	execCtx := NewExecutionContext(TestState{Counter: 0})
	execCtx.AddInterruptPoint("node1")

	_, err := executor.ExecuteWithContext(context.Background(), graph, execCtx)

	if !errors.Is(err, ErrInterrupted) {
		t.Errorf("expected ErrInterrupted, got %v", err)
	}

	if !execCtx.IsInterrupted() {
		t.Error("context should be interrupted")
	}
}

// TestExecutor_ExecuteWithResult 测试返回详细结果
func TestExecutor_ExecuteWithResult(t *testing.T) {
	executor := NewExecutor[TestState]()

	executor.RegisterNode("node1", func(ctx context.Context, state TestState) (TestState, error) {
		state.Counter++
		return state, nil
	})

	graph := &MockGraph{
		name:       "test",
		entryPoint: "node1",
		adjacency: map[string][]string{
			"node1": {"__end__"},
		},
	}

	result := executor.ExecuteWithResult(context.Background(), graph, TestState{Counter: 0})

	if result.Error != nil {
		t.Fatalf("Execute failed: %v", result.Error)
	}

	if result.State.Counter != 1 {
		t.Errorf("expected counter 1, got %d", result.State.Counter)
	}

	if len(result.History) == 0 {
		t.Error("expected history to be recorded")
	}
}

// TestExecutor_Execute_MaxSteps 测试最大步数
func TestExecutor_Execute_MaxSteps(t *testing.T) {
	executor := NewExecutor[TestState]()

	executor.RegisterNode("node1", func(ctx context.Context, state TestState) (TestState, error) {
		state.Counter++
		return state, nil
	})

	// 创建循环图
	graph := &MockGraph{
		name:       "test",
		entryPoint: "node1",
		adjacency: map[string][]string{
			"node1": {"node1"}, // 自循环
		},
	}

	execCtx := NewExecutionContext(TestState{Counter: 0}).WithMaxSteps(5)

	_, err := executor.ExecuteWithContext(context.Background(), graph, execCtx)

	if !errors.Is(err, ErrMaxStepsExceeded) {
		t.Errorf("expected ErrMaxStepsExceeded, got %v", err)
	}
}


// TestScheduler_ParallelExecution 测试真正的并行执行
func TestScheduler_ParallelExecution(t *testing.T) {
	scheduler := NewScheduler[TestState]().
		WithStrategy(StrategyParallel).
		WithMaxConcurrent(3)

	executors := make(map[string]NodeExecutor)

	executors["node1"] = &mockExecutor{
		fn: func(ctx context.Context, state any) (any, error) {
			time.Sleep(50 * time.Millisecond)
			s := state.(TestState)
			s.Counter += 10
			s.Message = "node1"
			return s, nil
		},
	}

	executors["node2"] = &mockExecutor{
		fn: func(ctx context.Context, state any) (any, error) {
			time.Sleep(50 * time.Millisecond)
			s := state.(TestState)
			s.Counter += 20
			s.Message = "node2"
			return s, nil
		},
	}

	executors["node3"] = &mockExecutor{
		fn: func(ctx context.Context, state any) (any, error) {
			time.Sleep(50 * time.Millisecond)
			s := state.(TestState)
			s.Counter += 30
			s.Message = "node3"
			return s, nil
		},
	}

	initialState := TestState{Counter: 0}
	nodes := []string{"node1", "node2", "node3"}

	start := time.Now()
	results, err := scheduler.ScheduleNodes(context.Background(), nodes, executors, initialState)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("parallel execution failed: %v", err)
	}

	if len(results) != 3 {
		t.Errorf("expected 3 results, got %d", len(results))
	}

	if results[0].Counter != 10 {
		t.Errorf("node1 expected Counter=10, got %d", results[0].Counter)
	}
	if results[1].Counter != 20 {
		t.Errorf("node2 expected Counter=20, got %d", results[1].Counter)
	}
	if results[2].Counter != 30 {
		t.Errorf("node3 expected Counter=30, got %d", results[2].Counter)
	}

	if elapsed > 150*time.Millisecond {
		t.Errorf("parallel execution took too long: %v", elapsed)
	}

	t.Logf("Parallel execution completed in %v", elapsed)
}

// TestScheduler_ParallelError 测试并行执行中的错误处理
func TestScheduler_ParallelError(t *testing.T) {
	scheduler := NewScheduler[TestState]().
		WithStrategy(StrategyParallel).
		WithMaxConcurrent(2)

	executors := make(map[string]NodeExecutor)

	executors["node1"] = &mockExecutor{
		fn: func(ctx context.Context, state any) (any, error) {
			time.Sleep(20 * time.Millisecond)
			return state, nil
		},
	}

	executors["node2"] = &mockExecutor{
		fn: func(ctx context.Context, state any) (any, error) {
			time.Sleep(10 * time.Millisecond)
			return nil, fmt.Errorf("node2 error")
		},
	}

	executors["node3"] = &mockExecutor{
		fn: func(ctx context.Context, state any) (any, error) {
			time.Sleep(30 * time.Millisecond)
			return state, nil
		},
	}

	initialState := TestState{Counter: 0}
	nodes := []string{"node1", "node2", "node3"}

	_, err := scheduler.ScheduleNodes(context.Background(), nodes, executors, initialState)
	if err == nil {
		t.Error("expected error from parallel execution")
	}

	if !strings.Contains(err.Error(), "node2 error") {
		t.Errorf("expected node2 error, got: %v", err)
	}
}

// TestStateMerger 测试状态合并器
func TestStateMerger(t *testing.T) {
	merger := &DefaultStateMerger[TestState]{}

	states := []TestState{
		{Counter: 10, Message: "first"},
		{Counter: 20, Message: "second"},
		{Counter: 30, Message: "third"},
	}

	merged, err := merger.Merge(states)
	if err != nil {
		t.Fatalf("merge failed: %v", err)
	}

	if merged.Counter != 30 || merged.Message != "third" {
		t.Errorf("expected last state, got Counter=%d, Message=%s", merged.Counter, merged.Message)
	}
}

// mockExecutor 是测试用的模拟执行器
type mockExecutor struct {
	fn func(ctx context.Context, state any) (any, error)
}

func (m *mockExecutor) Execute(ctx context.Context, state any) (any, error) {
	return m.fn(ctx, state)
}

