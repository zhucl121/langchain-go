package cache

import (
	"context"
	"log"
	"sync"
	"time"
)

// Warmer 缓存预热器
type Warmer struct {
	cache    DistributedCache
	strategy WarmupStrategy
	config   WarmerConfig
	stopCh   chan struct{}
	wg       sync.WaitGroup
}

// WarmerConfig 预热器配置
type WarmerConfig struct {
	// Interval 预热间隔
	Interval time.Duration

	// BatchSize 批量预热大小
	BatchSize int

	// Timeout 预热超时
	Timeout time.Duration

	// Workers 并发工作者数
	Workers int
}

// DefaultWarmerConfig 返回默认配置
func DefaultWarmerConfig() WarmerConfig {
	return WarmerConfig{
		Interval:  1 * time.Hour,
		BatchSize: 100,
		Timeout:   30 * time.Second,
		Workers:   5,
	}
}

// NewWarmer 创建缓存预热器
func NewWarmer(cache DistributedCache, strategy WarmupStrategy, config WarmerConfig) *Warmer {
	return &Warmer{
		cache:    cache,
		strategy: strategy,
		config:   config,
		stopCh:   make(chan struct{}),
	}
}

// Start 启动预热
func (w *Warmer) Start(ctx context.Context) {
	ticker := time.NewTicker(w.config.Interval)
	defer ticker.Stop()

	// 首次立即预热
	w.warmup(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-w.stopCh:
			return
		case <-ticker.C:
			w.warmup(ctx)
		}
	}
}

// Stop 停止预热
func (w *Warmer) Stop() {
	close(w.stopCh)
	w.wg.Wait()
}

// WarmupNow 立即预热
func (w *Warmer) WarmupNow(ctx context.Context) error {
	return w.warmup(ctx)
}

// warmup 执行预热
func (w *Warmer) warmup(ctx context.Context) error {
	keys := w.strategy.GetWarmupKeys()
	if len(keys) == 0 {
		return nil
	}

	log.Printf("开始缓存预热，共 %d 个键", len(keys))

	// 创建任务队列
	jobs := make(chan string, len(keys))
	for _, key := range keys {
		jobs <- key
	}
	close(jobs)

	// 启动工作者
	var wg sync.WaitGroup
	for i := 0; i < w.config.Workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w.worker(ctx, jobs)
		}()
	}

	wg.Wait()
	log.Printf("缓存预热完成")

	return nil
}

// worker 工作者
func (w *Warmer) worker(ctx context.Context, jobs <-chan string) {
	for key := range jobs {
		if !w.strategy.ShouldWarmup(key) {
			continue
		}

		// 检查是否已存在
		exists, err := w.cache.Exists(ctx, key)
		if err == nil && exists {
			continue
		}

		// 加载数据
		data, err := w.strategy.LoadData(ctx, key)
		if err != nil {
			log.Printf("加载预热数据失败: key=%s, error=%v", key, err)
			continue
		}

		// 写入缓存
		if err := w.cache.Set(ctx, key, data, 24*time.Hour); err != nil {
			log.Printf("写入预热数据失败: key=%s, error=%v", key, err)
		}
	}
}

// SimpleWarmupStrategy 简单预热策略
type SimpleWarmupStrategy struct {
	keys     []string
	loader   func(ctx context.Context, key string) ([]byte, error)
	patterns []string
}

// NewSimpleWarmupStrategy 创建简单预热策略
func NewSimpleWarmupStrategy(
	keys []string,
	loader func(ctx context.Context, key string) ([]byte, error),
) *SimpleWarmupStrategy {
	return &SimpleWarmupStrategy{
		keys:   keys,
		loader: loader,
	}
}

// ShouldWarmup 判断是否需要预热
func (s *SimpleWarmupStrategy) ShouldWarmup(key string) bool {
	for _, k := range s.keys {
		if k == key {
			return true
		}
	}
	return false
}

// GetWarmupKeys 获取需要预热的键列表
func (s *SimpleWarmupStrategy) GetWarmupKeys() []string {
	return s.keys
}

// LoadData 加载预热数据
func (s *SimpleWarmupStrategy) LoadData(ctx context.Context, key string) ([]byte, error) {
	if s.loader == nil {
		return nil, ErrCacheNotFound
	}
	return s.loader(ctx, key)
}

// TTLInvalidationStrategy TTL 失效策略
type TTLInvalidationStrategy struct {
	ttl time.Duration
}

// NewTTLInvalidationStrategy 创建 TTL 失效策略
func NewTTLInvalidationStrategy(ttl time.Duration) *TTLInvalidationStrategy {
	return &TTLInvalidationStrategy{ttl: ttl}
}

// ShouldInvalidate 判断是否应该失效
func (s *TTLInvalidationStrategy) ShouldInvalidate(entry *CacheEntry) bool {
	if entry.ExpiresAt.IsZero() {
		return false
	}
	return time.Now().After(entry.ExpiresAt)
}

// OnInvalidate 失效时的回调
func (s *TTLInvalidationStrategy) OnInvalidate(key string, value []byte) {
	log.Printf("缓存失效: key=%s", key)
}

// AccessTimeInvalidationStrategy 访问时间失效策略
type AccessTimeInvalidationStrategy struct {
	maxIdleTime time.Duration
}

// NewAccessTimeInvalidationStrategy 创建访问时间失效策略
func NewAccessTimeInvalidationStrategy(maxIdleTime time.Duration) *AccessTimeInvalidationStrategy {
	return &AccessTimeInvalidationStrategy{maxIdleTime: maxIdleTime}
}

// ShouldInvalidate 判断是否应该失效
func (s *AccessTimeInvalidationStrategy) ShouldInvalidate(entry *CacheEntry) bool {
	return time.Since(entry.LastAccessAt) > s.maxIdleTime
}

// OnInvalidate 失效时的回调
func (s *AccessTimeInvalidationStrategy) OnInvalidate(key string, value []byte) {
	log.Printf("缓存失效（长时间未访问）: key=%s", key)
}
