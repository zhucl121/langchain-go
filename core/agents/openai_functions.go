package agents

import (
	"context"
	"encoding/json"
	"fmt"
	
	"github.com/zhucl121/langchain-go/core/chat"
	"github.com/zhucl121/langchain-go/core/tools"
	"github.com/zhucl121/langchain-go/pkg/types"
)

// OpenAIFunctionsAgent 是专门针对 OpenAI Functions API 优化的 Agent。
//
// 与 ToolCallingAgent 的区别:
//   - OpenAIFunctionsAgent 使用 OpenAI 的 function_call API
//   - 更好的性能和可靠性
//   - 支持强制函数调用
//
type OpenAIFunctionsAgent struct {
	*BaseAgent
	config        AgentConfig
	forceFunction string // 强制调用的函数名称（可选）
}

// OpenAIFunctionsConfig 是 OpenAI Functions Agent 配置。
type OpenAIFunctionsConfig struct {
	// LLM 语言模型（需要是 OpenAI 模型）
	LLM chat.ChatModel
	
	// Tools 工具列表
	Tools []tools.Tool
	
	// SystemPrompt 系统提示词
	SystemPrompt string
	
	// ForceFunction 强制调用的函数（可选，用于单轮确定性调用）
	ForceFunction string
	
	// MaxSteps 最大步数
	MaxSteps int
	
	// Verbose 是否输出详细日志
	Verbose bool
}

// NewOpenAIFunctionsAgent 创建 OpenAI Functions Agent。
//
// 参数：
//   - config: Agent 配置
//
// 返回：
//   - *OpenAIFunctionsAgent: Agent 实例
//
func NewOpenAIFunctionsAgent(config OpenAIFunctionsConfig) *OpenAIFunctionsAgent {
	if config.MaxSteps <= 0 {
		config.MaxSteps = 10
	}
	
	if config.SystemPrompt == "" {
		config.SystemPrompt = "You are a helpful assistant with access to functions. Use them when appropriate."
	}
	
	baseConfig := AgentConfig{
		Type:         "openai_functions",
		LLM:          config.LLM,
		Tools:        config.Tools,
		MaxSteps:     config.MaxSteps,
		SystemPrompt: config.SystemPrompt,
		Verbose:      config.Verbose,
	}
	
	return &OpenAIFunctionsAgent{
		BaseAgent:     NewBaseAgent(baseConfig),
		config:        baseConfig,
		forceFunction: config.ForceFunction,
	}
}

// Plan 实现 Agent 接口。
func (ofa *OpenAIFunctionsAgent) Plan(ctx context.Context, input string, history []AgentStep) (*AgentAction, error) {
	// 构建消息
	messages := ofa.buildMessages(input, history)
	
	// 转换工具为 OpenAI Functions 格式
	functions := ofa.convertToolsToFunctions()
	
	// 调用 LLM（带 functions 参数）
	// 注意：这里假设 ChatModel 支持 functions 调用
	// 实际实现可能需要使用特定的 OpenAI client
	
	response, err := ofa.invokeLLMWithFunctions(ctx, messages, functions)
	if err != nil {
		return nil, fmt.Errorf("openai functions agent: failed to invoke LLM: %w", err)
	}
	
	// 解析响应
	action, err := ofa.parseResponse(response)
	if err != nil {
		return nil, fmt.Errorf("openai functions agent: failed to parse response: %w", err)
	}
	
	return action, nil
}

// buildMessages 构建消息列表。
func (ofa *OpenAIFunctionsAgent) buildMessages(input string, history []AgentStep) []types.Message {
	messages := []types.Message{
		types.NewSystemMessage(ofa.config.SystemPrompt),
	}
	
	// 添加历史
	for i, step := range history {
		if step.Action != nil && step.Action.Type == ActionToolCall {
			// 将 ToolInput 转换为 JSON 字符串
			toolInputJSON, err := json.Marshal(step.Action.ToolInput)
			if err != nil {
				toolInputJSON = []byte("{}")
			}
			
			// 添加 function call 消息
			messages = append(messages, types.Message{
				Role:    types.RoleAssistant,
				Content: "",
				ToolCalls: []types.ToolCall{
					{
						ID:   fmt.Sprintf("call_%d", i),
						Type: "function",
						Function: types.FunctionCall{
							Name:      step.Action.Tool,
							Arguments: string(toolInputJSON),
						},
					},
				},
			})
			
			// 添加 function response
			messages = append(messages, types.NewToolMessage(
				fmt.Sprintf("call_%d", i),
				step.Observation,
			))
		}
	}
	
	// 添加当前输入
	if len(history) == 0 {
		messages = append(messages, types.NewUserMessage(input))
	}
	
	return messages
}

