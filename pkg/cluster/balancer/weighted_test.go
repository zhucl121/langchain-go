package balancer

import (
	"context"
	"testing"
)

func TestWeightedBalancer_SelectNode(t *testing.T) {
	nodes := createTestNodes()
	weights := []int{1, 2, 3} // node-3 权重最高
	lb := NewWeightedBalancer(nodes, weights)

	ctx := context.Background()
	req := &Request{
		ID:   "req-1",
		Type: RequestTypeLLM,
	}

	// 选择多次，统计分布
	selectedNodes := make(map[string]int)
	for i := 0; i < 600; i++ {
		selected, err := lb.SelectNode(ctx, req)
		if err != nil {
			t.Fatalf("SelectNode() error = %v", err)
		}
		selectedNodes[selected.ID]++
	}

	// 验证分布大致符合权重比例
	// node-1: ~100 (1/6), node-2: ~200 (2/6), node-3: ~300 (3/6)
	count1 := selectedNodes["node-1"]
	count2 := selectedNodes["node-2"]
	count3 := selectedNodes["node-3"]

	// 允许一定误差
	if count1 < 80 || count1 > 120 {
		t.Logf("Warning: node-1 count %d not in expected range [80, 120]", count1)
	}
	if count2 < 180 || count2 > 220 {
		t.Logf("Warning: node-2 count %d not in expected range [180, 220]", count2)
	}
	if count3 < 280 || count3 > 320 {
		t.Logf("Warning: node-3 count %d not in expected range [280, 320]", count3)
	}

	// 至少验证顺序
	if !(count1 < count2 && count2 < count3) {
		t.Errorf("Weight distribution incorrect: node1=%d, node2=%d, node3=%d", count1, count2, count3)
	}
}

func TestWeightedBalancer_AutoWeights(t *testing.T) {
	nodes := createTestNodes()
	lb := NewWeightedBalancer(nodes, nil) // 自动计算权重

	weights := lb.GetWeights()
	if len(weights) != len(nodes) {
		t.Errorf("Expected %d weights, got %d", len(nodes), len(weights))
	}

	// 验证所有权重都大于 0
	for i, w := range weights {
		if w <= 0 {
			t.Errorf("Weight at index %d is %d, expected > 0", i, w)
		}
	}
}

func TestWeightedBalancer_UpdateWeights(t *testing.T) {
	nodes := createTestNodes()
	lb := NewWeightedBalancer(nodes, []int{1, 1, 1})

	// 更新权重
	newWeights := []int{1, 2, 3}
	err := lb.UpdateWeights(newWeights)
	if err != nil {
		t.Fatalf("UpdateWeights() error = %v", err)
	}

	weights := lb.GetWeights()
	for i, w := range weights {
		if w != newWeights[i] {
			t.Errorf("Weight at index %d = %d, expected %d", i, w, newWeights[i])
		}
	}
}

func TestWeightedBalancer_UpdateWeights_InvalidLength(t *testing.T) {
	nodes := createTestNodes()
	lb := NewWeightedBalancer(nodes, []int{1, 1, 1})

	// 尝试更新错误长度的权重
	err := lb.UpdateWeights([]int{1, 2})
	if err == nil {
		t.Error("Expected error for invalid weights length")
	}
}
