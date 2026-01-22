package tenant

import (
	"errors"
	"fmt"
	"time"
)

// 错误定义
var (
	ErrTenantNotFound        = errors.New("tenant: tenant not found")
	ErrTenantExists          = errors.New("tenant: tenant already exists")
	ErrInvalidTenant         = errors.New("tenant: invalid tenant")
	ErrTenantSuspended       = errors.New("tenant: tenant is suspended")
	ErrQuotaExceeded         = errors.New("tenant: quota exceeded")
	ErrMemberNotFound        = errors.New("tenant: member not found")
	ErrMemberExists          = errors.New("tenant: member already exists")
	ErrInsufficientResource  = errors.New("tenant: insufficient resource")
)

// TenantStatus 租户状态
type TenantStatus string

const (
	// StatusActive 活跃状态
	StatusActive TenantStatus = "active"

	// StatusSuspended 暂停状态
	StatusSuspended TenantStatus = "suspended"

	// StatusDeleted 已删除状态
	StatusDeleted TenantStatus = "deleted"

	// StatusTrial 试用状态
	StatusTrial TenantStatus = "trial"
)

// ResourceType 资源类型
type ResourceType string

const (
	// ResourceTypeAgent Agent 数量
	ResourceTypeAgent ResourceType = "agent"

	// ResourceTypeVectorStore 向量存储数量
	ResourceTypeVectorStore ResourceType = "vectorstore"

	// ResourceTypeDocument 文档数量
	ResourceTypeDocument ResourceType = "document"

	// ResourceTypeAPICall API 调用次数
	ResourceTypeAPICall ResourceType = "api_call"

	// ResourceTypeToken Token 使用量
	ResourceTypeToken ResourceType = "token"

	// ResourceTypeStorage 存储空间（GB）
	ResourceTypeStorage ResourceType = "storage"
)

// Quota 资源配额
type Quota struct {
	// MaxAgents 最大 Agent 数量
	MaxAgents int `json:"max_agents"`

	// MaxVectorStores 最大向量存储数量
	MaxVectorStores int `json:"max_vector_stores"`

	// MaxDocuments 最大文档数量
	MaxDocuments int `json:"max_documents"`

	// MaxAPIRequests 最大 API 请求数/天
	MaxAPIRequests int64 `json:"max_api_requests"`

	// MaxTokens 最大 Token 数/月
	MaxTokens int64 `json:"max_tokens"`

	// StorageGB 最大存储空间 (GB)
	StorageGB float64 `json:"storage_gb"`

	// CustomLimits 自定义限制
	CustomLimits map[string]int64 `json:"custom_limits,omitempty"`
}

// DefaultQuota 返回默认配额
func DefaultQuota() *Quota {
	return &Quota{
		MaxAgents:       10,
		MaxVectorStores: 5,
		MaxDocuments:    1000,
		MaxAPIRequests:  10000,
		MaxTokens:       1000000,
		StorageGB:       10,
		CustomLimits:    make(map[string]int64),
	}
}

// Usage 资源使用量
type Usage struct {
	// AgentCount 当前 Agent 数量
	AgentCount int `json:"agent_count"`

	// VectorStoreCount 当前向量存储数量
	VectorStoreCount int `json:"vector_store_count"`

	// DocumentCount 当前文档数量
	DocumentCount int `json:"document_count"`

	// APICallsToday 今日 API 调用数
	APICallsToday int64 `json:"api_calls_today"`

	// TokensThisMonth 本月 Token 使用量
	TokensThisMonth int64 `json:"tokens_this_month"`

	// StorageUsedGB 已使用存储空间 (GB)
	StorageUsedGB float64 `json:"storage_used_gb"`

	// LastResetAt 最后重置时间
	LastResetAt time.Time `json:"last_reset_at"`
}

// NewUsage 创建新的使用量记录
func NewUsage() *Usage {
	return &Usage{
		LastResetAt: time.Now(),
	}
}

