package compile

import (
	"errors"
	"fmt"
)

// 错误定义
var (
	ErrNoEntryPoint       = errors.New("compile: no entry point set")
	ErrInvalidEntryPoint  = errors.New("compile: invalid entry point")
	ErrNodeNotFound       = errors.New("compile: node not found")
	ErrUnreachableNode    = errors.New("compile: unreachable node detected")
	ErrNoOutgoingEdge     = errors.New("compile: node has no outgoing edge")
	ErrCyclicGraph        = errors.New("compile: cyclic graph detected")
	ErrDanglingEdge       = errors.New("compile: dangling edge")
	ErrInvalidConditional = errors.New("compile: invalid conditional edge")
)

// ValidationError 是验证错误。
type ValidationError struct {
	Message string
	Details []string
}

// Error 实现 error 接口。
func (e *ValidationError) Error() string {
	if len(e.Details) == 0 {
		return e.Message
	}
	return fmt.Sprintf("%s: %v", e.Message, e.Details)
}

// Validator 是图验证器。
//
// Validator 负责验证状态图的完整性和合法性。
//
// 验证项目：
//   - 入口点检查
//   - 节点完整性
//   - 边有效性
//   - 可达性分析
//   - 循环检测（可选）
//
type Validator[S any] struct {
	checkCycles bool // 是否检测循环（默认 false，因为合法的图可以有循环）
}

// NewValidator 创建验证器。
//
// 返回：
//   - *Validator[S]: 验证器实例
//
func NewValidator[S any]() *Validator[S] {
	return &Validator[S]{
		checkCycles: false,
	}
}

// WithCycleCheck 启用循环检测。
//
// 注意：
//   - 合法的状态图可以包含循环（通过条件边控制退出）
//   - 仅在需要检测无条件循环时启用
//
func (v *Validator[S]) WithCycleCheck(enabled bool) *Validator[S] {
	v.checkCycles = enabled
	return v
}

// GraphInfo 是图的信息接口（避免循环依赖）。
type GraphInfo[S any] interface {
	GetName() string
	GetNodes() map[string]NodeInfo
	GetEdges() []EdgeInfo
	GetConditionals() []ConditionalInfo[S]
	GetEntryPoint() string
}

// NodeInfo 是节点信息。
type NodeInfo struct {
	Name string
}

// EdgeInfo 是边信息。
type EdgeInfo struct {
	From string
	To   string
}

// ConditionalInfo 是条件边信息。
type ConditionalInfo[S any] struct {
	Source  string
	PathMap map[string]string
}

// Validate 验证图。
//
// 参数：
//   - graph: 要验证的图
//
// 返回：
//   - error: 验证错误
//
func (v *Validator[S]) Validate(graph GraphInfo[S]) error {
	var errors []string

	// 1. 检查入口点
	if err := v.validateEntryPoint(graph); err != nil {
		errors = append(errors, err.Error())
	}

	// 2. 检查节点
	if err := v.validateNodes(graph); err != nil {
		errors = append(errors, err.Error())
	}

	// 3. 检查边
	if err := v.validateEdges(graph); err != nil {
		errors = append(errors, err.Error())
	}

	// 4. 检查条件边
	if err := v.validateConditionals(graph); err != nil {
		errors = append(errors, err.Error())
	}

	// 5. 检查可达性
	if err := v.validateReachability(graph); err != nil {
		errors = append(errors, err.Error())
	}

	// 6. 检查循环（可选）
	if v.checkCycles {
		if err := v.validateNoCycles(graph); err != nil {
			errors = append(errors, err.Error())
		}
	}

	if len(errors) > 0 {
		return &ValidationError{
			Message: "graph validation failed",
			Details: errors,
		}
	}

	return nil
}

// validateEntryPoint 验证入口点。
func (v *Validator[S]) validateEntryPoint(graph GraphInfo[S]) error {
	entryPoint := graph.GetEntryPoint()

	if entryPoint == "" {
		return ErrNoEntryPoint
	}

	nodes := graph.GetNodes()
	if _, exists := nodes[entryPoint]; !exists {
		return fmt.Errorf("%w: %s", ErrInvalidEntryPoint, entryPoint)
	}

	return nil
}

// validateNodes 验证节点。
func (v *Validator[S]) validateNodes(graph GraphInfo[S]) error {
	nodes := graph.GetNodes()

	if len(nodes) == 0 {
		return fmt.Errorf("graph has no nodes")
	}

	return nil
}

// validateEdges 验证边。
func (v *Validator[S]) validateEdges(graph GraphInfo[S]) error {
	nodes := graph.GetNodes()
	edges := graph.GetEdges()

	for _, edge := range edges {
		// 验证源节点存在
		if _, exists := nodes[edge.From]; !exists {
			return fmt.Errorf("%w: edge from %s to %s", ErrNodeNotFound, edge.From, edge.To)
		}

		// 验证目标节点存在（END 除外）
		if edge.To != "__end__" {
			if _, exists := nodes[edge.To]; !exists {
				return fmt.Errorf("%w: edge from %s to %s", ErrNodeNotFound, edge.From, edge.To)
			}
		}
	}

	return nil
}

