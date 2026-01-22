// Package auth 提供企业级 API 鉴权功能。
//
// 核心功能：
//   - JWT 生成和验证
//   - API Key 生成和验证
//   - Token 刷新机制
//   - Token 撤销（黑名单）
//   - HTTP 认证中间件
//
// 使用示例：
//
//	// JWT 认证
//	jwtAuth := auth.NewJWTAuthenticator("secret-key", "myapp", 24*time.Hour)
//	token, _ := jwtAuth.GenerateToken("user-123", "tenant-123")
//	authCtx, _ := jwtAuth.Authenticate(ctx, token)
//
//	// API Key 认证
//	apiKeyAuth := auth.NewAPIKeyAuthenticator(store)
//	apiKey, _ := apiKeyAuth.GenerateAPIKey("user-123", "tenant-123", 30*24*time.Hour)
//	authCtx, _ := apiKeyAuth.Authenticate(ctx, apiKey)
//
//	// HTTP 中间件
//	router := http.NewServeMux()
//	router.Handle("/api/", auth.AuthMiddleware(jwtAuth)(handler))
//
package auth
