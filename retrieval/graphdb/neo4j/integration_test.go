// +build integration

package neo4j_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/zhucl121/langchain-go/retrieval/graphdb"
	"github.com/zhucl121/langchain-go/retrieval/graphdb/neo4j"
)

func getTestConfig() neo4j.Config {
	config := neo4j.DefaultConfig()
	config.URI = "bolt://localhost:7687"
	config.Username = "neo4j"
	config.Password = "password123"
	config.Database = "neo4j"
	return config
}

func setupTestDriver(t *testing.T) (*neo4j.Neo4jDriver, func()) {
	driver, err := neo4j.NewNeo4jDriver(getTestConfig())
	if err != nil {
		t.Fatalf("Failed to create driver: %v", err)
	}

	ctx := context.Background()
	if err := driver.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// 清理函数
	cleanup := func() {
		// 清理所有测试数据
		cleanupTestData(t, driver)
		driver.Close()
	}

	return driver, cleanup
}

func cleanupTestData(t *testing.T, driver *neo4j.Neo4jDriver) {
	ctx := context.Background()

	// 删除所有测试节点和边
	testIDs := []string{
		"test-person-1", "test-person-2", "test-person-3", "test-person-4",
		"test-org-1", "test-org-2",
	}

	for _, id := range testIDs {
		driver.DeleteNode(ctx, id)
	}
}

