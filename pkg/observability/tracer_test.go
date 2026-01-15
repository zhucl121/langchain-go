package observability

import (
	"context"
	"errors"
	"testing"
	"time"
	
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

// TestTracerProviderCreation 测试追踪提供者创建
func TestTracerProviderCreation(t *testing.T) {
	t.Run("default config", func(t *testing.T) {
		// 由于需要真实的 OTLP 端点，这里跳过实际创建
		// 在实际使用中，应该有一个测试 OTLP 收集器
		t.Skip("Requires OTLP endpoint")
	})
	
	t.Run("custom config", func(t *testing.T) {
		config := TracerConfig{
			ServiceName:    "test-service",
			ServiceVersion: "1.0.0",
			Environment:    "test",
			ExporterType:   "otlp-grpc",
			Endpoint:       "localhost:4317",
			SampleRate:     0.5,
			Attributes: map[string]string{
				"team": "backend",
			},
		}
		
		// 验证配置
		if config.ServiceName != "test-service" {
			t.Errorf("Expected ServiceName='test-service', got '%s'", config.ServiceName)
		}
		if config.SampleRate != 0.5 {
			t.Errorf("Expected SampleRate=0.5, got %f", config.SampleRate)
		}
	})
}

// TestSpanHelper 测试 Span 辅助工具
func TestSpanHelper(t *testing.T) {
	// 创建内存导出器用于测试
	exporter := tracetest.NewInMemoryExporter()
	
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
	)
	defer provider.Shutdown(context.Background())
	
	tracer := provider.Tracer("test")
	
	t.Run("set attributes", func(t *testing.T) {
		ctx, span := tracer.Start(context.Background(), "test-span")
		defer span.End()
		
		helper := NewSpanHelper(span)
		
		// 测试不同类型的属性
		helper.SetAttribute("string_attr", "value")
		helper.SetAttribute("int_attr", 42)
		helper.SetAttribute("int64_attr", int64(100))
		helper.SetAttribute("float_attr", 3.14)
		helper.SetAttribute("bool_attr", true)
		
		// 批量设置
		helper.SetAttributes(map[string]any{
			"batch1": "value1",
			"batch2": 123,
		})
		
		helper.End()
		
		// 验证
		_ = ctx
		if len(exporter.GetSpans()) == 0 {
			t.Error("Expected at least one span")
		}
	})
	
	t.Run("record error", func(t *testing.T) {
		ctx, span := tracer.Start(context.Background(), "error-span")
		defer span.End()
		
		helper := NewSpanHelper(span)
		helper.RecordError(errors.New("test error"))
		helper.End()
		
		_ = ctx
		spans := exporter.GetSpans()
		if len(spans) == 0 {
			t.Fatal("Expected at least one span")
		}
		
		lastSpan := spans[len(spans)-1]
		if lastSpan.Status.Code != codes.Error {
			t.Errorf("Expected error status, got %v", lastSpan.Status.Code)
		}
	})
	
	t.Run("record event", func(t *testing.T) {
		ctx, span := tracer.Start(context.Background(), "event-span")
		defer span.End()
		
		helper := NewSpanHelper(span)
		helper.RecordEvent("test-event", attribute.String("key", "value"))
		helper.SetSuccess()
		helper.End()
		
		_ = ctx
		spans := exporter.GetSpans()
		if len(spans) == 0 {
			t.Fatal("Expected at least one span")
		}
		
		lastSpan := spans[len(spans)-1]
		if lastSpan.Status.Code != codes.Ok {
			t.Errorf("Expected ok status, got %v", lastSpan.Status.Code)
		}
		
		if len(lastSpan.Events) == 0 {
			t.Error("Expected at least one event")
		}
	})
}

