package node

import (
	"encoding/json"
	"fmt"
	"time"
)

// Node 表示集群中的一个节点
type Node struct {
	// ID 节点唯一标识符
	ID string `json:"id"`

	// Name 节点名称
	Name string `json:"name"`

	// Address 节点地址（IP 或域名）
	Address string `json:"address"`

	// Port 节点端口
	Port int `json:"port"`

	// Status 节点状态
	Status NodeStatus `json:"status"`

	// Roles 节点角色列表
	Roles []NodeRole `json:"roles"`

	// Capacity 节点容量信息
	Capacity Capacity `json:"capacity"`

	// Load 节点当前负载
	Load Load `json:"load"`

	// Metadata 节点元数据
	Metadata map[string]string `json:"metadata"`

	// RegisterAt 注册时间
	RegisterAt time.Time `json:"register_at"`

	// LastSeen 最后活跃时间
	LastSeen time.Time `json:"last_seen"`

	// Version 节点版本
	Version string `json:"version,omitempty"`

	// Region 节点所在区域
	Region string `json:"region,omitempty"`

	// Zone 节点所在可用区
	Zone string `json:"zone,omitempty"`
}

// NodeStatus 表示节点状态
type NodeStatus string

const (
	// StatusOnline 节点在线，可以接受请求
	StatusOnline NodeStatus = "online"

	// StatusOffline 节点离线，不可用
	StatusOffline NodeStatus = "offline"

	// StatusBusy 节点繁忙，建议减少请求
	StatusBusy NodeStatus = "busy"

	// StatusDraining 节点正在排空，不接受新请求
	StatusDraining NodeStatus = "draining"

	// StatusMaintenance 节点维护中
	StatusMaintenance NodeStatus = "maintenance"
)

// IsAvailable 检查节点是否可用
func (s NodeStatus) IsAvailable() bool {
	return s == StatusOnline || s == StatusBusy
}

// NodeRole 表示节点角色
type NodeRole string

const (
	// RoleMaster 主节点，负责协调和管理
	RoleMaster NodeRole = "master"

	// RoleWorker 工作节点，处理实际请求
	RoleWorker NodeRole = "worker"

	// RoleCache 缓存节点，专门处理缓存
	RoleCache NodeRole = "cache"

	// RoleGateway 网关节点，处理外部请求
	RoleGateway NodeRole = "gateway"
)

// Capacity 表示节点容量信息
type Capacity struct {
	// MaxConnections 最大连接数
	MaxConnections int `json:"max_connections"`

	// MaxQPS 最大 QPS
	MaxQPS int `json:"max_qps"`

	// MaxMemoryMB 最大内存（MB）
	MaxMemoryMB int `json:"max_memory_mb"`

	// MaxGoroutines 最大协程数
	MaxGoroutines int `json:"max_goroutines"`

	// MaxDiskGB 最大磁盘空间（GB）
	MaxDiskGB int `json:"max_disk_gb,omitempty"`
}

// Load 表示节点当前负载
type Load struct {
	// CurrentConnections 当前连接数
	CurrentConnections int `json:"current_connections"`

	// CurrentQPS 当前 QPS
	CurrentQPS float64 `json:"current_qps"`

	// MemoryUsageMB 内存使用量（MB）
	MemoryUsageMB int `json:"memory_usage_mb"`

	// CPUUsagePercent CPU 使用率（0-100）
	CPUUsagePercent float64 `json:"cpu_usage_percent"`

	// GoroutineCount 当前协程数
	GoroutineCount int `json:"goroutine_count"`

	// DiskUsageGB 磁盘使用量（GB）
	DiskUsageGB int `json:"disk_usage_gb,omitempty"`

	// NetworkInMBPS 入站网络流量（MB/s）
	NetworkInMBPS float64 `json:"network_in_mbps,omitempty"`

	// NetworkOutMBPS 出站网络流量（MB/s）
	NetworkOutMBPS float64 `json:"network_out_mbps,omitempty"`
}

