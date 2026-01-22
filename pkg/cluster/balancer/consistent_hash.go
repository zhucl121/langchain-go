package balancer

import (
	"context"
	"crypto/md5"
	"encoding/binary"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/zhucl121/langchain-go/pkg/cluster/node"
)

// ConsistentHashBalancer 一致性哈希负载均衡器
//
// 使用一致性哈希算法，确保相同的请求总是路由到相同的节点。
// 适合需要会话保持或缓存亲和性的场景。
type ConsistentHashBalancer struct {
	ring        *hashRing
	nodes       []*node.Node
	virtualNum  int
	mu          sync.RWMutex
	stats       *Stats
}

// hashRing 哈希环
type hashRing struct {
	keys     []uint32
	ring     map[uint32]string
	nodeMap  map[string]*node.Node
	mu       sync.RWMutex
}

// NewConsistentHashBalancer 创建一致性哈希负载均衡器
//
// virtualNum 是虚拟节点数，越大分布越均匀，但内存占用也越大。
// 推荐值：150-200
func NewConsistentHashBalancer(nodes []*node.Node, virtualNum int) *ConsistentHashBalancer {
	if virtualNum <= 0 {
		virtualNum = 150
	}

	lb := &ConsistentHashBalancer{
		ring: &hashRing{
			ring:    make(map[uint32]string),
			nodeMap: make(map[string]*node.Node),
		},
		nodes:      filterHealthyNodes(nodes),
		virtualNum: virtualNum,
		stats: &Stats{
			NodeStats: make(map[string]*NodeStats),
		},
	}

	// 构建哈希环
	lb.rebuildRing()

	return lb
}

// SelectNode 选择节点
func (b *ConsistentHashBalancer) SelectNode(ctx context.Context, req *Request) (*node.Node, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if len(b.nodes) == 0 {
		return nil, ErrNoAvailableNodes
	}

	// 使用请求 ID 或 UserID 作为哈希键
	key := req.ID
	if req.UserID != "" {
		key = req.UserID
	}

	// 获取节点
	nodeID, ok := b.ring.GetNode(key)
	if !ok {
		// 如果找不到，返回第一个节点
		return b.nodes[0], nil
	}

	// 查找节点
	var selected *node.Node
	for _, n := range b.nodes {
		if n.ID == nodeID {
			selected = n
			break
		}
	}

	if selected == nil {
		return b.nodes[0], nil
	}

	// 更新统计
	b.stats.TotalRequests++
	if stats, ok := b.stats.NodeStats[selected.ID]; ok {
		stats.Requests++
		stats.LastUsed = time.Now()
	}

	return selected, nil
}

// UpdateNodes 更新节点列表
func (b *ConsistentHashBalancer) UpdateNodes(nodes []*node.Node) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.nodes = filterHealthyNodes(nodes)
	b.rebuildRing()

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

// RecordResult 记录请求结果
func (b *ConsistentHashBalancer) RecordResult(nodeID string, success bool, latency time.Duration) {
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
func (b *ConsistentHashBalancer) GetStats() *Stats {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.stats
}

// rebuildRing 重建哈希环
func (b *ConsistentHashBalancer) rebuildRing() {
	b.ring.mu.Lock()
	defer b.ring.mu.Unlock()

	// 清空现有环
	b.ring.ring = make(map[uint32]string)
	b.ring.nodeMap = make(map[string]*node.Node)
	b.ring.keys = nil

	// 添加节点及其虚拟节点
	for _, n := range b.nodes {
		b.ring.nodeMap[n.ID] = n

		for i := 0; i < b.virtualNum; i++ {
			// 为每个虚拟节点生成哈希值
			virtualKey := fmt.Sprintf("%s#%d", n.ID, i)
			hash := hashKey(virtualKey)
			b.ring.ring[hash] = n.ID
			b.ring.keys = append(b.ring.keys, hash)
		}
	}

	// 排序哈希值
	sort.Slice(b.ring.keys, func(i, j int) bool {
		return b.ring.keys[i] < b.ring.keys[j]
	})
}

// GetNode 获取哈希环中的节点
func (r *hashRing) GetNode(key string) (string, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(r.keys) == 0 {
		return "", false
	}

	hash := hashKey(key)

	// 二分查找第一个大于等于 hash 的位置
	idx := sort.Search(len(r.keys), func(i int) bool {
		return r.keys[i] >= hash
	})

	// 如果找不到，使用第一个节点（环形）
	if idx == len(r.keys) {
		idx = 0
	}

	return r.ring[r.keys[idx]], true
}

// hashKey 计算键的哈希值
func hashKey(key string) uint32 {
	h := md5.New()
	h.Write([]byte(key))
	hashBytes := h.Sum(nil)
	return binary.BigEndian.Uint32(hashBytes[:4])
}
