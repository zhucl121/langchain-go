package hitl

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// WorkflowStatus 工作流状态
type WorkflowStatus string

const (
	// WorkflowStatusPending 待处理
	WorkflowStatusPending WorkflowStatus = "pending"
	
	// WorkflowStatusInProgress 进行中
	WorkflowStatusInProgress WorkflowStatus = "in_progress"
	
	// WorkflowStatusApproved 已批准
	WorkflowStatusApproved WorkflowStatus = "approved"
	
	// WorkflowStatusRejected 已拒绝
	WorkflowStatusRejected WorkflowStatus = "rejected"
	
	// WorkflowStatusCancelled 已取消
	WorkflowStatusCancelled WorkflowStatus = "cancelled"
	
	// WorkflowStatusTimeout 超时
	WorkflowStatusTimeout WorkflowStatus = "timeout"
)

// ApprovalWorkflow 审批工作流
type ApprovalWorkflow struct {
	// ID 工作流 ID
	ID string
	
	// Title 标题
	Title string
	
	// Description 描述
	Description string
	
	// Status 状态
	Status WorkflowStatus
	
	// Steps 审批步骤
	Steps []*ApprovalStep
	
	// CurrentStepIndex 当前步骤索引
	CurrentStepIndex int
	
	// CreatedAt 创建时间
	CreatedAt time.Time
	
	// UpdatedAt 更新时间
	UpdatedAt time.Time
	
	// CompletedAt 完成时间
	CompletedAt *time.Time
	
	// Metadata 元数据
	Metadata map[string]interface{}
	
	mu sync.RWMutex
}

// ApprovalStep 审批步骤
type ApprovalStep struct {
	// ID 步骤 ID
	ID string
	
	// Name 步骤名称
	Name string
	
	// Approvers 审批人列表
	Approvers []string
	
	// RequiredApprovals 需要的审批数（默认全部）
	RequiredApprovals int
	
	// Timeout 超时时间
	Timeout time.Duration
	
	// Status 状态
	Status WorkflowStatus
	
	// Decisions 审批决策列表
	Decisions []*ApprovalDecision
	
	// StartedAt 开始时间
	StartedAt *time.Time
	
	// CompletedAt 完成时间
	CompletedAt *time.Time
}

// NewApprovalWorkflow 创建审批工作流
func NewApprovalWorkflow(id, title string) *ApprovalWorkflow {
	return &ApprovalWorkflow{
		ID:          id,
		Title:       title,
		Status:      WorkflowStatusPending,
		Steps:       make([]*ApprovalStep, 0),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Metadata:    make(map[string]interface{}),
	}
}

// AddStep 添加审批步骤
func (w *ApprovalWorkflow) AddStep(step *ApprovalStep) {
	w.mu.Lock()
	defer w.mu.Unlock()
	
	w.Steps = append(w.Steps, step)
}

// Start 启动工作流
func (w *ApprovalWorkflow) Start() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	
	if w.Status != WorkflowStatusPending {
		return fmt.Errorf("workflow already started")
	}
	
	if len(w.Steps) == 0 {
		return fmt.Errorf("no steps defined")
	}
	
	w.Status = WorkflowStatusInProgress
	w.CurrentStepIndex = 0
	w.UpdatedAt = time.Now()
	
	// 启动第一个步骤
	now := time.Now()
	w.Steps[0].Status = WorkflowStatusInProgress
	w.Steps[0].StartedAt = &now
	
	return nil
}

// SubmitApproval 提交审批
func (w *ApprovalWorkflow) SubmitApproval(stepID, approver string, decision *ApprovalDecision) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	
	// 查找步骤
	var step *ApprovalStep
	for _, s := range w.Steps {
		if s.ID == stepID {
			step = s
			break
		}
	}
	
	if step == nil {
		return fmt.Errorf("step %s not found", stepID)
	}
	
	// 检查步骤状态
	if step.Status != WorkflowStatusInProgress {
		return fmt.Errorf("step is not in progress")
	}
	
	// 检查审批人权限
	hasPermission := false
	for _, a := range step.Approvers {
		if a == approver {
			hasPermission = true
			break
		}
	}
	
	if !hasPermission {
		return fmt.Errorf("approver %s not authorized for this step", approver)
	}
	
	// 记录决策
	decision.Approver = approver
	step.Decisions = append(step.Decisions, decision)
	
	// 检查是否达到所需审批数
	approvedCount := 0
	rejectedCount := 0
	
	for _, d := range step.Decisions {
		switch d.Status {
		case ApprovalApproved:
			approvedCount++
		case ApprovalRejected:
			rejectedCount++
		}
	}
	
	// 更新步骤状态
	if rejectedCount > 0 {
		// 有拒绝，整个步骤被拒绝
		step.Status = WorkflowStatusRejected
		w.Status = WorkflowStatusRejected
		now := time.Now()
		step.CompletedAt = &now
		w.CompletedAt = &now
		w.UpdatedAt = now
	} else if approvedCount >= step.RequiredApprovals {
		// 批准数达到要求
		step.Status = WorkflowStatusApproved
		now := time.Now()
		step.CompletedAt = &now
		w.UpdatedAt = now
		
		// 进入下一步或完成工作流
		if err := w.advanceToNextStep(); err != nil {
			return err
		}
	}
	
	return nil
}

