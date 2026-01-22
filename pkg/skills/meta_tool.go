package skills

import (
	"context"
	"fmt"

	"github.com/zhucl121/langchain-go/pkg/types"
)

// SkillMetaTool 元工具：统一管理所有 Skills 的工具
//
// 核心概念：通过单一的 Skill 工具管理所有 Skills。
// - 不是：100 个 Skills = 100 个工具
// - 而是：100 个 Skills = 1 个 Skill 工具（元工具）
//
// 优势：
//   - 避免工具列表爆炸
//   - 统一管理
//   - 实现动态加载
//   - 节省 LLM 上下文 Token
type SkillMetaTool struct {
	manager SkillManager
	verbose bool
}

// NewSkillMetaTool 创建元工具
//
// 示例:
//
//	metaTool := skills.NewSkillMetaTool(skillManager)
//	agent.AddTool(metaTool) // 只需要添加这一个工具
func NewSkillMetaTool(manager SkillManager) *SkillMetaTool {
	return &SkillMetaTool{
		manager: manager,
		verbose: false,
	}
}

// WithVerbose 设置详细日志
func (t *SkillMetaTool) WithVerbose(verbose bool) *SkillMetaTool {
	t.verbose = verbose
	return t
}

// GetName 实现 Tool 接口
func (t *SkillMetaTool) GetName() string {
	return "use_skill"
}

// GetDescription 实现 Tool 接口
func (t *SkillMetaTool) GetDescription() string {
	return `Use a specific skill to accomplish a task. 
This is a meta-tool that provides access to all available skills.

Available skills can be queried using list_skills parameter.

Usage:
1. Set list_skills=true to see all available skills
2. Choose a skill by setting skill_name
3. Provide required parameters in the params field`
}

// GetParameters 实现 Tool 接口
func (t *SkillMetaTool) GetParameters() types.Schema {
	return types.Schema{
		Type: "object",
		Properties: map[string]types.Schema{
			"skill_name": {
				Type:        "string",
				Description: "Name of the skill to use (e.g., 'coding', 'data-analysis', 'knowledge-query', 'research')",
			},
			"action": {
				Type:        "string",
				Description: "Action to perform with the skill (e.g., 'analyze', 'generate', 'query', 'execute')",
			},
			"params": {
				Type:        "object",
				Description: "Parameters for the skill action",
			},
			"list_skills": {
				Type:        "boolean",
				Description: "Set to true to list all available skills",
			},
		},
		Required: []string{}, // list_skills 或 skill_name 至少提供一个
	}
}

// Execute 实现 Tool 接口
func (t *SkillMetaTool) Execute(ctx context.Context, args map[string]any) (any, error) {
	// 1. 检查是否查询可用 Skills
	if listSkills, ok := args["list_skills"].(bool); ok && listSkills {
		return t.listAvailableSkills(ctx)
	}

	// 2. 获取 skill_name
	skillName, ok := args["skill_name"].(string)
	if !ok || skillName == "" {
		return nil, fmt.Errorf("skill_name is required")
	}

	// 3. 获取或加载 Skill
	skill, err := t.getOrLoadSkill(ctx, skillName)
	if err != nil {
		return nil, fmt.Errorf("failed to get skill %s: %w", skillName, err)
	}

	// 4. 加载 Level 2 指令
	if progressiveSkill, ok := skill.(ProgressiveSkill); ok {
		if !progressiveSkill.IsInstructionsLoaded() {
			if _, err := progressiveSkill.LoadInstructions(ctx); err != nil {
				return nil, fmt.Errorf("failed to load instructions for skill %s: %w", skillName, err)
			}

			if t.verbose {
				fmt.Printf("[MetaTool] Loaded instructions for skill: %s\n", skillName)
			}
		}
	}

	// 5. 执行 Skill action
	action, _ := args["action"].(string)
	params, _ := args["params"].(map[string]any)

	result, err := t.executeSkillAction(ctx, skill, action, params)
	if err != nil {
		return nil, fmt.Errorf("failed to execute skill %s action %s: %w", skillName, action, err)
	}

	return result, nil
}

