package checkpoint

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

// 错误定义
var (
	ErrCheckpointNotFound = errors.New("checkpoint: checkpoint not found")
	ErrInvalidConfig      = errors.New("checkpoint: invalid config")
	ErrSerializeFailed    = errors.New("checkpoint: serialize failed")
	ErrDeserializeFailed  = errors.New("checkpoint: deserialize failed")
)

// CheckpointConfig 是检查点配置。
//
// CheckpointConfig 用于标识和定位检查点。
//
type CheckpointConfig struct {
	// ThreadID 线程标识（必需）
	ThreadID string

	// CheckpointNS 检查点命名空间（用于支持子图，默认为空字符串）
	// 命名空间格式: "subgraph.level1.level2"
	CheckpointNS string

	// CheckpointID 检查点标识（可选，用于加载特定检查点）
	CheckpointID string

	// Metadata 元数据
	Metadata map[string]any
}

// NewCheckpointConfig 创建检查点配置。
//
// 参数：
//   - threadID: 线程标识
//
// 返回：
//   - *CheckpointConfig: 配置实例
//
func NewCheckpointConfig(threadID string) *CheckpointConfig {
	return &CheckpointConfig{
		ThreadID: threadID,
		Metadata: make(map[string]any),
	}
}

// WithCheckpointID 设置检查点 ID。
func (c *CheckpointConfig) WithCheckpointID(id string) *CheckpointConfig {
	c.CheckpointID = id
	return c
}

// WithNamespace 设置检查点命名空间。
//
// 命名空间用于支持嵌套子图,格式: "subgraph.level1.level2"
//
func (c *CheckpointConfig) WithNamespace(ns string) *CheckpointConfig {
	c.CheckpointNS = ns
	return c
}

// WithMetadata 设置元数据。
func (c *CheckpointConfig) WithMetadata(key string, value any) *CheckpointConfig {
	c.Metadata[key] = value
	return c
}

// Validate 验证配置。
func (c *CheckpointConfig) Validate() error {
	if c.ThreadID == "" {
		return ErrInvalidConfig
	}
	return nil
}

// Checkpoint 是检查点数据结构。
//
// Checkpoint 表示图执行过程中的状态快照。
//
type Checkpoint[S any] struct {
	// ID 检查点唯一标识
	ID string

	// ThreadID 所属线程
	ThreadID string

	// CheckpointNS 检查点命名空间（用于支持子图）
	CheckpointNS string

	// ParentID 父检查点 ID（用于构建执行树）
	ParentID string

	// Type 检查点类型（用于反序列化识别）
	Type string

	// State 状态快照
	State S

	// Timestamp 创建时间
	Timestamp time.Time

	// Metadata 元数据
	Metadata map[string]any

	// Version 版本号
	Version int
}

// NewCheckpoint 创建检查点。
//
// 参数：
//   - id: 检查点 ID
//   - state: 状态
//   - config: 配置
//
// 返回：
//   - *Checkpoint[S]: 检查点实例
//
func NewCheckpoint[S any](id string, state S, config *CheckpointConfig) *Checkpoint[S] {
	cp := &Checkpoint[S]{
		ID:           id,
		ThreadID:     config.ThreadID,
		CheckpointNS: config.CheckpointNS,
		State:        state,
		Timestamp:    time.Now(),
		Metadata:     make(map[string]any),
		Version:      1,
	}

	// 复制元数据
	for k, v := range config.Metadata {
		cp.Metadata[k] = v
	}

	return cp
}

// GetState 获取状态。
func (c *Checkpoint[S]) GetState() S {
	return c.State
}

// GetID 获取 ID。
func (c *Checkpoint[S]) GetID() string {
	return c.ID
}

// GetThreadID 获取线程 ID。
func (c *Checkpoint[S]) GetThreadID() string {
	return c.ThreadID
}

// GetTimestamp 获取时间戳。
func (c *Checkpoint[S]) GetTimestamp() time.Time {
	return c.Timestamp
}

// GetCheckpointNS 获取命名空间。
func (c *Checkpoint[S]) GetCheckpointNS() string {
	return c.CheckpointNS
}

// GetType 获取类型。
func (c *Checkpoint[S]) GetType() string {
	return c.Type
}

// SetType 设置类型。
func (c *Checkpoint[S]) SetType(t string) {
	c.Type = t
}

// Clone 克隆检查点。
func (c *Checkpoint[S]) Clone() *Checkpoint[S] {
	clone := &Checkpoint[S]{
		ID:           c.ID,
		ThreadID:     c.ThreadID,
		CheckpointNS: c.CheckpointNS,
		ParentID:     c.ParentID,
		Type:         c.Type,
		State:        c.State,
		Timestamp:    c.Timestamp,
		Metadata:     make(map[string]any),
		Version:      c.Version,
	}

	for k, v := range c.Metadata {
		clone.Metadata[k] = v
	}

	return clone
}

