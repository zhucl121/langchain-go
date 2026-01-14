package hitl

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// ApprovalStatus 是审批状态。
type ApprovalStatus string

const (
	// ApprovalPending 待审批
	ApprovalPending ApprovalStatus = "pending"

	// ApprovalApproved 已批准
	ApprovalApproved ApprovalStatus = "approved"

	// ApprovalRejected 已拒绝
	ApprovalRejected ApprovalStatus = "rejected"

	// ApprovalTimeout 超时
	ApprovalTimeout ApprovalStatus = "timeout"
)

// ApprovalRequest 是审批请求。
type ApprovalRequest struct {
	// ID 请求 ID
	ID string

	// Title 标题
	Title string

	// Description 描述
	Description string

	// Options 可选项
	Options []string

	// RequiredApprovers 需要审批的人数
	RequiredApprovers int

	// Timeout 超时时间
	Timeout time.Duration

	// Metadata 元数据
	Metadata map[string]any
}

// NewApprovalRequest 创建审批请求。
func NewApprovalRequest(id, title string) *ApprovalRequest {
	return &ApprovalRequest{
		ID:                id,
		Title:             title,
		RequiredApprovers: 1,
		Timeout:           5 * time.Minute,
		Metadata:          make(map[string]any),
	}
}

// WithOptions 设置选项。
func (ar *ApprovalRequest) WithOptions(options ...string) *ApprovalRequest {
	ar.Options = options
	return ar
}

// ApprovalDecision 是审批决策。
type ApprovalDecision struct {
	// RequestID 请求 ID
	RequestID string

	// Status 状态
	Status ApprovalStatus

	// Decision 决策（选择的选项）
	Decision string

	// Comment 评论
	Comment string

	// Approver 审批人
	Approver string

	// Timestamp 时间戳
	Timestamp time.Time
}

// NewApprovalDecision 创建审批决策。
func NewApprovalDecision(requestID string, status ApprovalStatus) *ApprovalDecision {
	return &ApprovalDecision{
		RequestID: requestID,
		Status:    status,
		Timestamp: time.Now(),
	}
}

// ApprovalManager 是审批管理器。
type ApprovalManager struct {
	pendingRequests map[string]*ApprovalRequest
	decisions       map[string]*ApprovalDecision

	mu sync.RWMutex
}

// NewApprovalManager 创建审批管理器。
func NewApprovalManager() *ApprovalManager {
	return &ApprovalManager{
		pendingRequests: make(map[string]*ApprovalRequest),
		decisions:       make(map[string]*ApprovalDecision),
	}
}

// RequestApproval 请求审批。
func (am *ApprovalManager) RequestApproval(ctx context.Context, request *ApprovalRequest) (*ApprovalDecision, error) {
	am.mu.Lock()
	am.pendingRequests[request.ID] = request
	am.mu.Unlock()

	// 等待决策或超时
	timeoutCtx, cancel := context.WithTimeout(ctx, request.Timeout)
	defer cancel()

	return am.WaitForDecision(timeoutCtx, request.ID)
}

// SubmitDecision 提交决策。
func (am *ApprovalManager) SubmitDecision(decision *ApprovalDecision) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	if _, exists := am.pendingRequests[decision.RequestID]; !exists {
		return fmt.Errorf("approval request %s not found", decision.RequestID)
	}

	am.decisions[decision.RequestID] = decision
	delete(am.pendingRequests, decision.RequestID)

	return nil
}

// WaitForDecision 等待决策。
func (am *ApprovalManager) WaitForDecision(ctx context.Context, requestID string) (*ApprovalDecision, error) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// 超时，创建超时决策
			timeoutDecision := NewApprovalDecision(requestID, ApprovalTimeout)
			am.mu.Lock()
			am.decisions[requestID] = timeoutDecision
			delete(am.pendingRequests, requestID)
			am.mu.Unlock()
			return timeoutDecision, ErrTimeout

		case <-ticker.C:
			am.mu.RLock()
			decision, exists := am.decisions[requestID]
			am.mu.RUnlock()

			if exists {
				return decision, nil
			}
		}
	}
}

// ResumeManager 是恢复管理器（M47）。
type ResumeManager[S any] struct {
	interruptManager *InterruptManager
	
	mu sync.RWMutex
}

