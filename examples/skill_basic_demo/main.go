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

	fmt.Println("=== LangChain-Go Skill 系统基础示例 ===\n")

	// 1. 创建 Skill 管理器
	fmt.Println("1. 创建 Skill 管理器")
	skillManager := skills.NewSkillManager()
	fmt.Printf("   ✓ Skill 管理器已创建\n\n")

	// 2. 注册内置 Skills
	fmt.Println("2. 注册内置 Skills")

	codingSkill := builtin.NewCodingSkill()
	if err := skillManager.Register(codingSkill); err != nil {
		log.Fatalf("Failed to register coding skill: %v", err)
	}
	fmt.Printf("   ✓ 注册 Coding Skill (ID: %s)\n", codingSkill.ID())

	dataSkill := builtin.NewDataAnalysisSkill()
	if err := skillManager.Register(dataSkill); err != nil {
		log.Fatalf("Failed to register data analysis skill: %v", err)
	}
	fmt.Printf("   ✓ 注册 Data Analysis Skill (ID: %s)\n", dataSkill.ID())

	knowledgeSkill := builtin.NewKnowledgeQuerySkill()
	if err := skillManager.Register(knowledgeSkill); err != nil {
		log.Fatalf("Failed to register knowledge query skill: %v", err)
	}
	fmt.Printf("   ✓ 注册 Knowledge Query Skill (ID: %s)\n", knowledgeSkill.ID())

	researchSkill := builtin.NewResearchSkill()
	if err := skillManager.Register(researchSkill); err != nil {
		log.Fatalf("Failed to register research skill: %v", err)
	}
	fmt.Printf("   ✓ 注册 Research Skill (ID: %s)\n\n", researchSkill.ID())

	// 3. 列出所有已注册的 Skills
	fmt.Println("3. 列出所有已注册的 Skills")
	allSkills := skillManager.List()
	fmt.Printf("   总计: %d 个 Skills\n", len(allSkills))
	for _, skill := range allSkills {
		fmt.Printf("   - %s (%s): %s\n", skill.Name(), skill.ID(), skill.Description())
		fmt.Printf("     分类: %s, 标签: %v\n", skill.Category(), skill.Tags())
	}
	fmt.Println()

	// 4. 加载 Coding Skill
	fmt.Println("4. 加载 Coding Skill")
	config := skills.DefaultLoadConfig()
	if err := skillManager.Load(ctx, "coding", config); err != nil {
		log.Fatalf("Failed to load coding skill: %v", err)
	}
	fmt.Printf("   ✓ Coding Skill 已加载\n\n")

	// 5. 验证 Skill 状态
	fmt.Println("5. 验证 Skill 状态")
	fmt.Printf("   Coding Skill 已加载: %v\n", skillManager.IsLoaded("coding"))
	fmt.Printf("   Data Analysis Skill 已加载: %v\n", skillManager.IsLoaded("data-analysis"))
	fmt.Printf("   已加载的 Skill 数量: %d\n\n", skillManager.LoadedCount())

	// 6. 获取 Skill 信息
	fmt.Println("6. 获取 Coding Skill 详细信息")
	loadedSkill, err := skillManager.Get("coding")
	if err != nil {
		log.Fatalf("Failed to get coding skill: %v", err)
	}

	fmt.Printf("   名称: %s\n", loadedSkill.Name())
	fmt.Printf("   描述: %s\n", loadedSkill.Description())
	fmt.Printf("   分类: %s\n", loadedSkill.Category())

	// 显示系统提示词（前200个字符）
	prompt := loadedSkill.GetSystemPrompt()
	if len(prompt) > 200 {
		prompt = prompt[:200] + "..."
	}
	fmt.Printf("   系统提示词: %s\n", prompt)

	// 显示示例
	examples := loadedSkill.GetExamples()
	fmt.Printf("   示例数量: %d\n", len(examples))
	if len(examples) > 0 {
		fmt.Println("   第一个示例:")
		fmt.Printf("     输入: %s\n", examples[0].Input)
		if len(examples[0].Output) > 100 {
			fmt.Printf("     输出: %s...\n", examples[0].Output[:100])
		} else {
			fmt.Printf("     输出: %s\n", examples[0].Output)
		}
		if examples[0].Reasoning != "" {
			fmt.Printf("     推理: %s\n", examples[0].Reasoning)
		}
	}

	// 显示元数据
	metadata := loadedSkill.GetMetadata()
	fmt.Printf("   元数据:\n")
	fmt.Printf("     版本: %s\n", metadata.Version)
	fmt.Printf("     作者: %s\n", metadata.Author)
	fmt.Printf("     许可证: %s\n", metadata.License)
	fmt.Println()

	// 7. 按分类查找 Skills
	fmt.Println("7. 按分类查找 Skills")
	codingSkills := skillManager.FindByCategory(skills.CategoryCoding)
	fmt.Printf("   编程类 Skills: %d 个\n", len(codingSkills))
	for _, skill := range codingSkills {
		fmt.Printf("   - %s\n", skill.Name())
	}
	fmt.Println()

	// 8. 按标签查找 Skills
	fmt.Println("8. 按标签查找 Skills")
	researchSkills := skillManager.FindByTags([]string{"research"})
	fmt.Printf("   包含 'research' 标签的 Skills: %d 个\n", len(researchSkills))
	for _, skill := range researchSkills {
		fmt.Printf("   - %s (标签: %v)\n", skill.Name(), skill.Tags())
	}
	fmt.Println()

	// 9. 动态切换 Skill
	fmt.Println("9. 动态切换 Skill")

	// 卸载 Coding Skill
	if err := skillManager.Unload(ctx, "coding"); err != nil {
		log.Fatalf("Failed to unload coding skill: %v", err)
	}
	fmt.Printf("   ✓ Coding Skill 已卸载\n")

	// 加载 Data Analysis Skill
	if err := skillManager.Load(ctx, "data-analysis", config); err != nil {
		log.Fatalf("Failed to load data analysis skill: %v", err)
	}
	fmt.Printf("   ✓ Data Analysis Skill 已加载\n")

	// 显示当前已加载的 Skills
	loadedSkills := skillManager.ListLoaded()
	fmt.Printf("   当前已加载: %d 个 Skills\n", len(loadedSkills))
	for _, skill := range loadedSkills {
		fmt.Printf("   - %s\n", skill.Name())
	}
	fmt.Println()

	// 10. 清理
	fmt.Println("10. 清理资源")
	if err := skillManager.Unload(ctx, "data-analysis"); err != nil {
		log.Fatalf("Failed to unload data analysis skill: %v", err)
	}
	fmt.Printf("   ✓ 所有 Skills 已卸载\n")
	fmt.Printf("   ✓ 已加载的 Skill 数量: %d\n\n", skillManager.LoadedCount())

	fmt.Println("=== 示例完成 ===")
}
