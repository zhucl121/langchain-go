package balancer

import (
	"context"
	"sync"
	"time"

	"github.com/zhucl121/langchain-go/pkg/cluster/node"
)

// AdaptiveBalancer 自适应负载均衡器
//
// 根据节点的实时性能（响应时间、成功率、负载）动态调整选择策略。
type AdaptiveBalancer struct {
	nodes       []*node.Node
	scores      map[string]float64
	metrics     map[string]*metricsWindow
	windowSize  int
	mu          sync.RWMutex
	stats       *Stats
}

// metricsWindow 指标窗口
type metricsWindow struct {
	latencies   []time.Duration
	successes   []bool
	currentIdx  int
	full        bool
}

// NewAdaptiveBalancer 创建自适应负载均衡器
//
// windowSize 是指标窗口大小，用于计算平均性能。
func NewAdaptiveBalancer(nodes []*node.Node, windowSize int) *AdaptiveBalancer {
	if windowSize <= 0 {
		windowSize = 100
	}

	lb := &AdaptiveBalancer{
		nodes:      filterHealthyNodes(nodes),
		scores:     make(map[string]float64),
		metrics:    make(map[string]*metricsWindow),
		windowSize: windowSize,
		stats: &Stats{
			NodeStats: make(map[string]*NodeStats),
		},
	}

	// 初始化节点得分和指标窗口
	for _, n := range lb.nodes {
		lb.scores[n.ID] = 1.0 // 初始得分为 1.0
		lb.metrics[n.ID] = &metricsWindow{
			latencies: make([]time.Duration, windowSize),
			successes: make([]bool, windowSize),
		}
		lb.stats.NodeStats[n.ID] = &NodeStats{
			NodeID: n.ID,
			Score:  1.0,
		}
	}

	return lb
}

// SelectNode 选择节点
func (b *AdaptiveBalancer) SelectNode(ctx context.Context, req *Request) (*node.Node, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if len(b.nodes) == 0 {
		return nil, ErrNoAvailableNodes
	}

	// 选择得分最高的节点
	var selected *node.Node
	maxScore := 0.0

	for _, n := range b.nodes {
		score := b.calculateScore(n)
		if score > maxScore {
			maxScore = score
			selected = n
		}
	}

	if selected == nil {
		selected = b.nodes[0]
	}

	// 更新统计
	b.stats.TotalRequests++
	if stats, ok := b.stats.NodeStats[selected.ID]; ok {
		stats.Requests++
		stats.LastUsed = time.Now()
		stats.Score = maxScore
	}

	return selected, nil
}

// UpdateNodes 更新节点列表
func (b *AdaptiveBalancer) UpdateNodes(nodes []*node.Node) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.nodes = filterHealthyNodes(nodes)

	// 更新节点得分
	validNodeIDs := make(map[string]bool)
	for _, n := range b.nodes {
		validNodeIDs[n.ID] = true
		// 如果是新节点，初始化
		if _, exists := b.scores[n.ID]; !exists {
			b.scores[n.ID] = 1.0
			b.metrics[n.ID] = &metricsWindow{
				latencies: make([]time.Duration, b.windowSize),
				successes: make([]bool, b.windowSize),
			}
			b.stats.NodeStats[n.ID] = &NodeStats{
				NodeID: n.ID,
				Score:  1.0,
			}
		}
	}

	// 删除不存在的节点
	for nodeID := range b.scores {
		if !validNodeIDs[nodeID] {
			delete(b.scores, nodeID)
			delete(b.metrics, nodeID)
			delete(b.stats.NodeStats, nodeID)
		}
	}
}

// RecordResult 记录请求结果
func (b *AdaptiveBalancer) RecordResult(nodeID string, success bool, latency time.Duration) {
	b.mu.Lock()
	defer b.mu.Unlock()

	// 更新统计
	if success {
		b.stats.SuccessRequests++
	} else {
		b.stats.FailedRequests++
	}

	// 记录到指标窗口
	if metrics, ok := b.metrics[nodeID]; ok {
		idx := metrics.currentIdx
		metrics.latencies[idx] = latency
		metrics.successes[idx] = success

		metrics.currentIdx = (metrics.currentIdx + 1) % b.windowSize
		if !metrics.full && metrics.currentIdx == 0 {
			metrics.full = true
		}
	}

	// 重新计算节点得分
	for _, n := range b.nodes {
		if n.ID == nodeID {
			b.scores[nodeID] = b.calculateScore(n)
			break
		}
	}

	// 更新节点统计
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

		stats.Score = b.scores[nodeID]
	}
}

// GetStats 获取统计信息
func (b *AdaptiveBalancer) GetStats() *Stats {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.stats
}

// calculateScore 计算节点得分
//
// 考虑多个因素：
// 1. CPU 使用率（权重 0.3）
// 2. 内存使用率（权重 0.2）
// 3. 当前连接数（权重 0.2）
// 4. 历史成功率（权重 0.15）
// 5. 平均延迟（权重 0.15）
func (b *AdaptiveBalancer) calculateScore(n *node.Node) float64 {
	// 1. CPU 得分（使用率越低越好）
	cpuScore := 1.0
	if n.Load.CPUUsagePercent > 0 {
		cpuScore = 1.0 - (n.Load.CPUUsagePercent / 100.0)
	}

	// 2. 内存得分
	memScore := 1.0
	if n.Capacity.MaxMemoryMB > 0 && n.Load.MemoryUsageMB > 0 {
		memUsage := float64(n.Load.MemoryUsageMB) / float64(n.Capacity.MaxMemoryMB)
		memScore = 1.0 - memUsage
	}

	// 3. 连接数得分
	connScore := 1.0
	if n.Capacity.MaxConnections > 0 && n.Load.CurrentConnections > 0 {
		connUsage := float64(n.Load.CurrentConnections) / float64(n.Capacity.MaxConnections)
		connScore = 1.0 - connUsage
	}

	// 4. 历史成功率
	successScore := 1.0
	if metrics, ok := b.metrics[n.ID]; ok {
		successCount := 0
		totalCount := b.windowSize
		if !metrics.full {
			totalCount = metrics.currentIdx
		}

		if totalCount > 0 {
			for i := 0; i < totalCount; i++ {
				if metrics.successes[i] {
					successCount++
				}
			}
			successScore = float64(successCount) / float64(totalCount)
		}
	}

	// 5. 平均延迟得分
	latencyScore := 1.0
	if metrics, ok := b.metrics[n.ID]; ok {
		var totalLatency time.Duration
		totalCount := b.windowSize
		if !metrics.full {
			totalCount = metrics.currentIdx
		}

		if totalCount > 0 {
			for i := 0; i < totalCount; i++ {
				totalLatency += metrics.latencies[i]
			}
			avgLatency := totalLatency / time.Duration(totalCount)

			// 假设 1 秒是基准延迟
			baseLatency := 1 * time.Second
			if avgLatency > 0 && avgLatency < baseLatency {
				latencyScore = 1.0 - (float64(avgLatency) / float64(baseLatency))
			} else if avgLatency >= baseLatency {
				latencyScore = 0.1 // 延迟过高
			}
		}
	}

	// 加权计算总得分
	score := 0.3*cpuScore + 0.2*memScore + 0.2*connScore + 0.15*successScore + 0.15*latencyScore

	// 确保得分在 [0, 1] 范围内
	if score < 0 {
		score = 0
	}
	if score > 1 {
		score = 1
	}

	return score
}

// GetScore 获取指定节点的得分
func (b *AdaptiveBalancer) GetScore(nodeID string) float64 {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.scores[nodeID]
}
