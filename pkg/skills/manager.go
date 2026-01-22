package skills

import (
	"context"
	"fmt"
	"sync"
)

// SkillManager Skill 管理器接口
type SkillManager interface {
	// Register 注册 Skill
	Register(skill Skill) error

	// Unregister 注销 Skill
	Unregister(skillID string) error

	// Load 加载 Skill
	Load(ctx context.Context, skillID string, config *LoadConfig) error

	// Unload 卸载 Skill
	Unload(ctx context.Context, skillID string) error

	// Get 获取 Skill
	Get(skillID string) (Skill, error)

	// List 列出所有已注册的 Skill
	List() []Skill

	// ListLoaded 列出所有已加载的 Skill
	ListLoaded() []Skill

	// FindByCategory 按分类查找 Skill
	FindByCategory(category SkillCategory) []Skill

	// FindByTags 按标签查找 Skill
	FindByTags(tags []string) []Skill

	// LoadWithDependencies 加载 Skill 及其依赖
	LoadWithDependencies(ctx context.Context, skillID string, config *LoadConfig) error
}

// DefaultSkillManager 默认 Skill 管理器实现
type DefaultSkillManager struct {
	skills       map[string]Skill // skillID -> Skill
	loadedSkills map[string]bool  // skillID -> loaded
	mu           sync.RWMutex
}

// NewSkillManager 创建 Skill 管理器
func NewSkillManager() *DefaultSkillManager {
	return &DefaultSkillManager{
		skills:       make(map[string]Skill),
		loadedSkills: make(map[string]bool),
	}
}

