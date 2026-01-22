package agents

import (
	"context"
	"fmt"
	"time"
	
	"github.com/zhucl121/langchain-go/core/middleware"
	"github.com/zhucl121/langchain-go/core/tools"
)

// Executor 是 Agent 执行器。
//
// Executor 管理 Agent 的执行循环，包括：
//   - 思考-行动-观察循环
//   - 工具调用
//   - 最大步数控制
//   - 中间件集成
//   - Skill 集成
//
type Executor struct {
	agent           Agent
	maxSteps        int
	verbose         bool
	middlewareChain *middleware.Chain
	skillManager    SkillManager
	enabledSkills   []string
}

// NewExecutor 创建 Agent 执行器。
//
// 参数：
//   - agent: Agent 实例
//
// 返回：
//   - *Executor: 执行器实例
//
func NewExecutor(agent Agent) *Executor {
	return &Executor{
		agent:           agent,
		maxSteps:        10,
		verbose:         false,
		middlewareChain: middleware.NewChain(),
	}
}

// WithSkillManager 设置 Skill 管理器。
func (e *Executor) WithSkillManager(manager SkillManager) *Executor {
	e.skillManager = manager
	return e
}

// WithEnabledSkills 设置启用的 Skill 列表。
func (e *Executor) WithEnabledSkills(skills []string) *Executor {
	e.enabledSkills = skills
	return e
}

// WithMaxSteps 设置最大步数。
func (e *Executor) WithMaxSteps(maxSteps int) *Executor {
	e.maxSteps = maxSteps
	return e
}

// WithVerbose 设置是否输出详细日志。
func (e *Executor) WithVerbose(verbose bool) *Executor {
	e.verbose = verbose
	return e
}

// WithMiddleware 添加中间件。
func (e *Executor) WithMiddleware(mw middleware.Middleware) *Executor {
	e.middlewareChain.Use(mw)
	return e
}

// Execute 执行 Agent。
//
// 参数：
//   - ctx: 上下文
//   - input: 输入问题或任务
//
// 返回：
//   - *AgentResult: 执行结果
//   - error: 错误
//
func (e *Executor) Execute(ctx context.Context, input string) (*AgentResult, error) {
	// 初始化 Skills
	if err := e.initializeSkills(ctx); err != nil {
		return nil, fmt.Errorf("failed to initialize skills: %w", err)
	}

	result := &AgentResult{
		Steps:      make([]AgentStep, 0),
		TotalSteps: 0,
		Success:    false,
	}

	history := make([]AgentStep, 0)

	// 执行循环
	for step := 0; step < e.maxSteps; step++ {
		result.TotalSteps = step + 1

		if e.verbose {
			fmt.Printf("\n[Step %d]\n", step+1)
		}

		// 使用中间件包装 Plan 调用
		var action *AgentAction
		var err error

		if e.middlewareChain.Len() > 0 {
			// 通过中间件执行
			planResult, planErr := e.middlewareChain.Execute(ctx, input, func(ctx context.Context, in any) (any, error) {
				return e.agent.Plan(ctx, input, history)
			})

			if planErr != nil {
				err = planErr
			} else {
				action, _ = planResult.(*AgentAction)
			}
		} else {
			// 直接执行
			action, err = e.agent.Plan(ctx, input, history)
		}

		if err != nil {
			result.Error = err
			return result, fmt.Errorf("executor: plan failed at step %d: %w", step+1, err)
		}

		if e.verbose {
			fmt.Printf("Action: %+v\n", action)
		}

		// 检查行动类型
		switch action.Type {
		case ActionFinish:
			// 任务完成
			result.Output = action.FinalAnswer
			result.Success = true
			return result, nil

		case ActionToolCall:
			// 执行工具调用
			observation, toolErr := e.executeToolCall(ctx, action)

			currentStep := AgentStep{
				Action:      action,
				Observation: observation,
				Error:       toolErr,
			}

			result.Steps = append(result.Steps, currentStep)
			history = append(history, currentStep)

			if e.verbose {
				fmt.Printf("Observation: %s\n", observation)
				if toolErr != nil {
					fmt.Printf("Error: %v\n", toolErr)
				}
			}

		case ActionError:
			result.Error = fmt.Errorf("agent returned error action")
			return result, result.Error

		default:
			result.Error = fmt.Errorf("unknown action type: %s", action.Type)
			return result, result.Error
		}
	}

	// 达到最大步数
	result.Error = ErrAgentMaxSteps
	return result, ErrAgentMaxSteps
}

