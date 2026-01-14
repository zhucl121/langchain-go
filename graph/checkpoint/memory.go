package checkpoint

import (
	"context"
	"fmt"
	"sort"
	"sync"
)

// MemoryCheckpointSaver 是内存检查点保存器。
//
// MemoryCheckpointSaver 将检查点保存在内存中。
// 适用于开发、测试和不需要持久化的场景。
//
type MemoryCheckpointSaver[S any] struct {
	checkpoints map[string]*Checkpoint[S] // key: checkpointID
	threads     map[string][]string       // threadID -> []checkpointID

	mu sync.RWMutex
}

// NewMemoryCheckpointSaver 创建内存检查点保存器。
//
// 返回：
//   - *MemoryCheckpointSaver[S]: 保存器实例
//
func NewMemoryCheckpointSaver[S any]() *MemoryCheckpointSaver[S] {
	return &MemoryCheckpointSaver[S]{
		checkpoints: make(map[string]*Checkpoint[S]),
		threads:     make(map[string][]string),
	}
}

// Save 实现 CheckpointSaver 接口。
func (m *MemoryCheckpointSaver[S]) Save(ctx context.Context, checkpoint *Checkpoint[S]) error {
	if checkpoint == nil {
		return fmt.Errorf("checkpoint cannot be nil")
	}

	if checkpoint.ID == "" {
		return fmt.Errorf("checkpoint ID cannot be empty")
	}

	if checkpoint.ThreadID == "" {
		return fmt.Errorf("checkpoint ThreadID cannot be empty")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// 保存检查点（克隆以避免外部修改）
	m.checkpoints[checkpoint.ID] = checkpoint.Clone()

	// 添加到线程索引
	if _, exists := m.threads[checkpoint.ThreadID]; !exists {
		m.threads[checkpoint.ThreadID] = make([]string, 0)
	}

	// 检查是否已存在（避免重复）
	found := false
	for _, id := range m.threads[checkpoint.ThreadID] {
		if id == checkpoint.ID {
			found = true
			break
		}
	}

	if !found {
		m.threads[checkpoint.ThreadID] = append(m.threads[checkpoint.ThreadID], checkpoint.ID)
	}

	return nil
}

// Load 实现 CheckpointSaver 接口。
func (m *MemoryCheckpointSaver[S]) Load(ctx context.Context, config *CheckpointConfig) (*Checkpoint[S], error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	// 如果指定了 CheckpointID，直接加载
	if config.CheckpointID != "" {
		checkpoint, exists := m.checkpoints[config.CheckpointID]
		if !exists {
			return nil, fmt.Errorf("%w: %s", ErrCheckpointNotFound, config.CheckpointID)
		}

		// 验证线程 ID
		if checkpoint.ThreadID != config.ThreadID {
			return nil, fmt.Errorf("checkpoint %s does not belong to thread %s",
				config.CheckpointID, config.ThreadID)
		}

		return checkpoint.Clone(), nil
	}

	// 否则，加载该线程的最新检查点
	checkpointIDs, exists := m.threads[config.ThreadID]
	if !exists || len(checkpointIDs) == 0 {
		return nil, fmt.Errorf("%w: no checkpoints for thread %s", ErrCheckpointNotFound, config.ThreadID)
	}

	// 获取最新的检查点（最后一个）
	latestID := checkpointIDs[len(checkpointIDs)-1]
	checkpoint := m.checkpoints[latestID]

	return checkpoint.Clone(), nil
}

// List 实现 CheckpointSaver 接口。
func (m *MemoryCheckpointSaver[S]) List(ctx context.Context, threadID string) ([]*Checkpoint[S], error) {
	if threadID == "" {
		return nil, fmt.Errorf("threadID cannot be empty")
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	checkpointIDs, exists := m.threads[threadID]
	if !exists {
		return []*Checkpoint[S]{}, nil
	}

	// 收集所有检查点
	result := make([]*Checkpoint[S], 0, len(checkpointIDs))
	for _, id := range checkpointIDs {
		if checkpoint, exists := m.checkpoints[id]; exists {
			result = append(result, checkpoint.Clone())
		}
	}

	// 按时间戳排序
	sort.Slice(result, func(i, j int) bool {
		return result[i].Timestamp.Before(result[j].Timestamp)
	})

	return result, nil
}

// Delete 实现 CheckpointSaver 接口。
func (m *MemoryCheckpointSaver[S]) Delete(ctx context.Context, config *CheckpointConfig) error {
	if err := config.Validate(); err != nil {
		return err
	}

	if config.CheckpointID == "" {
		return fmt.Errorf("checkpoint ID must be specified for deletion")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// 检查检查点是否存在
	checkpoint, exists := m.checkpoints[config.CheckpointID]
	if !exists {
		return fmt.Errorf("%w: %s", ErrCheckpointNotFound, config.CheckpointID)
	}

	// 验证线程 ID
	if checkpoint.ThreadID != config.ThreadID {
		return fmt.Errorf("checkpoint %s does not belong to thread %s",
			config.CheckpointID, config.ThreadID)
	}

	// 从检查点映射中删除
	delete(m.checkpoints, config.CheckpointID)

	// 从线程索引中删除
	if checkpointIDs, exists := m.threads[config.ThreadID]; exists {
		newIDs := make([]string, 0, len(checkpointIDs)-1)
		for _, id := range checkpointIDs {
			if id != config.CheckpointID {
				newIDs = append(newIDs, id)
			}
		}
		m.threads[config.ThreadID] = newIDs
	}

	return nil
}

// GetStats 获取统计信息。
func (m *MemoryCheckpointSaver[S]) GetStats() map[string]int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return map[string]int{
		"total_checkpoints": len(m.checkpoints),
		"total_threads":     len(m.threads),
	}
}

// Clear 清空所有检查点。
func (m *MemoryCheckpointSaver[S]) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.checkpoints = make(map[string]*Checkpoint[S])
	m.threads = make(map[string][]string)
}

// GetCheckpoint 直接获取检查点（不通过 Load）。
//
// 这是一个便捷方法，用于测试和调试。
//
func (m *MemoryCheckpointSaver[S]) GetCheckpoint(checkpointID string) (*Checkpoint[S], bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	checkpoint, exists := m.checkpoints[checkpointID]
	if !exists {
		return nil, false
	}

	return checkpoint.Clone(), true
}
