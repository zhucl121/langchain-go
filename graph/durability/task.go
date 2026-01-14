package durability

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// DurableTask 是持久化任务。
//
// DurableTask 包装普通任务，添加持久性保证。
//
type DurableTask[S any] struct {
	// ID 任务唯一标识
	ID string

	// Func 任务函数
	Func TaskFunc[S]

	// RetryPolicy 重试策略
	RetryPolicy *RetryPolicy

	// IsIdempotent 是否幂等
	IsIdempotent bool

	// Metadata 元数据
	Metadata map[string]any

	mu sync.RWMutex
}

// NewDurableTask 创建持久化任务。
//
// 参数：
//   - id: 任务 ID
//   - fn: 任务函数
//
// 返回：
//   - *DurableTask[S]: 任务实例
//
func NewDurableTask[S any](id string, fn TaskFunc[S]) *DurableTask[S] {
	return &DurableTask[S]{
		ID:           id,
		Func:         fn,
		RetryPolicy:  NewRetryPolicy(3),
		IsIdempotent: false,
		Metadata:     make(map[string]any),
	}
}

// WithRetryPolicy 设置重试策略。
func (dt *DurableTask[S]) WithRetryPolicy(policy *RetryPolicy) *DurableTask[S] {
	dt.RetryPolicy = policy
	return dt
}

// WithIdempotent 设置是否幂等。
func (dt *DurableTask[S]) WithIdempotent(idempotent bool) *DurableTask[S] {
	dt.IsIdempotent = idempotent
	return dt
}

// WithMetadata 设置元数据。
func (dt *DurableTask[S]) WithMetadata(key string, value any) *DurableTask[S] {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	dt.Metadata[key] = value
	return dt
}

// Execute 执行任务（带持久性保证）。
//
// 参数：
//   - ctx: 上下文
//   - state: 当前状态
//   - execCtx: 执行上下文
//
// 返回：
//   - S: 新状态
//   - error: 执行错误
//
func (dt *DurableTask[S]) Execute(
	ctx context.Context,
	state S,
	execCtx *ExecutionContext,
) (S, error) {
	taskExec := execCtx.GetTaskExecution(dt.ID)

	// 检查是否已完成（ExactlyOnce 模式）
	if execCtx.Config.Mode == ExactlyOnce && taskExec.IsCompleted() {
		// 任务已完成，直接返回（避免重复执行）
		return state, nil
	}

	// 执行任务（带重试）
	return dt.executeWithRetry(ctx, state, taskExec)
}

// executeWithRetry 执行任务（带重试）。
func (dt *DurableTask[S]) executeWithRetry(
	ctx context.Context,
	state S,
	taskExec *TaskExecution,
) (S, error) {
	var lastErr error
	maxRetries := dt.RetryPolicy.MaxRetries

	for attempt := 1; attempt <= maxRetries+1; attempt++ {
		// 检查上下文
		select {
		case <-ctx.Done():
			var zero S
			return zero, ctx.Err()
		default:
		}

		// 标记为运行中
		taskExec.MarkRunning()

		// 执行任务
		newState, err := dt.Func(ctx, state)

		if err == nil {
			// 成功
			taskExec.MarkCompleted()
			return newState, nil
		}

		// 记录错误
		lastErr = err
		taskExec.MarkFailed(err)

		// 检查是否应该重试
		if attempt > maxRetries {
			break
		}

		if !dt.RetryPolicy.ShouldRetryError(err) {
			break
		}

		// 标记为重试中
		taskExec.MarkRetrying()

		// 等待重试延迟
		delay := dt.RetryPolicy.GetDelay(attempt)
		select {
		case <-ctx.Done():
			var zero S
			return zero, ctx.Err()
		case <-time.After(delay):
			// 继续重试
		}
	}

	// 所有重试都失败
	var zero S
	return zero, fmt.Errorf("%w: %v (after %d attempts)", ErrMaxRetriesReached, lastErr, taskExec.Attempts)
}

// GetID 返回任务 ID。
func (dt *DurableTask[S]) GetID() string {
	return dt.ID
}

