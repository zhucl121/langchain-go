# Changelog

所有重要的项目变更都会记录在这个文件中。

格式基于 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.0.0/)，
并且本项目遵循 [语义化版本](https://semver.org/lang/zh-CN/)。

## [Unreleased]

### 计划中
- v0.6.0: 企业级增强 - RBAC、审计与安全

## [0.5.0] - 2026-01-22

### 🎉 Added - 分布式部署：集群支持与负载均衡

本版本完整实现了分布式集群管理，包括节点管理、负载均衡、分布式缓存和故障转移，将 LangChain-Go 升级为生产级分布式 AI 框架。

#### 节点管理与服务发现
- **节点管理** (`pkg/cluster/node/`) - 588 行
  - 节点注册/注销/更新
  - 5 种节点状态（Online, Offline, Busy, Draining, Maintenance）
  - 4 种节点角色（Master, Worker, Router, Monitor）
  - 容量和负载监控
  - 节点过滤和查询

- **服务发现** (`pkg/cluster/discovery/`) - 632 行
  - Consul 完整集成
  - 自动心跳（TTL check）
  - 实时节点监听
  - 标签过滤

- **健康检查** (`pkg/cluster/health/`) - 693 行
  - HTTP 健康检查
  - TCP 健康检查
  - Composite 组合检查器
  - Periodic 周期性检查器
  - 3 种聚合策略（All, Any, Majority）

#### 负载均衡
- **5 种策略** (`pkg/cluster/balancer/`) - 1,113 行
  - Round Robin（轮询） - 25 ns/op (39.5M ops/s)
  - Least Connection（最少连接） - 50 ns/op
  - Weighted（加权） - 支持自动权重
  - Consistent Hash（一致性哈希） - 150 虚拟节点
  - Adaptive（自适应） - 实时性能评分

#### 分布式缓存
- **3 种缓存** (`pkg/cluster/cache/`) - 1,066 行
  - Memory Cache - 10.5M ops/s
  - Redis Cache - 分布式共享
  - Layered Cache - 本地 + 远程
  - 4 种驱逐策略（LRU, LFU, FIFO, TTL）
  - 写穿/写回模式

#### 故障转移与高可用
- **故障转移** (`pkg/cluster/failover/`) - 925 行
  - Circuit Breaker（熔断器） - 3 状态
  - Failover Manager（故障转移管理器）
  - 自动健康监控
  - 事件监听与告警

### 📊 统计数据
- **新增代码**: 5,017 行
- **测试代码**: 2,427 行
- **单元测试**: 84 个（100% 通过）
- **基准测试**: 12 个
- **测试覆盖率**: 85%+
- **示例程序**: 4 个（cluster_demo, balancer_demo, cache_demo, failover_demo）

### 📚 Documentation
- V0.5.0 用户指南 (500+ 行)
- V0.5.0 完成报告 (300+ 行)
- V0.5.0 发布说明 (详细)
- 示例 README (950+ 行)

## [0.4.2] - 2026-01-21

### 🎉 Added - Learning Retrieval (学习型检索)

完整的学习型检索系统，能够从用户反馈中自动学习并持续优化检索质量。

#### 用户反馈收集 (`retrieval/learning/feedback`)
- **显式反馈收集** - 点赞、评分、评论
- **隐式反馈收集** - 点击、阅读、复制、下载、忽略、跳过等 6 种用户行为
- **实时统计聚合** - 平均评分、CTR、阅读率等指标
- **双存储后端** - 内存存储（开发/测试）+ PostgreSQL（生产环境）
- **并发安全设计** - sync.RWMutex 保护共享状态

#### 检索质量评估 (`retrieval/learning/evaluation`)
- **相关性指标** - NDCG（排序质量金标准）、MRR（首个相关文档）、Precision/Recall/F1
- **用户满意度指标** - 平均评分、点击率（CTR）、阅读率
- **策略对比分析** - 对比两个检索策略的性能差异
- **统计显著性检验** - 置信区间计算、统计检验
- **可配置相关性模型** - 支持自定义相关性判断逻辑

#### 智能参数优化 (`retrieval/learning/optimization`)
- **贝叶斯优化算法** - 智能搜索最优参数配置
- **3 种参数类型** - Int（整数）、Float（浮点）、Choice（离散选择）
- **自动调优守护进程** - 持续监控和优化参数
- **探索-利用平衡** - 自动平衡探索新参数和利用已知好参数
- **参数验证和建议** - 验证参数合法性，建议下一步尝试的参数

#### A/B 测试框架 (`retrieval/learning/abtest`)
- **完整实验管理** - 创建、开始、暂停、结束实验
- **一致性哈希分流** - 确保同一用户始终看到相同变体
- **灵活流量控制** - 支持 0-100% 流量参与实验
- **多变体支持** - 支持 2+ 个变体对比
- **统计分析** - t-test 检验、95% 置信区间、p-value 显著性
- **实验状态管理** - Draft、Running、Paused、Ended

### 📊 统计数据
- **新增代码**: 11,056 行
  - 核心代码: 4,870 行（4 个模块）
  - 测试代码: 2,200 行（26 个测试）
  - 示例代码: 2,200 行（6 个示例）
  - 文档: 5,700 行
- **测试覆盖**: 69.1% 平均（abtest: 79.5%, evaluation: 78.3%, optimization: 75.5%, feedback: 42.9%）
- **测试通过率**: 100%（26/26）

### 📝 Documentation
- 新增 `RELEASE_NOTES_v0.4.2.md` - 完整发布说明（700+ 行）
- 新增 `docs/V0.4.2_USER_GUIDE.md` - 用户指南（500+ 行）
- 新增 `docs/V0.4.2_COMPLETION_REPORT.md` - 开发完成报告
- 新增 `docs/V0.4.2_RELEASE_SUMMARY.md` - 发布总结
- 新增 `docs/V0.4.2_PROGRESS.md` - 进度跟踪
- 新增 6 个完整示例程序
  - `learning_complete_demo` - 完整工作流（推荐）⭐
  - `learning_feedback_demo` - 反馈收集
  - `learning_evaluation_demo` - 质量评估
  - `learning_optimization_demo` - 参数优化
  - `learning_abtest_demo` - A/B 测试
  - `learning_postgres_demo` - PostgreSQL 存储

### ⚡ Performance
- 反馈收集: 0.1ms（内存）/ 10-20ms（PostgreSQL）
- 质量评估: ~1ms
- 参数优化: 5-10ms（50 次迭代）
- A/B 分析: ~2ms

### 💪 实测效果
- 文档检索优化：综合得分提升 16.5%（0.418 → 0.487）
- A/B 测试验证：实验组提升 12.0%（0.665 → 0.745，p=0.010 统计显著）

### 🌟 核心优势
- Go 生态首个完整学习型检索方案
- 闭环学习：收集 → 评估 → 优化 → 验证
- 专业方法：NDCG、贝叶斯优化、t-test
- 生产就绪：PostgreSQL 持久化、并发安全

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

## [0.4.0] - 2026-01-20

### 🎉 Added - Hybrid Search & Advanced RAG

完整的混合检索和高级 RAG 功能。

#### Milvus 向量存储增强
- **Hybrid Search** - 向量检索 + BM25 全文检索
- **重排序策略** - RRF（Reciprocal Rank Fusion）和加权融合
- **完整 CRUD** - AddDocuments, Search, Delete, Update
- **批量操作优化** - 支持批量插入和删除

#### 高级检索技术
- **Parent Document Retriever** - 索引小块，返回父文档
- **Multi-Query Generation** - 生成多个查询变体
- **HyDE** - 假设文档嵌入
- **Self-Query** - 自动提取结构化查询

#### 文档处理
- **Text Splitters** - RecursiveCharacterTextSplitter, CharacterTextSplitter
- **Document Loaders** - PDF, Word, Excel, HTML, Text

### 📊 统计数据
- 新增代码: ~3,500 行
- 测试覆盖: 85%+

### 📝 Documentation
- 新增 `RELEASE_NOTES_v0.4.0.md`
- 新增 Hybrid Search 使用指南
- 更新 Milvus 集成文档

## [0.3.0] - 2026-01-19

### 🎉 Added - Multi-Agent System

完整的多 Agent 协作系统。

#### Multi-Agent 核心
- **消息总线** - Agent 间高效通信
- **3 种协调策略** - Sequential（顺序）、Parallel（并行）、Hierarchical（层次化）
- **6 个专用 Agent** - Coordinator、Researcher、Writer、Reviewer、Analyst、Planner
- **共享状态管理** - 全局状态和私有状态
- **执行追踪** - 完整的执行历史

#### Agent 增强
- **流式输出** - 实时展示 Agent 思考过程
- **工具并行执行** - 提升性能 3 倍
- **状态持久化** - 支持长时间运行任务

### 📊 统计数据
- 新增代码: ~4,200 行
- 测试覆盖: 85%+

### 📝 Documentation
- 新增 `RELEASE_NOTES_v0.3.0.md`
- 新增 Multi-Agent 用户指南
- 新增 4 个示例程序

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

[Unreleased]: https://github.com/zhucl121/langchain-go/compare/v0.4.2...HEAD
[0.4.2]: https://github.com/zhucl121/langchain-go/compare/v0.4.1...v0.4.2
[0.4.1]: https://github.com/zhucl121/langchain-go/compare/v0.4.0...v0.4.1
[0.4.0]: https://github.com/zhucl121/langchain-go/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/zhucl121/langchain-go/compare/v0.1.1...v0.3.0
[0.1.1]: https://github.com/zhucl121/langchain-go/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/zhucl121/langchain-go/releases/tag/v0.1.0
