# 发布文档

本目录包含所有版本的发布相关文档。

## 📋 文档说明

### 文档类型

每个版本包含以下文档：

1. **RELEASE_NOTES_vX.X.X.md** - 完整的发布说明
   - 新功能详解
   - 代码统计
   - 使用示例
   - 性能数据

2. **GITHUB_RELEASE_vX.X.X.md** - GitHub Release 公告
   - 用于创建 GitHub Release
   - 简洁版发布说明
   - 适合社交媒体传播

3. **RELEASE_GUIDE_vX.X.X.md**（部分版本）- 发布指南
   - 发布步骤
   - 注意事项
   - 检查清单

4. **RELEASE_CHECKLIST_vX.X.X.md**（部分版本）- 发布检查清单
   - 详细的发布步骤
   - 验证项目
   - 推广计划

## 📦 版本历史

### v0.4.2 - 2026-01-21 🔥 最新

**Learning Retrieval（学习型检索）**

核心功能：
- ✅ 用户反馈收集
- ✅ 检索质量评估
- ✅ 智能参数优化
- ✅ A/B 测试框架

代码统计：11,056 行新增

文档：
- [发布说明](RELEASE_NOTES_v0.4.2.md)
- [GitHub Release](GITHUB_RELEASE_v0.4.2.md)
- [发布检查清单](RELEASE_CHECKLIST_v0.4.2.md)
- [用户指南](../V0.4.2_USER_GUIDE.md)

---

### v0.4.1 - 2026-01-21

**GraphRAG（图增强检索生成）**

核心功能：
- ✅ 3 个图数据库实现（Neo4j, NebulaGraph, Mock）
- ✅ 知识图谱构建器
- ✅ GraphRAG Retriever

代码统计：14,350 行新增

文档：
- [发布说明](RELEASE_NOTES_v0.4.1.md)

---

### v0.4.0 - 2026-01-20

**Hybrid Search & Advanced RAG**

核心功能：
- ✅ Milvus Hybrid Search
- ✅ 高级检索技术（Parent Document, Multi-Query, HyDE）
- ✅ 文档处理工具

代码统计：3,500 行新增

文档：
- [发布说明](RELEASE_NOTES_v0.4.0.md)

---

### v0.3.0 - 2026-01-19

**Multi-Agent System**

核心功能：
- ✅ 多 Agent 协作系统
- ✅ 3 种协调策略
- ✅ 6 个专用 Agent

代码统计：4,200 行新增

文档：
- [发布说明](RELEASE_NOTES_v0.3.0.md)
- [GitHub Release](GITHUB_RELEASE_v0.3.0.md)

---

### v0.1.1 - 2026-01-19

**向量存储、LLM 提供商、文档加载器扩展**

核心功能：
- ✅ 4 个向量存储（Chroma, Qdrant, Weaviate, Redis）
- ✅ 3 个 LLM 提供商（Gemini, Bedrock, Azure）
- ✅ 3 个文档加载器（GitHub, Confluence, PostgreSQL）

代码统计：10,900 行新增

---

### v0.1.0 - 初始版本

**基础功能**

核心功能：
- ✅ 7 种 Agent 类型
- ✅ 38 个内置工具
- ✅ LangGraph 实现
- ✅ RAG 基础功能

文档：
- [发布说明](RELEASE_NOTES_v0.1.0.md)

---

## 📊 累计统计

**总代码量**: 43,000+ 行

**版本数**: 6 个主要版本

**功能模块**:
- Agent 系统
- Multi-Agent 协作
- RAG（基础 + 高级 + GraphRAG + Learning）
- 向量存储（5 个）
- 图数据库（3 个）
- LLM 提供商（6 个）
- 工具生态（38+ 个）

---

## 🔗 相关资源

- [CHANGELOG.md](../../CHANGELOG.md) - 所有版本的变更日志
- [README.md](../../README.md) - 项目主页
- [CONTRIBUTING.md](../../CONTRIBUTING.md) - 贡献指南

---

**最后更新**: 2026-01-22
