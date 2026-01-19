# Changelog

所有重要的项目变更都会记录在这个文件中。

格式基于 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.0.0/)，
并且本项目遵循 [语义化版本](https://semver.org/lang/zh-CN/)。

## [Unreleased]

## [1.10.0] - 2026-01-19

### 🎉 Added - 15个重大新功能！

#### 向量存储 (4个)
- **Chroma 向量存储** - 开源轻量级向量数据库集成 (~610行)
- **Qdrant 向量存储** - 高性能向量搜索引擎 (~650行)
- **Weaviate 向量存储** - 企业级向量数据库，支持混合搜索 (~783行)
- **Redis Vector 向量存储** - 基于 Redis 的高性能向量搜索 (~580行)

#### LLM 提供商 (3个)
- **Google Gemini** - 多模态大模型支持 (~520行)
- **AWS Bedrock** - 企业级托管 LLM 服务 (~480行)
- **Azure OpenAI** - 微软云 OpenAI 服务集成 (~500行)

#### 文档加载器 (3个)
- **GitHub 加载器** - 代码仓库内容加载，支持文件过滤 (~600行)
- **Confluence 加载器** - 企业知识库集成 (~550行)
- **PostgreSQL 加载器** - 关系数据库内容加载 (~500行)

#### 高级 RAG 技术 (4个)
- **Multi-Query Generation** - 生成多个查询变体提高召回率 (~700行含测试)
- **HyDE (假设文档嵌入)** - 克服查询-文档语义鸿沟 (~600行含测试)
- **Parent Document Retriever** - 索引小块返回父文档，平衡精度和上下文 (~800行含测试)
- **Self-Query Retriever** - 自动提取结构化查询和过滤条件 (~600行含测试)

#### LCEL 等效语法 (1个)
- **Chain 链式语法** - Go 版本的 LCEL 实现 (~900行含测试)
  - Pipe 管道操作符
  - Parallel 并行执行
  - Route 条件路由
  - Fallback 失败回退
  - Retry 重试机制
  - Map/Filter 函数式操作

### 📊 统计数据
- 新增代码: ~10,900 行（含完整测试）
- 测试覆盖: 85%+
- Git 提交: 10 次
- 开发时长: 1 天

### 🏆 项目里程碑
- ✅ P0 任务 100% 完成 (10/10)
- ✅ P1 任务 100% 完成 (5/5)
- ✅ 总体完成度 100% (15/15)
- ✅ 生产就绪状态达成

### 📝 Documentation
- 新增 `COMPLETION_REPORT.md` - 完整项目完成报告
- 更新 `OPTIMIZATION_PROGRESS.md` - 100% 完成状态
- 更新 `archive/PENDING_FEATURES.md` - v1.9.0 和 v1.10.0 版本记录
- 更新 `README.md` - 添加所有新功能说明

## [1.9.0] - 2026-01-19 (早期版本)

### 🎉 Added
- 完整的测试环境配置
- Docker Compose 支持（Redis + Milvus）
- GitHub 标准配置文件

### 🔧 Changed
- 更新仓库地址为 `github.com/zhucl121/langchain-go`

### 📝 Documentation
- 添加测试指南
- 添加快速开始文档
- 完善 README

## [0.1.0] - TBD

### 🎉 Added
- 7种 Agent 类型实现
- Multi-Agent 协作系统
- 38个内置工具
- 完整的 LangGraph 实现
- Redis 缓存支持
- Milvus 2.6.1 向量存储
- RAG 实现
- 11个示例程序
- 50+文档页面

### ✅ Tests
- 35个包的单元测试
- 60%+ 测试覆盖率
- 集成测试环境

---

## 版本规范

- **Major**: 不兼容的 API 变更
- **Minor**: 向后兼容的功能新增
- **Patch**: 向后兼容的问题修正

[Unreleased]: https://github.com/zhucl121/langchain-go/compare/v1.10.0...HEAD
[1.10.0]: https://github.com/zhucl121/langchain-go/releases/tag/v1.10.0
[1.9.0]: https://github.com/zhucl121/langchain-go/releases/tag/v1.9.0
[0.1.0]: https://github.com/zhucl121/langchain-go/releases/tag/v0.1.0
