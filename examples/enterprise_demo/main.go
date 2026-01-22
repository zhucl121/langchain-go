package main

import (
	"context"
	"fmt"
	"log"
	"time"
	
	"github.com/zhucl121/langchain-go/pkg/enterprise/audit"
	"github.com/zhucl121/langchain-go/pkg/enterprise/auth"
	"github.com/zhucl121/langchain-go/pkg/enterprise/rbac"
	"github.com/zhucl121/langchain-go/pkg/enterprise/security"
	"github.com/zhucl121/langchain-go/pkg/enterprise/tenant"
)

func main() {
	fmt.Println("🏢 LangChain-Go v0.6.0 企业级功能演示")
	fmt.Println("=========================================")
	
	ctx := context.Background()
	
	// ========================================
	// 1. RBAC 权限控制演示
	// ========================================
	fmt.Println("\n📋 1. RBAC 权限控制演示")
	fmt.Println("----------------------------------------")
	
	demoRBAC(ctx)
	
	// ========================================
	// 2. 多租户隔离演示
	// ========================================
	fmt.Println("\n🏢 2. 多租户隔离演示")
	fmt.Println("----------------------------------------")
	
	demoTenant(ctx)
	
	// ========================================
	// 3. 审计日志演示
	// ========================================
	fmt.Println("\n📝 3. 审计日志演示")
	fmt.Println("----------------------------------------")
	
	demoAudit(ctx)
	
	// ========================================
	// 4. 数据安全演示
	// ========================================
	fmt.Println("\n🔒 4. 数据安全演示")
	fmt.Println("----------------------------------------")
	
	demoSecurity(ctx)
	
	// ========================================
	// 5. API 鉴权演示
	// ========================================
	fmt.Println("\n🔑 5. API 鉴权演示")
	fmt.Println("----------------------------------------")
	
	demoAuth(ctx)
	
	fmt.Println("\n✅ 演示完成！")
}

func demoRBAC(ctx context.Context) {
	// 创建 RBAC 管理器（使用内存存储）
	store := rbac.NewMemoryStore()
	rbacManager := rbac.NewDefaultRBACManager(store)
	
	// 注册内置角色
	roles := []*rbac.Role{
		rbac.RoleSystemAdmin,
		rbac.RoleTenantAdmin,
		rbac.RoleDeveloper,
		rbac.RoleViewer,
	}
	
	for _, role := range roles {
		if err := rbacManager.CreateRole(ctx, role); err != nil {
			log.Printf("Failed to create role: %v", err)
		}
	}
	
	// 分配角色给用户
	if err := rbacManager.AssignRole(ctx, "user-1", "developer"); err != nil {
		log.Printf("Failed to assign role: %v", err)
	}
	
	fmt.Println("✅ 已分配 developer 角色给 user-1")
	
	// 检查权限
	req := &rbac.PermissionRequest{
		UserID:   "user-1",
		Resource: "agent",
		Action:   "execute",
	}
	
	if err := rbacManager.CheckPermission(ctx, req); err != nil {
		fmt.Printf("❌ 权限检查失败: %v\n", err)
	} else {
		fmt.Println("✅ 权限检查通过: user-1 可以执行 agent")
	}
	
	// 检查无权限的操作
	req2 := &rbac.PermissionRequest{
		UserID:   "user-1",
		Resource: "system",
		Action:   "admin",
	}
	
	if err := rbacManager.CheckPermission(ctx, req2); err != nil {
		fmt.Printf("✅ 权限正确拒绝: %v\n", err)
	}
}

func demoTenant(ctx context.Context) {
	// 创建租户管理器（使用内存存储）
	store := tenant.NewMemoryStore()
	tenantManager := tenant.NewDefaultTenantManager(store)
	
	// 创建租户
	quota := &tenant.Quota{
		MaxAgents:       10,
		MaxVectorStores: 5,
		MaxDocuments:    10000,
		MaxAPIRequests:  1000000,
		MaxTokens:       100000000,
		StorageGB:       100,
	}
	
	t := &tenant.Tenant{
		ID:          "tenant-1",
		Name:        "示例公司",
		Description: "这是一个示例租户",
		Status:      tenant.StatusActive,
		Quota:       quota,
		CreatedAt:   time.Now(),
	}
	
	if err := tenantManager.CreateTenant(ctx, t); err != nil {
		log.Printf("Failed to create tenant: %v", err)
	}
	
	fmt.Printf("✅ 已创建租户: %s (ID: %s)\n", t.Name, t.ID)
	
	// 检查配额
	quotaCheck, err := tenantManager.CheckQuota(ctx, "tenant-1", tenant.ResourceTypeAgent)
	if err != nil {
		fmt.Printf("❌ 配额检查失败: %v\n", err)
	} else if !quotaCheck.Allowed {
		fmt.Println("❌ 配额已满，无法创建 agent")
	} else {
		fmt.Println("✅ 配额检查通过: 可以创建 agent")
	}
	
	// 增加使用量
	if err := tenantManager.IncrementUsage(ctx, "tenant-1", tenant.ResourceTypeAgent, 1); err != nil {
		log.Printf("Failed to increment usage: %v", err)
	}
	
	// 获取配额信息
	retrievedQuota, err := tenantManager.GetQuota(ctx, "tenant-1")
	if err != nil {
		log.Printf("Failed to get quota: %v", err)
	} else {
		fmt.Printf("📊 配额使用情况: Agent %d/%d\n", 
			retrievedQuota.MaxAgents, retrievedQuota.MaxAgents)
	}
}

