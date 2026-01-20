package observability

import (
	"context"
	"errors"
	"testing"
	"time"
	
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func TestObservabilityContext(t *testing.T) {
	// 创建 TracerProvider
	tp := sdktrace.NewTracerProvider()
	otel.SetTracerProvider(tp)
	defer tp.Shutdown(context.Background())
	
	tracer := tp.Tracer("test")
	
	// 创建 Logger
	logger, err := NewLogger(DefaultLoggerConfig())
	require.NoError(t, err)
	
	// 创建 MetricsCollector
	metrics := NewMetricsCollector(MetricsConfig{
		Namespace: "test",
		Subsystem: "test",
	})
	
	// 创建 ObservabilityContext
	obs := NewObservabilityContext(tracer, logger, metrics)
	assert.NotNil(t, obs)
	assert.NotNil(t, obs.Tracer)
	assert.NotNil(t, obs.Logger)
	assert.NotNil(t, obs.Metrics)
}

func TestWithObservability(t *testing.T) {
	tp := sdktrace.NewTracerProvider()
	defer tp.Shutdown(context.Background())
	
	tracer := tp.Tracer("test")
	logger, _ := NewLogger(DefaultLoggerConfig())
	metrics := NewMetricsCollector(MetricsConfig{})
	
	obs := NewObservabilityContext(tracer, logger, metrics)
	
	ctx := context.Background()
	ctx = WithObservability(ctx, obs)
	
	// 测试获取
	retrievedObs, ok := FromContext(ctx)
	assert.True(t, ok)
	assert.Equal(t, obs, retrievedObs)
}

func TestMustFromContext(t *testing.T) {
	tp := sdktrace.NewTracerProvider()
	defer tp.Shutdown(context.Background())
	
	tracer := tp.Tracer("test")
	logger, _ := NewLogger(DefaultLoggerConfig())
	metrics := NewMetricsCollector(MetricsConfig{})
	
	obs := NewObservabilityContext(tracer, logger, metrics)
	
	ctx := WithObservability(context.Background(), obs)
	
	// 测试成功获取
	retrievedObs := MustFromContext(ctx)
	assert.Equal(t, obs, retrievedObs)
	
	// 测试失败（应该 panic）
	assert.Panics(t, func() {
		MustFromContext(context.Background())
	})
}

func TestStartSpan(t *testing.T) {
	tp := sdktrace.NewTracerProvider()
	otel.SetTracerProvider(tp)
	defer tp.Shutdown(context.Background())
	
	tracer := tp.Tracer("test")
	logger, _ := NewLogger(DefaultLoggerConfig())
	metrics := NewMetricsCollector(MetricsConfig{})
	
	obs := NewObservabilityContext(tracer, logger, metrics)
	ctx := WithObservability(context.Background(), obs)
	
	// 启动 span
	newCtx, span := StartSpan(ctx, "test-operation")
	assert.NotNil(t, newCtx)
	assert.NotNil(t, span)
	
	spanContext := span.SpanContext()
	assert.True(t, spanContext.IsValid())
	
	span.End()
}

func TestLogFromContext(t *testing.T) {
	tp := sdktrace.NewTracerProvider()
	defer tp.Shutdown(context.Background())
	
	tracer := tp.Tracer("test")
	logger, _ := NewLogger(DefaultLoggerConfig())
	metrics := NewMetricsCollector(MetricsConfig{})
	
	obs := NewObservabilityContext(tracer, logger, metrics)
	ctx := WithObservability(context.Background(), obs)
	
	// 获取 logger
	retrievedLogger := LogFromContext(ctx)
	assert.NotNil(t, retrievedLogger)
}

func TestOperationTracker(t *testing.T) {
	tp := sdktrace.NewTracerProvider()
	otel.SetTracerProvider(tp)
	defer tp.Shutdown(context.Background())
	
	tracer := tp.Tracer("test")
	logger, _ := NewLogger(DefaultLoggerConfig())
	metrics := NewMetricsCollector(MetricsConfig{})
	
	obs := NewObservabilityContext(tracer, logger, metrics)
	ctx := WithObservability(context.Background(), obs)
	
	// 开始操作
	tracker := StartOperation(ctx, "test-operation", map[string]string{
		"key": "value",
	})
	assert.NotNil(t, tracker)
	assert.NotNil(t, tracker.Context())
	
	// 设置属性
	tracker.SetAttribute("count", 42)
	tracker.SetAttributes(map[string]any{
		"flag": true,
		"name": "test",
	})
	
	// 模拟操作
	time.Sleep(10 * time.Millisecond)
	
	// 结束操作
	tracker.End(nil)
}

func TestOperationTrackerWithError(t *testing.T) {
	tp := sdktrace.NewTracerProvider()
	otel.SetTracerProvider(tp)
	defer tp.Shutdown(context.Background())
	
	tracer := tp.Tracer("test")
	logger, _ := NewLogger(DefaultLoggerConfig())
	metrics := NewMetricsCollector(MetricsConfig{})
	
	obs := NewObservabilityContext(tracer, logger, metrics)
	ctx := WithObservability(context.Background(), obs)
	
	tracker := StartOperation(ctx, "failing-operation", nil)
	
	testErr := errors.New("test error")
	tracker.RecordError(testErr)
	tracker.End(testErr)
}

func TestTrackOperation(t *testing.T) {
	tp := sdktrace.NewTracerProvider()
	otel.SetTracerProvider(tp)
	defer tp.Shutdown(context.Background())
	
	tracer := tp.Tracer("test")
	logger, _ := NewLogger(DefaultLoggerConfig())
	metrics := NewMetricsCollector(MetricsConfig{})
	
	obs := NewObservabilityContext(tracer, logger, metrics)
	ctx := WithObservability(context.Background(), obs)
	
	// 测试成功操作
	err := TrackOperation(ctx, "test-operation", map[string]string{"key": "value"}, func(ctx context.Context) error {
		time.Sleep(10 * time.Millisecond)
		return nil
	})
	assert.NoError(t, err)
	
	// 测试失败操作
	testErr := errors.New("operation failed")
	err = TrackOperation(ctx, "failing-operation", nil, func(ctx context.Context) error {
		return testErr
	})
	assert.Error(t, err)
	assert.Equal(t, testErr, err)
}

func TestLLMOperationTracker(t *testing.T) {
	tp := sdktrace.NewTracerProvider()
	otel.SetTracerProvider(tp)
	defer tp.Shutdown(context.Background())
	
	tracer := tp.Tracer("test")
	logger, _ := NewLogger(DefaultLoggerConfig())
	metrics := NewMetricsCollector(MetricsConfig{})
	
	obs := NewObservabilityContext(tracer, logger, metrics)
	ctx := WithObservability(context.Background(), obs)
	
	// 开始 LLM 操作
	tracker := StartLLMOperation(ctx, "openai", "gpt-4")
	assert.NotNil(t, tracker)
	assert.Equal(t, "openai", tracker.provider)
	assert.Equal(t, "gpt-4", tracker.model)
	
	// 设置 tokens
	tracker.SetTokens(100, 50)
	assert.Equal(t, 100, tracker.inputTokens)
	assert.Equal(t, 50, tracker.outputTokens)
	assert.Equal(t, 150, tracker.totalTokens)
	
	// 结束操作
	time.Sleep(10 * time.Millisecond)
	tracker.End(nil)
}

func TestRAGOperationTracker(t *testing.T) {
	tp := sdktrace.NewTracerProvider()
	otel.SetTracerProvider(tp)
	defer tp.Shutdown(context.Background())
	
	tracer := tp.Tracer("test")
	logger, _ := NewLogger(DefaultLoggerConfig())
	metrics := NewMetricsCollector(MetricsConfig{})
	
	obs := NewObservabilityContext(tracer, logger, metrics)
	ctx := WithObservability(context.Background(), obs)
	
	// 开始 RAG 操作
	tracker := StartRAGOperation(ctx, "milvus", "test query")
	assert.NotNil(t, tracker)
	assert.Equal(t, "milvus", tracker.vectorStore)
	assert.Equal(t, "test query", tracker.query)
	
	// 设置文档数量
	tracker.SetDocumentCount(10)
	assert.Equal(t, 10, tracker.docCount)
	
	// 结束操作
	time.Sleep(10 * time.Millisecond)
	tracker.End(nil)
}

func TestToolOperationTracker(t *testing.T) {
	tp := sdktrace.NewTracerProvider()
	otel.SetTracerProvider(tp)
	defer tp.Shutdown(context.Background())
	
	tracer := tp.Tracer("test")
	logger, _ := NewLogger(DefaultLoggerConfig())
	metrics := NewMetricsCollector(MetricsConfig{})
	
	obs := NewObservabilityContext(tracer, logger, metrics)
	ctx := WithObservability(context.Background(), obs)
	
	// 开始 Tool 操作
	tracker := StartToolOperation(ctx, "calculator", "2+2")
	assert.NotNil(t, tracker)
	assert.Equal(t, "calculator", tracker.toolName)
	assert.Equal(t, "2+2", tracker.input)
	
	// 设置输出
	tracker.SetOutput("4")
	assert.Equal(t, "4", tracker.output)
	
	// 结束操作
	time.Sleep(10 * time.Millisecond)
	tracker.End(nil)
}

func TestAgentOperationTracker(t *testing.T) {
	tp := sdktrace.NewTracerProvider()
	otel.SetTracerProvider(tp)
	defer tp.Shutdown(context.Background())
	
	tracer := tp.Tracer("test")
	logger, _ := NewLogger(DefaultLoggerConfig())
	metrics := NewMetricsCollector(MetricsConfig{})
	
	obs := NewObservabilityContext(tracer, logger, metrics)
	ctx := WithObservability(context.Background(), obs)
	
	// 开始 Agent 操作
	tracker := StartAgentOperation(ctx, "react", 1)
	assert.NotNil(t, tracker)
	assert.Equal(t, "react", tracker.agentType)
	assert.Equal(t, 1, tracker.step)
	
	// 设置迭代和动作
	tracker.SetIteration(1)
	tracker.SetAction("think")
	
	// 结束操作
	time.Sleep(10 * time.Millisecond)
	tracker.End(nil)
}

func TestChainOperationTracker(t *testing.T) {
	tp := sdktrace.NewTracerProvider()
	otel.SetTracerProvider(tp)
	defer tp.Shutdown(context.Background())
	
	tracer := tp.Tracer("test")
	logger, _ := NewLogger(DefaultLoggerConfig())
	metrics := NewMetricsCollector(MetricsConfig{})
	
	obs := NewObservabilityContext(tracer, logger, metrics)
	ctx := WithObservability(context.Background(), obs)
	
	// 开始 Chain 操作
	tracker := StartChainOperation(ctx, "rag-chain")
	assert.NotNil(t, tracker)
	assert.Equal(t, "rag-chain", tracker.chainName)
	
	// 结束操作
	time.Sleep(10 * time.Millisecond)
	tracker.End(nil)
}

func TestOperationTrackerPanicRecovery(t *testing.T) {
	tp := sdktrace.NewTracerProvider()
	otel.SetTracerProvider(tp)
	defer tp.Shutdown(context.Background())
	
	tracer := tp.Tracer("test")
	logger, _ := NewLogger(DefaultLoggerConfig())
	metrics := NewMetricsCollector(MetricsConfig{})
	
	obs := NewObservabilityContext(tracer, logger, metrics)
	ctx := WithObservability(context.Background(), obs)
	
	// 测试 panic 恢复
	assert.Panics(t, func() {
		TrackOperation(ctx, "panic-operation", nil, func(ctx context.Context) error {
			panic("test panic")
		})
	})
}

func TestConvertLabelsToAny(t *testing.T) {
	labels := map[string]string{
		"key1": "value1",
		"key2": "value2",
	}
	
	result := convertLabelsToAny(labels)
	assert.Len(t, result, 2)
	assert.Equal(t, "value1", result["key1"])
	assert.Equal(t, "value2", result["key2"])
}
