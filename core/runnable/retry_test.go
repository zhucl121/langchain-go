package runnable

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zhucl121/langchain-go/pkg/types"
)

func TestRetryRunnable(t *testing.T) {
	ctx := context.Background()

	t.Run("succeed on first attempt", func(t *testing.T) {
		lambda := Lambda(func(ctx context.Context, x int) (int, error) {
			return x * 2, nil
		})

		policy := types.NewRetryPolicy()
		retry := NewRetryRunnable(lambda, policy)

		result, err := retry.Invoke(ctx, 5)
		require.NoError(t, err)
		assert.Equal(t, 10, result)
	})

	t.Run("succeed on second attempt", func(t *testing.T) {
		var attemptCount atomic.Int32

		flaky := Lambda(func(ctx context.Context, x int) (int, error) {
			count := attemptCount.Add(1)
			if count == 1 {
				return 0, errors.New("first attempt fails")
			}
			return x * 2, nil
		})

		policy := types.RetryPolicy{
			MaxRetries:   2,
			InitialDelay: 10 * time.Millisecond,
			MaxDelay:     1 * time.Second,
			Multiplier:   2.0,
		}
		retry := NewRetryRunnable(flaky, policy)

		result, err := retry.Invoke(ctx, 5)
		require.NoError(t, err)
		assert.Equal(t, 10, result)
		assert.Equal(t, int32(2), attemptCount.Load())
	})

	t.Run("fail all attempts", func(t *testing.T) {
		var attemptCount atomic.Int32

		alwaysFail := Lambda(func(ctx context.Context, x int) (int, error) {
			attemptCount.Add(1)
			return 0, errors.New("always fails")
		})

		policy := types.RetryPolicy{
			MaxRetries:   2,
			InitialDelay: 10 * time.Millisecond,
			MaxDelay:     1 * time.Second,
			Multiplier:   2.0,
		}
		retry := NewRetryRunnable(alwaysFail, policy)

		_, err := retry.Invoke(ctx, 5)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "retry exhausted")
		assert.Contains(t, err.Error(), "3 attempts")
		assert.Equal(t, int32(3), attemptCount.Load()) // 1 initial + 2 retries
	})

	t.Run("context cancelled during retry", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		var attemptCount atomic.Int32
		flaky := Lambda(func(ctx context.Context, x int) (int, error) {
			count := attemptCount.Add(1)
			if count == 2 {
				cancel() // 在第二次尝试时取消
			}
			return 0, errors.New("error")
		})

		policy := types.RetryPolicy{
			MaxRetries:   5,
			InitialDelay: 10 * time.Millisecond,
			MaxDelay:     1 * time.Second,
			Multiplier:   2.0,
		}
		retry := NewRetryRunnable(flaky, policy)

		_, err := retry.Invoke(ctx, 5)
		assert.Error(t, err)
		assert.Equal(t, context.Canceled, err)
	})
}

func TestRetryRunnable_ExponentialBackoff(t *testing.T) {
	ctx := context.Background()

	var attemptCount atomic.Int32
	var attemptTimes []time.Time

	flaky := Lambda(func(ctx context.Context, x int) (int, error) {
		attemptTimes = append(attemptTimes, time.Now())
		count := attemptCount.Add(1)
		if count <= 3 {
			return 0, errors.New("fail")
		}
		return x, nil
	})

	policy := types.RetryPolicy{
		MaxRetries:   5,
		InitialDelay: 50 * time.Millisecond,
		MaxDelay:     1 * time.Second,
		Multiplier:   2.0,
	}
	retry := NewRetryRunnable(flaky, policy)

	result, err := retry.Invoke(ctx, 42)
	require.NoError(t, err)
	assert.Equal(t, 42, result)

	// 验证指数退避
	assert.Equal(t, int32(4), attemptCount.Load())
	assert.Len(t, attemptTimes, 4)

	// 第一次和第二次之间应该有 ~50ms 延迟
	if len(attemptTimes) >= 2 {
		delay1 := attemptTimes[1].Sub(attemptTimes[0])
		assert.GreaterOrEqual(t, delay1, 45*time.Millisecond)
	}

	// 第二次和第三次之间应该有 ~100ms 延迟
	if len(attemptTimes) >= 3 {
		delay2 := attemptTimes[2].Sub(attemptTimes[1])
		assert.GreaterOrEqual(t, delay2, 95*time.Millisecond)
	}
}

