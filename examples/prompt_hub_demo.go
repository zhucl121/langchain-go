// Prompt Hub 演示
//
// 演示如何使用 Prompt Hub 从远程仓库拉取、管理和版本控制 prompt 模板。
//
package main

import (
	"context"
	"fmt"
	"log"
	
	"langchain-go/core/prompts"
)

func main() {
	ctx := context.Background()
	
	// 1. 基本使用 - 拉取 Prompt
	fmt.Println("========== Pull Prompt from Hub ==========")
	pullPromptDemo(ctx)
	
	// 2. 版本管理
	fmt.Println("\n========== Prompt Version Management ==========")
	versionManagementDemo(ctx)
	
	// 3. 搜索 Prompts
	fmt.Println("\n========== Search Prompts ==========")
	searchPromptsDemo(ctx)
	
	// 4. 自动生成 Prompt
	fmt.Println("\n========== Auto-Generate Prompt ==========")
	generatePromptDemo()
	
	// 5. 缓存管理
	fmt.Println("\n========== Cache Management ==========")
	cacheDemo(ctx)
}

func pullPromptDemo(ctx context.Context) {
	// 方法 1: 使用便捷函数
	prompt, err := prompts.PullPrompt("hwchase17/react")
	if err != nil {
		// 如果网络不可用，使用本地模板
		log.Printf("Failed to pull from hub: %v\n", err)
		log.Println("Using local template instead...")
		
		prompt, _ = prompts.NewPromptTemplate(prompts.PromptTemplateConfig{
			Template: `Answer the question: {{.question}}`,
			InputVariables: []string{"question"},
		})
	} else {
		fmt.Println("✅ Successfully pulled prompt from hub")
	}
	
	// 使用 prompt
	result, err := prompt.Format(map[string]any{
		"input": "What is the capital of France?",
		"tools": "calculator, search",
		"tool_names": "calculator, search",
		"history": "",
	})
	
	if err != nil {
		log.Fatal(err)
	}
	
	fmt.Printf("Formatted prompt:\n%s\n", result)
	
	// 方法 2: 创建自定义 Hub
	hub := prompts.NewPromptHub(&prompts.PromptHubConfig{
		BaseURL:      "https://smith.langchain.com/hub",
		CacheEnabled: true,
		CacheTTL:     24 * 60 * 60, // 24 hours
	})
	
	prompt2, err := hub.PullPrompt(ctx, "hwchase17/react-chat")
	if err != nil {
		log.Printf("Failed to pull: %v\n", err)
	} else {
		fmt.Println("✅ Pulled another prompt successfully")
		_ = prompt2
	}
}

func versionManagementDemo(ctx context.Context) {
	hub := prompts.NewPromptHub(nil)
	
	// 拉取特定版本
	prompt, err := hub.PullPromptVersion(ctx, "hwchase17/react", "v1.0")
	if err != nil {
		log.Printf("Failed to pull version: %v\n", err)
		return
	}
	
	fmt.Println("✅ Pulled specific version: v1.0")
	_ = prompt
	
	// 列出所有版本
	versions, err := hub.ListVersions(ctx, "hwchase17/react")
	if err != nil {
		log.Printf("Failed to list versions: %v\n", err)
		return
	}
	
	fmt.Printf("Available versions: %d\n", len(versions))
	for _, v := range versions {
		fmt.Printf("  - %s: %s\n", v.Version, v.Description)
	}
}

func searchPromptsDemo(ctx context.Context) {
	hub := prompts.NewPromptHub(nil)
	
	// 搜索 prompts
	results, err := hub.SearchPrompts(ctx, "react agent")
	if err != nil {
		log.Printf("Failed to search: %v\n", err)
		return
	}
	
	fmt.Printf("Found %d prompts:\n", len(results))
	for _, info := range results {
		fmt.Printf("  - %s/%s\n", info.Owner, info.Name)
		fmt.Printf("    Description: %s\n", info.Description)
		fmt.Printf("    Stars: %d, Downloads: %d\n", info.Stars, info.Downloads)
		fmt.Printf("    Tags: %v\n", info.Tags)
	}
}

