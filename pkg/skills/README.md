# Agent Skill 系统

Go 生态首个完整的 Agent Skill 系统，提供可组合、可扩展、可复用的智能体能力。

---

## 🌟 特性

- ✅ **统一接口** - 标准化的 Skill 定义
- ✅ **渐进式披露** - 按需加载专业能力
- ✅ **依赖管理** - 自动依赖解析和加载
- ✅ **动态工具注册** - 运行时注册/卸载工具
- ✅ **并发安全** - 完整的并发安全设计
- ✅ **零开销** - 未使用时无性能影响

---

## 🚀 快速开始

### 安装

```bash
go get github.com/zhucl121/langchain-go/pkg/skills
```

### 基础使用

```go
package main

import (
    "context"
    "github.com/zhucl121/langchain-go/pkg/skills"
    "github.com/zhucl121/langchain-go/pkg/skills/builtin"
)

func main() {
    ctx := context.Background()
    
    // 创建 Skill 管理器
    manager := skills.NewSkillManager()
    
    // 注册 Skill
    codingSkill := builtin.NewCodingSkill()
    manager.Register(codingSkill)
    
    // 加载 Skill
    config := &skills.LoadConfig{
        AutoLoadDependencies: true,
    }
    manager.Load(ctx, "coding", config)
    
    // 使用 Skill
    tools := codingSkill.GetTools()
    prompt := codingSkill.GetSystemPrompt()
}
```

---

## 📦 包结构

```
pkg/skills/
├── skill.go           # Skill 接口定义
├── base_skill.go      # BaseSkill 基础实现
├── manager.go         # SkillManager 管理器
├── errors.go          # 错误定义
├── doc.go             # 包文档
├── builtin/           # 内置 Skills
│   ├── coding_skill.go
│   ├── data_analysis_skill.go
│   ├── knowledge_skill.go
│   └── research_skill.go
└── README.md          # 本文件
```

---

## 🎯 核心概念

### Skill 接口

```go
type Skill interface {
    // 标识和元数据
    ID() string
    Name() string
    Description() string
    Category() SkillCategory
    Tags() []string
    
    // 生命周期
    Load(ctx context.Context, config *LoadConfig) error
    Unload(ctx context.Context) error
    IsLoaded() bool
    
    // 能力提供
    GetTools() []tools.Tool
    GetSystemPrompt() string
    GetExamples() []SkillExample
    GetMetadata() *SkillMetadata
    
    // 依赖管理
    Dependencies() []string
}
```

### Skill 分类

| 分类 | 说明 |
|------|------|
| `CategoryCoding` | 编程相关 |
| `CategoryDataAnalysis` | 数据分析 |
| `CategoryKnowledge` | 知识问答 |
| `CategoryCreative` | 创意写作 |
| `CategoryResearch` | 研究调研 |
| `CategoryAutomation` | 自动化 |
| `CategoryCommunication` | 沟通 |
| `CategoryGeneral` | 通用 |

---

## 📚 内置 Skills

### 1. Coding Skill

```go
skill := builtin.NewCodingSkill()
```

提供代码编写、调试、重构能力。

### 2. Data Analysis Skill

```go
skill := builtin.NewDataAnalysisSkill()
```

提供数据探索、统计分析、可视化建议。

### 3. Knowledge Query Skill

```go
skill := builtin.NewKnowledgeQuerySkill()
```

提供准确、全面的知识问答。

### 4. Research Skill

```go
skill := builtin.NewResearchSkill()
```

提供深度调研和分析能力。

---

## 🔧 自定义 Skill

### 使用 BaseSkill

```go
import "github.com/zhucl121/langchain-go/pkg/skills"

func NewMySkill() skills.Skill {
    return skills.NewBaseSkill(
        skills.WithID("my-skill"),
        skills.WithName("我的技能"),
        skills.WithCategory(skills.CategoryGeneral),
        skills.WithSystemPrompt("你是一个..."),
        skills.WithTools(tool1, tool2),
    )
}
```

### 实现 Skill 接口

```go
type MySkill struct {
    id     string
    name   string
    loaded bool
}

func (s *MySkill) ID() string { return s.id }
func (s *MySkill) Name() string { return s.name }
// ... 实现其他方法
```

---

## 🎓 示例

### 基础使用

参见 `examples/skill_basic_demo/`

### 组合多个 Skill

参见 `examples/skill_compose_demo/`

### 自定义 Skill

参见 `examples/skill_custom_demo/`

---

## 📖 文档

- [用户指南](../../docs/V0.5.1_USER_GUIDE.md)
- [实施计划](../../docs/V0.5.1_IMPLEMENTATION_PLAN.md)
- [API 文档](https://pkg.go.dev/github.com/zhucl121/langchain-go/pkg/skills)

---

## 🤝 贡献

欢迎贡献新的内置 Skill 或改进现有实现！

---

## 📄 许可证

MIT License

---

**版本**: v0.5.1  
**维护者**: LangChain-Go Team
