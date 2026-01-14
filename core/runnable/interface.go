package runnable

import (
	"context"
	"fmt"

	"langchain-go/pkg/types"
)

// Runnable 是所有可执行组件的核心接口。
//
// Runnable 使用泛型参数 I 和 O 表示输入和输出类型，提供类型安全的执行接口。
//
// 所有实现 Runnable 接口的组件都可以：
//   - 单独执行（Invoke）
//   - 批量并行执行（Batch）
//   - 流式输出（Stream）
//   - 链式组合（Pipe）
//
// 类型参数：
//   - I: 输入类型
//   - O: 输出类型
//
type Runnable[I, O any] interface {
	// Invoke 执行 Runnable，处理单个输入并返回单个输出。
	//
	// 参数：
	//   - ctx: 上下文，用于超时控制和取消
	//   - input: 输入数据
	//   - opts: 可选配置
	//
	// 返回：
	//   - O: 输出数据
	//   - error: 执行错误
	//
	Invoke(ctx context.Context, input I, opts ...Option) (O, error)

	// Batch 批量执行 Runnable，处理多个输入并返回多个输出。
	//
	// Batch 会自动并行执行多个输入，利用 goroutine 提高性能。
	//
	// 参数：
	//   - ctx: 上下文
	//   - inputs: 输入列表
	//   - opts: 可选配置
	//
	// 返回：
	//   - []O: 输出列表（顺序与输入对应）
	//   - error: 执行错误（返回第一个遇到的错误）
	//
	Batch(ctx context.Context, inputs []I, opts ...Option) ([]O, error)

	// Stream 流式执行 Runnable，逐步返回输出。
	//
	// Stream 返回一个 channel，调用方可以从中接收流式事件。
	// channel 会在执行完成或出错时自动关闭。
	//
	// 参数：
	//   - ctx: 上下文
	//   - input: 输入数据
	//   - opts: 可选配置
	//
	// 返回：
	//   - <-chan StreamEvent[O]: 流式事件 channel
	//   - error: 启动流式执行时的错误
	//
	Stream(ctx context.Context, input I, opts ...Option) (<-chan StreamEvent[O], error)

	// GetName 获取 Runnable 的名称。
	//
	// 返回：
	//   - string: Runnable 名称
	//
	GetName() string

	// WithConfig 使用指定配置创建新的 Runnable。
	//
	// WithConfig 返回一个新的 Runnable 实例，不会修改原实例。
	//
	// 参数：
	//   - config: 配置
	//
	// 返回：
	//   - Runnable[I, O]: 新的 Runnable 实例
	//
	WithConfig(config *types.Config) Runnable[I, O]

	// WithRetry 添加重试逻辑。
	//
	// WithRetry 返回一个新的 Runnable，当执行失败时会自动重试。
	//
	// 参数：
	//   - policy: 重试策略
	//
	// 返回：
	//   - Runnable[I, O]: 带重试的 Runnable
	//
	WithRetry(policy types.RetryPolicy) Runnable[I, O]

	// WithFallbacks 添加降级方案。
	//
	// WithFallbacks 返回一个新的 Runnable，当执行失败时会尝试
	// 执行 fallback Runnables。
	//
	// 参数：
	//   - fallbacks: 降级 Runnable 列表
	//
	// 返回：
	//   - Runnable[I, O]: 带降级的 Runnable
	//
	WithFallbacks(fallbacks ...Runnable[I, O]) Runnable[I, O]
}

// StreamEvent 表示流式执行的事件。
//
// StreamEvent 包含事件类型、数据、错误等信息。
//
// 类型参数：
//   - T: 数据类型
//
type StreamEvent[T any] struct {
	// Type 事件类型
	Type EventType

	// Data 事件数据
	Data T

	// Name Runnable 名称
	Name string

	// Error 错误信息（仅在 Type 为 EventError 时有值）
	Error error

	// Metadata 元数据
	Metadata map[string]any
}

// EventType 事件类型
type EventType string

const (
	// EventStart 开始事件
	EventStart EventType = "start"

	// EventStream 流式数据事件
	EventStream EventType = "stream"

	// EventEnd 结束事件
	EventEnd EventType = "end"

	// EventError 错误事件
	EventError EventType = "error"
)

// String 实现 Stringer 接口
func (e EventType) String() string {
	return string(e)
}

// Option 执行选项函数
type Option func(*Options)

// Options 执行选项集合
type Options struct {
	// Config 运行时配置
	Config *types.Config

	// Callbacks 回调处理器
	Callbacks []types.CallbackHandler

	// Tags 标签
	Tags []string

	// Metadata 元数据
	Metadata map[string]any

	// RunName 运行名称
	RunName string
}

// NewOptions 创建新的选项
func NewOptions(opts ...Option) *Options {
	options := &Options{
		Config:    types.NewConfig(),
		Callbacks: make([]types.CallbackHandler, 0),
		Tags:      make([]string, 0),
		Metadata:  make(map[string]any),
	}

	for _, opt := range opts {
		opt(options)
	}

	return options
}

// WithConfig 设置配置
func WithConfig(config *types.Config) Option {
	return func(o *Options) {
		o.Config = config
	}
}

// WithCallbacks 设置回调
func WithCallbacks(callbacks ...types.CallbackHandler) Option {
	return func(o *Options) {
		o.Callbacks = append(o.Callbacks, callbacks...)
	}
}

// WithTags 设置标签
func WithTags(tags ...string) Option {
	return func(o *Options) {
		o.Tags = append(o.Tags, tags...)
	}
}

// WithMetadata 设置元数据
func WithMetadata(key string, value any) Option {
	return func(o *Options) {
		if o.Metadata == nil {
			o.Metadata = make(map[string]any)
		}
		o.Metadata[key] = value
	}
}

