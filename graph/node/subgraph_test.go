package node

import (
	"context"
	"errors"
	"testing"
)

// 测试用的父状态和子状态
type ParentState struct {
	Data   map[string]any
	Result int
}

type ChildState struct {
	Value int
}

// TestNewSubgraphNode 测试创建子图节点
func TestNewSubgraphNode(t *testing.T) {
	mockSubgraph := &MockSubgraph[ChildState]{
		InvokeFn: func(ctx context.Context, state ChildState, opts ...interface{}) (ChildState, error) {
			state.Value *= 2
			return state, nil
		},
	}

	node := NewSubgraphNode[ParentState, ChildState](
		"nested",
		mockSubgraph,
		WithStateMapper(
			func(p ParentState) (ChildState, error) {
				return ChildState{Value: p.Data["input"].(int)}, nil
			},
			func(p ParentState, c ChildState) (ParentState, error) {
				p.Result = c.Value
				return p, nil
			},
		),
	)

	if node.GetName() != "nested" {
		t.Errorf("expected name 'nested', got %s", node.GetName())
	}
}

// TestSubgraphNode_Invoke 测试执行子图节点
func TestSubgraphNode_Invoke(t *testing.T) {
	// 创建模拟子图（将值翻倍）
	mockSubgraph := &MockSubgraph[ChildState]{
		InvokeFn: func(ctx context.Context, state ChildState, opts ...interface{}) (ChildState, error) {
			state.Value *= 2
			return state, nil
		},
	}

	// 创建子图节点
	node := NewSubgraphNode[ParentState, ChildState](
		"double",
		mockSubgraph,
		WithStateMapper(
			// 父状态 -> 子状态
			func(p ParentState) (ChildState, error) {
				return ChildState{Value: p.Data["input"].(int)}, nil
			},
			// 合并子状态 -> 父状态
			func(p ParentState, c ChildState) (ParentState, error) {
				p.Result = c.Value
				return p, nil
			},
		),
	)

	// 执行
	parentState := ParentState{
		Data: map[string]any{"input": 10},
	}

	result, err := node.Invoke(context.Background(), parentState)
	if err != nil {
		t.Fatalf("Invoke failed: %v", err)
	}

	if result.Result != 20 {
		t.Errorf("expected Result=20, got %d", result.Result)
	}
}

// TestSubgraphNode_Invoke_WithOptions 测试带选项的子图节点
func TestSubgraphNode_Invoke_WithOptions(t *testing.T) {
	mockSubgraph := &MockSubgraph[ChildState]{
		InvokeFn: func(ctx context.Context, state ChildState, opts ...interface{}) (ChildState, error) {
			state.Value += 5
			return state, nil
		},
	}

	node := NewSubgraphNode[ParentState, ChildState](
		"add",
		mockSubgraph,
		WithDescription("Add 5 to value"),
		WithTags("math", "add"),
		WithStateMapper(
			func(p ParentState) (ChildState, error) {
				return ChildState{Value: p.Data["x"].(int)}, nil
			},
			func(p ParentState, c ChildState) (ParentState, error) {
				p.Result = c.Value
				return p, nil
			},
		),
	)

	if node.GetDescription() != "Add 5 to value" {
		t.Errorf("unexpected description: %s", node.GetDescription())
	}

	tags := node.GetTags()
	if len(tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(tags))
	}
}