// GetMetadata 返回元数据。
func (dt *DurableTask[S]) GetMetadata() map[string]any {
	dt.mu.RLock()
	defer dt.mu.RUnlock()

	result := make(map[string]any)
	for k, v := range dt.Metadata {
		result[k] = v
	}
	return result
}

// TaskWrapper 是任务包装器。
//
// TaskWrapper 提供便捷的任务包装功能。
//
type TaskWrapper[S any] struct {
	// DefaultRetryPolicy 默认重试策略
	DefaultRetryPolicy *RetryPolicy

	// DefaultIdempotent 默认是否幂等
	DefaultIdempotent bool
}

// NewTaskWrapper 创建任务包装器。
func NewTaskWrapper[S any]() *TaskWrapper[S] {
	return &TaskWrapper[S]{
		DefaultRetryPolicy: NewRetryPolicy(3),
		DefaultIdempotent:  false,
	}
}

// Wrap 包装任务。
//
// 参数：
//   - id: 任务 ID
//   - fn: 任务函数
//
// 返回：
//   - *DurableTask[S]: 持久化任务
//
func (tw *TaskWrapper[S]) Wrap(id string, fn TaskFunc[S]) *DurableTask[S] {
	return NewDurableTask(id, fn).
		WithRetryPolicy(tw.DefaultRetryPolicy).
		WithIdempotent(tw.DefaultIdempotent)
}

// WrapIdempotent 包装幂等任务。
func (tw *TaskWrapper[S]) WrapIdempotent(id string, fn TaskFunc[S]) *DurableTask[S] {
	return NewDurableTask(id, fn).
		WithRetryPolicy(tw.DefaultRetryPolicy).
		WithIdempotent(true)
}

// WrapWithRetry 包装任务（自定义重试）。
func (tw *TaskWrapper[S]) WrapWithRetry(
	id string,
	fn TaskFunc[S],
	maxRetries int,
) *DurableTask[S] {
	return NewDurableTask(id, fn).
		WithRetryPolicy(NewRetryPolicy(maxRetries))
}

// TaskRegistry 是任务注册表。
//
// TaskRegistry 管理所有持久化任务。
//
type TaskRegistry[S any] struct {
	tasks map[string]*DurableTask[S]
	mu    sync.RWMutex
}

// NewTaskRegistry 创建任务注册表。
func NewTaskRegistry[S any]() *TaskRegistry[S] {
	return &TaskRegistry[S]{
		tasks: make(map[string]*DurableTask[S]),
	}
}

// Register 注册任务。
func (tr *TaskRegistry[S]) Register(task *DurableTask[S]) error {
	tr.mu.Lock()
	defer tr.mu.Unlock()

	if _, exists := tr.tasks[task.ID]; exists {
		return fmt.Errorf("task %s already registered", task.ID)
	}

	tr.tasks[task.ID] = task
	return nil
}

// Get 获取任务。
func (tr *TaskRegistry[S]) Get(id string) (*DurableTask[S], error) {
	tr.mu.RLock()
	defer tr.mu.RUnlock()

	task, exists := tr.tasks[id]
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrTaskNotFound, id)
	}

	return task, nil
}

// List 列出所有任务。
func (tr *TaskRegistry[S]) List() []*DurableTask[S] {
	tr.mu.RLock()
	defer tr.mu.RUnlock()

	result := make([]*DurableTask[S], 0, len(tr.tasks))
	for _, task := range tr.tasks {
		result = append(result, task)
	}

	return result
}

// Unregister 注销任务。
func (tr *TaskRegistry[S]) Unregister(id string) error {
	tr.mu.Lock()
	defer tr.mu.Unlock()

	if _, exists := tr.tasks[id]; !exists {
		return fmt.Errorf("%w: %s", ErrTaskNotFound, id)
	}

	delete(tr.tasks, id)
	return nil
}

// Clear 清空注册表。
func (tr *TaskRegistry[S]) Clear() {
	tr.mu.Lock()
	defer tr.mu.Unlock()

	tr.tasks = make(map[string]*DurableTask[S])
}

// Size 返回任务数量。
func (tr *TaskRegistry[S]) Size() int {
	tr.mu.RLock()
	defer tr.mu.RUnlock()

	return len(tr.tasks)
}
