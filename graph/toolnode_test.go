package graph

import (
	"context"
	"errors"
	"testing"
	
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	
	"github.com/zhucl121/langchain-go/core/tools"
	"github.com/zhucl121/langchain-go/pkg/types"
)

// TestState 是用于测试的状态类型
type TestState struct {
	ToolCalls   []types.ToolCall
	ToolResults []ToolCallResult
	Messages    []string
}

func (ts *TestState) GetToolCalls() []types.ToolCall {
	return ts.ToolCalls
}

func (ts *TestState) SetToolResults(results []ToolCallResult) {
	ts.ToolResults = results
}

// MockTool 是测试用的 Mock 工具
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

func (t *MockTool) Execute(ctx context.Context, input map[string]any) (any, error) {
	if t.executeFunc != nil {
		return t.executeFunc(ctx, input)
	}
	return "mock result", nil
}

func (t *MockTool) ToTypesTool() types.Tool {
	return types.Tool{
		Name:        t.name,
		Description: t.description,
		Parameters:  t.GetParameters(),
	}
}

// TestNewToolNode
func TestNewToolNode(t *testing.T) {
	tool1 := NewMockTool("tool1", "Test tool 1")
	tool2 := NewMockTool("tool2", "Test tool 2")
	
	node := NewToolNode[*TestState]("test-node", []tools.Tool{tool1, tool2})
	
	assert.NotNil(t, node)
	assert.Equal(t, "test-node", node.GetName())
	assert.Len(t, node.GetTools(), 2)
	assert.False(t, node.concurrent)
}

// TestToolNode_GetTool
func TestToolNode_GetTool(t *testing.T) {
	tool1 := NewMockTool("calculator", "Calculator")
	tool2 := NewMockTool("weather", "Weather")
	
	node := NewToolNode[*TestState]("test", []tools.Tool{tool1, tool2})
	
	// 存在的工具
	foundTool, exists := node.GetTool("calculator")
	assert.True(t, exists)
	assert.Equal(t, "calculator", foundTool.GetName())
	
	// 不存在的工具
	_, exists = node.GetTool("nonexistent")
	assert.False(t, exists)
}

// TestToolNode_AddRemoveTool
func TestToolNode_AddRemoveTool(t *testing.T) {
	tool1 := NewMockTool("tool1", "Tool 1")
	
	node := NewToolNode[*TestState]("test", []tools.Tool{tool1})
	assert.Len(t, node.GetTools(), 1)
	
	// 添加工具
	tool2 := NewMockTool("tool2", "Tool 2")
	node.AddTool(tool2)
	assert.Len(t, node.GetTools(), 2)
	
	// 移除工具
	node.RemoveTool("tool1")
	assert.Len(t, node.GetTools(), 1)
	
	_, exists := node.GetTool("tool1")
	assert.False(t, exists)
}

// TestToolNode_Execute_NoToolCalls
func TestToolNode_Execute_NoToolCalls(t *testing.T) {
	tool := NewMockTool("test", "Test")
	node := NewToolNode[*TestState]("test", []tools.Tool{tool})
	
	state := &TestState{
		ToolCalls: []types.ToolCall{},
	}
	
	newState, err := node.Execute(context.Background(), state)
	
	assert.NoError(t, err)
	// 返回的是指针
	assert.Same(t, state, newState)
}

// TestToolNode_Execute_SingleTool
func TestToolNode_Execute_SingleTool(t *testing.T) {
	tool := NewMockTool("calculator", "Calculator")
	tool.executeFunc = func(ctx context.Context, input map[string]any) (any, error) {
		return "Result: 42", nil
	}
	
	node := NewToolNode[*TestState]("test", []tools.Tool{tool})
	
	state := &TestState{
		ToolCalls: []types.ToolCall{
			{
				ID:   "call-1",
				Type: "function",
				Function: types.FunctionCall{
					Name:      "calculator",
					Arguments: "5+3",
				},
			},
		},
	}
	
	newState, err := node.Execute(context.Background(), state)
	
	assert.NoError(t, err)
	assert.NotNil(t, newState)
	assert.Len(t, newState.ToolResults, 1)
	assert.Equal(t, "calculator", newState.ToolResults[0].ToolName)
	assert.Equal(t, "Result: 42", newState.ToolResults[0].Output)
	assert.Nil(t, newState.ToolResults[0].Error)
}

// TestToolNode_Execute_MultipleTools
func TestToolNode_Execute_MultipleTools(t *testing.T) {
	tool1 := NewMockTool("tool1", "Tool 1")
	tool1.executeFunc = func(ctx context.Context, input map[string]any) (any, error) {
		return "Result 1", nil
	}
	
	tool2 := NewMockTool("tool2", "Tool 2")
	tool2.executeFunc = func(ctx context.Context, input map[string]any) (any, error) {
		return "Result 2", nil
	}
	
	node := NewToolNode[*TestState]("test", []tools.Tool{tool1, tool2})
	
	state := &TestState{
		ToolCalls: []types.ToolCall{
			{
				ID:   "call-1",
				Type: "function",
				Function: types.FunctionCall{
					Name:      "tool1",
					Arguments: "arg1",
				},
			},
			{
				ID:   "call-2",
				Type: "function",
				Function: types.FunctionCall{
					Name:      "tool2",
					Arguments: "arg2",
				},
			},
		},
	}
	
	newState, err := node.Execute(context.Background(), state)
	
	assert.NoError(t, err)
	assert.Len(t, newState.ToolResults, 2)
	assert.Equal(t, "Result 1", newState.ToolResults[0].Output)
	assert.Equal(t, "Result 2", newState.ToolResults[1].Output)
}

