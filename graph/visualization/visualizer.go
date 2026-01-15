package visualization

import (
	"fmt"
	"strings"
)

// GraphVisualizer 图可视化器
type GraphVisualizer struct {
	nodes           []NodeInfo
	edges           []EdgeInfo
	conditionalEdges []ConditionalEdgeInfo
	title           string
	config          VisualizerConfig
}

// NodeInfo 节点信息
type NodeInfo struct {
	ID          string
	Label       string
	Type        NodeType
	Description string
	Metadata    map[string]string
}

// EdgeInfo 边信息
type EdgeInfo struct {
	From  string
	To    string
	Label string
}

// ConditionalEdgeInfo 条件边信息
type ConditionalEdgeInfo struct {
	From    string
	Paths   map[string]string // path -> target
	Label   string
}

// NodeType 节点类型
type NodeType string

const (
	NodeTypeStart       NodeType = "start"
	NodeTypeEnd         NodeType = "end"
	NodeTypeRegular     NodeType = "regular"
	NodeTypeConditional NodeType = "conditional"
	NodeTypeSubgraph    NodeType = "subgraph"
)

// VisualizerConfig 可视化器配置
type VisualizerConfig struct {
	// Direction 图的方向: "TB" (top-bottom), "LR" (left-right), "BT", "RL"
	Direction string
	
	// Theme 主题: "default", "dark", "neutral"
	Theme string
	
	// ShowMetadata 是否显示节点元数据
	ShowMetadata bool
	
	// CompactMode 紧凑模式（减少空白）
	CompactMode bool
}

// NewGraphVisualizer 创建图可视化器
func NewGraphVisualizer(title string, config ...VisualizerConfig) *GraphVisualizer {
	cfg := VisualizerConfig{
		Direction:    "TB",
		Theme:        "default",
		ShowMetadata: false,
		CompactMode:  false,
	}
	
	if len(config) > 0 {
		cfg = config[0]
	}
	
	return &GraphVisualizer{
		nodes:           make([]NodeInfo, 0),
		edges:           make([]EdgeInfo, 0),
		conditionalEdges: make([]ConditionalEdgeInfo, 0),
		title:           title,
		config:          cfg,
	}
}

// AddNode 添加节点
func (gv *GraphVisualizer) AddNode(node NodeInfo) {
	gv.nodes = append(gv.nodes, node)
}

// AddEdge 添加边
func (gv *GraphVisualizer) AddEdge(edge EdgeInfo) {
	gv.edges = append(gv.edges, edge)
}

// AddConditionalEdge 添加条件边
func (gv *GraphVisualizer) AddConditionalEdge(edge ConditionalEdgeInfo) {
	gv.conditionalEdges = append(gv.conditionalEdges, edge)
}

// ToMermaid 导出为 Mermaid 格式
func (gv *GraphVisualizer) ToMermaid() string {
	var sb strings.Builder
	
	// 标题
	if gv.title != "" {
		sb.WriteString(fmt.Sprintf("---\ntitle: %s\n---\n", gv.title))
	}
	
	// 图定义
	sb.WriteString(fmt.Sprintf("graph %s\n", gv.config.Direction))
	
	// 节点定义
	for _, node := range gv.nodes {
		sb.WriteString(gv.formatMermaidNode(node))
	}
	
	// 边定义
	for _, edge := range gv.edges {
		if edge.Label != "" {
			sb.WriteString(fmt.Sprintf("    %s -->|%s| %s\n", edge.From, edge.Label, edge.To))
		} else {
			sb.WriteString(fmt.Sprintf("    %s --> %s\n", edge.From, edge.To))
		}
	}
	
	// 条件边定义
	for _, condEdge := range gv.conditionalEdges {
		for path, target := range condEdge.Paths {
			label := path
			if condEdge.Label != "" {
				label = fmt.Sprintf("%s: %s", condEdge.Label, path)
			}
			sb.WriteString(fmt.Sprintf("    %s -->|%s| %s\n", condEdge.From, label, target))
		}
	}
	
	// 样式
	sb.WriteString(gv.getMermaidStyles())
	
	return sb.String()
}

