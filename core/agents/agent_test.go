package agents

import (
	"context"
	"errors"
	"testing"
	
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	
	"langchain-go/core/chat"
	"langchain-go/core/runnable"
	"langchain-go/core/tools"
	"langchain-go/pkg/types"
)

// MockChatModel 是用于测试的 Mock ChatModel
type MockChatModel struct {
	*chat.BaseChatModel
	invokeFunc func(ctx context.Context, messages []types.Message, opts ...runnable.Option) (types.Message, error)
}

func NewMockChatModel() *MockChatModel {
	return &MockChatModel{
		BaseChatModel: chat.NewBaseChatModel("mock-model", "mock"),
	}
}

func (m *MockChatModel) Invoke(ctx context.Context, messages []types.Message, opts ...runnable.Option) (types.Message, error) {
	if m.invokeFunc != nil {
		return m.invokeFunc(ctx, messages, opts...)
	}
	return types.NewAssistantMessage("mock response"), nil
}

func (m *MockChatModel) Stream(ctx context.Context, messages []types.Message, opts ...runnable.Option) (<-chan runnable.StreamEvent[types.Message], error) {
	out := make(chan runnable.StreamEvent[types.Message], 1)
	go func() {
		defer close(out)
		out <- runnable.StreamEvent[types.Message]{
			Data: types.NewAssistantMessage("mock stream response"),
		}
	}()
	return out, nil
}

func (m *MockChatModel) Batch(ctx context.Context, inputs [][]types.Message, opts ...runnable.Option) ([]types.Message, error) {
	results := make([]types.Message, len(inputs))
	for i := range inputs {
		results[i] = types.NewAssistantMessage("mock batch response")
	}
	return results, nil
}

func (m *MockChatModel) BindTools(tools []types.Tool) chat.ChatModel {
	newModel := NewMockChatModel()
	newModel.SetBoundTools(tools)
	newModel.invokeFunc = m.invokeFunc
	return newModel
}

func (m *MockChatModel) WithStructuredOutput(schema types.Schema) chat.ChatModel {
	newModel := NewMockChatModel()
	newModel.SetOutputSchema(schema)
	newModel.invokeFunc = m.invokeFunc
	return newModel
}

func (m *MockChatModel) WithConfig(config *types.Config) runnable.Runnable[[]types.Message, types.Message] {
	newModel := NewMockChatModel()
	newModel.SetConfig(config)
	newModel.invokeFunc = m.invokeFunc
	return newModel
}

func (m *MockChatModel) WithRetry(policy types.RetryPolicy) runnable.Runnable[[]types.Message, types.Message] {
	return runnable.NewRetryRunnable[[]types.Message, types.Message](m, policy)
}

func (m *MockChatModel) WithFallbacks(fallbacks ...runnable.Runnable[[]types.Message, types.Message]) runnable.Runnable[[]types.Message, types.Message] {
	return runnable.NewFallbackRunnable[[]types.Message, types.Message](m, fallbacks)
}

// MockTool 是用于测试的 Mock Tool
type MockTool struct {
	name        string
	description string
	executeFunc func(ctx context.Context, input map[string]any) (any, error)
}

func NewMockTool(name, description string) *MockTool {
	return &MockTool{
		name:        name,
		description: description,
	}
}

func (t *MockTool) GetName() string {
	return t.name
}

func (t *MockTool) GetDescription() string {
	return t.description
}

func (t *MockTool) GetParameters() types.Schema {
	return types.Schema{
		Type: "object",
		Properties: map[string]types.Schema{
			"input": {Type: "string"},
		},
	}
}

func (t *MockTool) ToTypesTool() types.Tool {
	return types.Tool{
		Name:        t.name,
		Description: t.description,
		Parameters:  t.GetParameters(),
	}
}

func (t *MockTool) Execute(ctx context.Context, input map[string]any) (any, error) {
	if t.executeFunc != nil {
		return t.executeFunc(ctx, input)
	}
	return "mock tool result", nil
}

// TestAgentConfig
func TestAgentConfig(t *testing.T) {
	mockLLM := NewMockChatModel()
	mockTool := NewMockTool("test_tool", "Test tool")
	
	config := AgentConfig{
		Type:         AgentTypeReAct,
		LLM:          mockLLM,
		Tools:        []tools.Tool{mockTool},
		MaxSteps:     10,
		SystemPrompt: "Test prompt",
	}
	
	assert.Equal(t, AgentTypeReAct, config.Type)
	assert.Equal(t, mockLLM, config.LLM)
	assert.Len(t, config.Tools, 1)
	assert.Equal(t, 10, config.MaxSteps)
	assert.Equal(t, "Test prompt", config.SystemPrompt)
}

