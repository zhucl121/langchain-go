package auth

import (
	"context"
	"net/http"
	"strings"
)

// Authenticator 认证器接口
type Authenticator interface {
	// Authenticate 验证 token
	Authenticate(ctx context.Context, token string) (*AuthContext, error)
}

// AuthMiddleware HTTP 认证中间件
//
// 从 HTTP 请求中提取 token 并进行验证。
//
// 支持两种方式：
//   - Authorization: Bearer <token>
//   - X-API-Key: <api-key>
//
// 使用示例：
//
//	router := http.NewServeMux()
//	router.Handle("/api/", AuthMiddleware(authenticator)(handler))
func AuthMiddleware(authenticator Authenticator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 提取 token
			token := extractToken(r)
			if token == "" {
				http.Error(w, "Unauthorized: missing token", http.StatusUnauthorized)
				return
			}
			
			// 验证 token
			authCtx, err := authenticator.Authenticate(r.Context(), token)
			if err != nil {
				http.Error(w, "Unauthorized: "+err.Error(), http.StatusUnauthorized)
				return
			}
			
			// 注入认证上下文
			ctx := WithAuthContext(r.Context(), authCtx)
			
			// 继续处理
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// extractToken 从 HTTP 请求中提取 token
func extractToken(r *http.Request) string {
	// 1. 尝试从 Authorization header 提取
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		// Bearer token
		if strings.HasPrefix(authHeader, "Bearer ") {
			return strings.TrimPrefix(authHeader, "Bearer ")
		}
		// Basic auth or other schemes
		return authHeader
	}
	
	// 2. 尝试从 X-API-Key header 提取
	apiKey := r.Header.Get("X-API-Key")
	if apiKey != "" {
		return apiKey
	}
	
	// 3. 尝试从 query 参数提取（不推荐，但支持）
	token := r.URL.Query().Get("token")
	if token != "" {
		return token
	}
	
	apiKey = r.URL.Query().Get("api_key")
	if apiKey != "" {
		return apiKey
	}
	
	return ""
}

// RequireAuth 要求认证的中间件（简化版）
//
// 仅检查认证上下文是否存在，不进行权限检查。
func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, ok := GetAuthContext(r.Context())
		if !ok {
			http.Error(w, "Unauthorized: authentication required", http.StatusUnauthorized)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}

// RequireRoles 要求特定角色的中间件
func RequireRoles(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authCtx, ok := GetAuthContext(r.Context())
			if !ok {
				http.Error(w, "Unauthorized: authentication required", http.StatusUnauthorized)
				return
			}
			
			// 检查角色
			hasRole := false
			for _, requiredRole := range roles {
				for _, userRole := range authCtx.Roles {
					if userRole == requiredRole {
						hasRole = true
						break
					}
				}
				if hasRole {
					break
				}
			}
			
			if !hasRole {
				http.Error(w, "Forbidden: insufficient permissions", http.StatusForbidden)
				return
			}
			
			next.ServeHTTP(w, r)
		})
	}
}
