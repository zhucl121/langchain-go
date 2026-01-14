package edge

import (
	"fmt"
)

// PathFunc 是路径函数的类型。
//
// PathFunc 根据状态返回路径名称。
//
// 类型参数：
//   - S: 状态类型
//
type PathFunc[S any] func(state S) string

// ConditionalEdge 是条件边。
//
// ConditionalEdge 根据状态动态决定下一个节点。
// 这是实现动态路由和条件分支的核心机制。
//
// 示例：
//
//	condEdge := NewConditionalEdge("agent",
//	    func(s AgentState) string {
//	        if s.Done {
//	            return "end"
//	        }
//	        if s.NeedTools {
//	            return "tools"
//	        }
//	        return "continue"
//	    },
//	    map[string]string{
//	        "tools":    "tool_node",
//	        "continue": "agent",
//	        "end":      state.END,
//	    },
//	)
//
//	nextNode := condEdge.Route(currentState)
//
type ConditionalEdge[S any] struct {
	source   string
	pathFunc PathFunc[S]
	pathMap  map[string]string
	metadata *Metadata
}

// NewConditionalEdge 创建条件边。
//
// 参数：
//   - source: 源节点名称
//   - pathFunc: 路径函数
//   - pathMap: 路径名称到目标节点的映射
//
// 返回：
//   - *ConditionalEdge[S]: 条件边实例
//
func NewConditionalEdge[S any](
	source string,
	pathFunc PathFunc[S],
	pathMap map[string]string,
) *ConditionalEdge[S] {
	return &ConditionalEdge[S]{
		source:   source,
		pathFunc: pathFunc,
		pathMap:  pathMap,
		metadata: NewMetadata(),
	}
}

// GetSource 实现 Edge 接口。
func (e *ConditionalEdge[S]) GetSource() string {
	return e.source
}

// GetType 实现 Edge 接口。
func (e *ConditionalEdge[S]) GetType() EdgeType {
	return TypeConditional
}

// GetPathMap 返回路径映射。
func (e *ConditionalEdge[S]) GetPathMap() map[string]string {
	// 返回副本
	result := make(map[string]string)
	for k, v := range e.pathMap {
		result[k] = v
	}
	return result
}

// GetMetadata 返回元数据。
func (e *ConditionalEdge[S]) GetMetadata() *Metadata {
	return e.metadata.Clone()
}

// Route 根据状态路由到下一个节点。
//
// 参数：
//   - state: 当前状态
//
// 返回：
//   - string: 目标节点名称
//   - error: 路由错误
//
func (e *ConditionalEdge[S]) Route(state S) (string, error) {
	if e.pathFunc == nil {
		return "", fmt.Errorf("conditional edge from %s: path function is nil", e.source)
	}

	// 执行路径函数
	pathName := e.pathFunc(state)

	// 查找目标节点
	target, exists := e.pathMap[pathName]
	if !exists {
		return "", fmt.Errorf("%w: path '%s' from node %s", ErrPathNotFound, pathName, e.source)
	}

	return target, nil
}

// Validate 实现 Edge 接口。
func (e *ConditionalEdge[S]) Validate() error {
	if e.source == "" {
		return ErrEmptySourceNode
	}

	if e.pathFunc == nil {
		return fmt.Errorf("conditional edge from %s: path function is nil", e.source)
	}

	if len(e.pathMap) == 0 {
		return fmt.Errorf("%w: from node %s", ErrInvalidPathMapping, e.source)
	}

	// 验证所有目标节点不为空
	for pathName, target := range e.pathMap {
		if pathName == "" {
			return fmt.Errorf("%w: empty path name", ErrEmptyPathName)
		}
		if target == "" {
			return fmt.Errorf("%w: for path '%s'", ErrEmptyTargetNode, pathName)
		}
	}

	return nil
}

// Clone 实现 Edge 接口。
func (e *ConditionalEdge[S]) Clone() Edge {
	pathMapCopy := make(map[string]string)
	for k, v := range e.pathMap {
		pathMapCopy[k] = v
	}

	return &ConditionalEdge[S]{
		source:   e.source,
		pathFunc: e.pathFunc,
		pathMap:  pathMapCopy,
		metadata: e.metadata.Clone(),
	}
}

// String 返回边的字符串表示。
func (e *ConditionalEdge[S]) String() string {
	return fmt.Sprintf("%s -?-> {%d paths}", e.source, len(e.pathMap))
}

