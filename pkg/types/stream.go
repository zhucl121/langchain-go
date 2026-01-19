package types

// StreamEvent 表示流式执行的事件。
//
// StreamEvent 用于流式处理中传递数据和状态。
//
// 示例：
//
//	event := types.StreamEvent{
//	    Type: "data",
//	    Data: "some data",
//	}
//
type StreamEvent struct {
	// Type 事件类型（"start", "data", "end", "error"）
	Type string `json:"type"`

	// Data 事件数据
	Data any `json:"data,omitempty"`

	// Error 错误信息（Type 为 "error" 时有值）
	Error error `json:"error,omitempty"`

	// Metadata 元数据
	Metadata map[string]any `json:"metadata,omitempty"`
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
func NewStreamEvent(eventType string, data any) StreamEvent {
	return StreamEvent{
		Type: eventType,
		Data: data,
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

// IsError 检查是否为错误事件。
//
// 返回：
//   - bool: 是否为错误事件
//
func (e StreamEvent) IsError() bool {
	return e.Type == "error" || e.Error != nil
}

// IsEnd 检查是否为结束事件。
//
// 返回：
//   - bool: 是否为结束事件
//
func (e StreamEvent) IsEnd() bool {
	return e.Type == "end"
}
