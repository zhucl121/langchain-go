// Package auth 提供认证和授权功能
package auth

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// 错误定义
var (
	ErrPermissionDenied      = errors.New("auth: permission denied")
	ErrInvalidUser           = errors.New("auth: invalid user")
	ErrInvalidRole           = errors.New("auth: invalid role")
	ErrInvalidResource       = errors.New("auth: invalid resource")
	ErrRoleNotFound          = errors.New("auth: role not found")
	ErrUserNotFound          = errors.New("auth: user not found")
	ErrTenantNotFound        = errors.New("auth: tenant not found")
	ErrResourceQuotaExceeded = errors.New("auth: resource quota exceeded")
)

// Action 操作类型
type Action string

const (
	// ActionRead 读取操作
	ActionRead Action = "read"
	
	// ActionWrite 写入操作
	ActionWrite Action = "write"
	
	// ActionDelete 删除操作
	ActionDelete Action = "delete"
	
	// ActionExecute 执行操作
	ActionExecute Action = "execute"
	
	// ActionAdmin 管理操作
	ActionAdmin Action = "admin"
)

// Resource 资源类型
type Resource string

const (
	// ResourceVectorStore 向量存储
	ResourceVectorStore Resource = "vectorstore"
	
	// ResourceDocument 文档
	ResourceDocument Resource = "document"
	
	// ResourceAgent 智能体
	ResourceAgent Resource = "agent"
	
	// ResourceTool 工具
	ResourceTool Resource = "tool"
	
	// ResourceMemory 记忆
	ResourceMemory Resource = "memory"
	
	// ResourceGraph 状态图
	ResourceGraph Resource = "graph"
)

// Permission 权限
type Permission struct {
	// Resource 资源类型
	Resource Resource
	
	// Action 操作类型
	Action Action
	
	// ResourceID 具体资源 ID（可选，为空表示所有该类型资源）
	ResourceID string
}

// String 返回权限的字符串表示
func (p Permission) String() string {
	if p.ResourceID != "" {
		return fmt.Sprintf("%s:%s:%s", p.Resource, p.Action, p.ResourceID)
	}
	return fmt.Sprintf("%s:%s:*", p.Resource, p.Action)
}

// Role 角色
type Role struct {
	// Name 角色名称
	Name string
	
	// Permissions 权限列表
	Permissions []Permission
	
	// Description 角色描述
	Description string
	
	// CreatedAt 创建时间
	CreatedAt time.Time
	
	// UpdatedAt 更新时间
	UpdatedAt time.Time
}

// HasPermission 检查角色是否拥有权限
func (r *Role) HasPermission(resource Resource, action Action, resourceID string) bool {
	for _, perm := range r.Permissions {
		// 资源类型匹配
		if perm.Resource != resource {
			continue
		}
		
		// 操作匹配
		if perm.Action != action && perm.Action != ActionAdmin {
			continue
		}
		
		// 资源 ID 匹配（空表示所有资源）
		if perm.ResourceID == "" || perm.ResourceID == resourceID {
			return true
		}
	}
	return false
}

// User 用户
type User struct {
	// ID 用户 ID
	ID string
	
	// Name 用户名
	Name string
	
	// Email 邮箱
	Email string
	
	// Roles 用户角色列表
	Roles []string
	
	// TenantID 租户 ID
	TenantID string
	
	// Metadata 元数据
	Metadata map[string]interface{}
	
	// CreatedAt 创建时间
	CreatedAt time.Time
	
	// UpdatedAt 更新时间
	UpdatedAt time.Time
}

// RBACManager RBAC 管理器接口
type RBACManager interface {
	// CreateRole 创建角色
	CreateRole(ctx context.Context, role *Role) error
	
	// GetRole 获取角色
	GetRole(ctx context.Context, roleName string) (*Role, error)
	
	// UpdateRole 更新角色
	UpdateRole(ctx context.Context, role *Role) error
	
	// DeleteRole 删除角色
	DeleteRole(ctx context.Context, roleName string) error
	
	// ListRoles 列出所有角色
	ListRoles(ctx context.Context) ([]*Role, error)
	
	// AssignRole 为用户分配角色
	AssignRole(ctx context.Context, userID, roleName string) error
	
	// RevokeRole 撤销用户角色
	RevokeRole(ctx context.Context, userID, roleName string) error
	
	// CheckPermission 检查用户权限
	CheckPermission(ctx context.Context, userID string, resource Resource, action Action, resourceID string) error
	
	// GetUserPermissions 获取用户的所有权限
	GetUserPermissions(ctx context.Context, userID string) ([]Permission, error)
}