// formatMermaidNode 格式化 Mermaid 节点
func (gv *GraphVisualizer) formatMermaidNode(node NodeInfo) string {
	var shape string
	label := node.Label
	if label == "" {
		label = node.ID
	}
	
	// 根据节点类型选择形状
	switch node.Type {
	case NodeTypeStart:
		shape = fmt.Sprintf("    %s([%s])\n", node.ID, label)
	case NodeTypeEnd:
		shape = fmt.Sprintf("    %s([%s])\n", node.ID, label)
	case NodeTypeConditional:
		shape = fmt.Sprintf("    %s{%s}\n", node.ID, label)
	case NodeTypeSubgraph:
		shape = fmt.Sprintf("    %s[(%s)]\n", node.ID, label)
	default:
		shape = fmt.Sprintf("    %s[%s]\n", node.ID, label)
	}
	
	return shape
}

// getMermaidStyles 获取 Mermaid 样式
func (gv *GraphVisualizer) getMermaidStyles() string {
	var sb strings.Builder
	
	// 为不同类型的节点应用样式
	for _, node := range gv.nodes {
		switch node.Type {
		case NodeTypeStart:
			sb.WriteString(fmt.Sprintf("    style %s fill:#90EE90,stroke:#333,stroke-width:2px\n", node.ID))
		case NodeTypeEnd:
			sb.WriteString(fmt.Sprintf("    style %s fill:#FFB6C1,stroke:#333,stroke-width:2px\n", node.ID))
		case NodeTypeConditional:
			sb.WriteString(fmt.Sprintf("    style %s fill:#FFD700,stroke:#333,stroke-width:2px\n", node.ID))
		case NodeTypeSubgraph:
			sb.WriteString(fmt.Sprintf("    style %s fill:#87CEEB,stroke:#333,stroke-width:2px\n", node.ID))
		}
	}
	
	return sb.String()
}

// ToDOT 导出为 DOT/Graphviz 格式
func (gv *GraphVisualizer) ToDOT() string {
	var sb strings.Builder
	
	// 图头部
	sb.WriteString("digraph G {\n")
	sb.WriteString("    rankdir=" + gv.getDOTDirection() + ";\n")
	sb.WriteString("    node [shape=box, style=rounded];\n")
	sb.WriteString("    edge [fontsize=10];\n\n")
	
	// 标题
	if gv.title != "" {
		sb.WriteString(fmt.Sprintf("    label=\"%s\";\n", gv.title))
		sb.WriteString("    labelloc=t;\n")
		sb.WriteString("    fontsize=20;\n\n")
	}
	
	// 节点定义
	for _, node := range gv.nodes {
		sb.WriteString(gv.formatDOTNode(node))
	}
	
	sb.WriteString("\n")
	
	// 边定义
	for _, edge := range gv.edges {
		if edge.Label != "" {
			sb.WriteString(fmt.Sprintf("    \"%s\" -> \"%s\" [label=\"%s\"];\n", edge.From, edge.To, edge.Label))
		} else {
			sb.WriteString(fmt.Sprintf("    \"%s\" -> \"%s\";\n", edge.From, edge.To))
		}
	}
	
	// 条件边定义
	for _, condEdge := range gv.conditionalEdges {
		for path, target := range condEdge.Paths {
			label := path
			if condEdge.Label != "" {
				label = fmt.Sprintf("%s: %s", condEdge.Label, path)
			}
			sb.WriteString(fmt.Sprintf("    \"%s\" -> \"%s\" [label=\"%s\", style=dashed];\n", 
				condEdge.From, target, label))
		}
	}
	
	sb.WriteString("}\n")
	
	return sb.String()
}

