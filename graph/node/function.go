package node

import (
	"context"
	"fmt"
)

// FunctionNode 是基于函数的节点。
//
// FunctionNode 将普通函数包装为节点，是最常用的节点类型。
//
// 示例：
//
//	fn := func(ctx context.Context, s MyState) (MyState, error) {
//	    s.Counter++
//	    return s, nil
//	}
//
//	node := NewFunctionNode("increment", fn,
//	    WithDescription("增加计数器"),
//	    WithTags("counter", "math"),
//	)
//
//	result, err := node.Invoke(ctx, MyState{Counter: 0})
//	// result.Counter == 1
//
type FunctionNode[S any] struct {
	metadata *Metadata
	fn       NodeFunc[S]
}

// NewFunctionNode 创建函数节点。
//
// 参数：
//   - name: 节点名称
//   - fn: 节点函数
//   - opts: 节点选项
//
// 返回：
//   - *FunctionNode[S]: 函数节点实例
//
func NewFunctionNode[S any](name string, fn NodeFunc[S], opts ...NodeOption) *FunctionNode[S] {
	metadata := NewMetadata(name)
	for _, opt := range opts {
		opt(metadata)
	}

	return &FunctionNode[S]{
		metadata: metadata,
		fn:       fn,
	}
}

// GetName 实现 Node 接口。
func (n *FunctionNode[S]) GetName() string {
	return n.metadata.Name
}

// GetDescription 实现 Node 接口。
func (n *FunctionNode[S]) GetDescription() string {
	return n.metadata.Description
}

// GetTags 实现 Node 接口。
func (n *FunctionNode[S]) GetTags() []string {
	return n.metadata.Tags
}

// GetMetadata 返回节点元数据。
func (n *FunctionNode[S]) GetMetadata() *Metadata {
	return n.metadata.Clone()
}

// Invoke 实现 Node 接口。
func (n *FunctionNode[S]) Invoke(ctx context.Context, state S) (S, error) {
	if n.fn == nil {
		return state, fmt.Errorf("%w: %s", ErrNodeFuncNil, n.metadata.Name)
	}

	// 检查上下文
	select {
	case <-ctx.Done():
		return state, ctx.Err()
	default:
	}

	// 执行函数
	return n.fn(ctx, state)
}

// Validate 实现 Node 接口。
func (n *FunctionNode[S]) Validate() error {
	if err := n.metadata.Validate(); err != nil {
		return err
	}

	if n.fn == nil {
		return fmt.Errorf("%w: %s", ErrNodeFuncNil, n.metadata.Name)
	}

	return nil
}

// WithFunc 设置新的节点函数（返回新节点）。
//
// 参数：
//   - fn: 新的节点函数
//
// 返回：
//   - *FunctionNode[S]: 新的函数节点
//
func (n *FunctionNode[S]) WithFunc(fn NodeFunc[S]) *FunctionNode[S] {
	return &FunctionNode[S]{
		metadata: n.metadata.Clone(),
		fn:       fn,
	}
}

// Chain 将当前节点与另一个节点链接（串行执行）。
//
// 返回一个新节点，该节点先执行当前节点，再执行下一个节点。
//
// 参数：
//   - next: 下一个节点函数
//
// 返回：
//   - *FunctionNode[S]: 链接后的节点
//
// 示例：
//
//	node1 := NewFunctionNode("add", addFunc)
//	node2 := NewFunctionNode("multiply", multiplyFunc)
//	chained := node1.Chain(multiplyFunc)
//	// chained 会先执行 addFunc，再执行 multiplyFunc
//
func (n *FunctionNode[S]) Chain(next NodeFunc[S]) *FunctionNode[S] {
	chainedFunc := func(ctx context.Context, state S) (S, error) {
		// 执行当前节点
		result, err := n.Invoke(ctx, state)
		if err != nil {
			return state, err
		}

		// 执行下一个节点
		return next(ctx, result)
	}

	return NewFunctionNode(
		n.metadata.Name+"_chained",
		chainedFunc,
		WithDescription(n.metadata.Description+" (chained)"),
		WithTags(n.metadata.Tags...),
	)
}

