package graphdb_test

import (
	"context"
	"testing"

	"github.com/zhucl121/langchain-go/retrieval/graphdb"
	"github.com/zhucl121/langchain-go/retrieval/graphdb/mock"
)

func TestGraphDB_BasicOperations(t *testing.T) {
	ctx := context.Background()
	db := mock.NewMockGraphDB()

	// 连接
	if err := db.Connect(ctx); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	defer db.Close()

	// Ping
	if err := db.Ping(ctx); err != nil {
		t.Fatalf("Ping failed: %v", err)
	}

	// 添加节点
	node1 := &graphdb.Node{
		ID:    "person-1",
		Type:  "person",
		Label: "Alice",
		Properties: map[string]interface{}{
			"age":  30,
			"city": "Beijing",
		},
	}

	if err := db.AddNode(ctx, node1); err != nil {
		t.Fatalf("AddNode failed: %v", err)
	}

	// 获取节点
	retrieved, err := db.GetNode(ctx, "person-1")
	if err != nil {
		t.Fatalf("GetNode failed: %v", err)
	}

	if retrieved.ID != node1.ID || retrieved.Label != node1.Label {
		t.Errorf("Retrieved node mismatch: got %+v, want %+v", retrieved, node1)
	}

	// 更新节点
	node1.Properties["age"] = 31
	if err := db.UpdateNode(ctx, node1); err != nil {
		t.Fatalf("UpdateNode failed: %v", err)
	}

	// 删除节点
	if err := db.DeleteNode(ctx, "person-1"); err != nil {
		t.Fatalf("DeleteNode failed: %v", err)
	}

	// 验证删除
	_, err = db.GetNode(ctx, "person-1")
	if err != graphdb.ErrNodeNotFound {
		t.Errorf("Expected ErrNodeNotFound, got %v", err)
	}
}

func TestGraphDB_EdgeOperations(t *testing.T) {
	ctx := context.Background()
	db := mock.NewMockGraphDB()

	if err := db.Connect(ctx); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	defer db.Close()

	// 添加节点
	node1 := &graphdb.Node{ID: "person-1", Type: "person", Label: "Alice"}
	node2 := &graphdb.Node{ID: "person-2", Type: "person", Label: "Bob"}

	if err := db.AddNode(ctx, node1); err != nil {
		t.Fatalf("AddNode failed: %v", err)
	}
	if err := db.AddNode(ctx, node2); err != nil {
		t.Fatalf("AddNode failed: %v", err)
	}

	// 添加边
	edge := &graphdb.Edge{
		ID:       "edge-1",
		Source:   "person-1",
		Target:   "person-2",
		Type:     "knows",
		Label:    "认识",
		Directed: true,
		Weight:   1.0,
	}

	if err := db.AddEdge(ctx, edge); err != nil {
		t.Fatalf("AddEdge failed: %v", err)
	}

	// 获取边
	retrieved, err := db.GetEdge(ctx, "edge-1")
	if err != nil {
		t.Fatalf("GetEdge failed: %v", err)
	}

	if retrieved.Source != edge.Source || retrieved.Target != edge.Target {
		t.Errorf("Retrieved edge mismatch: got %+v, want %+v", retrieved, edge)
	}

	// 删除边
	if err := db.DeleteEdge(ctx, "edge-1"); err != nil {
		t.Fatalf("DeleteEdge failed: %v", err)
	}
}

func TestGraphDB_BatchOperations(t *testing.T) {
	ctx := context.Background()
	db := mock.NewMockGraphDB()

	if err := db.Connect(ctx); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	defer db.Close()

	// 批量添加节点
	nodes := []*graphdb.Node{
		{ID: "person-1", Type: "person", Label: "Alice"},
		{ID: "person-2", Type: "person", Label: "Bob"},
		{ID: "person-3", Type: "person", Label: "Charlie"},
	}

	if err := db.BatchAddNodes(ctx, nodes); err != nil {
		t.Fatalf("BatchAddNodes failed: %v", err)
	}

	// 批量添加边
	edges := []*graphdb.Edge{
		{ID: "edge-1", Source: "person-1", Target: "person-2", Type: "knows", Directed: true},
		{ID: "edge-2", Source: "person-2", Target: "person-3", Type: "knows", Directed: true},
	}

	if err := db.BatchAddEdges(ctx, edges); err != nil {
		t.Fatalf("BatchAddEdges failed: %v", err)
	}

	// 验证
	for _, node := range nodes {
		if _, err := db.GetNode(ctx, node.ID); err != nil {
			t.Errorf("Node %s not found: %v", node.ID, err)
		}
	}
}

func TestGraphDB_FindNodes(t *testing.T) {
	ctx := context.Background()
	db := mock.NewMockGraphDB()

	if err := db.Connect(ctx); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	defer db.Close()

	// 添加测试数据
	nodes := []*graphdb.Node{
		{ID: "person-1", Type: "person", Label: "Alice", Properties: map[string]interface{}{"city": "Beijing"}},
		{ID: "person-2", Type: "person", Label: "Bob", Properties: map[string]interface{}{"city": "Shanghai"}},
		{ID: "org-1", Type: "organization", Label: "Company A"},
	}

	if err := db.BatchAddNodes(ctx, nodes); err != nil {
		t.Fatalf("BatchAddNodes failed: %v", err)
	}

	// 按类型查找
	results, err := db.FindNodes(ctx, graphdb.NodeFilter{
		Types: []string{"person"},
	})
	if err != nil {
		t.Fatalf("FindNodes failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 person nodes, got %d", len(results))
	}

	// 按属性查找
	results, err = db.FindNodes(ctx, graphdb.NodeFilter{
		Properties: map[string]interface{}{"city": "Beijing"},
	})
	if err != nil {
		t.Fatalf("FindNodes failed: %v", err)
	}

	if len(results) != 1 || results[0].ID != "person-1" {
		t.Errorf("Property filter failed: got %+v", results)
	}
}

