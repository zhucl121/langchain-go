package visualization

import (
	"strings"
	"testing"
)

// TestGraphVisualizerBasic 测试基础功能
func TestGraphVisualizerBasic(t *testing.T) {
	gv := NewGraphVisualizer("Test Graph")
	
	// 添加节点
	gv.AddNode(NodeInfo{
		ID:    "start",
		Label: "Start",
		Type:  NodeTypeStart,
	})
	gv.AddNode(NodeInfo{
		ID:    "process",
		Label: "Process Data",
		Type:  NodeTypeRegular,
	})
	gv.AddNode(NodeInfo{
		ID:    "end",
		Label: "End",
		Type:  NodeTypeEnd,
	})
	
	// 添加边
	gv.AddEdge(EdgeInfo{From: "start", To: "process"})
	gv.AddEdge(EdgeInfo{From: "process", To: "end"})
	
	// 验证
	if err := gv.Validate(); err != nil {
		t.Errorf("Validation failed: %v", err)
	}
}

// TestMermaidOutput 测试 Mermaid 输出
func TestMermaidOutput(t *testing.T) {
	gv := NewGraphVisualizer("Mermaid Test", VisualizerConfig{
		Direction: "LR",
	})
	
	gv.AddNode(NodeInfo{ID: "A", Label: "Node A", Type: NodeTypeStart})
	gv.AddNode(NodeInfo{ID: "B", Label: "Node B", Type: NodeTypeRegular})
	gv.AddNode(NodeInfo{ID: "C", Label: "Node C", Type: NodeTypeEnd})
	
	gv.AddEdge(EdgeInfo{From: "A", To: "B", Label: "step1"})
	gv.AddEdge(EdgeInfo{From: "B", To: "C"})
	
	output := gv.ToMermaid()
	
	// 验证输出包含关键元素
	if !strings.Contains(output, "graph LR") {
		t.Error("Expected 'graph LR' in output")
	}
	
	if !strings.Contains(output, "A([Node A])") {
		t.Error("Expected start node definition")
	}
	
	if !strings.Contains(output, "A -->|step1| B") {
		t.Error("Expected edge with label")
	}
	
	if !strings.Contains(output, "style A fill:#90EE90") {
		t.Error("Expected start node style")
	}
}

// TestDOTOutput 测试 DOT 输出
func TestDOTOutput(t *testing.T) {
	gv := NewGraphVisualizer("DOT Test")
	
	gv.AddNode(NodeInfo{ID: "start", Label: "Start", Type: NodeTypeStart})
	gv.AddNode(NodeInfo{ID: "process", Label: "Process", Type: NodeTypeRegular})
	gv.AddNode(NodeInfo{ID: "end", Label: "End", Type: NodeTypeEnd})
	
	gv.AddEdge(EdgeInfo{From: "start", To: "process"})
	gv.AddEdge(EdgeInfo{From: "process", To: "end"})
	
	output := gv.ToDOT()
	
	// 验证输出包含关键元素
	if !strings.Contains(output, "digraph G") {
		t.Error("Expected 'digraph G' in output")
	}
	
	if !strings.Contains(output, "rankdir=TB") {
		t.Error("Expected rankdir in output")
	}
	
	if !strings.Contains(output, "\"start\" -> \"process\"") {
		t.Error("Expected edge definition")
	}
	
	if !strings.Contains(output, "shape=ellipse") {
		t.Error("Expected start node shape")
	}
}

// TestASCIIOutput 测试 ASCII 输出
func TestASCIIOutput(t *testing.T) {
	gv := NewGraphVisualizer("ASCII Test")
	
	gv.AddNode(NodeInfo{ID: "start", Label: "Start", Type: NodeTypeStart})
	gv.AddNode(NodeInfo{ID: "process", Label: "Process", Type: NodeTypeRegular})
	
	gv.AddEdge(EdgeInfo{From: "start", To: "process"})
	
	output := gv.ToASCII()
	
	// 验证输出包含关键元素
	if !strings.Contains(output, "=== ASCII Test ===") {
		t.Error("Expected title in output")
	}
	
	if !strings.Contains(output, "Nodes:") {
		t.Error("Expected 'Nodes:' section")
	}
	
	if !strings.Contains(output, "Edges:") {
		t.Error("Expected 'Edges:' section")
	}
	
	if !strings.Contains(output, "Start") {
		t.Error("Expected node label")
	}
}

