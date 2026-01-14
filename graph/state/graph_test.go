package state

import (
	"context"
	"errors"
	"testing"
)

// 测试用的状态类型
type TestState struct {
	Counter int
	Message string
	Done    bool
}

// TestNewStateGraph 测试创建状态图
func TestNewStateGraph(t *testing.T) {
	graph := NewStateGraph[TestState]("test-graph")

	if graph == nil {
		t.Fatal("NewStateGraph returned nil")
	}

	if graph.GetName() != "test-graph" {
		t.Errorf("expected name 'test-graph', got %s", graph.GetName())
	}

	if len(graph.nodes) != 0 {
		t.Errorf("expected 0 nodes, got %d", len(graph.nodes))
	}
}

// TestNewStateGraph_EmptyName 测试空名称会 panic
func TestNewStateGraph_EmptyName(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for empty name")
		}
	}()

	NewStateGraph[TestState]("")
}

// TestAddNode 测试添加节点
func TestAddNode(t *testing.T) {
	graph := NewStateGraph[TestState]("test")

	incrementNode := func(ctx context.Context, s TestState) (TestState, error) {
		s.Counter++
		return s, nil
	}

	graph.AddNode("increment", incrementNode)

	if len(graph.nodes) != 1 {
		t.Errorf("expected 1 node, got %d", len(graph.nodes))
	}

	node, exists := graph.nodes["increment"]
	if !exists {
		t.Error("node 'increment' not found")
	}

	if node.Name != "increment" {
		t.Errorf("expected node name 'increment', got %s", node.Name)
	}
}

// TestAddNode_ChainCall 测试链式调用
func TestAddNode_ChainCall(t *testing.T) {
	graph := NewStateGraph[TestState]("test")

	node1 := func(ctx context.Context, s TestState) (TestState, error) {
		return s, nil
	}
	node2 := func(ctx context.Context, s TestState) (TestState, error) {
		return s, nil
	}

	result := graph.AddNode("node1", node1).AddNode("node2", node2)

	if result != graph {
		t.Error("AddNode should return self for chaining")
	}

	if len(graph.nodes) != 2 {
		t.Errorf("expected 2 nodes, got %d", len(graph.nodes))
	}
}

// TestAddNode_EmptyName 测试空节点名称会 panic
func TestAddNode_EmptyName(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for empty node name")
		}
	}()

	graph := NewStateGraph[TestState]("test")
	graph.AddNode("", func(ctx context.Context, s TestState) (TestState, error) {
		return s, nil
	})
}

// TestAddNode_ReservedName 测试保留名称会 panic
func TestAddNode_ReservedName(t *testing.T) {
	tests := []struct {
		name         string
		reservedName string
	}{
		{"START", START},
		{"END", END},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("expected panic for reserved name %s", tt.reservedName)
				}
			}()

			graph := NewStateGraph[TestState]("test")
			graph.AddNode(tt.reservedName, func(ctx context.Context, s TestState) (TestState, error) {
				return s, nil
			})
		})
	}
}

// TestAddNode_NilFunction 测试 nil 函数会 panic
func TestAddNode_NilFunction(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for nil function")
		}
	}()

	graph := NewStateGraph[TestState]("test")
	graph.AddNode("test", nil)
}

// TestAddNode_Overwrite 测试覆盖节点
func TestAddNode_Overwrite(t *testing.T) {
	graph := NewStateGraph[TestState]("test")

	node1 := func(ctx context.Context, s TestState) (TestState, error) {
		s.Counter = 1
		return s, nil
	}
	node2 := func(ctx context.Context, s TestState) (TestState, error) {
		s.Counter = 2
		return s, nil
	}

	graph.AddNode("test", node1)
	graph.AddNode("test", node2) // 覆盖

	if len(graph.nodes) != 1 {
		t.Errorf("expected 1 node, got %d", len(graph.nodes))
	}

	// 验证节点被覆盖（通过执行测试）
	node := graph.nodes["test"]
	result, _ := node.Func(context.Background(), TestState{})
	if result.Counter != 2 {
		t.Error("node should be overwritten")
	}
}

// TestSetEntryPoint 测试设置入口点
func TestSetEntryPoint(t *testing.T) {
	graph := NewStateGraph[TestState]("test")

	graph.AddNode("start", func(ctx context.Context, s TestState) (TestState, error) {
		return s, nil
	})

	graph.SetEntryPoint("start")

	if graph.entryPoint != "start" {
		t.Errorf("expected entry point 'start', got %s", graph.entryPoint)
	}
}

// TestSetEntryPoint_NodeNotFound 测试设置不存在的入口点
func TestSetEntryPoint_NodeNotFound(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for non-existent entry point")
		}
	}()

	graph := NewStateGraph[TestState]("test")
	graph.SetEntryPoint("nonexistent")
}

