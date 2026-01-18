package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/zhucl121/langchain-go/core/agents"
	"github.com/zhucl121/langchain-go/core/chat"
	"github.com/zhucl121/langchain-go/core/runnable"
	"github.com/zhucl121/langchain-go/core/tools"
	"github.com/zhucl121/langchain-go/pkg/types"
)

// 创建一个简单的 Mock ChatModel 用于演示
type DemoChatModel struct {
	*chat.BaseChatModel
	callCount int
}

func (d *DemoChatModel) Invoke(ctx context.Context, messages []types.Message, opts ...runnable.Option) (types.Message, error) {
	// 简化演示：根据调用次数返回不同响应
	d.callCount++

	// 查看最后一条消息
	if len(messages) > 0 {
		lastMsg := messages[len(messages)-1]
		content := strings.ToLower(lastMsg.Content)

		// 规划阶段
		if strings.Contains(content, "plan") || strings.Contains(content, "task:") {
			return types.NewAssistantMessage(`Here's the execution plan:
1. Search for Tokyo population data
2. Extract the population number
3. Calculate 10% of the population
4. Format and present the final result`), nil
		}

		// 步骤 1: 搜索
		if strings.Contains(content, "step 1") || strings.Contains(content, "tokyo population") {
			return types.NewAssistantMessage("Using search tool to find Tokyo's population"), nil
		}

		// 步骤 2: 提取数据
		if strings.Contains(content, "step 2") || strings.Contains(content, "extract") {
			return types.NewAssistantMessage("Extracting the population number from search results"), nil
		}

		// 步骤 3: 计算
		if strings.Contains(content, "step 3") || strings.Contains(content, "calculate") {
			return types.NewAssistantMessage("13900000 * 0.10"), nil
		}

		// 步骤 4: 格式化结果
		if strings.Contains(content, "step 4") || strings.Contains(content, "format") {
			return types.NewAssistantMessage("Formatting the final result"), nil
		}

		// 最终答案
		if strings.Contains(content, "final answer") || strings.Contains(content, "comprehensive") {
			return types.NewAssistantMessage(
				"Based on the search results and calculations:\n\n" +
					"Tokyo's population is approximately 13.9 million people.\n" +
					"10% of Tokyo's population is 1.39 million people (1,390,000).\n\n" +
					"This represents a significant portion of one of the world's most populous metropolitan areas."), nil
		}
	}

	return types.NewAssistantMessage("I understand"), nil
}

func (d *DemoChatModel) Batch(ctx context.Context, inputs [][]types.Message, opts ...runnable.Option) ([]types.Message, error) {
	results := make([]types.Message, len(inputs))
	for i, messages := range inputs {
		msg, err := d.Invoke(ctx, messages, opts...)
		if err != nil {
			return nil, err
		}
		results[i] = msg
	}
	return results, nil
}

func (d *DemoChatModel) Stream(ctx context.Context, messages []types.Message, opts ...runnable.Option) (<-chan runnable.StreamEvent[types.Message], error) {
	ch := make(chan runnable.StreamEvent[types.Message], 1)
	go func() {
		defer close(ch)
		msg, _ := d.Invoke(ctx, messages, opts...)
		ch <- runnable.StreamEvent[types.Message]{Data: msg}
	}()
	return ch, nil
}

func (d *DemoChatModel) BindTools(tools []types.Tool) chat.ChatModel {
	d.SetBoundTools(tools)
	return d
}

func (d *DemoChatModel) WithStructuredOutput(schema types.Schema) chat.ChatModel {
	d.SetOutputSchema(schema)
	return d
}

func (d *DemoChatModel) WithConfig(config *types.Config) runnable.Runnable[[]types.Message, types.Message] {
	d.SetConfig(config)
	return d
}

func (d *DemoChatModel) WithRetry(policy types.RetryPolicy) runnable.Runnable[[]types.Message, types.Message] {
	return runnable.NewRetryRunnable[[]types.Message, types.Message](d, policy)
}

func (d *DemoChatModel) WithFallbacks(fallbacks ...runnable.Runnable[[]types.Message, types.Message]) runnable.Runnable[[]types.Message, types.Message] {
	return runnable.NewFallbackRunnable[[]types.Message, types.Message](d, fallbacks)
}

