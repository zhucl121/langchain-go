package runnable

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLambda(t *testing.T) {
	ctx := context.Background()

	t.Run("simple lambda", func(t *testing.T) {
		doubler := Lambda(func(ctx context.Context, x int) (int, error) {
			return x * 2, nil
		})

		result, err := doubler.Invoke(ctx, 5)
		require.NoError(t, err)
		assert.Equal(t, 10, result)
	})

	t.Run("lambda with name", func(t *testing.T) {
		lambda := LambdaWithName("MyLambda", func(ctx context.Context, x int) (int, error) {
			return x + 1, nil
		})

		assert.Equal(t, "MyLambda", lambda.GetName())
	})

	t.Run("lambda with error", func(t *testing.T) {
		failLambda := Lambda(func(ctx context.Context, x int) (int, error) {
			return 0, errors.New("intentional error")
		})

		_, err := failLambda.Invoke(ctx, 5)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "intentional error")
	})
}

func TestLambda_Batch(t *testing.T) {
	ctx := context.Background()

	t.Run("batch execution", func(t *testing.T) {
		doubler := Lambda(func(ctx context.Context, x int) (int, error) {
			return x * 2, nil
		})

		results, err := doubler.Batch(ctx, []int{1, 2, 3, 4, 5})
		require.NoError(t, err)
		assert.Equal(t, []int{2, 4, 6, 8, 10}, results)
	})

	t.Run("empty batch", func(t *testing.T) {
		doubler := Lambda(func(ctx context.Context, x int) (int, error) {
			return x * 2, nil
		})

		results, err := doubler.Batch(ctx, []int{})
		require.NoError(t, err)
		assert.Empty(t, results)
	})

	t.Run("batch with error", func(t *testing.T) {
		failOn3 := Lambda(func(ctx context.Context, x int) (int, error) {
			if x == 3 {
				return 0, errors.New("error on 3")
			}
			return x * 2, nil
		})

		_, err := failOn3.Batch(ctx, []int{1, 2, 3, 4, 5})
		assert.Error(t, err)
	})

	t.Run("batch context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // 立即取消

		slow := Lambda(func(ctx context.Context, x int) (int, error) {
			time.Sleep(100 * time.Millisecond)
			return x, nil
		})

		_, err := slow.Batch(ctx, []int{1, 2, 3})
		assert.Error(t, err)
	})
}

func TestLambda_Stream(t *testing.T) {
	ctx := context.Background()

	t.Run("stream execution", func(t *testing.T) {
		doubler := Lambda(func(ctx context.Context, x int) (int, error) {
			return x * 2, nil
		})

		stream, err := doubler.Stream(ctx, 5)
		require.NoError(t, err)

		events := collectStreamEvents(stream)

		assert.Len(t, events, 3) // start, stream, end
		assert.Equal(t, EventStart, events[0].Type)
		assert.Equal(t, EventStream, events[1].Type)
		assert.Equal(t, 10, events[1].Data)
		assert.Equal(t, EventEnd, events[2].Type)
	})

	t.Run("stream with error", func(t *testing.T) {
		failLambda := Lambda(func(ctx context.Context, x int) (int, error) {
			return 0, errors.New("stream error")
		})

		stream, err := failLambda.Stream(ctx, 5)
		require.NoError(t, err)

		events := collectStreamEvents(stream)

		assert.Len(t, events, 2) // start, error
		assert.Equal(t, EventStart, events[0].Type)
		assert.Equal(t, EventError, events[1].Type)
		assert.Error(t, events[1].Error)
	})
}

func TestLambda_Sequence(t *testing.T) {
	ctx := context.Background()

	t.Run("sequence two lambdas", func(t *testing.T) {
		doubler := Lambda(func(ctx context.Context, x int) (int, error) {
			return x * 2, nil
		})
		adder := Lambda(func(ctx context.Context, x int) (int, error) {
			return x + 1, nil
		})

		pipeline := NewSequence(doubler, adder)
		result, err := pipeline.Invoke(ctx, 5)

		require.NoError(t, err)
		assert.Equal(t, 11, result) // (5 * 2) + 1
	})

	t.Run("sequence multiple lambdas", func(t *testing.T) {
		times2 := Lambda(func(ctx context.Context, x int) (int, error) {
			return x * 2, nil
		})
		plus1 := Lambda(func(ctx context.Context, x int) (int, error) {
			return x + 1, nil
		})
		times3 := Lambda(func(ctx context.Context, x int) (int, error) {
			return x * 3, nil
		})

		pipeline := NewSequence(NewSequence(times2, plus1), times3)
		result, err := pipeline.Invoke(ctx, 5)

		require.NoError(t, err)
		assert.Equal(t, 33, result) // ((5 * 2) + 1) * 3
	})
}

func TestPassthrough(t *testing.T) {
	ctx := context.Background()

	t.Run("passthrough int", func(t *testing.T) {
		pt := Passthrough[int]()
		result, err := pt.Invoke(ctx, 42)

		require.NoError(t, err)
		assert.Equal(t, 42, result)
	})

	t.Run("passthrough string", func(t *testing.T) {
		pt := Passthrough[string]()
		result, err := pt.Invoke(ctx, "hello")

		require.NoError(t, err)
		assert.Equal(t, "hello", result)
	})

	t.Run("passthrough struct", func(t *testing.T) {
		type testStruct struct {
			Value int
		}

		pt := Passthrough[testStruct]()
		input := testStruct{Value: 100}
		result, err := pt.Invoke(ctx, input)

		require.NoError(t, err)
		assert.Equal(t, input, result)
	})
}

func TestLambda_ContextPropagation(t *testing.T) {
	t.Run("context is passed to lambda function", func(t *testing.T) {
		type ctxKey string
		const key ctxKey = "testKey"
		
		ctx := context.WithValue(context.Background(), key, "testValue")

		var receivedValue string
		lambda := Lambda(func(ctx context.Context, x int) (int, error) {
			// 验证上下文值传递
			value := ctx.Value(key)
			if v, ok := value.(string); ok {
				receivedValue = v
			}
			return x * 2, nil
		})

		result, err := lambda.Invoke(ctx, 5)
		require.NoError(t, err)
		assert.Equal(t, 10, result)
		assert.Equal(t, "testValue", receivedValue)
	})
}

// 辅助函数：收集流式事件
func collectStreamEvents[T any](stream <-chan StreamEvent[T]) []StreamEvent[T] {
	var events []StreamEvent[T]
	for event := range stream {
		events = append(events, event)
	}
	return events
}

// 基准测试
func BenchmarkLambda_Invoke(b *testing.B) {
	ctx := context.Background()
	doubler := Lambda(func(ctx context.Context, x int) (int, error) {
		return x * 2, nil
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = doubler.Invoke(ctx, i)
	}
}

func BenchmarkLambda_Batch(b *testing.B) {
	ctx := context.Background()
	doubler := Lambda(func(ctx context.Context, x int) (int, error) {
		return x * 2, nil
	})
	inputs := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = doubler.Batch(ctx, inputs)
	}
}

func BenchmarkLambda_Sequence(b *testing.B) {
	ctx := context.Background()
	times2 := Lambda(func(ctx context.Context, x int) (int, error) {
		return x * 2, nil
	})
	plus1 := Lambda(func(ctx context.Context, x int) (int, error) {
		return x + 1, nil
	})

	pipeline := NewSequence(times2, plus1)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = pipeline.Invoke(ctx, i)
	}
}
