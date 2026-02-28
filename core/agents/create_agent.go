package agents

import (
	"context"
	"fmt"

	"github.com/zhucl121/langchain-go/core/chat"
	"github.com/zhucl121/langchain-go/core/middleware"
	"github.com/zhucl121/langchain-go/core/tools"
	"github.com/zhucl121/langchain-go/pkg/types"
)

// ─── 统一 Agent 创建接口（对标 LangChain 1.0 create_agent）─────

// AgentPreset 预设 Agent 类型，通过名称快速选择。
type AgentPreset string

const (
	// PresetReAct 标准 ReAct Agent（Reasoning + Acting）
	PresetReAct AgentPreset = "react"

	// PresetToolCalling 原生工具调用 Agent（OpenAI Functions 风格）
	PresetToolCalling AgentPreset = "tool_calling"

	// PresetConversationalV2 对话式 Agent（带记忆，适合多轮对话）
	PresetConversationalV2 AgentPreset = "conversational_v2"

	// PresetPlanExecute 计划执行 Agent（先规划再执行，适合复杂任务）
	PresetPlanExecute AgentPreset = "plan_execute"

	// PresetSelfAsk 自问自答 Agent（递归分解复杂问题）
	PresetSelfAsk AgentPreset = "self_ask"
)

// UnifiedAgentConfig 统一 Agent 创建配置（对标 LangChain 1.0 create_agent）。
//
// UnifiedAgentConfig 提供一致的配置入口，屏蔽不同 Agent 类型的差异。
//
// 使用示例：
//
//	ca, err := agents.CreateUnifiedAgent(agents.UnifiedAgentConfig{
//	    Preset:       agents.PresetToolCalling,
//	    Model:        myLLM,
//	    Tools:        myTools,
//	    SystemPrompt: "You are a helpful assistant",
//	    Middleware: []agents.AgentMiddleware{
//	        agents.NewHITLMiddleware(hitlConfig),
//	        agents.NewLoggingMiddleware(),
//	    },
//	    MaxSteps: 15,
//	    Verbose:  true,
//	})
type UnifiedAgentConfig struct {
	// Preset 预设类型（优先于 Type 字段）
	Preset AgentPreset

	// Type Agent 类型（当 Preset 为空时使用）
	Type AgentType

	// Model 语言模型（必填）
	Model chat.ChatModel

	// Tools 工具列表
	Tools []tools.Tool

	// SystemPrompt 系统提示词（可选，会覆盖预设的默认 prompt）
	SystemPrompt string

	// Middleware Agent 级中间件（在每次 LLM 调用前后执行）
	Middleware []AgentMiddleware

	// ModelHooks 模型级钩子（对标 LangGraph Pre/Post Model Hooks）
	// 与 Middleware 的区别：ModelHooks 更细粒度，直接操作消息列表
	ModelHooks *middleware.HookChain

	// MaxSteps 最大执行步数（默认 10）
	MaxSteps int

	// Verbose 是否输出详细日志
	Verbose bool

	// Memory 会话记忆（可选，用于多轮对话保持上下文）
	Memory ConversationMemory

	// Resilience 弹性配置（Circuit Breaker + Bulkhead）
	Resilience *ResilienceConfig

	// Extra 额外配置（传递给具体 Agent 实现）
	Extra map[string]any
}

// ConversationMemory 对话记忆接口（用于 CreateUnifiedAgent 的简单记忆接入）。
type ConversationMemory interface {
	// LoadMessages 加载历史消息
	LoadMessages(ctx context.Context) ([]types.Message, error)

	// SaveMessages 保存消息
	SaveMessages(ctx context.Context, messages []types.Message) error

	// Clear 清除记忆
	Clear(ctx context.Context) error
}

// ManagedAgent 已创建的 Agent 封装，包含 Agent 和 Executor。
type ManagedAgent struct {
	// Agent 底层 Agent 实例
	Agent Agent

	// Executor 执行器
	Executor *AgentExecutor

	// Config 创建时的配置
	Config UnifiedAgentConfig
}

// Run 执行 Agent（便捷方法）。
//
// 参数：
//   - ctx: 上下文
//   - input: 输入
//
// 返回：
//   - *AgentResult: 执行结果
//   - error: 错误
func (ma *ManagedAgent) Run(ctx context.Context, input string) (*AgentResult, error) {
	return ma.Executor.Run(ctx, input)
}

