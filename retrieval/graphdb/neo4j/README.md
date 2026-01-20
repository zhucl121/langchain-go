# Neo4j 图数据库驱动器

Neo4j 5.x 图数据库的 Go 驱动器实现。

## 📦 安装

```bash
go get github.com/neo4j/neo4j-go-driver/v5
```

## 🚀 快速开始

### 1. 启动 Neo4j

```bash
# 使用 Docker Compose
cd ../../../
docker-compose -f docker-compose.graphdb.yml up -d neo4j

# 等待启动（约 10-15 秒）
docker-compose -f docker-compose.graphdb.yml ps

# 访问 Neo4j 浏览器
open http://localhost:7474
# 用户名: neo4j
# 密码: password123
```

### 2. 使用驱动器

```go
package main

import (
    "context"
    "log"
    
    "github.com/zhucl121/langchain-go/retrieval/graphdb"
    "github.com/zhucl121/langchain-go/retrieval/graphdb/neo4j"
)

func main() {
    // 创建配置
    config := neo4j.Config{
        URI:      "bolt://localhost:7687",
        Username: "neo4j",
        Password: "password123",
        Database: "neo4j",
    }
    
    // 创建驱动器
    driver, err := neo4j.NewNeo4jDriver(config)
    if err != nil {
        log.Fatal(err)
    }
    defer driver.Close()
    
    // 连接
    ctx := context.Background()
    if err := driver.Connect(ctx); err != nil {
        log.Fatal(err)
    }
    
    // 添加节点
    node := &graphdb.Node{
        ID:    "person-alice",
        Type:  "Person",
        Label: "Alice",
        Properties: map[string]interface{}{
            "age":  30,
            "city": "Beijing",
        },
    }
    
    if err := driver.AddNode(ctx, node); err != nil {
        log.Fatal(err)
    }
    
    // 查询节点
    retrieved, err := driver.GetNode(ctx, "person-alice")
    if err != nil {
        log.Fatal(err)
    }
    
    log.Printf("Retrieved: %s (%s)", retrieved.Label, retrieved.Type)
}
```

## 📖 功能特性

### ✅ 已实现

- **连接管理**: Connect/Close/Ping
- **节点操作**: Add/Get/Update/Delete/BatchAdd
- **边操作**: Add/Get/Delete/BatchAdd
- **查询**: FindNodes/FindEdges（支持过滤）
- **图遍历**: Traverse（支持 BFS/DFS）
- **最短路径**: ShortestPath（支持 Dijkstra/BFS）
- **事务支持**: 批量操作自动使用事务
- **连接池**: 可配置的连接池管理

### 🎯 核心方法

```go
// 节点操作
driver.AddNode(ctx, node)
driver.GetNode(ctx, "node-id")
driver.UpdateNode(ctx, node)
driver.DeleteNode(ctx, "node-id")
driver.BatchAddNodes(ctx, nodes)

// 边操作
driver.AddEdge(ctx, edge)
driver.GetEdge(ctx, "edge-id")
driver.DeleteEdge(ctx, "edge-id")
driver.BatchAddEdges(ctx, edges)

// 查询
driver.FindNodes(ctx, graphdb.NodeFilter{
    Types: []string{"Person"},
    Properties: map[string]interface{}{
        "city": "Beijing",
    },
})

// 图遍历
driver.Traverse(ctx, "start-id", graphdb.TraverseOptions{
    MaxDepth:  3,
    Direction: graphdb.DirectionBoth,
    Strategy:  graphdb.StrategyBFS,
})

// 最短路径
driver.ShortestPath(ctx, "start-id", "end-id", graphdb.PathOptions{
    MaxDepth:  5,
    Algorithm: graphdb.AlgorithmBFS,
})
```

## ⚙️ 配置说明

### 基础配置

```go
config := neo4j.Config{
    URI:      "bolt://localhost:7687",  // 连接地址
    Username: "neo4j",                  // 用户名
    Password: "password",               // 密码
    Database: "neo4j",                  // 数据库名
}
```

### 高级配置

```go
config := neo4j.Config{
    URI:                          "bolt://localhost:7687",
    Username:                     "neo4j",
    Password:                     "password",
    Database:                     "neo4j",
    MaxConnectionPoolSize:        100,              // 最大连接数
    ConnectionAcquisitionTimeout: 60 * time.Second, // 获取连接超时
    MaxConnectionLifetime:        1 * time.Hour,    // 连接生命周期
    MaxTransactionRetryTime:      30 * time.Second, // 事务重试时间
    Encrypted:                    false,            // 是否加密
    TrustStrategy:                neo4j.TrustSystemCAs,
}
```

### 默认配置

```go
config := neo4j.DefaultConfig()
config.Password = "your-password"  // 只需修改密码
```

## 🧪 测试

### 单元测试

```bash
# 安装 Driver（如果网络问题，手动下载）
go get github.com/neo4j/neo4j-go-driver/v5

# 运行测试（需要 Neo4j 运行）
go test -v
```

### 集成测试

```bash
# 1. 启动 Neo4j
docker-compose -f ../../../docker-compose.graphdb.yml up -d neo4j

# 2. 运行测试
go test -v -tags=integration

# 3. 停止 Neo4j
docker-compose -f ../../../docker-compose.graphdb.yml stop neo4j
```

## 📊 性能建议

### 1. 创建索引

```cypher
// 在 Neo4j 浏览器中执行
CREATE INDEX person_id FOR (n:Person) ON (n.id);
CREATE INDEX organization_id FOR (n:Organization) ON (n.id);
```

### 2. 使用批量操作

```go
// 避免循环调用
for _, node := range nodes {
    driver.AddNode(ctx, node)  // ❌ 慢
}

// 使用批量操作
driver.BatchAddNodes(ctx, nodes)  // ✅ 快
```

### 3. 连接池配置

```go
config := neo4j.Config{
    MaxConnectionPoolSize: 100,  // 根据并发需求调整
    // ...
}
```

### 4. 事务管理

批量操作自动使用事务，无需手动管理。

## 🔧 故障排查

### 问题：无法连接

```
Error: failed to verify connectivity
```

**解决方案**:
1. 检查 Neo4j 是否启动: `docker ps`
2. 检查端口占用: `lsof -i :7687`
3. 检查配置是否正确

### 问题：认证失败

```
Error: authentication failed
```

**解决方案**:
1. 检查用户名密码是否正确
2. 首次登录需要在浏览器中修改密码

### 问题：数据库不存在

```
Error: database not found
```

**解决方案**:
1. 使用默认数据库: `"neo4j"`
2. 或在 Neo4j 中创建新数据库

## 📚 更多资源

- [Neo4j 官方文档](https://neo4j.com/docs/)
- [Cypher 查询语言](https://neo4j.com/docs/cypher-manual/)
- [Neo4j Go Driver](https://neo4j.com/docs/go-manual/)
- [完整示例](../../../examples/graphdb_demo/)

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

## 📄 许可证

MIT License
