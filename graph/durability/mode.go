package durability

import (
	"context"
	"errors"
	"time"
)

// 错误定义
var (
	ErrTaskNotFound     = errors.New("durability: task not found")
	ErrInvalidMode      = errors.New("durability: invalid durability mode")
	ErrRecoveryFailed   = errors.New("durability: recovery failed")
	ErrTaskAlreadyDone  = errors.New("durability: task already completed")
	ErrMaxRetriesReached = errors.New("durability: max retries reached")
)

// DurabilityMode 是持久性模式。
//
// DurabilityMode 定义了任务执行的持久性保证级别。
//
type DurabilityMode string

const (
	// AtMostOnce 最多执行一次
	// - 不保证执行成功
	// - 失败后不重试
	// - 性能最高
	AtMostOnce DurabilityMode = "at_most_once"

	// AtLeastOnce 至少执行一次
	// - 保证执行成功
	// - 失败后会重试
	// - 可能重复执行
	AtLeastOnce DurabilityMode = "at_least_once"

	// ExactlyOnce 恰好执行一次
	// - 保证执行成功
	// - 保证不重复执行
	// - 需要幂等性支持
	// - 性能开销最大
	ExactlyOnce DurabilityMode = "exactly_once"
)

// Validate 验证持久性模式。
func (m DurabilityMode) Validate() error {
	switch m {
	case AtMostOnce, AtLeastOnce, ExactlyOnce:
		return nil
	default:
		return ErrInvalidMode
	}
}

// String 返回字符串表示。
func (m DurabilityMode) String() string {
	return string(m)
}

// NeedsCheckpoint 是否需要检查点。
func (m DurabilityMode) NeedsCheckpoint() bool {
	return m == AtLeastOnce || m == ExactlyOnce
}

// NeedsDeduplication 是否需要去重。
func (m DurabilityMode) NeedsDeduplication() bool {
	return m == ExactlyOnce
}

// DurabilityConfig 是持久性配置。
//
// DurabilityConfig 定义了执行的持久性策略。
//
type DurabilityConfig struct {
	// Mode 持久性模式
	Mode DurabilityMode

	// CheckpointInterval 检查点间隔（步数）
	// 0 表示每步都保存
	CheckpointInterval int

	// MaxRetries 最大重试次数
	// 0 表示不重试
	MaxRetries int

	// RetryDelay 重试延迟
	RetryDelay time.Duration

	// TimeoutPerTask 单个任务超时
	TimeoutPerTask time.Duration

	// EnableDeduplication 启用去重（仅 ExactlyOnce 模式）
	EnableDeduplication bool
}

// NewDurabilityConfig 创建持久性配置。
//
// 参数：
//   - mode: 持久性模式
//
// 返回：
//   - *DurabilityConfig: 配置实例
//
func NewDurabilityConfig(mode DurabilityMode) *DurabilityConfig {
	return &DurabilityConfig{
		Mode:                mode,
		CheckpointInterval:  1, // 默认每步保存
		MaxRetries:          3,
		RetryDelay:          time.Second,
		TimeoutPerTask:      5 * time.Minute,
		EnableDeduplication: mode == ExactlyOnce,
	}
}

// WithCheckpointInterval 设置检查点间隔。
func (c *DurabilityConfig) WithCheckpointInterval(interval int) *DurabilityConfig {
	c.CheckpointInterval = interval
	return c
}

// WithMaxRetries 设置最大重试次数。
func (c *DurabilityConfig) WithMaxRetries(retries int) *DurabilityConfig {
	c.MaxRetries = retries
	return c
}

// WithRetryDelay 设置重试延迟。
func (c *DurabilityConfig) WithRetryDelay(delay time.Duration) *DurabilityConfig {
	c.RetryDelay = delay
	return c
}

// WithTimeoutPerTask 设置任务超时。
func (c *DurabilityConfig) WithTimeoutPerTask(timeout time.Duration) *DurabilityConfig {
	c.TimeoutPerTask = timeout
	return c
}

// Validate 验证配置。
func (c *DurabilityConfig) Validate() error {
	if err := c.Mode.Validate(); err != nil {
		return err
	}

	if c.CheckpointInterval < 0 {
		return errors.New("checkpoint interval cannot be negative")
	}

	if c.MaxRetries < 0 {
		return errors.New("max retries cannot be negative")
	}

	return nil
}

// TaskStatus 是任务状态。
type TaskStatus string

const (
	// TaskPending 待执行
	TaskPending TaskStatus = "pending"

	// TaskRunning 执行中
	TaskRunning TaskStatus = "running"

	// TaskCompleted 已完成
	TaskCompleted TaskStatus = "completed"

	// TaskFailed 失败
	TaskFailed TaskStatus = "failed"

	// TaskRetrying 重试中
	TaskRetrying TaskStatus = "retrying"
)

// TaskExecution 是任务执行记录。
//
// TaskExecution 记录任务的执行历史。
//
type TaskExecution struct {
	// TaskID 任务 ID
	TaskID string

	// Status 状态
	Status TaskStatus

	// StartTime 开始时间
	StartTime time.Time

	// EndTime 结束时间
	EndTime time.Time

	// Attempts 尝试次数
	Attempts int

	// LastError 最后一次错误
	LastError error

	// Metadata 元数据
	Metadata map[string]any
}

