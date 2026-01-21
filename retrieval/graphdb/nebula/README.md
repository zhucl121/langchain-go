# NebulaGraph 集成使用指南

## 概述

NebulaGraph 是一个开源的分布式图数据库，支持超大规模图数据（千亿级节点/边）。

本集成为 LangChain-Go 提供了 NebulaGraph 支持，实现了统一的 `graphdb.GraphDB` 接口。

## 快速开始

### 1. 启动 NebulaGraph

使用 Docker Compose 启动 NebulaGraph 集群：

```bash
# 启动所有服务（metad、storaged、graphd）
docker compose -f docker-compose.graphdb.yml up -d nebula-metad nebula-storaged nebula-graphd

# 检查服务状态
docker ps | grep nebula

# 查看日志
docker logs langchain-nebula-graphd
```

### 2. 初始化图空间

连接到 NebulaGraph 并创建图空间和 Schema：

```bash
# 方法 1：使用 nebula-console（推荐）
docker run --rm -ti --network host vesoft/nebula-console:v3 \
  -addr 127.0.0.1 -port 9669 -u root -p nebula

# 方法 2：使用 HTTP API
curl -X POST http://localhost:19669/query \
  -H 'Content-Type: application/json' \
  -d '{"gql": "CREATE SPACE IF NOT EXISTS langchain(partition_num=10, replica_factor=1, vid_type=FIXED_STRING(256));"}'
```

创建 Schema：

```cypher
-- 创建图空间
CREATE SPACE IF NOT EXISTS langchain(partition_num=10, replica_factor=1, vid_type=FIXED_STRING(256));

-- 使用图空间
USE langchain;

-- 创建 Tag（节点类型）
CREATE TAG IF NOT EXISTS Person(name string, age int, city string);
CREATE TAG IF NOT EXISTS Organization(name string, industry string);
CREATE TAG IF NOT EXISTS Document(content string, source string);

-- 创建 Edge Type（边类型）
CREATE EDGE IF NOT EXISTS WORKS_FOR(since int, position string);
CREATE EDGE IF NOT EXISTS KNOWS(since int);
CREATE EDGE IF NOT EXISTS REFERENCES(context string);
```

### 3. Go 代码示例

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/zhucl121/langchain-go/retrieval/graphdb"
    "github.com/zhucl121/langchain-go/retrieval/graphdb/nebula"
)

func main() {
    // 1. 创建配置
    config := nebula.DefaultConfig().
        WithSpace("langchain").
        WithAddresses([]string{"127.0.0.1:9669"}).
        WithTimeout(30 * time.Second)

    // 2. 创建驱动器
    driver, err := nebula.NewNebulaDriver(config)
    if err != nil {
        log.Fatal(err)
    }

    // 3. 连接
    ctx := context.Background()
    if err := driver.Connect(ctx); err != nil {
        log.Fatal(err)
    }
    defer driver.Close()

    // 4. 添加节点
    node := &graphdb.Node{
        ID:    "person-1",
        Type:  "Person",
        Label: "John Smith",
        Properties: map[string]interface{}{
            "name": "John Smith",
            "age":  30,
            "city": "San Francisco",
        },
    }
    if err := driver.AddNode(ctx, node); err != nil {
        log.Fatal(err)
    }

    // 5. 添加边
    edge := &graphdb.Edge{
        Source:   "person-1",
        Target:   "org-1",
        Type:     "WORKS_FOR",
        Directed: true,
        Properties: map[string]interface{}{
            "since":    2020,
            "position": "Engineer",
        },
    }
    if err := driver.AddEdge(ctx, edge); err != nil {
        log.Fatal(err)
    }

    // 6. 图遍历
    result, err := driver.Traverse(ctx, "person-1", graphdb.TraverseOptions{
        MaxDepth:  3,
        Strategy:  graphdb.StrategyBFS,
        Direction: graphdb.DirectionBoth,
    })
    if err != nil {
        log.Fatal(err)
    }

    log.Printf("Found %d nodes, %d edges\n", len(result.Nodes), len(result.Edges))

    // 7. 最短路径
    path, err := driver.ShortestPath(ctx, "person-1", "org-1", graphdb.PathOptions{
        MaxDepth: 10,
    })
    if err != nil {
        log.Fatal(err)
    }

    log.Printf("Path length: %d\n", path.Length)
}
```

## 配置选项

### 基本配置

```go
config := nebula.Config{
    Addresses: []string{
        "127.0.0.1:9669",
        "192.168.1.100:9669",  // 多个地址用于高可用
    },
    Username:        "root",
    Password:        "nebula",
    Space:           "langchain",
    Timeout:         30 * time.Second,
    IdleTime:        3600 * time.Second,
    MaxConnPoolSize: 100,
    MinConnPoolSize: 10,
}
```

### 链式配置

```go
config := nebula.DefaultConfig().
    WithSpace("my_graph").
    WithAddresses([]string{"192.168.1.100:9669"}).
    WithUsername("my_user").
    WithPassword("my_pass").
    WithPoolSize(10, 50).
    WithTimeout(60 * time.Second)
```

## nGQL 查询语言

### 基础查询

```cypher
-- 插入节点
INSERT VERTEX Person(name, age) VALUES "person-1":("John", 30);

