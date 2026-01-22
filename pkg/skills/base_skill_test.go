package skills

import (
	"context"
	"testing"
)

func TestNewBaseSkill(t *testing.T) {
	skill := NewBaseSkill(
		WithID("test-skill"),
		WithName("Test Skill"),
		WithDescription("A test skill"),
		WithCategory(CategoryGeneral),
		WithTags("test", "example"),
	)

	if skill == nil {
		t.Fatal("Expected non-nil skill")
	}

	if skill.ID() != "test-skill" {
		t.Errorf("Expected ID 'test-skill', got '%s'", skill.ID())
	}

	if skill.Name() != "Test Skill" {
		t.Errorf("Expected name 'Test Skill', got '%s'", skill.Name())
	}

	if skill.Description() != "A test skill" {
		t.Errorf("Expected description 'A test skill', got '%s'", skill.Description())
	}

	if skill.Category() != CategoryGeneral {
		t.Errorf("Expected category CategoryGeneral, got %v", skill.Category())
	}

	tags := skill.Tags()
	if len(tags) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(tags))
	}
}

func TestBaseSkillLoadUnload(t *testing.T) {
	ctx := context.Background()

	skill := NewBaseSkill(
		WithID("test-skill"),
		WithName("Test Skill"),
	)

	// 初始状态
	if skill.IsLoaded() {
		t.Error("Expected skill to not be loaded initially")
	}

	// 加载
	config := DefaultLoadConfig()
	if err := skill.Load(ctx, config); err != nil {
		t.Fatalf("Failed to load skill: %v", err)
	}

	if !skill.IsLoaded() {
		t.Error("Expected skill to be loaded")
	}

	// 重复加载应该报错
	if err := skill.Load(ctx, config); err != ErrSkillAlreadyLoaded {
		t.Errorf("Expected ErrSkillAlreadyLoaded, got %v", err)
	}

	// 卸载
	if err := skill.Unload(ctx); err != nil {
		t.Fatalf("Failed to unload skill: %v", err)
	}

	if skill.IsLoaded() {
		t.Error("Expected skill to not be loaded after unload")
	}

	// 重复卸载应该报错
	if err := skill.Unload(ctx); err != ErrSkillNotLoaded {
		t.Errorf("Expected ErrSkillNotLoaded, got %v", err)
	}
}

func TestBaseSkillWithLoadHook(t *testing.T) {
	ctx := context.Background()
	loaded := false

	skill := NewBaseSkill(
		WithID("test-skill"),
		WithLoadHook(func(ctx context.Context, config *LoadConfig) error {
			loaded = true
			return nil
		}),
	)

	config := DefaultLoadConfig()
	if err := skill.Load(ctx, config); err != nil {
		t.Fatalf("Failed to load skill: %v", err)
	}

	if !loaded {
		t.Error("Expected load hook to be called")
	}
}

func TestBaseSkillWithUnloadHook(t *testing.T) {
	ctx := context.Background()
	unloaded := false

	skill := NewBaseSkill(
		WithID("test-skill"),
		WithUnloadHook(func(ctx context.Context) error {
			unloaded = true
			return nil
		}),
	)

	// 先加载
	config := DefaultLoadConfig()
	skill.Load(ctx, config)

	// 再卸载
	if err := skill.Unload(ctx); err != nil {
		t.Fatalf("Failed to unload skill: %v", err)
	}

	if !unloaded {
		t.Error("Expected unload hook to be called")
	}
}

func TestBaseSkillTools(t *testing.T) {
	skill := NewBaseSkill(
		WithID("test-skill"),
	)

	// 初始工具列表应为空
	tools := skill.GetTools()
	if len(tools) != 0 {
		t.Errorf("Expected 0 tools, got %d", len(tools))
	}

	// TODO: 添加实际工具测试（需要实现 Tool 接口）
}

func TestBaseSkillExamples(t *testing.T) {
	examples := []SkillExample{
		{Input: "example 1", Output: "output 1"},
		{Input: "example 2", Output: "output 2"},
	}

	skill := NewBaseSkill(
		WithID("test-skill"),
		WithExamples(examples...),
	)

	retrieved := skill.GetExamples()
	if len(retrieved) != 2 {
		t.Errorf("Expected 2 examples, got %d", len(retrieved))
	}
}

func TestBaseSkillMetadata(t *testing.T) {
	skill := NewBaseSkill(
		WithID("test-skill"),
		WithVersion("2.0.0"),
		WithAuthor("Test Author"),
		WithLicense("Apache-2.0"),
		WithRepository("https://github.com/test/repo"),
	)

	metadata := skill.GetMetadata()
	if metadata.Version != "2.0.0" {
		t.Errorf("Expected version '2.0.0', got '%s'", metadata.Version)
	}

	if metadata.Author != "Test Author" {
		t.Errorf("Expected author 'Test Author', got '%s'", metadata.Author)
	}

	if metadata.License != "Apache-2.0" {
		t.Errorf("Expected license 'Apache-2.0', got '%s'", metadata.License)
	}

	if metadata.Repository != "https://github.com/test/repo" {
		t.Errorf("Expected repository 'https://github.com/test/repo', got '%s'", metadata.Repository)
	}
}

func TestBaseSkillDependencies(t *testing.T) {
	skill := NewBaseSkill(
		WithID("test-skill"),
		WithDependencies("dep1", "dep2", "dep3"),
	)

	deps := skill.Dependencies()
	if len(deps) != 3 {
		t.Errorf("Expected 3 dependencies, got %d", len(deps))
	}

	expectedDeps := map[string]bool{"dep1": true, "dep2": true, "dep3": true}
	for _, dep := range deps {
		if !expectedDeps[dep] {
			t.Errorf("Unexpected dependency: %s", dep)
		}
	}
}

func TestBaseSkillAddRemoveTag(t *testing.T) {
	skill := NewBaseSkill(
		WithID("test-skill"),
		WithTags("tag1", "tag2"),
	)

	// 添加标签
	skill.AddTag("tag3")
	tags := skill.Tags()
	if len(tags) != 3 {
		t.Errorf("Expected 3 tags, got %d", len(tags))
	}

	// 添加重复标签（应该不增加）
	skill.AddTag("tag1")
	tags = skill.Tags()
	if len(tags) != 3 {
		t.Errorf("Expected 3 tags (no duplicate), got %d", len(tags))
	}

	// 移除标签
	skill.RemoveTag("tag2")
	tags = skill.Tags()
	if len(tags) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(tags))
	}
}

func TestBaseSkillSetSystemPrompt(t *testing.T) {
	skill := NewBaseSkill(
		WithID("test-skill"),
		WithSystemPrompt("Initial prompt"),
	)

	if skill.GetSystemPrompt() != "Initial prompt" {
		t.Error("Expected 'Initial prompt'")
	}

	// 修改提示词
	skill.SetSystemPrompt("Updated prompt")
	if skill.GetSystemPrompt() != "Updated prompt" {
		t.Error("Expected 'Updated prompt'")
	}
}

func TestBaseSkillAddExample(t *testing.T) {
	skill := NewBaseSkill(
		WithID("test-skill"),
	)

	// 初始没有示例
	if len(skill.GetExamples()) != 0 {
		t.Error("Expected 0 examples initially")
	}

	// 添加示例
	skill.AddExample(SkillExample{Input: "example 1", Output: "output 1"})
	skill.AddExample(SkillExample{Input: "example 2", Output: "output 2"})

	examples := skill.GetExamples()
	if len(examples) != 2 {
		t.Errorf("Expected 2 examples, got %d", len(examples))
	}
}