// InMemoryRBACManager 内存 RBAC 管理器
type InMemoryRBACManager struct {
	mu    sync.RWMutex
	roles map[string]*Role          // roleName -> Role
	users map[string]*User          // userID -> User
	userRoles map[string][]string   // userID -> []roleName
}

// NewInMemoryRBACManager 创建内存 RBAC 管理器
func NewInMemoryRBACManager() *InMemoryRBACManager {
	manager := &InMemoryRBACManager{
		roles:     make(map[string]*Role),
		users:     make(map[string]*User),
		userRoles: make(map[string][]string),
	}
	
	// 初始化默认角色
	manager.initDefaultRoles()
	
	return manager
}

// initDefaultRoles 初始化默认角色
func (m *InMemoryRBACManager) initDefaultRoles() {
	now := time.Now()
	
	// Admin 角色（所有权限）
	m.roles["admin"] = &Role{
		Name:        "admin",
		Description: "管理员角色，拥有所有权限",
		Permissions: []Permission{
			{Resource: ResourceVectorStore, Action: ActionAdmin},
			{Resource: ResourceDocument, Action: ActionAdmin},
			{Resource: ResourceAgent, Action: ActionAdmin},
			{Resource: ResourceTool, Action: ActionAdmin},
			{Resource: ResourceMemory, Action: ActionAdmin},
			{Resource: ResourceGraph, Action: ActionAdmin},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
	
	// User 角色（基础读写权限）
	m.roles["user"] = &Role{
		Name:        "user",
		Description: "普通用户角色，基础读写权限",
		Permissions: []Permission{
			{Resource: ResourceVectorStore, Action: ActionRead},
			{Resource: ResourceVectorStore, Action: ActionWrite},
			{Resource: ResourceDocument, Action: ActionRead},
			{Resource: ResourceDocument, Action: ActionWrite},
			{Resource: ResourceAgent, Action: ActionRead},
			{Resource: ResourceAgent, Action: ActionExecute},
			{Resource: ResourceTool, Action: ActionExecute},
			{Resource: ResourceMemory, Action: ActionRead},
			{Resource: ResourceMemory, Action: ActionWrite},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
	
	// ReadOnly 角色（只读权限）
	m.roles["readonly"] = &Role{
		Name:        "readonly",
		Description: "只读角色，仅能读取资源",
		Permissions: []Permission{
			{Resource: ResourceVectorStore, Action: ActionRead},
			{Resource: ResourceDocument, Action: ActionRead},
			{Resource: ResourceAgent, Action: ActionRead},
			{Resource: ResourceMemory, Action: ActionRead},
			{Resource: ResourceGraph, Action: ActionRead},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func (m *InMemoryRBACManager) CreateRole(ctx context.Context, role *Role) error {
	if role.Name == "" {
		return ErrInvalidRole
	}
	
	m.mu.Lock()
	defer m.mu.Unlock()
	
	role.CreatedAt = time.Now()
	role.UpdatedAt = time.Now()
	m.roles[role.Name] = role
	
	return nil
}

func (m *InMemoryRBACManager) GetRole(ctx context.Context, roleName string) (*Role, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	role, exists := m.roles[roleName]
	if !exists {
		return nil, ErrRoleNotFound
	}
	
	return role, nil
}

func (m *InMemoryRBACManager) UpdateRole(ctx context.Context, role *Role) error {
	if role.Name == "" {
		return ErrInvalidRole
	}
	
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if _, exists := m.roles[role.Name]; !exists {
		return ErrRoleNotFound
	}
	
	role.UpdatedAt = time.Now()
	m.roles[role.Name] = role
	
	return nil
}

func (m *InMemoryRBACManager) DeleteRole(ctx context.Context, roleName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	delete(m.roles, roleName)
	return nil
}

func (m *InMemoryRBACManager) ListRoles(ctx context.Context) ([]*Role, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	roles := make([]*Role, 0, len(m.roles))
	for _, role := range m.roles {
		roles = append(roles, role)
	}
	
	return roles, nil
}

func (m *InMemoryRBACManager) AssignRole(ctx context.Context, userID, roleName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// 检查角色是否存在
	if _, exists := m.roles[roleName]; !exists {
		return ErrRoleNotFound
	}
	
	// 添加角色到用户
	roles := m.userRoles[userID]
	for _, r := range roles {
		if r == roleName {
			return nil // 已存在
		}
	}
	
	m.userRoles[userID] = append(m.userRoles[userID], roleName)
	return nil
}

func (m *InMemoryRBACManager) RevokeRole(ctx context.Context, userID, roleName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	roles := m.userRoles[userID]
	newRoles := make([]string, 0, len(roles))
	
	for _, r := range roles {
		if r != roleName {
			newRoles = append(newRoles, r)
		}
	}
	
	m.userRoles[userID] = newRoles
	return nil
}

func (m *InMemoryRBACManager) CheckPermission(ctx context.Context, userID string, resource Resource, action Action, resourceID string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	// 获取用户角色
	roleNames, exists := m.userRoles[userID]
	if !exists || len(roleNames) == 0 {
		return ErrPermissionDenied
	}
	
	// 检查每个角色的权限
	for _, roleName := range roleNames {
		role, exists := m.roles[roleName]
		if !exists {
			continue
		}
		
		if role.HasPermission(resource, action, resourceID) {
			return nil // 有权限
		}
	}
	
	return ErrPermissionDenied
}

func (m *InMemoryRBACManager) GetUserPermissions(ctx context.Context, userID string) ([]Permission, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	roleNames, exists := m.userRoles[userID]
	if !exists {
		return []Permission{}, nil
	}
	
	// 收集所有权限
	permMap := make(map[string]Permission)
	
	for _, roleName := range roleNames {
		role, exists := m.roles[roleName]
		if !exists {
			continue
		}
		
		for _, perm := range role.Permissions {
			key := perm.String()
			permMap[key] = perm
		}
	}
	
	// 转换为列表
	perms := make([]Permission, 0, len(permMap))
	for _, perm := range permMap {
		perms = append(perms, perm)
	}
	
	return perms, nil
}

// CreateUser 创建用户
func (m *InMemoryRBACManager) CreateUser(ctx context.Context, user *User) error {
	if user.ID == "" {
		return ErrInvalidUser
	}
	
	m.mu.Lock()
	defer m.mu.Unlock()
	
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	m.users[user.ID] = user
	
	// 分配默认角色
	if len(user.Roles) > 0 {
		m.userRoles[user.ID] = user.Roles
	} else {
		m.userRoles[user.ID] = []string{"user"} // 默认用户角色
	}
	
	return nil
}

// GetUser 获取用户
func (m *InMemoryRBACManager) GetUser(ctx context.Context, userID string) (*User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	user, exists := m.users[userID]
	if !exists {
		return nil, ErrUserNotFound
	}
	
	return user, nil
}

// UpdateUser 更新用户
func (m *InMemoryRBACManager) UpdateUser(ctx context.Context, user *User) error {
	if user.ID == "" {
		return ErrInvalidUser
	}
	
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if _, exists := m.users[user.ID]; !exists {
		return ErrUserNotFound
	}
	
	user.UpdatedAt = time.Now()
	m.users[user.ID] = user
	
	return nil
}

// DeleteUser 删除用户
func (m *InMemoryRBACManager) DeleteUser(ctx context.Context, userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	delete(m.users, userID)
	delete(m.userRoles, userID)
	
	return nil
}

// GetUserRoles 获取用户的角色
func (m *InMemoryRBACManager) GetUserRoles(ctx context.Context, userID string) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	roles, exists := m.userRoles[userID]
	if !exists {
		return []string{}, nil
	}
	
	return roles, nil
}
