// +build sqlite

package checkpoint

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// SQLiteCheckpointSaver 是 SQLite 检查点保存器。
//
// SQLiteCheckpointSaver 将检查点保存到 SQLite 数据库。
// 适用于单机部署和需要持久化的场景。
//
type SQLiteCheckpointSaver[S any] struct {
	db *sql.DB
}

// NewSQLiteCheckpointSaver 创建 SQLite 检查点保存器。
//
// 参数：
//   - dbPath: 数据库文件路径（":memory:" 表示内存数据库）
//
// 返回：
//   - *SQLiteCheckpointSaver[S]: 保存器实例
//   - error: 创建错误
//
func NewSQLiteCheckpointSaver[S any](dbPath string) (*SQLiteCheckpointSaver[S], error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	saver := &SQLiteCheckpointSaver[S]{
		db: db,
	}

	// 初始化表结构
	if err := saver.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return saver, nil
}

// initSchema 初始化数据库表结构。
func (s *SQLiteCheckpointSaver[S]) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS checkpoints (
		id TEXT PRIMARY KEY,
		thread_id TEXT NOT NULL,
		parent_id TEXT,
		state TEXT NOT NULL,
		timestamp INTEGER NOT NULL,
		metadata TEXT,
		version INTEGER NOT NULL,
		created_at INTEGER NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_thread_id ON checkpoints(thread_id);
	CREATE INDEX IF NOT EXISTS idx_timestamp ON checkpoints(timestamp);
	`

	_, err := s.db.Exec(schema)
	return err
}

// Save 实现 CheckpointSaver 接口。
func (s *SQLiteCheckpointSaver[S]) Save(ctx context.Context, checkpoint *Checkpoint[S]) error {
	if checkpoint == nil {
		return fmt.Errorf("checkpoint cannot be nil")
	}

	// 序列化状态
	stateData, err := json.Marshal(checkpoint.State)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrSerializeFailed, err)
	}

	// 序列化元数据
	metadataData, err := json.Marshal(checkpoint.Metadata)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrSerializeFailed, err)
	}

	// 插入或替换
	query := `
	INSERT OR REPLACE INTO checkpoints 
	(id, thread_id, parent_id, state, timestamp, metadata, version, created_at)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = s.db.ExecContext(ctx, query,
		checkpoint.ID,
		checkpoint.ThreadID,
		checkpoint.ParentID,
		string(stateData),
		checkpoint.Timestamp.Unix(),
		string(metadataData),
		checkpoint.Version,
		time.Now().Unix(),
	)

	if err != nil {
		return fmt.Errorf("failed to save checkpoint: %w", err)
	}

	return nil
}

// Load 实现 CheckpointSaver 接口。
func (s *SQLiteCheckpointSaver[S]) Load(ctx context.Context, config *CheckpointConfig) (*Checkpoint[S], error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	var query string
	var args []any

	if config.CheckpointID != "" {
		// 加载特定检查点
		query = `
		SELECT id, thread_id, parent_id, state, timestamp, metadata, version
		FROM checkpoints
		WHERE id = ? AND thread_id = ?
		`
		args = []any{config.CheckpointID, config.ThreadID}
	} else {
		// 加载最新检查点
		query = `
		SELECT id, thread_id, parent_id, state, timestamp, metadata, version
		FROM checkpoints
		WHERE thread_id = ?
		ORDER BY timestamp DESC
		LIMIT 1
		`
		args = []any{config.ThreadID}
	}

	var id, threadID, parentID, stateData, metadataData string
	var timestamp int64
	var version int

	err := s.db.QueryRowContext(ctx, query, args...).Scan(
		&id, &threadID, &parentID, &stateData, &timestamp, &metadataData, &version,
	)

	if err == sql.ErrNoRows {
		return nil, ErrCheckpointNotFound
	}

	if err != nil {
		return nil, fmt.Errorf("failed to load checkpoint: %w", err)
	}

	// 反序列化状态
	var state S
	if err := json.Unmarshal([]byte(stateData), &state); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDeserializeFailed, err)
	}

	// 反序列化元数据
	var metadata map[string]any
	if err := json.Unmarshal([]byte(metadataData), &metadata); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDeserializeFailed, err)
	}

	checkpoint := &Checkpoint[S]{
		ID:        id,
		ThreadID:  threadID,
		ParentID:  parentID,
		State:     state,
		Timestamp: time.Unix(timestamp, 0),
		Metadata:  metadata,
		Version:   version,
	}

	return checkpoint, nil
}

