package types

import (
	"encoding/json"
	"fmt"
)

// Role 消息角色类型
type Role string

const (
	// RoleSystem 系统消息角色
	RoleSystem Role = "system"
	// RoleUser 用户消息角色
	RoleUser Role = "user"
	// RoleAssistant AI 助手消息角色
	RoleAssistant Role = "assistant"
	// RoleTool 工具调用结果消息角色
	RoleTool Role = "tool"
)

// String 实现 Stringer 接口
func (r Role) String() string {
	return string(r)
}

// IsValid 检查角色是否有效
func (r Role) IsValid() bool {
	switch r {
	case RoleSystem, RoleUser, RoleAssistant, RoleTool:
		return true
	default:
		return false
	}
}

// Message 表示一条对话消息。
//
// Message 是 LLM 对话的基本单位，包含角色、内容和可选的工具调用信息。
//
// 示例：
//
//	// 创建用户消息
//	msg := types.NewUserMessage("Hello!")
//
//	// 创建带工具调用的助手消息
//	assistantMsg := types.Message{
//	    Role: types.RoleAssistant,
//	    Content: "Let me search for that.",
//	    ToolCalls: []types.ToolCall{...},
//	}
//
type Message struct {
	// Role 消息角色
	Role Role `json:"role"`

	// Content 消息内容
	Content string `json:"content"`

	// Name 可选的消息发送者名称（用于多用户场景）
	Name string `json:"name,omitempty"`

	// ToolCalls 工具调用列表（仅 RoleAssistant 使用）
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`

	// ToolCallID 工具调用 ID（仅 RoleTool 使用，对应 ToolCall.ID）
	ToolCallID string `json:"tool_call_id,omitempty"`

	// Metadata 附加元数据
	Metadata map[string]any `json:"metadata,omitempty"`
}

// ToolCall 表示一次工具调用。
//
// ToolCall 用于 LLM 请求调用外部工具（函数）。
type ToolCall struct {
	// ID 工具调用的唯一标识符
	ID string `json:"id"`

	// Type 工具类型，通常为 "function"
	Type string `json:"type"`

	// Function 函数调用详情
	Function FunctionCall `json:"function"`
}

// FunctionCall 表示函数调用的详细信息。
type FunctionCall struct {
	// Name 函数名称
	Name string `json:"name"`

	// Arguments 函数参数，JSON 格式的字符串
	Arguments string `json:"arguments"`
}

// Function 表示一个可调用的函数定义（用于 OpenAI Functions）。
//
// Function 描述了工具的接口，用于 LLM 理解如何调用工具。
type Function struct {
	// Name 函数名称
	Name string `json:"name"`

	// Description 函数描述
	Description string `json:"description"`

	// Parameters 函数参数的 JSON Schema
	Parameters Schema `json:"parameters"`
}

// NewSystemMessage 创建一条系统消息。
//
// 系统消息通常用于设置 AI 的行为和角色。
//
// 参数：
//   - content: 消息内容
//
// 返回：
//   - Message: 系统消息
//
func NewSystemMessage(content string) Message {
	return Message{
		Role:    RoleSystem,
		Content: content,
	}
}

// NewUserMessage 创建一条用户消息。
//
// 用户消息表示来自人类用户的输入。
//
// 参数：
//   - content: 消息内容
//
// 返回：
//   - Message: 用户消息
//
func NewUserMessage(content string) Message {
	return Message{
		Role:    RoleUser,
		Content: content,
	}
}

// NewAssistantMessage 创建一条助手消息。
//
// 助手消息表示来自 AI 的响应。
//
// 参数：
//   - content: 消息内容
//
// 返回：
//   - Message: 助手消息
//
func NewAssistantMessage(content string) Message {
	return Message{
		Role:    RoleAssistant,
		Content: content,
	}
}

// NewToolMessage 创建一条工具结果消息。
//
// 工具消息用于返回工具调用的结果。
//
// 参数：
//   - toolCallID: 对应的工具调用 ID
//   - content: 工具执行结果
//
// 返回：
//   - Message: 工具消息
//
func NewToolMessage(toolCallID, content string) Message {
	return Message{
		Role:       RoleTool,
		Content:    content,
		ToolCallID: toolCallID,
	}
}

// WithName 设置消息发送者名称。
//
// 返回新的 Message 实例，不修改原消息。
//
// 参数：
//   - name: 发送者名称
//
// 返回：
//   - Message: 新的消息实例
//
func (m Message) WithName(name string) Message {
	m.Name = name
	return m
}

// WithMetadata 添加元数据。
//
// 返回新的 Message 实例，不修改原消息。
//
// 参数：
//   - key: 元数据键
//   - value: 元数据值
//
// 返回：
//   - Message: 新的消息实例
//
func (m Message) WithMetadata(key string, value any) Message {
	if m.Metadata == nil {
		m.Metadata = make(map[string]any)
	}
	m.Metadata[key] = value
	return m
}

// GetToolCallArgs 解析工具调用参数。
//
// 将 Arguments 字符串解析为 map。
//
// 返回：
//   - map[string]any: 解析后的参数
//   - error: 解析错误
//
func (tc ToolCall) GetToolCallArgs() (map[string]any, error) {
	if tc.Function.Arguments == "" {
		return make(map[string]any), nil
	}

	var args map[string]any
	if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
		return nil, fmt.Errorf("failed to parse tool call arguments: %w", err)
	}

	return args, nil
}

// Validate 验证消息的有效性。
//
// 返回：
//   - error: 验证失败时返回错误
//
func (m Message) Validate() error {
	if !m.Role.IsValid() {
		return fmt.Errorf("invalid role: %s", m.Role)
	}

	// 工具消息必须有 ToolCallID
	if m.Role == RoleTool && m.ToolCallID == "" {
		return fmt.Errorf("tool message must have tool_call_id")
	}

	// 助手消息如果有 ToolCalls，每个必须有有效的 ID 和 Name
	if m.Role == RoleAssistant && len(m.ToolCalls) > 0 {
		for i, tc := range m.ToolCalls {
			if tc.ID == "" {
				return fmt.Errorf("tool_call[%d] missing id", i)
			}
			if tc.Function.Name == "" {
				return fmt.Errorf("tool_call[%d] missing function name", i)
			}
		}
	}

	return nil
}

// Clone 创建消息的深拷贝。
//
// 返回：
//   - Message: 消息副本
//
func (m Message) Clone() Message {
	clone := m

	// 深拷贝 ToolCalls
	if len(m.ToolCalls) > 0 {
		clone.ToolCalls = make([]ToolCall, len(m.ToolCalls))
		copy(clone.ToolCalls, m.ToolCalls)
	}

	// 深拷贝 Metadata
	if m.Metadata != nil {
		clone.Metadata = make(map[string]any, len(m.Metadata))
		for k, v := range m.Metadata {
			clone.Metadata[k] = v
		}
	}

	return clone
}

// String 实现 Stringer 接口，用于调试输出。
//
// 不会输出完整内容，避免日志过长。
func (m Message) String() string {
	contentPreview := m.Content
	if len(contentPreview) > 50 {
		contentPreview = contentPreview[:50] + "..."
	}

	if len(m.ToolCalls) > 0 {
		return fmt.Sprintf("Message{Role:%s, Content:%q, ToolCalls:%d}",
			m.Role, contentPreview, len(m.ToolCalls))
	}

	return fmt.Sprintf("Message{Role:%s, Content:%q}", m.Role, contentPreview)
}
