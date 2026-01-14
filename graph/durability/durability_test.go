package durability

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

// TestState 测试用状态
type TestState struct {
	Counter int
	Message string
}

// TestDurabilityMode 测试持久性模式
func TestDurabilityMode(t *testing.T) {
	modes := []DurabilityMode{AtMostOnce, AtLeastOnce, ExactlyOnce}

	for _, mode := range modes {
		if err := mode.Validate(); err != nil {
			t.Errorf("mode %s validation failed: %v", mode, err)
		}
	}

	invalidMode := DurabilityMode("invalid")
	if !errors.Is(invalidMode.Validate(), ErrInvalidMode) {
		t.Error("expected ErrInvalidMode for invalid mode")
	}
}

// TestDurabilityMode_Features 测试模式特性
func TestDurabilityMode_Features(t *testing.T) {
	// AtMostOnce
	if AtMostOnce.NeedsCheckpoint() {
		t.Error("AtMostOnce should not need checkpoint")
	}
	if AtMostOnce.NeedsDeduplication() {
		t.Error("AtMostOnce should not need deduplication")
	}

	// AtLeastOnce
	if !AtLeastOnce.NeedsCheckpoint() {
		t.Error("AtLeastOnce should need checkpoint")
	}
	if AtLeastOnce.NeedsDeduplication() {
		t.Error("AtLeastOnce should not need deduplication")
	}

	// ExactlyOnce
	if !ExactlyOnce.NeedsCheckpoint() {
		t.Error("ExactlyOnce should need checkpoint")
	}
	if !ExactlyOnce.NeedsDeduplication() {
		t.Error("ExactlyOnce should need deduplication")
	}
}

// TestNewDurabilityConfig 测试创建配置
func TestNewDurabilityConfig(t *testing.T) {
	config := NewDurabilityConfig(AtLeastOnce)

	if config.Mode != AtLeastOnce {
		t.Errorf("expected mode AtLeastOnce, got %s", config.Mode)
	}

	if config.CheckpointInterval != 1 {
		t.Errorf("expected interval 1, got %d", config.CheckpointInterval)
	}

	if config.MaxRetries != 3 {
		t.Errorf("expected max retries 3, got %d", config.MaxRetries)
	}
}

// TestDurabilityConfig_WithMethods 测试配置链式调用
func TestDurabilityConfig_WithMethods(t *testing.T) {
	config := NewDurabilityConfig(ExactlyOnce).
		WithCheckpointInterval(5).
		WithMaxRetries(10).
		WithRetryDelay(2 * time.Second).
		WithTimeoutPerTask(10 * time.Minute)

	if config.CheckpointInterval != 5 {
		t.Error("interval not set")
	}

	if config.MaxRetries != 10 {
		t.Error("max retries not set")
	}

	if config.RetryDelay != 2*time.Second {
		t.Error("retry delay not set")
	}
}

// TestTaskExecution 测试任务执行记录
func TestTaskExecution(t *testing.T) {
	exec := NewTaskExecution("task-1")

	if exec.Status != TaskPending {
		t.Errorf("expected status Pending, got %s", exec.Status)
	}

	// 标记运行中
	exec.MarkRunning()
	if exec.Status != TaskRunning {
		t.Error("expected status Running")
	}
	if exec.Attempts != 1 {
		t.Errorf("expected 1 attempt, got %d", exec.Attempts)
	}

	// 标记完成
	exec.MarkCompleted()
	if exec.Status != TaskCompleted {
		t.Error("expected status Completed")
	}
	if !exec.IsCompleted() {
		t.Error("IsCompleted should return true")
	}
}

// TestTaskExecution_Retry 测试重试
func TestTaskExecution_Retry(t *testing.T) {
	exec := NewTaskExecution("task-1")

	// 第一次尝试
	exec.MarkRunning()
	exec.MarkFailed(errors.New("error 1"))

	if exec.Attempts != 1 {
		t.Errorf("expected 1 attempt, got %d", exec.Attempts)
	}

	// 检查是否可以重试
	if !exec.CanRetry(3) {
		t.Error("should be able to retry")
	}

	// 重试
	exec.MarkRetrying()
	exec.MarkRunning()
	exec.MarkCompleted()

	if exec.Attempts != 2 {
		t.Errorf("expected 2 attempts, got %d", exec.Attempts)
	}

	// 完成后不应再重试
	if exec.CanRetry(3) {
		t.Error("should not be able to retry after completion")
	}
}

