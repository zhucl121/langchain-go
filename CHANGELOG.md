# Changelog

所有重要的项目变更都会记录在这个文件中。

格式基于 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.0.0/)，
并且本项目遵循 [语义化版本](https://semver.org/lang/zh-CN/)。

## [Unreleased]

## [0.4.1] - 2026-01-21

### 🎉 Added - GraphRAG (图增强检索生成)

#### 图数据库抽象 (1个)
- **统一图数据库接口** - `retrieval/graphdb`
  - 统一的节点和边操作 API
  - 图遍历（BFS/DFS）和最短路径
  - 批量操作支持
  - 支持多种图数据库实现

#### 图数据库实现 (3个)
- **Neo4j** - 最成熟的图数据库
  - 完整的 CRUD 操作
  - Cypher 查询构建器
  - 事务支持，连接池管理
  - 性能：AddNode ~20ms, Traverse ~50ms
- **NebulaGraph** - 高性能分布式图数据库
  - nGQL 查询构建器
  - 完整结果转换器
  - 批量操作优化
  - 性能：AddNode ~50ms, Traverse ~260ms
- **MockGraphDB** - 内存图数据库
  - 零配置启动
  - 完整接口实现
  - 适合单元测试
  - 性能：AddNode ~0.1ms, Traverse ~2ms

#### 知识图谱 (2个)
- **KG Builder** - 知识图谱构建器
  - 基于 LLM 的实体提取
  - 关系抽取
  - 自动向量化
  - 批量构建和增量更新
- **GraphRAG Retriever** - 图增强检索器
  - 3 种搜索模式（Hybrid/Vector/Graph）
  - 4 种融合策略（Weighted/RRF/Max/Min）
  - 3 种重排序策略（Score/Diversity/MMR）
  - 混合检索性能 ~150ms

### 📊 统计数据
- 新增代码: ~5,670 行
- 测试代码: ~2,480 行
- 文档: ~6,200 行
- 总计: ~14,350 行
- 测试覆盖: 85%+

### 📝 Documentation
- 新增 `RELEASE_NOTES_v0.4.1.md` - 完整发布说明
- 新增 `docs/V0.4.1_USER_GUIDE.md` - 用户指南
- 新增 `retrieval/graphdb/neo4j/README.md` - Neo4j 使用指南
- 新增 `retrieval/graphdb/nebula/README.md` - NebulaGraph 使用指南
- 新增 `docs/V0.4.1_PERFORMANCE_COMPARISON.md` - 性能对比报告
- 新增 `docs/NEBULA_OPTIMIZATION_REPORT.md` - NebulaGraph 优化报告
- 新增 4 个示例程序（graphdb_demo, kg_builder_demo, graphrag_demo, graphrag_complete_demo）

### 🐛 Bug Fixes
- 修复 NebulaGraph GetNode/GetEdge 数据不完整问题
- 修复 Traverse 查询不返回完整对象问题
- 修复 ShortestPath 缺少节点属性问题

### ⚡ Performance
- Neo4j 连接池优化
- NebulaGraph 批量操作优化
- GraphRAG 检索缓存优化

### 🔧 Infrastructure
- 添加 `docker-compose.graphdb.yml` - 图数据库 Docker 配置
- 支持 Neo4j 5.15
- 支持 NebulaGraph 3.6.0

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

[Unreleased]: https://github.com/zhucl121/langchain-go/compare/v0.4.1...HEAD
[0.4.1]: https://github.com/zhucl121/langchain-go/compare/v0.1.1...v0.4.1
[0.1.1]: https://github.com/zhucl121/langchain-go/releases/tag/v0.1.1
[0.1.0]: https://github.com/zhucl121/langchain-go/releases/tag/v0.1.0
