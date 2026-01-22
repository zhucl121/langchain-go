package skills

import (
	"context"
	"testing"
)

func TestNewSkillMetaTool(t *testing.T) {
	manager := NewSkillManager()
	metaTool := NewSkillMetaTool(manager)

	if metaTool == nil {
		t.Fatal("Expected non-nil meta tool")
	}

	if metaTool.GetName() != "use_skill" {
		t.Errorf("Expected name 'use_skill', got '%s'", metaTool.GetName())
	}
}

func TestSkillMetaTool_GetDescription(t *testing.T) {
	manager := NewSkillManager()
	metaTool := NewSkillMetaTool(manager)

	desc := metaTool.GetDescription()
	if desc == "" {
		t.Error("Expected non-empty description")
	}
}

func TestSkillMetaTool_GetParameters(t *testing.T) {
	manager := NewSkillManager()
	metaTool := NewSkillMetaTool(manager)

	params := metaTool.GetParameters()

	if params.Type != "object" {
		t.Error("Expected object type")
	}

	// 验证必需参数
	if _, ok := params.Properties["skill_name"]; !ok {
		t.Error("Expected skill_name parameter")
	}

	if _, ok := params.Properties["action"]; !ok {
		t.Error("Expected action parameter")
	}

	if _, ok := params.Properties["params"]; !ok {
		t.Error("Expected params parameter")
	}

	if _, ok := params.Properties["list_skills"]; !ok {
		t.Error("Expected list_skills parameter")
	}
}

func TestSkillMetaTool_ListSkills(t *testing.T) {
	ctx := context.Background()
	manager := NewSkillManager()

	// 注册一些 Skills
	skill1 := NewBaseSkill(
		WithID("skill1"),
		WithName("Skill 1"),
		WithDescription("First skill"),
	)
	skill2 := NewBaseSkill(
		WithID("skill2"),
		WithName("Skill 2"),
		WithDescription("Second skill"),
	)

	manager.Register(skill1)
	manager.Register(skill2)

	metaTool := NewSkillMetaTool(manager)

	// 列出所有 Skills
	result, err := metaTool.Execute(ctx, map[string]any{
		"list_skills": true,
	})

	if err != nil {
		t.Fatalf("Failed to list skills: %v", err)
	}

	resultMap, ok := result.(map[string]any)
	if !ok {
		t.Fatal("Expected result to be a map")
	}

	skills, ok := resultMap["skills"].([]map[string]any)
	if !ok {
		t.Fatal("Expected skills to be a slice")
	}

	if len(skills) != 2 {
		t.Errorf("Expected 2 skills, got %d", len(skills))
	}
}

func TestSkillMetaTool_Execute_MissingSkillName(t *testing.T) {
	ctx := context.Background()
	manager := NewSkillManager()
	metaTool := NewSkillMetaTool(manager)

	// 没有提供 skill_name
	_, err := metaTool.Execute(ctx, map[string]any{
		"action": "test",
	})

	if err == nil {
		t.Error("Expected error for missing skill_name")
	}
}

func TestSkillMetaTool_Execute_NonExistentSkill(t *testing.T) {
	ctx := context.Background()
	manager := NewSkillManager()
	metaTool := NewSkillMetaTool(manager)

	// 不存在的 Skill
	_, err := metaTool.Execute(ctx, map[string]any{
		"skill_name": "non-existent",
		"action":     "test",
	})

	if err == nil {
		t.Error("Expected error for non-existent skill")
	}
}

func TestSkillMetaTool_Execute_Success(t *testing.T) {
	ctx := context.Background()
	manager := NewSkillManager()

	// 注册 Skill
	skill := NewBaseSkill(
		WithID("test-skill"),
		WithName("Test Skill"),
	)
	manager.Register(skill)

	metaTool := NewSkillMetaTool(manager)

	// 执行 Skill
	result, err := metaTool.Execute(ctx, map[string]any{
		"skill_name": "test-skill",
		"action":     "test-action",
		"params": map[string]any{
			"key": "value",
		},
	})

	if err != nil {
		t.Fatalf("Failed to execute skill: %v", err)
	}

	if result == nil {
		t.Error("Expected non-nil result")
	}
}

func TestSkillInfo(t *testing.T) {
	skill := NewBaseSkill(
		WithID("test-skill"),
		WithName("Test Skill"),
		WithDescription("A test skill"),
		WithCategory(CategoryCoding),
		WithTags("test", "example"),
	)

	info := SkillInfo(skill)

	if info == "" {
		t.Error("Expected non-empty skill info")
	}

	// 应该包含关键信息
	if !contains(info, "Test Skill") {
		t.Error("Expected info to contain skill name")
	}

	if !contains(info, "test-skill") {
		t.Error("Expected info to contain skill ID")
	}
}

func TestGetAllSkillsInfo(t *testing.T) {
	manager := NewSkillManager()

	// 注册多个 Skills
	for i := 1; i <= 3; i++ {
		skill := NewBaseSkill(
			WithID("skill" + string(rune('0'+i))),
			WithName("Skill " + string(rune('0'+i))),
		)
		manager.Register(skill)
	}

	info := GetAllSkillsInfo(manager)

	if info == "" {
		t.Error("Expected non-empty info")
	}

	if !contains(info, "Available Skills (3)") {
		t.Error("Expected info to contain skill count")
	}
}

func TestEstimateTokensForSkillList(t *testing.T) {
	manager := NewSkillManager()

	// 注册 Skills
	for i := 0; i < 10; i++ {
		skill := NewBaseSkill(
			WithID("skill"),
			WithName("Skill"),
		)
		manager.Register(skill)
	}

	tokens := EstimateTokensForSkillList(manager)

	if tokens == 0 {
		t.Error("Expected non-zero token count")
	}
}

func TestCompareTokenUsage(t *testing.T) {
	comparison := CompareTokenUsage(100)

	if comparison["skill_count"] != 100 {
		t.Error("Expected skill_count to be 100")
	}

	traditionalTokens := comparison["traditional_tokens"].(int)
	metaToolTokens := comparison["meta_tool_tokens"].(int)

	if traditionalTokens <= metaToolTokens {
		t.Error("Expected traditional tokens to be greater than meta tool tokens")
	}

	tokensSaved := comparison["tokens_saved"].(int)
	if tokensSaved <= 0 {
		t.Error("Expected positive tokens saved")
	}

	reductionPercent := comparison["reduction_percent"].(float64)
	if reductionPercent <= 0 || reductionPercent >= 100 {
		t.Errorf("Expected reduction percent between 0 and 100, got %.2f", reductionPercent)
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
