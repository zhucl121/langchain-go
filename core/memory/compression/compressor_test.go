package compression

import (
	"context"
	"testing"

	"github.com/zhucl121/langchain-go/pkg/types"
)

func TestSlidingWindowCompressor(t *testing.T) {
	config := &Config{
		Strategy:   StrategySlidingWindow,
		WindowSize: 5,
	}

	compressor := &SlidingWindowCompressor{config: config}

	// Create 10 messages
	messages := make([]types.Message, 10)
	for i := 0; i < 10; i++ {
		messages[i] = types.Message{
			Role:    types.RoleUser,
			Content: "Message content",
		}
	}

	// Test compression
	compressed, stats, err := compressor.Compress(context.Background(), messages)
	if err != nil {
		t.Fatalf("Compress failed: %v", err)
	}

	if len(compressed) != 5 {
		t.Errorf("Expected 5 messages, got %d", len(compressed))
	}

	if stats.OriginalCount != 10 {
		t.Errorf("Expected original count 10, got %d", stats.OriginalCount)
	}

	if stats.CompressedCount != 5 {
		t.Errorf("Expected compressed count 5, got %d", stats.CompressedCount)
	}

	t.Logf("Compression stats: %+v", stats)
}

func TestTokenAwareCompressor(t *testing.T) {
	config := &Config{
		Strategy:  StrategyTokenAware,
		MaxTokens: 100,
	}

	compressor := &TokenAwareCompressor{config: config}

	// Create messages with varying lengths
	messages := []types.Message{
		{Role: types.RoleUser, Content: "Short message"},
		{Role: types.RoleAssistant, Content: "Another short message"},
		{Role: types.RoleUser, Content: "This is a much longer message that contains significantly more text and will consume more tokens than the previous messages combined"},
		{Role: types.RoleAssistant, Content: "Response"},
	}

	// Test compression
	compressed, stats, err := compressor.Compress(context.Background(), messages)
	if err != nil {
		t.Fatalf("Compress failed: %v", err)
	}

	t.Logf("Original messages: %d, Compressed: %d", stats.OriginalCount, stats.CompressedCount)
	t.Logf("Original tokens: %d, Compressed tokens: %d", stats.OriginalTokens, stats.CompressedTokens)
	t.Logf("Compression ratio: %.2f%%", stats.CompressionRatio*100)

	if stats.CompressedTokens > config.MaxTokens {
		t.Errorf("Compressed tokens %d exceeds max %d", stats.CompressedTokens, config.MaxTokens)
	}

	if len(compressed) == 0 {
		t.Error("Expected at least one message after compression")
	}
}

func TestEstimateTokens(t *testing.T) {
	tests := []struct {
		text     string
		minRange int
		maxRange int
	}{
		{"Hello", 5, 10},
		{"你好世界", 5, 8},
		{"This is a test", 18, 25},
		{"中英文混合 mixed content", 28, 35},
	}

	for _, tt := range tests {
		tokens := EstimateTokens(tt.text)
		if tokens < tt.minRange || tokens > tt.maxRange {
			t.Errorf("EstimateTokens(%q) = %d, expected range [%d, %d]", 
				tt.text, tokens, tt.minRange, tt.maxRange)
		}
		t.Logf("Text: %q, Tokens: %d", tt.text, tokens)
	}
}

func TestShouldCompress(t *testing.T) {
	config := &Config{
		Strategy:    StrategySlidingWindow,
		WindowSize:  5,
		MaxMessages: 10,
		MaxTokens:   100,
	}

	compressor := &SlidingWindowCompressor{config: config}

	// Test 1: Few messages, should not compress
	messages1 := make([]types.Message, 3)
	for i := 0; i < 3; i++ {
		messages1[i] = types.Message{
			Role:    types.RoleUser,
			Content: "Short",
		}
	}
	if compressor.ShouldCompress(messages1) {
		t.Error("Should not compress 3 messages with window size 5")
	}

	// Test 2: Many messages, should compress
	messages2 := make([]types.Message, 10)
	for i := 0; i < 10; i++ {
		messages2[i] = types.Message{
			Role:    types.RoleUser,
			Content: "Message",
		}
	}
	if !compressor.ShouldCompress(messages2) {
		t.Error("Should compress 10 messages with window size 5")
	}
}

func TestHybridCompressor_NoCompressionNeeded(t *testing.T) {
	config := DefaultConfig()
	compressor := &HybridCompressor{config: config}

	// Small message list
	messages := []types.Message{
		{Role: types.RoleUser, Content: "Hello"},
		{Role: types.RoleAssistant, Content: "Hi"},
	}

	compressed, stats, err := compressor.Compress(context.Background(), messages)
	if err != nil {
		t.Fatalf("Compress failed: %v", err)
	}

	if len(compressed) != len(messages) {
		t.Errorf("Expected no compression, got %d messages from %d", len(compressed), len(messages))
	}

	if stats.CompressionRatio != 1.0 {
		t.Errorf("Expected compression ratio 1.0, got %.2f", stats.CompressionRatio)
	}
}

func TestMessagesToString(t *testing.T) {
	messages := []types.Message{
		{Role: types.RoleUser, Content: "Hello"},
		{Role: types.RoleAssistant, Content: "Hi there!"},
		{Role: types.RoleSystem, Content: "System message"},
	}

	result := MessagesToString(messages)
	
	expected := "Human: Hello\nAI: Hi there!\nSystem: System message"
	if result != expected {
		t.Errorf("MessagesToString output mismatch\nGot:      %q\nExpected: %q", result, expected)
	}
}

func BenchmarkSlidingWindowCompress(b *testing.B) {
	config := &Config{
		Strategy:   StrategySlidingWindow,
		WindowSize: 10,
	}
	compressor := &SlidingWindowCompressor{config: config}

	messages := make([]types.Message, 100)
	for i := 0; i < 100; i++ {
		messages[i] = types.Message{
			Role:    types.RoleUser,
			Content: "This is a test message for benchmarking purposes",
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = compressor.Compress(context.Background(), messages)
	}
}

func BenchmarkTokenEstimation(b *testing.B) {
	text := "This is a test message with mixed 中英文内容 for benchmarking token estimation performance"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = EstimateTokens(text)
	}
}

func BenchmarkMessagesToString(b *testing.B) {
	messages := make([]types.Message, 20)
	for i := 0; i < 20; i++ {
		messages[i] = types.Message{
			Role:    types.RoleUser,
			Content: "This is message content",
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = MessagesToString(messages)
	}
}