// Register 注册 Skill
func (m *DefaultSkillManager) Register(skill Skill) error {
	if skill == nil {
		return fmt.Errorf("%w: skill is nil", ErrInvalidSkillConfig)
	}

	if skill.ID() == "" {
		return fmt.Errorf("%w: skill ID is empty", ErrInvalidSkillConfig)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.skills[skill.ID()]; exists {
		return fmt.Errorf("%w: skill %s", ErrSkillAlreadyRegistered, skill.ID())
	}

	m.skills[skill.ID()] = skill
	return nil
}

// Unregister 注销 Skill
func (m *DefaultSkillManager) Unregister(skillID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	skill, exists := m.skills[skillID]
	if !exists {
		return fmt.Errorf("%w: skill %s", ErrSkillNotFound, skillID)
	}

	// 如果已加载，先卸载
	if m.loadedSkills[skillID] {
		if err := skill.Unload(context.Background()); err != nil {
			return fmt.Errorf("unload skill before unregister: %w", err)
		}
		delete(m.loadedSkills, skillID)
	}

	delete(m.skills, skillID)
	return nil
}

// Load 加载 Skill
func (m *DefaultSkillManager) Load(ctx context.Context, skillID string, config *LoadConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	skill, exists := m.skills[skillID]
	if !exists {
		return fmt.Errorf("%w: skill %s", ErrSkillNotFound, skillID)
	}

	if m.loadedSkills[skillID] {
		return fmt.Errorf("%w: skill %s", ErrSkillAlreadyLoaded, skillID)
	}

	// 使用默认配置
	if config == nil {
		config = DefaultLoadConfig()
	}

	// 加载 Skill
	if err := skill.Load(ctx, config); err != nil {
		return fmt.Errorf("load skill %s: %w", skillID, err)
	}

	m.loadedSkills[skillID] = true
	return nil
}

// Unload 卸载 Skill
func (m *DefaultSkillManager) Unload(ctx context.Context, skillID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	skill, exists := m.skills[skillID]
	if !exists {
		return fmt.Errorf("%w: skill %s", ErrSkillNotFound, skillID)
	}

	if !m.loadedSkills[skillID] {
		return fmt.Errorf("%w: skill %s", ErrSkillNotLoaded, skillID)
	}

	// 卸载 Skill
	if err := skill.Unload(ctx); err != nil {
		return fmt.Errorf("unload skill %s: %w", skillID, err)
	}

	delete(m.loadedSkills, skillID)
	return nil
}

// Get 获取 Skill
func (m *DefaultSkillManager) Get(skillID string) (Skill, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	skill, exists := m.skills[skillID]
	if !exists {
		return nil, fmt.Errorf("%w: skill %s", ErrSkillNotFound, skillID)
	}

	return skill, nil
}

// List 列出所有已注册的 Skill
func (m *DefaultSkillManager) List() []Skill {
	m.mu.RLock()
	defer m.mu.RUnlock()

	skills := make([]Skill, 0, len(m.skills))
	for _, skill := range m.skills {
		skills = append(skills, skill)
	}
	return skills
}

// ListLoaded 列出所有已加载的 Skill
func (m *DefaultSkillManager) ListLoaded() []Skill {
	m.mu.RLock()
	defer m.mu.RUnlock()

	skills := make([]Skill, 0)
	for skillID, loaded := range m.loadedSkills {
		if loaded {
			if skill, exists := m.skills[skillID]; exists {
				skills = append(skills, skill)
			}
		}
	}
	return skills
}

// FindByCategory 按分类查找 Skill
func (m *DefaultSkillManager) FindByCategory(category SkillCategory) []Skill {
	m.mu.RLock()
	defer m.mu.RUnlock()

	skills := make([]Skill, 0)
	for _, skill := range m.skills {
		if skill.Category() == category {
			skills = append(skills, skill)
		}
	}
	return skills
}

// FindByTags 按标签查找 Skill
func (m *DefaultSkillManager) FindByTags(tags []string) []Skill {
	if len(tags) == 0 {
		return []Skill{}
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	tagSet := make(map[string]bool)
	for _, tag := range tags {
		tagSet[tag] = true
	}

	skills := make([]Skill, 0)
	for _, skill := range m.skills {
		skillTags := skill.Tags()
		for _, tag := range skillTags {
			if tagSet[tag] {
				skills = append(skills, skill)
				break
			}
		}
	}
	return skills
}

// LoadWithDependencies 加载 Skill 及其依赖
func (m *DefaultSkillManager) LoadWithDependencies(ctx context.Context, skillID string, config *LoadConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	skill, exists := m.skills[skillID]
	if !exists {
		return fmt.Errorf("%w: skill %s", ErrSkillNotFound, skillID)
	}

	// 检查循环依赖
	if err := m.checkCircularDependency(skillID, make(map[string]bool)); err != nil {
		return err
	}

	// 使用默认配置
	if config == nil {
		config = DefaultLoadConfig()
	}

	// 递归加载依赖
	deps := skill.Dependencies()
	for _, depID := range deps {
		if !m.loadedSkills[depID] {
			if err := m.loadDependency(ctx, depID, config); err != nil {
				return fmt.Errorf("load dependency %s: %w", depID, err)
			}
		}
	}

	// 加载 Skill
	if m.loadedSkills[skillID] {
		return fmt.Errorf("%w: skill %s", ErrSkillAlreadyLoaded, skillID)
	}

	if err := skill.Load(ctx, config); err != nil {
		return fmt.Errorf("load skill %s: %w", skillID, err)
	}

	m.loadedSkills[skillID] = true
	return nil
}

// loadDependency 加载依赖（内部方法，不加锁）
func (m *DefaultSkillManager) loadDependency(ctx context.Context, skillID string, config *LoadConfig) error {
	skill, exists := m.skills[skillID]
	if !exists {
		return fmt.Errorf("%w: %s", ErrDependencyNotMet, skillID)
	}

	// 递归加载依赖的依赖
	deps := skill.Dependencies()
	for _, depID := range deps {
		if !m.loadedSkills[depID] {
			if err := m.loadDependency(ctx, depID, config); err != nil {
				return err
			}
		}
	}

	// 加载 Skill
	if err := skill.Load(ctx, config); err != nil {
		return fmt.Errorf("load skill %s: %w", skillID, err)
	}

	m.loadedSkills[skillID] = true
	return nil
}

// checkCircularDependency 检查循环依赖
func (m *DefaultSkillManager) checkCircularDependency(skillID string, visited map[string]bool) error {
	if visited[skillID] {
		return fmt.Errorf("%w: skill %s", ErrCircularDependency, skillID)
	}

	visited[skillID] = true

	skill, exists := m.skills[skillID]
	if !exists {
		// Skill 不存在，但不算循环依赖
		delete(visited, skillID)
		return nil
	}

	deps := skill.Dependencies()
	for _, depID := range deps {
		if err := m.checkCircularDependency(depID, visited); err != nil {
			return err
		}
	}

	delete(visited, skillID)
	return nil
}

// IsLoaded 检查 Skill 是否已加载
func (m *DefaultSkillManager) IsLoaded(skillID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.loadedSkills[skillID]
}

// Count 返回已注册的 Skill 数量
func (m *DefaultSkillManager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.skills)
}

// LoadedCount 返回已加载的 Skill 数量
func (m *DefaultSkillManager) LoadedCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	count := 0
	for _, loaded := range m.loadedSkills {
		if loaded {
			count++
		}
	}
	return count
}
