package state

import (
	"context"
	"errors"
	"fmt"
)

// 特殊节点名称常量
const (
	// START 是图的起始点（虚拟节点）
	START = "__start__"

	// END 是图的结束点（虚拟节点）
	END = "__end__"
)

// 错误定义
var (
	ErrEmptyGraphName      = errors.New("state: graph name cannot be empty")
	ErrNodeNotFound        = errors.New("state: node not found")
	ErrNodeAlreadyExists   = errors.New("state: node already exists")
	ErrNoEntryPoint        = errors.New("state: entry point not set")
	ErrInvalidEntryPoint   = errors.New("state: invalid entry point")
	ErrInvalidEdge         = errors.New("state: invalid edge")
	ErrCyclicGraph         = errors.New("state: cyclic graph detected")
	ErrUnreachableNodes    = errors.New("state: unreachable nodes detected")
	ErrGraphNotCompiled    = errors.New("state: graph not compiled")
	ErrEmptyNodeName       = errors.New("state: node name cannot be empty")
	ErrReservedNodeName    = errors.New("state: node name is reserved")
)

// NodeFunc 是节点函数的类型。
//
// 节点函数接收当前状态，返回新状态。
// 如果返回错误，图执行会停止。
//
// 类型参数：
//   - S: 状态类型
//
type NodeFunc[S any] func(ctx context.Context, state S) (S, error)

// Node 表示图中的一个节点。
//
// Node 包含节点的名称和执行函数。
//
type Node[S any] struct {
	Name string
	Func NodeFunc[S]
}

// Edge 表示图中的一条边。
//
// Edge 连接两个节点，定义执行顺序。
//
type Edge struct {
	From string // 起始节点
	To   string // 目标节点
}

// ConditionalEdge 表示条件边。
//
// ConditionalEdge 根据状态动态决定下一个节点。
//
type ConditionalEdge[S any] struct {
	Source  string                 // 源节点
	Path    func(S) string         // 路径函数（返回路径名称）
	PathMap map[string]string      // 路径名称到节点名称的映射
}

// StateGraph 是状态图的核心类型。
//
// StateGraph 使用泛型参数 S 表示状态类型，状态在节点间流转。
// 图的定义采用声明式 API，支持链式调用。
//
// StateGraph 是 LangGraph 的 Go 实现，支持：
//   - 节点和边的定义
//   - 条件边和动态路由
//   - 检查点持久化
//   - 持久化模式配置
//   - Human-in-the-Loop
//
// 类型参数：
//   - S: 状态类型（可以是任意类型）
//
// 示例：
//
//	type MyState struct {
//	    Counter int
//	}
//
//	graph := NewStateGraph[MyState]("counter")
//	graph.AddNode("increment", func(ctx context.Context, s MyState) (MyState, error) {
//	    s.Counter++
//	    return s, nil
//	})
//	graph.SetEntryPoint("increment")
//	graph.AddEdge("increment", END)
//
//	compiled, _ := graph.Compile()
//	result, _ := compiled.Invoke(ctx, MyState{Counter: 0})
//
type StateGraph[S any] struct {
	name         string
	nodes        map[string]Node[S]
	edges        []Edge
	conditionals []ConditionalEdge[S]
	entryPoint   string
	finishPoints map[string]bool

	// LangGraph 1.0 新增功能（将在后续模块实现）
	checkpointer interface{} // checkpoint.Saver (避免循环依赖，暂时用 interface{})
	durability   interface{} // durability.Mode
	channels     map[string]interface{} // Channel
}

// NewStateGraph 创建一个新的状态图。
//
// 参数：
//   - name: 图的名称（用于日志和调试）
//
// 返回：
//   - *StateGraph[S]: 状态图实例
//
// 如果名称为空，会 panic。
//
func NewStateGraph[S any](name string) *StateGraph[S] {
	if name == "" {
		panic(ErrEmptyGraphName)
	}

	return &StateGraph[S]{
		name:         name,
		nodes:        make(map[string]Node[S]),
		edges:        make([]Edge, 0),
		conditionals: make([]ConditionalEdge[S], 0),
		finishPoints: make(map[string]bool),
		channels:     make(map[string]interface{}),
	}
}

// GetName 返回图的名称。
func (g *StateGraph[S]) GetName() string {
	return g.name
}

// AddNode 向图中添加一个节点。
//
// 参数：
//   - name: 节点名称（必须唯一）
//   - fn: 节点函数
//
// 返回：
//   - *StateGraph[S]: 返回自身，支持链式调用
//
// 注意：
//   - 如果节点名称已存在，会覆盖原有节点
//   - 节点名称不能为空
//   - 节点名称不能使用保留名称（START, END）
//
func (g *StateGraph[S]) AddNode(name string, fn NodeFunc[S]) *StateGraph[S] {
	if name == "" {
		panic(ErrEmptyNodeName)
	}

	if name == START || name == END {
		panic(fmt.Errorf("%w: %s", ErrReservedNodeName, name))
	}

	if fn == nil {
		panic(fmt.Errorf("state: node function cannot be nil for node %s", name))
	}

	g.nodes[name] = Node[S]{
		Name: name,
		Func: fn,
	}

	return g
}