// advanceToNextStep 前进到下一步
func (w *ApprovalWorkflow) advanceToNextStep() error {
	w.CurrentStepIndex++
	
	if w.CurrentStepIndex >= len(w.Steps) {
		// 所有步骤完成
		w.Status = WorkflowStatusApproved
		now := time.Now()
		w.CompletedAt = &now
		return nil
	}
	
	// 启动下一步
	now := time.Now()
	nextStep := w.Steps[w.CurrentStepIndex]
	nextStep.Status = WorkflowStatusInProgress
	nextStep.StartedAt = &now
	
	return nil
}

// Cancel 取消工作流
func (w *ApprovalWorkflow) Cancel() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	
	if w.Status == WorkflowStatusApproved || w.Status == WorkflowStatusRejected {
		return fmt.Errorf("cannot cancel completed workflow")
	}
	
	w.Status = WorkflowStatusCancelled
	now := time.Now()
	w.CompletedAt = &now
	w.UpdatedAt = now
	
	return nil
}

// GetCurrentStep 获取当前步骤
func (w *ApprovalWorkflow) GetCurrentStep() *ApprovalStep {
	w.mu.RLock()
	defer w.mu.RUnlock()
	
	if w.CurrentStepIndex >= 0 && w.CurrentStepIndex < len(w.Steps) {
		return w.Steps[w.CurrentStepIndex]
	}
	
	return nil
}

// IsCompleted 是否已完成
func (w *ApprovalWorkflow) IsCompleted() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	
	return w.Status == WorkflowStatusApproved ||
		w.Status == WorkflowStatusRejected ||
		w.Status == WorkflowStatusCancelled
}

// WorkflowEngine 工作流引擎
type WorkflowEngine struct {
	workflows map[string]*ApprovalWorkflow
	mu        sync.RWMutex
}

// NewWorkflowEngine 创建工作流引擎
func NewWorkflowEngine() *WorkflowEngine {
	return &WorkflowEngine{
		workflows: make(map[string]*ApprovalWorkflow),
	}
}

// CreateWorkflow 创建工作流
func (e *WorkflowEngine) CreateWorkflow(workflow *ApprovalWorkflow) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	
	if _, exists := e.workflows[workflow.ID]; exists {
		return fmt.Errorf("workflow %s already exists", workflow.ID)
	}
	
	e.workflows[workflow.ID] = workflow
	return nil
}

// GetWorkflow 获取工作流
func (e *WorkflowEngine) GetWorkflow(workflowID string) (*ApprovalWorkflow, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	
	workflow, exists := e.workflows[workflowID]
	if !exists {
		return nil, fmt.Errorf("workflow %s not found", workflowID)
	}
	
	return workflow, nil
}

// StartWorkflow 启动工作流
func (e *WorkflowEngine) StartWorkflow(workflowID string) error {
	workflow, err := e.GetWorkflow(workflowID)
	if err != nil {
		return err
	}
	
	return workflow.Start()
}

// SubmitApproval 提交审批
func (e *WorkflowEngine) SubmitApproval(workflowID, stepID, approver string, decision *ApprovalDecision) error {
	workflow, err := e.GetWorkflow(workflowID)
	if err != nil {
		return err
	}
	
	return workflow.SubmitApproval(stepID, approver, decision)
}

// CancelWorkflow 取消工作流
func (e *WorkflowEngine) CancelWorkflow(workflowID string) error {
	workflow, err := e.GetWorkflow(workflowID)
	if err != nil {
		return err
	}
	
	return workflow.Cancel()
}

// ListWorkflows 列出所有工作流
func (e *WorkflowEngine) ListWorkflows(status WorkflowStatus) []*ApprovalWorkflow {
	e.mu.RLock()
	defer e.mu.RUnlock()
	
	workflows := make([]*ApprovalWorkflow, 0)
	for _, workflow := range e.workflows {
		if status == "" || workflow.Status == status {
			workflows = append(workflows, workflow)
		}
	}
	
	return workflows
}

// WaitForWorkflowCompletion 等待工作流完成
func (e *WorkflowEngine) WaitForWorkflowCompletion(ctx context.Context, workflowID string) (*ApprovalWorkflow, error) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			workflow, err := e.GetWorkflow(workflowID)
			if err != nil {
				return nil, err
			}
			
			if workflow.IsCompleted() {
				return workflow, nil
			}
		}
	}
}

// NewApprovalStep 创建审批步骤
func NewApprovalStep(id, name string, approvers []string) *ApprovalStep {
	return &ApprovalStep{
		ID:                id,
		Name:              name,
		Approvers:         approvers,
		RequiredApprovals: len(approvers), // 默认需要所有人审批
		Timeout:           5 * time.Minute,
		Status:            WorkflowStatusPending,
		Decisions:         make([]*ApprovalDecision, 0),
	}
}

// WithRequiredApprovals 设置所需审批数
func (s *ApprovalStep) WithRequiredApprovals(n int) *ApprovalStep {
	s.RequiredApprovals = n
	return s
}

// WithTimeout 设置超时时间
func (s *ApprovalStep) WithTimeout(d time.Duration) *ApprovalStep {
	s.Timeout = d
	return s
}