// executeToolCall 执行工具调用。
func (e *Executor) executeToolCall(ctx context.Context, action *AgentAction) (string, error) {
	// 获取工具
	tool, err := e.getToolByName(action.Tool)
	if err != nil {
		return "", err
	}

	// 执行工具
	toolResult, err := tool.Execute(ctx, action.ToolInput)
	if err != nil {
		return "", fmt.Errorf("tool execution failed: %w", err)
	}

	// 转换结果为字符串
	observation := fmt.Sprintf("%v", toolResult)

	return observation, nil
}

// getToolByName 根据名称获取工具。
func (e *Executor) getToolByName(name string) (tools.Tool, error) {
	// 获取所有可用工具（Agent 工具 + Skill 工具）
	allTools := e.getAllTools()
	for _, tool := range allTools {
		if tool.GetName() == name {
			return tool, nil
		}
	}
	return nil, fmt.Errorf("%w: %s", ErrAgentNoTool, name)
}

// Stream 流式执行 Agent（基础版本）。
//
// 参数：
//   - ctx: 上下文
//   - input: 输入
//   - callback: 步骤回调函数
//
// 返回：
//   - *AgentResult: 最终结果
//   - error: 错误
//
func (e *Executor) Stream(
	ctx context.Context,
	input string,
	callback func(step AgentStep) error,
) (*AgentResult, error) {
	result := &AgentResult{
		Steps:      make([]AgentStep, 0),
		TotalSteps: 0,
		Success:    false,
	}

	history := make([]AgentStep, 0)

	for step := 0; step < e.maxSteps; step++ {
		result.TotalSteps = step + 1

		// 规划
		action, err := e.agent.Plan(ctx, input, history)
		if err != nil {
			result.Error = err
			return result, err
		}

		// 检查完成
		if action.Type == ActionFinish {
			result.Output = action.FinalAnswer
			result.Success = true
			return result, nil
		}

		// 执行工具
		if action.Type == ActionToolCall {
			observation, toolErr := e.executeToolCall(ctx, action)

			currentStep := AgentStep{
				Action:      action,
				Observation: observation,
				Error:       toolErr,
			}

			result.Steps = append(result.Steps, currentStep)
			history = append(history, currentStep)

			// 调用回调
			if callback != nil {
				if err := callback(currentStep); err != nil {
					return result, fmt.Errorf("executor: callback failed: %w", err)
				}
			}
		}
	}

	result.Error = ErrAgentMaxSteps
	return result, ErrAgentMaxSteps
}

// Batch 批量执行 Agent。
//
// 参数：
//   - ctx: 上下文
//   - inputs: 输入列表
//
// 返回：
//   - []*AgentResult: 结果列表
//   - error: 错误
//
func (e *Executor) Batch(ctx context.Context, inputs []string) ([]*AgentResult, error) {
	results := make([]*AgentResult, len(inputs))

	for i, input := range inputs {
		result, err := e.Execute(ctx, input)
		if err != nil {
			// 记录错误但继续执行其他
			result = &AgentResult{
				Success: false,
				Error:   err,
			}
		}
		results[i] = result
	}

	return results, nil
}

// GetAgent 返回 Agent。
func (e *Executor) GetAgent() Agent {
	return e.agent
}

// GetMiddlewareChain 返回中间件链。
func (e *Executor) GetMiddlewareChain() *middleware.Chain {
	return e.middlewareChain
}

// AgentExecutor 是新的 Agent 执行器（对标 Python AgentExecutor）。
//
// 提供更完整的功能，包括：
//   - 工具执行器集成
//   - 错误处理和重试
//   - 流式输出
//   - 批量处理
//
type AgentExecutor struct {
	agent        Agent
	toolExecutor *tools.ToolExecutor
	maxSteps     int
	verbose      bool
	middleware   *middleware.Chain
}

// AgentExecutorConfig 是 AgentExecutor 配置。
type AgentExecutorConfig struct {
	// Agent Agent 实例
	Agent Agent

	// ToolExecutor 工具执行器
	ToolExecutor *tools.ToolExecutor

	// MaxSteps 最大步数
	MaxSteps int

	// Verbose 是否输出详细日志
	Verbose bool

	// Middlewares 中间件列表
	Middlewares []middleware.Middleware
}

