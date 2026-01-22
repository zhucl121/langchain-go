package tenant

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// DefaultTenantManager 默认租户管理器实现
type DefaultTenantManager struct {
	store Store
	mu    sync.RWMutex

	// 内存映射（用于快速查询）
	tenants        map[string]*Tenant     // tenantID -> Tenant
	membersByUser  map[string][]*Member   // userID -> []Member
	membersByTenant map[string][]*Member  // tenantID -> []Member
}

// NewDefaultTenantManager 创建默认租户管理器
func NewDefaultTenantManager(store Store) *DefaultTenantManager {
	return &DefaultTenantManager{
		store:           store,
		tenants:         make(map[string]*Tenant),
		membersByUser:   make(map[string][]*Member),
		membersByTenant: make(map[string][]*Member),
	}
}

// CreateTenant 创建租户
func (m *DefaultTenantManager) CreateTenant(ctx context.Context, tenant *Tenant) error {
	if tenant.ID == "" {
		return ErrInvalidTenant
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// 检查是否已存在
	if _, exists := m.tenants[tenant.ID]; exists {
		return ErrTenantExists
	}

	// 设置默认值
	now := time.Now()
	tenant.CreatedAt = now
	tenant.UpdatedAt = now
	tenant.Status = StatusActive

	if tenant.Quota == nil {
		tenant.Quota = DefaultQuota()
	}
	if tenant.Usage == nil {
		tenant.Usage = NewUsage()
	}
	if tenant.Settings == nil {
		tenant.Settings = make(map[string]any)
	}
	if tenant.Metadata == nil {
		tenant.Metadata = make(map[string]any)
	}

	// 保存到存储
	if err := m.store.SaveTenant(ctx, tenant); err != nil {
		return fmt.Errorf("tenant: save tenant: %w", err)
	}

	// 更新内存映射
	m.tenants[tenant.ID] = tenant

	return nil
}

// GetTenant 获取租户
func (m *DefaultTenantManager) GetTenant(ctx context.Context, tenantID string) (*Tenant, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tenant, exists := m.tenants[tenantID]
	if !exists {
		// 尝试从存储加载
		storeTenant, err := m.store.GetTenant(ctx, tenantID)
		if err != nil {
			return nil, ErrTenantNotFound
		}
		tenant = storeTenant
		m.tenants[tenantID] = tenant
	}

	return tenant, nil
}

// UpdateTenant 更新租户
func (m *DefaultTenantManager) UpdateTenant(ctx context.Context, tenant *Tenant) error {
	if tenant.ID == "" {
		return ErrInvalidTenant
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// 检查是否存在
	if _, exists := m.tenants[tenant.ID]; !exists {
		return ErrTenantNotFound
	}

	// 更新时间戳
	tenant.UpdatedAt = time.Now()

	// 保存到存储
	if err := m.store.SaveTenant(ctx, tenant); err != nil {
		return fmt.Errorf("tenant: update tenant: %w", err)
	}

	// 更新内存映射
	m.tenants[tenant.ID] = tenant

	return nil
}

// DeleteTenant 删除租户（软删除）
func (m *DefaultTenantManager) DeleteTenant(ctx context.Context, tenantID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	tenant, exists := m.tenants[tenantID]
	if !exists {
		return ErrTenantNotFound
	}

	// 软删除
	tenant.Status = StatusDeleted
	tenant.UpdatedAt = time.Now()

	// 保存到存储
	if err := m.store.SaveTenant(ctx, tenant); err != nil {
		return fmt.Errorf("tenant: delete tenant: %w", err)
	}

	return nil
}

// ListTenants 列出租户
func (m *DefaultTenantManager) ListTenants(ctx context.Context, opts *ListOptions) ([]*Tenant, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tenants := make([]*Tenant, 0, len(m.tenants))
	for _, tenant := range m.tenants {
		// 过滤状态
		if opts != nil && opts.Status != "" && tenant.Status != opts.Status {
			continue
		}

		// 跳过已删除的租户
		if tenant.Status != StatusDeleted {
			tenants = append(tenants, tenant)
		}
	}

	// 应用限制和偏移
	if opts != nil {
		if opts.Offset > 0 && opts.Offset < len(tenants) {
			tenants = tenants[opts.Offset:]
		}
		if opts.Limit > 0 && opts.Limit < len(tenants) {
			tenants = tenants[:opts.Limit]
		}
	}

	return tenants, nil
}

// SuspendTenant 暂停租户
func (m *DefaultTenantManager) SuspendTenant(ctx context.Context, tenantID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	tenant, exists := m.tenants[tenantID]
	if !exists {
		return ErrTenantNotFound
	}

	tenant.Status = StatusSuspended
	tenant.UpdatedAt = time.Now()

	return m.store.SaveTenant(ctx, tenant)
}

// ActivateTenant 激活租户
func (m *DefaultTenantManager) ActivateTenant(ctx context.Context, tenantID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	tenant, exists := m.tenants[tenantID]
	if !exists {
		return ErrTenantNotFound
	}

	tenant.Status = StatusActive
	tenant.UpdatedAt = time.Now()

	return m.store.SaveTenant(ctx, tenant)
}

// SetQuota 设置租户配额
func (m *DefaultTenantManager) SetQuota(ctx context.Context, tenantID string, quota *Quota) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	tenant, exists := m.tenants[tenantID]
	if !exists {
		return ErrTenantNotFound
	}

	tenant.Quota = quota
	tenant.UpdatedAt = time.Now()

	return m.store.SaveTenant(ctx, tenant)
}

// GetQuota 获取租户配额
func (m *DefaultTenantManager) GetQuota(ctx context.Context, tenantID string) (*Quota, error) {
	tenant, err := m.GetTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	return tenant.Quota, nil
}

// CheckQuota 检查配额
func (m *DefaultTenantManager) CheckQuota(ctx context.Context, tenantID string, resourceType ResourceType) (*QuotaCheck, error) {
	tenant, err := m.GetTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	// 检查租户状态
	if !tenant.IsActive() {
		return &QuotaCheck{
			Allowed: false,
			Message: "tenant is not active",
		}, nil
	}

	// 检查是否过期
	if tenant.IsExpired() {
		return &QuotaCheck{
			Allowed: false,
			Message: "tenant has expired",
		}, nil
	}

	// 检查配额
	var current, limit int64

	switch resourceType {
	case ResourceTypeAgent:
		current = int64(tenant.Usage.AgentCount)
		limit = int64(tenant.Quota.MaxAgents)

	case ResourceTypeVectorStore:
		current = int64(tenant.Usage.VectorStoreCount)
		limit = int64(tenant.Quota.MaxVectorStores)

	case ResourceTypeDocument:
		current = int64(tenant.Usage.DocumentCount)
		limit = int64(tenant.Quota.MaxDocuments)

	case ResourceTypeAPICall:
		current = tenant.Usage.APICallsToday
		limit = tenant.Quota.MaxAPIRequests

	case ResourceTypeToken:
		current = tenant.Usage.TokensThisMonth
		limit = tenant.Quota.MaxTokens

	case ResourceTypeStorage:
		current = int64(tenant.Usage.StorageUsedGB * 1024) // 转换为 MB
		limit = int64(tenant.Quota.StorageGB * 1024)

	default:
		return &QuotaCheck{
			Allowed: true,
			Message: "unknown resource type, allowed by default",
		}, nil
	}

	allowed := current < limit
	remaining := limit - current
	if remaining < 0 {
		remaining = 0
	}

	message := ""
	if !allowed {
		message = fmt.Sprintf("quota exceeded for %s", resourceType)
	}

	return &QuotaCheck{
		Allowed:   allowed,
		Current:   current,
		Limit:     limit,
		Remaining: remaining,
		Message:   message,
	}, nil
}

// IncrementUsage 增加资源使用量
func (m *DefaultTenantManager) IncrementUsage(ctx context.Context, tenantID string, resourceType ResourceType, amount int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	tenant, exists := m.tenants[tenantID]
	if !exists {
		return ErrTenantNotFound
	}

	// 更新使用量
	switch resourceType {
	case ResourceTypeAgent:
		tenant.Usage.AgentCount += amount
	case ResourceTypeVectorStore:
		tenant.Usage.VectorStoreCount += amount
	case ResourceTypeDocument:
		tenant.Usage.DocumentCount += amount
	case ResourceTypeAPICall:
		tenant.Usage.APICallsToday += int64(amount)
	case ResourceTypeToken:
		tenant.Usage.TokensThisMonth += int64(amount)
	case ResourceTypeStorage:
		tenant.Usage.StorageUsedGB += float64(amount)
	}

	// 保存到存储
	return m.store.SaveTenant(ctx, tenant)
}

// DecrementUsage 减少资源使用量
func (m *DefaultTenantManager) DecrementUsage(ctx context.Context, tenantID string, resourceType ResourceType, amount int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	tenant, exists := m.tenants[tenantID]
	if !exists {
		return ErrTenantNotFound
	}

	// 更新使用量
	switch resourceType {
	case ResourceTypeAgent:
		tenant.Usage.AgentCount -= amount
		if tenant.Usage.AgentCount < 0 {
			tenant.Usage.AgentCount = 0
		}
	case ResourceTypeVectorStore:
		tenant.Usage.VectorStoreCount -= amount
		if tenant.Usage.VectorStoreCount < 0 {
			tenant.Usage.VectorStoreCount = 0
		}
	case ResourceTypeDocument:
		tenant.Usage.DocumentCount -= amount
		if tenant.Usage.DocumentCount < 0 {
			tenant.Usage.DocumentCount = 0
		}
	case ResourceTypeAPICall:
		tenant.Usage.APICallsToday -= int64(amount)
		if tenant.Usage.APICallsToday < 0 {
			tenant.Usage.APICallsToday = 0
		}
	case ResourceTypeToken:
		tenant.Usage.TokensThisMonth -= int64(amount)
		if tenant.Usage.TokensThisMonth < 0 {
			tenant.Usage.TokensThisMonth = 0
		}
	case ResourceTypeStorage:
		tenant.Usage.StorageUsedGB -= float64(amount)
		if tenant.Usage.StorageUsedGB < 0 {
			tenant.Usage.StorageUsedGB = 0
		}
	}

	// 保存到存储
	return m.store.SaveTenant(ctx, tenant)
}

// AddMember 添加成员
func (m *DefaultTenantManager) AddMember(ctx context.Context, tenantID, userID, role string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 检查租户是否存在
	if _, exists := m.tenants[tenantID]; !exists {
		return ErrTenantNotFound
	}

	// 检查成员是否已存在
	members := m.membersByTenant[tenantID]
	for _, member := range members {
		if member.UserID == userID {
			return ErrMemberExists
		}
	}

	// 创建成员
	member := &Member{
		TenantID: tenantID,
		UserID:   userID,
		Role:     role,
		JoinedAt: time.Now(),
		Metadata: make(map[string]any),
	}

	// 保存到存储
	if err := m.store.SaveMember(ctx, member); err != nil {
		return fmt.Errorf("tenant: save member: %w", err)
	}

	// 更新内存映射
	m.membersByTenant[tenantID] = append(m.membersByTenant[tenantID], member)
	m.membersByUser[userID] = append(m.membersByUser[userID], member)

	return nil
}

// RemoveMember 移除成员
func (m *DefaultTenantManager) RemoveMember(ctx context.Context, tenantID, userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 从存储删除
	if err := m.store.DeleteMember(ctx, tenantID, userID); err != nil {
		return fmt.Errorf("tenant: delete member: %w", err)
	}

	// 从内存删除
	// 从 membersByTenant 删除
	members := m.membersByTenant[tenantID]
	newMembers := make([]*Member, 0, len(members))
	for _, member := range members {
		if member.UserID != userID {
			newMembers = append(newMembers, member)
		}
	}
	m.membersByTenant[tenantID] = newMembers

	// 从 membersByUser 删除
	userMembers := m.membersByUser[userID]
	newUserMembers := make([]*Member, 0, len(userMembers))
	for _, member := range userMembers {
		if member.TenantID != tenantID {
			newUserMembers = append(newUserMembers, member)
		}
	}
	m.membersByUser[userID] = newUserMembers

	return nil
}

// GetMembers 获取租户成员
func (m *DefaultTenantManager) GetMembers(ctx context.Context, tenantID string) ([]*Member, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	members, exists := m.membersByTenant[tenantID]
	if !exists {
		// 尝试从存储加载
		storeMembers, err := m.store.GetMembers(ctx, tenantID)
		if err != nil {
			return []*Member{}, nil
		}
		members = storeMembers
		m.membersByTenant[tenantID] = members
	}

	return members, nil
}

// GetUserTenants 获取用户所属的租户
func (m *DefaultTenantManager) GetUserTenants(ctx context.Context, userID string) ([]*Tenant, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	members, exists := m.membersByUser[userID]
	if !exists {
		// 尝试从存储加载
		storeTenants, err := m.store.GetUserTenants(ctx, userID)
		if err != nil {
			return []*Tenant{}, nil
		}
		return storeTenants, nil
	}

	tenants := make([]*Tenant, 0, len(members))
	for _, member := range members {
		tenant, exists := m.tenants[member.TenantID]
		if exists && tenant.Status != StatusDeleted {
			tenants = append(tenants, tenant)
		}
	}

	return tenants, nil
}
