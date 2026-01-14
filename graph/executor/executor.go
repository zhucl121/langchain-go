package executor

import (
	"context"
	"fmt"
	"time"
)

// CompiledGraphInfo 是已编译图的接口（避免循环依赖）。
type CompiledGraphInfo interface {
	GetName() string
	GetEntryPoint() string
	GetAdjacency() map[string][]string
}

// NodeFunc 是节点函数类型。
type NodeFunc[S any] func(ctx context.Context, state S) (S, error)

// ConditionalRouter 是条件路由器接口。
type ConditionalRouter[S any] interface {
	Route(state S) (string, error)
}

// Executor 是图执行器。
//
// Executor 负责执行已编译的状态图。
//
// 核心功能：
//   - 图执行
//   - 节点调度
//   - 状态传递
//   - 错误处理
//   - 中断支持
//
type Executor[S any] struct {
	scheduler *Scheduler[S]

	// 节点映射（需要在执行前设置）
	nodes        map[string]NodeFunc[S]
	conditionals map[string]ConditionalRouter[S]
}

// NewExecutor 创建执行器。
//
// 返回：
//   - *Executor[S]: 执行器实例
//
func NewExecutor[S any]() *Executor[S] {
	return &Executor[S]{
		scheduler:    NewScheduler[S](),
		nodes:        make(map[string]NodeFunc[S]),
		conditionals: make(map[string]ConditionalRouter[S]),
	}
}

// WithScheduler 设置调度器。
//
// 参数：
//   - scheduler: 调度器
//
// 返回：
//   - *Executor[S]: 返回自身，支持链式调用
//
func (e *Executor[S]) WithScheduler(scheduler *Scheduler[S]) *Executor[S] {
	e.scheduler = scheduler
	return e
}

// RegisterNode 注册节点函数。
//
// 参数：
//   - name: 节点名称
//   - nodeFunc: 节点函数
//
// 返回：
//   - *Executor[S]: 返回自身，支持链式调用
//
func (e *Executor[S]) RegisterNode(name string, nodeFunc NodeFunc[S]) *Executor[S] {
	e.nodes[name] = nodeFunc
	return e
}

// RegisterConditional 注册条件路由器。
//
// 参数：
//   - name: 节点名称
//   - router: 条件路由器
//
// 返回：
//   - *Executor[S]: 返回自身，支持链式调用
//
func (e *Executor[S]) RegisterConditional(name string, router ConditionalRouter[S]) *Executor[S] {
	e.conditionals[name] = router
	return e
}

// Execute 执行图（简单版本）。
//
// 参数：
//   - ctx: 上下文
//   - graph: 已编译的图
//   - initialState: 初始状态
//
// 返回：
//   - S: 最终状态
//   - error: 执行错误
//
func (e *Executor[S]) Execute(
	ctx context.Context,
	graph CompiledGraphInfo,
	initialState S,
) (S, error) {
	execCtx := NewExecutionContext(initialState)
	return e.ExecuteWithContext(ctx, graph, execCtx)
}

