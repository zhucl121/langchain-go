package executor

import (
	"context"
	"fmt"
	"sync"
)

// ScheduleStrategy 是调度策略。
type ScheduleStrategy string

const (
	// StrategySequential 顺序执行（默认）
	StrategySequential ScheduleStrategy = "sequential"

	// StrategyParallel 并行执行
	// 注意：并行执行时，每个节点使用相同的初始状态，
	// 返回的是所有节点执行后的状态列表
	StrategyParallel ScheduleStrategy = "parallel"
)

// NodeExecutor 是节点执行器接口（避免循环依赖）。
type NodeExecutor interface {
	Execute(ctx context.Context, state any) (any, error)
}

// StateMerger 是状态合并器接口。
//
// 当多个节点并行执行时，可以使用 StateMerger 自定义状态合并策略。
//
type StateMerger[S any] interface {
	// Merge 合并多个状态为一个状态
	//
	// 参数：
	//   - states: 多个节点的输出状态
	//
	// 返回：
	//   - S: 合并后的状态
	//   - error: 合并错误
	//
	Merge(states []S) (S, error)
}

// DefaultStateMerger 默认状态合并器（返回最后一个状态）。
type DefaultStateMerger[S any] struct{}

func (d *DefaultStateMerger[S]) Merge(states []S) (S, error) {
	if len(states) == 0 {
		var zero S
		return zero, fmt.Errorf("no states to merge")
	}
	return states[len(states)-1], nil
}

// Scheduler 是节点调度器。
//
// Scheduler 负责管理节点的执行调度。
//
// 功能：
//   - 节点执行调度
//   - 并发控制
//   - 资源管理
//
type Scheduler[S any] struct {
	strategy      ScheduleStrategy
	maxConcurrent int
	semaphore     chan struct{} // 并发控制信号量
	stateMerger   StateMerger[S] // 状态合并器（用于并行执行）

	mu sync.RWMutex
}

// NewScheduler 创建调度器。
//
// 返回：
//   - *Scheduler[S]: 调度器实例
//
func NewScheduler[S any]() *Scheduler[S] {
	return &Scheduler[S]{
		strategy:      StrategySequential,
		maxConcurrent: 1,
		semaphore:     make(chan struct{}, 1),
		stateMerger:   &DefaultStateMerger[S]{},
	}
}

// WithStrategy 设置调度策略。
//
// 参数：
//   - strategy: 调度策略
//
// 返回：
//   - *Scheduler[S]: 返回自身，支持链式调用
//
func (s *Scheduler[S]) WithStrategy(strategy ScheduleStrategy) *Scheduler[S] {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.strategy = strategy
	return s
}

// WithMaxConcurrent 设置最大并发数。
//
// 参数：
//   - max: 最大并发数
//
// 返回：
//   - *Scheduler[S]: 返回自身，支持链式调用
//
func (s *Scheduler[S]) WithMaxConcurrent(max int) *Scheduler[S] {
	s.mu.Lock()
	defer s.mu.Unlock()

	if max < 1 {
		max = 1
	}

	s.maxConcurrent = max
	s.semaphore = make(chan struct{}, max)
	return s
}

// WithStateMerger 设置状态合并器。
//
// 参数：
//   - merger: 状态合并器
//
// 返回：
//   - *Scheduler[S]: 返回自身，支持链式调用
//
func (s *Scheduler[S]) WithStateMerger(merger StateMerger[S]) *Scheduler[S] {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.stateMerger = merger
	return s
}

// ScheduleNode 调度单个节点执行。
//
// 参数：
//   - ctx: 上下文
//   - nodeName: 节点名称
//   - executor: 节点执行器
//   - state: 当前状态
//
// 返回：
//   - S: 新状态
//   - error: 执行错误
//
func (s *Scheduler[S]) ScheduleNode(
	ctx context.Context,
	nodeName string,
	executor NodeExecutor,
	state S,
) (S, error) {
	// 获取信号量
	select {
	case s.semaphore <- struct{}{}:
		defer func() { <-s.semaphore }()
	case <-ctx.Done():
		var zero S
		return zero, ctx.Err()
	}

	// 执行节点
	result, err := executor.Execute(ctx, state)
	if err != nil {
		var zero S
		return zero, fmt.Errorf("node %s execution failed: %w", nodeName, err)
	}

	// 类型断言
	newState, ok := result.(S)
	if !ok {
		var zero S
		return zero, fmt.Errorf("node %s returned invalid state type", nodeName)
	}

	return newState, nil
}

