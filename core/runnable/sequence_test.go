package runnable

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSequence(t *testing.T) {
	ctx := context.Background()

	t.Run("two step sequence", func(t *testing.T) {
		times2 := Lambda(func(ctx context.Context, x int) (int, error) {
			return x * 2, nil
		})
		plus1 := Lambda(func(ctx context.Context, x int) (int, error) {
			return x + 1, nil
		})

		seq := NewSequence(times2, plus1)

		result, err := seq.Invoke(ctx, 5)
		require.NoError(t, err)
		assert.Equal(t, 11, result) // (5 * 2) + 1
	})

	t.Run("sequence name", func(t *testing.T) {
		first := LambdaWithName("First", func(ctx context.Context, x int) (int, error) {
			return x, nil
		})
		second := LambdaWithName("Second", func(ctx context.Context, x int) (int, error) {
			return x, nil
		})

		seq := NewSequence(first, second)
		assert.Contains(t, seq.GetName(), "First")
		assert.Contains(t, seq.GetName(), "Second")
	})

	t.Run("type transformation", func(t *testing.T) {
		intToString := Lambda(func(ctx context.Context, x int) (string, error) {
			return fmt.Sprintf("number: %d", x), nil
		})
		stringLength := Lambda(func(ctx context.Context, s string) (int, error) {
			return len(s), nil
		})

		seq := NewSequence(intToString, stringLength)

		result, err := seq.Invoke(ctx, 42)
		require.NoError(t, err)
		assert.Equal(t, len("number: 42"), result)
	})
}

func TestSequence_Invoke(t *testing.T) {
	ctx := context.Background()

	t.Run("first step fails", func(t *testing.T) {
		failFirst := Lambda(func(ctx context.Context, x int) (int, error) {
			return 0, errors.New("first failed")
		})
		second := Lambda(func(ctx context.Context, x int) (int, error) {
			return x, nil
		})

		seq := NewSequence(failFirst, second)

		_, err := seq.Invoke(ctx, 5)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "first runnable failed")
	})

	t.Run("second step fails", func(t *testing.T) {
		first := Lambda(func(ctx context.Context, x int) (int, error) {
			return x * 2, nil
		})
		failSecond := Lambda(func(ctx context.Context, x int) (int, error) {
			return 0, errors.New("second failed")
		})

		seq := NewSequence(first, failSecond)

		_, err := seq.Invoke(ctx, 5)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "second runnable failed")
	})

	t.Run("context cancellation after first", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		first := Lambda(func(ctx context.Context, x int) (int, error) {
			cancel() // 取消上下文
			return x * 2, nil
		})
		second := Lambda(func(ctx context.Context, x int) (int, error) {
			return x + 1, nil
		})

		seq := NewSequence(first, second)

		_, err := seq.Invoke(ctx, 5)
		assert.Error(t, err)
		assert.Equal(t, context.Canceled, err)
	})
}

func TestSequence_Batch(t *testing.T) {
	ctx := context.Background()

	t.Run("batch sequence", func(t *testing.T) {
		times2 := Lambda(func(ctx context.Context, x int) (int, error) {
			return x * 2, nil
		})
		plus1 := Lambda(func(ctx context.Context, x int) (int, error) {
			return x + 1, nil
		})

		seq := NewSequence(times2, plus1)

		results, err := seq.Batch(ctx, []int{1, 2, 3})
		require.NoError(t, err)
		assert.Equal(t, []int{3, 5, 7}, results) // [(1*2)+1, (2*2)+1, (3*2)+1]
	})

	t.Run("batch first fails", func(t *testing.T) {
		failFirst := Lambda(func(ctx context.Context, x int) (int, error) {
			return 0, errors.New("first failed")
		})
		second := Lambda(func(ctx context.Context, x int) (int, error) {
			return x, nil
		})

		seq := NewSequence(failFirst, second)

		_, err := seq.Batch(ctx, []int{1, 2, 3})
		assert.Error(t, err)
	})
}