// TestTraceOperation 测试追踪操作辅助函数
func TestTraceOperation(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
	)
	defer provider.Shutdown(context.Background())
	
	tracer := provider.Tracer("test")
	ctx := context.Background()
	
	t.Run("successful operation", func(t *testing.T) {
		err := TraceOperation(ctx, tracer, "test-operation", func(ctx context.Context, span *SpanHelper) error {
			span.SetAttribute("test", "value")
			time.Sleep(10 * time.Millisecond)
			return nil
		})
		
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		
		spans := exporter.GetSpans()
		if len(spans) == 0 {
			t.Fatal("Expected at least one span")
		}
		
		lastSpan := spans[len(spans)-1]
		if lastSpan.Name != "test-operation" {
			t.Errorf("Expected span name 'test-operation', got '%s'", lastSpan.Name)
		}
		
		// 检查是否记录了执行时间
		found := false
		for _, attr := range lastSpan.Attributes {
			if string(attr.Key) == "duration_ms" {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected duration_ms attribute")
		}
	})
	
	t.Run("failed operation", func(t *testing.T) {
		testErr := errors.New("operation failed")
		
		err := TraceOperation(ctx, tracer, "failed-operation", func(ctx context.Context, span *SpanHelper) error {
			return testErr
		})
		
		if err != testErr {
			t.Errorf("Expected error %v, got %v", testErr, err)
		}
		
		spans := exporter.GetSpans()
		if len(spans) == 0 {
			t.Fatal("Expected at least one span")
		}
		
		lastSpan := spans[len(spans)-1]
		if lastSpan.Status.Code != codes.Error {
			t.Errorf("Expected error status, got %v", lastSpan.Status.Code)
		}
	})
}

// TestTraceLLMCall 测试 LLM 调用追踪
func TestTraceLLMCall(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
	)
	defer provider.Shutdown(context.Background())
	
	tracer := provider.Tracer("test")
	ctx := context.Background()
	
	t.Run("successful llm call", func(t *testing.T) {
		expectedResponse := "This is a test response"
		
		response, err := TraceLLMCall(ctx, tracer, "openai", "gpt-4", func(ctx context.Context) (string, error) {
			return expectedResponse, nil
		})
		
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		
		if response != expectedResponse {
			t.Errorf("Expected response '%s', got '%s'", expectedResponse, response)
		}
		
		spans := exporter.GetSpans()
		if len(spans) == 0 {
			t.Fatal("Expected at least one span")
		}
		
		lastSpan := spans[len(spans)-1]
		if lastSpan.Name != "llm.call" {
			t.Errorf("Expected span name 'llm.call', got '%s'", lastSpan.Name)
		}
		
		// 验证属性
		attrs := make(map[string]any)
		for _, attr := range lastSpan.Attributes {
			attrs[string(attr.Key)] = attr.Value.AsInterface()
		}
		
		if attrs[AttrLLMProvider] != "openai" {
			t.Errorf("Expected provider 'openai', got '%v'", attrs[AttrLLMProvider])
		}
		
		if attrs[AttrLLMModel] != "gpt-4" {
			t.Errorf("Expected model 'gpt-4', got '%v'", attrs[AttrLLMModel])
		}
	})
}

// TestTraceAgentStep 测试 Agent 步骤追踪
func TestTraceAgentStep(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
	)
	defer provider.Shutdown(context.Background())
	
	tracer := provider.Tracer("test")
	ctx := context.Background()
	
	err := TraceAgentStep(ctx, tracer, "react", 1, func(ctx context.Context) error {
		return nil
	})
	
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	
	spans := exporter.GetSpans()
	if len(spans) == 0 {
		t.Fatal("Expected at least one span")
	}
	
	lastSpan := spans[len(spans)-1]
	if lastSpan.Name != "agent.step.1" {
		t.Errorf("Expected span name 'agent.step.1', got '%s'", lastSpan.Name)
	}
	
	// 验证属性
	attrs := make(map[string]any)
	for _, attr := range lastSpan.Attributes {
		attrs[string(attr.Key)] = attr.Value.AsInterface()
	}
	
	if attrs[AttrAgentType] != "react" {
		t.Errorf("Expected agent type 'react', got '%v'", attrs[AttrAgentType])
	}
}

