package node

import (
	"context"
	"errors"
)

// 错误定义
var (
	ErrNodeNameEmpty     = errors.New("node: node name cannot be empty")
	ErrNodeFuncNil       = errors.New("node: node function cannot be nil")
	ErrNodeNotExecutable = errors.New("node: node is not executable")
)

// NodeFunc 是节点函数的类型。
//
// NodeFunc 接收当前状态，执行处理，返回新状态。
//
// 类型参数：
//   - S: 状态类型
//
type NodeFunc[S any] func(ctx context.Context, state S) (S, error)

// Node 是节点的通用接口。
//
// Node 定义了节点的基本行为：
//   - 获取节点信息（名称、描述等）
//   - 执行节点逻辑
//   - 验证节点配置
//
// 类型参数：
//   - S: 状态类型
//
type Node[S any] interface {
	// GetName 返回节点名称
	GetName() string

	// GetDescription 返回节点描述
	GetDescription() string

	// GetTags 返回节点标签
	GetTags() []string

	// Invoke 执行节点，返回新状态
	//
	// 参数：
	//   - ctx: 上下文
	//   - state: 当前状态
	//
	// 返回：
	//   - S: 新状态
	//   - error: 执行错误
	//
	Invoke(ctx context.Context, state S) (S, error)

	// Validate 验证节点配置
	//
	// 返回：
	//   - error: 验证错误
	//
	Validate() error
}

// Metadata 是节点元数据。
//
// Metadata 包含节点的描述性信息，用于：
//   - 日志和调试
//   - 监控和追踪
//   - 文档生成
//
type Metadata struct {
	// Name 节点名称（必需）
	Name string

	// Description 节点描述（可选）
	Description string

	// Tags 节点标签（可选）
	Tags []string

	// Version 节点版本（可选）
	Version string

	// Extra 额外的元数据（可选）
	Extra map[string]any
}

// NewMetadata 创建节点元数据。
//
// 参数：
//   - name: 节点名称
//
// 返回：
//   - *Metadata: 元数据实例
//
func NewMetadata(name string) *Metadata {
	return &Metadata{
		Name:  name,
		Tags:  make([]string, 0),
		Extra: make(map[string]any),
	}
}

// WithDescription 设置描述。
func (m *Metadata) WithDescription(desc string) *Metadata {
	m.Description = desc
	return m
}

// WithTags 设置标签。
func (m *Metadata) WithTags(tags ...string) *Metadata {
	m.Tags = tags
	return m
}

// WithVersion 设置版本。
func (m *Metadata) WithVersion(version string) *Metadata {
	m.Version = version
	return m
}

// WithExtra 设置额外元数据。
func (m *Metadata) WithExtra(key string, value any) *Metadata {
	m.Extra[key] = value
	return m
}

// Clone 克隆元数据。
func (m *Metadata) Clone() *Metadata {
	clone := &Metadata{
		Name:        m.Name,
		Description: m.Description,
		Tags:        make([]string, len(m.Tags)),
		Version:     m.Version,
		Extra:       make(map[string]any),
	}

	copy(clone.Tags, m.Tags)
	for k, v := range m.Extra {
		clone.Extra[k] = v
	}

	return clone
}

// Validate 验证元数据。
func (m *Metadata) Validate() error {
	if m.Name == "" {
		return ErrNodeNameEmpty
	}
	return nil
}

// NodeOption 是节点选项函数。
type NodeOption func(*Metadata)

// WithDescription 返回设置描述的选项。
func WithDescription(desc string) NodeOption {
	return func(m *Metadata) {
		m.Description = desc
	}
}

// WithTags 返回设置标签的选项。
func WithTags(tags ...string) NodeOption {
	return func(m *Metadata) {
		m.Tags = tags
	}
}

// WithVersion 返回设置版本的选项。
func WithVersion(version string) NodeOption {
	return func(m *Metadata) {
		m.Version = version
	}
}

// WithExtra 返回设置额外元数据的选项。
func WithExtra(key string, value any) NodeOption {
	return func(m *Metadata) {
		m.Extra[key] = value
	}
}
