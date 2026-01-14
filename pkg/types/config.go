package types

import (
	"context"
	"time"
)

// Config 表示运行时配置。
//
// Config 用于在执行过程中传递配置信息、标签、元数据等。
//
// 示例：
//
//	config := types.NewConfig().
//	    WithTags("production", "api-v1").
//	    WithMetadata("user_id", "123").
//	    WithTimeout(30 * time.Second)
//
type Config struct {
	// Tags 标签列表，用于分类和过滤
	Tags []string

	// Metadata 元数据，用于附加任意信息
	Metadata map[string]any

	// RunName 运行名称，用于标识特定的执行
	RunName string

	// RunID 运行ID，唯一标识一次执行
	RunID string

	// MaxConcurrency 最大并发数
	MaxConcurrency int

	// MaxRetries 最大重试次数
	MaxRetries int

	// Timeout 超时时间
	Timeout time.Duration

	// Callbacks 回调处理器列表
	Callbacks []CallbackHandler

	// Context 上下文（通常不序列化）
	Context context.Context `json:"-"`
}

// CallbackHandler 回调处理器接口（简化版本，完整版在 callbacks 包）。
type CallbackHandler interface {
	// OnStart 在执行开始时调用
	OnStart(ctx context.Context, input any) error

	// OnEnd 在执行结束时调用
	OnEnd(ctx context.Context, output any) error

	// OnError 在发生错误时调用
	OnError(ctx context.Context, err error) error
}

// NewConfig 创建新的配置。
//
// 返回：
//   - *Config: 配置实例
//
func NewConfig() *Config {
	return &Config{
		Tags:           make([]string, 0),
		Metadata:       make(map[string]any),
		MaxConcurrency: 10,
		MaxRetries:     3,
		Timeout:        30 * time.Second,
		Context:        context.Background(),
	}
}

// WithTags 添加标签。
//
// 参数：
//   - tags: 标签列表
//
// 返回：
//   - *Config: 配置实例（支持链式调用）
//
func (c *Config) WithTags(tags ...string) *Config {
	c.Tags = append(c.Tags, tags...)
	return c
}

// WithMetadata 添加元数据。
//
// 参数：
//   - key: 元数据键
//   - value: 元数据值
//
// 返回：
//   - *Config: 配置实例（支持链式调用）
//
func (c *Config) WithMetadata(key string, value any) *Config {
	if c.Metadata == nil {
		c.Metadata = make(map[string]any)
	}
	c.Metadata[key] = value
	return c
}

// WithRunName 设置运行名称。
//
// 参数：
//   - name: 运行名称
//
// 返回：
//   - *Config: 配置实例（支持链式调用）
//
func (c *Config) WithRunName(name string) *Config {
	c.RunName = name
	return c
}

// WithRunID 设置运行ID。
//
// 参数：
//   - id: 运行ID
//
// 返回：
//   - *Config: 配置实例（支持链式调用）
//
func (c *Config) WithRunID(id string) *Config {
	c.RunID = id
	return c
}

// WithMaxConcurrency 设置最大并发数。
//
// 参数：
//   - n: 最大并发数
//
// 返回：
//   - *Config: 配置实例（支持链式调用）
//
func (c *Config) WithMaxConcurrency(n int) *Config {
	c.MaxConcurrency = n
	return c
}

// WithMaxRetries 设置最大重试次数。
//
// 参数：
//   - n: 最大重试次数
//
// 返回：
//   - *Config: 配置实例（支持链式调用）
//
func (c *Config) WithMaxRetries(n int) *Config {
	c.MaxRetries = n
	return c
}

// WithTimeout 设置超时时间。
//
// 参数：
//   - d: 超时时间
//
// 返回：
//   - *Config: 配置实例（支持链式调用）
//
func (c *Config) WithTimeout(d time.Duration) *Config {
	c.Timeout = d
	return c
}

// WithCallbacks 添加回调处理器。
//
// 参数：
//   - handlers: 回调处理器列表
//
// 返回：
//   - *Config: 配置实例（支持链式调用）
//
func (c *Config) WithCallbacks(handlers ...CallbackHandler) *Config {
	c.Callbacks = append(c.Callbacks, handlers...)
	return c
}

// WithContext 设置上下文。
//
// 参数：
//   - ctx: 上下文
//
// 返回：
//   - *Config: 配置实例（支持链式调用）
//
func (c *Config) WithContext(ctx context.Context) *Config {
	c.Context = ctx
	return c
}

