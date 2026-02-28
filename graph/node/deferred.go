package node

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// DeferredState 延迟节点的状态追踪接口。
//
// 实现此接口的状态类型可以参与延迟节点的结果聚合。
type DeferredState interface {
	// MergePartial 合并一个并行分支的部分结果
	MergePartial(partial any) error
}

// BranchResult 并行分支的执行结果。
type BranchResult[S any] struct {
	BranchName string
	State      S
	Err        error
	Duration   time.Duration
}

// DeferredNodeConfig 延迟节点配置。
//
// 延迟节点（Deferred Node）在所有上游并行分支完成后才执行。
// 对标 LangGraph 1.0 Deferred Nodes，用于实现 Map-Reduce、
// 多 Agent 协作聚合、共识决策等模式。
type DeferredNodeConfig struct {
	// Timeout 等待所有上游分支完成的超时时间（0 表示不限时）
	Timeout time.Duration

	// FailFast 任意分支失败时是否立即终止等待（默认 false：等所有分支完成）
	FailFast bool

	// MinBranches 最少需要完成的分支数（0 表示必须全部完成）
	MinBranches int
}

// DeferredFunctionNode 延迟执行节点。
//
// DeferredFunctionNode 会等到所有声明的上游分支（Branches）都完成后，
// 再将各分支结果聚合并执行当前节点的聚合函数。
//
// 典型使用场景：
//   - Map-Reduce：分发给多个 worker 后聚合
//   - 多 Agent 协作：多个 Agent 并行分析后汇总
//   - 共识决策：等所有投票方结束再做决定
//
// 示例（Map-Reduce 模式）：
//
//	split := node.NewFunctionNode("split", splitFn)
//	worker1 := node.NewFunctionNode("worker-1", worker1Fn)
//	worker2 := node.NewFunctionNode("worker-2", worker2Fn)
//	reduce := node.NewDeferredFunctionNode("reduce", reduceFn,
//	    []string{"worker-1", "worker-2"},
//	    node.DeferredNodeConfig{Timeout: 30 * time.Second},
//	)
type DeferredFunctionNode[S any] struct {
	metadata         *Metadata
	fn               NodeFunc[S]
	upstreamBranches []string
	config           DeferredNodeConfig

	mu       sync.Mutex
	results  map[string]*BranchResult[S]
	waitCh   chan struct{}
	settled  bool
}

// NewDeferredFunctionNode 创建延迟函数节点。
//
// 参数：
//   - name: 节点名称
//   - fn: 聚合函数（接收已注入所有分支结果的状态）
//   - upstreamBranches: 需要等待完成的上游分支节点名称列表
//   - config: 延迟节点配置
//   - opts: 节点选项
func NewDeferredFunctionNode[S any](
	name string,
	fn NodeFunc[S],
	upstreamBranches []string,
	config DeferredNodeConfig,
	opts ...NodeOption,
) *DeferredFunctionNode[S] {
	metadata := NewMetadata(name)
	for _, opt := range opts {
		opt(metadata)
	}

	branches := make([]string, len(upstreamBranches))
	copy(branches, upstreamBranches)

	return &DeferredFunctionNode[S]{
		metadata:         metadata,
		fn:               fn,
		upstreamBranches: branches,
		config:           config,
		results:          make(map[string]*BranchResult[S]),
		waitCh:           make(chan struct{}),
	}
}

// GetName 实现 Node 接口。
func (d *DeferredFunctionNode[S]) GetName() string {
	return d.metadata.Name
}

// GetDescription 实现 Node 接口。
func (d *DeferredFunctionNode[S]) GetDescription() string {
	return d.metadata.Description
}

// GetTags 实现 Node 接口。
func (d *DeferredFunctionNode[S]) GetTags() []string {
	return d.metadata.Tags
}

// Validate 实现 Node 接口。
func (d *DeferredFunctionNode[S]) Validate() error {
	if err := d.metadata.Validate(); err != nil {
		return err
	}
	if d.fn == nil {
		return fmt.Errorf("%w: %s", ErrNodeFuncNil, d.metadata.Name)
	}
	if len(d.upstreamBranches) == 0 {
		return fmt.Errorf("deferred node %s: upstream branches cannot be empty", d.metadata.Name)
	}
	return nil
}

// RegisterBranchResult 注册一个上游分支的执行结果。
//
// 当分支节点执行完成后，调用此方法通知延迟节点。
// 所有分支都注册结果后，Invoke 的等待阶段将解除阻塞。
func (d *DeferredFunctionNode[S]) RegisterBranchResult(branchName string, state S, err error, duration time.Duration) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.results[branchName] = &BranchResult[S]{
		BranchName: branchName,
		State:      state,
		Err:        err,
		Duration:   duration,
	}

	if d.isReadyLocked() && !d.settled {
		d.settled = true
		close(d.waitCh)
	}
}

// isReadyLocked 判断是否满足执行条件（需持有锁）。
func (d *DeferredFunctionNode[S]) isReadyLocked() bool {
	if d.config.MinBranches > 0 {
		completed := 0
		for _, branch := range d.upstreamBranches {
			if _, ok := d.results[branch]; ok {
				completed++
			}
		}
		return completed >= d.config.MinBranches
	}

	// 默认：所有上游分支都完成
	for _, branch := range d.upstreamBranches {
		if _, ok := d.results[branch]; !ok {
			return false
		}
	}
	return true
}

