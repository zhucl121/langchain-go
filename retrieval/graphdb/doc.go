// Package graphdb 提供统一的图数据库抽象接口。
//
// # 概述
//
// graphdb 包定义了图数据库的统一接口，支持多种图数据库实现：
//   - Neo4j: 业界最成熟的图数据库
//   - NebulaGraph: 高性能分布式图数据库
//
// # 核心概念
//
// 节点（Node）: 图中的实体，具有 ID、类型、标签和属性。
//
// 边（Edge）: 连接两个节点的关系，具有类型、方向和权重。
//
// 遍历（Traverse）: 从起始节点按照一定规则访问相邻节点。
//
// 路径（Path）: 连接两个节点的一系列节点和边。
//
// # 使用示例
//
// 连接数据库:
//
//	import (
//	    "context"
//	    "github.com/zhucl121/langchain-go/retrieval/graphdb"
//	    "github.com/zhucl121/langchain-go/retrieval/graphdb/neo4j"
//	)
//
//	// 创建 Neo4j 实例
//	db, err := neo4j.NewNeo4jDriver(neo4j.Config{
//	    URI:      "bolt://localhost:7687",
//	    Username: "neo4j",
//	    Password: "password",
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer db.Close()
//
//	// 连接
//	if err := db.Connect(context.Background()); err != nil {
//	    log.Fatal(err)
//	}
//
// 添加节点:
//
//	node := &graphdb.Node{
//	    ID:    "person-1",
//	    Type:  "person",
//	    Label: "Alice",
//	    Properties: map[string]interface{}{
//	        "age":  30,
//	        "city": "Beijing",
//	    },
//	}
//	err = db.AddNode(context.Background(), node)
//
// 添加边:
//
//	edge := &graphdb.Edge{
//	    ID:       "edge-1",
//	    Source:   "person-1",
//	    Target:   "person-2",
//	    Type:     "knows",
//	    Label:    "认识",
//	    Directed: true,
//	}
//	err = db.AddEdge(context.Background(), edge)
//
// 图遍历:
//
//	result, err := db.Traverse(context.Background(), "person-1", graphdb.TraverseOptions{
//	    MaxDepth:  2,
//	    Direction: graphdb.DirectionBoth,
//	    Strategy:  graphdb.StrategyBFS,
//	    Limit:     10,
//	})
//
//	for _, node := range result.Nodes {
//	    fmt.Printf("Found node: %s (%s)\n", node.Label, node.Type)
//	}
//
// 最短路径:
//
//	path, err := db.ShortestPath(context.Background(), "person-1", "person-2", graphdb.PathOptions{
//	    MaxDepth:  5,
//	    Algorithm: graphdb.AlgorithmBFS,
//	})
//
//	fmt.Printf("Path length: %d, cost: %.2f\n", path.Length, path.Cost)
//
// # Mock 实现
//
// 用于测试的 Mock 实现:
//
//	import "github.com/zhucl121/langchain-go/retrieval/graphdb/mock"
//
//	mockDB := mock.NewMockGraphDB()
//	mockDB.AddNode(ctx, node)
//
// # 性能建议
//
// 1. 批量操作: 使用 BatchAddNodes 和 BatchAddEdges 提升性能
// 2. 限制深度: 遍历时设置合理的 MaxDepth 避免过度遍历
// 3. 连接池: 生产环境使用连接池管理连接
// 4. 索引: 在常用查询字段上创建索引
//
// # 错误处理
//
// 所有操作都返回错误，应该检查错误类型:
//
//	node, err := db.GetNode(ctx, "id")
//	if errors.Is(err, graphdb.ErrNodeNotFound) {
//	    // 节点不存在
//	}
package graphdb
