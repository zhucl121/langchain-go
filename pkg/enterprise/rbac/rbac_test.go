package rbac

import (
	"context"
	"testing"
	"time"
)

func TestDefaultRBACManager_CreateRole(t *testing.T) {
	store := NewMemoryStore()
	manager := NewDefaultRBACManager(store)
	ctx := context.Background()

	role := &Role{
		ID:          "test-role",
		Name:        "测试角色",
		Description: "测试用角色",
		Permissions: []*Permission{
			{Resource: "agent", Actions: []string{"read", "write"}, Scope: ScopeTenant},
		},
	}

	err := manager.CreateRole(ctx, role)
	if err != nil {
		t.Fatalf("CreateRole failed: %v", err)
	}

	// 验证角色创建成功
	got, err := manager.GetRole(ctx, "test-role")
	if err != nil {
		t.Fatalf("GetRole failed: %v", err)
	}

	if got.ID != role.ID {
		t.Errorf("Role ID mismatch: got %s, want %s", got.ID, role.ID)
	}
	if got.Name != role.Name {
		t.Errorf("Role Name mismatch: got %s, want %s", got.Name, role.Name)
	}
}

func TestDefaultRBACManager_CreateBuiltinRole(t *testing.T) {
	store := NewMemoryStore()
	manager := NewDefaultRBACManager(store)
	ctx := context.Background()

	// 尝试创建内置角色应该失败
	role := &Role{
		ID:   "system-admin",
		Name: "系统管理员",
		Permissions: []*Permission{
			{Resource: "*", Actions: []string{"*"}, Scope: ScopeGlobal},
		},
	}

	err := manager.CreateRole(ctx, role)
	if err == nil {
		t.Fatal("Expected error when creating builtin role, got nil")
	}
}

func TestDefaultRBACManager_UpdateRole(t *testing.T) {
	store := NewMemoryStore()
	manager := NewDefaultRBACManager(store)
	ctx := context.Background()

	// 创建角色
	role := &Role{
		ID:          "test-role",
		Name:        "测试角色",
		Description: "原始描述",
		Permissions: []*Permission{
			{Resource: "agent", Actions: []string{"read"}, Scope: ScopeTenant},
		},
	}
	manager.CreateRole(ctx, role)

	// 更新角色
	role.Description = "更新后的描述"
	role.Permissions = append(role.Permissions, &Permission{
		Resource: "model",
		Actions:  []string{"execute"},
		Scope:    ScopeTenant,
	})

	err := manager.UpdateRole(ctx, role)
	if err != nil {
		t.Fatalf("UpdateRole failed: %v", err)
	}

	// 验证更新成功
	got, _ := manager.GetRole(ctx, "test-role")
	if got.Description != "更新后的描述" {
		t.Errorf("Description not updated: got %s", got.Description)
	}
	if len(got.Permissions) != 2 {
		t.Errorf("Permissions count mismatch: got %d, want 2", len(got.Permissions))
	}
}

func TestDefaultRBACManager_DeleteRole(t *testing.T) {
	store := NewMemoryStore()
	manager := NewDefaultRBACManager(store)
	ctx := context.Background()

	// 创建角色
	role := &Role{
		ID:   "test-role",
		Name: "测试角色",
		Permissions: []*Permission{
			{Resource: "agent", Actions: []string{"read"}, Scope: ScopeTenant},
		},
	}
	manager.CreateRole(ctx, role)

	// 删除角色
	err := manager.DeleteRole(ctx, "test-role")
	if err != nil {
		t.Fatalf("DeleteRole failed: %v", err)
	}

	// 验证删除成功
	_, err = manager.GetRole(ctx, "test-role")
	if err != ErrRoleNotFound {
		t.Errorf("Expected ErrRoleNotFound, got %v", err)
	}
}

func TestDefaultRBACManager_AssignRole(t *testing.T) {
	store := NewMemoryStore()
	manager := NewDefaultRBACManager(store)
	ctx := context.Background()

	// 分配角色
	err := manager.AssignRole(ctx, "user1", "developer")
	if err != nil {
		t.Fatalf("AssignRole failed: %v", err)
	}

	// 验证角色分配成功
	roles, err := manager.GetUserRoles(ctx, "user1")
	if err != nil {
		t.Fatalf("GetUserRoles failed: %v", err)
	}

	if len(roles) != 1 {
		t.Fatalf("Expected 1 role, got %d", len(roles))
	}
	if roles[0].ID != "developer" {
		t.Errorf("Role ID mismatch: got %s, want developer", roles[0].ID)
	}
}

