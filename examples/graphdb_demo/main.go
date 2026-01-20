package main

import (
	"context"
	"fmt"
	"log"

	"github.com/zhucl121/langchain-go/retrieval/graphdb"
	"github.com/zhucl121/langchain-go/retrieval/graphdb/mock"
)

func main() {
	fmt.Println("=== LangChain-Go GraphDB Demo ===\n")

	// 使用 Mock 实现演示
	// 生产环境可以替换为 Neo4j 或 NebulaGraph
	db := mock.NewMockGraphDB()

	ctx := context.Background()

	// 1. 连接数据库
	fmt.Println("1. 连接数据库...")
	if err := db.Connect(ctx); err != nil {
		log.Fatalf("连接失败: %v", err)
	}
	defer db.Close()

	// 健康检查
	if err := db.Ping(ctx); err != nil {
		log.Fatalf("健康检查失败: %v", err)
	}
	fmt.Println("✓ 连接成功\n")

	// 2. 创建知识图谱
	fmt.Println("2. 创建知识图谱...")
	if err := createKnowledgeGraph(ctx, db); err != nil {
		log.Fatalf("创建知识图谱失败: %v", err)
	}
	fmt.Println("✓ 知识图谱创建成功\n")

	// 3. 查询节点
	fmt.Println("3. 查询节点...")
	if err := queryNodes(ctx, db); err != nil {
		log.Fatalf("查询节点失败: %v", err)
	}
	fmt.Println()

	// 4. 图遍历
	fmt.Println("4. 图遍历...")
	if err := traverseGraph(ctx, db); err != nil {
		log.Fatalf("图遍历失败: %v", err)
	}
	fmt.Println()

	// 5. 最短路径
	fmt.Println("5. 查找最短路径...")
	if err := findShortestPath(ctx, db); err != nil {
		log.Fatalf("查找最短路径失败: %v", err)
	}
	fmt.Println()

	fmt.Println("=== Demo 完成 ===")
}

// createKnowledgeGraph 创建示例知识图谱
func createKnowledgeGraph(ctx context.Context, db graphdb.GraphDB) error {
	// 创建人物节点
	people := []*graphdb.Node{
		{
			ID:    "person-alice",
			Type:  "person",
			Label: "Alice",
			Properties: map[string]interface{}{
				"age":        30,
				"city":       "Beijing",
				"occupation": "Engineer",
			},
		},
		{
			ID:    "person-bob",
			Type:  "person",
			Label: "Bob",
			Properties: map[string]interface{}{
				"age":        28,
				"city":       "Shanghai",
				"occupation": "Designer",
			},
		},
		{
			ID:    "person-charlie",
			Type:  "person",
			Label: "Charlie",
			Properties: map[string]interface{}{
				"age":        35,
				"city":       "Shenzhen",
				"occupation": "Manager",
			},
		},
		{
			ID:    "person-david",
			Type:  "person",
			Label: "David",
			Properties: map[string]interface{}{
				"age":        32,
				"city":       "Hangzhou",
				"occupation": "Developer",
			},
		},
	}

	// 批量添加人物节点
	if err := db.BatchAddNodes(ctx, people); err != nil {
		return fmt.Errorf("添加人物节点失败: %w", err)
	}
	fmt.Printf("  - 添加了 %d 个人物节点\n", len(people))

	// 创建公司节点
	companies := []*graphdb.Node{
		{
			ID:    "company-techcorp",
			Type:  "organization",
			Label: "TechCorp",
			Properties: map[string]interface{}{
				"industry":    "Technology",
				"size":        500,
				"founded":     2010,
				"location":    "Beijing",
			},
		},
		{
			ID:    "company-designco",
			Type:  "organization",
			Label: "DesignCo",
			Properties: map[string]interface{}{
				"industry":    "Design",
				"size":        200,
				"founded":     2015,
				"location":    "Shanghai",
			},
		},
	}

	if err := db.BatchAddNodes(ctx, companies); err != nil {
		return fmt.Errorf("添加公司节点失败: %w", err)
	}
	fmt.Printf("  - 添加了 %d 个公司节点\n", len(companies))

	// 创建关系（边）
	relationships := []*graphdb.Edge{
		// 人际关系
		{
			ID:       "edge-alice-knows-bob",
			Source:   "person-alice",
			Target:   "person-bob",
			Type:     "knows",
			Label:    "认识",
			Directed: false,
			Weight:   1.0,
			Properties: map[string]interface{}{
				"since": 2020,
				"relationship": "colleague",
			},
		},
		{
			ID:       "edge-bob-knows-charlie",
			Source:   "person-bob",
			Target:   "person-charlie",
			Type:     "knows",
			Label:    "认识",
			Directed: false,
			Weight:   1.0,
			Properties: map[string]interface{}{
				"since": 2019,
				"relationship": "friend",
			},
		},
		{
			ID:       "edge-charlie-knows-david",
			Source:   "person-charlie",
			Target:   "person-david",
			Type:     "knows",
			Label:    "认识",
			Directed: false,
			Weight:   1.0,
			Properties: map[string]interface{}{
				"since": 2021,
				"relationship": "friend",
			},
		},
		// 工作关系
		{
			ID:       "edge-alice-works-for-techcorp",
			Source:   "person-alice",
			Target:   "company-techcorp",
			Type:     "works_for",
			Label:    "工作于",
			Directed: true,
			Weight:   1.0,
			Properties: map[string]interface{}{
				"start_date": "2018-01-01",
				"position":   "Senior Engineer",
			},
		},
		{
			ID:       "edge-bob-works-for-designco",
			Source:   "person-bob",
			Target:   "company-designco",
			Type:     "works_for",
			Label:    "工作于",
			Directed: true,
			Weight:   1.0,
			Properties: map[string]interface{}{
				"start_date": "2019-06-01",
				"position":   "Lead Designer",
			},
		},
		{
			ID:       "edge-charlie-manages-alice",
			Source:   "person-charlie",
			Target:   "person-alice",
			Type:     "manages",
			Label:    "管理",
			Directed: true,
			Weight:   1.0,
			Properties: map[string]interface{}{
				"since": "2020-01-01",
			},
		},
	}

	if err := db.BatchAddEdges(ctx, relationships); err != nil {
		return fmt.Errorf("添加关系失败: %w", err)
	}
	fmt.Printf("  - 添加了 %d 条关系\n", len(relationships))

	return nil
}

