package observability

import (
	"context"
	"fmt"
	
	"github.com/zhucl121/langchain-go/core/chat"
	"github.com/zhucl121/langchain-go/core/runnable"
	"github.com/zhucl121/langchain-go/pkg/types"
	
	"go.opentelemetry.io/otel/trace"
)

// ChatModelTracer ChatModel 追踪包装器
type ChatModelTracer struct {
	model  chat.ChatModel
	tracer trace.Tracer
}

// NewChatModelTracer 创建 ChatModel 追踪包装器
func NewChatModelTracer(model chat.ChatModel, tracer trace.Tracer) *ChatModelTracer {
	return &ChatModelTracer{
		model:  model,
		tracer: tracer,
	}
}

// Invoke 实现 ChatModel 接口
func (cmt *ChatModelTracer) Invoke(ctx context.Context, messages []types.Message, opts ...runnable.Option) (types.Message, error) {
	var result types.Message
	
	err := TraceOperation(ctx, cmt.tracer, "chat_model.invoke", func(ctx context.Context, span *SpanHelper) error {
		// 记录模型信息
		span.SetAttribute(AttrLLMProvider, cmt.model.GetProvider())
		span.SetAttribute(AttrLLMModel, cmt.model.GetModelName())
		span.SetAttribute("message_count", len(messages))
		
		// 记录最后一条用户消息（通常是当前查询）
		if len(messages) > 0 {
			lastMsg := messages[len(messages)-1]
			if lastMsg.Role == types.RoleUser {
				truncated := lastMsg.Content
				if len(truncated) > 500 {
					truncated = truncated[:500] + "..."
				}
				span.SetAttribute(AttrLLMPrompt, truncated)
			}
		}
		
		// 调用底层模型
		var err error
		result, err = cmt.model.Invoke(ctx, messages, opts...)
		
		if err == nil {
			// 记录响应
			truncated := result.Content
			if len(truncated) > 500 {
				truncated = truncated[:500] + "..."
			}
			span.SetAttribute(AttrLLMResponse, truncated)
		}
		
		return err
	})
	
	return result, err
}

// Batch 实现 ChatModel 接口
func (cmt *ChatModelTracer) Batch(ctx context.Context, messages [][]types.Message, opts ...runnable.Option) ([]types.Message, error) {
	var results []types.Message
	
	err := TraceOperation(ctx, cmt.tracer, "chat_model.batch", func(ctx context.Context, span *SpanHelper) error {
		span.SetAttribute(AttrLLMProvider, cmt.model.GetProvider())
		span.SetAttribute(AttrLLMModel, cmt.model.GetModelName())
		span.SetAttribute("batch_size", len(messages))
		
		var err error
		results, err = cmt.model.Batch(ctx, messages, opts...)
		
		if err == nil {
			span.SetAttribute("results_count", len(results))
		}
		
		return err
	})
	
	return results, err
}

// Stream 实现 ChatModel 接口
func (cmt *ChatModelTracer) Stream(ctx context.Context, messages []types.Message, opts ...runnable.Option) (<-chan runnable.StreamEvent[types.Message], error) {
	ctx, span := cmt.tracer.Start(ctx, "chat_model.stream")
	helper := NewSpanHelper(span)
	
	helper.SetAttribute(AttrLLMProvider, cmt.model.GetProvider())
	helper.SetAttribute(AttrLLMModel, cmt.model.GetModelName())
	helper.SetAttribute("message_count", len(messages))
	
	// 调用底层模型的 Stream
	stream, err := cmt.model.Stream(ctx, messages, opts...)
	if err != nil {
		helper.RecordError(err)
		helper.End()
		return nil, err
	}
	
	// 包装流以在完成时结束 span
	wrappedStream := make(chan runnable.StreamEvent[types.Message])
	
	go func() {
		defer close(wrappedStream)
		defer helper.End()
		
		var chunkCount int
		for event := range stream {
			wrappedStream <- event
			chunkCount++
		}
		
		helper.SetAttribute("chunk_count", chunkCount)
		helper.SetSuccess()
	}()
	
	return wrappedStream, nil
}

// BindTools 实现 ChatModel 接口
func (cmt *ChatModelTracer) BindTools(tools []types.Tool) chat.ChatModel {
	return NewChatModelTracer(cmt.model.BindTools(tools), cmt.tracer)
}

// WithStructuredOutput 实现 ChatModel 接口
func (cmt *ChatModelTracer) WithStructuredOutput(schema types.Schema) chat.ChatModel {
	return NewChatModelTracer(cmt.model.WithStructuredOutput(schema), cmt.tracer)
}

// GetModelName 实现 ChatModel 接口
func (cmt *ChatModelTracer) GetModelName() string {
	return cmt.model.GetModelName()
}

// GetProvider 实现 ChatModel 接口
func (cmt *ChatModelTracer) GetProvider() string {
	return cmt.model.GetProvider()
}

// GetName 实现 Runnable 接口
func (cmt *ChatModelTracer) GetName() string {
	return cmt.model.GetName()
}

// WithConfig 实现 Runnable 接口
func (cmt *ChatModelTracer) WithConfig(config *types.Config) runnable.Runnable[[]types.Message, types.Message] {
	return NewChatModelTracer(cmt.model.WithConfig(config).(chat.ChatModel), cmt.tracer)
}

