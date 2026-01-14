package compile

import (
	"errors"
	"testing"
)

// MockGraph 是用于测试的模拟图。
type MockGraph[S any] struct {
	name         string
	nodes        map[string]NodeInfo
	edges        []EdgeInfo
	conditionals []ConditionalInfo[S]
	entryPoint   string
}

func (m *MockGraph[S]) GetName() string {
	return m.name
}

func (m *MockGraph[S]) GetNodes() map[string]NodeInfo {
	return m.nodes
}

func (m *MockGraph[S]) GetEdges() []EdgeInfo {
	return m.edges
}

func (m *MockGraph[S]) GetConditionals() []ConditionalInfo[S] {
	return m.conditionals
}

func (m *MockGraph[S]) GetEntryPoint() string {
	return m.entryPoint
}

// TestState 测试用状态
type TestState struct {
	Value int
}

// TestNewValidator 测试创建验证器
func TestNewValidator(t *testing.T) {
	validator := NewValidator[TestState]()

	if validator == nil {
		t.Fatal("NewValidator returned nil")
	}

	if validator.checkCycles {
		t.Error("expected checkCycles to be false by default")
	}
}

// TestValidator_WithCycleCheck 测试设置循环检测
func TestValidator_WithCycleCheck(t *testing.T) {
	validator := NewValidator[TestState]()

	result := validator.WithCycleCheck(true)

	if result != validator {
		t.Error("WithCycleCheck should return self")
	}

	if !validator.checkCycles {
		t.Error("expected checkCycles to be true")
	}
}