// WithMetadata 设置元数据。
//
// 参数：
//   - meta: 元数据
//
// 返回：
//   - *ConditionalEdge[S]: 返回自身，支持链式调用
//
func (e *ConditionalEdge[S]) WithMetadata(meta *Metadata) *ConditionalEdge[S] {
	e.metadata = meta
	return e
}

// AddPath 添加路径映射。
//
// 参数：
//   - pathName: 路径名称
//   - target: 目标节点
//
// 返回：
//   - *ConditionalEdge[S]: 返回自身，支持链式调用
//
func (e *ConditionalEdge[S]) AddPath(pathName, target string) *ConditionalEdge[S] {
	if e.pathMap == nil {
		e.pathMap = make(map[string]string)
	}
	e.pathMap[pathName] = target
	return e
}

// RemovePath 移除路径映射。
//
// 参数：
//   - pathName: 路径名称
//
// 返回：
//   - *ConditionalEdge[S]: 返回自身，支持链式调用
//
func (e *ConditionalEdge[S]) RemovePath(pathName string) *ConditionalEdge[S] {
	delete(e.pathMap, pathName)
	return e
}

// BranchEdge 是分支边（并行分支）。
//
// BranchEdge 可以同时路由到多个节点（用于并行执行）。
//
// 注意：
//   - 并行执行功能将在后续实现
//   - 目前仅作为接口预留
//
type BranchEdge[S any] struct {
	source   string
	branches map[string]string // branch name -> target node
	selector func(S) []string  // 选择要执行的分支
	metadata *Metadata
}

// NewBranchEdge 创建分支边。
//
// 参数：
//   - source: 源节点名称
//   - branches: 分支映射
//   - selector: 分支选择函数
//
// 返回：
//   - *BranchEdge[S]: 分支边实例
//
func NewBranchEdge[S any](
	source string,
	branches map[string]string,
	selector func(S) []string,
) *BranchEdge[S] {
	return &BranchEdge[S]{
		source:   source,
		branches: branches,
		selector: selector,
		metadata: NewMetadata(),
	}
}

// GetSource 实现 Edge 接口。
func (e *BranchEdge[S]) GetSource() string {
	return e.source
}

// GetType 实现 Edge 接口。
func (e *BranchEdge[S]) GetType() EdgeType {
	return TypeBranch
}

// GetBranches 返回分支映射。
func (e *BranchEdge[S]) GetBranches() map[string]string {
	result := make(map[string]string)
	for k, v := range e.branches {
		result[k] = v
	}
	return result
}

// Select 选择要执行的分支。
//
// 参数：
//   - state: 当前状态
//
// 返回：
//   - []string: 目标节点列表
//   - error: 选择错误
//
func (e *BranchEdge[S]) Select(state S) ([]string, error) {
	if e.selector == nil {
		// 如果没有选择器，返回所有分支
		targets := make([]string, 0, len(e.branches))
		for _, target := range e.branches {
			targets = append(targets, target)
		}
		return targets, nil
	}

	branchNames := e.selector(state)
	targets := make([]string, 0, len(branchNames))

	for _, branchName := range branchNames {
		target, exists := e.branches[branchName]
		if !exists {
			return nil, fmt.Errorf("%w: branch '%s' from node %s", ErrPathNotFound, branchName, e.source)
		}
		targets = append(targets, target)
	}

	return targets, nil
}

// Validate 实现 Edge 接口。
func (e *BranchEdge[S]) Validate() error {
	if e.source == "" {
		return ErrEmptySourceNode
	}

	if len(e.branches) == 0 {
		return fmt.Errorf("branch edge from %s: no branches defined", e.source)
	}

	for branchName, target := range e.branches {
		if branchName == "" {
			return fmt.Errorf("%w: in branch edge", ErrEmptyPathName)
		}
		if target == "" {
			return fmt.Errorf("%w: for branch '%s'", ErrEmptyTargetNode, branchName)
		}
	}

	return nil
}

// Clone 实现 Edge 接口。
func (e *BranchEdge[S]) Clone() Edge {
	branchesCopy := make(map[string]string)
	for k, v := range e.branches {
		branchesCopy[k] = v
	}

	return &BranchEdge[S]{
		source:   e.source,
		branches: branchesCopy,
		selector: e.selector,
		metadata: e.metadata.Clone(),
	}
}

// String 返回边的字符串表示。
func (e *BranchEdge[S]) String() string {
	return fmt.Sprintf("%s -||-> {%d branches}", e.source, len(e.branches))
}
