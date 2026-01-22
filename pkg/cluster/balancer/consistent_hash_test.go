package balancer

import (
	"context"
	"testing"
)

func TestConsistentHashBalancer_SelectNode(t *testing.T) {
	nodes := createTestNodes()
	lb := NewConsistentHashBalancer(nodes, 150)

	ctx := context.Background()

	// 使用相同的请求 ID，应该总是返回相同的节点
	req1 := &Request{
		ID:   "user-123",
		Type: RequestTypeLLM,
	}

	selected1, err := lb.SelectNode(ctx, req1)
	if err != nil {
		t.Fatalf("SelectNode() error = %v", err)
	}

	// 再次选择，应该是同一个节点
	for i := 0; i < 10; i++ {
		selected2, err := lb.SelectNode(ctx, req1)
		if err != nil {
			t.Fatalf("SelectNode() error = %v", err)
		}

		if selected2.ID != selected1.ID {
			t.Errorf("Consistent hash failed: expected %s, got %s", selected1.ID, selected2.ID)
		}
	}
}

func TestConsistentHashBalancer_DifferentKeys(t *testing.T) {
	nodes := createTestNodes()
	lb := NewConsistentHashBalancer(nodes, 150)

	ctx := context.Background()

	// 使用不同的请求，应该分布到不同的节点
	selectedNodes := make(map[string]bool)
	for i := 0; i < 100; i++ {
		req := &Request{
			ID:   string(rune(i)),
			Type: RequestTypeLLM,
		}
		selected, err := lb.SelectNode(ctx, req)
		if err != nil {
			t.Fatalf("SelectNode() error = %v", err)
		}
		selectedNodes[selected.ID] = true
	}

	// 应该使用多个节点
	if len(selectedNodes) < 2 {
		t.Errorf("Expected requests to be distributed across multiple nodes, got %d", len(selectedNodes))
	}
}

func TestConsistentHashBalancer_UserID(t *testing.T) {
	nodes := createTestNodes()
	lb := NewConsistentHashBalancer(nodes, 150)

	ctx := context.Background()

	// 使用 UserID 作为哈希键
	req1 := &Request{
		ID:     "req-1",
		UserID: "user-alice",
		Type:   RequestTypeLLM,
	}

	req2 := &Request{
		ID:     "req-2",
		UserID: "user-alice",
		Type:   RequestTypeLLM,
	}

	selected1, _ := lb.SelectNode(ctx, req1)
	selected2, _ := lb.SelectNode(ctx, req2)

	// 相同的 UserID 应该路由到相同的节点
	if selected1.ID != selected2.ID {
		t.Errorf("Same UserID should route to same node: %s vs %s", selected1.ID, selected2.ID)
	}
}

func TestConsistentHashBalancer_UpdateNodes(t *testing.T) {
	nodes := createTestNodes()
	lb := NewConsistentHashBalancer(nodes[:2], 150)

	ctx := context.Background()
	req := &Request{
		ID:   "user-123",
		Type: RequestTypeLLM,
	}

	_, _ = lb.SelectNode(ctx, req)

	// 添加新节点
	lb.UpdateNodes(nodes)

	// 相同的请求可能会路由到不同的节点（因为环重建了）
	selected2, _ := lb.SelectNode(ctx, req)

	// 但应该能成功选择节点
	if selected2 == nil {
		t.Error("Expected to select a node after update")
	}

	// 验证一致性：多次选择应该返回相同节点
	for i := 0; i < 5; i++ {
		selected3, _ := lb.SelectNode(ctx, req)
		if selected3.ID != selected2.ID {
			t.Errorf("Consistency broken after update: expected %s, got %s", selected2.ID, selected3.ID)
		}
	}
}
