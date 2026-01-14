package types

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTool_Validate(t *testing.T) {
	t.Run("valid tool", func(t *testing.T) {
		tool := Tool{
			Name:        "search",
			Description: "Search the internet",
			Parameters: Schema{
				Type: "object",
				Properties: map[string]Schema{
					"query": {Type: "string"},
				},
			},
		}

		err := tool.Validate()
		assert.NoError(t, err)
	})

	t.Run("missing name", func(t *testing.T) {
		tool := Tool{
			Description: "Search",
			Parameters:  Schema{Type: "object"},
		}

		err := tool.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "name is required")
	})

	t.Run("missing description", func(t *testing.T) {
		tool := Tool{
			Name:       "search",
			Parameters: Schema{Type: "object"},
		}

		err := tool.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "description is required")
	})

	t.Run("invalid schema", func(t *testing.T) {
		tool := Tool{
			Name:        "search",
			Description: "Search",
			Parameters: Schema{
				Type:    "object",
				Minimum: ptr(10.0),
				Maximum: ptr(5.0), // min > max
			},
		}

		err := tool.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid parameters schema")
	})
}

func TestTool_ToOpenAITool(t *testing.T) {
	tool := Tool{
		Name:        "search",
		Description: "Search the internet",
		Parameters: Schema{
			Type: "object",
			Properties: map[string]Schema{
				"query": {Type: "string", Description: "Search query"},
			},
			Required: []string{"query"},
		},
		Strict: true,
	}

	result := tool.ToOpenAITool()

	assert.Equal(t, "function", result["type"])
	
	fn := result["function"].(map[string]any)
	assert.Equal(t, "search", fn["name"])
	assert.Equal(t, "Search the internet", fn["description"])
	assert.True(t, fn["strict"].(bool))
	assert.NotNil(t, fn["parameters"])
}

func TestTool_ToAnthropicTool(t *testing.T) {
	tool := Tool{
		Name:        "calculator",
		Description: "Perform calculations",
		Parameters: Schema{
			Type: "object",
			Properties: map[string]Schema{
				"expression": {Type: "string"},
			},
		},
	}

	result := tool.ToAnthropicTool()

	assert.Equal(t, "calculator", result["name"])
	assert.Equal(t, "Perform calculations", result["description"])
	assert.NotNil(t, result["input_schema"])
}

func TestTool_Clone(t *testing.T) {
	original := Tool{
		Name:        "search",
		Description: "Search",
		Parameters: Schema{
			Type: "object",
			Properties: map[string]Schema{
				"query": {Type: "string"},
			},
		},
	}

	clone := original.Clone()

	// 验证值相等
	assert.Equal(t, original.Name, clone.Name)
	assert.Equal(t, original.Description, clone.Description)

	// 验证是深拷贝
	clone.Name = "modified"
	assert.NotEqual(t, original.Name, clone.Name)

	clone.Parameters.Properties["query"] = Schema{Type: "number"}
	assert.Equal(t, "string", original.Parameters.Properties["query"].Type)
}

func TestTool_String(t *testing.T) {
	tool := Tool{
		Name:        "search",
		Description: "Search the internet",
	}

	str := tool.String()
	assert.Contains(t, str, "search")
	assert.Contains(t, str, "Search the internet")
}

func TestNewToolResult(t *testing.T) {
	result := NewToolResult("call-123", "search", map[string]any{
		"results": []string{"result1", "result2"},
	})

	assert.Equal(t, "call-123", result.ToolCallID)
	assert.Equal(t, "search", result.ToolName)
	assert.NotNil(t, result.Output)
	assert.False(t, result.IsError)
	assert.Empty(t, result.Error)
}

func TestNewToolErrorResult(t *testing.T) {
	err := errors.New("search failed")
	result := NewToolErrorResult("call-123", "search", err)

	assert.Equal(t, "call-123", result.ToolCallID)
	assert.Equal(t, "search", result.ToolName)
	assert.True(t, result.IsError)
	assert.Equal(t, "search failed", result.Error)
}

func TestToolResult_ToMessage(t *testing.T) {
	t.Run("success result", func(t *testing.T) {
		result := NewToolResult("call-123", "search", map[string]string{
			"status": "success",
		})

		msg := result.ToMessage()
		assert.Equal(t, RoleTool, msg.Role)
		assert.Equal(t, "call-123", msg.ToolCallID)
		assert.Contains(t, msg.Content, "success")
	})

	t.Run("error result", func(t *testing.T) {
		result := NewToolErrorResult("call-123", "search", errors.New("failed"))

		msg := result.ToMessage()
		assert.Equal(t, RoleTool, msg.Role)
		assert.Equal(t, "call-123", msg.ToolCallID)
		assert.Contains(t, msg.Content, "Error")
		assert.Contains(t, msg.Content, "failed")
	})

	t.Run("non-json output", func(t *testing.T) {
		result := NewToolResult("call-123", "test", make(chan int)) // channel 不能序列化

		msg := result.ToMessage()
		assert.Equal(t, RoleTool, msg.Role)
		assert.NotEmpty(t, msg.Content)
	})
}

func TestToolResult_String(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		result := NewToolResult("call-123", "search", "data")
		str := result.String()
		assert.Contains(t, str, "search")
		assert.Contains(t, str, "Success")
	})

	t.Run("error", func(t *testing.T) {
		result := NewToolErrorResult("call-123", "search", errors.New("failed"))
		str := result.String()
		assert.Contains(t, str, "search")
		assert.Contains(t, str, "Error")
		assert.Contains(t, str, "failed")
	})
}

func TestToolResult_JSONSerialization(t *testing.T) {
	original := ToolResult{
		ToolCallID: "call-123",
		ToolName:   "search",
		Output: map[string]any{
			"count": 10,
		},
		IsError: false,
	}

	data, err := json.Marshal(original)
	require.NoError(t, err)

	var decoded ToolResult
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, original.ToolCallID, decoded.ToolCallID)
	assert.Equal(t, original.ToolName, decoded.ToolName)
	assert.Equal(t, original.IsError, decoded.IsError)
}

// 辅助函数
func ptr[T any](v T) *T {
	return &v
}

// 基准测试
func BenchmarkTool_ToOpenAITool(b *testing.B) {
	tool := Tool{
		Name:        "search",
		Description: "Search",
		Parameters: Schema{
			Type: "object",
			Properties: map[string]Schema{
				"query": {Type: "string"},
				"limit": {Type: "integer"},
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = tool.ToOpenAITool()
	}
}

func BenchmarkToolResult_ToMessage(b *testing.B) {
	result := NewToolResult("call-123", "search", map[string]any{
		"results": []string{"a", "b", "c"},
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = result.ToMessage()
	}
}
