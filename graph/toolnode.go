package graph

import (
	"context"
	"fmt"
	
	"github.com/zhucl121/langchain-go/core/tools"
	"github.com/zhucl121/langchain-go/pkg/types"
)

// ToolNode 是专门用于工具调用的节点。
//
// ToolNode 简化了在图中集成工具的过程，自动处理：
//   - 工具选择
//   - 参数提取
//   - 工具执行
//   - 结果格式化
//
// 使用场景：
//   - Agent 工作流中的工具调用
//   - 多工具并行执行
//   - 工具结果聚合
//
type ToolNode[S any] struct {
	name       string
	tools      map[string]tools.Tool
	fallback   tools.Tool
	concurrent bool
}

// NewToolNode 创建工具节点。
//
// 参数：
//   - name: 节点名称
//   - toolList: 工具列表
//
// 返回：
//   - *ToolNode[S]: 工具节点实例
//
func NewToolNode[S any](name string, toolList []tools.Tool) *ToolNode[S] {
	toolMap := make(map[string]tools.Tool)
	for _, tool := range toolList {
		toolMap[tool.GetName()] = tool
	}
	
	return &ToolNode[S]{
		name:       name,
		tools:      toolMap,
		concurrent: false,
	}
}

// WithFallback 设置后备工具。
//
// 当请求的工具不存在时，使用后备工具。
//
func (tn *ToolNode[S]) WithFallback(fallback tools.Tool) *ToolNode[S] {
	tn.fallback = fallback
	return tn
}

// WithConcurrent 设置是否并行执行多个工具。
func (tn *ToolNode[S]) WithConcurrent(concurrent bool) *ToolNode[S] {
	tn.concurrent = concurrent
	return tn
}

// GetName 实现 Node 接口。
func (tn *ToolNode[S]) GetName() string {
	return tn.name
}

// Execute 实现 Node 接口。
//
// 从状态中提取工具调用信息并执行。
//
func (tn *ToolNode[S]) Execute(ctx context.Context, state S) (S, error) {
	// 尝试提取工具调用信息
	toolCalls, err := tn.extractToolCalls(state)
	if err != nil {
		return state, fmt.Errorf("toolnode: extract tool calls failed: %w", err)
	}
	
	if len(toolCalls) == 0 {
		// 没有工具调用，直接返回
		return state, nil
	}
	
	// 执行工具调用
	var results []ToolCallResult
	if tn.concurrent && len(toolCalls) > 1 {
		results, err = tn.executeParallel(ctx, toolCalls)
	} else {
		results, err = tn.executeSequential(ctx, toolCalls)
	}
	
	if err != nil {
		return state, err
	}
	
	// 将结果写回状态
	newState, err := tn.updateStateWithResults(state, results)
	if err != nil {
		return state, fmt.Errorf("toolnode: update state failed: %w", err)
	}
	
	return newState, nil
}

// ToolCallResult 表示工具调用结果。
type ToolCallResult struct {
	ToolName string
	Input    map[string]any
	Output   any
	Error    error
}

// extractToolCalls 从状态中提取工具调用。
func (tn *ToolNode[S]) extractToolCalls(state S) ([]types.ToolCall, error) {
	// 尝试使用类型断言提取
	type ToolCallExtractor interface {
		GetToolCalls() []types.ToolCall
	}
	
	if extractor, ok := any(state).(ToolCallExtractor); ok {
		return extractor.GetToolCalls(), nil
	}
	
	// 尝试从 map 中提取
	if stateMap, ok := any(state).(map[string]any); ok {
		if toolCallsAny, exists := stateMap["tool_calls"]; exists {
			if toolCalls, ok := toolCallsAny.([]types.ToolCall); ok {
				return toolCalls, nil
			}
		}
	}
	
	// 没有找到工具调用
	return []types.ToolCall{}, nil
}

