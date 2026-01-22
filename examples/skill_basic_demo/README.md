# Skill 基础使用示例

演示 LangChain-Go Skill 系统的基本用法。

## 功能演示

本示例展示：

1. **创建 Skill 管理器**
2. **注册内置 Skills** - 注册 4 个内置 Skill
3. **列出所有 Skills** - 查看已注册的 Skills
4. **加载 Skill** - 动态加载 Skill
5. **验证状态** - 检查 Skill 是否已加载
6. **获取详细信息** - 查看 Skill 的提示词、示例和元数据
7. **按分类查找** - 根据分类筛选 Skills
8. **按标签查找** - 根据标签筛选 Skills
9. **动态切换** - 卸载和加载不同的 Skill
10. **清理资源** - 卸载所有 Skills

## 运行示例

```bash
cd examples/skill_basic_demo
go run main.go
```

## 预期输出

```
=== LangChain-Go Skill 系统基础示例 ===

1. 创建 Skill 管理器
   ✓ Skill 管理器已创建

2. 注册内置 Skills
   ✓ 注册 Coding Skill (ID: coding)
   ✓ 注册 Data Analysis Skill (ID: data-analysis)
   ✓ 注册 Knowledge Query Skill (ID: knowledge-query)
   ✓ 注册 Research Skill (ID: research)

3. 列出所有已注册的 Skills
   总计: 4 个 Skills
   - 代码助手 (coding): 提供代码编写、调试和重构能力
     分类: coding, 标签: [coding programming debug refactor development]
   ...

=== 示例完成 ===
```

## 核心概念

### Skill Manager

`SkillManager` 负责管理所有 Skill 的生命周期：
- 注册/注销 Skill
- 加载/卸载 Skill
- 查询和筛选 Skill

### 内置 Skills

LangChain-Go 提供 4 个开箱即用的 Skill：

1. **Coding Skill** (`coding`) - 编程助手
2. **Data Analysis Skill** (`data-analysis`) - 数据分析
3. **Knowledge Query Skill** (`knowledge-query`) - 知识问答
4. **Research Skill** (`research`) - 研究调研

### Skill 状态

每个 Skill 有两种状态：
- **已注册未加载** - Skill 已注册但未初始化
- **已加载** - Skill 已加载并可以使用

## 下一步

- [Skill 组合示例](../skill_compose_demo/) - 学习如何组合多个 Skill
- [自定义 Skill 示例](../skill_custom_demo/) - 学习如何创建自定义 Skill
- [完整用户指南](../../docs/V0.5.1_USER_GUIDE.md) - 详细的使用说明

## 相关文档

- [Skill 实施计划](../../docs/V0.5.1_IMPLEMENTATION_PLAN.md)
- [API 文档](https://pkg.go.dev/github.com/zhucl121/langchain-go/pkg/skills)
