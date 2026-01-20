// Package neo4j 提供 Neo4j 图数据库驱动器实现。
//
// # 概述
//
// neo4j 包实现了 graphdb.GraphDB 接口，提供对 Neo4j 图数据库的访问。
// Neo4j 是业界最成熟和广泛使用的图数据库。
//
// # 快速开始
//
// 基本使用:
//
//	import (
//	    "context"
//	    "github.com/zhucl121/langchain-go/retrieval/graphdb"
//	    "github.com/zhucl121/langchain-go/retrieval/graphdb/neo4j"
//	)
//
//	// 创建配置
//	config := neo4j.Config{
//	    URI:      "bolt://localhost:7687",
//	    Username: "neo4j",
//	    Password: "password",
//	    Database: "neo4j",
//	}
//
//	// 创建驱动器
//	driver, err := neo4j.NewNeo4jDriver(config)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer driver.Close()
//
//	// 连接数据库
//	if err := driver.Connect(context.Background()); err != nil {
//	    log.Fatal(err)
//	}
//
//	// 添加节点
//	node := &graphdb.Node{
//	    ID:    "person-1",
//	    Type:  "Person",
//	    Label: "Alice",
//	    Properties: map[string]interface{}{
//	        "age": 30,
//	    },
//	}
//	err = driver.AddNode(context.Background(), node)
//
// # Docker 部署
//
// 使用 Docker 快速启动 Neo4j:
//
//	docker run -d \
//	  --name neo4j \
//	  -p 7474:7474 -p 7687:7687 \
//	  -e NEO4J_AUTH=neo4j/password \
//	  neo4j:5.15
//
// 访问 Neo4j 浏览器: http://localhost:7474
//
// # 配置选项
//
// 完整配置:
//
//	config := neo4j.Config{
//	    URI:                          "bolt://localhost:7687",
//	    Username:                     "neo4j",
//	    Password:                     "password",
//	    Database:                     "neo4j",
//	    MaxConnectionPoolSize:        100,
//	    ConnectionAcquisitionTimeout: 60 * time.Second,
//	    MaxConnectionLifetime:        1 * time.Hour,
//	    MaxTransactionRetryTime:      30 * time.Second,
//	    Encrypted:                    false,
//	}
//
// 或使用默认配置:
//
//	config := neo4j.DefaultConfig()
//	config.Password = "your-password"
//
// # Cypher 查询
//
// Neo4j 使用 Cypher 查询语言。本包自动构建 Cypher 查询，但你也可以了解一些基础：
//
// 创建节点:
//
//	CREATE (n:Person {id: 'alice', name: 'Alice', age: 30})
//
// 创建关系:
//
//	MATCH (a:Person {id: 'alice'}), (b:Person {id: 'bob'})
//	CREATE (a)-[r:KNOWS {since: 2020}]->(b)
//
// 查询:
//
//	MATCH (n:Person {name: 'Alice'})
//	RETURN n
//
// 图遍历:
//
//	MATCH path = (start:Person {id: 'alice'})-[*1..3]-(end)
//	RETURN path
//
// # 性能优化
//
// 1. **使用索引**:
//
//	CREATE INDEX person_id FOR (n:Person) ON (n.id)
//
// 2. **批量操作**: 使用 BatchAddNodes 和 BatchAddEdges
//
// 3. **连接池**: 配置 MaxConnectionPoolSize
//
// 4. **事务**: 批量操作自动使用事务
//
// # 注意事项
//
// 1. **节点标签**: Neo4j 中节点类型对应标签（Label）
// 2. **关系方向**: Neo4j 关系总是有向的
// 3. **属性类型**: 支持基本类型和数组
// 4. **ID 管理**: 节点 ID 由应用层管理，不使用 Neo4j 内部 ID
//
// # 错误处理
//
// 所有操作都返回错误，应该检查错误类型:
//
//	node, err := driver.GetNode(ctx, "id")
//	if errors.Is(err, graphdb.ErrNodeNotFound) {
//	    // 节点不存在
//	}
//	if errors.Is(err, graphdb.ErrNotConnected) {
//	    // 未连接
//	}
//
// # 更多资源
//
// - Neo4j 文档: https://neo4j.com/docs/
// - Cypher 参考: https://neo4j.com/docs/cypher-manual/
// - Go Driver: https://neo4j.com/docs/go-manual/
package neo4j