func generatePromptDemo() {
	// 自动生成 prompt
	task := "Classify movie reviews as positive, negative, or neutral"
	examples := []string{
		"Input: This movie was amazing! Output: positive",
		"Input: Terrible film, waste of time. Output: negative",
		"Input: It was okay, nothing special. Output: neutral",
	}
	
	prompt, err := prompts.GeneratePrompt(task, examples)
	if err != nil {
		log.Fatal(err)
	}
	
	fmt.Println("✅ Generated prompt:")
	
	// 使用生成的 prompt
	result, _ := prompt.Format(map[string]any{
		"input": "The acting was superb and the plot was engaging!",
	})
	
	fmt.Println(result)
}

func cacheDemo(ctx context.Context) {
	hub := prompts.NewPromptHub(&prompts.PromptHubConfig{
		CacheEnabled: true,
		CacheTTL:     3600, // 1 hour
	})
	
	promptName := "hwchase17/react"
	
	// 第一次拉取 - 从网络
	fmt.Println("First pull (from network)...")
	prompt1, err := hub.PullPrompt(ctx, promptName)
	if err != nil {
		log.Printf("Failed: %v\n", err)
		return
	}
	fmt.Println("✅ Pulled from network")
	_ = prompt1
	
	// 第二次拉取 - 从缓存
	fmt.Println("Second pull (from cache)...")
	prompt2, err := hub.PullPrompt(ctx, promptName)
	if err != nil {
		log.Printf("Failed: %v\n", err)
		return
	}
	fmt.Println("✅ Pulled from cache (much faster!)")
	_ = prompt2
	
	// 清除缓存
	hub.ClearCache()
	fmt.Println("✅ Cache cleared")
	
	// 第三次拉取 - 再次从网络
	fmt.Println("Third pull (from network again)...")
	prompt3, err := hub.PullPrompt(ctx, promptName)
	if err != nil {
		log.Printf("Failed: %v\n", err)
		return
	}
	fmt.Println("✅ Pulled from network")
	_ = prompt3
}

// 实际应用示例：集成 Prompt Hub 到 Agent
func integrationExample() {
	ctx := context.Background()
	
	// 1. 从 Hub 拉取 prompt
	prompt, err := prompts.PullPrompt("hwchase17/react")
	if err != nil {
		log.Fatal(err)
	}
	
	// 2. 使用 prompt 创建 Agent
	// (这里省略了 LLM 和工具的创建)
	
	// 3. 使用自定义 prompt
	customPrompt := prompts.PromptTemplateConfig{
		Template: prompt.Template, // 从 Hub prompt 获取模板
		InputVariables: []string{"input", "tools", "history"},
	}
	
	_ = customPrompt
	_ = ctx
	
	fmt.Println("✅ Successfully integrated Prompt Hub with Agent")
}

// Prompt 库管理示例
func promptLibraryExample() {
	// 创建本地 prompt 库
	library := make(map[string]*prompts.PromptTemplate)
	
	// 从 Hub 拉取常用 prompts
	commonPrompts := []string{
		"hwchase17/react",
		"hwchase17/react-chat",
		"rlm/rag-prompt",
	}
	
	for _, name := range commonPrompts {
		prompt, err := prompts.PullPrompt(name)
		if err != nil {
			log.Printf("Failed to pull %s: %v\n", name, err)
			continue
		}
		library[name] = prompt
		fmt.Printf("✅ Added %s to library\n", name)
	}
	
	fmt.Printf("\nPrompt library size: %d\n", len(library))
	
	// 使用库中的 prompt
	if reactPrompt, ok := library["hwchase17/react"]; ok {
		result, _ := reactPrompt.Format(map[string]any{
			"input": "Test input",
		})
		fmt.Printf("Used prompt from library:\n%s\n", result)
	}
}