// WithRetry 实现 Runnable 接口
func (cmt *ChatModelTracer) WithRetry(policy types.RetryPolicy) runnable.Runnable[[]types.Message, types.Message] {
	return NewChatModelTracer(cmt.model.WithRetry(policy).(chat.ChatModel), cmt.tracer)
}

// WithFallbacks 实现 Runnable 接口
func (cmt *ChatModelTracer) WithFallbacks(fallbacks ...runnable.Runnable[[]types.Message, types.Message]) runnable.Runnable[[]types.Message, types.Message] {
	// 转换 fallbacks
	chatFallbacks := make([]runnable.Runnable[[]types.Message, types.Message], len(fallbacks))
	for i, fb := range fallbacks {
		chatFallbacks[i] = fb
	}
	
	wrapped := cmt.model.WithFallbacks(chatFallbacks...)
	return NewChatModelTracer(wrapped.(chat.ChatModel), cmt.tracer)
}

// RunnableTracer 通用 Runnable 追踪包装器
type RunnableTracer[I, O any] struct {
	runnable runnable.Runnable[I, O]
	tracer   trace.Tracer
	name     string
}

// NewRunnableTracer 创建 Runnable 追踪包装器
func NewRunnableTracer[I, O any](r runnable.Runnable[I, O], tracer trace.Tracer) *RunnableTracer[I, O] {
	return &RunnableTracer[I, O]{
		runnable: r,
		tracer:   tracer,
		name:     r.GetName(),
	}
}

// Invoke 实现 Runnable 接口
func (rt *RunnableTracer[I, O]) Invoke(ctx context.Context, input I, opts ...runnable.Option) (O, error) {
	var result O
	
	err := TraceOperation(ctx, rt.tracer, fmt.Sprintf("runnable.%s.invoke", rt.name), func(ctx context.Context, span *SpanHelper) error {
		var err error
		result, err = rt.runnable.Invoke(ctx, input, opts...)
		return err
	})
	
	return result, err
}

// Batch 实现 Runnable 接口
func (rt *RunnableTracer[I, O]) Batch(ctx context.Context, inputs []I, opts ...runnable.Option) ([]O, error) {
	var results []O
	
	err := TraceOperation(ctx, rt.tracer, fmt.Sprintf("runnable.%s.batch", rt.name), func(ctx context.Context, span *SpanHelper) error {
		span.SetAttribute("batch_size", len(inputs))
		
		var err error
		results, err = rt.runnable.Batch(ctx, inputs, opts...)
		
		if err == nil {
			span.SetAttribute("results_count", len(results))
		}
		
		return err
	})
	
	return results, err
}

// Stream 实现 Runnable 接口
func (rt *RunnableTracer[I, O]) Stream(ctx context.Context, input I, opts ...runnable.Option) (<-chan runnable.StreamEvent[O], error) {
	ctx, span := rt.tracer.Start(ctx, fmt.Sprintf("runnable.%s.stream", rt.name))
	helper := NewSpanHelper(span)
	
	stream, err := rt.runnable.Stream(ctx, input, opts...)
	if err != nil {
		helper.RecordError(err)
		helper.End()
		return nil, err
	}
	
	// 包装流
	wrappedStream := make(chan runnable.StreamEvent[O])
	
	go func() {
		defer close(wrappedStream)
		defer helper.End()
		
		var eventCount int
		for event := range stream {
			wrappedStream <- event
			eventCount++
		}
		
		helper.SetAttribute("event_count", eventCount)
		helper.SetSuccess()
	}()
	
	return wrappedStream, nil
}

// GetName 实现 Runnable 接口
func (rt *RunnableTracer[I, O]) GetName() string {
	return rt.runnable.GetName()
}

// WithConfig 实现 Runnable 接口
func (rt *RunnableTracer[I, O]) WithConfig(config *types.Config) runnable.Runnable[I, O] {
	return NewRunnableTracer(rt.runnable.WithConfig(config), rt.tracer)
}

// WithRetry 实现 Runnable 接口
func (rt *RunnableTracer[I, O]) WithRetry(policy types.RetryPolicy) runnable.Runnable[I, O] {
	return NewRunnableTracer(rt.runnable.WithRetry(policy), rt.tracer)
}

// WithFallbacks 实现 Runnable 接口
func (rt *RunnableTracer[I, O]) WithFallbacks(fallbacks ...runnable.Runnable[I, O]) runnable.Runnable[I, O] {
	return NewRunnableTracer(rt.runnable.WithFallbacks(fallbacks...), rt.tracer)
}

// ContextWithTracer 将 Tracer 添加到上下文
func ContextWithTracer(ctx context.Context, tracer trace.Tracer) context.Context {
	return context.WithValue(ctx, tracerKey, tracer)
}

// TracerFromContext 从上下文获取 Tracer
func TracerFromContext(ctx context.Context) (trace.Tracer, bool) {
	tracer, ok := ctx.Value(tracerKey).(trace.Tracer)
	return tracer, ok
}

type contextKey string

const tracerKey contextKey = "tracer"
