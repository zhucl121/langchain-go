// Package main 演示三层认知记忆系统（v0.7.0 新功能）。
//
// 展示功能：
//   - 语义记忆（Semantic Memory）：存储和检索事实知识
//   - 情节记忆（Episodic Memory）：存储和检索对话片段
//   - 程序记忆（Procedural Memory）：行为规则和反馈学习
//   - 统一 Recall 接口：跨层并行搜索
//   - AutoConsolidate：自动从对话中提取记忆
package main

import (
	"context"
	"fmt"
	"log"

	cogmem "github.com/zhucl121/langchain-go/core/memory/cognitive"
	"github.com/zhucl121/langchain-go/pkg/types"
)

func main() {
	ctx := context.Background()

	fmt.Println("=== LangChain-Go v0.7.0: 三层认知记忆系统演示 ===\n")

	// ─── 1. 创建记忆管理器 ───────────────────────────────────
	fmt.Println("【1】初始化认知记忆管理器...")

	manager := cogmem.NewMemoryManager(cogmem.ManagerConfig{
		UserID:   "user-alice",
		ThreadID: "session-main",
		// 使用内置的内存存储后端（生产环境可替换为 PostgreSQL/Redis）
		SemanticStorage:   cogmem.NewInMemorySemanticStorage(),
		EpisodicStorage:   cogmem.NewInMemoryEpisodicStorage(),
		ProceduralStorage: cogmem.NewInMemoryProceduralStorage(),
		Extractor:         cogmem.NewRuleBasedExtractor(),
	})
	fmt.Println("  ✅ 记忆管理器初始化完成（内存存储后端）")
	fmt.Println()

	// ─── 2. 语义记忆（Semantic Memory）────────────────────────
	fmt.Println("【2】语义记忆：存储事实知识")

	semanticFacts := []*cogmem.SemanticMemory{
		{
			Content:    "Go 语言在 2009 年由 Google 发布，专为系统编程设计",
			Category:   "technology",
			Confidence: 0.9,
			Source:     "tech-doc",
		},
		{
			Content:    "用户 Alice 是一名 Go 开发者，有 5 年编程经验",
			Category:   "user-profile",
			Confidence: 1.0,
			Source:     "onboarding",
		},
		{
			Content:    "Alice 偏好简洁、性能优先的代码风格",
			Category:   "user-preference",
			Confidence: 0.8,
			Source:     "feedback",
		},
		{
			Content:    "LangChain-Go v0.7.0 引入了认知记忆系统",
			Category:   "product",
			Confidence: 0.7,
			Source:     "release-notes",
		},
	}

	for _, fact := range semanticFacts {
		if err := manager.StoreSemanticMemory(ctx, fact); err != nil {
			log.Printf("  ⚠️ 存储语义记忆失败: %v", err)
			continue
		}
	}
	fmt.Printf("  ✅ 已存储 %d 条语义记忆\n", len(semanticFacts))

	// 语义搜索
	fmt.Println("\n  📚 搜索关键词: 'Alice Go'")
	results, err := manager.SearchSemanticMemory(ctx, "Alice Go", 3)
	if err != nil {
		log.Printf("  ⚠️ 搜索失败: %v", err)
	} else {
		for i, r := range results {
			fmt.Printf("    [%d] %s (类别: %s, 置信度: %.1f)\n",
				i+1, r.Content, r.Category, r.Confidence)
		}
	}
	fmt.Println()

	// ─── 3. 情节记忆（Episodic Memory）────────────────────────
	fmt.Println("【3】情节记忆：存储对话片段")

	episodes := []*cogmem.Episode{
		{
			ThreadID: "session-001",
			Summary:  "用户询问如何用 Go 实现并发爬虫",
			Messages: []types.Message{
				{Role: types.RoleUser, Content: "如何用 Go 实现高效的并发爬虫？"},
				{Role: types.RoleAssistant, Content: "可以使用 goroutine + channel 实现，推荐限速池模式..."},
			},
			Tags:    []string{"golang", "concurrency", "crawler"},
			Quality: 0.9,
		},
		{
			ThreadID: "session-002",
			Summary:  "讨论 LangGraph 中的 StateGraph 使用方法",
			Messages: []types.Message{
				{Role: types.RoleUser, Content: "StateGraph 如何处理并行节点？"},
				{Role: types.RoleAssistant, Content: "可以使用 AddConditionalEdges 配合 DeferredNode 实现..."},
			},
			Tags:    []string{"langgraph", "stategraph", "parallel"},
			Quality: 0.85,
		},
	}

	for _, ep := range episodes {
		if err := manager.StoreEpisode(ctx, ep); err != nil {
			log.Printf("  ⚠️ 存储情节失败: %v", err)
			continue
		}
	}
	fmt.Printf("  ✅ 已存储 %d 个对话情节\n", len(episodes))

	// 情节搜索
	fmt.Println("\n  🔍 搜索关键词: 'Go 并发'")
	episodeResults, err := manager.SearchEpisodes(ctx, "Go 并发", 2)
	if err != nil {
		log.Printf("  ⚠️ 搜索失败: %v", err)
	} else {
		for i, ep := range episodeResults {
			fmt.Printf("    [%d] Session: %s\n        摘要: %s\n        标签: %v\n",
				i+1, ep.ThreadID, ep.Summary, ep.Tags)
		}
	}
	fmt.Println()

	// ─── 4. 程序记忆（Procedural Memory）──────────────────────
	fmt.Println("【4】程序记忆：行为规则学习（基于反馈）")

	// 通过反馈更新行为规则
	feedbacks := []*cogmem.Feedback{
		{
			Score:   1.0,
			Comment: "用户对详细的代码示例表示满意",
		},
		{
			Score:         -0.5,
			Comment:       "用户不喜欢过长的解释",
			FailureReason: "回答过于冗长，超过用户期望",
		},
	}

	for _, fb := range feedbacks {
		if err := manager.UpdateProceduralMemory(ctx, fb); err != nil {
			log.Printf("  ⚠️ 更新程序记忆失败: %v", err)
			continue
		}
	}

	// 查看程序记忆
	procMem, err := manager.GetProceduralMemory(ctx)
	if err != nil {
		log.Printf("  ⚠️ 获取程序记忆失败: %v", err)
	} else {
		fmt.Printf("  ✅ 程序记忆已更新，共 %d 条行为规则:\n", len(procMem.BehaviorRules))
		for i, rule := range procMem.BehaviorRules {
			fmt.Printf("    [%d] %s (置信度: %.2f)\n", i+1, rule.Rule, rule.Confidence)
		}
		if len(procMem.SystemPromptAdditions) > 0 {
			fmt.Printf("  系统提示补充 (%d 条):\n", len(procMem.SystemPromptAdditions))
			for _, add := range procMem.SystemPromptAdditions {
				fmt.Printf("    + %s\n", add)
			}
		}
	}
	fmt.Println()

	// ─── 5. 统一 Recall（跨层搜索）────────────────────────────
	fmt.Println("【5】统一 Recall：跨三层记忆并行检索")

	recallOpts := cogmem.DefaultRecallOptions()
	recallOpts.K = 3

	recallResult, err := manager.Recall(ctx, "Alice Go 编程", recallOpts)
	if err != nil {
		log.Printf("  ⚠️ Recall 失败: %v", err)
	} else {
		fmt.Printf("  📦 检索结果:\n")
		fmt.Printf("    - 语义记忆: %d 条\n", len(recallResult.SemanticFacts))
		fmt.Printf("    - 情节记忆: %d 条\n", len(recallResult.RecentEpisodes))
		fmt.Printf("    - 程序提示: %d 条\n", len(recallResult.ProceduralHints))
		fmt.Printf("\n  📝 增强上下文（用于注入 LLM Prompt）:\n")
		preview := recallResult.AugmentedContext
		if len(preview) > 300 {
			preview = preview[:300] + "..."
		}
		fmt.Printf("  %s\n", preview)
	}
	fmt.Println()

	// ─── 6. AutoConsolidate（自动记忆提取）────────────────────
	fmt.Println("【6】AutoConsolidate：从对话自动提取记忆")

	conversation := []types.Message{
		{Role: types.RoleUser, Content: "我想学习 Kubernetes 容器编排"},
		{Role: types.RoleAssistant, Content: "Kubernetes 是 Google 开源的容器编排平台，我记住了您的学习偏好"},
		{Role: types.RoleUser, Content: "我喜欢从实践项目入手学习"},
		{Role: types.RoleAssistant, Content: "好的，我会推荐实战驱动的学习路径"},
	}

	if err := manager.AutoConsolidate(ctx, conversation); err != nil {
		log.Printf("  ⚠️ AutoConsolidate 失败: %v", err)
	} else {
		fmt.Println("  ✅ 已从对话自动提取并存储记忆")

		// 验证新记忆被存储
		newResults, _ := manager.SearchSemanticMemory(ctx, "Kubernetes 学习", 3)
		fmt.Printf("  📚 验证：搜索 'Kubernetes 学习' 找到 %d 条记忆\n", len(newResults))
		for _, r := range newResults {
			fmt.Printf("    - %s\n", r.Content)
		}
	}
	fmt.Println()

	// ─── 7. 完整 RAG 增强流程说明 ──────────────────────────────
	fmt.Println("【7】完整 RAG 增强流程示意:")
	fmt.Println(`
  1. 用户提问: "帮我写一个 Go 爬虫"
  2. Recall(query="Go 爬虫") → 检索三层记忆:
     - 语义: "用户 Alice 是 Go 开发者"
     - 情节: "上次讨论了并发爬虫实现方法"
     - 规则: "在回答技术问题时提供完整代码示例"
  3. 构建增强 Prompt:
     [系统] 以下是关于用户的历史记忆:
     用户是有5年经验的 Go 开发者，上次讨论了并发爬虫...
     用户喜欢完整代码示例，回答需简洁...
  4. LLM 基于个性化上下文生成答案
  5. 存储本次对话为新的情节记忆`)

	fmt.Println("\n=== 演示完成！v0.7.0 认知记忆系统功能展示完毕 ===")
}