// listAvailableSkills 列出所有可用的 Skills（Level 1: 元数据）
func (t *SkillMetaTool) listAvailableSkills(ctx context.Context) (any, error) {
	allSkills := t.manager.List()

	result := make([]map[string]any, 0, len(allSkills))
	for _, skill := range allSkills {
		skillInfo := map[string]any{
			"id":          skill.ID(),
			"name":        skill.Name(),
			"description": skill.Description(),
			"category":    string(skill.Category()),
			"tags":        skill.Tags(),
		}
		result = append(result, skillInfo)
	}

	return map[string]any{
		"skills": result,
		"total":  len(result),
	}, nil
}

// getOrLoadSkill 获取或加载 Skill
func (t *SkillMetaTool) getOrLoadSkill(ctx context.Context, skillName string) (Skill, error) {
	// 尝试获取 Skill
	skill, err := t.manager.Get(skillName)
	if err != nil {
		return nil, err
	}

	// 检查是否已加载
	if !skill.IsLoaded() {
		config := DefaultLoadConfig()
		if err := t.manager.Load(ctx, skillName, config); err != nil {
			return nil, fmt.Errorf("failed to load skill: %w", err)
		}

		if t.verbose {
			fmt.Printf("[MetaTool] Loaded skill: %s\n", skillName)
		}
	}

	return skill, nil
}

// executeSkillAction 执行 Skill 的具体 action
func (t *SkillMetaTool) executeSkillAction(ctx context.Context, skill Skill, action string, params map[string]any) (any, error) {
	// 如果 Skill 实现了 ActionExecutor 接口，使用它
	if executor, ok := skill.(SkillActionExecutor); ok {
		return executor.ExecuteAction(ctx, action, params)
	}

	// 默认实现：返回 Skill 信息和参数
	return map[string]any{
		"skill":  skill.Name(),
		"action": action,
		"params": params,
		"result": fmt.Sprintf("Skill %s action %s executed with params: %v", skill.Name(), action, params),
	}, nil
}

// SkillActionExecutor Skill 动作执行器接口
//
// Skills 可以实现此接口来支持特定的 actions。
type SkillActionExecutor interface {
	// ExecuteAction 执行特定的 action
	ExecuteAction(ctx context.Context, action string, params map[string]any) (any, error)
}

// SkillInfo 提供 Skill 的简要信息（用于 LLM）
func SkillInfo(skill Skill) string {
	return fmt.Sprintf("%s (%s): %s [Category: %s, Tags: %v]",
		skill.Name(),
		skill.ID(),
		skill.Description(),
		skill.Category(),
		skill.Tags(),
	)
}

// GetAllSkillsInfo 获取所有 Skills 的简要信息
//
// 用于构建 LLM 的系统提示词，只包含 Level 1 元数据。
func GetAllSkillsInfo(manager SkillManager) string {
	allSkills := manager.List()

	info := fmt.Sprintf("Available Skills (%d):\n\n", len(allSkills))

	for i, skill := range allSkills {
		info += fmt.Sprintf("%d. %s\n", i+1, SkillInfo(skill))
	}

	return info
}

// EstimateTokensForSkillList 估算 Skills 列表的 Token 数量
//
// 粗略估算：~4 字符 = 1 Token
func EstimateTokensForSkillList(manager SkillManager) int {
	info := GetAllSkillsInfo(manager)
	return len(info) / 4 // 粗略估算
}

// CompareTokenUsage 对比使用元工具前后的 Token 消耗
func CompareTokenUsage(skillCount int) map[string]any {
	// 传统方式：每个 Skill 的工具定义 ~500 Tokens
	traditionalTokens := skillCount * 500

	// 元工具方式：
	// - 元工具定义：~200 Tokens
	// - Skills 列表（Level 1 元数据）：~100 Tokens/skill
	metaToolTokens := 200 + (skillCount * 100)

	return map[string]any{
		"skill_count":        skillCount,
		"traditional_tokens": traditionalTokens,
		"meta_tool_tokens":   metaToolTokens,
		"tokens_saved":       traditionalTokens - metaToolTokens,
		"reduction_percent":  float64(traditionalTokens-metaToolTokens) / float64(traditionalTokens) * 100,
	}
}
