package feedback

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/zhucl121/langchain-go/pkg/types"
)

// PostgreSQLStorage PostgreSQL 存储实现
type PostgreSQLStorage struct {
	db *sql.DB
}

// NewPostgreSQLStorage 创建 PostgreSQL 存储
func NewPostgreSQLStorage(db *sql.DB) Storage {
	return &PostgreSQLStorage{
		db: db,
	}
}

// InitSchema 初始化数据库 Schema
func (s *PostgreSQLStorage) InitSchema(ctx context.Context) error {
	schema := `
	CREATE TABLE IF NOT EXISTS learning_queries (
		id VARCHAR(255) PRIMARY KEY,
		text TEXT NOT NULL,
		user_id VARCHAR(255),
		strategy VARCHAR(100),
		timestamp TIMESTAMP NOT NULL,
		metadata JSONB
	);

	CREATE TABLE IF NOT EXISTS learning_results (
		id SERIAL PRIMARY KEY,
		query_id VARCHAR(255) REFERENCES learning_queries(id) ON DELETE CASCADE,
		document_id VARCHAR(255),
		rank INT,
		score FLOAT,
		document JSONB,
		timestamp TIMESTAMP NOT NULL
	);

	CREATE TABLE IF NOT EXISTS learning_explicit_feedback (
		id SERIAL PRIMARY KEY,
		query_id VARCHAR(255) REFERENCES learning_queries(id) ON DELETE CASCADE,
		user_id VARCHAR(255),
		type VARCHAR(50),
		rating INT,
		comment TEXT,
		timestamp TIMESTAMP NOT NULL
	);

	CREATE TABLE IF NOT EXISTS learning_implicit_feedback (
		id SERIAL PRIMARY KEY,
		query_id VARCHAR(255) REFERENCES learning_queries(id) ON DELETE CASCADE,
		user_id VARCHAR(255),
		document_id VARCHAR(255),
		action VARCHAR(50),
		duration_ms BIGINT,
		timestamp TIMESTAMP NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_learning_queries_user ON learning_queries(user_id);
	CREATE INDEX IF NOT EXISTS idx_learning_queries_timestamp ON learning_queries(timestamp);
	CREATE INDEX IF NOT EXISTS idx_learning_queries_strategy ON learning_queries(strategy);
	CREATE INDEX IF NOT EXISTS idx_learning_explicit_query ON learning_explicit_feedback(query_id);
	CREATE INDEX IF NOT EXISTS idx_learning_implicit_query ON learning_implicit_feedback(query_id);
	`

	_, err := s.db.ExecContext(ctx, schema)
	return err
}

