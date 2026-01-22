// Package skills 提供 Agent Skill 系统实现。
//
// Skill 是可加载的智能体能力模块，包含专业化的提示词、工具和执行逻辑。
// Skill 系统采用渐进式披露（Progressive Disclosure）设计模式，按需加载专业能力。
//
// # 核心概念
//
// Skill 是独立的能力单元，每个 Skill 专注于特定领域：
//   - 编程能力（Coding Skill）
//   - 数据分析能力（Data Analysis Skill）
//   - 知识问答能力（Knowledge Query Skill）
//   - 研究调研能力（Research Skill）
//
// # 基础使用
//
//	// 创建 Skill 管理器
//	manager := skills.NewSkillManager()
//
//	// 注册 Skill
//	codingSkill := builtin.NewCodingSkill()
//	manager.Register(codingSkill)
//
//	// 加载 Skill
//	config := &skills.LoadConfig{
//	    AutoLoadDependencies: true,
//	}
//	manager.Load(ctx, "coding", config)
//
//	// 使用 Skill
//	tools := codingSkill.GetTools()
//	prompt := codingSkill.GetSystemPrompt()
//
// # 自定义 Skill
//
// 使用 BaseSkill 快速创建自定义 Skill：
//
//	func NewMySkill() skills.Skill {
//	    return skills.NewBaseSkill(
//	        skills.WithID("my-skill"),
//	        skills.WithName("我的技能"),
//	        skills.WithCategory(skills.CategoryGeneral),
//	        skills.WithSystemPrompt("你是一个..."),
//	        skills.WithTools(tool1, tool2),
//	    )
//	}
//
// 或实现 Skill 接口：
//
//	type MySkill struct {
//	    id     string
//	    name   string
//	    loaded bool
//	}
//
//	func (s *MySkill) ID() string { return s.id }
//	func (s *MySkill) Name() string { return s.name }
//	// ... 实现其他方法
//
// # Agent 集成
//
// Skill 可以无缝集成到现有 Agent：
//
//	executor := agents.NewAgentExecutor(agents.AgentConfig{
//	    Type:          agents.AgentTypeReAct,
//	    LLM:           chatModel,
//	    SkillManager:  skillManager,
//	    EnabledSkills: []string{"coding", "data-analysis"},
//	})
//
// Agent 会自动：
//   - 加载指定的 Skill
//   - 聚合 Skill 提供的工具
//   - 组合 Skill 的系统提示词
//
// # 依赖管理
//
// Skill 支持依赖管理：
//
//	advancedSkill := skills.NewBaseSkill(
//	    skills.WithID("advanced-coding"),
//	    skills.WithDependencies("coding", "data-analysis"),
//	)
//
//	// 自动加载依赖
//	manager.LoadWithDependencies(ctx, "advanced-coding", config)
//
// 系统会自动检测循环依赖并返回错误。
//
// # 性能
//
//   - Skill 加载: < 10ms
//   - 工具查找: < 1ms
//   - 零开销: 未加载 Skill 时无性能影响
//
// 更多信息请参考: https://github.com/zhucl121/langchain-go/docs/V0.5.1_USER_GUIDE.md
package skills
