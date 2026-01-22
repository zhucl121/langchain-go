package skills

import (
	"context"
	"time"

	"github.com/zhucl121/langchain-go/core/tools"
)

// Skill 表示一个可加载的智能体能力模块。
//
// Skill 是独立的能力单元，包含：
//   - 专业化的提示词和知识
//   - 特定领域的工具
//   - 执行逻辑和状态
//
// 示例:
//
//	codingSkill := skills.NewCodingSkill()
//	agent.LoadSkill(codingSkill)
//	defer agent.UnloadSkill(codingSkill.ID())
type Skill interface {
	// ID 返回 Skill 唯一标识
	ID() string

	// Name 返回 Skill 显示名称
	Name() string

	// Description 返回 Skill 描述
	Description() string

	// Category 返回 Skill 分类
	Category() SkillCategory

	// Tags 返回 Skill 标签（用于搜索和过滤）
	Tags() []string

	// Load 加载 Skill（初始化、注册工具等）
	//
	// 参数:
	//   - ctx: 上下文
	//   - config: 加载配置
	//
	// 返回:
	//   - error: 加载失败时返回错误
	Load(ctx context.Context, config *LoadConfig) error

	// Unload 卸载 Skill（清理资源）
	//
	// 参数:
	//   - ctx: 上下文
	//
	// 返回:
	//   - error: 卸载失败时返回错误
	Unload(ctx context.Context) error

	// IsLoaded 返回 Skill 是否已加载
	IsLoaded() bool

	// GetTools 返回 Skill 提供的工具列表
	GetTools() []tools.Tool

	// GetSystemPrompt 返回 Skill 的系统提示词
	GetSystemPrompt() string

	// GetExamples 返回 Skill 的示例（用于 Few-shot Learning）
	GetExamples() []SkillExample

	// GetMetadata 返回 Skill 的元数据
	GetMetadata() *SkillMetadata

	// Dependencies 返回 Skill 的依赖项（其他 Skill ID）
	Dependencies() []string
}

// SkillCategory Skill 分类
type SkillCategory string

const (
	// CategoryCoding 编程相关
	CategoryCoding SkillCategory = "coding"

	// CategoryDataAnalysis 数据分析
	CategoryDataAnalysis SkillCategory = "data_analysis"

	// CategoryKnowledge 知识问答
	CategoryKnowledge SkillCategory = "knowledge"

	// CategoryCreative 创意写作
	CategoryCreative SkillCategory = "creative"

	// CategoryResearch 研究调研
	CategoryResearch SkillCategory = "research"

	// CategoryAutomation 自动化
	CategoryAutomation SkillCategory = "automation"

	// CategoryCommunication 沟通
	CategoryCommunication SkillCategory = "communication"

	// CategoryGeneral 通用
	CategoryGeneral SkillCategory = "general"
)

// SkillExample Skill 示例（Few-shot）
type SkillExample struct {
	// Input 输入示例
	Input string `json:"input"`

	// Output 输出示例
	Output string `json:"output"`

	// Reasoning 推理过程（可选）
	Reasoning string `json:"reasoning,omitempty"`

	// Metadata 元数据（可选）
	Metadata map[string]any `json:"metadata,omitempty"`
}

// SkillMetadata Skill 元数据
type SkillMetadata struct {
	// Version Skill 版本
	Version string `json:"version"`

	// Author 作者
	Author string `json:"author"`

	// CreatedAt 创建时间
	CreatedAt time.Time `json:"created_at"`

	// UpdatedAt 更新时间
	UpdatedAt time.Time `json:"updated_at"`

	// License 许可证
	License string `json:"license"`

	// Repository 代码仓库（可选）
	Repository string `json:"repository,omitempty"`

	// Extra 额外信息（可选）
	Extra map[string]any `json:"extra,omitempty"`
}

// LoadConfig Skill 加载配置
type LoadConfig struct {
	// Lazy 是否延迟加载（仅在首次使用时初始化）
	Lazy bool

	// AutoLoadDependencies 是否自动加载依赖
	AutoLoadDependencies bool

	// Context 上下文数据
	Context map[string]any
}

// DefaultLoadConfig 返回默认加载配置
func DefaultLoadConfig() *LoadConfig {
	return &LoadConfig{
		Lazy:                 false,
		AutoLoadDependencies: true,
		Context:              make(map[string]any),
	}
}
