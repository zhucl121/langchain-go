package skills

import (
	"context"
	"testing"
)

func TestLoadLevel(t *testing.T) {
	levels := []LoadLevel{
		LoadLevelMetadata,
		LoadLevelInstructions,
		LoadLevelResources,
	}

	if len(levels) != 3 {
		t.Error("Expected 3 load levels")
	}

	if LoadLevelMetadata != 1 {
		t.Error("LoadLevelMetadata should be 1")
	}

	if LoadLevelInstructions != 2 {
		t.Error("LoadLevelInstructions should be 2")
	}

	if LoadLevelResources != 3 {
		t.Error("LoadLevelResources should be 3")
	}
}

func TestNewSkillInstructions(t *testing.T) {
	instructions := NewSkillInstructions()

	if instructions == nil {
		t.Fatal("Expected non-nil instructions")
	}

	if instructions.Examples == nil {
		t.Error("Expected examples to be initialized")
	}
}

func TestNewSkillResources(t *testing.T) {
	resources := NewSkillResources()

	if resources == nil {
		t.Fatal("Expected non-nil resources")
	}

	if resources.Scripts == nil {
		t.Error("Expected scripts to be initialized")
	}

	if resources.Templates == nil {
		t.Error("Expected templates to be initialized")
	}

	if resources.ConfigFiles == nil {
		t.Error("Expected config files to be initialized")
	}

	if resources.DataFiles == nil {
		t.Error("Expected data files to be initialized")
	}
}

func TestSkillInstructions_EstimateSize(t *testing.T) {
	instructions := &SkillInstructions{
		SystemPrompt: "Test prompt", // 11 bytes
		Examples: []SkillExample{
			{Input: "test", Output: "result"}, // 4 + 6 = 10 bytes
		},
	}

	size := instructions.EstimateSize()

	// 11 (prompt) + 10 (example) = 21
	if size != 21 {
		t.Errorf("Expected size 21, got %d", size)
	}
}

func TestSkillResources_EstimateSize(t *testing.T) {
	resources := &SkillResources{
		Scripts: map[string]string{
			"test.py": "print('hello')", // 14 bytes
		},
		Templates: map[string]string{
			"tmpl": "template", // 8 bytes
		},
	}

	size := resources.EstimateSize()

	// 14 (script) + 8 (template) = 22
	if size != 22 {
		t.Errorf("Expected size 22, got %d", size)
	}
}

func TestProgressiveBaseSkill_LoadLevels(t *testing.T) {
	ctx := context.Background()

	skill := NewProgressiveBaseSkill(
		WithProgressiveID("test-skill"),
		WithProgressiveName("Test Skill"),
		WithProgressiveDescription("A test skill"),
	)

	// 初始状态：Level 1
	if skill.GetLoadLevel() != LoadLevelMetadata {
		t.Errorf("Expected LoadLevelMetadata, got %v", skill.GetLoadLevel())
	}

	// 加载 Level 2
	instructions, err := skill.LoadInstructions(ctx)
	if err != nil {
		t.Fatalf("Failed to load instructions: %v", err)
	}

	if instructions == nil {
		t.Error("Expected non-nil instructions")
	}

	if skill.GetLoadLevel() != LoadLevelInstructions {
		t.Errorf("Expected LoadLevelInstructions, got %v", skill.GetLoadLevel())
	}

	if !skill.IsInstructionsLoaded() {
		t.Error("Expected instructions to be loaded")
	}

	// 加载 Level 3
	resources, err := skill.LoadResources(ctx)
	if err != nil {
		t.Fatalf("Failed to load resources: %v", err)
	}

	if resources == nil {
		t.Error("Expected non-nil resources")
	}

	if skill.GetLoadLevel() != LoadLevelResources {
		t.Errorf("Expected LoadLevelResources, got %v", skill.GetLoadLevel())
	}

	if !skill.IsResourcesLoaded() {
		t.Error("Expected resources to be loaded")
	}
}

func TestProgressiveBaseSkill_LazyLoading(t *testing.T) {
	ctx := context.Background()

	skill := NewProgressiveBaseSkill(
		WithProgressiveID("test-skill"),
		WithProgressiveName("Test Skill"),
	)

	// 第一次调用：加载
	instructions1, err := skill.LoadInstructions(ctx)
	if err != nil {
		t.Fatalf("Failed to load instructions: %v", err)
	}

	// 第二次调用：应该返回缓存的数据
	instructions2, err := skill.LoadInstructions(ctx)
	if err != nil {
		t.Fatalf("Failed to load instructions: %v", err)
	}

	// 应该是同一个对象
	if instructions1 != instructions2 {
		t.Error("Expected instructions to be cached")
	}
}

func TestProgressiveBaseSkill_Unload(t *testing.T) {
	ctx := context.Background()

	skill := NewProgressiveBaseSkill(
		WithProgressiveID("test-skill"),
		WithProgressiveName("Test Skill"),
	)

	// 加载
	config := DefaultLoadConfig()
	skill.Load(ctx, config)

	// 加载 Level 2 和 Level 3
	skill.LoadInstructions(ctx)
	skill.LoadResources(ctx)

	if skill.GetLoadLevel() != LoadLevelResources {
		t.Error("Expected LoadLevelResources")
	}

	// 卸载
	if err := skill.Unload(ctx); err != nil {
		t.Fatalf("Failed to unload: %v", err)
	}

	// 应该回到 Level 1
	if skill.GetLoadLevel() != LoadLevelMetadata {
		t.Errorf("Expected LoadLevelMetadata after unload, got %v", skill.GetLoadLevel())
	}

	if skill.IsInstructionsLoaded() {
		t.Error("Instructions should not be loaded after unload")
	}

	if skill.IsResourcesLoaded() {
		t.Error("Resources should not be loaded after unload")
	}
}

func TestProgressiveBaseSkill_Metadata(t *testing.T) {
	skill := NewProgressiveBaseSkill(
		WithProgressiveID("test-skill"),
		WithProgressiveName("Test Skill"),
		WithProgressiveDescription("Test description"),
		WithProgressiveCategory(CategoryCoding),
		WithProgressiveTags("tag1", "tag2"),
	)

	// 验证元数据（Level 1）可以无需加载立即访问
	if skill.ID() != "test-skill" {
		t.Errorf("Expected ID 'test-skill', got '%s'", skill.ID())
	}

	if skill.Name() != "Test Skill" {
		t.Errorf("Expected name 'Test Skill', got '%s'", skill.Name())
	}

	if skill.Description() != "Test description" {
		t.Errorf("Expected description 'Test description', got '%s'", skill.Description())
	}

	if skill.Category() != CategoryCoding {
		t.Errorf("Expected category CategoryCoding, got %v", skill.Category())
	}

	tags := skill.Tags()
	if len(tags) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(tags))
	}
}
