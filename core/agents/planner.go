package agents

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	
	"langchain-go/core/chat"
	"langchain-go/pkg/types"
)

// Planner 是任务规划器。
//
// Planner 负责将复杂任务分解为多个可执行的步骤。
//
type Planner struct {
	llm      chat.ChatModel
	prompt   string
	maxSteps int
}

// PlannerConfig 是 Planner 配置。
type PlannerConfig struct {
	// LLM 语言模型
	LLM chat.ChatModel
	
	// Prompt 规划提示词（可选）
	Prompt string
	
	// MaxSteps 最大步骤数
	MaxSteps int
}

// Plan 是执行计划。
type Plan struct {
	// Steps 步骤列表
	Steps []PlanStep
	
	// OriginalInput 原始输入
	OriginalInput string
}

// PlanStep 是计划中的单个步骤。
type PlanStep struct {
	// ID 步骤ID
	ID string
	
	// Description 步骤描述
	Description string
	
	// Dependencies 依赖的步骤ID列表
	Dependencies []string
	
	// ToolName 建议使用的工具名称（可选）
	ToolName string
}

// NewPlanner 创建规划器。
func NewPlanner(config PlannerConfig) *Planner {
	prompt := config.Prompt
	if prompt == "" {
		prompt = getDefaultPlannerPrompt()
	}
	
	maxSteps := config.MaxSteps
	if maxSteps <= 0 {
		maxSteps = 10
	}
	
	return &Planner{
		llm:      config.LLM,
		prompt:   prompt,
		maxSteps: maxSteps,
	}
}

// CreatePlan 创建执行计划。
//
// 参数：
//   - ctx: 上下文
//   - input: 任务描述
//
// 返回：
//   - *Plan: 执行计划
//   - error: 错误
//
func (p *Planner) CreatePlan(ctx context.Context, input string) (*Plan, error) {
	// 构建提示词
	prompt := fmt.Sprintf("%s\n\nTask: %s\n\nPlease create a step-by-step plan to complete this task.", p.prompt, input)
	
	messages := []types.Message{
		types.NewSystemMessage("You are an expert task planner."),
		types.NewUserMessage(prompt),
	}
	
	// 调用 LLM
	response, err := p.llm.Invoke(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("planner: LLM invoke failed: %w", err)
	}
	
	// 解析计划
	plan, err := p.parsePlan(response.Content, input)
	if err != nil {
		return nil, fmt.Errorf("planner: failed to parse plan: %w", err)
	}
	
	return plan, nil
}

// Replan 重新规划。
//
// 参数：
//   - ctx: 上下文
//   - input: 原始任务
//   - currentPlan: 当前计划
//   - history: 执行历史
//
// 返回：
//   - *Plan: 新的执行计划
//   - error: 错误
//
func (p *Planner) Replan(
	ctx context.Context,
	input string,
	currentPlan *Plan,
	history []AgentStep,
) (*Plan, error) {
	// 构建重新规划的提示词
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("Original Task: %s\n\n", input))
	builder.WriteString("Current Plan:\n")
	
	for i, step := range currentPlan.Steps {
		builder.WriteString(fmt.Sprintf("%d. %s\n", i+1, step.Description))
	}
	
	builder.WriteString("\nExecution History:\n")
	for i, step := range history {
		builder.WriteString(fmt.Sprintf("Step %d: %s\n", i+1, step.Action.Log))
		builder.WriteString(fmt.Sprintf("Result: %s\n", step.Observation))
		if step.Error != nil {
			builder.WriteString(fmt.Sprintf("Error: %s\n", step.Error.Error()))
		}
		builder.WriteString("\n")
	}
	
	builder.WriteString("Based on the execution history, please create a revised plan to complete the remaining task. " +
		"Keep the completed steps and adjust the remaining steps as needed.")
	
	messages := []types.Message{
		types.NewSystemMessage("You are an expert task planner who can adapt plans based on execution results."),
		types.NewUserMessage(builder.String()),
	}
	
	// 调用 LLM
	response, err := p.llm.Invoke(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("planner: replan LLM invoke failed: %w", err)
	}
	
	// 解析新计划
	plan, err := p.parsePlan(response.Content, input)
	if err != nil {
		return nil, fmt.Errorf("planner: failed to parse revised plan: %w", err)
	}
	
	return plan, nil
}

