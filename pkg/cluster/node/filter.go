package node

// NodeFilter 用于过滤节点
type NodeFilter struct {
	// Status 过滤状态（为空表示不过滤）
	Status []NodeStatus

	// Roles 过滤角色（为空表示不过滤）
	Roles []NodeRole

	// Region 过滤区域
	Region string

	// Zone 过滤可用区
	Zone string

	// MinCapacity 最小容量要求
	MinCapacity *Capacity

	// MaxLoad 最大负载限制
	MaxLoad *Load

	// Tags 标签过滤
	Tags map[string]string

	// HealthyOnly 只返回健康节点
	HealthyOnly bool
}

// Match 检查节点是否匹配过滤条件
func (f *NodeFilter) Match(n *Node) bool {
	// 检查状态
	if len(f.Status) > 0 {
		matched := false
		for _, s := range f.Status {
			if n.Status == s {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// 检查角色
	if len(f.Roles) > 0 {
		matched := false
		for _, r := range f.Roles {
			if n.HasRole(r) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// 检查区域
	if f.Region != "" && n.Region != f.Region {
		return false
	}

	// 检查可用区
	if f.Zone != "" && n.Zone != f.Zone {
		return false
	}

	// 检查最小容量
	if f.MinCapacity != nil {
		if n.Capacity.MaxConnections < f.MinCapacity.MaxConnections {
			return false
		}
		if n.Capacity.MaxQPS < f.MinCapacity.MaxQPS {
			return false
		}
		if n.Capacity.MaxMemoryMB < f.MinCapacity.MaxMemoryMB {
			return false
		}
	}

	// 检查最大负载
	if f.MaxLoad != nil {
		if n.Load.CurrentConnections > f.MaxLoad.CurrentConnections {
			return false
		}
		if n.Load.CPUUsagePercent > f.MaxLoad.CPUUsagePercent {
			return false
		}
	}

	// 检查标签
	if len(f.Tags) > 0 {
		for k, v := range f.Tags {
			if n.Metadata[k] != v {
				return false
			}
		}
	}

	// 检查健康状态
	if f.HealthyOnly && !n.IsHealthy() {
		return false
	}

	return true
}

// MatchAny 检查节点是否匹配任意一个过滤条件
func (f *NodeFilter) MatchAny(nodes []*Node) []*Node {
	var matched []*Node
	for _, n := range nodes {
		if f.Match(n) {
			matched = append(matched, n)
		}
	}
	return matched
}

// NewNodeFilter 创建默认的节点过滤器
func NewNodeFilter() *NodeFilter {
	return &NodeFilter{
		Tags: make(map[string]string),
	}
}

// WithStatus 添加状态过滤
func (f *NodeFilter) WithStatus(status ...NodeStatus) *NodeFilter {
	f.Status = status
	return f
}

// WithRoles 添加角色过滤
func (f *NodeFilter) WithRoles(roles ...NodeRole) *NodeFilter {
	f.Roles = roles
	return f
}

// WithRegion 添加区域过滤
func (f *NodeFilter) WithRegion(region string) *NodeFilter {
	f.Region = region
	return f
}

// WithZone 添加可用区过滤
func (f *NodeFilter) WithZone(zone string) *NodeFilter {
	f.Zone = zone
	return f
}

// WithTag 添加标签过滤
func (f *NodeFilter) WithTag(key, value string) *NodeFilter {
	if f.Tags == nil {
		f.Tags = make(map[string]string)
	}
	f.Tags[key] = value
	return f
}

// WithHealthyOnly 只返回健康节点
func (f *NodeFilter) WithHealthyOnly() *NodeFilter {
	f.HealthyOnly = true
	return f
}
