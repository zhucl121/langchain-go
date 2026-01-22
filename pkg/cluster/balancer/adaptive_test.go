package balancer

import (
	"context"
	"testing"
	"time"

	"github.com/zhucl121/langchain-go/pkg/cluster/node"
)

func TestAdaptiveBalancer_SelectNode(t *testing.T) {
	nodes := createTestNodes()
	lb := NewAdaptiveBalancer(nodes, 10)

	ctx := context.Background()
	req := &Request{
		ID:   "req-1",
		Type: RequestTypeLLM,
	}

	// 第一次选择
	selected, err := lb.SelectNode(ctx, req)
	if err != nil {
		t.Fatalf("SelectNode() error = %v", err)
	}

	if selected == nil {
		t.Error("Expected to select a node")
	}
}

func TestAdaptiveBalancer_ScoreCalculation(t *testing.T) {
	nodes := []*node.Node{
		{
			ID:      "good-node",
			Name:    "good-node",
			Address: "192.168.1.10",
			Port:    8080,
			Status:  node.StatusOnline,
			Roles:   []node.NodeRole{node.RoleWorker},
			Capacity: node.Capacity{
				MaxConnections: 1000,
				MaxMemoryMB:    4096,
			},
			Load: node.Load{
				CurrentConnections: 100, // 10% 使用
				CPUUsagePercent:    20,  // 低 CPU
				MemoryUsageMB:      512, // 12.5% 使用
			},
		},
		{
			ID:      "bad-node",
			Name:    "bad-node",
			Address: "192.168.1.11",
			Port:    8080,
			Status:  node.StatusOnline,
			Roles:   []node.NodeRole{node.RoleWorker},
			Capacity: node.Capacity{
				MaxConnections: 1000,
				MaxMemoryMB:    4096,
			},
			Load: node.Load{
				CurrentConnections: 900, // 90% 使用
				CPUUsagePercent:    85,  // 高 CPU
				MemoryUsageMB:      3500, // 85% 使用
			},
		},
	}

	lb := NewAdaptiveBalancer(nodes, 10)

	// 记录一些历史数据
	// good-node: 快速响应，高成功率
	for i := 0; i < 5; i++ {
		lb.RecordResult("good-node", true, 50*time.Millisecond)
	}

	// bad-node: 慢响应，低成功率
	for i := 0; i < 5; i++ {
		lb.RecordResult("bad-node", i < 2, 500*time.Millisecond) // 40% 成功率
	}

	// 获取得分
	goodScore := lb.GetScore("good-node")
	badScore := lb.GetScore("bad-node")

	// good-node 应该有更高的得分
	if goodScore <= badScore {
		t.Errorf("Expected good-node score (%.3f) > bad-node score (%.3f)", goodScore, badScore)
	}
}

func TestAdaptiveBalancer_AdaptiveSelection(t *testing.T) {
	nodes := createTestNodes()
	lb := NewAdaptiveBalancer(nodes, 10)

	ctx := context.Background()

	// 先给每个节点记录一些数据
	// node-1: 总是成功，快速
	for i := 0; i < 5; i++ {
		lb.RecordResult("node-1", true, 50*time.Millisecond)
	}

	// node-2: 成功率 50%，中等速度
	for i := 0; i < 5; i++ {
		lb.RecordResult("node-2", i < 2, 100*time.Millisecond)
	}

	// node-3: 成功率高，但较慢
	for i := 0; i < 5; i++ {
		lb.RecordResult("node-3", true, 200*time.Millisecond)
	}

	// 然后进行选择
	req := &Request{
		ID:   "req-1",
		Type: RequestTypeLLM,
	}

	selectedCounts := make(map[string]int)
	for i := 0; i < 30; i++ {
		selected, err := lb.SelectNode(ctx, req)
		if err != nil {
			t.Fatalf("SelectNode() error = %v", err)
		}
		selectedCounts[selected.ID]++
	}

	// 获取得分
	score1 := lb.GetScore("node-1")
	score2 := lb.GetScore("node-2")
	score3 := lb.GetScore("node-3")

	t.Logf("Scores: node-1=%.3f, node-2=%.3f, node-3=%.3f", score1, score2, score3)
	t.Logf("Selection counts: node-1=%d, node-2=%d, node-3=%d", 
		selectedCounts["node-1"], selectedCounts["node-2"], selectedCounts["node-3"])

	// node-2 应该有最低的得分（成功率低）
	if score2 > score1 || score2 > score3 {
		t.Errorf("Expected node-2 to have lowest score, got: node-1=%.3f, node-2=%.3f, node-3=%.3f",
			score1, score2, score3)
	}
}

func TestAdaptiveBalancer_UpdateNodes(t *testing.T) {
	nodes := createTestNodes()
	lb := NewAdaptiveBalancer(nodes[:2], 10)

	// 更新节点列表
	lb.UpdateNodes(nodes)

	// 验证所有节点都有得分
	for _, n := range nodes {
		score := lb.GetScore(n.ID)
		if score == 0 {
			t.Errorf("Node %s has zero score", n.ID)
		}
	}
}