// TestExecutionContext 测试执行上下文
func TestExecutionContext(t *testing.T) {
	config := NewDurabilityConfig(ExactlyOnce)
	execCtx := NewExecutionContext("thread-1", config)

	if execCtx.ThreadID != "thread-1" {
		t.Errorf("expected thread-1, got %s", execCtx.ThreadID)
	}

	// 获取任务执行
	exec := execCtx.GetTaskExecution("task-1")
	if exec.TaskID != "task-1" {
		t.Error("task ID mismatch")
	}

	// 再次获取应该返回同一个
	exec2 := execCtx.GetTaskExecution("task-1")
	if exec != exec2 {
		t.Error("should return same execution")
	}
}

// TestExecutionContext_ShouldCheckpoint 测试检查点判断
func TestExecutionContext_ShouldCheckpoint(t *testing.T) {
	config := NewDurabilityConfig(AtLeastOnce).WithCheckpointInterval(2)
	execCtx := NewExecutionContext("thread-1", config)

	// 步数 2 的倍数应该保存
	if !execCtx.ShouldCheckpoint(2) {
		t.Error("should checkpoint at step 2")
	}

	if execCtx.ShouldCheckpoint(3) {
		t.Error("should not checkpoint at step 3")
	}

	// 间隔为 0 表示每步都保存
	config2 := NewDurabilityConfig(AtLeastOnce).WithCheckpointInterval(0)
	execCtx2 := NewExecutionContext("thread-2", config2)

	if !execCtx2.ShouldCheckpoint(1) {
		t.Error("should checkpoint every step")
	}
}

// TestRetryPolicy 测试重试策略
func TestRetryPolicy(t *testing.T) {
	policy := NewRetryPolicy(3)

	if policy.MaxRetries != 3 {
		t.Errorf("expected max retries 3, got %d", policy.MaxRetries)
	}

	// 测试延迟计算
	delay1 := policy.GetDelay(1)
	if delay1 != time.Second {
		t.Errorf("expected 1s for first attempt, got %v", delay1)
	}

	delay2 := policy.GetDelay(2)
	if delay2 != 2*time.Second {
		t.Errorf("expected 2s for second attempt, got %v", delay2)
	}

	delay3 := policy.GetDelay(3)
	if delay3 != 4*time.Second {
		t.Errorf("expected 4s for third attempt, got %v", delay3)
	}
}

// TestDurableTask_Basic 测试持久化任务基本功能
func TestDurableTask_Basic(t *testing.T) {
	called := false
	task := NewDurableTask("task-1", func(ctx context.Context, state TestState) (TestState, error) {
		called = true
		state.Counter++
		return state, nil
	})

	if task.ID != "task-1" {
		t.Errorf("expected ID 'task-1', got %s", task.ID)
	}

	config := NewDurabilityConfig(AtLeastOnce)
	execCtx := NewExecutionContext("thread-1", config)

	state := TestState{Counter: 0}
	newState, err := task.Execute(context.Background(), state, execCtx)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !called {
		t.Error("task function was not called")
	}

	if newState.Counter != 1 {
		t.Errorf("expected Counter 1, got %d", newState.Counter)
	}
}

// TestDurableTask_Retry 测试重试
func TestDurableTask_Retry(t *testing.T) {
	attempts := 0
	task := NewDurableTask("task-1", func(ctx context.Context, state TestState) (TestState, error) {
		attempts++
		if attempts < 3 {
			return state, errors.New("temporary error")
		}
		state.Counter++
		return state, nil
	}).WithRetryPolicy(NewRetryPolicy(5))

	config := NewDurabilityConfig(AtLeastOnce)
	execCtx := NewExecutionContext("thread-1", config)

	state := TestState{Counter: 0}
	newState, err := task.Execute(context.Background(), state, execCtx)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}

	if newState.Counter != 1 {
		t.Errorf("expected Counter 1, got %d", newState.Counter)
	}
}

