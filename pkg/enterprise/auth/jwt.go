package auth

import (
	"context"
	"errors"
	"fmt"
	"time"
	
	"github.com/golang-jwt/jwt/v5"
)

var (
	// ErrInvalidToken 无效的 token
	ErrInvalidToken = errors.New("auth: invalid token")
	
	// ErrExpiredToken 过期的 token
	ErrExpiredToken = errors.New("auth: expired token")
	
	// ErrTokenRevoked token 已撤销
	ErrTokenRevoked = errors.New("auth: token revoked")
)

// JWTClaims JWT 声明
type JWTClaims struct {
	UserID   string   `json:"user_id"`
	TenantID string   `json:"tenant_id"`
	Roles    []string `json:"roles,omitempty"`
	jwt.RegisteredClaims
}

// JWTAuthenticator JWT 认证器
type JWTAuthenticator struct {
	secretKey []byte
	issuer    string
	expiry    time.Duration
}

// NewJWTAuthenticator 创建 JWT 认证器
//
// 参数：
//   - secretKey: 密钥
//   - issuer: 签发者
//   - expiry: 有效期
//
// 返回：
//   - *JWTAuthenticator: JWT 认证器
func NewJWTAuthenticator(secretKey string, issuer string, expiry time.Duration) *JWTAuthenticator {
	return &JWTAuthenticator{
		secretKey: []byte(secretKey),
		issuer:    issuer,
		expiry:    expiry,
	}
}

// GenerateToken 生成 JWT token
//
// 参数：
//   - userID: 用户 ID
//   - tenantID: 租户 ID
//   - roles: 角色列表（可选）
//
// 返回：
//   - string: JWT token
//   - error: 错误
func (a *JWTAuthenticator) GenerateToken(userID, tenantID string, roles ...string) (string, error) {
	now := time.Now()
	expiresAt := now.Add(a.expiry)
	
	claims := &JWTClaims{
		UserID:   userID,
		TenantID: tenantID,
		Roles:    roles,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    a.issuer,
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}
	
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(a.secretKey)
	if err != nil {
		return "", fmt.Errorf("auth: failed to sign token: %w", err)
	}
	
	return tokenString, nil
}

// Authenticate 验证 JWT token
//
// 参数：
//   - ctx: 上下文
//   - tokenString: JWT token 字符串
//
// 返回：
//   - *AuthContext: 认证上下文
//   - error: 错误
func (a *JWTAuthenticator) Authenticate(ctx context.Context, tokenString string) (*AuthContext, error) {
	// 解析 token
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		// 验证签名算法
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return a.secretKey, nil
	})
	
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}
	
	// 提取 claims
	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}
	
	// 检查过期
	if claims.ExpiresAt != nil && claims.ExpiresAt.Before(time.Now()) {
		return nil, ErrExpiredToken
	}
	
	// 构建认证上下文
	authCtx := &AuthContext{
		UserID:    claims.UserID,
		TenantID:  claims.TenantID,
		Roles:     claims.Roles,
		ExpiresAt: claims.ExpiresAt.Time,
	}
	
	return authCtx, nil
}

// RefreshToken 刷新 token
//
// 参数：
//   - ctx: 上下文
//   - tokenString: 旧 token
//
// 返回：
//   - string: 新 token
//   - error: 错误
func (a *JWTAuthenticator) RefreshToken(ctx context.Context, tokenString string) (string, error) {
	// 验证旧 token
	authCtx, err := a.Authenticate(ctx, tokenString)
	if err != nil {
		return "", err
	}
	
	// 生成新 token
	return a.GenerateToken(authCtx.UserID, authCtx.TenantID, authCtx.Roles...)
}

// contextKey 是认证上下文键类型
type contextKey string

const (
	// authContextKey 认证上下文键
	authContextKey contextKey = "auth.context"
)

// WithAuthContext 在 context 中保存认证上下文
func WithAuthContext(ctx context.Context, authCtx *AuthContext) context.Context {
	return context.WithValue(ctx, authContextKey, authCtx)
}

// GetAuthContext 从 context 获取认证上下文
func GetAuthContext(ctx context.Context) (*AuthContext, bool) {
	authCtx, ok := ctx.Value(authContextKey).(*AuthContext)
	return authCtx, ok
}

// MustGetAuthContext 获取认证上下文（必须存在）
func MustGetAuthContext(ctx context.Context) (*AuthContext, error) {
	authCtx, ok := GetAuthContext(ctx)
	if !ok {
		return nil, errors.New("auth: no auth context in context")
	}
	return authCtx, nil
}

// GetUserID 从 context 获取用户 ID
func GetUserID(ctx context.Context) string {
	authCtx, ok := GetAuthContext(ctx)
	if !ok {
		return ""
	}
	return authCtx.UserID
}

// GetTenantID 从 context 获取租户 ID
func GetTenantID(ctx context.Context) string {
	authCtx, ok := GetAuthContext(ctx)
	if !ok {
		return ""
	}
	return authCtx.TenantID
}
