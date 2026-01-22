package skills

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/zhucl121/langchain-go/core/tools"
)

// BaseSkill Skill 基础实现。
//
// 提供通用的 Skill 功能，具体 Skill 可以嵌入此结构体：
//
//	type CodingSkill struct {
//	    *skills.BaseSkill
//	    // 额外字段...
//	}
type BaseSkill struct {
	id           string
	name         string
	description  string
	category     SkillCategory
	tags         []string
	systemPrompt string
	examples     []SkillExample
	metadata     *SkillMetadata
	dependencies []string

	tools  []tools.Tool
	loaded bool
	mu     sync.RWMutex

	// Hooks
	onLoad   func(ctx context.Context, config *LoadConfig) error
	onUnload func(ctx context.Context) error
}

// NewBaseSkill 创建基础 Skill。
//
// 示例:
//
//	skill := skills.NewBaseSkill(
//	    skills.WithID("my-skill"),
//	    skills.WithName("我的技能"),
//	    skills.WithCategory(skills.CategoryGeneral),
//	)
func NewBaseSkill(opts ...BaseSkillOption) *BaseSkill {
	s := &BaseSkill{
		tags:         []string{},
		examples:     []SkillExample{},
		tools:        []tools.Tool{},
		dependencies: []string{},
		metadata: &SkillMetadata{
			Version:   "1.0.0",
			Author:    "LangChain-Go",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			License:   "MIT",
			Extra:     make(map[string]any),
		},
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

// BaseSkillOption 基础 Skill 选项
type BaseSkillOption func(*BaseSkill)

// WithID 设置 ID
func WithID(id string) BaseSkillOption {
	return func(s *BaseSkill) { s.id = id }
}

// WithName 设置名称
func WithName(name string) BaseSkillOption {
	return func(s *BaseSkill) { s.name = name }
}

// WithDescription 设置描述
func WithDescription(desc string) BaseSkillOption {
	return func(s *BaseSkill) { s.description = desc }
}

// WithCategory 设置分类
func WithCategory(cat SkillCategory) BaseSkillOption {
	return func(s *BaseSkill) { s.category = cat }
}

// WithTags 设置标签
func WithTags(tags ...string) BaseSkillOption {
	return func(s *BaseSkill) { s.tags = tags }
}

// WithSystemPrompt 设置系统提示词
func WithSystemPrompt(prompt string) BaseSkillOption {
	return func(s *BaseSkill) { s.systemPrompt = prompt }
}

// WithTools 设置工具
func WithTools(tools ...tools.Tool) BaseSkillOption {
	return func(s *BaseSkill) { s.tools = tools }
}

// WithExamples 设置示例
func WithExamples(examples ...SkillExample) BaseSkillOption {
	return func(s *BaseSkill) { s.examples = examples }
}

// WithDependencies 设置依赖
func WithDependencies(deps ...string) BaseSkillOption {
	return func(s *BaseSkill) { s.dependencies = deps }
}

// WithMetadata 设置元数据
func WithMetadata(metadata *SkillMetadata) BaseSkillOption {
	return func(s *BaseSkill) { s.metadata = metadata }
}

// WithVersion 设置版本
func WithVersion(version string) BaseSkillOption {
	return func(s *BaseSkill) {
		if s.metadata == nil {
			s.metadata = &SkillMetadata{}
		}
		s.metadata.Version = version
	}
}

// WithAuthor 设置作者
func WithAuthor(author string) BaseSkillOption {
	return func(s *BaseSkill) {
		if s.metadata == nil {
			s.metadata = &SkillMetadata{}
		}
		s.metadata.Author = author
	}
}

// WithLicense 设置许可证
func WithLicense(license string) BaseSkillOption {
	return func(s *BaseSkill) {
		if s.metadata == nil {
			s.metadata = &SkillMetadata{}
		}
		s.metadata.License = license
	}
}

// WithRepository 设置代码仓库
func WithRepository(repo string) BaseSkillOption {
	return func(s *BaseSkill) {
		if s.metadata == nil {
			s.metadata = &SkillMetadata{}
		}
		s.metadata.Repository = repo
	}
}

// WithLoadHook 设置加载钩子
func WithLoadHook(fn func(ctx context.Context, config *LoadConfig) error) BaseSkillOption {
	return func(s *BaseSkill) { s.onLoad = fn }
}

// WithUnloadHook 设置卸载钩子
func WithUnloadHook(fn func(ctx context.Context) error) BaseSkillOption {
	return func(s *BaseSkill) { s.onUnload = fn }
}

// ID 实现 Skill 接口
func (s *BaseSkill) ID() string {
	return s.id
}

// Name 实现 Skill 接口
func (s *BaseSkill) Name() string {
	return s.name
}

// Description 实现 Skill 接口
func (s *BaseSkill) Description() string {
	return s.description
}

// Category 实现 Skill 接口
func (s *BaseSkill) Category() SkillCategory {
	return s.category
}

// Tags 实现 Skill 接口
func (s *BaseSkill) Tags() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// 返回副本以防止外部修改
	tags := make([]string, len(s.tags))
	copy(tags, s.tags)
	return tags
}

// Load 实现 Skill 接口
func (s *BaseSkill) Load(ctx context.Context, config *LoadConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.loaded {
		return ErrSkillAlreadyLoaded
	}

	// 执行自定义加载逻辑
	if s.onLoad != nil {
		if err := s.onLoad(ctx, config); err != nil {
			return fmt.Errorf("%w: %v", ErrSkillLoadFailed, err)
		}
	}

	s.loaded = true
	s.metadata.UpdatedAt = time.Now()
	return nil
}

// Unload 实现 Skill 接口
func (s *BaseSkill) Unload(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.loaded {
		return ErrSkillNotLoaded
	}

	// 执行自定义卸载逻辑
	if s.onUnload != nil {
		if err := s.onUnload(ctx); err != nil {
			return fmt.Errorf("%w: %v", ErrSkillUnloadFailed, err)
		}
	}

	s.loaded = false
	s.metadata.UpdatedAt = time.Now()
	return nil
}

// IsLoaded 实现 Skill 接口
func (s *BaseSkill) IsLoaded() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.loaded
}