// TestDurableTask_ExactlyOnce 测试 ExactlyOnce 模式
func TestDurableTask_ExactlyOnce(t *testing.T) {
	calls := 0
	task := NewDurableTask("task-1", func(ctx context.Context, state TestState) (TestState, error) {
		calls++
		state.Counter++
		return state, nil
	}).WithIdempotent(true)

	config := NewDurabilityConfig(ExactlyOnce)
	execCtx := NewExecutionContext("thread-1", config)

	state := TestState{Counter: 0}

	// 第一次执行
	newState, err := task.Execute(context.Background(), state, execCtx)
	if err != nil {
		t.Fatalf("First execute failed: %v", err)
	}

	if newState.Counter != 1 {
		t.Errorf("expected Counter 1, got %d", newState.Counter)
	}

	// 第二次执行（应该跳过）
	newState2, err := task.Execute(context.Background(), newState, execCtx)
	if err != nil {
		t.Fatalf("Second execute failed: %v", err)
	}

	// Counter 不应该再增加
	if newState2.Counter != 1 {
		t.Errorf("expected Counter still 1, got %d", newState2.Counter)
	}

	if calls != 1 {
		t.Errorf("expected function called once, got %d", calls)
	}
}

// TestTaskWrapper 测试任务包装器
func TestTaskWrapper(t *testing.T) {
	wrapper := NewTaskWrapper[TestState]()

	task := wrapper.Wrap("task-1", func(ctx context.Context, state TestState) (TestState, error) {
		state.Counter++
		return state, nil
	})

	if task.ID != "task-1" {
		t.Error("task ID mismatch")
	}

	if task.IsIdempotent {
		t.Error("default should not be idempotent")
	}
}

// TestTaskWrapper_Idempotent 测试幂等包装
func TestTaskWrapper_Idempotent(t *testing.T) {
	wrapper := NewTaskWrapper[TestState]()

	task := wrapper.WrapIdempotent("task-1", func(ctx context.Context, state TestState) (TestState, error) {
		state.Counter++
		return state, nil
	})

	if !task.IsIdempotent {
		t.Error("should be idempotent")
	}
}

// TestTaskRegistry 测试任务注册表
func TestTaskRegistry(t *testing.T) {
	registry := NewTaskRegistry[TestState]()

	task1 := NewDurableTask("task-1", func(ctx context.Context, state TestState) (TestState, error) {
		return state, nil
	})

	// 注册
	err := registry.Register(task1)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// 获取
	retrieved, err := registry.Get("task-1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if retrieved.ID != "task-1" {
		t.Error("retrieved task mismatch")
	}

	// 重复注册应该失败
	err = registry.Register(task1)
	if err == nil {
		t.Error("expected error for duplicate registration")
	}
}

// TestTaskRegistry_List 测试列出任务
func TestTaskRegistry_List(t *testing.T) {
	registry := NewTaskRegistry[TestState]()

	for i := 1; i <= 3; i++ {
		task := NewDurableTask(fmt.Sprintf("task-%d", i), func(ctx context.Context, state TestState) (TestState, error) {
			return state, nil
		})
		registry.Register(task)
	}

	tasks := registry.List()
	if len(tasks) != 3 {
		t.Errorf("expected 3 tasks, got %d", len(tasks))
	}

	if registry.Size() != 3 {
		t.Errorf("expected size 3, got %d", registry.Size())
	}
}

// TestRecoveryPoint 测试恢复点
func TestRecoveryPoint(t *testing.T) {
	rp := NewRecoveryPoint("cp-1")

	rp.AddCompletedTask("task-1")
	rp.AddCompletedTask("task-2")
	rp.AddPendingTask("task-3")
	rp.AddPendingTask("task-4")

	// 检查完成状态
	if !rp.IsTaskCompleted("task-1") {
		t.Error("task-1 should be completed")
	}

	if rp.IsTaskCompleted("task-3") {
		t.Error("task-3 should not be completed")
	}

	// 获取下一个任务
	nextTask, ok := rp.GetNextTask()
	if !ok {
		t.Fatal("should have next task")
	}

	if nextTask != "task-3" {
		t.Errorf("expected next task 'task-3', got %s", nextTask)
	}

	// 移除任务
	rp.RemovePendingTask("task-3")
	nextTask2, _ := rp.GetNextTask()
	if nextTask2 != "task-4" {
		t.Errorf("expected next task 'task-4', got %s", nextTask2)
	}
}

