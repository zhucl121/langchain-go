package health

import (
	"context"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/zhucl121/langchain-go/pkg/cluster/node"
)

func TestTCPChecker_Check_Success(t *testing.T) {
	// 创建测试 TCP 服务器
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}
	defer listener.Close()

	// 解析地址
	addr := listener.Addr().String()
	host, portStr, _ := net.SplitHostPort(addr)
	port, _ := strconv.Atoi(portStr)

	// 创建测试节点
	testNode := &node.Node{
		ID:      "test-node",
		Name:    "test",
		Address: host,
		Port:    port,
		Status:  node.StatusOnline,
		Roles:   []node.NodeRole{node.RoleWorker},
	}

	// 创建检查器
	checker := NewTCPChecker(TCPConfig{
		Timeout:    3 * time.Second,
		RetryCount: 3,
	})

	// 执行检查
	ctx := context.Background()
	result, err := checker.Check(ctx, testNode)

	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}

	if !result.Healthy {
		t.Errorf("Expected node to be healthy, got unhealthy: %s", result.Message)
	}

	if result.Status != HealthStatusHealthy {
		t.Errorf("Expected status %s, got %s", HealthStatusHealthy, result.Status)
	}

	if result.Latency == 0 {
		t.Error("Expected non-zero latency")
	}
}

func TestTCPChecker_Check_ConnectionRefused(t *testing.T) {
	// 使用一个不存在的端口
	testNode := &node.Node{
		ID:      "test-node",
		Address: "127.0.0.1",
		Port:    54321, // 假设这个端口没有被占用
		Roles:   []node.NodeRole{node.RoleWorker},
	}

	checker := NewTCPChecker(TCPConfig{
		Timeout:    1 * time.Second,
		RetryCount: 1,
		RetryDelay: 100 * time.Millisecond,
	})

	ctx := context.Background()
	result, err := checker.Check(ctx, testNode)

	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}

	if result.Healthy {
		t.Error("Expected node to be unhealthy")
	}

	if result.Status != HealthStatusUnhealthy {
		t.Errorf("Expected status %s, got %s", HealthStatusUnhealthy, result.Status)
	}
}

func TestTCPChecker_Check_WithRetry(t *testing.T) {
	// 测试重试机制
	testNode := &node.Node{
		ID:      "test-node",
		Address: "127.0.0.1",
		Port:    54322,
		Roles:   []node.NodeRole{node.RoleWorker},
	}

	checker := NewTCPChecker(TCPConfig{
		Timeout:    500 * time.Millisecond,
		RetryCount: 3,
		RetryDelay: 200 * time.Millisecond,
	})

	ctx := context.Background()
	startTime := time.Now()
	result, _ := checker.Check(ctx, testNode)
	elapsed := time.Since(startTime)

	if result.Healthy {
		t.Error("Expected node to be unhealthy")
	}

	// 验证重试逻辑：应该至少花费 (RetryCount-1) * RetryDelay 的时间
	minExpected := 2 * 200 * time.Millisecond // 2 次重试延迟
	if elapsed < minExpected {
		t.Errorf("Expected at least %v elapsed time, got %v", minExpected, elapsed)
	}
}

func TestTCPChecker_Check_ContextCancellation(t *testing.T) {
	testNode := &node.Node{
		ID:      "test-node",
		Address: "127.0.0.1",
		Port:    54323,
		Roles:   []node.NodeRole{node.RoleWorker},
	}

	checker := NewTCPChecker(TCPConfig{
		Timeout:    5 * time.Second,
		RetryCount: 10,
		RetryDelay: 1 * time.Second,
	})

	// 创建一个会很快取消的 context
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	result, err := checker.Check(ctx, testNode)

	if err != context.DeadlineExceeded {
		t.Errorf("Expected context.DeadlineExceeded, got %v", err)
	}

	if result != nil && result.Healthy {
		t.Error("Expected unhealthy result on context cancellation")
	}
}

func TestTCPChecker_Type(t *testing.T) {
	checker := NewTCPChecker(DefaultTCPConfig())
	if checker.Type() != "tcp" {
		t.Errorf("Expected type 'tcp', got '%s'", checker.Type())
	}
}

func TestDefaultTCPConfig(t *testing.T) {
	config := DefaultTCPConfig()

	if config.Timeout != 3*time.Second {
		t.Errorf("Expected timeout 3s, got %v", config.Timeout)
	}

	if config.RetryCount != 3 {
		t.Errorf("Expected retry count 3, got %d", config.RetryCount)
	}

	if config.RetryDelay != 1*time.Second {
		t.Errorf("Expected retry delay 1s, got %v", config.RetryDelay)
	}
}
