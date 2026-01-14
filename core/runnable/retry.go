package runnable

import (
	"context"
	"fmt"
	"time"

	"langchain-go/pkg/types"
)

// RetryRunnable 为 Runnable 添加重试逻辑。
//
// RetryRunnable 在执行失败时会根据重试策略自动重试。
//
// 类型参数：
//   - I: 输入类型
//   - O: 输出类型
//
type RetryRunnable[I, O any] struct {
	runnable Runnable[I, O]
	policy   types.RetryPolicy
	name     string
}

// NewRetryRunnable 创建一个带重试的 Runnable。
//
// 参数：
//   - runnable: 要包装的 Runnable
//   - policy: 重试策略
//
// 返回：
//   - *RetryRunnable[I, O]: 带重试的 Runnable
//
func NewRetryRunnable[I, O any](runnable Runnable[I, O], policy types.RetryPolicy) *RetryRunnable[I, O] {
	return &RetryRunnable[I, O]{
		runnable: runnable,
		policy:   policy,
		name:     fmt.Sprintf("Retry(%s)", runnable.GetName()),
	}
}

// Invoke 实现 Runnable 接口。
//
// Invoke 执行 Runnable，失败时根据策略重试。
//
// 参数：
//   - ctx: 上下文
//   - input: 输入数据
//   - opts: 可选配置
//
// 返回：
//   - O: 输出数据
//   - error: 执行错误
//
func (r *RetryRunnable[I, O]) Invoke(ctx context.Context, input I, opts ...Option) (O, error) {
	var lastErr error

	for attempt := 0; attempt <= r.policy.MaxRetries; attempt++ {
		// 检查上下文取消
		select {
		case <-ctx.Done():
			var zero O
			return zero, ctx.Err()
		default:
		}

		// 执行
		result, err := r.runnable.Invoke(ctx, input, opts...)
		if err == nil {
			return result, nil
		}

		lastErr = err

		// 最后一次尝试不需要等待
		if attempt == r.policy.MaxRetries {
			break
		}

		// 计算延迟
		delay := r.policy.GetDelay(attempt)

		// 等待后重试
		select {
		case <-ctx.Done():
			var zero O
			return zero, ctx.Err()
		case <-time.After(delay):
			// 继续重试
		}
	}

	var zero O
	return zero, fmt.Errorf("retry exhausted after %d attempts: %w", r.policy.MaxRetries+1, lastErr)
}

// Batch 实现 Runnable 接口。
func (r *RetryRunnable[I, O]) Batch(ctx context.Context, inputs []I, opts ...Option) ([]O, error) {
	results := make([]O, len(inputs))
	for i, input := range inputs {
		result, err := r.Invoke(ctx, input, opts...)
		if err != nil {
			return nil, fmt.Errorf("batch failed at index %d: %w", i, err)
		}
		results[i] = result
	}
	return results, nil
}

// Stream 实现 Runnable 接口。
func (r *RetryRunnable[I, O]) Stream(ctx context.Context, input I, opts ...Option) (<-chan StreamEvent[O], error) {
	out := make(chan StreamEvent[O], 10)

	go func() {
		defer close(out)

		out <- StreamEvent[O]{Type: EventStart, Name: r.name}

		result, err := r.Invoke(ctx, input, opts...)
		if err != nil {
			out <- StreamEvent[O]{Type: EventError, Name: r.name, Error: err}
			return
		}

		out <- StreamEvent[O]{Type: EventStream, Name: r.name, Data: result}
		out <- StreamEvent[O]{Type: EventEnd, Name: r.name, Data: result}
	}()

	return out, nil
}

// GetName 实现 Runnable 接口。
func (r *RetryRunnable[I, O]) GetName() string {
	return r.name
}

// WithConfig 实现 Runnable 接口。
func (r *RetryRunnable[I, O]) WithConfig(config *types.Config) Runnable[I, O] {
	return &RetryRunnable[I, O]{
		runnable: r.runnable.WithConfig(config),
		policy:   r.policy,
		name:     r.name,
	}
}

// WithRetry 实现 Runnable 接口。
func (r *RetryRunnable[I, O]) WithRetry(policy types.RetryPolicy) Runnable[I, O] {
	return NewRetryRunnable(r, policy)
}

// WithFallbacks 实现 Runnable 接口。
func (r *RetryRunnable[I, O]) WithFallbacks(fallbacks ...Runnable[I, O]) Runnable[I, O] {
	return NewFallbackRunnable(r, fallbacks)
}

