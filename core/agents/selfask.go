package agents

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	
	"langchain-go/core/chat"
	"langchain-go/core/tools"
	"langchain-go/pkg/types"
)

// SelfAskAgent 是 Self-Ask Agent。
//
// Self-Ask Agent 通过递归分解问题的方式来解决复杂问题。
// 它会不断提问"我需要知道什么来回答这个问题？"，然后逐步解决子问题。
//
// 参考论文: "Measuring and Narrowing the Compositionality Gap in Language Models"
// https://arxiv.org/abs/2210.03350
//
type SelfAskAgent struct {
	*BaseAgent
	systemPrompt  string
	maxSubQuestions int
}

// SelfAskConfig 是 Self-Ask Agent 配置。
type SelfAskConfig struct {
	AgentConfig
	// MaxSubQuestions 最大子问题数量
	MaxSubQuestions int
}

// NewSelfAskAgent 创建 Self-Ask Agent。
func NewSelfAskAgent(config SelfAskConfig) *SelfAskAgent {
	if config.MaxSubQuestions <= 0 {
		config.MaxSubQuestions = 5
	}

	baseAgent := NewBaseAgent(config.AgentConfig)
	
	systemPrompt := config.SystemPrompt
	if systemPrompt == "" {
		systemPrompt = getDefaultSelfAskPrompt()
	}

	return &SelfAskAgent{
		BaseAgent:       baseAgent,
		systemPrompt:    systemPrompt,
		maxSubQuestions: config.MaxSubQuestions,
	}
}

// Plan 实现 Agent 接口。
func (sa *SelfAskAgent) Plan(ctx context.Context, input string, history []AgentStep) (*AgentAction, error) {
	// 检查子问题数量
	subQuestionCount := countSubQuestions(history)
	if subQuestionCount >= sa.maxSubQuestions {
		// 达到最大子问题数，强制生成最终答案
		return sa.planFinalAnswer(ctx, input, history)
	}

	// 构建提示词
	prompt := sa.buildPrompt(input, history)

	// 调用 LLM
	messages := []types.Message{
		types.NewSystemMessage(sa.systemPrompt),
		types.NewUserMessage(prompt),
	}

	response, err := sa.llm.Invoke(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("self-ask agent: LLM invoke failed: %w", err)
	}

	// 解析 LLM 响应
	action, err := sa.parseOutput(response.Content)
	if err != nil {
		return nil, fmt.Errorf("self-ask agent: parse output failed: %w", err)
	}

	return action, nil
}

// buildPrompt 构建提示词。
func (sa *SelfAskAgent) buildPrompt(input string, history []AgentStep) string {
	var builder strings.Builder

	// 添加原始问题
	builder.WriteString(fmt.Sprintf("Question: %s\n", input))

	// 添加历史
	if len(history) > 0 {
		for _, step := range history {
			if step.Action.Type == ActionToolCall {
				// 这是一个子问题
				followUpQuestion := extractFollowUpQuestion(step.Action.Log)
				if followUpQuestion != "" {
					builder.WriteString(fmt.Sprintf("Are follow up questions needed here: Yes.\n"))
					builder.WriteString(fmt.Sprintf("Follow up: %s\n", followUpQuestion))
					builder.WriteString(fmt.Sprintf("Intermediate answer: %s\n", step.Observation))
				}
			}
		}
	}

	builder.WriteString("Are follow up questions needed here:")

	return builder.String()
}

// parseOutput 解析 LLM 输出。
func (sa *SelfAskAgent) parseOutput(output string) (*AgentAction, error) {
	output = strings.TrimSpace(output)

	// 检查是否需要后续问题
	if strings.Contains(strings.ToLower(output), "yes") {
		// 提取后续问题
		followUpQuestion := extractFollowUpQuestion(output)
		if followUpQuestion == "" {
			return nil, fmt.Errorf("%w: no follow-up question found", ErrAgentParsing)
		}

		// 使用搜索工具回答子问题
		// 默认使用第一个可用工具（通常是搜索工具）
		if len(sa.tools) == 0 {
			return nil, fmt.Errorf("self-ask agent: no tools available")
		}

		return &AgentAction{
			Type: ActionToolCall,
			Tool: sa.tools[0].GetName(),
			ToolInput: map[string]any{
				"input": followUpQuestion,
			},
			Log: output,
		}, nil
	}

	// 不需要后续问题，提取最终答案
	finalAnswer := extractSoAnswer(output)
	if finalAnswer == "" {
		finalAnswer = output
	}

	return &AgentAction{
		Type:        ActionFinish,
		FinalAnswer: finalAnswer,
		Log:         output,
	}, nil
}

