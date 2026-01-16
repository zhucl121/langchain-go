// Multi-Agent System Demo
// 演示如何使用 Multi-Agent 系统完成复杂任务

package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"langchain-go/core/agents"
	"langchain-go/core/chat/providers/openai"
	"langchain-go/core/tools"
	"langchain-go/core/tools/search"
)

// createSearchTool 创建搜索工具
func createSearchTool() tools.Tool {
	provider := search.NewDuckDuckGoProvider(search.DuckDuckGoConfig{})
	searchTool, err := search.NewSearchTool(provider, search.SearchOptions{
		MaxResults: 5,
	})
	if err != nil {
		log.Fatalf("Failed to create search tool: %v", err)
	}
	return searchTool
}

func main() {
	fmt.Println("🤖 Multi-Agent System Demo")
	fmt.Println("=" + string(make([]byte, 50)))
	fmt.Println()

	// 运行示例
	runBasicExample()
	fmt.Println()
	runContentCreationPipeline()
	fmt.Println()
	runDataAnalysisPipeline()
}

// 示例 1: 基础 Multi-Agent 系统
func runBasicExample() {
	fmt.Println("📋 示例 1: 基础 Multi-Agent 系统")
	fmt.Println("-" + string(make([]byte, 40)))

	ctx := context.Background()
	llm, err := openai.New(openai.Config{APIKey: "your-api-key", Model: "gpt-3.5-turbo"})
	if err != nil {
		log.Fatal(err)
	}

	// 1. 创建协调策略
	strategy := agents.NewSequentialStrategy(llm)

	// 2. 创建协调器 Agent
	coordinator := agents.NewCoordinatorAgent("coordinator", llm, strategy)

	// 3. 创建 Multi-Agent 系统
	config := agents.DefaultMultiAgentConfig()
	config.MessageTimeout = 60 * time.Second
	config.TaskTimeout = 5 * time.Minute

	system := agents.NewMultiAgentSystem(coordinator, config)

	// 4. 添加专用 Agent
	fmt.Println("✓ 添加 Researcher Agent")
	searchTool := createSearchTool()
	researcher := agents.NewResearcherAgent("researcher", llm, searchTool)
	system.AddAgent("researcher", researcher)
	coordinator.RegisterAgent(researcher)

	fmt.Println("✓ 添加 Writer Agent")
	writer := agents.NewWriterAgent("writer", llm, "technical")
	system.AddAgent("writer", writer)
	coordinator.RegisterAgent(writer)

	fmt.Println("✓ 添加 Reviewer Agent")
	reviewer := agents.NewReviewerAgent("reviewer", llm, []string{"accuracy", "clarity"})
	system.AddAgent("reviewer", reviewer)
	coordinator.RegisterAgent(reviewer)

	// 5. 执行复杂任务
	fmt.Println("\n🚀 执行任务...")
	task := "Research the latest trends in AI and write a brief summary"

	result, err := system.Run(ctx, task)
	if err != nil {
		log.Fatalf("❌ 任务执行失败: %v", err)
	}

	// 6. 输出结果
	fmt.Println("\n✅ 任务完成!")
	fmt.Printf("总消息数: %d\n", result.MessageCount)
	fmt.Printf("执行时长: %v\n", result.Duration)
	fmt.Println("\n📄 最终结果:")
	fmt.Println(result.FinalResult)

	// 7. 显示统计信息
	metrics := system.GetMetrics()
	stats := metrics.GetStats()
	fmt.Println("\n📊 系统统计:")
	fmt.Printf("- 总运行次数: %v\n", stats["total_runs"])
	fmt.Printf("- 成功率: %.1f%%\n", stats["success_rate"])
	fmt.Printf("- 平均时间: %v\n", stats["average_time"])
}

