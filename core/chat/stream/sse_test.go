package stream

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/zhucl121/langchain-go/pkg/types"
)

func TestSSEWriter_WriteEvent(t *testing.T) {
	var buf bytes.Buffer
	sse := NewSSEWriter(&buf)

	event := types.NewTokenEvent("Hello")
	err := sse.WriteEvent(event)
	if err != nil {
		t.Fatalf("WriteEvent failed: %v", err)
	}

	output := buf.String()

	// 检查 SSE 格式
	if !strings.Contains(output, "event: token\n") {
		t.Error("output should contain event type")
	}

	if !strings.Contains(output, "data: ") {
		t.Error("output should contain data field")
	}

	if !strings.Contains(output, "Hello") {
		t.Error("output should contain token content")
	}

	// SSE 格式应该以双换行结束
	if !strings.HasSuffix(output, "\n\n") {
		t.Error("output should end with double newline")
	}
}

func TestSSEWriter_WriteToken(t *testing.T) {
	var buf bytes.Buffer
	sse := NewSSEWriter(&buf)

	err := sse.WriteToken("test")
	if err != nil {
		t.Fatalf("WriteToken failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "test") {
		t.Error("output should contain token")
	}
}

func TestSSEWriter_WriteError(t *testing.T) {
	var buf bytes.Buffer
	sse := NewSSEWriter(&buf)

	testErr := errors.New("test error")
	err := sse.WriteError(testErr)
	if err != nil {
		t.Fatalf("WriteError failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "event: error\n") {
		t.Error("output should contain error event type")
	}
}

func TestSSEWriter_WriteData(t *testing.T) {
	var buf bytes.Buffer
	sse := NewSSEWriter(&buf)

	err := sse.WriteData("custom data")
	if err != nil {
		t.Fatalf("WriteData failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "data: custom data\n") {
		t.Error("output should contain data")
	}
}

func TestSSEWriter_WriteComment(t *testing.T) {
	var buf bytes.Buffer
	sse := NewSSEWriter(&buf)

	err := sse.WriteComment("keep-alive")
	if err != nil {
		t.Fatalf("WriteComment failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, ": keep-alive\n") {
		t.Error("output should contain comment")
	}
}

func TestSSEWriter_WriteEnd(t *testing.T) {
	var buf bytes.Buffer
	sse := NewSSEWriter(&buf)

	err := sse.WriteEnd()
	if err != nil {
		t.Fatalf("WriteEnd failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "event: end\n") {
		t.Error("output should contain end event")
	}
}

func TestSSEWriter_EventCount(t *testing.T) {
	var buf bytes.Buffer
	sse := NewSSEWriter(&buf)

	if sse.GetEventCount() != 0 {
		t.Error("initial event count should be 0")
	}

	sse.WriteToken("test1")
	sse.WriteToken("test2")
	sse.WriteError(errors.New("test"))

	if sse.GetEventCount() != 3 {
		t.Errorf("expected 3 events, got %d", sse.GetEventCount())
	}
}

func TestSSEWriter_MultipleEvents(t *testing.T) {
	var buf bytes.Buffer
	sse := NewSSEWriter(&buf)

	// 写入多个事件
	events := []types.StreamEvent{
		types.NewTokenEvent("Hello"),
		types.NewTokenEvent(" "),
		types.NewTokenEvent("World"),
		{Type: types.StreamEventEnd, Done: true},
	}

	for _, event := range events {
		if err := sse.WriteEvent(event); err != nil {
			t.Fatalf("WriteEvent failed: %v", err)
		}
	}

	output := buf.String()

	// 检查输出包含所有事件
	eventCount := strings.Count(output, "event: ")
	if eventCount != 4 {
		t.Errorf("expected 4 events in output, got %d", eventCount)
	}

	// 检查每个事件都以双换行分隔
	dataCount := strings.Count(output, "data: ")
	if dataCount != 4 {
		t.Errorf("expected 4 data fields, got %d", dataCount)
	}
}

func TestSSEWriter_Close(t *testing.T) {
	var buf bytes.Buffer
	sse := NewSSEWriter(&buf)

	err := sse.Close()
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "event: end\n") {
		t.Error("Close should write end event")
	}
}

// mockFlusher 用于测试 Flusher 接口
type mockFlusher struct {
	bytes.Buffer
	flushCount int
}

func (m *mockFlusher) Flush() {
	m.flushCount++
}

func TestSSEWriter_Flusher(t *testing.T) {
	mock := &mockFlusher{}
	sse := NewSSEWriter(mock)

	// 写入事件应该触发 flush
	sse.WriteToken("test")

	if mock.flushCount != 1 {
		t.Errorf("expected 1 flush, got %d", mock.flushCount)
	}

	// 写入多个事件
	sse.WriteToken("test2")
	sse.WriteToken("test3")

	if mock.flushCount != 3 {
		t.Errorf("expected 3 flushes, got %d", mock.flushCount)
	}
}
