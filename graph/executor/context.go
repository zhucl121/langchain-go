package executor

import (
	"context"
	"errors"
	"sync"
	"time"
)

// 错误定义
var (
	ErrExecutionCancelled = errors.New("executor: execution cancelled")
	ErrExecutionTimeout   = errors.New("executor: execution timeout")
	ErrMaxStepsExceeded   = errors.New("executor: max steps exceeded")
	ErrNodeNotFound       = errors.New("executor: node not found")
	ErrNoNextNode         = errors.New("executor: no next node")
	ErrInterrupted        = errors.New("executor: execution interrupted")
)

// EventType 是事件类型。
type EventType string

const (
	// EventNodeStart 节点开始执行
	EventNodeStart EventType = "node_start"

	// EventNodeEnd 节点结束执行
	EventNodeEnd EventType = "node_end"

	// EventNodeError 节点执行错误
	EventNodeError EventType = "node_error"

	// EventStateUpdate 状态更新
	EventStateUpdate EventType = "state_update"

	// EventCheckpoint Checkpoint 创建
	EventCheckpoint EventType = "checkpoint"

	// EventInterrupt 执行中断
	EventInterrupt EventType = "interrupt"
)

// Event 是执行事件。
//
// Event 记录执行过程中的关键事件。
//
type Event struct {
	Type      EventType
	NodeName  string
	Timestamp time.Time
	Data      any
	Error     error
}

// ExecutionHistory 是执行历史记录。
type ExecutionHistory struct {
	NodeName  string
	Timestamp time.Time
	Duration  time.Duration
	Error     error
}

// ExecutionContext 是执行上下文。
//
// ExecutionContext 维护图执行过程中的运行时状态。
//
// 核心功能：
//   - 状态管理
//   - 执行历史
//   - 事件回调
//   - Checkpoint 支持
//
type ExecutionContext[S any] struct {
	// 状态
	state S

	// 执行控制
	maxSteps      int
	currentStep   int
	interruptAt   map[string]bool // 中断点
	interrupted   bool
	interruptNode string

	// 历史和事件
	history   []ExecutionHistory
	events    []Event
	callbacks []func(Event)

	// Checkpoint（预留接口）
	checkpointer interface{} // 将在 M38-M42 实现

	// 并发保护
	mu sync.RWMutex
}

// NewExecutionContext 创建执行上下文。
//
// 参数：
//   - initialState: 初始状态
//
// 返回：
//   - *ExecutionContext[S]: 执行上下文实例
//
func NewExecutionContext[S any](initialState S) *ExecutionContext[S] {
	return &ExecutionContext[S]{
		state:       initialState,
		maxSteps:    1000, // 默认最大步数
		interruptAt: make(map[string]bool),
		history:     make([]ExecutionHistory, 0),
		events:      make([]Event, 0),
		callbacks:   make([]func(Event), 0),
	}
}

// GetState 获取当前状态。
func (ec *ExecutionContext[S]) GetState() S {
	ec.mu.RLock()
	defer ec.mu.RUnlock()
	return ec.state
}

// UpdateState 更新状态。
func (ec *ExecutionContext[S]) UpdateState(newState S) {
	ec.mu.Lock()
	defer ec.mu.Unlock()

	ec.state = newState

	// 触发状态更新事件
	ec.emitEventLocked(Event{
		Type:      EventStateUpdate,
		Timestamp: time.Now(),
		Data:      newState,
	})
}

// GetCurrentStep 获取当前步数。
func (ec *ExecutionContext[S]) GetCurrentStep() int {
	ec.mu.RLock()
	defer ec.mu.RUnlock()
	return ec.currentStep
}

// IncrementStep 增加步数。
//
// 返回：
//   - error: 如果超过最大步数，返回错误
//
func (ec *ExecutionContext[S]) IncrementStep() error {
	ec.mu.Lock()
	defer ec.mu.Unlock()

	ec.currentStep++

	if ec.maxSteps > 0 && ec.currentStep > ec.maxSteps {
		return ErrMaxStepsExceeded
	}

	return nil
}

// WithMaxSteps 设置最大步数。
//
// 参数：
//   - maxSteps: 最大步数（0 表示无限制）
//
// 返回：
//   - *ExecutionContext[S]: 返回自身，支持链式调用
//
func (ec *ExecutionContext[S]) WithMaxSteps(maxSteps int) *ExecutionContext[S] {
	ec.mu.Lock()
	defer ec.mu.Unlock()

	ec.maxSteps = maxSteps
	return ec
}

// AddInterruptPoint 添加中断点。
//
// 参数：
//   - nodeName: 节点名称
//
// 返回：
//   - *ExecutionContext[S]: 返回自身，支持链式调用
//
func (ec *ExecutionContext[S]) AddInterruptPoint(nodeName string) *ExecutionContext[S] {
	ec.mu.Lock()
	defer ec.mu.Unlock()

	ec.interruptAt[nodeName] = true
	return ec
}