-- 插入边
INSERT EDGE WORKS_FOR(since) VALUES "person-1"->"org-1":(2020);

-- 查询节点
FETCH PROP ON Person "person-1" YIELD properties(vertex);

-- 遍历
GO FROM "person-1" OVER WORKS_FOR YIELD dst(edge) AS id;

-- 最短路径
FIND SHORTEST PATH FROM "person-1" TO "org-1" OVER * UPTO 5 STEPS;

-- MATCH 查询
MATCH (v:Person) WHERE v.Person.age > 30 RETURN v;
```

### 使用 QueryBuilder

```go
qb := nebula.NewQueryBuilder("langchain")

// 插入节点
query := qb.InsertVertex("person-1", "Person", map[string]interface{}{
    "name": "John",
    "age":  30,
})

// 插入边
query = qb.InsertEdge("person-1", "org-1", "WORKS_FOR", map[string]interface{}{
    "since": 2020,
})

// 遍历
query = qb.Traverse("person-1", 3, "BIDIRECT")

// 最短路径
query = qb.ShortestPath("person-1", "org-1", 5)

// 执行查询
result, err := driver.Execute(ctx, query)
```

## 与 Neo4j 对比

| 特性 | NebulaGraph | Neo4j |
|------|------------|-------|
| **架构** | 分布式原生 | 单机/集群 |
| **扩展性** | 水平扩展 | 垂直为主 |
| **节点规模** | 千亿级 | 十亿级 |
| **查询语言** | nGQL | Cypher |
| **Schema** | 强 Schema | 灵活 Schema |
| **适用场景** | 超大规模图 | 中等规模图 |
| **部署复杂度** | 高（多组件） | 低（单实例） |

## 最佳实践

### 1. Schema 设计

NebulaGraph 是强 Schema 的，必须先定义 Tag 和 Edge Type：

```cypher
-- 定义节点类型
CREATE TAG Person(name string, age int);

-- 定义边类型
CREATE EDGE KNOWS(since int);

-- 创建索引（用于查询优化）
CREATE TAG INDEX person_name_index ON Person(name(20));
```

### 2. 批量操作

使用批量插入提高性能：

```cypher
INSERT VERTEX Person(name, age) VALUES 
  "person-1":("John", 30),
  "person-2":("Jane", 25),
  "person-3":("Bob", 35);
```

### 3. 查询优化

- 限制遍历深度（避免过深）
- 使用索引加速查询
- 合理设置连接池大小

```go
config := nebula.DefaultConfig().
    WithPoolSize(20, 100).  // 根据并发需求调整
    WithTimeout(60 * time.Second)
```

### 4. 错误处理

```go
if err := driver.AddNode(ctx, node); err != nil {
    if strings.Contains(err.Error(), "existed") {
        // 节点已存在，更新
        driver.UpdateNode(ctx, node)
    } else {
        return fmt.Errorf("failed to add node: %w", err)
    }
}
```

## 故障排除

### 连接失败

```bash
# 检查服务状态
docker ps | grep nebula

# 查看 graphd 日志
docker logs langchain-nebula-graphd

# 测试连接
curl http://localhost:19669/status
```

### Schema 错误

```cypher
-- 查看已有的 Tag
SHOW TAGS;

-- 查看已有的 Edge Type
SHOW EDGES;

-- 查看 Tag 定义
DESCRIBE TAG Person;

-- 查看 Edge Type 定义
DESCRIBE EDGE WORKS_FOR;
```

### 性能问题

```cypher
-- 查看查询执行计划
EXPLAIN GO FROM "person-1" OVER WORKS_FOR YIELD dst(edge);

-- 查看索引
SHOW TAG INDEXES;

-- 重建索引
REBUILD TAG INDEX person_name_index;
```

## 限制与注意事项

### 1. 边的标识

NebulaGraph 的边通过 `(源节点, 边类型, 目标节点)` 唯一标识：
- 没有独立的边 ID
- 删除/查询边需要指定源、目标和类型

### 2. Schema 约束

- 必须先创建 Tag 和 Edge Type
- 属性类型固定，不能动态添加
- 需要提前规划 Schema

### 3. 数据类型

支持的数据类型：
- bool
- int (int8, int16, int32, int64)
- float, double
- string
- date, time, datetime
- geography

### 4. 当前实现状态

本集成提供基础功能，部分高级特性待完善：
- ✅ 节点/边 CRUD
- ✅ 图遍历
- ✅ 最短路径
- ✅ nGQL 查询构建器
- ⚠️ 结果集转换（部分实现）
- ❌ 全文搜索
- ❌ 图算法

## 参考资料

- [NebulaGraph 官方文档](https://docs.nebula-graph.io/)
- [nGQL 语法参考](https://docs.nebula-graph.io/3.6.0/3.ngql-guide/)
- [Go 客户端文档](https://github.com/vesoft-inc/nebula-go)
- [Docker 部署指南](https://docs.nebula-graph.io/3.6.0/2.quick-start/1.quick-start-workflow/)

## 下一步

- 完善结果集转换逻辑
- 添加更多 nGQL 查询方法
- 实现批量操作优化
- 添加图算法支持
- 完善错误处理和重试机制
