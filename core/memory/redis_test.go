package memory

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/zhucl121/langchain-go/pkg/types"
)

// Helper function to create test Redis client
func createTestRedisClient(t *testing.T) *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   15, // Use DB 15 for tests
	})

	// Test connection
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skip("Redis not available, skipping tests")
	}

	// Clear test DB
	client.FlushDB(ctx)

	return client
}

func TestNewRedisMemory(t *testing.T) {
	client := createTestRedisClient(t)
	defer client.Close()

	tests := []struct {
		name    string
		config  RedisMemoryConfig
		wantErr bool
	}{
		{
			name: "valid config with all fields",
			config: RedisMemoryConfig{
				Client:       client,
				KeyPrefix:    "test:",
				SessionTTL:   1 * time.Hour,
				WindowSize:   5,
				SessionIDKey: "session_id",
			},
			wantErr: false,
		},
		{
			name: "valid config with defaults",
			config: RedisMemoryConfig{
				Client: client,
			},
			wantErr: false,
		},
		{
			name: "missing client",
			config: RedisMemoryConfig{
				KeyPrefix: "test:",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mem, err := NewRedisMemory(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewRedisMemory() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && mem == nil {
				t.Error("NewRedisMemory() returned nil memory")
			}
		})
	}
}

func TestRedisMemory_SaveAndLoad(t *testing.T) {
	client := createTestRedisClient(t)
	defer client.Close()

	mem, err := NewRedisMemory(RedisMemoryConfig{
		Client:     client,
		KeyPrefix:  "test:",
		SessionTTL: 1 * time.Hour,
		WindowSize: 10,
	})
	if err != nil {
		t.Fatalf("Failed to create RedisMemory: %v", err)
	}

	ctx := context.Background()
	sessionID := "test-session-1"

	// Save first conversation turn
	err = mem.SaveContext(ctx, map[string]any{
		"session_id": sessionID,
		"input":      "Hello",
	}, map[string]any{
		"output": "Hi there!",
	})
	if err != nil {
		t.Fatalf("SaveContext() error = %v", err)
	}

	// Load memory
	vars, err := mem.LoadMemoryVariables(ctx, map[string]any{
		"session_id": sessionID,
	})
	if err != nil {
		t.Fatalf("LoadMemoryVariables() error = %v", err)
	}

	// Check history
	history, ok := vars["history"].([]types.Message)
	if !ok {
		t.Fatal("history is not []types.Message")
	}

	if len(history) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(history))
	}

	if history[0].Role != types.RoleUser || history[0].Content != "Hello" {
		t.Errorf("First message incorrect: %+v", history[0])
	}

	if history[1].Role != types.RoleAssistant || history[1].Content != "Hi there!" {
		t.Errorf("Second message incorrect: %+v", history[1])
	}
}

func TestRedisMemory_MultipleConversations(t *testing.T) {
	client := createTestRedisClient(t)
	defer client.Close()

	mem, err := NewRedisMemory(RedisMemoryConfig{
		Client:     client,
		SessionTTL: 1 * time.Hour,
		WindowSize: 10,
	})
	if err != nil {
		t.Fatalf("Failed to create RedisMemory: %v", err)
	}

	ctx := context.Background()

	// Save multiple conversation turns
	conversations := []struct {
		input  string
		output string
	}{
		{"Hello", "Hi!"},
		{"How are you?", "I'm good, thanks!"},
		{"What's the weather?", "It's sunny today."},
	}

	sessionID := "test-session-2"
	for _, conv := range conversations {
		err = mem.SaveContext(ctx, map[string]any{
			"session_id": sessionID,
			"input":      conv.input,
		}, map[string]any{
			"output": conv.output,
		})
		if err != nil {
			t.Fatalf("SaveContext() error = %v", err)
		}
	}

	// Load and verify
	vars, err := mem.LoadMemoryVariables(ctx, map[string]any{
		"session_id": sessionID,
	})
	if err != nil {
		t.Fatalf("LoadMemoryVariables() error = %v", err)
	}

	history := vars["history"].([]types.Message)
	if len(history) != 6 { // 3 turns * 2 messages
		t.Errorf("Expected 6 messages, got %d", len(history))
	}
}