// TestDurabilityStats 测试统计
func TestDurabilityStats(t *testing.T) {
	stats := NewDurabilityStats()

	config := NewDurabilityConfig(AtLeastOnce)
	execCtx := NewExecutionContext("thread-1", config)

	// 模拟执行
	exec1 := execCtx.GetTaskExecution("task-1")
	exec1.MarkRunning()
	exec1.MarkCompleted()

	exec2 := execCtx.GetTaskExecution("task-2")
	exec2.MarkRunning()
	exec2.MarkFailed(errors.New("error"))
	exec2.MarkRetrying()
	exec2.MarkRunning()
	exec2.MarkCompleted()

	// 更新统计
	stats.UpdateFromExecution(execCtx)

	if stats.TotalTasks != 2 {
		t.Errorf("expected 2 total tasks, got %d", stats.TotalTasks)
	}

	if stats.CompletedTasks != 2 {
		t.Errorf("expected 2 completed tasks, got %d", stats.CompletedTasks)
	}

	if stats.RetryCount != 1 {
		t.Errorf("expected 1 retry, got %d", stats.RetryCount)
	}
}

// TestDurabilityExecutor 测试持久性执行器
func TestDurabilityExecutor(t *testing.T) {
	config := NewDurabilityConfig(AtLeastOnce)
	executor := NewDurabilityExecutor[TestState](config)

	task := NewDurableTask("task-1", func(ctx context.Context, state TestState) (TestState, error) {
		state.Counter++
		return state, nil
	})

	err := executor.RegisterTask(task)
	if err != nil {
		t.Fatalf("RegisterTask failed: %v", err)
	}

	execCtx := NewExecutionContext("thread-1", config)
	state := TestState{Counter: 0}

	newState, err := executor.ExecuteTask(context.Background(), "task-1", state, execCtx)
	if err != nil {
		t.Fatalf("ExecuteTask failed: %v", err)
	}

	if newState.Counter != 1 {
		t.Errorf("expected Counter 1, got %d", newState.Counter)
	}
}


// TestRecoveryManager_Recover 测试恢复功能
func TestRecoveryManager_Recover(t *testing.T) {
	// 创建一个简单的内存检查点保存器
	checkpointer := &mockCheckpointSaver{
		checkpoints: make(map[string][]TestState),
	}

	config := &DurabilityConfig{
		Mode: AtLeastOnce,
	}

	manager := NewRecoveryManager[TestState](checkpointer, config)

	// 保存一个检查点
	checkpointer.saveCheckpoint("thread-1", TestState{Counter: 100, Message: "checkpoint"})

	// 注册一个任务
	task := NewDurableTask[TestState]("task-1", func(ctx context.Context, state TestState) (TestState, error) {
		state.Counter += 10
		return state, nil
	})

	manager.RegisterTask(task)

	// 恢复执行
	recovered, err := manager.Recover(context.Background(), "thread-1")
	if err != nil {
		t.Fatalf("Recover failed: %v", err)
	}

	if recovered.Counter != 110 {
		t.Errorf("expected Counter=110, got %d", recovered.Counter)
	}
}

// mockCheckpointSaver 模拟检查点保存器
type mockCheckpointSaver struct {
	checkpoints map[string][]TestState
}

func (m *mockCheckpointSaver) Save(ctx context.Context, checkpoint any) error {
	return nil
}

func (m *mockCheckpointSaver) Load(ctx context.Context, config any) (any, error) {
	return config, nil
}

func (m *mockCheckpointSaver) List(ctx context.Context, threadID string) ([]any, error) {
	states, exists := m.checkpoints[threadID]
	if !exists {
		return []any{}, nil
	}

	result := make([]any, len(states))
	for i, state := range states {
		result[i] = state
	}
	return result, nil
}

func (m *mockCheckpointSaver) saveCheckpoint(threadID string, state TestState) {
	if _, exists := m.checkpoints[threadID]; !exists {
		m.checkpoints[threadID] = make([]TestState, 0)
	}
	m.checkpoints[threadID] = append(m.checkpoints[threadID], state)
}

