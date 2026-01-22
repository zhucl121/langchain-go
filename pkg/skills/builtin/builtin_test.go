package builtin

import (
	"context"
	"testing"

	"github.com/zhucl121/langchain-go/pkg/skills"
)

func TestNewCodingSkill(t *testing.T) {
	skill := NewCodingSkill()

	if skill == nil {
		t.Fatal("Expected non-nil skill")
	}

	if skill.ID() != "coding" {
		t.Errorf("Expected ID 'coding', got '%s'", skill.ID())
	}

	if skill.Name() != "代码助手" {
		t.Errorf("Expected name '代码助手', got '%s'", skill.Name())
	}

	if skill.Category() != skills.CategoryCoding {
		t.Errorf("Expected category CategoryCoding, got %v", skill.Category())
	}

	// 验证系统提示词不为空
	if skill.GetSystemPrompt() == "" {
		t.Error("Expected non-empty system prompt")
	}

	// 验证有示例
	examples := skill.GetExamples()
	if len(examples) == 0 {
		t.Error("Expected at least one example")
	}

	// 验证元数据
	metadata := skill.GetMetadata()
	if metadata.Version == "" {
		t.Error("Expected version to be set")
	}
}

func TestCodingSkillLoadUnload(t *testing.T) {
	ctx := context.Background()
	skill := NewCodingSkill()

	config := skills.DefaultLoadConfig()

	// 加载
	if err := skill.Load(ctx, config); err != nil {
		t.Fatalf("Failed to load coding skill: %v", err)
	}

	if !skill.IsLoaded() {
		t.Error("Expected skill to be loaded")
	}

	// 卸载
	if err := skill.Unload(ctx); err != nil {
		t.Fatalf("Failed to unload coding skill: %v", err)
	}

	if skill.IsLoaded() {
		t.Error("Expected skill to not be loaded")
	}
}

func TestNewDataAnalysisSkill(t *testing.T) {
	skill := NewDataAnalysisSkill()

	if skill == nil {
		t.Fatal("Expected non-nil skill")
	}

	if skill.ID() != "data-analysis" {
		t.Errorf("Expected ID 'data-analysis', got '%s'", skill.ID())
	}

	if skill.Name() != "数据分析师" {
		t.Errorf("Expected name '数据分析师', got '%s'", skill.Name())
	}

	if skill.Category() != skills.CategoryDataAnalysis {
		t.Errorf("Expected category CategoryDataAnalysis, got %v", skill.Category())
	}

	// 验证系统提示词不为空
	if skill.GetSystemPrompt() == "" {
		t.Error("Expected non-empty system prompt")
	}

	// 验证有示例
	examples := skill.GetExamples()
	if len(examples) == 0 {
		t.Error("Expected at least one example")
	}
}

func TestDataAnalysisSkillLoadUnload(t *testing.T) {
	ctx := context.Background()
	skill := NewDataAnalysisSkill()

	config := skills.DefaultLoadConfig()

	// 加载
	if err := skill.Load(ctx, config); err != nil {
		t.Fatalf("Failed to load data analysis skill: %v", err)
	}

	// 卸载
	if err := skill.Unload(ctx); err != nil {
		t.Fatalf("Failed to unload data analysis skill: %v", err)
	}
}

func TestNewKnowledgeQuerySkill(t *testing.T) {
	skill := NewKnowledgeQuerySkill()

	if skill == nil {
		t.Fatal("Expected non-nil skill")
	}

	if skill.ID() != "knowledge-query" {
		t.Errorf("Expected ID 'knowledge-query', got '%s'", skill.ID())
	}

	if skill.Name() != "知识专家" {
		t.Errorf("Expected name '知识专家', got '%s'", skill.Name())
	}

	if skill.Category() != skills.CategoryKnowledge {
		t.Errorf("Expected category CategoryKnowledge, got %v", skill.Category())
	}

	// 验证系统提示词不为空
	if skill.GetSystemPrompt() == "" {
		t.Error("Expected non-empty system prompt")
	}

	// 验证有示例
	examples := skill.GetExamples()
	if len(examples) == 0 {
		t.Error("Expected at least one example")
	}
}

