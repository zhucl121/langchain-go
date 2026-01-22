package balancer

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/zhucl121/langchain-go/pkg/cluster/node"
)

// RoundRobinBalancer 轮询负载均衡器
//
// 按照轮询顺序依次选择节点，确保请求均匀分布。
type RoundRobinBalancer struct {
	nodes   []*node.Node
	current uint32
	mu      sync.RWMutex
	stats   *Stats
}

// NewRoundRobinBalancer 创建轮询负载均衡器
func NewRoundRobinBalancer(nodes []*node.Node) *RoundRobinBalancer {
	return &RoundRobinBalancer{
		nodes: filterHealthyNodes(nodes),
		stats: &Stats{
			NodeStats: make(map[string]*NodeStats),
		},
	}
}

// SelectNode 选择节点
func (b *RoundRobinBalancer) SelectNode(ctx context.Context, req *Request) (*node.Node, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if len(b.nodes) == 0 {
		return nil, ErrNoAvailableNodes
	}

	// 原子递增并取模
	index := atomic.AddUint32(&b.current, 1) % uint32(len(b.nodes))
	selected := b.nodes[index]

	// 更新统计
	atomic.AddInt64(&b.stats.TotalRequests, 1)
	if stats, ok := b.stats.NodeStats[selected.ID]; ok {
		atomic.AddInt64(&stats.Requests, 1)
	} else {
		b.stats.NodeStats[selected.ID] = &NodeStats{
			NodeID:   selected.ID,
			Requests: 1,
		}
	}

	return selected, nil
}

// UpdateNodes 更新节点列表
func (b *RoundRobinBalancer) UpdateNodes(nodes []*node.Node) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.nodes = filterHealthyNodes(nodes)
	// 重置计数器，避免索引越界
	atomic.StoreUint32(&b.current, 0)
}

// RecordResult 记录请求结果
func (b *RoundRobinBalancer) RecordResult(nodeID string, success bool, latency time.Duration) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if success {
		atomic.AddInt64(&b.stats.SuccessRequests, 1)
	} else {
		atomic.AddInt64(&b.stats.FailedRequests, 1)
	}

	if stats, ok := b.stats.NodeStats[nodeID]; ok {
		if success {
			atomic.AddInt64(&stats.SuccessRequests, 1)
		} else {
			atomic.AddInt64(&stats.FailedRequests, 1)
		}
	}
}

// GetStats 获取统计信息
func (b *RoundRobinBalancer) GetStats() *Stats {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.stats
}
