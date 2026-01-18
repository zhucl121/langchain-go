package memory

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql" // MySQL driver
	"github.com/zhucl121/langchain-go/pkg/types"
)

// MySQLMemoryConfig is the configuration for MySQLMemory.
type MySQLMemoryConfig struct {
	// Connection parameters (required)
	Host     string
	Port     int
	User     string
	Password string
	Database string

	// Connection pool settings
	MaxOpenConns int           // default: 10
	MaxIdleConns int           // default: 5
	ConnTimeout  time.Duration // default: 10s

	// Session settings
	SessionTTL time.Duration // default: 1 hour, 0 = no expiration
	WindowSize int           // default: 10 turns

	// Key names
	SessionIDKey string // default: "session_id"
}

// DefaultMySQLMemoryConfig returns the default configuration.
func DefaultMySQLMemoryConfig(host, user, password, database string) MySQLMemoryConfig {
	return MySQLMemoryConfig{
		Host:         host,
		Port:         3306,
		User:         user,
		Password:     password,
		Database:     database,
		MaxOpenConns: 10,
		MaxIdleConns: 5,
		ConnTimeout:  10 * time.Second,
		SessionTTL:   1 * time.Hour,
		WindowSize:   10,
		SessionIDKey: "session_id",
	}
}

// MySQLMemory is a persistent memory implementation using MySQL.
//
// MySQLMemory stores conversation history in MySQL, enabling:
//   - Durable persistence
//   - Wide deployment support
//   - JSON column support (MySQL 5.7+)
//   - Mature tooling ecosystem
//
// Example:
//
//	config := memory.DefaultMySQLMemoryConfig(
//	    "localhost", "user", "password", "dbname",
//	)
//	mem, err := memory.NewMySQLMemory(config)
//
//	// Use with session ID
//	mem.SaveContext(ctx, map[string]any{
//	    "session_id": "user-123",
//	    "input":      "Hello",
//	}, map[string]any{
//	    "output": "Hi there!",
//	})
type MySQLMemory struct {
	*BaseMemory
	db     *sql.DB
	config MySQLMemoryConfig
}

// NewMySQLMemory creates a new MySQL-based memory instance.
//
// Parameters:
//   - config: MySQL memory configuration
//
// Returns:
//   - *MySQLMemory: MySQL memory instance
//   - error: Connection or initialization error
func NewMySQLMemory(config MySQLMemoryConfig) (*MySQLMemory, error) {
	// Build DSN
	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		config.User, config.Password, config.Host, config.Port, config.Database,
	)

	// Connect to database
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open mysql connection: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(config.MaxOpenConns)
	db.SetMaxIdleConns(config.MaxIdleConns)
	db.SetConnMaxLifetime(time.Hour)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), config.ConnTimeout)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping mysql: %w", err)
	}

	mem := &MySQLMemory{
		BaseMemory: NewBaseMemory(),
		db:         db,
		config:     config,
	}

	// Initialize schema
	if err := mem.initSchema(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return mem, nil
}

