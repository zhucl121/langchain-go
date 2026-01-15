// +build postgres

package checkpoint

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

// PostgresCheckpointSaver 是 Postgres 检查点保存器。
//
// PostgresCheckpointSaver 将检查点保存到 PostgreSQL 数据库。
// 适用于生产环境和需要高可用性的场景。
//
type PostgresCheckpointSaver[S any] struct {
	db *sql.DB
}

// NewPostgresCheckpointSaver 创建 Postgres 检查点保存器。
//
// 参数：
//   - connStr: 数据库连接字符串
//     例如: "postgres://user:password@localhost/dbname?sslmode=disable"
//
// 返回：
//   - *PostgresCheckpointSaver[S]: 保存器实例
//   - error: 创建错误
//
func NewPostgresCheckpointSaver[S any](connStr string) (*PostgresCheckpointSaver[S], error) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// 测试连接
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	saver := &PostgresCheckpointSaver[S]{
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
func (p *PostgresCheckpointSaver[S]) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS checkpoints (
		id TEXT NOT NULL,
		thread_id TEXT NOT NULL,
		checkpoint_ns TEXT NOT NULL DEFAULT '',
		parent_id TEXT,
		type TEXT,
		state JSONB NOT NULL,
		timestamp BIGINT NOT NULL,
		metadata JSONB,
		version INTEGER NOT NULL,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (thread_id, checkpoint_ns, id)
	);

	CREATE INDEX IF NOT EXISTS idx_thread_ns ON checkpoints(thread_id, checkpoint_ns);
	CREATE INDEX IF NOT EXISTS idx_timestamp ON checkpoints(timestamp);
	CREATE INDEX IF NOT EXISTS idx_created_at ON checkpoints(created_at DESC);
	CREATE INDEX IF NOT EXISTS idx_metadata_gin ON checkpoints USING GIN (metadata);
	`

	_, err := p.db.Exec(schema)
	return err
}

// Save 实现 CheckpointSaver 接口。
func (p *PostgresCheckpointSaver[S]) Save(ctx context.Context, checkpoint *Checkpoint[S]) error {
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

	// 使用 UPSERT (ON CONFLICT)
	query := `
	INSERT INTO checkpoints 
	(id, thread_id, checkpoint_ns, parent_id, type, state, timestamp, metadata, version, created_at)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	ON CONFLICT (thread_id, checkpoint_ns, id) DO UPDATE SET
		state = EXCLUDED.state,
		type = EXCLUDED.type,
		timestamp = EXCLUDED.timestamp,
		metadata = EXCLUDED.metadata,
		version = EXCLUDED.version
	`

	_, err = p.db.ExecContext(ctx, query,
		checkpoint.ID,
		checkpoint.ThreadID,
		checkpoint.CheckpointNS,
		checkpoint.ParentID,
		checkpoint.Type,
		stateData,
		checkpoint.Timestamp.Unix(),
		metadataData,
		checkpoint.Version,
		time.Now(),
	)

	if err != nil {
		return fmt.Errorf("failed to save checkpoint: %w", err)
	}

	return nil
}

// Load 实现 CheckpointSaver 接口。
func (p *PostgresCheckpointSaver[S]) Load(ctx context.Context, config *CheckpointConfig) (*Checkpoint[S], error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	var query string
	var args []any

	if config.CheckpointID != "" {
		query = `
		SELECT id, thread_id, checkpoint_ns, parent_id, type, state, timestamp, metadata, version
		FROM checkpoints
		WHERE id = $1 AND thread_id = $2 AND checkpoint_ns = $3
		`
		args = []any{config.CheckpointID, config.ThreadID, config.CheckpointNS}
	} else {
		query = `
		SELECT id, thread_id, checkpoint_ns, parent_id, type, state, timestamp, metadata, version
		FROM checkpoints
		WHERE thread_id = $1 AND checkpoint_ns = $2
		ORDER BY timestamp DESC
		LIMIT 1
		`
		args = []any{config.ThreadID, config.CheckpointNS}
	}

	var id, threadID, checkpointNS, parentID, cpType string
	var stateData, metadataData []byte
	var timestamp int64
	var version int

	err := p.db.QueryRowContext(ctx, query, args...).Scan(
		&id, &threadID, &checkpointNS, &parentID, &cpType, &stateData, &timestamp, &metadataData, &version,
	)

	if err == sql.ErrNoRows {
		return nil, ErrCheckpointNotFound
	}

	if err != nil {
		return nil, fmt.Errorf("failed to load checkpoint: %w", err)
	}

	// 反序列化状态
	var state S
	if err := json.Unmarshal(stateData, &state); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDeserializeFailed, err)
	}

	// 反序列化元数据
	var metadata map[string]any
	if err := json.Unmarshal(metadataData, &metadata); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDeserializeFailed, err)
	}

	checkpoint := &Checkpoint[S]{
		ID:           id,
		ThreadID:     threadID,
		CheckpointNS: checkpointNS,
		ParentID:     parentID,
		Type:         cpType,
		State:        state,
		Timestamp:    time.Unix(timestamp, 0),
		Metadata:     metadata,
		Version:      version,
	}

	return checkpoint, nil
}

// List 实现 CheckpointSaver 接口。
func (p *PostgresCheckpointSaver[S]) List(ctx context.Context, threadID string) ([]*Checkpoint[S], error) {
	if threadID == "" {
		return nil, fmt.Errorf("threadID cannot be empty")
	}

	query := `
	SELECT id, thread_id, checkpoint_ns, parent_id, type, state, timestamp, metadata, version
	FROM checkpoints
	WHERE thread_id = $1
	ORDER BY timestamp ASC
	`

	rows, err := p.db.QueryContext(ctx, query, threadID)
	if err != nil {
		return nil, fmt.Errorf("failed to list checkpoints: %w", err)
	}
	defer rows.Close()

	result := make([]*Checkpoint[S], 0)

	for rows.Next() {
		var id, threadID, checkpointNS, parentID, cpType string
		var stateData, metadataData []byte
		var timestamp int64
		var version int

		err := rows.Scan(&id, &threadID, &checkpointNS, &parentID, &cpType, &stateData, &timestamp, &metadataData, &version)
		if err != nil {
			return nil, fmt.Errorf("failed to scan checkpoint: %w", err)
		}

		var state S
		if err := json.Unmarshal(stateData, &state); err != nil {
			continue
		}

		var metadata map[string]any
		if err := json.Unmarshal(metadataData, &metadata); err != nil {
			metadata = make(map[string]any)
		}

		checkpoint := &Checkpoint[S]{
			ID:           id,
			ThreadID:     threadID,
			CheckpointNS: checkpointNS,
			ParentID:     parentID,
			Type:         cpType,
			State:        state,
			Timestamp:    time.Unix(timestamp, 0),
			Metadata:     metadata,
			Version:      version,
		}

		result = append(result, checkpoint)
	}

	return result, rows.Err()
}

// Delete 实现 CheckpointSaver 接口。
func (p *PostgresCheckpointSaver[S]) Delete(ctx context.Context, config *CheckpointConfig) error {
	if err := config.Validate(); err != nil {
		return err
	}

	if config.CheckpointID == "" {
		return fmt.Errorf("checkpoint ID must be specified for deletion")
	}

	query := `
	DELETE FROM checkpoints
	WHERE id = $1 AND thread_id = $2 AND checkpoint_ns = $3
	`

	result, err := p.db.ExecContext(ctx, query, config.CheckpointID, config.ThreadID, config.CheckpointNS)
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
func (p *PostgresCheckpointSaver[S]) Close() error {
	if p.db != nil {
		return p.db.Close()
	}
	return nil
}

// GetStats 获取统计信息。
func (p *PostgresCheckpointSaver[S]) GetStats(ctx context.Context) (map[string]int, error) {
	stats := make(map[string]int)

	var totalCheckpoints int
	err := p.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM checkpoints").Scan(&totalCheckpoints)
	if err != nil {
		return nil, err
	}
	stats["total_checkpoints"] = totalCheckpoints

	var totalThreads int
	err = p.db.QueryRowContext(ctx, "SELECT COUNT(DISTINCT thread_id) FROM checkpoints").Scan(&totalThreads)
	if err != nil {
		return nil, err
	}
	stats["total_threads"] = totalThreads

	return stats, nil
}