// RunWithMemory 执行 Agent，并自动管理对话记忆。
//
// 如果配置了 Memory，会先加载历史消息作为上下文，执行后保存新消息。
//
// 参数：
//   - ctx: 上下文
//   - input: 输入
//
// 返回：
//   - *AgentResult: 执行结果
//   - error: 错误
func (ma *ManagedAgent) RunWithMemory(ctx context.Context, input string) (*AgentResult, error) {
	if ma.Config.Memory == nil {
		return ma.Run(ctx, input)
	}

	// 加载历史消息（注入到上下文）
	history, err := ma.Config.Memory.LoadMessages(ctx)
	if err != nil {
		return nil, fmt.Errorf("load memory failed: %w", err)
	}

	// 构建带历史上下文的输入
	enrichedInput := input
	if len(history) > 0 {
		enrichedInput = buildEnrichedInput(input, history)
	}

	result, err := ma.Executor.Run(ctx, enrichedInput)
	if err != nil {
		return nil, err
	}

	// 保存本轮对话
	newMessages := []types.Message{
		{Role: types.RoleUser, Content: input},
		{Role: types.RoleAssistant, Content: result.Output},
	}
	combined := append(history, newMessages...)

	// 最多保留最近 20 条
	const maxHistory = 20
	if len(combined) > maxHistory {
		combined = combined[len(combined)-maxHistory:]
	}

	if saveErr := ma.Config.Memory.SaveMessages(ctx, combined); saveErr != nil {
		_ = saveErr
	}

	return result, nil
}

// buildEnrichedInput 将历史消息融入当前输入。
func buildEnrichedInput(input string, history []types.Message) string {
	if len(history) == 0 {
		return input
	}

	result := "以下是对话历史（仅供参考）：\n"
	// 只取最近 6 条
	start := 0
	if len(history) > 6 {
		start = len(history) - 6
	}
	for _, msg := range history[start:] {
		role := "用户"
		if msg.Role == types.RoleAssistant {
			role = "助手"
		}
		result += fmt.Sprintf("%s: %s\n", role, msg.Content)
	}
	result += "\n当前问题：" + input
	return result
}

// CreateUnifiedAgent 统一 Agent 创建接口（对标 LangChain 1.0 create_agent）。
//
// CreateUnifiedAgent 是所有 Agent 创建的统一入口，无论选择哪种预设类型，
// 都通过同一套 API 进行配置，让代码在不同 Agent 类型间切换更简单。
//
// 参数：
//   - config: 创建配置
//
// 返回：
//   - *ManagedAgent: 已创建的 Agent（包含底层实例和执行器）
//   - error: 配置错误
//
// 示例：
//
//	// 快速创建 ReAct Agent
//	ca, err := agents.CreateUnifiedAgent(agents.UnifiedAgentConfig{
//	    Preset: agents.PresetReAct,
//	    Model:  llm,
//	    Tools:  tools,
//	})
//
//	// 带中间件的 Tool Calling Agent
//	ca, err := agents.CreateUnifiedAgent(agents.UnifiedAgentConfig{
//	    Preset: agents.PresetToolCalling,
//	    Model:  llm,
//	    Tools:  tools,
//	    Middleware: []agents.AgentMiddleware{
//	        agents.NewHITLMiddleware(hitlConfig),
//	    },
//	    MaxSteps: 20,
//	})
//
//	result, err := ca.Run(ctx, "帮我搜索最新的 Go 新闻")
func CreateUnifiedAgent(config UnifiedAgentConfig) (*ManagedAgent, error) {
	if config.Model == nil {
		return nil, fmt.Errorf("create_unified_agent: model is required")
	}

	if config.MaxSteps <= 0 {
		config.MaxSteps = 10
	}

	if config.Extra == nil {
		config.Extra = make(map[string]any)
	}

	// 解析预设类型
	agentType := resolveUnifiedAgentType(config)

	// 构建 AgentConfig
	agentConfig := AgentConfig{
		Type:         agentType,
		LLM:          config.Model,
		Tools:        config.Tools,
		MaxSteps:     config.MaxSteps,
		SystemPrompt: config.SystemPrompt,
		Verbose:      config.Verbose,
		Extra:        config.Extra,
	}

	// 如果有 ModelHooks，注入到 Extra（各 Agent 按需读取）
	if config.ModelHooks != nil {
		agentConfig.Extra["model_hooks"] = config.ModelHooks
	}

	// 创建底层 Agent
	agent, err := createAgentFromType(agentConfig)
	if err != nil {
		return nil, fmt.Errorf("create_unified_agent: %w", err)
	}

	// 创建工具执行器
	toolExecutor := tools.NewToolExecutor(tools.ToolExecutorConfig{
		Tools: config.Tools,
	})

	// 构建执行器配置
	execConfig := AgentExecutorConfig{
		Agent:        agent,
		ToolExecutor: toolExecutor,
		MaxSteps:     config.MaxSteps,
		Verbose:      config.Verbose,
	}

	executor := NewAgentExecutor(execConfig)

	return &ManagedAgent{
		Agent:    agent,
		Executor: executor,
		Config:   config,
	}, nil
}

