package memory

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/zhucl121/langchain-go/pkg/types"
)

func TestNewBaseMemory(t *testing.T) {
	base := NewBaseMemory()

	assert.NotNil(t, base)
	assert.Equal(t, "input", base.GetInputKey())
	assert.Equal(t, "output", base.GetOutputKey())
	assert.Equal(t, "history", base.GetMemoryKey())
	assert.True(t, base.GetReturnMessages())
}

func TestBaseMemory_SettersGetters(t *testing.T) {
	base := NewBaseMemory()

	// Test SetInputKey
	base.SetInputKey("user_input")
	assert.Equal(t, "user_input", base.GetInputKey())

	// Test SetOutputKey
	base.SetOutputKey("ai_output")
	assert.Equal(t, "ai_output", base.GetOutputKey())

	// Test SetMemoryKey
	base.SetMemoryKey("chat_history")
	assert.Equal(t, "chat_history", base.GetMemoryKey())

	// Test SetReturnMessages
	base.SetReturnMessages(false)
	assert.False(t, base.GetReturnMessages())
}

func TestBaseMemory_extractInputOutput(t *testing.T) {
	base := NewBaseMemory()

	tests := []struct {
		name           string
		inputs         map[string]any
		outputs        map[string]any
		expectedInput  string
		expectedOutput string
	}{
		{
			name: "normal case",
			inputs: map[string]any{
				"input": "Hello",
			},
			outputs: map[string]any{
				"output": "Hi there!",
			},
			expectedInput:  "Hello",
			expectedOutput: "Hi there!",
		},
		{
			name:           "empty inputs and outputs",
			inputs:         map[string]any{},
			outputs:        map[string]any{},
			expectedInput:  "",
			expectedOutput: "",
		},
		{
			name: "missing input key",
			inputs: map[string]any{
				"other": "value",
			},
			outputs: map[string]any{
				"output": "response",
			},
			expectedInput:  "",
			expectedOutput: "response",
		},
		{
			name:           "nil inputs",
			inputs:         nil,
			outputs:        map[string]any{"output": "test"},
			expectedInput:  "",
			expectedOutput: "test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input, output := base.extractInputOutput(tt.inputs, tt.outputs)
			assert.Equal(t, tt.expectedInput, input)
			assert.Equal(t, tt.expectedOutput, output)
		})
	}
}

func TestMessagesToString(t *testing.T) {
	tests := []struct {
		name     string
		messages []types.Message
		expected string
	}{
		{
			name:     "empty messages",
			messages: []types.Message{},
			expected: "",
		},
		{
			name: "single message",
			messages: []types.Message{
				types.NewUserMessage("Hello"),
			},
			expected: "Human: Hello",
		},
		{
			name: "multiple messages",
			messages: []types.Message{
				types.NewUserMessage("Hello"),
				types.NewAssistantMessage("Hi there!"),
				types.NewUserMessage("How are you?"),
			},
			expected: "Human: Hello\nAI: Hi there!\nHuman: How are you?",
		},
		{
			name: "different roles",
			messages: []types.Message{
				types.NewSystemMessage("You are helpful"),
				types.NewUserMessage("Hi"),
				types.NewAssistantMessage("Hello"),
			},
			expected: "System: You are helpful\nHuman: Hi\nAI: Hello",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := messagesToString(tt.messages)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBaseMemory_CustomKeys(t *testing.T) {
	base := NewBaseMemory()
	base.SetInputKey("user")
	base.SetOutputKey("bot")

	inputs := map[string]any{"user": "test input"}
	outputs := map[string]any{"bot": "test output"}

	input, output := base.extractInputOutput(inputs, outputs)
	assert.Equal(t, "test input", input)
	assert.Equal(t, "test output", output)
}