// List 实现 CheckpointSaver 接口。
func (s *SQLiteCheckpointSaver[S]) List(ctx context.Context, threadID string) ([]*Checkpoint[S], error) {
	if threadID == "" {
		return nil, fmt.Errorf("threadID cannot be empty")
	}

	query := `
	SELECT id, thread_id, parent_id, state, timestamp, metadata, version
	FROM checkpoints
	WHERE thread_id = ?
	ORDER BY timestamp ASC
	`

	rows, err := s.db.QueryContext(ctx, query, threadID)
	if err != nil {
		return nil, fmt.Errorf("failed to list checkpoints: %w", err)
	}
	defer rows.Close()

	result := make([]*Checkpoint[S], 0)

	for rows.Next() {
		var id, threadID, parentID, stateData, metadataData string
		var timestamp int64
		var version int

		err := rows.Scan(&id, &threadID, &parentID, &stateData, &timestamp, &metadataData, &version)
		if err != nil {
			return nil, fmt.Errorf("failed to scan checkpoint: %w", err)
		}

		// 反序列化状态
		var state S
		if err := json.Unmarshal([]byte(stateData), &state); err != nil {
			continue // 跳过无法反序列化的记录
		}

		// 反序列化元数据
		var metadata map[string]any
		if err := json.Unmarshal([]byte(metadataData), &metadata); err != nil {
			metadata = make(map[string]any)
		}

		checkpoint := &Checkpoint[S]{
			ID:        id,
			ThreadID:  threadID,
			ParentID:  parentID,
			State:     state,
			Timestamp: time.Unix(timestamp, 0),
			Metadata:  metadata,
			Version:   version,
		}

		result = append(result, checkpoint)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating checkpoints: %w", err)
	}

	return result, nil
}

// Delete 实现 CheckpointSaver 接口。
func (s *SQLiteCheckpointSaver[S]) Delete(ctx context.Context, config *CheckpointConfig) error {
	if err := config.Validate(); err != nil {
		return err
	}

	if config.CheckpointID == "" {
		return fmt.Errorf("checkpoint ID must be specified for deletion")
	}

	query := `
	DELETE FROM checkpoints
	WHERE id = ? AND thread_id = ?
	`

	result, err := s.db.ExecContext(ctx, query, config.CheckpointID, config.ThreadID)
	if err != nil {
		return fmt.Errorf("failed to delete checkpoint: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rows == 0 {
		return ErrCheckpointNotFound
	}

	return nil
}

// Close 关闭数据库连接。
func (s *SQLiteCheckpointSaver[S]) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// GetStats 获取统计信息。
func (s *SQLiteCheckpointSaver[S]) GetStats(ctx context.Context) (map[string]int, error) {
	stats := make(map[string]int)

	// 总检查点数
	var totalCheckpoints int
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM checkpoints").Scan(&totalCheckpoints)
	if err != nil {
		return nil, err
	}
	stats["total_checkpoints"] = totalCheckpoints

	// 总线程数
	var totalThreads int
	err = s.db.QueryRowContext(ctx, "SELECT COUNT(DISTINCT thread_id) FROM checkpoints").Scan(&totalThreads)
	if err != nil {
		return nil, err
	}
	stats["total_threads"] = totalThreads

	return stats, nil
}
