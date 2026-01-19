package main

import (
	"fmt"
	"time"

	"github.com/zhucl121/langchain-go/core/agents"
)

func main() {
	fmt.Println("=== LangChain-Go Agent 流式执行示例 ===\n")

	// 示例 1: 基础 Agent 流式执行
	example1BasicAgentStream()

	// 示例 2: 多步骤执行可视化
	example2MultiStepVisualization()
}

// 示例 1: 基础 Agent 流式执行
func example1BasicAgentStream() {
	fmt.Println("## 示例 1: 基础 Agent 流式执行")

	// 模拟 Agent 执行
	streamCh := simulateAgentExecution("计算 2+2 并返回结果", 2)

	fmt.Println("\n### 执行过程:")
	for event := range streamCh {
		printAgentEvent(event)
	}

	fmt.Println()
}

// 示例 2: 多步骤执行可视化
func example2MultiStepVisualization() {
	fmt.Println("## 示例 2: 多步骤 Agent 执行可视化")

	// 模拟复杂的 Agent 执行
	streamCh := simulateAgentExecution("分析用户反馈并生成报告", 4)

	fmt.Println("\n### 执行时间线:")
	fmt.Println("┌─────────────────────────────────────────────────────────┐")

	startTime := time.Now()
	var stepCount int

	for event := range streamCh {
		elapsed := time.Since(startTime).Milliseconds()

		switch event.Type {
		case agents.EventTypeStart:
			fmt.Printf("│ [%4dms] 🚀 开始执行\n", elapsed)

		case agents.EventTypeStep:
			stepCount = event.Step
			fmt.Printf("│ [%4dms] ├─ 步骤 %d\n", elapsed, stepCount)

		case agents.EventTypeToolCall:
			if event.Action != nil {
				fmt.Printf("│ [%4dms] │  ├─ 🔧 工具: %s\n", elapsed, event.Action.Tool)
			}

		case agents.EventTypeToolResult:
			fmt.Printf("│ [%4dms] │  └─ ✓ 完成\n", elapsed)
			if len(event.Observation) > 0 {
				obs := event.Observation
				if len(obs) > 40 {
					obs = obs[:40] + "..."
				}
				fmt.Printf("│ [%4dms] │     观察: %s\n", elapsed, obs)
			}

		case agents.EventTypeFinish:
			fmt.Printf("│ [%4dms] ✅ 任务完成\n", elapsed)
			if event.Action != nil && len(event.Action.FinalAnswer) > 0 {
				fmt.Printf("│         答案: %s\n", event.Action.FinalAnswer)
			}

		case agents.EventTypeError:
			fmt.Printf("│ [%4dms] ❌ 错误: %v\n", elapsed, event.Error)
		}
	}

	fmt.Println("└─────────────────────────────────────────────────────────┘")
	fmt.Printf("\n总耗时: %dms\n\n", time.Since(startTime).Milliseconds())
}

// simulateAgentExecution 模拟 Agent 执行过程
func simulateAgentExecution(input string, steps int) <-chan agents.AgentStreamEvent {
	out := make(chan agents.AgentStreamEvent, 100)

	go func() {
		defer close(out)

		// 开始事件
		out <- agents.AgentStreamEvent{
			Type:      agents.EventTypeStart,
			Timestamp: time.Now(),
		}

		time.Sleep(50 * time.Millisecond)

		// 执行步骤
		for i := 0; i < steps; i++ {
			step := i + 1

			// 步骤事件
			out <- agents.AgentStreamEvent{
				Type:      agents.EventTypeStep,
				Step:      step,
				Timestamp: time.Now(),
			}

			time.Sleep(30 * time.Millisecond)

			// 工具调用
			action := &agents.AgentAction{
				Type: agents.ActionToolCall,
				Tool: fmt.Sprintf("tool_%d", step),
				ToolInput: map[string]any{
					"query": input,
				},
			}

			// 工具调用事件
			out <- agents.AgentStreamEvent{
				Type:      agents.EventTypeToolCall,
				Step:      step,
				Action:    action,
				Timestamp: time.Now(),
			}

			time.Sleep(100 * time.Millisecond)

			// 工具结果事件
			out <- agents.AgentStreamEvent{
				Type:        agents.EventTypeToolResult,
				Step:        step,
				Action:      action,
				Observation: fmt.Sprintf("从工具 %d 获得了有用的信息", step),
				Timestamp:   time.Now(),
			}

			time.Sleep(30 * time.Millisecond)
		}

		// 完成事件
		finishAction := &agents.AgentAction{
			Type:        agents.ActionFinish,
			FinalAnswer: "任务已成功完成！基于所有工具的输出，我生成了最终答案。",
		}

		out <- agents.AgentStreamEvent{
			Type:        agents.EventTypeFinish,
			Action:      finishAction,
			Observation: finishAction.FinalAnswer,
			Timestamp:   time.Now(),
		}
	}()

	return out
}

// printAgentEvent 打印 Agent 事件（详细版本）
func printAgentEvent(event agents.AgentStreamEvent) {
	prefix := ""
	if event.Step > 0 {
		prefix = fmt.Sprintf("[步骤 %d]", event.Step)
	}

	switch event.Type {
	case agents.EventTypeStart:
		fmt.Printf("🚀 开始执行\n")

	case agents.EventTypeStep:
		fmt.Printf("%s 📍 步骤开始\n", prefix)

	case agents.EventTypeToolCall:
		if event.Action != nil {
			fmt.Printf("%s 🔧 调用工具: %s\n", prefix, event.Action.Tool)
		}

	case agents.EventTypeToolResult:
		fmt.Printf("%s ✓ 工具完成\n", prefix)
		if len(event.Observation) > 0 {
			fmt.Printf("%s    观察: %s\n", prefix, event.Observation)
		}

	case agents.EventTypeFinish:
		fmt.Printf("✅ 任务完成!\n")
		if event.Action != nil && len(event.Action.FinalAnswer) > 0 {
			fmt.Printf("   最终答案: %s\n", event.Action.FinalAnswer)
		}

	case agents.EventTypeError:
		fmt.Printf("❌ 错误: %v\n", event.Error)
	}
}
