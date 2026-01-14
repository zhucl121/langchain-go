package types

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRole(t *testing.T) {
	t.Run("IsValid", func(t *testing.T) {
		tests := []struct {
			role  Role
			valid bool
		}{
			{RoleSystem, true},
			{RoleUser, true},
			{RoleAssistant, true},
			{RoleTool, true},
			{Role("invalid"), false},
			{Role(""), false},
		}

		for _, tt := range tests {
			t.Run(string(tt.role), func(t *testing.T) {
				assert.Equal(t, tt.valid, tt.role.IsValid())
			})
		}
	})

	t.Run("String", func(t *testing.T) {
		assert.Equal(t, "user", RoleUser.String())
		assert.Equal(t, "system", RoleSystem.String())
	})
}

func TestNewSystemMessage(t *testing.T) {
	msg := NewSystemMessage("You are a helpful assistant.")

	assert.Equal(t, RoleSystem, msg.Role)
	assert.Equal(t, "You are a helpful assistant.", msg.Content)
	assert.Empty(t, msg.Name)
	assert.Empty(t, msg.ToolCalls)
	assert.Empty(t, msg.ToolCallID)
	assert.Nil(t, msg.Metadata)
}

func TestNewUserMessage(t *testing.T) {
	msg := NewUserMessage("Hello, AI!")

	assert.Equal(t, RoleUser, msg.Role)
	assert.Equal(t, "Hello, AI!", msg.Content)
}

func TestNewAssistantMessage(t *testing.T) {
	msg := NewAssistantMessage("Hello, human!")

	assert.Equal(t, RoleAssistant, msg.Role)
	assert.Equal(t, "Hello, human!", msg.Content)
}

func TestNewToolMessage(t *testing.T) {
	msg := NewToolMessage("call-123", "search results")

	assert.Equal(t, RoleTool, msg.Role)
	assert.Equal(t, "search results", msg.Content)
	assert.Equal(t, "call-123", msg.ToolCallID)
}

func TestMessage_WithName(t *testing.T) {
	msg := NewUserMessage("Hello").WithName("Alice")

	assert.Equal(t, "Alice", msg.Name)
	assert.Equal(t, "Hello", msg.Content)

	// 验证不修改原消息
	original := NewUserMessage("Test")
	modified := original.WithName("Bob")
	assert.Empty(t, original.Name)
	assert.Equal(t, "Bob", modified.Name)
}

func TestMessage_WithMetadata(t *testing.T) {
	msg := NewUserMessage("Hello").
		WithMetadata("user_id", "123").
		WithMetadata("timestamp", 1234567890)

	require.NotNil(t, msg.Metadata)
	assert.Equal(t, "123", msg.Metadata["user_id"])
	assert.Equal(t, 1234567890, msg.Metadata["timestamp"])
}

func TestToolCall_GetToolCallArgs(t *testing.T) {
	t.Run("valid json", func(t *testing.T) {
		tc := ToolCall{
			Function: FunctionCall{
				Name:      "search",
				Arguments: `{"query": "golang", "limit": 10}`,
			},
		}

		args, err := tc.GetToolCallArgs()
		require.NoError(t, err)
		assert.Equal(t, "golang", args["query"])
		assert.Equal(t, float64(10), args["limit"]) // JSON 数字解析为 float64
	})

	t.Run("empty arguments", func(t *testing.T) {
		tc := ToolCall{
			Function: FunctionCall{
				Name:      "ping",
				Arguments: "",
			},
		}

		args, err := tc.GetToolCallArgs()
		require.NoError(t, err)
		assert.Empty(t, args)
	})

	t.Run("invalid json", func(t *testing.T) {
		tc := ToolCall{
			Function: FunctionCall{
				Name:      "search",
				Arguments: `{invalid json}`,
			},
		}

		_, err := tc.GetToolCallArgs()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse tool call arguments")
	})
}

func TestMessage_Validate(t *testing.T) {
	t.Run("valid messages", func(t *testing.T) {
		tests := []struct {
			name string
			msg  Message
		}{
			{"system", NewSystemMessage("test")},
			{"user", NewUserMessage("test")},
			{"assistant", NewAssistantMessage("test")},
			{"tool", NewToolMessage("call-123", "result")},
			{
				"assistant with tool calls",
				Message{
					Role:    RoleAssistant,
					Content: "Let me search",
					ToolCalls: []ToolCall{
						{
							ID:   "call-1",
							Type: "function",
							Function: FunctionCall{
								Name:      "search",
								Arguments: `{"query": "test"}`,
							},
						},
					},
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := tt.msg.Validate()
				assert.NoError(t, err)
			})
		}
	})

	t.Run("invalid messages", func(t *testing.T) {
		tests := []struct {
			name   string
			msg    Message
			errMsg string
		}{
			{
				"invalid role",
				Message{Role: Role("invalid"), Content: "test"},
				"invalid role",
			},
			{
				"tool without tool_call_id",
				Message{Role: RoleTool, Content: "result"},
				"must have tool_call_id",
			},
			{
				"tool call without id",
				Message{
					Role: RoleAssistant,
					ToolCalls: []ToolCall{
						{Function: FunctionCall{Name: "search"}},
					},
				},
				"missing id",
			},
			{
				"tool call without name",
				Message{
					Role: RoleAssistant,
					ToolCalls: []ToolCall{
						{ID: "call-1", Function: FunctionCall{Arguments: "{}"}},
					},
				},
				"missing function name",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := tt.msg.Validate()
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			})
		}
	})
}