// initSchema initializes the database schema.
func (m *MySQLMemory) initSchema(ctx context.Context) error {
	schemas := []string{
		// Sessions table
		`CREATE TABLE IF NOT EXISTS langchain_sessions (
			session_id VARCHAR(255) PRIMARY KEY,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			expires_at TIMESTAMP NULL,
			INDEX idx_sessions_expires_at (expires_at)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,

		// Messages table
		`CREATE TABLE IF NOT EXISTS langchain_messages (
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
			session_id VARCHAR(255) NOT NULL,
			role VARCHAR(50) NOT NULL,
			content TEXT NOT NULL,
			metadata JSON,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (session_id) REFERENCES langchain_sessions(session_id) ON DELETE CASCADE,
			INDEX idx_messages_session_id (session_id),
			INDEX idx_messages_created_at (created_at)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
	}

	for _, schema := range schemas {
		if _, err := m.db.ExecContext(ctx, schema); err != nil {
			return fmt.Errorf("failed to execute schema: %w", err)
		}
	}

	return nil
}

// LoadMemoryVariables loads memory variables for the given session.
func (m *MySQLMemory) LoadMemoryVariables(ctx context.Context, inputs map[string]any) (map[string]any, error) {
	sessionID := m.getSessionID(inputs)
	if sessionID == "" {
		return map[string]any{m.GetMemoryKey(): []types.Message{}}, nil
	}

	// Get messages
	messages, err := m.getMessages(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to load messages: %w", err)
	}

	// Return as messages or string
	if m.GetReturnMessages() {
		return map[string]any{m.GetMemoryKey(): messages}, nil
	}
	return map[string]any{m.GetMemoryKey(): messagesToString(messages)}, nil
}

// SaveContext saves the conversation context.
func (m *MySQLMemory) SaveContext(ctx context.Context, inputs map[string]any, outputs map[string]any) error {
	sessionID := m.getSessionID(inputs)
	if sessionID == "" {
		return fmt.Errorf("session_id is required")
	}

	// Ensure session exists
	if err := m.ensureSession(ctx, sessionID); err != nil {
		return fmt.Errorf("failed to ensure session: %w", err)
	}

	// Extract input and output
	inputStr, outputStr := m.extractInputOutput(inputs, outputs)

	// Save user message
	if inputStr != "" {
		if err := m.saveMessage(ctx, sessionID, types.RoleUser, inputStr); err != nil {
			return err
		}
	}

	// Save assistant message
	if outputStr != "" {
		if err := m.saveMessage(ctx, sessionID, types.RoleAssistant, outputStr); err != nil {
			return err
		}
	}

	// Trim to window size
	if err := m.trimMessages(ctx, sessionID); err != nil {
		return fmt.Errorf("failed to trim messages: %w", err)
	}

	return nil
}

// Clear clears all messages for the given session.
func (m *MySQLMemory) Clear(ctx context.Context) error {
	// Note: This clears ALL sessions. For session-specific clear,
	// use ClearSession instead.
	_, err := m.db.ExecContext(ctx, "DELETE FROM langchain_sessions")
	return err
}

// ClearSession clears messages for a specific session.
func (m *MySQLMemory) ClearSession(ctx context.Context, sessionID string) error {
	query := "DELETE FROM langchain_sessions WHERE session_id = ?"
	_, err := m.db.ExecContext(ctx, query, sessionID)
	return err
}

// GetMessages retrieves all messages for a session.
func (m *MySQLMemory) GetMessages(ctx context.Context, sessionID string) ([]types.Message, error) {
	return m.getMessages(ctx, sessionID)
}

// Close closes the database connection.
func (m *MySQLMemory) Close() error {
	return m.db.Close()
}

// Private helper methods

func (m *MySQLMemory) getSessionID(inputs map[string]any) string {
	if sessionID, ok := inputs[m.config.SessionIDKey]; ok {
		if sid, ok := sessionID.(string); ok {
			return sid
		}
	}
	return ""
}

func (m *MySQLMemory) ensureSession(ctx context.Context, sessionID string) error {
	var expiresAt *time.Time
	if m.config.SessionTTL > 0 {
		expires := time.Now().Add(m.config.SessionTTL)
		expiresAt = &expires
	}

	query := `
		INSERT INTO langchain_sessions (session_id, expires_at)
		VALUES (?, ?)
		ON DUPLICATE KEY UPDATE
			updated_at = CURRENT_TIMESTAMP,
			expires_at = VALUES(expires_at)
	`

	_, err := m.db.ExecContext(ctx, query, sessionID, expiresAt)
	return err
}

func (m *MySQLMemory) saveMessage(ctx context.Context, sessionID string, role types.Role, content string) error {
	query := `
		INSERT INTO langchain_messages (session_id, role, content)
		VALUES (?, ?, ?)
	`

	_, err := m.db.ExecContext(ctx, query, sessionID, string(role), content)
	return err
}

func (m *MySQLMemory) getMessages(ctx context.Context, sessionID string) ([]types.Message, error) {
	query := `
		SELECT role, content, metadata, created_at
		FROM langchain_messages
		WHERE session_id = ?
		ORDER BY created_at ASC
	`

	rows, err := m.db.QueryContext(ctx, query, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []types.Message
	for rows.Next() {
		var msg types.Message
		var roleStr string
		var metadataJSON []byte
		var createdAt time.Time

		if err := rows.Scan(&roleStr, &msg.Content, &metadataJSON, &createdAt); err != nil {
			return nil, err
		}

		msg.Role = types.Role(roleStr)

		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &msg.Metadata); err != nil {
				// Ignore metadata unmarshal errors
				msg.Metadata = nil
			}
		}

		messages = append(messages, msg)
	}

	return messages, rows.Err()
}

func (m *MySQLMemory) trimMessages(ctx context.Context, sessionID string) error {
	// Keep only the most recent WindowSize turns (WindowSize * 2 messages)
	maxMessages := m.config.WindowSize * 2

	query := `
		DELETE FROM langchain_messages
		WHERE session_id = ?
		AND id NOT IN (
			SELECT id FROM (
				SELECT id FROM langchain_messages
				WHERE session_id = ?
				ORDER BY created_at DESC
				LIMIT ?
			) AS recent_messages
		)
	`

	_, err := m.db.ExecContext(ctx, query, sessionID, sessionID, maxMessages)
	return err
}

// CleanupExpiredSessions removes expired sessions (can be called periodically).
func (m *MySQLMemory) CleanupExpiredSessions(ctx context.Context) (int64, error) {
	query := `
		DELETE FROM langchain_sessions
		WHERE expires_at IS NOT NULL AND expires_at < NOW()
	`

	result, err := m.db.ExecContext(ctx, query)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