// TestValidator_Validate_ValidGraph 测试验证有效图
func TestValidator_Validate_ValidGraph(t *testing.T) {
	graph := &MockGraph[TestState]{
		name: "test",
		nodes: map[string]NodeInfo{
			"node1": {Name: "node1"},
			"node2": {Name: "node2"},
		},
		edges: []EdgeInfo{
			{From: "node1", To: "node2"},
			{From: "node2", To: "__end__"},
		},
		entryPoint: "node1",
	}

	validator := NewValidator[TestState]()
	if err := validator.Validate(graph); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

// TestValidator_Validate_NoEntryPoint 测试无入口点
func TestValidator_Validate_NoEntryPoint(t *testing.T) {
	graph := &MockGraph[TestState]{
		name: "test",
		nodes: map[string]NodeInfo{
			"node1": {Name: "node1"},
		},
		entryPoint: "",
	}

	validator := NewValidator[TestState]()
	err := validator.Validate(graph)

	if err == nil {
		t.Fatal("expected error for no entry point")
	}

	// ValidationError 会包装底层错误
	var valErr *ValidationError
	if !errors.As(err, &valErr) {
		t.Errorf("expected ValidationError, got %T", err)
	}
}

// TestValidator_Validate_InvalidEntryPoint 测试无效入口点
func TestValidator_Validate_InvalidEntryPoint(t *testing.T) {
	graph := &MockGraph[TestState]{
		name: "test",
		nodes: map[string]NodeInfo{
			"node1": {Name: "node1"},
		},
		entryPoint: "nonexistent",
	}

	validator := NewValidator[TestState]()
	err := validator.Validate(graph)

	if err == nil {
		t.Fatal("expected error for invalid entry point")
	}
}

// TestValidator_Validate_UnreachableNode 测试不可达节点
func TestValidator_Validate_UnreachableNode(t *testing.T) {
	graph := &MockGraph[TestState]{
		name: "test",
		nodes: map[string]NodeInfo{
			"node1": {Name: "node1"},
			"node2": {Name: "node2"},
			"node3": {Name: "node3"}, // 不可达
		},
		edges: []EdgeInfo{
			{From: "node1", To: "node2"},
			{From: "node2", To: "__end__"},
		},
		entryPoint: "node1",
	}

	validator := NewValidator[TestState]()
	err := validator.Validate(graph)

	if err == nil {
		t.Fatal("expected error for unreachable node")
	}

	// ValidationError 会包装底层错误
	var valErr *ValidationError
	if !errors.As(err, &valErr) {
		t.Errorf("expected ValidationError, got %T", err)
	}
}

// TestValidator_Validate_DanglingEdge 测试悬空边
func TestValidator_Validate_DanglingEdge(t *testing.T) {
	graph := &MockGraph[TestState]{
		name: "test",
		nodes: map[string]NodeInfo{
			"node1": {Name: "node1"},
		},
		edges: []EdgeInfo{
			{From: "node1", To: "nonexistent"},
		},
		entryPoint: "node1",
	}

	validator := NewValidator[TestState]()
	err := validator.Validate(graph)

	if err == nil {
		t.Fatal("expected error for dangling edge")
	}
}

// TestValidator_Validate_WithConditionals 测试带条件边的图
func TestValidator_Validate_WithConditionals(t *testing.T) {
	graph := &MockGraph[TestState]{
		name: "test",
		nodes: map[string]NodeInfo{
			"router": {Name: "router"},
			"node1":  {Name: "node1"},
			"node2":  {Name: "node2"},
		},
		conditionals: []ConditionalInfo[TestState]{
			{
				Source: "router",
				PathMap: map[string]string{
					"path1": "node1",
					"path2": "node2",
				},
			},
		},
		edges: []EdgeInfo{
			{From: "node1", To: "__end__"},
			{From: "node2", To: "__end__"},
		},
		entryPoint: "router",
	}

	validator := NewValidator[TestState]()
	if err := validator.Validate(graph); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

// TestValidator_Validate_Cycle 测试循环检测
func TestValidator_Validate_Cycle(t *testing.T) {
	graph := &MockGraph[TestState]{
		name: "test",
		nodes: map[string]NodeInfo{
			"node1": {Name: "node1"},
			"node2": {Name: "node2"},
		},
		edges: []EdgeInfo{
			{From: "node1", To: "node2"},
			{From: "node2", To: "node1"}, // 循环
		},
		entryPoint: "node1",
	}

	// 不检测循环（默认）
	validator1 := NewValidator[TestState]()
	if err := validator1.Validate(graph); err != nil {
		t.Errorf("expected no error without cycle check, got %v", err)
	}

	// 检测循环
	validator2 := NewValidator[TestState]().WithCycleCheck(true)
	err := validator2.Validate(graph)
	if err == nil {
		t.Fatal("expected error for cycle")
	}

	// ValidationError 会包装底层错误
	var valErr *ValidationError
	if !errors.As(err, &valErr) {
		t.Errorf("expected ValidationError, got %T", err)
	}
}

// TestValidateQuick 测试快速验证
func TestValidateQuick(t *testing.T) {
	graph := &MockGraph[TestState]{
		name: "test",
		nodes: map[string]NodeInfo{
			"node1": {Name: "node1"},
			"node2": {Name: "node2"},
		},
		edges: []EdgeInfo{
			{From: "node1", To: "node2"},
		},
		entryPoint: "node1",
	}

	if err := ValidateQuick(graph); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

// TestNewCompiler 测试创建编译器
func TestNewCompiler(t *testing.T) {
	compiler := NewCompiler[TestState]()

	if compiler == nil {
		t.Fatal("NewCompiler returned nil")
	}

	if compiler.validator == nil {
		t.Error("expected validator to be initialized")
	}

	if compiler.optimize {
		t.Error("expected optimize to be false by default")
	}
}

// TestCompiler_WithOptimization 测试设置优化
func TestCompiler_WithOptimization(t *testing.T) {
	compiler := NewCompiler[TestState]()

	result := compiler.WithOptimization(true)

	if result != compiler {
		t.Error("WithOptimization should return self")
	}

	if !compiler.optimize {
		t.Error("expected optimize to be true")
	}
}

// TestCompiler_Compile 测试编译图
func TestCompiler_Compile(t *testing.T) {
	graph := &MockGraph[TestState]{
		name: "test",
		nodes: map[string]NodeInfo{
			"node1": {Name: "node1"},
			"node2": {Name: "node2"},
		},
		edges: []EdgeInfo{
			{From: "node1", To: "node2"},
			{From: "node2", To: "__end__"},
		},
		entryPoint: "node1",
	}

	compiler := NewCompiler[TestState]()
	compiled, err := compiler.Compile(graph)

	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}

	if compiled == nil {
		t.Fatal("Compile returned nil")
	}

	if compiled.GetName() != "test" {
		t.Errorf("expected name 'test', got %s", compiled.GetName())
	}

	if compiled.GetEntryPoint() != "node1" {
		t.Errorf("expected entry point 'node1', got %s", compiled.GetEntryPoint())
	}
}

// TestCompiler_Compile_InvalidGraph 测试编译无效图
func TestCompiler_Compile_InvalidGraph(t *testing.T) {
	graph := &MockGraph[TestState]{
		name:       "test",
		nodes:      map[string]NodeInfo{},
		entryPoint: "",
	}

	compiler := NewCompiler[TestState]()
	_, err := compiler.Compile(graph)

	if err == nil {
		t.Fatal("expected error for invalid graph")
	}
}

// TestCompiler_Compile_WithOptimization 测试带优化的编译
func TestCompiler_Compile_WithOptimization(t *testing.T) {
	graph := &MockGraph[TestState]{
		name: "test",
		nodes: map[string]NodeInfo{
			"node1": {Name: "node1"},
		},
		edges: []EdgeInfo{
			{From: "node1", To: "__end__"},
		},
		entryPoint: "node1",
	}

	compiler := NewCompiler[TestState]().WithOptimization(true)
	compiled, err := compiler.Compile(graph)

	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}

	if compiled == nil {
		t.Fatal("expected compiled graph")
	}
}

// TestCompiledGraph_Adjacency 测试邻接表构建
func TestCompiledGraph_Adjacency(t *testing.T) {
	graph := &MockGraph[TestState]{
		name: "test",
		nodes: map[string]NodeInfo{
			"node1": {Name: "node1"},
			"node2": {Name: "node2"},
			"node3": {Name: "node3"},
		},
		edges: []EdgeInfo{
			{From: "node1", To: "node2"},
			{From: "node2", To: "node3"},
		},
		entryPoint: "node1",
	}

	compiler := NewCompiler[TestState]()
	compiled, _ := compiler.Compile(graph)

	adjacency := compiled.GetAdjacency()

	// 检查邻接表
	if len(adjacency["node1"]) != 1 {
		t.Errorf("expected node1 to have 1 neighbor, got %d", len(adjacency["node1"]))
	}

	if adjacency["node1"][0] != "node2" {
		t.Errorf("expected node1 -> node2")
	}
}

// TestCompiledGraph_String 测试字符串表示
func TestCompiledGraph_String(t *testing.T) {
	compiled := &CompiledGraph[TestState]{
		name: "test",
		nodes: map[string]NodeInfo{
			"node1": {Name: "node1"},
		},
		edges: []EdgeInfo{
			{From: "node1", To: "__end__"},
		},
	}

	str := compiled.String()
	expected := "CompiledGraph{name=test, nodes=1, edges=1}"

	if str != expected {
		t.Errorf("expected '%s', got '%s'", expected, str)
	}
}
