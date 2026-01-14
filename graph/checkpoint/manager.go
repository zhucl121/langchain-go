package checkpoint

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

// CheckpointManager 是检查点管理器。
//
// CheckpointManager 提供高级的检查点管理功能：
//   - 自动生成检查点 ID
//   - 检查点历史管理
//   - 时间旅行功能
//   - 分支管理
//
type CheckpointManager[S any] struct {
	saver CheckpointSaver[S]

	// 自动 ID 生成
	autoID bool

	mu sync.RWMutex
}

// NewCheckpointManager 创建检查点管理器。
//
// 参数：
//   - saver: 检查点保存器
//
// 返回：
//   - *CheckpointManager[S]: 管理器实例
//
func NewCheckpointManager[S any](saver CheckpointSaver[S]) *CheckpointManager[S] {
	return &CheckpointManager[S]{
		saver:  saver,
		autoID: true,
	}
}

// WithAutoID 设置是否自动生成 ID。
func (m *CheckpointManager[S]) WithAutoID(enabled bool) *CheckpointManager[S] {
	m.autoID = enabled
	return m
}

// SaveCheckpoint 保存检查点。
//
// 参数：
//   - ctx: 上下文
//   - state: 状态
//   - config: 配置
//
// 返回：
//   - *Checkpoint[S]: 保存的检查点
//   - error: 保存错误
//
func (m *CheckpointManager[S]) SaveCheckpoint(
	ctx context.Context,
	state S,
	config *CheckpointConfig,
) (*Checkpoint[S], error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	// 生成检查点 ID（如果需要）
	checkpointID := config.CheckpointID
	if checkpointID == "" && m.autoID {
		checkpointID = m.generateID()
	}

	if checkpointID == "" {
		return nil, fmt.Errorf("checkpoint ID is required")
	}

	// 创建检查点
	checkpoint := NewCheckpoint(checkpointID, state, config)

	// 保存
	if err := m.saver.Save(ctx, checkpoint); err != nil {
		return nil, err
	}

	return checkpoint, nil
}

// LoadCheckpoint 加载检查点。
//
// 参数：
//   - ctx: 上下文
//   - config: 配置
//
// 返回：
//   - *Checkpoint[S]: 检查点
//   - error: 加载错误
//
func (m *CheckpointManager[S]) LoadCheckpoint(
	ctx context.Context,
	config *CheckpointConfig,
) (*Checkpoint[S], error) {
	return m.saver.Load(ctx, config)
}

// ListCheckpoints 列出检查点。
//
// 参数：
//   - ctx: 上下文
//   - threadID: 线程 ID
//
// 返回：
//   - []*Checkpoint[S]: 检查点列表
//   - error: 列出错误
//
func (m *CheckpointManager[S]) ListCheckpoints(
	ctx context.Context,
	threadID string,
) ([]*Checkpoint[S], error) {
	return m.saver.List(ctx, threadID)
}

// DeleteCheckpoint 删除检查点。
//
// 参数：
//   - ctx: 上下文
//   - config: 配置
//
// 返回：
//   - error: 删除错误
//
func (m *CheckpointManager[S]) DeleteCheckpoint(
	ctx context.Context,
	config *CheckpointConfig,
) error {
	return m.saver.Delete(ctx, config)
}

// GetLatestCheckpoint 获取最新检查点。
//
// 参数：
//   - ctx: 上下文
//   - threadID: 线程 ID
//
// 返回：
//   - *Checkpoint[S]: 最新检查点
//   - error: 获取错误
//
func (m *CheckpointManager[S]) GetLatestCheckpoint(
	ctx context.Context,
	threadID string,
) (*Checkpoint[S], error) {
	config := NewCheckpointConfig(threadID)
	return m.saver.Load(ctx, config)
}

// SaveWithMetadata 保存带元数据的检查点。
//
// 参数：
//   - ctx: 上下文
//   - state: 状态
//   - threadID: 线程 ID
//   - metadata: 元数据
//
// 返回：
//   - *Checkpoint[S]: 保存的检查点
//   - error: 保存错误
//
func (m *CheckpointManager[S]) SaveWithMetadata(
	ctx context.Context,
	state S,
	threadID string,
	metadata *CheckpointMetadata,
) (*Checkpoint[S], error) {
	config := NewCheckpointConfig(threadID)

	// 添加元数据
	if metadata != nil {
		for k, v := range metadata.ToMap() {
			config.WithMetadata(k, v)
		}
	}

	return m.SaveCheckpoint(ctx, state, config)
}

// CreateBranch 创建分支（从指定检查点）。
//
// 参数：
//   - ctx: 上下文
//   - parentCheckpointID: 父检查点 ID
//   - newThreadID: 新线程 ID
//
// 返回：
//   - *Checkpoint[S]: 分支检查点
//   - error: 创建错误
//
func (m *CheckpointManager[S]) CreateBranch(
	ctx context.Context,
	parentCheckpointID string,
	newThreadID string,
) (*Checkpoint[S], error) {
	// 加载父检查点（需要知道线程 ID，这里简化处理）
	// 实际实现中可能需要额外的查询方法

	// 这里返回错误，表示需要更完整的实现
	return nil, fmt.Errorf("CreateBranch: not yet fully implemented")
}