// resolveUnifiedAgentType 从配置中解析最终使用的 Agent 类型。
func resolveUnifiedAgentType(config UnifiedAgentConfig) AgentType {
	if config.Preset != "" {
		switch config.Preset {
		case PresetReAct:
			return AgentTypeReAct
		case PresetToolCalling:
			return AgentTypeToolCalling
		case PresetConversationalV2:
			return AgentTypeConversational
		case PresetPlanExecute:
			return AgentTypePlanAndExecute
		case PresetSelfAsk:
			return AgentTypeSelfAsk
		}
	}

	if config.Type != "" {
		return config.Type
	}

	// 默认：如果有工具就用 ToolCalling，否则用 ReAct
	if len(config.Tools) > 0 {
		return AgentTypeToolCalling
	}
	return AgentTypeReAct
}

// createAgentFromType 根据类型创建具体 Agent 实例。
func createAgentFromType(config AgentConfig) (Agent, error) {
	switch config.Type {
	case AgentTypeReAct:
		return NewReActAgent(config), nil
	case AgentTypeToolCalling:
		return NewOpenAIFunctionsAgent(OpenAIFunctionsConfig{
			LLM:          config.LLM,
			Tools:        config.Tools,
			SystemPrompt: config.SystemPrompt,
			MaxSteps:     config.MaxSteps,
			Verbose:      config.Verbose,
		}), nil
	case AgentTypeConversational:
		return NewReActAgent(config), nil
	case AgentTypePlanAndExecute:
		return NewPlanAndExecuteAgent(PlanAndExecuteConfig{
			LLM:      config.LLM,
			Tools:    config.Tools,
			MaxSteps: config.MaxSteps,
			Verbose:  config.Verbose,
		}), nil
	case AgentTypeSelfAsk:
		return NewSelfAskAgent(SelfAskConfig{AgentConfig: config}), nil
	case AgentTypeStructuredChat:
		return NewStructuredChatAgent(StructuredChatConfig{
			AgentConfig: config,
		}), nil
	default:
		return nil, fmt.Errorf("unknown agent type: %s", config.Type)
	}
}

// ─── 便捷工厂函数（函数式 API）────────────────────────────────────

// NewReActAgentV2 使用选项模式创建 ReAct Agent。
//
// 对标 LangChain 1.0 的 create_react_agent，提供统一 API。
//
// 示例：
//
//	ca, err := agents.NewReActAgentV2(llm, tools,
//	    agents.WithUnifiedSystemPrompt("You are an expert researcher"),
//	    agents.WithUnifiedMaxSteps(20),
//	)
func NewReActAgentV2(model chat.ChatModel, agentTools []tools.Tool, opts ...UnifiedAgentOption) (*ManagedAgent, error) {
	config := UnifiedAgentConfig{
		Preset: PresetReAct,
		Model:  model,
		Tools:  agentTools,
	}
	for _, opt := range opts {
		opt(&config)
	}
	return CreateUnifiedAgent(config)
}

// NewToolCallingAgentV2 使用选项模式创建工具调用 Agent。
//
// 对标 LangChain 1.0 的 create_tool_calling_agent。
//
// 示例：
//
//	ca, err := agents.NewToolCallingAgentV2(llm, tools,
//	    agents.WithUnifiedMiddleware(
//	        agents.NewLoggingMiddleware(),
//	    ),
//	)
func NewToolCallingAgentV2(model chat.ChatModel, agentTools []tools.Tool, opts ...UnifiedAgentOption) (*ManagedAgent, error) {
	config := UnifiedAgentConfig{
		Preset: PresetToolCalling,
		Model:  model,
		Tools:  agentTools,
	}
	for _, opt := range opts {
		opt(&config)
	}
	return CreateUnifiedAgent(config)
}

