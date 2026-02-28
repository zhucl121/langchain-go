// Package main 演示统一 Agent 创建接口（v0.7.0 新功能）。
//
// 展示功能：
//   - CreateUnifiedAgent：统一配置入口（对标 LangChain 1.0 create_agent）
//   - 预设类型快速切换（react/tool_calling/plan_execute）
//   - 带记忆的对话 Agent（InMemoryConversationMemory）
//   - ModelHooks 集成（Pre/Post Model Hooks）
//   - 选项函数 API（functional options pattern）
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/zhucl121/langchain-go/core/agents"
	"github.com/zhucl121/langchain-go/core/middleware"
	"github.com/zhucl121/langchain-go/pkg/types"
)

func main() {
	fmt.Println("=== LangChain-Go v0.7.0: 统一 Agent 创建接口演示 ===\n")

	ctx := context.Background()
	// 使用 agents 包内建的 MockChatModel（实现完整的 ChatModel 接口）
	llm := agents.NewMockChatModel()

	// ─── 1. 基础使用：CreateUnifiedAgent ─────────────────────────
	fmt.Println("【1】CreateUnifiedAgent - 统一配置 API")
	fmt.Println("  对标 LangChain 1.0 create_agent，统一所有 Agent 类型的配置入口\n")

	ca1, err := agents.CreateUnifiedAgent(agents.UnifiedAgentConfig{
		Preset:       agents.PresetReAct,
		Model:        llm,
		SystemPrompt: "你是一个专业的技术助手，擅长解答编程问题。",
		MaxSteps:     3,
		Verbose:      false,
	})
	if err != nil {
		fmt.Printf("  ❌ 创建 ReAct Agent 失败: %v\n", err)
		return
	}
	fmt.Printf("  ✅ ReAct Agent 创建成功 (类型: %s, MaxSteps: %d)\n",
		ca1.Agent.GetType(), ca1.Config.MaxSteps)

	result1, err := ca1.Run(ctx, "Go 语言的特点是什么？")
	if err != nil {
		fmt.Printf("  ⚠️ 执行: %v\n", err)
	} else {
		fmt.Printf("  💬 问题: Go 语言的特点是什么？\n")
		fmt.Printf("  🤖 回答: %s\n", result1.Output)
	}
	fmt.Println()

	// ─── 2. 选项函数 API ──────────────────────────────────────────
	fmt.Println("【2】选项函数 API - 函数式配置风格")
	fmt.Println("  使用 functional options 模式，更灵活地配置 Agent\n")

	ca2, err := agents.NewReActAgentV2(llm, nil,
		agents.WithUnifiedSystemPrompt("你是一个 Go 专家"),
		agents.WithUnifiedMaxSteps(10),
		agents.WithUnifiedVerbose(false),
	)
	if err != nil {
		fmt.Printf("  ❌ 创建失败: %v\n", err)
	} else {
		fmt.Printf("  ✅ ReAct Agent V2 创建成功\n")
		fmt.Printf("     配置 - MaxSteps: %d, SystemPrompt: '%s'\n",
			ca2.Config.MaxSteps, ca2.Config.SystemPrompt)
	}
	fmt.Println()

	// ─── 3. 带记忆的对话 Agent ────────────────────────────────────
	fmt.Println("【3】带记忆的对话 Agent - 多轮对话上下文管理")
	fmt.Println("  自动保存和加载对话历史，支持跨轮次的上下文理解\n")

	memory := agents.NewInMemoryConversationMemory()
	ca3, err := agents.NewMemoryAgentV2(llm, nil,
		agents.WithUnifiedMemory(memory),
		agents.WithUnifiedSystemPrompt("你是一个友善的编程助手，保持连贯对话"),
		agents.WithUnifiedMaxSteps(3),
	)
	if err != nil {
		fmt.Printf("  ❌ 创建带记忆的 Agent 失败: %v\n", err)
	} else {
		fmt.Printf("  ✅ 带记忆的 Agent 创建成功\n\n")

		// 第一轮对话
		result3a, _ := ca3.RunWithMemory(ctx, "Go 语言的特点是什么？")
		fmt.Printf("  第 1 轮 - 问: Go 语言的特点是什么？\n")
		if result3a != nil {
			fmt.Printf("            答: %s\n", result3a.Output)
		}
		fmt.Printf("           记忆中消息数: %d\n\n", memory.MessageCount())

		// 第二轮对话（能感知历史上下文）
		result3b, _ := ca3.RunWithMemory(ctx, "请继续详细解释")
		fmt.Printf("  第 2 轮 - 问: 请继续详细解释\n")
		if result3b != nil {
			fmt.Printf("            答: %s\n", result3b.Output)
		}
		fmt.Printf("           记忆中消息数（增长）: %d\n", memory.MessageCount())
	}
	fmt.Println()

	// ─── 4. 带 ModelHooks 的 Agent ───────────────────────────────
	fmt.Println("【4】ModelHooks - Pre/Post 模型调用钩子")
	fmt.Println("  对标 LangGraph 1.0 Pre/Post Model Hooks，精细控制模型调用\n")

	loggingHook := &loggingModelHook{name: "logging-hook"}
	hookChain := middleware.NewHookChain(loggingHook)

	ca4, err := agents.CreateUnifiedAgent(agents.UnifiedAgentConfig{
		Preset:     agents.PresetReAct,
		Model:      llm,
		ModelHooks: hookChain,
		MaxSteps:   3,
	})
	if err != nil {
		fmt.Printf("  ❌ 创建带 Hooks 的 Agent 失败: %v\n", err)
	} else {
		fmt.Printf("  ✅ 带 ModelHooks 的 Agent 创建成功\n")
		fmt.Printf("     Hook 链已注入，在 LLM 调用前后执行自定义逻辑\n")
		_ = ca4
	}
	fmt.Println()

	// ─── 5. 弹性配置（Circuit Breaker + Bulkhead）────────────────
	fmt.Println("【5】弹性配置 - Circuit Breaker + Bulkhead")
	fmt.Println("  v0.7.0 新增生产级容错机制，防止级联故障\n")

	resilience := agents.ResilienceConfig{
		CircuitBreaker: agents.CircuitBreakerConfig{
			Name:             "llm-api",
			FailureThreshold: 5,
			SuccessThreshold: 2,
			Timeout:          30 * time.Second,
		},
		Bulkhead: agents.BulkheadConfig{
			Name:          "llm-api",
			MaxConcurrent: 10,
			MaxWaitTime:   5 * time.Second,
		},
	}

	ca5, err := agents.CreateUnifiedAgent(agents.UnifiedAgentConfig{
		Preset:     agents.PresetReAct,
		Model:      llm,
		Resilience: &resilience,
		MaxSteps:   3,
	})
	if err != nil {
		fmt.Printf("  ❌ 创建带弹性配置的 Agent 失败: %v\n", err)
	} else {
		fmt.Printf("  ✅ 带 Circuit Breaker + Bulkhead 的 Agent 创建成功\n")
		fmt.Printf("     CB: FailureThreshold=%d, Bulkhead: MaxConcurrent=%d\n",
			resilience.CircuitBreaker.FailureThreshold, resilience.Bulkhead.MaxConcurrent)
		_ = ca5
	}
	fmt.Println()

	// ─── 6. 不同预设类型对比 ─────────────────────────────────────
	fmt.Println("【6】预设类型快速切换对比")
	fmt.Println("  通过修改 Preset 字段，轻松切换不同 Agent 策略\n")

	presets := []struct {
		name   string
		preset agents.AgentPreset
		desc   string
	}{
		{string(agents.PresetReAct), agents.PresetReAct, "思考+行动循环，适合通用推理"},
		{string(agents.PresetToolCalling), agents.PresetToolCalling, "原生函数调用，适合工具密集型任务"},
		{string(agents.PresetPlanExecute), agents.PresetPlanExecute, "先规划后执行，适合复杂多步任务"},
		{string(agents.PresetSelfAsk), agents.PresetSelfAsk, "递归分解问题，适合复合查询"},
	}

	for _, p := range presets {
		ca, err := agents.CreateUnifiedAgent(agents.UnifiedAgentConfig{
			Preset:   p.preset,
			Model:    llm,
			MaxSteps: 3,
		})
		if err != nil {
			fmt.Printf("  ❌ %-20s 创建失败 - %v\n", p.name, err)
			continue
		}
		fmt.Printf("  ✅ %-20s 类型: %-20s 说明: %s\n",
			p.name, ca.Agent.GetType(), p.desc)
	}
	fmt.Println()

	fmt.Println("=== 演示完成！v0.7.0 统一 Agent 创建接口展示完毕 ===")
	fmt.Println(`
  核心优势:
  1. 统一 API：一个 CreateUnifiedAgent 入口，告别各类 NewXxxAgent
  2. 类型无关：修改 Preset 字段即可切换策略，无需改写代码
  3. 中间件生态：ModelHooks + AgentMiddleware 双层扩展点
  4. 生产就绪：内置 Circuit Breaker、Bulkhead 弹性机制
  5. 对话记忆：开箱即用的多轮对话上下文管理`)
}

// loggingModelHook 演示用的日志钩子。
type loggingModelHook struct {
	name string
}

func (h *loggingModelHook) Name() string { return h.name }

func (h *loggingModelHook) BeforeModel(_ context.Context, messages []types.Message) ([]types.Message, error) {
	fmt.Printf("    [Hook:before] 调用模型前 - 消息数: %d\n", len(messages))
	return messages, nil
}

func (h *loggingModelHook) AfterModel(_ context.Context, response *types.Message) (*types.Message, error) {
	preview := response.Content
	if len(preview) > 50 {
		preview = preview[:50] + "..."
	}
	fmt.Printf("    [Hook:after]  模型响应后 - 内容: %s\n", preview)
	return response, nil
}
