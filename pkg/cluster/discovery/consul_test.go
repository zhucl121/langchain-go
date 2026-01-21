package discovery

import (
	"context"
	"testing"
	"time"

	"github.com/zhucl121/langchain-go/pkg/cluster/node"
)

func TestConsulDiscovery_BuildTags(t *testing.T) {
	config := ConsulConfig{
		ServiceName: "test-service",
		Tags:        []string{"env:test"},
	}

	disco := &ConsulDiscovery{
		config: config,
	}

	n := &node.Node{
		ID:      "node-1",
		Name:    "worker-1",
		Address: "192.168.1.10",
		Port:    8080,
		Status:  node.StatusOnline,
		Roles:   []node.NodeRole{node.RoleWorker, node.RoleCache},
		Region:  "us-east-1",
		Zone:    "us-east-1a",
	}

	tags := disco.buildTags(n)

	// 验证标签包含预期的内容
	expectedTags := map[string]bool{
		"env:test":           true,
		"role:worker":        true,
		"role:cache":         true,
		"status:online":      true,
		"region:us-east-1":   true,
		"zone:us-east-1a":    true,
	}

	for _, tag := range tags {
		if !expectedTags[tag] {
			t.Errorf("Unexpected tag: %s", tag)
		}
		delete(expectedTags, tag)
	}

	if len(expectedTags) > 0 {
		t.Errorf("Missing expected tags: %v", expectedTags)
	}
}

func TestConsulDiscovery_NodesEqual(t *testing.T) {
	node1 := &node.Node{
		ID:      "node-1",
		Status:  node.StatusOnline,
		Address: "192.168.1.10",
		Port:    8080,
		Roles:   []node.NodeRole{node.RoleWorker},
	}

	node2 := &node.Node{
		ID:      "node-1",
		Status:  node.StatusOnline,
		Address: "192.168.1.10",
		Port:    8080,
		Roles:   []node.NodeRole{node.RoleWorker},
	}

	node3 := &node.Node{
		ID:      "node-1",
		Status:  node.StatusOffline,
		Address: "192.168.1.10",
		Port:    8080,
		Roles:   []node.NodeRole{node.RoleWorker},
	}

	if !nodesEqual(node1, node2) {
		t.Error("Expected nodes to be equal")
	}

	if nodesEqual(node1, node3) {
		t.Error("Expected nodes to be different")
	}
}

func TestConsulDiscovery_Close(t *testing.T) {
	disco := &ConsulDiscovery{}

	if err := disco.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}

	// 第二次 Close 应该不会出错
	if err := disco.Close(); err != nil {
		t.Errorf("Second Close() error = %v", err)
	}

	// 关闭后操作应该返回错误
	ctx := context.Background()
	n := &node.Node{
		ID:      "node-1",
		Name:    "worker-1",
		Address: "192.168.1.10",
		Port:    8080,
		Roles:   []node.NodeRole{node.RoleWorker},
	}

	if err := disco.RegisterNode(ctx, n); err != ErrDiscoveryNotAvailable {
		t.Errorf("Expected ErrDiscoveryNotAvailable, got %v", err)
	}
}

func TestDefaultConsulConfig(t *testing.T) {
	config := DefaultConsulConfig()

	if config.Address != "localhost:8500" {
		t.Errorf("Expected address localhost:8500, got %s", config.Address)
	}

	if config.ServiceName != "langchain-go" {
		t.Errorf("Expected service name langchain-go, got %s", config.ServiceName)
	}

	if config.CheckTTL != 10*time.Second {
		t.Errorf("Expected CheckTTL 10s, got %v", config.CheckTTL)
	}

	if config.DeregisterAfter != 30*time.Second {
		t.Errorf("Expected DeregisterAfter 30s, got %v", config.DeregisterAfter)
	}
}

// 注意：以下是集成测试，需要真实的 Consul 环境
// 可以通过 docker 启动 Consul 进行测试：
// docker run -d --name consul -p 8500:8500 consul:latest

func TestConsulDiscovery_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// 此测试需要 Consul 运行在 localhost:8500
	config := DefaultConsulConfig()
	disco, err := NewConsulDiscovery(config)
	if err != nil {
		t.Skip("Consul not available:", err)
	}
	defer disco.Close()

	ctx := context.Background()

	// 测试节点注册
	testNode := &node.Node{
		ID:      "test-node-1",
		Name:    "test-worker-1",
		Address: "192.168.1.10",
		Port:    8080,
		Status:  node.StatusOnline,
		Roles:   []node.NodeRole{node.RoleWorker},
		Metadata: map[string]string{
			"name":            "test-worker-1",
			"max_connections": "1000",
		},
	}

	// 注册
	if err := disco.RegisterNode(ctx, testNode); err != nil {
		t.Fatalf("RegisterNode() error = %v", err)
	}

	// 等待注册生效
	time.Sleep(1 * time.Second)

	// 获取节点
	retrieved, err := disco.GetNode(ctx, testNode.ID)
	if err != nil {
		t.Fatalf("GetNode() error = %v", err)
	}
	if retrieved.ID != testNode.ID {
		t.Errorf("Expected node ID %s, got %s", testNode.ID, retrieved.ID)
	}

	// 列出节点
	nodes, err := disco.ListNodes(ctx, nil)
	if err != nil {
		t.Fatalf("ListNodes() error = %v", err)
	}
	if len(nodes) == 0 {
		t.Error("Expected at least one node")
	}

	// 发送心跳
	if err := disco.Heartbeat(ctx, testNode.ID); err != nil {
		t.Errorf("Heartbeat() error = %v", err)
	}

	// 注销
	if err := disco.UnregisterNode(ctx, testNode.ID); err != nil {
		t.Errorf("UnregisterNode() error = %v", err)
	}

	// 等待注销生效
	time.Sleep(1 * time.Second)

	// 获取应该失败
	_, err = disco.GetNode(ctx, testNode.ID)
	if err != node.ErrNodeNotFound {
		t.Errorf("Expected ErrNodeNotFound after unregister, got %v", err)
	}
}
