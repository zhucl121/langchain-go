package health

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/zhucl121/langchain-go/pkg/cluster/node"
)

// HTTPChecker HTTP 健康检查器
type HTTPChecker struct {
	config HTTPConfig
	client *http.Client
}

// HTTPConfig HTTP 健康检查配置
type HTTPConfig struct {
	// Endpoint 健康检查端点（例如 "/health"）
	Endpoint string

	// Method HTTP 方法（默认 GET）
	Method string

	// Timeout 超时时间
	Timeout time.Duration

	// ExpectedStatus 期望的HTTP状态码（默认 200）
	ExpectedStatus int

	// ExpectedBody 期望的响应体内容（可选）
	ExpectedBody string

	// Scheme 协议（http 或 https，默认 http）
	Scheme string

	// Headers 自定义请求头
	Headers map[string]string

	// InsecureSkipVerify 是否跳过 TLS 验证
	InsecureSkipVerify bool
}

// DefaultHTTPConfig 返回默认的 HTTP 配置
func DefaultHTTPConfig() HTTPConfig {
	return HTTPConfig{
		Endpoint:       "/health",
		Method:         "GET",
		Timeout:        5 * time.Second,
		ExpectedStatus: http.StatusOK,
		Scheme:         "http",
	}
}

// NewHTTPChecker 创建 HTTP 健康检查器
func NewHTTPChecker(config HTTPConfig) *HTTPChecker {
	// 设置默认值
	if config.Method == "" {
		config.Method = "GET"
	}
	if config.Timeout == 0 {
		config.Timeout = 5 * time.Second
	}
	if config.ExpectedStatus == 0 {
		config.ExpectedStatus = http.StatusOK
	}
	if config.Scheme == "" {
		config.Scheme = "http"
	}
	if config.Endpoint == "" {
		config.Endpoint = "/health"
	}

	// 创建 HTTP 客户端
	client := &http.Client{
		Timeout: config.Timeout,
	}

	// 如果需要跳过 TLS 验证
	if config.InsecureSkipVerify {
		client.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		}
	}

	return &HTTPChecker{
		config: config,
		client: client,
	}
}

// Check 执行 HTTP 健康检查
func (h *HTTPChecker) Check(ctx context.Context, n *node.Node) (*CheckResult, error) {
	if n == nil {
		return nil, fmt.Errorf("node is nil")
	}

	// 构建 URL
	url := fmt.Sprintf("%s://%s:%d%s",
		h.config.Scheme,
		n.Address,
		n.Port,
		h.config.Endpoint,
	)

	// 记录开始时间
	startTime := time.Now()

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, h.config.Method, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 添加自定义请求头
	for key, value := range h.config.Headers {
		req.Header.Set(key, value)
	}

	// 发送请求
	resp, err := h.client.Do(req)
	latency := time.Since(startTime)

	if err != nil {
		return &CheckResult{
			Healthy:   false,
			Status:    HealthStatusUnhealthy,
			Message:   fmt.Sprintf("HTTP request failed: %v", err),
			Latency:   latency,
			Timestamp: time.Now(),
			Metadata: map[string]interface{}{
				"url":   url,
				"error": err.Error(),
			},
		}, nil
	}
	defer resp.Body.Close()

	// 检查状态码
	if resp.StatusCode != h.config.ExpectedStatus {
		return &CheckResult{
			Healthy: false,
			Status:  HealthStatusUnhealthy,
			Message: fmt.Sprintf("Unexpected status code: %d (expected %d)",
				resp.StatusCode, h.config.ExpectedStatus),
			Latency:   latency,
			Timestamp: time.Now(),
			Metadata: map[string]interface{}{
				"url":            url,
				"status_code":    resp.StatusCode,
				"expected_status": h.config.ExpectedStatus,
			},
		}, nil
	}

	// 如果配置了期望的响应体，验证它
	if h.config.ExpectedBody != "" {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return &CheckResult{
				Healthy:   false,
				Status:    HealthStatusUnhealthy,
				Message:   fmt.Sprintf("Failed to read response body: %v", err),
				Latency:   latency,
				Timestamp: time.Now(),
			}, nil
		}

		bodyStr := string(body)
		if bodyStr != h.config.ExpectedBody {
			return &CheckResult{
				Healthy: false,
				Status:  HealthStatusUnhealthy,
				Message: "Response body mismatch",
				Latency: latency,
				Timestamp: time.Now(),
				Metadata: map[string]interface{}{
					"expected": h.config.ExpectedBody,
					"actual":   bodyStr,
				},
			}, nil
		}
	}

	// 健康检查通过
	return &CheckResult{
		Healthy:   true,
		Status:    HealthStatusHealthy,
		Message:   "HTTP health check passed",
		Latency:   latency,
		Timestamp: time.Now(),
		Metadata: map[string]interface{}{
			"url":         url,
			"status_code": resp.StatusCode,
		},
	}, nil
}

// Type 返回类型
func (h *HTTPChecker) Type() string {
	return "http"
}
