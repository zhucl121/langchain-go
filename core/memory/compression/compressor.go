package compression

import (
	"context"
	"fmt"
	"strings"

	"github.com/zhucl121/langchain-go/core/chat"
	"github.com/zhucl121/langchain-go/pkg/types"
)

// Strategy defines the compression strategy type.
type Strategy string

const (
	// StrategySlidingWindow uses a sliding window approach (simple truncation).
	StrategySlidingWindow Strategy = "sliding_window"

	// StrategyLLMSummary uses LLM to generate summaries (intelligent compression).
	StrategyLLMSummary Strategy = "llm_summary"

	// StrategyHybrid combines sliding window and LLM summary with fallback.
	StrategyHybrid Strategy = "hybrid"

	// StrategyTokenAware dynamically adjusts based on token count.
	StrategyTokenAware Strategy = "token_aware"
)

// Compressor is the interface for context compression.
type Compressor interface {
	// Compress compresses the message list.
	Compress(ctx context.Context, messages []types.Message) ([]types.Message, *Stats, error)

	// EstimateTokens estimates the token count for messages.
	EstimateTokens(messages []types.Message) int

	// ShouldCompress determines if compression is needed.
	ShouldCompress(messages []types.Message) bool
}

// Stats contains compression statistics.
type Stats struct {
	OriginalCount    int     // Original message count
	CompressedCount  int     // Compressed message count
	OriginalTokens   int     // Original token count
	CompressedTokens int     // Compressed token count
	CompressionRatio float64 // Compression ratio (compressed/original)
	Strategy         Strategy // Strategy used
}

// Config is the compression configuration.
type Config struct {
	Strategy       Strategy          // Compression strategy
	MaxTokens      int               // Maximum token count
	MaxMessages    int               // Maximum message count
	WindowSize     int               // Sliding window size
	SummaryPrompt  string            // Prompt for LLM summary
	PreserveRecent int               // Number of recent messages to preserve
	ChatModel      chat.ChatModel    // ChatModel for LLM summary
}

// DefaultConfig returns the default configuration.
func DefaultConfig() *Config {
	return &Config{
		Strategy:       StrategyHybrid,
		MaxTokens:      2000,
		MaxMessages:    20,
		WindowSize:     10,
		PreserveRecent: 5,
		SummaryPrompt: `Please summarize the following conversation history, preserving key information:

%s

Provide a concise summary of the core content, maintaining important context.`,
	}
}

// NewCompressor creates a new compressor based on the strategy.
func NewCompressor(config *Config) (Compressor, error) {
	if config == nil {
		config = DefaultConfig()
	}

	switch config.Strategy {
	case StrategySlidingWindow:
		return &SlidingWindowCompressor{config: config}, nil
	case StrategyLLMSummary:
		if config.ChatModel == nil {
			return nil, fmt.Errorf("ChatModel is required for LLM summary strategy")
		}
		return &LLMSummaryCompressor{config: config}, nil
	case StrategyHybrid:
		return &HybridCompressor{config: config}, nil
	case StrategyTokenAware:
		return &TokenAwareCompressor{config: config}, nil
	default:
		return nil, fmt.Errorf("unknown compression strategy: %s", config.Strategy)
	}
}

// EstimateTokens estimates token count for text (simple approximation).
//
// Estimation: ~1.5 tokens per character for mixed Chinese/English text.
// This is a rough estimate. For accurate counts, use a tokenizer.
func EstimateTokens(text string) int {
	return len([]rune(text)) * 3 / 2
}

// EstimateMessagesTokens estimates token count for a message list.
func EstimateMessagesTokens(messages []types.Message) int {
	total := 0
	for _, msg := range messages {
		total += EstimateTokens(msg.Content)
		total += 4 // Role label overhead
	}
	return total
}

