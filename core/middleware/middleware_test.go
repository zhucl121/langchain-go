package middleware

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"
)

// TestMiddlewareFunc 测试函数中间件
func TestMiddlewareFunc(t *testing.T) {
	called := false
	
	mw := NewFunc(func(ctx context.Context, input any, next NextFunc) (any, error) {
		called = true
		return next(ctx, input)
	})

	handler := HandlerFunc(func(ctx context.Context, input any) (any, error) {
		return "result", nil
	})

	result, err := mw.Process(context.Background(), "input", NextFunc(handler))
	
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	if !called {
		t.Error("middleware not called")
	}

	if result != "result" {
		t.Errorf("expected 'result', got %v", result)
	}
}

// TestChain_Use 测试添加中间件
func TestChain_Use(t *testing.T) {
	chain := NewChain()

	if chain.Len() != 0 {
		t.Errorf("expected empty chain, got %d", chain.Len())
	}

	mw1 := NewFunc(func(ctx context.Context, input any, next NextFunc) (any, error) {
		return next(ctx, input)
	})

	chain.Use(mw1)

	if chain.Len() != 1 {
		t.Errorf("expected 1 middleware, got %d", chain.Len())
	}
}

// TestChain_Execute 测试执行链
func TestChain_Execute(t *testing.T) {
	chain := NewChain()
	
	order := []string{}

	mw1 := NewFunc(func(ctx context.Context, input any, next NextFunc) (any, error) {
		order = append(order, "mw1-before")
		result, err := next(ctx, input)
		order = append(order, "mw1-after")
		return result, err
	})

	mw2 := NewFunc(func(ctx context.Context, input any, next NextFunc) (any, error) {
		order = append(order, "mw2-before")
		result, err := next(ctx, input)
		order = append(order, "mw2-after")
		return result, err
	})

	chain.Use(mw1).Use(mw2)

	handler := HandlerFunc(func(ctx context.Context, input any) (any, error) {
		order = append(order, "handler")
		return "result", nil
	})

	result, err := chain.Execute(context.Background(), "input", handler)
	
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result != "result" {
		t.Errorf("expected 'result', got %v", result)
	}

	// 验证执行顺序（洋葱模型）
	expected := []string{"mw1-before", "mw2-before", "handler", "mw2-after", "mw1-after"}
	if len(order) != len(expected) {
		t.Fatalf("expected %d steps, got %d", len(expected), len(order))
	}

	for i, step := range expected {
		if order[i] != step {
			t.Errorf("step %d: expected %s, got %s", i, step, order[i])
		}
	}
}

// TestChain_Remove 测试移除中间件
func TestChain_Remove(t *testing.T) {
	chain := NewChain()

	meta := NewMetadata("test-middleware")
	mw := NewFunc(func(ctx context.Context, input any, next NextFunc) (any, error) {
		return next(ctx, input)
	})

	chain.UseWithMeta(mw, meta)

	if chain.Len() != 1 {
		t.Error("middleware not added")
	}

	removed := chain.Remove("test-middleware")
	if !removed {
		t.Error("middleware not removed")
	}

	if chain.Len() != 0 {
		t.Error("chain not empty after removal")
	}
}

// TestChain_SortByPriority 测试优先级排序
func TestChain_SortByPriority(t *testing.T) {
	chain := NewChain()

	mw1 := NewFunc(func(ctx context.Context, input any, next NextFunc) (any, error) {
		return next(ctx, input)
	})
	mw2 := NewFunc(func(ctx context.Context, input any, next NextFunc) (any, error) {
		return next(ctx, input)
	})
	mw3 := NewFunc(func(ctx context.Context, input any, next NextFunc) (any, error) {
		return next(ctx, input)
	})

	chain.UseWithMeta(mw1, NewMetadata("mw1").WithPriority(30))
	chain.UseWithMeta(mw2, NewMetadata("mw2").WithPriority(10))
	chain.UseWithMeta(mw3, NewMetadata("mw3").WithPriority(20))

	chain.SortByPriority()

	middlewares := chain.GetMiddlewaresWithMeta()
	
	if middlewares[0].metadata.Name != "mw2" {
		t.Error("mw2 should be first")
	}
	if middlewares[1].metadata.Name != "mw3" {
		t.Error("mw3 should be second")
	}
	if middlewares[2].metadata.Name != "mw1" {
		t.Error("mw1 should be third")
	}
}

