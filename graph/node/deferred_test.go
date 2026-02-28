package node

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"
)

type mapReduceState struct {
	Input   string
	Results []string
	Final   string
}

func TestDeferredFunctionNode_BasicWait(t *testing.T) {
	reduceFn := func(ctx context.Context, s mapReduceState) (mapReduceState, error) {
		results, _ := GetBranchResultsFromContext[mapReduceState](ctx, "reduce")
		for name, r := range results {
			if r.Err == nil {
				s.Results = append(s.Results, fmt.Sprintf("%s:%s", name, r.State.Final))
			}
		}
		return s, nil
	}

	branches := []string{"w1", "w2", "w3"}
	deferred := NewDeferredFunctionNode("reduce", reduceFn, branches, DeferredNodeConfig{
		Timeout: 5 * time.Second,
	})

	ctx := context.Background()
	initialState := mapReduceState{Input: "test"}

	// 模拟 3 个并行分支在 goroutine 中注册结果
	for _, b := range branches {
		b := b
		go func() {
			time.Sleep(10 * time.Millisecond)
			result := mapReduceState{Final: "result-" + b}
			deferred.RegisterBranchResult(b, result, nil, 10*time.Millisecond)
		}()
	}

	result, err := deferred.Invoke(ctx, initialState)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Results) != 3 {
		t.Errorf("want 3 branch results, got %d", len(result.Results))
	}
}

func TestDeferredFunctionNode_Timeout(t *testing.T) {
	reduceFn := func(ctx context.Context, s mapReduceState) (mapReduceState, error) {
		return s, nil
	}

	deferred := NewDeferredFunctionNode("reduce", reduceFn,
		[]string{"slow-branch"},
		DeferredNodeConfig{Timeout: 50 * time.Millisecond},
	)

	// 不注册任何分支结果 -> 应该超时
	ctx := context.Background()
	_, err := deferred.Invoke(ctx, mapReduceState{})
	if err == nil {
		t.Fatal("expected timeout error")
	}
	t.Logf("got expected timeout: %v", err)
}

func TestDeferredFunctionNode_ContextCancel(t *testing.T) {
	reduceFn := func(ctx context.Context, s mapReduceState) (mapReduceState, error) {
		return s, nil
	}

	deferred := NewDeferredFunctionNode("reduce", reduceFn,
		[]string{"branch"},
		DeferredNodeConfig{},
	)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // 立即取消

	_, err := deferred.Invoke(ctx, mapReduceState{})
	if err == nil {
		t.Fatal("expected context cancellation error")
	}
}

func TestDeferredFunctionNode_FailFast(t *testing.T) {
	var reduceCalled atomic.Bool
	reduceFn := func(ctx context.Context, s mapReduceState) (mapReduceState, error) {
		reduceCalled.Store(true)
		return s, nil
	}

	deferred := NewDeferredFunctionNode("reduce", reduceFn,
		[]string{"w1", "w2"},
		DeferredNodeConfig{
			Timeout:  2 * time.Second,
			FailFast: true,
		},
	)

	ctx := context.Background()

	// w1 失败
	deferred.RegisterBranchResult("w1", mapReduceState{}, fmt.Errorf("w1 failed"), 0)
	// w2 成功
	deferred.RegisterBranchResult("w2", mapReduceState{Final: "ok"}, nil, 0)

	_, err := deferred.Invoke(ctx, mapReduceState{})
	if err == nil {
		t.Fatal("expected error due to FailFast")
	}
	t.Logf("got expected FailFast error: %v", err)
}

