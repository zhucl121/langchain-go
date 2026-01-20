package observability

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func TestLogLevel(t *testing.T) {
	tests := []struct {
		name     string
		level    LogLevel
		expected string
	}{
		{"Debug", LogLevelDebug, "DEBUG"},
		{"Info", LogLevelInfo, "INFO"},
		{"Warn", LogLevelWarn, "WARN"},
		{"Error", LogLevelError, "ERROR"},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.level.String())
		})
	}
}

func TestNewLogger(t *testing.T) {
	tests := []struct {
		name    string
		config  LoggerConfig
		wantErr bool
	}{
		{
			name: "stdout json",
			config: LoggerConfig{
				Level:  LogLevelInfo,
				Format: "json",
				Output: "stdout",
			},
			wantErr: false,
		},
		{
			name: "stderr text",
			config: LoggerConfig{
				Level:  LogLevelDebug,
				Format: "text",
				Output: "stderr",
			},
			wantErr: false,
		},
		{
			name: "file without path",
			config: LoggerConfig{
				Level:  LogLevelInfo,
				Format: "json",
				Output: "file",
			},
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, err := NewLogger(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, logger)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, logger)
			}
		})
	}
}

func TestLoggerBasicLogging(t *testing.T) {
	// 创建临时文件
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.log")
	
	config := LoggerConfig{
		Level:    LogLevelDebug,
		Format:   "json",
		Output:   "file",
		FilePath: logFile,
	}
	
	logger, err := NewLogger(config)
	require.NoError(t, err)
	
	// 测试各级别日志
	logger.Debug("debug message", String("key", "value"))
	logger.Info("info message", Int("count", 42))
	logger.Warn("warn message", Bool("flag", true))
	logger.Error("error message", Float64("score", 3.14))
	
	// 读取日志文件
	content, err := os.ReadFile(logFile)
	require.NoError(t, err)
	
	// 验证日志内容
	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	assert.Len(t, lines, 4)
	
	// 验证 JSON 格式
	for _, line := range lines {
		var logEntry map[string]any
		err := json.Unmarshal([]byte(line), &logEntry)
		assert.NoError(t, err)
		assert.Contains(t, logEntry, "time")
		assert.Contains(t, logEntry, "level")
		assert.Contains(t, logEntry, "msg")
	}
}

func TestLoggerWith(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.log")
	
	config := LoggerConfig{
		Level:    LogLevelInfo,
		Format:   "json",
		Output:   "file",
		FilePath: logFile,
	}
	
	logger, err := NewLogger(config)
	require.NoError(t, err)
	
	// 创建子 logger
	childLogger := logger.With(
		String("service", "test"),
		String("version", "1.0.0"),
	)
	
	childLogger.Info("test message", String("extra", "data"))
	
	// 读取日志文件
	content, err := os.ReadFile(logFile)
	require.NoError(t, err)
	
	// 验证包含默认字段
	var logEntry map[string]any
	err = json.Unmarshal(content, &logEntry)
	require.NoError(t, err)
	
	assert.Equal(t, "test", logEntry["service"])
	assert.Equal(t, "1.0.0", logEntry["version"])
	assert.Equal(t, "data", logEntry["extra"])
}

func TestLoggerWithContext(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.log")
	
	config := LoggerConfig{
		Level:         LogLevelInfo,
		Format:        "json",
		Output:        "file",
		FilePath:      logFile,
		EnableTraceID: true,
	}
	
	logger, err := NewLogger(config)
	require.NoError(t, err)
	
	// 创建 TracerProvider
	tp := sdktrace.NewTracerProvider()
	otel.SetTracerProvider(tp)
	defer tp.Shutdown(context.Background())
	
	tracer := tp.Tracer("test")
	
	// 创建 span
	ctx, span := tracer.Start(context.Background(), "test-operation")
	defer span.End()
	
	// 使用带 context 的 logger
	contextLogger := logger.WithContext(ctx)
	contextLogger.Info("test message with trace")
	
	// 读取日志文件
	content, err := os.ReadFile(logFile)
	require.NoError(t, err)
	
	// 验证包含 TraceID
	var logEntry map[string]any
	err = json.Unmarshal(content, &logEntry)
	require.NoError(t, err)
	
	assert.Contains(t, logEntry, "trace_id")
	assert.Contains(t, logEntry, "span_id")
}

func TestFieldCreators(t *testing.T) {
	tests := []struct {
		name     string
		field    Field
		expected Field
	}{
		{
			name:     "String",
			field:    String("key", "value"),
			expected: Field{Key: "key", Value: "value"},
		},
		{
			name:     "Int",
			field:    Int("key", 42),
			expected: Field{Key: "key", Value: 42},
		},
		{
			name:     "Int64",
			field:    Int64("key", int64(100)),
			expected: Field{Key: "key", Value: int64(100)},
		},
		{
			name:     "Float64",
			field:    Float64("key", 3.14),
			expected: Field{Key: "key", Value: 3.14},
		},
		{
			name:     "Bool",
			field:    Bool("key", true),
			expected: Field{Key: "key", Value: true},
		},
		{
			name:     "Duration",
			field:    Duration("key", time.Second),
			expected: Field{Key: "key", Value: time.Second},
		},
		{
			name:     "Error",
			field:    Err(errors.New("test error")),
			expected: Field{Key: "error", Value: errors.New("test error")},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected.Key, tt.field.Key)
		})
	}
}