// CheckpointSaver 是检查点保存器接口。
//
// CheckpointSaver 定义了检查点的持久化操作。
//
type CheckpointSaver[S any] interface {
	// Save 保存检查点
	//
	// 参数：
	//   - ctx: 上下文
	//   - checkpoint: 检查点
	//
	// 返回：
	//   - error: 保存错误
	Save(ctx context.Context, checkpoint *Checkpoint[S]) error

	// Load 加载检查点
	//
	// 参数：
	//   - ctx: 上下文
	//   - config: 配置
	//
	// 返回：
	//   - *Checkpoint[S]: 检查点
	//   - error: 加载错误
	Load(ctx context.Context, config *CheckpointConfig) (*Checkpoint[S], error)

	// List 列出检查点
	//
	// 参数：
	//   - ctx: 上下文
	//   - threadID: 线程 ID
	//
	// 返回：
	//   - []*Checkpoint[S]: 检查点列表
	//   - error: 列出错误
	List(ctx context.Context, threadID string) ([]*Checkpoint[S], error)

	// Delete 删除检查点
	//
	// 参数：
	//   - ctx: 上下文
	//   - config: 配置
	//
	// 返回：
	//   - error: 删除错误
	Delete(ctx context.Context, config *CheckpointConfig) error
}

// SerializableCheckpoint 是可序列化的检查点（用于存储）。
type SerializableCheckpoint struct {
	ID           string          `json:"id"`
	ThreadID     string          `json:"thread_id"`
	CheckpointNS string          `json:"checkpoint_ns"`
	ParentID     string          `json:"parent_id"`
	Type         string          `json:"type"`
	State        json.RawMessage `json:"state"`
	Timestamp    time.Time       `json:"timestamp"`
	Metadata     map[string]any  `json:"metadata"`
	Version      int             `json:"version"`
}

// ToSerializable 转换为可序列化格式。
func ToSerializable[S any](cp *Checkpoint[S]) (*SerializableCheckpoint, error) {
	stateData, err := json.Marshal(cp.State)
	if err != nil {
		return nil, err
	}

	return &SerializableCheckpoint{
		ID:           cp.ID,
		ThreadID:     cp.ThreadID,
		CheckpointNS: cp.CheckpointNS,
		ParentID:     cp.ParentID,
		Type:         cp.Type,
		State:        stateData,
		Timestamp:    cp.Timestamp,
		Metadata:     cp.Metadata,
		Version:      cp.Version,
	}, nil
}

// FromSerializable 从可序列化格式转换。
func FromSerializable[S any](scp *SerializableCheckpoint) (*Checkpoint[S], error) {
	var state S
	if err := json.Unmarshal(scp.State, &state); err != nil {
		return nil, err
	}

	return &Checkpoint[S]{
		ID:           scp.ID,
		ThreadID:     scp.ThreadID,
		CheckpointNS: scp.CheckpointNS,
		ParentID:     scp.ParentID,
		Type:         scp.Type,
		State:        state,
		Timestamp:    scp.Timestamp,
		Metadata:     scp.Metadata,
		Version:      scp.Version,
	}, nil
}

// CheckpointMetadata 是检查点元数据。
type CheckpointMetadata struct {
	Source      string         `json:"source"`       // 来源（例如："manual", "auto"）
	Step        int            `json:"step"`         // 执行步数
	NodeName    string         `json:"node_name"`    // 当前节点
	Description string         `json:"description"`  // 描述
	Extra       map[string]any `json:"extra"`        // 额外信息
}

// NewCheckpointMetadata 创建检查点元数据。
func NewCheckpointMetadata() *CheckpointMetadata {
	return &CheckpointMetadata{
		Extra: make(map[string]any),
	}
}

// WithSource 设置来源。
func (m *CheckpointMetadata) WithSource(source string) *CheckpointMetadata {
	m.Source = source
	return m
}

// WithStep 设置步数。
func (m *CheckpointMetadata) WithStep(step int) *CheckpointMetadata {
	m.Step = step
	return m
}

// WithNodeName 设置节点名称。
func (m *CheckpointMetadata) WithNodeName(nodeName string) *CheckpointMetadata {
	m.NodeName = nodeName
	return m
}

// WithDescription 设置描述。
func (m *CheckpointMetadata) WithDescription(desc string) *CheckpointMetadata {
	m.Description = desc
	return m
}

// ToMap 转换为 map。
func (m *CheckpointMetadata) ToMap() map[string]any {
	result := make(map[string]any)
	result["source"] = m.Source
	result["step"] = m.Step
	result["node_name"] = m.NodeName
	result["description"] = m.Description
	for k, v := range m.Extra {
		result[k] = v
	}
	return result
}
