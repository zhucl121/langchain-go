package compile

import (
	"fmt"
)

// Compiler 是图编译器。
//
// Compiler 将声明式的 StateGraph 编译为可执行的 CompiledGraph。
//
// 编译过程包括：
//   - 验证图的完整性
//   - 构建执行计划
//   - 优化执行路径（可选）
//
type Compiler[S any] struct {
	validator *Validator[S]
	optimize  bool
}

// NewCompiler 创建编译器。
//
// 返回：
//   - *Compiler[S]: 编译器实例
//
func NewCompiler[S any]() *Compiler[S] {
	return &Compiler[S]{
		validator: NewValidator[S](),
		optimize:  false,
	}
}

// WithOptimization 启用优化。
//
// 参数：
//   - enabled: 是否启用优化
//
// 返回：
//   - *Compiler[S]: 返回自身，支持链式调用
//
func (c *Compiler[S]) WithOptimization(enabled bool) *Compiler[S] {
	c.optimize = enabled
	return c
}

// WithCycleCheck 启用循环检测。
//
// 参数：
//   - enabled: 是否启用循环检测
//
// 返回：
//   - *Compiler[S]: 返回自身，支持链式调用
//
func (c *Compiler[S]) WithCycleCheck(enabled bool) *Compiler[S] {
	c.validator.WithCycleCheck(enabled)
	return c
}

// Compile 编译图。
//
// 参数：
//   - graph: 要编译的图
//
// 返回：
//   - *CompiledGraph[S]: 已编译的图
//   - error: 编译错误
//
func (c *Compiler[S]) Compile(graph GraphInfo[S]) (*CompiledGraph[S], error) {
	// 1. 验证图
	if err := c.validator.Validate(graph); err != nil {
		return nil, fmt.Errorf("compilation failed: %w", err)
	}

	// 2. 构建编译结果
	compiled := &CompiledGraph[S]{
		name:         graph.GetName(),
		nodes:        graph.GetNodes(),
		edges:        graph.GetEdges(),
		conditionals: graph.GetConditionals(),
		entryPoint:   graph.GetEntryPoint(),
	}

	// 3. 优化（可选）
	if c.optimize {
		c.optimizeGraph(compiled)
	}

	// 4. 构建执行计划
	if err := c.buildExecutionPlan(compiled); err != nil {
		return nil, fmt.Errorf("failed to build execution plan: %w", err)
	}

	return compiled, nil
}

// optimizeGraph 优化图。
//
// 优化策略：
//   1. 消除冗余边（相同源和目标的重复边）
//   2. 识别死节点（永远无法到达的节点）
//   3. 识别可并行的节点（无依赖关系的节点）
//   4. 合并相同目标的条件边路径
//
func (c *Compiler[S]) optimizeGraph(compiled *CompiledGraph[S]) {
	// 1. 消除冗余边
	compiled.edges = c.deduplicateEdges(compiled.edges)

	// 2. 消除死节点
	reachableNodes := c.findReachableNodes(compiled)
	compiled.nodes = c.filterReachableNodesMap(compiled.nodes, reachableNodes)

	// 3. 识别可并行的节点并标记
	parallelGroups := c.identifyParallelGroups(compiled)
	compiled.parallelGroups = parallelGroups

	// 4. 优化条件边（合并相同目标）
	compiled.conditionals = c.optimizeConditionalEdges(compiled.conditionals)
}

// deduplicateEdges 去除重复的边。
func (c *Compiler[S]) deduplicateEdges(edges []EdgeInfo) []EdgeInfo {
	seen := make(map[string]bool)
	result := make([]EdgeInfo, 0, len(edges))

	for _, edge := range edges {
		key := fmt.Sprintf("%s->%s", edge.From, edge.To)
		if !seen[key] {
			seen[key] = true
			result = append(result, edge)
		}
	}

	return result
}

// findReachableNodes 找到所有可达的节点。
func (c *Compiler[S]) findReachableNodes(compiled *CompiledGraph[S]) map[string]bool {
	reachable := make(map[string]bool)
	visited := make(map[string]bool)

	// 从入口点开始 DFS
	var dfs func(node string)
	dfs = func(node string) {
		if visited[node] {
			return
		}
		visited[node] = true
		reachable[node] = true

		// 遍历所有出边
		for _, edge := range compiled.edges {
			if edge.From == node {
				dfs(edge.To)
			}
		}

		// 遍历所有条件边
		for _, cond := range compiled.conditionals {
			if cond.Source == node {
				for _, target := range cond.PathMap {
					dfs(target)
				}
			}
		}
	}

	dfs(compiled.entryPoint)
	return reachable
}

// filterReachableNodesMap 过滤只保留可达的节点（map版本）。
func (c *Compiler[S]) filterReachableNodesMap(nodes map[string]NodeInfo, reachable map[string]bool) map[string]NodeInfo {
	result := make(map[string]NodeInfo)
	for name, node := range nodes {
		if reachable[name] {
			result[name] = node
		}
	}
	return result
}

