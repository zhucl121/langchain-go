// Package graphdb 提供统一的图数据库抽象接口
//
// 支持多种图数据库实现：
// - Neo4j: 最成熟的图数据库
// - NebulaGraph: 高性能分布式图数据库
//
// 示例用法：
//
//	// 创建 Neo4j 实例
//	db, err := neo4j.NewNeo4jDriver(neo4j.Config{
//	    URI:      "bolt://localhost:7687",
//	    Username: "neo4j",
//	    Password: "password",
//	})
//
//	// 添加节点
//	node := &graphdb.Node{
//	    ID:   "entity-1",
//	    Type: "person",
//	    Label: "John Doe",
//	    Properties: map[string]interface{}{
//	        "age": 30,
//	        "city": "Beijing",
//	    },
//	}
//	err = db.AddNode(ctx, node)
//
//	// 图遍历
//	result, err := db.Traverse(ctx, "entity-1", graphdb.TraverseOptions{
//	    MaxDepth:  3,
//	    Direction: graphdb.DirectionBoth,
//	})
package graphdb

import "context"

// GraphDB 图数据库统一接口
//
// 所有图数据库实现（Neo4j、NebulaGraph 等）都应该实现此接口。
type GraphDB interface {
	// ========== 节点操作 ==========

	// AddNode 添加节点
	// 如果节点已存在，根据实现可能会更新或报错
	AddNode(ctx context.Context, node *Node) error

	// GetNode 获取节点
	// 返回 ErrNodeNotFound 如果节点不存在
	GetNode(ctx context.Context, id string) (*Node, error)

	// UpdateNode 更新节点
	// 只更新提供的字段，不会删除未提供的字段
	UpdateNode(ctx context.Context, node *Node) error

	// DeleteNode 删除节点
	// 如果节点不存在，不报错
	DeleteNode(ctx context.Context, id string) error

	// BatchAddNodes 批量添加节点
	// 使用事务保证原子性
	BatchAddNodes(ctx context.Context, nodes []*Node) error

	// ========== 边操作 ==========

	// AddEdge 添加边
	// 如果边已存在，根据实现可能会更新或报错
	AddEdge(ctx context.Context, edge *Edge) error

	// GetEdge 获取边
	// 返回 ErrEdgeNotFound 如果边不存在
	GetEdge(ctx context.Context, id string) (*Edge, error)

	// DeleteEdge 删除边
	// 如果边不存在，不报错
	DeleteEdge(ctx context.Context, id string) error

	// BatchAddEdges 批量添加边
	// 使用事务保证原子性
	BatchAddEdges(ctx context.Context, edges []*Edge) error

	// ========== 查询操作 ==========

	// FindNodes 查找节点
	// 支持按类型、属性等条件过滤
	FindNodes(ctx context.Context, filter NodeFilter) ([]*Node, error)

	// FindEdges 查找边
	// 支持按类型、源节点、目标节点等条件过滤
	FindEdges(ctx context.Context, filter EdgeFilter) ([]*Edge, error)

	// ========== 图遍历 ==========

	// Traverse 图遍历
	// 从起始节点开始，根据选项进行深度或广度优先遍历
	Traverse(ctx context.Context, startID string, opts TraverseOptions) (*TraverseResult, error)

	// ShortestPath 最短路径
	// 计算两个节点之间的最短路径
	ShortestPath(ctx context.Context, startID, endID string, opts PathOptions) (*Path, error)

	// ========== 连接管理 ==========

	// Connect 连接到数据库
	// 建立连接并验证可用性
	Connect(ctx context.Context) error

	// Close 关闭连接
	// 释放所有资源
	Close() error

	// Ping 健康检查
	// 验证连接是否正常
	Ping(ctx context.Context) error
}