func TestMessage_Clone(t *testing.T) {
	original := Message{
		Role:    RoleAssistant,
		Content: "Original",
		Name:    "AI",
		ToolCalls: []ToolCall{
			{
				ID:   "call-1",
				Type: "function",
				Function: FunctionCall{
					Name:      "search",
					Arguments: `{"query": "test"}`,
				},
			},
		},
		Metadata: map[string]any{
			"key": "value",
		},
	}

	clone := original.Clone()

	// 验证值相等
	assert.Equal(t, original.Role, clone.Role)
	assert.Equal(t, original.Content, clone.Content)
	assert.Equal(t, original.Name, clone.Name)
	assert.Equal(t, original.ToolCalls, clone.ToolCalls)
	assert.Equal(t, original.Metadata, clone.Metadata)

	// 验证是深拷贝
	clone.Content = "Modified"
	assert.NotEqual(t, original.Content, clone.Content)

	clone.ToolCalls[0].ID = "call-2"
	assert.NotEqual(t, original.ToolCalls[0].ID, clone.ToolCalls[0].ID)

	clone.Metadata["key"] = "new_value"
	assert.NotEqual(t, original.Metadata["key"], clone.Metadata["key"])
}

func TestMessage_String(t *testing.T) {
	t.Run("short content", func(t *testing.T) {
		msg := NewUserMessage("Hello")
		str := msg.String()
		assert.Contains(t, str, "user")
		assert.Contains(t, str, "Hello")
	})

	t.Run("long content truncated", func(t *testing.T) {
		// 创建一个超过50字符的内容
		longContent := "This is a very long message that should be truncated when converted to string for debugging purposes"
		msg := NewUserMessage(longContent)
		str := msg.String()
		assert.Contains(t, str, "...")
		assert.Less(t, len(str), 150) // String() 结果应该比原内容短
	})

	t.Run("with tool calls", func(t *testing.T) {
		msg := Message{
			Role:    RoleAssistant,
			Content: "Searching",
			ToolCalls: []ToolCall{
				{ID: "call-1"},
				{ID: "call-2"},
			},
		}
		str := msg.String()
		assert.Contains(t, str, "ToolCalls:2")
	})
}

func TestMessage_JSONSerialization(t *testing.T) {
	original := Message{
		Role:    RoleAssistant,
		Content: "Test message",
		Name:    "AI",
		ToolCalls: []ToolCall{
			{
				ID:   "call-123",
				Type: "function",
				Function: FunctionCall{
					Name:      "search",
					Arguments: `{"query": "test"}`,
				},
			},
		},
		Metadata: map[string]any{
			"timestamp": 1234567890,
		},
	}

	// 序列化
	data, err := json.Marshal(original)
	require.NoError(t, err)

	// 反序列化
	var decoded Message
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	// 验证
	assert.Equal(t, original.Role, decoded.Role)
	assert.Equal(t, original.Content, decoded.Content)
	assert.Equal(t, original.Name, decoded.Name)
	assert.Equal(t, len(original.ToolCalls), len(decoded.ToolCalls))
	assert.Equal(t, original.ToolCalls[0].ID, decoded.ToolCalls[0].ID)
}

// 基准测试
func BenchmarkNewUserMessage(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = NewUserMessage("Hello, world!")
	}
}

func BenchmarkMessage_Clone(b *testing.B) {
	msg := Message{
		Role:    RoleAssistant,
		Content: "Test",
		ToolCalls: []ToolCall{
			{ID: "call-1", Function: FunctionCall{Name: "test"}},
		},
		Metadata: map[string]any{"key": "value"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = msg.Clone()
	}
}

func BenchmarkToolCall_GetToolCallArgs(b *testing.B) {
	tc := ToolCall{
		Function: FunctionCall{
			Name:      "search",
			Arguments: `{"query": "golang", "limit": 10, "filter": {"category": "programming"}}`,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = tc.GetToolCallArgs()
	}
}
