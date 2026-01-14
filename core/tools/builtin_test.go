package tools

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewHTTPRequestTool(t *testing.T) {
	httpTool := NewHTTPRequestTool(HTTPRequestConfig{
		Timeout:        10 * time.Second,
		AllowedMethods: []string{"GET", "POST"},
		AllowedDomains: []string{"example.com"},
	})

	assert.NotNil(t, httpTool)
	assert.Equal(t, "http_request", httpTool.GetName())
	assert.NotEmpty(t, httpTool.GetDescription())
}

func TestHTTPRequestTool_Execute_GET(t *testing.T) {
	// 创建测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message": "success"}`))
	}))
	defer server.Close()

	httpTool := NewHTTPRequestTool(HTTPRequestConfig{
		Timeout: 5 * time.Second,
	})

	result, err := httpTool.Execute(context.Background(), map[string]any{
		"url":    server.URL,
		"method": "GET",
	})

	require.NoError(t, err)
	assert.NotNil(t, result)

	resultMap := result.(map[string]any)
	assert.Equal(t, 200, resultMap["status_code"])
	assert.Contains(t, resultMap["body"], "success")
}

func TestHTTPRequestTool_Execute_POST(t *testing.T) {
	// 创建测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id": 123}`))
	}))
	defer server.Close()

	httpTool := NewHTTPRequestTool(HTTPRequestConfig{
		Timeout: 5 * time.Second,
	})

	result, err := httpTool.Execute(context.Background(), map[string]any{
		"url":    server.URL,
		"method": "POST",
		"body":   `{"name": "test"}`,
	})

	require.NoError(t, err)
	assert.NotNil(t, result)

	resultMap := result.(map[string]any)
	assert.Equal(t, 201, resultMap["status_code"])
}

func TestHTTPRequestTool_Execute_WithHeaders(t *testing.T) {
	// 创建测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer token123", r.Header.Get("Authorization"))
		assert.Equal(t, "custom-value", r.Header.Get("X-Custom-Header"))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	httpTool := NewHTTPRequestTool(HTTPRequestConfig{
		Timeout: 5 * time.Second,
	})

	result, err := httpTool.Execute(context.Background(), map[string]any{
		"url":    server.URL,
		"method": "GET",
		"headers": map[string]any{
			"Authorization":   "Bearer token123",
			"X-Custom-Header": "custom-value",
		},
	})

	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestHTTPRequestTool_Execute_MethodNotAllowed(t *testing.T) {
	httpTool := NewHTTPRequestTool(HTTPRequestConfig{
		Timeout:        5 * time.Second,
		AllowedMethods: []string{"GET"},
	})

	result, err := httpTool.Execute(context.Background(), map[string]any{
		"url":    "http://example.com",
		"method": "POST",
	})

	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidArguments))
	assert.Nil(t, result)
}

func TestHTTPRequestTool_Execute_DomainNotAllowed(t *testing.T) {
	httpTool := NewHTTPRequestTool(HTTPRequestConfig{
		Timeout:        5 * time.Second,
		AllowedDomains: []string{"allowed.com"},
	})

	result, err := httpTool.Execute(context.Background(), map[string]any{
		"url":    "http://blocked.com",
		"method": "GET",
	})

	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidArguments))
	assert.Nil(t, result)
}

func TestHTTPRequestTool_Execute_InvalidArgs(t *testing.T) {
	httpTool := NewHTTPRequestTool(HTTPRequestConfig{})

	tests := []struct {
		name string
		args map[string]any
	}{
		{
			name: "missing url",
			args: map[string]any{
				"method": "GET",
			},
		},
		{
			name: "missing method",
			args: map[string]any{
				"url": "http://example.com",
			},
		},
		{
			name: "invalid url type",
			args: map[string]any{
				"url":    123,
				"method": "GET",
			},
		},
		{
			name: "invalid method type",
			args: map[string]any{
				"url":    "http://example.com",
				"method": 123,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := httpTool.Execute(context.Background(), tt.args)

			assert.Error(t, err)
			assert.True(t, errors.Is(err, ErrInvalidArguments))
			assert.Nil(t, result)
		})
	}
}

