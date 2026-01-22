package main

import (
	"context"
	"fmt"
	"testing"
	"time"
	
	"github.com/zhucl121/langchain-go/pkg/enterprise/audit"
	"github.com/zhucl121/langchain-go/pkg/enterprise/auth"
	"github.com/zhucl121/langchain-go/pkg/enterprise/rbac"
	"github.com/zhucl121/langchain-go/pkg/enterprise/security"
	"github.com/zhucl121/langchain-go/pkg/enterprise/tenant"
)

// TestRBAC 测试 RBAC 功能
func TestRBAC(t *testing.T) {
	ctx := context.Background()
	store := rbac.NewMemoryStore()
	manager := rbac.NewDefaultRBACManager(store)
	
	// 分配角色
	err := manager.AssignRole(ctx, "test-user", "developer")
	if err != nil {
		t.Fatalf("AssignRole failed: %v", err)
	}
	
	// 检查权限
	req := &rbac.PermissionRequest{
		UserID:   "test-user",
		Resource: "agent",
		Action:   "execute",
	}
	err = manager.CheckPermission(ctx, req)
	if err != nil {
		t.Fatalf("CheckPermission failed: %v", err)
	}
	
	fmt.Println("✅ RBAC 测试通过")
}

// TestTenant 测试多租户功能
func TestTenant(t *testing.T) {
	ctx := context.Background()
	store := tenant.NewMemoryStore()
	manager := tenant.NewDefaultTenantManager(store)
	
	// 创建租户
	ten := &tenant.Tenant{
		ID:   "test-tenant",
		Name: "测试租户",
		Quota: &tenant.Quota{
			MaxAgents: 10,
		},
	}
	err := manager.CreateTenant(ctx, ten)
	if err != nil {
		t.Fatalf("CreateTenant failed: %v", err)
	}
	
	// 检查配额
	check, err := manager.CheckQuota(ctx, "test-tenant", tenant.ResourceTypeAgent)
	if err != nil {
		t.Fatalf("CheckQuota failed: %v", err)
	}
	if !check.Allowed {
		t.Fatal("CheckQuota should allow")
	}
	
	fmt.Println("✅ Tenant 测试通过")
}

// TestAudit 测试审计日志功能
func TestAudit(t *testing.T) {
	ctx := context.Background()
	logger := audit.NewMemoryAuditLogger()
	
	// 记录事件
	event := &audit.AuditEvent{
		TenantID: "test-tenant",
		UserID:   "test-user",
		Action:   "test.action",
		Status:   audit.StatusSuccess,
	}
	err := logger.Log(ctx, event)
	if err != nil {
		t.Fatalf("Log failed: %v", err)
	}
	
	// 查询日志
	query := &audit.AuditQuery{
		TenantID:  "test-tenant",
		StartTime: time.Now().Add(-1 * time.Hour),
		EndTime:   time.Now(),
	}
	events, err := logger.Query(ctx, query)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("Expected 1 event, got %d", len(events))
	}
	
	fmt.Println("✅ Audit 测试通过")
}

// TestSecurity 测试数据安全功能
func TestSecurity(t *testing.T) {
	// 测试加密
	key, err := security.GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey failed: %v", err)
	}
	
	encryptor, err := security.NewAESEncryptor(key)
	if err != nil {
		t.Fatalf("NewAESEncryptor failed: %v", err)
	}
	
	plaintext := "test data"
	ciphertext, err := encryptor.EncryptString(plaintext)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}
	
	decrypted, err := encryptor.DecryptString(ciphertext)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}
	if decrypted != plaintext {
		t.Fatalf("Decrypted text mismatch: got %s, want %s", decrypted, plaintext)
	}
	
	// 测试脱敏
	emailMasker := security.NewEmailMasker()
	masked := emailMasker.Mask("test@example.com")
	if masked != "t***@example.com" {
		t.Fatalf("Email masking failed: got %s", masked)
	}
	
	phoneMasker := security.NewPhoneMasker()
	masked = phoneMasker.Mask("13812345678")
	if masked != "138****5678" {
		t.Fatalf("Phone masking failed: got %s", masked)
	}
	
	fmt.Println("✅ Security 测试通过")
}

// TestAuth 测试 API 鉴权功能
func TestAuth(t *testing.T) {
	ctx := context.Background()
	
	// 测试 JWT
	jwtAuth := auth.NewJWTAuthenticator("test-secret", "test-app", 24*time.Hour)
	token, err := jwtAuth.GenerateToken("test-user", "test-tenant")
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}
	
	authCtx, err := jwtAuth.Authenticate(ctx, token)
	if err != nil {
		t.Fatalf("Authenticate failed: %v", err)
	}
	if authCtx.UserID != "test-user" {
		t.Fatalf("UserID mismatch: got %s, want test-user", authCtx.UserID)
	}
	
	// 测试 API Key
	store := auth.NewMemoryAPIKeyStore()
	apiKeyAuth := auth.NewAPIKeyAuthenticator(store)
	apiKey, err := apiKeyAuth.GenerateAPIKey(ctx, "test-user", "test-tenant", "test-key", 30*24*time.Hour)
	if err != nil {
		t.Fatalf("GenerateAPIKey failed: %v", err)
	}
	
	authCtx2, err := apiKeyAuth.Authenticate(ctx, apiKey)
	if err != nil {
		t.Fatalf("Authenticate failed: %v", err)
	}
	if authCtx2.UserID != "test-user" {
		t.Fatalf("UserID mismatch: got %s, want test-user", authCtx2.UserID)
	}
	
	fmt.Println("✅ Auth 测试通过")
}
