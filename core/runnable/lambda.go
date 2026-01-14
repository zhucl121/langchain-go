package runnable

import (
	"context"
	"fmt"
	"sync"

	"langchain-go/pkg/types"
)

// RunnableLambda 将普通函数包装为 Runnable。
//
// RunnableLambda 是最简单的 Runnable 实现，用于将任意函数
// 转换为符合 Runnable 接口的组件。
//
// 类型参数：
//   - I: 输入类型
//   - O: 输出类型
//
// 示例：
//
//	// 创建一个简单的 lambda
//	doubler := runnable.Lambda(func(ctx context.Context, x int) (int, error) {
//	    return x * 2, nil
//	})
//
//	// 执行
//	result, _ := doubler.Invoke(ctx, 5) // 返回 10
//
type RunnableLambda[I, O any] struct {
	fn     func(context.Context, I) (O, error)
	name   string
	config *types.Config
}

// Lambda 创建一个 RunnableLambda。
//
// Lambda 是创建 RunnableLambda 的便捷函数。
//
// 参数：
//   - fn: 要包装的函数
//
// 返回：
//   - *RunnableLambda[I, O]: RunnableLambda 实例
//
// 示例：
//
//	increment := runnable.Lambda(func(ctx context.Context, x int) (int, error) {
//	    return x + 1, nil
//	})
//
func Lambda[I, O any](fn func(context.Context, I) (O, error)) *RunnableLambda[I, O] {
	return &RunnableLambda[I, O]{
		fn:     fn,
		name:   "Lambda",
		config: types.NewConfig(),
	}
}

// LambdaWithName 创建带名称的 RunnableLambda。
//
// 参数：
//   - name: Lambda 名称
//   - fn: 要包装的函数
//
// 返回：
//   - *RunnableLambda[I, O]: RunnableLambda 实例
//
func LambdaWithName[I, O any](name string, fn func(context.Context, I) (O, error)) *RunnableLambda[I, O] {
	return &RunnableLambda[I, O]{
		fn:     fn,
		name:   name,
		config: types.NewConfig(),
	}
}

// Invoke 实现 Runnable 接口。
//
// Invoke 执行包装的函数。
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
func (r *RunnableLambda[I, O]) Invoke(ctx context.Context, input I, opts ...Option) (O, error) {
	options := NewOptions(opts...)

	// 使用传入的上下文（不要被选项覆盖，除非明确指定）
	execCtx := ctx

	// 检查上下文取消
	select {
	case <-execCtx.Done():
		var zero O
		return zero, execCtx.Err()
	default:
	}

	// 执行回调：OnStart
	if options.Config != nil {
		for _, cb := range options.Config.Callbacks {
			if err := cb.OnStart(execCtx, input); err != nil {
				var zero O
				return zero, fmt.Errorf("callback OnStart failed: %w", err)
			}
		}
	}

	// 执行函数
	result, err := r.fn(execCtx, input)

	// 执行回调：OnEnd 或 OnError
	if options.Config != nil {
		for _, cb := range options.Config.Callbacks {
			if err != nil {
				cb.OnError(execCtx, err)
			} else {
				cb.OnEnd(execCtx, result)
			}
		}
	}

	return result, err
}

// Batch 实现 Runnable 接口。
//
// Batch 并行执行多个输入。
//
// 参数：
//   - ctx: 上下文
//   - inputs: 输入列表
//   - opts: 可选配置
//
// 返回：
//   - []O: 输出列表
//   - error: 执行错误
//
func (r *RunnableLambda[I, O]) Batch(ctx context.Context, inputs []I, opts ...Option) ([]O, error) {
	if len(inputs) == 0 {
		return []O{}, nil
	}

	options := NewOptions(opts...)
	maxConcurrency := 10
	if options.Config != nil && options.Config.MaxConcurrency > 0 {
		maxConcurrency = options.Config.MaxConcurrency
	}

	results := make([]O, len(inputs))
	errors := make([]error, len(inputs))

	// 使用信号量控制并发数
	sem := make(chan struct{}, maxConcurrency)
	var wg sync.WaitGroup

	for i, input := range inputs {
		wg.Add(1)
		go func(idx int, in I) {
			defer wg.Done()

			// 获取信号量
			sem <- struct{}{}
			defer func() { <-sem }()

			// 检查上下文取消
			select {
			case <-ctx.Done():
				errors[idx] = ctx.Err()
				return
			default:
			}

			// 执行
			result, err := r.Invoke(ctx, in, opts...)
			results[idx] = result
			errors[idx] = err
		}(i, input)
	}

	wg.Wait()

	// 检查是否有错误
	for i, err := range errors {
		if err != nil {
			return nil, fmt.Errorf("batch failed at index %d: %w", i, err)
		}
	}

	return results, nil
}