// AddEdge 添加一条从 from 到 to 的边。
//
// 参数：
//   - from: 起始节点名称
//   - to: 目标节点名称（可以是 END）
//
// 返回：
//   - *StateGraph[S]: 返回自身，支持链式调用
//
// 注意：
//   - from 节点必须存在（除非是 START）
//   - to 可以是 END（表示结束）
//
func (g *StateGraph[S]) AddEdge(from, to string) *StateGraph[S] {
	if from == "" || to == "" {
		panic(ErrInvalidEdge)
	}

	// 验证 from 节点存在（START 除外）
	if from != START {
		if _, exists := g.nodes[from]; !exists {
			panic(fmt.Errorf("%w: %s", ErrNodeNotFound, from))
		}
	}

	// 验证 to 节点存在（END 除外）
	if to != END {
		if _, exists := g.nodes[to]; !exists {
			panic(fmt.Errorf("%w: %s", ErrNodeNotFound, to))
		}
	}

	g.edges = append(g.edges, Edge{
		From: from,
		To:   to,
	})

	// 如果 to 是 END，标记为结束点
	if to == END {
		g.finishPoints[from] = true
	}

	return g
}

// AddConditionalEdges 添加条件边。
//
// 条件边根据状态动态决定下一个节点。
//
// 参数：
//   - source: 源节点名称
//   - path: 路径函数，根据状态返回路径名称
//   - pathMap: 路径名称到节点名称的映射
//
// 返回：
//   - *StateGraph[S]: 返回自身，支持链式调用
//
// 示例：
//
//	graph.AddConditionalEdges("agent",
//	    func(s State) string {
//	        if s.NeedTools {
//	            return "continue"
//	        }
//	        return "end"
//	    },
//	    map[string]string{
//	        "continue": "tools",
//	        "end":      END,
//	    },
//	)
//
func (g *StateGraph[S]) AddConditionalEdges(
	source string,
	path func(S) string,
	pathMap map[string]string,
) *StateGraph[S] {
	if source == "" {
		panic(fmt.Errorf("state: source node cannot be empty"))
	}

	if path == nil {
		panic(fmt.Errorf("state: path function cannot be nil"))
	}

	if len(pathMap) == 0 {
		panic(fmt.Errorf("state: pathMap cannot be empty"))
	}

	// 验证 source 节点存在
	if _, exists := g.nodes[source]; !exists {
		panic(fmt.Errorf("%w: %s", ErrNodeNotFound, source))
	}

	// 验证 pathMap 中的所有目标节点存在（END 除外）
	for pathName, target := range pathMap {
		if target != END {
			if _, exists := g.nodes[target]; !exists {
				panic(fmt.Errorf("%w: target %s in path %s", ErrNodeNotFound, target, pathName))
			}
		} else {
			// 如果目标是 END，标记 source 为可能的结束点
			g.finishPoints[source] = true
		}
	}

	g.conditionals = append(g.conditionals, ConditionalEdge[S]{
		Source:  source,
		Path:    path,
		PathMap: pathMap,
	})

	return g
}

// SetEntryPoint 设置图的入口点。
//
// 参数：
//   - name: 入口节点名称
//
// 返回：
//   - *StateGraph[S]: 返回自身，支持链式调用
//
// 注意：
//   - 入口节点必须存在
//   - 每个图只能有一个入口点
//
func (g *StateGraph[S]) SetEntryPoint(name string) *StateGraph[S] {
	if name == "" {
		panic(fmt.Errorf("state: entry point cannot be empty"))
	}

	if name == END {
		panic(fmt.Errorf("state: entry point cannot be END"))
	}

	// 验证节点存在
	if _, exists := g.nodes[name]; !exists {
		panic(fmt.Errorf("%w: %s", ErrNodeNotFound, name))
	}

	g.entryPoint = name
	return g
}

// SetFinishPoint 设置结束点（已废弃，使用 AddEdge(node, END) 代替）。
//
// 此方法保留用于兼容性。
//
// Deprecated: 使用 AddEdge(node, END) 代替
//
func (g *StateGraph[S]) SetFinishPoint(name string) *StateGraph[S] {
	if name == "" {
		panic(fmt.Errorf("state: finish point cannot be empty"))
	}

	// 验证节点存在
	if _, exists := g.nodes[name]; !exists {
		panic(fmt.Errorf("%w: %s", ErrNodeNotFound, name))
	}

	g.finishPoints[name] = true
	return g
}

