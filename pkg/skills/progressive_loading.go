package skills

import (
	"context"
)

// LoadLevel 表示 Skill 的加载级别
type LoadLevel int

const (
	// LoadLevelMetadata Level 1: 只加载元数据（~100B/skill）
	// 包含：ID, Name, Description, Category, Tags
	// 使用场景：系统启动时，让 LLM 知道"有哪些" Skills 可用
	LoadLevelMetadata LoadLevel = 1

	// LoadLevelInstructions Level 2: 加载完整指令（~2-5KB/skill）
	// 包含：SystemPrompt, Examples, Parameters
	// 使用场景：LLM 调用 Skill 工具时，让 LLM 知道"如何使用"这个 Skill
	LoadLevelInstructions LoadLevel = 2

	// LoadLevelResources Level 3: 加载资源文件（~10-100KB/skill）
	// 包含：Scripts, Templates, Dependencies
	// 使用场景：LLM 执行脚本时，实际执行 Skill 的代码逻辑
	// 注意：脚本代码不进入 LLM 上下文
	LoadLevelResources LoadLevel = 3
)

// SkillInstructions Level 2: 完整指令
//
// 包含 Skill 的详细使用说明，用于指导 LLM 如何使用该 Skill。
type SkillInstructions struct {
	// SystemPrompt 系统提示词（详细的使用说明）
	SystemPrompt string

	// Examples Few-shot 示例
	Examples []SkillExample

	// Parameters 参数定义
	Parameters SkillParameters

	// UsageGuidelines 使用指南
	UsageGuidelines string

	// Limitations 局限性说明
	Limitations string
}

// SkillParameters 参数定义
type SkillParameters struct {
	// Required 必需参数
	Required []ParameterDef

	// Optional 可选参数
	Optional []ParameterDef
}

// ParameterDef 参数定义
type ParameterDef struct {
	Name        string         `json:"name"`
	Type        string         `json:"type"`
	Description string         `json:"description"`
	Default     any            `json:"default,omitempty"`
	Example     any            `json:"example,omitempty"`
	Constraints map[string]any `json:"constraints,omitempty"`
}

// SkillResources Level 3: 资源文件
//
// 包含 Skill 的脚本、模板等资源文件。
// 这些文件在执行时加载，不进入 LLM 上下文。
type SkillResources struct {
	// Scripts 脚本文件（name -> content）
	Scripts map[string]string

	// Templates 模板文件（name -> content）
	Templates map[string]string

	// Dependencies 依赖的包/库
	Dependencies []string

	// ConfigFiles 配置文件
	ConfigFiles map[string]string

	// DataFiles 数据文件
	DataFiles map[string][]byte
}

// ProgressiveSkill 支持渐进式加载的 Skill 接口
//
// 扩展了基础 Skill 接口，支持按需、分级加载。
type ProgressiveSkill interface {
	Skill

	// LoadInstructions 加载 Level 2: 完整指令
	//
	// 在 LLM 需要调用此 Skill 时才加载。
	LoadInstructions(ctx context.Context) (*SkillInstructions, error)

	// LoadResources 加载 Level 3: 资源文件
	//
	// 在需要执行脚本时才加载，不进入 LLM 上下文。
	LoadResources(ctx context.Context) (*SkillResources, error)

	// GetLoadLevel 获取当前加载级别
	GetLoadLevel() LoadLevel

	// IsInstructionsLoaded 检查 Level 2 是否已加载
	IsInstructionsLoaded() bool

	// IsResourcesLoaded 检查 Level 3 是否已加载
	IsResourcesLoaded() bool
}

// SkillInstructionsLoader Level 2 加载器
//
// 负责加载 Skill 的完整指令。
type SkillInstructionsLoader interface {
	// LoadInstructions 加载指令
	LoadInstructions(ctx context.Context, skillID string) (*SkillInstructions, error)
}

// SkillResourcesLoader Level 3 加载器
//
// 负责加载 Skill 的资源文件。
type SkillResourcesLoader interface {
	// LoadResources 加载资源
	LoadResources(ctx context.Context, skillID string) (*SkillResources, error)
}

// NewSkillInstructions 创建空的 SkillInstructions
func NewSkillInstructions() *SkillInstructions {
	return &SkillInstructions{
		Examples:   []SkillExample{},
		Parameters: SkillParameters{},
	}
}

// NewSkillResources 创建空的 SkillResources
func NewSkillResources() *SkillResources {
	return &SkillResources{
		Scripts:      make(map[string]string),
		Templates:    make(map[string]string),
		Dependencies: []string{},
		ConfigFiles:  make(map[string]string),
		DataFiles:    make(map[string][]byte),
	}
}

// EstimateSize 估算 SkillInstructions 的大小（字节）
func (si *SkillInstructions) EstimateSize() int {
	size := len(si.SystemPrompt) + len(si.UsageGuidelines) + len(si.Limitations)

	for _, example := range si.Examples {
		size += len(example.Input) + len(example.Output) + len(example.Reasoning)
	}

	return size
}

// EstimateSize 估算 SkillResources 的大小（字节）
func (sr *SkillResources) EstimateSize() int {
	size := 0

	for _, script := range sr.Scripts {
		size += len(script)
	}

	for _, template := range sr.Templates {
		size += len(template)
	}

	for _, config := range sr.ConfigFiles {
		size += len(config)
	}

	for _, data := range sr.DataFiles {
		size += len(data)
	}

	return size
}
