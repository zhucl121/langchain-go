package hitl

import (
	"context"
	"errors"
	"sync"
	"time"
)

// 错误定义
var (
	ErrInterrupted      = errors.New("hitl: execution interrupted")
	ErrNoInterrupt      = errors.New("hitl: no active interrupt")
	ErrInvalidInput     = errors.New("hitl: invalid input")
	ErrTimeout          = errors.New("hitl: timeout waiting for input")
	ErrAlreadyResolved  = errors.New("hitl: interrupt already resolved")
)

// InterruptType 是中断类型。
type InterruptType string

const (
	// InterruptBefore 在节点执行前中断
	InterruptBefore InterruptType = "before"

	// InterruptAfter 在节点执行后中断
	InterruptAfter InterruptType = "after"

	// InterruptOnError 在错误发生时中断
	InterruptOnError InterruptType = "on_error"

	// InterruptManual 手动中断
	InterruptManual InterruptType = "manual"
)

// InterruptReason 是中断原因。
type InterruptReason string

const (
	// ReasonApprovalRequired 需要审批
	ReasonApprovalRequired InterruptReason = "approval_required"

	// ReasonInputRequired 需要输入
	ReasonInputRequired InterruptReason = "input_required"

	// ReasonErrorOccurred 发生错误
	ReasonErrorOccurred InterruptReason = "error_occurred"

	// ReasonManual 手动触发
	ReasonManual InterruptReason = "manual"
)

// InterruptPoint 是中断点定义。
//
// InterruptPoint 定义在哪里以及如何中断执行。
//
type InterruptPoint struct {
	// NodeName 节点名称
	NodeName string

	// Type 中断类型
	Type InterruptType

	// Condition 中断条件（可选）
	Condition func(state any) bool

	// Message 中断消息
	Message string

	// Metadata 元数据
	Metadata map[string]any
}

// NewInterruptPoint 创建中断点。
func NewInterruptPoint(nodeName string, interruptType InterruptType) *InterruptPoint {
	return &InterruptPoint{
		NodeName: nodeName,
		Type:     interruptType,
		Metadata: make(map[string]any),
	}
}

// WithCondition 设置中断条件。
func (ip *InterruptPoint) WithCondition(cond func(state any) bool) *InterruptPoint {
	ip.Condition = cond
	return ip
}

// WithMessage 设置中断消息。
func (ip *InterruptPoint) WithMessage(msg string) *InterruptPoint {
	ip.Message = msg
	return ip
}

// ShouldInterrupt 判断是否应该中断。
func (ip *InterruptPoint) ShouldInterrupt(state any) bool {
	if ip.Condition == nil {
		return true
	}
	return ip.Condition(state)
}

// Interrupt 是中断实例。
//
// Interrupt 表示一次具体的执行中断。
//
type Interrupt struct {
	// ID 中断 ID
	ID string

	// Point 中断点
	Point *InterruptPoint

	// Reason 中断原因
	Reason InterruptReason

	// State 中断时的状态（序列化）
	State any

	// Timestamp 中断时间
	Timestamp time.Time

	// ThreadID 所属线程
	ThreadID string

	// CheckpointID 关联的检查点
	CheckpointID string

	// Resolved 是否已解决
	Resolved bool

	// Resolution 解决方案
	Resolution *InterruptResolution

	// Metadata 元数据
	Metadata map[string]any

	mu sync.RWMutex
}

// NewInterrupt 创建中断。
func NewInterrupt(id string, point *InterruptPoint, threadID string) *Interrupt {
	return &Interrupt{
		ID:        id,
		Point:     point,
		Timestamp: time.Now(),
		ThreadID:  threadID,
		Resolved:  false,
		Metadata:  make(map[string]any),
	}
}

// GetID 返回 ID。
func (i *Interrupt) GetID() string {
	return i.ID
}

// GetNodeName 返回节点名称。
func (i *Interrupt) GetNodeName() string {
	if i.Point != nil {
		return i.Point.NodeName
	}
	return ""
}

// GetMessage 返回消息。
func (i *Interrupt) GetMessage() string {
	if i.Point != nil {
		return i.Point.Message
	}
	return ""
}

// IsResolved 是否已解决。
func (i *Interrupt) IsResolved() bool {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.Resolved
}

// Resolve 解决中断。
func (i *Interrupt) Resolve(resolution *InterruptResolution) error {
	i.mu.Lock()
	defer i.mu.Unlock()

	if i.Resolved {
		return ErrAlreadyResolved
	}

	i.Resolved = true
	i.Resolution = resolution
	return nil
}

// GetResolution 获取解决方案。
func (i *Interrupt) GetResolution() *InterruptResolution {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.Resolution
}

// InterruptResolution 是中断解决方案。
type InterruptResolution struct {
	// Action 操作
	Action ResolutionAction

	// Input 输入数据
	Input any

	// ModifiedState 修改后的状态
	ModifiedState any

	// Message 消息
	Message string

	// Timestamp 解决时间
	Timestamp time.Time
}

// ResolutionAction 是解决操作。
type ResolutionAction string

const (
	// ActionContinue 继续执行
	ActionContinue ResolutionAction = "continue"

	// ActionModify 修改状态后继续
	ActionModify ResolutionAction = "modify"

	// ActionSkip 跳过当前节点
	ActionSkip ResolutionAction = "skip"

	// ActionAbort 中止执行
	ActionAbort ResolutionAction = "abort"

	// ActionRetry 重试当前节点
	ActionRetry ResolutionAction = "retry"
)

// NewResolution 创建解决方案。
func NewResolution(action ResolutionAction) *InterruptResolution {
	return &InterruptResolution{
		Action:    action,
		Timestamp: time.Now(),
	}
}

