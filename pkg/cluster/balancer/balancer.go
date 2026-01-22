package balancer

import (
	"context"
	"errors"
	"time"

	"github.com/zhucl121/langchain-go/pkg/cluster/node"
)

var (
	// ErrNoAvailableNodes 没有可用节点
	ErrNoAvailableNodes = errors.New("balancer: no available nodes")

	// ErrNodeNotFound 节点未找到
	ErrNodeNotFound = errors.New("balancer: node not found")

	// ErrInvalidRequest 无效的请求
	ErrInvalidRequest = errors.New("balancer: invalid request")
)

// LoadBalancer 负载均衡器接口
//
// LoadBalancer 负责从可用节点中选择一个节点来处理请求。
type LoadBalancer interface {
	// SelectNode 选择一个节点处理请求
	//
	// 返回选中的节点，如果没有可用节点返回 ErrNoAvailableNodes。
	SelectNode(ctx context.Context, req *Request) (*node.Node, error)

	// UpdateNodes 更新节点列表
	//
	// 负载均衡器会根据新的节点列表调整选择策略。
	UpdateNodes(nodes []*node.Node)

	// RecordResult 记录请求结果
	//
	// 用于自适应负载均衡器收集节点性能数据。
	// success 表示请求是否成功，latency 表示请求延迟。
	RecordResult(nodeID string, success bool, latency time.Duration)

	// GetStats 获取统计信息（可选）
	GetStats() *Stats
}

// Request 请求信息
type Request struct {
	// ID 请求唯一标识符
	ID string

	// Type 请求类型
	Type RequestType

	// Size 请求大小（字节）
	Size int64

	// Metadata 请求元数据
	Metadata map[string]string

	// Priority 请求优先级（0-10，10 最高）
	Priority int

	// UserID 用户 ID（用于一致性哈希）
	UserID string
}

// RequestType 请求类型
type RequestType string

const (
	// RequestTypeLLM LLM 推理请求
	RequestTypeLLM RequestType = "llm"

	// RequestTypeRetrieval 检索请求
	RequestTypeRetrieval RequestType = "retrieval"

	// RequestTypeEmbedding 嵌入请求
	RequestTypeEmbedding RequestType = "embedding"

	// RequestTypeGeneric 通用请求
	RequestTypeGeneric RequestType = "generic"
)

// Stats 负载均衡器统计信息
type Stats struct {
	// TotalRequests 总请求数
	TotalRequests int64

	// SuccessRequests 成功请求数
	SuccessRequests int64

	// FailedRequests 失败请求数
	FailedRequests int64

	// AverageLatency 平均延迟
	AverageLatency time.Duration

	// NodeStats 每个节点的统计信息
	NodeStats map[string]*NodeStats
}

// NodeStats 节点统计信息
type NodeStats struct {
	// NodeID 节点 ID
	NodeID string

	// Requests 请求数
	Requests int64

	// SuccessRequests 成功请求数
	SuccessRequests int64

	// FailedRequests 失败请求数
	FailedRequests int64

	// AverageLatency 平均延迟
	AverageLatency time.Duration

	// LastUsed 最后使用时间
	LastUsed time.Time

	// CurrentConnections 当前连接数
	CurrentConnections int

	// Score 节点得分（用于自适应均衡）
	Score float64
}

// Strategy 负载均衡策略
type Strategy string

const (
	// StrategyRoundRobin 轮询策略
	StrategyRoundRobin Strategy = "round_robin"

	// StrategyLeastConnection 最少连接策略
	StrategyLeastConnection Strategy = "least_connection"

	// StrategyWeighted 加权策略
	StrategyWeighted Strategy = "weighted"

	// StrategyConsistentHash 一致性哈希策略
	StrategyConsistentHash Strategy = "consistent_hash"

	// StrategyAdaptive 自适应策略
	StrategyAdaptive Strategy = "adaptive"
)

// Config 负载均衡器配置
type Config struct {
	// Strategy 负载均衡策略
	Strategy Strategy

	// HealthyOnly 是否只选择健康节点
	HealthyOnly bool

	// EnableMetrics 是否启用指标收集
	EnableMetrics bool

	// MetricsWindowSize 指标窗口大小（用于自适应均衡）
	MetricsWindowSize int

	// ConsistentHashVirtualNodes 一致性哈希虚拟节点数
	ConsistentHashVirtualNodes int
}

// DefaultConfig 返回默认配置
func DefaultConfig() Config {
	return Config{
		Strategy:                   StrategyRoundRobin,
		HealthyOnly:                true,
		EnableMetrics:              true,
		MetricsWindowSize:          100,
		ConsistentHashVirtualNodes: 150,
	}
}

// NewLoadBalancer 创建负载均衡器
func NewLoadBalancer(config Config, nodes []*node.Node) (LoadBalancer, error) {
	switch config.Strategy {
	case StrategyRoundRobin:
		return NewRoundRobinBalancer(nodes), nil
	case StrategyLeastConnection:
		return NewLeastConnectionBalancer(nodes), nil
	case StrategyWeighted:
		return NewWeightedBalancer(nodes, nil), nil
	case StrategyConsistentHash:
		return NewConsistentHashBalancer(nodes, config.ConsistentHashVirtualNodes), nil
	case StrategyAdaptive:
		return NewAdaptiveBalancer(nodes, config.MetricsWindowSize), nil
	default:
		return nil, errors.New("balancer: unsupported strategy: " + string(config.Strategy))
	}
}

// filterHealthyNodes 过滤出健康的节点
func filterHealthyNodes(nodes []*node.Node) []*node.Node {
	healthy := make([]*node.Node, 0, len(nodes))
	for _, n := range nodes {
		if n.IsHealthy() && n.Status.IsAvailable() {
			healthy = append(healthy, n)
		}
	}
	return healthy
}
