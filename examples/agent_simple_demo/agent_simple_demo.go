// Package main 提供完整的 Agent 使用示例。
//
// 本示例展示如何使用新的高层 API 快速创建和使用 Agent。
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/zhucl121/langchain-go/core/agents"
	"github.com/zhucl121/langchain-go/core/chat/providers/openai"
	"github.com/zhucl121/langchain-go/core/tools"
	"github.com/zhucl121/langchain-go/pkg/types"
)

func main() {
	// 选择要运行的示例
	examples := map[string]func(){
		"1": exampleSimpleAgent,
		"2": exampleAgentWithBasicTools,
		"3": exampleAgentWithAllTools,
		"4": exampleStreamingAgent,
		"5": exampleToolCallingAgent,
		"6": exampleCustomTools,
	}

	fmt.Println("=== LangChain-Go Agent 示例 ===")
	fmt.Println()
	fmt.Println("选择示例:")
	fmt.Println("1. 简单 Agent")
	fmt.Println("2. 带基础工具的 Agent")
	fmt.Println("3. 带所有内置工具的 Agent")
	fmt.Println("4. 流式 Agent")
	fmt.Println("5. Tool Calling Agent")
	fmt.Println("6. 自定义工具 Agent")

	// 默认运行示例 2
	examples["2"]()
}

// exampleSimpleAgent 示例1：创建最简单的 Agent。
func exampleSimpleAgent() {
	fmt.Println("\n=== 示例1：简单 Agent ===\n")

	ctx := context.Background()

	// 1. 创建 LLM
	llm, err := openai.New(openai.Config{APIKey: "your-api-key", Model: "gpt-3.5-turbo"})
	if err != nil {
		log.Fatal(err)
	}

	// 2. 创建工具（只有计算器）
	agentTools := []tools.Tool{
		tools.NewCalculatorTool(),
	}

	// 3. 创建 Agent（1 行！）
	agent := agents.CreateReActAgent(llm, agentTools)

	// 4. 创建执行器
	executor := agents.NewSimplifiedAgentExecutor(agent, agentTools,
		agents.WithMaxSteps(5),
		agents.WithVerbose(true),
	)

	// 5. 执行任务
	result, err := executor.Run(ctx, "Calculate 25 * 4")
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	fmt.Printf("\n✅ Result: %s\n", result.Output)
	fmt.Printf("📊 Steps taken: %d\n", result.TotalSteps)
}

// exampleAgentWithBasicTools 示例2：使用基础工具的 Agent。
func exampleAgentWithBasicTools() {
	fmt.Println("\n=== 示例2：带基础工具的 Agent ===\n")

	ctx := context.Background()

	// 1. 创建 LLM
	llm, err := openai.New(openai.Config{APIKey: "your-api-key", Model: "gpt-3.5-turbo"})
	if err != nil {
		log.Fatal(err)
	}

	// 2. 获取基础工具（只需 1 行！）
	agentTools := tools.GetBasicTools()

	fmt.Printf("📦 Loaded %d basic tools\n", len(agentTools))
	for _, tool := range agentTools {
		fmt.Printf("  - %s: %s\n", tool.GetName(), tool.GetDescription())
	}
	fmt.Println()

	// 3. 创建 Agent（1 行！）
	agent := agents.CreateReActAgent(llm, agentTools,
		agents.WithMaxSteps(10),
		agents.WithVerbose(true),
	)

	// 4. 创建执行器
	executor := agents.NewSimplifiedAgentExecutor(agent, agentTools)

	// 5. 执行多个任务
	tasks := []string{
		"What time is it now?",
		"What is today's date?",
		"Calculate 123 + 456",
	}

	for i, task := range tasks {
		fmt.Printf("\n📝 Task %d: %s\n", i+1, task)
		fmt.Println("---")

		result, err := executor.Run(ctx, task)
		if err != nil {
			log.Printf("Error: %v", err)
			continue
		}

		fmt.Printf("\n✅ Answer: %s\n", result.Output)
		fmt.Printf("📊 Steps: %d\n", result.TotalSteps)
	}
}

// exampleAgentWithAllTools 示例3：使用所有内置工具的 Agent。
func exampleAgentWithAllTools() {
	fmt.Println("\n=== 示例3：带所有内置工具的 Agent ===\n")

	ctx := context.Background()

	// 1. 创建 LLM
	llm, err := openai.New(openai.Config{APIKey: "your-api-key", Model: "gpt-3.5-turbo"})
	if err != nil {
		log.Fatal(err)
	}

	// 2. 获取所有内置工具（1 行！）
	agentTools := tools.GetBuiltinTools()

	fmt.Printf("📦 Loaded %d tools\n\n", len(agentTools))

	// 按分类显示工具
	categories := []struct {
		name     string
		category tools.ToolCategory
	}{
		{"时间工具", tools.CategoryTime},
		{"HTTP 工具", tools.CategoryHTTP},
		{"JSON 工具", tools.CategoryJSON},
		{"字符串工具", tools.CategoryString},
	}

	for _, cat := range categories {
		categoryTools := tools.GetToolsByCategory(cat.category)
		fmt.Printf("%s (%d):\n", cat.name, len(categoryTools))
		for _, tool := range categoryTools {
			fmt.Printf("  - %s\n", tool.GetName())
		}
		fmt.Println()
	}

	// 3. 创建 Agent
	agent := agents.CreateReActAgent(llm, agentTools,
		agents.WithMaxSteps(15),
		agents.WithSystemPrompt("You are a helpful assistant with access to many tools."),
	)

	// 4. 创建执行器
	executor := agents.NewSimplifiedAgentExecutor(agent, agentTools)

	// 5. 执行复杂任务
	task := "Get the current date and tell me what day of the week it is"

	fmt.Printf("📝 Task: %s\n", task)
	fmt.Println("---\n")

	result, err := executor.Run(ctx, task)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	fmt.Printf("\n✅ Answer: %s\n", result.Output)
	fmt.Printf("📊 Steps: %d\n", result.TotalSteps)
}