// 示例 2: 内容创作流水线
func runContentCreationPipeline() {
	fmt.Println("📝 示例 2: 内容创作流水线")
	fmt.Println("-" + string(make([]byte, 40)))

	ctx := context.Background()
	llm, err := openai.New(openai.Config{APIKey: "your-api-key", Model: "gpt-3.5-turbo"})
	if err != nil {
		log.Fatal(err)
	}

	strategy := agents.NewSequentialStrategy(llm)
	coordinator := agents.NewCoordinatorAgent("coordinator", llm, strategy)

	config := &agents.MultiAgentConfig{
		Strategy:            strategy,
		MaxConcurrentAgents: 4,
		MessageTimeout:      60 * time.Second,
		TaskTimeout:         10 * time.Minute,
		EnableSharedState:   true,
		EnableHistory:       true,
		MessageQueueSize:    100,
	}

	system := agents.NewMultiAgentSystem(coordinator, config)

	// 创建内容创作团队
	fmt.Println("🎭 组建内容创作团队:")

	planner := agents.NewPlannerAgent("planner", llm)
	system.AddAgent("planner", planner)
	coordinator.RegisterAgent(planner)
	fmt.Println("✓ Planner (规划)")

	researcher := agents.NewResearcherAgent("researcher", llm, createSearchTool())
	system.AddAgent("researcher", researcher)
	coordinator.RegisterAgent(researcher)
	fmt.Println("✓ Researcher (研究)")

	writer := agents.NewWriterAgent("writer", llm, "creative")
	system.AddAgent("writer", writer)
	coordinator.RegisterAgent(writer)
	fmt.Println("✓ Writer (写作)")

	reviewer := agents.NewReviewerAgent("reviewer", llm, []string{"grammar", "clarity", "engagement"})
	system.AddAgent("reviewer", reviewer)
	coordinator.RegisterAgent(reviewer)
	fmt.Println("✓ Reviewer (审核)")

	// 执行内容创作任务
	fmt.Println("\n🚀 开始创作...")
	task := "Create a comprehensive blog post about the future of artificial intelligence"

	result, err := system.Run(ctx, task)
	if err != nil {
		log.Fatalf("❌ 创作失败: %v", err)
	}

	fmt.Println("\n✅ 创作完成!")
	fmt.Printf("协作消息: %d 条\n", result.MessageCount)
	fmt.Printf("总耗时: %v\n", result.Duration)
	fmt.Println("\n📄 创作成果:")
	fmt.Println(result.FinalResult[:min(500, len(result.FinalResult))] + "...")

	// 显示共享状态
	sharedState := system.GetSharedState()
	fmt.Println("\n🔄 共享状态:")
	for key, value := range sharedState.GetAll() {
		fmt.Printf("- %s: %v\n", key, value)
	}
}

// 示例 3: 数据分析流水线
func runDataAnalysisPipeline() {
	fmt.Println("📊 示例 3: 数据分析流水线")
	fmt.Println("-" + string(make([]byte, 40)))

	ctx := context.Background()
	llm, err := openai.New(openai.Config{APIKey: "your-api-key", Model: "gpt-3.5-turbo"})
	if err != nil {
		log.Fatal(err)
	}

	strategy := agents.NewSequentialStrategy(llm)
	coordinator := agents.NewCoordinatorAgent("coordinator", llm, strategy)

	config := agents.DefaultMultiAgentConfig()
	system := agents.NewMultiAgentSystem(coordinator, config)

	// 创建数据分析团队
	fmt.Println("🔬 组建数据分析团队:")

	analyst := agents.NewAnalystAgent("analyst", llm)
	system.AddAgent("analyst", analyst)
	coordinator.RegisterAgent(analyst)
	fmt.Println("✓ Analyst (分析)")

	researcher := agents.NewResearcherAgent("researcher", llm, nil)
	system.AddAgent("researcher", researcher)
	coordinator.RegisterAgent(researcher)
	fmt.Println("✓ Researcher (研究)")

	writer := agents.NewWriterAgent("writer", llm, "technical")
	system.AddAgent("writer", writer)
	coordinator.RegisterAgent(writer)
	fmt.Println("✓ Writer (报告)")

	// 执行数据分析任务
	fmt.Println("\n🚀 开始分析...")
	task := "Analyze the trends in AI development over the past 5 years and provide insights"

	result, err := system.Run(ctx, task)
	if err != nil {
		log.Fatalf("❌ 分析失败: %v", err)
	}

	fmt.Println("\n✅ 分析完成!")
	fmt.Printf("处理步骤: %d 个\n", result.MessageCount)
	fmt.Printf("分析时长: %v\n", result.Duration)
	fmt.Println("\n📈 分析报告:")
	fmt.Println(result.FinalResult[:min(500, len(result.FinalResult))] + "...")

	// 显示历史记录
	history := system.GetHistory()
	records := history.GetAllRecords()
	fmt.Println("\n📜 执行历史:")
	for i, record := range records {
		if i >= 3 { // 只显示前3条
			break
		}
		fmt.Printf("%d. [%s] %s -> %s (耗时: %v)\n",
			i+1,
			record.Status,
			record.MessageID,
			record.Status,
			record.EndTime.Sub(record.StartTime))
	}

	// 显示 Agent 使用率
	metrics := system.GetMetrics()
	stats := metrics.GetStats()
	fmt.Println("\n👥 Agent 使用率:")
	if utilization, ok := stats["agent_utilization"].(map[string]int64); ok {
		for agentID, count := range utilization {
			fmt.Printf("- %s: %d 次\n", agentID, count)
		}
	}
}

