package chat

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zhucl121/langchain-go/pkg/types"
)

func TestMessagesToOpenAI(t *testing.T) {
	tests := []struct {
		name      string
		messages  []types.Message
		expectErr bool
		validate  func(t *testing.T, result []map[string]any)
	}{
		{
			name: "simple messages",
			messages: []types.Message{
				types.NewSystemMessage("You are helpful."),
				types.NewUserMessage("Hello"),
				types.NewAssistantMessage("Hi there!"),
			},
			expectErr: false,
			validate: func(t *testing.T, result []map[string]any) {
				require.Len(t, result, 3)
				assert.Equal(t, "system", result[0]["role"])
				assert.Equal(t, "You are helpful.", result[0]["content"])
				assert.Equal(t, "user", result[1]["role"])
				assert.Equal(t, "Hello", result[1]["content"])
				assert.Equal(t, "assistant", result[2]["role"])
				assert.Equal(t, "Hi there!", result[2]["content"])
			},
		},
		{
			name: "message with tool calls",
			messages: []types.Message{
				{
					Role:    types.RoleAssistant,
					Content: "Let me check that.",
					ToolCalls: []types.ToolCall{
						{
							ID:   "call_123",
							Type: "function",
							Function: types.FunctionCall{
								Name:      "get_weather",
								Arguments: `{"location":"NYC"}`,
							},
						},
					},
				},
			},
			expectErr: false,
			validate: func(t *testing.T, result []map[string]any) {
				require.Len(t, result, 1)
				assert.Equal(t, "assistant", result[0]["role"])

				toolCalls, ok := result[0]["tool_calls"].([]map[string]any)
				require.True(t, ok)
				require.Len(t, toolCalls, 1)
				assert.Equal(t, "call_123", toolCalls[0]["id"])
				assert.Equal(t, "function", toolCalls[0]["type"])
			},
		},
		{
			name: "tool result message",
			messages: []types.Message{
				types.NewToolMessage("call_123", "Sunny, 72°F"),
			},
			expectErr: false,
			validate: func(t *testing.T, result []map[string]any) {
				require.Len(t, result, 1)
				assert.Equal(t, "tool", result[0]["role"])
				assert.Equal(t, "call_123", result[0]["tool_call_id"])
				assert.Equal(t, "Sunny, 72°F", result[0]["content"])
			},
		},
		{
			name: "invalid message",
			messages: []types.Message{
				{Role: types.Role("invalid"), Content: "test"},
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := MessagesToOpenAI(tt.messages)

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				if tt.validate != nil {
					tt.validate(t, result)
				}
			}
		})
	}
}

