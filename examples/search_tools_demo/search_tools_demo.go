package main

import (
	"context"
	"fmt"
	"time"

	"github.com/zhucl121/langchain-go/core/tools/search"
)

func main() {
	fmt.Println("=== 搜索工具集成演示 ===")

	// 演示 1: DuckDuckGo 搜索（无需 API Key）
	fmt.Println("--- 演示 1: DuckDuckGo 搜索 ---")
	demoDuckDuckGo()

	// 演示 2: 配置选项
	fmt.Println("\n--- 演示 2: 自定义配置 ---")
	demoCustomOptions()

	// 演示 3: 多搜索引擎
	fmt.Println("\n--- 演示 3: 多搜索引擎可用性检查 ---")
	demoMultipleEngines()

	// 演示 4: 错误处理
	fmt.Println("\n--- 演示 4: 错误处理 ---")
	demoErrorHandling()

	fmt.Println("\n=== 演示完成 ===")
}

// demoDuckDuckGo 演示 DuckDuckGo 搜索
func demoDuckDuckGo() {
	// 1. 创建提供者
	provider := search.NewDuckDuckGoProvider(search.DuckDuckGoConfig{})

	fmt.Printf("引擎: %s\n", provider.GetName())
	fmt.Printf("可用: %v\n\n", provider.IsAvailable())

	// 2. 创建搜索工具
	options := search.DefaultSearchOptions()
	options.MaxResults = 3

	tool, err := search.NewSearchTool(provider, options)
	if err != nil {
		fmt.Printf("创建工具失败: %v\n", err)
		return
	}

	fmt.Printf("工具名称: %s\n", tool.GetName())
	fmt.Printf("工具描述: %s\n\n", tool.GetDescription())

	// 3. 执行搜索（模拟）
	fmt.Println("执行搜索查询: 'golang tutorial'")
	fmt.Println("(注意: 这是一个演示，实际搜索需要网络连接)")
	fmt.Println("\n预期输出格式:")
	fmt.Println("---")
	printMockSearchResults()
}

// demoCustomOptions 演示自定义配置
func demoCustomOptions() {
	// 创建自定义配置
	options := search.SearchOptions{
		MaxResults: 10,
		Language:   "zh-CN",
		Region:     "CN",
		SafeSearch: "strict",
		Timeout:    30 * time.Second,
	}

	fmt.Println("自定义配置:")
	fmt.Printf("  最大结果数: %d\n", options.MaxResults)
	fmt.Printf("  语言: %s\n", options.Language)
	fmt.Printf("  地区: %s\n", options.Region)
	fmt.Printf("  安全搜索: %s\n", options.SafeSearch)
	fmt.Printf("  超时: %v\n", options.Timeout)

	// 验证配置
	if err := options.Validate(); err != nil {
		fmt.Printf("\n配置验证失败: %v\n", err)
	} else {
		fmt.Println("\n✓ 配置验证通过")
	}
}

// demoMultipleEngines 演示多搜索引擎
func demoMultipleEngines() {
	engines := []struct {
		name     string
		provider search.SearchProvider
	}{
		{
			name:     "DuckDuckGo",
			provider: search.NewDuckDuckGoProvider(search.DuckDuckGoConfig{}),
		},
		{
			name:     "Google",
			provider: search.NewGoogleProvider(search.GoogleConfig{}),
		},
		{
			name:     "Bing",
			provider: search.NewBingProvider(search.BingConfig{}),
		},
	}

	fmt.Println("检查搜索引擎可用性:")

	availableCount := 0
	for _, engine := range engines {
		available := engine.provider.IsAvailable()
		status := "❌ 不可用"
		if available {
			status = "✓ 可用"
			availableCount++
		}

		fmt.Printf("  %s: %s", engine.name, status)

		if !available && engine.name != "DuckDuckGo" {
			fmt.Printf(" (需要配置 API Key)")
		}
		fmt.Println()
	}

	fmt.Printf("\n总计: %d/%d 个搜索引擎可用\n", availableCount, len(engines))

	if availableCount == 1 {
		fmt.Println("\n💡 提示: DuckDuckGo 无需配置即可使用")
		fmt.Println("要使用 Google 或 Bing，请配置相应的 API Key:")
		fmt.Println("  export GOOGLE_API_KEY=your-key")
		fmt.Println("  export GOOGLE_SEARCH_ENGINE_ID=your-id")
		fmt.Println("  export BING_API_KEY=your-key")
	}
}

// demoErrorHandling 演示错误处理
func demoErrorHandling() {
	provider := search.NewDuckDuckGoProvider(search.DuckDuckGoConfig{})
	options := search.DefaultSearchOptions()

	tool, _ := search.NewSearchTool(provider, options)

	// 测试各种错误情况
	testCases := []struct {
		name string
		args map[string]any
	}{
		{
			name: "缺少查询参数",
			args: map[string]any{},
		},
		{
			name: "空查询字符串",
			args: map[string]any{"query": ""},
		},
		{
			name: "无效的最大结果数",
			args: map[string]any{
				"query":       "test",
				"max_results": 200,
			},
		},
	}

	ctx := context.Background()

	for i, tc := range testCases {
		fmt.Printf("%d. 测试: %s\n", i+1, tc.name)
		_, err := tool.Execute(ctx, tc.args)
		if err != nil {
			fmt.Printf("   预期错误: %v\n", err)
		} else {
			fmt.Println("   意外: 没有错误")
		}
	}
}

// printMockSearchResults 打印模拟搜索结果
func printMockSearchResults() {
	mockResults := `Search Results for 'golang tutorial' (found 3 results):

1. Go by Example
   Link: https://gobyexample.com/
   Snippet: Go by Example is a hands-on introduction to Go using annotated example programs. Check out the first example or browse the full list below.

2. A Tour of Go
   Link: https://go.dev/tour/
   Snippet: The tour is divided into a list of modules that you can access by clicking on A Tour of Go on the top left of the page. You can also view the table of contents at any time by clicking on the menu on the top right of the page.

3. Go Tutorial - W3Schools
   Link: https://www.w3schools.com/go/
   Snippet: Go is a popular programming language. Go is used to create computer programs. Start learning Go now. Learn by Examples. Learn by examples! This tutorial supplements all explanations with clarifying examples.
`
	fmt.Println(mockResults)
}
