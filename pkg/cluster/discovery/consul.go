package discovery

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/zhucl121/langchain-go/pkg/cluster/node"
)

// ConsulDiscovery Consul 服务发现实现
type ConsulDiscovery struct {
	client *api.Client
	config ConsulConfig
	mu     sync.RWMutex
	closed bool
}

// ConsulConfig Consul 配置
type ConsulConfig struct {
	// Address Consul 地址
	Address string

	// Datacenter 数据中心
	Datacenter string

	// ServiceName 服务名称
	ServiceName string

	// Tags 服务标签
	Tags []string

	// CheckTTL 健康检查 TTL
	CheckTTL time.Duration

	// CheckInterval 健康检查间隔
	CheckInterval time.Duration

	// DeregisterAfter 超过多久未心跳后自动注销
	DeregisterAfter time.Duration
}

// DefaultConsulConfig 返回默认的 Consul 配置
func DefaultConsulConfig() ConsulConfig {
	return ConsulConfig{
		Address:         "localhost:8500",
		ServiceName:     "langchain-go",
		Tags:            []string{},
		CheckTTL:        10 * time.Second,
		CheckInterval:   5 * time.Second,
		DeregisterAfter: 30 * time.Second,
	}
}

// NewConsulDiscovery 创建 Consul 服务发现实例
func NewConsulDiscovery(config ConsulConfig) (*ConsulDiscovery, error) {
	// 设置默认值
	if config.ServiceName == "" {
		config.ServiceName = "langchain-go"
	}
	if config.CheckTTL == 0 {
		config.CheckTTL = 10 * time.Second
	}
	if config.DeregisterAfter == 0 {
		config.DeregisterAfter = 30 * time.Second
	}

	// 创建 Consul 客户端
	clientConfig := api.DefaultConfig()
	clientConfig.Address = config.Address
	if config.Datacenter != "" {
		clientConfig.Datacenter = config.Datacenter
	}

	client, err := api.NewClient(clientConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create consul client: %w", err)
	}

	return &ConsulDiscovery{
		client: client,
		config: config,
	}, nil
}

// RegisterNode 注册节点
func (d *ConsulDiscovery) RegisterNode(ctx context.Context, n *node.Node) error {
	d.mu.RLock()
	if d.closed {
		d.mu.RUnlock()
		return ErrDiscoveryNotAvailable
	}
	d.mu.RUnlock()

	// 验证节点
	if err := n.Validate(); err != nil {
		return fmt.Errorf("%w: %v", ErrRegistrationFailed, err)
	}

	// 构建服务注册信息
	registration := &api.AgentServiceRegistration{
		ID:      n.ID,
		Name:    d.config.ServiceName,
		Address: n.Address,
		Port:    n.Port,
		Tags:    d.buildTags(n),
		Meta:    n.Metadata,
		Check: &api.AgentServiceCheck{
			TTL:                            d.config.CheckTTL.String(),
			DeregisterCriticalServiceAfter: d.config.DeregisterAfter.String(),
			Notes:                          "Node health check",
		},
	}

	// 注册服务
	if err := d.client.Agent().ServiceRegister(registration); err != nil {
		return fmt.Errorf("%w: %v", ErrRegistrationFailed, err)
	}

	// 立即发送一次健康检查，标记为健康
	checkID := "service:" + n.ID
	if err := d.client.Agent().UpdateTTL(checkID, "Node registered", api.HealthPassing); err != nil {
		// 注册成功但更新 TTL 失败，记录警告但不返回错误
		// log.Warnf("Failed to update TTL for node %s: %v", n.ID, err)
	}

	return nil
}

// UnregisterNode 注销节点
func (d *ConsulDiscovery) UnregisterNode(ctx context.Context, nodeID string) error {
	d.mu.RLock()
	if d.closed {
		d.mu.RUnlock()
		return ErrDiscoveryNotAvailable
	}
	d.mu.RUnlock()

	if err := d.client.Agent().ServiceDeregister(nodeID); err != nil {
		return fmt.Errorf("%w: %v", ErrDeregistrationFailed, err)
	}

	return nil
}

