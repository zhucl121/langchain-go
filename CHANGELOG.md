# Changelog

所有重要的项目变更都会记录在这个文件中。

格式基于 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.0.0/)，
并且本项目遵循 [语义化版本](https://semver.org/lang/zh-CN/)。

## [Unreleased]

## [0.1.1] - 2026-01-19

### 🎉 Added - 15个重大新功能

#### 向量存储 (4个)
- **Chroma** - 开源轻量级向量数据库集成
- **Qdrant** - 高性能向量搜索引擎
- **Weaviate** - 企业级向量数据库，支持混合搜索
- **Redis Vector** - 基于 Redis 的高性能向量搜索

#### LLM 提供商 (3个)
- **Google Gemini** - 多模态大模型支持
- **AWS Bedrock** - 企业级托管 LLM 服务
- **Azure OpenAI** - 微软云 OpenAI 服务集成

#### 文档加载器 (3个)
- **GitHub Loader** - 代码仓库内容加载，支持文件过滤
- **Confluence Loader** - 企业知识库集成
- **PostgreSQL Loader** - 关系数据库内容加载

#### 高级 RAG 技术 (4个)
- **Multi-Query Generation** - 生成多个查询变体提高召回率
- **HyDE (Hypothetical Document Embeddings)** - 克服查询-文档语义鸿沟
- **Parent Document Retriever** - 索引小块返回父文档，平衡精度和上下文
- **Self-Query Retriever** - 自动提取结构化查询和过滤条件

#### LCEL 等效语法 (1个)
- **Chain 链式语法** - Go 版本的 LCEL 实现
  - Pipe 管道操作符
  - Parallel 并行执行
  - Route 条件路由
  - Fallback 失败回退
  - Retry 重试机制
  - Map/Filter 函数式操作

### 📊 统计数据
- 新增代码: ~10,900 行（含完整测试）
- 测试覆盖: 85%+

### 📝 Documentation
- 新增 `docs/COMPLETION_REPORT.md` - 完整项目完成报告
- 新增 `docs/guides/rag/advanced-retrievers.md` - 高级 RAG 使用指南
- 更新 `README.md` - 添加所有新功能说明

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

[Unreleased]: https://github.com/zhucl121/langchain-go/compare/v0.1.1...HEAD
[0.1.1]: https://github.com/zhucl121/langchain-go/releases/tag/v0.1.1
[0.1.0]: https://github.com/zhucl121/langchain-go/releases/tag/v0.1.0