// ShouldInterrupt 检查是否应该在给定节点中断。
//
// 参数：
//   - nodeName: 节点名称
//
// 返回：
//   - bool: 是否应该中断
//
func (ec *ExecutionContext[S]) ShouldInterrupt(nodeName string) bool {
	ec.mu.RLock()
	defer ec.mu.RUnlock()

	return ec.interruptAt[nodeName]
}

// Interrupt 标记为中断。
//
// 参数：
//   - nodeName: 中断所在节点
//
func (ec *ExecutionContext[S]) Interrupt(nodeName string) {
	ec.mu.Lock()
	defer ec.mu.Unlock()

	ec.interrupted = true
	ec.interruptNode = nodeName

	// 触发中断事件
	ec.emitEventLocked(Event{
		Type:      EventInterrupt,
		NodeName:  nodeName,
		Timestamp: time.Now(),
	})
}

// IsInterrupted 检查是否已中断。
func (ec *ExecutionContext[S]) IsInterrupted() bool {
	ec.mu.RLock()
	defer ec.mu.RUnlock()
	return ec.interrupted
}

// GetInterruptNode 获取中断所在节点。
func (ec *ExecutionContext[S]) GetInterruptNode() string {
	ec.mu.RLock()
	defer ec.mu.RUnlock()
	return ec.interruptNode
}

// AddHistory 添加执行历史。
//
// 参数：
//   - history: 历史记录
//
func (ec *ExecutionContext[S]) AddHistory(history ExecutionHistory) {
	ec.mu.Lock()
	defer ec.mu.Unlock()

	ec.history = append(ec.history, history)
}

// GetHistory 获取执行历史。
func (ec *ExecutionContext[S]) GetHistory() []ExecutionHistory {
	ec.mu.RLock()
	defer ec.mu.RUnlock()

	result := make([]ExecutionHistory, len(ec.history))
	copy(result, ec.history)
	return result
}

// AddCallback 添加事件回调。
//
// 参数：
//   - callback: 回调函数
//
// 返回：
//   - *ExecutionContext[S]: 返回自身，支持链式调用
//
func (ec *ExecutionContext[S]) AddCallback(callback func(Event)) *ExecutionContext[S] {
	ec.mu.Lock()
	defer ec.mu.Unlock()

	ec.callbacks = append(ec.callbacks, callback)
	return ec
}

// EmitEvent 触发事件。
//
// 参数：
//   - event: 事件
//
func (ec *ExecutionContext[S]) EmitEvent(event Event) {
	ec.mu.Lock()
	defer ec.mu.Unlock()

	ec.emitEventLocked(event)
}

// emitEventLocked 触发事件（内部方法，需要持有锁）。
func (ec *ExecutionContext[S]) emitEventLocked(event Event) {
	ec.events = append(ec.events, event)

	// 调用所有回调
	for _, callback := range ec.callbacks {
		go callback(event) // 异步调用
	}
}

// GetEvents 获取所有事件。
func (ec *ExecutionContext[S]) GetEvents() []Event {
	ec.mu.RLock()
	defer ec.mu.RUnlock()

	result := make([]Event, len(ec.events))
	copy(result, ec.events)
	return result
}

// WithCheckpointer 设置 Checkpointer（预留）。
//
// 参数：
//   - checkpointer: Checkpointer 实例
//
// 返回：
//   - *ExecutionContext[S]: 返回自身，支持链式调用
//
func (ec *ExecutionContext[S]) WithCheckpointer(checkpointer interface{}) *ExecutionContext[S] {
	ec.mu.Lock()
	defer ec.mu.Unlock()

	ec.checkpointer = checkpointer
	return ec
}

// GetCheckpointer 获取 Checkpointer。
func (ec *ExecutionContext[S]) GetCheckpointer() interface{} {
	ec.mu.RLock()
	defer ec.mu.RUnlock()
	return ec.checkpointer
}

// CheckContext 检查上下文状态。
//
// 参数：
//   - ctx: context.Context
//
// 返回：
//   - error: 如果上下文已取消，返回错误
//
func (ec *ExecutionContext[S]) CheckContext(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

// Clone 克隆执行上下文。
//
// 返回：
//   - *ExecutionContext[S]: 克隆的上下文
//
func (ec *ExecutionContext[S]) Clone() *ExecutionContext[S] {
	ec.mu.RLock()
	defer ec.mu.RUnlock()

	// 复制中断点
	interruptAt := make(map[string]bool)
	for k, v := range ec.interruptAt {
		interruptAt[k] = v
	}

	clone := &ExecutionContext[S]{
		state:         ec.state,
		maxSteps:      ec.maxSteps,
		currentStep:   ec.currentStep,
		interruptAt:   interruptAt,
		interrupted:   ec.interrupted,
		interruptNode: ec.interruptNode,
		history:       make([]ExecutionHistory, len(ec.history)),
		events:        make([]Event, len(ec.events)),
		callbacks:     make([]func(Event), len(ec.callbacks)),
		checkpointer:  ec.checkpointer,
	}

	copy(clone.history, ec.history)
	copy(clone.events, ec.events)
	copy(clone.callbacks, ec.callbacks)

	return clone
}
