package skills

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/zhucl121/langchain-go/core/tools"
)

// ProgressiveBaseSkill 支持渐进式加载的 BaseSkill 实现
type ProgressiveBaseSkill struct {
	// Level 1: 元数据（始终加载）
	id           string
	name         string
	description  string
	category     SkillCategory
	tags         []string
	metadata     *SkillMetadata
	dependencies []string

	// Level 2: 指令（按需加载）
	instructions       *SkillInstructions
	instructionsLoader SkillInstructionsLoader

	// Level 3: 资源（执行时加载）
	resources       *SkillResources
	resourcesLoader SkillResourcesLoader

	// 状态管理
	loaded           bool
	currentLoadLevel LoadLevel
	mu               sync.RWMutex

	// Hooks
	onLoad   func(ctx context.Context, config *LoadConfig) error
	onUnload func(ctx context.Context) error
}

// NewProgressiveBaseSkill 创建支持渐进式加载的 BaseSkill
func NewProgressiveBaseSkill(opts ...ProgressiveSkillOption) *ProgressiveBaseSkill {
	s := &ProgressiveBaseSkill{
		tags:               []string{},
		dependencies:       []string{},
		currentLoadLevel:   LoadLevelMetadata,
		instructions:       NewSkillInstructions(),
		resources:          NewSkillResources(),
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

// ProgressiveSkillOption 渐进式 Skill 选项
type ProgressiveSkillOption func(*ProgressiveBaseSkill)

// WithProgressiveID 设置 ID
func WithProgressiveID(id string) ProgressiveSkillOption {
	return func(s *ProgressiveBaseSkill) { s.id = id }
}

// WithProgressiveName 设置名称
func WithProgressiveName(name string) ProgressiveSkillOption {
	return func(s *ProgressiveBaseSkill) { s.name = name }
}

// WithProgressiveDescription 设置描述
func WithProgressiveDescription(desc string) ProgressiveSkillOption {
	return func(s *ProgressiveBaseSkill) { s.description = desc }
}

// WithProgressiveCategory 设置分类
func WithProgressiveCategory(cat SkillCategory) ProgressiveSkillOption {
	return func(s *ProgressiveBaseSkill) { s.category = cat }
}

// WithProgressiveTags 设置标签
func WithProgressiveTags(tags ...string) ProgressiveSkillOption {
	return func(s *ProgressiveBaseSkill) { s.tags = tags }
}

// WithProgressiveInstructions 设置指令
func WithProgressiveInstructions(instructions *SkillInstructions) ProgressiveSkillOption {
	return func(s *ProgressiveBaseSkill) { s.instructions = instructions }
}

// WithInstructionsLoader 设置指令加载器
func WithInstructionsLoader(loader SkillInstructionsLoader) ProgressiveSkillOption {
	return func(s *ProgressiveBaseSkill) { s.instructionsLoader = loader }
}

// WithResourcesLoader 设置资源加载器
func WithResourcesLoader(loader SkillResourcesLoader) ProgressiveSkillOption {
	return func(s *ProgressiveBaseSkill) { s.resourcesLoader = loader }
}

// WithProgressiveDependencies 设置依赖
func WithProgressiveDependencies(deps ...string) ProgressiveSkillOption {
	return func(s *ProgressiveBaseSkill) { s.dependencies = deps }
}

// WithProgressiveMetadata 设置元数据
func WithProgressiveMetadata(metadata *SkillMetadata) ProgressiveSkillOption {
	return func(s *ProgressiveBaseSkill) { s.metadata = metadata }
}

// 实现 Skill 接口（Level 1: 元数据）

func (s *ProgressiveBaseSkill) ID() string {
	return s.id
}

func (s *ProgressiveBaseSkill) Name() string {
	return s.name
}

func (s *ProgressiveBaseSkill) Description() string {
	return s.description
}

func (s *ProgressiveBaseSkill) Category() SkillCategory {
	return s.category
}

func (s *ProgressiveBaseSkill) Tags() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tags := make([]string, len(s.tags))
	copy(tags, s.tags)
	return tags
}

func (s *ProgressiveBaseSkill) Load(ctx context.Context, config *LoadConfig) error {
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

func (s *ProgressiveBaseSkill) Unload(ctx context.Context) error {
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

	// 清理 Level 2 和 Level 3 数据
	s.instructions = NewSkillInstructions()
	s.resources = NewSkillResources()
	s.currentLoadLevel = LoadLevelMetadata

	s.loaded = false
	s.metadata.UpdatedAt = time.Now()
	return nil
}

func (s *ProgressiveBaseSkill) IsLoaded() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.loaded
}

func (s *ProgressiveBaseSkill) GetTools() []tools.Tool {
	// 渐进式 Skill 的工具通过元工具访问
	return []tools.Tool{}
}

func (s *ProgressiveBaseSkill) GetSystemPrompt() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.instructions != nil {
		return s.instructions.SystemPrompt
	}
	return ""
}