// executeSequential 顺序执行工具调用。
func (tn *ToolNode[S]) executeSequential(ctx context.Context, toolCalls []types.ToolCall) ([]ToolCallResult, error) {
	results := make([]ToolCallResult, len(toolCalls))
	
	for i, toolCall := range toolCalls {
		result := tn.executeOne(ctx, toolCall)
		results[i] = result
		
		// 如果有错误且没有设置后备，立即返回
		if result.Error != nil && tn.fallback == nil {
			return results[:i+1], result.Error
		}
	}
	
	return results, nil
}

// executeParallel 并行执行工具调用。
func (tn *ToolNode[S]) executeParallel(ctx context.Context, toolCalls []types.ToolCall) ([]ToolCallResult, error) {
	results := make([]ToolCallResult, len(toolCalls))
	resultChan := make(chan struct {
		index  int
		result ToolCallResult
	}, len(toolCalls))
	
	// 启动并行执行
	for i, toolCall := range toolCalls {
		go func(idx int, tc types.ToolCall) {
			result := tn.executeOne(ctx, tc)
			resultChan <- struct {
				index  int
				result ToolCallResult
			}{idx, result}
		}(i, toolCall)
	}
	
	// 收集结果
	var firstError error
	for i := 0; i < len(toolCalls); i++ {
		res := <-resultChan
		results[res.index] = res.result
		
		if res.result.Error != nil && firstError == nil {
			firstError = res.result.Error
		}
	}
	close(resultChan)
	
	return results, firstError
}

// executeOne 执行单个工具调用。
func (tn *ToolNode[S]) executeOne(ctx context.Context, toolCall types.ToolCall) ToolCallResult {
	toolName := toolCall.Function.Name
	
	// 查找工具
	tool, exists := tn.tools[toolName]
	if !exists {
		if tn.fallback != nil {
			tool = tn.fallback
		} else {
			return ToolCallResult{
				ToolName: toolName,
				Error:    fmt.Errorf("tool not found: %s", toolName),
			}
		}
	}
	
	// 解析参数
	input, err := tn.parseToolInput(toolCall)
	if err != nil {
		return ToolCallResult{
			ToolName: toolName,
			Input:    input,
			Error:    fmt.Errorf("parse input failed: %w", err),
		}
	}
	
	// 执行工具
	output, err := tool.Execute(ctx, input)
	
	return ToolCallResult{
		ToolName: toolName,
		Input:    input,
		Output:   output,
		Error:    err,
	}
}

// parseToolInput 解析工具输入。
func (tn *ToolNode[S]) parseToolInput(toolCall types.ToolCall) (map[string]any, error) {
	// toolCall.Function.Arguments 是 string，需要解析
	// 简单实现：直接包装为 map
	return map[string]any{
		"input": toolCall.Function.Arguments,
	}, nil
}

// updateStateWithResults 将结果更新到状态。
func (tn *ToolNode[S]) updateStateWithResults(state S, results []ToolCallResult) (S, error) {
	// 使用反射获取实际类型
	stateValue := any(state)
	
	// 尝试使用类型断言更新（针对指针类型）
	type ToolResultUpdater interface {
		SetToolResults(results []ToolCallResult)
	}
	
	if updater, ok := stateValue.(ToolResultUpdater); ok {
		updater.SetToolResults(results)
		return state, nil
	}
	
	// 尝试作为 map 更新
	if stateMap, ok := stateValue.(map[string]any); ok {
		stateMap["tool_results"] = results
		return state, nil
	}
	
	// 无法更新，返回原状态
	// 这种情况下结果会丢失，但不报错
	return state, nil
}

// GetTools 获取所有工具。
func (tn *ToolNode[S]) GetTools() []tools.Tool {
	toolList := make([]tools.Tool, 0, len(tn.tools))
	for _, tool := range tn.tools {
		toolList = append(toolList, tool)
	}
	return toolList
}

// GetTool 根据名称获取工具。
func (tn *ToolNode[S]) GetTool(name string) (tools.Tool, bool) {
	tool, exists := tn.tools[name]
	return tool, exists
}

// AddTool 添加工具。
func (tn *ToolNode[S]) AddTool(tool tools.Tool) {
	tn.tools[tool.GetName()] = tool
}

// RemoveTool 移除工具。
func (tn *ToolNode[S]) RemoveTool(name string) {
	delete(tn.tools, name)
}