// GetNode 获取节点信息
func (d *ConsulDiscovery) GetNode(ctx context.Context, nodeID string) (*node.Node, error) {
	d.mu.RLock()
	if d.closed {
		d.mu.RUnlock()
		return nil, ErrDiscoveryNotAvailable
	}
	d.mu.RUnlock()

	services, _, err := d.client.Health().Service(d.config.ServiceName, "", false, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to query service: %w", err)
	}

	for _, service := range services {
		if service.Service.ID == nodeID {
			return d.serviceToNode(service), nil
		}
	}

	return nil, node.ErrNodeNotFound
}

// ListNodes 列出所有节点
func (d *ConsulDiscovery) ListNodes(ctx context.Context, filter *node.NodeFilter) ([]*node.Node, error) {
	d.mu.RLock()
	if d.closed {
		d.mu.RUnlock()
		return nil, ErrDiscoveryNotAvailable
	}
	d.mu.RUnlock()

	// 查询服务
	// passingOnly 设置为 false，获取所有节点（包括不健康的）
	services, _, err := d.client.Health().Service(d.config.ServiceName, "", false, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list services: %w", err)
	}

	nodes := make([]*node.Node, 0, len(services))
	for _, service := range services {
		n := d.serviceToNode(service)

		// 应用过滤器
		if filter != nil && !filter.Match(n) {
			continue
		}

		nodes = append(nodes, n)
	}

	return nodes, nil
}

// Watch 监听节点变化
func (d *ConsulDiscovery) Watch(ctx context.Context) (<-chan node.NodeEvent, error) {
	d.mu.RLock()
	if d.closed {
		d.mu.RUnlock()
		return nil, ErrDiscoveryNotAvailable
	}
	d.mu.RUnlock()

	eventCh := make(chan node.NodeEvent, 100)

	go func() {
		defer close(eventCh)

		var lastIndex uint64
		knownNodes := make(map[string]*node.Node)

		for {
			select {
			case <-ctx.Done():
				return
			default:
				// 长轮询查询服务变化
				services, meta, err := d.client.Health().Service(
					d.config.ServiceName,
					"",
					false,
					&api.QueryOptions{
						WaitIndex: lastIndex,
						WaitTime:  30 * time.Second,
					},
				)
				if err != nil {
					time.Sleep(5 * time.Second)
					continue
				}

				lastIndex = meta.LastIndex

				// 检测变化
				currentNodes := make(map[string]*node.Node)
				for _, service := range services {
					n := d.serviceToNode(service)
					currentNodes[n.ID] = n

					if old, exists := knownNodes[n.ID]; !exists {
						// 新节点加入
						select {
						case eventCh <- node.NodeEvent{
							Type:      node.EventNodeJoined,
							Node:      n,
							Timestamp: time.Now(),
							Message:   "Node joined cluster",
						}:
						case <-ctx.Done():
							return
						}
					} else if !nodesEqual(old, n) {
						// 节点信息更新
						select {
						case eventCh <- node.NodeEvent{
							Type:      node.EventNodeUpdated,
							Node:      n,
							Timestamp: time.Now(),
							Message:   "Node information updated",
						}:
						case <-ctx.Done():
							return
						}
					}
				}

				// 检测离开的节点
				for id, n := range knownNodes {
					if _, exists := currentNodes[id]; !exists {
						select {
						case eventCh <- node.NodeEvent{
							Type:      node.EventNodeLeft,
							Node:      n,
							Timestamp: time.Now(),
							Message:   "Node left cluster",
						}:
						case <-ctx.Done():
							return
						}
					}
				}

				knownNodes = currentNodes
			}
		}
	}()

	return eventCh, nil
}

