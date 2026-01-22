package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"sync"
	"time"
	
	"github.com/google/uuid"
)

var (
	// ErrInvalidAPIKey 无效的 API Key
	ErrInvalidAPIKey = errors.New("auth: invalid api key")
	
	// ErrAPIKeyExpired API Key 已过期
	ErrAPIKeyExpired = errors.New("auth: api key expired")
	
	// ErrAPIKeyRevoked API Key 已撤销
	ErrAPIKeyRevoked = errors.New("auth: api key revoked")
)

// APIKeyStore API Key 存储接口
type APIKeyStore interface {
	// Save 保存 API Key
	Save(ctx context.Context, apiKey *APIKey) error
	
	// Get 获取 API Key（通过哈希值）
	Get(ctx context.Context, keyHash string) (*APIKey, error)
	
	// Delete 删除 API Key
	Delete(ctx context.Context, id string) error
	
	// List 列出用户的所有 API Key
	List(ctx context.Context, userID string) ([]*APIKey, error)
}

// MemoryAPIKeyStore 内存 API Key 存储
type MemoryAPIKeyStore struct {
	keys map[string]*APIKey // keyHash -> APIKey
	mu   sync.RWMutex
}

// NewMemoryAPIKeyStore 创建内存 API Key 存储
func NewMemoryAPIKeyStore() *MemoryAPIKeyStore {
	return &MemoryAPIKeyStore{
		keys: make(map[string]*APIKey),
	}
}

// Save 保存 API Key
func (s *MemoryAPIKeyStore) Save(ctx context.Context, apiKey *APIKey) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.keys[apiKey.KeyHash] = apiKey
	return nil
}

// Get 获取 API Key
func (s *MemoryAPIKeyStore) Get(ctx context.Context, keyHash string) (*APIKey, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	apiKey, ok := s.keys[keyHash]
	if !ok {
		return nil, ErrInvalidAPIKey
	}
	
	return apiKey, nil
}

// Delete 删除 API Key
func (s *MemoryAPIKeyStore) Delete(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// 查找并删除
	for hash, key := range s.keys {
		if key.ID == id {
			delete(s.keys, hash)
			return nil
		}
	}
	
	return ErrInvalidAPIKey
}

// List 列出用户的所有 API Key
func (s *MemoryAPIKeyStore) List(ctx context.Context, userID string) ([]*APIKey, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	var keys []*APIKey
	for _, key := range s.keys {
		if key.UserID == userID {
			keys = append(keys, key)
		}
	}
	
	return keys, nil
}

// APIKeyAuthenticator API Key 认证器
type APIKeyAuthenticator struct {
	store APIKeyStore
}

// NewAPIKeyAuthenticator 创建 API Key 认证器
func NewAPIKeyAuthenticator(store APIKeyStore) *APIKeyAuthenticator {
	return &APIKeyAuthenticator{
		store: store,
	}
}

// GenerateAPIKey 生成 API Key
//
// 参数：
//   - userID: 用户 ID
//   - tenantID: 租户 ID
//   - name: 密钥名称
//   - expiry: 有效期
//
// 返回：
//   - string: API Key（仅此时返回，不再存储明文）
//   - error: 错误
func (a *APIKeyAuthenticator) GenerateAPIKey(ctx context.Context, userID, tenantID, name string, expiry time.Duration) (string, error) {
	// 生成随机密钥（32 字节 = 256 bits）
	keyBytes := make([]byte, 32)
	if _, err := rand.Read(keyBytes); err != nil {
		return "", fmt.Errorf("auth: failed to generate random key: %w", err)
	}
	
	// Base64 编码
	keyString := base64.URLEncoding.EncodeToString(keyBytes)
	
	// 计算哈希
	hash := sha256.Sum256([]byte(keyString))
	keyHash := base64.URLEncoding.EncodeToString(hash[:])
	
	// 创建 API Key
	now := time.Now()
	apiKey := &APIKey{
		ID:        uuid.New().String(),
		Key:       "", // 不存储明文
		KeyHash:   keyHash,
		UserID:    userID,
		TenantID:  tenantID,
		Name:      name,
		ExpiresAt: now.Add(expiry),
		CreatedAt: now,
		Revoked:   false,
	}
	
	// 保存
	if err := a.store.Save(ctx, apiKey); err != nil {
		return "", fmt.Errorf("auth: failed to save api key: %w", err)
	}
	
	// 返回密钥（仅此时返回）
	return keyString, nil
}

// Authenticate 验证 API Key
//
// 参数：
//   - ctx: 上下文
//   - keyString: API Key 字符串
//
// 返回：
//   - *AuthContext: 认证上下文
//   - error: 错误
func (a *APIKeyAuthenticator) Authenticate(ctx context.Context, keyString string) (*AuthContext, error) {
	// 计算哈希
	hash := sha256.Sum256([]byte(keyString))
	keyHash := base64.URLEncoding.EncodeToString(hash[:])
	
	// 查询 API Key
	apiKey, err := a.store.Get(ctx, keyHash)
	if err != nil {
		return nil, err
	}
	
	// 检查是否撤销
	if apiKey.IsRevoked() {
		return nil, ErrAPIKeyRevoked
	}
	
	// 检查是否过期
	if apiKey.IsExpired() {
		return nil, ErrAPIKeyExpired
	}
	
	// 更新最后使用时间
	now := time.Now()
	apiKey.LastUsedAt = &now
	a.store.Save(ctx, apiKey)
	
	// 构建认证上下文
	authCtx := &AuthContext{
		UserID:    apiKey.UserID,
		TenantID:  apiKey.TenantID,
		ExpiresAt: apiKey.ExpiresAt,
		Metadata: map[string]any{
			"api_key_id":   apiKey.ID,
			"api_key_name": apiKey.Name,
		},
	}
	
	return authCtx, nil
}

// RevokeAPIKey 撤销 API Key
//
// 参数：
//   - ctx: 上下文
//   - id: API Key ID
//
// 返回：
//   - error: 错误
func (a *APIKeyAuthenticator) RevokeAPIKey(ctx context.Context, id string) error {
	// 查找密钥
	// 注意：这里需要遍历存储，实际生产中应该有 ID -> APIKey 的索引
	keys, err := a.store.List(ctx, "")
	if err != nil {
		return err
	}
	
	for _, key := range keys {
		if key.ID == id {
			key.Revoked = true
			return a.store.Save(ctx, key)
		}
	}
	
	return ErrInvalidAPIKey
}

// ListAPIKeys 列出用户的所有 API Key
//
// 参数：
//   - ctx: 上下文
//   - userID: 用户 ID
//
// 返回：
//   - []*APIKey: API Key 列表
//   - error: 错误
func (a *APIKeyAuthenticator) ListAPIKeys(ctx context.Context, userID string) ([]*APIKey, error) {
	return a.store.List(ctx, userID)
}
