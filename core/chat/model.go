package chat

import (
	"fmt"

	"langchain-go/core/runnable"
	"langchain-go/pkg/types"
)

// ChatModel 是聊天模型的核心接口。
//
// ChatModel 继承了 Runnable 接口，输入为消息列表，输出为单条消息。
// 所有 LLM 提供商（OpenAI、Anthropic 等）都应实现此接口。
//
// 类型约束：
//   - 输入：[]types.Message（消息列表）
//   - 输出：types.Message（单条响应消息）
//
// 核心功能：
//   - 基本对话：通过 Invoke 方法进行单次调用
//   - 批量调用：通过 Batch 方法并行处理多组对话
//   - 流式输出：通过 Stream 方法逐步返回响应
//   - 工具调用：通过 BindTools 绑定工具，支持 Function Calling
//   - 结构化输出：通过 WithStructuredOutput 强制返回特定格式
//
// 实现要求：
//   - 必须正确处理各种角色的消息（system、user、assistant、tool）
//   - 必须支持工具调用的完整流程（请求工具 -> 执行 -> 返回结果）
//   - 必须正确处理错误和超时
//   - 建议支持流式输出以提升用户体验
//
type ChatModel interface {
	// 继承 Runnable 接口
	runnable.Runnable[[]types.Message, types.Message]

	// BindTools 绑定工具到模型。
	//
	// BindTools 返回一个新的 ChatModel 实例，该实例会在调用时
	// 告知 LLM 可用的工具列表。LLM 可以选择调用这些工具。
	//
	// 参数：
	//   - tools: 工具列表
	//
	// 返回：
	//   - ChatModel: 绑定了工具的新模型实例
	//
	// 示例：
	//
	//	tool := types.Tool{
	//	    Name:        "calculator",
	//	    Description: "Calculate math expressions",
	//	    Parameters:  calcSchema,
	//	}
	//	modelWithTools := model.BindTools([]types.Tool{tool})
	//
	BindTools(tools []types.Tool) ChatModel

	// WithStructuredOutput 配置模型返回结构化输出。
	//
	// WithStructuredOutput 强制模型按照指定的 JSON Schema
	// 返回结构化数据，而不是自由格式的文本。
	//
	// 参数：
	//   - schema: JSON Schema 定义
	//
	// 返回：
	//   - ChatModel: 配置了结构化输出的新模型实例
	//
	// 注意：
	//   - 不是所有提供商都支持结构化输出
	//   - OpenAI 支持 JSON Mode 和 Strict Mode
	//   - 其他提供商可能通过提示词模拟
	//
	WithStructuredOutput(schema types.Schema) ChatModel

	// GetModelName 获取模型名称。
	//
	// 返回：
	//   - string: 模型名称（如 "gpt-4"、"claude-3-opus-20240229"）
	//
	GetModelName() string

	// GetProvider 获取提供商名称。
	//
	// 返回：
	//   - string: 提供商名称（如 "openai"、"anthropic"）
	//
	GetProvider() string
}

// BaseChatModel 提供 ChatModel 的基础实现。
//
// BaseChatModel 包含所有 ChatModel 共用的字段和方法，
// 具体的提供商只需嵌入此结构体并实现核心的 Invoke 和 Stream 方法。
//
// 字段：
//   - modelName: 模型名称
//   - provider: 提供商名称
//   - boundTools: 绑定的工具列表
//   - outputSchema: 结构化输出的 Schema
//   - config: 运行时配置
//
type BaseChatModel struct {
	modelName    string
	provider     string
	boundTools   []types.Tool
	outputSchema *types.Schema
	config       *types.Config
}

// NewBaseChatModel 创建基础 ChatModel。
//
// 参数：
//   - modelName: 模型名称
//   - provider: 提供商名称
//
// 返回：
//   - *BaseChatModel: 基础模型实例
//
func NewBaseChatModel(modelName, provider string) *BaseChatModel {
	return &BaseChatModel{
		modelName:    modelName,
		provider:     provider,
		boundTools:   make([]types.Tool, 0),
		outputSchema: nil,
		config:       types.NewConfig(),
	}
}

// GetModelName 实现 ChatModel 接口。
func (b *BaseChatModel) GetModelName() string {
	return b.modelName
}

// GetProvider 实现 ChatModel 接口。
func (b *BaseChatModel) GetProvider() string {
	return b.provider
}

// GetBoundTools 获取绑定的工具列表。
//
// 返回：
//   - []types.Tool: 工具列表
//
func (b *BaseChatModel) GetBoundTools() []types.Tool {
	return b.boundTools
}

// GetOutputSchema 获取结构化输出的 Schema。
//
// 返回：
//   - *types.Schema: Schema（如果未配置则返回 nil）
//
func (b *BaseChatModel) GetOutputSchema() *types.Schema {
	return b.outputSchema
}

// GetConfig 获取配置。
//
// 返回：
//   - *types.Config: 配置
//
func (b *BaseChatModel) GetConfig() *types.Config {
	return b.config
}

// SetBoundTools 设置绑定的工具。
//
// 此方法用于子类实现 BindTools。
//
// 参数：
//   - tools: 工具列表
//
func (b *BaseChatModel) SetBoundTools(tools []types.Tool) {
	b.boundTools = tools
}

// SetOutputSchema 设置输出 Schema。
//
// 此方法用于子类实现 WithStructuredOutput。
//
// 参数：
//   - schema: JSON Schema
//
func (b *BaseChatModel) SetOutputSchema(schema types.Schema) {
	b.outputSchema = &schema
}

// SetConfig 设置配置。
//
// 参数：
//   - config: 配置
//
func (b *BaseChatModel) SetConfig(config *types.Config) {
	b.config = config
}

// GetName 实现 Runnable 接口。
func (b *BaseChatModel) GetName() string {
	return fmt.Sprintf("%s/%s", b.provider, b.modelName)
}

// Note: BaseChatModel 不实现 Runnable 接口的方法（Invoke, Batch, Stream, WithConfig 等）。
// 这些方法必须由具体的 ChatModel 实现类提供。
// BaseChatModel 只提供共用的字段和辅助方法。

// ValidateMessages 验证消息列表的有效性。
//
// 参数：
//   - messages: 消息列表
//
// 返回：
//   - error: 验证失败时返回错误
//
func ValidateMessages(messages []types.Message) error {
	if len(messages) == 0 {
		return fmt.Errorf("messages cannot be empty")
	}

	for i, msg := range messages {
		if err := msg.Validate(); err != nil {
			return fmt.Errorf("invalid message at index %d: %w", i, err)
		}
	}

	return nil
}

// ConvertToolsToOpenAI 将工具列表转换为 OpenAI 格式。
//
// 参数：
//   - tools: 工具列表
//
// 返回：
//   - []map[string]any: OpenAI 格式的工具列表
//
func ConvertToolsToOpenAI(tools []types.Tool) []map[string]any {
	result := make([]map[string]any, len(tools))
	for i, tool := range tools {
		result[i] = tool.ToOpenAITool()
	}
	return result
}

// ConvertToolsToAnthropic 将工具列表转换为 Anthropic 格式。
//
// 参数：
//   - tools: 工具列表
//
// 返回：
//   - []map[string]any: Anthropic 格式的工具列表
//
func ConvertToolsToAnthropic(tools []types.Tool) []map[string]any {
	result := make([]map[string]any, len(tools))
	for i, tool := range tools {
		result[i] = tool.ToAnthropicTool()
	}
	return result
}
