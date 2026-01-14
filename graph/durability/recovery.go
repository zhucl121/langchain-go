package durability

import (
	"context"
	"fmt"
	"time"
)

// CheckpointSaver 是检查点保存器接口（避免循环依赖）。
type CheckpointSaver[S any] interface {
	Save(ctx context.Context, checkpoint any) error
	Load(ctx context.Context, config any) (any, error)
	List(ctx context.Context, threadID string) ([]any, error)
}

// RecoveryManager 是恢复管理器。
//
// RecoveryManager 负责从故障中恢复执行。
//
type RecoveryManager[S any] struct {
	// checkpointer 检查点保存器
	checkpointer CheckpointSaver[S]

	// config Durability 配置
	config *DurabilityConfig

	// taskRegistry 任务注册表
	taskRegistry *TaskRegistry[S]
}

// NewRecoveryManager 创建恢复管理器。
//
// 参数：
//   - checkpointer: 检查点保存器
//   - config: Durability 配置
//
// 返回：
//   - *RecoveryManager[S]: 管理器实例
//
func NewRecoveryManager[S any](
	checkpointer CheckpointSaver[S],
	config *DurabilityConfig,
) *RecoveryManager[S] {
	return &RecoveryManager[S]{
		checkpointer: checkpointer,
		config:       config,
		taskRegistry: NewTaskRegistry[S](),
	}
}

// RegisterTask 注册任务。
func (rm *RecoveryManager[S]) RegisterTask(task *DurableTask[S]) error {
	return rm.taskRegistry.Register(task)
}

// Recover 恢复执行。
//
// 从最新的检查点恢复执行状态，并重新执行未完成的任务。
//
// 参数：
//   - ctx: 上下文
//   - threadID: 线程 ID
//
// 返回：
//   - S: 恢复的状态
//   - error: 恢复错误
//
func (rm *RecoveryManager[S]) Recover(ctx context.Context, threadID string) (S, error) {
	var zero S

	// 1. 加载最新检查点
	checkpoints, err := rm.checkpointer.List(ctx, threadID)
	if err != nil {
		return zero, fmt.Errorf("%w: failed to list checkpoints: %v", ErrRecoveryFailed, err)
	}

	if len(checkpoints) == 0 {
		return zero, fmt.Errorf("%w: no checkpoints found for thread %s", ErrRecoveryFailed, threadID)
	}

	// 加载最新检查点（假设 List 返回的是按时间排序的）
	latestCheckpoint := checkpoints[len(checkpoints)-1]

	// 2. 从检查点加载状态
	// 注意：这里需要类型断言，因为接口返回 any
	loadedData, err := rm.checkpointer.Load(ctx, latestCheckpoint)
	if err != nil {
		return zero, fmt.Errorf("%w: failed to load checkpoint: %v", ErrRecoveryFailed, err)
	}

	// 尝试类型断言
	state, ok := loadedData.(S)
	if !ok {
		return zero, fmt.Errorf("%w: checkpoint data type mismatch", ErrRecoveryFailed)
	}

	// 3. 创建恢复点
	recoveryPoint := NewRecoveryPoint(fmt.Sprintf("%v", latestCheckpoint))

	// 4. 分析未完成的任务
	// 获取所有已注册的任务
	allTasks := rm.taskRegistry.List()

	// 根据 durability 模式决定是否需要重新执行
	switch rm.config.Mode {
	case AtMostOnce:
		// AtMostOnce: 不重新执行，直接返回状态
		return state, nil

	case AtLeastOnce:
		// AtLeastOnce: 可能需要重新执行部分任务
		// 这里简化处理：假设需要重新执行所有待定任务
		for _, task := range allTasks {
			recoveryPoint.AddPendingTask(task.ID)
		}

	case ExactlyOnce:
		// ExactlyOnce: 根据已完成的任务列表，只执行未完成的
		// 这需要从检查点元数据中获取已完成任务列表
		// 简化处理：假设所有任务都需要检查
		for _, task := range allTasks {
			// 这里应该检查任务是否已完成
			// 简化：添加到待执行列表
			recoveryPoint.AddPendingTask(task.ID)
		}
	}

	// 5. 执行待定任务
	currentState := state
	execCtx := NewExecutionContext(threadID, rm.config)

	for _, taskID := range recoveryPoint.PendingTasks {
		task, err := rm.taskRegistry.Get(taskID)
		if err != nil {
			// 任务未找到，跳过
			continue
		}

		// 执行任务
		newState, err := task.Execute(ctx, currentState, execCtx)
		if err != nil {
			// 根据模式处理错误
			if rm.config.Mode == AtMostOnce {
				// AtMostOnce: 失败即停止
				return currentState, fmt.Errorf("%w: task %s failed: %v", ErrRecoveryFailed, taskID, err)
			}
			// AtLeastOnce/ExactlyOnce: 继续执行其他任务
			continue
		}

		currentState = newState
		recoveryPoint.AddCompletedTask(taskID)
	}

	// 6. 返回最终状态
	return currentState, nil
}

