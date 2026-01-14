package tools

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"langchain-go/pkg/types"
)

func TestNewFunctionTool(t *testing.T) {
	tool := NewFunctionTool(FunctionToolConfig{
		Name:        "test_tool",
		Description: "A test tool",
		Parameters: types.Schema{
			Type: "object",
			Properties: map[string]types.Schema{
				"input": {Type: "string"},
			},
		},
		Fn: func(ctx context.Context, args map[string]any) (any, error) {
			return "result", nil
		},
	})

	assert.NotNil(t, tool)
	assert.Equal(t, "test_tool", tool.GetName())
	assert.Equal(t, "A test tool", tool.GetDescription())
}

func TestFunctionTool_Execute(t *testing.T) {
	tool := NewFunctionTool(FunctionToolConfig{
		Name:        "echo",
		Description: "Echo the input",
		Parameters: types.Schema{
			Type: "object",
			Properties: map[string]types.Schema{
				"message": {Type: "string"},
			},
		},
		Fn: func(ctx context.Context, args map[string]any) (any, error) {
			return args["message"], nil
		},
	})

	result, err := tool.Execute(context.Background(), map[string]any{
		"message": "hello",
	})

	require.NoError(t, err)
	assert.Equal(t, "hello", result)
}

func TestFunctionTool_ExecuteWithError(t *testing.T) {
	expectedErr := errors.New("test error")

	tool := NewFunctionTool(FunctionToolConfig{
		Name:        "error_tool",
		Description: "A tool that returns error",
		Parameters:  types.Schema{Type: "object"},
		Fn: func(ctx context.Context, args map[string]any) (any, error) {
			return nil, expectedErr
		},
	})

	result, err := tool.Execute(context.Background(), map[string]any{})

	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
	assert.Nil(t, result)
}

func TestFunctionTool_ExecuteWithNilFunction(t *testing.T) {
	tool := NewFunctionTool(FunctionToolConfig{
		Name:        "nil_fn",
		Description: "Tool with nil function",
		Parameters:  types.Schema{Type: "object"},
		Fn:          nil,
	})

	result, err := tool.Execute(context.Background(), map[string]any{})

	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrExecutionFailed))
	assert.Nil(t, result)
}

func TestFunctionTool_ToTypesTool(t *testing.T) {
	tool := NewFunctionTool(FunctionToolConfig{
		Name:        "test",
		Description: "Test tool",
		Parameters: types.Schema{
			Type: "object",
			Properties: map[string]types.Schema{
				"param": {Type: "string"},
			},
		},
		Fn: func(ctx context.Context, args map[string]any) (any, error) {
			return nil, nil
		},
	})

	typesTool := tool.ToTypesTool()

	assert.Equal(t, "test", typesTool.Name)
	assert.Equal(t, "Test tool", typesTool.Description)
	assert.Equal(t, "object", typesTool.Parameters.Type)
}

func TestNewToolExecutor(t *testing.T) {
	tool1 := NewFunctionTool(FunctionToolConfig{
		Name:        "tool1",
		Description: "Tool 1",
		Parameters:  types.Schema{Type: "object"},
		Fn:          func(ctx context.Context, args map[string]any) (any, error) { return "result1", nil },
	})

	tool2 := NewFunctionTool(FunctionToolConfig{
		Name:        "tool2",
		Description: "Tool 2",
		Parameters:  types.Schema{Type: "object"},
		Fn:          func(ctx context.Context, args map[string]any) (any, error) { return "result2", nil },
	})

	executor := NewToolExecutor(ToolExecutorConfig{
		Tools:   []Tool{tool1, tool2},
		Timeout: 5 * time.Second,
	})

	assert.NotNil(t, executor)
	assert.True(t, executor.HasTool("tool1"))
	assert.True(t, executor.HasTool("tool2"))
	assert.False(t, executor.HasTool("tool3"))
}

func TestToolExecutor_Execute(t *testing.T) {
	tool := NewFunctionTool(FunctionToolConfig{
		Name:        "add",
		Description: "Add two numbers",
		Parameters:  types.Schema{Type: "object"},
		Fn: func(ctx context.Context, args map[string]any) (any, error) {
			a := args["a"].(float64)
			b := args["b"].(float64)
			return a + b, nil
		},
	})

	executor := NewToolExecutor(ToolExecutorConfig{
		Tools: []Tool{tool},
	})

	result, err := executor.Execute(context.Background(), "add", map[string]any{
		"a": 10.0,
		"b": 20.0,
	})

	require.NoError(t, err)
	assert.Equal(t, 30.0, result)
}

func TestToolExecutor_ExecuteToolNotFound(t *testing.T) {
	executor := NewToolExecutor(ToolExecutorConfig{
		Tools: []Tool{},
	})

	result, err := executor.Execute(context.Background(), "nonexistent", map[string]any{})

	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrToolNotFound))
	assert.Nil(t, result)
}

