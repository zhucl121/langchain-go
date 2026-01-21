package node

import (
	"context"
	"errors"
	"time"
)

var (
	// ErrNodeNotFound 节点未找到
	ErrNodeNotFound = errors.New("node: node not found")

	// ErrNodeAlreadyExists 节点已存在
	ErrNodeAlreadyExists = errors.New("node: node already exists")

	// ErrInvalidNode 无效的节点
	ErrInvalidNode = errors.New("node: invalid node")

	// ErrNodeOffline 节点离线
	ErrNodeOffline = errors.New("node: node is offline")
)

// NodeManager 节点管理器接口
//
// NodeManager 负责集群中节点的注册、注销、查询和状态管理。
type NodeManager interface {
	// RegisterNode 注册节点到集群
	//
	// 如果节点已存在，返回 ErrNodeAlreadyExists。
	// 如果节点数据无效，返回 ErrInvalidNode。
	RegisterNode(ctx context.Context, node *Node) error

	// UnregisterNode 从集群注销节点
	//
	// 如果节点不存在，返回 ErrNodeNotFound。
	UnregisterNode(ctx context.Context, nodeID string) error

	// GetNode 获取指定节点信息
	//
	// 如果节点不存在，返回 ErrNodeNotFound。
	GetNode(ctx context.Context, nodeID string) (*Node, error)

	// ListNodes 列出所有节点
	//
	// 可以使用 NodeFilter 过滤节点。
	ListNodes(ctx context.Context, filter *NodeFilter) ([]*Node, error)

	// UpdateNode 更新节点信息
	//
	// 如果节点不存在，返回 ErrNodeNotFound。
	UpdateNode(ctx context.Context, node *Node) error

	// UpdateNodeStatus 更新节点状态
	//
	// 如果节点不存在，返回 ErrNodeNotFound。
	UpdateNodeStatus(ctx context.Context, nodeID string, status NodeStatus) error

	// UpdateNodeLoad 更新节点负载信息
	//
	// 如果节点不存在，返回 ErrNodeNotFound。
	UpdateNodeLoad(ctx context.Context, nodeID string, load Load) error

	// Heartbeat 发送心跳，更新节点的 LastSeen 时间
	//
	// 如果节点不存在，返回 ErrNodeNotFound。
	Heartbeat(ctx context.Context, nodeID string) error

	// Watch 监听节点变化
	//
	// 返回一个事件通道，当节点加入、离开或更新时会发送事件。
	// 调用者应该在不再需要时取消 context 以停止监听。
	Watch(ctx context.Context) (<-chan NodeEvent, error)

	// Close 关闭管理器，释放资源
	Close() error
}

// HeartbeatConfig 心跳配置
type HeartbeatConfig struct {
	// Interval 心跳间隔
	Interval time.Duration

	// Timeout 心跳超时时间（超过此时间未收到心跳视为节点离线）
	Timeout time.Duration

	// RetryCount 重试次数
	RetryCount int
}

// DefaultHeartbeatConfig 返回默认的心跳配置
func DefaultHeartbeatConfig() HeartbeatConfig {
	return HeartbeatConfig{
		Interval:   10 * time.Second,
		Timeout:    30 * time.Second,
		RetryCount: 3,
	}
}

// NodeStats 节点统计信息
type NodeStats struct {
	// TotalNodes 总节点数
	TotalNodes int

	// OnlineNodes 在线节点数
	OnlineNodes int

	// OfflineNodes 离线节点数
	OfflineNodes int

	// BusyNodes 繁忙节点数
	BusyNodes int

	// TotalCapacity 总容量
	TotalCapacity Capacity

	// TotalLoad 总负载
	TotalLoad Load

	// AverageLoadPercent 平均负载百分比
	AverageLoadPercent float64
}

// NodeManagerStats 获取节点管理器统计信息的接口
type NodeManagerStats interface {
	// GetStats 获取节点统计信息
	GetStats(ctx context.Context) (*NodeStats, error)
}
