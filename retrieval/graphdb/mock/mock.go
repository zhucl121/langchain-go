// Package mock 提供图数据库的内存 Mock 实现，用于测试。
package mock

import (
	"context"
	"fmt"
	"sync"

	"github.com/zhucl121/langchain-go/retrieval/graphdb"
)

// MockGraphDB 内存图数据库实现
type MockGraphDB struct {
	nodes     map[string]*graphdb.Node
	edges     map[string]*graphdb.Edge
	mu        sync.RWMutex
	connected bool
}

// NewMockGraphDB 创建 Mock 图数据库
func NewMockGraphDB() *MockGraphDB {
	return &MockGraphDB{
		nodes:     make(map[string]*graphdb.Node),
		edges:     make(map[string]*graphdb.Edge),
		connected: false,
	}
}

// Connect 连接数据库
func (m *MockGraphDB) Connect(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connected = true
	return nil
}

// Close 关闭连接
func (m *MockGraphDB) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connected = false
	return nil
}

// Ping 健康检查
func (m *MockGraphDB) Ping(ctx context.Context) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if !m.connected {
		return graphdb.ErrNotConnected
	}
	return nil
}

// AddNode 添加节点
func (m *MockGraphDB) AddNode(ctx context.Context, node *graphdb.Node) error {
	if node == nil || node.ID == "" {
		return graphdb.ErrInvalidNode
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.connected {
		return graphdb.ErrNotConnected
	}

	if _, exists := m.nodes[node.ID]; exists {
		return graphdb.ErrNodeExists
	}

	// 深拷贝节点
	m.nodes[node.ID] = m.copyNode(node)
	return nil
}

// GetNode 获取节点
func (m *MockGraphDB) GetNode(ctx context.Context, id string) (*graphdb.Node, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.connected {
		return nil, graphdb.ErrNotConnected
	}

	node, exists := m.nodes[id]
	if !exists {
		return nil, graphdb.ErrNodeNotFound
	}

	return m.copyNode(node), nil
}

// UpdateNode 更新节点
func (m *MockGraphDB) UpdateNode(ctx context.Context, node *graphdb.Node) error {
	if node == nil || node.ID == "" {
		return graphdb.ErrInvalidNode
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.connected {
		return graphdb.ErrNotConnected
	}

	existing, exists := m.nodes[node.ID]
	if !exists {
		return graphdb.ErrNodeNotFound
	}

	// 更新字段
	if node.Type != "" {
		existing.Type = node.Type
	}
	if node.Label != "" {
		existing.Label = node.Label
	}
	if node.Properties != nil {
		for k, v := range node.Properties {
			existing.Properties[k] = v
		}
	}
	if node.Embedding != nil {
		existing.Embedding = make([]float32, len(node.Embedding))
		copy(existing.Embedding, node.Embedding)
	}

	return nil
}

// DeleteNode 删除节点
func (m *MockGraphDB) DeleteNode(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.connected {
		return graphdb.ErrNotConnected
	}

	delete(m.nodes, id)
	return nil
}

// BatchAddNodes 批量添加节点
func (m *MockGraphDB) BatchAddNodes(ctx context.Context, nodes []*graphdb.Node) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.connected {
		return graphdb.ErrNotConnected
	}

	for _, node := range nodes {
		if node == nil || node.ID == "" {
			return graphdb.ErrInvalidNode
		}
		if _, exists := m.nodes[node.ID]; exists {
			return graphdb.ErrNodeExists
		}
		m.nodes[node.ID] = m.copyNode(node)
	}

	return nil
}

// AddEdge 添加边
func (m *MockGraphDB) AddEdge(ctx context.Context, edge *graphdb.Edge) error {
	if edge == nil || edge.ID == "" {
		return graphdb.ErrInvalidEdge
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.connected {
		return graphdb.ErrNotConnected
	}

	// 验证源和目标节点存在
	if _, exists := m.nodes[edge.Source]; !exists {
		return fmt.Errorf("source node %s not found", edge.Source)
	}
	if _, exists := m.nodes[edge.Target]; !exists {
		return fmt.Errorf("target node %s not found", edge.Target)
	}

	if _, exists := m.edges[edge.ID]; exists {
		return graphdb.ErrEdgeExists
	}

	m.edges[edge.ID] = m.copyEdge(edge)
	return nil
}

// GetEdge 获取边
func (m *MockGraphDB) GetEdge(ctx context.Context, id string) (*graphdb.Edge, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.connected {
		return nil, graphdb.ErrNotConnected
	}

	edge, exists := m.edges[id]
	if !exists {
		return nil, graphdb.ErrEdgeNotFound
	}

	return m.copyEdge(edge), nil
}

// DeleteEdge 删除边
func (m *MockGraphDB) DeleteEdge(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.connected {
		return graphdb.ErrNotConnected
	}

	delete(m.edges, id)
	return nil
}

// BatchAddEdges 批量添加边
func (m *MockGraphDB) BatchAddEdges(ctx context.Context, edges []*graphdb.Edge) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.connected {
		return graphdb.ErrNotConnected
	}

	for _, edge := range edges {
		if edge == nil || edge.ID == "" {
			return graphdb.ErrInvalidEdge
		}
		if _, exists := m.edges[edge.ID]; exists {
			return graphdb.ErrEdgeExists
		}
		m.edges[edge.ID] = m.copyEdge(edge)
	}

	return nil
}

// FindNodes 查找节点
func (m *MockGraphDB) FindNodes(ctx context.Context, filter graphdb.NodeFilter) ([]*graphdb.Node, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.connected {
		return nil, graphdb.ErrNotConnected
	}

	var result []*graphdb.Node
	for _, node := range m.nodes {
		if m.matchesNodeFilter(node, filter) {
			result = append(result, m.copyNode(node))
		}
	}

	// 应用 Limit 和 Offset
	if filter.Offset > 0 && filter.Offset < len(result) {
		result = result[filter.Offset:]
	}
	if filter.Limit > 0 && filter.Limit < len(result) {
		result = result[:filter.Limit]
	}

	return result, nil
}

// FindEdges 查找边
func (m *MockGraphDB) FindEdges(ctx context.Context, filter graphdb.EdgeFilter) ([]*graphdb.Edge, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.connected {
		return nil, graphdb.ErrNotConnected
	}

	var result []*graphdb.Edge
	for _, edge := range m.edges {
		if m.matchesEdgeFilter(edge, filter) {
			result = append(result, m.copyEdge(edge))
		}
	}

	// 应用 Limit 和 Offset
	if filter.Offset > 0 && filter.Offset < len(result) {
		result = result[filter.Offset:]
	}
	if filter.Limit > 0 && filter.Limit < len(result) {
		result = result[:filter.Limit]
	}

	return result, nil
}

// Traverse 图遍历（简化实现）
func (m *MockGraphDB) Traverse(ctx context.Context, startID string, opts graphdb.TraverseOptions) (*graphdb.TraverseResult, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.connected {
		return nil, graphdb.ErrNotConnected
	}

	if _, exists := m.nodes[startID]; !exists {
		return nil, graphdb.ErrNodeNotFound
	}

	visited := make(map[string]bool)
	result := &graphdb.TraverseResult{
		Nodes: []*graphdb.Node{},
		Edges: []*graphdb.Edge{},
	}

	// 使用 BFS 遍历
	queue := []string{startID}
	depths := map[string]int{startID: 0}

	for len(queue) > 0 {
		currentID := queue[0]
		queue = queue[1:]

		if visited[currentID] {
			continue
		}
		visited[currentID] = true

		currentNode := m.nodes[currentID]
		result.Nodes = append(result.Nodes, m.copyNode(currentNode))

		currentDepth := depths[currentID]
		if currentDepth >= opts.MaxDepth {
			continue
		}

		// 查找相邻边
		for _, edge := range m.edges {
			var nextID string
			var shouldAdd bool

			if edge.Source == currentID && (opts.Direction == graphdb.DirectionOutbound || opts.Direction == graphdb.DirectionBoth) {
				nextID = edge.Target
				shouldAdd = true
			} else if edge.Target == currentID && (opts.Direction == graphdb.DirectionInbound || opts.Direction == graphdb.DirectionBoth) {
				nextID = edge.Source
				shouldAdd = true
			}

			if shouldAdd && !visited[nextID] {
				result.Edges = append(result.Edges, m.copyEdge(edge))
				queue = append(queue, nextID)
				depths[nextID] = currentDepth + 1
			}
		}

		// 应用 Limit
		if opts.Limit > 0 && len(result.Nodes) >= opts.Limit {
			break
		}
	}

	return result, nil
}

// ShortestPath 最短路径（简化实现）
func (m *MockGraphDB) ShortestPath(ctx context.Context, startID, endID string, opts graphdb.PathOptions) (*graphdb.Path, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.connected {
		return nil, graphdb.ErrNotConnected
	}

	if _, exists := m.nodes[startID]; !exists {
		return nil, graphdb.ErrNodeNotFound
	}
	if _, exists := m.nodes[endID]; !exists {
		return nil, graphdb.ErrNodeNotFound
	}

	// 使用 BFS 查找最短路径
	queue := []string{startID}
	visited := make(map[string]bool)
	parent := make(map[string]string)
	edgeMap := make(map[string]*graphdb.Edge)

	visited[startID] = true

	for len(queue) > 0 {
		currentID := queue[0]
		queue = queue[1:]

		if currentID == endID {
			// 找到目标，构建路径
			return m.buildPath(startID, endID, parent, edgeMap), nil
		}

		// 查找邻居
		for _, edge := range m.edges {
			var nextID string
			if edge.Source == currentID {
				nextID = edge.Target
			} else if edge.Target == currentID {
				nextID = edge.Source
			} else {
				continue
			}

			if !visited[nextID] {
				visited[nextID] = true
				parent[nextID] = currentID
				edgeMap[nextID] = edge
				queue = append(queue, nextID)
			}
		}
	}

	return nil, graphdb.ErrNoPathFound
}

// 辅助方法

func (m *MockGraphDB) copyNode(node *graphdb.Node) *graphdb.Node {
	if node == nil {
		return nil
	}
	nodeCopy := &graphdb.Node{
		ID:         node.ID,
		Type:       node.Type,
		Label:      node.Label,
		Properties: make(map[string]interface{}),
		Metadata:   make(map[string]interface{}),
	}
	for k, v := range node.Properties {
		nodeCopy.Properties[k] = v
	}
	for k, v := range node.Metadata {
		nodeCopy.Metadata[k] = v
	}
	if node.Embedding != nil {
		nodeCopy.Embedding = make([]float32, len(node.Embedding))
		copy(nodeCopy.Embedding, node.Embedding)
	}
	return nodeCopy
}

func (m *MockGraphDB) copyEdge(edge *graphdb.Edge) *graphdb.Edge {
	if edge == nil {
		return nil
	}
	edgeCopy := &graphdb.Edge{
		ID:         edge.ID,
		Source:     edge.Source,
		Target:     edge.Target,
		Type:       edge.Type,
		Label:      edge.Label,
		Properties: make(map[string]interface{}),
		Weight:     edge.Weight,
		Directed:   edge.Directed,
	}
	for k, v := range edge.Properties {
		edgeCopy.Properties[k] = v
	}
	return edgeCopy
}

func (m *MockGraphDB) matchesNodeFilter(node *graphdb.Node, filter graphdb.NodeFilter) bool {
	// 检查类型
	if len(filter.Types) > 0 {
		matched := false
		for _, t := range filter.Types {
			if node.Type == t {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// 检查属性
	for k, v := range filter.Properties {
		if node.Properties[k] != v {
			return false
		}
	}

	// 检查标签
	if len(filter.Labels) > 0 {
		matched := false
		for _, l := range filter.Labels {
			if node.Label == l {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	return true
}

func (m *MockGraphDB) matchesEdgeFilter(edge *graphdb.Edge, filter graphdb.EdgeFilter) bool {
	// 检查类型
	if len(filter.Types) > 0 {
		matched := false
		for _, t := range filter.Types {
			if edge.Type == t {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// 检查源节点
	if len(filter.SourceIDs) > 0 {
		matched := false
		for _, id := range filter.SourceIDs {
			if edge.Source == id {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// 检查目标节点
	if len(filter.TargetIDs) > 0 {
		matched := false
		for _, id := range filter.TargetIDs {
			if edge.Target == id {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// 检查属性
	for k, v := range filter.Properties {
		if edge.Properties[k] != v {
			return false
		}
	}

	return true
}

func (m *MockGraphDB) buildPath(startID, endID string, parent map[string]string, edgeMap map[string]*graphdb.Edge) *graphdb.Path {
	path := &graphdb.Path{
		Nodes: []*graphdb.Node{},
		Edges: []*graphdb.Edge{},
	}

	// 从终点回溯到起点
	current := endID
	for current != startID {
		path.Nodes = append([]*graphdb.Node{m.copyNode(m.nodes[current])}, path.Nodes...)
		if edge, exists := edgeMap[current]; exists {
			path.Edges = append([]*graphdb.Edge{m.copyEdge(edge)}, path.Edges...)
		}
		current = parent[current]
	}

	// 添加起点
	path.Nodes = append([]*graphdb.Node{m.copyNode(m.nodes[startID])}, path.Nodes...)
	path.Length = len(path.Edges)

	// 计算总成本
	for _, edge := range path.Edges {
		path.Cost += edge.Weight
	}

	return path
}