// NewTaskExecution 创建任务执行记录。
func NewTaskExecution(taskID string) *TaskExecution {
	return &TaskExecution{
		TaskID:    taskID,
		Status:    TaskPending,
		StartTime: time.Now(),
		Attempts:  0,
		Metadata:  make(map[string]any),
	}
}

// MarkRunning 标记为运行中。
func (te *TaskExecution) MarkRunning() {
	te.Status = TaskRunning
	te.Attempts++
	te.StartTime = time.Now()
}

// MarkCompleted 标记为完成。
func (te *TaskExecution) MarkCompleted() {
	te.Status = TaskCompleted
	te.EndTime = time.Now()
}

// MarkFailed 标记为失败。
func (te *TaskExecution) MarkFailed(err error) {
	te.Status = TaskFailed
	te.EndTime = time.Now()
	te.LastError = err
}

// MarkRetrying 标记为重试中。
func (te *TaskExecution) MarkRetrying() {
	te.Status = TaskRetrying
	te.EndTime = time.Now()
}

// Duration 返回执行时长。
func (te *TaskExecution) Duration() time.Duration {
	if te.EndTime.IsZero() {
		return time.Since(te.StartTime)
	}
	return te.EndTime.Sub(te.StartTime)
}

// IsCompleted 是否已完成。
func (te *TaskExecution) IsCompleted() bool {
	return te.Status == TaskCompleted
}

// CanRetry 是否可以重试。
func (te *TaskExecution) CanRetry(maxRetries int) bool {
	return te.Attempts < maxRetries && te.Status != TaskCompleted
}

// ExecutionContext 是执行上下文（扩展）。
//
// ExecutionContext 包含 durability 相关的信息。
//
type ExecutionContext struct {
	// ThreadID 线程 ID
	ThreadID string

	// TaskExecutions 任务执行记录
	TaskExecutions map[string]*TaskExecution

	// LastCheckpoint 最后一个检查点 ID
	LastCheckpoint string

	// Config Durability 配置
	Config *DurabilityConfig
}

// NewExecutionContext 创建执行上下文。
func NewExecutionContext(threadID string, config *DurabilityConfig) *ExecutionContext {
	return &ExecutionContext{
		ThreadID:       threadID,
		TaskExecutions: make(map[string]*TaskExecution),
		Config:         config,
	}
}

// GetTaskExecution 获取任务执行记录。
func (ec *ExecutionContext) GetTaskExecution(taskID string) *TaskExecution {
	if exec, exists := ec.TaskExecutions[taskID]; exists {
		return exec
	}

	exec := NewTaskExecution(taskID)
	ec.TaskExecutions[taskID] = exec
	return exec
}

// ShouldCheckpoint 是否应该保存检查点。
func (ec *ExecutionContext) ShouldCheckpoint(step int) bool {
	if !ec.Config.Mode.NeedsCheckpoint() {
		return false
	}

	interval := ec.Config.CheckpointInterval
	if interval == 0 {
		return true // 每步都保存
	}

	return step%interval == 0
}

// IsTaskCompleted 任务是否已完成。
func (ec *ExecutionContext) IsTaskCompleted(taskID string) bool {
	if exec, exists := ec.TaskExecutions[taskID]; exists {
		return exec.IsCompleted()
	}
	return false
}

// TaskFunc 是任务函数类型。
type TaskFunc[S any] func(ctx context.Context, state S) (S, error)

// RetryPolicy 是重试策略。
type RetryPolicy struct {
	// MaxRetries 最大重试次数
	MaxRetries int

	// InitialDelay 初始延迟
	InitialDelay time.Duration

	// MaxDelay 最大延迟
	MaxDelay time.Duration

	// Multiplier 延迟倍增因子
	Multiplier float64

	// ShouldRetry 是否应该重试（可选）
	ShouldRetry func(error) bool
}

// NewRetryPolicy 创建重试策略。
func NewRetryPolicy(maxRetries int) *RetryPolicy {
	return &RetryPolicy{
		MaxRetries:   maxRetries,
		InitialDelay: time.Second,
		MaxDelay:     time.Minute,
		Multiplier:   2.0,
		ShouldRetry:  nil, // nil 表示所有错误都重试
	}
}

// GetDelay 获取重试延迟。
//
// 参数：
//   - attempt: 当前尝试次数（从 1 开始）
//
// 返回：
//   - time.Duration: 延迟时间
//
func (rp *RetryPolicy) GetDelay(attempt int) time.Duration {
	if attempt <= 1 {
		return rp.InitialDelay
	}

	delay := float64(rp.InitialDelay)
	for i := 1; i < attempt; i++ {
		delay *= rp.Multiplier
		if delay > float64(rp.MaxDelay) {
			return rp.MaxDelay
		}
	}

	return time.Duration(delay)
}

// ShouldRetryError 判断是否应该重试该错误。
func (rp *RetryPolicy) ShouldRetryError(err error) bool {
	if rp.ShouldRetry == nil {
		return true // 默认所有错误都重试
	}
	return rp.ShouldRetry(err)
}