// planFinalAnswer 强制生成最终答案。
func (sa *SelfAskAgent) planFinalAnswer(ctx context.Context, input string, history []AgentStep) (*AgentAction, error) {
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("Question: %s\n\n", input))
	builder.WriteString("Based on the following intermediate answers, provide a final answer:\n\n")

	for i, step := range history {
		if step.Action.Type == ActionToolCall {
			followUpQuestion := extractFollowUpQuestion(step.Action.Log)
			builder.WriteString(fmt.Sprintf("%d. Question: %s\n", i+1, followUpQuestion))
			builder.WriteString(fmt.Sprintf("   Answer: %s\n\n", step.Observation))
		}
	}

	builder.WriteString("Final Answer:")

	messages := []types.Message{
		types.NewSystemMessage("You are a helpful assistant. Synthesize the intermediate answers into a comprehensive final answer."),
		types.NewUserMessage(builder.String()),
	}

	response, err := sa.llm.Invoke(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("self-ask agent: final answer generation failed: %w", err)
	}

	return &AgentAction{
		Type:        ActionFinish,
		FinalAnswer: strings.TrimSpace(response.Content),
		Log:         response.Content,
	}, nil
}

// getDefaultSelfAskPrompt 返回默认的 Self-Ask 提示词。
func getDefaultSelfAskPrompt() string {
	return `You are a helpful assistant that uses the "Self-Ask" method to answer questions.

When answering a question, you should:
1. Think about whether you need to ask follow-up questions to get more information
2. If yes, ask ONE follow-up question at a time
3. Use the intermediate answers to formulate your final answer

Format your response as:
- If you need more information:
  "Yes.
  Follow up: [your follow-up question]"

- If you can answer directly:
  "No.
  So the final answer is: [your answer]"

Begin!`
}

// 辅助函数

func extractFollowUpQuestion(text string) string {
	// 尝试多种模式
	patterns := []string{
		`(?i)Follow\s*up:\s*(.+?)(?:\n|$)`,
		`(?i)Follow[-\s]*up\s+question:\s*(.+?)(?:\n|$)`,
		`(?i)Sub[-\s]*question:\s*(.+?)(?:\n|$)`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(text)
		if len(matches) > 1 {
			return strings.TrimSpace(matches[1])
		}
	}

	return ""
}

func extractSoAnswer(text string) string {
	// 提取 "So the final answer is:" 后面的内容
	patterns := []string{
		`(?i)So\s+the\s+final\s+answer\s+is:\s*(.+)`,
		`(?i)Final\s+answer:\s*(.+)`,
		`(?i)Answer:\s*(.+)`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(text)
		if len(matches) > 1 {
			return strings.TrimSpace(matches[1])
		}
	}

	return ""
}

func countSubQuestions(history []AgentStep) int {
	count := 0
	for _, step := range history {
		if step.Action.Type == ActionToolCall {
			count++
		}
	}
	return count
}

// CreateSelfAskAgent 创建 Self-Ask Agent (简化工厂函数)。
//
// Self-Ask Agent 通过递归分解问题的方式来解决复杂问题。
// 适合需要多步推理的问题，如"谁是美国总统的母亲的故乡？"
//
// 参数：
//   - llm: 语言模型
//   - searchTool: 搜索工具（用于回答子问题）
//   - opts: 可选配置
//
// 返回：
//   - Agent: Self-Ask Agent 实例
//
// 示例：
//
//	searchTool := tools.NewWikipediaSearch(nil)
//	agent := agents.CreateSelfAskAgent(llm, searchTool,
//	    agents.WithSelfAskMaxSubQuestions(5),
//	    agents.WithVerbose(true),
//	)
//
func CreateSelfAskAgent(llm chat.ChatModel, searchTool tools.Tool, opts ...SelfAskOption) Agent {
	config := SelfAskConfig{
		AgentConfig: AgentConfig{
			Type:         "self_ask",
			LLM:          llm,
			Tools:        []tools.Tool{searchTool},
			MaxSteps:     10,
			SystemPrompt: getDefaultSelfAskPrompt(),
			Verbose:      false,
			Extra:        make(map[string]any),
		},
		MaxSubQuestions: 5,
	}

	// 应用选项
	for _, opt := range opts {
		opt(&config)
	}

	return NewSelfAskAgent(config)
}

// SelfAskOption 是 Self-Ask Agent 配置选项。
type SelfAskOption func(*SelfAskConfig)

// WithSelfAskMaxSteps 设置最大步数。
func WithSelfAskMaxSteps(maxSteps int) SelfAskOption {
	return func(config *SelfAskConfig) {
		config.MaxSteps = maxSteps
	}
}

// WithSelfAskMaxSubQuestions 设置最大子问题数量。
func WithSelfAskMaxSubQuestions(maxSubQuestions int) SelfAskOption {
	return func(config *SelfAskConfig) {
		config.MaxSubQuestions = maxSubQuestions
	}
}

// WithSelfAskVerbose 设置是否输出详细日志。
func WithSelfAskVerbose(verbose bool) SelfAskOption {
	return func(config *SelfAskConfig) {
		config.Verbose = verbose
	}
}

// WithSelfAskSystemPrompt 设置系统提示词。
func WithSelfAskSystemPrompt(prompt string) SelfAskOption {
	return func(config *SelfAskConfig) {
		config.SystemPrompt = prompt
	}
}