func demoAudit(ctx context.Context) {
	// 创建审计日志记录器
	auditLogger := audit.NewMemoryAuditLogger()
	
	// 记录审计事件
	event := &audit.AuditEvent{
		TenantID:   "tenant-1",
		UserID:     "user-1",
		Action:     "agent.execute",
		Resource:   "agent",
		ResourceID: "agent-123",
		Status:     audit.StatusSuccess,
		Duration:   150 * time.Millisecond,
	}
	
	if err := auditLogger.Log(ctx, event); err != nil {
		log.Printf("Failed to log audit event: %v", err)
	}
	
	fmt.Println("✅ 已记录审计事件: agent.execute")
	
	// 查询审计日志
	query := &audit.AuditQuery{
		TenantID:  "tenant-1",
		StartTime: time.Now().Add(-1 * time.Hour),
		EndTime:   time.Now(),
	}
	
	events, err := auditLogger.Query(ctx, query)
	if err != nil {
		log.Printf("Failed to query audit log: %v", err)
	} else {
		fmt.Printf("📊 查询到 %d 条审计日志\n", len(events))
		for i, e := range events {
			fmt.Printf("  %d. %s - %s - %s\n", i+1, e.Action, e.Status, e.Timestamp.Format("15:04:05"))
		}
	}
	
	// 导出日志（JSON 格式）
	reader, err := auditLogger.Export(ctx, query, audit.ExportFormatJSON)
	if err != nil {
		log.Printf("Failed to export audit log: %v", err)
	} else {
		fmt.Println("✅ 审计日志已导出为 JSON 格式")
		_ = reader // 实际使用中可以写入文件
	}
}

func demoSecurity(ctx context.Context) {
	// 1. AES 加密演示
	fmt.Println("🔐 AES 加密演示:")
	
	key, err := security.GenerateKey()
	if err != nil {
		log.Fatalf("Failed to generate key: %v", err)
	}
	
	encryptor, err := security.NewAESEncryptor(key)
	if err != nil {
		log.Fatalf("Failed to create encryptor: %v", err)
	}
	
	plaintext := "这是敏感数据"
	ciphertext, err := encryptor.EncryptString(plaintext)
	if err != nil {
		log.Fatalf("Failed to encrypt: %v", err)
	}
	
	fmt.Printf("  明文: %s\n", plaintext)
	fmt.Printf("  密文: %s...\n", ciphertext[:30])
	
	decrypted, err := encryptor.DecryptString(ciphertext)
	if err != nil {
		log.Fatalf("Failed to decrypt: %v", err)
	}
	
	fmt.Printf("  解密: %s\n", decrypted)
	
	// 2. 数据脱敏演示
	fmt.Println("\n🎭 数据脱敏演示:")
	
	emailMasker := security.NewEmailMasker()
	phoneMasker := security.NewPhoneMasker()
	idCardMasker := security.NewIDCardMasker()
	bankCardMasker := security.NewBankCardMasker()
	
	email := "user@example.com"
	phone := "13812345678"
	idCard := "110101199001011234"
	bankCard := "6222021234567890123"
	
	fmt.Printf("  邮箱: %s -> %s\n", email, emailMasker.Mask(email))
	fmt.Printf("  手机: %s -> %s\n", phone, phoneMasker.Mask(phone))
	fmt.Printf("  身份证: %s -> %s\n", idCard, idCardMasker.Mask(idCard))
	fmt.Printf("  银行卡: %s -> %s\n", bankCard, bankCardMasker.Mask(bankCard))
}

func demoAuth(ctx context.Context) {
	// 1. JWT 认证演示
	fmt.Println("🔑 JWT 认证演示:")
	
	jwtAuth := auth.NewJWTAuthenticator("secret-key-123", "langchain-go", 24*time.Hour)
	
	// 生成 token
	token, err := jwtAuth.GenerateToken("user-1", "tenant-1", "developer")
	if err != nil {
		log.Fatalf("Failed to generate token: %v", err)
	}
	
	fmt.Printf("  Token: %s...\n", token[:50])
	
	// 验证 token
	authCtx, err := jwtAuth.Authenticate(ctx, token)
	if err != nil {
		log.Fatalf("Failed to authenticate: %v", err)
	}
	
	fmt.Printf("  ✅ 验证成功: UserID=%s, TenantID=%s\n", authCtx.UserID, authCtx.TenantID)
	
	// 2. API Key 认证演示
	fmt.Println("\n🔐 API Key 认证演示:")
	
	store := auth.NewMemoryAPIKeyStore()
	apiKeyAuth := auth.NewAPIKeyAuthenticator(store)
	
	// 生成 API Key
	apiKey, err := apiKeyAuth.GenerateAPIKey(ctx, "user-1", "tenant-1", "测试密钥", 30*24*time.Hour)
	if err != nil {
		log.Fatalf("Failed to generate API key: %v", err)
	}
	
	fmt.Printf("  API Key: %s...\n", apiKey[:30])
	
	// 验证 API Key
	authCtx2, err := apiKeyAuth.Authenticate(ctx, apiKey)
	if err != nil {
		log.Fatalf("Failed to authenticate: %v", err)
	}
	
	fmt.Printf("  ✅ 验证成功: UserID=%s, TenantID=%s\n", authCtx2.UserID, authCtx2.TenantID)
}