// GetTools 实现 Skill 接口
func (s *BaseSkill) GetTools() []tools.Tool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// 返回副本以防止外部修改
	tools := make([]tools.Tool, len(s.tools))
	copy(tools, s.tools)
	return tools
}

// GetSystemPrompt 实现 Skill 接口
func (s *BaseSkill) GetSystemPrompt() string {
	return s.systemPrompt
}

// GetExamples 实现 Skill 接口
func (s *BaseSkill) GetExamples() []SkillExample {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// 返回副本以防止外部修改
	examples := make([]SkillExample, len(s.examples))
	copy(examples, s.examples)
	return examples
}

// GetMetadata 实现 Skill 接口
func (s *BaseSkill) GetMetadata() *SkillMetadata {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// 返回副本以防止外部修改
	metadata := *s.metadata
	if s.metadata.Extra != nil {
		metadata.Extra = make(map[string]any, len(s.metadata.Extra))
		for k, v := range s.metadata.Extra {
			metadata.Extra[k] = v
		}
	}
	return &metadata
}

// Dependencies 实现 Skill 接口
func (s *BaseSkill) Dependencies() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// 返回副本以防止外部修改
	deps := make([]string, len(s.dependencies))
	copy(deps, s.dependencies)
	return deps
}

// AddTool 动态添加工具
func (s *BaseSkill) AddTool(tool tools.Tool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tools = append(s.tools, tool)
}

// RemoveTool 动态移除工具
func (s *BaseSkill) RemoveTool(toolName string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	filtered := make([]tools.Tool, 0, len(s.tools))
	for _, t := range s.tools {
		if t.GetName() != toolName {
			filtered = append(filtered, t)
		}
	}
	s.tools = filtered
}

// AddTag 添加标签
func (s *BaseSkill) AddTag(tag string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 检查是否已存在
	for _, t := range s.tags {
		if t == tag {
			return
		}
	}
	s.tags = append(s.tags, tag)
}

// RemoveTag 移除标签
func (s *BaseSkill) RemoveTag(tag string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	filtered := make([]string, 0, len(s.tags))
	for _, t := range s.tags {
		if t != tag {
			filtered = append(filtered, t)
		}
	}
	s.tags = filtered
}

// SetSystemPrompt 设置系统提示词
func (s *BaseSkill) SetSystemPrompt(prompt string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.systemPrompt = prompt
}

// AddExample 添加示例
func (s *BaseSkill) AddExample(example SkillExample) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.examples = append(s.examples, example)
}
