package health

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/zhucl121/langchain-go/pkg/cluster/node"
)

func TestHTTPChecker_Check_Success(t *testing.T) {
	// 创建测试 HTTP 服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("healthy"))
	}))
	defer server.Close()

	// 解析服务器地址
	addr := server.Listener.Addr().String()
	host, port := parseAddr(addr)

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
	checker := NewHTTPChecker(HTTPConfig{
		Endpoint:       "/",
		Timeout:        5 * time.Second,
		ExpectedStatus: http.StatusOK,
		Scheme:         "http",
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

func TestHTTPChecker_Check_UnexpectedStatus(t *testing.T) {
	// 创建返回 500 错误的测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	addr := server.Listener.Addr().String()
	host, port := parseAddr(addr)

	testNode := &node.Node{
		ID:      "test-node",
		Address: host,
		Port:    port,
		Roles:   []node.NodeRole{node.RoleWorker},
	}

	checker := NewHTTPChecker(HTTPConfig{
		Endpoint:       "/",
		Timeout:        5 * time.Second,
		ExpectedStatus: http.StatusOK,
		Scheme:         "http",
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

func TestHTTPChecker_Check_Timeout(t *testing.T) {
	// 创建一个会延迟响应的服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	addr := server.Listener.Addr().String()
	host, port := parseAddr(addr)

	testNode := &node.Node{
		ID:      "test-node",
		Address: host,
		Port:    port,
		Roles:   []node.NodeRole{node.RoleWorker},
	}

	// 设置一个很短的超时时间
	checker := NewHTTPChecker(HTTPConfig{
		Endpoint: "/",
		Timeout:  100 * time.Millisecond,
		Scheme:   "http",
	})

	ctx := context.Background()
	result, err := checker.Check(ctx, testNode)

	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}

	if result.Healthy {
		t.Error("Expected node to be unhealthy due to timeout")
	}
}

func TestHTTPChecker_Check_ExpectedBody(t *testing.T) {
	expectedBody := "OK"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(expectedBody))
	}))
	defer server.Close()

	addr := server.Listener.Addr().String()
	host, port := parseAddr(addr)

	testNode := &node.Node{
		ID:      "test-node",
		Address: host,
		Port:    port,
		Roles:   []node.NodeRole{node.RoleWorker},
	}

	checker := NewHTTPChecker(HTTPConfig{
		Endpoint:     "/",
		Timeout:      5 * time.Second,
		Scheme:       "http",
		ExpectedBody: expectedBody,
	})

	ctx := context.Background()
	result, err := checker.Check(ctx, testNode)

	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}

	if !result.Healthy {
		t.Errorf("Expected node to be healthy, got: %s", result.Message)
	}
}

func TestHTTPChecker_Type(t *testing.T) {
	checker := NewHTTPChecker(DefaultHTTPConfig())
	if checker.Type() != "http" {
		t.Errorf("Expected type 'http', got '%s'", checker.Type())
	}
}

func TestDefaultHTTPConfig(t *testing.T) {
	config := DefaultHTTPConfig()

	if config.Endpoint != "/health" {
		t.Errorf("Expected endpoint '/health', got '%s'", config.Endpoint)
	}

	if config.Method != "GET" {
		t.Errorf("Expected method 'GET', got '%s'", config.Method)
	}

	if config.Timeout != 5*time.Second {
		t.Errorf("Expected timeout 5s, got %v", config.Timeout)
	}

	if config.ExpectedStatus != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, config.ExpectedStatus)
	}

	if config.Scheme != "http" {
		t.Errorf("Expected scheme 'http', got '%s'", config.Scheme)
	}
}

// parseAddr 解析地址字符串，返回 host 和 port
func parseAddr(addr string) (string, int) {
	host, portStr, _ := net.SplitHostPort(addr)
	port, _ := strconv.Atoi(portStr)
	return host, port
}