func TestHTTPRequestTool_ToTypesTool(t *testing.T) {
	httpTool := NewHTTPRequestTool(HTTPRequestConfig{})
	typesTool := httpTool.ToTypesTool()

	assert.Equal(t, "http_request", typesTool.Name)
	assert.NotEmpty(t, typesTool.Description)
	assert.Equal(t, "object", typesTool.Parameters.Type)
	assert.Contains(t, typesTool.Parameters.Required, "url")
	assert.Contains(t, typesTool.Parameters.Required, "method")
}

func TestNewShellTool(t *testing.T) {
	shellTool := NewShellTool(ShellToolConfig{
		AllowedCommands: []string{"ls", "pwd"},
	})

	assert.NotNil(t, shellTool)
	assert.Equal(t, "shell", shellTool.GetName())
}

func TestShellTool_Execute_NotImplemented(t *testing.T) {
	shellTool := NewShellTool(ShellToolConfig{
		AllowedCommands: []string{"ls"},
	})

	result, err := shellTool.Execute(context.Background(), map[string]any{
		"command": "ls",
	})

	// 由于安全原因，Shell 工具未实现
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestShellTool_Execute_CommandNotAllowed(t *testing.T) {
	shellTool := NewShellTool(ShellToolConfig{
		AllowedCommands: []string{"ls"},
	})

	result, err := shellTool.Execute(context.Background(), map[string]any{
		"command": "rm -rf /",
	})

	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidArguments))
	assert.Nil(t, result)
}

func TestShellTool_isCommandAllowed(t *testing.T) {
	shellTool := NewShellTool(ShellToolConfig{
		AllowedCommands: []string{"ls", "pwd", "echo"},
	})

	tests := []struct {
		name     string
		command  string
		expected bool
	}{
		{name: "allowed simple", command: "ls", expected: true},
		{name: "allowed with args", command: "ls -la", expected: true},
		{name: "allowed echo", command: "echo hello", expected: true},
		{name: "not allowed", command: "rm file", expected: false},
		{name: "empty", command: "", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shellTool.isCommandAllowed(tt.command)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNewJSONPlaceholderTool(t *testing.T) {
	tool := NewJSONPlaceholderTool()
	assert.NotNil(t, tool)
	assert.Equal(t, "jsonplaceholder", tool.GetName())
}

func TestJSONPlaceholderTool_GetParameters(t *testing.T) {
	tool := NewJSONPlaceholderTool()
	params := tool.GetParameters()

	assert.Equal(t, "object", params.Type)
	assert.Contains(t, params.Required, "resource")
}

func TestJSONPlaceholderTool_ToTypesTool(t *testing.T) {
	tool := NewJSONPlaceholderTool()
	typesTool := tool.ToTypesTool()

	assert.Equal(t, "jsonplaceholder", typesTool.Name)
	assert.NotEmpty(t, typesTool.Description)
}

// 注意：以下测试需要网络连接，可能会失败
// 如果在 CI/CD 环境中，建议跳过或使用 mock

func TestJSONPlaceholderTool_Execute_RealAPI(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping network test in short mode")
	}

	tool := NewJSONPlaceholderTool()

	result, err := tool.Execute(context.Background(), map[string]any{
		"resource": "posts",
		"id":       1.0,
	})

	if err != nil {
		// 网络错误可以接受
		t.Logf("Network test failed (acceptable): %v", err)
		return
	}

	assert.NotNil(t, result)
	// 验证返回的数据结构
	resultMap, ok := result.(map[string]any)
	if ok {
		assert.NotNil(t, resultMap)
		// JSONPlaceholder 返回的 post 包含 id, title, body 等字段
	}
}

func TestJSONPlaceholderTool_Execute_InvalidArgs(t *testing.T) {
	tool := NewJSONPlaceholderTool()

	tests := []struct {
		name string
		args map[string]any
	}{
		{
			name: "missing resource",
			args: map[string]any{},
		},
		{
			name: "invalid resource type",
			args: map[string]any{
				"resource": 123,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tool.Execute(context.Background(), tt.args)

			assert.Error(t, err)
			assert.True(t, errors.Is(err, ErrInvalidArguments))
			assert.Nil(t, result)
		})
	}
}
