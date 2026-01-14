package chat

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"langchain-go/pkg/types"
)

func TestNewBaseChatModel(t *testing.T) {
	model := NewBaseChatModel("gpt-4", "openai")

	assert.NotNil(t, model)
	assert.Equal(t, "gpt-4", model.GetModelName())
	assert.Equal(t, "openai", model.GetProvider())
	assert.Equal(t, "openai/gpt-4", model.GetName())
	assert.Empty(t, model.GetBoundTools())
	assert.Nil(t, model.GetOutputSchema())
}

func TestBaseChatModel_SetBoundTools(t *testing.T) {
	model := NewBaseChatModel("gpt-4", "openai")

	tools := []types.Tool{
		{
			Name:        "get_weather",
			Description: "Get weather info",
			Parameters: types.Schema{
				Type: "object",
				Properties: map[string]types.Schema{
					"location": {Type: "string"},
				},
			},
		},
	}

	model.SetBoundTools(tools)

	assert.Equal(t, tools, model.GetBoundTools())
}

func TestBaseChatModel_SetOutputSchema(t *testing.T) {
	model := NewBaseChatModel("gpt-4", "openai")

	schema := types.Schema{
		Type: "object",
		Properties: map[string]types.Schema{
			"name": {Type: "string"},
			"age":  {Type: "integer"},
		},
	}

	model.SetOutputSchema(schema)

	assert.NotNil(t, model.GetOutputSchema())
	assert.Equal(t, schema, *model.GetOutputSchema())
}

func TestValidateMessages(t *testing.T) {
	tests := []struct {
		name      string
		messages  []types.Message
		expectErr bool
	}{
		{
			name:      "empty messages",
			messages:  []types.Message{},
			expectErr: true,
		},
		{
			name: "valid messages",
			messages: []types.Message{
				types.NewSystemMessage("You are a helpful assistant."),
				types.NewUserMessage("Hello"),
			},
			expectErr: false,
		},
		{
			name: "invalid role",
			messages: []types.Message{
				{Role: types.Role("invalid"), Content: "test"},
			},
			expectErr: true,
		},
		{
			name: "tool message without ID",
			messages: []types.Message{
				{Role: types.RoleTool, Content: "result"},
			},
			expectErr: true,
		},
		{
			name: "valid tool message",
			messages: []types.Message{
				types.NewToolMessage("call_123", "result"),
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMessages(tt.messages)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConvertToolsToOpenAI(t *testing.T) {
	tools := []types.Tool{
		{
			Name:        "calculator",
			Description: "Calculate math expressions",
			Parameters: types.Schema{
				Type: "object",
				Properties: map[string]types.Schema{
					"expression": {
						Type:        "string",
						Description: "Math expression to calculate",
					},
				},
				Required: []string{"expression"},
			},
		},
	}

	result := ConvertToolsToOpenAI(tools)

	require.Len(t, result, 1)
	assert.Equal(t, "function", result[0]["type"])

	function, ok := result[0]["function"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "calculator", function["name"])
	assert.Equal(t, "Calculate math expressions", function["description"])
	assert.NotNil(t, function["parameters"])
}

func TestConvertToolsToAnthropic(t *testing.T) {
	tools := []types.Tool{
		{
			Name:        "search",
			Description: "Search the web",
			Parameters: types.Schema{
				Type: "object",
				Properties: map[string]types.Schema{
					"query": {
						Type:        "string",
						Description: "Search query",
					},
				},
				Required: []string{"query"},
			},
		},
	}

	result := ConvertToolsToAnthropic(tools)

	require.Len(t, result, 1)
	assert.Equal(t, "search", result[0]["name"])
	assert.Equal(t, "Search the web", result[0]["description"])
	assert.NotNil(t, result[0]["input_schema"])
}

func TestBaseChatModel_Batch(t *testing.T) {
	// 注意：BaseChatModel 的 Batch 方法会调用 Invoke
	// 但 BaseChatModel 本身没有实现 Invoke，所以这里无法直接测试
	// 这个测试应该在具体的实现类中进行
	t.Skip("BaseChatModel.Batch requires concrete implementation")
}
