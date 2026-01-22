package skills

import (
	"context"
	"errors"
	"testing"
)

func TestNewSkillManager(t *testing.T) {
	manager := NewSkillManager()

	if manager == nil {
		t.Fatal("Expected non-nil manager")
	}

	if manager.Count() != 0 {
		t.Errorf("Expected 0 skills, got %d", manager.Count())
	}
}

func TestSkillManagerRegisterUnregister(t *testing.T) {
	manager := NewSkillManager()

	skill := NewBaseSkill(
		WithID("test-skill"),
		WithName("Test Skill"),
	)

	// 注册
	if err := manager.Register(skill); err != nil {
		t.Fatalf("Failed to register skill: %v", err)
	}

	if manager.Count() != 1 {
		t.Errorf("Expected 1 skill, got %d", manager.Count())
	}

	// 重复注册应该报错
	if err := manager.Register(skill); !errors.Is(err, ErrSkillAlreadyRegistered) {
		t.Errorf("Expected ErrSkillAlreadyRegistered, got %v", err)
	}

	// 注销
	if err := manager.Unregister("test-skill"); err != nil {
		t.Fatalf("Failed to unregister skill: %v", err)
	}

	if manager.Count() != 0 {
		t.Errorf("Expected 0 skills after unregister, got %d", manager.Count())
	}

	// 注销不存在的 Skill 应该报错
	if err := manager.Unregister("non-existent"); !errors.Is(err, ErrSkillNotFound) {
		t.Errorf("Expected ErrSkillNotFound, got %v", err)
	}
}

func TestSkillManagerLoadUnload(t *testing.T) {
	ctx := context.Background()
	manager := NewSkillManager()

	skill := NewBaseSkill(
		WithID("test-skill"),
		WithName("Test Skill"),
	)

	manager.Register(skill)

	// 加载
	config := DefaultLoadConfig()
	if err := manager.Load(ctx, "test-skill", config); err != nil {
		t.Fatalf("Failed to load skill: %v", err)
	}

	if !manager.IsLoaded("test-skill") {
		t.Error("Expected skill to be loaded")
	}

	if manager.LoadedCount() != 1 {
		t.Errorf("Expected 1 loaded skill, got %d", manager.LoadedCount())
	}

	// 重复加载应该报错
	if err := manager.Load(ctx, "test-skill", config); !errors.Is(err, ErrSkillAlreadyLoaded) {
		t.Errorf("Expected ErrSkillAlreadyLoaded, got %v", err)
	}

	// 卸载
	if err := manager.Unload(ctx, "test-skill"); err != nil {
		t.Fatalf("Failed to unload skill: %v", err)
	}

	if manager.IsLoaded("test-skill") {
		t.Error("Expected skill to not be loaded after unload")
	}

	if manager.LoadedCount() != 0 {
		t.Errorf("Expected 0 loaded skills, got %d", manager.LoadedCount())
	}
}

func TestSkillManagerGet(t *testing.T) {
	manager := NewSkillManager()

	skill := NewBaseSkill(
		WithID("test-skill"),
		WithName("Test Skill"),
	)

	manager.Register(skill)

	// 获取存在的 Skill
	retrieved, err := manager.Get("test-skill")
	if err != nil {
		t.Fatalf("Failed to get skill: %v", err)
	}

	if retrieved.ID() != "test-skill" {
		t.Errorf("Expected ID 'test-skill', got '%s'", retrieved.ID())
	}

	// 获取不存在的 Skill
	_, err = manager.Get("non-existent")
	if !errors.Is(err, ErrSkillNotFound) {
		t.Errorf("Expected ErrSkillNotFound, got %v", err)
	}
}

func TestSkillManagerList(t *testing.T) {
	manager := NewSkillManager()

	skill1 := NewBaseSkill(WithID("skill1"), WithName("Skill 1"))
	skill2 := NewBaseSkill(WithID("skill2"), WithName("Skill 2"))
	skill3 := NewBaseSkill(WithID("skill3"), WithName("Skill 3"))

	manager.Register(skill1)
	manager.Register(skill2)
	manager.Register(skill3)

	// 列出所有 Skill
	skills := manager.List()
	if len(skills) != 3 {
		t.Errorf("Expected 3 skills, got %d", len(skills))
	}
}

func TestSkillManagerListLoaded(t *testing.T) {
	ctx := context.Background()
	manager := NewSkillManager()

	skill1 := NewBaseSkill(WithID("skill1"), WithName("Skill 1"))
	skill2 := NewBaseSkill(WithID("skill2"), WithName("Skill 2"))
	skill3 := NewBaseSkill(WithID("skill3"), WithName("Skill 3"))

	manager.Register(skill1)
	manager.Register(skill2)
	manager.Register(skill3)

	// 加载部分 Skill
	config := DefaultLoadConfig()
	manager.Load(ctx, "skill1", config)
	manager.Load(ctx, "skill3", config)

	// 列出已加载的 Skill
	loadedSkills := manager.ListLoaded()
	if len(loadedSkills) != 2 {
		t.Errorf("Expected 2 loaded skills, got %d", len(loadedSkills))
	}

	// 验证加载的是正确的 Skill
	loadedIDs := make(map[string]bool)
	for _, skill := range loadedSkills {
		loadedIDs[skill.ID()] = true
	}

	if !loadedIDs["skill1"] || !loadedIDs["skill3"] {
		t.Error("Expected skill1 and skill3 to be loaded")
	}

	if loadedIDs["skill2"] {
		t.Error("Expected skill2 to not be loaded")
	}
}

