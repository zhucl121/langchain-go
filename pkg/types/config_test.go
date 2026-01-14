package types

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewConfig(t *testing.T) {
	config := NewConfig()

	assert.NotNil(t, config)
	assert.NotNil(t, config.Tags)
	assert.NotNil(t, config.Metadata)
	assert.Equal(t, 10, config.MaxConcurrency)
	assert.Equal(t, 3, config.MaxRetries)
	assert.Equal(t, 30*time.Second, config.Timeout)
	assert.NotNil(t, config.Context)
}

func TestConfig_WithTags(t *testing.T) {
	config := NewConfig().WithTags("tag1", "tag2")

	assert.Len(t, config.Tags, 2)
	assert.Contains(t, config.Tags, "tag1")
	assert.Contains(t, config.Tags, "tag2")

	// 链式调用
	config.WithTags("tag3")
	assert.Len(t, config.Tags, 3)
}

func TestConfig_WithMetadata(t *testing.T) {
	config := NewConfig().
		WithMetadata("key1", "value1").
		WithMetadata("key2", 123)

	assert.Equal(t, "value1", config.Metadata["key1"])
	assert.Equal(t, 123, config.Metadata["key2"])
}

func TestConfig_WithRunName(t *testing.T) {
	config := NewConfig().WithRunName("test-run")

	assert.Equal(t, "test-run", config.RunName)
}

func TestConfig_WithRunID(t *testing.T) {
	config := NewConfig().WithRunID("run-123")

	assert.Equal(t, "run-123", config.RunID)
}

func TestConfig_WithMaxConcurrency(t *testing.T) {
	config := NewConfig().WithMaxConcurrency(20)

	assert.Equal(t, 20, config.MaxConcurrency)
}

func TestConfig_WithMaxRetries(t *testing.T) {
	config := NewConfig().WithMaxRetries(5)

	assert.Equal(t, 5, config.MaxRetries)
}

func TestConfig_WithTimeout(t *testing.T) {
	config := NewConfig().WithTimeout(1 * time.Minute)

	assert.Equal(t, 1*time.Minute, config.Timeout)
}

func TestConfig_WithCallbacks(t *testing.T) {
	callback1 := &mockCallbackHandler{}
	callback2 := &mockCallbackHandler{}

	config := NewConfig().WithCallbacks(callback1, callback2)

	assert.Len(t, config.Callbacks, 2)
}

func TestConfig_WithContext(t *testing.T) {
	ctx := context.WithValue(context.Background(), "key", "value")
	config := NewConfig().WithContext(ctx)

	assert.Equal(t, ctx, config.Context)
}

func TestConfig_Clone(t *testing.T) {
	original := NewConfig().
		WithRunName("test").
		WithTags("tag1", "tag2").
		WithMetadata("key", "value").
		WithMaxRetries(5)

	clone := original.Clone()

	// 验证值相等
	assert.Equal(t, original.RunName, clone.RunName)
	assert.Equal(t, original.MaxRetries, clone.MaxRetries)
	assert.Equal(t, len(original.Tags), len(clone.Tags))
	assert.Equal(t, len(original.Metadata), len(clone.Metadata))

	// 验证是深拷贝
	clone.RunName = "modified"
	assert.NotEqual(t, original.RunName, clone.RunName)

	clone.Tags = append(clone.Tags, "tag3")
	assert.NotEqual(t, len(original.Tags), len(clone.Tags))

	clone.Metadata["key"] = "new_value"
	assert.NotEqual(t, original.Metadata["key"], clone.Metadata["key"])
}

func TestConfig_Merge(t *testing.T) {
	config1 := NewConfig().
		WithRunName("run1").
		WithMaxRetries(3).
		WithTags("tag1")

	config2 := NewConfig().
		WithRunName("run2").
		WithMaxConcurrency(20).
		WithTags("tag2").
		WithMetadata("key", "value")

	config1.Merge(config2)

	// 验证合并结果
	assert.Equal(t, "run2", config1.RunName) // 覆盖
	assert.Equal(t, 3, config1.MaxRetries)   // 保留原值（config2 是默认值 3）
	assert.Equal(t, 20, config1.MaxConcurrency)
	assert.Contains(t, config1.Tags, "tag1")
	assert.Contains(t, config1.Tags, "tag2")
	assert.Equal(t, "value", config1.Metadata["key"])
}

func TestConfig_MergeNil(t *testing.T) {
	config := NewConfig().WithRunName("test")
	original := config.RunName

	config.Merge(nil)

	assert.Equal(t, original, config.RunName)
}

func TestConfig_GetContext(t *testing.T) {
	t.Run("with context", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), "key", "value")
		config := NewConfig().WithContext(ctx)

		result := config.GetContext()
		assert.Equal(t, ctx, result)
	})

	t.Run("without context", func(t *testing.T) {
		config := &Config{} // 空配置

		result := config.GetContext()
		assert.NotNil(t, result)
		assert.Equal(t, context.Background(), result)
	})
}