// formatDOTNode 格式化 DOT 节点
func (gv *GraphVisualizer) formatDOTNode(node NodeInfo) string {
	label := node.Label
	if label == "" {
		label = node.ID
	}
	
	// 添加描述
	if node.Description != "" && gv.config.ShowMetadata {
		label = fmt.Sprintf("%s\\n%s", label, node.Description)
	}
	
	// 根据节点类型设置样式
	var attrs []string
	attrs = append(attrs, fmt.Sprintf("label=\"%s\"", label))
	
	switch node.Type {
	case NodeTypeStart:
		attrs = append(attrs, "shape=ellipse", "fillcolor=\"#90EE90\"", "style=filled")
	case NodeTypeEnd:
		attrs = append(attrs, "shape=ellipse", "fillcolor=\"#FFB6C1\"", "style=filled")
	case NodeTypeConditional:
		attrs = append(attrs, "shape=diamond", "fillcolor=\"#FFD700\"", "style=filled")
	case NodeTypeSubgraph:
		attrs = append(attrs, "shape=component", "fillcolor=\"#87CEEB\"", "style=filled")
	default:
		attrs = append(attrs, "shape=box", "style=\"rounded,filled\"", "fillcolor=\"#E0E0E0\"")
	}
	
	return fmt.Sprintf("    \"%s\" [%s];\n", node.ID, strings.Join(attrs, ", "))
}

// getDOTDirection 获取 DOT 方向
func (gv *GraphVisualizer) getDOTDirection() string {
	switch gv.config.Direction {
	case "LR":
		return "LR"
	case "RL":
		return "RL"
	case "BT":
		return "BT"
	default:
		return "TB"
	}
}

// ToASCII 导出为 ASCII 艺术格式
func (gv *GraphVisualizer) ToASCII() string {
	var sb strings.Builder
	
	if gv.title != "" {
		sb.WriteString(fmt.Sprintf("=== %s ===\n\n", gv.title))
	}
	
	// 简单的层次化显示
	sb.WriteString("Graph Structure:\n\n")
	
	// 显示节点
	sb.WriteString("Nodes:\n")
	for _, node := range gv.nodes {
		symbol := "•"
		switch node.Type {
		case NodeTypeStart:
			symbol = "▶"
		case NodeTypeEnd:
			symbol = "■"
		case NodeTypeConditional:
			symbol = "◆"
		case NodeTypeSubgraph:
			symbol = "⊞"
		}
		
		label := node.Label
		if label == "" {
			label = node.ID
		}
		
		sb.WriteString(fmt.Sprintf("  %s %s", symbol, label))
		
		if node.Description != "" && gv.config.ShowMetadata {
			sb.WriteString(fmt.Sprintf(" (%s)", node.Description))
		}
		sb.WriteString("\n")
	}
	
	sb.WriteString("\nEdges:\n")
	
	// 显示普通边
	for _, edge := range gv.edges {
		arrow := "→"
		if edge.Label != "" {
			sb.WriteString(fmt.Sprintf("  %s --%s--> %s\n", edge.From, edge.Label, edge.To))
		} else {
			sb.WriteString(fmt.Sprintf("  %s %s %s\n", edge.From, arrow, edge.To))
		}
	}
	
	// 显示条件边
	for _, condEdge := range gv.conditionalEdges {
		for path, target := range condEdge.Paths {
			label := path
			if condEdge.Label != "" {
				label = fmt.Sprintf("%s: %s", condEdge.Label, path)
			}
			sb.WriteString(fmt.Sprintf("  %s --%s--> %s (conditional)\n", condEdge.From, label, target))
		}
	}
	
	return sb.String()
}

