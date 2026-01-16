package agents

import (
	"context"
	
	"langchain-go/core/chat"
	"langchain-go/core/runnable"
	"langchain-go/pkg/types"
)

// MockChatModel 是一个用于测试的mock ChatModel
type MockChatModel struct {
	*chat.BaseChatModel
	InvokeFunc func(ctx context.Context, messages []types.Message, opts ...runnable.Option) (types.Message, error)
}

// NewMockChatModel 创建一个新的MockChatModel
func NewMockChatModel() *MockChatModel {
	return &MockChatModel{
		BaseChatModel: chat.NewBaseChatModel("mock-model", "mock-provider"),
	}
}

// Invoke 实现 ChatModel 接口
func (m *MockChatModel) Invoke(ctx context.Context, messages []types.Message, opts ...runnable.Option) (types.Message, error) {
	if m.InvokeFunc != nil {
		return m.InvokeFunc(ctx, messages, opts...)
	}
	// 默认返回一个简单的响应
	return types.NewAssistantMessage("Final Answer: Mock response"), nil
}

// Batch 实现 ChatModel 接口
func (m *MockChatModel) Batch(ctx context.Context, inputs [][]types.Message, opts ...runnable.Option) ([]types.Message, error) {
	results := make([]types.Message, len(inputs))
	for i := range inputs {
		msg, err := m.Invoke(ctx, inputs[i], opts...)
		if err != nil {
			return nil, err
		}
		results[i] = msg
	}
	return results, nil
}

// Stream 实现 ChatModel 接口
func (m *MockChatModel) Stream(ctx context.Context, messages []types.Message, opts ...runnable.Option) (<-chan runnable.StreamEvent[types.Message], error) {
	out := make(chan runnable.StreamEvent[types.Message], 1)
	go func() {
		defer close(out)
		msg, _ := m.Invoke(ctx, messages, opts...)
		out <- runnable.StreamEvent[types.Message]{
			Data: msg,
		}
	}()
	return out, nil
}

// BindTools 实现 ChatModel 接口
func (m *MockChatModel) BindTools(tools []types.Tool) chat.ChatModel {
	newModel := NewMockChatModel()
	newModel.SetBoundTools(tools)
	newModel.InvokeFunc = m.InvokeFunc
	return newModel
}

// WithStructuredOutput 实现 ChatModel 接口
func (m *MockChatModel) WithStructuredOutput(schema types.Schema) chat.ChatModel {
	newModel := NewMockChatModel()
	newModel.SetOutputSchema(schema)
	newModel.InvokeFunc = m.InvokeFunc
	return newModel
}

// WithConfig 实现 ChatModel 接口
func (m *MockChatModel) WithConfig(config *types.Config) runnable.Runnable[[]types.Message, types.Message] {
	newModel := NewMockChatModel()
	newModel.SetConfig(config)
	newModel.InvokeFunc = m.InvokeFunc
	return newModel
}

// WithRetry 实现 Runnable 接口
func (m *MockChatModel) WithRetry(policy types.RetryPolicy) runnable.Runnable[[]types.Message, types.Message] {
	return runnable.NewRetryRunnable[[]types.Message, types.Message](m, policy)
}

// WithFallbacks 实现 Runnable 接口
func (m *MockChatModel) WithFallbacks(fallbacks ...runnable.Runnable[[]types.Message, types.Message]) runnable.Runnable[[]types.Message, types.Message] {
	return runnable.NewFallbackRunnable[[]types.Message, types.Message](m, fallbacks)
}