// identifyParallelGroups 识别可并行执行的节点组。
//
// 两个节点可以并行执行的条件：
//   - 它们有相同的前驱节点
//   - 它们之间没有依赖关系
//
func (c *Compiler[S]) identifyParallelGroups(compiled *CompiledGraph[S]) [][]string {
	// 构建前驱映射
	predecessors := make(map[string][]string)
	successors := make(map[string][]string)

	for _, edge := range compiled.edges {
		predecessors[edge.To] = append(predecessors[edge.To], edge.From)
		successors[edge.From] = append(successors[edge.From], edge.To)
	}

	for _, cond := range compiled.conditionals {
		for _, target := range cond.PathMap {
			predecessors[target] = append(predecessors[target], cond.Source)
			successors[cond.Source] = append(successors[cond.Source], target)
		}
	}

	// 找到具有相同前驱的节点
	groups := make([][]string, 0)
	processed := make(map[string]bool)

	for node, preds := range predecessors {
		if processed[node] || len(preds) == 0 {
			continue
		}

		// 找到具有相同前驱的其他节点
		group := []string{node}
		predSet := makeSet(preds)

		for otherNode, otherPreds := range predecessors {
			if otherNode == node || processed[otherNode] {
				continue
			}

			otherSet := makeSet(otherPreds)
			if setsEqual(predSet, otherSet) {
				// 检查两个节点之间是否有依赖
				if !c.hasDependency(node, otherNode, successors) &&
					!c.hasDependency(otherNode, node, successors) {
					group = append(group, otherNode)
				}
			}
		}

		// 如果组有多个节点，添加到结果
		if len(group) > 1 {
			for _, n := range group {
				processed[n] = true
			}
			groups = append(groups, group)
		}
	}

	return groups
}

// hasDependency 检查节点 from 是否依赖于节点 to。
func (c *Compiler[S]) hasDependency(from, to string, successors map[string][]string) bool {
	visited := make(map[string]bool)

	var dfs func(node string) bool
	dfs = func(node string) bool {
		if node == to {
			return true
		}
		if visited[node] {
			return false
		}
		visited[node] = true

		for _, next := range successors[node] {
			if dfs(next) {
				return true
			}
		}
		return false
	}

	return dfs(from)
}

// optimizeConditionalEdges 优化条件边（合并相同目标）。
func (c *Compiler[S]) optimizeConditionalEdges(conditionals []ConditionalInfo[S]) []ConditionalInfo[S] {
	// 简单实现：暂时返回原样
	// 可以添加更复杂的优化，如合并相同源和目标的条件
	return conditionals
}

// makeSet 创建字符串集合。
func makeSet(items []string) map[string]bool {
	set := make(map[string]bool)
	for _, item := range items {
		set[item] = true
	}
	return set
}

// setsEqual 检查两个集合是否相等。
func setsEqual(set1, set2 map[string]bool) bool {
	if len(set1) != len(set2) {
		return false
	}
	for key := range set1 {
		if !set2[key] {
			return false
		}
	}
	return true
}

// buildExecutionPlan 构建执行计划。
func (c *Compiler[S]) buildExecutionPlan(compiled *CompiledGraph[S]) error {
	// 构建邻接表（用于快速查找出边）
	adjacency := make(map[string][]string)

	for _, edge := range compiled.edges {
		adjacency[edge.From] = append(adjacency[edge.From], edge.To)
	}

	for _, cond := range compiled.conditionals {
		// 条件边的所有可能目标
		for _, target := range cond.PathMap {
			adjacency[cond.Source] = append(adjacency[cond.Source], target)
		}
	}

	compiled.adjacency = adjacency
	return nil
}

// CompiledGraph 是已编译的图。
//
// CompiledGraph 包含经过验证和优化的图结构，可以高效执行。
//
type CompiledGraph[S any] struct {
	name         string
	nodes        map[string]NodeInfo
	edges        []EdgeInfo
	conditionals []ConditionalInfo[S]
	entryPoint   string

	// 执行计划
	adjacency map[string][]string // 邻接表

	// 优化信息
	parallelGroups [][]string // 可并行执行的节点组
}

// GetName 返回图名称。
func (c *CompiledGraph[S]) GetName() string {
	return c.name
}

// GetEntryPoint 返回入口点。
func (c *CompiledGraph[S]) GetEntryPoint() string {
	return c.entryPoint
}

// GetNodes 返回节点信息。
func (c *CompiledGraph[S]) GetNodes() map[string]NodeInfo {
	return c.nodes
}

// GetEdges 返回边信息。
func (c *CompiledGraph[S]) GetEdges() []EdgeInfo {
	return c.edges
}

// GetAdjacency 返回邻接表。
func (c *CompiledGraph[S]) GetAdjacency() map[string][]string {
	return c.adjacency
}

// String 返回图的字符串表示。
func (c *CompiledGraph[S]) String() string {
	return fmt.Sprintf("CompiledGraph{name=%s, nodes=%d, edges=%d}",
		c.name, len(c.nodes), len(c.edges))
}
