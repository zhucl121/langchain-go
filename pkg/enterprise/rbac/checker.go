package rbac

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// DefaultRBACManager 默认 RBAC 管理器实现
type DefaultRBACManager struct {
	store Store
	cache *PermissionCache
	mu    sync.RWMutex

	// 内存映射（用于快速查询）
	roles        map[string]*Role          // roleID -> Role
	userBindings map[string][]*RoleBinding // userID -> []RoleBinding
}

// NewDefaultRBACManager 创建默认 RBAC 管理器
func NewDefaultRBACManager(store Store) *DefaultRBACManager {
	manager := &DefaultRBACManager{
		store:        store,
		cache:        NewPermissionCache(10000, 5*time.Minute),
		roles:        make(map[string]*Role),
		userBindings: make(map[string][]*RoleBinding),
	}

	// 初始化内置角色
	manager.initBuiltinRoles()

	return manager
}

// initBuiltinRoles 初始化内置角色
func (m *DefaultRBACManager) initBuiltinRoles() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, role := range GetBuiltinRoles() {
		m.roles[role.ID] = role
	}
}

// CreateRole 创建角色
func (m *DefaultRBACManager) CreateRole(ctx context.Context, role *Role) error {
	if role.ID == "" {
		return ErrInvalidRole
	}

	// 检查是否为内置角色
	if IsBuiltinRole(role.ID) {
		return fmt.Errorf("rbac: cannot create builtin role %s", role.ID)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// 检查是否已存在
	if _, exists := m.roles[role.ID]; exists {
		return ErrRoleExists
	}

	// 设置时间戳
	now := time.Now()
	role.CreatedAt = now
	role.UpdatedAt = now

	// 保存到存储
	if err := m.store.SaveRole(ctx, role); err != nil {
		return fmt.Errorf("rbac: save role: %w", err)
	}

	// 更新内存映射
	m.roles[role.ID] = role

	return nil
}

// UpdateRole 更新角色
func (m *DefaultRBACManager) UpdateRole(ctx context.Context, role *Role) error {
	if role.ID == "" {
		return ErrInvalidRole
	}

	// 检查是否为内置角色
	if IsBuiltinRole(role.ID) {
		return fmt.Errorf("rbac: cannot update builtin role %s", role.ID)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// 检查是否存在
	if _, exists := m.roles[role.ID]; !exists {
		return ErrRoleNotFound
	}

	// 更新时间戳
	role.UpdatedAt = time.Now()

	// 保存到存储
	if err := m.store.SaveRole(ctx, role); err != nil {
		return fmt.Errorf("rbac: update role: %w", err)
	}

	// 更新内存映射
	m.roles[role.ID] = role

	// 清除相关缓存
	m.cache.InvalidateAll()

	return nil
}

// DeleteRole 删除角色
func (m *DefaultRBACManager) DeleteRole(ctx context.Context, roleID string) error {
	// 检查是否为内置角色
	if IsBuiltinRole(roleID) {
		return fmt.Errorf("rbac: cannot delete builtin role %s", roleID)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// 从存储删除
	if err := m.store.DeleteRole(ctx, roleID); err != nil {
		return fmt.Errorf("rbac: delete role: %w", err)
	}

	// 从内存删除
	delete(m.roles, roleID)

	// 清除相关缓存
	m.cache.InvalidateAll()

	return nil
}

// GetRole 获取角色
func (m *DefaultRBACManager) GetRole(ctx context.Context, roleID string) (*Role, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	role, exists := m.roles[roleID]
	if !exists {
		return nil, ErrRoleNotFound
	}

	return role, nil
}

// ListRoles 列出所有角色
func (m *DefaultRBACManager) ListRoles(ctx context.Context, opts *ListOptions) ([]*Role, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	roles := make([]*Role, 0, len(m.roles))
	for _, role := range m.roles {
		roles = append(roles, role)
	}

	// 应用限制和偏移
	if opts != nil {
		if opts.Offset > 0 && opts.Offset < len(roles) {
			roles = roles[opts.Offset:]
		}
		if opts.Limit > 0 && opts.Limit < len(roles) {
			roles = roles[:opts.Limit]
		}
	}

	return roles, nil
}

// AssignRole 为用户分配角色
func (m *DefaultRBACManager) AssignRole(ctx context.Context, userID, roleID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 检查角色是否存在
	if _, exists := m.roles[roleID]; !exists {
		return ErrRoleNotFound
	}

	// 检查是否已分配
	bindings := m.userBindings[userID]
	for _, binding := range bindings {
		if binding.RoleID == roleID && !binding.IsExpired() {
			return nil // 已经分配
		}
	}

	// 创建角色绑定
	binding := &RoleBinding{
		UserID:    userID,
		RoleID:    roleID,
		CreatedAt: time.Now(),
	}

	// 保存到存储
	if err := m.store.SaveRoleBinding(ctx, binding); err != nil {
		return fmt.Errorf("rbac: save role binding: %w", err)
	}

	// 更新内存映射
	m.userBindings[userID] = append(m.userBindings[userID], binding)

	// 清除用户权限缓存
	m.cache.InvalidateUser(userID)

	return nil
}

// RevokeRole 撤销用户角色
func (m *DefaultRBACManager) RevokeRole(ctx context.Context, userID, roleID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 从存储删除
	if err := m.store.DeleteRoleBinding(ctx, userID, roleID); err != nil {
		return fmt.Errorf("rbac: delete role binding: %w", err)
	}

	// 从内存删除
	bindings := m.userBindings[userID]
	newBindings := make([]*RoleBinding, 0, len(bindings))
	for _, binding := range bindings {
		if binding.RoleID != roleID {
			newBindings = append(newBindings, binding)
		}
	}
	m.userBindings[userID] = newBindings

	// 清除用户权限缓存
	m.cache.InvalidateUser(userID)

	return nil
}

// GetUserRoles 获取用户的所有角色
func (m *DefaultRBACManager) GetUserRoles(ctx context.Context, userID string) ([]*Role, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 先尝试从内存获取
	bindings, exists := m.userBindings[userID]
	if !exists || len(bindings) == 0 {
		// 如果内存中没有，尝试从存储加载
		storeBindings, err := m.store.GetUserRoleBindings(ctx, userID)
		if err != nil {
			return []*Role{}, nil
		}
		bindings = storeBindings
	}

	roles := make([]*Role, 0, len(bindings))

	for _, binding := range bindings {
		// 跳过过期的绑定
		if binding.IsExpired() {
			continue
		}

		role, exists := m.roles[binding.RoleID]
		if exists {
			roles = append(roles, role)
		}
	}

	return roles, nil
}

// CheckPermission 检查用户权限
func (m *DefaultRBACManager) CheckPermission(ctx context.Context, req *PermissionRequest) error {
	// 检查缓存
	if m.cache.Has(req) {
		return nil
	}

	// 获取用户角色
	roles, err := m.GetUserRoles(ctx, req.UserID)
	if err != nil {
		return fmt.Errorf("rbac: get user roles: %w", err)
	}

	if len(roles) == 0 {
		return ErrPermissionDenied
	}

	// 检查每个角色的权限
	for _, role := range roles {
		for _, perm := range role.Permissions {
			if m.matchPermission(perm, req) {
				// 权限匹配，缓存结果
				m.cache.Set(req)
				return nil
			}
		}
	}

	return ErrPermissionDenied
}

// matchPermission 匹配权限
func (m *DefaultRBACManager) matchPermission(perm *Permission, req *PermissionRequest) bool {
	// 1. 资源匹配
	if perm.Resource != "*" && perm.Resource != req.Resource {
		return false
	}

	// 2. 操作匹配
	if !perm.HasAction(req.Action) {
		return false
	}

	// 3. 范围匹配
	switch perm.Scope {
	case ScopeGlobal:
		// 全局权限，匹配所有
		return true

	case ScopeTenant:
		// 租户权限，当前简化为默认允许
		// 实际生产中需要检查用户和请求的租户ID是否匹配
		return true

	case ScopeResource:
		// 资源权限，需要资源ID匹配
		// 这里可以增加更细粒度的检查
		return true

	default:
		// 默认允许（向后兼容）
		return true
	}
}

// GrantPermission 向角色授予权限
func (m *DefaultRBACManager) GrantPermission(ctx context.Context, roleID string, perm *Permission) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	role, exists := m.roles[roleID]
	if !exists {
		return ErrRoleNotFound
	}

	// 检查是否已存在
	for _, p := range role.Permissions {
		if p.Resource == perm.Resource && p.Scope == perm.Scope {
			// 合并操作
			p.Actions = append(p.Actions, perm.Actions...)
			goto save
		}
	}

	// 添加新权限
	role.Permissions = append(role.Permissions, perm)

save:
	// 更新时间戳
	role.UpdatedAt = time.Now()

	// 保存到存储
	if err := m.store.SaveRole(ctx, role); err != nil {
		return fmt.Errorf("rbac: save role: %w", err)
	}

	// 清除相关缓存
	m.cache.InvalidateAll()

	return nil
}

// RevokePermission 从角色撤销权限
func (m *DefaultRBACManager) RevokePermission(ctx context.Context, roleID string, perm *Permission) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	role, exists := m.roles[roleID]
	if !exists {
		return ErrRoleNotFound
	}

	// 移除权限
	newPermissions := make([]*Permission, 0, len(role.Permissions))
	for _, p := range role.Permissions {
		if p.Resource != perm.Resource || p.Scope != perm.Scope {
			newPermissions = append(newPermissions, p)
		}
	}
	role.Permissions = newPermissions

	// 更新时间戳
	role.UpdatedAt = time.Now()

	// 保存到存储
	if err := m.store.SaveRole(ctx, role); err != nil {
		return fmt.Errorf("rbac: save role: %w", err)
	}

	// 清除相关缓存
	m.cache.InvalidateAll()

	return nil
}