// Stream 实现 Runnable 接口。
//
// Stream 流式执行函��。
//
// 参数：
//   - ctx: 上下文
//   - input: 输入数据
//   - opts: 可选配置
//
// 返回：
//   - <-chan StreamEvent[O]: 流式事件 channel
//   - error: 启动错误
//
func (r *RunnableLambda[I, O]) Stream(ctx context.Context, input I, opts ...Option) (<-chan StreamEvent[O], error) {
	out := make(chan StreamEvent[O], 10)

	go func() {
		defer close(out)

		// 发送开始事件
		out <- StreamEvent[O]{
			Type: EventStart,
			Name: r.name,
		}

		// 执行函数
		result, err := r.Invoke(ctx, input, opts...)

		if err != nil {
			// 发送错误事件
			out <- StreamEvent[O]{
				Type:  EventError,
				Name:  r.name,
				Error: err,
			}
			return
		}

		// 发送流式数据事件
		out <- StreamEvent[O]{
			Type: EventStream,
			Name: r.name,
			Data: result,
		}

		// 发送结束事件
		out <- StreamEvent[O]{
			Type: EventEnd,
			Name: r.name,
			Data: result,
		}
	}()

	return out, nil
}

// GetName 实现 Runnable 接口。
//
// 返回：
//   - string: Lambda 名称
//
func (r *RunnableLambda[I, O]) GetName() string {
	return r.name
}

// WithConfig 实现 Runnable 接口。
//
// WithConfig 创建一个新的 Lambda 实例，使用指定配置。
//
// 参数：
//   - config: 配置
//
// 返回：
//   - Runnable[I, O]: 新的 Lambda 实例
//
func (r *RunnableLambda[I, O]) WithConfig(config *types.Config) Runnable[I, O] {
	return &RunnableLambda[I, O]{
		fn:     r.fn,
		name:   r.name,
		config: config,
	}
}

// WithRetry 实现 Runnable 接口。
//
// WithRetry 添加重试逻辑。
//
// 参数：
//   - policy: 重试策略
//
// 返回：
//   - Runnable[I, O]: 带重试的 Lambda
//
func (r *RunnableLambda[I, O]) WithRetry(policy types.RetryPolicy) Runnable[I, O] {
	return NewRetryRunnable[I, O](r, policy)
}

// WithFallbacks 实现 Runnable 接口。
//
// WithFallbacks 添加降级方案。
//
// 参数：
//   - fallbacks: 降级 Runnable 列表
//
// 返回：
//   - Runnable[I, O]: 带降级的 Lambda
//
func (r *RunnableLambda[I, O]) WithFallbacks(fallbacks ...Runnable[I, O]) Runnable[I, O] {
	return NewFallbackRunnable[I, O](r, fallbacks)
}

// Passthrough 创建一个直接返回输入的 Lambda。
//
// Passthrough 对于测试和调试很有用。
//
// 类型参数：
//   - T: 输入输出类型
//
// 返回：
//   - *RunnableLambda[T, T]: Passthrough Lambda
//
// 示例：
//
//	pt := runnable.Passthrough[int]()
//	result, _ := pt.Invoke(ctx, 42) // 返回 42
//
func Passthrough[T any]() *RunnableLambda[T, T] {
	return Lambda(func(ctx context.Context, input T) (T, error) {
		return input, nil
	})
}