// TestSubgraphNode_MapToChild_Error 测试映射到子状态错误
func TestSubgraphNode_MapToChild_Error(t *testing.T) {
	mockSubgraph := &MockSubgraph[ChildState]{}

	expectedErr := errors.New("map error")
	node := NewSubgraphNode[ParentState, ChildState](
		"error",
		mockSubgraph,
		WithStateMapper(
			func(p ParentState) (ChildState, error) {
				return ChildState{}, expectedErr
			},
			func(p ParentState, c ChildState) (ParentState, error) {
				return p, nil
			},
		),
	)

	_, err := node.Invoke(context.Background(), ParentState{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, expectedErr) {
		t.Errorf("expected error to wrap expectedErr")
	}
}

// TestSubgraphNode_Subgraph_Error 测试子图执行错误
func TestSubgraphNode_Subgraph_Error(t *testing.T) {
	expectedErr := errors.New("subgraph error")
	mockSubgraph := &MockSubgraph[ChildState]{
		InvokeFn: func(ctx context.Context, state ChildState, opts ...interface{}) (ChildState, error) {
			return state, expectedErr
		},
	}

	node := NewSubgraphNode[ParentState, ChildState](
		"failing",
		mockSubgraph,
		WithStateMapper(
			func(p ParentState) (ChildState, error) {
				return ChildState{}, nil
			},
			func(p ParentState, c ChildState) (ParentState, error) {
				return p, nil
			},
		),
	)

	_, err := node.Invoke(context.Background(), ParentState{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, expectedErr) {
		t.Errorf("expected error to wrap expectedErr")
	}
}

// TestSubgraphNode_MapFromChild_Error 测试映射回父状态错误
func TestSubgraphNode_MapFromChild_Error(t *testing.T) {
	mockSubgraph := &MockSubgraph[ChildState]{
		InvokeFn: func(ctx context.Context, state ChildState, opts ...interface{}) (ChildState, error) {
			return state, nil
		},
	}

	expectedErr := errors.New("map back error")
	node := NewSubgraphNode[ParentState, ChildState](
		"map_error",
		mockSubgraph,
		WithStateMapper(
			func(p ParentState) (ChildState, error) {
				return ChildState{}, nil
			},
			func(p ParentState, c ChildState) (ParentState, error) {
				return p, expectedErr
			},
		),
	)

	_, err := node.Invoke(context.Background(), ParentState{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, expectedErr) {
		t.Errorf("expected error to wrap expectedErr")
	}
}

// TestSubgraphNode_Validate 测试验证子图节点
func TestSubgraphNode_Validate(t *testing.T) {
	mockSubgraph := &MockSubgraph[ChildState]{}

	// 有效节点
	validNode := NewSubgraphNode[ParentState, ChildState](
		"valid",
		mockSubgraph,
		WithStateMapper(
			func(p ParentState) (ChildState, error) {
				return ChildState{}, nil
			},
			func(p ParentState, c ChildState) (ParentState, error) {
				return p, nil
			},
		),
	)

	if err := validNode.Validate(); err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	// 无效节点（缺少 mapToChild）
	invalidNode1 := &SubgraphNode[ParentState, ChildState]{
		metadata: NewMetadata("invalid1"),
		subgraph: mockSubgraph,
		mapFromChild: func(p ParentState, c ChildState) (ParentState, error) {
			return p, nil
		},
	}

	if err := invalidNode1.Validate(); err == nil {
		t.Error("expected error for missing mapToChild")
	}

	// 无效节点（缺少 mapFromChild）
	invalidNode2 := &SubgraphNode[ParentState, ChildState]{
		metadata: NewMetadata("invalid2"),
		subgraph: mockSubgraph,
		mapToChild: func(p ParentState) (ChildState, error) {
			return ChildState{}, nil
		},
	}

	if err := invalidNode2.Validate(); err == nil {
		t.Error("expected error for missing mapFromChild")
	}

	// 无效节点（nil subgraph）
	invalidNode3 := &SubgraphNode[ParentState, ChildState]{
		metadata: NewMetadata("invalid3"),
		subgraph: nil,
		mapToChild: func(p ParentState) (ChildState, error) {
			return ChildState{}, nil
		},
		mapFromChild: func(p ParentState, c ChildState) (ParentState, error) {
			return p, nil
		},
	}

	if err := invalidNode3.Validate(); err == nil {
		t.Error("expected error for nil subgraph")
	}
}

// TestSubgraphNode_ContextCancellation 测试上下文取消
func TestSubgraphNode_ContextCancellation(t *testing.T) {
	mockSubgraph := &MockSubgraph[ChildState]{
		InvokeFn: func(ctx context.Context, state ChildState, opts ...interface{}) (ChildState, error) {
			<-ctx.Done()
			return state, ctx.Err()
		},
	}

	node := NewSubgraphNode[ParentState, ChildState](
		"slow",
		mockSubgraph,
		WithStateMapper(
			func(p ParentState) (ChildState, error) {
				return ChildState{}, nil
			},
			func(p ParentState, c ChildState) (ParentState, error) {
				return p, nil
			},
		),
	)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // 立即取消

	_, err := node.Invoke(ctx, ParentState{})
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

// TestSubgraphNode_ComplexMapping 测试复杂的状态映射
func TestSubgraphNode_ComplexMapping(t *testing.T) {
	// 子图：处理数组求和
	mockSubgraph := &MockSubgraph[ChildState]{
		InvokeFn: func(ctx context.Context, state ChildState, opts ...interface{}) (ChildState, error) {
			// 假设 Value 是数组长度，这里只是演示
			state.Value = state.Value * 10
			return state, nil
		},
	}

	node := NewSubgraphNode[ParentState, ChildState](
		"process_array",
		mockSubgraph,
		WithStateMapper(
			// 从父状态提取数组长度
			func(p ParentState) (ChildState, error) {
				arr, ok := p.Data["array"].([]int)
				if !ok {
					return ChildState{}, errors.New("array not found")
				}
				return ChildState{Value: len(arr)}, nil
			},
			// 将结果写回父状态
			func(p ParentState, c ChildState) (ParentState, error) {
				p.Result = c.Value
				p.Data["processed"] = true
				return p, nil
			},
		),
	)

	parentState := ParentState{
		Data: map[string]any{
			"array": []int{1, 2, 3, 4, 5},
		},
	}

	result, err := node.Invoke(context.Background(), parentState)
	if err != nil {
		t.Fatalf("Invoke failed: %v", err)
	}

	// len([1,2,3,4,5]) * 10 = 50
	if result.Result != 50 {
		t.Errorf("expected Result=50, got %d", result.Result)
	}

	if !result.Data["processed"].(bool) {
		t.Error("expected processed flag to be true")
	}
}
