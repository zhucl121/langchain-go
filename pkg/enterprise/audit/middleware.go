package audit

import (
	"context"
	"time"
	
	"github.com/google/uuid"
)

// contextKey 是审计日志上下文键类型
type contextKey string

const (
	// auditStartTimeKey 审计开始时间键
	auditStartTimeKey contextKey = "audit.start_time"
	
	// auditEventKey 审计事件键
	auditEventKey contextKey = "audit.event"
)

// WithStartTime 在 context 中保存开始时间
func WithStartTime(ctx context.Context) context.Context {
	return context.WithValue(ctx, auditStartTimeKey, time.Now())
}

// GetStartTime 从 context 获取开始时间
func GetStartTime(ctx context.Context) (time.Time, bool) {
	t, ok := ctx.Value(auditStartTimeKey).(time.Time)
	return t, ok
}

// WithEvent 在 context 中保存审计事件
func WithEvent(ctx context.Context, event *AuditEvent) context.Context {
	return context.WithValue(ctx, auditEventKey, event)
}

// GetEvent 从 context 获取审计事件
func GetEvent(ctx context.Context) (*AuditEvent, bool) {
	event, ok := ctx.Value(auditEventKey).(*AuditEvent)
	return event, ok
}

// Middleware 审计中间件
//
// 自动记录 Agent 执行的审计日志。
type Middleware struct {
	logger AuditLogger
}

// NewMiddleware 创建审计中间件
func NewMiddleware(logger AuditLogger) *Middleware {
	return &Middleware{
		logger: logger,
	}
}

// BeforeExecution 执行前
//
// 保存开始时间。
func (m *Middleware) BeforeExecution(ctx context.Context, input map[string]any) (context.Context, error) {
	// 保存开始时间
	ctx = WithStartTime(ctx)
	
	// 创建审计事件（稍后填充）
	event := &AuditEvent{
		ID:        uuid.New().String(),
		Timestamp: time.Now(),
		Request:   input,
	}
	ctx = WithEvent(ctx, event)
	
	return ctx, nil
}

// AfterExecution 执行后
//
// 记录成功的审计日志。
func (m *Middleware) AfterExecution(ctx context.Context, output map[string]any) error {
	startTime, ok := GetStartTime(ctx)
	if !ok {
		startTime = time.Now()
	}
	
	event, ok := GetEvent(ctx)
	if !ok {
		event = &AuditEvent{
			ID:        uuid.New().String(),
			Timestamp: time.Now(),
		}
	}
	
	// 填充事件信息
	event.Status = StatusSuccess
	event.Duration = time.Since(startTime)
	event.Response = output
	
	// 记录日志
	return m.logger.Log(ctx, event)
}

// OnError 执行失败
//
// 记录失败的审计日志。
func (m *Middleware) OnError(ctx context.Context, err error) error {
	startTime, ok := GetStartTime(ctx)
	if !ok {
		startTime = time.Now()
	}
	
	event, ok := GetEvent(ctx)
	if !ok {
		event = &AuditEvent{
			ID:        uuid.New().String(),
			Timestamp: time.Now(),
		}
	}
	
	// 填充事件信息
	event.Status = StatusFailure
	event.Duration = time.Since(startTime)
	event.ErrorMessage = err.Error()
	
	// 记录日志
	m.logger.Log(ctx, event)
	
	return err // 返回原始错误
}

// PopulateFromContext 从 context 填充事件信息
//
// 辅助函数，用于从 context 中提取常见信息。
func PopulateFromContext(ctx context.Context, event *AuditEvent) {
	// 这里可以从 context 中提取租户ID、用户ID等信息
	// 具体实现取决于应用的 context 设计
	
	// 示例：
	// if tenantID, ok := tenant.GetTenant(ctx); ok {
	//     event.TenantID = tenantID
	// }
	// if userID, ok := auth.GetUserID(ctx); ok {
	//     event.UserID = userID
	// }
}
