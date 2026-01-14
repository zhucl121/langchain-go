package node

import (
	"context"
	"fmt"
)

// SubgraphNode 是嵌套图节点。
//
// SubgraphNode 允许在节点中嵌套另一个完整的状态图。
// 这对于构建复杂的层次化工作流非常有用。
//
// 特性：
//   - 状态映射（父状态 <-> 子状态）
//   - 独立的图执行
//   - 支持不同的状态类型
//
// 注意：
//   - 需要提供状态映射函数
//   - 子图必须已编译
//
// 示例：
//
//	// 假设有父图状态和子图状态
//	type ParentState struct {
//	    Data map[string]any
//	}
//
//	type ChildState struct {
//	    Value int
//	}
//
//	// 创建子图
//	subgraph := state.NewStateGraph[ChildState]("sub")
//	// ... 配置子图节点
//	compiled, _ := subgraph.Compile()
//
//	// 创建子图节点
//	subgraphNode := NewSubgraphNode("nested", compiled,
//	    WithStateMapper(
//	        func(parent ParentState) (ChildState, error) {
//	            return ChildState{Value: parent.Data["value"].(int)}, nil
//	        },
//	        func(parent ParentState, child ChildState) (ParentState, error) {
//	            parent.Data["result"] = child.Value
//	            return parent, nil
//	        },
//	    ),
//	)
//
type SubgraphNode[ParentState, ChildState any] struct {
	metadata       *Metadata
	subgraph       SubgraphExecutor[ChildState]
	mapToChild     func(ParentState) (ChildState, error)
	mapFromChild   func(ParentState, ChildState) (ParentState, error)
}

// SubgraphExecutor 是子图执行器接口。
//
// 这是一个简化的接口，用于执行子图。
// 实际的 CompiledGraph 会实现此接口。
//
type SubgraphExecutor[S any] interface {
	Invoke(ctx context.Context, state S, opts ...interface{}) (S, error)
}

// StateMapper 配置状态映射。
type StateMapper[ParentState, ChildState any] struct {
	ToChild   func(ParentState) (ChildState, error)
	FromChild func(ParentState, ChildState) (ParentState, error)
}

// NewSubgraphNode 创建子图节点。
//
// 参数：
//   - name: 节点名称
//   - subgraph: 已编译的子图
//   - opts: 节点选项
//
// 返回：
//   - *SubgraphNode: 子图节点实例
//
// 注意：
//   - 必须通过 WithStateMapper 配置状态映射
//   - 子图必须已编译
//
func NewSubgraphNode[ParentState, ChildState any](
	name string,
	subgraph SubgraphExecutor[ChildState],
	opts ...interface{},
) *SubgraphNode[ParentState, ChildState] {
	node := &SubgraphNode[ParentState, ChildState]{
		metadata: NewMetadata(name),
		subgraph: subgraph,
	}

	// 应用选项
	for _, opt := range opts {
		switch o := opt.(type) {
		case NodeOption:
			o(node.metadata)
		case StateMapper[ParentState, ChildState]:
			node.mapToChild = o.ToChild
			node.mapFromChild = o.FromChild
		}
	}

	return node
}

// WithStateMapper 返回状态映射器选项。
//
// 参数：
//   - toChild: 父状态到子状态的映射函数
//   - fromChild: 合并子状态回父状态的函数
//
// 返回：
//   - StateMapper: 状态映射器
//
func WithStateMapper[ParentState, ChildState any](
	toChild func(ParentState) (ChildState, error),
	fromChild func(ParentState, ChildState) (ParentState, error),
) StateMapper[ParentState, ChildState] {
	return StateMapper[ParentState, ChildState]{
		ToChild:   toChild,
		FromChild: fromChild,
	}
}

// GetName 实现 Node 接口。
func (n *SubgraphNode[ParentState, ChildState]) GetName() string {
	return n.metadata.Name
}

// GetDescription 实现 Node 接口。
func (n *SubgraphNode[ParentState, ChildState]) GetDescription() string {
	return n.metadata.Description
}

// GetTags 实现 Node 接口。
func (n *SubgraphNode[ParentState, ChildState]) GetTags() []string {
	return n.metadata.Tags
}

// GetMetadata 返回节点元数据。
func (n *SubgraphNode[ParentState, ChildState]) GetMetadata() *Metadata {
	return n.metadata.Clone()
}

// Invoke 实现 Node 接口。
func (n *SubgraphNode[ParentState, ChildState]) Invoke(
	ctx context.Context,
	parentState ParentState,
) (ParentState, error) {
	// 检查上下文
	select {
	case <-ctx.Done():
		return parentState, ctx.Err()
	default:
	}

	// 映射到子状态
	if n.mapToChild == nil {
		return parentState, fmt.Errorf("subgraph node %s: mapToChild not configured", n.metadata.Name)
	}

	childState, err := n.mapToChild(parentState)
	if err != nil {
		return parentState, fmt.Errorf("subgraph node %s: failed to map to child state: %w", n.metadata.Name, err)
	}

	// 执行子图
	resultChildState, err := n.subgraph.Invoke(ctx, childState)
	if err != nil {
		return parentState, fmt.Errorf("subgraph node %s: subgraph execution failed: %w", n.metadata.Name, err)
	}

	// 映射回父状态
	if n.mapFromChild == nil {
		return parentState, fmt.Errorf("subgraph node %s: mapFromChild not configured", n.metadata.Name)
	}

	resultParentState, err := n.mapFromChild(parentState, resultChildState)
	if err != nil {
		return parentState, fmt.Errorf("subgraph node %s: failed to map from child state: %w", n.metadata.Name, err)
	}

	return resultParentState, nil
}

// Validate 实现 Node 接口。
func (n *SubgraphNode[ParentState, ChildState]) Validate() error {
	if err := n.metadata.Validate(); err != nil {
		return err
	}

	if n.subgraph == nil {
		return fmt.Errorf("subgraph node %s: subgraph is nil", n.metadata.Name)
	}

	if n.mapToChild == nil {
		return fmt.Errorf("subgraph node %s: mapToChild is nil", n.metadata.Name)
	}

	if n.mapFromChild == nil {
		return fmt.Errorf("subgraph node %s: mapFromChild is nil", n.metadata.Name)
	}

	return nil
}

// MockSubgraph 是用于测试的模拟子图执行器。
//
// 注意：仅用于测试，不应在生产代码中使用。
//
type MockSubgraph[S any] struct {
	InvokeFn func(ctx context.Context, state S, opts ...interface{}) (S, error)
}

// Invoke 实现 SubgraphExecutor 接口。
func (m *MockSubgraph[S]) Invoke(ctx context.Context, state S, opts ...interface{}) (S, error) {
	if m.InvokeFn != nil {
		return m.InvokeFn(ctx, state, opts...)
	}
	return state, nil
}
