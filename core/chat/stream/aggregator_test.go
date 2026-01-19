package stream

import (
	"errors"
	"testing"

	"github.com/zhucl121/langchain-go/pkg/types"
)

func TestStreamAggregator_Basic(t *testing.T) {
	agg := NewStreamAggregator()

	// 添加 token 事件
	agg.Add(types.NewTokenEvent("Hello"))
	agg.Add(types.NewTokenEvent(" "))
	agg.Add(types.NewTokenEvent("World"))

	content := agg.GetContent()
	if content != "Hello World" {
		t.Errorf("expected 'Hello World', got '%s'", content)
	}

	// 检查事件数量
	if agg.GetEventCount() != 3 {
		t.Errorf("expected 3 events, got %d", agg.GetEventCount())
	}

	// 获取结果
	message, err := agg.GetResult()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if message.Content != "Hello World" {
		t.Errorf("expected 'Hello World', got '%s'", message.Content)
	}

	if message.Role != types.RoleAssistant {
		t.Errorf("expected role %s, got %s", types.RoleAssistant, message.Role)
	}
}

func TestStreamAggregator_ToolCalls(t *testing.T) {
	agg := NewStreamAggregator()

	// 添加工具调用事件
	toolCall := &types.ToolCall{
		ID:   "call_123",
		Type: "function",
		Function: types.FunctionCall{
			Name:      "get_weather",
			Arguments: `{"city":"Beijing"}`,
		},
	}

	agg.Add(types.NewToolCallEvent(toolCall))

	toolCalls := agg.GetToolCalls()
	if len(toolCalls) != 1 {
		t.Errorf("expected 1 tool call, got %d", len(toolCalls))
	}

	if toolCalls[0].ID != "call_123" {
		t.Errorf("expected ID 'call_123', got '%s'", toolCalls[0].ID)
	}
}

func TestStreamAggregator_Error(t *testing.T) {
	agg := NewStreamAggregator()

	// 添加一些正常事件
	agg.Add(types.NewTokenEvent("Hello"))

	// 添加错误事件
	testErr := errors.New("test error")
	err := agg.Add(types.NewErrorEvent(testErr))

	if err == nil {
		t.Error("expected error, got nil")
	}

	if !agg.HasError() {
		t.Error("expected HasError() to be true")
	}

	if agg.GetError() != testErr {
		t.Errorf("expected error '%v', got '%v'", testErr, agg.GetError())
	}

	// 获取结果应该返回错误
	_, err = agg.GetResult()
	if err == nil {
		t.Error("expected GetResult() to return error")
	}
}

func TestStreamAggregator_Metadata(t *testing.T) {
	agg := NewStreamAggregator()

	// 添加带元数据的事件
	event := types.NewTokenEvent("test")
	event = event.WithMetadata("key1", "value1")
	event = event.WithMetadata("key2", 123)

	agg.Add(event)

	message, _ := agg.GetResult()
	if message.Metadata["key1"] != "value1" {
		t.Errorf("expected metadata key1='value1', got '%v'", message.Metadata["key1"])
	}

	if message.Metadata["key2"] != 123 {
		t.Errorf("expected metadata key2=123, got '%v'", message.Metadata["key2"])
	}
}

func TestStreamAggregator_Reset(t *testing.T) {
	agg := NewStreamAggregator()

	// 添加一些事件
	agg.Add(types.NewTokenEvent("Hello"))
	agg.Add(types.NewTokenEvent(" World"))

	if agg.GetContent() != "Hello World" {
		t.Error("content should be 'Hello World' before reset")
	}

	// 重置
	agg.Reset()

	if agg.GetContent() != "" {
		t.Error("content should be empty after reset")
	}

	if agg.GetEventCount() != 0 {
		t.Error("event count should be 0 after reset")
	}
}

func TestStreamAggregator_Concurrent(t *testing.T) {
	agg := NewStreamAggregator()

	// 并发添加事件
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(n int) {
			agg.Add(types.NewTokenEvent("test"))
			done <- true
		}(i)
	}

	// 等待所有 goroutine 完成
	for i := 0; i < 10; i++ {
		<-done
	}

	if agg.GetEventCount() != 10 {
		t.Errorf("expected 10 events, got %d", agg.GetEventCount())
	}
}

func TestStreamAggregator_ContentEvent(t *testing.T) {
	agg := NewStreamAggregator()

	// 先添加一些 token
	agg.Add(types.NewTokenEvent("Hello"))
	agg.Add(types.NewTokenEvent(" World"))

	// 然后添加完整内容事件（应该覆盖）
	contentEvent := types.StreamEvent{
		Type:    types.StreamEventContent,
		Content: "Complete message",
	}
	agg.Add(contentEvent)

	if agg.GetContent() != "Complete message" {
		t.Errorf("expected 'Complete message', got '%s'", agg.GetContent())
	}
}