// TestJSONOutput 测试 JSON 输出
func TestJSONOutput(t *testing.T) {
	gv := NewGraphVisualizer("JSON Test")
	
	gv.AddNode(NodeInfo{ID: "A", Label: "Node A", Type: NodeTypeRegular})
	gv.AddEdge(EdgeInfo{From: "A", To: "B"})
	
	output := gv.ToJSON()
	
	// 验证输出包含关键元素
	if !strings.Contains(output, "\"title\": \"JSON Test\"") {
		t.Error("Expected title in JSON")
	}
	
	if !strings.Contains(output, "\"nodes\":") {
		t.Error("Expected nodes array")
	}
	
	if !strings.Contains(output, "\"edges\":") {
		t.Error("Expected edges array")
	}
}

// TestConditionalEdges 测试条件边
func TestConditionalEdges(t *testing.T) {
	gv := NewGraphVisualizer("Conditional Test")
	
	gv.AddNode(NodeInfo{ID: "start", Type: NodeTypeStart})
	gv.AddNode(NodeInfo{ID: "decision", Type: NodeTypeConditional})
	gv.AddNode(NodeInfo{ID: "pathA", Label: "Path A", Type: NodeTypeRegular})
	gv.AddNode(NodeInfo{ID: "pathB", Label: "Path B", Type: NodeTypeRegular})
	gv.AddNode(NodeInfo{ID: "end", Type: NodeTypeEnd})
	
	gv.AddEdge(EdgeInfo{From: "start", To: "decision"})
	gv.AddConditionalEdge(ConditionalEdgeInfo{
		From: "decision",
		Paths: map[string]string{
			"yes": "pathA",
			"no":  "pathB",
		},
		Label: "choice",
	})
	gv.AddEdge(EdgeInfo{From: "pathA", To: "end"})
	gv.AddEdge(EdgeInfo{From: "pathB", To: "end"})
	
	// 测试 Mermaid 输出
	mermaid := gv.ToMermaid()
	if !strings.Contains(mermaid, "decision{") {
		t.Error("Expected diamond shape for conditional node")
	}
	
	// 测试 DOT 输出
	dot := gv.ToDOT()
	if !strings.Contains(dot, "style=dashed") {
		t.Error("Expected dashed style for conditional edges")
	}
	
	// 验证
	if err := gv.Validate(); err != nil {
		t.Errorf("Validation failed: %v", err)
	}
}

// TestSimpleGraphBuilder 测试简单图构建器
func TestSimpleGraphBuilder(t *testing.T) {
	builder := NewSimpleGraphBuilder("Builder Test")
	
	gv := builder.
		AddNode("start", "Start", NodeTypeStart).
		AddNode("process", "Process Data", NodeTypeRegular).
		AddNode("end", "End", NodeTypeEnd).
		AddEdge("start", "process", "begin").
		AddEdge("process", "end").
		Build()
	
	if err := gv.Validate(); err != nil {
		t.Errorf("Validation failed: %v", err)
	}
	
	output := gv.ToMermaid()
	if !strings.Contains(output, "Start") {
		t.Error("Expected 'Start' in output")
	}
}

// TestValidation 测试验证功能
func TestValidation(t *testing.T) {
	t.Run("empty graph", func(t *testing.T) {
		gv := NewGraphVisualizer("Empty")
		err := gv.Validate()
		if err == nil {
			t.Error("Expected error for empty graph")
		}
	})
	
	t.Run("invalid edge reference", func(t *testing.T) {
		gv := NewGraphVisualizer("Invalid Edge")
		gv.AddNode(NodeInfo{ID: "A", Type: NodeTypeRegular})
		gv.AddEdge(EdgeInfo{From: "A", To: "B"}) // B doesn't exist
		
		err := gv.Validate()
		if err == nil {
			t.Error("Expected error for invalid edge reference")
		}
	})
	
	t.Run("valid graph", func(t *testing.T) {
		gv := NewGraphVisualizer("Valid")
		gv.AddNode(NodeInfo{ID: "A", Type: NodeTypeRegular})
		gv.AddNode(NodeInfo{ID: "B", Type: NodeTypeRegular})
		gv.AddEdge(EdgeInfo{From: "A", To: "B"})
		
		err := gv.Validate()
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	})
}

