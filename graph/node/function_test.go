package node

import (
	"context"
	"errors"
	"testing"
)

// 测试用的状态类型
type TestState struct {
	Value   int
	Message string
	Done    bool
}

// TestNewMetadata 测试创建元数据
func TestNewMetadata(t *testing.T) {
	meta := NewMetadata("test-node")

	if meta.Name != "test-node" {
		t.Errorf("expected name 'test-node', got %s", meta.Name)
	}

	if meta.Tags == nil {
		t.Error("expected Tags to be initialized")
	}

	if meta.Extra == nil {
		t.Error("expected Extra to be initialized")
	}
}

// TestMetadata_WithMethods 测试元数据链式调用
func TestMetadata_WithMethods(t *testing.T) {
	meta := NewMetadata("test").
		WithDescription("Test node").
		WithTags("tag1", "tag2").
		WithVersion("1.0.0").
		WithExtra("key", "value")

	if meta.Description != "Test node" {
		t.Errorf("expected description 'Test node', got %s", meta.Description)
	}

	if len(meta.Tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(meta.Tags))
	}

	if meta.Version != "1.0.0" {
		t.Errorf("expected version '1.0.0', got %s", meta.Version)
	}

	if meta.Extra["key"] != "value" {
		t.Error("expected Extra['key'] to be 'value'")
	}
}

// TestMetadata_Clone 测试克隆元数据
func TestMetadata_Clone(t *testing.T) {
	original := NewMetadata("original").
		WithDescription("Original").
		WithTags("tag1").
		WithExtra("key", "value")

	clone := original.Clone()

	// 验证克隆内容相同
	if clone.Name != original.Name {
		t.Error("cloned name mismatch")
	}

	if clone.Description != original.Description {
		t.Error("cloned description mismatch")
	}

	// 修改克隆不应影响原始
	clone.Tags = append(clone.Tags, "tag2")
	if len(original.Tags) != 1 {
		t.Error("modifying clone affected original")
	}
}

// TestMetadata_Validate 测试验证元数据
func TestMetadata_Validate(t *testing.T) {
	// 有效的元数据
	valid := NewMetadata("valid")
	if err := valid.Validate(); err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	// 无效的元数据（空名称）
	invalid := &Metadata{Name: ""}
	if err := invalid.Validate(); !errors.Is(err, ErrNodeNameEmpty) {
		t.Errorf("expected ErrNodeNameEmpty, got %v", err)
	}
}

// TestNewFunctionNode 测试创建函数节点
func TestNewFunctionNode(t *testing.T) {
	fn := func(ctx context.Context, s TestState) (TestState, error) {
		s.Value++
		return s, nil
	}

	node := NewFunctionNode("increment", fn)

	if node.GetName() != "increment" {
		t.Errorf("expected name 'increment', got %s", node.GetName())
	}
}

// TestNewFunctionNode_WithOptions 测试带选项创建节点
func TestNewFunctionNode_WithOptions(t *testing.T) {
	fn := func(ctx context.Context, s TestState) (TestState, error) {
		return s, nil
	}

	node := NewFunctionNode("test", fn,
		WithDescription("Test node"),
		WithTags("test", "example"),
		WithVersion("1.0.0"),
	)

	if node.GetDescription() != "Test node" {
		t.Errorf("expected description 'Test node', got %s", node.GetDescription())
	}

	tags := node.GetTags()
	if len(tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(tags))
	}

	meta := node.GetMetadata()
	if meta.Version != "1.0.0" {
		t.Errorf("expected version '1.0.0', got %s", meta.Version)
	}
}