// TestAddEdge 测试添加边
func TestAddEdge(t *testing.T) {
	graph := NewStateGraph[TestState]("test")

	graph.AddNode("node1", func(ctx context.Context, s TestState) (TestState, error) {
		return s, nil
	})
	graph.AddNode("node2", func(ctx context.Context, s TestState) (TestState, error) {
		return s, nil
	})

	graph.AddEdge("node1", "node2")

	if len(graph.edges) != 1 {
		t.Errorf("expected 1 edge, got %d", len(graph.edges))
	}

	edge := graph.edges[0]
	if edge.From != "node1" || edge.To != "node2" {
		t.Errorf("expected edge from node1 to node2, got %s -> %s", edge.From, edge.To)
	}
}

// TestAddEdge_ToEND 测试添加到 END 的边
func TestAddEdge_ToEND(t *testing.T) {
	graph := NewStateGraph[TestState]("test")

	graph.AddNode("final", func(ctx context.Context, s TestState) (TestState, error) {
		return s, nil
	})

	graph.AddEdge("final", END)

	if len(graph.edges) != 1 {
		t.Errorf("expected 1 edge, got %d", len(graph.edges))
	}

	if !graph.finishPoints["final"] {
		t.Error("expected 'final' to be marked as finish point")
	}
}

// TestAddEdge_NodeNotFound 测试添加不存在节点的边
func TestAddEdge_NodeNotFound(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for non-existent node")
		}
	}()

	graph := NewStateGraph[TestState]("test")
	graph.AddEdge("nonexistent", END)
}

// TestAddConditionalEdges 测试添加条件边
func TestAddConditionalEdges(t *testing.T) {
	graph := NewStateGraph[TestState]("test")

	graph.AddNode("router", func(ctx context.Context, s TestState) (TestState, error) {
		return s, nil
	})
	graph.AddNode("branch1", func(ctx context.Context, s TestState) (TestState, error) {
		return s, nil
	})
	graph.AddNode("branch2", func(ctx context.Context, s TestState) (TestState, error) {
		return s, nil
	})

	pathFunc := func(s TestState) string {
		if s.Counter > 0 {
			return "positive"
		}
		return "zero"
	}

	pathMap := map[string]string{
		"positive": "branch1",
		"zero":     "branch2",
	}

	graph.AddConditionalEdges("router", pathFunc, pathMap)

	if len(graph.conditionals) != 1 {
		t.Errorf("expected 1 conditional edge, got %d", len(graph.conditionals))
	}

	conditional := graph.conditionals[0]
	if conditional.Source != "router" {
		t.Errorf("expected source 'router', got %s", conditional.Source)
	}
}

// TestAddConditionalEdges_ToEND 测试条件边到 END
func TestAddConditionalEdges_ToEND(t *testing.T) {
	graph := NewStateGraph[TestState]("test")

	graph.AddNode("router", func(ctx context.Context, s TestState) (TestState, error) {
		return s, nil
	})

	pathFunc := func(s TestState) string {
		if s.Done {
			return "end"
		}
		return "continue"
	}

	pathMap := map[string]string{
		"end":      END,
		"continue": "router",
	}

	graph.AddConditionalEdges("router", pathFunc, pathMap)

	if !graph.finishPoints["router"] {
		t.Error("expected 'router' to be marked as potential finish point")
	}
}

// TestCompile 测试编译图
func TestCompile(t *testing.T) {
	graph := NewStateGraph[TestState]("test")

	graph.AddNode("start", func(ctx context.Context, s TestState) (TestState, error) {
		s.Counter++
		return s, nil
	})

	graph.SetEntryPoint("start")
	graph.AddEdge("start", END)

	compiled, err := graph.Compile()
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}

	if compiled == nil {
		t.Fatal("Compile returned nil")
	}
}

// TestCompile_NoEntryPoint 测试没有入口点的编译错误
func TestCompile_NoEntryPoint(t *testing.T) {
	graph := NewStateGraph[TestState]("test")

	graph.AddNode("node1", func(ctx context.Context, s TestState) (TestState, error) {
		return s, nil
	})

	_, err := graph.Compile()
	if !errors.Is(err, ErrNoEntryPoint) {
		t.Errorf("expected ErrNoEntryPoint, got %v", err)
	}
}

// TestInvoke_Simple 测试简单执行
func TestInvoke_Simple(t *testing.T) {
	graph := NewStateGraph[TestState]("test")

	graph.AddNode("increment", func(ctx context.Context, s TestState) (TestState, error) {
		s.Counter++
		return s, nil
	})

	graph.SetEntryPoint("increment")
	graph.AddEdge("increment", END)

	compiled, err := graph.Compile()
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}

	result, err := compiled.Invoke(context.Background(), TestState{Counter: 0})
	if err != nil {
		t.Fatalf("Invoke failed: %v", err)
	}

	if result.Counter != 1 {
		t.Errorf("expected Counter=1, got %d", result.Counter)
	}
}

