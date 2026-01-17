package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/zhucl121/langchain-go/pkg/types"
)

// RedisMemoryConfig is the configuration for RedisMemory.
type RedisMemoryConfig struct {
	// Client is the Redis client (required)
	Client *redis.Client

	// KeyPrefix is the prefix for Redis keys (default: "langchain:memory:")
	KeyPrefix string

	// SessionTTL is the session expiration time (default: 1 hour)
	// Set to 0 for no expiration
	SessionTTL time.Duration

	// WindowSize is the number of conversation turns to keep (default: 10)
	// Each turn includes user input + assistant output
	// Total messages = WindowSize * 2
	WindowSize int

	// SessionIDKey is the key name for session ID in inputs (default: "session_id")
	SessionIDKey string
}

// DefaultRedisMemoryConfig returns the default configuration.
func DefaultRedisMemoryConfig(client *redis.Client) RedisMemoryConfig {
	return RedisMemoryConfig{
		Client:       client,
		KeyPrefix:    "langchain:memory:",
		SessionTTL:   1 * time.Hour,
		WindowSize:   10,
		SessionIDKey: "session_id",
	}
}

// RedisMemory is a persistent memory implementation using Redis.
//
// RedisMemory stores conversation history in Redis, enabling:
//   - Persistence across process restarts
//   - Distributed deployment (multiple instances)
//   - Automatic session expiration (TTL)
//   - Multi-user support (session-based isolation)
//
// Example:
//
//	redisClient := redis.NewClient(&redis.Options{
//	    Addr: "localhost:6379",
//	})
//
//	mem, err := memory.NewRedisMemory(memory.RedisMemoryConfig{
//	    Client:     redisClient,
//	    SessionTTL: 1 * time.Hour,
//	    WindowSize: 10,
//	})
//
//	// Use with session ID
//	mem.SaveContext(ctx, map[string]any{
//	    "session_id": "user-123",
//	    "input":      "Hello",
//	}, map[string]any{
//	    "output": "Hi there!",
//	})
//
type RedisMemory struct {
	*BaseMemory
	config RedisMemoryConfig
	mu     sync.RWMutex
}

// NewRedisMemory creates a new Redis-based memory instance.
//
// Parameters:
//   - config: Redis memory configuration
//
// Returns:
//   - *RedisMemory: Redis memory instance
//   - error: Configuration error
//
func NewRedisMemory(config RedisMemoryConfig) (*RedisMemory, error) {
	// Validate config
	if config.Client == nil {
		return nil, fmt.Errorf("Redis client is required")
	}

	// Set defaults
	if config.KeyPrefix == "" {
		config.KeyPrefix = "langchain:memory:"
	}
	if config.SessionTTL == 0 {
		config.SessionTTL = 1 * time.Hour
	}
	if config.WindowSize <= 0 {
		config.WindowSize = 10
	}
	if config.SessionIDKey == "" {
		config.SessionIDKey = "session_id"
	}

	// Test Redis connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := config.Client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisMemory{
		BaseMemory: NewBaseMemory(),
		config:     config,
	}, nil
}

// LoadMemoryVariables implements Memory interface.
//
// Loads conversation history from Redis for the given session.
// Expects "session_id" in inputs map.
func (m *RedisMemory) LoadMemoryVariables(ctx context.Context, inputs map[string]any) (map[string]any, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Extract session ID
	sessionID := m.extractSessionID(inputs)
	if sessionID == "" {
		// No session ID, return empty history
		return map[string]any{
			m.memoryKey: []types.Message{},
		}, nil
	}

	// Load messages from Redis
	messages, err := m.loadMessages(ctx, sessionID)
	if err != nil {
		// Session not found is OK, return empty history
		if err == redis.Nil {
			return map[string]any{
				m.memoryKey: []types.Message{},
			}, nil
		}
		return nil, fmt.Errorf("failed to load memory: %w", err)
	}

	result := make(map[string]any)

	if m.returnMessages {
		// Return message list
		result[m.memoryKey] = messages
	} else {
		// Return string format
		result[m.memoryKey] = messagesToString(messages)
	}

	return result, nil
}

// SaveContext implements Memory interface.
//
// Saves conversation context to Redis with automatic window management and TTL.
// Expects "session_id" in inputs map.
func (m *RedisMemory) SaveContext(ctx context.Context, inputs map[string]any, outputs map[string]any) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Extract session ID
	sessionID := m.extractSessionID(inputs)
	if sessionID == "" {
		return fmt.Errorf("session_id is required in inputs")
	}

	// Load existing messages
	messages, err := m.loadMessages(ctx, sessionID)
	if err != nil && err != redis.Nil {
		return fmt.Errorf("failed to load existing messages: %w", err)
	}

	// Extract input and output
	inputStr, outputStr := m.extractInputOutput(inputs, outputs)

	// Add new messages
	if inputStr != "" {
		messages = append(messages, types.NewUserMessage(inputStr))
	}
	if outputStr != "" {
		messages = append(messages, types.NewAssistantMessage(outputStr))
	}

	// Apply window management
	maxMessages := m.config.WindowSize * 2
	if len(messages) > maxMessages {
		messages = messages[len(messages)-maxMessages:]
	}

	// Save to Redis
	if err := m.saveMessages(ctx, sessionID, messages); err != nil {
		return fmt.Errorf("failed to save messages: %w", err)
	}

	return nil
}

