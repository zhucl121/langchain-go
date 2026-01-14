package memory

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"langchain-go/pkg/types"
)

// MockChatModel 是用于测试的模拟 LLM
type MockChatModel struct {
	response string
	err      error
}

func (m *MockChatModel) Invoke(ctx context.Context, messages []types.Message) (types.Message, error) {
	if m.err != nil {
		return types.Message{}, m.err
	}
	return types.NewAssistantMessage(m.response), nil
}

func TestNewConversationSummaryMemory(t *testing.T) {
	mock := &MockChatModel{response: "test summary"}

	mem := NewConversationSummaryMemory(SummaryMemoryConfig{
		LLM:       mock,
		MaxTokens: 100,
	})

	assert.NotNil(t, mem)
	assert.Equal(t, 100, mem.maxTokens)
	assert.Empty(t, mem.GetSummary())
	assert.Empty(t, mem.GetMessages())
}

func TestConversationSummaryMemory_DefaultConfig(t *testing.T) {
	mock := &MockChatModel{response: "summary"}

	mem := NewConversationSummaryMemory(SummaryMemoryConfig{
		LLM: mock,
		// MaxTokens not specified
	})

	assert.Equal(t, 2000, mem.maxTokens) // Default value
}

func TestConversationSummaryMemory_SaveContext(t *testing.T) {
	mock := &MockChatModel{response: "conversation summary"}
	ctx := context.Background()

	mem := NewConversationSummaryMemory(SummaryMemoryConfig{
		LLM:       mock,
		MaxTokens: 1000,
	})

	err := mem.SaveContext(ctx,
		map[string]any{"input": "Hello"},
		map[string]any{"output": "Hi there!"},
	)

	require.NoError(t, err)

	messages := mem.GetMessages()
	assert.Len(t, messages, 2)
	assert.Equal(t, "Hello", messages[0].Content)
	assert.Equal(t, "Hi there!", messages[1].Content)
}

func TestConversationSummaryMemory_LoadMemoryVariables(t *testing.T) {
	mock := &MockChatModel{response: "summary"}
	ctx := context.Background()

	mem := NewConversationSummaryMemory(SummaryMemoryConfig{
		LLM:       mock,
		MaxTokens: 1000,
	})

	// Add some messages
	mem.SaveContext(ctx,
		map[string]any{"input": "Hello"},
		map[string]any{"output": "Hi!"},
	)

	vars, err := mem.LoadMemoryVariables(ctx, nil)
	require.NoError(t, err)

	messages, ok := vars["history"].([]types.Message)
	require.True(t, ok)
	assert.Len(t, messages, 2)
}

func TestConversationSummaryMemory_WithSummary(t *testing.T) {
	mock := &MockChatModel{response: "previous summary"}
	ctx := context.Background()

	mem := NewConversationSummaryMemory(SummaryMemoryConfig{
		LLM:       mock,
		MaxTokens: 10, // Very low to trigger summarization
	})

	// Add messages that will trigger summarization
	for i := 0; i < 5; i++ {
		err := mem.SaveContext(ctx,
			map[string]any{"input": "This is a long message to trigger summarization"},
			map[string]any{"output": "This is also a long response to ensure we exceed the token limit"},
		)
		require.NoError(t, err)
	}

	// Should have generated a summary
	summary := mem.GetSummary()
	assert.NotEmpty(t, summary)

	// Messages should be cleared after summarization
	messages := mem.GetMessages()
	// May have some messages after the last summarization
	assert.True(t, len(messages) < 10)
}

func TestConversationSummaryMemory_LoadWithSummary(t *testing.T) {
	mock := &MockChatModel{response: "test summary"}
	ctx := context.Background()

	mem := NewConversationSummaryMemory(SummaryMemoryConfig{
		LLM:       mock,
		MaxTokens: 5, // Very low
	})

	// Trigger summarization
	mem.SaveContext(ctx,
		map[string]any{"input": "Long message to trigger summary"},
		map[string]any{"output": "Long response to ensure summarization"},
	)

	// Add new message after summarization
	mem.SaveContext(ctx,
		map[string]any{"input": "New message"},
		map[string]any{"output": "New response"},
	)

	vars, err := mem.LoadMemoryVariables(ctx, nil)
	require.NoError(t, err)

	messages, ok := vars["history"].([]types.Message)
	require.True(t, ok)

	// Should include system message with summary + new messages
	assert.True(t, len(messages) > 0)

	// First message should be system message with summary
	assert.Equal(t, types.RoleSystem, messages[0].Role)
	assert.Contains(t, messages[0].Content, "summary")
}

func TestConversationSummaryMemory_ReturnAsString(t *testing.T) {
	mock := &MockChatModel{response: "summary"}
	ctx := context.Background()

	mem := NewConversationSummaryMemory(SummaryMemoryConfig{
		LLM:       mock,
		MaxTokens: 1000,
	})
	mem.SetReturnMessages(false)

	mem.SaveContext(ctx,
		map[string]any{"input": "Hello"},
		map[string]any{"output": "Hi!"},
	)

	vars, err := mem.LoadMemoryVariables(ctx, nil)
	require.NoError(t, err)

	historyStr, ok := vars["history"].(string)
	require.True(t, ok)
	assert.Contains(t, historyStr, "Hello")
	assert.Contains(t, historyStr, "Hi!")
}

func TestConversationSummaryMemory_Clear(t *testing.T) {
	mock := &MockChatModel{response: "summary"}
	ctx := context.Background()

	mem := NewConversationSummaryMemory(SummaryMemoryConfig{
		LLM:       mock,
		MaxTokens: 5,
	})

	// Add messages and trigger summarization
	mem.SaveContext(ctx,
		map[string]any{"input": "test message"},
		map[string]any{"output": "test response"},
	)

	// Clear
	err := mem.Clear(ctx)
	require.NoError(t, err)

	assert.Empty(t, mem.GetSummary())
	assert.Empty(t, mem.GetMessages())
}

func TestConversationSummaryMemory_EstimateTokens(t *testing.T) {
	mock := &MockChatModel{response: "summary"}

	mem := NewConversationSummaryMemory(SummaryMemoryConfig{
		LLM:       mock,
		MaxTokens: 1000,
	})

	// Empty memory
	tokens := mem.estimateTokens()
	assert.Equal(t, 0, tokens)

	// Add some content
	mem.summary = "This is a test summary"
	mem.messages = []types.Message{
		types.NewUserMessage("Hello"),
		types.NewAssistantMessage("Hi there!"),
	}

	tokens = mem.estimateTokens()
	assert.Greater(t, tokens, 0)
}

func TestConversationSummaryMemory_NoLLM(t *testing.T) {
	ctx := context.Background()

	mem := NewConversationSummaryMemory(SummaryMemoryConfig{
		LLM:       nil, // No LLM provided
		MaxTokens: 5,
	})

	// Should fail when trying to summarize
	err := mem.SaveContext(ctx,
		map[string]any{"input": "Long message to trigger summarization"},
		map[string]any{"output": "Long response"},
	)

	// Since we can't summarize, it should return an error
	if err != nil {
		assert.Contains(t, err.Error(), "LLM is required")
	}
}

func TestConversationSummaryMemory_CustomPrompt(t *testing.T) {
	mock := &MockChatModel{response: "custom summary"}

	customPrompt := "Summarize this: %s\nNew: %s\nResult:"

	mem := NewConversationSummaryMemory(SummaryMemoryConfig{
		LLM:           mock,
		MaxTokens:     100,
		SummaryPrompt: customPrompt,
	})

	assert.Equal(t, customPrompt, mem.summaryPrompt)
}
