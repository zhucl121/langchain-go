package rbac

import (
	"context"
	"fmt"
)

// Middleware RBAC 中间件接口
type Middleware interface {
	// BeforeExecute 执行前检查权限
	BeforeExecute(ctx context.Context, req *PermissionRequest) error

	// AfterExecute 执行后记录
	AfterExecute(ctx context.Context, req *PermissionRequest, err error) error
}

// RBACMiddleware RBAC 中间件实现
type RBACMiddleware struct {
	manager  RBACManager
	resource string // 默认资源类型
	action   string // 默认操作类型
}

// NewRBACMiddleware 创建 RBAC 中间件
func NewRBACMiddleware(manager RBACManager, resource, action string) *RBACMiddleware {
	return &RBACMiddleware{
		manager:  manager,
		resource: resource,
		action:   action,
	}
}

// BeforeExecute 执行前检查权限
func (m *RBACMiddleware) BeforeExecute(ctx context.Context, req *PermissionRequest) error {
	// 如果请求中没有指定资源和操作，使用默认值
	if req.Resource == "" {
		req.Resource = m.resource
	}
	if req.Action == "" {
		req.Action = m.action
	}

	// 检查权限
	if err := m.manager.CheckPermission(ctx, req); err != nil {
		return fmt.Errorf("rbac middleware: %w", err)
	}

	return nil
}

// AfterExecute 执行后记录
func (m *RBACMiddleware) AfterExecute(ctx context.Context, req *PermissionRequest, err error) error {
	// 这里可以添加审计日志等
	return nil
}

// CheckPermissionMiddleware 快捷权限检查中间件
func CheckPermissionMiddleware(manager RBACManager, resource, action string) func(context.Context, string) error {
	return func(ctx context.Context, userID string) error {
		req := &PermissionRequest{
			UserID:   userID,
			Resource: resource,
			Action:   action,
		}
		return manager.CheckPermission(ctx, req)
	}
}

// RequireRole 要求特定角色的中间件
func RequireRole(manager RBACManager, roleID string) func(context.Context, string) error {
	return func(ctx context.Context, userID string) error {
		roles, err := manager.GetUserRoles(ctx, userID)
		if err != nil {
			return fmt.Errorf("get user roles: %w", err)
		}

		for _, role := range roles {
			if role.ID == roleID {
				return nil
			}
		}

		return fmt.Errorf("rbac: user does not have required role %s", roleID)
	}
}

// RequireAnyRole 要求任意一个角色的中间件
func RequireAnyRole(manager RBACManager, roleIDs ...string) func(context.Context, string) error {
	return func(ctx context.Context, userID string) error {
		roles, err := manager.GetUserRoles(ctx, userID)
		if err != nil {
			return fmt.Errorf("get user roles: %w", err)
		}

		roleSet := make(map[string]bool)
		for _, role := range roles {
			roleSet[role.ID] = true
		}

		for _, requiredRoleID := range roleIDs {
			if roleSet[requiredRoleID] {
				return nil
			}
		}

		return fmt.Errorf("rbac: user does not have any of required roles: %v", roleIDs)
	}
}

// RequireAllRoles 要求所有角色的中间件
func RequireAllRoles(manager RBACManager, roleIDs ...string) func(context.Context, string) error {
	return func(ctx context.Context, userID string) error {
		roles, err := manager.GetUserRoles(ctx, userID)
		if err != nil {
			return fmt.Errorf("get user roles: %w", err)
		}

		roleSet := make(map[string]bool)
		for _, role := range roles {
			roleSet[role.ID] = true
		}

		for _, requiredRoleID := range roleIDs {
			if !roleSet[requiredRoleID] {
				return fmt.Errorf("rbac: user does not have required role %s", requiredRoleID)
			}
		}

		return nil
	}
}

// ContextKey 类型
type contextKey string

const (
	// ContextKeyUserID 用户ID上下文键
	ContextKeyUserID contextKey = "rbac:user_id"

	// ContextKeyTenantID 租户ID上下文键
	ContextKeyTenantID contextKey = "rbac:tenant_id"

	// ContextKeyRoles 角色列表上下文键
	ContextKeyRoles contextKey = "rbac:roles"
)

// WithUserID 在上下文中设置用户ID
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, ContextKeyUserID, userID)
}

// GetUserID 从上下文中获取用户ID
func GetUserID(ctx context.Context) string {
	if userID, ok := ctx.Value(ContextKeyUserID).(string); ok {
		return userID
	}
	return ""
}

// WithTenantID 在上下文中设置租户ID
func WithTenantID(ctx context.Context, tenantID string) context.Context {
	return context.WithValue(ctx, ContextKeyTenantID, tenantID)
}

// GetTenantID 从上下文中获取租户ID
func GetTenantID(ctx context.Context) string {
	if tenantID, ok := ctx.Value(ContextKeyTenantID).(string); ok {
		return tenantID
	}
	return ""
}

// WithRoles 在上下文中设置角色列表
func WithRoles(ctx context.Context, roles []*Role) context.Context {
	return context.WithValue(ctx, ContextKeyRoles, roles)
}

// GetRoles 从上下文中获取角色列表
func GetRoles(ctx context.Context) []*Role {
	if roles, ok := ctx.Value(ContextKeyRoles).([]*Role); ok {
		return roles
	}
	return nil
}

// WithAuth 在上下文中设置完整的认证信息
func WithAuth(ctx context.Context, userID, tenantID string, roles []*Role) context.Context {
	ctx = WithUserID(ctx, userID)
	ctx = WithTenantID(ctx, tenantID)
	ctx = WithRoles(ctx, roles)
	return ctx
}
