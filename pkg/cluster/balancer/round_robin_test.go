package balancer

import (
	"context"
	"testing"
	"time"

	"github.com/zhucl121/langchain-go/pkg/cluster/node"
)

func createTestNodes() []*node.Node {
	return []*node.Node{
		{
			ID:      "node-1",
			Name:    "node-1",
			Address: "192.168.1.10",
			Port:    8080,
			Status:  node.StatusOnline,
			Roles:   []node.NodeRole{node.RoleWorker},
			Capacity: node.Capacity{
				MaxConnections: 1000,
			},
			Load: node.Load{
				CurrentConnections: 100,
				CPUUsagePercent:    30,
			},
		},
		{
			ID:      "node-2",
			Name:    "node-2",
			Address: "192.168.1.11",
			Port:    8080,
			Status:  node.StatusOnline,
			Roles:   []node.NodeRole{node.RoleWorker},
			Capacity: node.Capacity{
				MaxConnections: 1000,
			},
			Load: node.Load{
				CurrentConnections: 200,
				CPUUsagePercent:    50,
			},
		},
		{
			ID:      "node-3",
			Name:    "node-3",
			Address: "192.168.1.12",
			Port:    8080,
			Status:  node.StatusOnline,
			Roles:   []node.NodeRole{node.RoleWorker},
			Capacity: node.Capacity{
				MaxConnections: 1000,
			},
			Load: node.Load{
				CurrentConnections: 150,
				CPUUsagePercent:    40,
			},
		},
	}
}

func TestRoundRobinBalancer_SelectNode(t *testing.T) {
	nodes := createTestNodes()
	lb := NewRoundRobinBalancer(nodes)

	ctx := context.Background()
	req := &Request{
		ID:   "req-1",
		Type: RequestTypeLLM,
	}

	// 测试轮询
	selectedNodes := make(map[string]int)
	for i := 0; i < 9; i++ {
		selected, err := lb.SelectNode(ctx, req)
		if err != nil {
			t.Fatalf("SelectNode() error = %v", err)
		}
		selectedNodes[selected.ID]++
	}

	// 每个节点应该被选中 3 次
	for _, n := range nodes {
		count := selectedNodes[n.ID]
		if count != 3 {
			t.Errorf("Node %s selected %d times, expected 3", n.ID, count)
		}
	}
}

func TestRoundRobinBalancer_NoNodes(t *testing.T) {
	lb := NewRoundRobinBalancer([]*node.Node{})

	ctx := context.Background()
	req := &Request{ID: "req-1"}

	_, err := lb.SelectNode(ctx, req)
	if err != ErrNoAvailableNodes {
		t.Errorf("Expected ErrNoAvailableNodes, got %v", err)
	}
}

func TestRoundRobinBalancer_UpdateNodes(t *testing.T) {
	nodes := createTestNodes()
	lb := NewRoundRobinBalancer(nodes[:2]) // 只添加前两个节点

	ctx := context.Background()
	req := &Request{ID: "req-1"}

	// 选择几次
	for i := 0; i < 4; i++ {
		lb.SelectNode(ctx, req)
	}

	// 更新节点列表
	lb.UpdateNodes(nodes) // 添加所有节点

	// 继续选择
	selectedNodes := make(map[string]int)
	for i := 0; i < 9; i++ {
		selected, err := lb.SelectNode(ctx, req)
		if err != nil {
			t.Fatalf("SelectNode() error = %v", err)
		}
		selectedNodes[selected.ID]++
	}

	// 所有节点都应该被选中
	if len(selectedNodes) != 3 {
		t.Errorf("Expected 3 nodes to be selected, got %d", len(selectedNodes))
	}
}

func TestRoundRobinBalancer_RecordResult(t *testing.T) {
	nodes := createTestNodes()
	lb := NewRoundRobinBalancer(nodes)

	// 记录一些结果
	lb.RecordResult("node-1", true, 100*time.Millisecond)
	lb.RecordResult("node-1", true, 150*time.Millisecond)
	lb.RecordResult("node-2", false, 200*time.Millisecond)

	stats := lb.GetStats()
	if stats.SuccessRequests != 2 {
		t.Errorf("Expected 2 success requests, got %d", stats.SuccessRequests)
	}
	if stats.FailedRequests != 1 {
		t.Errorf("Expected 1 failed request, got %d", stats.FailedRequests)
	}
}
