package runnable

import (
	"context"
	"fmt"

	"github.com/zhucl121/langchain-go/pkg/types"
)

// RunnableSequence 表示串联执行的 Runnable 序列。
//
// RunnableSequence 将多个 Runnable 串联起来，前一个的输出
// 作为后一个的输入，类似于 Unix 管道。
//
// 类型参数：
//   - I: 整体输入类型
//   - M: 中间类型
//   - O: 整体输出类型
//
// 示例：
//
//	// 创建序列：先乘2，再加1
//	doubler := runnable.Lambda(func(ctx context.Context, x int) (int, error) {
//	    return x * 2, nil
//	})
//	adder := runnable.Lambda(func(ctx context.Context, x int) (int, error) {
//	    return x + 1, nil
//	})
//	sequence := runnable.NewSequence(doubler, adder)
//	result, _ := sequence.Invoke(ctx, 5) // 返回 11 (5*2+1)
//
type RunnableSequence[I, M, O any] struct {
	first  Runnable[I, M]
	second Runnable[M, O]
	name   string
	config *types.Config
}

// NewSequence 创建一个 RunnableSequence。
//
// NewSequence 将两个 Runnable 串联起来。
//
// 参数：
//   - first: 第一个 Runnable
//   - second: 第二个 Runnable
//
// 返回：
//   - *RunnableSequence[I, M, O]: 序列实例
//
func NewSequence[I, M, O any](first Runnable[I, M], second Runnable[M, O]) *RunnableSequence[I, M, O] {
	return &RunnableSequence[I, M, O]{
		first:  first,
		second: second,
		name:   fmt.Sprintf("%s | %s", first.GetName(), second.GetName()),
		config: types.NewConfig(),
	}
}

// Invoke 实现 Runnable 接口。
//
// Invoke 依次执行序列中的 Runnable。
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
func (s *RunnableSequence[I, M, O]) Invoke(ctx context.Context, input I, opts ...Option) (O, error) {
	// 检查上下文取消
	select {
	case <-ctx.Done():
		var zero O
		return zero, ctx.Err()
	default:
	}

	// 执行第一个 Runnable
	mid, err := s.first.Invoke(ctx, input, opts...)
	if err != nil {
		var zero O
		return zero, fmt.Errorf("first runnable failed: %w", err)
	}

	// 检查上下文取消
	select {
	case <-ctx.Done():
		var zero O
		return zero, ctx.Err()
	default:
	}

	// 执行第二个 Runnable
	result, err := s.second.Invoke(ctx, mid, opts...)
	if err != nil {
		return result, fmt.Errorf("second runnable failed: %w", err)
	}

	return result, nil
}

// Batch 实现 Runnable 接口。
//
// Batch 批量执行序列。
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
func (s *RunnableSequence[I, M, O]) Batch(ctx context.Context, inputs []I, opts ...Option) ([]O, error) {
	if len(inputs) == 0 {
		return []O{}, nil
	}

	// 先批量执行第一个 Runnable
	mids, err := s.first.Batch(ctx, inputs, opts...)
	if err != nil {
		return nil, fmt.Errorf("first runnable batch failed: %w", err)
	}

	// 再批量执行第二个 Runnable
	results, err := s.second.Batch(ctx, mids, opts...)
	if err != nil {
		return nil, fmt.Errorf("second runnable batch failed: %w", err)
	}

	return results, nil
}

// Stream 实现 Runnable 接口。
//
// Stream 流式执行序列。
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
func (s *RunnableSequence[I, M, O]) Stream(ctx context.Context, input I, opts ...Option) (<-chan StreamEvent[O], error) {
	out := make(chan StreamEvent[O], 10)

	go func() {
		defer close(out)

		// 发送开始事件
		out <- StreamEvent[O]{
			Type: EventStart,
			Name: s.name,
		}

		// 执行第一个 Runnable
		mid, err := s.first.Invoke(ctx, input, opts...)
		if err != nil {
			out <- StreamEvent[O]{
				Type:  EventError,
				Name:  s.name,
				Error: fmt.Errorf("first runnable failed: %w", err),
			}
			return
		}

		// 流式执行第二个 Runnable
		stream, err := s.second.Stream(ctx, mid, opts...)
		if err != nil {
			out <- StreamEvent[O]{
				Type:  EventError,
				Name:  s.name,
				Error: fmt.Errorf("second runnable stream failed: %w", err),
			}
			return
		}

		// 转发第二个 Runnable 的流式事件
		for event := range stream {
			// 更新事件名称为序列名称
			event.Name = s.name
			out <- event
		}
	}()

	return out, nil
}

// GetName 实现 Runnable 接口。
//
// 返回：
//   - string: 序列名称
//
func (s *RunnableSequence[I, M, O]) GetName() string {
	return s.name
}

// WithConfig 实现 Runnable 接口。
//
// WithConfig 创建一个新的序列实例，使用指定配置。
//
// 参数：
//   - config: 配置
//
// 返回：
//   - Runnable[I, O]: 新的序列实例
//
func (s *RunnableSequence[I, M, O]) WithConfig(config *types.Config) Runnable[I, O] {
	return &RunnableSequence[I, M, O]{
		first:  s.first.WithConfig(config),
		second: s.second.WithConfig(config),
		name:   s.name,
		config: config,
	}
}

// WithRetry 实现 Runnable 接口。
//
// WithRetry 为整个序列添加重试逻辑。
//
// 参数：
//   - policy: 重试策略
//
// 返回：
//   - Runnable[I, O]: 带重试的序列
//
func (s *RunnableSequence[I, M, O]) WithRetry(policy types.RetryPolicy) Runnable[I, O] {
	return NewRetryRunnable[I, O](s, policy)
}

// WithFallbacks 实现 Runnable 接口。
//
// WithFallbacks 为整个序列添加降级方案。
//
// 参数：
//   - fallbacks: 降级 Runnable 列表
//
// 返回：
//   - Runnable[I, O]: 带降级的序列
//
func (s *RunnableSequence[I, M, O]) WithFallbacks(fallbacks ...Runnable[I, O]) Runnable[I, O] {
	return NewFallbackRunnable[I, O](s, fallbacks)
}

// Sequence 创建一个多步序列（便捷函数）。
//
// Sequence 接受可变数量的 Runnable，按顺序串联。
//
// 注意：由于 Go 泛型的限制，这个函数只能处理相同类型的 Runnable。
//
// 类型参数：
//   - T: 统一的输入输出类型
//
// 参数：
//   - runnables: Runnable 列表
//
// 返回：
//   - Runnable[T, T]: 序列 Runnable
//
// 示例：
//
//	// 创建一个处理链：x -> x*2 -> x+1 -> x*3
//	chain := runnable.Sequence(
//	    runnable.Lambda(func(ctx context.Context, x int) (int, error) { return x * 2, nil }),
//	    runnable.Lambda(func(ctx context.Context, x int) (int, error) { return x + 1, nil }),
//	    runnable.Lambda(func(ctx context.Context, x int) (int, error) { return x * 3, nil }),
//	)
//	result, _ := chain.Invoke(ctx, 5) // 返回 33 ((5*2+1)*3)
//
func Sequence[T any](runnables ...Runnable[T, T]) Runnable[T, T] {
	if len(runnables) == 0 {
		return Passthrough[T]()
	}

	if len(runnables) == 1 {
		return runnables[0]
	}

	// 从左到右依次组合
	result := runnables[0]
	for i := 1; i < len(runnables); i++ {
		result = NewSequence[T, T, T](result, runnables[i])
	}

	return result
}
