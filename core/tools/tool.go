package tools

import (
	"context"
	"errors"
	"fmt"
	"time"

	"langchain-go/pkg/types"
)

// 常见错误
var (
	// ErrToolNotFound 工具未找到
	ErrToolNotFound = errors.New("tool not found")

	// ErrInvalidArguments 参数无效
	ErrInvalidArguments = errors.New("invalid arguments")

	// ErrTimeout 执行超时
	ErrTimeout = errors.New("tool execution timeout")

	// ErrExecutionFailed 执行失败
	ErrExecutionFailed = errors.New("tool execution failed")
)

// Tool 是可执行工具的接口。
//
// Tool 定义了工具的基本信息和执行方法。
// 所有工具都必须实现此接口。
//
// 示例：
//
//	type MyTool struct {
//	    name        string
//	    description string
//	}
//
//	func (t *MyTool) GetName() string {
//	    return t.name
//	}
//
//	func (t *MyTool) Execute(ctx context.Context, args map[string]any) (any, error) {
//	    // 执行工具逻辑
//	    return result, nil
//	}
//
type Tool interface {
	// GetName 获取工具名称
	GetName() string

	// GetDescription 获取工具描述
	GetDescription() string

	// GetParameters 获取参数 Schema
	GetParameters() types.Schema

	// Execute 执行工具
	//
	// 参数：
	//   - ctx: 上下文（用于取消和超时）
	//   - args: 工具参数
	//
	// 返回：
	//   - any: 执行结果
	//   - error: 执行错误
	//
	Execute(ctx context.Context, args map[string]any) (any, error)

	// ToTypesTool 转换为 types.Tool
	ToTypesTool() types.Tool
}

// FunctionTool 是基于函数的工具实现。
//
// FunctionTool 允许将任意函数包装为工具。
type FunctionTool struct {
	name        string
	description string
	parameters  types.Schema
	fn          func(ctx context.Context, args map[string]any) (any, error)
}

// FunctionToolConfig 是 FunctionTool 的配置。
type FunctionToolConfig struct {
	// Name 工具名称
	Name string

	// Description 工具描述
	Description string

	// Parameters 参数 Schema
	Parameters types.Schema

	// Fn 执行函数
	Fn func(ctx context.Context, args map[string]any) (any, error)
}

// NewFunctionTool 创建基于函数的工具。
//
// 参数：
//   - config: 工具配置
//
// 返回：
//   - *FunctionTool: 函数工具实例
//
func NewFunctionTool(config FunctionToolConfig) *FunctionTool {
	return &FunctionTool{
		name:        config.Name,
		description: config.Description,
		parameters:  config.Parameters,
		fn:          config.Fn,
	}
}

// GetName 实现 Tool 接口。
func (t *FunctionTool) GetName() string {
	return t.name
}

// GetDescription 实现 Tool 接口。
func (t *FunctionTool) GetDescription() string {
	return t.description
}

// GetParameters 实现 Tool 接口。
func (t *FunctionTool) GetParameters() types.Schema {
	return t.parameters
}

// Execute 实现 Tool 接口。
func (t *FunctionTool) Execute(ctx context.Context, args map[string]any) (any, error) {
	if t.fn == nil {
		return nil, fmt.Errorf("%w: function is nil", ErrExecutionFailed)
	}

	return t.fn(ctx, args)
}

// ToTypesTool 实现 Tool 接口。
func (t *FunctionTool) ToTypesTool() types.Tool {
	return types.Tool{
		Name:        t.name,
		Description: t.description,
		Parameters:  t.parameters,
	}
}

// ToolExecutor 是工具执行器。
//
// ToolExecutor 管理多个工具，并提供统一的执行接口。
type ToolExecutor struct {
	tools   map[string]Tool
	timeout time.Duration
}

// ToolExecutorConfig 是 ToolExecutor 的配置。
type ToolExecutorConfig struct {
	// Tools 工具列表
	Tools []Tool

	// Timeout 默认超时时间（0 表示无超时）
	Timeout time.Duration
}

