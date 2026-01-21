package nebula

import (
	"context"
	"testing"
	"time"

	"github.com/zhucl121/langchain-go/retrieval/graphdb"
)

// TestOptimizations_GetNode 测试优化后的 GetNode
func TestOptimizations_GetNode(t *testing.T) {
	if testing.Short() {
		t.Skip("Integration test - requires NebulaGraph instance")
	}

	ctx := context.Background()

	// 创建驱动器
	config := DefaultConfig().
		WithSpace("langchain_test").
		WithAddresses([]string{"127.0.0.1:9669"}).
		WithTimeout(30 * time.Second)

	driver, err := NewNebulaDriver(config)
	if err != nil {
		t.Fatalf("Failed to create driver: %v", err)
	}

	// 连接
	err = driver.Connect(ctx)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer driver.Close()

	// 添加测试节点
	testNode := &graphdb.Node{
		ID:    "test_opt_person_1",
		Type:  "Person",
		Label: "Alice",
		Properties: map[string]interface{}{
			"name": "Alice",
			"age":  30,
			"city": "Shanghai",
		},
	}

	err = driver.AddNode(ctx, testNode)
	if err != nil {
		t.Fatalf("Failed to add node: %v", err)
	}

	// 等待数据生效
	time.Sleep(100 * time.Millisecond)

	// 获取节点
	retrievedNode, err := driver.GetNode(ctx, "test_opt_person_1")
	if err != nil {
		t.Fatalf("Failed to get node: %v", err)
	}

	// 验证结果
	t.Logf("Retrieved node: ID=%s, Type=%s, Label=%s", retrievedNode.ID, retrievedNode.Type, retrievedNode.Label)

	if retrievedNode.ID != "test_opt_person_1" {
		t.Errorf("Expected ID 'test_opt_person_1', got '%s'", retrievedNode.ID)
	}

	if retrievedNode.Type != "Person" {
		t.Errorf("Expected Type 'Person', got '%s'", retrievedNode.Type)
	}

	if retrievedNode.Label != "Alice" {
		t.Errorf("Expected Label 'Alice', got '%s'", retrievedNode.Label)
	}

	// 验证属性
	if name, ok := retrievedNode.Properties["name"].(string); !ok || name != "Alice" {
		t.Errorf("Expected name 'Alice', got '%v'", retrievedNode.Properties["name"])
	}

	if age, ok := retrievedNode.Properties["age"].(int64); !ok || age != 30 {
		t.Errorf("Expected age 30, got '%v'", retrievedNode.Properties["age"])
	}

	// 清理
	driver.DeleteNode(ctx, "test_opt_person_1")
}

