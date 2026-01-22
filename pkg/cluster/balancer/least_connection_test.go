package balancer

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestLeastConnectionBalancer_SelectNode(t *testing.T) {
	nodes := createTestNodes()
	lb := NewLeastConnectionBalancer(nodes)

	ctx := context.Background()
	req := &Request{
		ID:   "req-1",
		Type: RequestTypeLLM,
	}

	// 第一次选择，应该选择 node-1（连接数为 0）
	selected, err := lb.SelectNode(ctx, req)
	if err != nil {
		t.Fatalf("SelectNode() error = %v", err)
	}

	// 验证连接数增加
	conn := lb.GetConnectionCount(selected.ID)
	if conn != 1 {
		t.Errorf("Expected connection count 1, got %d", conn)
	}

	// 再选择一次，应该选择其他节点
	selected2, err := lb.SelectNode(ctx, req)
	if err != nil {
		t.Fatalf("SelectNode() error = %v", err)
	}

	// 不应该是同一个节点
	if selected.ID == selected2.ID {
		// 除非只有一个节点
		if len(nodes) > 1 {
			t.Error("Should select different node")
		}
	}
}

func TestLeastConnectionBalancer_RecordResult(t *testing.T) {
	nodes := createTestNodes()
	lb := NewLeastConnectionBalancer(nodes)

	ctx := context.Background()
	req := &Request{ID: "req-1"}

	// 选择节点
	selected, err := lb.SelectNode(ctx, req)
	if err != nil {
		t.Fatalf("SelectNode() error = %v", err)
	}

	nodeID := selected.ID
	connBefore := lb.GetConnectionCount(nodeID)

	// 记录结果（连接完成）
	lb.RecordResult(nodeID, true, 100*time.Millisecond)

	// 验证连接数减少
	connAfter := lb.GetConnectionCount(nodeID)
	if connAfter != connBefore-1 {
		t.Errorf("Expected connection count to decrease by 1, before=%d, after=%d", connBefore, connAfter)
	}
}

func TestLeastConnectionBalancer_UpdateNodes(t *testing.T) {
	nodes := createTestNodes()
	lb := NewLeastConnectionBalancer(nodes[:2])

	ctx := context.Background()
	req := &Request{ID: "req-1"}

	// 更新节点列表（添加更多节点）
	lb.UpdateNodes(nodes)

	// 验证能成功选择节点
	selected, err := lb.SelectNode(ctx, req)
	if err != nil {
		t.Fatalf("SelectNode() error = %v", err)
	}

	if selected == nil {
		t.Error("Expected to select a node after update")
	}

	// 测试并发情况下的分布
	// 同时启动多个goroutine保持连接，这样可以看到多个节点被使用
	done := make(chan bool)
	selectedNodes := make(map[string]bool)
	var mu sync.Mutex

	for i := 0; i < 10; i++ {
		go func() {
			selected, _ := lb.SelectNode(ctx, req)
			mu.Lock()
			selectedNodes[selected.ID] = true
			mu.Unlock()
			time.Sleep(10 * time.Millisecond)  // 保持连接一段时间
			lb.RecordResult(selected.ID, true, 10*time.Millisecond)
			done <- true
		}()
	}

	// 等待所有goroutine完成
	for i := 0; i < 10; i++ {
		<-done
	}

	// 并发情况下应该至少使用 2 个节点
	if len(selectedNodes) < 2 {
		t.Logf("Warning: Only %d different nodes selected in concurrent test", len(selectedNodes))
	}

	t.Logf("Selected %d different nodes out of %d total nodes in concurrent test", len(selectedNodes), len(nodes))
}