func TestMessagesToAnthropic(t *testing.T) {
	tests := []struct {
		name           string
		messages       []types.Message
		expectErr      bool
		expectSystem   string
		validateMsgs   func(t *testing.T, msgs []map[string]any)
	}{
		{
			name: "messages with system",
			messages: []types.Message{
				types.NewSystemMessage("You are helpful."),
				types.NewUserMessage("Hello"),
				types.NewAssistantMessage("Hi!"),
			},
			expectErr:    false,
			expectSystem: "You are helpful.",
			validateMsgs: func(t *testing.T, msgs []map[string]any) {
				require.Len(t, msgs, 2)
				assert.Equal(t, "user", msgs[0]["role"])
				assert.Equal(t, "Hello", msgs[0]["content"])
				assert.Equal(t, "assistant", msgs[1]["role"])
				assert.Equal(t, "Hi!", msgs[1]["content"])
			},
		},
		{
			name: "multiple system messages",
			messages: []types.Message{
				types.NewSystemMessage("Part 1"),
				types.NewSystemMessage("Part 2"),
				types.NewUserMessage("Hello"),
			},
			expectErr:    false,
			expectSystem: "Part 1\n\nPart 2",
			validateMsgs: func(t *testing.T, msgs []map[string]any) {
				require.Len(t, msgs, 1)
				assert.Equal(t, "user", msgs[0]["role"])
			},
		},
		{
			name: "assistant message with tool calls",
			messages: []types.Message{
				{
					Role:    types.RoleAssistant,
					Content: "Let me check.",
					ToolCalls: []types.ToolCall{
						{
							ID:   "call_123",
							Type: "function",
							Function: types.FunctionCall{
								Name:      "get_weather",
								Arguments: `{"location":"NYC"}`,
							},
						},
					},
				},
			},
			expectErr:    false,
			expectSystem: "",
			validateMsgs: func(t *testing.T, msgs []map[string]any) {
				require.Len(t, msgs, 1)
				assert.Equal(t, "assistant", msgs[0]["role"])

				content, ok := msgs[0]["content"].([]map[string]any)
				require.True(t, ok)
				require.GreaterOrEqual(t, len(content), 2)

				// 应该有 text 和 tool_use
				hasText := false
				hasToolUse := false
				for _, block := range content {
					if block["type"] == "text" {
						hasText = true
					}
					if block["type"] == "tool_use" {
						hasToolUse = true
					}
				}
				assert.True(t, hasText)
				assert.True(t, hasToolUse)
			},
		},
		{
			name: "tool result message",
			messages: []types.Message{
				types.NewToolMessage("call_123", "Sunny, 72°F"),
			},
			expectErr:    false,
			expectSystem: "",
			validateMsgs: func(t *testing.T, msgs []map[string]any) {
				require.Len(t, msgs, 1)
				assert.Equal(t, "user", msgs[0]["role"]) // Tool messages -> user in Anthropic

				content, ok := msgs[0]["content"].([]map[string]any)
				require.True(t, ok)
				require.Len(t, content, 1)
				assert.Equal(t, "tool_result", content[0]["type"])
				assert.Equal(t, "call_123", content[0]["tool_use_id"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			system, msgs, err := MessagesToAnthropic(tt.messages)

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectSystem, system)
				if tt.validateMsgs != nil {
					tt.validateMsgs(t, msgs)
				}
			}
		})
	}
}

func TestOpenAIResponseToMessage(t *testing.T) {
	tests := []struct {
		name     string
		response map[string]any
		validate func(t *testing.T, msg types.Message)
	}{
		{
			name: "simple text response",
			response: map[string]any{
				"role":    "assistant",
				"content": "Hello, world!",
			},
			validate: func(t *testing.T, msg types.Message) {
				assert.Equal(t, types.RoleAssistant, msg.Role)
				assert.Equal(t, "Hello, world!", msg.Content)
				assert.Empty(t, msg.ToolCalls)
			},
		},
		{
			name: "response with tool calls",
			response: map[string]any{
				"role":    "assistant",
				"content": "",
				"tool_calls": []any{
					map[string]any{
						"id":   "call_123",
						"type": "function",
						"function": map[string]any{
							"name":      "get_weather",
							"arguments": `{"location":"NYC"}`,
						},
					},
				},
			},
			validate: func(t *testing.T, msg types.Message) {
				assert.Equal(t, types.RoleAssistant, msg.Role)
				require.Len(t, msg.ToolCalls, 1)
				assert.Equal(t, "call_123", msg.ToolCalls[0].ID)
				assert.Equal(t, "function", msg.ToolCalls[0].Type)
				assert.Equal(t, "get_weather", msg.ToolCalls[0].Function.Name)
				assert.Equal(t, `{"location":"NYC"}`, msg.ToolCalls[0].Function.Arguments)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg, err := OpenAIResponseToMessage(tt.response)
			require.NoError(t, err)
			tt.validate(t, msg)
		})
	}
}

