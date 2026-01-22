# 企业级功能综合演示

这个示例演示了 LangChain-Go v0.6.0 的所有企业级功能。

## 功能演示

### 1. RBAC 权限控制

- 创建和管理角色
- 分配角色给用户
- 检查权限

### 2. 多租户隔离

- 创建和管理租户
- 配额管理
- 使用量追踪

### 3. 审计日志

- 记录审计事件
- 查询审计日志
- 导出日志（JSON/CSV）

### 4. 数据安全

- AES-256-GCM 加密
- 数据脱敏（邮箱、手机号、身份证、银行卡）

### 5. API 鉴权

- JWT 生成和验证
- API Key 生成和验证

## 运行示例

```bash
cd examples/enterprise_demo
go run main.go
```

## 输出示例

```
🏢 LangChain-Go v0.6.0 企业级功能演示
=========================================

📋 1. RBAC 权限控制演示
----------------------------------------
✅ 已分配 developer 角色给 user-1
✅ 权限检查通过: user-1 可以执行 agent
✅ 权限正确拒绝: rbac: permission denied

🏢 2. 多租户隔离演示
----------------------------------------
✅ 已创建租户: 示例公司 (ID: tenant-1)
✅ 配额检查通过: 可以创建 agent
📊 配额使用情况: Agent 10/10

📝 3. 审计日志演示
----------------------------------------
✅ 已记录审计事件: agent.execute
📊 查询到 1 条审计日志
  1. agent.execute - success - 14:30:45
✅ 审计日志已导出为 JSON 格式

🔒 4. 数据安全演示
----------------------------------------
🔐 AES 加密演示:
  明文: 这是敏感数据
  密文: 5K+b6L+Y5piv5Y+R...
  解密: 这是敏感数据

🎭 数据脱敏演示:
  邮箱: user@example.com -> u***@example.com
  手机: 13812345678 -> 138****5678
  身份证: 110101199001011234 -> 110101********1234
  银行卡: 6222021234567890123 -> 6222********0123

🔑 5. API 鉴权演示
----------------------------------------
🔑 JWT 认证演示:
  Token: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
  ✅ 验证成功: UserID=user-1, TenantID=tenant-1

🔐 API Key 认证演示:
  API Key: sk_abc123...
  ✅ 验证成功: UserID=user-1, TenantID=tenant-1

✅ 演示完成！
```

## 实际使用

在实际应用中，可以这样组合使用：

```go
// 1. 创建 HTTP 服务器
router := http.NewServeMux()

// 2. 设置认证中间件
jwtAuth := auth.NewJWTAuthenticator("secret", "app", 24*time.Hour)
router.Handle("/api/", auth.AuthMiddleware(jwtAuth)(apiHandler))

// 3. 在处理器中检查权限
func apiHandler(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    
    // 获取认证信息
    authCtx, _ := auth.GetAuthContext(ctx)
    
    // 检查权限
    req := &rbac.PermissionRequest{
        UserID:   authCtx.UserID,
        Resource: "agent",
        Action:   "execute",
    }
    if err := rbacManager.CheckPermission(ctx, req); err != nil {
        http.Error(w, "Forbidden", http.StatusForbidden)
        return
    }
    
    // 记录审计日志
    event := &audit.AuditEvent{
        TenantID: authCtx.TenantID,
        UserID:   authCtx.UserID,
        Action:   "agent.execute",
        Status:   audit.StatusSuccess,
    }
    auditLogger.Log(ctx, event)
    
    // 处理请求...
}
```

## 相关文档

- [v0.6.0 用户指南](../../docs/V0.6.0_USER_GUIDE.md)
- [v0.6.0 安全指南](../../docs/V0.6.0_SECURITY_GUIDE.md)
- [v0.6.0 API 参考](../../docs/V0.6.0_API_REFERENCE.md)
