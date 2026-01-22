package tenant

import "context"

// ContextKey 类型
type contextKey string

const (
	// ContextKeyTenantID 租户ID上下文键
	ContextKeyTenantID contextKey = "tenant:id"

	// ContextKeyTenant 租户对象上下文键
	ContextKeyTenant contextKey = "tenant:object"
)

// WithTenantID 在上下文中设置租户ID
func WithTenantID(ctx context.Context, tenantID string) context.Context {
	return context.WithValue(ctx, ContextKeyTenantID, tenantID)
}

// GetTenantID 从上下文中获取租户ID
func GetTenantID(ctx context.Context) (string, bool) {
	tenantID, ok := ctx.Value(ContextKeyTenantID).(string)
	return tenantID, ok
}

// MustGetTenantID 从上下文中获取租户ID（必须存在）
func MustGetTenantID(ctx context.Context) string {
	tenantID, ok := GetTenantID(ctx)
	if !ok {
		panic("tenant ID not found in context")
	}
	return tenantID
}

// WithTenant 在上下文中设置租户对象
func WithTenant(ctx context.Context, tenant *Tenant) context.Context {
	ctx = WithTenantID(ctx, tenant.ID)
	return context.WithValue(ctx, ContextKeyTenant, tenant)
}

// GetTenant 从上下文中获取租户对象
func GetTenant(ctx context.Context) (*Tenant, bool) {
	tenant, ok := ctx.Value(ContextKeyTenant).(*Tenant)
	return tenant, ok
}
