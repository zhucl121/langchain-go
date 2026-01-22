package rbac

import (
	"context"
)

// RBACManager RBAC 管理器接口
type RBACManager interface {
	// CreateRole 创建角色
	CreateRole(ctx context.Context, role *Role) error

	// UpdateRole 更新角色
	UpdateRole(ctx context.Context, role *Role) error

	// DeleteRole 删除角色
	DeleteRole(ctx context.Context, roleID string) error

	// GetRole 获取角色
	GetRole(ctx context.Context, roleID string) (*Role, error)

	// ListRoles 列出所有角色
	ListRoles(ctx context.Context, opts *ListOptions) ([]*Role, error)

	// AssignRole 为用户分配角色
	AssignRole(ctx context.Context, userID, roleID string) error

	// RevokeRole 撤销用户角色
	RevokeRole(ctx context.Context, userID, roleID string) error

	// GetUserRoles 获取用户的所有角色
	GetUserRoles(ctx context.Context, userID string) ([]*Role, error)

	// CheckPermission 检查用户权限
	CheckPermission(ctx context.Context, req *PermissionRequest) error

	// GrantPermission 向角色授予权限
	GrantPermission(ctx context.Context, roleID string, perm *Permission) error

	// RevokePermission 从角色撤销权限
	RevokePermission(ctx context.Context, roleID string, perm *Permission) error
}

// Store 存储接口
type Store interface {
	// Role 相关
	SaveRole(ctx context.Context, role *Role) error
	GetRole(ctx context.Context, roleID string) (*Role, error)
	DeleteRole(ctx context.Context, roleID string) error
	ListRoles(ctx context.Context, opts *ListOptions) ([]*Role, error)

	// RoleBinding 相关
	SaveRoleBinding(ctx context.Context, binding *RoleBinding) error
	DeleteRoleBinding(ctx context.Context, userID, roleID string) error
	GetUserRoleBindings(ctx context.Context, userID string) ([]*RoleBinding, error)
}
