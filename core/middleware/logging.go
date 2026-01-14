package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"
)

// Logger 是日志接口。
type Logger interface {
	// Log 记录日志
	Log(level string, message string, fields map[string]any)
}

// DefaultLogger 是默认日志实现。
type DefaultLogger struct{}

// Log 实现 Logger 接口。
func (l *DefaultLogger) Log(level string, message string, fields map[string]any) {
	fieldsJSON, _ := json.Marshal(fields)
	log.Printf("[%s] %s %s", level, message, string(fieldsJSON))
}

// LoggingMiddleware 是日志记录中间件。
type LoggingMiddleware struct {
	logger        Logger
	logInput      bool
	logOutput     bool
	logDuration   bool
	logError      bool
	includeFields map[string]any
}

// NewLoggingMiddleware 创建日志中间件。
//
// 返回：
//   - *LoggingMiddleware: 日志中间件实例
//
func NewLoggingMiddleware() *LoggingMiddleware {
	return &LoggingMiddleware{
		logger:        &DefaultLogger{},
		logInput:      true,
		logOutput:     true,
		logDuration:   true,
		logError:      true,
		includeFields: make(map[string]any),
	}
}

// WithLogger 设置自定义日志器。
func (lm *LoggingMiddleware) WithLogger(logger Logger) *LoggingMiddleware {
	lm.logger = logger
	return lm
}

// WithLogInput 设置是否记录输入。
func (lm *LoggingMiddleware) WithLogInput(enabled bool) *LoggingMiddleware {
	lm.logInput = enabled
	return lm
}

// WithLogOutput 设置是否记录输出。
func (lm *LoggingMiddleware) WithLogOutput(enabled bool) *LoggingMiddleware {
	lm.logOutput = enabled
	return lm
}

// WithLogDuration 设置是否记录耗时。
func (lm *LoggingMiddleware) WithLogDuration(enabled bool) *LoggingMiddleware {
	lm.logDuration = enabled
	return lm
}

// WithLogError 设置是否记录错误。
func (lm *LoggingMiddleware) WithLogError(enabled bool) *LoggingMiddleware {
	lm.logError = enabled
	return lm
}

// WithField 添加额外字段。
func (lm *LoggingMiddleware) WithField(key string, value any) *LoggingMiddleware {
	lm.includeFields[key] = value
	return lm
}

// Process 实现 Middleware 接口。
func (lm *LoggingMiddleware) Process(ctx context.Context, input any, next NextFunc) (any, error) {
	start := time.Now()

	// 记录请求开始
	fields := make(map[string]any)
	for k, v := range lm.includeFields {
		fields[k] = v
	}

	// 添加中间件名称
	if name := GetMiddlewareNameFromContext(ctx); name != "" {
		fields["middleware"] = name
	}

	if lm.logInput {
		fields["input"] = fmt.Sprintf("%v", input)
	}

	lm.logger.Log("INFO", "Processing started", fields)

	// 执行下一个处理函数
	result, err := next(ctx, input)

	// 计算耗时
	duration := time.Since(start)

	// 记录结果
	resultFields := make(map[string]any)
	for k, v := range fields {
		resultFields[k] = v
	}

	if lm.logDuration {
		resultFields["duration_ms"] = duration.Milliseconds()
	}

	if err != nil {
		if lm.logError {
			resultFields["error"] = err.Error()
			lm.logger.Log("ERROR", "Processing failed", resultFields)
		}
	} else {
		if lm.logOutput {
			resultFields["output"] = fmt.Sprintf("%v", result)
		}
		lm.logger.Log("INFO", "Processing completed", resultFields)
	}

	return result, err
}

// PerformanceMiddleware 是性能监控中间件。
type PerformanceMiddleware struct {
	slowThreshold time.Duration
	logger        Logger
	onSlow        func(duration time.Duration, input any, output any)
}