// MessagesToString converts messages to string format.
func MessagesToString(messages []types.Message) string {
	var result strings.Builder
	for _, msg := range messages {
		var prefix string
		switch msg.Role {
		case types.RoleUser:
			prefix = "Human"
		case types.RoleAssistant:
			prefix = "AI"
		case types.RoleSystem:
			prefix = "System"
		case types.RoleTool:
			prefix = "Tool"
		default:
			prefix = string(msg.Role)
		}
		if result.Len() > 0 {
			result.WriteString("\n")
		}
		result.WriteString(prefix + ": " + msg.Content)
	}
	return result.String()
}

// SlidingWindowCompressor implements sliding window compression.
type SlidingWindowCompressor struct {
	config *Config
}

func (c *SlidingWindowCompressor) Compress(ctx context.Context, messages []types.Message) ([]types.Message, *Stats, error) {
	originalCount := len(messages)
	originalTokens := EstimateMessagesTokens(messages)

	if len(messages) <= c.config.WindowSize {
		return messages, &Stats{
			OriginalCount:    originalCount,
			CompressedCount:  originalCount,
			OriginalTokens:   originalTokens,
			CompressedTokens: originalTokens,
			CompressionRatio: 1.0,
			Strategy:         StrategySlidingWindow,
		}, nil
	}

	compressed := messages[len(messages)-c.config.WindowSize:]
	compressedTokens := EstimateMessagesTokens(compressed)

	return compressed, &Stats{
		OriginalCount:    originalCount,
		CompressedCount:  len(compressed),
		OriginalTokens:   originalTokens,
		CompressedTokens: compressedTokens,
		CompressionRatio: float64(compressedTokens) / float64(originalTokens),
		Strategy:         StrategySlidingWindow,
	}, nil
}

func (c *SlidingWindowCompressor) EstimateTokens(messages []types.Message) int {
	return EstimateMessagesTokens(messages)
}

func (c *SlidingWindowCompressor) ShouldCompress(messages []types.Message) bool {
	return len(messages) > c.config.WindowSize ||
		EstimateMessagesTokens(messages) > c.config.MaxTokens
}

// LLMSummaryCompressor implements LLM-based summary compression.
type LLMSummaryCompressor struct {
	config *Config
}

func (c *LLMSummaryCompressor) Compress(ctx context.Context, messages []types.Message) ([]types.Message, *Stats, error) {
	originalCount := len(messages)
	originalTokens := EstimateMessagesTokens(messages)

	if len(messages) <= c.config.PreserveRecent+2 {
		return messages, &Stats{
			OriginalCount:    originalCount,
			CompressedCount:  originalCount,
			OriginalTokens:   originalTokens,
			CompressedTokens: originalTokens,
			CompressionRatio: 1.0,
			Strategy:         StrategyLLMSummary,
		}, nil
	}

	// Split messages
	toCompress := messages[:len(messages)-c.config.PreserveRecent]
	toPreserve := messages[len(messages)-c.config.PreserveRecent:]

	// Build history text
	historyText := MessagesToString(toCompress)

	// Generate summary using ChatModel's Invoke method
	prompt := fmt.Sprintf(c.config.SummaryPrompt, historyText)
	result, err := c.config.ChatModel.Invoke(ctx, []types.Message{
		{Role: types.RoleUser, Content: prompt},
	})
	if err != nil {
		// Fallback to sliding window
		fallback := &SlidingWindowCompressor{config: c.config}
		return fallback.Compress(ctx, messages)
	}

	// Create summary message
	summaryMsg := types.Message{
		Role:    types.RoleSystem,
		Content: "Conversation summary: " + result.Content,
	}

	// Combine summary and recent messages
	compressed := append([]types.Message{summaryMsg}, toPreserve...)
	compressedTokens := EstimateMessagesTokens(compressed)

	return compressed, &Stats{
		OriginalCount:    originalCount,
		CompressedCount:  len(compressed),
		OriginalTokens:   originalTokens,
		CompressedTokens: compressedTokens,
		CompressionRatio: float64(compressedTokens) / float64(originalTokens),
		Strategy:         StrategyLLMSummary,
	}, nil
}