// convertToolsToFunctions 转换工具为 OpenAI Functions 格式。
func (ofa *OpenAIFunctionsAgent) convertToolsToFunctions() []types.Function {
	functions := make([]types.Function, len(ofa.tools))
	
	for i, tool := range ofa.tools {
		functions[i] = types.Function{
			Name:        tool.GetName(),
			Description: tool.GetDescription(),
			Parameters:  tool.GetParameters(),
		}
	}
	
	return functions
}

// invokeLLMWithFunctions 调用 LLM（带 functions）。
func (ofa *OpenAIFunctionsAgent) invokeLLMWithFunctions(
	ctx context.Context,
	messages []types.Message,
	functions []types.Function,
) (*types.Message, error) {
	// 构建调用选项
	options := map[string]any{
		"functions": functions,
	}
	
	// 如果强制调用某个函数
	if ofa.forceFunction != "" {
		options["function_call"] = map[string]any{
			"name": ofa.forceFunction,
		}
	}
	
	// 调用 LLM
	// 注意：这里需要模型支持 functions 调用
	response, err := ofa.llm.Invoke(ctx, messages)
	if err != nil {
		return nil, err
	}
	
	return &response, nil
}

// parseResponse 解析 LLM 响应。
func (ofa *OpenAIFunctionsAgent) parseResponse(response *types.Message) (*AgentAction, error) {
	// 检查是否有 tool calls
	if len(response.ToolCalls) > 0 {
		// 使用第一个 tool call
		toolCall := response.ToolCalls[0]
		
		// 解析参数
		var toolInput map[string]any
		if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &toolInput); err != nil {
			return nil, fmt.Errorf("failed to parse function arguments: %w", err)
		}
		
		return &AgentAction{
			Type:      ActionToolCall,
			Tool:      toolCall.Function.Name,
			ToolInput: toolInput,
			Log:       fmt.Sprintf("Calling function: %s with arguments: %s", toolCall.Function.Name, toolCall.Function.Arguments),
		}, nil
	}
	
	// 如果没有 tool call，说明任务完成
	return &AgentAction{
		Type:        ActionFinish,
		FinalAnswer: response.Content,
		Log:         "Task completed without function call",
	}, nil
}

// GetType 返回 Agent 类型。
func (ofa *OpenAIFunctionsAgent) GetType() AgentType {
	return "openai_functions"
}

// CreateOpenAIFunctionsAgent 创建 OpenAI Functions Agent (工厂函数)。
//
// 这是一个高层 API，针对 OpenAI Functions API 优化。
//
// 参数：
//   - llm: OpenAI 语言模型
//   - agentTools: 工具列表
//   - opts: 可选配置
//
// 返回：
//   - Agent: OpenAI Functions Agent 实例
//
// 示例：
//
//	agent := agents.CreateOpenAIFunctionsAgent(openaiLLM, tools,
//	    agents.WithOpenAIFunctionsSystemPrompt("You are a helpful assistant"),
//	    agents.WithOpenAIFunctionsForceFunction("weather"),
//	)
//
func CreateOpenAIFunctionsAgent(llm chat.ChatModel, agentTools []tools.Tool, opts ...OpenAIFunctionsOption) Agent {
	config := OpenAIFunctionsConfig{
		LLM:      llm,
		Tools:    agentTools,
		MaxSteps: 10,
		Verbose:  false,
	}
	
	// 应用选项
	for _, opt := range opts {
		opt(&config)
	}
	
	return NewOpenAIFunctionsAgent(config)
}

// OpenAIFunctionsOption 是 OpenAI Functions Agent 配置选项。
type OpenAIFunctionsOption func(*OpenAIFunctionsConfig)

// WithOpenAIFunctionsSystemPrompt 设置系统提示词。
func WithOpenAIFunctionsSystemPrompt(prompt string) OpenAIFunctionsOption {
	return func(config *OpenAIFunctionsConfig) {
		config.SystemPrompt = prompt
	}
}

// WithOpenAIFunctionsMaxSteps 设置最大步数。
func WithOpenAIFunctionsMaxSteps(maxSteps int) OpenAIFunctionsOption {
	return func(config *OpenAIFunctionsConfig) {
		config.MaxSteps = maxSteps
	}
}

// WithOpenAIFunctionsVerbose 设置是否输出详细日志。
func WithOpenAIFunctionsVerbose(verbose bool) OpenAIFunctionsOption {
	return func(config *OpenAIFunctionsConfig) {
		config.Verbose = verbose
	}
}

// WithOpenAIFunctionsForceFunction 设置强制调用的函数。
func WithOpenAIFunctionsForceFunction(funcName string) OpenAIFunctionsOption {
	return func(config *OpenAIFunctionsConfig) {
		config.ForceFunction = funcName
	}
}