// ScheduleNodes 调度多个节点执行。
//
// 根据调度策略执行：
//   - StrategySequential: 顺序执行，每个节点使用前一个节点的输出状态
//   - StrategyParallel: 并行执行，所有节点使用相同的初始状态
//
// 参数：
//   - ctx: 上下文
//   - nodes: 节点名称列表
//   - executors: 节点执行器映射
//   - state: 当前状态
//
// 返回：
//   - []S: 新状态列表（顺序执行时，最后一个是最终状态；并行执行时，所有节点的状态）
//   - error: 执行错误
//
func (s *Scheduler[S]) ScheduleNodes(
	ctx context.Context,
	nodes []string,
	executors map[string]NodeExecutor,
	state S,
) ([]S, error) {
	s.mu.RLock()
	strategy := s.strategy
	s.mu.RUnlock()

	switch strategy {
	case StrategySequential:
		return s.scheduleSequential(ctx, nodes, executors, state)
	case StrategyParallel:
		return s.scheduleParallel(ctx, nodes, executors, state)
	default:
		return s.scheduleSequential(ctx, nodes, executors, state)
	}
}

// scheduleSequential 顺序执行多个节点。
func (s *Scheduler[S]) scheduleSequential(
	ctx context.Context,
	nodes []string,
	executors map[string]NodeExecutor,
	state S,
) ([]S, error) {
	results := make([]S, 0, len(nodes))
	currentState := state

	for _, nodeName := range nodes {
		executor, exists := executors[nodeName]
		if !exists {
			return nil, fmt.Errorf("%w: %s", ErrNodeNotFound, nodeName)
		}

		newState, err := s.ScheduleNode(ctx, nodeName, executor, currentState)
		if err != nil {
			return nil, err
		}

		results = append(results, newState)
		currentState = newState
	}

	return results, nil
}

// scheduleParallel 并行执行多个节点。
//
// 并行执行策略：
//   - 所有节点使用相同的初始状态
//   - 每个节点在独立的 goroutine 中执行
//   - 使用信号量控制并发数
//   - 收集所有节点的执行结果
//   - 任何节点失败都会导致整体失败
//
func (s *Scheduler[S]) scheduleParallel(
	ctx context.Context,
	nodes []string,
	executors map[string]NodeExecutor,
	state S,
) ([]S, error) {
	if len(nodes) == 0 {
		return []S{}, nil
	}

	// 如果只有一个节点，直接执行
	if len(nodes) == 1 {
		executor, exists := executors[nodes[0]]
		if !exists {
			return nil, fmt.Errorf("%w: %s", ErrNodeNotFound, nodes[0])
		}
		newState, err := s.ScheduleNode(ctx, nodes[0], executor, state)
		if err != nil {
			return nil, err
		}
		return []S{newState}, nil
	}

	// 并行执行多个节点
	type result struct {
		index int
		state S
		err   error
	}

	resultChan := make(chan result, len(nodes))
	var wg sync.WaitGroup

	// 启动并行任务
	for i, nodeName := range nodes {
		executor, exists := executors[nodeName]
		if !exists {
			return nil, fmt.Errorf("%w: %s", ErrNodeNotFound, nodeName)
		}

		wg.Add(1)
		go func(idx int, name string, exec NodeExecutor) {
			defer wg.Done()

			// 使用信号量控制并发
			select {
			case s.semaphore <- struct{}{}:
				defer func() { <-s.semaphore }()
			case <-ctx.Done():
				resultChan <- result{index: idx, err: ctx.Err()}
				return
			}

			// 执行节点
			newState, err := exec.Execute(ctx, state)
			if err != nil {
				resultChan <- result{
					index: idx,
					err:   fmt.Errorf("node %s execution failed: %w", name, err),
				}
				return
			}

			// 类型断言
			typedState, ok := newState.(S)
			if !ok {
				resultChan <- result{
					index: idx,
					err:   fmt.Errorf("node %s returned invalid state type", name),
				}
				return
			}

			resultChan <- result{index: idx, state: typedState, err: nil}
		}(i, nodeName, executor)
	}

	// 等待所有任务完成
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// 收集结果
	results := make([]S, len(nodes))
	var firstErr error

	for res := range resultChan {
		if res.err != nil && firstErr == nil {
			firstErr = res.err
		}
		if res.err == nil {
			results[res.index] = res.state
		}
	}

	if firstErr != nil {
		return nil, firstErr
	}

	return results, nil
}

// GetStrategy 获取调度策略。
func (s *Scheduler[S]) GetStrategy() ScheduleStrategy {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.strategy
}

// GetMaxConcurrent 获取最大并发数。
func (s *Scheduler[S]) GetMaxConcurrent() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.maxConcurrent
}