// Invoke 实现 Node 接口。
//
// 执行顺序：
//  1. 等待所有上游分支完成（或超时）
//  2. 将各分支结果收集到状态中
//  3. 调用聚合函数
func (d *DeferredFunctionNode[S]) Invoke(ctx context.Context, state S) (S, error) {
	select {
	case <-ctx.Done():
		return state, ctx.Err()
	default:
	}

	// 等待所有上游分支完成
	if err := d.waitForBranches(ctx); err != nil {
		return state, fmt.Errorf("deferred node %s: %w", d.metadata.Name, err)
	}

	// 收集并检查分支结果
	d.mu.Lock()
	results := make(map[string]*BranchResult[S], len(d.results))
	for k, v := range d.results {
		results[k] = v
	}
	d.mu.Unlock()

	// 检查是否有分支失败
	if d.config.FailFast {
		for _, r := range results {
			if r.Err != nil {
				return state, fmt.Errorf("deferred node %s: branch %s failed: %w",
					d.metadata.Name, r.BranchName, r.Err)
			}
		}
	}

	// 将所有分支结果注入到 context，供聚合函数使用
	ctxWithResults := contextWithBranchResults(ctx, d.GetName(), results)

	return d.fn(ctxWithResults, state)
}

// waitForBranches 等待上游分支完成。
func (d *DeferredFunctionNode[S]) waitForBranches(ctx context.Context) error {
	if d.config.Timeout > 0 {
		timeoutCtx, cancel := context.WithTimeout(ctx, d.config.Timeout)
		defer cancel()

		select {
		case <-d.waitCh:
			return nil
		case <-timeoutCtx.Done():
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return fmt.Errorf("timeout waiting for upstream branches after %s", d.config.Timeout)
		}
	}

	select {
	case <-d.waitCh:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Reset 重置延迟节点状态，允许在新一轮执行中复用。
func (d *DeferredFunctionNode[S]) Reset() {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.results = make(map[string]*BranchResult[S])
	d.waitCh = make(chan struct{})
	d.settled = false
}

// GetBranchResults 获取已注册的分支结果（仅在 Invoke 后调用有意义）。
func (d *DeferredFunctionNode[S]) GetBranchResults() map[string]*BranchResult[S] {
	d.mu.Lock()
	defer d.mu.Unlock()

	results := make(map[string]*BranchResult[S], len(d.results))
	for k, v := range d.results {
		results[k] = v
	}
	return results
}

// UpstreamBranches 返回需要等待的上游分支名称列表。
func (d *DeferredFunctionNode[S]) UpstreamBranches() []string {
	branches := make([]string, len(d.upstreamBranches))
	copy(branches, d.upstreamBranches)
	return branches
}

// ─── Context 工具函数 ───────────────────────────────────────────

type deferredResultsKey struct{ nodeName string }

// contextWithBranchResults 将分支结果注入 context。
func contextWithBranchResults[S any](ctx context.Context, nodeName string, results map[string]*BranchResult[S]) context.Context {
	return context.WithValue(ctx, deferredResultsKey{nodeName}, results)
}

// GetBranchResultsFromContext 从 context 中获取延迟节点的分支结果。
//
// 在聚合函数（DeferredFunctionNode 的 fn）内部调用，获取各分支的执行结果。
//
// 参数：
//   - ctx: 节点执行 context
//   - nodeName: 延迟节点名称
//
// 返回：
//   - map[string]*BranchResult[S]: 分支结果映射（branchName -> result）
//   - bool: 是否找到结果
func GetBranchResultsFromContext[S any](ctx context.Context, nodeName string) (map[string]*BranchResult[S], bool) {
	v := ctx.Value(deferredResultsKey{nodeName})
	if v == nil {
		return nil, false
	}
	results, ok := v.(map[string]*BranchResult[S])
	return results, ok
}

// ─── MapReduceRunner 便捷工具 ────────────────────────────────────

// MapReduceRunner 提供便捷的 Map-Reduce 执行模式。
//
// 自动管理分支注册、并行执行和结果聚合，无需手动管理 DeferredFunctionNode。
//
// 示例：
//
//	runner := node.NewMapReduceRunner([]string{"w1", "w2", "w3"})
//	result, err := runner.Run(ctx, initialState,
//	    map[string]node.NodeFunc[MyState]{
//	        "w1": worker1Fn,
//	        "w2": worker2Fn,
//	        "w3": worker3Fn,
//	    },
//	    reduceFn,
//	)
type MapReduceRunner[S any] struct {
	branches []string
	config   DeferredNodeConfig
}

// NewMapReduceRunner 创建 Map-Reduce 执行器。
func NewMapReduceRunner[S any](branches []string, opts ...DeferredNodeConfig) *MapReduceRunner[S] {
	config := DeferredNodeConfig{}
	if len(opts) > 0 {
		config = opts[0]
	}
	return &MapReduceRunner[S]{
		branches: branches,
		config:   config,
	}
}

// Run 并行执行所有 worker 并用 reduceFn 聚合结果。
//
// 参数：
//   - ctx: 上下文
//   - initialState: 初始状态（每个 worker 收到相同的初始状态副本）
//   - workers: 分支名称到节点函数的映射
//   - reduceFn: 聚合函数（ctx 中包含各分支结果，可用 GetBranchResultsFromContext 获取）
func (r *MapReduceRunner[S]) Run(
	ctx context.Context,
	initialState S,
	workers map[string]NodeFunc[S],
	reduceFn NodeFunc[S],
) (S, error) {
	deferredNode := NewDeferredFunctionNode(
		"map-reduce",
		reduceFn,
		r.branches,
		r.config,
	)

	var wg sync.WaitGroup
	for _, branch := range r.branches {
		branch := branch
		workerFn, ok := workers[branch]
		if !ok {
			continue
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			start := time.Now()
			result, err := workerFn(ctx, initialState)
			deferredNode.RegisterBranchResult(branch, result, err, time.Since(start))
		}()
	}

	// 不等待 wg，让 Invoke 自行等待 waitCh
	go func() {
		wg.Wait()
	}()

	return deferredNode.Invoke(ctx, initialState)
}
