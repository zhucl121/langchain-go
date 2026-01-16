// Self-Ask Agent 演示
//
// Self-Ask Agent 通过递归分解问题的方式来解决复杂问题。
// 适合需要多步推理的问题，如"谁是美国总统的母亲的故乡？"
//
package main

import (
	"context"
	"fmt"
	"log"
	
	"langchain-go/core/agents"
	"langchain-go/core/chat/providers/openai"
	"langchain-go/core/tools"
)

func main() {
	// 1. 创建 LLM
	llm, err := openai.New(openai.Config{APIKey: "your-api-key", Model: "gpt-4"})
	if err != nil {
		log.Fatal(err)
	}
	
	// 2. 创建搜索工具（Self-Ask Agent 需要搜索工具来回答子问题）
	searchTool := tools.NewWikipediaSearch(&tools.WikipediaSearchConfig{
		Language:   "en",
		MaxResults: 3,
	})
	
	// 3. 创建 Self-Ask Agent
	agent := agents.CreateSelfAskAgent(
		llm,
		searchTool,
		agents.WithSelfAskMaxSubQuestions(5),  // 最多 5 个子问题
		agents.WithSelfAskMaxSteps(10),        // 最多 10 步
		agents.WithSelfAskVerbose(true),       // 详细输出
	)
	
	// 4. 创建执行器
	executor := agents.NewSimplifiedAgentExecutor(
		agent,
		[]tools.Tool{searchTool},
	)
	
	// 5. 运行 Agent
	ctx := context.Background()
	
	// 示例问题：需要递归分解的复杂问题
	questions := []string{
		"Who is the spouse of the person who directed Inception?",
		"What is the capital of the country where the Eiffel Tower is located?",
		"Who is the mother of the current US president?",
	}
	
	for i, question := range questions {
		fmt.Printf("\n========== Question %d ==========\n", i+1)
		fmt.Printf("Q: %s\n\n", question)
		
		result, err := executor.Run(ctx, question)
		if err != nil {
			log.Printf("Error: %v\n", err)
			continue
		}
		
		fmt.Printf("Answer: %s\n", result.Output)
		fmt.Printf("Steps taken: %d\n", result.TotalSteps)
		
		// 显示推理过程
		fmt.Println("\nReasoning process:")
		for j, step := range result.Steps {
			fmt.Printf("  Step %d: %s\n", j+1, step.Action.Log)
			if step.Action.Type == agents.ActionToolCall {
				fmt.Printf("    Tool: %s\n", step.Action.Tool)
				fmt.Printf("    Result: %s\n", step.Observation)
			}
		}
	}
	
	// 6. 流式执行示例
	fmt.Println("\n========== Streaming Example ==========")
	question := "What is the birth year of the person who painted the Mona Lisa?"
	fmt.Printf("Q: %s\n\n", question)
	
	eventChan := executor.Stream(ctx, question)
	
	for event := range eventChan {
		switch event.Type {
		case agents.EventTypeStart:
			fmt.Println("🚀 Starting...")
		case agents.EventTypeStep:
			fmt.Printf("📝 Step %d\n", event.Step)
		case agents.EventTypeToolCall:
			if event.Action != nil {
				fmt.Printf("🔧 Using tool: %s\n", event.Action.Tool)
			}
		case agents.EventTypeToolResult:
			fmt.Printf("✅ Tool result: %s\n", event.Observation)
		case agents.EventTypeFinish:
			fmt.Printf("🎉 Final answer: %s\n", event.Observation)
		case agents.EventTypeError:
			fmt.Printf("❌ Error: %v\n", event.Error)
		}
	}
}
