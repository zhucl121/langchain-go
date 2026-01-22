package rbac

import (
	"errors"
	"fmt"
	"time"
)

// 错误定义
var (
	ErrPermissionDenied = errors.New("rbac: permission denied")
	ErrInvalidRole      = errors.New("rbac: invalid role")
	ErrInvalidUser      = errors.New("rbac: invalid user")
	ErrRoleNotFound     = errors.New("rbac: role not found")
	ErrUserNotFound     = errors.New("rbac: user not found")
	ErrRoleExists       = errors.New("rbac: role already exists")
)

// PermissionScope 权限范围
type PermissionScope string

const (
	// ScopeGlobal 全局权限
	ScopeGlobal PermissionScope = "global"

	// ScopeTenant 租户级权限
	ScopeTenant PermissionScope = "tenant"

	// ScopeResource 资源级权限
	ScopeResource PermissionScope = "resource"
)

// Permission 权限定义
type Permission struct {
	// Resource 资源类型（如 agent, vectorstore, model）
	Resource string `json:"resource"`

	// Actions 允许的操作列表（如 read, write, delete, execute）
	// "*" 表示所有操作
	Actions []string `json:"actions"`

	// Scope 权限范围
	Scope PermissionScope `json:"scope"`

	// Conditions 条件约束（可选）
	Conditions map[string]any `json:"conditions,omitempty"`
}

// String 返回权限的字符串表示
func (p *Permission) String() string {
	return fmt.Sprintf("%s:%v@%s", p.Resource, p.Actions, p.Scope)
}

// HasAction 检查是否包含指定操作
func (p *Permission) HasAction(action string) bool {
	for _, a := range p.Actions {
		if a == "*" || a == action {
			return true
		}
	}
	return false
}

// Role 角色定义
type Role struct {
	// ID 角色唯一标识
	ID string `json:"id"`

	// Name 角色名称
	Name string `json:"name"`

	// Description 角色描述
	Description string `json:"description"`

	// Permissions 权限列表
	Permissions []*Permission `json:"permissions"`

	// IsSystem 是否为系统内置角色
	IsSystem bool `json:"is_system"`

	// CreatedAt 创建时间
	CreatedAt time.Time `json:"created_at"`

	// UpdatedAt 更新时间
	UpdatedAt time.Time `json:"updated_at"`

	// Metadata 元数据
	Metadata map[string]any `json:"metadata,omitempty"`
}

// HasPermission 检查角色是否拥有指定权限
func (r *Role) HasPermission(resource, action string) bool {
	for _, perm := range r.Permissions {
		// 资源匹配（支持通配符）
		if perm.Resource != "*" && perm.Resource != resource {
			continue
		}

		// 操作匹配
		if perm.HasAction(action) {
			return true
		}
	}
	return false
}

// PermissionRequest 权限检查请求
type PermissionRequest struct {
	// UserID 用户ID
	UserID string `json:"user_id"`

	// TenantID 租户ID（可选）
	TenantID string `json:"tenant_id,omitempty"`

	// Resource 资源类型
	Resource string `json:"resource"`

	// Action 操作类型
	Action string `json:"action"`

	// ResourceID 具体资源ID（可选）
	ResourceID string `json:"resource_id,omitempty"`

	// Context 上下文信息
	Context map[string]any `json:"context,omitempty"`
}

// ListOptions 列表查询选项
type ListOptions struct {
	// Limit 返回数量限制
	Limit int `json:"limit,omitempty"`

	// Offset 偏移量
	Offset int `json:"offset,omitempty"`

	// Filter 过滤条件
	Filter map[string]any `json:"filter,omitempty"`

	// OrderBy 排序字段
	OrderBy string `json:"order_by,omitempty"`

	// Descending 是否降序
	Descending bool `json:"descending,omitempty"`
}

// RoleBinding 角色绑定
type RoleBinding struct {
	// UserID 用户ID
	UserID string `json:"user_id"`

	// RoleID 角色ID
	RoleID string `json:"role_id"`

	// TenantID 租户ID（可选）
	TenantID string `json:"tenant_id,omitempty"`

	// ExpiresAt 过期时间（可选）
	ExpiresAt *time.Time `json:"expires_at,omitempty"`

	// CreatedAt 创建时间
	CreatedAt time.Time `json:"created_at"`
}

// IsExpired 检查角色绑定是否过期
func (rb *RoleBinding) IsExpired() bool {
	if rb.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*rb.ExpiresAt)
}
