package audit

import (
	"context"
	"io"
)

// AuditLogger 审计日志记录器接口
type AuditLogger interface {
	// Log 记录审计事件
	//
	// 参数：
	//   - ctx: 上下文
	//   - event: 审计事件
	//
	// 返回：
	//   - error: 错误
	Log(ctx context.Context, event *AuditEvent) error
	
	// Query 查询审计日志
	//
	// 参数：
	//   - ctx: 上下文
	//   - query: 查询条件
	//
	// 返回：
	//   - []*AuditEvent: 审计事件列表
	//   - error: 错误
	Query(ctx context.Context, query *AuditQuery) ([]*AuditEvent, error)
	
	// Export 导出审计日志
	//
	// 参数：
	//   - ctx: 上下文
	//   - query: 查询条件
	//   - format: 导出格式
	//
	// 返回：
	//   - io.Reader: 数据流
	//   - error: 错误
	Export(ctx context.Context, query *AuditQuery, format ExportFormat) (io.Reader, error)
	
	// Count 统计审计日志数量
	//
	// 参数：
	//   - ctx: 上下文
	//   - query: 查询条件
	//
	// 返回：
	//   - int: 数量
	//   - error: 错误
	Count(ctx context.Context, query *AuditQuery) (int, error)
}
