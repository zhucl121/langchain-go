package agents

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	
	"langchain-go/pkg/types"
)

// ReActAgent 是 ReAct (Reasoning + Acting) Agent。
//
// ReAct Agent 通过交替进行推理和行动来解决问题。
//
type ReActAgent struct {
	*BaseAgent
	systemPrompt string
}

// NewReActAgent 创建 ReAct Agent。
func NewReActAgent(config AgentConfig) *ReActAgent {
	baseAgent := NewBaseAgent(config)
	
	systemPrompt := config.SystemPrompt
	if systemPrompt == "" {
		systemPrompt = getDefaultReActPrompt()
	}

	return &ReActAgent{
		BaseAgent:    baseAgent,
		systemPrompt: systemPrompt,
	}
}

// Plan 实现 Agent 接口。
func (ra *ReActAgent) Plan(ctx context.Context, input string, history []AgentStep) (*AgentAction, error) {
	// 构建提示词
	prompt := ra.buildPrompt(input, history)

	// 调用 LLM
	messages := []types.Message{
		types.NewSystemMessage(ra.systemPrompt),
		types.NewUserMessage(prompt),
	}

	response, err := ra.llm.Invoke(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("react agent: LLM invoke failed: %w", err)
	}

	// 解析 LLM 响应
	action, err := ra.parseOutput(response.Content)
	if err != nil {
		return nil, fmt.Errorf("react agent: parse output failed: %w", err)
	}

	return action, nil
}

// buildPrompt 构建提示词。
func (ra *ReActAgent) buildPrompt(input string, history []AgentStep) string {
	var builder strings.Builder

	// 添加工具列表
	builder.WriteString(ra.FormatToolsForPrompt())
	builder.WriteString("\n")

	// 添加问题
	builder.WriteString(fmt.Sprintf("Question: %s\n\n", input))

	// 添加历史
	if len(history) > 0 {
		builder.WriteString("Previous steps:\n")
		for i, step := range history {
			builder.WriteString(fmt.Sprintf("Step %d:\n", i+1))
			builder.WriteString(fmt.Sprintf("Thought: %s\n", step.Action.Log))
			if step.Action.Type == ActionToolCall {
				builder.WriteString(fmt.Sprintf("Action: %s\n", step.Action.Tool))
				builder.WriteString(fmt.Sprintf("Action Input: %v\n", step.Action.ToolInput))
				builder.WriteString(fmt.Sprintf("Observation: %s\n", step.Observation))
			}
			builder.WriteString("\n")
		}
	}

	builder.WriteString("Now, what should I do next?\n")

	return builder.String()
}

// parseOutput 解析 LLM 输出。
func (ra *ReActAgent) parseOutput(output string) (*AgentAction, error) {
	// 尝试提取 Thought、Action、Action Input、Final Answer

	// 检查是否是最终答案
	if strings.Contains(output, "Final Answer:") {
		finalAnswer := extractFinalAnswer(output)
		return &AgentAction{
			Type:        ActionFinish,
			FinalAnswer: finalAnswer,
			Log:         output,
		}, nil
	}

	// 提取 Thought
	thought := extractThought(output)

	// 提取 Action
	actionName := extractAction(output)
	if actionName == "" {
		return nil, fmt.Errorf("%w: no action found in output", ErrAgentParsing)
	}

	// 提取 Action Input
	actionInput := extractActionInput(output)

	return &AgentAction{
		Type:      ActionToolCall,
		Tool:      actionName,
		ToolInput: map[string]any{"input": actionInput},
		Log:       thought,
	}, nil
}

// getDefaultReActPrompt 返回默认的 ReAct 提示词。
func getDefaultReActPrompt() string {
	return `You are a helpful assistant that can use tools to answer questions.

Answer the following questions as best you can. You have access to the following tools:

When responding, use this format:

Thought: you should always think about what to do
Action: the action to take, should be one of the available tools
Action Input: the input to the action
Observation: the result of the action
... (this Thought/Action/Action Input/Observation can repeat N times)
Thought: I now know the final answer
Final Answer: the final answer to the original input question

Begin!`
}

// 辅助函数：提取各个部分