// validateConditionals 验证条件边。
func (v *Validator[S]) validateConditionals(graph GraphInfo[S]) error {
	nodes := graph.GetNodes()
	conditionals := graph.GetConditionals()

	for _, cond := range conditionals {
		// 验证源节点存在
		if _, exists := nodes[cond.Source]; !exists {
			return fmt.Errorf("%w: conditional edge from %s", ErrNodeNotFound, cond.Source)
		}

		// 验证所有目标节点存在（END 除外）
		for pathName, target := range cond.PathMap {
			if target != "__end__" {
				if _, exists := nodes[target]; !exists {
					return fmt.Errorf("%w: conditional target %s (path %s from %s)",
						ErrNodeNotFound, target, pathName, cond.Source)
				}
			}
		}
	}

	return nil
}

// validateReachability 验证可达性。
func (v *Validator[S]) validateReachability(graph GraphInfo[S]) error {
	nodes := graph.GetNodes()
	entryPoint := graph.GetEntryPoint()

	// 构建可达节点集合
	reachable := make(map[string]bool)
	v.markReachable(graph, entryPoint, reachable)

	// 检查是否有不可达节点
	unreachable := make([]string, 0)
	for nodeName := range nodes {
		if !reachable[nodeName] {
			unreachable = append(unreachable, nodeName)
		}
	}

	if len(unreachable) > 0 {
		return fmt.Errorf("%w: %v", ErrUnreachableNode, unreachable)
	}

	return nil
}

// markReachable 标记从给定节点可达的所有节点（DFS）。
func (v *Validator[S]) markReachable(graph GraphInfo[S], nodeName string, reachable map[string]bool) {
	if reachable[nodeName] {
		return // 已访问
	}

	reachable[nodeName] = true

	// 遍历普通边
	edges := graph.GetEdges()
	for _, edge := range edges {
		if edge.From == nodeName && edge.To != "__end__" {
			v.markReachable(graph, edge.To, reachable)
		}
	}

	// 遍历条件边
	conditionals := graph.GetConditionals()
	for _, cond := range conditionals {
		if cond.Source == nodeName {
			for _, target := range cond.PathMap {
				if target != "__end__" {
					v.markReachable(graph, target, reachable)
				}
			}
		}
	}
}

// validateNoCycles 验证无循环（严格模式）。
//
// 注意：此检查默认禁用，因为合法的图可以有循环（通过条件边控制）。
//
func (v *Validator[S]) validateNoCycles(graph GraphInfo[S]) error {
	// 使用 DFS 检测循环
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	nodes := graph.GetNodes()
	for nodeName := range nodes {
		if !visited[nodeName] {
			if v.hasCycleDFS(graph, nodeName, visited, recStack) {
				return fmt.Errorf("%w: cycle detected involving node %s", ErrCyclicGraph, nodeName)
			}
		}
	}

	return nil
}

// hasCycleDFS 使用 DFS 检测循环。
func (v *Validator[S]) hasCycleDFS(
	graph GraphInfo[S],
	nodeName string,
	visited map[string]bool,
	recStack map[string]bool,
) bool {
	visited[nodeName] = true
	recStack[nodeName] = true

	// 检查普通边
	edges := graph.GetEdges()
	for _, edge := range edges {
		if edge.From == nodeName && edge.To != "__end__" {
			if !visited[edge.To] {
				if v.hasCycleDFS(graph, edge.To, visited, recStack) {
					return true
				}
			} else if recStack[edge.To] {
				return true // 发现循环
			}
		}
	}

	// 检查条件边
	conditionals := graph.GetConditionals()
	for _, cond := range conditionals {
		if cond.Source == nodeName {
			for _, target := range cond.PathMap {
				if target != "__end__" {
					if !visited[target] {
						if v.hasCycleDFS(graph, target, visited, recStack) {
							return true
						}
					} else if recStack[target] {
						return true // 发现循环
					}
				}
			}
		}
	}

	recStack[nodeName] = false
	return false
}

// ValidateQuick 快速验证（仅基础检查）。
//
// 参数：
//   - graph: 要验证的图
//
// 返回：
//   - error: 验证错误
//
func ValidateQuick[S any](graph GraphInfo[S]) error {
	validator := NewValidator[S]()
	validator.checkCycles = false

	// 仅做基础验证，跳过可达性和循环检测
	if err := validator.validateEntryPoint(graph); err != nil {
		return err
	}

	if err := validator.validateNodes(graph); err != nil {
		return err
	}

	if err := validator.validateEdges(graph); err != nil {
		return err
	}

	if err := validator.validateConditionals(graph); err != nil {
		return err
	}

	return nil
}