func (s *ProgressiveBaseSkill) GetExamples() []SkillExample {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.instructions != nil {
		examples := make([]SkillExample, len(s.instructions.Examples))
		copy(examples, s.instructions.Examples)
		return examples
	}
	return []SkillExample{}
}

func (s *ProgressiveBaseSkill) GetMetadata() *SkillMetadata {
	s.mu.RLock()
	defer s.mu.RUnlock()

	metadata := *s.metadata
	if s.metadata.Extra != nil {
		metadata.Extra = make(map[string]any, len(s.metadata.Extra))
		for k, v := range s.metadata.Extra {
			metadata.Extra[k] = v
		}
	}
	return &metadata
}

func (s *ProgressiveBaseSkill) Dependencies() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	deps := make([]string, len(s.dependencies))
	copy(deps, s.dependencies)
	return deps
}

// 实现 ProgressiveSkill 接口

func (s *ProgressiveBaseSkill) LoadInstructions(ctx context.Context) (*SkillInstructions, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 如果已加载，直接返回
	if s.currentLoadLevel >= LoadLevelInstructions && s.instructions != nil {
		return s.instructions, nil
	}

	// 使用加载器加载
	if s.instructionsLoader != nil {
		instructions, err := s.instructionsLoader.LoadInstructions(ctx, s.id)
		if err != nil {
			return nil, fmt.Errorf("load instructions for skill %s: %w", s.id, err)
		}
		s.instructions = instructions
	}

	// 更新加载级别
	s.currentLoadLevel = LoadLevelInstructions
	s.metadata.UpdatedAt = time.Now()

	return s.instructions, nil
}

func (s *ProgressiveBaseSkill) LoadResources(ctx context.Context) (*SkillResources, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 如果已加载，直接返回
	if s.currentLoadLevel >= LoadLevelResources && s.resources != nil {
		return s.resources, nil
	}

	// 使用加载器加载
	if s.resourcesLoader != nil {
		resources, err := s.resourcesLoader.LoadResources(ctx, s.id)
		if err != nil {
			return nil, fmt.Errorf("load resources for skill %s: %w", s.id, err)
		}
		s.resources = resources
	}

	// 更新加载级别
	s.currentLoadLevel = LoadLevelResources
	s.metadata.UpdatedAt = time.Now()

	return s.resources, nil
}

func (s *ProgressiveBaseSkill) GetLoadLevel() LoadLevel {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.currentLoadLevel
}

func (s *ProgressiveBaseSkill) IsInstructionsLoaded() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.currentLoadLevel >= LoadLevelInstructions
}

func (s *ProgressiveBaseSkill) IsResourcesLoaded() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.currentLoadLevel >= LoadLevelResources
}

// GetInstructions 获取已加载的指令（不触发加载）
func (s *ProgressiveBaseSkill) GetInstructions() *SkillInstructions {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.instructions
}

// GetResources 获取已加载的资源（不触发加载）
func (s *ProgressiveBaseSkill) GetResources() *SkillResources {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.resources
}
