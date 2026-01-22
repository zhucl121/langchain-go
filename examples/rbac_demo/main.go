package main

import (
	"context"
	"fmt"
	"log"

	"github.com/zhucl121/langchain-go/pkg/enterprise/rbac"
	"github.com/zhucl121/langchain-go/pkg/enterprise/tenant"
)

func main() {
	fmt.Println("=== LangChain-Go v0.6.0 - RBAC & 多租户演示 ===\n")

	// 1. 创建 RBAC 管理器
	rbacStore := rbac.NewMemoryStore()
	rbacManager := rbac.NewDefaultRBACManager(rbacStore)
	
	ctx := context.Background()

	// 2. 创建租户管理器
	tenantStore := tenant.NewMemoryStore()
	tenantManager := tenant.NewDefaultTenantManager(tenantStore)

	// 3. 创建租户
	fmt.Println(">>> 创建租户")
	t := &tenant.Tenant{
		ID:          "tenant-demo",
		Name:        "演示公司",
		Description: "用于演示的测试租户",
		Quota: &tenant.Quota{
			MaxAgents:       100,
			MaxVectorStores: 10,
			MaxDocuments:    10000,
			MaxAPIRequests:  100000,
			MaxTokens:       10000000,
			StorageGB:       100,
		},
	}

	if err := tenantManager.CreateTenant(ctx, t); err != nil {
		log.Fatalf("创建租户失败: %v", err)
	}
	fmt.Printf("✓ 租户创建成功: %s (%s)\n\n", t.Name, t.ID)

	// 4. 添加租户成员
	fmt.Println(">>> 添加租户成员")
	users := []struct {
		userID string
		role   string
	}{
		{"user-alice", "developer"},
		{"user-bob", "viewer"},
		{"user-charlie", "data-scientist"},
	}

	for _, u := range users {
		// 分配角色
		if err := rbacManager.AssignRole(ctx, u.userID, u.role); err != nil {
			log.Fatalf("分配角色失败: %v", err)
		}

		// 添加到租户
		if err := tenantManager.AddMember(ctx, t.ID, u.userID, u.role); err != nil {
			log.Fatalf("添加成员失败: %v", err)
		}

		fmt.Printf("✓ 用户 %s 添加为 %s\n", u.userID, u.role)
	}
	fmt.Println()

	// 5. 查看租户信息
	fmt.Println(">>> 租户信息")
	members, _ := tenantManager.GetMembers(ctx, t.ID)
	fmt.Printf("租户: %s\n", t.Name)
	fmt.Printf("成员数量: %d\n", len(members))
	fmt.Printf("配额:\n")
	fmt.Printf("  - 最大 Agent 数: %d\n", t.Quota.MaxAgents)
	fmt.Printf("  - 最大向量存储: %d\n", t.Quota.MaxVectorStores)
	fmt.Printf("  - 最大文档数: %d\n", t.Quota.MaxDocuments)
	fmt.Println()

	// 6. 测试权限检查
	fmt.Println(">>> 权限检查测试")
	
	testCases := []struct {
		userID   string
		resource string
		action   string
	}{
		{"user-alice", "agent", "read"},
		{"user-alice", "agent", "write"},
		{"user-alice", "agent", "delete"},
		{"user-bob", "agent", "read"},
		{"user-bob", "agent", "write"},
		{"user-charlie", "model", "execute"},
	}

	for _, tc := range testCases {
		req := &rbac.PermissionRequest{
			UserID:   tc.userID,
			TenantID: t.ID,
			Resource: tc.resource,
			Action:   tc.action,
		}

		err := rbacManager.CheckPermission(ctx, req)
		status := "✓ 允许"
		if err != nil {
			status = "✗ 拒绝"
		}

		fmt.Printf("%s: %s 对 %s 执行 %s 操作\n", 
			status, tc.userID, tc.resource, tc.action)
	}
	fmt.Println()

	// 7. 测试配额检查
	fmt.Println(">>> 配额检查测试")
	
	// 增加使用量
	tenantManager.IncrementUsage(ctx, t.ID, tenant.ResourceTypeAgent, 50)
	tenantManager.IncrementUsage(ctx, t.ID, tenant.ResourceTypeDocument, 5000)

	// 检查配额
	quotaChecks := []tenant.ResourceType{
		tenant.ResourceTypeAgent,
		tenant.ResourceTypeDocument,
		tenant.ResourceTypeAPICall,
	}

	for _, rt := range quotaChecks {
		check, err := tenantManager.CheckQuota(ctx, t.ID, rt)
		if err != nil {
			log.Printf("配额检查失败: %v", err)
			continue
		}

		fmt.Printf("%s: %s\n", rt, check.String())
	}
	fmt.Println()

	// 8. 创建自定义角色
	fmt.Println(">>> 创建自定义角色")
	
	customRole := &rbac.Role{
		ID:          "ml-engineer",
		Name:        "机器学习工程师",
		Description: "负责模型训练和部署",
		Permissions: []*rbac.Permission{
			{Resource: "model", Actions: []string{"read", "write", "execute"}, Scope: rbac.ScopeTenant},
			{Resource: "vectorstore", Actions: []string{"read", "write"}, Scope: rbac.ScopeTenant},
			{Resource: "document", Actions: []string{"read"}, Scope: rbac.ScopeTenant},
		},
	}

	if err := rbacManager.CreateRole(ctx, customRole); err != nil {
		log.Fatalf("创建自定义角色失败: %v", err)
	}

	fmt.Printf("✓ 自定义角色创建成功: %s\n", customRole.Name)
	fmt.Printf("  权限数量: %d\n", len(customRole.Permissions))
	
	// 分配自定义角色
	if err := rbacManager.AssignRole(ctx, "user-david", "ml-engineer"); err != nil {
		log.Fatalf("分配自定义角色失败: %v", err)
	}
	fmt.Printf("✓ 用户 user-david 已分配角色: %s\n", customRole.Name)
	fmt.Println()

	// 9. 查看用户权限
	fmt.Println(">>> 查看用户权限")
	
	userRoles, _ := rbacManager.GetUserRoles(ctx, "user-alice")
	fmt.Printf("用户 user-alice 的角色:\n")
	for _, role := range userRoles {
		fmt.Printf("  - %s (%s)\n", role.Name, role.ID)
		fmt.Printf("    权限: %d 个\n", len(role.Permissions))
	}
	fmt.Println()

	// 10. 列出所有内置角色
	fmt.Println(">>> 内置角色")
	
	builtinRoles := rbac.GetBuiltinRoles()
	for _, role := range builtinRoles {
		fmt.Printf("- %s (%s)\n", role.Name, role.ID)
		fmt.Printf("  描述: %s\n", role.Description)
		fmt.Printf("  权限: %d 个\n\n", len(role.Permissions))
	}

	fmt.Println("=== 演示完成 ===")
}