// FallbackRunnable 为 Runnable 添加降级逻辑。
//
// FallbackRunnable 在执行失败时会尝试执行 fallback Runnables。
//
// 类型参数：
//   - I: 输入类型
//   - O: 输出类型
//
type FallbackRunnable[I, O any] struct {
	primary   Runnable[I, O]
	fallbacks []Runnable[I, O]
	name      string
}

// NewFallbackRunnable 创建一个带降级的 Runnable。
//
// 参数：
//   - primary: 主 Runnable
//   - fallbacks: 降级 Runnable 列表
//
// 返回：
//   - *FallbackRunnable[I, O]: 带降级的 Runnable
//
func NewFallbackRunnable[I, O any](primary Runnable[I, O], fallbacks []Runnable[I, O]) *FallbackRunnable[I, O] {
	return &FallbackRunnable[I, O]{
		primary:   primary,
		fallbacks: fallbacks,
		name:      fmt.Sprintf("Fallback(%s)", primary.GetName()),
	}
}

// Invoke 实现 Runnable 接口。
//
// Invoke 执行主 Runnable，失败时尝试 fallbacks。
//
// 参数：
//   - ctx: 上下文
//   - input: 输入数据
//   - opts: 可选配置
//
// 返回：
//   - O: 输出数据
//   - error: 执行错误
//
func (f *FallbackRunnable[I, O]) Invoke(ctx context.Context, input I, opts ...Option) (O, error) {
	// 尝试执行主 Runnable
	result, err := f.primary.Invoke(ctx, input, opts...)
	if err == nil {
		return result, nil
	}

	primaryErr := err

	// 依次尝试 fallbacks
	for i, fallback := range f.fallbacks {
		// 检查上下文取消
		select {
		case <-ctx.Done():
			var zero O
			return zero, ctx.Err()
		default:
		}

		result, err = fallback.Invoke(ctx, input, opts...)
		if err == nil {
			return result, nil
		}

		// 继续尝试下一个 fallback
		if i == len(f.fallbacks)-1 {
			// 最后一个 fallback 也失败了
			var zero O
			return zero, fmt.Errorf("all fallbacks failed, primary error: %w, last fallback error: %v", primaryErr, err)
		}
	}

	var zero O
	return zero, fmt.Errorf("primary failed: %w", primaryErr)
}

// Batch 实现 Runnable 接口。
func (f *FallbackRunnable[I, O]) Batch(ctx context.Context, inputs []I, opts ...Option) ([]O, error) {
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

// Stream 实现 Runnable 接口。
func (f *FallbackRunnable[I, O]) Stream(ctx context.Context, input I, opts ...Option) (<-chan StreamEvent[O], error) {
	out := make(chan StreamEvent[O], 10)

	go func() {
		defer close(out)

		out <- StreamEvent[O]{Type: EventStart, Name: f.name}

		result, err := f.Invoke(ctx, input, opts...)
		if err != nil {
			out <- StreamEvent[O]{Type: EventError, Name: f.name, Error: err}
			return
		}

		out <- StreamEvent[O]{Type: EventStream, Name: f.name, Data: result}
		out <- StreamEvent[O]{Type: EventEnd, Name: f.name, Data: result}
	}()

	return out, nil
}

// GetName 实现 Runnable 接口。
func (f *FallbackRunnable[I, O]) GetName() string {
	return f.name
}

// WithConfig 实现 Runnable 接口。
func (f *FallbackRunnable[I, O]) WithConfig(config *types.Config) Runnable[I, O] {
	newFallbacks := make([]Runnable[I, O], len(f.fallbacks))
	for i, fb := range f.fallbacks {
		newFallbacks[i] = fb.WithConfig(config)
	}

	return &FallbackRunnable[I, O]{
		primary:   f.primary.WithConfig(config),
		fallbacks: newFallbacks,
		name:      f.name,
	}
}

// WithRetry 实现 Runnable 接口。
func (f *FallbackRunnable[I, O]) WithRetry(policy types.RetryPolicy) Runnable[I, O] {
	return NewRetryRunnable(f, policy)
}

// WithFallbacks 实现 Runnable 接口。
func (f *FallbackRunnable[I, O]) WithFallbacks(fallbacks ...Runnable[I, O]) Runnable[I, O] {
	return NewFallbackRunnable(f, fallbacks)
}
