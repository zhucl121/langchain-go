// 高级搜索工具演示
//
// 演示如何使用 Tavily 和 Google Custom Search API 进行高质量搜索。
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/zhucl121/langchain-go/core/agents"
	"github.com/zhucl121/langchain-go/core/chat/providers/openai"
	"github.com/zhucl121/langchain-go/core/tools"
)

func main() {
	ctx := context.Background()

	// 1. Tavily Search 示例
	fmt.Println("========== Tavily Search Demo ==========")
	tavilyDemo(ctx)

	// 2. Google Custom Search 示例
	fmt.Println("\n========== Google Custom Search Demo ==========")
	googleSearchDemo(ctx)

	// 3. 集成到 Agent 中使用
	fmt.Println("\n========== Agent with Advanced Search ==========")
	agentDemo(ctx)
}

func tavilyDemo(ctx context.Context) {
	// 从环境变量获取 API key
	apiKey := os.Getenv("TAVILY_API_KEY")
	if apiKey == "" {
		log.Println("⚠️  TAVILY_API_KEY not set, skipping Tavily demo")
		return
	}

	// 创建 Tavily 搜索工具
	tavilyTool := tools.NewTavilySearch(apiKey, &tools.TavilySearchConfig{
		MaxResults:    5,
		SearchDepth:   "advanced", // 或 "basic"
		IncludeAnswer: true,       // 包含 AI 生成的答案
	})

	// 执行搜索
	query := "Latest developments in artificial intelligence 2026"
	fmt.Printf("Searching Tavily for: %s\n\n", query)

	result, err := tavilyTool.Execute(ctx, map[string]any{
		"query": query,
	})

	if err != nil {
		log.Printf("Error: %v\n", err)
		return
	}

	fmt.Println(result)
}

func googleSearchDemo(ctx context.Context) {
	// 从环境变量获取 API key 和 Engine ID
	apiKey := os.Getenv("GOOGLE_API_KEY")
	engineID := os.Getenv("GOOGLE_ENGINE_ID")

	if apiKey == "" || engineID == "" {
		log.Println("⚠️  GOOGLE_API_KEY or GOOGLE_ENGINE_ID not set, skipping Google demo")
		return
	}

	// 创建 Google 搜索工具
	googleTool := tools.NewGoogleSearch(apiKey, engineID, &tools.GoogleSearchConfig{
		MaxResults: 5,
		Language:   "en",
		SafeSearch: "medium",
	})

	// 执行搜索
	query := "Go programming language best practices"
	fmt.Printf("Searching Google for: %s\n\n", query)

	result, err := googleTool.Execute(ctx, map[string]any{
		"query": query,
	})

	if err != nil {
		log.Printf("Error: %v\n", err)
		return
	}

	fmt.Println(result)
}

func agentDemo(ctx context.Context) {
	// 创建 LLM
	llm, err := openai.New(openai.Config{APIKey: "your-api-key", Model: "gpt-4"})
	if err != nil {
		log.Fatal(err)
	}

	// 创建多个搜索工具
	searchTools := []tools.Tool{
		// Wikipedia - 免费，适合百科知识
		tools.NewWikipediaSearch(&tools.WikipediaSearchConfig{
			Language:   "en",
			MaxResults: 3,
		}),

		// Arxiv - 免费，适合学术论文
		tools.NewArxivSearch(&tools.ArxivSearchConfig{
			MaxResults: 3,
			SortBy:     "relevance",
		}),
	}

	// 如果有 API key，添加高级搜索工具
	if tavilyKey := os.Getenv("TAVILY_API_KEY"); tavilyKey != "" {
		searchTools = append(searchTools, tools.NewTavilySearch(tavilyKey, nil))
	}

	if googleKey := os.Getenv("GOOGLE_API_KEY"); googleKey != "" {
		if engineID := os.Getenv("GOOGLE_ENGINE_ID"); engineID != "" {
			searchTools = append(searchTools, tools.NewGoogleSearch(googleKey, engineID, nil))
		}
	}

	// 创建 Agent
	agent := agents.CreateReActAgent(
		llm,
		searchTools,
		agents.WithMaxSteps(10),
		agents.WithVerbose(true),
	)

	executor := agents.NewSimplifiedAgentExecutor(agent, searchTools)

	// 测试问题
	questions := []string{
		"What are the latest breakthroughs in quantum computing?",
		"Find recent research papers about large language models.",
		"What is the capital of France and when was it founded?",
	}

	for i, question := range questions {
		fmt.Printf("\n--- Question %d ---\n", i+1)
		fmt.Printf("Q: %s\n", question)

		result, err := executor.Run(ctx, question)
		if err != nil {
			log.Printf("Error: %v\n", err)
			continue
		}

		fmt.Printf("A: %s\n", result.Output)
		fmt.Printf("Tools used: %d\n", result.TotalSteps)
	}
}

// 比较不同搜索工具的性能
func compareSearchTools(ctx context.Context) {
	fmt.Println("========== Search Tools Comparison ==========")

	query := "artificial intelligence"

	// 1. Wikipedia - 免费，百科知识
	fmt.Println("\n1. Wikipedia Search:")
	wikiTool := tools.NewWikipediaSearch(nil)
	wikiResult, _ := wikiTool.Execute(ctx, map[string]any{"query": query})
	fmt.Println(wikiResult)

	// 2. Arxiv - 免费，学术论文
	fmt.Println("\n2. Arxiv Search:")
	arxivTool := tools.NewArxivSearch(nil)
	arxivResult, _ := arxivTool.Execute(ctx, map[string]any{"query": query})
	fmt.Println(arxivResult)

	// 3. Tavily - 付费，AI 优化结果
	if apiKey := os.Getenv("TAVILY_API_KEY"); apiKey != "" {
		fmt.Println("\n3. Tavily Search:")
		tavilyTool := tools.NewTavilySearch(apiKey, nil)
		tavilyResult, _ := tavilyTool.Execute(ctx, map[string]any{"query": query})
		fmt.Println(tavilyResult)
	}

	// 4. Google - 付费，高质量搜索
	if apiKey := os.Getenv("GOOGLE_API_KEY"); apiKey != "" {
		if engineID := os.Getenv("GOOGLE_ENGINE_ID"); engineID != "" {
			fmt.Println("\n4. Google Custom Search:")
			googleTool := tools.NewGoogleSearch(apiKey, engineID, nil)
			googleResult, _ := googleTool.Execute(ctx, map[string]any{"query": query})
			fmt.Println(googleResult)
		}
	}
}