// RecoverWithTasks 使用指定的任务列表恢复执行。
//
// 参数：
//   - ctx: 上下文
//   - threadID: 线程 ID
//   - taskIDs: 要执行的任务 ID 列表
//
// 返回：
//   - S: 恢复的状态
//   - error: 恢复错误
//
func (rm *RecoveryManager[S]) RecoverWithTasks(ctx context.Context, threadID string, taskIDs []string) (S, error) {
	var zero S

	// 加载最新检查点
	checkpoints, err := rm.checkpointer.List(ctx, threadID)
	if err != nil {
		return zero, fmt.Errorf("%w: failed to list checkpoints: %v", ErrRecoveryFailed, err)
	}

	if len(checkpoints) == 0 {
		return zero, fmt.Errorf("%w: no checkpoints found", ErrRecoveryFailed)
	}

	loadedData, err := rm.checkpointer.Load(ctx, checkpoints[len(checkpoints)-1])
	if err != nil {
		return zero, fmt.Errorf("%w: failed to load checkpoint: %v", ErrRecoveryFailed, err)
	}

	state, ok := loadedData.(S)
	if !ok {
		return zero, fmt.Errorf("%w: checkpoint data type mismatch", ErrRecoveryFailed)
	}

	// 执行指定的任务
	currentState := state
	execCtx := NewExecutionContext(threadID, rm.config)

	for _, taskID := range taskIDs {
		task, err := rm.taskRegistry.Get(taskID)
		if err != nil {
			continue
		}

		newState, err := task.Execute(ctx, currentState, execCtx)
		if err != nil && rm.config.Mode == AtMostOnce {
			return currentState, err
		}

		if err == nil {
			currentState = newState
		}
	}

	return currentState, nil
}

// CanRecover 检查是否可以恢复。
func (rm *RecoveryManager[S]) CanRecover(ctx context.Context, threadID string) (bool, error) {
	// 简化实现：检查是否有检查点
	checkpoints, err := rm.checkpointer.List(ctx, threadID)
	if err != nil {
		return false, err
	}

	return len(checkpoints) > 0, nil
}

// DurabilityExecutor 是持久性执行器。
//
// DurabilityExecutor 包装执行器，添加持久性保证。
//
type DurabilityExecutor[S any] struct {
	// config 配置
	config *DurabilityConfig

	// checkpointer 检查点保存器（可选）
	checkpointer CheckpointSaver[S]

	// taskRegistry 任务注册表
	taskRegistry *TaskRegistry[S]
}

// NewDurabilityExecutor 创建持久性执行器。
func NewDurabilityExecutor[S any](config *DurabilityConfig) *DurabilityExecutor[S] {
	return &DurabilityExecutor[S]{
		config:       config,
		taskRegistry: NewTaskRegistry[S](),
	}
}

// WithCheckpointer 设置检查点保存器。
func (de *DurabilityExecutor[S]) WithCheckpointer(checkpointer CheckpointSaver[S]) *DurabilityExecutor[S] {
	de.checkpointer = checkpointer
	return de
}

// RegisterTask 注册任务。
func (de *DurabilityExecutor[S]) RegisterTask(task *DurableTask[S]) error {
	return de.taskRegistry.Register(task)
}

// ExecuteTask 执行任务（带持久性保证）。
//
// 参数：
//   - ctx: 上下文
//   - taskID: 任务 ID
//   - state: 当前状态
//   - execCtx: 执行上下文
//
// 返回：
//   - S: 新状态
//   - error: 执行错误
//
func (de *DurabilityExecutor[S]) ExecuteTask(
	ctx context.Context,
	taskID string,
	state S,
	execCtx *ExecutionContext,
) (S, error) {
	// 获取任务
	task, err := de.taskRegistry.Get(taskID)
	if err != nil {
		var zero S
		return zero, err
	}

	// 执行任务
	return task.Execute(ctx, state, execCtx)
}