// Heartbeat 发送心跳
func (d *ConsulDiscovery) Heartbeat(ctx context.Context, nodeID string) error {
	d.mu.RLock()
	if d.closed {
		d.mu.RUnlock()
		return ErrDiscoveryNotAvailable
	}
	d.mu.RUnlock()

	checkID := "service:" + nodeID
	if err := d.client.Agent().UpdateTTL(checkID, "Heartbeat", api.HealthPassing); err != nil {
		return fmt.Errorf("failed to send heartbeat: %w", err)
	}

	return nil
}

// Close 关闭服务发现客户端
func (d *ConsulDiscovery) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.closed {
		return nil
	}

	d.closed = true
	return nil
}

// buildTags 构建服务标签
func (d *ConsulDiscovery) buildTags(n *node.Node) []string {
	tags := make([]string, 0, len(d.config.Tags)+len(n.Roles)+3)

	// 添加配置的标签
	tags = append(tags, d.config.Tags...)

	// 添加角色标签
	for _, role := range n.Roles {
		tags = append(tags, "role:"+string(role))
	}

	// 添加状态标签
	tags = append(tags, "status:"+string(n.Status))

	// 添加区域和可用区标签
	if n.Region != "" {
		tags = append(tags, "region:"+n.Region)
	}
	if n.Zone != "" {
		tags = append(tags, "zone:"+n.Zone)
	}

	return tags
}

// serviceToNode 将 Consul 服务转换为节点
func (d *ConsulDiscovery) serviceToNode(service *api.ServiceEntry) *node.Node {
	n := &node.Node{
		ID:       service.Service.ID,
		Address:  service.Service.Address,
		Port:     service.Service.Port,
		Metadata: service.Service.Meta,
	}

	// 从标签解析信息
	for _, tag := range service.Service.Tags {
		if len(tag) > 5 && tag[:5] == "role:" {
			n.Roles = append(n.Roles, node.NodeRole(tag[5:]))
		} else if len(tag) > 7 && tag[:7] == "status:" {
			n.Status = node.NodeStatus(tag[7:])
		} else if len(tag) > 7 && tag[:7] == "region:" {
			n.Region = tag[7:]
		} else if len(tag) > 5 && tag[:5] == "zone:" {
			n.Zone = tag[5:]
		}
	}

	// 从元数据解析名称和版本
	if name, ok := n.Metadata["name"]; ok {
		n.Name = name
	} else {
		n.Name = n.ID
	}

	if version, ok := n.Metadata["version"]; ok {
		n.Version = version
	}

	// 解析容量信息
	if maxConn, ok := n.Metadata["max_connections"]; ok {
		if val, err := strconv.Atoi(maxConn); err == nil {
			n.Capacity.MaxConnections = val
		}
	}
	if maxQPS, ok := n.Metadata["max_qps"]; ok {
		if val, err := strconv.Atoi(maxQPS); err == nil {
			n.Capacity.MaxQPS = val
		}
	}
	if maxMem, ok := n.Metadata["max_memory_mb"]; ok {
		if val, err := strconv.Atoi(maxMem); err == nil {
			n.Capacity.MaxMemoryMB = val
		}
	}

	// 检查健康状态
	for _, check := range service.Checks {
		if check.Status == api.HealthPassing {
			// 健康
			if n.Status == "" {
				n.Status = node.StatusOnline
			}
		} else if check.Status == api.HealthCritical {
			// 不健康
			n.Status = node.StatusOffline
		}
	}

	// 设置默认状态
	if n.Status == "" {
		n.Status = node.StatusOnline
	}

	// 更新时间
	n.LastSeen = time.Now()

	return n
}

// nodesEqual 比较两个节点是否相等
func nodesEqual(a, b *node.Node) bool {
	if a.ID != b.ID || a.Status != b.Status {
		return false
	}
	if a.Address != b.Address || a.Port != b.Port {
		return false
	}
	if len(a.Roles) != len(b.Roles) {
		return false
	}
	for i := range a.Roles {
		if a.Roles[i] != b.Roles[i] {
			return false
		}
	}
	return true
}
