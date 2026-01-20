# GraphDB Demo

这是一个图数据库功能演示程序，展示了 LangChain-Go 的图数据库抽象接口的使用方法。

## 功能演示

1. **连接数据库**: 连接到图数据库并进行健康检查
2. **创建知识图谱**: 批量创建节点和边
3. **查询节点**: 按类型和属性查询节点
4. **图遍历**: 从起始节点进行图遍历
5. **最短路径**: 查找两个节点之间的最短路径

## 快速开始

### 使用 Mock 实现（本地测试）

```bash
# 运行演示程序（使用内存 Mock）
go run main.go
```

### 使用 Neo4j（Docker）

```bash
# 1. 启动 Neo4j
cd ../../
docker-compose -f docker-compose.graphdb.yml up -d neo4j

# 2. 等待 Neo4j 启动（约 10-15 秒）
docker-compose -f docker-compose.graphdb.yml ps

# 3. 访问 Neo4j 浏览器
open http://localhost:7474
# 用户名: neo4j
# 密码: password123

# 4. 修改 main.go，使用 Neo4j 实现
# import "github.com/zhucl121/langchain-go/retrieval/graphdb/neo4j"
# db, _ := neo4j.NewNeo4jDriver(neo4j.Config{...})

# 5. 运行演示
go run main.go
```

### 使用 NebulaGraph（Docker）

```bash
# 1. 启动 NebulaGraph（包含 metad, storaged, graphd）
cd ../../
docker-compose -f docker-compose.graphdb.yml up -d nebula-graphd

# 2. 等待 NebulaGraph 启动（约 30 秒）
docker-compose -f docker-compose.graphdb.yml ps

# 3. 修改 main.go，使用 NebulaGraph 实现
# import "github.com/zhucl121/langchain-go/retrieval/graphdb/nebula"
# db, _ := nebula.NewNebulaDriver(nebula.Config{...})

# 4. 运行演示
go run main.go
```

## 示例输出

```
=== LangChain-Go GraphDB Demo ===

1. 连接数据库...
✓ 连接成功

2. 创建知识图谱...
  - 添加了 4 个人物节点
  - 添加了 2 个公司节点
  - 添加了 6 条关系
✓ 知识图谱创建成功

3. 查询节点...
  找到 4 个人物:
    - Alice (年龄: 30, 城市: Beijing)
    - Bob (年龄: 28, 城市: Shanghai)
    - Charlie (年龄: 35, 城市: Shenzhen)
    - David (年龄: 32, 城市: Hangzhou)

  北京的人物:
    - Alice

4. 图遍历...
  从 Alice 开始遍历 (深度=2):
    发现 5 个节点:
      - Alice (person)
      - Bob (person)
      - TechCorp (organization)
      - Charlie (person)
      - DesignCo (organization)
    发现 4 条边:
      - Alice -[knows]-> Bob
      - Alice -[works_for]-> TechCorp
      - Charlie -[manages]-> Alice
      - Bob -[works_for]-> DesignCo

5. 查找最短路径...
  Alice 到 David 的最短路径:
    路径长度: 3
    路径成本: 3.00
    路径节点:
      1. Alice (person)
      2. Bob (person)
      3. Charlie (person)
      4. David (person)

=== Demo 完成 ===
```

## 知识图谱结构

演示程序创建的知识图谱结构如下：

```
People (人物):
  - Alice (30岁, 北京, 工程师)
  - Bob (28岁, 上海, 设计师)
  - Charlie (35岁, 深圳, 经理)
  - David (32岁, 杭州, 开发者)

Organizations (组织):
  - TechCorp (科技公司, 500人, 北京)
  - DesignCo (设计公司, 200人, 上海)

Relationships (关系):
  - Alice --认识--> Bob (同事)
  - Bob --认识--> Charlie (朋友)
  - Charlie --认识--> David (朋友)
  - Alice --工作于--> TechCorp
  - Bob --工作于--> DesignCo
  - Charlie --管理--> Alice
```

## 停止服务

```bash
# 停止 Neo4j
docker-compose -f ../../docker-compose.graphdb.yml stop neo4j

# 停止 NebulaGraph
docker-compose -f ../../docker-compose.graphdb.yml stop nebula-graphd nebula-storaged nebula-metad

# 停止所有图数据库服务
docker-compose -f ../../docker-compose.graphdb.yml down

# 清理数据（包括持久化数据）
docker-compose -f ../../docker-compose.graphdb.yml down -v
```

## 下一步

1. 查看 GraphRAG 检索器示例：`examples/graphrag_demo/`
2. 查看知识图谱构建示例：`examples/kg_builder_demo/`
3. 阅读 API 文档：`docs/V0.4.1_USER_GUIDE.md`

## 注意事项

1. **Mock 实现**: 适合快速测试和开发，数据存储在内存中
2. **Neo4j**: 生产环境推荐，成熟稳定，有丰富的可视化工具
3. **NebulaGraph**: 适合大规模图数据，性能优秀，支持分布式
4. **性能**: 图遍历时注意控制 `MaxDepth`，避免遍历过深导致性能问题
5. **内存**: Neo4j 默认配置需要至少 2GB 内存