// WithCheckpointer 配置检查点保存器（LangGraph 1.0）。
//
// 参数：
//   - saver: 检查点保存器实例
//
// 返回：
//   - *StateGraph[S]: 返回自身，支持链式调用
//
// 注意：
//   - 此功能将在 M38-M42 (Checkpoint 系统) 实现后启用
//   - 目前仅保存配置，实际功能待后续模块实现
//
func (g *StateGraph[S]) WithCheckpointer(saver interface{}) *StateGraph[S] {
	g.checkpointer = saver
	return g
}

// WithDurability 配置持久化模式（LangGraph 1.0）。
//
// 参数：
//   - mode: 持久化模式（exit/async/sync）
//
// 返回：
//   - *StateGraph[S]: 返回自身，支持链式调用
//
// 注意：
//   - 此功能将在 M43-M45 (Durability 系统) 实现后启用
//   - 目前仅保存配置，实际功能待后续模块实现
//
func (g *StateGraph[S]) WithDurability(mode interface{}) *StateGraph[S] {
	g.durability = mode
	return g
}

// Compile 编译图，返回可执行的已编译图。
//
// 编译过程包括：
//   - 验证图的完整性
//   - 检查循环
//   - 检查不可达节点
//   - 构建执行计划
//
// 返回：
//   - *CompiledGraph[S]: 已编译的图
//   - error: 编译错误
//
// 注意：
//   - 图必须设置入口点
//   - 图不能包含循环（除非有条件边）
//   - 所有节点必须可达
//
func (g *StateGraph[S]) Compile() (*CompiledGraph[S], error) {
	// 验证入口点
	if g.entryPoint == "" {
		return nil, ErrNoEntryPoint
	}

	// 验证入口点存在
	if _, exists := g.nodes[g.entryPoint]; !exists {
		return nil, fmt.Errorf("%w: %s", ErrInvalidEntryPoint, g.entryPoint)
	}

	// TODO: 在 M33-M34 (Compile 系统) 中实现完整的验证逻辑
	// - 检查循环
	// - 检查不可达节点
	// - 构建拓扑排序

	// 创建已编译的图
	compiled := &CompiledGraph[S]{
		graph: g,
	}

	return compiled, nil
}

// CompiledGraph 是已编译的状态图。
//
// CompiledGraph 可以执行，支持：
//   - Invoke: 同步执行
//   - Stream: 流式执行
//   - Batch: 批量执行
//
// 注意：
//   - 完整的执行功能将在 M35-M37 (Execute 系统) 实现
//   - 目前提供基础的 Invoke 实现
//
type CompiledGraph[S any] struct {
	graph *StateGraph[S]
}

// Invoke 执行图，返回最终状态。
//
// 参数：
//   - ctx: 上下文
//   - initialState: 初始状态
//   - opts: 执行选项（可选，将在 M35 实现）
//
// 返回：
//   - S: 最终状态
//   - error: 执行错误
//
// 注意：
//   - 这是简化版实现，完整功能在 M35-M37 (Execute 系统) 实现
//   - 目前不支持 Checkpoint、HITL、Streaming 等高级功能
//
func (c *CompiledGraph[S]) Invoke(ctx context.Context, initialState S, opts ...interface{}) (S, error) {
	// 简化版执行逻辑
	// 完整实现在 M35: execute/executor.go

	state := initialState
	currentNode := c.graph.entryPoint

	// 执行循环
	for currentNode != END {
		// 检查上下文取消
		select {
		case <-ctx.Done():
			return state, ctx.Err()
		default:
		}

		// 获取当前节点
		node, exists := c.graph.nodes[currentNode]
		if !exists {
			return state, fmt.Errorf("%w: %s", ErrNodeNotFound, currentNode)
		}

		// 执行节点
		newState, err := node.Func(ctx, state)
		if err != nil {
			return state, fmt.Errorf("error executing node %s: %w", currentNode, err)
		}

		state = newState

		// 确定下一个节点
		nextNode, err := c.getNextNode(currentNode, state)
		if err != nil {
			return state, err
		}

		currentNode = nextNode
	}

	return state, nil
}

// getNextNode 确定下一个要执行的节点。
func (c *CompiledGraph[S]) getNextNode(currentNode string, state S) (string, error) {
	// 首先检查条件边
	for _, conditional := range c.graph.conditionals {
		if conditional.Source == currentNode {
			pathName := conditional.Path(state)
			if target, exists := conditional.PathMap[pathName]; exists {
				return target, nil
			}
			return "", fmt.Errorf("state: no target for path %s from node %s", pathName, currentNode)
		}
	}

	// 然后检查普通边
	for _, edge := range c.graph.edges {
		if edge.From == currentNode {
			return edge.To, nil
		}
	}

	// 如果没有找到边，且当前节点是结束点，返回 END
	if c.graph.finishPoints[currentNode] {
		return END, nil
	}

	// 没有找到出边
	return "", fmt.Errorf("state: no outgoing edge from node %s", currentNode)
}

// GetGraph 返回底层的 StateGraph（用于测试和调试）。
func (c *CompiledGraph[S]) GetGraph() *StateGraph[S] {
	return c.graph
}
