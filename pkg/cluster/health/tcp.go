package health

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/zhucl121/langchain-go/pkg/cluster/node"
)

// TCPChecker TCP 连接健康检查器
type TCPChecker struct {
	config TCPConfig
}

// TCPConfig TCP 健康检查配置
type TCPConfig struct {
	// Timeout 连接超时时间
	Timeout time.Duration

	// RetryCount 重试次数
	RetryCount int

	// RetryDelay 重试延迟
	RetryDelay time.Duration
}

// DefaultTCPConfig 返回默认的 TCP 配置
func DefaultTCPConfig() TCPConfig {
	return TCPConfig{
		Timeout:    3 * time.Second,
		RetryCount: 3,
		RetryDelay: 1 * time.Second,
	}
}

// NewTCPChecker 创建 TCP 健康检查器
func NewTCPChecker(config TCPConfig) *TCPChecker {
	// 设置默认值
	if config.Timeout == 0 {
		config.Timeout = 3 * time.Second
	}
	if config.RetryCount == 0 {
		config.RetryCount = 3
	}
	if config.RetryDelay == 0 {
		config.RetryDelay = 1 * time.Second
	}

	return &TCPChecker{
		config: config,
	}
}

// Check 执行 TCP 连接检查
func (t *TCPChecker) Check(ctx context.Context, n *node.Node) (*CheckResult, error) {
	if n == nil {
		return nil, fmt.Errorf("node is nil")
	}

	address := fmt.Sprintf("%s:%d", n.Address, n.Port)
	var lastErr error
	startTime := time.Now()

	// 尝试连接，支持重试
	for i := 0; i < t.config.RetryCount; i++ {
		if i > 0 {
			// 重试前等待
			select {
			case <-time.After(t.config.RetryDelay):
			case <-ctx.Done():
				return &CheckResult{
					Healthy:   false,
					Status:    HealthStatusUnhealthy,
					Message:   "Check cancelled",
					Latency:   time.Since(startTime),
					Timestamp: time.Now(),
				}, ctx.Err()
			}
		}

		conn, err := net.DialTimeout("tcp", address, t.config.Timeout)
		if err == nil {
			// 连接成功，立即关闭
			conn.Close()
			latency := time.Since(startTime)

			return &CheckResult{
				Healthy:   true,
				Status:    HealthStatusHealthy,
				Message:   "TCP connection successful",
				Latency:   latency,
				Timestamp: time.Now(),
				Metadata: map[string]interface{}{
					"address": address,
					"retries": i,
				},
			}, nil
		}

		lastErr = err
	}

	// 所有重试都失败
	return &CheckResult{
		Healthy: false,
		Status:  HealthStatusUnhealthy,
		Message: fmt.Sprintf("TCP connection failed after %d retries: %v",
			t.config.RetryCount, lastErr),
		Latency:   time.Since(startTime),
		Timestamp: time.Now(),
		Metadata: map[string]interface{}{
			"address": address,
			"retries": t.config.RetryCount,
			"error":   lastErr.Error(),
		},
	}, nil
}

// Type 返回类型
func (t *TCPChecker) Type() string {
	return "tcp"
}
