package auth

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// ResourceType 资源类型
type ResourceType string

const (
	// ResourceTypeAPICall API 调用次数
	ResourceTypeAPICall ResourceType = "api_call"
	
	// ResourceTypeTokens Token 使用量
	ResourceTypeTokens ResourceType = "tokens"
	
	// ResourceTypeDocuments 文档数量
	ResourceTypeDocuments ResourceType = "documents"
	
	// ResourceTypeVectors 向量数量
	ResourceTypeVectors ResourceType = "vectors"
	
	// ResourceTypeStorage 存储空间（字节）
	ResourceTypeStorage ResourceType = "storage"
	
	// ResourceTypeBandwidth 带宽（字节）
	ResourceTypeBandwidth ResourceType = "bandwidth"
	
	// ResourceTypeConcurrentRequests 并发请求数
	ResourceTypeConcurrentRequests ResourceType = "concurrent_requests"
)

// ResourceQuota 资源配额
type ResourceQuota struct {
	// API 调用配额
	APICallsPerDay int64
	
	// Token 使用配额
	TokensPerDay int64
	
	// 文档数量配额
	MaxDocuments int64
	
	// 向量数量配额
	MaxVectors int64
	
	// 存储空间配额（字节）
	MaxStorage int64
	
	// 带宽配额（字节/天）
	BandwidthPerDay int64
	
	// 最大并发请求数
	MaxConcurrentRequests int
	
	// 自定义配额
	Custom map[string]int64
}

// DefaultResourceQuota 返回默认配额
func DefaultResourceQuota() *ResourceQuota {
	return &ResourceQuota{
		APICallsPerDay:        10000,
		TokensPerDay:          1000000,
		MaxDocuments:          10000,
		MaxVectors:            100000,
		MaxStorage:            10 * 1024 * 1024 * 1024, // 10 GB
		BandwidthPerDay:       100 * 1024 * 1024 * 1024, // 100 GB
		MaxConcurrentRequests: 10,
		Custom:                make(map[string]int64),
	}
}

// ResourceUsage 资源使用量
type ResourceUsage struct {
	// API 调用数
	APICallsToday int64
	
	// Token 使用量
	TokensToday int64
	
	// 文档数量
	DocumentCount int64
	
	// 向量数量
	VectorCount int64
	
	// 存储使用量（字节）
	StorageUsed int64
	
	// 带宽使用量（字节）
	BandwidthToday int64
	
	// 当前并发请求数
	ConcurrentRequests int
	
	// 自定义使用量
	Custom map[string]int64
	
	// 最后重置时间
	LastResetAt time.Time
}

// QuotaManager 配额管理器接口
type QuotaManager interface {
	// CheckQuota 检查配额是否超限
	CheckQuota(ctx context.Context, tenantID string, resourceType ResourceType, amount int64) error
	
	// IncrementUsage 增加使用量
	IncrementUsage(ctx context.Context, tenantID string, resourceType ResourceType, amount int64) error
	
	// DecrementUsage 减少使用量
	DecrementUsage(ctx context.Context, tenantID string, resourceType ResourceType, amount int64) error
	
	// GetUsage 获取当前使用量
	GetUsage(ctx context.Context, tenantID string) (*ResourceUsage, error)
	
	// ResetDailyUsage 重置每日使用量
	ResetDailyUsage(ctx context.Context, tenantID string) error
}

// InMemoryQuotaManager 内存配额管理器
type InMemoryQuotaManager struct {
	mu            sync.RWMutex
	tenantManager TenantManager
	usage         map[string]*ResourceUsage
}

// NewInMemoryQuotaManager 创建内存配额管理器
func NewInMemoryQuotaManager(tenantManager TenantManager) *InMemoryQuotaManager {
	return &InMemoryQuotaManager{
		tenantManager: tenantManager,
		usage:         make(map[string]*ResourceUsage),
	}
}

func (m *InMemoryQuotaManager) CheckQuota(ctx context.Context, tenantID string, resourceType ResourceType, amount int64) error {
	tenant, err := m.tenantManager.GetTenant(ctx, tenantID)
	if err != nil {
		return err
	}
	
	if tenant.Quota == nil {
		return nil // 无配额限制
	}
	
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	usage, exists := m.usage[tenantID]
	if !exists {
		return nil // 尚未有使用记录，允许
	}
	
	// 检查每日配额（需要先重置）
	if time.Since(usage.LastResetAt) > 24*time.Hour {
		// 应该重置，但这里不直接重置（需要写锁）
		// 实际实现应该有定时任务重置
	}
	
	// 检查配额
	switch resourceType {
	case ResourceTypeAPICall:
		if usage.APICallsToday+amount > tenant.Quota.APICallsPerDay {
			return ErrResourceQuotaExceeded
		}
	case ResourceTypeTokens:
		if usage.TokensToday+amount > tenant.Quota.TokensPerDay {
			return ErrResourceQuotaExceeded
		}
	case ResourceTypeDocuments:
		if usage.DocumentCount+amount > tenant.Quota.MaxDocuments {
			return ErrResourceQuotaExceeded
		}
	case ResourceTypeVectors:
		if usage.VectorCount+amount > tenant.Quota.MaxVectors {
			return ErrResourceQuotaExceeded
		}
	case ResourceTypeStorage:
		if usage.StorageUsed+amount > tenant.Quota.MaxStorage {
			return ErrResourceQuotaExceeded
		}
	case ResourceTypeBandwidth:
		if usage.BandwidthToday+amount > tenant.Quota.BandwidthPerDay {
			return ErrResourceQuotaExceeded
		}
	case ResourceTypeConcurrentRequests:
		if usage.ConcurrentRequests+int(amount) > tenant.Quota.MaxConcurrentRequests {
			return ErrResourceQuotaExceeded
		}
	}
	
	return nil
}