// WithInput 设置输入。
func (ir *InterruptResolution) WithInput(input any) *InterruptResolution {
	ir.Input = input
	return ir
}

// WithModifiedState 设置修改后的状态。
func (ir *InterruptResolution) WithModifiedState(state any) *InterruptResolution {
	ir.ModifiedState = state
	return ir
}

// WithMessage 设置消息。
func (ir *InterruptResolution) WithMessage(msg string) *InterruptResolution {
	ir.Message = msg
	return ir
}

// InterruptManager 是中断管理器。
//
// InterruptManager 管理执行过程中的所有中断。
//
type InterruptManager struct {
	// points 中断点
	points map[string][]*InterruptPoint // key: nodeName

	// activeInterrupts 活跃的中断
	activeInterrupts map[string]*Interrupt // key: interruptID

	// history 中断历史
	history []*Interrupt

	mu sync.RWMutex
}

// NewInterruptManager 创建中断管理器。
func NewInterruptManager() *InterruptManager {
	return &InterruptManager{
		points:           make(map[string][]*InterruptPoint),
		activeInterrupts: make(map[string]*Interrupt),
		history:          make([]*Interrupt, 0),
	}
}

// AddInterruptPoint 添加中断点。
func (im *InterruptManager) AddInterruptPoint(point *InterruptPoint) {
	im.mu.Lock()
	defer im.mu.Unlock()

	if _, exists := im.points[point.NodeName]; !exists {
		im.points[point.NodeName] = make([]*InterruptPoint, 0)
	}

	im.points[point.NodeName] = append(im.points[point.NodeName], point)
}

// RemoveInterruptPoint 移除中断点。
func (im *InterruptManager) RemoveInterruptPoint(nodeName string, interruptType InterruptType) {
	im.mu.Lock()
	defer im.mu.Unlock()

	if points, exists := im.points[nodeName]; exists {
		newPoints := make([]*InterruptPoint, 0)
		for _, p := range points {
			if p.Type != interruptType {
				newPoints = append(newPoints, p)
			}
		}
		im.points[nodeName] = newPoints
	}
}

// GetInterruptPoints 获取节点的中断点。
func (im *InterruptManager) GetInterruptPoints(nodeName string, interruptType InterruptType) []*InterruptPoint {
	im.mu.RLock()
	defer im.mu.RUnlock()

	points, exists := im.points[nodeName]
	if !exists {
		return nil
	}

	result := make([]*InterruptPoint, 0)
	for _, p := range points {
		if p.Type == interruptType {
			result = append(result, p)
		}
	}

	return result
}

// ShouldInterrupt 判断是否应该中断。
func (im *InterruptManager) ShouldInterrupt(nodeName string, interruptType InterruptType, state any) bool {
	points := im.GetInterruptPoints(nodeName, interruptType)

	for _, point := range points {
		if point.ShouldInterrupt(state) {
			return true
		}
	}

	return false
}

// CreateInterrupt 创建中断。
func (im *InterruptManager) CreateInterrupt(
	interruptID string,
	point *InterruptPoint,
	threadID string,
	state any,
) *Interrupt {
	im.mu.Lock()
	defer im.mu.Unlock()

	interrupt := NewInterrupt(interruptID, point, threadID)
	interrupt.State = state

	im.activeInterrupts[interruptID] = interrupt
	im.history = append(im.history, interrupt)

	return interrupt
}

// GetInterrupt 获取中断。
func (im *InterruptManager) GetInterrupt(interruptID string) (*Interrupt, error) {
	im.mu.RLock()
	defer im.mu.RUnlock()

	interrupt, exists := im.activeInterrupts[interruptID]
	if !exists {
		return nil, ErrNoInterrupt
	}

	return interrupt, nil
}

// ResolveInterrupt 解决中断。
func (im *InterruptManager) ResolveInterrupt(interruptID string, resolution *InterruptResolution) error {
	interrupt, err := im.GetInterrupt(interruptID)
	if err != nil {
		return err
	}

	if err := interrupt.Resolve(resolution); err != nil {
		return err
	}

	// 从活跃列表中移除
	im.mu.Lock()
	delete(im.activeInterrupts, interruptID)
	im.mu.Unlock()

	return nil
}

// GetActiveInterrupts 获取活跃中断。
func (im *InterruptManager) GetActiveInterrupts() []*Interrupt {
	im.mu.RLock()
	defer im.mu.RUnlock()

	result := make([]*Interrupt, 0, len(im.activeInterrupts))
	for _, interrupt := range im.activeInterrupts {
		result = append(result, interrupt)
	}

	return result
}

// GetHistory 获取中断历史。
func (im *InterruptManager) GetHistory() []*Interrupt {
	im.mu.RLock()
	defer im.mu.RUnlock()

	result := make([]*Interrupt, len(im.history))
	copy(result, im.history)
	return result
}

// Clear 清空管理器。
func (im *InterruptManager) Clear() {
	im.mu.Lock()
	defer im.mu.Unlock()

	im.activeInterrupts = make(map[string]*Interrupt)
}

// WaitForResolution 等待中断解决。
//
// 参数：
//   - ctx: 上下文
//   - interruptID: 中断 ID
//
// 返回：
//   - *InterruptResolution: 解决方案
//   - error: 错误
//
func (im *InterruptManager) WaitForResolution(ctx context.Context, interruptID string) (*InterruptResolution, error) {
	interrupt, err := im.GetInterrupt(interruptID)
	if err != nil {
		return nil, err
	}

	// 轮询等待解决
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			if interrupt.IsResolved() {
				return interrupt.GetResolution(), nil
			}
		}
	}
}
