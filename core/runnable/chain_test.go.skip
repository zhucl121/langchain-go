package runnable

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"
)

// Mock runnables for testing

type mockUppercase struct{}

func (m *mockUppercase) Invoke(ctx context.Context, input string, opts ...Option) (string, error) {
	return strings.ToUpper(input), nil
}

func (m *mockUppercase) Batch(ctx context.Context, inputs []string, opts ...Option) ([]string, error) {
	results := make([]string, len(inputs))
	for i, input := range inputs {
		results[i] = strings.ToUpper(input)
	}
	return results, nil
}

func (m *mockUppercase) Stream(ctx context.Context, input string, opts ...Option) (<-chan StreamEvent[string], error) {
	ch := make(chan StreamEvent[string], 1)
	ch <- StreamEvent[string]{Data: strings.ToUpper(input)}
	close(ch)
	return ch, nil
}

type mockAddPrefix struct {
	prefix string
}

func (m *mockAddPrefix) Invoke(ctx context.Context, input string, opts ...Option) (string, error) {
	return m.prefix + input, nil
}

func (m *mockAddPrefix) Batch(ctx context.Context, inputs []string, opts ...Option) ([]string, error) {
	results := make([]string, len(inputs))
	for i, input := range inputs {
		results[i] = m.prefix + input
	}
	return results, nil
}

func (m *mockAddPrefix) Stream(ctx context.Context, input string, opts ...Option) (<-chan StreamEvent[string], error) {
	ch := make(chan StreamEvent[string], 1)
	ch <- StreamEvent[string]{Data: m.prefix + input}
	close(ch)
	return ch, nil
}

// Tests

func TestChainPipe(t *testing.T) {
	// 创建链: input -> uppercase -> add prefix
	chain := NewChain[string, string](&mockUppercase{}).
		Pipe(&mockAddPrefix{prefix: "Result: "})
	
	ctx := context.Background()
	result, err := chain.Invoke(ctx, "hello")
	
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	
	expected := "Result: HELLO"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestParallel(t *testing.T) {
	// 并行执行多个 runnable
	parallel := Parallel[string, string](
		&mockUppercase{},
		&mockAddPrefix{prefix: ">>> "},
	)
	
	ctx := context.Background()
	results, err := parallel.Invoke(ctx, "test")
	
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	
	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}
	
	if results[0] != "TEST" {
		t.Errorf("expected 'TEST', got %q", results[0])
	}
	
	if results[1] != ">>> test" {
		t.Errorf("expected '>>> test', got %q", results[1])
	}
}

func TestParallelMap(t *testing.T) {
	parallelMap := ParallelMap(map[string]Runnable[string, string]{
		"upper": &mockUppercase{},
		"prefix": &mockAddPrefix{prefix: "==> "},
	})
	
	ctx := context.Background()
	results, err := parallelMap.Invoke(ctx, "test")
	
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	
	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}
	
	if results["upper"] != "TEST" {
		t.Errorf("expected 'TEST', got %q", results["upper"])
	}
	
	if results["prefix"] != "==> test" {
		t.Errorf("expected '==> test', got %q", results["prefix"])
	}
}

func TestRoute(t *testing.T) {
	router := Route(
		func(input string) string {
			if strings.HasPrefix(input, "?") {
				return "question"
			}
			return "statement"
		},
		map[string]Runnable[string, string]{
			"question":  &mockAddPrefix{prefix: "Q: "},
			"statement": &mockAddPrefix{prefix: "S: "},
		},
	)
	
	ctx := context.Background()
	
	t.Run("question route", func(t *testing.T) {
		result, err := router.Invoke(ctx, "?How are you")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		
		if result != "Q: ?How are you" {
			t.Errorf("expected 'Q: ?How are you', got %q", result)
		}
	})
	
	t.Run("statement route", func(t *testing.T) {
		result, err := router.Invoke(ctx, "Hello")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		
		if result != "S: Hello" {
			t.Errorf("expected 'S: Hello', got %q", result)
		}
	})
}