// Node 图节点
type Node struct {
	// ID 节点唯一标识
	ID string `json:"id"`

	// Type 节点类型
	// 例如: "entity", "concept", "document", "person", "organization"
	Type string `json:"type"`

	// Label 节点标签（显示名称）
	Label string `json:"label"`

	// Properties 节点属性
	// 存储节点的元数据
	Properties map[string]interface{} `json:"properties"`

	// Embedding 节点向量（可选）
	// 用于向量相似度计算
	Embedding []float32 `json:"embedding,omitempty"`

	// Metadata 元数据（可选）
	// 存储额外信息，不参与查询
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// Edge 图边
type Edge struct {
	// ID 边唯一标识
	ID string `json:"id"`

	// Source 源节点 ID
	Source string `json:"source"`

	// Target 目标节点 ID
	Target string `json:"target"`

	// Type 边类型
	// 例如: "relates_to", "part_of", "caused_by", "works_for"
	Type string `json:"type"`

	// Label 边标签（显示名称）
	Label string `json:"label"`

	// Properties 边属性
	// 存储边的元数据
	Properties map[string]interface{} `json:"properties"`

	// Weight 边权重（可选）
	// 用于路径计算
	Weight float64 `json:"weight,omitempty"`

	// Directed 是否有向边（默认 true）
	Directed bool `json:"directed"`
}

// TraverseResult 遍历结果
type TraverseResult struct {
	// Nodes 遍历到的节点列表
	Nodes []*Node `json:"nodes"`

	// Edges 遍历到的边列表
	Edges []*Edge `json:"edges"`

	// Paths 路径列表（如果 IncludePath 为 true）
	Paths []*Path `json:"paths,omitempty"`
}

// Path 路径
type Path struct {
	// Nodes 路径上的节点列表（有序）
	Nodes []*Node `json:"nodes"`

	// Edges 路径上的边列表（有序）
	Edges []*Edge `json:"edges"`

	// Cost 路径总成本
	// 通常是边权重的总和
	Cost float64 `json:"cost"`

	// Length 路径长度（边的数量）
	Length int `json:"length"`
}

// NodeFilter 节点过滤器
type NodeFilter struct {
	// Types 节点类型列表（OR 关系）
	Types []string `json:"types,omitempty"`

	// Properties 属性过滤（AND 关系）
	// key: 属性名, value: 属性值
	Properties map[string]interface{} `json:"properties,omitempty"`

	// Labels 标签列表（OR 关系）
	Labels []string `json:"labels,omitempty"`

	// Limit 返回结果数量限制
	Limit int `json:"limit,omitempty"`

	// Offset 结果偏移量
	Offset int `json:"offset,omitempty"`
}

// EdgeFilter 边过滤器
type EdgeFilter struct {
	// Types 边类型列表（OR 关系）
	Types []string `json:"types,omitempty"`

	// SourceIDs 源节点 ID 列表（OR 关系）
	SourceIDs []string `json:"source_ids,omitempty"`

	// TargetIDs 目标节点 ID 列表（OR 关系）
	TargetIDs []string `json:"target_ids,omitempty"`

	// Properties 属性过滤（AND 关系）
	Properties map[string]interface{} `json:"properties,omitempty"`

	// Limit 返回结果数量限制
	Limit int `json:"limit,omitempty"`

	// Offset 结果偏移量
	Offset int `json:"offset,omitempty"`
}

// TraverseOptions 遍历选项
type TraverseOptions struct {
	// MaxDepth 最大遍历深度
	// 0 表示仅起始节点，1 表示直接邻居，以此类推
	MaxDepth int `json:"max_depth"`

	// Direction 遍历方向
	Direction Direction `json:"direction"`

	// EdgeTypes 限制边类型
	// 为空表示所有类型
	EdgeTypes []string `json:"edge_types,omitempty"`

	// NodeTypes 限制节点类型
	// 为空表示所有类型
	NodeTypes []string `json:"node_types,omitempty"`

	// Limit 返回结果数量限制
	Limit int `json:"limit,omitempty"`

	// IncludePath 是否返回路径信息
	IncludePath bool `json:"include_path"`

	// Strategy 遍历策略
	Strategy TraverseStrategy `json:"strategy"`
}

// Direction 遍历方向
type Direction string

const (
	// DirectionOutbound 仅出边
	DirectionOutbound Direction = "outbound"

	// DirectionInbound 仅入边
	DirectionInbound Direction = "inbound"

	// DirectionBoth 双向
	DirectionBoth Direction = "both"
)

// TraverseStrategy 遍历策略
type TraverseStrategy string

const (
	// StrategyBFS 广度优先搜索
	StrategyBFS TraverseStrategy = "bfs"

	// StrategyDFS 深度优先搜索
	StrategyDFS TraverseStrategy = "dfs"
)

// PathOptions 路径选项
type PathOptions struct {
	// MaxDepth 最大搜索深度
	MaxDepth int `json:"max_depth"`

	// EdgeTypes 限制边类型
	EdgeTypes []string `json:"edge_types,omitempty"`

	// Algorithm 路径算法
	Algorithm PathAlgorithm `json:"algorithm"`

	// Limit 返回路径数量限制
	// 1 表示只返回最短路径
	Limit int `json:"limit,omitempty"`
}

// PathAlgorithm 路径算法
type PathAlgorithm string

const (
	// AlgorithmDijkstra Dijkstra 算法（考虑权重）
	AlgorithmDijkstra PathAlgorithm = "dijkstra"

	// AlgorithmBFS BFS 算法（不考虑权重）
	AlgorithmBFS PathAlgorithm = "bfs"
)
