// Package stream 提供流式处理的辅助工具。
//
// 包含 StreamAggregator（流聚合器）和 SSEWriter（SSE 写入器）。
//
package stream

import (
	"fmt"
	"strings"
	"sync"

	"github.com/zhucl121/langchain-go/pkg/types"
)

// StreamAggregator 用于聚合流式事件，构建完整的响应。
//
// StreamAggregator 是线程安全的，可以在并发环境中使用。
//
// 示例：
//
//	aggregator := stream.NewStreamAggregator()
//	for event := range streamCh {
//	    aggregator.Add(event)
//	    fmt.Print(event.Token) // 实时显示
//	}
//	message, _ := aggregator.GetResult()
//
type StreamAggregator struct {
	mu sync.RWMutex

	events    []types.StreamEvent
	content   strings.Builder
	toolCalls []types.ToolCall
	metadata  map[string]any
	eventCount int
	hasError  bool
	lastError error
}

// NewStreamAggregator 创建新的流聚合器。
//
// 返回：
//   - *StreamAggregator: 流聚合器实例
//
func NewStreamAggregator() *StreamAggregator {
	return &StreamAggregator{
		events:    make([]types.StreamEvent, 0, 100),
		toolCalls: make([]types.ToolCall, 0),
		metadata:  make(map[string]any),
	}
}

// Add 添加一个流式事件。
//
// 参数：
//   - event: 流式事件
//
// 返回：
//   - error: 错误（如果事件是错误类型）
//
func (a *StreamAggregator) Add(event types.StreamEvent) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.events = append(a.events, event)
	a.eventCount++

	switch event.Type {
	case types.StreamEventToken:
		// 累积 token
		if event.Token != "" {
			a.content.WriteString(event.Token)
		} else if event.Delta != "" {
			a.content.WriteString(event.Delta)
		}

	case types.StreamEventContent:
		// 设置完整内容
		if event.Content != "" {
			a.content.Reset()
			a.content.WriteString(event.Content)
		}

	case types.StreamEventToolCall:
		// 添加工具调用
		if event.ToolCall != nil {
			a.toolCalls = append(a.toolCalls, *event.ToolCall)
		}

	case types.StreamEventError:
		// 记录错误
		a.hasError = true
		a.lastError = event.Error
		return event.Error
	}

	// 合并元数据
	for k, v := range event.Metadata {
		a.metadata[k] = v
	}

	return nil
}

// GetContent 获取当前聚合的内容。
//
// 返回：
//   - string: 聚合的内容
//
func (a *StreamAggregator) GetContent() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.content.String()
}

// GetResult 获取最终的消息结果。
//
// 返回：
//   - *types.Message: 完整的消息
//   - error: 错误
//
func (a *StreamAggregator) GetResult() (*types.Message, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if a.hasError {
		return nil, a.lastError
	}

	message := &types.Message{
		Role:     types.RoleAssistant,
		Content:  a.content.String(),
		Metadata: a.metadata,
	}

	if len(a.toolCalls) > 0 {
		message.ToolCalls = a.toolCalls
	}

	return message, nil
}

// GetEvents 获取所有事件。
//
// 返回：
//   - []types.StreamEvent: 所有事件的副本
//
func (a *StreamAggregator) GetEvents() []types.StreamEvent {
	a.mu.RLock()
	defer a.mu.RUnlock()

	events := make([]types.StreamEvent, len(a.events))
	copy(events, a.events)
	return events
}

// GetEventCount 获取事件数量。
//
// 返回：
//   - int: 事件数量
//
func (a *StreamAggregator) GetEventCount() int {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.eventCount
}

// GetToolCalls 获取所有工具调用。
//
// 返回：
//   - []types.ToolCall: 工具调用列表
//
func (a *StreamAggregator) GetToolCalls() []types.ToolCall {
	a.mu.RLock()
	defer a.mu.RUnlock()

	toolCalls := make([]types.ToolCall, len(a.toolCalls))
	copy(toolCalls, a.toolCalls)
	return toolCalls
}

// HasError 检查是否有错误。
//
// 返回：
//   - bool: 是否有错误
//
func (a *StreamAggregator) HasError() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.hasError
}

// GetError 获取最后的错误。
//
// 返回：
//   - error: 最后的错误
//
func (a *StreamAggregator) GetError() error {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.lastError
}

// Reset 重置聚合器状态。
func (a *StreamAggregator) Reset() {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.events = make([]types.StreamEvent, 0, 100)
	a.content.Reset()
	a.toolCalls = make([]types.ToolCall, 0)
	a.metadata = make(map[string]any)
	a.eventCount = 0
	a.hasError = false
	a.lastError = nil
}

// String 实现 Stringer 接口。
func (a *StreamAggregator) String() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return fmt.Sprintf("StreamAggregator(events=%d, content_len=%d, tool_calls=%d, has_error=%v)",
		a.eventCount, a.content.Len(), len(a.toolCalls), a.hasError)
}