func TestLogOperation(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.log")
	
	config := LoggerConfig{
		Level:    LogLevelDebug,
		Format:   "json",
		Output:   "file",
		FilePath: logFile,
	}
	
	logger, err := NewLogger(config)
	require.NoError(t, err)
	
	// 设置全局 logger
	globalLogger = logger
	
	// 测试成功操作
	err = LogOperation(context.Background(), "test_operation", func() error {
		time.Sleep(10 * time.Millisecond)
		return nil
	})
	assert.NoError(t, err)
	
	// 测试失败操作
	testErr := errors.New("operation failed")
	err = LogOperation(context.Background(), "failing_operation", func() error {
		return testErr
	})
	assert.Error(t, err)
	assert.Equal(t, testErr, err)
	
	// 读取日志文件
	content, err := os.ReadFile(logFile)
	require.NoError(t, err)
	
	// 验证日志条目
	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	assert.GreaterOrEqual(t, len(lines), 3) // started, completed, error
}

func TestLogLLMCall(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.log")
	
	config := LoggerConfig{
		Level:    LogLevelInfo,
		Format:   "json",
		Output:   "file",
		FilePath: logFile,
	}
	
	logger, err := NewLogger(config)
	require.NoError(t, err)
	
	// 设置全局 logger
	globalLogger = logger
	
	// 测试成功的 LLM 调用
	result, err := LogLLMCall(context.Background(), "openai", "gpt-4", func() (string, error) {
		time.Sleep(10 * time.Millisecond)
		return "This is a test response", nil
	})
	assert.NoError(t, err)
	assert.Equal(t, "This is a test response", result)
	
	// 测试失败的 LLM 调用
	testErr := errors.New("llm error")
	_, err = LogLLMCall(context.Background(), "openai", "gpt-4", func() (string, error) {
		return "", testErr
	})
	assert.Error(t, err)
	
	// 读取日志文件
	content, err := os.ReadFile(logFile)
	require.NoError(t, err)
	
	// 验证日志包含 LLM 相关字段
	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	assert.GreaterOrEqual(t, len(lines), 2)
	
	// 验证第一条日志（成功）
	var logEntry map[string]any
	err = json.Unmarshal([]byte(lines[0]), &logEntry)
	require.NoError(t, err)
	assert.Equal(t, "openai", logEntry["provider"])
	assert.Equal(t, "gpt-4", logEntry["model"])
}

func TestGlobalLogger(t *testing.T) {
	// 重置全局 logger
	globalLogger = nil
	
	// 测试自动初始化
	logger := GetGlobalLogger()
	assert.NotNil(t, logger)
	
	// 测试全局日志函数
	config := LoggerConfig{
		Level:  LogLevelInfo,
		Format: "json",
		Output: "stdout",
	}
	
	logger, err := NewLogger(config)
	require.NoError(t, err)
	
	globalLogger = logger
	
	// 使用全局函数
	Info("test info")
	Warn("test warn", String("key", "value"))
	Error("test error", Err(errors.New("test")))
}

func TestLogHelperFunctions(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.log")
	
	config := LoggerConfig{
		Level:    LogLevelDebug,
		Format:   "json",
		Output:   "file",
		FilePath: logFile,
	}
	
	logger, err := NewLogger(config)
	require.NoError(t, err)
	
	globalLogger = logger
	ctx := context.Background()
	
	// 测试各种日志辅助函数
	LogRAGQuery(ctx, "test query", 10, 100*time.Millisecond)
	LogToolCall(ctx, "calculator", "2+2", "4", nil)
	LogAgentStep(ctx, "react", 1, "think", nil)
	
	// 读取日志文件
	content, err := os.ReadFile(logFile)
	require.NoError(t, err)
	
	// 验证日志条目
	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	assert.Equal(t, 3, len(lines))
}

func TestDefaultLoggerConfig(t *testing.T) {
	config := DefaultLoggerConfig()
	
	assert.Equal(t, LogLevelInfo, config.Level)
	assert.Equal(t, "json", config.Format)
	assert.Equal(t, "stdout", config.Output)
	assert.True(t, config.EnableTraceID)
	assert.False(t, config.EnableCaller)
	assert.False(t, config.AddSource)
	assert.NotNil(t, config.Attributes)
}

func TestLoggerAttributes(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.log")
	
	config := LoggerConfig{
		Level:    LogLevelInfo,
		Format:   "json",
		Output:   "file",
		FilePath: logFile,
		Attributes: map[string]any{
			"service":     "test-service",
			"environment": "testing",
		},
	}
	
	logger, err := NewLogger(config)
	require.NoError(t, err)
	
	logger.Info("test message")
	
	// 读取日志文件
	content, err := os.ReadFile(logFile)
	require.NoError(t, err)
	
	// 验证默认属性
	var logEntry map[string]any
	err = json.Unmarshal(content, &logEntry)
	require.NoError(t, err)
	
	assert.Equal(t, "test-service", logEntry["service"])
	assert.Equal(t, "testing", logEntry["environment"])
}