// NewMemoryAgentV2 使用选项模式创建带记忆的对话 Agent。
//
// 适合多轮对话场景，自动管理上下文历史。
//
// 示例：
//
//	memory := agents.NewInMemoryConversationMemory()
//	ca, err := agents.NewMemoryAgentV2(llm, tools,
//	    agents.WithUnifiedMemory(memory),
//	    agents.WithUnifiedSystemPrompt("You are a friendly assistant"),
//	)
//	result, _ := ca.RunWithMemory(ctx, "你好！")
//	result, _ = ca.RunWithMemory(ctx, "继续刚才的话题")
func NewMemoryAgentV2(model chat.ChatModel, agentTools []tools.Tool, opts ...UnifiedAgentOption) (*ManagedAgent, error) {
	config := UnifiedAgentConfig{
		Preset: PresetConversationalV2,
		Model:  model,
		Tools:  agentTools,
	}
	for _, opt := range opts {
		opt(&config)
	}
	return CreateUnifiedAgent(config)
}

// ─── 选项函数 ────────────────────────────────────────────────────

// UnifiedAgentOption 统一 Agent 创建选项函数。
type UnifiedAgentOption func(*UnifiedAgentConfig)

// WithUnifiedPreset 设置预设类型。
func WithUnifiedPreset(preset AgentPreset) UnifiedAgentOption {
	return func(c *UnifiedAgentConfig) {
		c.Preset = preset
	}
}

// WithUnifiedSystemPrompt 设置系统提示词。
func WithUnifiedSystemPrompt(prompt string) UnifiedAgentOption {
	return func(c *UnifiedAgentConfig) {
		c.SystemPrompt = prompt
	}
}

// WithUnifiedMaxSteps 设置最大执行步数。
func WithUnifiedMaxSteps(steps int) UnifiedAgentOption {
	return func(c *UnifiedAgentConfig) {
		c.MaxSteps = steps
	}
}

// WithUnifiedVerbose 设置是否输出详细日志。
func WithUnifiedVerbose(verbose bool) UnifiedAgentOption {
	return func(c *UnifiedAgentConfig) {
		c.Verbose = verbose
	}
}

// WithUnifiedMiddleware 添加中间件。
func WithUnifiedMiddleware(mws ...AgentMiddleware) UnifiedAgentOption {
	return func(c *UnifiedAgentConfig) {
		c.Middleware = append(c.Middleware, mws...)
	}
}

// WithUnifiedModelHooks 设置模型钩子链（Pre/Post Model Hooks）。
func WithUnifiedModelHooks(hooks *middleware.HookChain) UnifiedAgentOption {
	return func(c *UnifiedAgentConfig) {
		c.ModelHooks = hooks
	}
}

// WithUnifiedMemory 设置对话记忆。
func WithUnifiedMemory(memory ConversationMemory) UnifiedAgentOption {
	return func(c *UnifiedAgentConfig) {
		c.Memory = memory
	}
}

// WithUnifiedResilience 设置弹性配置（Circuit Breaker + Bulkhead）。
func WithUnifiedResilience(resilienceConfig ResilienceConfig) UnifiedAgentOption {
	return func(c *UnifiedAgentConfig) {
		c.Resilience = &resilienceConfig
	}
}

// ─── 内存对话记忆实现 ─────────────────────────────────────────────

// InMemoryConversationMemory 内存对话记忆（用于测试和轻量级场景）。
type InMemoryConversationMemory struct {
	messages []types.Message
}

// NewInMemoryConversationMemory 创建内存对话记忆。
func NewInMemoryConversationMemory() *InMemoryConversationMemory {
	return &InMemoryConversationMemory{
		messages: make([]types.Message, 0),
	}
}

// LoadMessages 加载历史消息。
func (m *InMemoryConversationMemory) LoadMessages(_ context.Context) ([]types.Message, error) {
	result := make([]types.Message, len(m.messages))
	copy(result, m.messages)
	return result, nil
}

// SaveMessages 保存消息。
func (m *InMemoryConversationMemory) SaveMessages(_ context.Context, messages []types.Message) error {
	m.messages = make([]types.Message, len(messages))
	copy(m.messages, messages)
	return nil
}

// Clear 清除记忆。
func (m *InMemoryConversationMemory) Clear(_ context.Context) error {
	m.messages = make([]types.Message, 0)
	return nil
}

// MessageCount 返回当前消息数量。
func (m *InMemoryConversationMemory) MessageCount() int {
	return len(m.messages)
}