func TestConfig_GetContextWithTimeout(t *testing.T) {
	config := NewConfig().WithTimeout(100 * time.Millisecond)

	ctx, cancel := config.GetContextWithTimeout()
	defer cancel()

	// 验证超时
	select {
	case <-time.After(200 * time.Millisecond):
		t.Fatal("context should have timed out")
	case <-ctx.Done():
		assert.Error(t, ctx.Err())
	}
}

func TestConfig_GetContextWithoutTimeout(t *testing.T) {
	config := &Config{Timeout: 0}

	ctx, cancel := config.GetContextWithTimeout()
	defer cancel()

	// 验证不会超时
	select {
	case <-time.After(50 * time.Millisecond):
		// 正常
	case <-ctx.Done():
		t.Fatal("context should not timeout")
	}
}

func TestConfig_HasTag(t *testing.T) {
	config := NewConfig().WithTags("production", "api-v1")

	assert.True(t, config.HasTag("production"))
	assert.True(t, config.HasTag("api-v1"))
	assert.False(t, config.HasTag("development"))
}

func TestConfig_GetMetadata(t *testing.T) {
	config := NewConfig().
		WithMetadata("key1", "value1").
		WithMetadata("key2", 123)

	t.Run("existing key", func(t *testing.T) {
		val, ok := config.GetMetadata("key1")
		assert.True(t, ok)
		assert.Equal(t, "value1", val)
	})

	t.Run("non-existing key", func(t *testing.T) {
		val, ok := config.GetMetadata("key3")
		assert.False(t, ok)
		assert.Nil(t, val)
	})

	t.Run("nil metadata", func(t *testing.T) {
		emptyConfig := &Config{}
		val, ok := emptyConfig.GetMetadata("key")
		assert.False(t, ok)
		assert.Nil(t, val)
	})
}

func TestNewRetryPolicy(t *testing.T) {
	policy := NewRetryPolicy()

	assert.Equal(t, 3, policy.MaxRetries)
	assert.Equal(t, 1*time.Second, policy.InitialDelay)
	assert.Equal(t, 30*time.Second, policy.MaxDelay)
	assert.Equal(t, 2.0, policy.Multiplier)
}

func TestRetryPolicy_GetDelay(t *testing.T) {
	policy := NewRetryPolicy()

	tests := []struct {
		retryCount    int
		expectedDelay time.Duration
	}{
		{0, 1 * time.Second},   // 初始延迟
		{1, 2 * time.Second},   // 1 * 2^1
		{2, 4 * time.Second},   // 1 * 2^2
		{3, 8 * time.Second},   // 1 * 2^3
		{4, 16 * time.Second},  // 1 * 2^4
		{5, 30 * time.Second},  // 超过 MaxDelay，返回 MaxDelay
		{10, 30 * time.Second}, // 超过 MaxDelay
	}

	for _, tt := range tests {
		t.Run(string(rune(tt.retryCount)), func(t *testing.T) {
			delay := policy.GetDelay(tt.retryCount)
			assert.Equal(t, tt.expectedDelay, delay)
		})
	}
}

func TestRetryPolicy_CustomConfig(t *testing.T) {
	policy := RetryPolicy{
		MaxRetries:   5,
		InitialDelay: 500 * time.Millisecond,
		MaxDelay:     10 * time.Second,
		Multiplier:   1.5,
	}

	delay0 := policy.GetDelay(0)
	assert.Equal(t, 500*time.Millisecond, delay0)

	delay1 := policy.GetDelay(1)
	assert.Equal(t, 750*time.Millisecond, delay1) // 500 * 1.5

	delay2 := policy.GetDelay(2)
	assert.InDelta(t, float64(1125*time.Millisecond), float64(delay2), float64(10*time.Millisecond)) // 500 * 1.5^2
}

// Mock CallbackHandler
type mockCallbackHandler struct{}

func (m *mockCallbackHandler) OnStart(ctx context.Context, input any) error {
	return nil
}

func (m *mockCallbackHandler) OnEnd(ctx context.Context, output any) error {
	return nil
}

func (m *mockCallbackHandler) OnError(ctx context.Context, err error) error {
	return nil
}

// 基准测试
func BenchmarkNewConfig(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = NewConfig()
	}
}

func BenchmarkConfig_Clone(b *testing.B) {
	config := NewConfig().
		WithRunName("test").
		WithTags("tag1", "tag2", "tag3").
		WithMetadata("key1", "value1").
		WithMetadata("key2", "value2")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = config.Clone()
	}
}

func BenchmarkConfig_Merge(b *testing.B) {
	config1 := NewConfig().WithRunName("run1").WithMaxRetries(3)
	config2 := NewConfig().WithRunName("run2").WithMaxConcurrency(20)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		config1.Clone().Merge(config2)
	}
}

func BenchmarkRetryPolicy_GetDelay(b *testing.B) {
	policy := NewRetryPolicy()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = policy.GetDelay(i % 10)
	}
}