func TestRedisMemory_WindowManagement(t *testing.T) {
	client := createTestRedisClient(t)
	defer client.Close()

	mem, err := NewRedisMemory(RedisMemoryConfig{
		Client:     client,
		WindowSize: 2, // Keep only 2 turns (4 messages)
	})
	if err != nil {
		t.Fatalf("Failed to create RedisMemory: %v", err)
	}

	ctx := context.Background()
	sessionID := "test-session-3"

	// Save 5 conversation turns (10 messages)
	for i := 0; i < 5; i++ {
		err = mem.SaveContext(ctx, map[string]any{
			"session_id": sessionID,
			"input":      "Question",
		}, map[string]any{
			"output": "Answer",
		})
		if err != nil {
			t.Fatalf("SaveContext() error = %v", err)
		}
	}

	// Load and verify window size
	vars, err := mem.LoadMemoryVariables(ctx, map[string]any{
		"session_id": sessionID,
	})
	if err != nil {
		t.Fatalf("LoadMemoryVariables() error = %v", err)
	}

	history := vars["history"].([]types.Message)
	// Should only keep last 2 turns (4 messages)
	if len(history) != 4 {
		t.Errorf("Expected 4 messages (2 turns), got %d", len(history))
	}
}

func TestRedisMemory_SessionIsolation(t *testing.T) {
	client := createTestRedisClient(t)
	defer client.Close()

	mem, err := NewRedisMemory(RedisMemoryConfig{
		Client: client,
	})
	if err != nil {
		t.Fatalf("Failed to create RedisMemory: %v", err)
	}

	ctx := context.Background()

	// Save to different sessions
	session1 := "user-1"
	session2 := "user-2"

	mem.SaveContext(ctx, map[string]any{
		"session_id": session1,
		"input":      "Session 1 message",
	}, map[string]any{
		"output": "Session 1 response",
	})

	mem.SaveContext(ctx, map[string]any{
		"session_id": session2,
		"input":      "Session 2 message",
	}, map[string]any{
		"output": "Session 2 response",
	})

	// Load session 1
	vars1, _ := mem.LoadMemoryVariables(ctx, map[string]any{
		"session_id": session1,
	})
	history1 := vars1["history"].([]types.Message)

	// Load session 2
	vars2, _ := mem.LoadMemoryVariables(ctx, map[string]any{
		"session_id": session2,
	})
	history2 := vars2["history"].([]types.Message)

	// Verify isolation
	if history1[0].Content != "Session 1 message" {
		t.Error("Session 1 contaminated")
	}
	if history2[0].Content != "Session 2 message" {
		t.Error("Session 2 contaminated")
	}
}

func TestRedisMemory_ClearSession(t *testing.T) {
	client := createTestRedisClient(t)
	defer client.Close()

	mem, err := NewRedisMemory(RedisMemoryConfig{
		Client: client,
	})
	if err != nil {
		t.Fatalf("Failed to create RedisMemory: %v", err)
	}

	ctx := context.Background()
	sessionID := "test-session-4"

	// Save some data
	mem.SaveContext(ctx, map[string]any{
		"session_id": sessionID,
		"input":      "Test",
	}, map[string]any{
		"output": "Response",
	})

	// Clear session
	err = mem.ClearSession(ctx, sessionID)
	if err != nil {
		t.Fatalf("ClearSession() error = %v", err)
	}

	// Verify cleared
	vars, _ := mem.LoadMemoryVariables(ctx, map[string]any{
		"session_id": sessionID,
	})
	history := vars["history"].([]types.Message)
	if len(history) != 0 {
		t.Errorf("Expected empty history after clear, got %d messages", len(history))
	}
}

func TestRedisMemory_TTL(t *testing.T) {
	client := createTestRedisClient(t)
	defer client.Close()

	mem, err := NewRedisMemory(RedisMemoryConfig{
		Client:     client,
		SessionTTL: 2 * time.Second, // Short TTL for testing
	})
	if err != nil {
		t.Fatalf("Failed to create RedisMemory: %v", err)
	}

	ctx := context.Background()
	sessionID := "test-session-5"

	// Save data
	mem.SaveContext(ctx, map[string]any{
		"session_id": sessionID,
		"input":      "Test",
	}, map[string]any{
		"output": "Response",
	})

	// Check TTL is set
	ttl, err := mem.GetSessionTTL(ctx, sessionID)
	if err != nil {
		t.Fatalf("GetSessionTTL() error = %v", err)
	}
	if ttl <= 0 {
		t.Error("Expected positive TTL")
	}

	// Wait for expiration
	time.Sleep(3 * time.Second)

	// Verify expired
	vars, _ := mem.LoadMemoryVariables(ctx, map[string]any{
		"session_id": sessionID,
	})
	history := vars["history"].([]types.Message)
	if len(history) != 0 {
		t.Errorf("Expected empty history after TTL expiration, got %d messages", len(history))
	}
}

