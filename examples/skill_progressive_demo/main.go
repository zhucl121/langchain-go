package main

import (
	"context"
	"fmt"
	"log"

	"github.com/zhucl121/langchain-go/pkg/skills"
	"github.com/zhucl121/langchain-go/pkg/skills/builtin"
)

func main() {
	ctx := context.Background()

	fmt.Println("=== LangChain-Go Skill 渐进式加载与元工具示例 ===\n")

	// 1. 创建 Skill 管理器
	fmt.Println("【1】创建 Skill 管理器")
	skillManager := skills.NewSkillManager()
	fmt.Println("   ✓ Skill 管理器已创建\n")

	// 2. 注册多个 Skills
	fmt.Println("【2】注册 10 个 Skills（模拟大规模场景）")
	skillManager.Register(builtin.NewCodingSkill())
	skillManager.Register(builtin.NewDataAnalysisSkill())
	skillManager.Register(builtin.NewKnowledgeQuerySkill())
	skillManager.Register(builtin.NewResearchSkill())

	// 添加更多 Skills（模拟）
	for i := 5; i <= 10; i++ {
		skill := skills.NewBaseSkill(
			skills.WithID(fmt.Sprintf("skill-%d", i)),
			skills.WithName(fmt.Sprintf("示例技能 %d", i)),
			skills.WithDescription(fmt.Sprintf("这是第 %d 个示例技能", i)),
			skills.WithCategory(skills.CategoryGeneral),
		)
		skillManager.Register(skill)
	}

	fmt.Printf("   ✓ 已注册 %d 个 Skills\n\n", skillManager.Count())

	// 3. Token 消耗对比
	fmt.Println("【3】Token 消耗对比分析")
	comparison := skills.CompareTokenUsage(10)
	fmt.Printf("   Skills 数量: %d\n", comparison["skill_count"])
	fmt.Printf("   传统方式 Token 消耗: %d tokens\n", comparison["traditional_tokens"])
	fmt.Printf("   元工具方式 Token 消耗: %d tokens\n", comparison["meta_tool_tokens"])
	fmt.Printf("   节省 Token: %d tokens (%.1f%%)\n",
		comparison["tokens_saved"],
		comparison["reduction_percent"])
	fmt.Println()

	// 4. 创建元工具
	fmt.Println("【4】创建元工具（Meta-Tool）")
	metaTool := skills.NewSkillMetaTool(skillManager).WithVerbose(true)
	fmt.Printf("   ✓ 元工具已创建: %s\n", metaTool.GetName())
	fmt.Printf("   工具数量: 1 个（而非 10+ 个）\n")
	fmt.Println()

	// 5. 使用元工具：列出可用 Skills（Level 1）
	fmt.Println("【5】使用元工具：列出可用 Skills（Level 1: 元数据）")
	result, err := metaTool.Execute(ctx, map[string]any{
		"list_skills": true,
	})
	if err != nil {
		log.Fatalf("Failed to list skills: %v", err)
	}

	resultMap := result.(map[string]any)
	skillsList := resultMap["skills"].([]map[string]any)
	fmt.Printf("   可用 Skills: %d 个\n", resultMap["total"])
	fmt.Println("   Skills 列表（仅元数据，~100B/skill）:")
	for i, skillInfo := range skillsList {
		if i < 4 { // 只显示前4个
			fmt.Printf("     %d. %s (%s): %s\n",
				i+1,
				skillInfo["name"],
				skillInfo["id"],
				skillInfo["description"])
		}
	}
	fmt.Printf("     ... 其他 %d 个\n", len(skillsList)-4)

	// 估算 Token 消耗
	tokens := skills.EstimateTokensForSkillList(skillManager)
	fmt.Printf("   Level 1 Token 消耗: ~%d tokens\n", tokens)
	fmt.Println()

	// 6. 使用元工具：调用 Coding Skill（Level 2）
	fmt.Println("【6】使用元工具：调用 Coding Skill（Level 2: 加载指令）")
	result, err = metaTool.Execute(ctx, map[string]any{
		"skill_name": "coding",
		"action":     "write_code",
		"params": map[string]any{
			"language": "go",
			"task":     "implement quick sort",
		},
	})
	if err != nil {
		log.Fatalf("Failed to execute coding skill: %v", err)
	}

	fmt.Printf("   执行结果: %v\n", result)
	fmt.Println("   注意: 此时才加载 Coding Skill 的完整指令（~2-5KB）")
	fmt.Println()

	// 7. 演示渐进式加载
	fmt.Println("【7】演示渐进式 Skill 的三级加载")

	// 创建支持渐进式加载的 Skill
	progressiveSkill := skills.NewProgressiveBaseSkill(
		skills.WithProgressiveID("progressive-demo"),
		skills.WithProgressiveName("渐进式演示 Skill"),
		skills.WithProgressiveDescription("演示三级加载机制"),
		skills.WithProgressiveCategory(skills.CategoryGeneral),
		skills.WithProgressiveTags("demo", "progressive"),
	)

	skillManager.Register(progressiveSkill)

	fmt.Printf("   Level 1 (元数据): 始终可用\n")
	fmt.Printf("     - ID: %s\n", progressiveSkill.ID())
	fmt.Printf("     - Name: %s\n", progressiveSkill.Name())
	fmt.Printf("     - Description: %s\n", progressiveSkill.Description())
	fmt.Printf("     - 当前加载级别: Level %d\n", progressiveSkill.GetLoadLevel())
	fmt.Println()

	// 加载 Level 2
	fmt.Printf("   Level 2 (指令): 按需加载\n")
	instructions, err := progressiveSkill.LoadInstructions(ctx)
	if err != nil {
		log.Fatalf("Failed to load instructions: %v", err)
	}
	fmt.Printf("     ✓ 已加载指令\n")
	fmt.Printf("     - 当前加载级别: Level %d\n", progressiveSkill.GetLoadLevel())
	fmt.Printf("     - 估算大小: %d bytes\n", instructions.EstimateSize())
	fmt.Println()

	// 加载 Level 3
	fmt.Printf("   Level 3 (资源): 执行时加载\n")
	resources, err := progressiveSkill.LoadResources(ctx)
	if err != nil {
		log.Fatalf("Failed to load resources: %v", err)
	}
	fmt.Printf("     ✓ 已加载资源\n")
	fmt.Printf("     - 当前加载级别: Level %d\n", progressiveSkill.GetLoadLevel())
	fmt.Printf("     - 估算大小: %d bytes\n", resources.EstimateSize())
	fmt.Printf("     - 注意: 资源文件不进入 LLM 上下文\n")
	fmt.Println()

	// 8. Token 优化总结
	fmt.Println("【8】Token 优化总结")
	fmt.Println("   传统方式（全量加载）:")
	fmt.Println("     - 10 个 Skills × 500 tokens/skill = 5,000 tokens")
	fmt.Println()
	fmt.Println("   优化方式（渐进式 + 元工具）:")
	fmt.Println("     - Level 1: 10 个 Skills × 100 tokens = 1,000 tokens（始终）")
	fmt.Println("     - Level 2: 1 个 Skill × 500 tokens = 500 tokens（按需）")
	fmt.Println("     - Level 3: 不进入 LLM 上下文 = 0 tokens")
	fmt.Println("     - 总计: 1,500 tokens")
	fmt.Println()
	fmt.Printf("   ✅ Token 节省: 3,500 tokens (70%% 优化)\n")
	fmt.Println()

	// 9. 性能优势
	fmt.Println("【9】性能优势")
	fmt.Println("   ✅ 减少 Token 消耗 70%+")
	fmt.Println("   ✅ 降低 API 成本 70%+")
	fmt.Println("   ✅ 提升响应速度（更少的 Token 处理）")
	fmt.Println("   ✅ 支持更多 Skills（不受工具列表限制）")
	fmt.Println("   ✅ 按需加载，降低内存占用")
	fmt.Println()

	// 10. 清理
	fmt.Println("【10】清理资源")
	progressiveSkill.Unload(ctx)
	fmt.Println("   ✓ 渐进式 Skill 已卸载")
	fmt.Printf("   ✓ 回到 Level %d（元数据）\n", progressiveSkill.GetLoadLevel())
	fmt.Println()

	fmt.Println("=== 示例完成 ===")
	fmt.Println()
	fmt.Println("💡 关键要点:")
	fmt.Println("   1. 使用元工具统一管理所有 Skills")
	fmt.Println("   2. 采用三级加载机制按需加载内容")
	fmt.Println("   3. Level 1 始终可用，Level 2/3 按需加载")
	fmt.Println("   4. 大幅降低 Token 消耗和 API 成本")
}