// queryNodes 查询节点示例
func queryNodes(ctx context.Context, db graphdb.GraphDB) error {
	// 查询所有人物节点
	people, err := db.FindNodes(ctx, graphdb.NodeFilter{
		Types: []string{"person"},
	})
	if err != nil {
		return err
	}
	fmt.Printf("  找到 %d 个人物:\n", len(people))
	for _, person := range people {
		age := person.Properties["age"]
		city := person.Properties["city"]
		fmt.Printf("    - %s (年龄: %v, 城市: %v)\n", person.Label, age, city)
	}

	// 查询特定城市的人
	beijingPeople, err := db.FindNodes(ctx, graphdb.NodeFilter{
		Types: []string{"person"},
		Properties: map[string]interface{}{
			"city": "Beijing",
		},
	})
	if err != nil {
		return err
	}
	fmt.Printf("\n  北京的人物:\n")
	for _, person := range beijingPeople {
		fmt.Printf("    - %s\n", person.Label)
	}

	return nil
}

// traverseGraph 图遍历示例
func traverseGraph(ctx context.Context, db graphdb.GraphDB) error {
	// 从 Alice 开始遍历，深度为 2
	result, err := db.Traverse(ctx, "person-alice", graphdb.TraverseOptions{
		MaxDepth:  2,
		Direction: graphdb.DirectionBoth,
		Strategy:  graphdb.StrategyBFS,
		Limit:     10,
	})
	if err != nil {
		return err
	}

	fmt.Printf("  从 Alice 开始遍历 (深度=2):\n")
	fmt.Printf("    发现 %d 个节点:\n", len(result.Nodes))
	for _, node := range result.Nodes {
		fmt.Printf("      - %s (%s)\n", node.Label, node.Type)
	}
	fmt.Printf("    发现 %d 条边:\n", len(result.Edges))
	for _, edge := range result.Edges {
		srcNode, _ := db.GetNode(ctx, edge.Source)
		tgtNode, _ := db.GetNode(ctx, edge.Target)
		fmt.Printf("      - %s -[%s]-> %s\n", srcNode.Label, edge.Type, tgtNode.Label)
	}

	return nil
}

// findShortestPath 最短路径示例
func findShortestPath(ctx context.Context, db graphdb.GraphDB) error {
	// 查找 Alice 到 David 的最短路径
	path, err := db.ShortestPath(ctx, "person-alice", "person-david", graphdb.PathOptions{
		MaxDepth:  5,
		Algorithm: graphdb.AlgorithmBFS,
	})
	if err != nil {
		return err
	}

	fmt.Printf("  Alice 到 David 的最短路径:\n")
	fmt.Printf("    路径长度: %d\n", path.Length)
	fmt.Printf("    路径成本: %.2f\n", path.Cost)
	fmt.Printf("    路径节点:\n")
	for i, node := range path.Nodes {
		fmt.Printf("      %d. %s (%s)\n", i+1, node.Label, node.Type)
	}

	return nil
}