func TestDefaultRBACManager_RevokeRole(t *testing.T) {
	store := NewMemoryStore()
	manager := NewDefaultRBACManager(store)
	ctx := context.Background()

	// 分配并撤销角色
	manager.AssignRole(ctx, "user1", "developer")
	err := manager.RevokeRole(ctx, "user1", "developer")
	if err != nil {
		t.Fatalf("RevokeRole failed: %v", err)
	}

	// 验证角色撤销成功
	roles, _ := manager.GetUserRoles(ctx, "user1")
	if len(roles) != 0 {
		t.Errorf("Expected 0 roles, got %d", len(roles))
	}
}

func TestDefaultRBACManager_CheckPermission(t *testing.T) {
	store := NewMemoryStore()
	manager := NewDefaultRBACManager(store)
	ctx := context.Background()

	// 分配开发者角色
	manager.AssignRole(ctx, "user1", "developer")

	tests := []struct {
		name      string
		req       *PermissionRequest
		wantError bool
	}{
		{
			name: "允许读取 agent",
			req: &PermissionRequest{
				UserID:   "user1",
				Resource: "agent",
				Action:   "read",
			},
			wantError: false,
		},
		{
			name: "允许执行 agent",
			req: &PermissionRequest{
				UserID:   "user1",
				Resource: "agent",
				Action:   "execute",
			},
			wantError: false,
		},
		{
			name: "不允许删除 agent",
			req: &PermissionRequest{
				UserID:   "user1",
				Resource: "agent",
				Action:   "delete",
			},
			wantError: true,
		},
		{
			name: "不允许写入 tool",
			req: &PermissionRequest{
				UserID:   "user1",
				Resource: "tool",
				Action:   "write",
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.CheckPermission(ctx, tt.req)
			if (err != nil) != tt.wantError {
				t.Errorf("CheckPermission() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestDefaultRBACManager_CheckPermissionCache(t *testing.T) {
	store := NewMemoryStore()
	manager := NewDefaultRBACManager(store)
	ctx := context.Background()

	// 分配角色
	manager.AssignRole(ctx, "user1", "developer")

	req := &PermissionRequest{
		UserID:   "user1",
		Resource: "agent",
		Action:   "read",
	}

	// 第一次检查（未缓存）
	err := manager.CheckPermission(ctx, req)
	if err != nil {
		t.Fatalf("First CheckPermission failed: %v", err)
	}

	// 第二次检查（应该使用缓存）
	err = manager.CheckPermission(ctx, req)
	if err != nil {
		t.Fatalf("Second CheckPermission failed: %v", err)
	}

	// 验证缓存生效
	if !manager.cache.Has(req) {
		t.Error("Expected permission to be cached")
	}
}

func TestDefaultRBACManager_GrantPermission(t *testing.T) {
	store := NewMemoryStore()
	manager := NewDefaultRBACManager(store)
	ctx := context.Background()

	// 创建角色
	role := &Role{
		ID:   "custom-role",
		Name: "自定义角色",
		Permissions: []*Permission{
			{Resource: "agent", Actions: []string{"read"}, Scope: ScopeTenant},
		},
	}
	manager.CreateRole(ctx, role)

	// 授予新权限
	newPerm := &Permission{
		Resource: "model",
		Actions:  []string{"execute"},
		Scope:    ScopeTenant,
	}
	err := manager.GrantPermission(ctx, "custom-role", newPerm)
	if err != nil {
		t.Fatalf("GrantPermission failed: %v", err)
	}

	// 验证权限已添加
	got, _ := manager.GetRole(ctx, "custom-role")
	if len(got.Permissions) != 2 {
		t.Errorf("Expected 2 permissions, got %d", len(got.Permissions))
	}
}

func TestDefaultRBACManager_RevokePermission(t *testing.T) {
	store := NewMemoryStore()
	manager := NewDefaultRBACManager(store)
	ctx := context.Background()

	// 创建角色
	role := &Role{
		ID:   "custom-role",
		Name: "自定义角色",
		Permissions: []*Permission{
			{Resource: "agent", Actions: []string{"read"}, Scope: ScopeTenant},
			{Resource: "model", Actions: []string{"execute"}, Scope: ScopeTenant},
		},
	}
	manager.CreateRole(ctx, role)

	// 撤销权限
	perm := &Permission{
		Resource: "model",
		Actions:  []string{"execute"},
		Scope:    ScopeTenant,
	}
	err := manager.RevokePermission(ctx, "custom-role", perm)
	if err != nil {
		t.Fatalf("RevokePermission failed: %v", err)
	}

	// 验证权限已移除
	got, _ := manager.GetRole(ctx, "custom-role")
	if len(got.Permissions) != 1 {
		t.Errorf("Expected 1 permission, got %d", len(got.Permissions))
	}
}

func TestRoleBinding_IsExpired(t *testing.T) {
	now := time.Now()
	past := now.Add(-1 * time.Hour)
	future := now.Add(1 * time.Hour)

	tests := []struct {
		name    string
		binding *RoleBinding
		want    bool
	}{
		{
			name: "无过期时间",
			binding: &RoleBinding{
				UserID: "user1",
				RoleID: "developer",
			},
			want: false,
		},
		{
			name: "已过期",
			binding: &RoleBinding{
				UserID:    "user1",
				RoleID:    "developer",
				ExpiresAt: &past,
			},
			want: true,
		},
		{
			name: "未过期",
			binding: &RoleBinding{
				UserID:    "user1",
				RoleID:    "developer",
				ExpiresAt: &future,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.binding.IsExpired(); got != tt.want {
				t.Errorf("IsExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPermission_HasAction(t *testing.T) {
	tests := []struct {
		name       string
		permission *Permission
		action     string
		want       bool
	}{
		{
			name: "完全匹配",
			permission: &Permission{
				Resource: "agent",
				Actions:  []string{"read", "write"},
			},
			action: "read",
			want:   true,
		},
		{
			name: "通配符匹配",
			permission: &Permission{
				Resource: "agent",
				Actions:  []string{"*"},
			},
			action: "delete",
			want:   true,
		},
		{
			name: "不匹配",
			permission: &Permission{
				Resource: "agent",
				Actions:  []string{"read"},
			},
			action: "write",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.permission.HasAction(tt.action); got != tt.want {
				t.Errorf("HasAction() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRole_HasPermission(t *testing.T) {
	role := &Role{
		ID:   "test-role",
		Name: "测试角色",
		Permissions: []*Permission{
			{Resource: "agent", Actions: []string{"read", "write"}, Scope: ScopeTenant},
			{Resource: "model", Actions: []string{"*"}, Scope: ScopeTenant},
		},
	}

	tests := []struct {
		name     string
		resource string
		action   string
		want     bool
	}{
		{
			name:     "允许读取 agent",
			resource: "agent",
			action:   "read",
			want:     true,
		},
		{
			name:     "允许写入 agent",
			resource: "agent",
			action:   "write",
			want:     true,
		},
		{
			name:     "不允许删除 agent",
			resource: "agent",
			action:   "delete",
			want:     false,
		},
		{
			name:     "允许 model 任意操作（通配符）",
			resource: "model",
			action:   "delete",
			want:     true,
		},
		{
			name:     "不允许操作 tool",
			resource: "tool",
			action:   "read",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := role.HasPermission(tt.resource, tt.action); got != tt.want {
				t.Errorf("HasPermission() = %v, want %v", got, tt.want)
			}
		})
	}
}

func BenchmarkCheckPermission(b *testing.B) {
	store := NewMemoryStore()
	manager := NewDefaultRBACManager(store)
	ctx := context.Background()

	manager.AssignRole(ctx, "user1", "developer")

	req := &PermissionRequest{
		UserID:   "user1",
		Resource: "agent",
		Action:   "read",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		manager.CheckPermission(ctx, req)
	}
}

func BenchmarkCheckPermissionCached(b *testing.B) {
	store := NewMemoryStore()
	manager := NewDefaultRBACManager(store)
	ctx := context.Background()

	manager.AssignRole(ctx, "user1", "developer")

	req := &PermissionRequest{
		UserID:   "user1",
		Resource: "agent",
		Action:   "read",
	}

	// 预热缓存
	manager.CheckPermission(ctx, req)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		manager.CheckPermission(ctx, req)
	}
}
