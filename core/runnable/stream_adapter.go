package runnable

import (
	"context"
	"fmt"

	"github.com/zhucl121/langchain-go/pkg/types"
)

// StreamAdapter 将 types.StreamEvent 转换为 Runnable StreamEvent。
//
// StreamAdapter 用于桥接 Provider 的流式输出和 Runnable 接口。
//
type StreamAdapter struct{}

// NewStreamAdapter 创建新的流适配器。
func NewStreamAdapter() *StreamAdapter {
	return &StreamAdapter{}
}

// AdaptMessageStream 将消息流转换为 Runnable 流。
//
// 参数：
//   - ctx: 上下文
//   - name: Runnable 名称
//   - stream: 输入的 types.StreamEvent channel
//
// 返回：
//   - <-chan StreamEvent[types.Message]: Runnable 流式事件 channel
//
func (a *StreamAdapter) AdaptMessageStream(
	ctx context.Context,
	name string,
	stream <-chan types.StreamEvent,
) <-chan StreamEvent[types.Message] {
	out := make(chan StreamEvent[types.Message], 100)

	go func() {
		defer close(out)

		var contentBuilder string
		var toolCalls []types.ToolCall

		for event := range stream {
			select {
			case <-ctx.Done():
				out <- StreamEvent[types.Message]{
					Type:  EventError,
					Name:  name,
					Error: ctx.Err(),
				}
				return
			default:
			}

			switch event.Type {
			case types.StreamEventStart:
				// 发送开始事件
				out <- StreamEvent[types.Message]{
					Type:     EventStart,
					Name:     name,
					Metadata: event.Metadata,
				}

			case types.StreamEventToken:
				// 累积 token
				contentBuilder += event.Token

				// 发送流式事件（增量消息）
				out <- StreamEvent[types.Message]{
					Type: EventStream,
					Data: types.Message{
						Role:    types.RoleAssistant,
						Content: event.Token, // 发送增量
					},
					Name:     name,
					Metadata: event.Metadata,
				}

			case types.StreamEventContent:
				// 发送内容事件（完整或部分）
				if event.Content != "" {
					out <- StreamEvent[types.Message]{
						Type: EventStream,
						Data: types.Message{
							Role:    types.RoleAssistant,
							Content: event.Content,
						},
						Name:     name,
						Metadata: event.Metadata,
					}
				}

			case types.StreamEventToolCall:
				// 收集工具调用
				if event.ToolCall != nil {
					toolCalls = append(toolCalls, *event.ToolCall)
				}

			case types.StreamEventEnd:
				// 发送最终消息
				finalMessage := types.Message{
					Role:     types.RoleAssistant,
					Content:  contentBuilder,
					ToolCalls: toolCalls,
					Metadata: event.Metadata,
				}

				out <- StreamEvent[types.Message]{
					Type:     EventEnd,
					Data:     finalMessage,
					Name:     name,
					Metadata: event.Metadata,
				}

			case types.StreamEventError:
				// 发送错误事件
				out <- StreamEvent[types.Message]{
					Type:  EventError,
					Name:  name,
					Error: event.Error,
				}
				return
			}
		}
	}()

	return out
}

// AdaptTokenStream 将 token 流直接转换为字符串流。
//
// 参数：
//   - ctx: 上下文
//   - name: Runnable 名称
//   - stream: 输入的 types.StreamEvent channel
//
// 返回：
//   - <-chan StreamEvent[string]: Runnable 流式事件 channel（字符串类型）
//
func (a *StreamAdapter) AdaptTokenStream(
	ctx context.Context,
	name string,
	stream <-chan types.StreamEvent,
) <-chan StreamEvent[string] {
	out := make(chan StreamEvent[string], 100)

	go func() {
		defer close(out)

		var fullContent string

		for event := range stream {
			select {
			case <-ctx.Done():
				out <- StreamEvent[string]{
					Type:  EventError,
					Name:  name,
					Error: ctx.Err(),
				}
				return
			default:
			}

			switch event.Type {
			case types.StreamEventStart:
				out <- StreamEvent[string]{
					Type:     EventStart,
					Name:     name,
					Metadata: event.Metadata,
				}

			case types.StreamEventToken:
				fullContent += event.Token

				out <- StreamEvent[string]{
					Type:     EventStream,
					Data:     event.Token,
					Name:     name,
					Metadata: event.Metadata,
				}

			case types.StreamEventEnd:
				out <- StreamEvent[string]{
					Type:     EventEnd,
					Data:     fullContent,
					Name:     name,
					Metadata: event.Metadata,
				}

			case types.StreamEventError:
				out <- StreamEvent[string]{
					Type:  EventError,
					Name:  name,
					Error: event.Error,
				}
				return
			}
		}
	}()

	return out
}

// StreamableRunnable 是支持流式输出的 Runnable 包装器。
//
// StreamableRunnable 将实现了流式接口的组件包装为标准 Runnable。
//
type StreamableRunnable[I, O any] struct {
	name        string
	invokeFunc  func(context.Context, I) (O, error)
	streamFunc  func(context.Context, I) (<-chan types.StreamEvent, error)
	adapter     *StreamAdapter
	convertFunc func(types.StreamEvent) StreamEvent[O]
}