// SaveQuery 保存查询
func (s *PostgreSQLStorage) SaveQuery(ctx context.Context, query *Query) error {
	metadata, err := json.Marshal(query.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO learning_queries (id, text, user_id, strategy, timestamp, metadata)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (id) DO UPDATE SET
			text = EXCLUDED.text,
			user_id = EXCLUDED.user_id,
			strategy = EXCLUDED.strategy,
			timestamp = EXCLUDED.timestamp,
			metadata = EXCLUDED.metadata
	`, query.ID, query.Text, query.UserID, query.Strategy, query.Timestamp, metadata)

	return err
}

// SaveResults 保存检索结果
func (s *PostgreSQLStorage) SaveResults(ctx context.Context, queryID string, results []types.Document) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 先删除已有结果
	_, err = tx.ExecContext(ctx, "DELETE FROM learning_results WHERE query_id = $1", queryID)
	if err != nil {
		return err
	}

	// 插入新结果
	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO learning_results (query_id, document_id, rank, score, document, timestamp)
		VALUES ($1, $2, $3, $4, $5, $6)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	now := time.Now()
	for i, doc := range results {
		docJSON, err := json.Marshal(doc)
		if err != nil {
			return fmt.Errorf("failed to marshal document: %w", err)
		}

		score := 0.0
		if doc.Metadata != nil {
			if s, ok := doc.Metadata["score"].(float64); ok {
				score = s
			}
		}

		_, err = stmt.ExecContext(ctx, queryID, doc.ID, i+1, score, docJSON, now)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// SaveExplicitFeedback 保存显式反馈
func (s *PostgreSQLStorage) SaveExplicitFeedback(ctx context.Context, feedback *ExplicitFeedback) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO learning_explicit_feedback (query_id, user_id, type, rating, comment, timestamp)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, feedback.QueryID, feedback.UserID, feedback.Type, feedback.Rating, feedback.Comment, feedback.Timestamp)

	return err
}

// SaveImplicitFeedback 保存隐式反馈
func (s *PostgreSQLStorage) SaveImplicitFeedback(ctx context.Context, feedback *ImplicitFeedback) error {
	durationMS := feedback.Duration.Milliseconds()

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO learning_implicit_feedback (query_id, user_id, document_id, action, duration_ms, timestamp)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, feedback.QueryID, feedback.UserID, feedback.DocumentID, feedback.Action, durationMS, feedback.Timestamp)

	return err
}

// GetQueryFeedback 获取查询反馈
func (s *PostgreSQLStorage) GetQueryFeedback(ctx context.Context, queryID string) (*QueryFeedback, error) {
	qf := &QueryFeedback{
		ExplicitFeedback: make([]ExplicitFeedback, 0),
		ImplicitFeedback: make([]ImplicitFeedback, 0),
		Results:          make([]types.Document, 0),
	}

	// 获取查询信息
	var metadataJSON []byte
	err := s.db.QueryRowContext(ctx, `
		SELECT id, text, user_id, strategy, timestamp, metadata
		FROM learning_queries
		WHERE id = $1
	`, queryID).Scan(
		&qf.Query.ID,
		&qf.Query.Text,
		&qf.Query.UserID,
		&qf.Query.Strategy,
		&qf.Query.Timestamp,
		&metadataJSON,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get query: %w", err)
	}

	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &qf.Query.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	// 获取结果
	rows, err := s.db.QueryContext(ctx, `
		SELECT document FROM learning_results
		WHERE query_id = $1
		ORDER BY rank
	`, queryID)
	if err != nil {
		return nil, fmt.Errorf("failed to get results: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var docJSON []byte
		if err := rows.Scan(&docJSON); err != nil {
			return nil, err
		}

		var doc types.Document
		if err := json.Unmarshal(docJSON, &doc); err != nil {
			return nil, fmt.Errorf("failed to unmarshal document: %w", err)
		}

		qf.Results = append(qf.Results, doc)
	}

	// 获取显式反馈
	rows, err = s.db.QueryContext(ctx, `
		SELECT query_id, user_id, type, rating, comment, timestamp
		FROM learning_explicit_feedback
		WHERE query_id = $1
	`, queryID)
	if err != nil {
		return nil, fmt.Errorf("failed to get explicit feedback: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var fb ExplicitFeedback
		if err := rows.Scan(&fb.QueryID, &fb.UserID, &fb.Type, &fb.Rating, &fb.Comment, &fb.Timestamp); err != nil {
			return nil, err
		}
		qf.ExplicitFeedback = append(qf.ExplicitFeedback, fb)
	}

	// 获取隐式反馈
	rows, err = s.db.QueryContext(ctx, `
		SELECT query_id, user_id, document_id, action, duration_ms, timestamp
		FROM learning_implicit_feedback
		WHERE query_id = $1
	`, queryID)
	if err != nil {
		return nil, fmt.Errorf("failed to get implicit feedback: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var fb ImplicitFeedback
		var durationMS int64
		if err := rows.Scan(&fb.QueryID, &fb.UserID, &fb.DocumentID, &fb.Action, &durationMS, &fb.Timestamp); err != nil {
			return nil, err
		}
		fb.Duration = time.Duration(durationMS) * time.Millisecond
		qf.ImplicitFeedback = append(qf.ImplicitFeedback, fb)
	}

	// 计算统计指标
	qf.AvgRating = calculateAvgRating(qf.ExplicitFeedback)
	qf.CTR = calculateCTR(qf.Results, qf.ImplicitFeedback)
	qf.AvgReadDuration = calculateAvgReadDuration(qf.ImplicitFeedback)

	return qf, nil
}

// ListQueries 列出查询
func (s *PostgreSQLStorage) ListQueries(ctx context.Context, opts ListOptions) ([]Query, error) {
	query := `SELECT id, text, user_id, strategy, timestamp, metadata FROM learning_queries WHERE 1=1`
	args := make([]interface{}, 0)
	argCount := 1

	if opts.UserID != "" {
		query += fmt.Sprintf(" AND user_id = $%d", argCount)
		args = append(args, opts.UserID)
		argCount++
	}

	if opts.Strategy != "" {
		query += fmt.Sprintf(" AND strategy = $%d", argCount)
		args = append(args, opts.Strategy)
		argCount++
	}

	if !opts.StartTime.IsZero() {
		query += fmt.Sprintf(" AND timestamp >= $%d", argCount)
		args = append(args, opts.StartTime)
		argCount++
	}

	if !opts.EndTime.IsZero() {
		query += fmt.Sprintf(" AND timestamp <= $%d", argCount)
		args = append(args, opts.EndTime)
		argCount++
	}

	query += " ORDER BY timestamp DESC"

	if opts.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argCount)
		args = append(args, opts.Limit)
		argCount++
	}

	if opts.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argCount)
		args = append(args, opts.Offset)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	queries := make([]Query, 0)
	for rows.Next() {
		var q Query
		var metadataJSON []byte

		if err := rows.Scan(&q.ID, &q.Text, &q.UserID, &q.Strategy, &q.Timestamp, &metadataJSON); err != nil {
			return nil, err
		}

		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &q.Metadata); err != nil {
				return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
			}
		}

		queries = append(queries, q)
	}

	return queries, nil
}

// Aggregate 聚合统计
func (s *PostgreSQLStorage) Aggregate(ctx context.Context, opts AggregateOptions) (*FeedbackStats, error) {
	stats := &FeedbackStats{
		TopQueries:       make([]string, 0),
		LowRatingQueries: make([]string, 0),
	}

	// 构建时间过滤条件
	timeFilter := ""
	var timeArg interface{}
	if opts.TimeRange > 0 {
		cutoffTime := time.Now().Add(-opts.TimeRange)
		timeFilter = " AND q.timestamp >= $1"
		timeArg = cutoffTime
	}

	strategyFilter := ""
	if opts.Strategy != "" {
		if timeFilter != "" {
			strategyFilter = " AND q.strategy = $2"
		} else {
			strategyFilter = " AND q.strategy = $1"
		}
	}

	// 查询总数
	query := "SELECT COUNT(*) FROM learning_queries q WHERE 1=1" + timeFilter + strategyFilter
	args := make([]interface{}, 0)
	if timeArg != nil {
		args = append(args, timeArg)
	}
	if opts.Strategy != "" {
		args = append(args, opts.Strategy)
	}

	err := s.db.QueryRowContext(ctx, query, args...).Scan(&stats.TotalQueries)
	if err != nil {
		return nil, err
	}

	// 平均评分
	query = `
		SELECT COALESCE(AVG(rating), 0)
		FROM learning_explicit_feedback ef
		JOIN learning_queries q ON ef.query_id = q.id
		WHERE rating > 0` + timeFilter + strategyFilter

	err = s.db.QueryRowContext(ctx, query, args...).Scan(&stats.AvgRating)
	if err != nil {
		return nil, err
	}

	// 正负反馈率
	query = `
		SELECT 
			COALESCE(SUM(CASE WHEN type = 'positive' THEN 1 ELSE 0 END)::FLOAT / NULLIF(COUNT(*), 0), 0) as positive_rate,
			COALESCE(SUM(CASE WHEN type = 'negative' THEN 1 ELSE 0 END)::FLOAT / NULLIF(COUNT(*), 0), 0) as negative_rate
		FROM learning_explicit_feedback ef
		JOIN learning_queries q ON ef.query_id = q.id
		WHERE type IN ('positive', 'negative')` + timeFilter + strategyFilter

	err = s.db.QueryRowContext(ctx, query, args...).Scan(&stats.PositiveRate, &stats.NegativeRate)
	if err != nil {
		return nil, err
	}

	// 平均 CTR
	query = `
		SELECT COALESCE(AVG(click_rate), 0)
		FROM (
			SELECT 
				q.id,
				COUNT(DISTINCT CASE WHEN if.action IN ('click', 'read') THEN if.document_id END)::FLOAT / 
				NULLIF(COUNT(DISTINCT lr.document_id), 0) as click_rate
			FROM learning_queries q
			LEFT JOIN learning_results lr ON q.id = lr.query_id
			LEFT JOIN learning_implicit_feedback if ON q.id = if.query_id
			WHERE 1=1` + timeFilter + strategyFilter + `
			GROUP BY q.id
		) as ctr_table
	`

	err = s.db.QueryRowContext(ctx, query, args...).Scan(&stats.AvgCTR)
	if err != nil {
		return nil, err
	}

	// 平均阅读时长
	query = `
		SELECT COALESCE(AVG(duration_ms), 0)
		FROM learning_implicit_feedback if
		JOIN learning_queries q ON if.query_id = q.id
		WHERE action = 'read' AND duration_ms > 0` + timeFilter + strategyFilter

	var avgDurationMS float64
	err = s.db.QueryRowContext(ctx, query, args...).Scan(&avgDurationMS)
	if err != nil {
		return nil, err
	}
	stats.AvgReadDuration = time.Duration(avgDurationMS) * time.Millisecond

	// 低评分查询（如果指定了最低评分）
	if opts.MinRating > 0 {
		query = `
			SELECT DISTINCT q.text
			FROM learning_queries q
			JOIN learning_explicit_feedback ef ON q.id = ef.query_id
			WHERE ef.rating > 0` + timeFilter + strategyFilter + `
			GROUP BY q.id, q.text
			HAVING AVG(ef.rating) < $` + fmt.Sprintf("%d", len(args)+1) + `
			LIMIT 10
		`
		args = append(args, opts.MinRating)

		rows, err := s.db.QueryContext(ctx, query, args...)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		for rows.Next() {
			var queryText string
			if err := rows.Scan(&queryText); err != nil {
				return nil, err
			}
			stats.LowRatingQueries = append(stats.LowRatingQueries, queryText)
		}
	}

	return stats, nil
}

// 辅助函数

func calculateAvgRating(feedbacks []ExplicitFeedback) float64 {
	if len(feedbacks) == 0 {
		return 0
	}

	total := 0.0
	count := 0
	for _, fb := range feedbacks {
		if fb.Rating > 0 {
			total += float64(fb.Rating)
			count++
		}
	}

	if count == 0 {
		return 0
	}
	return total / float64(count)
}

func calculateCTR(results []types.Document, feedbacks []ImplicitFeedback) float64 {
	if len(results) == 0 {
		return 0
	}

	clickCount := 0
	for _, fb := range feedbacks {
		if fb.Action == ActionClick || fb.Action == ActionRead {
			clickCount++
		}
	}

	return float64(clickCount) / float64(len(results))
}

func calculateAvgReadDuration(feedbacks []ImplicitFeedback) time.Duration {
	if len(feedbacks) == 0 {
		return 0
	}

	total := time.Duration(0)
	count := 0
	for _, fb := range feedbacks {
		if fb.Action == ActionRead && fb.Duration > 0 {
			total += fb.Duration
			count++
		}
	}

	if count == 0 {
		return 0
	}
	return total / time.Duration(count)
}