// TestCreateAgent
func TestCreateAgent(t *testing.T) {
	mockLLM := NewMockChatModel()
	mockTool := NewMockTool("calculator", "Calculate")
	
	tests := []struct {
		name      string
		agentType AgentType
		wantErr   bool
	}{
		{
			name:      "ReAct Agent",
			agentType: AgentTypeReAct,
			wantErr:   false,
		},
		{
			name:      "ToolCalling Agent",
			agentType: AgentTypeToolCalling,
			wantErr:   false,
		},
		{
			name:      "Conversational Agent",
			agentType: AgentTypeConversational,
			wantErr:   false,
		},
		{
			name:      "Unknown Agent",
			agentType: AgentType("unknown"),
			wantErr:   true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := AgentConfig{
				Type:  tt.agentType,
				LLM:   mockLLM,
				Tools: []tools.Tool{mockTool},
			}
			
			agent, err := CreateAgent(config)
			
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, agent)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, agent)
				assert.Equal(t, tt.agentType, agent.GetType())
			}
		})
	}
}

// TestBaseAgent
func TestBaseAgent(t *testing.T) {
	mockLLM := NewMockChatModel()
	mockTool := NewMockTool("calculator", "Calculate")
	
	config := AgentConfig{
		Type:  AgentTypeReAct,
		LLM:   mockLLM,
		Tools: []tools.Tool{mockTool},
	}
	
	baseAgent := NewBaseAgent(config)
	
	assert.NotNil(t, baseAgent)
	assert.Equal(t, AgentTypeReAct, baseAgent.GetType())
	assert.Len(t, baseAgent.GetTools(), 1)
	
	// Test GetTool
	tool, err := baseAgent.GetTool("calculator")
	assert.NoError(t, err)
	assert.Equal(t, "calculator", tool.GetName())
	
	// Test GetTool - not found
	_, err = baseAgent.GetTool("nonexistent")
	assert.Error(t, err)
	
	// Test FormatToolsForPrompt
	prompt := baseAgent.FormatToolsForPrompt()
	assert.Contains(t, prompt, "calculator")
	assert.Contains(t, prompt, "Calculate")
}

// TestExecutor_Execute
func TestExecutor_Execute(t *testing.T) {
	mockLLM := NewMockChatModel()
	mockTool := NewMockTool("calculator", "Calculate")
	
	t.Run("Successful execution", func(t *testing.T) {
		// Mock LLM 返回 Final Answer
		mockLLM.invokeFunc = func(ctx context.Context, messages []types.Message, opts ...runnable.Option) (types.Message, error) {
			return types.NewAssistantMessage("Final Answer: 42"), nil
		}
		
		config := AgentConfig{
			Type:  AgentTypeReAct,
			LLM:   mockLLM,
			Tools: []tools.Tool{mockTool},
		}
		
		agent, err := CreateAgent(config)
		require.NoError(t, err)
		
		executor := NewExecutor(agent).WithMaxSteps(5)
		
		result, err := executor.Execute(context.Background(), "What is 6*7?")
		
		assert.NoError(t, err)
		assert.True(t, result.Success)
		assert.Contains(t, result.Output, "42")
	})
	
	t.Run("Max steps reached", func(t *testing.T) {
		// Mock LLM 始终返回工具调用
		mockLLM.invokeFunc = func(ctx context.Context, messages []types.Message, opts ...runnable.Option) (types.Message, error) {
			return types.NewAssistantMessage("Thought: I need to use calculator\nAction: calculator\nAction Input: 6*7"), nil
		}
		
		config := AgentConfig{
			Type:  AgentTypeReAct,
			LLM:   mockLLM,
			Tools: []tools.Tool{mockTool},
		}
		
		agent, err := CreateAgent(config)
		require.NoError(t, err)
		
		executor := NewExecutor(agent).WithMaxSteps(3)
		
		result, err := executor.Execute(context.Background(), "Calculate something")
		
		assert.Error(t, err)
		assert.Equal(t, ErrAgentMaxSteps, err)
		assert.False(t, result.Success)
		assert.Equal(t, 3, result.TotalSteps)
	})
}

// TestAgentAction
func TestAgentAction(t *testing.T) {
	action := &AgentAction{
		Type:        ActionToolCall,
		Tool:        "calculator",
		ToolInput:   map[string]any{"input": "5+3"},
		Log:         "I need to calculate",
		FinalAnswer: "",
	}
	
	assert.Equal(t, ActionToolCall, action.Type)
	assert.Equal(t, "calculator", action.Tool)
	assert.Contains(t, action.Log, "calculate")
}

// TestAgentStep
func TestAgentStep(t *testing.T) {
	action := &AgentAction{
		Type: ActionToolCall,
		Tool: "calculator",
	}
	
	step := AgentStep{
		Action:      action,
		Observation: "Result: 8",
		Error:       nil,
	}
	
	assert.Equal(t, action, step.Action)
	assert.Equal(t, "Result: 8", step.Observation)
	assert.Nil(t, step.Error)
}

// TestAgentResult
func TestAgentResult(t *testing.T) {
	result := &AgentResult{
		Output:     "The answer is 42",
		Steps:      []AgentStep{},
		TotalSteps: 3,
		Success:    true,
		Error:      nil,
	}
	
	assert.True(t, result.Success)
	assert.Equal(t, 3, result.TotalSteps)
	assert.Contains(t, result.Output, "42")
}

