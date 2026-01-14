package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"langchain-go/pkg/types"
)

// HTTPRequestTool 是 HTTP 请求工具。
//
// 支持 GET、POST、PUT、DELETE 等 HTTP 方法。
type HTTPRequestTool struct {
	name           string
	description    string
	client         *http.Client
	allowedMethods []string
	allowedDomains []string
}

// HTTPRequestConfig 是 HTTP 请求工具的配置。
type HTTPRequestConfig struct {
	// Timeout HTTP 请求超时时间
	Timeout time.Duration

	// AllowedMethods 允许的 HTTP 方法（为空则允许所有）
	AllowedMethods []string

	// AllowedDomains 允许的域名列表（为空则允许所有）
	AllowedDomains []string
}

// NewHTTPRequestTool 创建 HTTP 请求工具。
//
// 参数：
//   - config: 工具配置
//
// 返回：
//   - *HTTPRequestTool: HTTP 请求工具实例
//
func NewHTTPRequestTool(config HTTPRequestConfig) *HTTPRequestTool {
	timeout := config.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	allowedMethods := config.AllowedMethods
	if len(allowedMethods) == 0 {
		allowedMethods = []string{"GET", "POST", "PUT", "DELETE", "PATCH"}
	}

	return &HTTPRequestTool{
		name:           "http_request",
		description:    "Make HTTP requests to external APIs or web services",
		client:         &http.Client{Timeout: timeout},
		allowedMethods: allowedMethods,
		allowedDomains: config.AllowedDomains,
	}
}

// GetName 实现 Tool 接口。
func (t *HTTPRequestTool) GetName() string {
	return t.name
}

// GetDescription 实现 Tool 接口。
func (t *HTTPRequestTool) GetDescription() string {
	return t.description
}

// GetParameters 实现 Tool 接口。
func (t *HTTPRequestTool) GetParameters() types.Schema {
	return types.Schema{
		Type: "object",
		Properties: map[string]types.Schema{
			"url": {
				Type:        "string",
				Description: "The URL to send the request to",
			},
			"method": {
				Type:        "string",
				Description: fmt.Sprintf("HTTP method (%s)", strings.Join(t.allowedMethods, ", ")),
			},
			"headers": {
				Type:        "object",
				Description: "HTTP headers (optional)",
			},
			"body": {
				Type:        "string",
				Description: "Request body for POST/PUT/PATCH (optional)",
			},
		},
		Required: []string{"url", "method"},
	}
}

// Execute 实现 Tool 接口。
func (t *HTTPRequestTool) Execute(ctx context.Context, args map[string]any) (any, error) {
	// 提取参数
	url, ok := args["url"].(string)
	if !ok || url == "" {
		return nil, fmt.Errorf("%w: missing or invalid 'url'", ErrInvalidArguments)
	}

	method, ok := args["method"].(string)
	if !ok || method == "" {
		return nil, fmt.Errorf("%w: missing or invalid 'method'", ErrInvalidArguments)
	}
	method = strings.ToUpper(method)

	// 验证方法
	if !t.isMethodAllowed(method) {
		return nil, fmt.Errorf("%w: method '%s' not allowed", ErrInvalidArguments, method)
	}

	// 验证域名
	if !t.isDomainAllowed(url) {
		return nil, fmt.Errorf("%w: domain not allowed", ErrInvalidArguments)
	}

	// 构建请求体
	var body io.Reader
	if bodyStr, ok := args["body"].(string); ok && bodyStr != "" {
		body = strings.NewReader(bodyStr)
	}

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to create request: %v", ErrExecutionFailed, err)
	}

	// 设置 headers
	if headers, ok := args["headers"].(map[string]any); ok {
		for key, value := range headers {
			if valueStr, ok := value.(string); ok {
				req.Header.Set(key, valueStr)
			}
		}
	}

	// 如果有 body 但没设置 Content-Type，默认为 JSON
	if body != nil && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
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

	// 构建结果
	result := map[string]any{
		"status_code": resp.StatusCode,
		"headers":     resp.Header,
		"body":        string(respBody),
	}

	// 尝试解析 JSON 响应
	if strings.Contains(resp.Header.Get("Content-Type"), "application/json") {
		var jsonBody any
		if err := json.Unmarshal(respBody, &jsonBody); err == nil {
			result["json"] = jsonBody
		}
	}

	return result, nil
}

// ToTypesTool 实现 Tool 接口。
func (t *HTTPRequestTool) ToTypesTool() types.Tool {
	return types.Tool{
		Name:        t.name,
		Description: t.description,
		Parameters:  t.GetParameters(),
	}
}

// isMethodAllowed 检查 HTTP 方法是否允许
func (t *HTTPRequestTool) isMethodAllowed(method string) bool {
	for _, allowed := range t.allowedMethods {
		if strings.EqualFold(allowed, method) {
			return true
		}
	}
	return false
}

// isDomainAllowed 检查域名是否允许
func (t *HTTPRequestTool) isDomainAllowed(url string) bool {
	if len(t.allowedDomains) == 0 {
		return true // 允许所有域名
	}

	for _, domain := range t.allowedDomains {
		if strings.Contains(url, domain) {
			return true
		}
	}

	return false
}