func main() {
	fmt.Println("=== Plan-and-Execute Agent Demo ===")

	// 1. 创建 Mock LLM
	llm := &DemoChatModel{
		BaseChatModel: chat.NewBaseChatModel("demo-model", "demo-provider"),
	}

	// 2. 创建工具
	searchTool := tools.NewFunctionTool(tools.FunctionToolConfig{
		Name:        "search",
		Description: "Search the internet for information",
		Fn: func(ctx context.Context, input map[string]any) (any, error) {
			fmt.Println("🔍 [Tool: search] Searching for information...")
			return "Tokyo is the capital of Japan with a population of approximately 13.9 million people in the city proper.", nil
		},
	})

	calculatorTool := tools.NewFunctionTool(tools.FunctionToolConfig{
		Name:        "calculator",
		Description: "Perform mathematical calculations",
		Fn: func(ctx context.Context, input map[string]any) (any, error) {
			fmt.Println("🧮 [Tool: calculator] Performing calculation...")
			expression := input["input"].(string)
			// 简单演示：解析并计算
			if strings.Contains(expression, "13900000 * 0.10") {
				return "1390000", nil
			}
			return "Result: " + expression, nil
		},
	})

	// 3. 配置 Plan-and-Execute Agent
	config := agents.PlanAndExecuteConfig{
		LLM:          llm,
		Tools:        []tools.Tool{searchTool, calculatorTool},
		EnableReplan: false,
		MaxSteps:     10,
		Verbose:      true,
	}

	agent := agents.NewPlanAndExecuteAgent(config)

	fmt.Printf("✅ Agent Type: %s\n\n", agent.GetType())

	// 4. 创建执行器
	executor := agents.NewExecutor(agent).
		WithMaxSteps(10).
		WithVerbose(true)

	// 5. 执行任务
	ctx := context.Background()
	task := "Search for the population of Tokyo and calculate what 10% of it is"

	fmt.Printf("📋 Task: %s\n\n", task)
	fmt.Println("--- Execution Started ---")

	result, err := executor.Execute(ctx, task)

	fmt.Println("\n--- Execution Completed ---")

	// 6. 处理结果
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		return
	}

	// 7. 显示执行摘要
	fmt.Println("=== Execution Summary ===")
	fmt.Printf("✅ Success: %v\n", result.Success)
	fmt.Printf("📊 Total Steps: %d\n", result.TotalSteps)
	fmt.Printf("📝 Final Answer:\n%s\n\n", result.Output)

	// 8. 显示详细步骤
	if len(result.Steps) > 0 {
		fmt.Println("=== Detailed Steps ===")
		for i, step := range result.Steps {
			fmt.Printf("\n--- Step %d ---\n", i+1)
			fmt.Printf("Thought: %s\n", step.Action.Log)

			if step.Action.Type == agents.ActionToolCall {
				fmt.Printf("Tool: %s\n", step.Action.Tool)

				// 特殊处理 LLM 直接回答
				if step.Action.Tool == "__llm_answer__" {
					if answer, ok := step.Action.ToolInput["answer"].(string); ok {
						fmt.Printf("Direct Answer: %s\n", answer)
					}
				} else {
					fmt.Printf("Input: %v\n", step.Action.ToolInput)
				}
			}

			fmt.Printf("Observation: %s\n", step.Observation)

			if step.Error != nil {
				fmt.Printf("⚠️  Error: %v\n", step.Error)
			}
		}
	}

	fmt.Println("\n=== Demo Completed ===")

	// 9. 演示完整的工作流说明
	fmt.Println("\n=== How Plan-and-Execute Agent Works ===")
	fmt.Println("1. 📋 Planning Phase:")
	fmt.Println("   - Agent analyzes the task")
	fmt.Println("   - Creates a step-by-step execution plan")
	fmt.Println("   - Identifies required tools and dependencies")
	fmt.Println()
	fmt.Println("2. 🚀 Execution Phase:")
	fmt.Println("   - Executes each step sequentially")
	fmt.Println("   - Uses tools when needed")
	fmt.Println("   - Passes results between steps")
	fmt.Println()
	fmt.Println("3. 🎯 Completion Phase:")
	fmt.Println("   - Aggregates all step results")
	fmt.Println("   - Generates comprehensive final answer")
	fmt.Println("   - Returns execution summary")
	fmt.Println()
	fmt.Println("💡 Key Benefits:")
	fmt.Println("   ✅ Structured approach to complex tasks")
	fmt.Println("   ✅ Clear visibility into execution process")
	fmt.Println("   ✅ Automatic step dependency management")
	fmt.Println("   ✅ Optional replanning on failures")
}
