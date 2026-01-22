package tenant

import (
	"context"
	"testing"
	"time"
)

func TestDefaultTenantManager_CreateTenant(t *testing.T) {
	store := NewMemoryStore()
	manager := NewDefaultTenantManager(store)
	ctx := context.Background()

	tenant := &Tenant{
		ID:          "test-tenant",
		Name:        "测试租户",
		Description: "测试用租户",
	}

	err := manager.CreateTenant(ctx, tenant)
	if err != nil {
		t.Fatalf("CreateTenant failed: %v", err)
	}

	// 验证租户创建成功
	got, err := manager.GetTenant(ctx, "test-tenant")
	if err != nil {
		t.Fatalf("GetTenant failed: %v", err)
	}

	if got.ID != tenant.ID {
		t.Errorf("Tenant ID mismatch: got %s, want %s", got.ID, tenant.ID)
	}
	if got.Status != StatusActive {
		t.Errorf("Expected status active, got %s", got.Status)
	}
	if got.Quota == nil {
		t.Error("Expected default quota, got nil")
	}
}

func TestDefaultTenantManager_QuotaCheck(t *testing.T) {
	store := NewMemoryStore()
	manager := NewDefaultTenantManager(store)
	ctx := context.Background()

	// 创建租户
	tenant := &Tenant{
		ID:   "test-tenant",
		Name: "测试租户",
		Quota: &Quota{
			MaxAgents: 10,
		},
	}
	manager.CreateTenant(ctx, tenant)

	// 检查配额
	check, err := manager.CheckQuota(ctx, "test-tenant", ResourceTypeAgent)
	if err != nil {
		t.Fatalf("CheckQuota failed: %v", err)
	}

	if !check.Allowed {
		t.Error("Expected quota check to pass")
	}
	if check.Limit != 10 {
		t.Errorf("Expected limit 10, got %d", check.Limit)
	}
}

func TestDefaultTenantManager_IncrementUsage(t *testing.T) {
	store := NewMemoryStore()
	manager := NewDefaultTenantManager(store)
	ctx := context.Background()

	// 创建租户
	tenant := &Tenant{
		ID:   "test-tenant",
		Name: "测试租户",
	}
	manager.CreateTenant(ctx, tenant)

	// 增加使用量
	err := manager.IncrementUsage(ctx, "test-tenant", ResourceTypeAgent, 5)
	if err != nil {
		t.Fatalf("IncrementUsage failed: %v", err)
	}

	// 验证使用量
	got, _ := manager.GetTenant(ctx, "test-tenant")
	if got.Usage.AgentCount != 5 {
		t.Errorf("Expected AgentCount 5, got %d", got.Usage.AgentCount)
	}
}

func TestDefaultTenantManager_Member(t *testing.T) {
	store := NewMemoryStore()
	manager := NewDefaultTenantManager(store)
	ctx := context.Background()

	// 创建租户
	tenant := &Tenant{
		ID:   "test-tenant",
		Name: "测试租户",
	}
	manager.CreateTenant(ctx, tenant)

	// 添加成员
	err := manager.AddMember(ctx, "test-tenant", "user1", "developer")
	if err != nil {
		t.Fatalf("AddMember failed: %v", err)
	}

	// 获取成员
	members, err := manager.GetMembers(ctx, "test-tenant")
	if err != nil {
		t.Fatalf("GetMembers failed: %v", err)
	}

	if len(members) != 1 {
		t.Fatalf("Expected 1 member, got %d", len(members))
	}
	if members[0].UserID != "user1" {
		t.Errorf("Expected UserID user1, got %s", members[0].UserID)
	}
	if members[0].Role != "developer" {
		t.Errorf("Expected role developer, got %s", members[0].Role)
	}
}

func TestTenant_IsExpired(t *testing.T) {
	now := time.Now()
	past := now.Add(-1 * time.Hour)
	future := now.Add(1 * time.Hour)

	tests := []struct {
		name   string
		tenant *Tenant
		want   bool
	}{
		{
			name: "无过期时间",
			tenant: &Tenant{
				ID: "tenant1",
			},
			want: false,
		},
		{
			name: "已过期",
			tenant: &Tenant{
				ID:        "tenant2",
				ExpiresAt: &past,
			},
			want: true,
		},
		{
			name: "未过期",
			tenant: &Tenant{
				ID:        "tenant3",
				ExpiresAt: &future,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.tenant.IsExpired(); got != tt.want {
				t.Errorf("IsExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWithTenantID(t *testing.T) {
	ctx := context.Background()

	// 设置租户ID
	ctx = WithTenantID(ctx, "tenant-123")

	// 获取租户ID
	tenantID, ok := GetTenantID(ctx)
	if !ok {
		t.Fatal("Expected tenant ID in context")
	}

	if tenantID != "tenant-123" {
		t.Errorf("Expected tenant-123, got %s", tenantID)
	}
}

func TestQuotaCheck_String(t *testing.T) {
	tests := []struct {
		name  string
		check *QuotaCheck
		want  string
	}{
		{
			name: "允许",
			check: &QuotaCheck{
				Allowed:   true,
				Current:   5,
				Limit:     10,
				Remaining: 5,
			},
			want: "Allowed: 5/10 (remaining: 5)",
		},
		{
			name: "拒绝",
			check: &QuotaCheck{
				Allowed: false,
				Current: 15,
				Limit:   10,
				Message: "quota exceeded",
			},
			want: "Denied: 15/10 (quota exceeded)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.check.String()
			if got != tt.want {
				t.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}