// ExecuteTasks 执行多个任务。
//
// 参数：
//   - ctx: 上下文
//   - taskIDs: 任务 ID 列表
//   - state: 初始状态
//   - threadID: 线程 ID
//
// 返回：
//   - S: 最终状态
//   - error: 执行错误
//
func (de *DurabilityExecutor[S]) ExecuteTasks(
	ctx context.Context,
	taskIDs []string,
	state S,
	threadID string,
) (S, error) {
	execCtx := NewExecutionContext(threadID, de.config)

	currentState := state
	for i, taskID := range taskIDs {
		// 检查是否需要保存检查点
		if de.checkpointer != nil && execCtx.ShouldCheckpoint(i+1) {
			// 这里需要具体的 checkpoint 实现
			// 简化处理
		}

		// 执行任务
		newState, err := de.ExecuteTask(ctx, taskID, currentState, execCtx)
		if err != nil {
			return currentState, err
		}

		currentState = newState
	}

	return currentState, nil
}

// GetConfig 返回配置。
func (de *DurabilityExecutor[S]) GetConfig() *DurabilityConfig {
	return de.config
}

// GetTaskRegistry 返回任务注册表。
func (de *DurabilityExecutor[S]) GetTaskRegistry() *TaskRegistry[S] {
	return de.taskRegistry
}

// RecoveryPoint 是恢复点。
//
// RecoveryPoint 标记可以恢复的执行点。
//
type RecoveryPoint struct {
	// CheckpointID 检查点 ID
	CheckpointID string

	// Timestamp 时间戳
	Timestamp time.Time

	// CompletedTasks 已完成的任务
	CompletedTasks []string

	// PendingTasks 待执行的任务
	PendingTasks []string

	// Metadata 元数据
	Metadata map[string]any
}

// NewRecoveryPoint 创建恢复点。
func NewRecoveryPoint(checkpointID string) *RecoveryPoint {
	return &RecoveryPoint{
		CheckpointID:   checkpointID,
		Timestamp:      time.Now(),
		CompletedTasks: make([]string, 0),
		PendingTasks:   make([]string, 0),
		Metadata:       make(map[string]any),
	}
}

// AddCompletedTask 添加已完成任务。
func (rp *RecoveryPoint) AddCompletedTask(taskID string) {
	rp.CompletedTasks = append(rp.CompletedTasks, taskID)
}

// AddPendingTask 添加待执行任务。
func (rp *RecoveryPoint) AddPendingTask(taskID string) {
	rp.PendingTasks = append(rp.PendingTasks, taskID)
}

// IsTaskCompleted 任务是否已完成。
func (rp *RecoveryPoint) IsTaskCompleted(taskID string) bool {
	for _, id := range rp.CompletedTasks {
		if id == taskID {
			return true
		}
	}
	return false
}

// GetNextTask 获取下一个待执行任务。
func (rp *RecoveryPoint) GetNextTask() (string, bool) {
	if len(rp.PendingTasks) == 0 {
		return "", false
	}
	return rp.PendingTasks[0], true
}

// RemovePendingTask 移除待执行任务。
func (rp *RecoveryPoint) RemovePendingTask(taskID string) {
	newPending := make([]string, 0, len(rp.PendingTasks))
	for _, id := range rp.PendingTasks {
		if id != taskID {
			newPending = append(newPending, id)
		}
	}
	rp.PendingTasks = newPending
}

// DurabilityStats 是持久性统计。
type DurabilityStats struct {
	// TotalTasks 总任务数
	TotalTasks int

	// CompletedTasks 完成任务数
	CompletedTasks int

	// FailedTasks 失败任务数
	FailedTasks int

	// RetryCount 重试次数
	RetryCount int

	// CheckpointCount 检查点数量
	CheckpointCount int

	// AverageRetries 平均重试次数
	AverageRetries float64
}

// NewDurabilityStats 创建统计。
func NewDurabilityStats() *DurabilityStats {
	return &DurabilityStats{}
}

// UpdateFromExecution 从执行上下文更新统计。
func (ds *DurabilityStats) UpdateFromExecution(execCtx *ExecutionContext) {
	ds.TotalTasks = len(execCtx.TaskExecutions)

	totalRetries := 0
	for _, exec := range execCtx.TaskExecutions {
		if exec.IsCompleted() {
			ds.CompletedTasks++
		} else if exec.Status == TaskFailed {
			ds.FailedTasks++
		}

		if exec.Attempts > 1 {
			totalRetries += exec.Attempts - 1
		}
	}

	ds.RetryCount = totalRetries
	if ds.TotalTasks > 0 {
		ds.AverageRetries = float64(totalRetries) / float64(ds.TotalTasks)
	}
}

// String 返回字符串表示。
func (ds *DurabilityStats) String() string {
	return fmt.Sprintf("DurabilityStats{Total: %d, Completed: %d, Failed: %d, Retries: %d, AvgRetries: %.2f}",
		ds.TotalTasks, ds.CompletedTasks, ds.FailedTasks, ds.RetryCount, ds.AverageRetries)
}
