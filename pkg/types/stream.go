package types

// StreamEventType 表示流式事件的类型。
type StreamEventType string

const (
	// StreamEventStart 开始事件
	StreamEventStart StreamEventType = "start"

	// StreamEventToken 单个 token 事件
	StreamEventToken StreamEventType = "token"

	// StreamEventContent 内容块事件
	StreamEventContent StreamEventType = "content"

	// StreamEventToolCall 工具调用事件
	StreamEventToolCall StreamEventType = "tool_call"

	// StreamEventToolResult 工具结果事件
	StreamEventToolResult StreamEventType = "tool_result"

	// StreamEventEnd 结束事件
	StreamEventEnd StreamEventType = "end"

	// StreamEventError 错误事件
	StreamEventError StreamEventType = "error"
)

// StreamEvent 表示流式执行的事件。
//
// StreamEvent 用于流式处理中传递数据和状态。
//
// 示例：
//
//	event := types.StreamEvent{
//	    Type: types.StreamEventToken,
//	    Token: "Hello",
//	    Delta: "Hello",
//	}
//
type StreamEvent struct {
	// Type 事件类型
	Type StreamEventType `json:"type"`

	// Data 事件数据（通用字段）
	Data any `json:"data,omitempty"`

	// Token 单个 token 内容（用于 token 级流式）
	Token string `json:"token,omitempty"`

	// Delta 增量内容（用于累积显示）
	Delta string `json:"delta,omitempty"`

	// Content 完整内容（用于内容块事件）
	Content string `json:"content,omitempty"`

	// ToolCall 工具调用信息
	ToolCall *ToolCall `json:"tool_call,omitempty"`

	// ToolResult 工具执行结果
	ToolResult string `json:"tool_result,omitempty"`

	// Error 错误信息（Type 为 error 时有值）
	Error error `json:"error,omitempty"`

	// Metadata 元数据
	Metadata map[string]any `json:"metadata,omitempty"`

	// Index 事件索引（用于排序）
	Index int `json:"index,omitempty"`

	// Done 是否完成（用于标记流结束）
	Done bool `json:"done,omitempty"`
}

// NewStreamEvent 创建流式事件。
//
// 参数：
//   - eventType: 事件类型
//   - data: 事件数据
//
// 返回：
//   - StreamEvent: 流式事件
//
func NewStreamEvent(eventType StreamEventType, data any) StreamEvent {
	return StreamEvent{
		Type: eventType,
		Data: data,
	}
}

// NewTokenEvent 创建 token 事件。
//
// 参数：
//   - token: token 内容
//
// 返回：
//   - StreamEvent: token 事件
//
func NewTokenEvent(token string) StreamEvent {
	return StreamEvent{
		Type:  StreamEventToken,
		Token: token,
		Delta: token,
	}
}

// NewToolCallEvent 创建工具调用事件。
//
// 参数：
//   - toolCall: 工具调用信息
//
// 返回：
//   - StreamEvent: 工具调用事件
//
func NewToolCallEvent(toolCall *ToolCall) StreamEvent {
	return StreamEvent{
		Type:     StreamEventToolCall,
		ToolCall: toolCall,
	}
}

// NewErrorEvent 创建错误事件。
//
// 参数：
//   - err: 错误
//
// 返回：
//   - StreamEvent: 错误事件
//
func NewErrorEvent(err error) StreamEvent {
	return StreamEvent{
		Type:  StreamEventError,
		Error: err,
	}
}

// WithMetadata 添加元数据。
//
// 参数：
//   - key: 元数据键
//   - value: 元数据值
//
// 返回：
//   - StreamEvent: 返回自身，支持链式调用
//
func (e StreamEvent) WithMetadata(key string, value any) StreamEvent {
	if e.Metadata == nil {
		e.Metadata = make(map[string]any)
	}
	e.Metadata[key] = value
	return e
}

// WithIndex 设置事件索引。
//
// 参数：
//   - index: 索引值
//
// 返回：
//   - StreamEvent: 返回自身，支持链式调用
//
func (e StreamEvent) WithIndex(index int) StreamEvent {
	e.Index = index
	return e
}

// IsError 检查是否为错误事件。
//
// 返回：
//   - bool: 是否为错误事件
//
func (e StreamEvent) IsError() bool {
	return e.Type == StreamEventError || e.Error != nil
}

// IsEnd 检查是否为结束事件。
//
// 返回：
//   - bool: 是否为结束事件
//
func (e StreamEvent) IsEnd() bool {
	return e.Type == StreamEventEnd || e.Done
}

// IsToken 检查是否为 token 事件。
//
// 返回：
//   - bool: 是否为 token 事件
//
func (e StreamEvent) IsToken() bool {
	return e.Type == StreamEventToken
}

// IsToolCall 检查是否为工具调用事件。
//
// 返回：
//   - bool: 是否为工具调用事件
//
func (e StreamEvent) IsToolCall() bool {
	return e.Type == StreamEventToolCall
}
