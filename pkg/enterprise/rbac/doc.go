// Package rbac 提供基于角色的访问控制（RBAC）实现。
//
// RBAC 是企业级应用的核心安全机制，提供细粒度的权限控制。
//
// # 核心概念
//
// - Permission: 权限，定义对资源的操作（如 read, write, delete, execute）
// - Role: 角色，权限的集合
// - User: 用户，可以分配多个角色
// - Scope: 权限范围（Global, Tenant, Resource）
//
// # 使用示例
//
//	// 创建 RBAC 管理器
//	manager := rbac.NewDefaultRBACManager()
//
//	// 创建自定义角色
//	role := &rbac.Role{
//		ID:   "data-scientist",
//		Name: "数据科学家",
//		Permissions: []*rbac.Permission{
//			{Resource: "vectorstore", Actions: []string{"read", "write"}},
//			{Resource: "model", Actions: []string{"execute"}},
//		},
//	}
//	manager.CreateRole(ctx, role)
//
//	// 分配角色给用户
//	manager.AssignRole(ctx, "user123", "data-scientist")
//
//	// 检查权限
//	err := manager.CheckPermission(ctx, &rbac.PermissionRequest{
//		UserID:   "user123",
//		Resource: "vectorstore",
//		Action:   "read",
//	})
//	if err != nil {
//		// 权限被拒绝
//	}
//
// # 内置角色
//
// - system-admin: 系统管理员，拥有所有权限
// - tenant-admin: 租户管理员，租户内所有权限
// - developer: 开发者，读写和执行权限
// - viewer: 查看者，只读权限
//
// # Middleware 集成
//
//	// 创建 RBAC 中间件
//	middleware := rbac.NewRBACMiddleware(manager)
//
//	// 在 Agent 中使用
//	agent := agents.CreateAgent(agents.Config{
//		Model: model,
//		Middleware: []Middleware{middleware},
//	})
//
package rbac