// ShellTool 是 Shell 命令工具。
//
// ⚠️ 警告：此工具允许执行 shell 命令，存在安全风险。
// 仅在受信任的环境中使用，并严格限制允许的命令。
type ShellTool struct {
	name            string
	description     string
	allowedCommands []string
}

// ShellToolConfig 是 Shell 工具的配置。
type ShellToolConfig struct {
	// AllowedCommands 允许的命令列表（白名单）
	AllowedCommands []string
}

// NewShellTool 创建 Shell 命令工具。
//
// ⚠️ 警告：此工具具有安全风险，请谨慎使用。
//
// 参数：
//   - config: 工具配置
//
// 返回：
//   - *ShellTool: Shell 工具实例
//
func NewShellTool(config ShellToolConfig) *ShellTool {
	return &ShellTool{
		name:            "shell",
		description:     "Execute shell commands (restricted to allowed commands only)",
		allowedCommands: config.AllowedCommands,
	}
}

// GetName 实现 Tool 接口。
func (t *ShellTool) GetName() string {
	return t.name
}

// GetDescription 实现 Tool 接口。
func (t *ShellTool) GetDescription() string {
	return t.description
}

// GetParameters 实现 Tool 接口。
func (t *ShellTool) GetParameters() types.Schema {
	return types.Schema{
		Type: "object",
		Properties: map[string]types.Schema{
			"command": {
				Type:        "string",
				Description: fmt.Sprintf("Shell command to execute. Allowed commands: %s", strings.Join(t.allowedCommands, ", ")),
			},
		},
		Required: []string{"command"},
	}
}

// Execute 实现 Tool 接口。
func (t *ShellTool) Execute(ctx context.Context, args map[string]any) (any, error) {
	command, ok := args["command"].(string)
	if !ok || command == "" {
		return nil, fmt.Errorf("%w: missing or invalid 'command'", ErrInvalidArguments)
	}

	// 验证命令是否在白名单中
	if !t.isCommandAllowed(command) {
		return nil, fmt.Errorf("%w: command not allowed", ErrInvalidArguments)
	}

	// 注意：实际执行 shell 命令需要使用 os/exec 包
	// 这里只返回一个模拟结果，实际实现需要谨慎处理安全问题
	return nil, fmt.Errorf("shell execution not implemented for security reasons")
}

// ToTypesTool 实现 Tool 接口。
func (t *ShellTool) ToTypesTool() types.Tool {
	return types.Tool{
		Name:        t.name,
		Description: t.description,
		Parameters:  t.GetParameters(),
	}
}

// isCommandAllowed 检查命令是否在白名单中
func (t *ShellTool) isCommandAllowed(command string) bool {
	if len(t.allowedCommands) == 0 {
		return false // 如果没有白名单，不允许任何命令
	}

	// 提取命令名称（第一个词）
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return false
	}

	cmdName := parts[0]

	for _, allowed := range t.allowedCommands {
		if cmdName == allowed {
			return true
		}
	}

	return false
}

// JSONPlaceholderTool 是一个示例工具，用于测试。
//
// 它调用 JSONPlaceholder API 获取数据。
type JSONPlaceholderTool struct {
	client *http.Client
}

// NewJSONPlaceholderTool 创建 JSONPlaceholder 工具。
//
// 返回：
//   - *JSONPlaceholderTool: JSONPlaceholder 工具实例
//
func NewJSONPlaceholderTool() *JSONPlaceholderTool {
	return &JSONPlaceholderTool{
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

// GetName 实现 Tool 接口。
func (t *JSONPlaceholderTool) GetName() string {
	return "jsonplaceholder"
}

// GetDescription 实现 Tool 接口。
func (t *JSONPlaceholderTool) GetDescription() string {
	return "Fetch data from JSONPlaceholder API (for testing purposes)"
}

// GetParameters 实现 Tool 接口。
func (t *JSONPlaceholderTool) GetParameters() types.Schema {
	return types.Schema{
		Type: "object",
		Properties: map[string]types.Schema{
			"resource": {
				Type:        "string",
				Description: "Resource type (posts, users, comments, etc.)",
			},
			"id": {
				Type:        "integer",
				Description: "Resource ID (optional)",
			},
		},
		Required: []string{"resource"},
	}
}

// Execute 实现 Tool 接口。
func (t *JSONPlaceholderTool) Execute(ctx context.Context, args map[string]any) (any, error) {
	resource, ok := args["resource"].(string)
	if !ok || resource == "" {
		return nil, fmt.Errorf("%w: missing or invalid 'resource'", ErrInvalidArguments)
	}

	url := fmt.Sprintf("https://jsonplaceholder.typicode.com/%s", resource)

	// 如果提供了 ID
	if id, ok := args["id"]; ok {
		switch idVal := id.(type) {
		case float64:
			url = fmt.Sprintf("%s/%.0f", url, idVal)
		case int:
			url = fmt.Sprintf("%s/%d", url, idVal)
		}
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrExecutionFailed, err)
	}

	resp, err := t.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrExecutionFailed, err)
	}
	defer resp.Body.Close()

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, resp.Body); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrExecutionFailed, err)
	}

	var result any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrExecutionFailed, err)
	}

	return result, nil
}

// ToTypesTool 实现 Tool 接口。
func (t *JSONPlaceholderTool) ToTypesTool() types.Tool {
	return types.Tool{
		Name:        t.GetName(),
		Description: t.GetDescription(),
		Parameters:  t.GetParameters(),
	}
}