// NewStreamableRunnable 创建新的流式 Runnable。
//
// 参数：
//   - name: Runnable 名称
//   - invokeFunc: 同步调用函数
//   - streamFunc: 流式调用函数
//   - convertFunc: 事件转换函数
//
// 返回：
//   - *StreamableRunnable[I, O]: 流式 Runnable 实例
//
func NewStreamableRunnable[I, O any](
	name string,
	invokeFunc func(context.Context, I) (O, error),
	streamFunc func(context.Context, I) (<-chan types.StreamEvent, error),
	convertFunc func(types.StreamEvent) StreamEvent[O],
) *StreamableRunnable[I, O] {
	return &StreamableRunnable[I, O]{
		name:        name,
		invokeFunc:  invokeFunc,
		streamFunc:  streamFunc,
		adapter:     NewStreamAdapter(),
		convertFunc: convertFunc,
	}
}

// Invoke 实现 Runnable 接口。
func (s *StreamableRunnable[I, O]) Invoke(ctx context.Context, input I, opts ...Option) (O, error) {
	return s.invokeFunc(ctx, input)
}

// Batch 实现 Runnable 接口。
func (s *StreamableRunnable[I, O]) Batch(ctx context.Context, inputs []I, opts ...Option) ([]O, error) {
	results := make([]O, len(inputs))
	errors := make([]error, len(inputs))

	// 并行执行
	type result struct {
		index int
		data  O
		err   error
	}

	resultChan := make(chan result, len(inputs))

	for i, input := range inputs {
		go func(idx int, in I) {
			data, err := s.Invoke(ctx, in, opts...)
			resultChan <- result{index: idx, data: data, err: err}
		}(i, input)
	}

	// 收集结果
	for i := 0; i < len(inputs); i++ {
		res := <-resultChan
		results[res.index] = res.data
		errors[res.index] = res.err
	}

	close(resultChan)

	// 返回第一个错误
	for _, err := range errors {
		if err != nil {
			return results, err
		}
	}

	return results, nil
}

// Stream 实现 Runnable 接口。
func (s *StreamableRunnable[I, O]) Stream(ctx context.Context, input I, opts ...Option) (<-chan StreamEvent[O], error) {
	// 调用流式函数
	stream, err := s.streamFunc(ctx, input)
	if err != nil {
		return nil, err
	}

	// 转换事件
	out := make(chan StreamEvent[O], 100)

	go func() {
		defer close(out)

		for event := range stream {
			select {
			case <-ctx.Done():
				out <- StreamEvent[O]{
					Type:  EventError,
					Name:  s.name,
					Error: ctx.Err(),
				}
				return
			default:
			}

			// 使用转换函数
			if s.convertFunc != nil {
				out <- s.convertFunc(event)
			}
		}
	}()

	return out, nil
}

// GetName 实现 Runnable 接口。
func (s *StreamableRunnable[I, O]) GetName() string {
	return s.name
}

// WithConfig 实现 Runnable 接口。
func (s *StreamableRunnable[I, O]) WithConfig(config *types.Config) Runnable[I, O] {
	// 返回新实例（配置不变，因为已经在 invokeFunc 中）
	return s
}

// WithRetry 实现 Runnable 接口。
func (s *StreamableRunnable[I, O]) WithRetry(policy types.RetryPolicy) Runnable[I, O] {
	// 暂时返回自身（TODO: 实现 Retry 包装）
	return s
}

// WithFallbacks 实现 Runnable 接口。
func (s *StreamableRunnable[I, O]) WithFallbacks(fallbacks ...Runnable[I, O]) Runnable[I, O] {
	// 暂时返回自身（TODO: 实现 Fallback 包装）
	return s
}

// ConvertToRunnableEvent 将 types.StreamEvent 转换为 Runnable StreamEvent[T]。
//
// 这是一个通用辅助函数，用于事件转换。
//
func ConvertToRunnableEvent[T any](name string, event types.StreamEvent, data T) StreamEvent[T] {
	switch event.Type {
	case types.StreamEventStart:
		return StreamEvent[T]{
			Type:     EventStart,
			Name:     name,
			Metadata: event.Metadata,
		}

	case types.StreamEventToken, types.StreamEventContent:
		return StreamEvent[T]{
			Type:     EventStream,
			Data:     data,
			Name:     name,
			Metadata: event.Metadata,
		}

	case types.StreamEventEnd:
		return StreamEvent[T]{
			Type:     EventEnd,
			Data:     data,
			Name:     name,
			Metadata: event.Metadata,
		}

	case types.StreamEventError:
		return StreamEvent[T]{
			Type:  EventError,
			Name:  name,
			Error: event.Error,
		}

	default:
		return StreamEvent[T]{
			Type:  EventError,
			Name:  name,
			Error: fmt.Errorf("unknown event type: %s", event.Type),
		}
	}
}
