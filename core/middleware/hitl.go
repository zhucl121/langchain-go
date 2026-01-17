package middleware

import (
	"context"
	"fmt"
	
	"github.com/zhucl121/langchain-go/graph/hitl"
)

// HITLMiddleware 是 Human-in-the-Loop 中间件。
//
// 集成 HITL 系统到中间件链，支持自动中断和审批。
//
type HITLMiddleware struct {
	interruptManager *hitl.InterruptManager
	approvalManager  *hitl.ApprovalManager
	config           *hitl.HITLConfig
	
	// 中断条件
	shouldInterrupt func(ctx context.Context, input any) bool
}

// NewHITLMiddleware 创建 HITL 中间件。
//
// 参数：
//   - interruptManager: 中断管理器
//   - approvalManager: 审批管理器
//
// 返回：
//   - *HITLMiddleware: HITL 中间件实例
//
func NewHITLMiddleware(
	interruptManager *hitl.InterruptManager,
	approvalManager *hitl.ApprovalManager,
) *HITLMiddleware {
	return &HITLMiddleware{
		interruptManager: interruptManager,
		approvalManager:  approvalManager,
		config:           hitl.NewHITLConfig(),
	}
}

// WithConfig 设置 HITL 配置。
func (hm *HITLMiddleware) WithConfig(config *hitl.HITLConfig) *HITLMiddleware {
	hm.config = config
	return hm
}

// WithInterruptCondition 设置中断条件。
//
// 参数：
//   - condition: 中断条件函数
//
// 返回：
//   - *HITLMiddleware: 返回自身
//
func (hm *HITLMiddleware) WithInterruptCondition(
	condition func(ctx context.Context, input any) bool,
) *HITLMiddleware {
	hm.shouldInterrupt = condition
	return hm
}

// Process 实现 Middleware 接口。
func (hm *HITLMiddleware) Process(ctx context.Context, input any, next NextFunc) (any, error) {
	// 检查是否需要中断
	if hm.shouldInterrupt != nil && hm.shouldInterrupt(ctx, input) {
		return hm.handleInterrupt(ctx, input, next)
	}

	// 正常执行
	return next(ctx, input)
}

// handleInterrupt 处理中断。
func (hm *HITLMiddleware) handleInterrupt(
	ctx context.Context,
	input any,
	next NextFunc,
) (any, error) {
	// 生成中断 ID
	interruptID := fmt.Sprintf("hitl-middleware-%d", len(hm.interruptManager.GetHistory()))
	
	// 创建中断点
	point := hitl.NewInterruptPoint("middleware", hitl.InterruptManual).
		WithMessage("HITL middleware interrupt")

	// 创建中断
	interrupt := hm.interruptManager.CreateInterrupt(
		interruptID,
		point,
		"middleware-thread",
		input,
	)

	// 如果配置了处理器，调用
	if hm.config.Handler != nil {
		if err := hm.config.Handler.OnInterrupt(ctx, interrupt); err != nil {
			return nil, fmt.Errorf("hitl middleware: interrupt handler failed: %w", err)
		}
	}

	// 等待解决
	resolution, err := hm.interruptManager.WaitForResolution(ctx, interruptID)
	if err != nil {
		return nil, fmt.Errorf("hitl middleware: wait for resolution failed: %w", err)
	}

	// 根据解决方案决定下一步
	switch resolution.Action {
	case hitl.ActionContinue:
		// 继续执行
		if resolution.Input != nil {
			return next(ctx, resolution.Input)
		}
		return next(ctx, input)

	case hitl.ActionModify:
		// 使用修改后的状态
		if resolution.ModifiedState != nil {
			return resolution.ModifiedState, nil
		}
		return next(ctx, input)

	case hitl.ActionSkip:
		// 跳过执行，返回输入
		return input, nil

	case hitl.ActionAbort:
		// 中止执行
		return nil, fmt.Errorf("hitl middleware: execution aborted: %s", resolution.Message)

	default:
		return next(ctx, input)
	}
}