func TestKnowledgeQuerySkillLoadUnload(t *testing.T) {
	ctx := context.Background()
	skill := NewKnowledgeQuerySkill()

	config := skills.DefaultLoadConfig()

	// 加载
	if err := skill.Load(ctx, config); err != nil {
		t.Fatalf("Failed to load knowledge query skill: %v", err)
	}

	// 卸载
	if err := skill.Unload(ctx); err != nil {
		t.Fatalf("Failed to unload knowledge query skill: %v", err)
	}
}

func TestNewResearchSkill(t *testing.T) {
	skill := NewResearchSkill()

	if skill == nil {
		t.Fatal("Expected non-nil skill")
	}

	if skill.ID() != "research" {
		t.Errorf("Expected ID 'research', got '%s'", skill.ID())
	}

	if skill.Name() != "研究员" {
		t.Errorf("Expected name '研究员', got '%s'", skill.Name())
	}

	if skill.Category() != skills.CategoryResearch {
		t.Errorf("Expected category CategoryResearch, got %v", skill.Category())
	}

	// 验证系统提示词不为空
	if skill.GetSystemPrompt() == "" {
		t.Error("Expected non-empty system prompt")
	}

	// 验证有示例
	examples := skill.GetExamples()
	if len(examples) == 0 {
		t.Error("Expected at least one example")
	}
}

func TestResearchSkillLoadUnload(t *testing.T) {
	ctx := context.Background()
	skill := NewResearchSkill()

	config := skills.DefaultLoadConfig()

	// 加载
	if err := skill.Load(ctx, config); err != nil {
		t.Fatalf("Failed to load research skill: %v", err)
	}

	// 卸载
	if err := skill.Unload(ctx); err != nil {
		t.Fatalf("Failed to unload research skill: %v", err)
	}
}

// 测试所有内置 Skill 的标签
func TestBuiltinSkillTags(t *testing.T) {
	skills := []skills.Skill{
		NewCodingSkill(),
		NewDataAnalysisSkill(),
		NewKnowledgeQuerySkill(),
		NewResearchSkill(),
	}

	for _, skill := range skills {
		tags := skill.Tags()
		if len(tags) == 0 {
			t.Errorf("Expected skill %s to have tags", skill.ID())
		}
	}
}

// 测试所有内置 Skill 的元数据
func TestBuiltinSkillMetadata(t *testing.T) {
	skills := []skills.Skill{
		NewCodingSkill(),
		NewDataAnalysisSkill(),
		NewKnowledgeQuerySkill(),
		NewResearchSkill(),
	}

	for _, skill := range skills {
		metadata := skill.GetMetadata()

		if metadata.Version == "" {
			t.Errorf("Expected skill %s to have version", skill.ID())
		}

		if metadata.Author == "" {
			t.Errorf("Expected skill %s to have author", skill.ID())
		}

		if metadata.License == "" {
			t.Errorf("Expected skill %s to have license", skill.ID())
		}
	}
}

// 测试示例的完整性
func TestBuiltinSkillExamplesCompleteness(t *testing.T) {
	skills := []skills.Skill{
		NewCodingSkill(),
		NewDataAnalysisSkill(),
		NewKnowledgeQuerySkill(),
		NewResearchSkill(),
	}

	for _, skill := range skills {
		examples := skill.GetExamples()

		for i, example := range examples {
			if example.Input == "" {
				t.Errorf("Example %d of skill %s has empty input", i, skill.ID())
			}

			if example.Output == "" {
				t.Errorf("Example %d of skill %s has empty output", i, skill.ID())
			}

			// Reasoning 是可选的，不检查
		}
	}
}