// NewAgentExecutor 创建 AgentExecutor。
//
// 参数：
//   - config: 执行器配置
//
// 返回：
//   - *AgentExecutor: 执行器实例
//
func NewAgentExecutor(config AgentExecutorConfig) *AgentExecutor {
	if config.MaxSteps <= 0 {
		config.MaxSteps = 10
	}

	chain := middleware.NewChain()
	for _, mw := range config.Middlewares {
		chain.Use(mw)
	}

	return &AgentExecutor{
		agent:        config.Agent,
		toolExecutor: config.ToolExecutor,
		maxSteps:     config.MaxSteps,
		verbose:      config.Verbose,
		middleware:   chain,
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
func (ae *AgentExecutor) Run(ctx context.Context, input string) (*AgentResult, error) {
	result := &AgentResult{
		Steps:      make([]AgentStep, 0),
		TotalSteps: 0,
		Success:    false,
	}

	history := make([]AgentStep, 0)

	for step := 0; step < ae.maxSteps; step++ {
		result.TotalSteps = step + 1

		if ae.verbose {
			fmt.Printf("\n[Step %d]\n", step+1)
		}

		// 规划下一步
		action, err := ae.agent.Plan(ctx, input, history)
		if err != nil {
			result.Error = err
			return result, fmt.Errorf("agent executor: plan failed at step %d: %w", step+1, err)
		}

		if ae.verbose {
			fmt.Printf("Action: %+v\n", action)
		}

		// 检查是否完成
		if action.Type == ActionFinish {
			result.Output = action.FinalAnswer
			result.Success = true
			return result, nil
		}

		// 执行工具调用
		if action.Type == ActionToolCall {
			observation, toolErr := ae.executeToolWithExecutor(ctx, action)

			currentStep := AgentStep{
				Action:      action,
				Observation: observation,
				Error:       toolErr,
			}

			result.Steps = append(result.Steps, currentStep)
			history = append(history, currentStep)

			if ae.verbose {
				fmt.Printf("Observation: %s\n", observation)
				if toolErr != nil {
					fmt.Printf("Error: %v\n", toolErr)
				}
			}
		}
	}

	// 达到最大步数
	result.Error = ErrAgentMaxSteps
	return result, ErrAgentMaxSteps
}

// executeToolWithExecutor 使用 ToolExecutor 执行工具。
func (ae *AgentExecutor) executeToolWithExecutor(ctx context.Context, action *AgentAction) (string, error) {
	if ae.toolExecutor == nil {
		return "", fmt.Errorf("agent executor: tool executor is nil")
	}

	// 执行工具
	toolResult, err := ae.toolExecutor.Execute(ctx, action.Tool, action.ToolInput)
	if err != nil {
		return "", fmt.Errorf("tool execution failed: %w", err)
	}

	// 转换结果为字符串
	observation := fmt.Sprintf("%v", toolResult)

	return observation, nil
}

// AgentStreamEvent 是 Agent 流式事件。
type AgentStreamEvent struct {
	// Type 事件类型
	Type AgentStreamEventType

	// Step 当前步骤
	Step int

	// Action 当前行动
	Action *AgentAction

	// Observation 观察结果
	Observation string

	// Error 错误
	Error error

	// Timestamp 时间戳
	Timestamp time.Time
}

// AgentStreamEventType 是流式事件类型。
type AgentStreamEventType string

const (
	// EventTypeStart 开始执行
	EventTypeStart AgentStreamEventType = "start"

	// EventTypeStep 执行步骤
	EventTypeStep AgentStreamEventType = "step"

	// EventTypeToolCall 工具调用
	EventTypeToolCall AgentStreamEventType = "tool_call"

	// EventTypeToolResult 工具结果
	EventTypeToolResult AgentStreamEventType = "tool_result"

	// EventTypeFinish 执行完成
	EventTypeFinish AgentStreamEventType = "finish"

	// EventTypeError 执行错误
	EventTypeError AgentStreamEventType = "error"
)

// Stream 流式执行 Agent。
//
// 参数：
//   - ctx: 上下文
//   - input: 输入问题
//
// 返回：
//   - <-chan AgentStreamEvent: 事件流
//
func (ae *AgentExecutor) Stream(ctx context.Context, input string) <-chan AgentStreamEvent {
	eventChan := make(chan AgentStreamEvent, 10)

	go func() {
		defer close(eventChan)

		// 发送开始事件
		eventChan <- AgentStreamEvent{
			Type:      EventTypeStart,
			Timestamp: time.Now(),
		}

		history := make([]AgentStep, 0)

		for step := 0; step < ae.maxSteps; step++ {
			// 发送步骤事件
			eventChan <- AgentStreamEvent{
				Type:      EventTypeStep,
				Step:      step + 1,
				Timestamp: time.Now(),
			}

			// 规划
			action, err := ae.agent.Plan(ctx, input, history)
			if err != nil {
				eventChan <- AgentStreamEvent{
					Type:      EventTypeError,
					Error:     err,
					Timestamp: time.Now(),
				}
				return
			}

			// 检查完成
			if action.Type == ActionFinish {
				eventChan <- AgentStreamEvent{
					Type:        EventTypeFinish,
					Action:      action,
					Observation: action.FinalAnswer,
					Timestamp:   time.Now(),
				}
				return
			}

			// 执行工具
			if action.Type == ActionToolCall {
				// 发送工具调用事件
				eventChan <- AgentStreamEvent{
					Type:      EventTypeToolCall,
					Step:      step + 1,
					Action:    action,
					Timestamp: time.Now(),
				}

				observation, toolErr := ae.executeToolWithExecutor(ctx, action)

				// 发送工具结果事件
				eventChan <- AgentStreamEvent{
					Type:        EventTypeToolResult,
					Step:        step + 1,
					Action:      action,
					Observation: observation,
					Error:       toolErr,
					Timestamp:   time.Now(),
				}

				history = append(history, AgentStep{
					Action:      action,
					Observation: observation,
					Error:       toolErr,
				})
			}
		}

		// 达到最大步数
		eventChan <- AgentStreamEvent{
			Type:      EventTypeError,
			Error:     ErrAgentMaxSteps,
			Timestamp: time.Now(),
		}
	}()

	return eventChan
}

// getAllTools 获取所有可用工具（Agent 工具 + Skill 工具）
func (e *Executor) getAllTools() []tools.Tool {
	// 获取 Agent 的基础工具
	allTools := e.agent.GetTools()

	// 如果没有 SkillManager，直接返回基础工具
	if e.skillManager == nil {
		return allTools
	}

	// 获取所有已加载的 Skill
	loadedSkills := e.skillManager.ListLoaded()

	// 聚合 Skill 提供的工具
	for _, skill := range loadedSkills {
		skillTools := skill.GetTools()
		allTools = append(allTools, skillTools...)
	}

	return allTools
}

// getSystemPromptWithSkills 获取组合后的系统提示词（基础提示词 + Skill 提示词）
func (e *Executor) getSystemPromptWithSkills(basePrompt string) string {
	// 如果没有 SkillManager，返回基础提示词
	if e.skillManager == nil {
		return basePrompt
	}

	// 获取所有已加载的 Skill
	loadedSkills := e.skillManager.ListLoaded()
	if len(loadedSkills) == 0 {
		return basePrompt
	}

	// 组合提示词
	prompts := []string{basePrompt}

	for _, skill := range loadedSkills {
		skillPrompt := skill.GetSystemPrompt()
		if skillPrompt != "" {
			prompts = append(prompts, fmt.Sprintf("\n## %s\n%s", skill.ID(), skillPrompt))
		}
	}

	// 将 []string 转换为 []any
	promptsAny := make([]any, len(prompts))
	for i, p := range prompts {
		promptsAny[i] = p
	}
	return fmt.Sprintf("%s\n", fmt.Sprint(promptsAny...))
}

// initializeSkills 初始化 Skills（在执行前调用）
func (e *Executor) initializeSkills(ctx context.Context) error {
	if e.skillManager == nil || len(e.enabledSkills) == 0 {
		return nil
	}

	// 加载启用的 Skills
	for _, skillID := range e.enabledSkills {
		// 检查是否已加载
		if e.skillManager.IsLoaded(skillID) {
			continue
		}

		// 加载 Skill
		config := map[string]any{
			"auto_load_dependencies": true,
		}
		if err := e.skillManager.Load(ctx, skillID, config); err != nil {
			return fmt.Errorf("failed to load skill %s: %w", skillID, err)
		}

		if e.verbose {
			fmt.Printf("[Skill] Loaded skill: %s\n", skillID)
		}
	}

	return nil
}
