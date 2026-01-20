# GraphRAG 完整演示

这个示例程序展示了 GraphRAG (Graph Retrieval Augmented Generation) 的完整功能，包括：

- ✅ 知识图谱自动构建
- ✅ 混合检索（向量 + 图遍历）
- ✅ 4种融合策略对比
- ✅ 3种重排序策略对比
- ✅ 上下文增强演示
- ✅ 性能统计展示

---

## 📋 功能演示

### Step 1: 准备示例文档
准备5个关于 TechCorp 公司的示例文档，包含人物、组织、产品和地点信息。

### Step 2: 构建知识图谱
使用 KGBuilder 自动从文档中提取实体和关系：
- 提取实体（人物、组织、产品、地点）
- 提取关系（WORKS_FOR, LOCATED_IN, LAUNCHED 等）
- 存储到图数据库

### Step 3: 向量化文档
将文档转换为向量并存储到向量数据库。

### Step 4: GraphRAG 检索
使用混合检索（向量 + 图）回答3个查询：
- "Who is the CEO of TechCorp?"
- "What products does TechCorp offer?"
- "Where is TechCorp located?"

### Step 5: 融合策略对比
对比4种融合策略的效果：
- **Weighted** - 加权融合
- **RRF** - Reciprocal Rank Fusion
- **Max** - 最大值融合
- **Min** - 最小值融合

### Step 6: 重排序策略对比
对比3种重排序策略的效果：
- **Score** - 基于分数排序
- **Diversity** - 基于多样性排序
- **MMR** - Maximal Marginal Relevance

### Step 7: 上下文增强
对比启用和不启用上下文增强的差异，展示图结构信息如何增强检索结果。

---

## 🚀 运行模式

### Mode 1: Mock 模式（推荐入门）⭐

**无需任何外部服务**，使用 Mock 组件快速体验。

```bash
# 直接运行
cd examples/graphrag_demo
go run main.go

# 或显式指定
DEMO_MODE=mock go run main.go
```

**特点**:
- ✅ 无需安装数据库
- ✅ 无需 API Key
- ✅ 快速启动
- ✅ 适合学习和测试

**输出示例**:
```
🚀 GraphRAG Demo - Mode: mock
============================================================
📦 使用 Mock 组件（无需外部服务）

📚 Step 1: 准备示例文档
  1. John Smith is the CEO of TechCorp...
  2. TechCorp is headquartered in San Francisco...
  总计: 5 个文档

🔨 Step 2: 构建知识图谱
  提取实体: 8 个
  提取关系: 6 个
  示例实体:
    - John Smith (Person)
    - TechCorp (Organization)
    - Alice Johnson (Person)
  ...
```

---

### Mode 2: OpenAI 模式

使用真实的 OpenAI API 进行实体提取和向量化。

**前置要求**:
```bash
export OPENAI_API_KEY="your-openai-api-key"
```

**运行**:
```bash
DEMO_MODE=openai go run main.go
```

**特点**:
- ✅ 真实的 LLM 实体提取
- ✅ 高质量的向量化
- ✅ 更准确的检索结果
- ⚠️ 需要 OpenAI API Key
- ⚠️ 会产生 API 费用（约 $0.01-0.05）

---

### Mode 3: Neo4j 模式（完整体验）⭐⭐⭐

使用 Neo4j 图数据库 + OpenAI，体验完整的生产环境。

**前置要求**:

1. **启动 Neo4j**:
```bash
# 使用项目提供的 docker-compose
cd /path/to/langchain-go
docker compose -f docker-compose.graphdb.yml up -d neo4j

# 等待启动
docker logs -f langchain-go-neo4j-1
```

2. **设置环境变量**:
```bash
export OPENAI_API_KEY="your-openai-api-key"
export NEO4J_URI="bolt://localhost:7687"
export NEO4J_USER="neo4j"
export NEO4J_PASSWORD="testpassword"
```

**运行**:
```bash
DEMO_MODE=neo4j go run main.go
```

**特点**:
- ✅ 完整的图数据库支持
- ✅ 持久化存储
- ✅ 可视化图结构
- ✅ 真实的生产环境
- ⚠️ 需要 Docker
- ⚠️ 需要 OpenAI API Key

**查看图数据**:
```bash
# Neo4j Browser
open http://localhost:7474

# 登录后执行 Cypher 查询
MATCH (n) RETURN n LIMIT 25
MATCH (n)-[r]->(m) RETURN n,r,m LIMIT 50
```

---

## 📊 示例输出

### 混合检索结果

```
🔍 Step 4: GraphRAG 检索演示

  查询 1: Who is the CEO of TechCorp?
  ✅ 找到 6 个结果
    1. [0.850] John Smith is the CEO of TechCorp, a leading technology...
       相关实体: John Smith (Person), TechCorp (Organization)
    2. [0.720] Person: John Smith
        CEO of TechCorp
        ...
    3. [0.680] Alice Johnson works at TechCorp as the Chief Technology...
  📊 统计: 向量=3, 图=3, 融合=6, 耗时=15ms
```

### 融合策略对比

