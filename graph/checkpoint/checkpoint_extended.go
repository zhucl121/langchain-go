package checkpoint

import "time"

// CheckpointBlob 表示大数据块
//
// 用于存储 channel 的大数据,实现与主 checkpoint 表的分离
// 当状态数据超过阈值时,会自动存储到 Blob 表
//
type CheckpointBlob struct {
	ThreadID     string    // 线程 ID
	CheckpointNS string    // 命名空间
	Channel      string    // Channel 名称
	Version      string    // Channel 版本(通常是 checkpoint ID)
	Type         string    // 数据类型
	Data         []byte    // 二进制数据
	CreatedAt    time.Time // 创建时间
}

// CheckpointWrite 表示写入记录
//
// 用于追踪 checkpoint 的细粒度写入操作
// 支持追踪每个任务的写入历史,便于调试和回滚
//
type CheckpointWrite struct {
	ThreadID     string         // 线程 ID
	CheckpointNS string         // 命名空间
	CheckpointID string         // Checkpoint ID
	TaskID       string         // 任务 ID
	Idx          int            // 写入索引(排序用)
	Channel      string         // Channel 名称
	Type         string         // 写入类型
	Value        map[string]any // 写入值
	CreatedAt    time.Time      // 创建时间
}

// NewCheckpointWrite 创建写入记录
func NewCheckpointWrite(threadID, checkpointNS, checkpointID, taskID, channel string, idx int) *CheckpointWrite {
	return &CheckpointWrite{
		ThreadID:     threadID,
		CheckpointNS: checkpointNS,
		CheckpointID: checkpointID,
		TaskID:       taskID,
		Idx:          idx,
		Channel:      channel,
		Value:        make(map[string]any),
		CreatedAt:    time.Now(),
	}
}

// WithType 设置写入类型
func (w *CheckpointWrite) WithType(t string) *CheckpointWrite {
	w.Type = t
	return w
}

// WithValue 设置写入值
func (w *CheckpointWrite) WithValue(key string, value any) *CheckpointWrite {
	w.Value[key] = value
	return w
}

// WithValues 批量设置写入值
func (w *CheckpointWrite) WithValues(values map[string]any) *CheckpointWrite {
	for k, v := range values {
		w.Value[k] = v
	}
	return w
}
