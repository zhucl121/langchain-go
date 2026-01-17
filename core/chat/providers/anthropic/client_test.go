package anthropic

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
				APIKey:      "sk-ant-test123",
				Model:       "claude-3-opus-20240229",
				Temperature: 0.7,
				MaxTokens:   1000,
			},
			expectErr: false,
		},
		{
			name: "missing API key",
			config: Config{
				Model:     "claude-3-opus-20240229",
				MaxTokens: 1000,
			},
			expectErr: true,
		},
		{
			name: "missing MaxTokens",
			config: Config{
				APIKey: "sk-ant-test123",
				Model:  "claude-3-opus-20240229",
			},
			expectErr: true,
		},
		{
			name: "negative MaxTokens",
			config: Config{
				APIKey:    "sk-ant-test123",
				MaxTokens: -1,
			},
			expectErr: true,
		},
		{
			name: "invalid temperature - too high",
			config: Config{
				APIKey:      "sk-ant-test123",
				MaxTokens:   1000,
				Temperature: 1.5,
			},
			expectErr: true,
		},
		{
			name: "invalid temperature - negative",
			config: Config{
				APIKey:      "sk-ant-test123",
				MaxTokens:   1000,
				Temperature: -0.5,
			},
			expectErr: true,
		},
		{
			name: "invalid TopP",
			config: Config{
				APIKey:    "sk-ant-test123",
				MaxTokens: 1000,
				TopP:      1.5,
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
				APIKey:    "sk-ant-test123",
				MaxTokens: 1000,
			},
			expectErr: false,
			validate: func(t *testing.T, model *ChatModel) {
				assert.NotNil(t, model)
				assert.Equal(t, DefaultModel, model.GetModelName())
				assert.Equal(t, "anthropic", model.GetProvider())
				assert.Equal(t, DefaultBaseURL, model.config.BaseURL)
				assert.Equal(t, 1.0, model.config.Temperature)
				assert.Equal(t, DefaultTimeout, model.config.Timeout)
			},
		},
		{
			name: "valid full config",
			config: Config{
				APIKey:      "sk-ant-test123",
				BaseURL:     "https://custom.anthropic.com",
				Model:       "claude-3-sonnet-20240229",
				Temperature: 0.5,
				MaxTokens:   2000,
				TopP:        0.9,
				TopK:        50,
				Timeout:     30 * time.Second,
			},
			expectErr: false,
			validate: func(t *testing.T, model *ChatModel) {
				assert.NotNil(t, model)
				assert.Equal(t, "claude-3-sonnet-20240229", model.GetModelName())
				assert.Equal(t, "https://custom.anthropic.com", model.config.BaseURL)
				assert.Equal(t, 0.5, model.config.Temperature)
				assert.Equal(t, 2000, model.config.MaxTokens)
				assert.Equal(t, 0.9, model.config.TopP)
				assert.Equal(t, 50, model.config.TopK)
			},
		},
		{
			name: "invalid config - no MaxTokens",
			config: Config{
				APIKey: "sk-ant-test123",
			},
			expectErr: true,
		},
		{
			name: "invalid config - bad temperature",
			config: Config{
				APIKey:      "sk-ant-test123",
				MaxTokens:   1000,
				Temperature: 2.0,
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
	model, err := New(Config{
		APIKey:    "sk-ant-test123",
		MaxTokens: 1000,
	})
	require.NoError(t, err)

	// 初始没有工具
	assert.Empty(t, model.GetBoundTools())

	// 绑定工具
	newModel := model.BindTools(nil)

	// BindTools 应该返回新实例
	assert.NotEqual(t, model, newModel)
	assert.IsType(t, &ChatModel{}, newModel)
}

func TestChatModel_WithStructuredOutput(t *testing.T) {
	model, err := New(Config{
		APIKey:    "sk-ant-test123",
		MaxTokens: 1000,
	})
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

func TestChatModel_GetName(t *testing.T) {
	model, err := New(Config{
		APIKey:    "sk-ant-test123",
		Model:     "claude-3-opus-20240229",
		MaxTokens: 1000,
	})
	require.NoError(t, err)

	assert.Equal(t, "anthropic/claude-3-opus-20240229", model.GetName())
}

func TestChatModel_GetProvider(t *testing.T) {
	model, err := New(Config{
		APIKey:    "sk-ant-test123",
		MaxTokens: 1000,
	})
	require.NoError(t, err)

	assert.Equal(t, "anthropic", model.GetProvider())
}

func TestChatModel_GetModelName(t *testing.T) {
	model, err := New(Config{
		APIKey:    "sk-ant-test123",
		Model:     "claude-3-haiku-20240307",
		MaxTokens: 1000,
	})
	require.NoError(t, err)

	assert.Equal(t, "claude-3-haiku-20240307", model.GetModelName())
}

func TestChatModel_buildRequest(t *testing.T) {
	// 这是一个内部方法的测试
	// 主要验证请求构建的正确性
	_ = New
	// 注意：实际测试需要真实的 types.Message
	// 这里只是演示测试结构
	t.Skip("Requires real messages for full testing")
}

// Note: 完整的集成测试需要真实的 API Key 和网络请求
// 可以通过环境变量控制是否运行集成测试：
//
// func TestChatModel_Invoke_Integration(t *testing.T) {
//     apiKey := os.Getenv("ANTHROPIC_API_KEY")
//     if apiKey == "" {
//         t.Skip("ANTHROPIC_API_KEY not set, skipping integration test")
//     }
//
//     model, err := New(Config{
//         APIKey:    apiKey,
//         Model:     "claude-3-haiku-20240307",
//         MaxTokens: 100,
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