// TestFunctionNode_Invoke 测试执行节点
func TestFunctionNode_Invoke(t *testing.T) {
	fn := func(ctx context.Context, s TestState) (TestState, error) {
		s.Value += 10
		s.Message = "processed"
		return s, nil
	}

	node := NewFunctionNode("process", fn)

	result, err := node.Invoke(context.Background(), TestState{Value: 5})
	if err != nil {
		t.Fatalf("Invoke failed: %v", err)
	}

	if result.Value != 15 {
		t.Errorf("expected Value=15, got %d", result.Value)
	}

	if result.Message != "processed" {
		t.Errorf("expected Message='processed', got %s", result.Message)
	}
}

// TestFunctionNode_Invoke_Error 测试节点错误处理
func TestFunctionNode_Invoke_Error(t *testing.T) {
	expectedErr := errors.New("test error")
	fn := func(ctx context.Context, s TestState) (TestState, error) {
		return s, expectedErr
	}

	node := NewFunctionNode("error", fn)

	_, err := node.Invoke(context.Background(), TestState{})
	if !errors.Is(err, expectedErr) {
		t.Errorf("expected error to be %v, got %v", expectedErr, err)
	}
}

// TestFunctionNode_Invoke_ContextCancellation 测试上下文取消
func TestFunctionNode_Invoke_ContextCancellation(t *testing.T) {
	fn := func(ctx context.Context, s TestState) (TestState, error) {
		<-ctx.Done()
		return s, ctx.Err()
	}

	node := NewFunctionNode("slow", fn)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // 立即取消

	_, err := node.Invoke(ctx, TestState{})
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

// TestFunctionNode_Validate 测试验证节点
func TestFunctionNode_Validate(t *testing.T) {
	// 有效节点
	fn := func(ctx context.Context, s TestState) (TestState, error) {
		return s, nil
	}
	validNode := NewFunctionNode("valid", fn)
	if err := validNode.Validate(); err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	// 无效节点（nil 函数）
	invalidNode := &FunctionNode[TestState]{
		metadata: NewMetadata("invalid"),
		fn:       nil,
	}
	if err := invalidNode.Validate(); !errors.Is(err, ErrNodeFuncNil) {
		t.Errorf("expected ErrNodeFuncNil, got %v", err)
	}
}

// TestFunctionNode_Chain 测试链接节点
func TestFunctionNode_Chain(t *testing.T) {
	add := func(ctx context.Context, s TestState) (TestState, error) {
		s.Value += 10
		return s, nil
	}

	multiply := func(ctx context.Context, s TestState) (TestState, error) {
		s.Value *= 2
		return s, nil
	}

	node := NewFunctionNode("add", add)
	chained := node.Chain(multiply)

	result, err := chained.Invoke(context.Background(), TestState{Value: 5})
	if err != nil {
		t.Fatalf("Chain failed: %v", err)
	}

	// (5 + 10) * 2 = 30
	if result.Value != 30 {
		t.Errorf("expected Value=30, got %d", result.Value)
	}
}

// TestFunctionNode_Retry 测试重试逻辑
func TestFunctionNode_Retry(t *testing.T) {
	attempts := 0
	fn := func(ctx context.Context, s TestState) (TestState, error) {
		attempts++
		if attempts < 3 {
			return s, errors.New("temporary error")
		}
		s.Value = 100
		return s, nil
	}

	node := NewFunctionNode("flaky", fn)
	retryNode := node.Retry(3)

	result, err := retryNode.Invoke(context.Background(), TestState{})
	if err != nil {
		t.Fatalf("Retry failed: %v", err)
	}

	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}

	if result.Value != 100 {
		t.Errorf("expected Value=100, got %d", result.Value)
	}
}