func (c *LLMSummaryCompressor) EstimateTokens(messages []types.Message) int {
	return EstimateMessagesTokens(messages)
}

func (c *LLMSummaryCompressor) ShouldCompress(messages []types.Message) bool {
	return len(messages) > c.config.MaxMessages ||
		EstimateMessagesTokens(messages) > c.config.MaxTokens
}

// HybridCompressor combines multiple strategies with fallback.
type HybridCompressor struct {
	config *Config
}

func (c *HybridCompressor) Compress(ctx context.Context, messages []types.Message) ([]types.Message, *Stats, error) {
	originalCount := len(messages)
	originalTokens := EstimateMessagesTokens(messages)

	if !c.ShouldCompress(messages) {
		return messages, &Stats{
			OriginalCount:    originalCount,
			CompressedCount:  originalCount,
			OriginalTokens:   originalTokens,
			CompressedTokens: originalTokens,
			CompressionRatio: 1.0,
			Strategy:         StrategyHybrid,
		}, nil
	}

	// Try LLM summary first
	if c.config.ChatModel != nil && len(messages) > c.config.MaxMessages {
		summaryCompressor := &LLMSummaryCompressor{config: c.config}
		compressed, stats, err := summaryCompressor.Compress(ctx, messages)
		if err == nil {
			stats.Strategy = StrategyHybrid
			return compressed, stats, nil
		}
	}

	// Fallback to sliding window
	windowCompressor := &SlidingWindowCompressor{config: c.config}
	compressed, stats, err := windowCompressor.Compress(ctx, messages)
	if err == nil {
		stats.Strategy = StrategyHybrid
	}
	return compressed, stats, err
}

func (c *HybridCompressor) EstimateTokens(messages []types.Message) int {
	return EstimateMessagesTokens(messages)
}

func (c *HybridCompressor) ShouldCompress(messages []types.Message) bool {
	return len(messages) > c.config.MaxMessages ||
		EstimateMessagesTokens(messages) > c.config.MaxTokens
}

// TokenAwareCompressor implements token-aware compression.
type TokenAwareCompressor struct {
	config *Config
}

func (c *TokenAwareCompressor) Compress(ctx context.Context, messages []types.Message) ([]types.Message, *Stats, error) {
	originalCount := len(messages)
	originalTokens := EstimateMessagesTokens(messages)

	if originalTokens <= c.config.MaxTokens {
		return messages, &Stats{
			OriginalCount:    originalCount,
			CompressedCount:  originalCount,
			OriginalTokens:   originalTokens,
			CompressedTokens: originalTokens,
			CompressionRatio: 1.0,
			Strategy:         StrategyTokenAware,
		}, nil
	}

	// Keep messages from the end until token limit
	var compressed []types.Message
	currentTokens := 0

	for i := len(messages) - 1; i >= 0; i-- {
		msgTokens := EstimateTokens(messages[i].Content) + 4
		if currentTokens+msgTokens > c.config.MaxTokens {
			break
		}
		compressed = append([]types.Message{messages[i]}, compressed...)
		currentTokens += msgTokens
	}

	// Ensure at least one message
	if len(compressed) == 0 && len(messages) > 0 {
		compressed = messages[len(messages)-1:]
		currentTokens = EstimateMessagesTokens(compressed)
	}

	return compressed, &Stats{
		OriginalCount:    originalCount,
		CompressedCount:  len(compressed),
		OriginalTokens:   originalTokens,
		CompressedTokens: currentTokens,
		CompressionRatio: float64(currentTokens) / float64(originalTokens),
		Strategy:         StrategyTokenAware,
	}, nil
}

func (c *TokenAwareCompressor) EstimateTokens(messages []types.Message) int {
	return EstimateMessagesTokens(messages)
}

func (c *TokenAwareCompressor) ShouldCompress(messages []types.Message) bool {
	return EstimateMessagesTokens(messages) > c.config.MaxTokens
}
