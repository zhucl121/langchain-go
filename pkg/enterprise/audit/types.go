package audit

import (
	"time"
)

// EventStatus 事件状态
type EventStatus string

const (
	// StatusSuccess 成功
	StatusSuccess EventStatus = "success"
	
	// StatusFailure 失败
	StatusFailure EventStatus = "failure"
	
	// StatusDenied 拒绝（权限不足）
	StatusDenied EventStatus = "denied"
)

// ExportFormat 导出格式
type ExportFormat string

const (
	// ExportFormatJSON JSON 格式
	ExportFormatJSON ExportFormat = "json"
	
	// ExportFormatCSV CSV 格式
	ExportFormatCSV ExportFormat = "csv"
)

// AuditEvent 审计事件
//
// 记录系统中的所有关键操作。
type AuditEvent struct {
	// ID 事件唯一标识
	ID string `json:"id"`
	
	// TenantID 租户 ID
	TenantID string `json:"tenant_id"`
	
	// UserID 用户 ID
	UserID string `json:"user_id"`
	
	// Action 操作（如 "agent.execute", "model.invoke"）
	Action string `json:"action"`
	
	// Resource 资源类型（如 "agent", "model", "vectorstore"）
	Resource string `json:"resource"`
	
	// ResourceID 资源 ID
	ResourceID string `json:"resource_id,omitempty"`
	
	// Status 状态
	Status EventStatus `json:"status"`
	
	// ErrorMessage 错误信息（失败时）
	ErrorMessage string `json:"error_message,omitempty"`
	
	// IPAddress 客户端 IP 地址
	IPAddress string `json:"ip_address,omitempty"`
	
	// UserAgent 用户代理
	UserAgent string `json:"user_agent,omitempty"`
	
	// Request 请求参数（脱敏后）
	Request map[string]any `json:"request,omitempty"`
	
	// Response 响应数据（脱敏后）
	Response map[string]any `json:"response,omitempty"`
	
	// Duration 执行时长
	Duration time.Duration `json:"duration"`
	
	// Timestamp 时间戳
	Timestamp time.Time `json:"timestamp"`
	
	// Metadata 额外元数据
	Metadata map[string]any `json:"metadata,omitempty"`
}

// AuditQuery 审计查询条件
type AuditQuery struct {
	// TenantID 租户 ID（可选）
	TenantID string
	
	// UserID 用户 ID（可选）
	UserID string
	
	// Action 操作（可选）
	Action string
	
	// Resource 资源类型（可选）
	Resource string
	
	// Status 状态（可选）
	Status EventStatus
	
	// StartTime 开始时间
	StartTime time.Time
	
	// EndTime 结束时间
	EndTime time.Time
	
	// Limit 返回数量限制
	Limit int
	
	// Offset 偏移量
	Offset int
}

// ListOptions 列表选项
type ListOptions struct {
	// Limit 返回数量限制
	Limit int
	
	// Offset 偏移量
	Offset int
	
	// OrderBy 排序字段
	OrderBy string
	
	// OrderDesc 是否降序
	OrderDesc bool
}
