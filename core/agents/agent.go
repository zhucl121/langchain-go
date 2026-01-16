package agents

import (
	"context"
	"fmt"
	
	"langchain-go/core/chat"
	"langchain-go/core/tools"
	"langchain-go/pkg/types"
)

// AgentType 是 Agent 类型。
type AgentType string

const (
	// AgentTypeReAct 是 ReAct (Reasoning + Acting) Agent
	AgentTypeReAct AgentType = "react"

	// AgentTypeToolCalling 使用原生工具调用
	AgentTypeToolCalling AgentType = "tool_calling"

	// AgentTypeConversational 对话式 Agent
	AgentTypeConversational AgentType = "conversational"

	// AgentTypePlanAndExecute Plan-and-Execute Agent
	AgentTypePlanAndExecute AgentType = "plan_and_execute"
	
	// AgentTypeSelfAsk Self-Ask Agent (递归分解问题)
	AgentTypeSelfAsk AgentType = "self_ask"
	
	// AgentTypeStructuredChat Structured Chat Agent (结构化对话)
	AgentTypeStructuredChat AgentType = "structured_chat"
)

// Agent 是 Agent 接口。
type Agent interface {
	// Plan 规划下一步行动
	//
	// 参数：
	//   - ctx: 上下文
	//   - input: 输入
	//   - history: 历史记录
	//
	// 返回：
	//   - *AgentAction: 下一步行动
	//   - error: 错误
	//
	Plan(ctx context.Context, input string, history []AgentStep) (*AgentAction, error)

	// GetType 返回 Agent 类型
	GetType() AgentType

	// GetTools 返回可用工具
	GetTools() []tools.Tool
}

// AgentConfig 是 Agent 配置。
type AgentConfig struct {
	// Type Agent 类型
	Type AgentType

	// LLM 语言模型
	LLM chat.ChatModel

	// Tools 工具列表
	Tools []tools.Tool

	// MaxSteps 最大步数
	MaxSteps int

	// SystemPrompt 系统提示词
	SystemPrompt string

	// Verbose 是否输出详细日志
	Verbose bool

	// Extra 额外配置
	Extra map[string]any
}

// AgentAction 是 Agent 行动。
type AgentAction struct {
	// Type 行动类型
	Type AgentActionType

	// Tool 工具名称（如果是工具调用）
	Tool string

	// ToolInput 工具输入
	ToolInput map[string]any

	// Log 思考过程日志
	Log string

	// FinalAnswer 最终答案（如果已完成）
	FinalAnswer string
}

// AgentActionType 是行动类型。
type AgentActionType string

const (
	// ActionToolCall 工具调用
	ActionToolCall AgentActionType = "tool_call"

	// ActionFinish 完成任务
	ActionFinish AgentActionType = "finish"

	// ActionError 错误
	ActionError AgentActionType = "error"
)

// AgentStep 是 Agent 执行步骤。
type AgentStep struct {
	// Action 执行的行动
	Action *AgentAction

	// Observation 观察结果
	Observation string

	// Error 错误（如果有）
	Error error
}

// AgentResult 是 Agent 执行结果。
type AgentResult struct {
	// Output 最终输出
	Output string

	// Steps 执行步骤
	Steps []AgentStep

	// TotalSteps 总步数
	TotalSteps int

	// Success 是否成功
	Success bool

	// Error 错误
	Error error
}

// CreateAgent 创建 Agent。
//
// 参数：
//   - config: Agent 配置
//
// 返回：
//   - Agent: Agent 实例
//   - error: 错误
//
func CreateAgent(config AgentConfig) (Agent, error) {
	if config.LLM == nil {
		return nil, fmt.Errorf("agents: LLM is required")
	}

	if len(config.Tools) == 0 {
		return nil, fmt.Errorf("agents: at least one tool is required")
	}

	if config.MaxSteps <= 0 {
		config.MaxSteps = 10
	}

	switch config.Type {
	case AgentTypeReAct:
		return NewReActAgent(config), nil
	case AgentTypeToolCalling:
		return NewToolCallingAgent(config), nil
	case AgentTypeConversational:
		return NewConversationalAgent(config), nil
	default:
		return nil, fmt.Errorf("agents: unknown agent type: %s", config.Type)
	}
}

// BaseAgent 是基础 Agent 实现。
type BaseAgent struct {
	config AgentConfig
	llm    chat.ChatModel
	tools  []tools.Tool
}

// NewBaseAgent 创建基础 Agent。
func NewBaseAgent(config AgentConfig) *BaseAgent {
	return &BaseAgent{
		config: config,
		llm:    config.LLM,
		tools:  config.Tools,
	}
}

// GetType 返回 Agent 类型。
func (ba *BaseAgent) GetType() AgentType {
	return ba.config.Type
}

// GetTools 返回工具列表。
func (ba *BaseAgent) GetTools() []tools.Tool {
	return ba.tools
}

// GetTool 根据名称获取工具。
func (ba *BaseAgent) GetTool(name string) (tools.Tool, error) {
	for _, tool := range ba.tools {
		if tool.GetName() == name {
			return tool, nil
		}
	}
	return nil, fmt.Errorf("agents: tool not found: %s", name)
}

// FormatToolsForPrompt 格式化工具列表为提示词。
func (ba *BaseAgent) FormatToolsForPrompt() string {
	result := "Available tools:\n"
	for _, tool := range ba.tools {
		result += fmt.Sprintf("- %s: %s\n", tool.GetName(), tool.GetDescription())
	}
	return result
}

// ConvertToolsToTypesTools 转换为 types.Tool。
func (ba *BaseAgent) ConvertToolsToTypesTools() []types.Tool {
	result := make([]types.Tool, len(ba.tools))
	for i, tool := range ba.tools {
		result[i] = tool.ToTypesTool()
	}
	return result
}

// 错误定义
var (
	ErrAgentMaxSteps = fmt.Errorf("agents: max steps reached")
	ErrAgentParsing  = fmt.Errorf("agents: failed to parse agent output")
	ErrAgentNoTool   = fmt.Errorf("agents: tool not found")
)
