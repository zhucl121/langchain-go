package memory

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "github.com/lib/pq" // PostgreSQL driver
	"github.com/zhucl121/langchain-go/pkg/types"
)

// PostgresMemoryConfig is the configuration for PostgresMemory.
type PostgresMemoryConfig struct {
	// Connection parameters (required)
	Host     string
	Port     int
	User     string
	Password string
	Database string
	SSLMode  string // disable, require, verify-ca, verify-full

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

// DefaultPostgresMemoryConfig returns the default configuration.
func DefaultPostgresMemoryConfig(host, user, password, database string) PostgresMemoryConfig {
	return PostgresMemoryConfig{
		Host:         host,
		Port:         5432,
		User:         user,
		Password:     password,
		Database:     database,
		SSLMode:      "disable",
		MaxOpenConns: 10,
		MaxIdleConns: 5,
		ConnTimeout:  10 * time.Second,
		SessionTTL:   1 * time.Hour,
		WindowSize:   10,
		SessionIDKey: "session_id",
	}
}

// PostgresMemory is a persistent memory implementation using PostgreSQL.
//
// PostgresMemory stores conversation history in PostgreSQL, enabling:
//   - Durable persistence
//   - Rich querying capabilities (using JSONB)
//   - Transaction support
//   - Full SQL features
//
// The schema is automatically created on first use:
//   - sessions table: stores session metadata
//   - messages table: stores conversation messages
//
// Example:
//
//	config := memory.DefaultPostgresMemoryConfig(
//	    "localhost", "user", "password", "dbname",
//	)
//	mem, err := memory.NewPostgresMemory(config)
//
//	// Use with session ID
//	mem.SaveContext(ctx, map[string]any{
//	    "session_id": "user-123",
//	    "input":      "Hello",
//	}, map[string]any{
//	    "output": "Hi there!",
//	})
type PostgresMemory struct {
	*BaseMemory
	db     *sql.DB
	config PostgresMemoryConfig
}

// NewPostgresMemory creates a new PostgreSQL-based memory instance.
//
// Parameters:
//   - config: PostgreSQL memory configuration
//
// Returns:
//   - *PostgresMemory: PostgreSQL memory instance
//   - error: Connection or initialization error
func NewPostgresMemory(config PostgresMemoryConfig) (*PostgresMemory, error) {
	// Build connection string
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		config.Host, config.Port, config.User, config.Password,
		config.Database, config.SSLMode,
	)

	// Connect to database
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open postgres connection: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(config.MaxOpenConns)
	db.SetMaxIdleConns(config.MaxIdleConns)
	db.SetConnMaxLifetime(time.Hour)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), config.ConnTimeout)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping postgres: %w", err)
	}

	mem := &PostgresMemory{
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
func (m *PostgresMemory) initSchema(ctx context.Context) error {
	schema := `
	-- Sessions table
	CREATE TABLE IF NOT EXISTS langchain_sessions (
		session_id VARCHAR(255) PRIMARY KEY,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		expires_at TIMESTAMP
	);

	-- Messages table
	CREATE TABLE IF NOT EXISTS langchain_messages (
		id SERIAL PRIMARY KEY,
		session_id VARCHAR(255) NOT NULL,
		role VARCHAR(50) NOT NULL,
		content TEXT NOT NULL,
		metadata JSONB,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (session_id) REFERENCES langchain_sessions(session_id) ON DELETE CASCADE
	);

	-- Indexes
	CREATE INDEX IF NOT EXISTS idx_messages_session_id ON langchain_messages(session_id);
	CREATE INDEX IF NOT EXISTS idx_messages_created_at ON langchain_messages(created_at);
	CREATE INDEX IF NOT EXISTS idx_sessions_expires_at ON langchain_sessions(expires_at);
	`

	_, err := m.db.ExecContext(ctx, schema)
	return err
}

// LoadMemoryVariables loads memory variables for the given session.
func (m *PostgresMemory) LoadMemoryVariables(ctx context.Context, inputs map[string]any) (map[string]any, error) {
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
func (m *PostgresMemory) SaveContext(ctx context.Context, inputs map[string]any, outputs map[string]any) error {
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
func (m *PostgresMemory) Clear(ctx context.Context) error {
	// Note: This clears ALL sessions. For session-specific clear,
	// use ClearSession instead.
	_, err := m.db.ExecContext(ctx, "TRUNCATE TABLE langchain_sessions CASCADE")
	return err
}

// ClearSession clears messages for a specific session.
func (m *PostgresMemory) ClearSession(ctx context.Context, sessionID string) error {
	query := "DELETE FROM langchain_sessions WHERE session_id = $1"
	_, err := m.db.ExecContext(ctx, query, sessionID)
	return err
}

// GetMessages retrieves all messages for a session.
func (m *PostgresMemory) GetMessages(ctx context.Context, sessionID string) ([]types.Message, error) {
	return m.getMessages(ctx, sessionID)
}

// Close closes the database connection.
func (m *PostgresMemory) Close() error {
	return m.db.Close()
}

// Private helper methods

func (m *PostgresMemory) getSessionID(inputs map[string]any) string {
	if sessionID, ok := inputs[m.config.SessionIDKey]; ok {
		if sid, ok := sessionID.(string); ok {
			return sid
		}
	}
	return ""
}

func (m *PostgresMemory) ensureSession(ctx context.Context, sessionID string) error {
	var expiresAt *time.Time
	if m.config.SessionTTL > 0 {
		expires := time.Now().Add(m.config.SessionTTL)
		expiresAt = &expires
	}

	query := `
		INSERT INTO langchain_sessions (session_id, expires_at)
		VALUES ($1, $2)
		ON CONFLICT (session_id) DO UPDATE
		SET updated_at = CURRENT_TIMESTAMP,
		    expires_at = EXCLUDED.expires_at
	`

	_, err := m.db.ExecContext(ctx, query, sessionID, expiresAt)
	return err
}

func (m *PostgresMemory) saveMessage(ctx context.Context, sessionID string, role types.Role, content string) error {
	query := `
		INSERT INTO langchain_messages (session_id, role, content)
		VALUES ($1, $2, $3)
	`

	_, err := m.db.ExecContext(ctx, query, sessionID, string(role), content)
	return err
}

func (m *PostgresMemory) getMessages(ctx context.Context, sessionID string) ([]types.Message, error) {
	query := `
		SELECT role, content, metadata, created_at
		FROM langchain_messages
		WHERE session_id = $1
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

func (m *PostgresMemory) trimMessages(ctx context.Context, sessionID string) error {
	// Keep only the most recent WindowSize turns (WindowSize * 2 messages)
	maxMessages := m.config.WindowSize * 2

	query := `
		DELETE FROM langchain_messages
		WHERE session_id = $1
		AND id NOT IN (
			SELECT id FROM langchain_messages
			WHERE session_id = $1
			ORDER BY created_at DESC
			LIMIT $2
		)
	`

	_, err := m.db.ExecContext(ctx, query, sessionID, maxMessages)
	return err
}

// CleanupExpiredSessions removes expired sessions (can be called periodically).
func (m *PostgresMemory) CleanupExpiredSessions(ctx context.Context) (int64, error) {
	query := `
		DELETE FROM langchain_sessions
		WHERE expires_at IS NOT NULL AND expires_at < CURRENT_TIMESTAMP
	`

	result, err := m.db.ExecContext(ctx, query)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