```
🎯 Step 5: 融合策略对比

  策略: 加权融合 (Weighted)
  找到 5 个结果
    1. [0.820] John Smith is the CEO of TechCorp...
    2. [0.750] TechCorp is headquartered in San Francisco...
    3. [0.680] Alice Johnson works at TechCorp...

  策略: RRF 融合
  找到 5 个结果
    1. [0.032] John Smith is the CEO of TechCorp...
    2. [0.028] TechCorp recently launched CloudMax...
    3. [0.025] TechCorp is headquartered in San Francisco...

  策略: 最大值融合 (Max)
  找到 5 个结果
    1. [1.000] John Smith is the CEO of TechCorp...
    2. [0.950] TechCorp is headquartered in San Francisco...
    3. [0.850] Alice Johnson works at TechCorp...

  策略: 最小值融合 (Min)
  找到 5 个结果
    1. [0.600] John Smith is the CEO of TechCorp...
    2. [0.500] TechCorp is headquartered in San Francisco...
    3. [0.400] TechCorp recently launched CloudMax...
```

### 重排序策略对比

```
🔄 Step 6: 重排序策略对比

  策略: 分数排序 (Score)
  找到 5 个结果
  类别分布: leadership=2 company=1 product=1 location=1

  策略: 多样性排序 (Diversity)
  找到 5 个结果
  类别分布: leadership=1 company=1 product=1 location=2

  策略: MMR 排序
  找到 5 个结果
  类别分布: leadership=2 company=1 product=1 location=1
```

### 上下文增强对比

```
✨ Step 7: 上下文增强展示

  不启用上下文增强:
  内容长度: 120 字符
  元数据键数: 3

  启用上下文增强:
  内容长度: 180 字符
  元数据键数: 8
  增强的元数据:
    - related_entities: [John Smith (Person), TechCorp (Organization)]
    - neighbor_count: 2
    - graph_depth: 1
  相关实体: John Smith (Person), TechCorp (Organization)
```

---

## 🔧 自定义配置

### 修改融合权重

编辑 `main.go` 中的配置：

```go
config := graphrag.DefaultConfig(graphDB, vectorStore)
config.VectorWeight = 0.7  // 增加向量权重
config.GraphWeight = 0.3   // 降低图权重
```

### 修改遍历深度

```go
config.MaxTraverseDepth = 3  // 增加到3层
```

### 修改返回数量

```go
config.TopK = 20  // 返回更多结果
```

---

## 📚 代码结构

### setupMockMode
创建 Mock 组件，无需外部服务。

### setupOpenAIMode
创建 OpenAI 组件，使用真实的 LLM 和 Embeddings。

### setupNeo4jMode
创建 Neo4j + OpenAI 完整环境。

### prepareDocuments
准备5个示例文档，涵盖不同类型的信息。

### buildKnowledgeGraph
使用 KGBuilder 构建知识图谱：
- 提取实体
- 提取关系
- 存储到图数据库

### vectorizeDocuments
将文档向量化并存储。

### demoGraphRAGRetrieval
演示基础的 GraphRAG 混合检索。

### demoFusionStrategies
对比4种融合策略的效果。

### demoRerankStrategies
对比3种重排序策略的效果。

### demoContextAugmentation
展示上下文增强的效果。

---

## 🎯 学习要点

### 1. 混合检索的优势

向量检索擅长语义相似度，图遍历擅长结构化关系。结合两者可以：
- 找到语义相似的文档
- 发现实体间的关联
- 提供更全面的上下文

### 2. 融合策略的选择

- **Weighted**: 适合大多数场景，简单直观
- **RRF**: 对分数 scale 更鲁棒，适合多检索器融合
- **Max**: 适合任一来源的高分结果都重要的场景
- **Min**: 适合需要同时在两个来源中得分都高的场景

### 3. 重排序策略的选择

- **Score**: 纯分数排序，最简单
- **Diversity**: 增加结果多样性，展示不同角度
- **MMR**: 平衡相关性和多样性，通用性强

### 4. 上下文增强的价值

GraphRAG 自动为检索结果添加：
- 相关实体列表
- 关系路径
- 图结构信息
- 邻居统计

这些信息帮助 LLM 更好地理解上下文和实体关系。

---

## 🐛 故障排除

### Q1: "Failed to connect to mock GraphDB"

**解决**: Mock 模式不应该出现连接失败。检查是否有其他错误信息。

### Q2: "OPENAI_API_KEY environment variable is required"

**解决**: 
```bash
export OPENAI_API_KEY="your-key-here"
```

### Q3: "Failed to connect to Neo4j"

**解决**:
1. 确认 Neo4j 已启动: `docker ps | grep neo4j`
2. 检查端口: `lsof -i :7687`
3. 查看日志: `docker logs langchain-go-neo4j-1`

### Q4: OpenAI API 调用失败

**解决**:
1. 检查 API Key 是否有效
2. 检查网络连接
3. 检查账户余额

---

## 📖 相关文档

- [GraphRAG Package 文档](../../retrieval/retrievers/graphrag/doc.go)
- [KGBuilder 文档](../../retrieval/graphdb/builder/doc.go)
- [Phase 4 完成报告](../../docs/V0.4.1_PHASE4_COMPLETE.md)
- [v0.4.1 实现计划](../../docs/V0.4.1_IMPLEMENTATION_PLAN.md)

---

## 🤝 贡献

欢迎提交问题和改进建议！

---

## 📝 许可

MIT License

---

**作者**: LangChain-Go Team  
**版本**: v0.4.1  
**最后更新**: 2026-01-21
