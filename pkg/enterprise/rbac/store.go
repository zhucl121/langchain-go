package rbac

import (
	"context"
	"fmt"
	"sync"
)

// MemoryStore 内存存储实现
type MemoryStore struct {
	mu           sync.RWMutex
	roles        map[string]*Role
	roleBindings map[string][]*RoleBinding // userID -> []RoleBinding
}

// NewMemoryStore 创建内存存储
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		roles:        make(map[string]*Role),
		roleBindings: make(map[string][]*RoleBinding),
	}
}

// SaveRole 保存角色
func (s *MemoryStore) SaveRole(ctx context.Context, role *Role) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.roles[role.ID] = role
	return nil
}

// GetRole 获取角色
func (s *MemoryStore) GetRole(ctx context.Context, roleID string) (*Role, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	role, exists := s.roles[roleID]
	if !exists {
		return nil, ErrRoleNotFound
	}

	return role, nil
}

// DeleteRole 删除角色
func (s *MemoryStore) DeleteRole(ctx context.Context, roleID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.roles, roleID)
	return nil
}

// ListRoles 列出所有角色
func (s *MemoryStore) ListRoles(ctx context.Context, opts *ListOptions) ([]*Role, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	roles := make([]*Role, 0, len(s.roles))
	for _, role := range s.roles {
		roles = append(roles, role)
	}

	return roles, nil
}

// SaveRoleBinding 保存角色绑定
func (s *MemoryStore) SaveRoleBinding(ctx context.Context, binding *RoleBinding) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.roleBindings[binding.UserID] = append(s.roleBindings[binding.UserID], binding)
	return nil
}

// DeleteRoleBinding 删除角色绑定
func (s *MemoryStore) DeleteRoleBinding(ctx context.Context, userID, roleID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	bindings := s.roleBindings[userID]
	newBindings := make([]*RoleBinding, 0, len(bindings))

	for _, binding := range bindings {
		if binding.RoleID != roleID {
			newBindings = append(newBindings, binding)
		}
	}

	s.roleBindings[userID] = newBindings
	return nil
}

// GetUserRoleBindings 获取用户角色绑定
func (s *MemoryStore) GetUserRoleBindings(ctx context.Context, userID string) ([]*RoleBinding, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	bindings, exists := s.roleBindings[userID]
	if !exists {
		return []*RoleBinding{}, nil
	}

	return bindings, nil
}

// LoadFromManager 从管理器加载数据（用于初始化）
func (s *MemoryStore) LoadFromManager(ctx context.Context, manager *DefaultRBACManager) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	manager.mu.RLock()
	defer manager.mu.RUnlock()

	// 复制角色
	for id, role := range manager.roles {
		s.roles[id] = role
	}

	// 复制绑定
	for userID, bindings := range manager.userBindings {
		s.roleBindings[userID] = append([]*RoleBinding{}, bindings...)
	}

	return nil
}

// SyncToManager 同步到管理器（用于热加载）
func (s *MemoryStore) SyncToManager(ctx context.Context, manager *DefaultRBACManager) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	manager.mu.Lock()
	defer manager.mu.Unlock()

	// 同步角色
	for id, role := range s.roles {
		if !IsBuiltinRole(id) {
			manager.roles[id] = role
		}
	}

	// 同步绑定
	for userID, bindings := range s.roleBindings {
		manager.userBindings[userID] = append([]*RoleBinding{}, bindings...)
	}

	// 清除缓存
	manager.cache.InvalidateAll()

	return nil
}

// validateRole 验证角色
func validateRole(role *Role) error {
	if role == nil {
		return fmt.Errorf("role cannot be nil")
	}
	if role.ID == "" {
		return fmt.Errorf("role ID cannot be empty")
	}
	if role.Name == "" {
		return fmt.Errorf("role name cannot be empty")
	}
	if len(role.Permissions) == 0 {
		return fmt.Errorf("role must have at least one permission")
	}
	return nil
}

// validateRoleBinding 验证角色绑定
func validateRoleBinding(binding *RoleBinding) error {
	if binding == nil {
		return fmt.Errorf("role binding cannot be nil")
	}
	if binding.UserID == "" {
		return fmt.Errorf("user ID cannot be empty")
	}
	if binding.RoleID == "" {
		return fmt.Errorf("role ID cannot be empty")
	}
	return nil
}
