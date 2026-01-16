package agents

import (
	"context"
	
	"langchain-go/core/chat"
	"langchain-go/core/prompts/templates"
	"langchain-go/core/tools"
)

// CreateReActAgent 创建 ReAct Agent (简化工厂函数)。
//
// 这是一个高层 API，对标 Python 的 create_react_agent。
//
// 参数：
//   - llm: 语言模型
//   - agentTools: 工具列表
//   - opts: 可选配置
//
// 返回：
//   - Agent: ReAct Agent 实例
//
// 示例：
//
//	agent := agents.CreateReActAgent(llm, tools,
//	    agents.WithMaxSteps(10),
//	    agents.WithVerbose(true),
//	)
//
func CreateReActAgent(llm chat.ChatModel, agentTools []tools.Tool, opts ...AgentOption) Agent {
	config := AgentConfig{
		Type:         AgentTypeReAct,
		LLM:          llm,
		Tools:        agentTools,
		MaxSteps:     10,
		SystemPrompt: templates.ReActPrompt,
		Verbose:      false,
		Extra:        make(map[string]any),
	}

	// 应用选项
	for _, opt := range opts {
		opt(&config)
	}

	return NewReActAgent(config)
}

// CreateToolCallingAgent 创建 Tool Calling Agent (简化工厂函数)。
//
// 这是一个高层 API，对标 Python 的 create_tool_calling_agent。
// 使用原生工具调用能力 (如 OpenAI Functions)。
//
// 参数：
//   - llm: 语言模型 (需支持工具调用)
//   - agentTools: 工具列表
//   - opts: 可选配置
//
// 返回：
//   - Agent: Tool Calling Agent 实例
//
// 示例：
//
//	agent := agents.CreateToolCallingAgent(llm, tools,
//	    agents.WithSystemPrompt("You are a helpful assistant"),
//	)
//
func CreateToolCallingAgent(llm chat.ChatModel, agentTools []tools.Tool, opts ...AgentOption) Agent {
	config := AgentConfig{
		Type:         AgentTypeToolCalling,
		LLM:          llm,
		Tools:        agentTools,
		MaxSteps:     10,
		SystemPrompt: templates.ToolCallingPrompt,
		Verbose:      false,
		Extra:        make(map[string]any),
	}

	// 应用选项
	for _, opt := range opts {
		opt(&config)
	}

	return NewToolCallingAgent(config)
}

// CreateConversationalAgent 创建对话式 Agent (简化工厂函数)。
//
// 对话式 Agent 不使用工具，只进行对话。
//
// 参数：
//   - llm: 语言模型
//   - opts: 可选配置
//
// 返回：
//   - Agent: 对话式 Agent 实例
//
// 示例：
//
//	agent := agents.CreateConversationalAgent(llm,
//	    agents.WithSystemPrompt("You are a friendly assistant"),
//	)
//
func CreateConversationalAgent(llm chat.ChatModel, opts ...AgentOption) Agent {
	config := AgentConfig{
		Type:         AgentTypeConversational,
		LLM:          llm,
		Tools:        nil,
		MaxSteps:     1,
		SystemPrompt: "You are a helpful conversational assistant.",
		Verbose:      false,
		Extra:        make(map[string]any),
	}

	// 应用选项
	for _, opt := range opts {
		opt(&config)
	}

	return NewConversationalAgent(config)
}

// CreatePlanExecuteAgent 创建 Plan-Execute Agent (简化工厂函数)。
//
// 这是一个高层 API，先规划再执行的 Agent，适合复杂多步骤任务。
//
// 参数：
//   - llm: 语言模型
//   - agentTools: 工具列表
//   - opts: 可选配置
//
// 返回：
//   - Agent: Plan-Execute Agent 实例
//
// 示例：
//
//	agent := agents.CreatePlanExecuteAgent(llm, tools,
//	    agents.WithMaxSteps(10),
//	    agents.WithVerbose(true),
//	    agents.WithPlanExecuteReplan(true),
//	)
//
func CreatePlanExecuteAgent(llm chat.ChatModel, agentTools []tools.Tool, opts ...PlanExecuteOption) Agent {
	config := PlanAndExecuteConfig{
		LLM:          llm,
		Tools:        agentTools,
		MaxSteps:     10,
		EnableReplan: false,
		Verbose:      false,
	}

	// 应用选项
	for _, opt := range opts {
		opt(&config)
	}

	return NewPlanAndExecuteAgent(config)
}

// PlanExecuteOption 是 Plan-Execute Agent 配置选项。
type PlanExecuteOption func(*PlanAndExecuteConfig)

// WithPlanExecuteMaxSteps 设置最大步数。
func WithPlanExecuteMaxSteps(maxSteps int) PlanExecuteOption {
	return func(config *PlanAndExecuteConfig) {
		config.MaxSteps = maxSteps
	}
}

