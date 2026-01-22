// Package tenant 提供多租户隔离和管理功能。
//
// 租户（Tenant）是资源和数据隔离的基本单位，用于实现 SaaS 多租户架构。
//
// # 核心功能
//
// - 租户管理：创建、更新、删除、查询租户
// - 配额管理：限制租户的资源使用
// - 数据隔离：确保租户数据完全隔离
// - 成员管理：管理租户成员和角色
//
// # 使用示例
//
//	// 创建租户管理器
//	manager := tenant.NewDefaultTenantManager(store, quotaManager)
//
//	// 创建租户
//	t := &tenant.Tenant{
//		ID:   "tenant-123",
//		Name: "测试公司",
//		Quota: &tenant.Quota{
//			MaxAgents: 100,
//			MaxVectorStores: 10,
//			MaxDocuments: 10000,
//		},
//	}
//	manager.CreateTenant(ctx, t)
//
//	// 添加成员
//	manager.AddMember(ctx, "tenant-123", "user-456", "developer")
//
//	// 检查配额
//	err := manager.CheckQuota(ctx, "tenant-123", tenant.ResourceTypeAPICall)
//
// # 数据隔离
//
//	// 在上下文中设置租户ID
//	ctx = tenant.WithTenantID(ctx, "tenant-123")
//
//	// 隔离向量存储
//	isolatedStore := tenant.IsolateVectorStore(vectorStore, "tenant-123")
//
package tenant