// NewResumeManager 创建恢复管理器。
func NewResumeManager[S any](interruptManager *InterruptManager) *ResumeManager[S] {
	return &ResumeManager[S]{
		interruptManager: interruptManager,
	}
}

// Resume 恢复执行。
//
// 参数：
//   - ctx: 上下文
//   - interruptID: 中断 ID
//   - resolution: 解决方案
//
// 返回：
//   - error: 恢复错误
//
func (rm *ResumeManager[S]) Resume(
	ctx context.Context,
	interruptID string,
	resolution *InterruptResolution,
) error {
	// 解决中断
	return rm.interruptManager.ResolveInterrupt(interruptID, resolution)
}

// ResumeWithInput 使用输入恢复。
func (rm *ResumeManager[S]) ResumeWithInput(
	ctx context.Context,
	interruptID string,
	input any,
) error {
	resolution := NewResolution(ActionContinue).WithInput(input)
	return rm.Resume(ctx, interruptID, resolution)
}

// ResumeWithModifiedState 使用修改后的状态恢复。
func (rm *ResumeManager[S]) ResumeWithModifiedState(
	ctx context.Context,
	interruptID string,
	state S,
) error {
	resolution := NewResolution(ActionModify).WithModifiedState(state)
	return rm.Resume(ctx, interruptID, resolution)
}

// Abort 中止执行。
func (rm *ResumeManager[S]) Abort(ctx context.Context, interruptID string, message string) error {
	resolution := NewResolution(ActionAbort).WithMessage(message)
	return rm.Resume(ctx, interruptID, resolution)
}

// InterruptHandler 是中断处理器（M49）。
type InterruptHandler interface {
	// OnInterrupt 中断发生时调用
	OnInterrupt(ctx context.Context, interrupt *Interrupt) error

	// OnResume 恢复时调用
	OnResume(ctx context.Context, interrupt *Interrupt, resolution *InterruptResolution) error
}

// CallbackHandler 是回调处理器。
type CallbackHandler struct {
	onInterruptFunc func(ctx context.Context, interrupt *Interrupt) error
	onResumeFunc    func(ctx context.Context, interrupt *Interrupt, resolution *InterruptResolution) error
}

// NewCallbackHandler 创建回调处理器。
func NewCallbackHandler() *CallbackHandler {
	return &CallbackHandler{}
}

// OnInterrupt 实现 InterruptHandler 接口。
func (ch *CallbackHandler) OnInterrupt(ctx context.Context, interrupt *Interrupt) error {
	if ch.onInterruptFunc != nil {
		return ch.onInterruptFunc(ctx, interrupt)
	}
	return nil
}

// OnResume 实现 InterruptHandler 接口。
func (ch *CallbackHandler) OnResume(ctx context.Context, interrupt *Interrupt, resolution *InterruptResolution) error {
	if ch.onResumeFunc != nil {
		return ch.onResumeFunc(ctx, interrupt, resolution)
	}
	return nil
}

// WithOnInterrupt 设置中断回调。
func (ch *CallbackHandler) WithOnInterrupt(fn func(ctx context.Context, interrupt *Interrupt) error) *CallbackHandler {
	ch.onInterruptFunc = fn
	return ch
}

// WithOnResume 设置恢复回调。
func (ch *CallbackHandler) WithOnResume(fn func(ctx context.Context, interrupt *Interrupt, resolution *InterruptResolution) error) *CallbackHandler {
	ch.onResumeFunc = fn
	return ch
}

// HITLConfig 是 HITL 配置。
type HITLConfig struct {
	// DefaultTimeout 默认超时
	DefaultTimeout time.Duration

	// AutoResume 自动恢复（无需人工干预）
	AutoResume bool

	// Handler 处理器
	Handler InterruptHandler
}

// NewHITLConfig 创建 HITL 配置。
func NewHITLConfig() *HITLConfig {
	return &HITLConfig{
		DefaultTimeout: 5 * time.Minute,
		AutoResume:     false,
	}
}

// WithHandler 设置处理器。
func (hc *HITLConfig) WithHandler(handler InterruptHandler) *HITLConfig {
	hc.Handler = handler
	return hc
}
