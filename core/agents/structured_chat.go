package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	
	"langchain-go/core/memory"
	"langchain-go/pkg/types"
)

// StructuredChatAgent 是结构化对话 Agent。
//
// 与 ConversationalAgent 不同，StructuredChatAgent 支持：
// 1. 结构化的输入/输出格式
// 2. 对话记忆管理
// 3. 工具调用能力
// 4. 更复杂的对话流程控制
//
type StructuredChatAgent struct {
	*BaseAgent
	systemPrompt    string
	memory          memory.Memory
	outputFormat    string
	conversationID  string
}

// StructuredChatConfig 是 Structured Chat Agent 配置。
type StructuredChatConfig struct {
	AgentConfig
	
	// Memory 对话记忆
	Memory memory.Memory
	
	// OutputFormat 输出格式 (json, markdown, plain)
	OutputFormat string
	
	// ConversationID 对话 ID
	ConversationID string
	
	// EnableToolCalling 是否启用工具调用
	EnableToolCalling bool
}

// NewStructuredChatAgent 创建 Structured Chat Agent。
func NewStructuredChatAgent(config StructuredChatConfig) *StructuredChatAgent {
	baseAgent := NewBaseAgent(config.AgentConfig)
	
	systemPrompt := config.SystemPrompt
	if systemPrompt == "" {
		systemPrompt = getDefaultStructuredChatPrompt()
	}
	
	outputFormat := config.OutputFormat
	if outputFormat == "" {
		outputFormat = "plain"
	}
	
	conversationID := config.ConversationID
	if conversationID == "" {
		conversationID = "default"
	}

	return &StructuredChatAgent{
		BaseAgent:      baseAgent,
		systemPrompt:   systemPrompt,
		memory:         config.Memory,
		outputFormat:   outputFormat,
		conversationID: conversationID,
	}
}

// Plan 实现 Agent 接口。
func (sca *StructuredChatAgent) Plan(ctx context.Context, input string, history []AgentStep) (*AgentAction, error) {
	// 构建消息列表
	messages := []types.Message{
		types.NewSystemMessage(sca.systemPrompt),
	}

	// 从记忆中加载历史对话
	if sca.memory != nil {
		memoryVars, err := sca.memory.LoadMemory(ctx, map[string]any{
			"conversation_id": sca.conversationID,
		})
		if err == nil {
			if chatHistory, ok := memoryVars["history"].(string); ok && chatHistory != "" {
				messages = append(messages, types.NewSystemMessage(fmt.Sprintf("Previous conversation:\n%s", chatHistory)))
			}
		}
	}

	// 添加当前步骤的历史
	for _, step := range history {
		if step.Action.Type == ActionToolCall {
			messages = append(messages, types.NewAssistantMessage(step.Action.Log))
			messages = append(messages, types.NewToolMessage(
				step.Action.Tool,
				step.Observation,
			))
		}
	}

	// 添加当前输入
	messages = append(messages, types.NewUserMessage(input))

	// 决定是否需要工具调用
	needsToolCall := sca.needsToolCall(input, history)

	var response *types.Message
	var err error

	if needsToolCall && len(sca.tools) > 0 {
		// 使用工具调用
		modelWithTools := sca.llm.BindTools(sca.ConvertToolsToTypesTools())
		response, err = modelWithTools.Invoke(ctx, messages)
	} else {
		// 直接对话
		response, err = sca.llm.Invoke(ctx, messages)
	}

	if err != nil {
		return nil, fmt.Errorf("structured chat agent: invoke failed: %w", err)
	}

	// 检查是否有工具调用
	if len(response.ToolCalls) > 0 {
		toolCall := response.ToolCalls[0]
		return &AgentAction{
			Type: ActionToolCall,
			Tool: toolCall.Function.Name,
			ToolInput: map[string]any{
				"input": toolCall.Function.Arguments,
			},
			Log: response.Content,
		}, nil
	}

	// 格式化输出
	formattedOutput := sca.formatOutput(response.Content)

	// 保存到记忆
	if sca.memory != nil {
		_ = sca.memory.SaveContext(ctx, map[string]any{
			"conversation_id": sca.conversationID,
			"input":          input,
			"output":         formattedOutput,
		})
	}

	return &AgentAction{
		Type:        ActionFinish,
		FinalAnswer: formattedOutput,
		Log:         response.Content,
	}, nil
}

// needsToolCall 判断是否需要工具调用。
func (sca *StructuredChatAgent) needsToolCall(input string, history []AgentStep) bool {
	if len(sca.tools) == 0 {
		return false
	}

	// 简单的启发式规则
	keywords := []string{
		"search", "calculate", "find", "lookup", "query",
		"搜索", "计算", "查找", "查询",
	}

	inputLower := strings.ToLower(input)
	for _, keyword := range keywords {
		if strings.Contains(inputLower, keyword) {
			return true
		}
	}

	return false
}

