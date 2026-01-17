package agents

import (
	"context"
	"fmt"
	"strings"
	
	"github.com/zhucl121/langchain-go/core/chat"
	"github.com/zhucl121/langchain-go/core/tools"
	"github.com/zhucl121/langchain-go/pkg/types"
)

// StepExecutor 是步骤执行器。
//
// StepExecutor 负责执行计划中的单个步骤。
//
type StepExecutor struct {
	llm     chat.ChatModel
	tools   []tools.Tool
	prompt  string
	verbose bool
}

// StepExecutorConfig 是步骤执行器配置。
type StepExecutorConfig struct {
	// LLM 语言模型
	LLM chat.ChatModel
	
	// Tools 工具列表
	Tools []tools.Tool
	
	// Prompt 执行提示词（可选）
	Prompt string
	
	// Verbose 是否输出详细日志
	Verbose bool
}

// NewStepExecutor 创建步骤执行器。
func NewStepExecutor(config StepExecutorConfig) *StepExecutor {
	prompt := config.Prompt
	if prompt == "" {
		prompt = getDefaultExecutorPrompt()
	}
	
	return &StepExecutor{
		llm:     config.LLM,
		tools:   config.Tools,
		prompt:  prompt,
		verbose: config.Verbose,
	}
}

// ExecuteStep 执行单个步骤。
//
// 参数：
//   - ctx: 上下文
//   - step: 要执行的步骤
//   - originalInput: 原始任务输入
//   - previousResults: 之前步骤的结果
//
// 返回：
//   - *AgentAction: 执行动作
//   - error: 错误
//
func (se *StepExecutor) ExecuteStep(
	ctx context.Context,
	step PlanStep,
	originalInput string,
	previousResults map[string]string,
) (*AgentAction, error) {
	// 构建提示词
	prompt := se.buildStepPrompt(step, originalInput, previousResults)
	
	messages := []types.Message{
		types.NewSystemMessage(se.prompt),
		types.NewUserMessage(prompt),
	}
	
	// 如果步骤指定了工具，尝试使用工具
	if step.ToolName != "" {
		tool := se.findToolByName(step.ToolName)
		if tool != nil {
			return se.executeWithTool(ctx, step, tool, messages)
		}
	}
	
	// 尝试使用工具调用
	if len(se.tools) > 0 {
		// 尝试让 LLM 选择工具
		return se.executeWithToolSelection(ctx, step, messages)
	}
	
	// 没有工具，直接使用 LLM
	return se.executeWithLLM(ctx, messages)
}

// buildStepPrompt 构建步骤执行提示词。
func (se *StepExecutor) buildStepPrompt(
	step PlanStep,
	originalInput string,
	previousResults map[string]string,
) string {
	var builder strings.Builder
	
	builder.WriteString(fmt.Sprintf("Original Task: %s\n\n", originalInput))
	builder.WriteString(fmt.Sprintf("Current Step: %s\n\n", step.Description))
	
	// 添加依赖步骤的结果
	if len(step.Dependencies) > 0 && previousResults != nil {
		builder.WriteString("Results from previous steps:\n")
		for _, depID := range step.Dependencies {
			if result, ok := previousResults[depID]; ok {
				builder.WriteString(fmt.Sprintf("- %s: %s\n", depID, result))
			}
		}
		builder.WriteString("\n")
	}
	
	// 添加所有之前的结果（按顺序）
	if previousResults != nil && len(previousResults) > 0 {
		builder.WriteString("All previous step results:\n")
		for i := 1; i <= len(previousResults); i++ {
			stepID := fmt.Sprintf("step_%d", i)
			if result, ok := previousResults[stepID]; ok {
				builder.WriteString(fmt.Sprintf("Step %d: %s\n", i, result))
			}
		}
		builder.WriteString("\n")
	}
	
	builder.WriteString("Please execute this step. ")
	
	if len(se.tools) > 0 {
		builder.WriteString("You can use the available tools if needed, or provide a direct answer.")
	} else {
		builder.WriteString("Provide your answer based on the information available.")
	}
	
	return builder.String()
}

// executeWithTool 使用指定工具执行。
func (se *StepExecutor) executeWithTool(
	ctx context.Context,
	step PlanStep,
	tool tools.Tool,
	messages []types.Message,
) (*AgentAction, error) {
	// 调用 LLM 生成工具输入
	response, err := se.llm.Invoke(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("step executor: LLM invoke failed: %w", err)
	}
	
	// 使用 LLM 的输出作为工具输入
	toolInput := map[string]any{
		"input": response.Content,
	}
	
	return &AgentAction{
		Type:      ActionToolCall,
		Tool:      tool.GetName(),
		ToolInput: toolInput,
		Log:       fmt.Sprintf("Using tool: %s", tool.GetName()),
	}, nil
}

// executeWithToolSelection 让 LLM 选择工具执行。
func (se *StepExecutor) executeWithToolSelection(
	ctx context.Context,
	step PlanStep,
	messages []types.Message,
) (*AgentAction, error) {
	// 转换工具为 types.Tool
	typesTools := make([]types.Tool, len(se.tools))
	for i, tool := range se.tools {
		typesTools[i] = tool.ToTypesTool()
	}
	
	// 绑定工具
	modelWithTools := se.llm.BindTools(typesTools)
	
	// 调用 LLM
	response, err := modelWithTools.Invoke(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("step executor: tool selection invoke failed: %w", err)
	}
	
	// 检查是否有工具调用
	if len(response.ToolCalls) > 0 {
		toolCall := response.ToolCalls[0]
		return &AgentAction{
			Type:      ActionToolCall,
			Tool:      toolCall.Function.Name,
			ToolInput: map[string]any{"input": toolCall.Function.Arguments},
			Log:       fmt.Sprintf("Tool selected: %s", toolCall.Function.Name),
		}, nil
	}
	
	// 没有工具调用，返回 LLM 的直接回答
	return &AgentAction{
		Type:      ActionToolCall,
		Tool:      "__llm_answer__", // 特殊标记
		ToolInput: map[string]any{"answer": response.Content},
		Log:       "No tool needed, direct answer",
	}, nil
}

// executeWithLLM 直接使用 LLM 执行（无工具）。
func (se *StepExecutor) executeWithLLM(
	ctx context.Context,
	messages []types.Message,
) (*AgentAction, error) {
	response, err := se.llm.Invoke(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("step executor: LLM invoke failed: %w", err)
	}
	
	// 返回特殊的工具调用，表示这是 LLM 的直接答案
	return &AgentAction{
		Type:      ActionToolCall,
		Tool:      "__llm_answer__",
		ToolInput: map[string]any{"answer": response.Content},
		Log:       "Direct LLM answer",
	}, nil
}

// findToolByName 根据名称查找工具。
func (se *StepExecutor) findToolByName(name string) tools.Tool {
	for _, tool := range se.tools {
		if tool.GetName() == name {
			return tool
		}
	}
	return nil
}

// getDefaultExecutorPrompt 返回默认的执行器提示词。
func getDefaultExecutorPrompt() string {
	return `You are a helpful assistant that executes specific steps of a plan.

Your job is to:
1. Understand the current step you need to execute
2. Use the available tools if appropriate
3. Use information from previous steps to inform your execution
4. Provide clear and accurate results

Execute the step to the best of your ability using the available information and tools.`
}