func TestToolExecutor_ExecuteWithTimeout(t *testing.T) {
	slowTool := NewFunctionTool(FunctionToolConfig{
		Name:        "slow",
		Description: "Slow tool",
		Parameters:  types.Schema{Type: "object"},
		Fn: func(ctx context.Context, args map[string]any) (any, error) {
			time.Sleep(2 * time.Second)
			return "done", nil
		},
	})

	executor := NewToolExecutor(ToolExecutorConfig{
		Tools:   []Tool{slowTool},
		Timeout: 100 * time.Millisecond,
	})

	result, err := executor.Execute(context.Background(), "slow", map[string]any{})

	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrTimeout))
	assert.Nil(t, result)
}

func TestToolExecutor_GetTool(t *testing.T) {
	tool := NewFunctionTool(FunctionToolConfig{
		Name:        "test",
		Description: "Test",
		Parameters:  types.Schema{Type: "object"},
		Fn:          func(ctx context.Context, args map[string]any) (any, error) { return nil, nil },
	})

	executor := NewToolExecutor(ToolExecutorConfig{
		Tools: []Tool{tool},
	})

	// 找到工具
	found, exists := executor.GetTool("test")
	assert.True(t, exists)
	assert.NotNil(t, found)
	assert.Equal(t, "test", found.GetName())

	// 找不到工具
	found, exists = executor.GetTool("nonexistent")
	assert.False(t, exists)
	assert.Nil(t, found)
}

func TestToolExecutor_GetAllTools(t *testing.T) {
	tool1 := NewFunctionTool(FunctionToolConfig{
		Name:        "tool1",
		Description: "Tool 1",
		Parameters:  types.Schema{Type: "object"},
		Fn:          func(ctx context.Context, args map[string]any) (any, error) { return nil, nil },
	})

	tool2 := NewFunctionTool(FunctionToolConfig{
		Name:        "tool2",
		Description: "Tool 2",
		Parameters:  types.Schema{Type: "object"},
		Fn:          func(ctx context.Context, args map[string]any) (any, error) { return nil, nil },
	})

	executor := NewToolExecutor(ToolExecutorConfig{
		Tools: []Tool{tool1, tool2},
	})

	tools := executor.GetAllTools()
	assert.Len(t, tools, 2)
}

func TestToolExecutor_GetTypesTools(t *testing.T) {
	tool1 := NewFunctionTool(FunctionToolConfig{
		Name:        "tool1",
		Description: "Tool 1",
		Parameters:  types.Schema{Type: "object"},
		Fn:          func(ctx context.Context, args map[string]any) (any, error) { return nil, nil },
	})

	executor := NewToolExecutor(ToolExecutorConfig{
		Tools: []Tool{tool1},
	})

	typesTools := executor.GetTypesTools()
	assert.Len(t, typesTools, 1)
	assert.Equal(t, "tool1", typesTools[0].Name)
}

func TestToolExecutor_AddTool(t *testing.T) {
	executor := NewToolExecutor(ToolExecutorConfig{
		Tools: []Tool{},
	})

	assert.False(t, executor.HasTool("new"))

	newTool := NewFunctionTool(FunctionToolConfig{
		Name:        "new",
		Description: "New tool",
		Parameters:  types.Schema{Type: "object"},
		Fn:          func(ctx context.Context, args map[string]any) (any, error) { return nil, nil },
	})

	executor.AddTool(newTool)

	assert.True(t, executor.HasTool("new"))
}

func TestToolExecutor_RemoveTool(t *testing.T) {
	tool := NewFunctionTool(FunctionToolConfig{
		Name:        "test",
		Description: "Test",
		Parameters:  types.Schema{Type: "object"},
		Fn:          func(ctx context.Context, args map[string]any) (any, error) { return nil, nil },
	})

	executor := NewToolExecutor(ToolExecutorConfig{
		Tools: []Tool{tool},
	})

	assert.True(t, executor.HasTool("test"))

	executor.RemoveTool("test")

	assert.False(t, executor.HasTool("test"))
}

func TestToolExecutor_ExecuteToolCall(t *testing.T) {
	tool := NewFunctionTool(FunctionToolConfig{
		Name:        "multiply",
		Description: "Multiply two numbers",
		Parameters:  types.Schema{Type: "object"},
		Fn: func(ctx context.Context, args map[string]any) (any, error) {
			a := args["a"].(float64)
			b := args["b"].(float64)
			return a * b, nil
		},
	})

	executor := NewToolExecutor(ToolExecutorConfig{
		Tools: []Tool{tool},
	})

	toolCall := types.ToolCall{
		ID:   "call_123",
		Type: "function",
		Function: types.FunctionCall{
			Name:      "multiply",
			Arguments: `{"a": 5.0, "b": 4.0}`,
		},
	}

	result, err := executor.ExecuteToolCall(context.Background(), toolCall)

	require.NoError(t, err)
	assert.Equal(t, 20.0, result)
}
