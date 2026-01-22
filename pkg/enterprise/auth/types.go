package auth

import (
	"time"
)

// AuthContext 认证上下文
//
// 包含认证后的用户信息。
type AuthContext struct {
	// UserID 用户 ID
	UserID string `json:"user_id"`
	
	// TenantID 租户 ID
	TenantID string `json:"tenant_id"`
	
	// Roles 角色列表
	Roles []string `json:"roles"`
	
	// ExpiresAt 过期时间
	ExpiresAt time.Time `json:"expires_at"`
	
	// Metadata 额外元数据
	Metadata map[string]any `json:"metadata,omitempty"`
}

// IsExpired 是否已过期
func (c *AuthContext) IsExpired() bool {
	return time.Now().After(c.ExpiresAt)
}

// APIKey API 密钥
type APIKey struct {
	// ID 密钥 ID
	ID string `json:"id"`
	
	// Key 密钥值（哈希前）
	Key string `json:"key"`
	
	// KeyHash 密钥哈希值
	KeyHash string `json:"key_hash"`
	
	// UserID 用户 ID
	UserID string `json:"user_id"`
	
	// TenantID 租户 ID
	TenantID string `json:"tenant_id"`
	
	// Name 密钥名称
	Name string `json:"name"`
	
	// ExpiresAt 过期时间
	ExpiresAt time.Time `json:"expires_at"`
	
	// CreatedAt 创建时间
	CreatedAt time.Time `json:"created_at"`
	
	// LastUsedAt 最后使用时间
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	
	// Revoked 是否已撤销
	Revoked bool `json:"revoked"`
	
	// Metadata 额外元数据
	Metadata map[string]any `json:"metadata,omitempty"`
}

// IsExpired 是否已过期
func (k *APIKey) IsExpired() bool {
	return time.Now().After(k.ExpiresAt)
}

// IsRevoked 是否已撤销
func (k *APIKey) IsRevoked() bool {
	return k.Revoked
}

// IsValid 是否有效（未过期且未撤销）
func (k *APIKey) IsValid() bool {
	return !k.IsExpired() && !k.IsRevoked()
}
