// Package nebula 提供 NebulaGraph 图数据库驱动器。
//
// NebulaGraph 是一个开源的分布式图数据库，支持超大规模图数据（千亿级节点/边）。
//
// # 核心特性
//
// ## 分布式架构
//
// NebulaGraph 采用原生分布式架构，支持水平扩展：
//   - 存储层（Storage）：分布式存储节点和边
//   - 图计算层（Graph）：分布式查询处理
//   - 元数据层（Meta）：管理集群元数据
//
// ## nGQL 查询语言
//
// nGQL 是类似 Cypher 的查询语言：
//
//	// 插入节点
//	INSERT VERTEX Person(name, age) VALUES "person-1":("John", 30)
//
//	// 插入边
//	INSERT EDGE WORKS_FOR(since) VALUES "person-1"->"org-1":(2020)
//
//	// 遍历
//	GO FROM "person-1" OVER WORKS_FOR YIELD dst(edge)
//
//	// 最短路径
//	FIND SHORTEST PATH FROM "person-1" TO "org-1" OVER *
//
// # 使用示例
//
// ## 基础用法
//
//	import "github.com/zhucl121/langchain-go/retrieval/graphdb/nebula"
//
//	// 1. 创建配置
//	config := nebula.DefaultConfig()
//	config.Space = "my_graph"
//
//	// 2. 创建驱动器
//	driver, err := nebula.NewNebulaDriver(config)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// 3. 连接
//	ctx := context.Background()
//	if err := driver.Connect(ctx); err != nil {
//	    log.Fatal(err)
//	}
//	defer driver.Close()
//
//	// 4. 添加节点
//	node := &graphdb.Node{
//	    ID:    "person-1",
//	    Type:  "Person",
//	    Label: "John Smith",
//	    Properties: map[string]interface{}{
//	        "age":  30,
//	        "city": "San Francisco",
//	    },
//	}
//	if err := driver.AddNode(ctx, node); err != nil {
//	    log.Fatal(err)
//	}
//
//	// 5. 添加边
//	edge := &graphdb.Edge{
//	    Source:   "person-1",
//	    Target:   "org-1",
//	    Type:     "WORKS_FOR",
//	    Directed: true,
//	    Properties: map[string]interface{}{
//	        "since": 2020,
//	    },
//	}
//	if err := driver.AddEdge(ctx, edge); err != nil {
//	    log.Fatal(err)
//	}
//
// ## 图遍历
//
//	// BFS 遍历
//	result, err := driver.Traverse(ctx, "person-1", graphdb.TraverseOptions{
//	    MaxDepth:  3,
//	    Strategy:  graphdb.StrategyBFS,
//	    Direction: graphdb.DirectionBoth,
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	fmt.Printf("Found %d nodes, %d edges\n",
//	    len(result.Nodes), len(result.Edges))
//
// ## 最短路径
//
//	path, err := driver.ShortestPath(ctx, "person-1", "org-1", graphdb.PathOptions{
//	    MaxDepth: 10,
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	fmt.Printf("Path length: %d\n", path.Length)
//
// ## 原生 nGQL 查询
//
//	query := "MATCH (v:Person) WHERE v.Person.age > 30 RETURN v"
//	result, err := driver.ExecuteQuery(ctx, query)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// # 配置选项
//
// ## 连接配置
//
//	config := nebula.Config{
//	    Addresses: []string{
//	        "127.0.0.1:9669",
//	        "127.0.0.1:9670",  // 多个地址用于高可用
//	    },
//	    Username:        "root",
//	    Password:        "nebula",
//	    Space:           "my_graph",
//	    Timeout:         30 * time.Second,
//	    MaxConnPoolSize: 100,
//	    MinConnPoolSize: 10,
//	}
//
// ## 使用链式调用
//
//	config := nebula.DefaultConfig().
//	    WithSpace("my_graph").
//	    WithAddresses([]string{"192.168.1.100:9669"}).
//	    WithPoolSize(10, 50).
//	    WithTimeout(60 * time.Second)
//
// # 性能优化
//
// ## 1. 连接池大小
//
//	// 根据并发需求调整
//	config.MaxConnPoolSize = 200  // 高并发场景
//	config.MinConnPoolSize = 20
//
// ## 2. 批量操作
//
//	// 使用批量插入提高性能
//	for _, node := range nodes {
//	    driver.AddNode(ctx, node)
//	}
//
// ## 3. 查询优化
//
//	// 限制遍历深度
//	opts := graphdb.TraverseOptions{
//	    MaxDepth: 3,  // 不要设置过大
//	    Limit:    100, // 限制结果数量
//	}
//
// # 与 Neo4j 对比
//
// | 特性 | NebulaGraph | Neo4j |
// |------|------------|-------|
// | 架构 | 分布式原生 | 单机/集群 |
// | 扩展性 | 水平扩展 | 垂直为主 |
// | 节点规模 | 千亿级 | 十亿级 |
// | 查询语言 | nGQL | Cypher |
// | 适用场景 | 超大规模 | 中等规模 |
//
// # 注意事项
//
// ## 1. 图空间
//
// NebulaGraph 使用图空间（Space）来隔离不同的图：
//   - 必须先创建图空间
//   - 需要定义 Tag（节点类型）和 Edge Type（边类型）
//
// ## 2. Schema
//
// NebulaGraph 是强 Schema 的：
//   - 必须先定义 Tag 和 Edge Type
//   - 属性必须先声明
//
// ## 3. 边的标识
//
// NebulaGraph 的边通过 (源节点, 边类型, 目标节点) 唯一标识：
//   - 没有独立的边 ID
//   - 删除/查询边需要指定源、目标和类型
//
// # Docker 部署
//
//	# 使用 Docker Compose
//	docker compose -f docker-compose.graphdb.yml up -d nebula
//
//	# 创建图空间
//	docker exec -it nebula-console nebula-console -addr graphd -port 9669 -u root -p nebula
//	CREATE SPACE langchain(partition_num=10, replica_factor=1);
//	USE langchain;
//	CREATE TAG Person(name string, age int);
//	CREATE EDGE WORKS_FOR(since int);
//
// # 故障排除
//
// ## 连接失败
//
//	// 检查服务状态
//	docker ps | grep nebula
//
//	// 查看日志
//	docker logs nebula-graphd
//
// ## Schema 错误
//
//	// 创建 Tag
//	CREATE TAG IF NOT EXISTS Person(name string, age int);
//
//	// 创建 Edge Type
//	CREATE EDGE IF NOT EXISTS WORKS_FOR(since int);
//
// # 参考资料
//
//   - NebulaGraph 官方文档: https://docs.nebula-graph.io/
//   - nGQL 参考: https://docs.nebula-graph.io/3.0.0/3.ngql-guide/
//   - Go 客户端: https://github.com/vesoft-inc/nebula-go
//
package nebula