// Clone 创建配置的深拷贝。
//
// 返回：
//   - *Config: 配置副本
//
func (c *Config) Clone() *Config {
	clone := &Config{
		RunName:        c.RunName,
		RunID:          c.RunID,
		MaxConcurrency: c.MaxConcurrency,
		MaxRetries:     c.MaxRetries,
		Timeout:        c.Timeout,
		Context:        c.Context,
	}

	// 深拷贝 Tags
	if c.Tags != nil {
		clone.Tags = make([]string, len(c.Tags))
		copy(clone.Tags, c.Tags)
	}

	// 深拷贝 Metadata
	if c.Metadata != nil {
		clone.Metadata = make(map[string]any, len(c.Metadata))
		for k, v := range c.Metadata {
			clone.Metadata[k] = v
		}
	}

	// 深拷贝 Callbacks
	if c.Callbacks != nil {
		clone.Callbacks = make([]CallbackHandler, len(c.Callbacks))
		copy(clone.Callbacks, c.Callbacks)
	}

	return clone
}

// Merge 合并另一个配置。
//
// 非零值会覆盖当前配置。
//
// 参数：
//   - other: 要合并的配置
//
// 返回：
//   - *Config: 配置实例（支持链式调用）
//
func (c *Config) Merge(other *Config) *Config {
	if other == nil {
		return c
	}

	if other.RunName != "" {
		c.RunName = other.RunName
	}
	if other.RunID != "" {
		c.RunID = other.RunID
	}
	if other.MaxConcurrency > 0 {
		c.MaxConcurrency = other.MaxConcurrency
	}
	if other.MaxRetries > 0 {
		c.MaxRetries = other.MaxRetries
	}
	if other.Timeout > 0 {
		c.Timeout = other.Timeout
	}
	if other.Context != nil {
		c.Context = other.Context
	}

	// 合并 Tags
	if len(other.Tags) > 0 {
		c.Tags = append(c.Tags, other.Tags...)
	}

	// 合并 Metadata
	if len(other.Metadata) > 0 {
		if c.Metadata == nil {
			c.Metadata = make(map[string]any)
		}
		for k, v := range other.Metadata {
			c.Metadata[k] = v
		}
	}

	// 合并 Callbacks
	if len(other.Callbacks) > 0 {
		c.Callbacks = append(c.Callbacks, other.Callbacks...)
	}

	return c
}

// GetContext 获取上下文。
//
// 如果未设置，返回 context.Background()。
//
// 返回：
//   - context.Context: 上下文
//
func (c *Config) GetContext() context.Context {
	if c.Context == nil {
		return context.Background()
	}
	return c.Context
}

// GetContextWithTimeout 获取带超时的上下文。
//
// 返回：
//   - context.Context: 带超时的上下文
//   - context.CancelFunc: 取消函数
//
func (c *Config) GetContextWithTimeout() (context.Context, context.CancelFunc) {
	ctx := c.GetContext()
	if c.Timeout > 0 {
		return context.WithTimeout(ctx, c.Timeout)
	}
	return ctx, func() {}
}

// HasTag 检查是否包含指定标签。
//
// 参数：
//   - tag: 标签名称
//
// 返回：
//   - bool: 是否包含
//
func (c *Config) HasTag(tag string) bool {
	for _, t := range c.Tags {
		if t == tag {
			return true
		}
	}
	return false
}

// GetMetadata 获取元数据。
//
// 参数：
//   - key: 元数据键
//
// 返回：
//   - any: 元数据值
//   - bool: 是否存在
//
func (c *Config) GetMetadata(key string) (any, bool) {
	if c.Metadata == nil {
		return nil, false
	}
	val, ok := c.Metadata[key]
	return val, ok
}

// RetryPolicy 重试策略配置。
type RetryPolicy struct {
	// MaxRetries 最大重试次数
	MaxRetries int

	// InitialDelay 初始延迟
	InitialDelay time.Duration

	// MaxDelay 最大延迟
	MaxDelay time.Duration

	// Multiplier 延迟倍数（指数退避）
	Multiplier float64

	// RetryableErrors 可重试的错误类型
	RetryableErrors []error
}

// NewRetryPolicy 创建默认的重试策略。
//
// 返回：
//   - RetryPolicy: 重试策略
//
func NewRetryPolicy() RetryPolicy {
	return RetryPolicy{
		MaxRetries:   3,
		InitialDelay: 1 * time.Second,
		MaxDelay:     30 * time.Second,
		Multiplier:   2.0,
	}
}

// GetDelay 计算第 n 次重试的延迟时间。
//
// 参数：
//   - retryCount: 当前重试次数（从 0 开始）
//
// 返回：
//   - time.Duration: 延迟时间
//
func (p RetryPolicy) GetDelay(retryCount int) time.Duration {
	delay := float64(p.InitialDelay)
	for i := 0; i < retryCount; i++ {
		delay *= p.Multiplier
	}

	result := time.Duration(delay)
	if result > p.MaxDelay {
		return p.MaxDelay
	}
	return result
}
