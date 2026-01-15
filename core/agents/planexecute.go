package agents

import (
	"context"
	"fmt"
	"strings"
	
	"langchain-go/core/chat"
	"langchain-go/core/tools"
	"langchain-go/pkg/types"
)

// PlanAndExecuteAgent 是 Plan-and-Execute Agent。
//
// Plan-and-Execute Agent 先规划整体执行计划，然后逐步执行。
// 这种模式适合处理复杂、多步骤的任务。
//
// 工作流程：
//   1. Plan: 将复杂任务分解为多个步骤的计划
//   2. Execute: 逐步执行计划中的每个步骤
//   3. Replan (可选): 根据执行结果重新规划后续步骤
//
type PlanAndExecuteAgent struct {
	*BaseAgent
	planner  *Planner
	executor *StepExecutor
	config   PlanAndExecuteConfig
}

// PlanAndExecuteConfig 是 Plan-and-Execute Agent 配置。
type PlanAndExecuteConfig struct {
	// LLM 语言模型
	LLM chat.ChatModel
	
	// Tools 工具列表
	Tools []tools.Tool
	
	// PlannerPrompt 规划器提示词（可选）
	PlannerPrompt string
	
	// ExecutorPrompt 执行器提示词（可选）
	ExecutorPrompt string
	
	// EnableReplan 是否启用重新规划
	EnableReplan bool
	
	// MaxSteps 最大步数
	MaxSteps int
	
	// Verbose 是否输出详细日志
	Verbose bool
}

// NewPlanAndExecuteAgent 创建 Plan-and-Execute Agent。
//
// 参数：
//   - config: Agent 配置
//
// 返回：
//   - *PlanAndExecuteAgent: Agent 实例
//
func NewPlanAndExecuteAgent(config PlanAndExecuteConfig) *PlanAndExecuteAgent {
	if config.MaxSteps <= 0 {
		config.MaxSteps = 10
	}
	
	// 创建基础 Agent
	baseConfig := AgentConfig{
		Type:     "plan_and_execute",
		LLM:      config.LLM,
		Tools:    config.Tools,
		MaxSteps: config.MaxSteps,
		Verbose:  config.Verbose,
	}
	baseAgent := NewBaseAgent(baseConfig)
	
	// 创建 Planner
	plannerConfig := PlannerConfig{
		LLM:        config.LLM,
		Prompt:     config.PlannerPrompt,
		MaxSteps:   config.MaxSteps,
	}
	planner := NewPlanner(plannerConfig)
	
	// 创建 Step Executor
	executorConfig := StepExecutorConfig{
		LLM:     config.LLM,
		Tools:   config.Tools,
		Prompt:  config.ExecutorPrompt,
		Verbose: config.Verbose,
	}
	executor := NewStepExecutor(executorConfig)
	
	return &PlanAndExecuteAgent{
		BaseAgent: baseAgent,
		planner:   planner,
		executor:  executor,
		config:    config,
	}
}

// Plan 实现 Agent 接口。
//
// Plan-and-Execute Agent 的 Plan 方法返回一个包含完整计划的 AgentAction。
//
func (pea *PlanAndExecuteAgent) Plan(ctx context.Context, input string, history []AgentStep) (*AgentAction, error) {
	// 检查是否已经有计划
	if len(history) == 0 {
		// 第一步：创建计划
		plan, err := pea.planner.CreatePlan(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("plan-execute: failed to create plan: %w", err)
		}
		
		if pea.config.Verbose {
			fmt.Printf("\n[Plan Created]\n%s\n", pea.formatPlan(plan))
		}
		
		// 执行第一步
		if len(plan.Steps) == 0 {
			return &AgentAction{
				Type:        ActionFinish,
				FinalAnswer: "No steps to execute",
				Log:         "Plan is empty",
			}, nil
		}
		
		firstStep := plan.Steps[0]
		action, err := pea.executeStep(ctx, firstStep, input, nil)
		if err != nil {
			return nil, fmt.Errorf("plan-execute: failed to execute first step: %w", err)
		}
		
		// 在 action 的 Log 中保存完整计划
		action.Log = fmt.Sprintf("Plan:\n%s\n\nExecuting Step 1: %s\n%s",
			pea.formatPlan(plan), firstStep.Description, action.Log)
		
		return action, nil
	}
	
	// 从 history 中提取计划
	plan, err := pea.extractPlanFromHistory(history)
	if err != nil {
		return nil, fmt.Errorf("plan-execute: failed to extract plan: %w", err)
	}
	
	// 检查是否完成所有步骤
	currentStepIndex := len(history)
	if currentStepIndex >= len(plan.Steps) {
		// 所有步骤已完成，生成最终答案
		finalAnswer, err := pea.generateFinalAnswer(ctx, input, plan, history)
		if err != nil {
			return nil, fmt.Errorf("plan-execute: failed to generate final answer: %w", err)
		}
		
		return &AgentAction{
			Type:        ActionFinish,
			FinalAnswer: finalAnswer,
			Log:         fmt.Sprintf("All %d steps completed", len(plan.Steps)),
		}, nil
	}
	
	// 检查是否需要重新规划
	if pea.config.EnableReplan && pea.shouldReplan(history) {
		// 重新规划
		plan, err = pea.planner.Replan(ctx, input, plan, history)
		if err != nil {
			return nil, fmt.Errorf("plan-execute: failed to replan: %w", err)
		}
		
		if pea.config.Verbose {
			fmt.Printf("\n[Plan Updated]\n%s\n", pea.formatPlan(plan))
		}
	}
	
	// 执行下一步
	nextStep := plan.Steps[currentStepIndex]
	previousResults := pea.extractPreviousResults(history)
	
	action, err := pea.executeStep(ctx, nextStep, input, previousResults)
	if err != nil {
		return nil, fmt.Errorf("plan-execute: failed to execute step %d: %w", currentStepIndex+1, err)
	}
	
	action.Log = fmt.Sprintf("Executing Step %d: %s\n%s",
		currentStepIndex+1, nextStep.Description, action.Log)
	
	return action, nil
}