// TestOptimizations_GetEdge 测试优化后的 GetEdge
func TestOptimizations_GetEdge(t *testing.T) {
	if testing.Short() {
		t.Skip("Integration test - requires NebulaGraph instance")
	}

	ctx := context.Background()

	// 创建驱动器
	config := DefaultConfig().
		WithSpace("langchain_test").
		WithAddresses([]string{"127.0.0.1:9669"}).
		WithTimeout(30 * time.Second)

	driver, err := NewNebulaDriver(config)
	if err != nil {
		t.Fatalf("Failed to create driver: %v", err)
	}

	// 连接
	err = driver.Connect(ctx)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer driver.Close()

	// 添加测试节点
	node1 := &graphdb.Node{
		ID:    "test_opt_person_2",
		Type:  "Person",
		Label: "Bob",
		Properties: map[string]interface{}{
			"name": "Bob",
		},
	}
	node2 := &graphdb.Node{
		ID:    "test_opt_person_3",
		Type:  "Person",
		Label: "Carol",
		Properties: map[string]interface{}{
			"name": "Carol",
		},
	}

	driver.AddNode(ctx, node1)
	driver.AddNode(ctx, node2)
	time.Sleep(100 * time.Millisecond)

	// 添加测试边
	testEdge := &graphdb.Edge{
		Source: "test_opt_person_2",
		Target: "test_opt_person_3",
		Type:   "KNOWS",
		Properties: map[string]interface{}{
			"since": 2020,
		},
	}

	err = driver.AddEdge(ctx, testEdge)
	if err != nil {
		t.Fatalf("Failed to add edge: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	// 获取边 (ID 格式: source-type-target)
	edgeID := "test_opt_person_2-KNOWS-test_opt_person_3"
	retrievedEdge, err := driver.GetEdge(ctx, edgeID)
	if err != nil {
		t.Fatalf("Failed to get edge: %v", err)
	}

	// 验证结果
	t.Logf("Retrieved edge: ID=%s, Type=%s, Source=%s, Target=%s",
		retrievedEdge.ID, retrievedEdge.Type, retrievedEdge.Source, retrievedEdge.Target)

	if retrievedEdge.Source != "test_opt_person_2" {
		t.Errorf("Expected Source 'test_opt_person_2', got '%s'", retrievedEdge.Source)
	}

	if retrievedEdge.Target != "test_opt_person_3" {
		t.Errorf("Expected Target 'test_opt_person_3', got '%s'", retrievedEdge.Target)
	}

	if retrievedEdge.Type != "KNOWS" {
		t.Errorf("Expected Type 'KNOWS', got '%s'", retrievedEdge.Type)
	}

	// 验证属性
	if since, ok := retrievedEdge.Properties["since"].(int64); !ok || since != 2020 {
		t.Errorf("Expected since 2020, got '%v'", retrievedEdge.Properties["since"])
	}

	// 清理
	driver.DeleteEdgeByEndpoints(ctx, "test_opt_person_2", "test_opt_person_3", "KNOWS")
	driver.DeleteNode(ctx, "test_opt_person_2")
	driver.DeleteNode(ctx, "test_opt_person_3")
}

// TestOptimizations_Traverse 测试优化后的 Traverse
func TestOptimizations_Traverse(t *testing.T) {
	if testing.Short() {
		t.Skip("Integration test - requires NebulaGraph instance")
	}

	ctx := context.Background()

	// 创建驱动器
	config := DefaultConfig().
		WithSpace("langchain_test").
		WithAddresses([]string{"127.0.0.1:9669"}).
		WithTimeout(30 * time.Second)

	driver, err := NewNebulaDriver(config)
	if err != nil {
		t.Fatalf("Failed to create driver: %v", err)
	}

	// 连接
	err = driver.Connect(ctx)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer driver.Close()

	// 创建测试图：A -> B -> C
	nodes := []*graphdb.Node{
		{
			ID:    "test_traverse_a",
			Type:  "Person",
			Label: "A",
			Properties: map[string]interface{}{"name": "A"},
		},
		{
			ID:    "test_traverse_b",
			Type:  "Person",
			Label: "B",
			Properties: map[string]interface{}{"name": "B"},
		},
		{
			ID:    "test_traverse_c",
			Type:  "Person",
			Label: "C",
			Properties: map[string]interface{}{"name": "C"},
		},
	}

	for _, node := range nodes {
		driver.AddNode(ctx, node)
	}

	time.Sleep(100 * time.Millisecond)

	edges := []*graphdb.Edge{
		{Source: "test_traverse_a", Target: "test_traverse_b", Type: "KNOWS"},
		{Source: "test_traverse_b", Target: "test_traverse_c", Type: "KNOWS"},
	}

	for _, edge := range edges {
		driver.AddEdge(ctx, edge)
	}

	time.Sleep(100 * time.Millisecond)

	// 执行遍历
	result, err := driver.Traverse(ctx, "test_traverse_a", graphdb.TraverseOptions{
		MaxDepth:  2,
		Direction: graphdb.DirectionOutbound,
	})
	if err != nil {
		t.Fatalf("Failed to traverse: %v", err)
	}

	t.Logf("Traverse result: %d nodes, %d edges, %d paths",
		len(result.Nodes), len(result.Edges), len(result.Paths))

	// 验证结果
	if len(result.Nodes) < 2 {
		t.Errorf("Expected at least 2 nodes, got %d", len(result.Nodes))
	}

	if len(result.Edges) < 1 {
		t.Errorf("Expected at least 1 edge, got %d", len(result.Edges))
	}

	// 验证节点有正确的 Type 和 Label
	for _, node := range result.Nodes {
		if node.Type == "" {
			t.Errorf("Node %s has empty Type", node.ID)
		}
		t.Logf("  Node: ID=%s, Type=%s, Label=%s", node.ID, node.Type, node.Label)
	}

	// 清理
	for _, edge := range edges {
		driver.DeleteEdgeByEndpoints(ctx, edge.Source, edge.Target, edge.Type)
	}
	for _, node := range nodes {
		driver.DeleteNode(ctx, node.ID)
	}
}

// TestOptimizations_ShortestPath 测试优化后的 ShortestPath
func TestOptimizations_ShortestPath(t *testing.T) {
	if testing.Short() {
		t.Skip("Integration test - requires NebulaGraph instance")
	}

	ctx := context.Background()

	// 创建驱动器
	config := DefaultConfig().
		WithSpace("langchain_test").
		WithAddresses([]string{"127.0.0.1:9669"}).
		WithTimeout(30 * time.Second)

	driver, err := NewNebulaDriver(config)
	if err != nil {
		t.Fatalf("Failed to create driver: %v", err)
	}

	// 连接
	err = driver.Connect(ctx)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer driver.Close()

	// 创建测试图：X -> Y -> Z
	nodes := []*graphdb.Node{
		{
			ID:    "test_path_x",
			Type:  "Person",
			Label: "X",
			Properties: map[string]interface{}{"name": "X"},
		},
		{
			ID:    "test_path_y",
			Type:  "Person",
			Label: "Y",
			Properties: map[string]interface{}{"name": "Y"},
		},
		{
			ID:    "test_path_z",
			Type:  "Person",
			Label: "Z",
			Properties: map[string]interface{}{"name": "Z"},
		},
	}

	for _, node := range nodes {
		driver.AddNode(ctx, node)
	}

	time.Sleep(100 * time.Millisecond)

	edges := []*graphdb.Edge{
		{Source: "test_path_x", Target: "test_path_y", Type: "KNOWS"},
		{Source: "test_path_y", Target: "test_path_z", Type: "KNOWS"},
	}

	for _, edge := range edges {
		driver.AddEdge(ctx, edge)
	}

	time.Sleep(100 * time.Millisecond)

	// 查找最短路径
	path, err := driver.ShortestPath(ctx, "test_path_x", "test_path_z", graphdb.PathOptions{
		MaxDepth: 5,
	})
	if err != nil {
		t.Fatalf("Failed to find shortest path: %v", err)
	}

	t.Logf("Shortest path: %d nodes, %d edges, length=%d",
		len(path.Nodes), len(path.Edges), path.Length)

	// 验证结果
	if len(path.Nodes) < 2 {
		t.Errorf("Expected at least 2 nodes in path, got %d", len(path.Nodes))
	}

	// 验证节点有正确的 Type 和 Label
	for _, node := range path.Nodes {
		if node.Type == "" {
			t.Errorf("Node %s in path has empty Type", node.ID)
		}
		t.Logf("  Path Node: ID=%s, Type=%s, Label=%s", node.ID, node.Type, node.Label)
	}

	// 清理
	for _, edge := range edges {
		driver.DeleteEdgeByEndpoints(ctx, edge.Source, edge.Target, edge.Type)
	}
	for _, node := range nodes {
		driver.DeleteNode(ctx, node.ID)
	}
}