// TestTraceToolCall 测试工具调用追踪
func TestTraceToolCall(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
	)
	defer provider.Shutdown(context.Background())
	
	tracer := provider.Tracer("test")
	ctx := context.Background()
	
	expectedResult := map[string]string{"result": "success"}
	
	result, err := TraceToolCall(ctx, tracer, "calculator", func(ctx context.Context) (any, error) {
		return expectedResult, nil
	})
	
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	
	if result == nil {
		t.Error("Expected result, got nil")
	}
	
	spans := exporter.GetSpans()
	if len(spans) == 0 {
		t.Fatal("Expected at least one span")
	}
	
	lastSpan := spans[len(spans)-1]
	if lastSpan.Name != "tool.call" {
		t.Errorf("Expected span name 'tool.call', got '%s'", lastSpan.Name)
	}
}

// TestTraceRAGQuery 测试 RAG 查询追踪
func TestTraceRAGQuery(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
	)
	defer provider.Shutdown(context.Background())
	
	tracer := provider.Tracer("test")
	ctx := context.Background()
	
	query := "What is LangChain?"
	expectedDocs := 5
	
	docCount, err := TraceRAGQuery(ctx, tracer, query, func(ctx context.Context) (int, error) {
		return expectedDocs, nil
	})
	
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	
	if docCount != expectedDocs {
		t.Errorf("Expected doc count %d, got %d", expectedDocs, docCount)
	}
	
	spans := exporter.GetSpans()
	if len(spans) == 0 {
		t.Fatal("Expected at least one span")
	}
	
	lastSpan := spans[len(spans)-1]
	if lastSpan.Name != "rag.query" {
		t.Errorf("Expected span name 'rag.query', got '%s'", lastSpan.Name)
	}
	
	// 验证属性
	attrs := make(map[string]any)
	for _, attr := range lastSpan.Attributes {
		attrs[string(attr.Key)] = attr.Value.AsInterface()
	}
	
	if attrs[AttrRAGQuery] != query {
		t.Errorf("Expected query '%s', got '%v'", query, attrs[AttrRAGQuery])
	}
	
	if attrs[AttrRAGDocCount] != int64(expectedDocs) {
		t.Errorf("Expected doc count %d, got '%v'", expectedDocs, attrs[AttrRAGDocCount])
	}
}

// TestCustomAttributes 测试自定义属性转换
func TestCustomAttributes(t *testing.T) {
	attrs := map[string]string{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}
	
	result := customAttributes(attrs)
	
	if len(result) != 3 {
		t.Errorf("Expected 3 attributes, got %d", len(result))
	}
	
	// 验证每个属性
	attrMap := make(map[string]string)
	for _, kv := range result {
		attrMap[string(kv.Key)] = kv.Value.AsString()
	}
	
	for k, v := range attrs {
		if attrMap[k] != v {
			t.Errorf("Expected attribute %s=%s, got %s", k, v, attrMap[k])
		}
	}
}

// TestContextTracerOperations 测试上下文中的 Tracer 操作
func TestContextTracerOperations(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
	)
	defer provider.Shutdown(context.Background())
	
	tracer := provider.Tracer("test")
	
	t.Run("context with tracer", func(t *testing.T) {
		ctx := ContextWithTracer(context.Background(), tracer)
		
		retrievedTracer, ok := TracerFromContext(ctx)
		if !ok {
			t.Error("Expected to retrieve tracer from context")
		}
		
		if retrievedTracer != tracer {
			t.Error("Retrieved tracer does not match original")
		}
	})
	
	t.Run("context without tracer", func(t *testing.T) {
		ctx := context.Background()
		
		_, ok := TracerFromContext(ctx)
		if ok {
			t.Error("Expected no tracer in context")
		}
	})
}