func TestNeo4j_ConnectionManagement(t *testing.T) {
	driver, cleanup := setupTestDriver(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("Ping", func(t *testing.T) {
		err := driver.Ping(ctx)
		if err != nil {
			t.Errorf("Ping failed: %v", err)
		}
	})

	t.Run("Close and Reconnect", func(t *testing.T) {
		// 关闭连接
		if err := driver.Close(); err != nil {
			t.Errorf("Close failed: %v", err)
		}

		// 重新连接
		if err := driver.Connect(ctx); err != nil {
			t.Errorf("Reconnect failed: %v", err)
		}

		// 验证连接
		if err := driver.Ping(ctx); err != nil {
			t.Errorf("Ping after reconnect failed: %v", err)
		}
	})
}

func TestNeo4j_NodeOperations(t *testing.T) {
	driver, cleanup := setupTestDriver(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("AddNode", func(t *testing.T) {
		node := &graphdb.Node{
			ID:    "test-person-1",
			Type:  "Person",
			Label: "Alice",
			Properties: map[string]interface{}{
				"age":  30,
				"city": "Beijing",
			},
		}

		err := driver.AddNode(ctx, node)
		if err != nil {
			t.Fatalf("AddNode failed: %v", err)
		}
	})

	t.Run("GetNode", func(t *testing.T) {
		node, err := driver.GetNode(ctx, "test-person-1")
		if err != nil {
			t.Fatalf("GetNode failed: %v", err)
		}

		if node.ID != "test-person-1" {
			t.Errorf("Node ID mismatch: got %s, want test-person-1", node.ID)
		}
		if node.Label != "Alice" {
			t.Errorf("Node Label mismatch: got %s, want Alice", node.Label)
		}
		if age, ok := node.Properties["age"].(int64); !ok || age != 30 {
			t.Errorf("Node age mismatch: got %v, want 30", node.Properties["age"])
		}
	})

	t.Run("UpdateNode", func(t *testing.T) {
		node := &graphdb.Node{
			ID: "test-person-1",
			Properties: map[string]interface{}{
				"age": 31,
			},
		}

		err := driver.UpdateNode(ctx, node)
		if err != nil {
			t.Fatalf("UpdateNode failed: %v", err)
		}

		// 验证更新
		updated, err := driver.GetNode(ctx, "test-person-1")
		if err != nil {
			t.Fatalf("GetNode after update failed: %v", err)
		}

		if age, ok := updated.Properties["age"].(int64); !ok || age != 31 {
			t.Errorf("Node age not updated: got %v, want 31", updated.Properties["age"])
		}
	})

	t.Run("DeleteNode", func(t *testing.T) {
		err := driver.DeleteNode(ctx, "test-person-1")
		if err != nil {
			t.Fatalf("DeleteNode failed: %v", err)
		}

		// 验证删除
		_, err = driver.GetNode(ctx, "test-person-1")
		if err != graphdb.ErrNodeNotFound {
			t.Errorf("Expected ErrNodeNotFound, got %v", err)
		}
	})
}

func TestNeo4j_BatchNodeOperations(t *testing.T) {
	driver, cleanup := setupTestDriver(t)
	defer cleanup()

	ctx := context.Background()

	nodes := []*graphdb.Node{
		{
			ID:    "test-person-1",
			Type:  "Person",
			Label: "Alice",
			Properties: map[string]interface{}{
				"age": 30,
			},
		},
		{
			ID:    "test-person-2",
			Type:  "Person",
			Label: "Bob",
			Properties: map[string]interface{}{
				"age": 28,
			},
		},
		{
			ID:    "test-person-3",
			Type:  "Person",
			Label: "Charlie",
			Properties: map[string]interface{}{
				"age": 35,
			},
		},
	}

	t.Run("BatchAddNodes", func(t *testing.T) {
		err := driver.BatchAddNodes(ctx, nodes)
		if err != nil {
			t.Fatalf("BatchAddNodes failed: %v", err)
		}

		// 验证所有节点都已添加
		for _, node := range nodes {
			retrieved, err := driver.GetNode(ctx, node.ID)
			if err != nil {
				t.Errorf("Node %s not found after batch add: %v", node.ID, err)
			}
			if retrieved.Label != node.Label {
				t.Errorf("Node %s label mismatch: got %s, want %s", node.ID, retrieved.Label, node.Label)
			}
		}
	})
}

func TestNeo4j_EdgeOperations(t *testing.T) {
	driver, cleanup := setupTestDriver(t)
	defer cleanup()

	ctx := context.Background()

	// 先创建两个节点
	node1 := &graphdb.Node{
		ID:    "test-person-1",
		Type:  "Person",
		Label: "Alice",
	}
	node2 := &graphdb.Node{
		ID:    "test-person-2",
		Type:  "Person",
		Label: "Bob",
	}

	driver.AddNode(ctx, node1)
	driver.AddNode(ctx, node2)

	t.Run("AddEdge", func(t *testing.T) {
		edge := &graphdb.Edge{
			ID:       "test-edge-1",
			Source:   "test-person-1",
			Target:   "test-person-2",
			Type:     "KNOWS",
			Label:    "认识",
			Directed: true,
			Weight:   1.0,
			Properties: map[string]interface{}{
				"since": 2020,
			},
		}

		err := driver.AddEdge(ctx, edge)
		if err != nil {
			t.Fatalf("AddEdge failed: %v", err)
		}
	})

	t.Run("GetEdge", func(t *testing.T) {
		edge, err := driver.GetEdge(ctx, "test-edge-1")
		if err != nil {
			t.Fatalf("GetEdge failed: %v", err)
		}

		if edge.ID != "test-edge-1" {
			t.Errorf("Edge ID mismatch: got %s, want test-edge-1", edge.ID)
		}
		if edge.Source != "test-person-1" {
			t.Errorf("Edge Source mismatch: got %s, want test-person-1", edge.Source)
		}
		if edge.Target != "test-person-2" {
			t.Errorf("Edge Target mismatch: got %s, want test-person-2", edge.Target)
		}
	})

	t.Run("DeleteEdge", func(t *testing.T) {
		err := driver.DeleteEdge(ctx, "test-edge-1")
		if err != nil {
			t.Fatalf("DeleteEdge failed: %v", err)
		}

		// 验证删除
		_, err = driver.GetEdge(ctx, "test-edge-1")
		if err != graphdb.ErrEdgeNotFound {
			t.Errorf("Expected ErrEdgeNotFound, got %v", err)
		}
	})
}

func TestNeo4j_FindNodes(t *testing.T) {
	driver, cleanup := setupTestDriver(t)
	defer cleanup()

	ctx := context.Background()

	// 准备测试数据
	nodes := []*graphdb.Node{
		{ID: "test-person-1", Type: "Person", Label: "Alice", Properties: map[string]interface{}{"city": "Beijing", "age": 30}},
		{ID: "test-person-2", Type: "Person", Label: "Bob", Properties: map[string]interface{}{"city": "Shanghai", "age": 28}},
		{ID: "test-org-1", Type: "Organization", Label: "Company A"},
	}
	driver.BatchAddNodes(ctx, nodes)

	t.Run("FindNodesByType", func(t *testing.T) {
		results, err := driver.FindNodes(ctx, graphdb.NodeFilter{
			Types: []string{"Person"},
		})
		if err != nil {
			t.Fatalf("FindNodes failed: %v", err)
		}

		if len(results) < 2 {
			t.Errorf("Expected at least 2 Person nodes, got %d", len(results))
		}
	})

	t.Run("FindNodesByProperty", func(t *testing.T) {
		results, err := driver.FindNodes(ctx, graphdb.NodeFilter{
			Properties: map[string]interface{}{
				"city": "Beijing",
			},
		})
		if err != nil {
			t.Fatalf("FindNodes failed: %v", err)
		}

		if len(results) < 1 {
			t.Errorf("Expected at least 1 node in Beijing, got %d", len(results))
		}
		
		found := false
		for _, node := range results {
			if node.ID == "test-person-1" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected to find test-person-1")
		}
	})

	t.Run("FindNodesWithLimit", func(t *testing.T) {
		results, err := driver.FindNodes(ctx, graphdb.NodeFilter{
			Types: []string{"Person"},
			Limit: 1,
		})
		if err != nil {
			t.Fatalf("FindNodes failed: %v", err)
		}

		if len(results) != 1 {
			t.Errorf("Expected exactly 1 node with limit, got %d", len(results))
		}
	})
}

func TestNeo4j_Traverse(t *testing.T) {
	driver, cleanup := setupTestDriver(t)
	defer cleanup()

	ctx := context.Background()

	// 构建测试图: A -> B -> C -> D
	nodes := []*graphdb.Node{
		{ID: "test-person-1", Type: "Person", Label: "Alice"},
		{ID: "test-person-2", Type: "Person", Label: "Bob"},
		{ID: "test-person-3", Type: "Person", Label: "Charlie"},
		{ID: "test-person-4", Type: "Person", Label: "David"},
	}
	driver.BatchAddNodes(ctx, nodes)

	edges := []*graphdb.Edge{
		{ID: "edge-1", Source: "test-person-1", Target: "test-person-2", Type: "KNOWS", Directed: true, Weight: 1.0},
		{ID: "edge-2", Source: "test-person-2", Target: "test-person-3", Type: "KNOWS", Directed: true, Weight: 1.0},
		{ID: "edge-3", Source: "test-person-3", Target: "test-person-4", Type: "KNOWS", Directed: true, Weight: 1.0},
	}
	driver.BatchAddEdges(ctx, edges)

	// 等待数据写入
	time.Sleep(100 * time.Millisecond)

	t.Run("TraverseDepth1", func(t *testing.T) {
		result, err := driver.Traverse(ctx, "test-person-1", graphdb.TraverseOptions{
			MaxDepth:  1,
			Direction: graphdb.DirectionOutbound,
			Strategy:  graphdb.StrategyBFS,
		})
		if err != nil {
			t.Fatalf("Traverse failed: %v", err)
		}

		// 应该包含 Alice 和 Bob
		if len(result.Nodes) < 1 {
			t.Errorf("Expected at least 1 nodes at depth 1, got %d", len(result.Nodes))
		}

		t.Logf("Found %d nodes at depth 1", len(result.Nodes))
		for _, node := range result.Nodes {
			t.Logf("  - %s (%s)", node.Label, node.ID)
		}
	})

	t.Run("TraverseDepth2", func(t *testing.T) {
		result, err := driver.Traverse(ctx, "test-person-1", graphdb.TraverseOptions{
			MaxDepth:  2,
			Direction: graphdb.DirectionOutbound,
			Strategy:  graphdb.StrategyBFS,
		})
		if err != nil {
			t.Fatalf("Traverse failed: %v", err)
		}

		// 应该包含 Alice, Bob, Charlie
		if len(result.Nodes) < 2 {
			t.Errorf("Expected at least 2 nodes at depth 2, got %d", len(result.Nodes))
		}

		t.Logf("Found %d nodes at depth 2", len(result.Nodes))
		for _, node := range result.Nodes {
			t.Logf("  - %s (%s)", node.Label, node.ID)
		}
	})

	t.Run("TraverseWithLimit", func(t *testing.T) {
		result, err := driver.Traverse(ctx, "test-person-1", graphdb.TraverseOptions{
			MaxDepth:  3,
			Direction: graphdb.DirectionOutbound,
			Strategy:  graphdb.StrategyBFS,
			Limit:     2,
		})
		if err != nil {
			t.Fatalf("Traverse failed: %v", err)
		}

		if len(result.Nodes) > 2 {
			t.Errorf("Expected at most 2 nodes with limit, got %d", len(result.Nodes))
		}
	})
}

func TestNeo4j_ShortestPath(t *testing.T) {
	driver, cleanup := setupTestDriver(t)
	defer cleanup()

	ctx := context.Background()

	// 构建测试图: A -> B -> C
	nodes := []*graphdb.Node{
		{ID: "test-person-1", Type: "Person", Label: "Alice"},
		{ID: "test-person-2", Type: "Person", Label: "Bob"},
		{ID: "test-person-3", Type: "Person", Label: "Charlie"},
	}
	driver.BatchAddNodes(ctx, nodes)

	edges := []*graphdb.Edge{
		{ID: "edge-1", Source: "test-person-1", Target: "test-person-2", Type: "KNOWS", Directed: true, Weight: 1.0},
		{ID: "edge-2", Source: "test-person-2", Target: "test-person-3", Type: "KNOWS", Directed: true, Weight: 1.0},
	}
	driver.BatchAddEdges(ctx, edges)

	// 等待数据写入
	time.Sleep(100 * time.Millisecond)

	t.Run("FindShortestPath", func(t *testing.T) {
		path, err := driver.ShortestPath(ctx, "test-person-1", "test-person-3", graphdb.PathOptions{
			MaxDepth:  5,
			Algorithm: graphdb.AlgorithmBFS,
		})
		if err != nil {
			t.Fatalf("ShortestPath failed: %v", err)
		}

		if path.Length != 2 {
			t.Errorf("Expected path length 2, got %d", path.Length)
		}

		if len(path.Nodes) != 3 {
			t.Errorf("Expected 3 nodes in path, got %d", len(path.Nodes))
		}

		t.Logf("Shortest path: %d nodes, %d edges, cost: %.2f", len(path.Nodes), len(path.Edges), path.Cost)
		for i, node := range path.Nodes {
			t.Logf("  %d. %s (%s)", i+1, node.Label, node.ID)
		}
	})

	t.Run("NoPathExists", func(t *testing.T) {
		// 添加一个孤立节点
		isolatedNode := &graphdb.Node{
			ID:    "test-person-isolated",
			Type:  "Person",
			Label: "Isolated",
		}
		driver.AddNode(ctx, isolatedNode)
		defer driver.DeleteNode(ctx, "test-person-isolated")

		_, err := driver.ShortestPath(ctx, "test-person-1", "test-person-isolated", graphdb.PathOptions{
			MaxDepth:  5,
			Algorithm: graphdb.AlgorithmBFS,
		})
		if err != graphdb.ErrNoPathFound {
			t.Errorf("Expected ErrNoPathFound, got %v", err)
		}
	})
}

func TestNeo4j_PerformanceTest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	driver, cleanup := setupTestDriver(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("BatchInsertPerformance", func(t *testing.T) {
		// 创建 100 个节点
		nodes := make([]*graphdb.Node, 100)
		for i := 0; i < 100; i++ {
			nodes[i] = &graphdb.Node{
				ID:    fmt.Sprintf("perf-node-%d", i),
				Type:  "TestNode",
				Label: fmt.Sprintf("Node %d", i),
				Properties: map[string]interface{}{
					"index": i,
				},
			}
		}

		start := time.Now()
		err := driver.BatchAddNodes(ctx, nodes)
		duration := time.Since(start)

		if err != nil {
			t.Fatalf("BatchAddNodes failed: %v", err)
		}

		t.Logf("Inserted 100 nodes in %v (%.2f nodes/sec)", duration, 100.0/duration.Seconds())

		// 清理
		for i := 0; i < 100; i++ {
			driver.DeleteNode(ctx, fmt.Sprintf("perf-node-%d", i))
		}
	})
}
