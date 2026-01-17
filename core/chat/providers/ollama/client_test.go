package ollama

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/zhucl121/langchain-go/core/runnable"
	"github.com/zhucl121/langchain-go/pkg/types"
)

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: Config{
				Model:       "llama2",
				Temperature: 0.7,
				TopP:        0.9,
			},
			wantErr: false,
		},
		{
			name: "missing model",
			config: Config{
				Temperature: 0.7,
			},
			wantErr: true,
		},
		{
			name: "invalid temperature (too high)",
			config: Config{
				Model:       "llama2",
				Temperature: 2.5,
			},
			wantErr: true,
		},
		{
			name: "invalid temperature (negative)",
			config: Config{
				Model:       "llama2",
				Temperature: -0.1,
			},
			wantErr: true,
		},
		{
			name: "invalid top_p",
			config: Config{
				Model: "llama2",
				TopP:  1.5,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid config with defaults",
			config: Config{
				Model: "llama2",
			},
			wantErr: false,
		},
		{
			name: "valid config with custom values",
			config: Config{
				BaseURL:       "http://custom:11434",
				Model:         "mistral",
				Temperature:   0.8,
				NumPredict:    100,
				TopK:          50,
				TopP:          0.95,
				RepeatPenalty: 1.2,
				Timeout:       60 * time.Second,
			},
			wantErr: false,
		},
		{
			name:    "invalid config",
			config:  Config{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model, err := New(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if model == nil {
					t.Error("New() returned nil model")
					return
				}
				// Check defaults
				if model.config.BaseURL == "" {
					t.Error("BaseURL should have default value")
				}
				if model.config.Temperature == 0 {
					t.Error("Temperature should have default value")
				}
				if model.config.TopK == 0 {
					t.Error("TopK should have default value")
				}
				if model.config.TopP == 0 {
					t.Error("TopP should have default value")
				}
				if model.config.RepeatPenalty == 0 {
					t.Error("RepeatPenalty should have default value")
				}
				if model.config.Timeout == 0 {
					t.Error("Timeout should have default value")
				}
			}
		})
	}
}

func TestChatModel_Invoke(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check request
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if r.URL.Path != "/api/chat" {
			t.Errorf("Expected /api/chat path, got %s", r.URL.Path)
		}

		// Parse request
		var req ollamaRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("Failed to decode request: %v", err)
		}

		// Check request fields
		if req.Model != "llama2" {
			t.Errorf("Expected model llama2, got %s", req.Model)
		}
		if req.Stream {
			t.Error("Expected stream=false")
		}
		if len(req.Messages) == 0 {
			t.Error("Expected messages")
		}

		// Send response
		resp := ollamaResponse{
			Model:     "llama2",
			CreatedAt: time.Now().Format(time.RFC3339),
			Message: ollamaMessage{
				Role:    "assistant",
				Content: "Hello! How can I help you?",
			},
			Done: true,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Create model
	model, err := New(Config{
		BaseURL: server.URL,
		Model:   "llama2",
	})
	if err != nil {
		t.Fatalf("Failed to create model: %v", err)
	}

	// Test invoke
	ctx := context.Background()
	messages := []types.Message{
		{Role: types.RoleUser, Content: "Hello"},
	}

	response, err := model.Invoke(ctx, messages)
	if err != nil {
		t.Fatalf("Invoke() error = %v", err)
	}

	if response.Role != types.RoleAssistant {
		t.Errorf("Expected assistant role, got %s", response.Role)
	}
	if response.Content == "" {
		t.Error("Expected non-empty content")
	}
}

func TestChatModel_Stream(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check request
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		// Parse request
		var req ollamaRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("Failed to decode request: %v", err)
		}

		if !req.Stream {
			t.Error("Expected stream=true")
		}

		// Send streaming response
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Transfer-Encoding", "chunked")

		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("Expected http.ResponseWriter to be an http.Flusher")
		}

		// Send multiple chunks
		chunks := []string{"Hello", " ", "World", "!"}
		for i, chunk := range chunks {
			resp := ollamaStreamChunk{
				Model:     "llama2",
				CreatedAt: time.Now().Format(time.RFC3339),
				Message: ollamaMessage{
					Role:    "assistant",
					Content: chunk,
				},
				Done: i == len(chunks)-1,
			}
			json.NewEncoder(w).Encode(resp)
			flusher.Flush()
		}
	}))
	defer server.Close()

	// Create model
	model, err := New(Config{
		BaseURL: server.URL,
		Model:   "llama2",
	})
	if err != nil {
		t.Fatalf("Failed to create model: %v", err)
	}

	// Test stream
	ctx := context.Background()
	messages := []types.Message{
		{Role: types.RoleUser, Content: "Hello"},
	}

	stream, err := model.Stream(ctx, messages)
	if err != nil {
		t.Fatalf("Stream() error = %v", err)
	}

	// Collect events
	var content string
	eventCount := 0
	hasStart := false
	hasEnd := false

	for event := range stream {
		eventCount++
		switch event.Type {
		case runnable.EventStart:
			hasStart = true
		case runnable.EventStream:
			content += event.Data.Content
		case runnable.EventEnd:
			hasEnd = true
		case runnable.EventError:
			t.Fatalf("Stream error: %v", event.Error)
		}
	}

	if !hasStart {
		t.Error("Expected start event")
	}
	if !hasEnd {
		t.Error("Expected end event")
	}
	if content == "" {
		t.Error("Expected non-empty content")
	}
	if eventCount < 3 {
		t.Errorf("Expected at least 3 events (start, stream, end), got %d", eventCount)
	}
}

func TestChatModel_InvalidMessages(t *testing.T) {
	model, err := New(Config{
		BaseURL: "http://localhost:11434",
		Model:   "llama2",
	})
	if err != nil {
		t.Fatalf("Failed to create model: %v", err)
	}

	ctx := context.Background()

	tests := []struct {
		name     string
		messages []types.Message
	}{
		{
			name:     "empty messages",
			messages: []types.Message{},
		},
		{
			name: "message with empty content",
			messages: []types.Message{
				{Role: types.RoleUser, Content: ""},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := model.Invoke(ctx, tt.messages)
			if err == nil {
				t.Error("Expected error for invalid messages")
			}
		})
	}
}

func TestChatModel_GetType(t *testing.T) {
	model, err := New(Config{
		Model: "llama2",
	})
	if err != nil {
		t.Fatalf("Failed to create model: %v", err)
	}

	if model.GetType() != "ollama" {
		t.Errorf("Expected type 'ollama', got '%s'", model.GetType())
	}
}

// Benchmark tests

func BenchmarkChatModel_Invoke(b *testing.B) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := ollamaResponse{
			Model:     "llama2",
			CreatedAt: time.Now().Format(time.RFC3339),
			Message: ollamaMessage{
				Role:    "assistant",
				Content: "Hello! How can I help you?",
			},
			Done: true,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	model, _ := New(Config{
		BaseURL: server.URL,
		Model:   "llama2",
	})

	ctx := context.Background()
	messages := []types.Message{
		{Role: types.RoleUser, Content: "Hello"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = model.Invoke(ctx, messages)
	}
}