func TestGraphDB_Traverse(t *testing.T) {
	ctx := context.Background()
	db := mock.NewMockGraphDB()

	if err := db.Connect(ctx); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	defer db.Close()

	// 构建测试图: A -> B -> C
	nodes := []*graphdb.Node{
		{ID: "A", Type: "person", Label: "Alice"},
		{ID: "B", Type: "person", Label: "Bob"},
		{ID: "C", Type: "person", Label: "Charlie"},
	}
	if err := db.BatchAddNodes(ctx, nodes); err != nil {
		t.Fatalf("BatchAddNodes failed: %v", err)
	}

	edges := []*graphdb.Edge{
		{ID: "edge-1", Source: "A", Target: "B", Type: "knows", Directed: true},
		{ID: "edge-2", Source: "B", Target: "C", Type: "knows", Directed: true},
	}
	if err := db.BatchAddEdges(ctx, edges); err != nil {
		t.Fatalf("BatchAddEdges failed: %v", err)
	}

	// 遍历深度 1
	result, err := db.Traverse(ctx, "A", graphdb.TraverseOptions{
		MaxDepth:  1,
		Direction: graphdb.DirectionOutbound,
		Strategy:  graphdb.StrategyBFS,
	})
	if err != nil {
		t.Fatalf("Traverse failed: %v", err)
	}

	if len(result.Nodes) != 2 { // A, B
		t.Errorf("Expected 2 nodes at depth 1, got %d", len(result.Nodes))
	}

	// 遍历深度 2
	result, err = db.Traverse(ctx, "A", graphdb.TraverseOptions{
		MaxDepth:  2,
		Direction: graphdb.DirectionOutbound,
		Strategy:  graphdb.StrategyBFS,
	})
	if err != nil {
		t.Fatalf("Traverse failed: %v", err)
	}

	if len(result.Nodes) != 3 { // A, B, C
		t.Errorf("Expected 3 nodes at depth 2, got %d", len(result.Nodes))
	}
}

func TestGraphDB_ShortestPath(t *testing.T) {
	ctx := context.Background()
	db := mock.NewMockGraphDB()

	if err := db.Connect(ctx); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	defer db.Close()

	// 构建测试图: A -> B -> C
	nodes := []*graphdb.Node{
		{ID: "A", Type: "person", Label: "Alice"},
		{ID: "B", Type: "person", Label: "Bob"},
		{ID: "C", Type: "person", Label: "Charlie"},
	}
	if err := db.BatchAddNodes(ctx, nodes); err != nil {
		t.Fatalf("BatchAddNodes failed: %v", err)
	}

	edges := []*graphdb.Edge{
		{ID: "edge-1", Source: "A", Target: "B", Type: "knows", Directed: true, Weight: 1.0},
		{ID: "edge-2", Source: "B", Target: "C", Type: "knows", Directed: true, Weight: 1.0},
	}
	if err := db.BatchAddEdges(ctx, edges); err != nil {
		t.Fatalf("BatchAddEdges failed: %v", err)
	}

	// 查找最短路径 A -> C
	path, err := db.ShortestPath(ctx, "A", "C", graphdb.PathOptions{
		MaxDepth:  5,
		Algorithm: graphdb.AlgorithmBFS,
	})
	if err != nil {
		t.Fatalf("ShortestPath failed: %v", err)
	}

	if path.Length != 2 {
		t.Errorf("Expected path length 2, got %d", path.Length)
	}

	if len(path.Nodes) != 3 { // A, B, C
		t.Errorf("Expected 3 nodes in path, got %d", len(path.Nodes))
	}

	// 测试不存在的路径
	db.AddNode(ctx, &graphdb.Node{ID: "D", Type: "person", Label: "David"})
	_, err = db.ShortestPath(ctx, "A", "D", graphdb.PathOptions{
		MaxDepth:  5,
		Algorithm: graphdb.AlgorithmBFS,
	})
	if err != graphdb.ErrNoPathFound {
		t.Errorf("Expected ErrNoPathFound, got %v", err)
	}
}

func TestGraphDB_ErrorHandling(t *testing.T) {
	ctx := context.Background()
	db := mock.NewMockGraphDB()

	// 未连接时操作
	err := db.AddNode(ctx, &graphdb.Node{ID: "test", Type: "test"})
	if err != graphdb.ErrNotConnected {
		t.Errorf("Expected ErrNotConnected, got %v", err)
	}

	// 连接后测试
	if err := db.Connect(ctx); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	defer db.Close()

	// 无效节点
	err = db.AddNode(ctx, nil)
	if err != graphdb.ErrInvalidNode {
		t.Errorf("Expected ErrInvalidNode, got %v", err)
	}

	err = db.AddNode(ctx, &graphdb.Node{Type: "test"}) // 缺少 ID
	if err != graphdb.ErrInvalidNode {
		t.Errorf("Expected ErrInvalidNode, got %v", err)
	}

	// 重复添加
	node := &graphdb.Node{ID: "test", Type: "test"}
	if err := db.AddNode(ctx, node); err != nil {
		t.Fatalf("AddNode failed: %v", err)
	}

	err = db.AddNode(ctx, node)
	if err != graphdb.ErrNodeExists {
		t.Errorf("Expected ErrNodeExists, got %v", err)
	}

	// 节点不存在
	_, err = db.GetNode(ctx, "nonexistent")
	if err != graphdb.ErrNodeNotFound {
		t.Errorf("Expected ErrNodeNotFound, got %v", err)
	}
}
