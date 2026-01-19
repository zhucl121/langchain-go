package bedrock

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
				Region:    "us-east-1",
				AccessKey: "test-access-key",
				SecretKey: "test-secret-key",
				Model:     "anthropic.claude-v2",
			},
			wantError: false,
		},
		{
			name: "missing region",
			config: Config{
				AccessKey: "test-access-key",
				SecretKey: "test-secret-key",
			},
			wantError: true,
		},
		{
			name: "missing access key",
			config: Config{
				Region:    "us-east-1",
				SecretKey: "test-secret-key",
			},
			wantError: true,
		},
		{
			name: "missing secret key",
			config: Config{
				Region:    "us-east-1",
				AccessKey: "test-access-key",
			},
			wantError: true,
		},
		{
			name: "default model",
			config: Config{
				Region:    "us-east-1",
				AccessKey: "test-access-key",
				SecretKey: "test-secret-key",
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
			}
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.Region == "" {
		t.Error("expected default region")
	}

	if config.Model == "" {
		t.Error("expected default model")
	}

	if config.Temperature == 0 {
		t.Error("expected default temperature")
	}

	if config.Timeout == 0 {
		t.Error("expected default timeout")
	}
}

func TestModelDetection(t *testing.T) {
	tests := []struct {
		name     string
		model    string
		checkFn  func(string) bool
		expected bool
	}{
		{"anthropic model", "anthropic.claude-v2", isAnthropicModel, true},
		{"titan model", "amazon.titan-text-v1", isTitanModel, true},
		{"llama model", "meta.llama2-13b", isLlamaModel, true},
		{"not anthropic", "amazon.titan", isAnthropicModel, false},
		{"not titan", "anthropic.claude", isTitanModel, false},
		{"not llama", "anthropic.claude", isLlamaModel, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.checkFn(tt.model)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestBuildAnthropicRequest(t *testing.T) {
	client := &BedrockClient{
		config: Config{
			Region:    "us-east-1",
			AccessKey: "test",
			SecretKey: "test",
			Model:     "anthropic.claude-v2",
		},
	}

	messages := []types.Message{
		{Role: types.RoleSystem, Content: "You are helpful"},
		{Role: types.RoleUser, Content: "Hello"},
		{Role: types.RoleAssistant, Content: "Hi there!"},
	}

	reqBody, err := client.buildAnthropicRequest(&client.config, messages)
	if err != nil {
		t.Fatalf("Failed to build request: %v", err)
	}

	// 验证系统消息
	if system, ok := reqBody["system"].(string); !ok || system != "You are helpful" {
		t.Error("Expected system prompt in request")
	}

	// 验证消息格式
	msgs, ok := reqBody["messages"].([]map[string]interface{})
	if !ok {
		t.Fatal("Expected messages array")
	}

	if len(msgs) != 2 {
		t.Errorf("Expected 2 messages (excluding system), got %d", len(msgs))
	}

	// 验证参数
	if _, ok := reqBody["max_tokens"]; !ok {
		t.Error("Expected max_tokens in request")
	}

	if _, ok := reqBody["temperature"]; !ok {
		t.Error("Expected temperature in request")
	}
}

func TestBuildTitanRequest(t *testing.T) {
	client := &BedrockClient{
		config: Config{
			Region:    "us-east-1",
			AccessKey: "test",
			SecretKey: "test",
			Model:     "amazon.titan-text-v1",
		},
	}

	messages := []types.Message{
		{Role: types.RoleUser, Content: "Hello"},
	}

	reqBody, err := client.buildTitanRequest(&client.config, messages)
	if err != nil {
		t.Fatalf("Failed to build request: %v", err)
	}

	if _, ok := reqBody["inputText"]; !ok {
		t.Error("Expected inputText in Titan request")
	}

	if _, ok := reqBody["textGenerationConfig"]; !ok {
		t.Error("Expected textGenerationConfig in Titan request")
	}
}

func TestBuildLlamaRequest(t *testing.T) {
	client := &BedrockClient{
		config: Config{
			Region:    "us-east-1",
			AccessKey: "test",
			SecretKey: "test",
			Model:     "meta.llama2-13b",
		},
	}

	messages := []types.Message{
		{Role: types.RoleSystem, Content: "You are helpful"},
		{Role: types.RoleUser, Content: "Hello"},
	}

	reqBody, err := client.buildLlamaRequest(&client.config, messages)
	if err != nil {
		t.Fatalf("Failed to build request: %v", err)
	}

	prompt, ok := reqBody["prompt"].(string)
	if !ok || prompt == "" {
		t.Error("Expected prompt in Llama request")
	}

	// Llama 格式应该包含特殊标记
	if len(prompt) < 10 {
		t.Error("Expected formatted prompt with special tokens")
	}
}

func TestOptions(t *testing.T) {
	config := Config{
		Region:    "us-east-1",
		AccessKey: "test",
		SecretKey: "test",
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

	t.Run("WithStopSequences", func(t *testing.T) {
		sequences := []string{"END", "STOP"}
		opt := WithStopSequences(sequences)
		opt(&config)
		if len(config.StopSequences) != 2 {
			t.Errorf("expected 2 stop sequences, got %d", len(config.StopSequences))
		}
	})
}
