package gemini

import (
	"context"
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
				APIKey: "test-api-key",
				Model:  "gemini-pro",
			},
			wantError: false,
		},
		{
			name: "missing API key",
			config: Config{
				Model: "gemini-pro",
			},
			wantError: true,
		},
		{
			name: "default model",
			config: Config{
				APIKey: "test-api-key",
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
				if client != nil && client.config.Model == "" {
					t.Error("expected default model to be set")
				}
			}
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.Model == "" {
		t.Error("expected default model")
	}

	if config.Temperature == 0 {
		t.Error("expected default temperature")
	}

	if config.BaseURL == "" {
		t.Error("expected default base URL")
	}

	if config.Timeout == 0 {
		t.Error("expected default timeout")
	}
}

func TestConvertMessages(t *testing.T) {
	client := &GeminiClient{
		config: Config{
			APIKey: "test",
			Model:  "gemini-pro",
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
			name: "system message converted to user",
			messages: []types.Message{
				{Role: types.RoleSystem, Content: "You are helpful"},
				{Role: types.RoleUser, Content: "Hello"},
			},
			wantError: false,
			wantLen:   2,
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
			contents, err := client.convertMessages(tt.messages)
			if tt.wantError {
				if err == nil {
					t.Error("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if len(contents) != tt.wantLen {
					t.Errorf("expected %d contents, got %d", tt.wantLen, len(contents))
				}
			}
		})
	}
}

func TestOptions(t *testing.T) {
	config := Config{
		APIKey: "test",
		Model:  "gemini-pro",
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

	t.Run("WithTopK", func(t *testing.T) {
		opt := WithTopK(50)
		opt(&config)
		if config.TopK != 50 {
			t.Errorf("expected topK 50, got %d", config.TopK)
		}
	})

	t.Run("WithMaxTokens", func(t *testing.T) {
		opt := WithMaxTokens(1000)
		opt(&config)
		if config.MaxTokens != 1000 {
			t.Errorf("expected maxTokens 1000, got %d", config.MaxTokens)
		}
	})

	t.Run("WithStopSequences", func(t *testing.T) {
		sequences := []string{"END", "STOP"}
		opt := WithStopSequences(sequences)
		opt(&config)
		if len(config.StopSequences) != 2 {
			t.Errorf("expected 2 stop sequences, got %d", len(config.StopSequences))
		}
	})

	t.Run("WithSafetySettings", func(t *testing.T) {
		settings := []SafetySetting{
			{Category: "HARM_CATEGORY_DANGEROUS", Threshold: "BLOCK_MEDIUM_AND_ABOVE"},
		}
		opt := WithSafetySettings(settings)
		opt(&config)
		if len(config.SafetySettings) != 1 {
			t.Errorf("expected 1 safety setting, got %d", len(config.SafetySettings))
		}
	})
}

// 集成测试（需要真实 API Key）
func TestGeminiIntegration(t *testing.T) {
	t.Skip("Integration test - requires GOOGLE_API_KEY environment variable")

	// 取消注释以运行集成测试
	// apiKey := os.Getenv("GOOGLE_API_KEY")
	// if apiKey == "" {
	// 	t.Skip("GOOGLE_API_KEY not set")
	// }
	//
	// config := Config{
	// 	APIKey: apiKey,
	// 	Model:  "gemini-pro",
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