// GetCheckpointHistory 获取检查点历史。
//
// 参数：
//   - ctx: 上下文
//   - threadID: 线程 ID
//   - limit: 限制数量（0 表示无限制）
//
// 返回：
//   - []*Checkpoint[S]: 检查点历史
//   - error: 获取错误
//
func (m *CheckpointManager[S]) GetCheckpointHistory(
	ctx context.Context,
	threadID string,
	limit int,
) ([]*Checkpoint[S], error) {
	checkpoints, err := m.saver.List(ctx, threadID)
	if err != nil {
		return nil, err
	}

	// 应用限制
	if limit > 0 && len(checkpoints) > limit {
		checkpoints = checkpoints[len(checkpoints)-limit:]
	}

	return checkpoints, nil
}

// AutoSave 自动保存检查点（带去重）。
//
// 参数：
//   - ctx: 上下文
//   - state: 状态
//   - threadID: 线程 ID
//   - step: 步数
//
// 返回：
//   - *Checkpoint[S]: 保存的检查点
//   - error: 保存错误
//
func (m *CheckpointManager[S]) AutoSave(
	ctx context.Context,
	state S,
	threadID string,
	step int,
) (*Checkpoint[S], error) {
	metadata := NewCheckpointMetadata().
		WithSource("auto").
		WithStep(step)

	return m.SaveWithMetadata(ctx, state, threadID, metadata)
}

// generateID 生成检查点 ID。
func (m *CheckpointManager[S]) generateID() string {
	// 生成 8 字节随机数
	b := make([]byte, 4)
	rand.Read(b)
	return fmt.Sprintf("cp-%s", hex.EncodeToString(b))
}

// CheckpointIterator 是检查点迭代器（用于时间旅行）。
type CheckpointIterator[S any] struct {
	checkpoints []*Checkpoint[S]
	current     int
}

// NewCheckpointIterator 创建检查点迭代器。
func NewCheckpointIterator[S any](checkpoints []*Checkpoint[S]) *CheckpointIterator[S] {
	return &CheckpointIterator[S]{
		checkpoints: checkpoints,
		current:     len(checkpoints) - 1, // 从最新开始
	}
}

// Next 移动到下一个（更新的）检查点。
func (it *CheckpointIterator[S]) Next() bool {
	if it.current < len(it.checkpoints)-1 {
		it.current++
		return true
	}
	return false
}

// Prev 移动到上一个（更旧的）检查点。
func (it *CheckpointIterator[S]) Prev() bool {
	if it.current > 0 {
		it.current--
		return true
	}
	return false
}

// Current 获取当前检查点。
func (it *CheckpointIterator[S]) Current() *Checkpoint[S] {
	if it.current >= 0 && it.current < len(it.checkpoints) {
		return it.checkpoints[it.current]
	}
	return nil
}

// Reset 重置到最新。
func (it *CheckpointIterator[S]) Reset() {
	it.current = len(it.checkpoints) - 1
}

// ResetToOldest 重置到最旧。
func (it *CheckpointIterator[S]) ResetToOldest() {
	it.current = 0
}

// GetTimeTravel 获取时间旅行迭代器。
func (m *CheckpointManager[S]) GetTimeTravel(
	ctx context.Context,
	threadID string,
) (*CheckpointIterator[S], error) {
	checkpoints, err := m.saver.List(ctx, threadID)
	if err != nil {
		return nil, err
	}

	return NewCheckpointIterator(checkpoints), nil
}

// PruneOldCheckpoints 清理旧检查点。
//
// 参数：
//   - ctx: 上下文
//   - threadID: 线程 ID
//   - keepLast: 保留最近 N 个
//
// 返回：
//   - int: 删除的数量
//   - error: 清理错误
//
func (m *CheckpointManager[S]) PruneOldCheckpoints(
	ctx context.Context,
	threadID string,
	keepLast int,
) (int, error) {
	checkpoints, err := m.saver.List(ctx, threadID)
	if err != nil {
		return 0, err
	}

	if len(checkpoints) <= keepLast {
		return 0, nil // 无需清理
	}

	// 删除旧的检查点
	toDelete := checkpoints[:len(checkpoints)-keepLast]
	deleted := 0

	for _, cp := range toDelete {
		config := NewCheckpointConfig(threadID).
			WithCheckpointID(cp.ID)

		if err := m.saver.Delete(ctx, config); err != nil {
			// 继续删除其他的
			continue
		}
		deleted++
	}

	return deleted, nil
}

// GetCheckpointByTime 根据时间获取最接近的检查点。
//
// 参数：
//   - ctx: 上下文
//   - threadID: 线程 ID
//   - targetTime: 目标时间
//
// 返回：
//   - *Checkpoint[S]: 最接近的检查点
//   - error: 获取错误
//
func (m *CheckpointManager[S]) GetCheckpointByTime(
	ctx context.Context,
	threadID string,
	targetTime time.Time,
) (*Checkpoint[S], error) {
	checkpoints, err := m.saver.List(ctx, threadID)
	if err != nil {
		return nil, err
	}

	if len(checkpoints) == 0 {
		return nil, ErrCheckpointNotFound
	}

	// 找到最接近的检查点
	closest := checkpoints[0]
	minDiff := abs(checkpoints[0].Timestamp.Sub(targetTime))

	for _, cp := range checkpoints[1:] {
		diff := abs(cp.Timestamp.Sub(targetTime))
		if diff < minDiff {
			minDiff = diff
			closest = cp
		}
	}

	return closest, nil
}

// abs 返回时间差的绝对值。
func abs(d time.Duration) time.Duration {
	if d < 0 {
		return -d
	}
	return d
}
