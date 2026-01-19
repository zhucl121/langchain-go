package azure

import (
	"testing"

	"github.com/zhucl121/langchain-go/pkg/types"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name      string
		config    Config
		wantError bool
	}{
		{
			name: "valid config",
			config: Config{
				Endpoint:   "https://test.openai.azure.com",
				APIKey:     "test-api-key",
				Deployment: "gpt-35-turbo",
				APIVersion: "2024-02-01",
			},
			wantError: false,
		},
		{
			name: "missing endpoint",
			config: Config{
				APIKey:     "test-api-key",
				Deployment: "gpt-35-turbo",
			},
			wantError: true,
		},
		{
			name: "missing API key",
			config: Config{
				Endpoint:   "https://test.openai.azure.com",
				Deployment: "gpt-35-turbo",
			},
			wantError: true,
		},
		{
			name: "missing deployment",
			config: Config{
				Endpoint: "https://test.openai.azure.com",
				APIKey:   "test-api-key",
			},
			wantError: true,
		},
		{
			name: "default API version",
			config: Config{
				Endpoint:   "https://test.openai.azure.com",
				APIKey:     "test-api-key",
				Deployment: "gpt-35-turbo",
			},
			wantError: false,
		},
		{
			name: "endpoint with trailing slash",
			config: Config{
				Endpoint:   "https://test.openai.azure.com/",
				APIKey:     "test-api-key",
				Deployment: "gpt-35-turbo",
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := New(tt.config)
			if tt.wantError {
				if err == nil {
					t.Error("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if client == nil {
					t.Error("expected client, got nil")
				}
				if client != nil {
					// 验证端点格式
					if client.config.Endpoint[len(client.config.Endpoint)-1] == '/' {
						t.Error("endpoint should not end with slash")
					}
					// 验证默认 API 版本
					if client.config.APIVersion == "" {
						t.Error("expected API version to be set")
					}
				}
			}
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.APIVersion == "" {
		t.Error("expected default API version")
	}

	if config.Temperature == 0 {
		t.Error("expected default temperature")
	}

	if config.Timeout == 0 {
		t.Error("expected default timeout")
	}
}

func TestConvertMessages(t *testing.T) {
	client := &AzureOpenAIClient{
		config: Config{
			Endpoint:   "https://test.openai.azure.com",
			APIKey:     "test",
			Deployment: "test",
		},
	}

	tests := []struct {
		name      string
		messages  []types.Message
		wantError bool
		wantLen   int
	}{
		{
			name: "single user message",
			messages: []types.Message{
				{Role: types.RoleUser, Content: "Hello"},
			},
			wantError: false,
			wantLen:   1,
		},
		{
			name: "user and assistant messages",
			messages: []types.Message{
				{Role: types.RoleUser, Content: "Hello"},
				{Role: types.RoleAssistant, Content: "Hi there!"},
			},
			wantError: false,
			wantLen:   2,
		},
		{
			name: "all role types",
			messages: []types.Message{
				{Role: types.RoleSystem, Content: "You are helpful"},
				{Role: types.RoleUser, Content: "Hello"},
				{Role: types.RoleAssistant, Content: "Hi there!"},
			},
			wantError: false,
			wantLen:   3,
		},
		{
			name:      "empty messages",
			messages:  []types.Message{},
			wantError: true,
			wantLen:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			azureMessages, err := client.convertMessages(tt.messages)
			if tt.wantError {
				if err == nil {
					t.Error("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if len(azureMessages) != tt.wantLen {
					t.Errorf("expected %d messages, got %d", tt.wantLen, len(azureMessages))
				}
			}
		})
	}
}

func TestOptions(t *testing.T) {
	config := Config{
		Endpoint:   "https://test.openai.azure.com",
		APIKey:     "test",
		Deployment: "test",
	}

	t.Run("WithTemperature", func(t *testing.T) {
		opt := WithTemperature(0.9)
		opt(&config)
		if config.Temperature != 0.9 {
			t.Errorf("expected temperature 0.9, got %f", config.Temperature)
		}
	})

	t.Run("WithTopP", func(t *testing.T) {
		opt := WithTopP(0.8)
		opt(&config)
		if config.TopP != 0.8 {
			t.Errorf("expected topP 0.8, got %f", config.TopP)
		}
	})

	t.Run("WithMaxTokens", func(t *testing.T) {
		opt := WithMaxTokens(1000)
		opt(&config)
		if config.MaxTokens != 1000 {
			t.Errorf("expected maxTokens 1000, got %d", config.MaxTokens)
		}
	})

	t.Run("WithPresencePenalty", func(t *testing.T) {
		opt := WithPresencePenalty(0.5)
		opt(&config)
		if config.PresencePenalty != 0.5 {
			t.Errorf("expected presencePenalty 0.5, got %f", config.PresencePenalty)
		}
	})

	t.Run("WithFrequencyPenalty", func(t *testing.T) {
		opt := WithFrequencyPenalty(0.5)
		opt(&config)
		if config.FrequencyPenalty != 0.5 {
			t.Errorf("expected frequencyPenalty 0.5, got %f", config.FrequencyPenalty)
		}
	})

	t.Run("WithStop", func(t *testing.T) {
		stop := []string{"END", "STOP"}
		opt := WithStop(stop)
		opt(&config)
		if len(config.Stop) != 2 {
			t.Errorf("expected 2 stop sequences, got %d", len(config.Stop))
		}
	})
}

func TestParseResponse(t *testing.T) {
	client := &AzureOpenAIClient{
		config: Config{
			Endpoint:   "https://test.openai.azure.com",
			APIKey:     "test",
			Deployment: "test",
		},
	}

	t.Run("valid response", func(t *testing.T) {
		response := &AzureResponse{
			Choices: []AzureChoice{
				{
					Message: AzureMessage{
						Role:    "assistant",
						Content: "Hello there!",
					},
				},
			},
		}

		msg, err := client.parseResponse(response)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if msg.Content != "Hello there!" {
			t.Errorf("expected 'Hello there!', got '%s'", msg.Content)
		}

		if msg.Role != types.RoleAssistant {
			t.Errorf("expected assistant role, got %s", msg.Role)
		}
	})

	t.Run("empty choices", func(t *testing.T) {
		response := &AzureResponse{
			Choices: []AzureChoice{},
		}

		_, err := client.parseResponse(response)
		if err == nil {
			t.Error("expected error for empty choices")
		}
	})
}

// 集成测试（需要真实 Azure OpenAI 资源）
func TestAzureOpenAIIntegration(t *testing.T) {
	t.Skip("Integration test - requires Azure OpenAI configuration")

	// 取消注释以运行集成测试
	// endpoint := os.Getenv("AZURE_OPENAI_ENDPOINT")
	// apiKey := os.Getenv("AZURE_OPENAI_API_KEY")
	// deployment := os.Getenv("AZURE_OPENAI_DEPLOYMENT")
	//
	// if endpoint == "" || apiKey == "" || deployment == "" {
	// 	t.Skip("Azure OpenAI configuration not set")
	// }
	//
	// config := Config{
	// 	Endpoint:   endpoint,
	// 	APIKey:     apiKey,
	// 	Deployment: deployment,
	// 	APIVersion: "2024-02-01",
	// }
	// client, err := New(config)
	// if err != nil {
	// 	t.Fatalf("Failed to create client: %v", err)
	// }
	//
	// ctx := context.Background()
	// messages := []types.Message{
	// 	types.NewUserMessage("Say hello in one word"),
	// }
	//
	// response, err := client.Invoke(ctx, messages)
	// if err != nil {
	// 	t.Fatalf("Failed to invoke: %v", err)
	// }
	//
	// if response.Content == "" {
	// 	t.Error("Expected non-empty response")
	// }
	//
	// t.Logf("Response: %s", response.Content)
}
