package openai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/zhucl121/langchain-go/pkg/types"
)

func TestStreamTokens(t *testing.T) {
	// 创建模拟服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 验证请求
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Error("missing or invalid authorization header")
		}

		// 发送 SSE 流
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("streaming not supported")
		}

		// 发送 token
		tokens := []string{"Hello", " ", "from", " ", "OpenAI"}
		for i, token := range tokens {
			chunk := map[string]any{
				"id":      "chatcmpl-123",
				"object":  "chat.completion.chunk",
				"created": time.Now().Unix(),
				"model":   "gpt-4",
				"choices": []map[string]any{
					{
						"index": 0,
						"delta": map[string]any{
							"content": token,
						},
						"finish_reason": nil,
					},
				},
			}

			if i == len(tokens)-1 {
				chunk["choices"].([]map[string]any)[0]["finish_reason"] = "stop"
			}

			data, _ := json.Marshal(chunk)
			w.Write([]byte("data: " + string(data) + "\n\n"))
			flusher.Flush()
			time.Sleep(10 * time.Millisecond)
		}

		// 发送结束标记
		w.Write([]byte("data: [DONE]\n\n"))
		flusher.Flush()
	}))
	defer server.Close()

	// 创建客户端
	client, err := New(Config{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Model:   "gpt-4",
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// 测试流式调用
	ctx := context.Background()
	messages := []types.Message{
		{Role: types.RoleUser, Content: "Hello"},
	}

	streamCh, err := client.StreamTokens(ctx, messages)
	if err != nil {
		t.Fatalf("StreamTokens failed: %v", err)
	}

	// 收集事件
	var tokens []string
	var hasStart, hasEnd bool

	for event := range streamCh {
		switch event.Type {
		case types.StreamEventStart:
			hasStart = true
		case types.StreamEventToken:
			tokens = append(tokens, event.Token)
		case types.StreamEventEnd:
			hasEnd = true
		case types.StreamEventError:
			t.Errorf("unexpected error event: %v", event.Error)
		}
	}

	// 验证结果
	if !hasStart {
		t.Error("missing start event")
	}
	if !hasEnd {
		t.Error("missing end event")
	}

	if len(tokens) != 5 {
		t.Errorf("expected 5 tokens, got %d", len(tokens))
	}

	expected := "Hello from OpenAI"
	got := ""
	for _, token := range tokens {
		got += token
	}

	if got != expected {
		t.Errorf("expected '%s', got '%s'", expected, got)
	}
}

func TestStreamWithAggregation(t *testing.T) {
	// 创建模拟服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		flusher, _ := w.(http.Flusher)

		// 发送 token
		tokens := []string{"Go", " ", "is", " ", "great"}
		for _, token := range tokens {
			chunk := map[string]any{
				"id":      "chatcmpl-456",
				"object":  "chat.completion.chunk",
				"created": time.Now().Unix(),
				"model":   "gpt-4",
				"choices": []map[string]any{
					{
						"index": 0,
						"delta": map[string]any{
							"content": token,
						},
					},
				},
			}

			data, _ := json.Marshal(chunk)
			w.Write([]byte("data: " + string(data) + "\n\n"))
			flusher.Flush()
			time.Sleep(5 * time.Millisecond)
		}

		w.Write([]byte("data: [DONE]\n\n"))
		flusher.Flush()
	}))
	defer server.Close()

	client, err := New(Config{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Model:   "gpt-4",
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	messages := []types.Message{
		{Role: types.RoleUser, Content: "Test"},
	}

	streamCh, err := client.StreamWithAggregation(ctx, messages)
	if err != nil {
		t.Fatalf("StreamWithAggregation failed: %v", err)
	}

	// 收集内容事件
	var contents []string

	for event := range streamCh {
		if event.Type == types.StreamEventContent && !event.Done {
			contents = append(contents, event.Content)
		}
	}

	// 验证聚合效果
	if len(contents) == 0 {
		t.Error("no content events received")
	}

	// 最后一个内容应该是完整的
	lastContent := contents[len(contents)-1]
	if lastContent != "Go is great" {
		t.Errorf("expected 'Go is great', got '%s'", lastContent)
	}

	// 内容应该逐步增长
	for i := 1; i < len(contents); i++ {
		if len(contents[i]) <= len(contents[i-1]) {
			t.Error("content should grow incrementally")
		}
	}
}

func TestStreamTokens_Error(t *testing.T) {
	// 创建返回错误的服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":{"message":"Invalid API key"}}`))
	}))
	defer server.Close()

	client, err := New(Config{
		APIKey:  "invalid-key",
		BaseURL: server.URL,
		Model:   "gpt-4",
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	messages := []types.Message{
		{Role: types.RoleUser, Content: "Test"},
	}

	streamCh, err := client.StreamTokens(ctx, messages)
	if err != nil {
		t.Fatalf("StreamTokens failed to start: %v", err)
	}

	// 应该收到错误事件
	hasError := false
	for event := range streamCh {
		if event.Type == types.StreamEventError {
			hasError = true
			if event.Error == nil {
				t.Error("error event should have error")
			}
		}
	}

	if !hasError {
		t.Error("expected error event")
	}
}
