package visualization

import (
	"fmt"
	"strings"
)

// GraphAdapter 图适配器接口
// 用于从不同的图结构提取可视化信息
type GraphAdapter interface {
	GetNodes() []NodeInfo
	GetEdges() []EdgeInfo
	GetConditionalEdges() []ConditionalEdgeInfo
	GetTitle() string
}

// FromAdapter 从适配器创建可视化器
func FromAdapter(adapter GraphAdapter, config ...VisualizerConfig) *GraphVisualizer {
	gv := NewGraphVisualizer(adapter.GetTitle(), config...)
	
	for _, node := range adapter.GetNodes() {
		gv.AddNode(node)
	}
	
	for _, edge := range adapter.GetEdges() {
		gv.AddEdge(edge)
	}
	
	for _, condEdge := range adapter.GetConditionalEdges() {
		gv.AddConditionalEdge(condEdge)
	}
	
	return gv
}

// SimpleGraphBuilder 简单图构建器
// 用于手动构建图结构进行可视化
type SimpleGraphBuilder struct {
	title            string
	nodes            []NodeInfo
	edges            []EdgeInfo
	conditionalEdges []ConditionalEdgeInfo
}

// NewSimpleGraphBuilder 创建简单图构建器
func NewSimpleGraphBuilder(title string) *SimpleGraphBuilder {
	return &SimpleGraphBuilder{
		title:            title,
		nodes:            make([]NodeInfo, 0),
		edges:            make([]EdgeInfo, 0),
		conditionalEdges: make([]ConditionalEdgeInfo, 0),
	}
}

// AddNode 添加节点
func (sgb *SimpleGraphBuilder) AddNode(id, label string, nodeType NodeType) *SimpleGraphBuilder {
	sgb.nodes = append(sgb.nodes, NodeInfo{
		ID:    id,
		Label: label,
		Type:  nodeType,
	})
	return sgb
}

// AddNodeWithDescription 添加带描述的节点
func (sgb *SimpleGraphBuilder) AddNodeWithDescription(id, label, description string, nodeType NodeType) *SimpleGraphBuilder {
	sgb.nodes = append(sgb.nodes, NodeInfo{
		ID:          id,
		Label:       label,
		Type:        nodeType,
		Description: description,
	})
	return sgb
}

// AddEdge 添加边
func (sgb *SimpleGraphBuilder) AddEdge(from, to string, label ...string) *SimpleGraphBuilder {
	edge := EdgeInfo{
		From: from,
		To:   to,
	}
	if len(label) > 0 {
		edge.Label = label[0]
	}
	sgb.edges = append(sgb.edges, edge)
	return sgb
}

// AddConditionalEdge 添加条件边
func (sgb *SimpleGraphBuilder) AddConditionalEdge(from string, paths map[string]string, label ...string) *SimpleGraphBuilder {
	condEdge := ConditionalEdgeInfo{
		From:  from,
		Paths: paths,
	}
	if len(label) > 0 {
		condEdge.Label = label[0]
	}
	sgb.conditionalEdges = append(sgb.conditionalEdges, condEdge)
	return sgb
}

// Build 构建可视化器
func (sgb *SimpleGraphBuilder) Build(config ...VisualizerConfig) *GraphVisualizer {
	gv := NewGraphVisualizer(sgb.title, config...)
	
	for _, node := range sgb.nodes {
		gv.AddNode(node)
	}
	
	for _, edge := range sgb.edges {
		gv.AddEdge(edge)
	}
	
	for _, condEdge := range sgb.conditionalEdges {
		gv.AddConditionalEdge(condEdge)
	}
	
	return gv
}

// GetNodes 实现 GraphAdapter 接口
func (sgb *SimpleGraphBuilder) GetNodes() []NodeInfo {
	return sgb.nodes
}

// GetEdges 实现 GraphAdapter 接口
func (sgb *SimpleGraphBuilder) GetEdges() []EdgeInfo {
	return sgb.edges
}

// GetConditionalEdges 实现 GraphAdapter 接口
func (sgb *SimpleGraphBuilder) GetConditionalEdges() []ConditionalEdgeInfo {
	return sgb.conditionalEdges
}

// GetTitle 实现 GraphAdapter 接口
func (sgb *SimpleGraphBuilder) GetTitle() string {
	return sgb.title
}

// ExecutionTracer 执行追踪器
// 用于可视化图的执行路径
type ExecutionTracer struct {
	visualizer *GraphVisualizer
	path       []string
	current    string
}

// NewExecutionTracer 创建执行追踪器
func NewExecutionTracer(gv *GraphVisualizer) *ExecutionTracer {
	return &ExecutionTracer{
		visualizer: gv,
		path:       make([]string, 0),
	}
}

// Visit 访问节点
func (et *ExecutionTracer) Visit(nodeID string) {
	et.path = append(et.path, nodeID)
	et.current = nodeID
}

// GetPath 获取执行路径
func (et *ExecutionTracer) GetPath() []string {
	return et.path
}

// ToMermaidWithPath 导出带执行路径的 Mermaid
func (et *ExecutionTracer) ToMermaidWithPath() string {
	base := et.visualizer.ToMermaid()
	
	// 添加路径高亮
	var styles strings.Builder
	for i, nodeID := range et.path {
		opacity := float64(i+1) / float64(len(et.path))
		color := fmt.Sprintf("#FF%02X%02X", 
			255-int(opacity*100), 
			255-int(opacity*100))
		styles.WriteString(fmt.Sprintf("    style %s fill:%s,stroke:#FF0000,stroke-width:3px\n", 
			nodeID, color))
	}
	
	return base + styles.String()
}

// VisualizationFormat 可视化格式
type VisualizationFormat string

const (
	FormatMermaid VisualizationFormat = "mermaid"
	FormatDOT     VisualizationFormat = "dot"
	FormatASCII   VisualizationFormat = "ascii"
	FormatJSON    VisualizationFormat = "json"
)

// Export 导出为指定格式
func Export(gv *GraphVisualizer, format VisualizationFormat) (string, error) {
	if err := gv.Validate(); err != nil {
		return "", fmt.Errorf("validation failed: %w", err)
	}
	
	switch format {
	case FormatMermaid:
		return gv.ToMermaid(), nil
	case FormatDOT:
		return gv.ToDOT(), nil
	case FormatASCII:
		return gv.ToASCII(), nil
	case FormatJSON:
		return gv.ToJSON(), nil
	default:
		return "", fmt.Errorf("unsupported format: %s", format)
	}
}
