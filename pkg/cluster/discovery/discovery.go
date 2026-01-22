package discovery

import (
	"context"
	"errors"

	"github.com/zhucl121/langchain-go/pkg/cluster/node"
)

var (
	// ErrDiscoveryNotAvailable 服务发现不可用
	ErrDiscoveryNotAvailable = errors.New("discovery: service discovery not available")

	// ErrRegistrationFailed 注册失败
	ErrRegistrationFailed = errors.New("discovery: registration failed")

	// ErrDeregistrationFailed 注销失败
	ErrDeregistrationFailed = errors.New("discovery: deregistration failed")
)

// Discovery 服务发现接口
//
// Discovery 提供节点的注册、发现和监听功能。
type Discovery interface {
	// RegisterNode 注册节点
	RegisterNode(ctx context.Context, n *node.Node) error

	// UnregisterNode 注销节点
	UnregisterNode(ctx context.Context, nodeID string) error

	// GetNode 获取节点信息
	GetNode(ctx context.Context, nodeID string) (*node.Node, error)

	// ListNodes 列出所有节点
	ListNodes(ctx context.Context, filter *node.NodeFilter) ([]*node.Node, error)

	// Watch 监听节点变化
	//
	// 返回的通道会接收节点的加入、离开和更新事件。
	// 当 context 被取消时，通道会被关闭。
	Watch(ctx context.Context) (<-chan node.NodeEvent, error)

	// Heartbeat 发送心跳
	//
	// 用于保持节点的在线状态。
	Heartbeat(ctx context.Context, nodeID string) error

	// Close 关闭服务发现客户端
	Close() error
}

// Config 服务发现配置
type Config struct {
	// Backend 后端类型（consul, etcd）
	Backend string

	// Address 服务发现地址
	Address string

	// ServiceName 服务名称
	ServiceName string

	// Datacenter 数据中心（Consul）
	Datacenter string

	// Namespace 命名空间（Etcd）
	Namespace string

	// Username 用户名（可选）
	Username string

	// Password 密码（可选）
	Password string

	// TLS TLS 配置（可选）
	TLS *TLSConfig
}

// TLSConfig TLS 配置
type TLSConfig struct {
	// Enabled 是否启用 TLS
	Enabled bool

	// CertFile 证书文件路径
	CertFile string

	// KeyFile 密钥文件路径
	KeyFile string

	// CAFile CA 证书文件路径
	CAFile string

	// InsecureSkipVerify 是否跳过证书验证
	InsecureSkipVerify bool
}

// NewDiscovery 创建服务发现实例
func NewDiscovery(config Config) (Discovery, error) {
	switch config.Backend {
	case "consul":
		return NewConsulDiscovery(ConsulConfig{
			Address:     config.Address,
			ServiceName: config.ServiceName,
			Datacenter:  config.Datacenter,
		})
	case "etcd":
		// TODO: 实现 Etcd 服务发现
		return nil, errors.New("discovery: etcd backend not implemented yet")
	default:
		return nil, errors.New("discovery: unsupported backend: " + config.Backend)
	}
}