// ApprovalMiddleware 是审批中间件。
//
// 在执行前请求审批。
//
type ApprovalMiddleware struct {
	approvalManager *hitl.ApprovalManager
	requestBuilder  func(ctx context.Context, input any) *hitl.ApprovalRequest
}

// NewApprovalMiddleware 创建审批中间件。
//
// 参数：
//   - approvalManager: 审批管理器
//
// 返回：
//   - *ApprovalMiddleware: 审批中间件实例
//
func NewApprovalMiddleware(approvalManager *hitl.ApprovalManager) *ApprovalMiddleware {
	return &ApprovalMiddleware{
		approvalManager: approvalManager,
	}
}

// WithRequestBuilder 设置审批请求构建器。
func (am *ApprovalMiddleware) WithRequestBuilder(
	builder func(ctx context.Context, input any) *hitl.ApprovalRequest,
) *ApprovalMiddleware {
	am.requestBuilder = builder
	return am
}

// Process 实现 Middleware 接口。
func (am *ApprovalMiddleware) Process(ctx context.Context, input any, next NextFunc) (any, error) {
	// 构建审批请求
	var request *hitl.ApprovalRequest
	if am.requestBuilder != nil {
		request = am.requestBuilder(ctx, input)
	} else {
		// 默认审批请求
		request = hitl.NewApprovalRequest(
			fmt.Sprintf("approval-%p", input),
			"Approval required",
		).WithOptions("approve", "reject")
	}

	// 请求审批
	decision, err := am.approvalManager.RequestApproval(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("approval middleware: request failed: %w", err)
	}

	// 根据决策执行
	switch decision.Status {
	case hitl.ApprovalApproved:
		// 批准，继续执行
		return next(ctx, input)

	case hitl.ApprovalRejected:
		// 拒绝，返回错误
		return nil, fmt.Errorf("approval middleware: request rejected: %s", decision.Comment)

	case hitl.ApprovalTimeout:
		// 超时
		return nil, fmt.Errorf("approval middleware: request timeout")

	default:
		return nil, fmt.Errorf("approval middleware: unknown status: %s", decision.Status)
	}
}

// InterruptOnErrorMiddleware 是错误中断中间件。
//
// 当发生错误时自动创建中断。
//
type InterruptOnErrorMiddleware struct {
	interruptManager *hitl.InterruptManager
}

// NewInterruptOnErrorMiddleware 创建错误中断中间件。
func NewInterruptOnErrorMiddleware(interruptManager *hitl.InterruptManager) *InterruptOnErrorMiddleware {
	return &InterruptOnErrorMiddleware{
		interruptManager: interruptManager,
	}
}

// Process 实现 Middleware 接口。
func (iem *InterruptOnErrorMiddleware) Process(ctx context.Context, input any, next NextFunc) (any, error) {
	result, err := next(ctx, input)
	
	if err != nil {
		// 创建错误中断
		point := hitl.NewInterruptPoint("error", hitl.InterruptOnError).
			WithMessage(fmt.Sprintf("Error occurred: %v", err))

		interruptID := fmt.Sprintf("error-interrupt-%d", len(iem.interruptManager.GetHistory()))
		
		_ = iem.interruptManager.CreateInterrupt(
			interruptID,
			point,
			"error-thread",
			input,
		)

		// 等待人工处理
		resolution, waitErr := iem.interruptManager.WaitForResolution(ctx, interruptID)
		if waitErr != nil {
			// 等待失败，返回原始错误
			return nil, err
		}

		// 根据解决方案处理
		switch resolution.Action {
		case hitl.ActionRetry:
			// 重试
			return next(ctx, input)
		case hitl.ActionContinue:
			// 忽略错误继续
			return result, nil
		default:
			// 返回原始错误
			return nil, err
		}
	}

	return result, nil
}