// formatOutput 格式化输出。
func (sca *StructuredChatAgent) formatOutput(content string) string {
	switch sca.outputFormat {
	case "json":
		return sca.formatAsJSON(content)
	case "markdown":
		return sca.formatAsMarkdown(content)
	default:
		return content
	}
}

// formatAsJSON 格式化为 JSON。
func (sca *StructuredChatAgent) formatAsJSON(content string) string {
	output := map[string]any{
		"response":        content,
		"conversation_id": sca.conversationID,
		"format":          "json",
	}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return content
	}

	return string(data)
}

// formatAsMarkdown 格式化为 Markdown。
func (sca *StructuredChatAgent) formatAsMarkdown(content string) string {
	var builder strings.Builder

	builder.WriteString("## Response\n\n")
	builder.WriteString(content)
	builder.WriteString("\n\n")
	builder.WriteString(fmt.Sprintf("*Conversation ID: %s*\n", sca.conversationID))

	return builder.String()
}

// ClearMemory 清除对话记忆。
func (sca *StructuredChatAgent) ClearMemory(ctx context.Context) error {
	if sca.memory != nil {
		return sca.memory.Clear(ctx)
	}
	return nil
}

// getDefaultStructuredChatPrompt 返回默认的 Structured Chat 提示词。
func getDefaultStructuredChatPrompt() string {
	return `You are a helpful assistant engaged in a structured conversation.

Guidelines:
1. Maintain context from previous messages in the conversation
2. Use available tools when appropriate to provide accurate information
3. Provide clear, well-structured responses
4. If you don't know something, admit it and suggest using tools to find the answer
5. Be conversational but professional

You have access to tools that can help you answer questions more accurately.
Use them when needed.`
}

// CreateStructuredChatAgent 创建 Structured Chat Agent (简化工厂函数)。
//
// Structured Chat Agent 支持结构化的对话，带有记忆管理和工具调用能力。
// 适合需要维护对话上下文的多轮对话场景。
//
// 参数：
//   - llm: 语言模型
//   - agentTools: 工具列表（可选）
//   - opts: 可选配置
//
// 返回：
//   - Agent: Structured Chat Agent 实例
//
// 示例：
//
//	// 创建带记忆的 Agent
//	mem := memory.NewBufferMemory(10)
//	agent := agents.CreateStructuredChatAgent(llm, tools,
//	    agents.WithStructuredChatMemory(mem),
//	    agents.WithStructuredChatOutputFormat("json"),
//	    agents.WithStructuredChatConversationID("user-123"),
//	)
//
func CreateStructuredChatAgent(llm chat.ChatModel, agentTools []tools.Tool, opts ...StructuredChatOption) Agent {
	config := StructuredChatConfig{
		AgentConfig: AgentConfig{
			Type:         "structured_chat",
			LLM:          llm,
			Tools:        agentTools,
			MaxSteps:     10,
			SystemPrompt: getDefaultStructuredChatPrompt(),
			Verbose:      false,
			Extra:        make(map[string]any),
		},
		Memory:            nil,
		OutputFormat:      "plain",
		ConversationID:    "default",
		EnableToolCalling: true,
	}

	// 应用选项
	for _, opt := range opts {
		opt(&config)
	}

	return NewStructuredChatAgent(config)
}

// StructuredChatOption 是 Structured Chat Agent 配置选项。
type StructuredChatOption func(*StructuredChatConfig)

// WithStructuredChatMemory 设置对话记忆。
func WithStructuredChatMemory(mem memory.Memory) StructuredChatOption {
	return func(config *StructuredChatConfig) {
		config.Memory = mem
	}
}

// WithStructuredChatOutputFormat 设置输出格式。
func WithStructuredChatOutputFormat(format string) StructuredChatOption {
	return func(config *StructuredChatConfig) {
		config.OutputFormat = format
	}
}

// WithStructuredChatConversationID 设置对话 ID。
func WithStructuredChatConversationID(id string) StructuredChatOption {
	return func(config *StructuredChatConfig) {
		config.ConversationID = id
	}
}

// WithStructuredChatSystemPrompt 设置系统提示词。
func WithStructuredChatSystemPrompt(prompt string) StructuredChatOption {
	return func(config *StructuredChatConfig) {
		config.SystemPrompt = prompt
	}
}

// WithStructuredChatVerbose 设置是否输出详细日志。
func WithStructuredChatVerbose(verbose bool) StructuredChatOption {
	return func(config *StructuredChatConfig) {
		config.Verbose = verbose
	}
}

// WithStructuredChatMaxSteps 设置最大步数。
func WithStructuredChatMaxSteps(maxSteps int) StructuredChatOption {
	return func(config *StructuredChatConfig) {
		config.MaxSteps = maxSteps
	}
}

// WithStructuredChatToolCalling 设置是否启用工具调用。
func WithStructuredChatToolCalling(enable bool) StructuredChatOption {
	return func(config *StructuredChatConfig) {
		config.EnableToolCalling = enable
	}
}