// 示例 4: 自定义 Agent
func runCustomAgentExample() {
	fmt.Println("🎨 示例 4: 自定义 Agent")
	fmt.Println("-" + string(make([]byte, 40)))

	_ = context.Background()
	_, err := openai.New(openai.Config{APIKey: "your-api-key", Model: "gpt-3.5-turbo"})
	if err != nil {
		log.Fatal(err)
	}

	// 创建自定义 Agent
	type CustomAgent struct {
		agents.BaseMultiAgent
		domain string
	}

	customAgent := &CustomAgent{
		BaseMultiAgent: agents.BaseMultiAgent{},
		domain:         "finance",
	}

	// 实现 ReceiveMessage
	customAgent.BaseMultiAgent = agents.BaseMultiAgent{}

	fmt.Println("✓ 自定义 Agent 创建成功")
	fmt.Printf("- 领域: %s\n", customAgent.domain)
}

// 示例 5: 性能基准测试
func runPerformanceBenchmark() {
	fmt.Println("⚡ 示例 5: 性能基准测试")
	fmt.Println("-" + string(make([]byte, 40)))

	ctx := context.Background()
	llm, err := openai.New(openai.Config{APIKey: "your-api-key", Model: "gpt-3.5-turbo"})
	if err != nil {
		log.Fatal(err)
	}

	strategy := agents.NewSequentialStrategy(llm)
	coordinator := agents.NewCoordinatorAgent("coordinator", llm, strategy)

	config := agents.DefaultMultiAgentConfig()
	system := agents.NewMultiAgentSystem(coordinator, config)

	// 添加 Agent
	for i := 1; i <= 5; i++ {
		agent := agents.NewResearcherAgent(fmt.Sprintf("agent_%d", i), llm, nil)
		system.AddAgent(agent.ID(), agent)
		coordinator.RegisterAgent(agent)
	}

	// 执行多次测试
	fmt.Println("🏃 运行基准测试...")
	iterations := 10
	totalDuration := time.Duration(0)

	for i := 0; i < iterations; i++ {
		start := time.Now()
		task := fmt.Sprintf("Task %d: Simple research task", i+1)
		_, err := system.Run(ctx, task)
		if err == nil {
			totalDuration += time.Since(start)
		}
	}

	avgDuration := totalDuration / time.Duration(iterations)
	fmt.Printf("\n📊 基准测试结果 (%d 次迭代):\n", iterations)
	fmt.Printf("- 总耗时: %v\n", totalDuration)
	fmt.Printf("- 平均耗时: %v\n", avgDuration)
	fmt.Printf("- 吞吐量: %.2f 任务/秒\n", float64(iterations)/totalDuration.Seconds())

	// 显示最终统计
	metrics := system.GetMetrics()
	stats := metrics.GetStats()
	fmt.Println("\n📈 系统统计:")
	fmt.Printf("- 总运行: %v\n", stats["total_runs"])
	fmt.Printf("- 成功率: %.1f%%\n", stats["success_rate"])
	fmt.Printf("- 消息总数: %v\n", stats["total_messages"])
}

// 辅助函数
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// 示例 6: 错误处理和重试
func runErrorHandlingExample() {
	fmt.Println("🛡️ 示例 6: 错误处理和重试")
	fmt.Println("-" + string(make([]byte, 40)))

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	llm, err := openai.New(openai.Config{APIKey: "your-api-key", Model: "gpt-3.5-turbo"})
	if err != nil {
		log.Fatal(err)
	}
	strategy := agents.NewSequentialStrategy(llm)
	coordinator := agents.NewCoordinatorAgent("coordinator", llm, strategy)

	config := agents.DefaultMultiAgentConfig()
	config.MaxRetries = 3
	config.MessageTimeout = 10 * time.Second

	system := agents.NewMultiAgentSystem(coordinator, config)

	// 添加 Agent
	agent := agents.NewResearcherAgent("researcher", llm, nil)
	system.AddAgent("researcher", agent)
	coordinator.RegisterAgent(agent)

	// 执行可能失败的任务
	fmt.Println("🚀 执行任务（带超时）...")
	task := "Complex task that might timeout"

	result, err := system.Run(ctx, task)
	if err != nil {
		fmt.Printf("❌ 任务失败: %v\n", err)
		fmt.Println("✓ 错误已被捕获和处理")
	} else {
		fmt.Println("✅ 任务成功完成")
		fmt.Printf("结果: %s\n", result.FinalResult[:min(100, len(result.FinalResult))])
	}

	// 检查执行历史中的错误
	history := system.GetHistory()
	records := history.GetAllRecords()
	errorCount := 0
	for _, record := range records {
		if record.Error != nil {
			errorCount++
		}
	}
	fmt.Printf("\n📊 错误统计: %d 个错误记录\n", errorCount)
}