// parsePlan 解析 LLM 输出的计划。
func (p *Planner) parsePlan(output string, originalInput string) (*Plan, error) {
	plan := &Plan{
		Steps:         make([]PlanStep, 0),
		OriginalInput: originalInput,
	}
	
	// 使用正则表达式提取步骤
	// 支持格式：
	// 1. Step description
	// Step 1: Step description
	// - Step description
	
	lines := strings.Split(output, "\n")
	stepCounter := 0
	
	// 正则表达式匹配各种步骤格式
	stepRegexes := []*regexp.Regexp{
		regexp.MustCompile(`^\s*(\d+)\.\s+(.+)$`),                    // 1. Description
		regexp.MustCompile(`^[Ss]tep\s+(\d+):\s*(.+)$`),            // Step 1: Description
		regexp.MustCompile(`^\s*-\s+(.+)$`),                         // - Description
		regexp.MustCompile(`^\s*\*\s+(.+)$`),                        // * Description
	}
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		
		// 尝试匹配各种格式
		matched := false
		var description string
		
		for _, re := range stepRegexes {
			matches := re.FindStringSubmatch(line)
			if len(matches) > 0 {
				matched = true
				if len(matches) == 3 {
					// 格式 1 和 2: 有编号
					description = strings.TrimSpace(matches[2])
				} else if len(matches) == 2 {
					// 格式 3 和 4: 无编号
					description = strings.TrimSpace(matches[1])
				}
				break
			}
		}
		
		if matched && description != "" {
			stepCounter++
			if stepCounter > p.maxSteps {
				break
			}
			
			step := PlanStep{
				ID:           fmt.Sprintf("step_%d", stepCounter),
				Description:  description,
				Dependencies: []string{},
			}
			
			// 检查是否有依赖关系描述
			if strings.Contains(strings.ToLower(description), "after") ||
				strings.Contains(strings.ToLower(description), "depends on") {
				// 简单处理：前一步是依赖
				if stepCounter > 1 {
					step.Dependencies = append(step.Dependencies, fmt.Sprintf("step_%d", stepCounter-1))
				}
			}
			
			// 尝试提取工具名称（如果有）
			toolMatch := regexp.MustCompile(`using\s+([A-Za-z_]+)`).FindStringSubmatch(description)
			if len(toolMatch) > 1 {
				step.ToolName = toolMatch[1]
			}
			
			plan.Steps = append(plan.Steps, step)
		}
	}
	
	// 如果没有解析到步骤，创建一个默认步骤
	if len(plan.Steps) == 0 {
		// 尝试将整个输出作为一个步骤
		if len(output) > 0 {
			plan.Steps = append(plan.Steps, PlanStep{
				ID:           "step_1",
				Description:  strings.TrimSpace(output),
				Dependencies: []string{},
			})
		} else {
			return nil, fmt.Errorf("failed to parse any steps from plan output")
		}
	}
	
	return plan, nil
}

// getDefaultPlannerPrompt 返回默认的规划提示词。
func getDefaultPlannerPrompt() string {
	return `You are an expert task planner. Your job is to break down complex tasks into simple, actionable steps.

For the given task, create a numbered step-by-step plan. Each step should:
1. Be clear and specific
2. Be executable with available tools or knowledge
3. Build on previous steps logically
4. Move towards completing the overall task

Format your plan as a numbered list:
1. First step description
2. Second step description
3. Third step description
...

Keep the plan concise (typically 3-7 steps) but comprehensive enough to complete the task.`
}
