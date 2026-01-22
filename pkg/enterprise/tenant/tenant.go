package tenant

import (
	"context"
)

// TenantManager 租户管理器接口
type TenantManager interface {
	// CreateTenant 创建租户
	CreateTenant(ctx context.Context, tenant *Tenant) error

	// GetTenant 获取租户
	GetTenant(ctx context.Context, tenantID string) (*Tenant, error)

	// UpdateTenant 更新租户
	UpdateTenant(ctx context.Context, tenant *Tenant) error

	// DeleteTenant 删除租户（软删除）
	DeleteTenant(ctx context.Context, tenantID string) error

	// ListTenants 列出租户
	ListTenants(ctx context.Context, opts *ListOptions) ([]*Tenant, error)

	// SuspendTenant 暂停租户
	SuspendTenant(ctx context.Context, tenantID string) error

	// ActivateTenant 激活租户
	ActivateTenant(ctx context.Context, tenantID string) error

	// SetQuota 设置租户配额
	SetQuota(ctx context.Context, tenantID string, quota *Quota) error

	// GetQuota 获取租户配额
	GetQuota(ctx context.Context, tenantID string) (*Quota, error)

	// CheckQuota 检查配额
	CheckQuota(ctx context.Context, tenantID string, resourceType ResourceType) (*QuotaCheck, error)

	// IncrementUsage 增加资源使用量
	IncrementUsage(ctx context.Context, tenantID string, resourceType ResourceType, amount int) error

	// DecrementUsage 减少资源使用量
	DecrementUsage(ctx context.Context, tenantID string, resourceType ResourceType, amount int) error

	// AddMember 添加成员
	AddMember(ctx context.Context, tenantID, userID, role string) error

	// RemoveMember 移除成员
	RemoveMember(ctx context.Context, tenantID, userID string) error

	// GetMembers 获取租户成员
	GetMembers(ctx context.Context, tenantID string) ([]*Member, error)

	// GetUserTenants 获取用户所属的租户
	GetUserTenants(ctx context.Context, userID string) ([]*Tenant, error)
}

// Store 存储接口
type Store interface {
	// Tenant 相关
	SaveTenant(ctx context.Context, tenant *Tenant) error
	GetTenant(ctx context.Context, tenantID string) (*Tenant, error)
	DeleteTenant(ctx context.Context, tenantID string) error
	ListTenants(ctx context.Context, opts *ListOptions) ([]*Tenant, error)

	// Member 相关
	SaveMember(ctx context.Context, member *Member) error
	DeleteMember(ctx context.Context, tenantID, userID string) error
	GetMembers(ctx context.Context, tenantID string) ([]*Member, error)
	GetUserTenants(ctx context.Context, userID string) ([]*Tenant, error)
}