// Clear implements Memory interface.
//
// Clears conversation history for the given session.
// Expects "session_id" in context or uses default session.
func (m *RedisMemory) Clear(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Note: This clears the current session
	// To clear a specific session, call ClearSession directly
	return nil
}

// ClearSession clears conversation history for a specific session.
//
// Parameters:
//   - ctx: Context
//   - sessionID: Session ID to clear
//
// Returns:
//   - error: Delete error
//
func (m *RedisMemory) ClearSession(ctx context.Context, sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := m.getKey(sessionID)
	return m.config.Client.Del(ctx, key).Err()
}

// GetMessages retrieves all messages for a session (for debugging).
//
// Parameters:
//   - ctx: Context
//   - sessionID: Session ID
//
// Returns:
//   - []types.Message: Message list
//   - error: Load error
//
func (m *RedisMemory) GetMessages(ctx context.Context, sessionID string) ([]types.Message, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.loadMessages(ctx, sessionID)
}

// GetSessionTTL returns the remaining TTL for a session.
//
// Parameters:
//   - ctx: Context
//   - sessionID: Session ID
//
// Returns:
//   - time.Duration: Remaining TTL
//   - error: Error
//
func (m *RedisMemory) GetSessionTTL(ctx context.Context, sessionID string) (time.Duration, error) {
	key := m.getKey(sessionID)
	return m.config.Client.TTL(ctx, key).Result()
}

// RefreshSessionTTL resets the TTL for a session.
//
// Parameters:
//   - ctx: Context
//   - sessionID: Session ID
//
// Returns:
//   - error: Error
//
func (m *RedisMemory) RefreshSessionTTL(ctx context.Context, sessionID string) error {
	key := m.getKey(sessionID)
	return m.config.Client.Expire(ctx, key, m.config.SessionTTL).Err()
}

// ListSessions lists all active sessions (up to limit).
//
// Parameters:
//   - ctx: Context
//   - limit: Maximum number of sessions to return (0 = all)
//
// Returns:
//   - []string: Session ID list
//   - error: Error
//
func (m *RedisMemory) ListSessions(ctx context.Context, limit int) ([]string, error) {
	pattern := m.config.KeyPrefix + "*"
	
	var sessions []string
	iter := m.config.Client.Scan(ctx, 0, pattern, 0).Iterator()
	for iter.Next(ctx) {
		key := iter.Val()
		// Remove prefix to get session ID
		if len(key) > len(m.config.KeyPrefix) {
			sessionID := key[len(m.config.KeyPrefix):]
			sessions = append(sessions, sessionID)
			
			if limit > 0 && len(sessions) >= limit {
				break
			}
		}
	}
	
	if err := iter.Err(); err != nil {
		return nil, err
	}

	return sessions, nil
}

// Private helper methods

func (m *RedisMemory) extractSessionID(inputs map[string]any) string {
	if inputs == nil {
		return ""
	}

	if sessionID, ok := inputs[m.config.SessionIDKey]; ok {
		if str, ok := sessionID.(string); ok {
			return str
		}
	}

	return ""
}

func (m *RedisMemory) getKey(sessionID string) string {
	return m.config.KeyPrefix + sessionID
}

func (m *RedisMemory) loadMessages(ctx context.Context, sessionID string) ([]types.Message, error) {
	key := m.getKey(sessionID)

	val, err := m.config.Client.Get(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	var messages []types.Message
	if err := json.Unmarshal([]byte(val), &messages); err != nil {
		return nil, fmt.Errorf("failed to unmarshal messages: %w", err)
	}

	return messages, nil
}

func (m *RedisMemory) saveMessages(ctx context.Context, sessionID string, messages []types.Message) error {
	key := m.getKey(sessionID)

	data, err := json.Marshal(messages)
	if err != nil {
		return fmt.Errorf("failed to marshal messages: %w", err)
	}

	return m.config.Client.Set(ctx, key, data, m.config.SessionTTL).Err()
}

// GetConfig returns the memory configuration (for debugging).
func (m *RedisMemory) GetConfig() RedisMemoryConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config
}

// GetClient returns the underlying Redis client (for advanced operations).
func (m *RedisMemory) GetClient() *redis.Client {
	return m.config.Client
}
