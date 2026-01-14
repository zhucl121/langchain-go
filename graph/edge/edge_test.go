package edge

import (
	"errors"
	"testing"
)

// 测试用的状态类型
type TestState struct {
	Counter int
	Message string
	Done    bool
}

// TestNewNormalEdge 测试创建普通边
func TestNewNormalEdge(t *testing.T) {
	edge := NewNormalEdge("node1", "node2")

	if edge.GetSource() != "node1" {
		t.Errorf("expected source 'node1', got %s", edge.GetSource())
	}

	if edge.GetTarget() != "node2" {
		t.Errorf("expected target 'node2', got %s", edge.GetTarget())
	}

	if edge.GetType() != TypeNormal {
		t.Errorf("expected type %s, got %s", TypeNormal, edge.GetType())
	}
}

// TestNormalEdge_Validate 测试验证普通边
func TestNormalEdge_Validate(t *testing.T) {
	// 有效边
	validEdge := NewNormalEdge("source", "target")
	if err := validEdge.Validate(); err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	// 无效边（空源节点）
	invalidEdge1 := NewNormalEdge("", "target")
	if !errors.Is(invalidEdge1.Validate(), ErrEmptySourceNode) {
		t.Error("expected ErrEmptySourceNode")
	}

	// 无效边（空目标节点）
	invalidEdge2 := NewNormalEdge("source", "")
	if !errors.Is(invalidEdge2.Validate(), ErrEmptyTargetNode) {
		t.Error("expected ErrEmptyTargetNode")
	}
}

// TestNormalEdge_Clone 测试克隆普通边
func TestNormalEdge_Clone(t *testing.T) {
	original := NewNormalEdge("node1", "node2")
	cloned := original.Clone().(*NormalEdge)

	if cloned.GetSource() != original.GetSource() {
		t.Error("cloned source mismatch")
	}

	if cloned.GetTarget() != original.GetTarget() {
		t.Error("cloned target mismatch")
	}
}

// TestNormalEdge_String 测试字符串表示
func TestNormalEdge_String(t *testing.T) {
	edge := NewNormalEdge("A", "B")
	str := edge.String()

	expected := "A -> B"
	if str != expected {
		t.Errorf("expected '%s', got '%s'", expected, str)
	}
}

// TestNewMetadata 测试创建元数据
func TestNewMetadata(t *testing.T) {
	meta := NewMetadata()

	if meta.Tags == nil {
		t.Error("expected Tags to be initialized")
	}

	if meta.Extra == nil {
		t.Error("expected Extra to be initialized")
	}
}

// TestMetadata_WithMethods 测试元数据链式调用
func TestMetadata_WithMethods(t *testing.T) {
	meta := NewMetadata().
		WithName("edge1").
		WithDescription("Test edge").
		WithTags("tag1", "tag2").
		WithWeight(1.5).
		WithExtra("key", "value")

	if meta.Name != "edge1" {
		t.Errorf("expected name 'edge1', got %s", meta.Name)
	}

	if meta.Description != "Test edge" {
		t.Errorf("expected description 'Test edge', got %s", meta.Description)
	}

	if len(meta.Tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(meta.Tags))
	}

	if meta.Weight != 1.5 {
		t.Errorf("expected weight 1.5, got %f", meta.Weight)
	}

	if meta.Extra["key"] != "value" {
		t.Error("expected Extra['key'] to be 'value'")
	}
}

// TestMetadata_Clone 测试克隆元数据
func TestMetadata_Clone(t *testing.T) {
	original := NewMetadata().
		WithName("original").
		WithTags("tag1").
		WithExtra("key", "value")

	clone := original.Clone()

	// 验证克隆内容相同
	if clone.Name != original.Name {
		t.Error("cloned name mismatch")
	}

	// 修改克隆不应影响原始
	clone.Tags = append(clone.Tags, "tag2")
	if len(original.Tags) != 1 {
		t.Error("modifying clone affected original")
	}

	clone.Extra["key2"] = "value2"
	if _, exists := original.Extra["key2"]; exists {
		t.Error("modifying clone affected original")
	}
}

// TestNewConditionalEdge 测试创建条件边
func TestNewConditionalEdge(t *testing.T) {
	pathFunc := func(s TestState) string {
		if s.Counter > 0 {
			return "positive"
		}
		return "zero"
	}

	pathMap := map[string]string{
		"positive": "node1",
		"zero":     "node2",
	}

	edge := NewConditionalEdge("router", pathFunc, pathMap)

	if edge.GetSource() != "router" {
		t.Errorf("expected source 'router', got %s", edge.GetSource())
	}

	if edge.GetType() != TypeConditional {
		t.Errorf("expected type %s, got %s", TypeConditional, edge.GetType())
	}
}