// Retry 包装节点，添加重试逻辑。
//
// 参数：
//   - maxRetries: 最大重试次数
//
// 返回：
//   - *FunctionNode[S]: 带重试的节点
//
// 示例：
//
//	node := NewFunctionNode("api_call", apiCallFunc)
//	retryNode := node.Retry(3) // 最多重试 3 次
//
func (n *FunctionNode[S]) Retry(maxRetries int) *FunctionNode[S] {
	retryFunc := func(ctx context.Context, state S) (S, error) {
		var lastErr error

		for attempt := 0; attempt <= maxRetries; attempt++ {
			result, err := n.Invoke(ctx, state)
			if err == nil {
				return result, nil
			}

			lastErr = err

			// 检查上下文是否取消
			select {
			case <-ctx.Done():
				return state, ctx.Err()
			default:
			}
		}

		return state, fmt.Errorf("node %s failed after %d retries: %w",
			n.metadata.Name, maxRetries, lastErr)
	}

	return NewFunctionNode(
		n.metadata.Name+"_retry",
		retryFunc,
		WithDescription(fmt.Sprintf("%s (retry=%d)", n.metadata.Description, maxRetries)),
		WithTags(append(n.metadata.Tags, "retry")...),
	)
}

// Fallback 包装节点，添加降级逻辑。
//
// 如果当前节点失败，会执行降级函数。
//
// 参数：
//   - fallback: 降级函数
//
// 返回：
//   - *FunctionNode[S]: 带降级的节点
//
// 示例：
//
//	primaryNode := NewFunctionNode("primary", primaryFunc)
//	fallbackNode := primaryNode.Fallback(func(ctx context.Context, s State) (State, error) {
//	    s.Message = "Using fallback"
//	    return s, nil
//	})
//
func (n *FunctionNode[S]) Fallback(fallback NodeFunc[S]) *FunctionNode[S] {
	fallbackFunc := func(ctx context.Context, state S) (S, error) {
		result, err := n.Invoke(ctx, state)
		if err == nil {
			return result, nil
		}

		// 主节点失败，使用降级
		return fallback(ctx, state)
	}

	return NewFunctionNode(
		n.metadata.Name+"_fallback",
		fallbackFunc,
		WithDescription(n.metadata.Description+" (with fallback)"),
		WithTags(append(n.metadata.Tags, "fallback")...),
	)
}

// Transform 对节点输出进行转换。
//
// 参数：
//   - transformer: 转换函数
//
// 返回：
//   - *FunctionNode[S]: 转换后的节点
//
// 示例：
//
//	node := NewFunctionNode("process", processFunc)
//	transformed := node.Transform(func(ctx context.Context, s State) (State, error) {
//	    s.Value = s.Value * 2 // 将输出值翻倍
//	    return s, nil
//	})
//
func (n *FunctionNode[S]) Transform(transformer NodeFunc[S]) *FunctionNode[S] {
	transformFunc := func(ctx context.Context, state S) (S, error) {
		result, err := n.Invoke(ctx, state)
		if err != nil {
			return state, err
		}

		return transformer(ctx, result)
	}

	return NewFunctionNode(
		n.metadata.Name+"_transform",
		transformFunc,
		WithDescription(n.metadata.Description+" (transformed)"),
		WithTags(n.metadata.Tags...),
	)
}

// Conditional 包装节点，添加条件执行。
//
// 只有当条件满足时才执行节点，否则直接返回原状态。
//
// 参数：
//   - condition: 条件函数
//
// 返回：
//   - *FunctionNode[S]: 条件节点
//
// 示例：
//
//	node := NewFunctionNode("expensive", expensiveFunc)
//	conditional := node.Conditional(func(ctx context.Context, s State) bool {
//	    return s.NeedsProcessing
//	})
//
func (n *FunctionNode[S]) Conditional(condition func(context.Context, S) bool) *FunctionNode[S] {
	conditionalFunc := func(ctx context.Context, state S) (S, error) {
		if !condition(ctx, state) {
			// 条件不满足，直接返回原状态
			return state, nil
		}

		return n.Invoke(ctx, state)
	}

	return NewFunctionNode(
		n.metadata.Name+"_conditional",
		conditionalFunc,
		WithDescription(n.metadata.Description+" (conditional)"),
		WithTags(append(n.metadata.Tags, "conditional")...),
	)
}
