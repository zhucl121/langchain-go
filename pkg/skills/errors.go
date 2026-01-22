package skills

import "errors"

var (
	// ErrSkillNotFound Skill 未找到
	ErrSkillNotFound = errors.New("skill: not found")

	// ErrSkillAlreadyLoaded Skill 已加载
	ErrSkillAlreadyLoaded = errors.New("skill: already loaded")

	// ErrSkillNotLoaded Skill 未加载
	ErrSkillNotLoaded = errors.New("skill: not loaded")

	// ErrSkillAlreadyRegistered Skill 已注册
	ErrSkillAlreadyRegistered = errors.New("skill: already registered")

	// ErrCircularDependency 循环依赖
	ErrCircularDependency = errors.New("skill: circular dependency detected")

	// ErrDependencyNotMet 依赖未满足
	ErrDependencyNotMet = errors.New("skill: dependency not met")

	// ErrInvalidSkillConfig 无效的 Skill 配置
	ErrInvalidSkillConfig = errors.New("skill: invalid config")

	// ErrSkillLoadFailed Skill 加载失败
	ErrSkillLoadFailed = errors.New("skill: load failed")

	// ErrSkillUnloadFailed Skill 卸载失败
	ErrSkillUnloadFailed = errors.New("skill: unload failed")
)
