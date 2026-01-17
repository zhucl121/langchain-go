package memory

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zhucl121/langchain-go/pkg/types"
)

func TestNewBufferMemory(t *testing.T) {
	mem := NewBufferMemory()

	assert.NotNil(t, mem)
	assert.NotNil(t, mem.BaseMemory)
	assert.Empty(t, mem.GetMessages())
}

func TestBufferMemory_SaveAndLoad(t *testing.T) {
	mem := NewBufferMemory()
	ctx := context.Background()

	// Save first conversation
	err := mem.SaveContext(ctx,
		map[string]any{"input": "Hello"},
		map[string]any{"output": "Hi there!"},
	)
	require.NoError(t, err)

	// Load memory
	vars, err := mem.LoadMemoryVariables(ctx, nil)
	require.NoError(t, err)

	messages, ok := vars["history"].([]types.Message)
	require.True(t, ok)
	require.Len(t, messages, 2)

	assert.Equal(t, types.RoleUser, messages[0].Role)
	assert.Equal(t, "Hello", messages[0].Content)
	assert.Equal(t, types.RoleAssistant, messages[1].Role)
	assert.Equal(t, "Hi there!", messages[1].Content)
}

func TestBufferMemory_MultipleConversations(t *testing.T) {
	mem := NewBufferMemory()
	ctx := context.Background()

	// Save multiple conversations
	conversations := []struct {
		input  string
		output string
	}{
		{"Hello", "Hi!"},
		{"How are you?", "I'm fine, thanks!"},
		{"What's your name?", "I'm an AI assistant."},
	}

	for _, conv := range conversations {
		err := mem.SaveContext(ctx,
			map[string]any{"input": conv.input},
			map[string]any{"output": conv.output},
		)
		require.NoError(t, err)
	}

	// Check all messages are saved
	messages := mem.GetMessages()
	assert.Len(t, messages, 6) // 3 conversations * 2 messages each

	// Verify order
	assert.Equal(t, "Hello", messages[0].Content)
	assert.Equal(t, "Hi!", messages[1].Content)
	assert.Equal(t, "How are you?", messages[2].Content)
}

func TestBufferMemory_ReturnAsString(t *testing.T) {
	mem := NewBufferMemory()
	mem.SetReturnMessages(false) // Return as string
	ctx := context.Background()

	err := mem.SaveContext(ctx,
		map[string]any{"input": "Hello"},
		map[string]any{"output": "Hi!"},
	)
	require.NoError(t, err)

	vars, err := mem.LoadMemoryVariables(ctx, nil)
	require.NoError(t, err)

	historyStr, ok := vars["history"].(string)
	require.True(t, ok)
	assert.Equal(t, "Human: Hello\nAI: Hi!", historyStr)
}

func TestBufferMemory_Clear(t *testing.T) {
	mem := NewBufferMemory()
	ctx := context.Background()

	// Add some messages
	mem.SaveContext(ctx,
		map[string]any{"input": "test"},
		map[string]any{"output": "response"},
	)

	assert.Len(t, mem.GetMessages(), 2)

	// Clear memory
	err := mem.Clear(ctx)
	require.NoError(t, err)

	assert.Empty(t, mem.GetMessages())
}

func TestBufferMemory_CustomKeys(t *testing.T) {
	mem := NewBufferMemory()
	mem.SetInputKey("user_input")
	mem.SetOutputKey("ai_response")
	mem.SetMemoryKey("chat_history")

	ctx := context.Background()

	err := mem.SaveContext(ctx,
		map[string]any{"user_input": "test"},
		map[string]any{"ai_response": "reply"},
	)
	require.NoError(t, err)

	vars, err := mem.LoadMemoryVariables(ctx, nil)
	require.NoError(t, err)

	_, ok := vars["chat_history"]
	assert.True(t, ok)
}

func TestNewConversationBufferWindowMemory(t *testing.T) {
	mem := NewConversationBufferWindowMemory(WindowMemoryConfig{K: 3})

	assert.NotNil(t, mem)
	assert.Equal(t, 3, mem.GetK())
	assert.Empty(t, mem.GetMessages())
}

func TestConversationBufferWindowMemory_WindowLimit(t *testing.T) {
	mem := NewConversationBufferWindowMemory(WindowMemoryConfig{K: 2})
	ctx := context.Background()

	// Add 4 conversations (8 messages)
	for i := 1; i <= 4; i++ {
		err := mem.SaveContext(ctx,
			map[string]any{"input": "input"},
			map[string]any{"output": "output"},
		)
		require.NoError(t, err)
	}

	// Should only keep last 2 conversations (4 messages)
	messages := mem.GetMessages()
	assert.Len(t, messages, 4)
}

func TestConversationBufferWindowMemory_DefaultK(t *testing.T) {
	// K <= 0 should use default value (5)
	mem := NewConversationBufferWindowMemory(WindowMemoryConfig{K: 0})
	assert.Equal(t, 5, mem.GetK())

	mem2 := NewConversationBufferWindowMemory(WindowMemoryConfig{K: -1})
	assert.Equal(t, 5, mem2.GetK())
}

func TestConversationBufferWindowMemory_LoadMemoryVariables(t *testing.T) {
	mem := NewConversationBufferWindowMemory(WindowMemoryConfig{K: 3})
	ctx := context.Background()

	// Add messages
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

func TestConversationBufferWindowMemory_Clear(t *testing.T) {
	mem := NewConversationBufferWindowMemory(WindowMemoryConfig{K: 2})
	ctx := context.Background()

	mem.SaveContext(ctx,
		map[string]any{"input": "test"},
		map[string]any{"output": "response"},
	)

	assert.Len(t, mem.GetMessages(), 2)

	err := mem.Clear(ctx)
	require.NoError(t, err)

	assert.Empty(t, mem.GetMessages())
}

func TestConversationBufferWindowMemory_ReturnAsString(t *testing.T) {
	mem := NewConversationBufferWindowMemory(WindowMemoryConfig{K: 2})
	mem.SetReturnMessages(false)
	ctx := context.Background()

	mem.SaveContext(ctx,
		map[string]any{"input": "Hi"},
		map[string]any{"output": "Hello"},
	)

	vars, err := mem.LoadMemoryVariables(ctx, nil)
	require.NoError(t, err)

	historyStr, ok := vars["history"].(string)
	require.True(t, ok)
	assert.Equal(t, "Human: Hi\nAI: Hello", historyStr)
}

func TestBufferMemory_Concurrency(t *testing.T) {
	mem := NewBufferMemory()
	ctx := context.Background()

	// Test concurrent writes
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			err := mem.SaveContext(ctx,
				map[string]any{"input": "test"},
				map[string]any{"output": "response"},
			)
			assert.NoError(t, err)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Should have 20 messages (10 conversations * 2 messages)
	assert.Len(t, mem.GetMessages(), 20)
}