func TestFallback(t *testing.T) {
	// 创建会失败的 runnable
	failing := Map(func(ctx context.Context, input string) (string, error) {
		return "", fmt.Errorf("primary failed")
	})
	
	success := &mockAddPrefix{prefix: "fallback: "}
	
	fallback := Fallback(failing, success)
	
	ctx := context.Background()
	result, err := fallback.Invoke(ctx, "test")
	
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	
	if result != "fallback: test" {
		t.Errorf("expected 'fallback: test', got %q", result)
	}
}

func TestRetry(t *testing.T) {
	attempts := 0
	unstable := Map(func(ctx context.Context, input string) (string, error) {
		attempts++
		if attempts < 2 {
			return "", fmt.Errorf("temporary error")
		}
		return "success", nil
	})
	
	retry := Retry(unstable, 3, time.Millisecond)
	
	ctx := context.Background()
	result, err := retry.Invoke(ctx, "test")
	
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	
	if result != "success" {
		t.Errorf("expected 'success', got %q", result)
	}
	
	if attempts != 2 {
		t.Errorf("expected 2 attempts, got %d", attempts)
	}
}

func TestMap(t *testing.T) {
	doubler := Map(func(ctx context.Context, input int) (int, error) {
		return input * 2, nil
	})
	
	ctx := context.Background()
	result, err := doubler.Invoke(ctx, 5)
	
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	
	if result != 10 {
		t.Errorf("expected 10, got %d", result)
	}
}

func TestFilter(t *testing.T) {
	notEmpty := Filter(func(s string) bool {
		return len(s) > 0
	})
	
	ctx := context.Background()
	
	t.Run("pass filter", func(t *testing.T) {
		result, err := notEmpty.Invoke(ctx, "hello")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		
		if result != "hello" {
			t.Errorf("expected 'hello', got %q", result)
		}
	})
	
	t.Run("fail filter", func(t *testing.T) {
		_, err := notEmpty.Invoke(ctx, "")
		if err == nil {
			t.Error("expected error, got nil")
		}
	})
}

func TestChainMetadata(t *testing.T) {
	chain := NewChain[string, string](&mockUppercase{}).
		WithName("test-chain").
		WithMetadata("version", "1.0").
		WithMetadata("author", "test")
	
	if chain.name != "test-chain" {
		t.Errorf("expected name 'test-chain', got %q", chain.name)
	}
	
	if chain.metadata["version"] != "1.0" {
		t.Errorf("expected version '1.0', got %v", chain.metadata["version"])
	}
	
	if chain.metadata["author"] != "test" {
		t.Errorf("expected author 'test', got %v", chain.metadata["author"])
	}
}

func TestComplexChain(t *testing.T) {
	// 复杂链示例：input -> uppercase -> route -> add prefix
	upper := &mockUppercase{}
	
	router := Route(
		func(input string) string {
			if strings.Contains(input, "URGENT") {
				return "urgent"
			}
			return "normal"
		},
		map[string]Runnable[string, string]{
			"urgent": &mockAddPrefix{prefix: "⚠️ "},
			"normal": &mockAddPrefix{prefix: "✓ "},
		},
	)
	
	chain := NewChain[string, string](upper).Pipe(router)
	
	ctx := context.Background()
	
	t.Run("urgent message", func(t *testing.T) {
		result, err := chain.Invoke(ctx, "URGENT: help")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		
		if result != "⚠️ URGENT: HELP" {
			t.Errorf("expected '⚠️ URGENT: HELP', got %q", result)
		}
	})
	
	t.Run("normal message", func(t *testing.T) {
		result, err := chain.Invoke(ctx, "hello")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		
		if result != "✓ HELLO" {
			t.Errorf("expected '✓ HELLO', got %q", result)
		}
	})
}
