# .cursorrules 更新 - 文档国际化规范

## 更新内容

已将文档国际化的最佳实践精炼并添加到项目的 `.cursorrules` 文件中。

## 新增章节

### 6. 文档国际化规范 ⭐

在文档规范部分新增了完整的国际化指南，包括：

#### 1. 文件命名规范
```
README.md              - 中文主页
README_EN.md           - 英文主页
CONTRIBUTING.md        - 英文贡献指南（原版）
CONTRIBUTING_ZH.md     - 中文贡献指南
```

#### 2. 语言切换标识
统一格式的语言切换链接：
```markdown
🌍 **Language**: 中文 | [English](文档名_EN.md)
```

#### 3. 必需双语文档清单
- README.md + README_EN.md
- QUICK_START.md + QUICK_START_EN.md
- CONTRIBUTING.md + CONTRIBUTING_ZH.md
- TESTING.md + TESTING_EN.md
- SECURITY.md + SECURITY_EN.md

#### 4. 文档链接双语化
```markdown
- 📘 [快速开始](QUICK_START.md) | [Quick Start (EN)](QUICK_START_EN.md)
```

#### 5. 内容组织要求
- 结构一致性
- 技术术语处理
- 代码示例规范

#### 6. 维护规范
- 同步更新原则
- 质量保证
- 社区贡献

#### 7. 国际化检查清单
- 新增文档时的检查项
- 更新文档时的检查项
- 发布前的完整检查

## 更新位置

文件：`.cursorrules`
章节：`文档规范` → `6. 文档国际化规范`
行数：约 460-590 行

## 核心原则

1. **所有用户面向文档必须提供中英文双语版本**
2. **保持中英文内容对等和一致性**
3. **便于国际开发者社区访问和贡献**

## 实施效果

✅ 已按照此规范完成的文档：
- README.md + README_EN.md
- QUICK_START.md + QUICK_START_EN.md
- CONTRIBUTING_ZH.md（原版为英文）
- TESTING_EN.md（原版为中文）
- SECURITY_EN.md（原版为中文）

✅ 所有文档均添加了统一的语言切换标识
✅ 主 README.md 中的文档链接已更新为双语版本

## 后续开发指引

当需要创建或更新文档时：

1. **新增文档**：
   - 确定是否需要国际化（用户面向文档必须）
   - 创建对应语言版本
   - 添加语言切换标识
   - 更新相关引用

2. **更新文档**：
   - 同步更新所有语言版本
   - 检查术语翻译一致性
   - 验证代码示例可运行

3. **发布前**：
   - 使用国际化检查清单
   - 确保所有核心文档有双语版本
   - 验证所有链接有效

## 参考

完整实施总结：`docs/archive/development/DOCUMENTATION_I18N_COMPLETE.md`

---

**更新时间**: 2026-01-24
**更新人**: AI Assistant
**状态**: ✅ 已完成并生效