func TestSequence_Stream(t *testing.T) {
	ctx := context.Background()

	t.Run("stream sequence", func(t *testing.T) {
		times2 := Lambda(func(ctx context.Context, x int) (int, error) {
			return x * 2, nil
		})
		plus1 := Lambda(func(ctx context.Context, x int) (int, error) {
			return x + 1, nil
		})

		seq := NewSequence(times2, plus1)

		stream, err := seq.Stream(ctx, 5)
		require.NoError(t, err)

		events := collectStreamEvents(stream)

		assert.Greater(t, len(events), 0)
		assert.Equal(t, EventStart, events[0].Type)

		// 查找最终结果
		var finalResult int
		for _, event := range events {
			if event.Type == EventEnd {
				finalResult = event.Data
			}
		}
		assert.Equal(t, 11, finalResult)
	})
}

func TestSequence_Composition(t *testing.T) {
	ctx := context.Background()

	times2 := Lambda(func(ctx context.Context, x int) (int, error) {
		return x * 2, nil
	})
	plus1 := Lambda(func(ctx context.Context, x int) (int, error) {
		return x + 1, nil
	})
	times3 := Lambda(func(ctx context.Context, x int) (int, error) {
		return x * 3, nil
	})

	// 先创建序列，再组合
	seq1 := NewSequence(times2, plus1)
	seq2 := NewSequence(seq1, times3)

	result, err := seq2.Invoke(ctx, 5)
	require.NoError(t, err)
	assert.Equal(t, 33, result) // ((5 * 2) + 1) * 3
}

func TestSequence_Function(t *testing.T) {
	ctx := context.Background()

	t.Run("sequence helper", func(t *testing.T) {
		times2 := Lambda(func(ctx context.Context, x int) (int, error) {
			return x * 2, nil
		})
		plus1 := Lambda(func(ctx context.Context, x int) (int, error) {
			return x + 1, nil
		})
		times3 := Lambda(func(ctx context.Context, x int) (int, error) {
			return x * 3, nil
		})

		seq := Sequence(times2, plus1, times3)

		result, err := seq.Invoke(ctx, 5)
		require.NoError(t, err)
		assert.Equal(t, 33, result) // ((5 * 2) + 1) * 3
	})

	t.Run("empty sequence", func(t *testing.T) {
		seq := Sequence[int]()

		result, err := seq.Invoke(ctx, 42)
		require.NoError(t, err)
		assert.Equal(t, 42, result) // Passthrough
	})

	t.Run("single runnable sequence", func(t *testing.T) {
		times2 := Lambda(func(ctx context.Context, x int) (int, error) {
			return x * 2, nil
		})

		seq := Sequence(times2)

		result, err := seq.Invoke(ctx, 5)
		require.NoError(t, err)
		assert.Equal(t, 10, result)
	})
}

// 基准测试
func BenchmarkSequence_Invoke(b *testing.B) {
	ctx := context.Background()
	times2 := Lambda(func(ctx context.Context, x int) (int, error) {
		return x * 2, nil
	})
	plus1 := Lambda(func(ctx context.Context, x int) (int, error) {
		return x + 1, nil
	})

	seq := NewSequence(times2, plus1)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = seq.Invoke(ctx, i)
	}
}

func BenchmarkSequence_Batch(b *testing.B) {
	ctx := context.Background()
	times2 := Lambda(func(ctx context.Context, x int) (int, error) {
		return x * 2, nil
	})
	plus1 := Lambda(func(ctx context.Context, x int) (int, error) {
		return x + 1, nil
	})

	seq := NewSequence(times2, plus1)
	inputs := []int{1, 2, 3, 4, 5}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = seq.Batch(ctx, inputs)
	}
}

func BenchmarkSequence_MultiComposition(b *testing.B) {
	ctx := context.Background()
	
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

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = pipeline.Invoke(ctx, i)
	}
}