// executeStep 执行单个步骤。
func (pea *PlanAndExecuteAgent) executeStep(
	ctx context.Context,
	step PlanStep,
	originalInput string,
	previousResults map[string]string,
) (*AgentAction, error) {
	return pea.executor.ExecuteStep(ctx, step, originalInput, previousResults)
}

// extractPlanFromHistory 从历史中提取计划。
func (pea *PlanAndExecuteAgent) extractPlanFromHistory(history []AgentStep) (*Plan, error) {
	if len(history) == 0 {
		return nil, fmt.Errorf("history is empty")
	}
	
	// 简单的提取逻辑：解析 Log 中的 Plan 部分
	// 实际项目中可能需要更复杂的状态管理
	
	// 创建一个简化的计划（基于历史步骤数量）
	plan := &Plan{
		Steps: make([]PlanStep, pea.config.MaxSteps),
	}
	
	// 从 firstLog 中提取步骤数量
	// 这里简化处理，假设每个历史步骤对应一个计划步骤
	for i := range history {
		plan.Steps[i] = PlanStep{
			ID:          fmt.Sprintf("step_%d", i+1),
			Description: fmt.Sprintf("Step %d", i+1),
			Dependencies: []string{},
		}
	}
	
	// 添加剩余步骤
	for i := len(history); i < pea.config.MaxSteps; i++ {
		plan.Steps[i] = PlanStep{
			ID:          fmt.Sprintf("step_%d", i+1),
			Description: fmt.Sprintf("Step %d", i+1),
			Dependencies: []string{},
		}
	}
	
	return plan, nil
}

// extractPreviousResults 从历史中提取之前步骤的结果。
func (pea *PlanAndExecuteAgent) extractPreviousResults(history []AgentStep) map[string]string {
	results := make(map[string]string)
	
	for i, step := range history {
		stepID := fmt.Sprintf("step_%d", i+1)
		results[stepID] = step.Observation
	}
	
	return results
}

// shouldReplan 判断是否应该重新规划。
func (pea *PlanAndExecuteAgent) shouldReplan(history []AgentStep) bool {
	if len(history) == 0 {
		return false
	}
	
	// 检查最后一步是否有错误
	lastStep := history[len(history)-1]
	if lastStep.Error != nil {
		return true
	}
	
	// 检查是否有明确的失败信号
	if strings.Contains(strings.ToLower(lastStep.Observation), "failed") ||
		strings.Contains(strings.ToLower(lastStep.Observation), "error") {
		return true
	}
	
	return false
}

// generateFinalAnswer 生成最终答案。
func (pea *PlanAndExecuteAgent) generateFinalAnswer(
	ctx context.Context,
	input string,
	plan *Plan,
	history []AgentStep,
) (string, error) {
	// 构建提示词
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("Original Question: %s\n\n", input))
	builder.WriteString("Execution Results:\n")
	
	for i, step := range history {
		builder.WriteString(fmt.Sprintf("\nStep %d: %s\n", i+1, plan.Steps[i].Description))
		builder.WriteString(fmt.Sprintf("Result: %s\n", step.Observation))
	}
	
	builder.WriteString("\nBased on the above execution results, please provide a comprehensive final answer to the original question.")
	
	// 调用 LLM 生成最终答案
	messages := []types.Message{
		types.NewSystemMessage("You are a helpful assistant that summarizes task execution results."),
		types.NewUserMessage(builder.String()),
	}
	
	response, err := pea.config.LLM.Invoke(ctx, messages)
	if err != nil {
		return "", fmt.Errorf("failed to generate final answer: %w", err)
	}
	
	return response.Content, nil
}

// formatPlan 格式化计划为字符串。
func (pea *PlanAndExecuteAgent) formatPlan(plan *Plan) string {
	var builder strings.Builder
	
	for i, step := range plan.Steps {
		builder.WriteString(fmt.Sprintf("%d. %s", i+1, step.Description))
		if len(step.Dependencies) > 0 {
			builder.WriteString(fmt.Sprintf(" (depends on: %s)", strings.Join(step.Dependencies, ", ")))
		}
		builder.WriteString("\n")
	}
	
	return builder.String()
}

// GetType 返回 Agent 类型。
func (pea *PlanAndExecuteAgent) GetType() AgentType {
	return "plan_and_execute"
}
