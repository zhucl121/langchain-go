package runnable

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewParallel(t *testing.T) {
	ctx := context.Background()

	t.Run("parallel execution", func(t *testing.T) {
		// 创建 Runnable[int, int] 然后包装为 Runnable[int, any]
		double := Lambda(func(ctx context.Context, x int) (int, error) {
			return x * 2, nil
		})
		triple := Lambda(func(ctx context.Context, x int) (int, error) {
			return x * 3, nil
		})
		increment := Lambda(func(ctx context.Context, x int) (int, error) {
			return x + 1, nil
		})

		parallel := NewParallel(map[string]Runnable[int, any]{
			"double":    AsAny[int, int](double),
			"triple":    AsAny[int, int](triple),
			"increment": AsAny[int, int](increment),
		})

		results, err := parallel.Invoke(ctx, 5)
		require.NoError(t, err)

		assert.Len(t, results, 3)
		assert.Equal(t, 10, results["double"])
		assert.Equal(t, 15, results["triple"])
		assert.Equal(t, 6, results["increment"])
	})

	t.Run("empty parallel", func(t *testing.T) {
		parallel := NewParallel[int](map[string]Runnable[int, any]{})

		results, err := parallel.Invoke(ctx, 5)
		require.NoError(t, err)
		assert.Empty(t, results)
	})
}

func TestParallel_Invoke(t *testing.T) {
	ctx := context.Background()

	t.Run("one runnable fails", func(t *testing.T) {
		success := Lambda(func(ctx context.Context, x int) (int, error) {
			return x * 2, nil
		})
		failure := Lambda(func(ctx context.Context, x int) (int, error) {
			return 0, errors.New("intentional error")
		})

		parallel := NewParallel(map[string]Runnable[int, any]{
			"success": AsAny[int, int](success),
			"failure": AsAny[int, int](failure),
		})

		_, err := parallel.Invoke(ctx, 5)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "parallel runnable")
		assert.Contains(t, err.Error(), "failed")
	})

	t.Run("all runnables succeed", func(t *testing.T) {
		upper := Lambda(func(ctx context.Context, s string) (string, error) {
			return fmt.Sprintf("UPPER: %s", s), nil
		})
		lower := Lambda(func(ctx context.Context, s string) (string, error) {
			return fmt.Sprintf("lower: %s", s), nil
		})
		length := Lambda(func(ctx context.Context, s string) (int, error) {
			return len(s), nil
		})

		parallel := NewParallel(map[string]Runnable[string, any]{
			"upper":  AsAny[string, string](upper),
			"lower":  AsAny[string, string](lower),
			"length": AsAny[string, int](length),
		})

		results, err := parallel.Invoke(ctx, "Hello")
		require.NoError(t, err)

		assert.Equal(t, "UPPER: Hello", results["upper"])
		assert.Equal(t, "lower: Hello", results["lower"])
		assert.Equal(t, 5, results["length"])
	})

	t.Run("context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		// 等待确保上下文被取消
		<-ctx.Done()

		slow := Lambda(func(ctx context.Context, x int) (int, error) {
			time.Sleep(50 * time.Millisecond)
			return x, nil
		})

		parallel := NewParallel(map[string]Runnable[int, any]{
			"slow": AsAny[int, int](slow),
		})

		_, err := parallel.Invoke(ctx, 5)
		assert.Error(t, err)
	})
}

func TestParallel_Batch(t *testing.T) {
	ctx := context.Background()

	t.Run("batch parallel", func(t *testing.T) {
		double := Lambda(func(ctx context.Context, x int) (int, error) {
			return x * 2, nil
		})
		square := Lambda(func(ctx context.Context, x int) (int, error) {
			return x * x, nil
		})

		parallel := NewParallel(map[string]Runnable[int, any]{
			"double": AsAny[int, int](double),
			"square": AsAny[int, int](square),
		})

		results, err := parallel.Batch(ctx, []int{2, 3, 4})
		require.NoError(t, err)

		assert.Len(t, results, 3)
		assert.Equal(t, 4, results[0]["double"])
		assert.Equal(t, 4, results[0]["square"])
		assert.Equal(t, 6, results[1]["double"])
		assert.Equal(t, 9, results[1]["square"])
	})

	t.Run("empty batch", func(t *testing.T) {
		double := Lambda(func(ctx context.Context, x int) (int, error) {
			return x * 2, nil
		})

		parallel := NewParallel(map[string]Runnable[int, any]{
			"double": AsAny[int, int](double),
		})

		results, err := parallel.Batch(ctx, []int{})
		require.NoError(t, err)
		assert.Empty(t, results)
	})
}

