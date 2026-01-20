# Knowledge Graph Builder Demo

这个示例演示如何使用 `builder` 包从文本自动构建知识图谱。

## 功能展示

- ✅ 从文本中提取实体（Entity Extraction）
- ✅ 从文本中提取关系（Relation Extraction）
- ✅ 实体向量化（Entity Embedding）
- ✅ 批量构建知识图谱（Batch Building）
- ✅ 合并多个知识图谱（Graph Merging）
- ✅ 存储到图数据库（Storage to GraphDB）
- ✅ 支持 Mock 和 Neo4j 两种模式

## 运行方式

### 方式 1: Mock 模式（无需外部依赖）

使用内存中的 Mock 数据库和 Mock 提取器：

```bash
cd examples/kg_builder_demo
go run main.go
```

**输出示例**:
```
Running in mock mode...

Using Mock GraphDB!
Using Mock extractors (set OPENAI_API_KEY for real extraction)!

=== Processing Texts ===

Graph 1 (from text 1):
  Text: John Smith is the CEO of TechCorp...
  Entities: 1
    - [entity-mock-92] Mock Person (Person) - confidence: 0.80
      Embedding: [0.240, 0.243, ...] (dim=384)
  Relations: 0

...

✓ Stored 4 nodes
✓ Stored 0 edges

=== Demo Complete ===
```

### 方式 2: OpenAI 模式（需要 API Key）

使用真实的 OpenAI GPT-4 和 Embeddings：

```bash
export OPENAI_API_KEY="sk-..."
cd examples/kg_builder_demo
go run main.go
```

这将使用 GPT-4 进行实体和关系提取，并使用 `text-embedding-3-small` 生成向量。

**预期输出**（使用真实 LLM）:
```
Running in mock mode...

Using Mock GraphDB!
Using OpenAI for extraction and embedding!

=== Processing Texts ===

Graph 1 (from text 1):
  Text: John Smith is the CEO of TechCorp...
  Entities: 3
    - [entity-0-john-smith] John Smith (Person) - confidence: 0.95
      Embedding: [0.012, -0.043, ...] (dim=1536)
    - [entity-1-techcorp] TechCorp (Organization) - confidence: 0.92
      Embedding: [-0.008, 0.031, ...] (dim=1536)
    - [entity-2-san-francisco] San Francisco (Location) - confidence: 0.88
      Embedding: [0.021, -0.012, ...] (dim=1536)
  Relations: 2
    - entity-0-john-smith -[WORKS_FOR]-> entity-1-techcorp (weight: 1.00, confidence: 0.90)
    - entity-1-techcorp -[LOCATED_IN]-> entity-2-san-francisco (weight: 1.00, confidence: 0.85)

...
```

### 方式 3: Neo4j 模式（需要 Neo4j 数据库）

先启动 Neo4j（如果还没启动）：

```bash
cd ../../
docker compose -f docker-compose.graphdb.yml up -d neo4j
```

然后运行：

```bash
export MODE=neo4j
export NEO4J_URI="bolt://localhost:7687"
export NEO4J_USERNAME="neo4j"
export NEO4J_PASSWORD="password123"
export OPENAI_API_KEY="sk-..."  # 可选，用于真实提取

cd examples/kg_builder_demo
go run main.go
```

## 环境变量

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `MODE` | 运行模式：`mock`, `neo4j` | `mock` |
| `OPENAI_API_KEY` | OpenAI API Key（用于真实提取） | - |
| `NEO4J_URI` | Neo4j 连接 URI | `bolt://localhost:7687` |
| `NEO4J_USERNAME` | Neo4j 用户名 | `neo4j` |
| `NEO4J_PASSWORD` | Neo4j 密码 | `password123` |
| `NEO4J_DATABASE` | Neo4j 数据库名 | `neo4j` |

## 代码说明

### 1. 创建图数据库实例

```go
// Mock 模式
graphDB := mock.NewMockGraphDB()

// Neo4j 模式
config := neo4j.DefaultConfig()
config.URI = "bolt://localhost:7687"
graphDB, _ := neo4j.NewNeo4jDriver(config)
graphDB.Connect(ctx)
```

### 2. 创建提取器

```go
// 使用 OpenAI
chatModel := openai.NewChatModel(config)
entityExtractor := builder.NewLLMEntityExtractor(chatModel, nil)
relationExtractor := builder.NewLLMRelationExtractor(chatModel, nil)

// 使用 Mock（测试用）
entityExtractor := &mockEntityExtractor{}
relationExtractor := &mockRelationExtractor{}
```