// TestChain_Clone 测试克隆
func TestChain_Clone(t *testing.T) {
	chain := NewChain()

	mw := NewFunc(func(ctx context.Context, input any, next NextFunc) (any, error) {
		return next(ctx, input)
	})

	chain.Use(mw)

	cloned := chain.Clone()

	if cloned.Len() != chain.Len() {
		t.Error("cloned chain has different length")
	}

	// 修改原链不应影响克隆
	chain.Clear()

	if cloned.Len() != 1 {
		t.Error("cloned chain affected by original")
	}
}

// TestChain_ExecuteWithRecovery 测试 panic 恢复
func TestChain_ExecuteWithRecovery(t *testing.T) {
	chain := NewChain()

	panicMiddleware := NewFunc(func(ctx context.Context, input any, next NextFunc) (any, error) {
		panic("test panic")
	})

	chain.Use(panicMiddleware)

	handler := HandlerFunc(func(ctx context.Context, input any) (any, error) {
		return "result", nil
	})

	_, err := chain.ExecuteWithRecovery(context.Background(), "input", handler)
	
	if err == nil {
		t.Fatal("expected error from panic")
	}

	if !strings.Contains(err.Error(), "panic") {
		t.Errorf("expected panic error, got: %v", err)
	}
}

// TestMiddlewareContext 测试上下文传递
func TestMiddlewareContext(t *testing.T) {
	chain := NewChain()

	mw := NewFunc(func(ctx context.Context, input any, next NextFunc) (any, error) {
		name := GetMiddlewareNameFromContext(ctx)
		if name == "" {
			t.Error("middleware name not found in context")
		}
		return next(ctx, input)
	})

	chain.UseWithMeta(mw, NewMetadata("test-mw"))

	handler := HandlerFunc(func(ctx context.Context, input any) (any, error) {
		return "result", nil
	})

	_, err := chain.Execute(context.Background(), "input", handler)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
}

// TestCompose 测试组合中间件
func TestCompose(t *testing.T) {
	order := []string{}

	mw1 := NewFunc(func(ctx context.Context, input any, next NextFunc) (any, error) {
		order = append(order, "mw1")
		return next(ctx, input)
	})

	mw2 := NewFunc(func(ctx context.Context, input any, next NextFunc) (any, error) {
		order = append(order, "mw2")
		return next(ctx, input)
	})

	composed := Compose(mw1, mw2)

	handler := HandlerFunc(func(ctx context.Context, input any) (any, error) {
		order = append(order, "handler")
		return "result", nil
	})

	_, err := composed.Process(context.Background(), "input", NextFunc(handler))
	if err != nil {
		t.Fatalf("Compose execution failed: %v", err)
	}

	if len(order) != 3 {
		t.Errorf("expected 3 steps, got %d", len(order))
	}
}

// TestMiddleware_InputTransform 测试输入转换
func TestMiddleware_InputTransform(t *testing.T) {
	chain := NewChain()

	transformMiddleware := NewFunc(func(ctx context.Context, input any, next NextFunc) (any, error) {
		// 转换输入
		str, ok := input.(string)
		if !ok {
			return nil, fmt.Errorf("input is not string")
		}
		transformed := strings.ToUpper(str)
		return next(ctx, transformed)
	})

	chain.Use(transformMiddleware)

	handler := HandlerFunc(func(ctx context.Context, input any) (any, error) {
		return input, nil
	})

	result, err := chain.Execute(context.Background(), "hello", handler)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result != "HELLO" {
		t.Errorf("expected 'HELLO', got %v", result)
	}
}

// TestMiddleware_Timing 测试性能监控中间件
func TestMiddleware_Timing(t *testing.T) {
	chain := NewChain()

	var duration time.Duration

	timingMiddleware := NewFunc(func(ctx context.Context, input any, next NextFunc) (any, error) {
		start := time.Now()
		result, err := next(ctx, input)
		duration = time.Since(start)
		return result, err
	})

	chain.Use(timingMiddleware)

	handler := HandlerFunc(func(ctx context.Context, input any) (any, error) {
		time.Sleep(10 * time.Millisecond)
		return "result", nil
	})

	_, err := chain.Execute(context.Background(), "input", handler)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if duration < 10*time.Millisecond {
		t.Errorf("duration too short: %v", duration)
	}
}
