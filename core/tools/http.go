package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/zhucl121/langchain-go/pkg/types"
)

// HTTPGetTool HTTP GET 请求工具。
//
// 功能：发送 HTTP GET 请求并返回响应
type HTTPGetTool struct {
	name        string
	description string
	client      *http.Client
}

// HTTPGetToolConfig 是 HTTPGetTool 配置。
type HTTPGetToolConfig struct {
	// Timeout HTTP 请求超时时间 (默认 30 秒)
	Timeout time.Duration

	// Client 自定义 HTTP 客户端 (可选)
	Client *http.Client
}

// NewHTTPGetTool 创建 HTTP GET 工具。
//
// 参数：
//   - config: 工具配置 (可选)
//
// 返回：
//   - *HTTPGetTool: HTTP GET 工具实例
//
// 示例：
//
//	tool := tools.NewHTTPGetTool(&tools.HTTPGetToolConfig{
//	    Timeout: 10 * time.Second,
//	})
//	result, _ := tool.Execute(ctx, map[string]any{
//	    "url": "https://api.example.com/data",
//	    "headers": map[string]string{"Authorization": "Bearer token"},
//	})
//
func NewHTTPGetTool(config *HTTPGetToolConfig) *HTTPGetTool {
	var client *http.Client

	if config != nil && config.Client != nil {
		client = config.Client
	} else {
		timeout := 30 * time.Second
		if config != nil && config.Timeout > 0 {
			timeout = config.Timeout
		}
		client = &http.Client{
			Timeout: timeout,
		}
	}

	return &HTTPGetTool{
		name: "http_get",
		description: `Send an HTTP GET request and return the response body.
Parameters:
- url: The URL to send the GET request to (required)
- headers: Optional HTTP headers as a map (e.g., {"Authorization": "Bearer token"})`,
		client: client,
	}
}

// GetName 实现 Tool 接口。
func (t *HTTPGetTool) GetName() string {
	return t.name
}

// GetDescription 实现 Tool 接口。
func (t *HTTPGetTool) GetDescription() string {
	return t.description
}

// GetParameters 实现 Tool 接口。
func (t *HTTPGetTool) GetParameters() types.Schema {
	return types.Schema{
		Type: "object",
		Properties: map[string]types.Schema{
			"url": {
				Type:        "string",
				Description: "The URL to send the GET request to",
			},
			"headers": {
				Type:        "object",
				Description: "Optional HTTP headers",
			},
		},
		Required: []string{"url"},
	}
}

// Execute 实现 Tool 接口。
func (t *HTTPGetTool) Execute(ctx context.Context, args map[string]any) (any, error) {
	// 获取 URL
	urlStr, ok := args["url"].(string)
	if !ok {
		return nil, fmt.Errorf("%w: 'url' must be a string", ErrInvalidArguments)
	}

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to create request: %v", ErrExecutionFailed, err)
	}

	// 添加自定义 headers
	if headers, ok := args["headers"].(map[string]any); ok {
		for key, value := range headers {
			if strVal, ok := value.(string); ok {
				req.Header.Set(key, strVal)
			}
		}
	}

	// 发送请求
	resp, err := t.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: request failed: %v", ErrExecutionFailed, err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to read response: %v", ErrExecutionFailed, err)
	}

	// 检查状态码
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("%w: HTTP %d: %s", ErrExecutionFailed, resp.StatusCode, string(body))
	}

	return string(body), nil
}

// ToTypesTool 实现 Tool 接口。
func (t *HTTPGetTool) ToTypesTool() types.Tool {
	return types.Tool{
		Name:        t.name,
		Description: t.description,
		Parameters:  t.GetParameters(),
	}
}

// HTTPPostTool HTTP POST 请求工具。
//
// 功能：发送 HTTP POST 请求并返回响应
type HTTPPostTool struct {
	name        string
	description string
	client      *http.Client
}

// HTTPPostToolConfig 是 HTTPPostTool 配置。
type HTTPPostToolConfig struct {
	// Timeout HTTP 请求超时时间 (默认 30 秒)
	Timeout time.Duration

	// Client 自定义 HTTP 客户端 (可选)
	Client *http.Client
}

