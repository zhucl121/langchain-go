package memory

import (
	"context"

	"github.com/zhucl121/langchain-go/pkg/types"
)

// Memory 是记忆系统的核心接口。
//
// Memory 管理对话历史和上下文信息，让 AI Agent 能够：
//   - 记住之前的对话
//   - 保持上下文连贯性
//   - 提供相关的历史信息
//
// 所有 Memory 实现都必须实现此接口。
//
// 使用场景：
//   - 聊天机器人（记住对话历史）
//   - 问答系统（记住之前的问题和答案）
//   - Agent（记住之前的操作和结果）
//
type Memory interface {
	// LoadMemoryVariables 加载记忆变量。
	//
	// 返回一个 map，包含记忆相关的变量（通常是 "history"）。
	//
	// 参数：
	//   - ctx: 上下文
	//   - inputs: 输入变量（可用于过滤或定制记忆）
	//
	// 返回：
	//   - map[string]any: 记忆变量（如 {"history": []types.Message}）
	//   - error: 加载错误
	//
	LoadMemoryVariables(ctx context.Context, inputs map[string]any) (map[string]any, error)

	// SaveContext 保存对话上下文。
	//
	// 将新的对话轮次保存到记忆中。
	//
	// 参数：
	//   - ctx: 上下文
	//   - inputs: 输入变量（通常包含 "input" 或自定义键）
	//   - outputs: 输出变量（通常包含 "output" 或自定义键）
	//
	// 返回：
	//   - error: 保存错误
	//
	SaveContext(ctx context.Context, inputs map[string]any, outputs map[string]any) error

	// Clear 清空记忆。
	//
	// 删除所有保存的对话历史。
	//
	// 参数：
	//   - ctx: 上下文
	//
	// 返回：
	//   - error: 清空错误
	//
	Clear(ctx context.Context) error
}

// BaseMemory 提供 Memory 的基础实现。
//
// BaseMemory 包含通用的字段和方法，具体的 Memory 类型可以嵌入此类型。
type BaseMemory struct {
	// inputKey 是输入消息的键名（默认 "input"）
	inputKey string

	// outputKey 是输出消息的键名（默认 "output"）
	outputKey string

	// memoryKey 是返回记忆变量的键名（默认 "history"）
	memoryKey string

	// returnMessages 是否返回 Message 列表（true）还是字符串（false）
	returnMessages bool
}

// NewBaseMemory 创建基础记忆实例。
//
// 返回：
//   - *BaseMemory: 基础记忆实例
//
func NewBaseMemory() *BaseMemory {
	return &BaseMemory{
		inputKey:       "input",
		outputKey:      "output",
		memoryKey:      "history",
		returnMessages: true,
	}
}

// GetInputKey 获取输入键名。
func (b *BaseMemory) GetInputKey() string {
	return b.inputKey
}

// SetInputKey 设置输入键名。
func (b *BaseMemory) SetInputKey(key string) {
	b.inputKey = key
}

// GetOutputKey 获取输出键名。
func (b *BaseMemory) GetOutputKey() string {
	return b.outputKey
}

// SetOutputKey 设置输出键名。
func (b *BaseMemory) SetOutputKey(key string) {
	b.outputKey = key
}

// GetMemoryKey 获取记忆键名。
func (b *BaseMemory) GetMemoryKey() string {
	return b.memoryKey
}

// SetMemoryKey 设置记忆键名。
func (b *BaseMemory) SetMemoryKey(key string) {
	b.memoryKey = key
}

// GetReturnMessages 获取是否返回消息列表。
func (b *BaseMemory) GetReturnMessages() bool {
	return b.returnMessages
}

// SetReturnMessages 设置是否返回消息列表。
func (b *BaseMemory) SetReturnMessages(returnMessages bool) {
	b.returnMessages = returnMessages
}

// extractInputOutput 从 inputs 和 outputs 中提取输入和输出内容。
//
// 参数：
//   - inputs: 输入变量
//   - outputs: 输出变量
//
// 返回：
//   - string: 输入内容
//   - string: 输出内容
//
func (b *BaseMemory) extractInputOutput(inputs map[string]any, outputs map[string]any) (string, string) {
	var inputStr, outputStr string

	if inputs != nil {
		if val, ok := inputs[b.inputKey]; ok {
			if str, ok := val.(string); ok {
				inputStr = str
			}
		}
	}

	if outputs != nil {
		if val, ok := outputs[b.outputKey]; ok {
			if str, ok := val.(string); ok {
				outputStr = str
			}
		}
	}

	return inputStr, outputStr
}

// messagesToString 将消息列表转换为字符串格式。
//
// 格式：Human: xxx\nAI: yyy\n...
//
// 参数：
//   - messages: 消息列表
//
// 返回：
//   - string: 字符串格式的对话历史
//
func messagesToString(messages []types.Message) string {
	var result string

	for _, msg := range messages {
		var prefix string
		switch msg.Role {
		case types.RoleUser:
			prefix = "Human"
		case types.RoleAssistant:
			prefix = "AI"
		case types.RoleSystem:
			prefix = "System"
		case types.RoleTool:
			prefix = "Tool"
		default:
			prefix = string(msg.Role)
		}

		if result != "" {
			result += "\n"
		}
		result += prefix + ": " + msg.Content
	}

	return result
}
