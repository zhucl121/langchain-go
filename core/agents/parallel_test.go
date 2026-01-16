package agents_test

import (
	"context"
	"fmt"
	"testing"
	"time"
	
	"langchain-go/core/agents"
	"langchain-go/core/tools"
	"langchain-go/pkg/types"
)

// mockSlowTool 是一个模拟的慢速工具（用于测试并行执行）。
type mockSlowTool struct {
	name     string
	delay    time.Duration
	result   string
	executed bool
}

func (m *mockSlowTool) GetName() string {
	return m.name
}

func (m *mockSlowTool) GetDescription() string {
	return fmt.Sprintf("A slow tool that takes %v to execute", m.delay)
}

func (m *mockSlowTool) GetParameters() types.Schema {
	return types.Schema{
		Type: "object",
		Properties: map[string]types.Schema{
			"input": {Type: "string"},
		},
	}
}

func (m *mockSlowTool) Execute(ctx context.Context, input map[string]any) (any, error) {
	// 模拟耗时操作
	select {
	case <-time.After(m.delay):
		m.executed = true
		return m.result, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (m *mockSlowTool) ToTypesTool() types.Tool {
	return types.Tool{
		Name:        m.name,
		Description: m.GetDescription(),
	}
}

// mockAgent 是模拟的 Agent（用于测试）。
type mockAgent struct {
	tools []tools.Tool
}

func (m *mockAgent) Plan(ctx context.Context, input string, history []agents.AgentStep) (*agents.AgentAction, error) {
	// 简单的模拟实现
	if len(history) >= 2 {
		return &agents.AgentAction{
			Type:        agents.ActionFinish,
			FinalAnswer: "Task completed",
		}, nil
	}
	
	if len(m.tools) > 0 {
		return &agents.AgentAction{
			Type:      agents.ActionToolCall,
			Tool:      m.tools[0].GetName(),
			ToolInput: map[string]any{"input": input},
		}, nil
	}
	
	return &agents.AgentAction{
		Type:        agents.ActionFinish,
		FinalAnswer: "No tools available",
	}, nil
}

func (m *mockAgent) GetType() agents.AgentType {
	return agents.AgentTypeReAct
}

func (m *mockAgent) GetTools() []tools.Tool {
	return m.tools
}

func TestParallelExecutor_RunParallel(t *testing.T) {
	// 创建模拟工具
	tool1 := &mockSlowTool{
		name:   "tool1",
		delay:  100 * time.Millisecond,
		result: "result1",
	}
	tool2 := &mockSlowTool{
		name:   "tool2",
		delay:  100 * time.Millisecond,
		result: "result2",
	}
	tool3 := &mockSlowTool{
		name:   "tool3",
		delay:  100 * time.Millisecond,
		result: "result3",
	}

	// 创建基础执行器
	agent := &mockAgent{
		tools: []tools.Tool{tool1, tool2, tool3},
	}
	
	toolExecutor := tools.NewToolExecutor(tools.ToolExecutorConfig{
		Tools: []tools.Tool{tool1, tool2, tool3},
	})
	
	baseExecutor := agents.NewAgentExecutor(agents.AgentExecutorConfig{
		Agent:        agent,
		ToolExecutor: toolExecutor,
		MaxSteps:     10,
	})

	// 创建并行执行器
	parallelExecutor := agents.NewParallelExecutor(agents.ParallelExecutorConfig{
		Executor:       baseExecutor,
		MaxConcurrency: 3,
		Timeout:        5 * time.Second,
	})

	// 创建要执行的行动
	actions := []*agents.AgentAction{
		{
			Type:      agents.ActionToolCall,
			Tool:      "tool1",
			ToolInput: map[string]any{"input": "test1"},
		},
		{
			Type:      agents.ActionToolCall,
			Tool:      "tool2",
			ToolInput: map[string]any{"input": "test2"},
		},
		{
			Type:      agents.ActionToolCall,
			Tool:      "tool3",
			ToolInput: map[string]any{"input": "test3"},
		},
	}

	// 测试并行执行
	ctx := context.Background()
	startTime := time.Now()
	
	results, err := parallelExecutor.RunParallel(ctx, actions)
	
	duration := time.Since(startTime)

	// 验证结果
	if err != nil {
		t.Fatalf("RunParallel failed: %v", err)
	}

	if len(results) != 3 {
		t.Errorf("Expected 3 results, got %d", len(results))
	}

	// 验证并行执行（应该比顺序执行快）
	// 顺序执行需要 300ms，并行执行应该接近 100ms
	if duration > 200*time.Millisecond {
		t.Errorf("Parallel execution too slow: %v (expected ~100ms)", duration)
	}

	// 验证每个工具都被执行
	if !tool1.executed || !tool2.executed || !tool3.executed {
		t.Error("Not all tools were executed")
	}

	// 验证结果顺序
	for i, result := range results {
		if result.Index != i {
			t.Errorf("Result %d has wrong index: %d", i, result.Index)
		}
		if result.Error != nil {
			t.Errorf("Result %d has error: %v", i, result.Error)
		}
	}
}

func TestParallelExecutor_Timeout(t *testing.T) {
	// 创建一个非常慢的工具
	slowTool := &mockSlowTool{
		name:   "slow_tool",
		delay:  5 * time.Second,
		result: "result",
	}

	agent := &mockAgent{
		tools: []tools.Tool{slowTool},
	}
	
	toolExecutor := tools.NewToolExecutor(tools.ToolExecutorConfig{
		Tools: []tools.Tool{slowTool},
	})
	
	baseExecutor := agents.NewAgentExecutor(agents.AgentExecutorConfig{
		Agent:        agent,
		ToolExecutor: toolExecutor,
		MaxSteps:     10,
	})

	// 创建并行执行器，设置短超时
	parallelExecutor := agents.NewParallelExecutor(agents.ParallelExecutorConfig{
		Executor:       baseExecutor,
		MaxConcurrency: 1,
		Timeout:        100 * time.Millisecond,
	})

	actions := []*agents.AgentAction{
		{
			Type:      agents.ActionToolCall,
			Tool:      "slow_tool",
			ToolInput: map[string]any{"input": "test"},
		},
	}

	ctx := context.Background()
	results, _ := parallelExecutor.RunParallel(ctx, actions)

	// 验证超时错误
	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	if results[0].Error == nil {
		t.Error("Expected timeout error, got nil")
	}

	if results[0].Duration < 100*time.Millisecond {
		t.Errorf("Timeout happened too early: %v", results[0].Duration)
	}
}

func TestParallelExecutor_MaxConcurrency(t *testing.T) {
	// 创建多个工具
	testTools := make([]tools.Tool, 10)
	for i := 0; i < 10; i++ {
		testTools[i] = &mockSlowTool{
			name:   fmt.Sprintf("tool%d", i),
			delay:  50 * time.Millisecond,
			result: fmt.Sprintf("result%d", i),
		}
	}

	agent := &mockAgent{
		tools: testTools,
	}
	
	toolExecutor := tools.NewToolExecutor(tools.ToolExecutorConfig{
		Tools: testTools,
	})
	
	baseExecutor := agents.NewAgentExecutor(agents.AgentExecutorConfig{
		Agent:        agent,
		ToolExecutor: toolExecutor,
		MaxSteps:     10,
	})

	// 限制并发数为 2
	parallelExecutor := agents.NewParallelExecutor(agents.ParallelExecutorConfig{
		Executor:       baseExecutor,
		MaxConcurrency: 2,
		Timeout:        5 * time.Second,
	})

	// 创建 10 个行动
	actions := make([]*agents.AgentAction, 10)
	for i := 0; i < 10; i++ {
		actions[i] = &agents.AgentAction{
			Type:      agents.ActionToolCall,
			Tool:      fmt.Sprintf("tool%d", i),
			ToolInput: map[string]any{"input": fmt.Sprintf("test%d", i)},
		}
	}

	ctx := context.Background()
	startTime := time.Now()
	
	results, err := parallelExecutor.RunParallel(ctx, actions)
	
	duration := time.Since(startTime)

	if err != nil {
		t.Fatalf("RunParallel failed: %v", err)
	}

	if len(results) != 10 {
		t.Errorf("Expected 10 results, got %d", len(results))
	}

	// 验证执行时间
	// 10 个工具，每个 50ms，并发数为 2
	// 预期时间约为 250ms (5 批次 * 50ms)
	expectedMin := 200 * time.Millisecond
	expectedMax := 350 * time.Millisecond
	
	if duration < expectedMin || duration > expectedMax {
		t.Errorf("Execution time %v not in expected range [%v, %v]", duration, expectedMin, expectedMax)
	}
}

func TestParallelExecutor_DefaultMergeStrategy(t *testing.T) {
	results := []agents.ParallelToolResult{
		{
			Action: &agents.AgentAction{
				Tool: "tool1",
			},
			Observation: "result1",
			Duration:    100 * time.Millisecond,
		},
		{
			Action: &agents.AgentAction{
				Tool: "tool2",
			},
			Observation: "result2",
			Duration:    150 * time.Millisecond,
		},
	}

	merged := agents.DefaultMergeStrategy(results)

	// 验证合并结果包含所有信息
	if merged == "" {
		t.Error("Merged result is empty")
	}

	t.Logf("Merged result:\n%s", merged)
}

func TestAgentExecutor_WithParallelExecution(t *testing.T) {
	tool := &mockSlowTool{
		name:   "test_tool",
		delay:  50 * time.Millisecond,
		result: "test_result",
	}

	agent := &mockAgent{
		tools: []tools.Tool{tool},
	}
	
	toolExecutor := tools.NewToolExecutor(tools.ToolExecutorConfig{
		Tools: []tools.Tool{tool},
	})
	
	baseExecutor := agents.NewAgentExecutor(agents.AgentExecutorConfig{
		Agent:        agent,
		ToolExecutor: toolExecutor,
		MaxSteps:     10,
	})

	// 启用并行执行
	executorWithParallel := baseExecutor.WithParallelExecution(3, 5*time.Second)

	if executorWithParallel == nil {
		t.Fatal("WithParallelExecution returned nil")
	}

	if executorWithParallel.GetParallelExecutor() == nil {
		t.Error("Parallel executor is nil")
	}

	// 测试基本执行
	ctx := context.Background()
	result, err := executorWithParallel.RunWithParallelTools(ctx, "test input")

	if err != nil {
		t.Fatalf("RunWithParallelTools failed: %v", err)
	}

	if result == nil {
		t.Error("Result is nil")
	}
}

func TestParallelExecutor_EmptyActions(t *testing.T) {
	agent := &mockAgent{
		tools: []tools.Tool{},
	}
	
	toolExecutor := tools.NewToolExecutor(tools.ToolExecutorConfig{
		Tools: []tools.Tool{},
	})
	
	baseExecutor := agents.NewAgentExecutor(agents.AgentExecutorConfig{
		Agent:        agent,
		ToolExecutor: toolExecutor,
		MaxSteps:     10,
	})

	parallelExecutor := agents.NewParallelExecutor(agents.ParallelExecutorConfig{
		Executor:       baseExecutor,
		MaxConcurrency: 3,
		Timeout:        5 * time.Second,
	})

	ctx := context.Background()
	results, err := parallelExecutor.RunParallel(ctx, []*agents.AgentAction{})

	if err != nil {
		t.Errorf("Expected no error for empty actions, got: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("Expected 0 results, got %d", len(results))
	}
}