// TestToolNode_Execute_ToolNotFound
func TestToolNode_Execute_ToolNotFound(t *testing.T) {
	tool := NewMockTool("existing", "Existing")
	node := NewToolNode[*TestState]("test", []tools.Tool{tool})
	
	state := &TestState{
		ToolCalls: []types.ToolCall{
			{
				ID:   "call-1",
				Type: "function",
				Function: types.FunctionCall{
					Name:      "nonexistent",
					Arguments: "arg",
				},
			},
		},
	}
	
	newState, err := node.Execute(context.Background(), state)
	
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "tool not found")
	assert.NotNil(t, newState)
}

// TestToolNode_Execute_WithFallback
func TestToolNode_Execute_WithFallback(t *testing.T) {
	tool := NewMockTool("existing", "Existing")
	
	fallback := NewMockTool("fallback", "Fallback")
	fallback.executeFunc = func(ctx context.Context, input map[string]any) (any, error) {
		return "Fallback result", nil
	}
	
	node := NewToolNode[*TestState]("test", []tools.Tool{tool}).
		WithFallback(fallback)
	
	state := &TestState{
		ToolCalls: []types.ToolCall{
			{
				ID:   "call-1",
				Type: "function",
				Function: types.FunctionCall{
					Name:      "nonexistent",
					Arguments: "arg",
				},
			},
		},
	}
	
	newState, err := node.Execute(context.Background(), state)
	
	assert.NoError(t, err)
	assert.Len(t, newState.ToolResults, 1)
	assert.Equal(t, "Fallback result", newState.ToolResults[0].Output)
}

// TestToolNode_Execute_ToolError
func TestToolNode_Execute_ToolError(t *testing.T) {
	tool := NewMockTool("failing", "Failing tool")
	tool.executeFunc = func(ctx context.Context, input map[string]any) (any, error) {
		return nil, errors.New("tool execution failed")
	}
	
	node := NewToolNode[*TestState]("test", []tools.Tool{tool})
	
	state := &TestState{
		ToolCalls: []types.ToolCall{
			{
				ID:   "call-1",
				Type: "function",
				Function: types.FunctionCall{
					Name:      "failing",
					Arguments: "arg",
				},
			},
		},
	}
	
	newState, err := node.Execute(context.Background(), state)
	
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "tool execution failed")
	assert.NotNil(t, newState)
	// 由于错误，结果可能还没设置
	if len(newState.ToolResults) > 0 {
		assert.NotNil(t, newState.ToolResults[0].Error)
	}
}

// TestToolNode_Execute_Concurrent
func TestToolNode_Execute_Concurrent(t *testing.T) {
	tool1 := NewMockTool("tool1", "Tool 1")
	tool1.executeFunc = func(ctx context.Context, input map[string]any) (any, error) {
		return "Result 1", nil
	}
	
	tool2 := NewMockTool("tool2", "Tool 2")
	tool2.executeFunc = func(ctx context.Context, input map[string]any) (any, error) {
		return "Result 2", nil
	}
	
	node := NewToolNode[*TestState]("test", []tools.Tool{tool1, tool2}).
		WithConcurrent(true)
	
	state := &TestState{
		ToolCalls: []types.ToolCall{
			{
				ID:       "call-1",
				Type:     "function",
				Function: types.FunctionCall{Name: "tool1", Arguments: "arg1"},
			},
			{
				ID:       "call-2",
				Type:     "function",
				Function: types.FunctionCall{Name: "tool2", Arguments: "arg2"},
			},
		},
	}
	
	newState, err := node.Execute(context.Background(), state)
	
	assert.NoError(t, err)
	assert.Len(t, newState.ToolResults, 2)
	
	// 检查两个结果都存在（顺序可能不同）
	results := make(map[string]any)
	for _, result := range newState.ToolResults {
		results[result.ToolName] = result.Output
	}
	assert.Equal(t, "Result 1", results["tool1"])
	assert.Equal(t, "Result 2", results["tool2"])
}

// TestToolNode_WithMapState
func TestToolNode_WithMapState(t *testing.T) {
	tool := NewMockTool("test", "Test")
	tool.executeFunc = func(ctx context.Context, input map[string]any) (any, error) {
		return "Result", nil
	}
	
	node := NewToolNode[map[string]any]("test", []tools.Tool{tool})
	
	state := map[string]any{
		"tool_calls": []types.ToolCall{
			{
				ID:       "call-1",
				Type:     "function",
				Function: types.FunctionCall{Name: "test", Arguments: "arg"},
			},
		},
	}
	
	newState, err := node.Execute(context.Background(), state)
	
	assert.NoError(t, err)
	assert.NotNil(t, newState)
	
	// 检查结果被写入 map
	results, ok := newState["tool_results"].([]ToolCallResult)
	require.True(t, ok)
	assert.Len(t, results, 1)
	assert.Equal(t, "Result", results[0].Output)
}