// ExecuteWithContext 使用执行上下文执行图。
//
// 参数：
//   - ctx: 上下文
//   - graph: 已编译的图
//   - execCtx: 执行上下文
//
// 返回：
//   - S: 最终状态
//   - error: 执行错误
//
func (e *Executor[S]) ExecuteWithContext(
	ctx context.Context,
	graph CompiledGraphInfo,
	execCtx *ExecutionContext[S],
) (S, error) {
	// 获取入口点
	currentNode := graph.GetEntryPoint()
	if currentNode == "" {
		var zero S
		return zero, fmt.Errorf("no entry point in graph %s", graph.GetName())
	}

	// 获取邻接表
	adjacency := graph.GetAdjacency()

	// 主执行循环
	for {
		// 检查上下文
		if err := execCtx.CheckContext(ctx); err != nil {
			var zero S
			return zero, ErrExecutionCancelled
		}

		// 检查步数限制
		if err := execCtx.IncrementStep(); err != nil {
			var zero S
			return zero, err
		}

		// 检查是否到达终点
		if currentNode == "__end__" {
			return execCtx.GetState(), nil
		}

		// 检查中断点
		if execCtx.ShouldInterrupt(currentNode) {
			execCtx.Interrupt(currentNode)
			return execCtx.GetState(), ErrInterrupted
		}

		// 执行节点
		startTime := time.Now()
		execCtx.EmitEvent(Event{
			Type:      EventNodeStart,
			NodeName:  currentNode,
			Timestamp: startTime,
		})

		newState, err := e.executeNode(ctx, currentNode, execCtx.GetState())
		duration := time.Since(startTime)

		if err != nil {
			// 记录错误
			execCtx.AddHistory(ExecutionHistory{
				NodeName:  currentNode,
				Timestamp: startTime,
				Duration:  duration,
				Error:     err,
			})

			execCtx.EmitEvent(Event{
				Type:      EventNodeError,
				NodeName:  currentNode,
				Timestamp: time.Now(),
				Error:     err,
			})

			var zero S
			return zero, fmt.Errorf("node %s failed: %w", currentNode, err)
		}

		// 更新状态
		execCtx.UpdateState(newState)

		// 记录历史
		execCtx.AddHistory(ExecutionHistory{
			NodeName:  currentNode,
			Timestamp: startTime,
			Duration:  duration,
		})

		execCtx.EmitEvent(Event{
			Type:      EventNodeEnd,
			NodeName:  currentNode,
			Timestamp: time.Now(),
		})

		// 路由到下一个节点
		nextNode, err := e.routeNext(currentNode, newState, adjacency)
		if err != nil {
			if err == ErrNoNextNode {
				// 没有下一个节点，正常结束
				return newState, nil
			}
			var zero S
			return zero, fmt.Errorf("routing from %s failed: %w", currentNode, err)
		}

		currentNode = nextNode
	}
}

// executeNode 执行单个节点。
func (e *Executor[S]) executeNode(
	ctx context.Context,
	nodeName string,
	state S,
) (S, error) {
	nodeFunc, exists := e.nodes[nodeName]
	if !exists {
		var zero S
		return zero, fmt.Errorf("%w: %s", ErrNodeNotFound, nodeName)
	}

	return nodeFunc(ctx, state)
}

// routeNext 路由到下一个节点。
func (e *Executor[S]) routeNext(
	currentNode string,
	state S,
	adjacency map[string][]string,
) (string, error) {
	// 检查是否有条件路由器
	if router, exists := e.conditionals[currentNode]; exists {
		return router.Route(state)
	}

	// 使用普通边
	neighbors, exists := adjacency[currentNode]
	if !exists || len(neighbors) == 0 {
		return "", ErrNoNextNode
	}

	// 返回第一个邻居（对于普通边，应该只有一个）
	return neighbors[0], nil
}

// ExecutionResult 是执行结果。
type ExecutionResult[S any] struct {
	State       S
	History     []ExecutionHistory
	Events      []Event
	Interrupted bool
	Error       error
}

// ExecuteWithResult 执行并返回详细结果。
//
// 参数：
//   - ctx: 上下文
//   - graph: 已编译的图
//   - initialState: 初始状态
//
// 返回：
//   - *ExecutionResult[S]: 执行结果
//
func (e *Executor[S]) ExecuteWithResult(
	ctx context.Context,
	graph CompiledGraphInfo,
	initialState S,
) *ExecutionResult[S] {
	execCtx := NewExecutionContext(initialState)

	state, err := e.ExecuteWithContext(ctx, graph, execCtx)

	return &ExecutionResult[S]{
		State:       state,
		History:     execCtx.GetHistory(),
		Events:      execCtx.GetEvents(),
		Interrupted: execCtx.IsInterrupted(),
		Error:       err,
	}
}
