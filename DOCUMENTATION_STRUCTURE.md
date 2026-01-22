# 文档结构说明

LangChain-Go 项目文档组织结构。

## 📂 文档目录结构

```
langchain-go/
├── README.md                    # 项目主页（必读）⭐
├── CHANGELOG.md                 # 变更日志（Keep a Changelog 格式）⭐
├── CONTRIBUTING.md              # 贡献指南
├── LICENSE                      # MIT 许可证
├── QUICK_START.md              # 快速开始指南
├── TESTING.md                  # 测试指南
│
├── docs/                       # 文档目录
│   ├── README.md               # 文档索引
│   │
│   ├── releases/               # 发布文档 🆕
│   │   ├── README.md           # 发布历史索引
│   │   ├── RELEASE_NOTES_v0.6.0.md        # v0.6.0 完整说明
│   │   ├── GITHUB_RELEASE_v0.6.0.md       # v0.6.0 GitHub Release
│   │   ├── RELEASE_CHECKLIST_v0.6.0.md    # v0.6.0 检查清单
│   │   ├── V0.6.0_RELEASE_COMPLETE.md     # v0.6.0 发布完成 ⭐
│   │   ├── CI_FIX_SUMMARY_v0.6.0.md       # v0.6.0 CI 修复 ⭐
│   │   ├── ... (其他版本文档)
│   │   └── archive/                       # 旧版本归档 🆕
│   │       ├── README.md
│   │       ├── V0.4.1_READY_TO_PUBLISH.md
│   │       └── V0.5.0_发布说明.md
│   │
│   ├── guides/                 # 使用指南
│   │   ├── agents/            # Agent 指南
│   │   ├── rag/               # RAG 指南
│   │   ├── tools/             # 工具指南
│   │   └── multi-agent-guide.md
│   │
│   ├── reference/             # API 参考
│   │   └── enhancements.md
│   │
│   ├── V0.4.2_USER_GUIDE.md         # v0.4.2 用户指南
│   ├── V0.4.2_COMPLETION_REPORT.md  # v0.4.2 完成报告
│   ├── V0.4.2_RELEASE_SUMMARY.md    # v0.4.2 发布总结
│   ├── V0.4.2_PROGRESS.md           # v0.4.2 进度跟踪
│   ├── V0.4.2_IMPLEMENTATION_PLAN.md # v0.4.2 实现计划
│   │
│   └── ... (其他版本的技术文档)
│
└── examples/                   # 示例程序
    ├── agent_simple_demo.go
    ├── multi_agent_demo.go
    ├── learning_complete_demo/
    ├── learning_feedback_demo/
    └── ...
```

---

## 📖 文档分类

### 1. 根目录文档（面向用户）

#### README.md ⭐ 必读
- **用途**: 项目主页，第一印象
- **内容**: 
  - 项目简介
  - 核心特性
  - 快速开始（30 秒上手）
  - 安装说明
  - 主要功能展示
  - 文档链接
  - 社区信息