// NewHTTPPostTool 创建 HTTP POST 工具。
//
// 参数：
//   - config: 工具配置 (可选)
//
// 返回：
//   - *HTTPPostTool: HTTP POST 工具实例
//
// 示例：
//
//	tool := tools.NewHTTPPostTool(nil)
//	result, _ := tool.Execute(ctx, map[string]any{
//	    "url": "https://api.example.com/data",
//	    "body": map[string]any{"key": "value"},
//	    "content_type": "application/json",
//	})
//
func NewHTTPPostTool(config *HTTPPostToolConfig) *HTTPPostTool {
	var client *http.Client

	if config != nil && config.Client != nil {
		client = config.Client
	} else {
		timeout := 30 * time.Second
		if config != nil && config.Timeout > 0 {
			timeout = config.Timeout
		}
		client = &http.Client{
			Timeout: timeout,
		}
	}

	return &HTTPPostTool{
		name: "http_post",
		description: `Send an HTTP POST request and return the response body.
Parameters:
- url: The URL to send the POST request to (required)
- body: The request body (can be string or object for JSON) (required)
- content_type: Content-Type header (default: "application/json")
- headers: Optional HTTP headers as a map`,
		client: client,
	}
}

// GetName 实现 Tool 接口。
func (t *HTTPPostTool) GetName() string {
	return t.name
}

// GetDescription 实现 Tool 接口。
func (t *HTTPPostTool) GetDescription() string {
	return t.description
}

// GetParameters 实现 Tool 接口。
func (t *HTTPPostTool) GetParameters() types.Schema {
	return types.Schema{
		Type: "object",
		Properties: map[string]types.Schema{
			"url": {
				Type:        "string",
				Description: "The URL to send the POST request to",
			},
			"body": {
				Type:        "string",
				Description: "The request body (string or JSON object)",
			},
			"content_type": {
				Type:        "string",
				Description: "Content-Type header (default: application/json)",
			},
			"headers": {
				Type:        "object",
				Description: "Optional HTTP headers",
			},
		},
		Required: []string{"url", "body"},
	}
}

// Execute 实现 Tool 接口。
func (t *HTTPPostTool) Execute(ctx context.Context, args map[string]any) (any, error) {
	// 获取 URL
	urlStr, ok := args["url"].(string)
	if !ok {
		return nil, fmt.Errorf("%w: 'url' must be a string", ErrInvalidArguments)
	}

	// 获取 body
	body := args["body"]
	if body == nil {
		return nil, fmt.Errorf("%w: 'body' is required", ErrInvalidArguments)
	}

	// 确定 Content-Type
	contentType := "application/json"
	if ct, ok := args["content_type"].(string); ok {
		contentType = ct
	}

	// 构建请求体
	var bodyReader io.Reader
	switch v := body.(type) {
	case string:
		bodyReader = strings.NewReader(v)
	case map[string]any:
		jsonBytes, err := json.Marshal(v)
		if err != nil {
			return nil, fmt.Errorf("%w: failed to marshal body: %v", ErrInvalidArguments, err)
		}
		bodyReader = strings.NewReader(string(jsonBytes))
	default:
		// 尝试 JSON 序列化
		jsonBytes, err := json.Marshal(v)
		if err != nil {
			return nil, fmt.Errorf("%w: unsupported body type", ErrInvalidArguments)
		}
		bodyReader = strings.NewReader(string(jsonBytes))
	}

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, "POST", urlStr, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to create request: %v", ErrExecutionFailed, err)
	}

	// 设置 Content-Type
	req.Header.Set("Content-Type", contentType)

	// 添加自定义 headers
	if headers, ok := args["headers"].(map[string]any); ok {
		for key, value := range headers {
			if strVal, ok := value.(string); ok {
				req.Header.Set(key, strVal)
			}
		}
	}

	// 发送请求
	resp, err := t.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: request failed: %v", ErrExecutionFailed, err)
	}
	defer resp.Body.Close()

	// 读取响应
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to read response: %v", ErrExecutionFailed, err)
	}

	// 检查状态码
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("%w: HTTP %d: %s", ErrExecutionFailed, resp.StatusCode, string(respBody))
	}

	return string(respBody), nil
}

// ToTypesTool 实现 Tool 接口。
func (t *HTTPPostTool) ToTypesTool() types.Tool {
	return types.Tool{
		Name:        t.name,
		Description: t.description,
		Parameters:  t.GetParameters(),
	}
}

// HTTPRequestTool 已在 builtin.go 中定义，此处不再重复
// 使用 NewHTTPRequestTool() 从 builtin.go 创建通用 HTTP 请求工具
