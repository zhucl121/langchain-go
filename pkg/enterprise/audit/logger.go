package audit

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"
	
	"github.com/google/uuid"
)

var (
	// ErrEventNotFound 事件未找到
	ErrEventNotFound = errors.New("audit: event not found")
	
	// ErrInvalidQuery 无效的查询
	ErrInvalidQuery = errors.New("audit: invalid query")
)

// MemoryAuditLogger 内存审计日志记录器
//
// 用于开发和测试环境。生产环境建议使用 PostgreSQL 等持久化存储。
type MemoryAuditLogger struct {
	events []*AuditEvent
	mu     sync.RWMutex
}

// NewMemoryAuditLogger 创建内存审计日志记录器
func NewMemoryAuditLogger() *MemoryAuditLogger {
	return &MemoryAuditLogger{
		events: make([]*AuditEvent, 0),
	}
}

// Log 记录审计事件
func (l *MemoryAuditLogger) Log(ctx context.Context, event *AuditEvent) error {
	if event == nil {
		return errors.New("audit: event is nil")
	}
	
	// 生成 ID
	if event.ID == "" {
		event.ID = uuid.New().String()
	}
	
	// 设置时间戳
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}
	
	l.mu.Lock()
	defer l.mu.Unlock()
	
	l.events = append(l.events, event)
	return nil
}

// Query 查询审计日志
func (l *MemoryAuditLogger) Query(ctx context.Context, query *AuditQuery) ([]*AuditEvent, error) {
	if query == nil {
		return nil, ErrInvalidQuery
	}
	
	l.mu.RLock()
	defer l.mu.RUnlock()
	
	var results []*AuditEvent
	
	for _, event := range l.events {
		if l.matchEvent(event, query) {
			results = append(results, event)
		}
	}
	
	// 应用分页
	if query.Limit > 0 {
		start := query.Offset
		if start >= len(results) {
			return []*AuditEvent{}, nil
		}
		
		end := start + query.Limit
		if end > len(results) {
			end = len(results)
		}
		
		results = results[start:end]
	}
	
	return results, nil
}

// matchEvent 判断事件是否匹配查询条件
func (l *MemoryAuditLogger) matchEvent(event *AuditEvent, query *AuditQuery) bool {
	// 租户过滤
	if query.TenantID != "" && event.TenantID != query.TenantID {
		return false
	}
	
	// 用户过滤
	if query.UserID != "" && event.UserID != query.UserID {
		return false
	}
	
	// 操作过滤
	if query.Action != "" && event.Action != query.Action {
		return false
	}
	
	// 资源过滤
	if query.Resource != "" && event.Resource != query.Resource {
		return false
	}
	
	// 状态过滤
	if query.Status != "" && event.Status != query.Status {
		return false
	}
	
	// 时间范围过滤
	if !query.StartTime.IsZero() && event.Timestamp.Before(query.StartTime) {
		return false
	}
	if !query.EndTime.IsZero() && event.Timestamp.After(query.EndTime) {
		return false
	}
	
	return true
}

// Export 导出审计日志
func (l *MemoryAuditLogger) Export(ctx context.Context, query *AuditQuery, format ExportFormat) (io.Reader, error) {
	events, err := l.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	
	switch format {
	case ExportFormatJSON:
		return l.exportJSON(events)
	case ExportFormatCSV:
		return l.exportCSV(events)
	default:
		return nil, fmt.Errorf("audit: unsupported export format: %s", format)
	}
}

// exportJSON 导出为 JSON 格式
func (l *MemoryAuditLogger) exportJSON(events []*AuditEvent) (io.Reader, error) {
	data, err := json.MarshalIndent(events, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("audit: failed to marshal JSON: %w", err)
	}
	return strings.NewReader(string(data)), nil
}

// exportCSV 导出为 CSV 格式
func (l *MemoryAuditLogger) exportCSV(events []*AuditEvent) (io.Reader, error) {
	var buf strings.Builder
	writer := csv.NewWriter(&buf)
	
	// 写入标题行
	header := []string{
		"ID", "TenantID", "UserID", "Action", "Resource", "ResourceID",
		"Status", "ErrorMessage", "IPAddress", "Duration", "Timestamp",
	}
	if err := writer.Write(header); err != nil {
		return nil, fmt.Errorf("audit: failed to write CSV header: %w", err)
	}
	
	// 写入数据行
	for _, event := range events {
		row := []string{
			event.ID,
			event.TenantID,
			event.UserID,
			event.Action,
			event.Resource,
			event.ResourceID,
			string(event.Status),
			event.ErrorMessage,
			event.IPAddress,
			event.Duration.String(),
			event.Timestamp.Format(time.RFC3339),
		}
		if err := writer.Write(row); err != nil {
			return nil, fmt.Errorf("audit: failed to write CSV row: %w", err)
		}
	}
	
	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, fmt.Errorf("audit: CSV writer error: %w", err)
	}
	
	return strings.NewReader(buf.String()), nil
}

// Count 统计审计日志数量
func (l *MemoryAuditLogger) Count(ctx context.Context, query *AuditQuery) (int, error) {
	events, err := l.Query(ctx, query)
	if err != nil {
		return 0, err
	}
	return len(events), nil
}