// TestConditionalEdge_Route 测试条件边路由
func TestConditionalEdge_Route(t *testing.T) {
	pathFunc := func(s TestState) string {
		if s.Counter > 0 {
			return "positive"
		} else if s.Counter < 0 {
			return "negative"
		}
		return "zero"
	}

	pathMap := map[string]string{
		"positive": "positive_node",
		"negative": "negative_node",
		"zero":     "zero_node",
	}

	edge := NewConditionalEdge("router", pathFunc, pathMap)

	// 测试正数
	target1, err := edge.Route(TestState{Counter: 10})
	if err != nil {
		t.Fatalf("Route failed: %v", err)
	}
	if target1 != "positive_node" {
		t.Errorf("expected 'positive_node', got %s", target1)
	}

	// 测试负数
	target2, err := edge.Route(TestState{Counter: -5})
	if err != nil {
		t.Fatalf("Route failed: %v", err)
	}
	if target2 != "negative_node" {
		t.Errorf("expected 'negative_node', got %s", target2)
	}

	// 测试零
	target3, err := edge.Route(TestState{Counter: 0})
	if err != nil {
		t.Fatalf("Route failed: %v", err)
	}
	if target3 != "zero_node" {
		t.Errorf("expected 'zero_node', got %s", target3)
	}
}

// TestConditionalEdge_Route_PathNotFound 测试路径未找到错误
func TestConditionalEdge_Route_PathNotFound(t *testing.T) {
	pathFunc := func(s TestState) string {
		return "unknown"
	}

	pathMap := map[string]string{
		"known": "node1",
	}

	edge := NewConditionalEdge("router", pathFunc, pathMap)

	_, err := edge.Route(TestState{})
	if !errors.Is(err, ErrPathNotFound) {
		t.Errorf("expected ErrPathNotFound, got %v", err)
	}
}

// TestConditionalEdge_Validate 测试验证条件边
func TestConditionalEdge_Validate(t *testing.T) {
	// 有效边
	validEdge := NewConditionalEdge("source",
		func(s TestState) string { return "path1" },
		map[string]string{"path1": "target"},
	)
	if err := validEdge.Validate(); err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	// 无效边（空源节点）
	invalidEdge1 := NewConditionalEdge("",
		func(s TestState) string { return "path1" },
		map[string]string{"path1": "target"},
	)
	if !errors.Is(invalidEdge1.Validate(), ErrEmptySourceNode) {
		t.Error("expected ErrEmptySourceNode")
	}

	// 无效边（nil 路径函数）
	invalidEdge2 := &ConditionalEdge[TestState]{
		source:  "source",
		pathMap: map[string]string{"path1": "target"},
	}
	if err := invalidEdge2.Validate(); err == nil {
		t.Error("expected error for nil path function")
	}

	// 无效边（空路径映射）
	invalidEdge3 := NewConditionalEdge("source",
		func(s TestState) string { return "path1" },
		map[string]string{},
	)
	if !errors.Is(invalidEdge3.Validate(), ErrInvalidPathMapping) {
		t.Error("expected ErrInvalidPathMapping")
	}
}

// TestConditionalEdge_AddPath 测试添加路径
func TestConditionalEdge_AddPath(t *testing.T) {
	edge := NewConditionalEdge("router",
		func(s TestState) string { return "new_path" },
		map[string]string{"old_path": "old_node"},
	)

	edge.AddPath("new_path", "new_node")

	target, err := edge.Route(TestState{})
	if err != nil {
		t.Fatalf("Route failed: %v", err)
	}

	if target != "new_node" {
		t.Errorf("expected 'new_node', got %s", target)
	}
}

// TestConditionalEdge_RemovePath 测试移除路径
func TestConditionalEdge_RemovePath(t *testing.T) {
	edge := NewConditionalEdge("router",
		func(s TestState) string { return "path1" },
		map[string]string{
			"path1": "node1",
			"path2": "node2",
		},
	)

	edge.RemovePath("path1")

	pathMap := edge.GetPathMap()
	if _, exists := pathMap["path1"]; exists {
		t.Error("path1 should be removed")
	}

	if _, exists := pathMap["path2"]; !exists {
		t.Error("path2 should still exist")
	}
}

// TestConditionalEdge_Clone 测试克隆条件边
func TestConditionalEdge_Clone(t *testing.T) {
	original := NewConditionalEdge("router",
		func(s TestState) string { return "path1" },
		map[string]string{"path1": "node1"},
	)

	cloned := original.Clone().(*ConditionalEdge[TestState])

	if cloned.GetSource() != original.GetSource() {
		t.Error("cloned source mismatch")
	}

	// 修改克隆不应影响原始
	cloned.AddPath("path2", "node2")
	originalMap := original.GetPathMap()
	if len(originalMap) != 1 {
		t.Error("modifying clone affected original")
	}
}

// TestConditionalEdge_String 测试字符串表示
func TestConditionalEdge_String(t *testing.T) {
	edge := NewConditionalEdge("router",
		func(s TestState) string { return "p" },
		map[string]string{
			"p1": "n1",
			"p2": "n2",
		},
	)

	str := edge.String()
	expected := "router -?-> {2 paths}"
	if str != expected {
		t.Errorf("expected '%s', got '%s'", expected, str)
	}
}
