// Package audit 提供企业级审计日志功能。
//
// 审计日志系统用于记录所有关键操作，满足合规要求（如 SOC2、ISO27001）。
//
// 核心功能：
//   - 审计事件记录
//   - 日志查询和过滤
//   - 日志导出（CSV/JSON）
//   - 日志归档和轮转
//   - Middleware 自动记录
//
// 使用示例：
//
//	// 创建审计日志记录器
//	logger := audit.NewMemoryAuditLogger()
//
//	// 记录审计事件
//	event := &audit.AuditEvent{
//	    TenantID:  "tenant-1",
//	    UserID:    "user-1",
//	    Action:    "agent.execute",
//	    Resource:  "agent",
//	    Status:    audit.StatusSuccess,
//	    Duration:  150 * time.Millisecond,
//	}
//	logger.Log(ctx, event)
//
//	// 查询审计日志
//	query := &audit.AuditQuery{
//	    TenantID:  "tenant-1",
//	    StartTime: time.Now().Add(-24 * time.Hour),
//	    EndTime:   time.Now(),
//	}
//	events, _ := logger.Query(ctx, query)
//
//	// 导出日志
//	reader, _ := logger.Export(ctx, query, audit.ExportFormatJSON)
//
package audit