func TestSkillManagerFindByCategory(t *testing.T) {
	manager := NewSkillManager()

	skill1 := NewBaseSkill(WithID("skill1"), WithCategory(CategoryCoding))
	skill2 := NewBaseSkill(WithID("skill2"), WithCategory(CategoryCoding))
	skill3 := NewBaseSkill(WithID("skill3"), WithCategory(CategoryDataAnalysis))

	manager.Register(skill1)
	manager.Register(skill2)
	manager.Register(skill3)

	// 查找编程类 Skill
	codingSkills := manager.FindByCategory(CategoryCoding)
	if len(codingSkills) != 2 {
		t.Errorf("Expected 2 coding skills, got %d", len(codingSkills))
	}

	// 查找数据分析类 Skill
	dataSkills := manager.FindByCategory(CategoryDataAnalysis)
	if len(dataSkills) != 1 {
		t.Errorf("Expected 1 data analysis skill, got %d", len(dataSkills))
	}

	// 查找不存在的分类
	unknownSkills := manager.FindByCategory("unknown")
	if len(unknownSkills) != 0 {
		t.Errorf("Expected 0 unknown skills, got %d", len(unknownSkills))
	}
}

func TestSkillManagerFindByTags(t *testing.T) {
	manager := NewSkillManager()

	skill1 := NewBaseSkill(WithID("skill1"), WithTags("tag1", "tag2"))
	skill2 := NewBaseSkill(WithID("skill2"), WithTags("tag2", "tag3"))
	skill3 := NewBaseSkill(WithID("skill3"), WithTags("tag4"))

	manager.Register(skill1)
	manager.Register(skill2)
	manager.Register(skill3)

	// 查找包含 tag2 的 Skill
	skills := manager.FindByTags([]string{"tag2"})
	if len(skills) != 2 {
		t.Errorf("Expected 2 skills with tag2, got %d", len(skills))
	}

	// 查找包含多个标签的 Skill（或关系）
	skills = manager.FindByTags([]string{"tag1", "tag3"})
	if len(skills) < 2 {
		t.Errorf("Expected at least 2 skills, got %d", len(skills))
	}

	// 查找不存在的标签
	skills = manager.FindByTags([]string{"nonexistent"})
	if len(skills) != 0 {
		t.Errorf("Expected 0 skills, got %d", len(skills))
	}
}

func TestSkillManagerDependencies(t *testing.T) {
	ctx := context.Background()
	manager := NewSkillManager()

	// 创建有依赖关系的 Skills
	baseSkill := NewBaseSkill(
		WithID("base"),
		WithName("Base Skill"),
	)

	dependentSkill := NewBaseSkill(
		WithID("dependent"),
		WithName("Dependent Skill"),
		WithDependencies("base"),
	)

	manager.Register(baseSkill)
	manager.Register(dependentSkill)

	// 使用 LoadWithDependencies 加载
	config := DefaultLoadConfig()
	if err := manager.LoadWithDependencies(ctx, "dependent", config); err != nil {
		t.Fatalf("Failed to load with dependencies: %v", err)
	}

	// 验证依赖也被加载
	if !manager.IsLoaded("base") {
		t.Error("Expected base skill to be loaded")
	}

	if !manager.IsLoaded("dependent") {
		t.Error("Expected dependent skill to be loaded")
	}
}

func TestSkillManagerCircularDependency(t *testing.T) {
	ctx := context.Background()
	manager := NewSkillManager()

	// 创建循环依赖
	skill1 := NewBaseSkill(
		WithID("skill1"),
		WithDependencies("skill2"),
	)

	skill2 := NewBaseSkill(
		WithID("skill2"),
		WithDependencies("skill1"),
	)

	manager.Register(skill1)
	manager.Register(skill2)

	// 尝试加载应该检测到循环依赖
	config := DefaultLoadConfig()
	err := manager.LoadWithDependencies(ctx, "skill1", config)

	if !errors.Is(err, ErrCircularDependency) {
		t.Errorf("Expected ErrCircularDependency, got %v", err)
	}
}

func TestSkillManagerUnregisterLoadedSkill(t *testing.T) {
	ctx := context.Background()
	manager := NewSkillManager()

	skill := NewBaseSkill(
		WithID("test-skill"),
		WithName("Test Skill"),
	)

	manager.Register(skill)

	// 加载 Skill
	config := DefaultLoadConfig()
	manager.Load(ctx, "test-skill", config)

	// 注销已加载的 Skill（应该先卸载再注销）
	if err := manager.Unregister("test-skill"); err != nil {
		t.Fatalf("Failed to unregister loaded skill: %v", err)
	}

	// 验证已被注销
	if manager.Count() != 0 {
		t.Errorf("Expected 0 skills, got %d", manager.Count())
	}
}

func TestSkillManagerNilSkill(t *testing.T) {
	manager := NewSkillManager()

	// 注册 nil Skill
	if err := manager.Register(nil); err == nil {
		t.Error("Expected error when registering nil skill")
	}
}

func TestSkillManagerEmptyID(t *testing.T) {
	manager := NewSkillManager()

	skill := NewBaseSkill(
		WithID(""),
		WithName("Invalid Skill"),
	)

	// 注册空 ID 的 Skill
	if err := manager.Register(skill); err == nil {
		t.Error("Expected error when registering skill with empty ID")
	}
}
