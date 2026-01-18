package memory

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/zhucl121/langchain-go/pkg/types"
)

// Note: These tests require a running PostgreSQL instance.
// Set POSTGRES_TEST_DSN environment variable or skip with:
// go test -short

func getTestPostgresConfig(t *testing.T) PostgresMemoryConfig {
	if testing.Short() {
		t.Skip("Skipping PostgreSQL tests in short mode")
	}

	// Use environment variables or default test values
	return DefaultPostgresMemoryConfig(
		"localhost",
		"postgres",
		"postgres",
		"langchain_test",
	)
}

func TestPostgresMemory_SaveAndLoad(t *testing.T) {
	config := getTestPostgresConfig(t)
	mem, err := NewPostgresMemory(config)
	if err != nil {
		t.Fatalf("Failed to create PostgresMemory: %v", err)
	}
	defer mem.Close()

	ctx := context.Background()
	sessionID := "test-session-1"

	// Clear first
	_ = mem.ClearSession(ctx, sessionID)

	// Save context
	err = mem.SaveContext(ctx,
		map[string]any{
			"session_id": sessionID,
			"input":      "Hello",
		},
		map[string]any{
			"output": "Hi there!",
		},
	)
	if err != nil {
		t.Fatalf("SaveContext failed: %v", err)
	}

	// Load memory variables
	vars, err := mem.LoadMemoryVariables(ctx, map[string]any{"session_id": sessionID})
	if err != nil {
		t.Fatalf("LoadMemoryVariables failed: %v", err)
	}

	// Check history
	history, ok := vars["history"].([]types.Message)
	if !ok {
		t.Fatal("Expected history to be []types.Message")
	}

	if len(history) != 2 {
		t.Fatalf("Expected 2 messages, got %d", len(history))
	}

	if history[0].Role != types.RoleUser || history[0].Content != "Hello" {
		t.Errorf("Unexpected first message: %+v", history[0])
	}

	if history[1].Role != types.RoleAssistant || history[1].Content != "Hi there!" {
		t.Errorf("Unexpected second message: %+v", history[1])
	}
}

func TestPostgresMemory_WindowSize(t *testing.T) {
	config := getTestPostgresConfig(t)
	config.WindowSize = 2 // Keep only 2 turns (4 messages)
	
	mem, err := NewPostgresMemory(config)
	if err != nil {
		t.Fatalf("Failed to create PostgresMemory: %v", err)
	}
	defer mem.Close()

	ctx := context.Background()
	sessionID := "test-session-window"

	// Clear first
	_ = mem.ClearSession(ctx, sessionID)

	// Save 4 turns (8 messages)
	for i := 1; i <= 4; i++ {
		err = mem.SaveContext(ctx,
			map[string]any{
				"session_id": sessionID,
				"input":      fmt.Sprintf("Message %d", i),
			},
			map[string]any{
				"output": fmt.Sprintf("Response %d", i),
			},
		)
		if err != nil {
			t.Fatalf("SaveContext failed: %v", err)
		}
	}

	// Load and check
	messages, err := mem.GetMessages(ctx, sessionID)
	if err != nil {
		t.Fatalf("GetMessages failed: %v", err)
	}

	// Should only have last 2 turns (4 messages)
	if len(messages) != 4 {
		t.Fatalf("Expected 4 messages (2 turns), got %d", len(messages))
	}

	// Check it's the most recent messages
	if messages[0].Content != "Message 3" {
		t.Errorf("Expected 'Message 3', got '%s'", messages[0].Content)
	}
}

func TestPostgresMemory_ClearSession(t *testing.T) {
	config := getTestPostgresConfig(t)
	mem, err := NewPostgresMemory(config)
	if err != nil {
		t.Fatalf("Failed to create PostgresMemory: %v", err)
	}
	defer mem.Close()

	ctx := context.Background()
	sessionID := "test-session-clear"

	// Save some messages
	_ = mem.SaveContext(ctx,
		map[string]any{"session_id": sessionID, "input": "Test"},
		map[string]any{"output": "Response"},
	)

	// Clear session
	err = mem.ClearSession(ctx, sessionID)
	if err != nil {
		t.Fatalf("ClearSession failed: %v", err)
	}

	// Verify empty
	messages, err := mem.GetMessages(ctx, sessionID)
	if err != nil {
		t.Fatalf("GetMessages failed: %v", err)
	}

	if len(messages) != 0 {
		t.Errorf("Expected 0 messages after clear, got %d", len(messages))
	}
}