// WithRunName 设置运行名称
func WithRunName(name string) Option {
	return func(o *Options) {
		o.RunName = name
	}
}

// GetContext 从选项中获取上下文
func (o *Options) GetContext() context.Context {
	if o.Config != nil {
		return o.Config.GetContext()
	}
	return context.Background()
}

// MergeOptions 合并多个选项
func MergeOptions(opts ...Option) *Options {
	return NewOptions(opts...)
}

// RunnableFunc 函数类型，用于简化 Runnable 的创建
type RunnableFunc[I, O any] func(ctx context.Context, input I, opts ...Option) (O, error)

// Invoke 实现 Runnable 接口
func (f RunnableFunc[I, O]) Invoke(ctx context.Context, input I, opts ...Option) (O, error) {
	return f(ctx, input, opts...)
}

// Batch 实现 Runnable 接口（默认实现：顺序执行）
func (f RunnableFunc[I, O]) Batch(ctx context.Context, inputs []I, opts ...Option) ([]O, error) {
	results := make([]O, len(inputs))
	for i, input := range inputs {
		result, err := f.Invoke(ctx, input, opts...)
		if err != nil {
			return nil, fmt.Errorf("batch failed at index %d: %w", i, err)
		}
		results[i] = result
	}
	return results, nil
}

// Stream 实现 Runnable 接口（默认实现：单次执行）
func (f RunnableFunc[I, O]) Stream(ctx context.Context, input I, opts ...Option) (<-chan StreamEvent[O], error) {
	out := make(chan StreamEvent[O], 1)

	go func() {
		defer close(out)

		out <- StreamEvent[O]{Type: EventStart}

		result, err := f.Invoke(ctx, input, opts...)
		if err != nil {
			out <- StreamEvent[O]{Type: EventError, Error: err}
			return
		}

		out <- StreamEvent[O]{Type: EventStream, Data: result}
		out <- StreamEvent[O]{Type: EventEnd, Data: result}
	}()

	return out, nil
}

// GetName 实现 Runnable 接口
func (f RunnableFunc[I, O]) GetName() string {
	return "RunnableFunc"
}

// WithConfig 实现 Runnable 接口
func (f RunnableFunc[I, O]) WithConfig(config *types.Config) Runnable[I, O] {
	return f
}

// WithRetry 实现 Runnable 接口
func (f RunnableFunc[I, O]) WithRetry(policy types.RetryPolicy) Runnable[I, O] {
	return NewRetryRunnable(f, policy)
}

// WithFallbacks 实现 Runnable 接口
func (f RunnableFunc[I, O]) WithFallbacks(fallbacks ...Runnable[I, O]) Runnable[I, O] {
	return NewFallbackRunnable(f, fallbacks)
}

// AsAny 将 Runnable[I, O] 转换为 Runnable[I, any]。
//
// 这是一个辅助函数，用于解决 Go 泛型无法协变的问题。
// 主要用于 RunnableParallel 中需要混合不同输出类型的场景。
//
// 类型参数：
//   - I: 输入类型
//   - O: 输出类型
//
// 参数：
//   - r: 要转换的 Runnable
//
// 返回：
//   - Runnable[I, any]: 转换后的 Runnable
//
// 示例：
//
//	doubler := runnable.Lambda(func(ctx context.Context, x int) (int, error) {
//	    return x * 2, nil
//	})
//	// 转换为 Runnable[int, any]
//	anyRunnable := runnable.AsAny[int, int](doubler)
//
func AsAny[I, O any](r Runnable[I, O]) Runnable[I, any] {
	return &runnableAnyAdapter[I, O]{runnable: r}
}

// runnableAnyAdapter 适配器实现
type runnableAnyAdapter[I, O any] struct {
	runnable Runnable[I, O]
}

func (r *runnableAnyAdapter[I, O]) Invoke(ctx context.Context, input I, opts ...Option) (any, error) {
	result, err := r.runnable.Invoke(ctx, input, opts...)
	return result, err
}

func (r *runnableAnyAdapter[I, O]) Batch(ctx context.Context, inputs []I, opts ...Option) ([]any, error) {
	results, err := r.runnable.Batch(ctx, inputs, opts...)
	if err != nil {
		return nil, err
	}
	anyResults := make([]any, len(results))
	for i, result := range results {
		anyResults[i] = result
	}
	return anyResults, nil
}

func (r *runnableAnyAdapter[I, O]) Stream(ctx context.Context, input I, opts ...Option) (<-chan StreamEvent[any], error) {
	stream, err := r.runnable.Stream(ctx, input, opts...)
	if err != nil {
		return nil, err
	}

	out := make(chan StreamEvent[any], 10)
	go func() {
		defer close(out)
		for event := range stream {
			out <- StreamEvent[any]{
				Type:     event.Type,
				Data:     event.Data,
				Name:     event.Name,
				Error:    event.Error,
				Metadata: event.Metadata,
			}
		}
	}()

	return out, nil
}

func (r *runnableAnyAdapter[I, O]) GetName() string {
	return r.runnable.GetName()
}

func (r *runnableAnyAdapter[I, O]) WithConfig(config *types.Config) Runnable[I, any] {
	return &runnableAnyAdapter[I, O]{runnable: r.runnable.WithConfig(config)}
}

func (r *runnableAnyAdapter[I, O]) WithRetry(policy types.RetryPolicy) Runnable[I, any] {
	return &runnableAnyAdapter[I, O]{runnable: r.runnable.WithRetry(policy)}
}

func (r *runnableAnyAdapter[I, O]) WithFallbacks(fallbacks ...Runnable[I, any]) Runnable[I, any] {
	// 这个实现较复杂，暂时返回自身
	return r
}

