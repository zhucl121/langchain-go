package agents

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// ParallelExecutor 是并行执行器，支持同时执行多个工具调用。
//
// 提供以下功能：
//   - 并行执行多个工具调用
//   - 可配置并发数
//   - 超时控制
//   - 错误聚合
//
type ParallelExecutor struct {
	executor       *AgentExecutor
	maxConcurrency int
	timeout        time.Duration
}

// ParallelExecutorConfig 是并行执行器配置。
type ParallelExecutorConfig struct {
	// Executor 基础执行器
	Executor *AgentExecutor

	// MaxConcurrency 最大并发数（0 表示无限制）
	MaxConcurrency int

	// Timeout 单个工具调用超时时间
	Timeout time.Duration
}

// NewParallelExecutor 创建并行执行器。
//
// 参数：
//   - config: 并行执行器配置
//
// 返回：
//   - *ParallelExecutor: 并行执行器实例
//
func NewParallelExecutor(config ParallelExecutorConfig) *ParallelExecutor {
	if config.MaxConcurrency <= 0 {
		config.MaxConcurrency = 5 // 默认最多 5 个并发
	}

	if config.Timeout <= 0 {
		config.Timeout = 30 * time.Second // 默认 30 秒超时
	}

	return &ParallelExecutor{
		executor:       config.Executor,
		maxConcurrency: config.MaxConcurrency,
		timeout:        config.Timeout,
	}
}

// ParallelToolResult 是并行工具执行结果。
type ParallelToolResult struct {
	// Action 执行的行动
	Action *AgentAction

	// Observation 观察结果
	Observation string

	// Error 错误（如果有）
	Error error

	// Duration 执行时长
	Duration time.Duration

	// Index 在原始行动列表中的索引
	Index int
}

// RunParallel 并行执行多个工具调用。
//
// 参数：
//   - ctx: 上下文
//   - actions: 要执行的行动列表
//
// 返回：
//   - []ParallelToolResult: 执行结果列表（按原始顺序）
//   - error: 错误（如果所有工具都失败）
//
func (pe *ParallelExecutor) RunParallel(ctx context.Context, actions []*AgentAction) ([]ParallelToolResult, error) {
	if len(actions) == 0 {
		return []ParallelToolResult{}, nil
	}

	// 创建结果通道
	resultChan := make(chan ParallelToolResult, len(actions))
	
	// 创建信号量控制并发
	semaphore := make(chan struct{}, pe.maxConcurrency)
	
	// 创建 WaitGroup
	var wg sync.WaitGroup

	// 并行执行所有工具调用
	for i, action := range actions {
		wg.Add(1)
		go func(index int, act *AgentAction) {
			defer wg.Done()

			// 获取信号量
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// 执行工具调用
			result := pe.executeToolWithTimeout(ctx, act, index)
			resultChan <- result
		}(i, action)
	}

	// 等待所有工具执行完成
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// 收集结果
	results := make([]ParallelToolResult, len(actions))
	successCount := 0

	for result := range resultChan {
		results[result.Index] = result
		if result.Error == nil {
			successCount++
		}
	}

	// 如果所有工具都失败，返回错误
	if successCount == 0 {
		return results, fmt.Errorf("parallel executor: all %d tools failed", len(actions))
	}

	return results, nil
}

// executeToolWithTimeout 执行单个工具调用（带超时）。
func (pe *ParallelExecutor) executeToolWithTimeout(
	ctx context.Context,
	action *AgentAction,
	index int,
) ParallelToolResult {
	startTime := time.Now()

	result := ParallelToolResult{
		Action: action,
		Index:  index,
	}

	// 创建带超时的上下文
	timeoutCtx, cancel := context.WithTimeout(ctx, pe.timeout)
	defer cancel()

	// 在 goroutine 中执行工具
	resultChan := make(chan struct {
		observation string
		err         error
	}, 1)

	go func() {
		observation, err := pe.executor.executeToolWithExecutor(timeoutCtx, action)
		resultChan <- struct {
			observation string
			err         error
		}{observation, err}
	}()

	// 等待结果或超时
	select {
	case res := <-resultChan:
		result.Observation = res.observation
		result.Error = res.err
	case <-timeoutCtx.Done():
		result.Error = fmt.Errorf("tool execution timeout after %v", pe.timeout)
	}

	result.Duration = time.Since(startTime)
	return result
}

// RunParallelWithMerge 并行执行多个工具调用，并合并结果。
//
// 参数：
//   - ctx: 上下文
//   - actions: 要执行的行动列表
//   - mergeFn: 结果合并函数
//
// 返回：
//   - string: 合并后的观察结果
//   - error: 错误
//
func (pe *ParallelExecutor) RunParallelWithMerge(
	ctx context.Context,
	actions []*AgentAction,
	mergeFn func([]ParallelToolResult) string,
) (string, error) {
	results, err := pe.RunParallel(ctx, actions)
	if err != nil {
		return "", err
	}

	if mergeFn != nil {
		return mergeFn(results), nil
	}

	// 默认合并策略：拼接所有结果
	return DefaultMergeStrategy(results), nil
}

// DefaultMergeStrategy 默认合并策略：拼接所有工具结果。
func DefaultMergeStrategy(results []ParallelToolResult) string {
	output := "Tool Results:\n\n"
	
	for _, result := range results {
		output += fmt.Sprintf("Tool: %s\n", result.Action.Tool)
		
		if result.Error != nil {
			output += fmt.Sprintf("Error: %v\n", result.Error)
		} else {
			output += fmt.Sprintf("Result: %s\n", result.Observation)
		}
		
		output += fmt.Sprintf("Duration: %v\n\n", result.Duration)
	}
	
	return output
}

// AgentExecutorWithParallel 为 AgentExecutor 添加并行执行方法。
type AgentExecutorWithParallel struct {
	*AgentExecutor
	parallelExecutor *ParallelExecutor
}

// WithParallelExecution 为 AgentExecutor 启用并行执行。
//
// 参数：
//   - maxConcurrency: 最大并发数
//   - timeout: 超时时间
//
// 返回：
//   - *AgentExecutorWithParallel: 带并行执行功能的执行器
//
func (ae *AgentExecutor) WithParallelExecution(maxConcurrency int, timeout time.Duration) *AgentExecutorWithParallel {
	parallelExecutor := NewParallelExecutor(ParallelExecutorConfig{
		Executor:       ae,
		MaxConcurrency: maxConcurrency,
		Timeout:        timeout,
	})

	return &AgentExecutorWithParallel{
		AgentExecutor:    ae,
		parallelExecutor: parallelExecutor,
	}
}

// RunWithParallelTools 执行 Agent，支持并行工具调用。
//
// 当 Agent 返回多个工具调用时，会并行执行这些工具。
//
// 参数：
//   - ctx: 上下文
//   - input: 输入问题
//
// 返回：
//   - *AgentResult: 执行结果
//   - error: 错误
//
func (aeWithParallel *AgentExecutorWithParallel) RunWithParallelTools(
	ctx context.Context,
	input string,
) (*AgentResult, error) {
	// 注意：这里需要 Agent 支持返回多个工具调用
	// 当前实现假设 Agent.Plan 只返回单个行动
	// 在实际使用中，可能需要扩展 Agent 接口来支持批量规划
	
	result, err := aeWithParallel.Run(ctx, input)
	return result, err
}

// GetParallelExecutor 返回并行执行器。
func (aeWithParallel *AgentExecutorWithParallel) GetParallelExecutor() *ParallelExecutor {
	return aeWithParallel.parallelExecutor
}
