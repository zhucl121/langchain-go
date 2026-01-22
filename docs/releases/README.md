# 发布文档

本目录包含所有版本的发布相关文档。

## 📂 目录结构

```
releases/
├── README.md                          # 本文件
├── RELEASE_NOTES_vX.X.X.md           # 完整的发布说明
├── GITHUB_RELEASE_vX.X.X.md          # GitHub Release 公告
├── RELEASE_CHECKLIST_vX.X.X.md       # 发布检查清单
├── RELEASE_GUIDE_vX.X.X.md           # 发布指南（部分版本）
├── CI_FIX_SUMMARY_vX.X.X.md          # CI 修复总结（v0.6.0+）
├── VX.X.X_RELEASE_COMPLETE.md        # 发布完成报告（v0.6.0+）
└── archive/                           # 旧版本文档归档
    ├── V0.4.1_READY_TO_PUBLISH.md    # v0.4.1 发布准备
    └── V0.5.0_发布说明.md            # v0.5.0 发布说明
```

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

3. **RELEASE_CHECKLIST_vX.X.X.md** - 发布检查清单
   - 详细的发布步骤
   - 验证项目
   - 推广计划

4. **VX.X.X_RELEASE_COMPLETE.md** - 发布完成报告（v0.6.0+）
   - 代码实现总结
   - 测试结果
   - Git 操作记录
   - 质量评估

5. **CI_FIX_SUMMARY_vX.X.X.md** - CI 修复总结（必要时）
   - CI 问题分析
   - 修复内容
   - 验证结果

## 📦 版本历史

### v0.6.0 - 2026-01-22 🔥 最新

**企业级安全完整版**

核心功能：
- ✅ RBAC 权限控制系统
- ✅ 多租户隔离
- ✅ 审计日志系统
- ✅ 数据安全（加密和脱敏）
- ✅ API 鉴权（JWT 和 API Key）

代码统计：5,880 行新增（5 个企业级模块）

文档：
- [发布说明](RELEASE_NOTES_v0.6.0.md)
- [GitHub Release](GITHUB_RELEASE_v0.6.0.md)
- [发布检查清单](RELEASE_CHECKLIST_v0.6.0.md)
- [发布完成报告](V0.6.0_RELEASE_COMPLETE.md) ⭐ 新增
- [CI 修复总结](CI_FIX_SUMMARY_v0.6.0.md) ⭐ 新增
- [进度文档](../V0.6.0_PROGRESS.md)

---

### v0.5.1 - 2026-01-21

**Agent Skills + Token 优化**

核心功能：
- ✅ 元工具模式（Meta-Tool Pattern）
- ✅ 三级渐进式加载
- ✅ Token 优化（70-79% 节省）

代码统计：3,200 行新增

文档：
- [发布说明](RELEASE_NOTES_v0.5.1.md)
- [GitHub Release](GITHUB_RELEASE_v0.5.1.md)
- [用户指南](../V0.5.1_USER_GUIDE.md)

---

### v0.5.0 - 2026-01-21

**分布式部署**

核心功能：
- ✅ 集群管理
- ✅ 负载均衡
- ✅ 分布式缓存
- ✅ 故障转移

代码统计：5,420 行新增

文档：
- [发布说明](RELEASE_NOTES_v0.5.0.md)
- [GitHub Release](GITHUB_RELEASE_v0.5.0.md)

---

### v0.4.2 - 2026-01-21

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

**总代码量**: 58,000+ 行（含企业级功能）

**版本数**: 9 个主要版本

**功能模块**:
- ✅ Agent 系统（7 种类型）
- ✅ Multi-Agent 协作
- ✅ RAG（基础 + 高级 + GraphRAG + Learning）
- ✅ 向量存储（5 个）
- ✅ 图数据库（3 个）
- ✅ LLM 提供商（6 个）
- ✅ 工具生态（38+ 个）
- ✅ 分布式部署
- ✅ 企业级安全（RBAC + 多租户 + 审计 + 加密）

## 🗂️ 归档文档

旧版本的发布相关文档已移至 [`archive/`](archive/) 目录：
- [V0.4.1_READY_TO_PUBLISH.md](archive/V0.4.1_READY_TO_PUBLISH.md) - v0.4.1 发布准备
- [V0.5.0_发布说明.md](archive/V0.5.0_发布说明.md) - v0.5.0 发布说明

---

## 🔗 相关资源

- [CHANGELOG.md](../../CHANGELOG.md) - 所有版本的变更日志
- [README.md](../../README.md) - 项目主页
- [CONTRIBUTING.md](../../CONTRIBUTING.md) - 贡献指南
- [文档结构说明](../../DOCUMENTATION_STRUCTURE.md) - 完整文档结构

---

**最后更新**: 2026-01-22  
**最新版本**: v0.6.0 - 企业级安全完整版
