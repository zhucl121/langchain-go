// Package memory provides persistent memory implementations for LangChain-Go.
//
// This package extends the core memory module with Redis-based persistent storage,
// enabling distributed deployments and production-ready memory management.
//
// # Redis Memory
//
// RedisMemory provides a persistent, distributed memory implementation using Redis.
// It's ideal for production environments where:
//   - Memory needs to survive process restarts
//   - Multiple instances need to share conversation history
//   - Automatic expiration (TTL) is required
//   - Distributed deployment is necessary
//
// # Quick Start
//
//	import (
//	    "github.com/redis/go-redis/v9"
//	    "github.com/zhucl121/langchain-go/core/memory"
//	)
//
//	// Create Redis client
//	redisClient := redis.NewClient(&redis.Options{
//	    Addr: "localhost:6379",
//	})
//
//	// Create RedisMemory
//	mem, err := memory.NewRedisMemory(memory.RedisMemoryConfig{
//	    Client:     redisClient,
//	    KeyPrefix:  "chat:",
//	    SessionTTL: 1 * time.Hour,
//	    WindowSize: 10,
//	})
//
//	// Use with Agent or Chain
//	agent := agents.NewReActAgent(llm, tools, agents.ReActConfig{
//	    Memory: mem,
//	})
//
// # Features
//
//   - Persistent storage in Redis
//   - Automatic session expiration (TTL)
//   - Session-based isolation (multi-user support)
//   - Sliding window memory management
//   - Thread-safe operations
//   - Distributed deployment ready
//
// # Session Management
//
// RedisMemory uses session IDs to isolate conversations:
//
//	// Save context with session ID
//	mem.SaveContext(ctx, map[string]any{
//	    "session_id": "user-123",
//	    "input":      "Hello",
//	}, map[string]any{
//	    "output": "Hi!",
//	})
//
//	// Load memory for specific session
//	vars, _ := mem.LoadMemoryVariables(ctx, map[string]any{
//	    "session_id": "user-123",
//	})
//
// # Window Management
//
// RedisMemory automatically maintains a sliding window:
//
//	config := memory.RedisMemoryConfig{
//	    WindowSize: 5, // Keep last 5 conversation turns (10 messages)
//	}
//
// # TTL Management
//
// Sessions automatically expire after the configured TTL:
//
//	config := memory.RedisMemoryConfig{
//	    SessionTTL: 1 * time.Hour, // Sessions expire after 1 hour
//	}
//
// # Best Practices
//
//  1. Always use session IDs in production
//  2. Set appropriate TTL based on your use case
//  3. Use a dedicated Redis instance for memory storage
//  4. Monitor Redis memory usage
//  5. Consider Redis persistence settings for critical data
//
// # Performance Considerations
//
//   - Each LoadMemoryVariables makes a Redis GET call
//   - Each SaveContext makes a Redis SET call
//   - Use connection pooling (Redis client handles this)
//   - Consider Redis Cluster for high availability
//
// # Error Handling
//
// Common errors:
//   - Redis connection failed: Check Redis is running
//   - Session not found: Normal for new sessions
//   - TTL expired: Session was cleaned up automatically
package memory