### 3. 创建 Embedder

```go
// 使用 OpenAI Embeddings
embedModel := embeddings.NewOpenAIEmbeddings(apiKey)
embedder := builder.NewEmbeddingModelAdapter(embedModel)

// 使用 Mock Embedder（测试用）
embedder := builder.NewMockEmbedder(384)
```

### 4. 配置并创建 KGBuilder

```go
config := builder.KGBuilderConfig{
    GraphDB:              graphDB,
    EntityExtractor:      entityExtractor,
    RelationExtractor:    relationExtractor,
    Embedder:             embedder,
    EnableEmbedding:      true,
    EnableDisambiguation: false,
    EnableValidation:     false,
    BatchSize:            10,
    MaxConcurrency:       5,
}

kgBuilder, _ := builder.NewKGBuilder(config)
```

### 5. 批量构建知识图谱

```go
texts := []string{
    "John Smith is the CEO of TechCorp...",
    "Alice Johnson works at TechCorp...",
    // ...
}

// 批量构建
graphs, _ := kgBuilder.BuildBatch(ctx, texts)

// 合并所有图谱
mergedKG, _ := kgBuilder.Merge(ctx, graphs)
```

### 6. 存储到数据库

```go
// 转换为 Node 和 Edge
nodes := make([]*graphdb.Node, len(mergedKG.Entities))
for i, entity := range mergedKG.Entities {
    nodes[i] = entity.ToNode()
}

edges := make([]*graphdb.Edge, len(mergedKG.Relations))
for i, relation := range mergedKG.Relations {
    edges[i] = relation.ToEdge()
}

// 批量存储
graphDB.BatchAddNodes(ctx, nodes)
graphDB.BatchAddEdges(ctx, edges)
```

## 示例文本

程序使用以下示例文本构建知识图谱：

1. "John Smith is the CEO of TechCorp, a leading technology company based in San Francisco."
2. "Alice Johnson works as a senior engineer at TechCorp. She specializes in AI and machine learning."
3. "Bob Chen founded DataFlow in 2020. The company focuses on big data analytics."
4. "TechCorp acquired DataFlow in 2023 for $500 million."

**预期提取结果**（使用真实 LLM）:

**实体**:
- John Smith (Person)
- Alice Johnson (Person)
- Bob Chen (Person)
- TechCorp (Organization)
- DataFlow (Organization)
- San Francisco (Location)
- AI (Technology)
- machine learning (Technology)
- big data analytics (Technology)

**关系**:
- John Smith -[WORKS_FOR]-> TechCorp
- John Smith -[HAS_ROLE]-> CEO
- Alice Johnson -[WORKS_FOR]-> TechCorp
- Alice Johnson -[SPECIALIZES_IN]-> AI
- Bob Chen -[FOUNDED]-> DataFlow
- TechCorp -[LOCATED_IN]-> San Francisco
- TechCorp -[ACQUIRED]-> DataFlow

## 在 Neo4j Browser 中查看

如果使用 Neo4j 模式，可以在浏览器中查看图谱：

1. 打开 http://localhost:7474
2. 登录（用户名：neo4j，密码：password123）
3. 运行 Cypher 查询：

```cypher
// 查看所有节点和关系
MATCH (n)-[r]->(m)
RETURN n, r, m
LIMIT 100

// 查看所有人
MATCH (p:Person)
RETURN p

// 查看某人的所有关系
MATCH (p:Person {name: "John Smith"})-[r]-(other)
RETURN p, r, other

// 查找两个实体之间的路径
MATCH path = shortestPath(
  (a:Person {name: "John Smith"})-[*]-(b:Organization {name: "DataFlow"})
)
RETURN path
```

## 最佳实践

1. **选择合适的 LLM**: 实体和关系提取的质量取决于 LLM 能力，推荐 GPT-4 或 Claude-3
2. **批量处理**: 使用 `BuildBatch` 处理多个文本可以提高效率
3. **启用向量化**: `EnableEmbedding: true` 可以为后续的语义搜索做准备
4. **合并图谱**: 使用 `Merge` 合并多个图谱，自动去重实体和关系
5. **错误处理**: 生产环境中应该添加完整的错误处理和重试逻辑

## 下一步

- 查看 [GraphRAG Demo](../graphrag_demo/) 了解如何使用构建的知识图谱进行检索
- 查看 [Neo4j Integration Test](../../retrieval/graphdb/neo4j/) 了解更多图数据库操作
- 查看 [Builder 包文档](../../retrieval/graphdb/builder/) 了解完整 API