// TestFunctionNode_Retry_AllFailed 测试重试全部失败
func TestFunctionNode_Retry_AllFailed(t *testing.T) {
	attempts := 0
	fn := func(ctx context.Context, s TestState) (TestState, error) {
		attempts++
		return s, errors.New("permanent error")
	}

	node := NewFunctionNode("failing", fn)
	retryNode := node.Retry(2)

	_, err := retryNode.Invoke(context.Background(), TestState{})
	if err == nil {
		t.Error("expected error, got nil")
	}

	if attempts != 3 { // 1 initial + 2 retries
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}

// TestFunctionNode_Fallback 测试降级逻辑
func TestFunctionNode_Fallback(t *testing.T) {
	primary := func(ctx context.Context, s TestState) (TestState, error) {
		return s, errors.New("primary failed")
	}

	fallback := func(ctx context.Context, s TestState) (TestState, error) {
		s.Message = "fallback"
		return s, nil
	}

	node := NewFunctionNode("primary", primary)
	fallbackNode := node.Fallback(fallback)

	result, err := fallbackNode.Invoke(context.Background(), TestState{})
	if err != nil {
		t.Fatalf("Fallback failed: %v", err)
	}

	if result.Message != "fallback" {
		t.Errorf("expected Message='fallback', got %s", result.Message)
	}
}

// TestFunctionNode_Fallback_PrimarySuccess 测试主节点成功时不用降级
func TestFunctionNode_Fallback_PrimarySuccess(t *testing.T) {
	primary := func(ctx context.Context, s TestState) (TestState, error) {
		s.Message = "primary"
		return s, nil
	}

	fallback := func(ctx context.Context, s TestState) (TestState, error) {
		s.Message = "fallback"
		return s, nil
	}

	node := NewFunctionNode("primary", primary)
	fallbackNode := node.Fallback(fallback)

	result, _ := fallbackNode.Invoke(context.Background(), TestState{})

	if result.Message != "primary" {
		t.Errorf("expected Message='primary', got %s", result.Message)
	}
}

// TestFunctionNode_Transform 测试转换节点
func TestFunctionNode_Transform(t *testing.T) {
	process := func(ctx context.Context, s TestState) (TestState, error) {
		s.Value = 10
		return s, nil
	}

	transform := func(ctx context.Context, s TestState) (TestState, error) {
		s.Value *= 2
		return s, nil
	}

	node := NewFunctionNode("process", process)
	transformed := node.Transform(transform)

	result, err := transformed.Invoke(context.Background(), TestState{})
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	if result.Value != 20 {
		t.Errorf("expected Value=20, got %d", result.Value)
	}
}

// TestFunctionNode_Conditional 测试条件执行
func TestFunctionNode_Conditional(t *testing.T) {
	fn := func(ctx context.Context, s TestState) (TestState, error) {
		s.Value = 100
		return s, nil
	}

	node := NewFunctionNode("expensive", fn)
	conditional := node.Conditional(func(ctx context.Context, s TestState) bool {
		return s.Done
	})

	// 条件为 false，不执行
	result1, _ := conditional.Invoke(context.Background(), TestState{Value: 10, Done: false})
	if result1.Value != 10 {
		t.Errorf("expected Value=10 (unchanged), got %d", result1.Value)
	}

	// 条件为 true，执行
	result2, _ := conditional.Invoke(context.Background(), TestState{Value: 10, Done: true})
	if result2.Value != 100 {
		t.Errorf("expected Value=100, got %d", result2.Value)
	}
}

// TestFunctionNode_WithFunc 测试替换函数
func TestFunctionNode_WithFunc(t *testing.T) {
	fn1 := func(ctx context.Context, s TestState) (TestState, error) {
		s.Value = 1
		return s, nil
	}

	fn2 := func(ctx context.Context, s TestState) (TestState, error) {
		s.Value = 2
		return s, nil
	}

	node1 := NewFunctionNode("test", fn1)
	node2 := node1.WithFunc(fn2)

	// 原节点不变
	result1, _ := node1.Invoke(context.Background(), TestState{})
	if result1.Value != 1 {
		t.Errorf("expected Value=1 from node1, got %d", result1.Value)
	}

	// 新节点使用新函数
	result2, _ := node2.Invoke(context.Background(), TestState{})
	if result2.Value != 2 {
		t.Errorf("expected Value=2 from node2, got %d", result2.Value)
	}
}