// NodeEvent 表示节点事件
type NodeEvent struct {
	// Type 事件类型
	Type EventType `json:"type"`

	// Node 相关节点
	Node *Node `json:"node"`

	// Timestamp 事件时间
	Timestamp time.Time `json:"timestamp"`

	// Message 事件消息
	Message string `json:"message,omitempty"`
}

// EventType 表示事件类型
type EventType string

const (
	// EventNodeJoined 节点加入集群
	EventNodeJoined EventType = "joined"

	// EventNodeLeft 节点离开集群
	EventNodeLeft EventType = "left"

	// EventNodeUpdated 节点信息更新
	EventNodeUpdated EventType = "updated"

	// EventNodeFailed 节点故障
	EventNodeFailed EventType = "failed"

	// EventNodeRecovered 节点恢复
	EventNodeRecovered EventType = "recovered"
)

// String 返回节点的字符串表示
func (n *Node) String() string {
	return fmt.Sprintf("Node{ID=%s, Name=%s, Address=%s:%d, Status=%s}",
		n.ID, n.Name, n.Address, n.Port, n.Status)
}

// GetEndpoint 返回节点的完整端点地址
func (n *Node) GetEndpoint() string {
	return fmt.Sprintf("%s:%d", n.Address, n.Port)
}

// GetURL 返回节点的 HTTP URL
func (n *Node) GetURL(scheme string) string {
	if scheme == "" {
		scheme = "http"
	}
	return fmt.Sprintf("%s://%s:%d", scheme, n.Address, n.Port)
}

// HasRole 检查节点是否具有指定角色
func (n *Node) HasRole(role NodeRole) bool {
	for _, r := range n.Roles {
		if r == role {
			return true
		}
	}
	return false
}

// IsHealthy 检查节点是否健康
func (n *Node) IsHealthy() bool {
	// 检查状态
	if !n.Status.IsAvailable() {
		return false
	}

	// 检查负载
	if n.Capacity.MaxConnections > 0 &&
		n.Load.CurrentConnections >= n.Capacity.MaxConnections {
		return false
	}

	if n.Capacity.MaxMemoryMB > 0 &&
		n.Load.MemoryUsageMB >= n.Capacity.MaxMemoryMB {
		return false
	}

	// 检查 CPU 使用率（>95% 视为不健康）
	if n.Load.CPUUsagePercent > 95.0 {
		return false
	}

	return true
}

// GetLoadPercent 获取负载百分比（0-100）
func (n *Node) GetLoadPercent() float64 {
	if n.Capacity.MaxConnections == 0 {
		return 0
	}
	return float64(n.Load.CurrentConnections) / float64(n.Capacity.MaxConnections) * 100.0
}

// Clone 创建节点的深拷贝
func (n *Node) Clone() *Node {
	clone := *n
	clone.Roles = make([]NodeRole, len(n.Roles))
	copy(clone.Roles, n.Roles)

	if n.Metadata != nil {
		clone.Metadata = make(map[string]string, len(n.Metadata))
		for k, v := range n.Metadata {
			clone.Metadata[k] = v
		}
	}

	return &clone
}

// MarshalJSON 自定义 JSON 序列化
func (n *Node) MarshalJSON() ([]byte, error) {
	type Alias Node
	return json.Marshal(&struct {
		*Alias
		RegisterAt string `json:"register_at"`
		LastSeen   string `json:"last_seen"`
	}{
		Alias:      (*Alias)(n),
		RegisterAt: n.RegisterAt.Format(time.RFC3339),
		LastSeen:   n.LastSeen.Format(time.RFC3339),
	})
}

// Validate 验证节点数据的有效性
func (n *Node) Validate() error {
	if n.ID == "" {
		return fmt.Errorf("node ID is required")
	}
	if n.Name == "" {
		return fmt.Errorf("node name is required")
	}
	if n.Address == "" {
		return fmt.Errorf("node address is required")
	}
	if n.Port <= 0 || n.Port > 65535 {
		return fmt.Errorf("invalid node port: %d", n.Port)
	}
	if len(n.Roles) == 0 {
		return fmt.Errorf("node must have at least one role")
	}
	return nil
}