// NewToolExecutor 创建工具执行器。
//
// 参数：
//   - config: 执行器配置
//
// 返回：
//   - *ToolExecutor: 执行器实例
//
func NewToolExecutor(config ToolExecutorConfig) *ToolExecutor {
	toolsMap := make(map[string]Tool)
	for _, tool := range config.Tools {
		toolsMap[tool.GetName()] = tool
	}

	return &ToolExecutor{
		tools:   toolsMap,
		timeout: config.Timeout,
	}
}

// Execute 执行指定的工具。
//
// 参数：
//   - ctx: 上下文
//   - toolName: 工具名称
//   - args: 工具参数
//
// 返回：
//   - any: 执行结果
//   - error: 执行错误
//
func (e *ToolExecutor) Execute(ctx context.Context, toolName string, args map[string]any) (any, error) {
	// 查找工具
	tool, exists := e.tools[toolName]
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrToolNotFound, toolName)
	}

	// 应用超时
	if e.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, e.timeout)
		defer cancel()
	}

	// 执行工具
	resultChan := make(chan executeResult, 1)

	go func() {
		result, err := tool.Execute(ctx, args)
		resultChan <- executeResult{result: result, err: err}
	}()

	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("%w: %v", ErrTimeout, ctx.Err())
	case res := <-resultChan:
		return res.result, res.err
	}
}

// ExecuteToolCall 执行 ToolCall。
//
// 这是一个便捷方法，直接接受 types.ToolCall。
//
// 参数：
//   - ctx: 上下文
//   - toolCall: 工具调用
//
// 返回：
//   - any: 执行结果
//   - error: 执行错误
//
func (e *ToolExecutor) ExecuteToolCall(ctx context.Context, toolCall types.ToolCall) (any, error) {
	// 解析参数
	args, err := toolCall.GetToolCallArgs()
	if err != nil {
		return nil, fmt.Errorf("%w: failed to parse arguments: %v", ErrInvalidArguments, err)
	}

	return e.Execute(ctx, toolCall.Function.Name, args)
}

// GetTool 获取指定名称的工具。
//
// 参数：
//   - name: 工具名称
//
// 返回：
//   - Tool: 工具实例
//   - bool: 是否存在
//
func (e *ToolExecutor) GetTool(name string) (Tool, bool) {
	tool, exists := e.tools[name]
	return tool, exists
}

// GetAllTools 获取所有工具。
//
// 返回：
//   - []Tool: 工具列表
//
func (e *ToolExecutor) GetAllTools() []Tool {
	tools := make([]Tool, 0, len(e.tools))
	for _, tool := range e.tools {
		tools = append(tools, tool)
	}
	return tools
}

// GetTypesTools 获取所有工具的 types.Tool 表示。
//
// 用于绑定到 ChatModel。
//
// 返回：
//   - []types.Tool: 工具列表
//
func (e *ToolExecutor) GetTypesTools() []types.Tool {
	tools := make([]types.Tool, 0, len(e.tools))
	for _, tool := range e.tools {
		tools = append(tools, tool.ToTypesTool())
	}
	return tools
}

// AddTool 添加工具。
//
// 参数：
//   - tool: 工具实例
//
func (e *ToolExecutor) AddTool(tool Tool) {
	e.tools[tool.GetName()] = tool
}

// RemoveTool 移除工具。
//
// 参数：
//   - name: 工具名称
//
func (e *ToolExecutor) RemoveTool(name string) {
	delete(e.tools, name)
}

// HasTool 检查工具是否存在。
//
// 参数：
//   - name: 工具名称
//
// 返回：
//   - bool: 是否存在
//
func (e *ToolExecutor) HasTool(name string) bool {
	_, exists := e.tools[name]
	return exists
}

// executeResult 是执行结果的内部类型。
type executeResult struct {
	result any
	err    error
}
