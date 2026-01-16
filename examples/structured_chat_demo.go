// Structured Chat Agent 演示
//
// Structured Chat Agent 支持结构化的对话，带有记忆管理和工具调用能力。
// 适合需要维护对话上下文的多轮对话场景。
//
package main

import (
	"context"
	"fmt"
	"log"
	
	"langchain-go/core/agents"
	"langchain-go/core/chat/providers/openai"
	"langchain-go/core/memory"
	"langchain-go/core/tools"
)

func main() {
	// 1. 创建 LLM
	llm, err := openai.New(openai.Config{APIKey: "your-api-key", Model: "gpt-4"})
	if err != nil {
		log.Fatal(err)
	}
	
	// 2. 创建工具
	toolsList := []tools.Tool{
		tools.NewCalculatorTool(),
		tools.NewWikipediaSearch(nil),
	}
	
	// 3. 创建对话记忆
	mem := memory.NewBufferMemory() // 保留最近 10 条消息
	
	// 4. 创建 Structured Chat Agent
	agent := agents.CreateStructuredChatAgent(
		llm,
		toolsList,
		agents.WithStructuredChatMemory(mem),
		agents.WithStructuredChatOutputFormat("plain"), // 或 "json", "markdown"
		agents.WithStructuredChatConversationID("user-123"),
		agents.WithStructuredChatVerbose(true),
	)
	
	// 5. 创建执行器
	executor := agents.NewSimplifiedAgentExecutor(
		agent,
		toolsList,
	)
	
	ctx := context.Background()
	
	// 6. 多轮对话示例
	fmt.Println("========== Structured Chat Agent Demo ==========")
	fmt.Println("This agent maintains conversation context and can use tools.")
	fmt.Println()
	
	conversations := []string{
		"Hello! What's your name?",
		"What's 25 * 17?",
		"What's the result plus 100?", // 引用上一个结果
		"When is Albert Einstein born? Please search Wikipedia.",
		"What did we talk about earlier?", // 引用对话历史
		"What time is it now?",
	}
	
	for i, userInput := range conversations {
		fmt.Printf("\n--- Turn %d ---\n", i+1)
		fmt.Printf("User: %s\n", userInput)
		
		result, err := executor.Run(ctx, userInput)
		if err != nil {
			log.Printf("Error: %v\n", err)
			continue
		}
		
		fmt.Printf("Agent: %s\n", result.Output)
		
		if result.TotalSteps > 1 {
			fmt.Printf("(Used %d steps to answer)\n", result.TotalSteps)
		}
	}
	
	// 7. JSON 输出格式示例
	fmt.Println("\n========== JSON Output Format ==========")
	
	agentJSON := agents.CreateStructuredChatAgent(
		llm,
		toolsList,
		agents.WithStructuredChatOutputFormat("json"),
		agents.WithStructuredChatConversationID("user-456"),
	)
	
	executorJSON := agents.NewSimplifiedAgentExecutor(
		agentJSON,
		toolsList,
	)
	
	result, err := executorJSON.Run(ctx, "Calculate 123 + 456")
	if err != nil {
		log.Fatal(err)
	}
	
	fmt.Printf("JSON Response:\n%s\n", result.Output)
	
	// 8. Markdown 输出格式示例
	fmt.Println("\n========== Markdown Output Format ==========")
	
	agentMD := agents.CreateStructuredChatAgent(
		llm,
		toolsList,
		agents.WithStructuredChatOutputFormat("markdown"),
		agents.WithStructuredChatConversationID("user-789"),
	)
	
	executorMD := agents.NewSimplifiedAgentExecutor(
		agentMD,
		toolsList,
	)
	
	result, err = executorMD.Run(ctx, "What's the weather like today?")
	if err != nil {
		log.Fatal(err)
	}
	
	fmt.Printf("Markdown Response:\n%s\n", result.Output)
	
	// 9. 清除记忆示例
	fmt.Println("\n========== Memory Management ==========")
	
	// 类型断言以访问特定方法
	if structuredAgent, ok := agent.(*agents.StructuredChatAgent); ok {
		err := structuredAgent.ClearMemory(ctx)
		if err != nil {
			log.Printf("Failed to clear memory: %v\n", err)
		} else {
			fmt.Println("✅ Memory cleared successfully")
		}
	}
}