func (m *InMemoryQuotaManager) IncrementUsage(ctx context.Context, tenantID string, resourceType ResourceType, amount int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	usage, exists := m.usage[tenantID]
	if !exists {
		usage = &ResourceUsage{
			LastResetAt: time.Now(),
			Custom:      make(map[string]int64),
		}
		m.usage[tenantID] = usage
	}
	
	// 增加使用量
	switch resourceType {
	case ResourceTypeAPICall:
		usage.APICallsToday += amount
	case ResourceTypeTokens:
		usage.TokensToday += amount
	case ResourceTypeDocuments:
		usage.DocumentCount += amount
	case ResourceTypeVectors:
		usage.VectorCount += amount
	case ResourceTypeStorage:
		usage.StorageUsed += amount
	case ResourceTypeBandwidth:
		usage.BandwidthToday += amount
	case ResourceTypeConcurrentRequests:
		usage.ConcurrentRequests += int(amount)
	}
	
	return nil
}

func (m *InMemoryQuotaManager) DecrementUsage(ctx context.Context, tenantID string, resourceType ResourceType, amount int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	usage, exists := m.usage[tenantID]
	if !exists {
		return nil // 无使用记录，无需减少
	}
	
	// 减少使用量
	switch resourceType {
	case ResourceTypeAPICall:
		usage.APICallsToday -= amount
		if usage.APICallsToday < 0 {
			usage.APICallsToday = 0
		}
	case ResourceTypeTokens:
		usage.TokensToday -= amount
		if usage.TokensToday < 0 {
			usage.TokensToday = 0
		}
	case ResourceTypeDocuments:
		usage.DocumentCount -= amount
		if usage.DocumentCount < 0 {
			usage.DocumentCount = 0
		}
	case ResourceTypeVectors:
		usage.VectorCount -= amount
		if usage.VectorCount < 0 {
			usage.VectorCount = 0
		}
	case ResourceTypeStorage:
		usage.StorageUsed -= amount
		if usage.StorageUsed < 0 {
			usage.StorageUsed = 0
		}
	case ResourceTypeBandwidth:
		usage.BandwidthToday -= amount
		if usage.BandwidthToday < 0 {
			usage.BandwidthToday = 0
		}
	case ResourceTypeConcurrentRequests:
		usage.ConcurrentRequests -= int(amount)
		if usage.ConcurrentRequests < 0 {
			usage.ConcurrentRequests = 0
		}
	}
	
	return nil
}

func (m *InMemoryQuotaManager) GetUsage(ctx context.Context, tenantID string) (*ResourceUsage, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	usage, exists := m.usage[tenantID]
	if !exists {
		return &ResourceUsage{
			LastResetAt: time.Now(),
			Custom:      make(map[string]int64),
		}, nil
	}
	
	return usage, nil
}

func (m *InMemoryQuotaManager) ResetDailyUsage(ctx context.Context, tenantID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	usage, exists := m.usage[tenantID]
	if !exists {
		return nil
	}
	
	// 重置每日配额
	usage.APICallsToday = 0
	usage.TokensToday = 0
	usage.BandwidthToday = 0
	usage.LastResetAt = time.Now()
	
	return nil
}

// QuotaCheckMiddleware 配额检查中间件
type QuotaCheckMiddleware struct {
	quotaManager QuotaManager
	resourceType ResourceType
	amount       int64
}

// NewQuotaCheckMiddleware 创建配额检查中间件
func NewQuotaCheckMiddleware(quotaManager QuotaManager, resourceType ResourceType, amount int64) *QuotaCheckMiddleware {
	return &QuotaCheckMiddleware{
		quotaManager: quotaManager,
		resourceType: resourceType,
		amount:       amount,
	}
}

// Check 检查并记录使用量
func (m *QuotaCheckMiddleware) Check(ctx context.Context) error {
	// 从上下文获取租户 ID
	tenantID, ok := TenantFromContext(ctx)
	if !ok {
		return fmt.Errorf("tenant ID not found in context")
	}
	
	// 检查配额
	if err := m.quotaManager.CheckQuota(ctx, tenantID, m.resourceType, m.amount); err != nil {
		return err
	}
	
	// 增加使用量
	if err := m.quotaManager.IncrementUsage(ctx, tenantID, m.resourceType, m.amount); err != nil {
		return err
	}
	
	return nil
}