func TestDeferredFunctionNode_MinBranches(t *testing.T) {
	var callCount atomic.Int32
	reduceFn := func(ctx context.Context, s mapReduceState) (mapReduceState, error) {
		callCount.Add(1)
		results, _ := GetBranchResultsFromContext[mapReduceState](ctx, "reduce")
		s.Final = fmt.Sprintf("completed:%d", len(results))
		return s, nil
	}

	deferred := NewDeferredFunctionNode("reduce", reduceFn,
		[]string{"w1", "w2", "w3"},
		DeferredNodeConfig{
			Timeout:     2 * time.Second,
			MinBranches: 2, // 只需要 2 个分支完成
		},
	)

	ctx := context.Background()

	// 只注册 2 个分支
	deferred.RegisterBranchResult("w1", mapReduceState{Final: "r1"}, nil, 0)
	deferred.RegisterBranchResult("w2", mapReduceState{Final: "r2"}, nil, 0)
	// w3 未注册

	result, err := deferred.Invoke(ctx, mapReduceState{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	t.Logf("result with min branches: %s", result.Final)
}

func TestDeferredFunctionNode_Reset(t *testing.T) {
	var callCount atomic.Int32
	reduceFn := func(ctx context.Context, s mapReduceState) (mapReduceState, error) {
		callCount.Add(1)
		return s, nil
	}

	deferred := NewDeferredFunctionNode("reduce", reduceFn,
		[]string{"w1"},
		DeferredNodeConfig{Timeout: 2 * time.Second},
	)

	ctx := context.Background()

	// 第一轮
	deferred.RegisterBranchResult("w1", mapReduceState{Final: "round1"}, nil, 0)
	_, err := deferred.Invoke(ctx, mapReduceState{})
	if err != nil {
		t.Fatalf("round 1 failed: %v", err)
	}

	// 重置，第二轮
	deferred.Reset()
	deferred.RegisterBranchResult("w1", mapReduceState{Final: "round2"}, nil, 0)
	_, err = deferred.Invoke(ctx, mapReduceState{})
	if err != nil {
		t.Fatalf("round 2 failed: %v", err)
	}

	if callCount.Load() != 2 {
		t.Errorf("want 2 reduce calls, got %d", callCount.Load())
	}
}

func TestDeferredFunctionNode_Validate(t *testing.T) {
	tests := []struct {
		name     string
		branches []string
		fn       NodeFunc[mapReduceState]
		wantErr  bool
	}{
		{"valid", []string{"b1"}, func(ctx context.Context, s mapReduceState) (mapReduceState, error) { return s, nil }, false},
		{"no branches", []string{}, func(ctx context.Context, s mapReduceState) (mapReduceState, error) { return s, nil }, true},
		{"nil fn", []string{"b1"}, nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewDeferredFunctionNode("test", tt.fn, tt.branches, DeferredNodeConfig{})
			err := d.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMapReduceRunner(t *testing.T) {
	runner := NewMapReduceRunner[mapReduceState](
		[]string{"w1", "w2", "w3"},
		DeferredNodeConfig{Timeout: 5 * time.Second},
	)

	workers := map[string]NodeFunc[mapReduceState]{
		"w1": func(ctx context.Context, s mapReduceState) (mapReduceState, error) {
			s.Final = "result-w1"
			return s, nil
		},
		"w2": func(ctx context.Context, s mapReduceState) (mapReduceState, error) {
			time.Sleep(5 * time.Millisecond)
			s.Final = "result-w2"
			return s, nil
		},
		"w3": func(ctx context.Context, s mapReduceState) (mapReduceState, error) {
			s.Final = "result-w3"
			return s, nil
		},
	}

	reduceFn := func(ctx context.Context, s mapReduceState) (mapReduceState, error) {
		results, _ := GetBranchResultsFromContext[mapReduceState](ctx, "map-reduce")
		for name, r := range results {
			if r.Err == nil {
				s.Results = append(s.Results, fmt.Sprintf("%s=%s", name, r.State.Final))
			}
		}
		s.Final = fmt.Sprintf("aggregated(%d branches)", len(results))
		return s, nil
	}

	ctx := context.Background()
	result, err := runner.Run(ctx, mapReduceState{Input: "input"}, workers, reduceFn)
	if err != nil {
		t.Fatalf("MapReduce failed: %v", err)
	}

	if len(result.Results) != 3 {
		t.Errorf("want 3 results, got %d: %v", len(result.Results), result.Results)
	}
	t.Logf("MapReduce result: %s, items: %v", result.Final, result.Results)
}
