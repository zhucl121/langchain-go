package edge

import (
	"errors"
	"fmt"
)

// 错误定义
var (
	ErrEmptySourceNode    = errors.New("edge: source node cannot be empty")
	ErrEmptyTargetNode    = errors.New("edge: target node cannot be empty")
	ErrEmptyPathName      = errors.New("edge: path name cannot be empty")
	ErrPathNotFound       = errors.New("edge: path not found")
	ErrNoRouteMatched     = errors.New("edge: no route matched")
	ErrInvalidPathMapping = errors.New("edge: invalid path mapping")
)

// EdgeType 是边的类型。
type EdgeType string

const (
	// TypeNormal 普通边（静态连接）
	TypeNormal EdgeType = "normal"

	// TypeConditional 条件边（动态路由）
	TypeConditional EdgeType = "conditional"

	// TypeBranch 分支边（多路分支）
	TypeBranch EdgeType = "branch"
)

// Edge 是边的通用接口。
//
// Edge 定义了边的基本行为：
//   - 获取边信息（源节点、目标节点等）
//   - 验证边配置
//
type Edge interface {
	// GetSource 返回源节点名称
	GetSource() string

	// GetType 返回边的类型
	GetType() EdgeType

	// Validate 验证边配置
	Validate() error

	// Clone 克隆边
	Clone() Edge
}

// NormalEdge 是普通边。
//
// NormalEdge 表示从一个节点到另一个节点的静态连接。
// 这是最简单的边类型，执行流程总是从源节点到目标节点。
//
// 示例：
//
//	edge := NewNormalEdge("process", "output")
//	fmt.Println(edge.GetSource()) // "process"
//	fmt.Println(edge.GetTarget()) // "output"
//
type NormalEdge struct {
	source string
	target string
}

// NewNormalEdge 创建普通边。
//
// 参数：
//   - source: 源节点名称
//   - target: 目标节点名称
//
// 返回：
//   - *NormalEdge: 普通边实例
//
func NewNormalEdge(source, target string) *NormalEdge {
	return &NormalEdge{
		source: source,
		target: target,
	}
}

// GetSource 实现 Edge 接口。
func (e *NormalEdge) GetSource() string {
	return e.source
}

// GetTarget 返回目标节点名称。
func (e *NormalEdge) GetTarget() string {
	return e.target
}

// GetType 实现 Edge 接口。
func (e *NormalEdge) GetType() EdgeType {
	return TypeNormal
}

// Validate 实现 Edge 接口。
func (e *NormalEdge) Validate() error {
	if e.source == "" {
		return ErrEmptySourceNode
	}
	if e.target == "" {
		return ErrEmptyTargetNode
	}
	return nil
}

// Clone 实现 Edge 接口。
func (e *NormalEdge) Clone() Edge {
	return &NormalEdge{
		source: e.source,
		target: e.target,
	}
}

// String 返回边的字符串表示。
func (e *NormalEdge) String() string {
	return fmt.Sprintf("%s -> %s", e.source, e.target)
}

// Metadata 是边的元数据。
//
// Metadata 包含边的描述性信息，用于：
//   - 日志和调试
//   - 图可视化
//   - 文档生成
//
type Metadata struct {
	// Name 边的名称（可选）
	Name string

	// Description 边的描述（可选）
	Description string

	// Tags 边的标签（可选）
	Tags []string

	// Weight 边的权重（可选，用于优化）
	Weight float64

	// Extra 额外的元数据（可选）
	Extra map[string]any
}

// NewMetadata 创建边元数据。
//
// 返回：
//   - *Metadata: 元数据实例
//
func NewMetadata() *Metadata {
	return &Metadata{
		Tags:  make([]string, 0),
		Extra: make(map[string]any),
	}
}

// WithName 设置名称。
func (m *Metadata) WithName(name string) *Metadata {
	m.Name = name
	return m
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

// WithWeight 设置权重。
func (m *Metadata) WithWeight(weight float64) *Metadata {
	m.Weight = weight
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
		Weight:      m.Weight,
		Extra:       make(map[string]any),
	}

	copy(clone.Tags, m.Tags)
	for k, v := range m.Extra {
		clone.Extra[k] = v
	}

	return clone
}