#### CHANGELOG.md ⭐ 必读
- **用途**: 所有版本的变更记录
- **格式**: [Keep a Changelog](https://keepachangelog.com/)
- **内容**:
  - Unreleased（开发中）
  - 每个版本的变更（Added, Changed, Fixed, Removed）
  - 发布日期
  - 版本链接

#### CONTRIBUTING.md
- **用途**: 贡献指南
- **内容**:
  - 如何报告 Bug
  - 如何提出新功能
  - 开发环境设置
  - 代码规范
  - PR 流程

#### QUICK_START.md
- **用途**: 5 分钟快速上手
- **内容**:
  - 安装
  - 第一个程序
  - 核心概念
  - 下一步

#### TESTING.md
- **用途**: 测试指南
- **内容**:
  - 测试环境设置
  - 运行测试
  - 编写测试
  - CI/CD

---

### 2. docs/ 目录（详细文档）

#### docs/README.md
- **用途**: 文档索引和导航
- **内容**: 所有文档的链接和说明

#### docs/releases/ 🆕 发布文档
- **用途**: 集中管理所有发布相关文档
- **内容**:
  - **RELEASE_NOTES_vX.X.X.md** - 完整发布说明
  - **GITHUB_RELEASE_vX.X.X.md** - GitHub Release 公告
  - **RELEASE_CHECKLIST_vX.X.X.md** - 发布检查清单
  - **RELEASE_GUIDE_vX.X.X.md** - 发布指南
  - **README.md** - 发布历史索引

#### docs/guides/ 使用指南
- **用途**: 功能使用教程
- **内容**:
  - `agents/` - Agent 系统指南
  - `rag/` - RAG 系统指南
  - `tools/` - 工具使用指南
  - `multi-agent-guide.md` - Multi-Agent 指南

#### docs/reference/ API 参考
- **用途**: API 文档和技术参考
- **内容**: 详细的 API 说明

#### docs/VX.X.X_*.md 版本技术文档
- **用途**: 特定版本的技术文档
- **命名规范**:
  - `VX.X.X_USER_GUIDE.md` - 用户指南
  - `VX.X.X_IMPLEMENTATION_PLAN.md` - 实现计划
  - `VX.X.X_COMPLETION_REPORT.md` - 完成报告
  - `VX.X.X_PROGRESS.md` - 进度跟踪
  - `VX.X.X_RELEASE_SUMMARY.md` - 发布总结

---

### 3. examples/ 示例程序

- **用途**: 可运行的示例代码
- **组织**: 按功能分类
  - Agent 相关
  - Multi-Agent 相关
  - Learning Retrieval 相关
  - 工具相关
- **每个示例包含**: 
  - `main.go` - 示例代码
  - `README.md` - 使用说明

---

## 🎯 文档使用场景

### 场景 1: 新用户上手

**阅读顺序**:
1. `README.md` - 了解项目
2. `QUICK_START.md` - 快速开始
3. `examples/` - 运行示例
4. `docs/guides/` - 深入学习

### 场景 2: 查看新功能

**阅读顺序**:
1. `CHANGELOG.md` - 查看最新版本
2. `docs/releases/RELEASE_NOTES_vX.X.X.md` - 详细功能说明
3. `docs/VX.X.X_USER_GUIDE.md` - 使用指南
4. `examples/` - 示例代码

### 场景 3: 贡献代码

**阅读顺序**:
1. `CONTRIBUTING.md` - 贡献指南
2. `TESTING.md` - 测试指南
3. `.cursorrules` - 代码规范
4. `docs/VX.X.X_IMPLEMENTATION_PLAN.md` - 实现计划

### 场景 4: 发布新版本

**使用文档**:
1. `docs/releases/RELEASE_CHECKLIST_vX.X.X.md` - 检查清单
2. `docs/releases/RELEASE_GUIDE_vX.X.X.md` - 发布指南
3. `CHANGELOG.md` - 更新变更日志
4. `docs/releases/RELEASE_NOTES_vX.X.X.md` - 编写发布说明

---

## 📝 文档编写规范

### 通用规范

1. **Markdown 格式** - 所有文档使用 Markdown
2. **中文为主** - 主要文档使用中文，代码注释可中英混合
3. **清晰的标题层级** - 使用 H1-H6
4. **代码示例** - 提供完整可运行的代码
5. **链接** - 使用相对路径链接其他文档

### README.md 规范

- 第一段：项目简介（30 字内）
- 徽章：版本、许可证、测试状态等
- 核心特性：3-5 个亮点
- 快速开始：30 秒上手代码
- 文档链接：清晰的导航

### CHANGELOG.md 规范

遵循 [Keep a Changelog](https://keepachangelog.com/):
- 按版本倒序排列
- 使用 Added, Changed, Deprecated, Removed, Fixed, Security
- 包含版本号、发布日期
- 底部包含版本对比链接

### 发布文档规范

**RELEASE_NOTES_vX.X.X.md**:
- 完整的功能说明
- 代码统计
- 使用示例
- 性能数据
- 升级指南

**GITHUB_RELEASE_vX.X.X.md**:
- 简洁版发布说明
- 适合复制到 GitHub Release
- 包含快速开始示例

---

## 🔄 文档维护

### 每次发布时

1. ✅ 更新 `CHANGELOG.md`
2. ✅ 创建 `docs/releases/RELEASE_NOTES_vX.X.X.md`
3. ✅ 创建 `docs/releases/GITHUB_RELEASE_vX.X.X.md`
4. ✅ 更新 `README.md`（如有新功能）
5. ✅ 创建版本用户指南 `docs/VX.X.X_USER_GUIDE.md`

### 定期维护

- ✅ 检查所有链接是否有效
- ✅ 更新过时的示例代码
- ✅ 补充用户反馈的常见问题
- ✅ 同步代码注释和文档

---

## 📊 文档统计

**总文档数**: 50+ 页

**分类统计**:
- 根目录文档: 6 个
- 发布文档: 20+ 个
- 使用指南: 15+ 个
- 技术文档: 10+ 个
- 示例 README: 17+ 个

**总字数**: 约 10 万字

---

## 🔗 相关资源

- [Keep a Changelog](https://keepachangelog.com/)
- [Semantic Versioning](https://semver.org/)
- [Markdown Guide](https://www.markdownguide.org/)
- [Write the Docs](https://www.writethedocs.org/)

---

**文档版本**: v1.0  
**最后更新**: 2026-01-22  
**维护者**: LangChain-Go Team
