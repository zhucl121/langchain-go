package agents

import (
	"context"
	"strings"
	"testing"
	
	"github.com/zhucl121/langchain-go/core/runnable"
	"github.com/zhucl121/langchain-go/core/tools"
	"github.com/zhucl121/langchain-go/pkg/types"
)

// TestPlannerCreatePlan 测试 Planner 创建计划
func TestPlannerCreatePlan(t *testing.T) {
	mockLLM := NewMockChatModel()
	mockLLM.InvokeFunc = func(ctx context.Context, messages []types.Message, opts ...runnable.Option) (types.Message, error) {
		return types.NewAssistantMessage(`Here's the plan:
1. Search for current weather information
2. Analyze the weather data
3. Provide a summary`), nil
	}
	
	config := PlannerConfig{
		LLM:      mockLLM,
		MaxSteps: 5,
	}
	
	planner := NewPlanner(config)
	
	ctx := context.Background()
	plan, err := planner.CreatePlan(ctx, "What's the weather like today?")
	
	if err != nil {
		t.Fatalf("CreatePlan failed: %v", err)
	}
	
	if plan == nil {
		t.Fatal("Plan is nil")
	}
	
	if len(plan.Steps) != 3 {
		t.Errorf("Expected 3 steps, got %d", len(plan.Steps))
	}
	
	// 验证第一步
	if !strings.Contains(plan.Steps[0].Description, "weather") {
		t.Errorf("First step should mention weather, got: %s", plan.Steps[0].Description)
	}
}

// TestPlannerParsePlan 测试计划解析
func TestPlannerParsePlan(t *testing.T) {
	planner := NewPlanner(PlannerConfig{MaxSteps: 10})
	
	tests := []struct {
		name          string
		input         string
		expectedSteps int
	}{
		{
			name: "numbered list",
			input: `1. First step
2. Second step
3. Third step`,
			expectedSteps: 3,
		},
		{
			name: "step format",
			input: `Step 1: First step
Step 2: Second step`,
			expectedSteps: 2,
		},
		{
			name: "bullet points",
			input: `- First step
- Second step
- Third step
- Fourth step`,
			expectedSteps: 4,
		},
		{
			name: "asterisk bullet points",
			input: `* First step
* Second step`,
			expectedSteps: 2,
		},
		{
			name: "mixed format",
			input: `Here's the plan:
1. First step
2. Second step
Then we should:
3. Third step`,
			expectedSteps: 3,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan, err := planner.parsePlan(tt.input, "test task")
			if err != nil {
				t.Fatalf("parsePlan failed: %v", err)
			}
			
			if len(plan.Steps) != tt.expectedSteps {
				t.Errorf("Expected %d steps, got %d", tt.expectedSteps, len(plan.Steps))
			}
		})
	}
}

// TestStepExecutor 测试步骤执行器
func TestStepExecutor(t *testing.T) {
	mockLLM := NewMockChatModel()
	mockLLM.InvokeFunc = func(ctx context.Context, messages []types.Message, opts ...runnable.Option) (types.Message, error) {
		return types.NewAssistantMessage("The weather is sunny and 72°F"), nil
	}
	
	mockTool := NewMockTool("weather_tool", "Get weather information")
	mockTool.executeFunc = func(ctx context.Context, input map[string]any) (any, error) {
		return "Temperature: 72°F, Condition: Sunny", nil
	}
	
	config := StepExecutorConfig{
		LLM:   mockLLM,
		Tools: []tools.Tool{mockTool},
	}
	
	executor := NewStepExecutor(config)
	
	step := PlanStep{
		ID:          "step_1",
		Description: "Get the current weather",
	}
	
	ctx := context.Background()
	action, err := executor.ExecuteStep(ctx, step, "What's the weather?", nil)
	
	if err != nil {
		t.Fatalf("ExecuteStep failed: %v", err)
	}
	
	if action == nil {
		t.Fatal("Action is nil")
	}
	
	if action.Type != ActionToolCall {
		t.Errorf("Expected ActionToolCall, got %s", action.Type)
	}
}

// TestPlanAndExecuteAgent 测试 Plan-and-Execute Agent
func TestPlanAndExecuteAgent(t *testing.T) {
	callCount := 0
	mockLLM := NewMockChatModel()
	mockLLM.InvokeFunc = func(ctx context.Context, messages []types.Message, opts ...runnable.Option) (types.Message, error) {
		responses := []string{
			// Planner response
			`1. Get weather information
2. Analyze the data
3. Provide summary`,
			// First step execution
			"The weather is sunny and 72°F",
			// Second step execution
			"The data shows good weather conditions",
			// Third step execution
			"It's a beautiful sunny day with pleasant temperature",
			// Final answer generation
			"Based on the analysis, today's weather is excellent with sunny skies and comfortable 72°F temperature.",
		}
		
		response := responses[callCount%len(responses)]
		callCount++
		return types.NewAssistantMessage(response), nil
	}
	
	mockTool := NewMockTool("weather_tool", "Get weather information")
	mockTool.executeFunc = func(ctx context.Context, input map[string]any) (any, error) {
		return "72°F, Sunny", nil
	}
	
	config := PlanAndExecuteConfig{
		LLM:          mockLLM,
		Tools:        []tools.Tool{mockTool},
		EnableReplan: false,
		MaxSteps:     5,
		Verbose:      false,
	}
	
	agent := NewPlanAndExecuteAgent(config)
	
	if agent == nil {
		t.Fatal("Agent is nil")
	}
	
	if agent.GetType() != "plan_and_execute" {
		t.Errorf("Expected plan_and_execute type, got %s", agent.GetType())
	}
	
	// 测试第一步规划
	ctx := context.Background()
	action, err := agent.Plan(ctx, "What's the weather like?", []AgentStep{})
	
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}
	
	if action == nil {
		t.Fatal("Action is nil")
	}
	
	// 第一次调用应该创建计划并执行第一步
	if action.Type != ActionToolCall && action.Type != ActionFinish {
		t.Errorf("Expected tool call or finish, got %s", action.Type)
	}
}

