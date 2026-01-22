package tenant

import (
	"context"
	"sync"
)

// MemoryStore 内存存储实现
type MemoryStore struct {
	mu       sync.RWMutex
	tenants  map[string]*Tenant
	members  map[string][]*Member // tenantID -> []Member
	userRels map[string][]*Member // userID -> []Member
}

// NewMemoryStore 创建内存存储
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		tenants:  make(map[string]*Tenant),
		members:  make(map[string][]*Member),
		userRels: make(map[string][]*Member),
	}
}

// SaveTenant 保存租户
func (s *MemoryStore) SaveTenant(ctx context.Context, tenant *Tenant) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.tenants[tenant.ID] = tenant
	return nil
}

// GetTenant 获取租户
func (s *MemoryStore) GetTenant(ctx context.Context, tenantID string) (*Tenant, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tenant, exists := s.tenants[tenantID]
	if !exists {
		return nil, ErrTenantNotFound
	}

	return tenant, nil
}

// DeleteTenant 删除租户
func (s *MemoryStore) DeleteTenant(ctx context.Context, tenantID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.tenants, tenantID)
	return nil
}

// ListTenants 列出所有租户
func (s *MemoryStore) ListTenants(ctx context.Context, opts *ListOptions) ([]*Tenant, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tenants := make([]*Tenant, 0, len(s.tenants))
	for _, tenant := range s.tenants {
		tenants = append(tenants, tenant)
	}

	return tenants, nil
}

// SaveMember 保存成员
func (s *MemoryStore) SaveMember(ctx context.Context, member *Member) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.members[member.TenantID] = append(s.members[member.TenantID], member)
	s.userRels[member.UserID] = append(s.userRels[member.UserID], member)

	return nil
}

// DeleteMember 删除成员
func (s *MemoryStore) DeleteMember(ctx context.Context, tenantID, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 从 members 删除
	members := s.members[tenantID]
	newMembers := make([]*Member, 0, len(members))
	for _, member := range members {
		if member.UserID != userID {
			newMembers = append(newMembers, member)
		}
	}
	s.members[tenantID] = newMembers

	// 从 userRels 删除
	userMembers := s.userRels[userID]
	newUserMembers := make([]*Member, 0, len(userMembers))
	for _, member := range userMembers {
		if member.TenantID != tenantID {
			newUserMembers = append(newUserMembers, member)
		}
	}
	s.userRels[userID] = newUserMembers

	return nil
}

// GetMembers 获取租户成员
func (s *MemoryStore) GetMembers(ctx context.Context, tenantID string) ([]*Member, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	members, exists := s.members[tenantID]
	if !exists {
		return []*Member{}, nil
	}

	return members, nil
}

// GetUserTenants 获取用户所属的租户
func (s *MemoryStore) GetUserTenants(ctx context.Context, userID string) ([]*Tenant, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	members, exists := s.userRels[userID]
	if !exists {
		return []*Tenant{}, nil
	}

	tenants := make([]*Tenant, 0, len(members))
	for _, member := range members {
		tenant, exists := s.tenants[member.TenantID]
		if exists {
			tenants = append(tenants, tenant)
		}
	}

	return tenants, nil
}
