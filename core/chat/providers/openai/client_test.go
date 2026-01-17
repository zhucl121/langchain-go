package openai

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zhucl121/langchain-go/pkg/types"
)

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name      string
		config    Config
		expectErr bool
	}{
		{
			name: "valid config",
			config: Config{
				APIKey:      "sk-test123",
				Model:       "gpt-4",
				Temperature: 0.7,
				MaxTokens:   1000,
			},
			expectErr: false,
		},
		{
			name: "missing API key",
			config: Config{
				Model: "gpt-4",
			},
			expectErr: true,
		},
		{
			name: "invalid temperature - too high",
			config: Config{
				APIKey:      "sk-test123",
				Temperature: 2.5,
			},
			expectErr: true,
		},
		{
			name: "invalid temperature - negative",
			config: Config{
				APIKey:      "sk-test123",
				Temperature: -0.5,
			},
			expectErr: true,
		},
		{
			name: "invalid TopP",
			config: Config{
				APIKey: "sk-test123",
				TopP:   1.5,
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNew(t *testing.T) {
	tests := []struct {
		name      string
		config    Config
		expectErr bool
		validate  func(t *testing.T, model *ChatModel)
	}{
		{
			name: "valid minimal config",
			config: Config{
				APIKey: "sk-test123",
			},
			expectErr: false,
			validate: func(t *testing.T, model *ChatModel) {
				assert.NotNil(t, model)
				assert.Equal(t, DefaultModel, model.GetModelName())
				assert.Equal(t, "openai", model.GetProvider())
				assert.Equal(t, DefaultBaseURL, model.config.BaseURL)
				assert.Equal(t, 0.7, model.config.Temperature)
				assert.Equal(t, DefaultTimeout, model.config.Timeout)
			},
		},
		{
			name: "valid full config",
			config: Config{
				APIKey:           "sk-test123",
				BaseURL:          "https://custom.api.com/v1",
				Model:            "gpt-4-turbo",
				Temperature:      0.5,
				MaxTokens:        2000,
				TopP:             0.9,
				FrequencyPenalty: 0.5,
				PresencePenalty:  0.5,
				Timeout:          30 * time.Second,
				User:             "test-user",
				Organization:     "org-123",
			},
			expectErr: false,
			validate: func(t *testing.T, model *ChatModel) {
				assert.NotNil(t, model)
				assert.Equal(t, "gpt-4-turbo", model.GetModelName())
				assert.Equal(t, "https://custom.api.com/v1", model.config.BaseURL)
				assert.Equal(t, 0.5, model.config.Temperature)
				assert.Equal(t, 2000, model.config.MaxTokens)
			},
		},
		{
			name: "invalid config",
			config: Config{
				APIKey:      "sk-test123",
				Temperature: 3.0, // Invalid
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model, err := New(tt.config)

			if tt.expectErr {
				assert.Error(t, err)
				assert.Nil(t, model)
			} else {
				require.NoError(t, err)
				require.NotNil(t, model)
				if tt.validate != nil {
					tt.validate(t, model)
				}
			}
		})
	}
}

func TestChatModel_BindTools(t *testing.T) {
	model, err := New(Config{APIKey: "sk-test123"})
	require.NoError(t, err)

	// 初始没有工具
	assert.Empty(t, model.GetBoundTools())

	// 绑定工具
	// 注意：实际使用需要完整的 types.Tool
	// 这里只是测试方法调用
	newModel := model.BindTools(nil)

	// BindTools 应该返回新实例
	assert.NotEqual(t, model, newModel)
	assert.IsType(t, &ChatModel{}, newModel)
}

func TestChatModel_WithStructuredOutput(t *testing.T) {
	model, err := New(Config{APIKey: "sk-test123"})
	require.NoError(t, err)

	// 初始没有 schema
	assert.Nil(t, model.GetOutputSchema())

	// 设置 structured output
	schema := types.Schema{
		Type: "object",
		Properties: map[string]types.Schema{
			"name": {Type: "string"},
		},
	}
	newModel := model.WithStructuredOutput(schema)

	// WithStructuredOutput 应该返回新实例
	assert.NotEqual(t, model, newModel)
	assert.IsType(t, &ChatModel{}, newModel)
}

func TestChatModel_buildRequest(t *testing.T) {
	// 这是一个内部方法的测试
	// 主要验证请求构建的正确性
	_ = New
	// 注意：实际测试需要真实的 types.Message
	// 这里只是演示测试结构
	t.Skip("Requires real messages for full testing")
}

func TestChatModel_GetName(t *testing.T) {
	model, err := New(Config{
		APIKey: "sk-test123",
		Model:  "gpt-4",
	})
	require.NoError(t, err)

	assert.Equal(t, "openai/gpt-4", model.GetName())
}

func TestChatModel_GetProvider(t *testing.T) {
	model, err := New(Config{
		APIKey: "sk-test123",
	})
	require.NoError(t, err)

	assert.Equal(t, "openai", model.GetProvider())
}

func TestChatModel_GetModelName(t *testing.T) {
	model, err := New(Config{
		APIKey: "sk-test123",
		Model:  "gpt-4-turbo",
	})
	require.NoError(t, err)

	assert.Equal(t, "gpt-4-turbo", model.GetModelName())
}

// Note: 完整的集成测试需要真实的 API Key 和网络请求
// 可以通过环境变量控制是否运行集成测试：
//
// func TestChatModel_Invoke_Integration(t *testing.T) {
//     apiKey := os.Getenv("OPENAI_API_KEY")
//     if apiKey == "" {
//         t.Skip("OPENAI_API_KEY not set, skipping integration test")
//     }
//
//     model, err := New(Config{
//         APIKey: apiKey,
//         Model:  "gpt-3.5-turbo",
//     })
//     require.NoError(t, err)
//
//     messages := []types.Message{
//         types.NewUserMessage("Say 'Hello, World!' and nothing else."),
//     }
//
//     response, err := model.Invoke(context.Background(), messages)
//     require.NoError(t, err)
//     assert.NotEmpty(t, response.Content)
//     assert.Contains(t, strings.ToLower(response.Content), "hello")
// }
