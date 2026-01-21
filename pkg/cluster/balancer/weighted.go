package balancer

import (
	"context"
	"errors"
	"math/rand"
	"sync"
	"time"

	"github.com/zhucl121/langchain-go/pkg/cluster/node"
)

// WeightedBalancer 加权负载均衡器
//
// 根据节点权重分配请求，权重越高的节点接收的请求越多。
type WeightedBalancer struct {
	nodes   []*node.Node
	weights []int
	total   int
	mu      sync.RWMutex
	stats   *Stats
	rng     *rand.Rand
}

// NewWeightedBalancer 创建加权负载均衡器
//
// 如果 weights 为 nil，则根据节点容量自动计算权重。
func NewWeightedBalancer(nodes []*node.Node, weights []int) *WeightedBalancer {
	healthyNodes := filterHealthyNodes(nodes)

	// 如果没有提供权重，根据节点容量自动计算
	if weights == nil || len(weights) != len(healthyNodes) {
		weights = calculateWeights(healthyNodes)
	}

	// 计算总权重
	total := 0
	for _, w := range weights {
		total += w
	}

	lb := &WeightedBalancer{
		nodes:   healthyNodes,
		weights: weights,
		total:   total,
		stats: &Stats{
			NodeStats: make(map[string]*NodeStats),
		},
		rng: rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	// 初始化统计
	for _, n := range lb.nodes {
		lb.stats.NodeStats[n.ID] = &NodeStats{
			NodeID: n.ID,
		}
	}

	return lb
}

// SelectNode 选择节点
func (b *WeightedBalancer) SelectNode(ctx context.Context, req *Request) (*node.Node, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if len(b.nodes) == 0 {
		return nil, ErrNoAvailableNodes
	}

	if b.total == 0 {
		// 如果总权重为 0，随机选择
		index := b.rng.Intn(len(b.nodes))
		return b.nodes[index], nil
	}

	// 加权随机选择
	randVal := b.rng.Intn(b.total)
	cumulative := 0

	for i, w := range b.weights {
		cumulative += w
		if randVal < cumulative {
			selected := b.nodes[i]

			// 更新统计
			b.stats.TotalRequests++
			if stats, ok := b.stats.NodeStats[selected.ID]; ok {
				stats.Requests++
				stats.LastUsed = time.Now()
			}

			return selected, nil
		}
	}

	// 理论上不应该到这里，但作为后备方案
	return b.nodes[0], nil
}

// UpdateNodes 更新节点列表
func (b *WeightedBalancer) UpdateNodes(nodes []*node.Node) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.nodes = filterHealthyNodes(nodes)
	b.weights = calculateWeights(b.nodes)

	// 重新计算总权重
	b.total = 0
	for _, w := range b.weights {
		b.total += w
	}

	// 更新统计
	validNodeIDs := make(map[string]bool)
	for _, n := range b.nodes {
		validNodeIDs[n.ID] = true
		if _, exists := b.stats.NodeStats[n.ID]; !exists {
			b.stats.NodeStats[n.ID] = &NodeStats{
				NodeID: n.ID,
			}
		}
	}

	// 删除不存在的节点统计
	for nodeID := range b.stats.NodeStats {
		if !validNodeIDs[nodeID] {
			delete(b.stats.NodeStats, nodeID)
		}
	}
}

// UpdateWeights 更新权重
func (b *WeightedBalancer) UpdateWeights(weights []int) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if len(weights) != len(b.nodes) {
		return errors.New("balancer: weights length must match nodes length")
	}

	b.weights = weights

	// 重新计算总权重
	b.total = 0
	for _, w := range b.weights {
		b.total += w
	}

	return nil
}

// RecordResult 记录请求结果
func (b *WeightedBalancer) RecordResult(nodeID string, success bool, latency time.Duration) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if success {
		b.stats.SuccessRequests++
	} else {
		b.stats.FailedRequests++
	}

	if stats, ok := b.stats.NodeStats[nodeID]; ok {
		if success {
			stats.SuccessRequests++
		} else {
			stats.FailedRequests++
		}

		// 更新平均延迟
		if stats.AverageLatency == 0 {
			stats.AverageLatency = latency
		} else {
			stats.AverageLatency = (stats.AverageLatency + latency) / 2
		}
	}
}

// GetStats 获取统计信息
func (b *WeightedBalancer) GetStats() *Stats {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.stats
}

// GetWeights 获取当前权重
func (b *WeightedBalancer) GetWeights() []int {
	b.mu.RLock()
	defer b.mu.RUnlock()

	weights := make([]int, len(b.weights))
	copy(weights, b.weights)
	return weights
}

// calculateWeights 根据节点容量计算权重
func calculateWeights(nodes []*node.Node) []int {
	weights := make([]int, len(nodes))

	for i, n := range nodes {
		// 基于最大连接数计算权重
		weight := n.Capacity.MaxConnections
		if weight <= 0 {
			weight = 100 // 默认权重
		}

		// 根据当前负载调整权重
		loadPercent := n.GetLoadPercent()
		if loadPercent > 0 {
			// 负载越高，权重越低
			weight = int(float64(weight) * (1.0 - loadPercent/100.0))
		}

		if weight < 1 {
			weight = 1 // 最小权重为 1
		}

		weights[i] = weight
	}

	return weights
}