// ToJSON 导出为 JSON 格式
func (gv *GraphVisualizer) ToJSON() string {
	var sb strings.Builder
	
	sb.WriteString("{\n")
	
	// 标题
	if gv.title != "" {
		sb.WriteString(fmt.Sprintf("  \"title\": \"%s\",\n", gv.title))
	}
	
	// 配置
	sb.WriteString("  \"config\": {\n")
	sb.WriteString(fmt.Sprintf("    \"direction\": \"%s\",\n", gv.config.Direction))
	sb.WriteString(fmt.Sprintf("    \"theme\": \"%s\"\n", gv.config.Theme))
	sb.WriteString("  },\n")
	
	// 节点
	sb.WriteString("  \"nodes\": [\n")
	for i, node := range gv.nodes {
		sb.WriteString("    {\n")
		sb.WriteString(fmt.Sprintf("      \"id\": \"%s\",\n", node.ID))
		sb.WriteString(fmt.Sprintf("      \"label\": \"%s\",\n", node.Label))
		sb.WriteString(fmt.Sprintf("      \"type\": \"%s\"", node.Type))
		if node.Description != "" {
			sb.WriteString(",\n")
			sb.WriteString(fmt.Sprintf("      \"description\": \"%s\"", node.Description))
		}
		sb.WriteString("\n    }")
		if i < len(gv.nodes)-1 {
			sb.WriteString(",")
		}
		sb.WriteString("\n")
	}
	sb.WriteString("  ],\n")
	
	// 边
	sb.WriteString("  \"edges\": [\n")
	totalEdges := len(gv.edges) + len(gv.conditionalEdges)
	edgeCount := 0
	
	for _, edge := range gv.edges {
		sb.WriteString("    {\n")
		sb.WriteString(fmt.Sprintf("      \"from\": \"%s\",\n", edge.From))
		sb.WriteString(fmt.Sprintf("      \"to\": \"%s\"", edge.To))
		if edge.Label != "" {
			sb.WriteString(",\n")
			sb.WriteString(fmt.Sprintf("      \"label\": \"%s\"", edge.Label))
		}
		sb.WriteString("\n    }")
		edgeCount++
		if edgeCount < totalEdges {
			sb.WriteString(",")
		}
		sb.WriteString("\n")
	}
	
	for _, condEdge := range gv.conditionalEdges {
		for path, target := range condEdge.Paths {
			sb.WriteString("    {\n")
			sb.WriteString(fmt.Sprintf("      \"from\": \"%s\",\n", condEdge.From))
			sb.WriteString(fmt.Sprintf("      \"to\": \"%s\",\n", target))
			sb.WriteString(fmt.Sprintf("      \"label\": \"%s\",\n", path))
			sb.WriteString("      \"type\": \"conditional\"\n")
			sb.WriteString("    }")
			edgeCount++
			if edgeCount < totalEdges {
				sb.WriteString(",")
			}
			sb.WriteString("\n")
		}
	}
	
	sb.WriteString("  ]\n")
	sb.WriteString("}\n")
	
	return sb.String()
}

// Validate 验证图结构
func (gv *GraphVisualizer) Validate() error {
	if len(gv.nodes) == 0 {
		return fmt.Errorf("graph has no nodes")
	}
	
	// 验证所有边引用的节点都存在
	nodeMap := make(map[string]bool)
	for _, node := range gv.nodes {
		nodeMap[node.ID] = true
	}
	
	for _, edge := range gv.edges {
		if !nodeMap[edge.From] {
			return fmt.Errorf("edge references unknown node: %s", edge.From)
		}
		if !nodeMap[edge.To] {
			return fmt.Errorf("edge references unknown node: %s", edge.To)
		}
	}
	
	for _, condEdge := range gv.conditionalEdges {
		if !nodeMap[condEdge.From] {
			return fmt.Errorf("conditional edge references unknown node: %s", condEdge.From)
		}
		for _, target := range condEdge.Paths {
			if !nodeMap[target] {
				return fmt.Errorf("conditional edge references unknown node: %s", target)
			}
		}
	}
	
	return nil
}
