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

// initSchema 初始化数据库表结构(三表架构)。
func (p *PostgresCheckpointSaver[S]) initSchema() error {
	// 主 checkpoint 表
	checkpointsTable := `
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

	CREATE INDEX IF NOT EXISTS idx_checkpoints_thread_ns ON checkpoints(thread_id, checkpoint_ns);
	CREATE INDEX IF NOT EXISTS idx_checkpoints_timestamp ON checkpoints(timestamp);
	CREATE INDEX IF NOT EXISTS idx_checkpoints_created_at ON checkpoints(created_at DESC);
	CREATE INDEX IF NOT EXISTS idx_checkpoints_metadata_gin ON checkpoints USING GIN (metadata);
	`

	// Blob 存储表(用于大数据分离)
	blobsTable := `
	CREATE TABLE IF NOT EXISTS checkpoint_blobs (
		thread_id TEXT NOT NULL,
		checkpoint_ns TEXT NOT NULL DEFAULT '',
		channel TEXT NOT NULL,
		version TEXT NOT NULL,
		type TEXT,
		data BYTEA NOT NULL,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (thread_id, checkpoint_ns, channel, version)
	);

	CREATE INDEX IF NOT EXISTS idx_checkpoint_blobs_thread_ns ON checkpoint_blobs(thread_id, checkpoint_ns);
	CREATE INDEX IF NOT EXISTS idx_checkpoint_blobs_created_at ON checkpoint_blobs(created_at DESC);
	`

	// 写入追踪表(用于细粒度状态管理)
	writesTable := `
	CREATE TABLE IF NOT EXISTS checkpoint_writes (
		thread_id TEXT NOT NULL,
		checkpoint_ns TEXT NOT NULL DEFAULT '',
		checkpoint_id TEXT NOT NULL,
		task_id TEXT NOT NULL,
		idx INTEGER NOT NULL,
		channel TEXT NOT NULL,
		type TEXT,
		value JSONB NOT NULL,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (thread_id, checkpoint_ns, checkpoint_id, task_id, idx)
	);

	CREATE INDEX IF NOT EXISTS idx_checkpoint_writes_thread_ns ON checkpoint_writes(thread_id, checkpoint_ns);
	CREATE INDEX IF NOT EXISTS idx_checkpoint_writes_checkpoint ON checkpoint_writes(checkpoint_id);
	CREATE INDEX IF NOT EXISTS idx_checkpoint_writes_task ON checkpoint_writes(task_id);
	CREATE INDEX IF NOT EXISTS idx_checkpoint_writes_idx ON checkpoint_writes(idx);
	`

	// 按顺序创建表
	if _, err := p.db.Exec(checkpointsTable); err != nil {
		return fmt.Errorf("failed to create checkpoints table: %w", err)
	}

	if _, err := p.db.Exec(blobsTable); err != nil {
		return fmt.Errorf("failed to create checkpoint_blobs table: %w", err)
	}

	if _, err := p.db.Exec(writesTable); err != nil {
		return fmt.Errorf("failed to create checkpoint_writes table: %w", err)
	}

	return nil
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

// SaveWrite 保存写入记录
//
// 用于追踪细粒度的写入操作,支持调试和状态追踪
//
// 参数:
//   - ctx: 上下文
//   - write: 写入记录
//
// 返回:
//   - error: 保存错误
//
func (p *PostgresCheckpointSaver[S]) SaveWrite(ctx context.Context, write *CheckpointWrite) error {
	if write == nil {
		return fmt.Errorf("write cannot be nil")
	}

	valueData, err := json.Marshal(write.Value)
	if err != nil {
		return fmt.Errorf("failed to marshal write value: %w", err)
	}

	query := `
	INSERT INTO checkpoint_writes
	(thread_id, checkpoint_ns, checkpoint_id, task_id, idx, channel, type, value, created_at)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	ON CONFLICT (thread_id, checkpoint_ns, checkpoint_id, task_id, idx) DO UPDATE SET
		value = EXCLUDED.value,
		type = EXCLUDED.type
	`

	_, err = p.db.ExecContext(ctx, query,
		write.ThreadID,
		write.CheckpointNS,
		write.CheckpointID,
		write.TaskID,
		write.Idx,
		write.Channel,
		write.Type,
		valueData,
		time.Now(),
	)

	if err != nil {
		return fmt.Errorf("failed to save write: %w", err)
	}

	return nil
}

// ListWrites 列出写入记录
//
// 获取指定 checkpoint 的所有写入记录,按索引排序
//
// 参数:
//   - ctx: 上下文
//   - threadID: 线程 ID
//   - checkpointNS: 命名空间
//   - checkpointID: Checkpoint ID
//
// 返回:
//   - []*CheckpointWrite: 写入记录列表
//   - error: 列出错误
//
func (p *PostgresCheckpointSaver[S]) ListWrites(ctx context.Context, threadID, checkpointNS, checkpointID string) ([]*CheckpointWrite, error) {
	query := `
	SELECT thread_id, checkpoint_ns, checkpoint_id, task_id, idx, channel, type, value, created_at
	FROM checkpoint_writes
	WHERE thread_id = $1 AND checkpoint_ns = $2 AND checkpoint_id = $3
	ORDER BY idx ASC
	`

	rows, err := p.db.QueryContext(ctx, query, threadID, checkpointNS, checkpointID)
	if err != nil {
		return nil, fmt.Errorf("failed to query writes: %w", err)
	}
	defer rows.Close()

	var writes []*CheckpointWrite
	for rows.Next() {
		var write CheckpointWrite
		var valueData []byte

		err := rows.Scan(
			&write.ThreadID,
			&write.CheckpointNS,
			&write.CheckpointID,
			&write.TaskID,
			&write.Idx,
			&write.Channel,
			&write.Type,
			&valueData,
			&write.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan write: %w", err)
		}

		if err := json.Unmarshal(valueData, &write.Value); err != nil {
			return nil, fmt.Errorf("failed to unmarshal write value: %w", err)
		}

		writes = append(writes, &write)
	}

	return writes, rows.Err()
}

// DeleteWrites 删除写入记录
//
// 删除指定 checkpoint 的所有写入记录
//
// 参数:
//   - ctx: 上下文
//   - threadID: 线程 ID
//   - checkpointNS: 命名空间
//   - checkpointID: Checkpoint ID
//
// 返回:
//   - error: 删除错误
//
func (p *PostgresCheckpointSaver[S]) DeleteWrites(ctx context.Context, threadID, checkpointNS, checkpointID string) error {
	query := `
	DELETE FROM checkpoint_writes
	WHERE thread_id = $1 AND checkpoint_ns = $2 AND checkpoint_id = $3
	`

	_, err := p.db.ExecContext(ctx, query, threadID, checkpointNS, checkpointID)
	if err != nil {
		return fmt.Errorf("failed to delete writes: %w", err)
	}

	return nil
}

// SaveBlob 保存 Blob 数据
//
// 用于存储大数据块,实现与主表的分离
//
// 参数:
//   - ctx: 上下文
//   - blob: Blob 数据
//
// 返回:
//   - error: 保存错误
//
func (p *PostgresCheckpointSaver[S]) SaveBlob(ctx context.Context, blob *CheckpointBlob) error {
	if blob == nil {
		return fmt.Errorf("blob cannot be nil")
	}

	query := `
	INSERT INTO checkpoint_blobs
	(thread_id, checkpoint_ns, channel, version, type, data, created_at)
	VALUES ($1, $2, $3, $4, $5, $6, $7)
	ON CONFLICT (thread_id, checkpoint_ns, channel, version) DO UPDATE SET
		data = EXCLUDED.data,
		type = EXCLUDED.type
	`

	_, err := p.db.ExecContext(ctx, query,
		blob.ThreadID,
		blob.CheckpointNS,
		blob.Channel,
		blob.Version,
		blob.Type,
		blob.Data,
		time.Now(),
	)

	if err != nil {
		return fmt.Errorf("failed to save blob: %w", err)
	}

	return nil
}

// LoadBlob 加载 Blob 数据
//
// 参数:
//   - ctx: 上下文
//   - threadID: 线程 ID
//   - checkpointNS: 命名空间
//   - channel: Channel 名称
//   - version: 版本
//
// 返回:
//   - *CheckpointBlob: Blob 数据
//   - error: 加载错误
//
func (p *PostgresCheckpointSaver[S]) LoadBlob(ctx context.Context, threadID, checkpointNS, channel, version string) (*CheckpointBlob, error) {
	query := `
	SELECT thread_id, checkpoint_ns, channel, version, type, data, created_at
	FROM checkpoint_blobs
	WHERE thread_id = $1 AND checkpoint_ns = $2 AND channel = $3 AND version = $4
	`

	var blob CheckpointBlob
	err := p.db.QueryRowContext(ctx, query, threadID, checkpointNS, channel, version).Scan(
		&blob.ThreadID,
		&blob.CheckpointNS,
		&blob.Channel,
		&blob.Version,
		&blob.Type,
		&blob.Data,
		&blob.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("blob not found")
	}

	if err != nil {
		return nil, fmt.Errorf("failed to load blob: %w", err)
	}

	return &blob, nil
}

// DeleteBlob 删除 Blob 数据
//
// 参数:
//   - ctx: 上下文
//   - threadID: 线程 ID
//   - checkpointNS: 命名空间
//   - channel: Channel 名称
//   - version: 版本
//
// 返回:
//   - error: 删除错误
//
func (p *PostgresCheckpointSaver[S]) DeleteBlob(ctx context.Context, threadID, checkpointNS, channel, version string) error {
	query := `
	DELETE FROM checkpoint_blobs
	WHERE thread_id = $1 AND checkpoint_ns = $2 AND channel = $3 AND version = $4
	`

	_, err := p.db.ExecContext(ctx, query, threadID, checkpointNS, channel, version)
	if err != nil {
		return fmt.Errorf("failed to delete blob: %w", err)
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