// WithPlanExecuteReplan 设置是否启用重新规划。
func WithPlanExecuteReplan(enable bool) PlanExecuteOption {
	return func(config *PlanAndExecuteConfig) {
		config.EnableReplan = enable
	}
}

// WithPlanExecuteVerbose 设置是否输出详细日志。
func WithPlanExecuteVerbose(verbose bool) PlanExecuteOption {
	return func(config *PlanAndExecuteConfig) {
		config.Verbose = verbose
	}
}

// WithPlanExecutePlannerPrompt 设置规划器提示词。
func WithPlanExecutePlannerPrompt(prompt string) PlanExecuteOption {
	return func(config *PlanAndExecuteConfig) {
		config.PlannerPrompt = prompt
	}
}

// WithPlanExecuteExecutorPrompt 设置执行器提示词。
func WithPlanExecuteExecutorPrompt(prompt string) PlanExecuteOption {
	return func(config *PlanAndExecuteConfig) {
		config.ExecutorPrompt = prompt
	}
}

// AgentOption 是 Agent 配置选项。
type AgentOption func(*AgentConfig)

// WithMaxSteps 设置最大步数。
//
// 参数：
//   - maxSteps: 最大步数
//
// 返回：
//   - AgentOption: 配置选项
//
func WithMaxSteps(maxSteps int) AgentOption {
	return func(config *AgentConfig) {
		config.MaxSteps = maxSteps
	}
}

// WithSystemPrompt 设置系统提示词。
//
// 参数：
//   - prompt: 系统提示词
//
// 返回：
//   - AgentOption: 配置选项
//
func WithSystemPrompt(prompt string) AgentOption {
	return func(config *AgentConfig) {
		config.SystemPrompt = prompt
	}
}

// WithVerbose 设置是否输出详细日志。
//
// 参数：
//   - verbose: 是否详细输出
//
// 返回：
//   - AgentOption: 配置选项
//
func WithVerbose(verbose bool) AgentOption {
	return func(config *AgentConfig) {
		config.Verbose = verbose
	}
}

// WithExtra 设置额外配置。
//
// 参数：
//   - key: 配置键
//   - value: 配置值
//
// 返回：
//   - AgentOption: 配置选项
//
func WithExtra(key string, value any) AgentOption {
	return func(config *AgentConfig) {
		if config.Extra == nil {
			config.Extra = make(map[string]any)
		}
		config.Extra[key] = value
	}
}

// SimplifiedAgentExecutor 是简化的 Agent 执行器。
//
// 提供更简单的 API，类似 Python 的 AgentExecutor。
type SimplifiedAgentExecutor struct {
	executor *AgentExecutor
}

// NewSimplifiedAgentExecutor 创建简化的 Agent 执行器。
//
// 参数：
//   - agent: Agent 实例
//   - agentTools: 工具列表
//   - opts: 可选配置
//
// 返回：
//   - *SimplifiedAgentExecutor: 执行器实例
//
// 示例：
//
//	agent := agents.CreateReActAgent(llm, tools)
//	executor := agents.NewSimplifiedAgentExecutor(agent, tools,
//	    agents.WithMaxSteps(15),
//	)
//	result, _ := executor.Run(ctx, "What is 25 * 4?")
//
func NewSimplifiedAgentExecutor(agent Agent, agentTools []tools.Tool, opts ...AgentOption) *SimplifiedAgentExecutor {
	// 创建默认配置
	config := AgentConfig{
		MaxSteps: 10,
		Verbose:  false,
	}

	// 应用选项
	for _, opt := range opts {
		opt(&config)
	}

	// 创建 ToolExecutor
	toolExecutor := tools.NewToolExecutor(tools.ToolExecutorConfig{
		Tools: agentTools,
	})

	// 创建 AgentExecutor
	executorConfig := AgentExecutorConfig{
		Agent:        agent,
		ToolExecutor: toolExecutor,
		MaxSteps:     config.MaxSteps,
		Verbose:      config.Verbose,
	}

	return &SimplifiedAgentExecutor{
		executor: NewAgentExecutor(executorConfig),
	}
}

// Run 执行 Agent (同步)。
//
// 参数：
//   - ctx: 上下文
//   - input: 输入问题
//
// 返回：
//   - *AgentResult: 执行结果
//   - error: 错误
//
func (se *SimplifiedAgentExecutor) Run(ctx context.Context, input string) (*AgentResult, error) {
	return se.executor.Run(ctx, input)
}

// Stream 流式执行 Agent。
//
// 参数：
//   - ctx: 上下文
//   - input: 输入问题
//
// 返回：
//   - <-chan AgentStreamEvent: 事件流
//
func (se *SimplifiedAgentExecutor) Stream(ctx context.Context, input string) <-chan AgentStreamEvent {
	return se.executor.Stream(ctx, input)
}