// exampleStreamingAgent 示例4：流式执行 Agent。
func exampleStreamingAgent() {
	fmt.Println("\n=== 示例4：流式 Agent ===\n")

	ctx := context.Background()

	// 1. 创建 LLM 和工具
	llm, err := openai.New(openai.Config{APIKey: "your-api-key", Model: "gpt-3.5-turbo"})
	if err != nil {
		log.Fatal(err)
	}
	agentTools := tools.GetBasicTools()

	// 2. 创建 Agent
	agent := agents.CreateReActAgent(llm, agentTools)
	executor := agents.NewSimplifiedAgentExecutor(agent, agentTools,
		agents.WithVerbose(false),
	)

	// 3. 流式执行
	task := "Calculate 100 + 200 and tell me the current time"
	fmt.Printf("📝 Task: %s\n\n", task)

	eventChan := executor.Stream(ctx, task)

	for event := range eventChan {
		switch event.Type {
		case agents.EventTypeStart:
			fmt.Println("🚀 Agent started...")

		case agents.EventTypeStep:
			fmt.Printf("\n📍 Step %d\n", event.Step)

		case agents.EventTypeToolCall:
			fmt.Printf("🔧 Tool call: %s\n", event.Action.Tool)
			fmt.Printf("   Input: %v\n", event.Action.ToolInput)

		case agents.EventTypeToolResult:
			fmt.Printf("📊 Tool result: %s\n", event.Observation)
			if event.Error != nil {
				fmt.Printf("❌ Error: %v\n", event.Error)
			}

		case agents.EventTypeFinish:
			fmt.Printf("\n✅ Agent finished!\n")
			fmt.Printf("Answer: %s\n", event.Observation)

		case agents.EventTypeError:
			fmt.Printf("❌ Error: %v\n", event.Error)
		}
	}
}

// exampleToolCallingAgent 示例5：使用 Tool Calling Agent。
func exampleToolCallingAgent() {
	fmt.Println("\n=== 示例5：Tool Calling Agent ===\n")

	ctx := context.Background()

	// 1. 创建支持工具调用的 LLM
	llm, err := openai.New(openai.Config{APIKey: "your-api-key", Model: "gpt-3.5-turbo"})
	if err != nil {
		log.Fatal(err)
	}

	// 2. 创建工具
	agentTools := []tools.Tool{
		tools.NewCalculatorTool(),
		tools.NewGetTimeTool(nil),
		tools.NewGetDateTool(nil),
	}

	// 3. 创建 Tool Calling Agent（1 行！）
	agent := agents.CreateToolCallingAgent(llm, agentTools,
		agents.WithSystemPrompt("You are a helpful assistant."),
		agents.WithMaxSteps(10),
	)

	// 4. 创建执行器
	executor := agents.NewSimplifiedAgentExecutor(agent, agentTools,
		agents.WithVerbose(true),
	)

	// 5. 执行任务
	task := "What is 50 * 3?"
	fmt.Printf("📝 Task: %s\n\n", task)

	result, err := executor.Run(ctx, task)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	fmt.Printf("\n✅ Result: %s\n", result.Output)
}

// exampleCustomTools 示例6：使用自定义工具。
func exampleCustomTools() {
	fmt.Println("\n=== 示例6：自定义工具 Agent ===\n")

	ctx := context.Background()

	// 1. 创建 LLM
	llm, err := openai.New(openai.Config{APIKey: "your-api-key", Model: "gpt-3.5-turbo"})
	if err != nil {
		log.Fatal(err)
	}

	// 2. 创建自定义工具（使用工具注册表）
	registry := tools.NewToolRegistry()

	// 添加内置工具
	registry.RegisterAll(tools.GetBasicTools())

	// 添加自定义工具
	greetTool := tools.NewFunctionTool(tools.FunctionToolConfig{
		Name:        "greet",
		Description: "Greet someone by name",
		Parameters: func() types.Schema {
			return types.Schema{
				Type: "object",
				Properties: map[string]types.Schema{
					"name": {
						Type:        "string",
						Description: "The name to greet",
					},
				},
				Required: []string{"name"},
			}
		}(),
		Fn: func(ctx context.Context, args map[string]any) (any, error) {
			name := args["name"].(string)
			return fmt.Sprintf("Hello, %s! Nice to meet you!", name), nil
		},
	})

	registry.Register(greetTool)

	fmt.Printf("📦 Total tools: %d\n", registry.Count())
	for _, tool := range registry.GetAll() {
		fmt.Printf("  - %s\n", tool.GetName())
	}
	fmt.Println()

	// 3. 创建 Agent
	agentTools := registry.GetAll()
	agent := agents.CreateReActAgent(llm, agentTools)

	// 4. 创建执行器
	executor := agents.NewSimplifiedAgentExecutor(agent, agentTools,
		agents.WithVerbose(true),
	)

	// 5. 执行任务
	task := "Greet John"
	fmt.Printf("📝 Task: %s\n\n", task)

	result, err := executor.Run(ctx, task)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	fmt.Printf("\n✅ Result: %s\n", result.Output)
}