func extractThought(text string) string {
	re := regexp.MustCompile(`(?i)Thought:\s*(.+?)(?:\n|$)`)
	matches := re.FindStringSubmatch(text)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}
	return ""
}

func extractAction(text string) string {
	re := regexp.MustCompile(`(?i)Action:\s*(.+?)(?:\n|$)`)
	matches := re.FindStringSubmatch(text)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}
	return ""
}

func extractActionInput(text string) string {
	re := regexp.MustCompile(`(?i)Action Input:\s*(.+?)(?:\n|$)`)
	matches := re.FindStringSubmatch(text)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}
	return ""
}

func extractFinalAnswer(text string) string {
	re := regexp.MustCompile(`(?i)Final Answer:\s*(.+?)$`)
	matches := re.FindStringSubmatch(text)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}
	// 如果没有匹配到，返回整个文本
	return strings.TrimSpace(text)
}

// ToolCallingAgent 是使用原生工具调用的 Agent。
type ToolCallingAgent struct {
	*BaseAgent
}

// NewToolCallingAgent 创建 ToolCalling Agent。
func NewToolCallingAgent(config AgentConfig) *ToolCallingAgent {
	baseAgent := NewBaseAgent(config)
	return &ToolCallingAgent{
		BaseAgent: baseAgent,
	}
}

// Plan 实现 Agent 接口。
func (tca *ToolCallingAgent) Plan(ctx context.Context, input string, history []AgentStep) (*AgentAction, error) {
	// 构建消息
	messages := []types.Message{
		types.NewUserMessage(input),
	}

	// 添加历史
	for _, step := range history {
		if step.Action.Type == ActionToolCall {
			// 添加助手的工具调用
			messages = append(messages, types.NewAssistantMessage(step.Action.Log))
			// 添加工具结果
			messages = append(messages, types.NewToolMessage(
				step.Action.Tool,
				step.Observation,
			))
		}
	}

	// 先绑定工具，然后调用
	modelWithTools := tca.llm.BindTools(tca.ConvertToolsToTypesTools())
	
	response, err := modelWithTools.Invoke(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("tool calling agent: invoke failed: %w", err)
	}

	// 检查是否有工具调用
	if len(response.ToolCalls) > 0 {
		toolCall := response.ToolCalls[0]
		// Arguments 是 string，需要转换为 map
		args := map[string]any{"input": toolCall.Function.Arguments}
		return &AgentAction{
			Type:      ActionToolCall,
			Tool:      toolCall.Function.Name,
			ToolInput: args,
			Log:       response.Content,
		}, nil
	}

	// 没有工具调用，返回最终答案
	return &AgentAction{
		Type:        ActionFinish,
		FinalAnswer: response.Content,
		Log:         response.Content,
	}, nil
}

// ConversationalAgent 是对话式 Agent。
type ConversationalAgent struct {
	*BaseAgent
	systemPrompt string
}

// NewConversationalAgent 创建对话式 Agent。
func NewConversationalAgent(config AgentConfig) *ConversationalAgent {
	baseAgent := NewBaseAgent(config)
	
	systemPrompt := config.SystemPrompt
	if systemPrompt == "" {
		systemPrompt = "You are a helpful conversational assistant."
	}

	return &ConversationalAgent{
		BaseAgent:    baseAgent,
		systemPrompt: systemPrompt,
	}
}

// Plan 实现 Agent 接口。
func (ca *ConversationalAgent) Plan(ctx context.Context, input string, history []AgentStep) (*AgentAction, error) {
	// 构建对话历史
	messages := []types.Message{
		types.NewSystemMessage(ca.systemPrompt),
	}

	// 添加历史
	for _, step := range history {
		messages = append(messages, types.NewAssistantMessage(step.Observation))
	}

	// 添加当前输入
	messages = append(messages, types.NewUserMessage(input))

	// 调用 LLM
	response, err := ca.llm.Invoke(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("conversational agent: invoke failed: %w", err)
	}

	// 对话式 Agent 通常直接返回答案
	return &AgentAction{
		Type:        ActionFinish,
		FinalAnswer: response.Content,
		Log:         response.Content,
	}, nil
}
