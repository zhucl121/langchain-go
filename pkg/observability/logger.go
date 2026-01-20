package observability

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"
	
	"go.opentelemetry.io/otel/trace"
)

// LogLevel 日志级别
type LogLevel int

const (
	LogLevelDebug LogLevel = iota
	LogLevelInfo
	LogLevelWarn
	LogLevelError
)

// String 返回日志级别字符串
func (l LogLevel) String() string {
	switch l {
	case LogLevelDebug:
		return "DEBUG"
	case LogLevelInfo:
		return "INFO"
	case LogLevelWarn:
		return "WARN"
	case LogLevelError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// ToSlogLevel 转换为 slog.Level
func (l LogLevel) ToSlogLevel() slog.Level {
	switch l {
	case LogLevelDebug:
		return slog.LevelDebug
	case LogLevelInfo:
		return slog.LevelInfo
	case LogLevelWarn:
		return slog.LevelWarn
	case LogLevelError:
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// LoggerConfig 日志配置
type LoggerConfig struct {
	// Level 日志级别
	Level LogLevel
	
	// Format 日志格式: "json" 或 "text"
	Format string
	
	// Output 输出目标: "stdout", "stderr", "file"
	Output string
	
	// FilePath 日志文件路径（当 Output 为 "file" 时）
	FilePath string
	
	// EnableTraceID 是否自动记录 TraceID
	EnableTraceID bool
	
	// EnableCaller 是否记录调用者信息
	EnableCaller bool
	
	// AddSource 是否添加源代码位置
	AddSource bool
	
	// Attributes 默认属性
	Attributes map[string]any
}

// DefaultLoggerConfig 返回默认配置
func DefaultLoggerConfig() LoggerConfig {
	return LoggerConfig{
		Level:         LogLevelInfo,
		Format:        "json",
		Output:        "stdout",
		EnableTraceID: true,
		EnableCaller:  false,
		AddSource:     false,
		Attributes:    make(map[string]any),
	}
}

// Logger 结构化日志器接口
type Logger interface {
	// Debug 调试日志
	Debug(msg string, fields ...Field)
	
	// Info 信息日志
	Info(msg string, fields ...Field)
	
	// Warn 警告日志
	Warn(msg string, fields ...Field)
	
	// Error 错误日志
	Error(msg string, fields ...Field)
	
	// With 创建带有额外字段的子 Logger
	With(fields ...Field) Logger
	
	// WithContext 创建带有上下文的子 Logger（自动提取 TraceID）
	WithContext(ctx context.Context) Logger
}

// Field 日志字段
type Field struct {
	Key   string
	Value any
}

// F 创建日志字段的便捷函数
func F(key string, value any) Field {
	return Field{Key: key, Value: value}
}

// String 创建字符串字段
func String(key, value string) Field {
	return Field{Key: key, Value: value}
}

// Int 创建整数字段
func Int(key string, value int) Field {
	return Field{Key: key, Value: value}
}

// Int64 创建 int64 字段
func Int64(key string, value int64) Field {
	return Field{Key: key, Value: value}
}

// Float64 创建 float64 字段
func Float64(key string, value float64) Field {
	return Field{Key: key, Value: value}
}

// Bool 创建布尔字段
func Bool(key string, value bool) Field {
	return Field{Key: key, Value: value}
}

// Duration 创建时长字段
func Duration(key string, value time.Duration) Field {
	return Field{Key: key, Value: value}
}

// Error 创建错误字段
func Err(err error) Field {
	return Field{Key: "error", Value: err}
}

// Any 创建任意类型字段
func Any(key string, value any) Field {
	return Field{Key: key, Value: value}
}

// slogLogger slog 日志实现
type slogLogger struct {
	logger     *slog.Logger
	config     LoggerConfig
	attributes []slog.Attr
}

// NewLogger 创建日志器
func NewLogger(config LoggerConfig) (Logger, error) {
	// 创建输出目标
	var output io.Writer
	switch config.Output {
	case "stdout":
		output = os.Stdout
	case "stderr":
		output = os.Stderr
	case "file":
		if config.FilePath == "" {
			return nil, fmt.Errorf("file path is required when output is 'file'")
		}
		
		// 确保目录存在
		dir := filepath.Dir(config.FilePath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create log directory: %w", err)
		}
		
		// 打开文件
		file, err := os.OpenFile(config.FilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file: %w", err)
		}
		output = file
	default:
		output = os.Stdout
	}
	
	// 创建 Handler
	var handler slog.Handler
	opts := &slog.HandlerOptions{
		Level:     config.Level.ToSlogLevel(),
		AddSource: config.AddSource,
	}
	
	switch config.Format {
	case "json":
		handler = slog.NewJSONHandler(output, opts)
	case "text":
		handler = slog.NewTextHandler(output, opts)
	default:
		handler = slog.NewJSONHandler(output, opts)
	}
	
	// 创建 logger
	logger := slog.New(handler)
	
	// 添加默认属性
	var attrs []slog.Attr
	for k, v := range config.Attributes {
		attrs = append(attrs, slog.Any(k, v))
	}
	
	return &slogLogger{
		logger:     logger,
		config:     config,
		attributes: attrs,
	}, nil
}

// Debug 调试日志
func (l *slogLogger) Debug(msg string, fields ...Field) {
	l.log(slog.LevelDebug, msg, fields...)
}

// Info 信息日志
func (l *slogLogger) Info(msg string, fields ...Field) {
	l.log(slog.LevelInfo, msg, fields...)
}

// Warn 警告日志
func (l *slogLogger) Warn(msg string, fields ...Field) {
	l.log(slog.LevelWarn, msg, fields...)
}

// Error 错误日志
func (l *slogLogger) Error(msg string, fields ...Field) {
	l.log(slog.LevelError, msg, fields...)
}

// log 内部日志方法
func (l *slogLogger) log(level slog.Level, msg string, fields ...Field) {
	attrs := make([]any, 0, len(l.attributes)+len(fields))
	
	// 添加默认属性
	for _, attr := range l.attributes {
		attrs = append(attrs, attr)
	}
	
	// 添加字段
	for _, f := range fields {
		attrs = append(attrs, slog.Any(f.Key, f.Value))
	}
	
	l.logger.Log(context.Background(), level, msg, attrs...)
}

// With 创建带有额外字段的子 Logger
func (l *slogLogger) With(fields ...Field) Logger {
	attrs := make([]slog.Attr, 0, len(l.attributes)+len(fields))
	
	// 复制现有属性
	attrs = append(attrs, l.attributes...)
	
	// 添加新字段
	for _, f := range fields {
		attrs = append(attrs, slog.Any(f.Key, f.Value))
	}
	
	return &slogLogger{
		logger:     l.logger,
		config:     l.config,
		attributes: attrs,
	}
}

// WithContext 创建带有上下文的子 Logger
func (l *slogLogger) WithContext(ctx context.Context) Logger {
	if !l.config.EnableTraceID {
		return l
	}
	
	// 提取 TraceID 和 SpanID
	span := trace.SpanFromContext(ctx)
	if !span.SpanContext().IsValid() {
		return l
	}
	
	traceID := span.SpanContext().TraceID().String()
	spanID := span.SpanContext().SpanID().String()
	
	return l.With(
		F("trace_id", traceID),
		F("span_id", spanID),
	)
}

// GlobalLogger 全局日志器
var globalLogger Logger

// InitGlobalLogger 初始化全局日志器
func InitGlobalLogger(config LoggerConfig) error {
	logger, err := NewLogger(config)
	if err != nil {
		return err
	}
	globalLogger = logger
	return nil
}

// GetGlobalLogger 获取全局日志器
func GetGlobalLogger() Logger {
	if globalLogger == nil {
		// 使用默认配置创建
		logger, _ := NewLogger(DefaultLoggerConfig())
		globalLogger = logger
	}
	return globalLogger
}

// Debug 全局调试日志
func Debug(msg string, fields ...Field) {
	GetGlobalLogger().Debug(msg, fields...)
}

// Info 全局信息日志
func Info(msg string, fields ...Field) {
	GetGlobalLogger().Info(msg, fields...)
}

// Warn 全局警告日志
func Warn(msg string, fields ...Field) {
	GetGlobalLogger().Warn(msg, fields...)
}

// Error 全局错误日志
func Error(msg string, fields ...Field) {
	GetGlobalLogger().Error(msg, fields...)
}

// WithFields 创建带有字段的全局日志器
func WithFields(fields ...Field) Logger {
	return GetGlobalLogger().With(fields...)
}

// WithContext 创建带有上下文的全局日志器
func WithContext(ctx context.Context) Logger {
	return GetGlobalLogger().WithContext(ctx)
}

// LogOperation 日志操作辅助函数
func LogOperation(ctx context.Context, operation string, fn func() error) error {
	logger := GetGlobalLogger().WithContext(ctx)
	
	logger.Debug("operation started",
		F("operation", operation),
	)
	
	start := time.Now()
	err := fn()
	duration := time.Since(start)
	
	if err != nil {
		logger.Error("operation failed",
			F("operation", operation),
			F("duration_ms", duration.Milliseconds()),
			Err(err),
		)
		return err
	}
	
	logger.Info("operation completed",
		F("operation", operation),
		F("duration_ms", duration.Milliseconds()),
	)
	
	return nil
}

// LogLLMCall 记录 LLM 调用
func LogLLMCall(ctx context.Context, provider, model string, fn func() (string, error)) (string, error) {
	logger := GetGlobalLogger().WithContext(ctx)
	
	logger.Debug("llm call started",
		F("provider", provider),
		F("model", model),
	)
	
	start := time.Now()
	result, err := fn()
	duration := time.Since(start)
	
	if err != nil {
		logger.Error("llm call failed",
			F("provider", provider),
			F("model", model),
			F("duration_ms", duration.Milliseconds()),
			Err(err),
		)
		return "", err
	}
	
	// 截断响应避免日志过大
	truncated := result
	if len(truncated) > 500 {
		truncated = truncated[:500] + "..."
	}
	
	logger.Info("llm call completed",
		F("provider", provider),
		F("model", model),
		F("duration_ms", duration.Milliseconds()),
		F("response_length", len(result)),
		F("response_preview", truncated),
	)
	
	return result, nil
}

// LogRAGQuery 记录 RAG 查询
func LogRAGQuery(ctx context.Context, query string, docCount int, duration time.Duration) {
	logger := GetGlobalLogger().WithContext(ctx)
	
	logger.Info("rag query completed",
		F("query", query),
		F("document_count", docCount),
		F("duration_ms", duration.Milliseconds()),
	)
}

// LogToolCall 记录工具调用
func LogToolCall(ctx context.Context, toolName string, input any, output any, err error) {
	logger := GetGlobalLogger().WithContext(ctx)
	
	if err != nil {
		logger.Error("tool call failed",
			F("tool_name", toolName),
			F("input", input),
			Err(err),
		)
	} else {
		logger.Debug("tool call completed",
			F("tool_name", toolName),
			F("input", input),
			F("output", output),
		)
	}
}

// LogAgentStep 记录 Agent 步骤
func LogAgentStep(ctx context.Context, agentType string, step int, action string, err error) {
	logger := GetGlobalLogger().WithContext(ctx)
	
	if err != nil {
		logger.Error("agent step failed",
			F("agent_type", agentType),
			F("step", step),
			F("action", action),
			Err(err),
		)
	} else {
		logger.Debug("agent step completed",
			F("agent_type", agentType),
			F("step", step),
			F("action", action),
		)
	}
}
