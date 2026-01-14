package types

import (
	"encoding/json"
	"fmt"
)

// Tool 表示一个可供 AI 调用的工具（函数）。
//
// Tool 定义了工具的名称、描述和参数 Schema，供 LLM 理解如何使用。
//
// 示例：
//
//	tool := types.Tool{
//	    Name:        "search",
//	    Description: "Search the internet for information",
//	    Parameters: types.Schema{
//	        Type: "object",
//	        Properties: map[string]types.Schema{
//	            "query": {
//	                Type:        "string",
//	                Description: "Search query",
//	            },
//	        },
//	        Required: []string{"query"},
//	    },
//	}
//
type Tool struct {
	// Name 工具名称，必须唯一
	Name string `json:"name"`

	// Description 工具描述，告诉 LLM 该工具的用途
	Description string `json:"description"`

	// Parameters 工具参数的 JSON Schema
	Parameters Schema `json:"parameters"`

	// Strict 是否严格模式（OpenAI 专用）
	// 严格模式会强制 LLM 生成符合 Schema 的参数
	Strict bool `json:"strict,omitempty"`
}

// Validate 验证工具定义的有效性。
//
// 返回：
//   - error: 验证失败时返回错误
//
func (t Tool) Validate() error {
	if t.Name == "" {
		return fmt.Errorf("tool name is required")
	}

	if t.Description == "" {
		return fmt.Errorf("tool description is required")
	}

	if err := t.Parameters.Validate(); err != nil {
		return fmt.Errorf("invalid parameters schema: %w", err)
	}

	return nil
}

// ToOpenAITool 转换为 OpenAI 工具格式。
//
// 返回：
//   - map[string]any: OpenAI 格式的工具定义
//
func (t Tool) ToOpenAITool() map[string]any {
	tool := map[string]any{
		"type": "function",
		"function": map[string]any{
			"name":        t.Name,
			"description": t.Description,
			"parameters":  t.Parameters.ToMap(),
		},
	}

	if t.Strict {
		tool["function"].(map[string]any)["strict"] = true
	}

	return tool
}

// ToAnthropicTool 转换为 Anthropic 工具格式。
//
// 返回：
//   - map[string]any: Anthropic 格式的工具定义
//
func (t Tool) ToAnthropicTool() map[string]any {
	return map[string]any{
		"name":         t.Name,
		"description":  t.Description,
		"input_schema": t.Parameters.ToMap(),
	}
}

// Clone 创建工具的深拷贝。
//
// 返回：
//   - Tool: 工具副本
//
func (t Tool) Clone() Tool {
	clone := t
	clone.Parameters = t.Parameters.Clone()
	return clone
}

// String 实现 Stringer 接口，用于调试输出。
func (t Tool) String() string {
	return fmt.Sprintf("Tool{Name:%s, Description:%s}", t.Name, t.Description)
}

// ToolResult 表示工具执行的结果。
type ToolResult struct {
	// ToolCallID 对应的工具调用 ID
	ToolCallID string `json:"tool_call_id"`

	// ToolName 工具名称
	ToolName string `json:"tool_name"`

	// Output 工具输出，可以是任意类型
	Output any `json:"output"`

	// Error 执行错误（如果有）
	Error string `json:"error,omitempty"`

	// IsError 是否是错误结果
	IsError bool `json:"is_error"`
}

// NewToolResult 创建成功的工具结果。
//
// 参数：
//   - toolCallID: 工具调用 ID
//   - toolName: 工具名称
//   - output: 工具输出
//
// 返回：
//   - ToolResult: 工具结果
//
func NewToolResult(toolCallID, toolName string, output any) ToolResult {
	return ToolResult{
		ToolCallID: toolCallID,
		ToolName:   toolName,
		Output:     output,
		IsError:    false,
	}
}

// NewToolErrorResult 创建错误的工具结果。
//
// 参数：
//   - toolCallID: 工具调用 ID
//   - toolName: 工具名称
//   - err: 错误信息
//
// 返回：
//   - ToolResult: 工具错误结果
//
func NewToolErrorResult(toolCallID, toolName string, err error) ToolResult {
	return ToolResult{
		ToolCallID: toolCallID,
		ToolName:   toolName,
		Error:      err.Error(),
		IsError:    true,
	}
}

// ToMessage 将工具结果转换为消息。
//
// 返回：
//   - Message: 工具消息
//
func (tr ToolResult) ToMessage() Message {
	var content string
	if tr.IsError {
		content = fmt.Sprintf("Error: %s", tr.Error)
	} else {
		// 尝试将输出序列化为 JSON
		if data, err := json.Marshal(tr.Output); err == nil {
			content = string(data)
		} else {
			content = fmt.Sprintf("%v", tr.Output)
		}
	}

	return NewToolMessage(tr.ToolCallID, content)
}

// String 实现 Stringer 接口。
func (tr ToolResult) String() string {
	if tr.IsError {
		return fmt.Sprintf("ToolResult{Tool:%s, Error:%s}", tr.ToolName, tr.Error)
	}
	return fmt.Sprintf("ToolResult{Tool:%s, Success}", tr.ToolName)
}
