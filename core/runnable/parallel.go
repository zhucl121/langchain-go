package runnable

import (
	"context"
	"fmt"
	"sync"

	"langchain-go/pkg/types"
)

// RunnableParallel 表示并行执行的 Runnable 集合。
//
// RunnableParallel 同时执行多个 Runnable，收集它们的输出。
//
// 类型参数：
//   - I: 输入类型（所有 Runnable 共享相同输入）
//
// 示例：
//
//	// 创建并行执行：同时计算 x*2 和 x+1
//	parallel := runnable.NewParallel(map[string]runnable.Runnable[int, int]{
//	    "double": runnable.Lambda(func(ctx context.Context, x int) (int, error) {
//	        return x * 2, nil
//	    }),
//	    "increment": runnable.Lambda(func(ctx context.Context, x int) (int, error) {
//	        return x + 1, nil
//	    }),
//	})
//	results, _ := parallel.Invoke(ctx, 5)
//	// 返回 map[string]any{"double": 10, "increment": 6}
//
type RunnableParallel[I any] struct {
	runnables map[string]Runnable[I, any]
	name      string
	config    *types.Config
}

// NewParallel 创建一个 RunnableParallel。
//
// NewParallel 接受一个 map，key 是输出的字段名，value 是 Runnable。
//
// 参数：
//   - runnables: Runnable map
//
// 返回：
//   - *RunnableParallel[I]: 并行实例
//
func NewParallel[I any](runnables map[string]Runnable[I, any]) *RunnableParallel[I] {
	return &RunnableParallel[I]{
		runnables: runnables,
		name:      "Parallel",
		config:    types.NewConfig(),
	}
}

// Invoke 实现 Runnable 接口。
//
// Invoke 并行执行所有 Runnable，返回结果 map。
//
// 参数：
//   - ctx: 上下文
//   - input: 输入数据
//   - opts: 可选配置
//
// 返回：
//   - map[string]any: 结果 map
//   - error: 执行错误
//
func (p *RunnableParallel[I]) Invoke(ctx context.Context, input I, opts ...Option) (map[string]any, error) {
	if len(p.runnables) == 0 {
		return make(map[string]any), nil
	}

	results := make(map[string]any)
	errors := make(map[string]error)

	var mu sync.Mutex
	var wg sync.WaitGroup

	for key, r := range p.runnables {
		wg.Add(1)
		go func(k string, runnable Runnable[I, any]) {
			defer wg.Done()

			// 检查上下文取消
			select {
			case <-ctx.Done():
				mu.Lock()
				errors[k] = ctx.Err()
				mu.Unlock()
				return
			default:
			}

			// 执行 Runnable
			result, err := runnable.Invoke(ctx, input, opts...)

			mu.Lock()
			if err != nil {
				errors[k] = err
			} else {
				results[k] = result
			}
			mu.Unlock()
		}(key, r)
	}

	wg.Wait()

	// 检查是否有错误
	if len(errors) > 0 {
		// 返回第一个错误
		for key, err := range errors {
			return nil, fmt.Errorf("parallel runnable '%s' failed: %w", key, err)
		}
	}

	return results, nil
}

// Batch 实现 Runnable 接口。
//
// Batch 批量并行执行。
//
// 参数：
//   - ctx: 上下文
//   - inputs: 输入列表
//   - opts: 可选配置
//
// 返回：
//   - []map[string]any: 结果列表
//   - error: 执行错误
//
func (p *RunnableParallel[I]) Batch(ctx context.Context, inputs []I, opts ...Option) ([]map[string]any, error) {
	if len(inputs) == 0 {
		return []map[string]any{}, nil
	}

	results := make([]map[string]any, len(inputs))
	var mu sync.Mutex
	var wg sync.WaitGroup
	var firstErr error

	for i, input := range inputs {
		wg.Add(1)
		go func(idx int, in I) {
			defer wg.Done()

			result, err := p.Invoke(ctx, in, opts...)

			mu.Lock()
			defer mu.Unlock()

			if err != nil && firstErr == nil {
				firstErr = err
			}
			results[idx] = result
		}(i, input)
	}

	wg.Wait()

	if firstErr != nil {
		return nil, firstErr
	}

	return results, nil
}

// Stream 实现 Runnable 接口。
//
// Stream 流式并行执行。
//
// 参数：
//   - ctx: 上下文
//   - input: 输入数据
//   - opts: 可选配置
//
// 返回：
//   - <-chan StreamEvent[map[string]any]: 流式事件 channel
//   - error: 启动错误
//
func (p *RunnableParallel[I]) Stream(ctx context.Context, input I, opts ...Option) (<-chan StreamEvent[map[string]any], error) {
	out := make(chan StreamEvent[map[string]any], 10)

	go func() {
		defer close(out)

		// 发送开始事件
		out <- StreamEvent[map[string]any]{
			Type: EventStart,
			Name: p.name,
		}

		// 执行并行
		result, err := p.Invoke(ctx, input, opts...)

		if err != nil {
			out <- StreamEvent[map[string]any]{
				Type:  EventError,
				Name:  p.name,
				Error: err,
			}
			return
		}

		// 发送结果
		out <- StreamEvent[map[string]any]{
			Type: EventStream,
			Name: p.name,
			Data: result,
		}

		out <- StreamEvent[map[string]any]{
			Type: EventEnd,
			Name: p.name,
			Data: result,
		}
	}()

	return out, nil
}

// GetName 实现 Runnable 接口。
//
// 返回：
//   - string: 并行名称
//
func (p *RunnableParallel[I]) GetName() string {
	return p.name
}

// WithConfig 实现 Runnable 接口。
//
// WithConfig 创建一个新的并行实例，使用指定配置。
//
// 参数：
//   - config: 配置
//
// 返回：
//   - Runnable[I, map[string]any]: 新的并行实例
//
func (p *RunnableParallel[I]) WithConfig(config *types.Config) Runnable[I, map[string]any] {
	newRunnables := make(map[string]Runnable[I, any])
	for k, r := range p.runnables {
		newRunnables[k] = r.WithConfig(config)
	}

	return &RunnableParallel[I]{
		runnables: newRunnables,
		name:      p.name,
		config:    config,
	}
}

// WithRetry 实现 Runnable 接口。
//
// WithRetry 为整个并行添加重试逻辑。
//
// 参数：
//   - policy: 重试策略
//
// 返回：
//   - Runnable[I, map[string]any]: 带重试的并行
//
func (p *RunnableParallel[I]) WithRetry(policy types.RetryPolicy) Runnable[I, map[string]any] {
	return NewRetryRunnable[I, map[string]any](p, policy)
}

// WithFallbacks 实现 Runnable 接口。
//
// WithFallbacks 为整个并行添加降级方案。
//
// 参数：
//   - fallbacks: 降级 Runnable 列表
//
// 返回：
//   - Runnable[I, map[string]any]: 带降级的并行
//
func (p *RunnableParallel[I]) WithFallbacks(fallbacks ...Runnable[I, map[string]any]) Runnable[I, map[string]any] {
	return NewFallbackRunnable[I, map[string]any](p, fallbacks)
}

// Parallel 创建并行 Runnable（便捷函数）。
//
// Parallel 是 NewParallel 的别名。
//
// 类型参数：
//   - I: 输入类型
//
// 参数：
//   - runnables: Runnable map
//
// 返回：
//   - *RunnableParallel[I]: 并行实例
//
func Parallel[I any](runnables map[string]Runnable[I, any]) *RunnableParallel[I] {
	return NewParallel(runnables)
}