// NewPerformanceMiddleware 创建性能监控中间件。
//
// 参数：
//   - slowThreshold: 慢查询阈值
//
// 返回：
//   - *PerformanceMiddleware: 性能监控中间件实例
//
func NewPerformanceMiddleware(slowThreshold time.Duration) *PerformanceMiddleware {
	return &PerformanceMiddleware{
		slowThreshold: slowThreshold,
		logger:        &DefaultLogger{},
	}
}

// WithLogger 设置日志器。
func (pm *PerformanceMiddleware) WithLogger(logger Logger) *PerformanceMiddleware {
	pm.logger = logger
	return pm
}

// OnSlow 设置慢查询回调。
func (pm *PerformanceMiddleware) OnSlow(callback func(duration time.Duration, input any, output any)) *PerformanceMiddleware {
	pm.onSlow = callback
	return pm
}

// Process 实现 Middleware 接口。
func (pm *PerformanceMiddleware) Process(ctx context.Context, input any, next NextFunc) (any, error) {
	start := time.Now()

	result, err := next(ctx, input)

	duration := time.Since(start)

	// 检查是否超过阈值
	if duration > pm.slowThreshold {
		fields := map[string]any{
			"duration_ms": duration.Milliseconds(),
			"threshold_ms": pm.slowThreshold.Milliseconds(),
			"input":       fmt.Sprintf("%v", input),
		}

		if err == nil {
			fields["output"] = fmt.Sprintf("%v", result)
		} else {
			fields["error"] = err.Error()
		}

		pm.logger.Log("WARN", "Slow processing detected", fields)

		// 调用回调
		if pm.onSlow != nil {
			pm.onSlow(duration, input, result)
		}
	}

	return result, err
}

// MetricsMiddleware 是指标收集中间件。
type MetricsMiddleware struct {
	totalRequests   int64
	successRequests int64
	failedRequests  int64
	totalDuration   time.Duration
	onMetricsUpdate func(metrics *Metrics)
}

// Metrics 是统计指标。
type Metrics struct {
	TotalRequests   int64
	SuccessRequests int64
	FailedRequests  int64
	TotalDuration   time.Duration
	AverageDuration time.Duration
	SuccessRate     float64
}

// NewMetricsMiddleware 创建指标收集中间件。
func NewMetricsMiddleware() *MetricsMiddleware {
	return &MetricsMiddleware{}
}

// OnMetricsUpdate 设置指标更新回调。
func (mm *MetricsMiddleware) OnMetricsUpdate(callback func(metrics *Metrics)) *MetricsMiddleware {
	mm.onMetricsUpdate = callback
	return mm
}

// Process 实现 Middleware 接口。
func (mm *MetricsMiddleware) Process(ctx context.Context, input any, next NextFunc) (any, error) {
	start := time.Now()
	mm.totalRequests++

	result, err := next(ctx, input)

	duration := time.Since(start)
	mm.totalDuration += duration

	if err != nil {
		mm.failedRequests++
	} else {
		mm.successRequests++
	}

	// 触发回调
	if mm.onMetricsUpdate != nil {
		mm.onMetricsUpdate(mm.GetMetrics())
	}

	return result, err
}

// GetMetrics 获取当前指标。
func (mm *MetricsMiddleware) GetMetrics() *Metrics {
	avgDuration := time.Duration(0)
	if mm.totalRequests > 0 {
		avgDuration = mm.totalDuration / time.Duration(mm.totalRequests)
	}

	successRate := 0.0
	if mm.totalRequests > 0 {
		successRate = float64(mm.successRequests) / float64(mm.totalRequests) * 100
	}

	return &Metrics{
		TotalRequests:   mm.totalRequests,
		SuccessRequests: mm.successRequests,
		FailedRequests:  mm.failedRequests,
		TotalDuration:   mm.totalDuration,
		AverageDuration: avgDuration,
		SuccessRate:     successRate,
	}
}

// Reset 重置指标。
func (mm *MetricsMiddleware) Reset() {
	mm.totalRequests = 0
	mm.successRequests = 0
	mm.failedRequests = 0
	mm.totalDuration = 0
}