// TestNodeTypes 测试不同节点类型
func TestNodeTypes(t *testing.T) {
	gv := NewGraphVisualizer("Node Types Test")
	
	nodeTypes := []struct {
		id   string
		typ  NodeType
		name string
	}{
		{"start", NodeTypeStart, "Start Node"},
		{"end", NodeTypeEnd, "End Node"},
		{"regular", NodeTypeRegular, "Regular Node"},
		{"conditional", NodeTypeConditional, "Conditional Node"},
		{"subgraph", NodeTypeSubgraph, "Subgraph Node"},
	}
	
	for _, nt := range nodeTypes {
		gv.AddNode(NodeInfo{
			ID:    nt.id,
			Label: nt.name,
			Type:  nt.typ,
		})
	}
	
	// 测试 Mermaid 输出包含所有节点
	mermaid := gv.ToMermaid()
	for _, nt := range nodeTypes {
		if !strings.Contains(mermaid, nt.id) {
			t.Errorf("Expected node %s in Mermaid output", nt.id)
		}
	}
	
	// 测试 DOT 输出包含所有节点
	dot := gv.ToDOT()
	for _, nt := range nodeTypes {
		if !strings.Contains(dot, nt.id) {
			t.Errorf("Expected node %s in DOT output", nt.id)
		}
	}
}

// TestExport 测试导出功能
func TestExport(t *testing.T) {
	gv := NewGraphVisualizer("Export Test")
	gv.AddNode(NodeInfo{ID: "A", Type: NodeTypeRegular})
	gv.AddNode(NodeInfo{ID: "B", Type: NodeTypeRegular})
	gv.AddEdge(EdgeInfo{From: "A", To: "B"})
	
	formats := []VisualizationFormat{
		FormatMermaid,
		FormatDOT,
		FormatASCII,
		FormatJSON,
	}
	
	for _, format := range formats {
		output, err := Export(gv, format)
		if err != nil {
			t.Errorf("Export failed for format %s: %v", format, err)
		}
		if output == "" {
			t.Errorf("Empty output for format %s", format)
		}
	}
}

// TestExecutionTracer 测试执行追踪器
func TestExecutionTracer(t *testing.T) {
	gv := NewGraphVisualizer("Tracer Test")
	gv.AddNode(NodeInfo{ID: "A", Type: NodeTypeStart})
	gv.AddNode(NodeInfo{ID: "B", Type: NodeTypeRegular})
	gv.AddNode(NodeInfo{ID: "C", Type: NodeTypeEnd})
	gv.AddEdge(EdgeInfo{From: "A", To: "B"})
	gv.AddEdge(EdgeInfo{From: "B", To: "C"})
	
	tracer := NewExecutionTracer(gv)
	tracer.Visit("A")
	tracer.Visit("B")
	tracer.Visit("C")
	
	path := tracer.GetPath()
	if len(path) != 3 {
		t.Errorf("Expected path length 3, got %d", len(path))
	}
	
	if path[0] != "A" || path[1] != "B" || path[2] != "C" {
		t.Error("Incorrect execution path")
	}
	
	output := tracer.ToMermaidWithPath()
	if !strings.Contains(output, "style A fill:#FF") {
		t.Error("Expected path highlighting in output")
	}
}

// TestConfigurations 测试不同配置
func TestConfigurations(t *testing.T) {
	t.Run("LR direction", func(t *testing.T) {
		gv := NewGraphVisualizer("LR Test", VisualizerConfig{
			Direction: "LR",
		})
		gv.AddNode(NodeInfo{ID: "A", Type: NodeTypeRegular})
		
		mermaid := gv.ToMermaid()
		if !strings.Contains(mermaid, "graph LR") {
			t.Error("Expected LR direction")
		}
		
		dot := gv.ToDOT()
		if !strings.Contains(dot, "rankdir=LR") {
			t.Error("Expected LR rankdir in DOT")
		}
	})
	
	t.Run("with metadata", func(t *testing.T) {
		gv := NewGraphVisualizer("Metadata Test", VisualizerConfig{
			ShowMetadata: true,
		})
		gv.AddNode(NodeInfo{
			ID:          "A",
			Label:       "Node A",
			Description: "This is a test node",
			Type:        NodeTypeRegular,
		})
		
		ascii := gv.ToASCII()
		if !strings.Contains(ascii, "This is a test node") {
			t.Error("Expected description in ASCII output with ShowMetadata=true")
		}
	})
}