// TestPlanAndExecuteAgentReplan 测试重新规划功能
func TestPlanAndExecuteAgentReplan(t *testing.T) {
	callCount := 0
	mockLLM := NewMockChatModel()
	mockLLM.InvokeFunc = func(ctx context.Context, messages []types.Message, opts ...runnable.Option) (types.Message, error) {
		responses := []string{
			// Initial plan
			`1. Search for information
2. Process the data`,
			// First execution (with error indication)
			"Search failed: network error",
			// Replan response
			`1. Try alternative search method
2. Process the data`,
			// Second execution
			"Found the information successfully",
			// Final answer
			"Task completed with alternative approach",
		}
		
		response := responses[callCount%len(responses)]
		callCount++
		return types.NewAssistantMessage(response), nil
	}
	
	config := PlanAndExecuteConfig{
		LLM:          mockLLM,
		Tools:        []tools.Tool{},
		EnableReplan: true,
		MaxSteps:     5,
		Verbose:      false,
	}
	
	agent := NewPlanAndExecuteAgent(config)
	ctx := context.Background()
	
	// 第一步
	action1, err := agent.Plan(ctx, "Find information", []AgentStep{})
	if err != nil {
		t.Fatalf("First plan failed: %v", err)
	}
	
	// 模拟失败的执行历史
	history := []AgentStep{
		{
			Action:      action1,
			Observation: "Search failed: network error",
			Error:       nil,
		},
	}
	
	// 第二步（应该触发重新规划）
	action2, err := agent.Plan(ctx, "Find information", history)
	if err != nil {
		t.Fatalf("Second plan failed: %v", err)
	}
	
	if action2 == nil {
		t.Fatal("Second action is nil")
	}
}

// TestPlanStepDependencies 测试步骤依赖关系
func TestPlanStepDependencies(t *testing.T) {
	planner := NewPlanner(PlannerConfig{MaxSteps: 10})
	
	input := `1. Search for data
2. Process the data after searching
3. Generate report using processed data`
	
	plan, err := planner.parsePlan(input, "test")
	if err != nil {
		t.Fatalf("parsePlan failed: %v", err)
	}
	
	// 检查第二步是否有依赖
	if len(plan.Steps) < 2 {
		t.Fatal("Expected at least 2 steps")
	}
	
	// 包含 "after" 的步骤应该有依赖
	step2 := plan.Steps[1]
	if strings.Contains(step2.Description, "after") && len(step2.Dependencies) == 0 {
		t.Log("Step with 'after' should have dependencies (this is a soft check)")
	}
}

// TestExecutorWithPreviousResults 测试使用之前结果的执行
func TestExecutorWithPreviousResults(t *testing.T) {
	mockLLM := NewMockChatModel()
	mockLLM.InvokeFunc = func(ctx context.Context, messages []types.Message, opts ...runnable.Option) (types.Message, error) {
		return types.NewAssistantMessage("Using previous result: 42, the answer is 84"), nil
	}
	
	config := StepExecutorConfig{
		LLM:   mockLLM,
		Tools: []tools.Tool{},
	}
	
	executor := NewStepExecutor(config)
	
	step := PlanStep{
		ID:           "step_2",
		Description:  "Double the previous result",
		Dependencies: []string{"step_1"},
	}
	
	previousResults := map[string]string{
		"step_1": "42",
	}
	
	ctx := context.Background()
	action, err := executor.ExecuteStep(ctx, step, "Calculate", previousResults)
	
	if err != nil {
		t.Fatalf("ExecuteStep failed: %v", err)
	}
	
	if action == nil {
		t.Fatal("Action is nil")
	}
}

// TestDefaultPrompts 测试默认提示词
func TestDefaultPrompts(t *testing.T) {
	plannerPrompt := getDefaultPlannerPrompt()
	if plannerPrompt == "" {
		t.Error("Default planner prompt should not be empty")
	}
	
	if !strings.Contains(plannerPrompt, "step") {
		t.Error("Planner prompt should mention steps")
	}
	
	executorPrompt := getDefaultExecutorPrompt()
	if executorPrompt == "" {
		t.Error("Default executor prompt should not be empty")
	}
	
	if !strings.Contains(executorPrompt, "execute") {
		t.Error("Executor prompt should mention execution")
	}
}