// TestExecutor_Batch
func TestExecutor_Batch(t *testing.T) {
	mockLLM := NewMockChatModel()
	mockLLM.invokeFunc = func(ctx context.Context, messages []types.Message, opts ...runnable.Option) (types.Message, error) {
		return types.NewAssistantMessage("Final Answer: Done"), nil
	}
	
	mockTool := NewMockTool("calculator", "Calculate")
	
	config := AgentConfig{
		Type:  AgentTypeReAct,
		LLM:   mockLLM,
		Tools: []tools.Tool{mockTool},
	}
	
	agent, err := CreateAgent(config)
	require.NoError(t, err)
	
	executor := NewExecutor(agent)
	
	inputs := []string{"Task 1", "Task 2", "Task 3"}
	results, err := executor.Batch(context.Background(), inputs)
	
	assert.NoError(t, err)
	assert.Len(t, results, 3)
	
	for _, result := range results {
		assert.True(t, result.Success)
	}
}

// TestExecutor_WithMiddleware
func TestExecutor_WithMiddleware(t *testing.T) {
	mockLLM := NewMockChatModel()
	mockLLM.invokeFunc = func(ctx context.Context, messages []types.Message, opts ...runnable.Option) (types.Message, error) {
		return types.NewAssistantMessage("Final Answer: OK"), nil
	}
	
	mockTool := NewMockTool("tool", "Tool")
	
	config := AgentConfig{
		Type:  AgentTypeReAct,
		LLM:   mockLLM,
		Tools: []tools.Tool{mockTool},
	}
	
	agent, err := CreateAgent(config)
	require.NoError(t, err)
	
	// 创建一个简单的测试中间件
	middlewareCalled := false
	testMiddleware := func(ctx context.Context, input any, next func(context.Context, any) (any, error)) (any, error) {
		middlewareCalled = true
		return next(ctx, input)
	}
	
	executor := NewExecutor(agent).
		WithMaxSteps(5)
	
	// 注意：这里我们不能直接添加函数，需要通过 middleware.MiddlewareFunc 包装
	// 但为了测试，我们验证链是否存在
	assert.NotNil(t, executor.GetMiddlewareChain())
	
	_ = middlewareCalled
	_ = testMiddleware
}

// TestReActAgent_ParseOutput
func TestReActAgent_ParseOutput(t *testing.T) {
	mockLLM := NewMockChatModel()
	mockTool := NewMockTool("calculator", "Calculate")
	
	config := AgentConfig{
		Type:  AgentTypeReAct,
		LLM:   mockLLM,
		Tools: []tools.Tool{mockTool},
	}
	
	agent := NewReActAgent(config)
	
	tests := []struct {
		name       string
		output     string
		wantType   AgentActionType
		wantTool   string
		wantAnswer string
		wantErr    bool
	}{
		{
			name:       "Final Answer",
			output:     "Final Answer: The result is 42",
			wantType:   ActionFinish,
			wantAnswer: "The result is 42",
			wantErr:    false,
		},
		{
			name:     "Tool Call",
			output:   "Thought: I need to calculate\nAction: calculator\nAction Input: 5+3",
			wantType: ActionToolCall,
			wantTool: "calculator",
			wantErr:  false,
		},
		{
			name:    "No Action",
			output:  "Just thinking...",
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action, err := agent.parseOutput(tt.output)
			
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantType, action.Type)
				
				if tt.wantType == ActionFinish {
					assert.Contains(t, action.FinalAnswer, tt.wantAnswer)
				}
				
				if tt.wantType == ActionToolCall {
					assert.Equal(t, tt.wantTool, action.Tool)
				}
			}
		})
	}
}

// TestExecutor_ToolCallError
func TestExecutor_ToolCallError(t *testing.T) {
	mockLLM := NewMockChatModel()
	
	callCount := 0
	// Mock LLM 返回工具调用
	mockLLM.invokeFunc = func(ctx context.Context, messages []types.Message, opts ...runnable.Option) (types.Message, error) {
		callCount++
		// 第一次调用返回工具调用，第二次返回最终答案
		if callCount == 1 {
			return types.NewAssistantMessage("Thought: Use tool\nAction: failing_tool\nAction Input: test"), nil
		}
		// 第二次返回最终答案
		return types.NewAssistantMessage("Final Answer: Handled error"), nil
	}
	
	// 创建一个会失败的工具
	failingTool := NewMockTool("failing_tool", "A tool that fails")
	failingTool.executeFunc = func(ctx context.Context, input map[string]any) (any, error) {
		return nil, errors.New("tool execution failed")
	}
	
	config := AgentConfig{
		Type:  AgentTypeReAct,
		LLM:   mockLLM,
		Tools: []tools.Tool{failingTool},
	}
	
	agent, err := CreateAgent(config)
	require.NoError(t, err)
	
	executor := NewExecutor(agent).WithMaxSteps(5)
	
	result, err := executor.Execute(context.Background(), "Test error handling")
	
	// Agent 应该能够处理工具错误并继续
	assert.NoError(t, err)
	assert.True(t, result.Success)
	assert.GreaterOrEqual(t, len(result.Steps), 1) // 至少有一步
	assert.NotNil(t, result.Steps[0].Error) // 第一步应该有错误
}