// TestInvoke_MultipleNodes 测试多节点执行
func TestInvoke_MultipleNodes(t *testing.T) {
	graph := NewStateGraph[TestState]("test")

	graph.AddNode("increment", func(ctx context.Context, s TestState) (TestState, error) {
		s.Counter++
		return s, nil
	})

	graph.AddNode("double", func(ctx context.Context, s TestState) (TestState, error) {
		s.Counter *= 2
		return s, nil
	})

	graph.SetEntryPoint("increment")
	graph.AddEdge("increment", "double")
	graph.AddEdge("double", END)

	compiled, _ := graph.Compile()
	result, err := compiled.Invoke(context.Background(), TestState{Counter: 5})
	if err != nil {
		t.Fatalf("Invoke failed: %v", err)
	}

	// (5 + 1) * 2 = 12
	if result.Counter != 12 {
		t.Errorf("expected Counter=12, got %d", result.Counter)
	}
}

// TestInvoke_ConditionalEdge 测试条件边执行
func TestInvoke_ConditionalEdge(t *testing.T) {
	graph := NewStateGraph[TestState]("test")

	graph.AddNode("check", func(ctx context.Context, s TestState) (TestState, error) {
		s.Counter++
		return s, nil
	})

	graph.AddNode("positive", func(ctx context.Context, s TestState) (TestState, error) {
		s.Message = "positive"
		return s, nil
	})

	graph.AddNode("negative", func(ctx context.Context, s TestState) (TestState, error) {
		s.Message = "negative"
		return s, nil
	})

	graph.SetEntryPoint("check")

	graph.AddConditionalEdges("check",
		func(s TestState) string {
			if s.Counter > 0 {
				return "pos"
			}
			return "neg"
		},
		map[string]string{
			"pos": "positive",
			"neg": "negative",
		},
	)

	graph.AddEdge("positive", END)
	graph.AddEdge("negative", END)

	compiled, _ := graph.Compile()

	// 测试正数路径
	result, err := compiled.Invoke(context.Background(), TestState{Counter: 0})
	if err != nil {
		t.Fatalf("Invoke failed: %v", err)
	}

	if result.Message != "positive" {
		t.Errorf("expected Message='positive', got %s", result.Message)
	}

	// 测试负数路径
	result, err = compiled.Invoke(context.Background(), TestState{Counter: -5})
	if err != nil {
		t.Fatalf("Invoke failed: %v", err)
	}

	if result.Message != "negative" {
		t.Errorf("expected Message='negative', got %s", result.Message)
	}
}

// TestInvoke_Loop 测试循环（自循环）
func TestInvoke_Loop(t *testing.T) {
	graph := NewStateGraph[TestState]("test")

	graph.AddNode("loop", func(ctx context.Context, s TestState) (TestState, error) {
		s.Counter++
		return s, nil
	})

	graph.SetEntryPoint("loop")

	graph.AddConditionalEdges("loop",
		func(s TestState) string {
			if s.Counter >= 5 {
				return "end"
			}
			return "continue"
		},
		map[string]string{
			"continue": "loop",
			"end":      END,
		},
	)

	compiled, _ := graph.Compile()
	result, err := compiled.Invoke(context.Background(), TestState{Counter: 0})
	if err != nil {
		t.Fatalf("Invoke failed: %v", err)
	}

	if result.Counter != 5 {
		t.Errorf("expected Counter=5, got %d", result.Counter)
	}
}

// TestInvoke_NodeError 测试节点错误处理
func TestInvoke_NodeError(t *testing.T) {
	graph := NewStateGraph[TestState]("test")

	expectedErr := errors.New("node error")
	graph.AddNode("error", func(ctx context.Context, s TestState) (TestState, error) {
		return s, expectedErr
	})

	graph.SetEntryPoint("error")
	graph.AddEdge("error", END)

	compiled, _ := graph.Compile()
	_, err := compiled.Invoke(context.Background(), TestState{})

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, expectedErr) {
		t.Errorf("expected error to wrap expectedErr, got %v", err)
	}
}

// TestInvoke_ContextCancellation 测试上下文取消
func TestInvoke_ContextCancellation(t *testing.T) {
	graph := NewStateGraph[TestState]("test")

	graph.AddNode("slow", func(ctx context.Context, s TestState) (TestState, error) {
		<-ctx.Done()
		return s, ctx.Err()
	})

	graph.SetEntryPoint("slow")
	graph.AddEdge("slow", END)

	compiled, _ := graph.Compile()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // 立即取消

	_, err := compiled.Invoke(ctx, TestState{})
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

// TestWithCheckpointer 测试设置检查点器
func TestWithCheckpointer(t *testing.T) {
	graph := NewStateGraph[TestState]("test")

	// 使用占位符（实际实现在 M38-M42）
	mockCheckpointer := "mock_checkpointer"

	result := graph.WithCheckpointer(mockCheckpointer)

	if result != graph {
		t.Error("WithCheckpointer should return self")
	}

	if graph.checkpointer != mockCheckpointer {
		t.Error("checkpointer not set correctly")
	}
}

// TestWithDurability 测试设置持久化模式
func TestWithDurability(t *testing.T) {
	graph := NewStateGraph[TestState]("test")

	// 使用占位符（实际实现在 M43-M45）
	mockMode := "sync"

	result := graph.WithDurability(mockMode)

	if result != graph {
		t.Error("WithDurability should return self")
	}

	if graph.durability != mockMode {
		t.Error("durability not set correctly")
	}
}