func TestFallbackRunnable(t *testing.T) {
	ctx := context.Background()

	t.Run("primary succeeds", func(t *testing.T) {
		primary := Lambda(func(ctx context.Context, x int) (int, error) {
			return x * 2, nil
		})
		fallback := Lambda(func(ctx context.Context, x int) (int, error) {
			return x * 10, nil
		})

		fb := NewFallbackRunnable(primary, []Runnable[int, int]{fallback})

		result, err := fb.Invoke(ctx, 5)
		require.NoError(t, err)
		assert.Equal(t, 10, result) // 使用 primary 结果
	})

	t.Run("primary fails, fallback succeeds", func(t *testing.T) {
		primary := Lambda(func(ctx context.Context, x int) (int, error) {
			return 0, errors.New("primary fails")
		})
		fallback := Lambda(func(ctx context.Context, x int) (int, error) {
			return x * 10, nil
		})

		fb := NewFallbackRunnable(primary, []Runnable[int, int]{fallback})

		result, err := fb.Invoke(ctx, 5)
		require.NoError(t, err)
		assert.Equal(t, 50, result) // 使用 fallback 结果
	})

	t.Run("multiple fallbacks", func(t *testing.T) {
		primary := Lambda(func(ctx context.Context, x int) (int, error) {
			return 0, errors.New("primary fails")
		})
		fallback1 := Lambda(func(ctx context.Context, x int) (int, error) {
			return 0, errors.New("fallback1 fails")
		})
		fallback2 := Lambda(func(ctx context.Context, x int) (int, error) {
			return x * 100, nil
		})

		fb := NewFallbackRunnable(primary, []Runnable[int, int]{fallback1, fallback2})

		result, err := fb.Invoke(ctx, 5)
		require.NoError(t, err)
		assert.Equal(t, 500, result) // 使用 fallback2 结果
	})

	t.Run("all fail", func(t *testing.T) {
		primary := Lambda(func(ctx context.Context, x int) (int, error) {
			return 0, errors.New("primary fails")
		})
		fallback := Lambda(func(ctx context.Context, x int) (int, error) {
			return 0, errors.New("fallback fails")
		})

		fb := NewFallbackRunnable(primary, []Runnable[int, int]{fallback})

		_, err := fb.Invoke(ctx, 5)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "all fallbacks failed")
		assert.Contains(t, err.Error(), "primary error")
	})

	t.Run("context cancelled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		primary := Lambda(func(ctx context.Context, x int) (int, error) {
			return 0, errors.New("primary fails")
		})
		fallback := Lambda(func(ctx context.Context, x int) (int, error) {
			return x, nil
		})

		fb := NewFallbackRunnable(primary, []Runnable[int, int]{fallback})

		_, err := fb.Invoke(ctx, 5)
		assert.Error(t, err)
		assert.Equal(t, context.Canceled, err)
	})
}

func TestRetryAndFallback_Combined(t *testing.T) {
	ctx := context.Background()

	t.Run("retry then fallback", func(t *testing.T) {
		var primaryAttempts atomic.Int32

		primary := Lambda(func(ctx context.Context, x int) (int, error) {
			primaryAttempts.Add(1)
			return 0, errors.New("always fails")
		})

		fallback := Lambda(func(ctx context.Context, x int) (int, error) {
			return x * 100, nil
		})

		policy := types.RetryPolicy{
			MaxRetries:   2,
			InitialDelay: 5 * time.Millisecond,
			MaxDelay:     1 * time.Second,
			Multiplier:   2.0,
		}

		// 先包装重试，再包装降级
		withRetry := NewRetryRunnable(primary, policy)
		withFallback := NewFallbackRunnable(withRetry, []Runnable[int, int]{fallback})

		result, err := withFallback.Invoke(ctx, 5)
		require.NoError(t, err)
		assert.Equal(t, 500, result)
		assert.Equal(t, int32(3), primaryAttempts.Load()) // 1 + 2 retries
	})
}

// 基准测试
func BenchmarkRetryRunnable_NoRetry(b *testing.B) {
	ctx := context.Background()
	lambda := Lambda(func(ctx context.Context, x int) (int, error) {
		return x * 2, nil
	})

	policy := types.NewRetryPolicy()
	retry := NewRetryRunnable(lambda, policy)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = retry.Invoke(ctx, i)
	}
}

func BenchmarkRetryRunnable_WithRetries(b *testing.B) {
	ctx := context.Background()
	var attemptCount atomic.Int32

	flaky := Lambda(func(ctx context.Context, x int) (int, error) {
		// 每3次成功1次
		if attemptCount.Add(1)%3 == 0 {
			return x * 2, nil
		}
		return 0, errors.New("flaky error")
	})

	policy := types.RetryPolicy{
		MaxRetries:   3,
		InitialDelay: 1 * time.Microsecond,
		MaxDelay:     1 * time.Millisecond,
		Multiplier:   2.0,
	}
	retry := NewRetryRunnable(flaky, policy)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = retry.Invoke(ctx, i)
	}
}

func BenchmarkFallbackRunnable(b *testing.B) {
	ctx := context.Background()

	primary := Lambda(func(ctx context.Context, x int) (int, error) {
		return x * 2, nil
	})
	fallback := Lambda(func(ctx context.Context, x int) (int, error) {
		return x * 10, nil
	})

	fb := NewFallbackRunnable(primary, []Runnable[int, int]{fallback})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = fb.Invoke(ctx, i)
	}
}