func TestParallel_Stream(t *testing.T) {
	ctx := context.Background()

	t.Run("stream parallel", func(t *testing.T) {
		double := Lambda(func(ctx context.Context, x int) (int, error) {
			return x * 2, nil
		})
		triple := Lambda(func(ctx context.Context, x int) (int, error) {
			return x * 3, nil
		})

		parallel := NewParallel(map[string]Runnable[int, any]{
			"double": AsAny[int, int](double),
			"triple": AsAny[int, int](triple),
		})

		stream, err := parallel.Stream(ctx, 5)
		require.NoError(t, err)

		events := collectStreamEvents(stream)

		assert.Greater(t, len(events), 0)
		assert.Equal(t, EventStart, events[0].Type)

		// 查找最终结果
		var finalResult map[string]any
		for _, event := range events {
			if event.Type == EventEnd {
				finalResult = event.Data
			}
		}

		require.NotNil(t, finalResult)
		assert.Equal(t, 10, finalResult["double"])
		assert.Equal(t, 15, finalResult["triple"])
	})
}

func TestParallel_Composition(t *testing.T) {
	ctx := context.Background()

	t.Run("parallel then process", func(t *testing.T) {
		double := Lambda(func(ctx context.Context, x int) (int, error) {
			return x * 2, nil
		})
		triple := Lambda(func(ctx context.Context, x int) (int, error) {
			return x * 3, nil
		})

		parallel := NewParallel(map[string]Runnable[int, any]{
			"double": AsAny[int, int](double),
			"triple": AsAny[int, int](triple),
		})

		// 处理并行结果
		sumResults := Lambda(func(ctx context.Context, m map[string]any) (int, error) {
			sum := 0
			for _, v := range m {
				if num, ok := v.(int); ok {
					sum += num
				}
			}
			return sum, nil
		})

		pipeline := NewSequence(parallel, sumResults)

		result, err := pipeline.Invoke(ctx, 5)
		require.NoError(t, err)
		assert.Equal(t, 25, result) // 10 + 15
	})
}

func TestParallel_Function(t *testing.T) {
	ctx := context.Background()

	t.Run("parallel helper", func(t *testing.T) {
		a := Lambda(func(ctx context.Context, x int) (int, error) { return x + 1, nil })
		b := Lambda(func(ctx context.Context, x int) (int, error) { return x + 2, nil })

		parallel := Parallel(map[string]Runnable[int, any]{
			"a": AsAny[int, int](a),
			"b": AsAny[int, int](b),
		})

		results, err := parallel.Invoke(ctx, 10)
		require.NoError(t, err)
		assert.Equal(t, 11, results["a"])
		assert.Equal(t, 12, results["b"])
	})
}

func TestParallel_ConcurrentSafety(t *testing.T) {
	ctx := context.Background()

	t.Run("concurrent access to results", func(t *testing.T) {
		// 测试并发安全性
		r1 := Lambda(func(ctx context.Context, x int) (int, error) {
			time.Sleep(1 * time.Millisecond)
			return x * 1, nil
		})
		r2 := Lambda(func(ctx context.Context, x int) (int, error) {
			time.Sleep(2 * time.Millisecond)
			return x * 2, nil
		})
		r3 := Lambda(func(ctx context.Context, x int) (int, error) {
			time.Sleep(3 * time.Millisecond)
			return x * 3, nil
		})

		parallel := NewParallel(map[string]Runnable[int, any]{
			"r1": AsAny[int, int](r1),
			"r2": AsAny[int, int](r2),
			"r3": AsAny[int, int](r3),
		})

		// 多次执行以测试并发安全
		for i := 0; i < 10; i++ {
			results, err := parallel.Invoke(ctx, 5)
			require.NoError(t, err)
			assert.Len(t, results, 3)
		}
	})
}

// 基准测试
func BenchmarkParallel_Invoke(b *testing.B) {
	ctx := context.Background()
	double := Lambda(func(ctx context.Context, x int) (int, error) {
		return x * 2, nil
	})
	triple := Lambda(func(ctx context.Context, x int) (int, error) {
		return x * 3, nil
	})
	square := Lambda(func(ctx context.Context, x int) (int, error) {
		return x * x, nil
	})

	parallel := NewParallel(map[string]Runnable[int, any]{
		"double": AsAny[int, int](double),
		"triple": AsAny[int, int](triple),
		"square": AsAny[int, int](square),
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parallel.Invoke(ctx, i)
	}
}

func BenchmarkParallel_vs_Sequential(b *testing.B) {
	ctx := context.Background()

	slowOp := func(ctx context.Context, x int) (int, error) {
		time.Sleep(1 * time.Millisecond)
		return x, nil
	}

	b.Run("sequential", func(b *testing.B) {
		r1 := Lambda(slowOp)
		r2 := Lambda(slowOp)
		r3 := Lambda(slowOp)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = r1.Invoke(ctx, i)
			_, _ = r2.Invoke(ctx, i)
			_, _ = r3.Invoke(ctx, i)
		}
	})

	b.Run("parallel", func(b *testing.B) {
		r1 := Lambda(slowOp)
		r2 := Lambda(slowOp)
		r3 := Lambda(slowOp)

		parallel := NewParallel(map[string]Runnable[int, any]{
			"r1": AsAny[int, int](r1),
			"r2": AsAny[int, int](r2),
			"r3": AsAny[int, int](r3),
		})

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = parallel.Invoke(ctx, i)
		}
	})
}