func TestRedisMemory_ReturnStringFormat(t *testing.T) {
	client := createTestRedisClient(t)
	defer client.Close()

	mem, err := NewRedisMemory(RedisMemoryConfig{
		Client: client,
	})
	if err != nil {
		t.Fatalf("Failed to create RedisMemory: %v", err)
	}

	// Set to return string format
	mem.SetReturnMessages(false)

	ctx := context.Background()
	sessionID := "test-session-6"

	// Save data
	mem.SaveContext(ctx, map[string]any{
		"session_id": sessionID,
		"input":      "Hello",
	}, map[string]any{
		"output": "Hi!",
	})

	// Load as string
	vars, _ := mem.LoadMemoryVariables(ctx, map[string]any{
		"session_id": sessionID,
	})

	historyStr, ok := vars["history"].(string)
	if !ok {
		t.Fatal("history is not string")
	}

	if historyStr == "" {
		t.Error("Expected non-empty history string")
	}
}

func TestRedisMemory_ListSessions(t *testing.T) {
	client := createTestRedisClient(t)
	defer client.Close()

	mem, err := NewRedisMemory(RedisMemoryConfig{
		Client: client,
	})
	if err != nil {
		t.Fatalf("Failed to create RedisMemory: %v", err)
	}

	ctx := context.Background()

	// Create multiple sessions
	sessions := []string{"user-1", "user-2", "user-3"}
	for _, sessionID := range sessions {
		mem.SaveContext(ctx, map[string]any{
			"session_id": sessionID,
			"input":      "Test",
		}, map[string]any{
			"output": "Response",
		})
	}

	// List sessions
	list, err := mem.ListSessions(ctx, 0)
	if err != nil {
		t.Fatalf("ListSessions() error = %v", err)
	}

	if len(list) != 3 {
		t.Errorf("Expected 3 sessions, got %d", len(list))
	}
}

func TestRedisMemory_RefreshTTL(t *testing.T) {
	client := createTestRedisClient(t)
	defer client.Close()

	mem, err := NewRedisMemory(RedisMemoryConfig{
		Client:     client,
		SessionTTL: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("Failed to create RedisMemory: %v", err)
	}

	ctx := context.Background()
	sessionID := "test-session-7"

	// Save data
	mem.SaveContext(ctx, map[string]any{
		"session_id": sessionID,
		"input":      "Test",
	}, map[string]any{
		"output": "Response",
	})

	// Wait a bit
	time.Sleep(2 * time.Second)

	// Refresh TTL
	err = mem.RefreshSessionTTL(ctx, sessionID)
	if err != nil {
		t.Fatalf("RefreshSessionTTL() error = %v", err)
	}

	// Check TTL is refreshed
	ttl, _ := mem.GetSessionTTL(ctx, sessionID)
	if ttl < 4*time.Second {
		t.Errorf("Expected TTL to be refreshed to ~5s, got %v", ttl)
	}
}

// Benchmark tests

func BenchmarkRedisMemory_SaveContext(b *testing.B) {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   15,
	})
	defer client.Close()

	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		b.Skip("Redis not available")
	}

	mem, _ := NewRedisMemory(RedisMemoryConfig{
		Client: client,
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mem.SaveContext(ctx, map[string]any{
			"session_id": "bench-session",
			"input":      "test input",
		}, map[string]any{
			"output": "test output",
		})
	}
}

func BenchmarkRedisMemory_LoadMemoryVariables(b *testing.B) {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   15,
	})
	defer client.Close()

	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		b.Skip("Redis not available")
	}

	mem, _ := NewRedisMemory(RedisMemoryConfig{
		Client: client,
	})

	// Prepare data
	mem.SaveContext(ctx, map[string]any{
		"session_id": "bench-session",
		"input":      "test",
	}, map[string]any{
		"output": "response",
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mem.LoadMemoryVariables(ctx, map[string]any{
			"session_id": "bench-session",
		})
	}
}
