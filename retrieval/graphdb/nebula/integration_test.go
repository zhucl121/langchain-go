package nebula_test

import (
	"context"
	"testing"
	"time"

	"github.com/zhucl121/langchain-go/retrieval/graphdb"
	"github.com/zhucl121/langchain-go/retrieval/graphdb/nebula"
)

// 注意：这些测试需要 NebulaGraph 实例运行
// 使用 docker-compose.graphdb.yml 启动 NebulaGraph:
// docker compose -f docker-compose.graphdb.yml up -d nebula-metad nebula-storaged nebula-graphd

func TestNebulaDriver_Integration(t *testing.T) {
	t.Skip("Integration test - requires NebulaGraph instance")

	config := nebula.DefaultConfig()
	config.Space = "langchain_test"
	config.Timeout = 30 * time.Second

	driver, err := nebula.NewNebulaDriver(config)
	if err != nil {
		t.Fatalf("Failed to create driver: %v", err)
	}

	ctx := context.Background()

	// 连接
	if err := driver.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer driver.Close()

	// 测试连接状态
	if !driver.IsConnected() {
		t.Error("Driver should be connected")
	}

	t.Run("Add and Get Node", func(t *testing.T) {
		node := &graphdb.Node{
			ID:    "test_person_1",
			Type:  "Person",
			Label: "John Doe",
			Properties: map[string]interface{}{
				"name": "John Doe",
				"age":  30,
				"city": "Beijing",
			},
		}

		// 添加节点
		err := driver.AddNode(ctx, node)
		if err != nil {
			t.Errorf("Failed to add node: %v", err)
		}

		// 获取节点
		retrieved, err := driver.GetNode(ctx, "test_person_1")
		if err != nil {
			t.Errorf("Failed to get node: %v", err)
		}

		if retrieved.ID != node.ID {
			t.Errorf("Expected ID %s, got %s", node.ID, retrieved.ID)
		}
	})

	t.Run("Add and Get Edge", func(t *testing.T) {
		// 先添加两个节点
		node1 := &graphdb.Node{
			ID:   "test_person_2",
			Type: "Person",
			Properties: map[string]interface{}{
				"name": "Alice",
			},
		}
		node2 := &graphdb.Node{
			ID:   "test_org_1",
			Type: "Organization",
			Properties: map[string]interface{}{
				"name": "ACME Corp",
			},
		}

		driver.AddNode(ctx, node1)
		driver.AddNode(ctx, node2)

		// 添加边
		edge := &graphdb.Edge{
			Source:   "test_person_2",
			Target:   "test_org_1",
			Type:     "WORKS_FOR",
			Directed: true,
			Properties: map[string]interface{}{
				"since": 2020,
			},
		}

		err := driver.AddEdge(ctx, edge)
		if err != nil {
			t.Errorf("Failed to add edge: %v", err)
		}
	})

	t.Run("Traverse", func(t *testing.T) {
		result, err := driver.Traverse(ctx, "test_person_1", graphdb.TraverseOptions{
			MaxDepth:  2,
			Strategy:  graphdb.StrategyBFS,
			Direction: graphdb.DirectionBoth,
		})

		if err != nil {
			t.Errorf("Failed to traverse: %v", err)
		}

		if result == nil {
			t.Error("Expected traverse result, got nil")
		}
	})

	t.Run("Shortest Path", func(t *testing.T) {
		path, err := driver.ShortestPath(ctx, "test_person_2", "test_org_1", graphdb.PathOptions{
			MaxDepth: 5,
		})

		if err != nil {
			t.Errorf("Failed to find shortest path: %v", err)
		}

		if path == nil {
			t.Error("Expected path, got nil")
		}
	})

	t.Run("Delete Edge", func(t *testing.T) {
		err := driver.DeleteEdgeByEndpoints(ctx, "test_person_2", "test_org_1", "WORKS_FOR")
		if err != nil {
			t.Errorf("Failed to delete edge: %v", err)
		}
	})

	t.Run("Delete Node", func(t *testing.T) {
		err := driver.DeleteNode(ctx, "test_person_1")
		if err != nil {
			t.Errorf("Failed to delete node: %v", err)
		}
	})
}

func TestNebulaDriver_QueryBuilder(t *testing.T) {
	qb := nebula.NewQueryBuilder("test_space")

	t.Run("InsertVertex", func(t *testing.T) {
		query := qb.InsertVertex("person-1", "Person", map[string]interface{}{
			"name": "John",
			"age":  30,
		})

		// 注意：由于 map 遍历顺序不确定，这里只检查关键部分
		if query == "" {
			t.Error("Expected non-empty query")
		}
	})

	t.Run("InsertEdge", func(t *testing.T) {
		query := qb.InsertEdge("person-1", "org-1", "WORKS_FOR", map[string]interface{}{
			"since": 2020,
		})

		if query == "" {
			t.Error("Expected non-empty query")
		}
	})

	t.Run("Traverse", func(t *testing.T) {
		query := qb.Traverse("person-1", 3, "BIDIRECT")
		expected := `GO 1 TO 3 STEPS FROM "person-1" OVER * BIDIRECT YIELD $$ AS dst, edge AS e`

		if query != expected {
			t.Errorf("Expected query:\n%s\nGot:\n%s", expected, query)
		}
	})

	t.Run("ShortestPath", func(t *testing.T) {
		query := qb.ShortestPath("person-1", "org-1", 5)
		expected := `FIND SHORTEST PATH WITH PROP FROM "person-1" TO "org-1" OVER * UPTO 5 STEPS YIELD path AS p`

		if query != expected {
			t.Errorf("Expected query:\n%s\nGot:\n%s", expected, query)
		}
	})
}