// Tenant 租户
type Tenant struct {
	// ID 租户唯一标识
	ID string `json:"id"`

	// Name 租户名称
	Name string `json:"name"`

	// Description 租户描述
	Description string `json:"description"`

	// Status 租户状态
	Status TenantStatus `json:"status"`

	// Quota 资源配额
	Quota *Quota `json:"quota"`

	// Usage 当前资源使用量
	Usage *Usage `json:"usage"`

	// Settings 租户配置
	Settings map[string]any `json:"settings,omitempty"`

	// Metadata 元数据
	Metadata map[string]any `json:"metadata,omitempty"`

	// CreatedAt 创建时间
	CreatedAt time.Time `json:"created_at"`

	// UpdatedAt 更新时间
	UpdatedAt time.Time `json:"updated_at"`

	// ExpiresAt 过期时间（可选，用于试用租户）
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

// IsActive 检查租户是否活跃
func (t *Tenant) IsActive() bool {
	return t.Status == StatusActive
}

// IsExpired 检查租户是否过期
func (t *Tenant) IsExpired() bool {
	if t.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*t.ExpiresAt)
}

// CanUseResource 检查是否可以使用资源
func (t *Tenant) CanUseResource(resourceType ResourceType, amount int) bool {
	if t.Quota == nil {
		return true // 无配额限制
	}

	switch resourceType {
	case ResourceTypeAgent:
		return t.Usage.AgentCount+amount <= t.Quota.MaxAgents

	case ResourceTypeVectorStore:
		return t.Usage.VectorStoreCount+amount <= t.Quota.MaxVectorStores

	case ResourceTypeDocument:
		return t.Usage.DocumentCount+amount <= t.Quota.MaxDocuments

	case ResourceTypeAPICall:
		return t.Usage.APICallsToday+int64(amount) <= t.Quota.MaxAPIRequests

	case ResourceTypeToken:
		return t.Usage.TokensThisMonth+int64(amount) <= t.Quota.MaxTokens

	case ResourceTypeStorage:
		return t.Usage.StorageUsedGB+float64(amount) <= t.Quota.StorageGB

	default:
		return true
	}
}

// Member 租户成员
type Member struct {
	// TenantID 租户ID
	TenantID string `json:"tenant_id"`

	// UserID 用户ID
	UserID string `json:"user_id"`

	// Role 角色（如 admin, developer, viewer）
	Role string `json:"role"`

	// JoinedAt 加入时间
	JoinedAt time.Time `json:"joined_at"`

	// Metadata 元数据
	Metadata map[string]any `json:"metadata,omitempty"`
}

// ListOptions 列表查询选项
type ListOptions struct {
	// Limit 返回数量限制
	Limit int `json:"limit,omitempty"`

	// Offset 偏移量
	Offset int `json:"offset,omitempty"`

	// Status 过滤状态
	Status TenantStatus `json:"status,omitempty"`

	// OrderBy 排序字段
	OrderBy string `json:"order_by,omitempty"`

	// Descending 是否降序
	Descending bool `json:"descending,omitempty"`
}

// QuotaCheck 配额检查结果
type QuotaCheck struct {
	// Allowed 是否允许
	Allowed bool `json:"allowed"`

	// Current 当前使用量
	Current int64 `json:"current"`

	// Limit 配额限制
	Limit int64 `json:"limit"`

	// Remaining 剩余配额
	Remaining int64 `json:"remaining"`

	// Message 消息
	Message string `json:"message,omitempty"`
}

// String 返回配额检查结果的字符串表示
func (qc *QuotaCheck) String() string {
	if qc.Allowed {
		return fmt.Sprintf("Allowed: %d/%d (remaining: %d)", qc.Current, qc.Limit, qc.Remaining)
	}
	return fmt.Sprintf("Denied: %d/%d (%s)", qc.Current, qc.Limit, qc.Message)
}
