package balancer

import (
	"context"
	"sync"
	"time"

	"github.com/zhucl121/langchain-go/pkg/cluster/node"
)

// LeastConnectionBalancer 最少连接负载均衡器
//
// 选择当前连接数最少的节点，适合长连接场景。
type LeastConnectionBalancer struct {
	nodes       []*node.Node
	connections map[string]int
	mu          sync.RWMutex
	stats       *Stats
}

// NewLeastConnectionBalancer 创建最少连接负载均衡器
func NewLeastConnectionBalancer(nodes []*node.Node) *LeastConnectionBalancer {
	lb := &LeastConnectionBalancer{
		nodes:       filterHealthyNodes(nodes),
		connections: make(map[string]int),
		stats: &Stats{
			NodeStats: make(map[string]*NodeStats),
		},
	}

	// 初始化连接计数
	for _, n := range lb.nodes {
		lb.connections[n.ID] = 0
		lb.stats.NodeStats[n.ID] = &NodeStats{
			NodeID:             n.ID,
			CurrentConnections: 0,
		}
	}

	return lb
}

// SelectNode 选择节点
func (b *LeastConnectionBalancer) SelectNode(ctx context.Context, req *Request) (*node.Node, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if len(b.nodes) == 0 {
		return nil, ErrNoAvailableNodes
	}

	// 找出连接数最少的节点
	var selected *node.Node
	minConn := int(^uint(0) >> 1) // 最大整数

	for _, n := range b.nodes {
		conn := b.connections[n.ID]
		if conn < minConn {
			minConn = conn
			selected = n
		}
	}

	if selected == nil {
		selected = b.nodes[0]
	}

	// 增加连接计数
	b.connections[selected.ID]++

	// 更新统计
	b.stats.TotalRequests++
	if stats, ok := b.stats.NodeStats[selected.ID]; ok {
		stats.Requests++
		stats.CurrentConnections = b.connections[selected.ID]
		stats.LastUsed = time.Now()
	}

	return selected, nil
}

// UpdateNodes 更新节点列表
func (b *LeastConnectionBalancer) UpdateNodes(nodes []*node.Node) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.nodes = filterHealthyNodes(nodes)

	// 清理已删除节点的连接计数
	validNodeIDs := make(map[string]bool)
	for _, n := range b.nodes {
		validNodeIDs[n.ID] = true
		// 如果是新节点，初始化连接计数
		if _, exists := b.connections[n.ID]; !exists {
			b.connections[n.ID] = 0
			b.stats.NodeStats[n.ID] = &NodeStats{
				NodeID:             n.ID,
				CurrentConnections: 0,
			}
		}
	}

	// 删除不存在的节点
	for nodeID := range b.connections {
		if !validNodeIDs[nodeID] {
			delete(b.connections, nodeID)
			delete(b.stats.NodeStats, nodeID)
		}
	}
}

// RecordResult 记录请求结果
func (b *LeastConnectionBalancer) RecordResult(nodeID string, success bool, latency time.Duration) {
	b.mu.Lock()
	defer b.mu.Unlock()

	// 减少连接计数（请求完成）
	if conn, ok := b.connections[nodeID]; ok && conn > 0 {
		b.connections[nodeID]--
	}

	// 更新统计
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
		stats.CurrentConnections = b.connections[nodeID]

		// 更新平均延迟
		if stats.AverageLatency == 0 {
			stats.AverageLatency = latency
		} else {
			stats.AverageLatency = (stats.AverageLatency + latency) / 2
		}
	}
}

// GetStats 获取统计信息
func (b *LeastConnectionBalancer) GetStats() *Stats {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.stats
}

// GetConnectionCount 获取指定节点的连接数
func (b *LeastConnectionBalancer) GetConnectionCount(nodeID string) int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.connections[nodeID]
}