func TestPostgresMemory_MultiSession(t *testing.T) {
	config := getTestPostgresConfig(t)
	mem, err := NewPostgresMemory(config)
	if err != nil {
		t.Fatalf("Failed to create PostgresMemory: %v", err)
	}
	defer mem.Close()

	ctx := context.Background()

	// Create two sessions
	sessions := []string{"session-a", "session-b"}
	for _, sid := range sessions {
		_ = mem.ClearSession(ctx, sid)
		_ = mem.SaveContext(ctx,
			map[string]any{"session_id": sid, "input": "Hello " + sid},
			map[string]any{"output": "Hi " + sid},
		)
	}

	// Verify isolation
	for _, sid := range sessions {
		messages, err := mem.GetMessages(ctx, sid)
		if err != nil {
			t.Fatalf("GetMessages failed for %s: %v", sid, err)
		}

		if len(messages) != 2 {
			t.Fatalf("Expected 2 messages for %s, got %d", sid, len(messages))
		}

		if messages[0].Content != "Hello "+sid {
			t.Errorf("Session isolation failed for %s", sid)
		}
	}
}

func TestPostgresMemory_SessionExpiration(t *testing.T) {
	config := getTestPostgresConfig(t)
	config.SessionTTL = 2 * time.Second
	
	mem, err := NewPostgresMemory(config)
	if err != nil {
		t.Fatalf("Failed to create PostgresMemory: %v", err)
	}
	defer mem.Close()

	ctx := context.Background()
	sessionID := "test-session-expire"

	// Save message
	_ = mem.ClearSession(ctx, sessionID)
	_ = mem.SaveContext(ctx,
		map[string]any{"session_id": sessionID, "input": "Test"},
		map[string]any{"output": "Response"},
	)

	// Wait for expiration
	time.Sleep(3 * time.Second)

	// Cleanup expired sessions
	count, err := mem.CleanupExpiredSessions(ctx)
	if err != nil {
		t.Fatalf("CleanupExpiredSessions failed: %v", err)
	}

	t.Logf("Cleaned up %d expired sessions", count)

	// Session should be gone
	messages, err := mem.GetMessages(ctx, sessionID)
	if err != nil {
		t.Fatalf("GetMessages failed: %v", err)
	}

	if len(messages) != 0 {
		t.Errorf("Expected session to be expired and cleaned up")
	}
}

func BenchmarkPostgresMemory_SaveContext(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping benchmark in short mode")
	}

	config := DefaultPostgresMemoryConfig(
		"localhost", "postgres", "postgres", "langchain_test",
	)
	mem, err := NewPostgresMemory(config)
	if err != nil {
		b.Fatalf("Failed to create PostgresMemory: %v", err)
	}
	defer mem.Close()

	ctx := context.Background()
	sessionID := "bench-session"
	_ = mem.ClearSession(ctx, sessionID)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = mem.SaveContext(ctx,
			map[string]any{"session_id": sessionID, "input": "Test"},
			map[string]any{"output": "Response"},
		)
	}
}

func BenchmarkPostgresMemory_LoadMemoryVariables(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping benchmark in short mode")
	}

	config := DefaultPostgresMemoryConfig(
		"localhost", "postgres", "postgres", "langchain_test",
	)
	mem, err := NewPostgresMemory(config)
	if err != nil {
		b.Fatalf("Failed to create PostgresMemory: %v", err)
	}
	defer mem.Close()

	ctx := context.Background()
	sessionID := "bench-session-load"
	
	// Prepare data
	_ = mem.ClearSession(ctx, sessionID)
	for i := 0; i < 10; i++ {
		_ = mem.SaveContext(ctx,
			map[string]any{"session_id": sessionID, "input": "Test"},
			map[string]any{"output": "Response"},
		)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = mem.LoadMemoryVariables(ctx, map[string]any{"session_id": sessionID})
	}
}