func TestAnthropicResponseToMessage(t *testing.T) {
	tests := []struct {
		name     string
		content  []any
		validate func(t *testing.T, msg types.Message)
	}{
		{
			name: "simple text response",
			content: []any{
				map[string]any{
					"type": "text",
					"text": "Hello, world!",
				},
			},
			validate: func(t *testing.T, msg types.Message) {
				assert.Equal(t, types.RoleAssistant, msg.Role)
				assert.Equal(t, "Hello, world!", msg.Content)
				assert.Empty(t, msg.ToolCalls)
			},
		},
		{
			name: "multiple text blocks",
			content: []any{
				map[string]any{
					"type": "text",
					"text": "Part 1",
				},
				map[string]any{
					"type": "text",
					"text": "Part 2",
				},
			},
			validate: func(t *testing.T, msg types.Message) {
				assert.Equal(t, types.RoleAssistant, msg.Role)
				assert.Equal(t, "Part 1\nPart 2", msg.Content)
			},
		},
		{
			name: "response with tool use",
			content: []any{
				map[string]any{
					"type": "text",
					"text": "Let me check.",
				},
				map[string]any{
					"type": "tool_use",
					"id":   "call_123",
					"name": "get_weather",
					"input": map[string]any{
						"location": "NYC",
					},
				},
			},
			validate: func(t *testing.T, msg types.Message) {
				assert.Equal(t, types.RoleAssistant, msg.Role)
				assert.Equal(t, "Let me check.", msg.Content)
				require.Len(t, msg.ToolCalls, 1)
				assert.Equal(t, "call_123", msg.ToolCalls[0].ID)
				assert.Equal(t, "get_weather", msg.ToolCalls[0].Function.Name)

				// 验证参数是有效的 JSON
				var args map[string]any
				err := json.Unmarshal([]byte(msg.ToolCalls[0].Function.Arguments), &args)
				require.NoError(t, err)
				assert.Equal(t, "NYC", args["location"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg, err := AnthropicResponseToMessage(tt.content)
			require.NoError(t, err)
			tt.validate(t, msg)
		})
	}
}

func TestMergeMessages(t *testing.T) {
	tests := []struct {
		name     string
		messages []types.Message
		expected int
	}{
		{
			name:     "empty messages",
			messages: []types.Message{},
			expected: 0,
		},
		{
			name: "no merge needed",
			messages: []types.Message{
				types.NewUserMessage("Hello"),
				types.NewAssistantMessage("Hi"),
			},
			expected: 2,
		},
		{
			name: "merge consecutive user messages",
			messages: []types.Message{
				types.NewUserMessage("Hello"),
				types.NewUserMessage("How are you?"),
			},
			expected: 1,
		},
		{
			name: "don't merge with tool calls",
			messages: []types.Message{
				types.NewAssistantMessage("Hi"),
				{
					Role:    types.RoleAssistant,
					Content: "Let me check.",
					ToolCalls: []types.ToolCall{
						{ID: "call_123", Type: "function"},
					},
				},
			},
			expected: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MergeMessages(tt.messages)
			assert.Len(t, result, tt.expected)

			if tt.expected == 1 && len(tt.messages) > 1 {
				// 验证内容被合并
				assert.Contains(t, result[0].Content, "\n")
			}
		})
	}
}

func TestExtractSystemMessage(t *testing.T) {
	tests := []struct {
		name           string
		messages       []types.Message
		expectSystem   string
		expectRemaining int
	}{
		{
			name:           "no system messages",
			messages:       []types.Message{types.NewUserMessage("Hello")},
			expectSystem:   "",
			expectRemaining: 1,
		},
		{
			name: "single system message",
			messages: []types.Message{
				types.NewSystemMessage("You are helpful."),
				types.NewUserMessage("Hello"),
			},
			expectSystem:   "You are helpful.",
			expectRemaining: 1,
		},
		{
			name: "multiple system messages",
			messages: []types.Message{
				types.NewSystemMessage("Part 1"),
				types.NewUserMessage("Hello"),
				types.NewSystemMessage("Part 2"),
				types.NewAssistantMessage("Hi"),
			},
			expectSystem:   "Part 1\n\nPart 2",
			expectRemaining: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			system, remaining := ExtractSystemMessage(tt.messages)
			assert.Equal(t, tt.expectSystem, system)
			assert.Len(t, remaining, tt.expectRemaining)

			// 验证剩余消息中没有系统消息
			for _, msg := range remaining {
				assert.NotEqual(t, types.RoleSystem, msg.Role)
			}
		})
	}
}
