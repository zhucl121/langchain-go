package auth

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// TenantStatus 租户状态
type TenantStatus string

const (
	// TenantStatusActive 活跃状态
	TenantStatusActive TenantStatus = "active"
	
	// TenantStatusSuspended 暂停状态
	TenantStatusSuspended TenantStatus = "suspended"
	
	// TenantStatusDeleted 已删除状态
	TenantStatusDeleted TenantStatus = "deleted"
)

// Tenant 租户
type Tenant struct {
	// ID 租户 ID
	ID string
	
	// Name 租户名称
	Name string
	
	// Status 租户状态
	Status TenantStatus
	
	// Quota 资源配额
	Quota *ResourceQuota
	
	// Usage 当前资源使用量
	Usage *ResourceUsage
	
	// Settings 租户配置
	Settings map[string]interface{}
	
	// Metadata 元数据
	Metadata map[string]interface{}
	
	// CreatedAt 创建时间
	CreatedAt time.Time
	
	// UpdatedAt 更新时间
	UpdatedAt time.Time
}

// IsActive 租户是否活跃
func (t *Tenant) IsActive() bool {
	return t.Status == TenantStatusActive
}

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
	ListTenants(ctx context.Context) ([]*Tenant, error)
	
	// SuspendTenant 暂停租户
	SuspendTenant(ctx context.Context, tenantID string) error
	
	// ActivateTenant 激活租户
	ActivateTenant(ctx context.Context, tenantID string) error
	
	// IsolateTenantData 隔离租户数据
	IsolateTenantData(ctx context.Context, tenantID string) error
}

// InMemoryTenantManager 内存租户管理器
type InMemoryTenantManager struct {
	mu      sync.RWMutex
	tenants map[string]*Tenant
}

// NewInMemoryTenantManager 创建内存租户管理器
func NewInMemoryTenantManager() *InMemoryTenantManager {
	return &InMemoryTenantManager{
		tenants: make(map[string]*Tenant),
	}
}

func (m *InMemoryTenantManager) CreateTenant(ctx context.Context, tenant *Tenant) error {
	if tenant.ID == "" {
		return fmt.Errorf("tenant ID is required")
	}
	
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if _, exists := m.tenants[tenant.ID]; exists {
		return fmt.Errorf("tenant %s already exists", tenant.ID)
	}
	
	tenant.CreatedAt = time.Now()
	tenant.UpdatedAt = time.Now()
	tenant.Status = TenantStatusActive
	
	// 设置默认配额
	if tenant.Quota == nil {
		tenant.Quota = DefaultResourceQuota()
	}
	
	// 初始化使用量
	if tenant.Usage == nil {
		tenant.Usage = &ResourceUsage{}
	}
	
	m.tenants[tenant.ID] = tenant
	
	return nil
}

func (m *InMemoryTenantManager) GetTenant(ctx context.Context, tenantID string) (*Tenant, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	tenant, exists := m.tenants[tenantID]
	if !exists {
		return nil, ErrTenantNotFound
	}
	
	return tenant, nil
}

func (m *InMemoryTenantManager) UpdateTenant(ctx context.Context, tenant *Tenant) error {
	if tenant.ID == "" {
		return fmt.Errorf("tenant ID is required")
	}
	
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if _, exists := m.tenants[tenant.ID]; !exists {
		return ErrTenantNotFound
	}
	
	tenant.UpdatedAt = time.Now()
	m.tenants[tenant.ID] = tenant
	
	return nil
}

func (m *InMemoryTenantManager) DeleteTenant(ctx context.Context, tenantID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	tenant, exists := m.tenants[tenantID]
	if !exists {
		return ErrTenantNotFound
	}
	
	// 软删除
	tenant.Status = TenantStatusDeleted
	tenant.UpdatedAt = time.Now()
	
	return nil
}

func (m *InMemoryTenantManager) ListTenants(ctx context.Context) ([]*Tenant, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	tenants := make([]*Tenant, 0, len(m.tenants))
	for _, tenant := range m.tenants {
		if tenant.Status != TenantStatusDeleted {
			tenants = append(tenants, tenant)
		}
	}
	
	return tenants, nil
}

func (m *InMemoryTenantManager) SuspendTenant(ctx context.Context, tenantID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	tenant, exists := m.tenants[tenantID]
	if !exists {
		return ErrTenantNotFound
	}
	
	tenant.Status = TenantStatusSuspended
	tenant.UpdatedAt = time.Now()
	
	return nil
}

func (m *InMemoryTenantManager) ActivateTenant(ctx context.Context, tenantID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	tenant, exists := m.tenants[tenantID]
	if !exists {
		return ErrTenantNotFound
	}
	
	tenant.Status = TenantStatusActive
	tenant.UpdatedAt = time.Now()
	
	return nil
}

func (m *InMemoryTenantManager) IsolateTenantData(ctx context.Context, tenantID string) error {
	// 在实际实现中，这里应该确保：
	// 1. 租户数据存储在独立的命名空间
	// 2. 租户之间的数据完全隔离
	// 3. 数据访问时验证租户 ID
	
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	_, exists := m.tenants[tenantID]
	if !exists {
		return ErrTenantNotFound
	}
	
	// 简化实现，仅验证租户存在
	return nil
}

// TenantContextKey 租户上下文键
type tenantContextKey struct{}

// ContextWithTenant 在上下文中设置租户 ID
func ContextWithTenant(ctx context.Context, tenantID string) context.Context {
	return context.WithValue(ctx, tenantContextKey{}, tenantID)
}

// TenantFromContext 从上下文中获取租户 ID
func TenantFromContext(ctx context.Context) (string, bool) {
	tenantID, ok := ctx.Value(tenantContextKey{}).(string)
	return tenantID, ok
}

// UserContextKey 用户上下文键
type userContextKey struct{}

// ContextWithUser 在上下文中设置用户 ID
func ContextWithUser(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userContextKey{}, userID)
}

// UserFromContext 从上下文中获取用户 ID
func UserFromContext(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(userContextKey{}).(string)
	return userID, ok
}

// ContextWithAuth 在上下文中设置用户和租户信息
func ContextWithAuth(ctx context.Context, userID, tenantID string) context.Context {
	ctx = ContextWithUser(ctx, userID)
	ctx = ContextWithTenant(ctx, tenantID)
	return ctx
}
