package observability

import (
	"context"
	"fmt"
	"time"
	
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
)

// TracerConfig OpenTelemetry 追踪配置
type TracerConfig struct {
	// ServiceName 服务名称
	ServiceName string
	
	// ServiceVersion 服务版本
	ServiceVersion string
	
	// Environment 环境 (dev/staging/prod)
	Environment string
	
	// ExporterType 导出器类型: "otlp-grpc", "otlp-http", "jaeger", "zipkin"
	ExporterType string
	
	// Endpoint 追踪后端地址
	Endpoint string
	
	// SampleRate 采样率 (0.0-1.0)
	SampleRate float64
	
	// Attributes 自定义属性
	Attributes map[string]string
}

// TracerProvider OpenTelemetry 追踪提供者
type TracerProvider struct {
	provider *sdktrace.TracerProvider
	tracer   trace.Tracer
	config   TracerConfig
}

// NewTracerProvider 创建追踪提供者
func NewTracerProvider(config TracerConfig) (*TracerProvider, error) {
	// 设置默认值
	if config.ServiceName == "" {
		config.ServiceName = "langchain-go"
	}
	if config.ServiceVersion == "" {
		config.ServiceVersion = "1.0.0"
	}
	if config.Environment == "" {
		config.Environment = "development"
	}
	if config.ExporterType == "" {
		config.ExporterType = "otlp-grpc"
	}
	if config.Endpoint == "" {
		config.Endpoint = "localhost:4317"
	}
	if config.SampleRate == 0 {
		config.SampleRate = 1.0 // 默认全采样
	}
	
	// 创建资源
	res, err := resource.New(context.Background(),
		resource.WithAttributes(
			semconv.ServiceName(config.ServiceName),
			semconv.ServiceVersion(config.ServiceVersion),
			attribute.String("environment", config.Environment),
		),
		resource.WithAttributes(customAttributes(config.Attributes)...),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}
	
	// 创建导出器
	exporter, err := createExporter(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create exporter: %w", err)
	}
	
	// 创建采样器
	sampler := createSampler(config.SampleRate)
	
	// 创建追踪提供者
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
	)
	
	// 设置全局追踪提供者
	otel.SetTracerProvider(provider)
	
	// 获取 Tracer
	tracer := provider.Tracer(config.ServiceName)
	
	return &TracerProvider{
		provider: provider,
		tracer:   tracer,
		config:   config,
	}, nil
}

// createExporter 创建导出器
func createExporter(config TracerConfig) (sdktrace.SpanExporter, error) {
	ctx := context.Background()
	
	switch config.ExporterType {
	case "otlp-grpc":
		client := otlptracegrpc.NewClient(
			otlptracegrpc.WithEndpoint(config.Endpoint),
			otlptracegrpc.WithInsecure(), // 生产环境应使用 TLS
		)
		return otlptrace.New(ctx, client)
		
	case "otlp-http":
		client := otlptracehttp.NewClient(
			otlptracehttp.WithEndpoint(config.Endpoint),
			otlptracehttp.WithInsecure(), // 生产环境应使用 TLS
		)
		return otlptrace.New(ctx, client)
		
	default:
		return nil, fmt.Errorf("unsupported exporter type: %s", config.ExporterType)
	}
}

// createSampler 创建采样器
func createSampler(rate float64) sdktrace.Sampler {
	if rate >= 1.0 {
		return sdktrace.AlwaysSample()
	} else if rate <= 0.0 {
		return sdktrace.NeverSample()
	}
	return sdktrace.TraceIDRatioBased(rate)
}

// customAttributes 转换自定义属性
func customAttributes(attrs map[string]string) []attribute.KeyValue {
	if len(attrs) == 0 {
		return nil
	}
	
	result := make([]attribute.KeyValue, 0, len(attrs))
	for k, v := range attrs {
		result = append(result, attribute.String(k, v))
	}
	return result
}

// Shutdown 关闭追踪提供者
func (tp *TracerProvider) Shutdown(ctx context.Context) error {
	if tp.provider != nil {
		return tp.provider.Shutdown(ctx)
	}
	return nil
}

// GetTracer 获取 Tracer
func (tp *TracerProvider) GetTracer() trace.Tracer {
	return tp.tracer
}

// StartSpan 开始一个新的 Span
func (tp *TracerProvider) StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return tp.tracer.Start(ctx, name, opts...)
}

// SpanHelper Span 辅助工具
type SpanHelper struct {
	span trace.Span
}

// NewSpanHelper 创建 Span 辅助工具
func NewSpanHelper(span trace.Span) *SpanHelper {
	return &SpanHelper{span: span}
}

// SetAttribute 设置属性
func (sh *SpanHelper) SetAttribute(key string, value any) {
	switch v := value.(type) {
	case string:
		sh.span.SetAttributes(attribute.String(key, v))
	case int:
		sh.span.SetAttributes(attribute.Int(key, v))
	case int64:
		sh.span.SetAttributes(attribute.Int64(key, v))
	case float64:
		sh.span.SetAttributes(attribute.Float64(key, v))
	case bool:
		sh.span.SetAttributes(attribute.Bool(key, v))
	default:
		sh.span.SetAttributes(attribute.String(key, fmt.Sprintf("%v", v)))
	}
}

// SetAttributes 批量设置属性
func (sh *SpanHelper) SetAttributes(attrs map[string]any) {
	for k, v := range attrs {
		sh.SetAttribute(k, v)
	}
}

// RecordError 记录错误
func (sh *SpanHelper) RecordError(err error) {
	if err != nil {
		sh.span.RecordError(err)
		sh.span.SetStatus(codes.Error, err.Error())
	}
}

// RecordEvent 记录事件
func (sh *SpanHelper) RecordEvent(name string, attrs ...attribute.KeyValue) {
	sh.span.AddEvent(name, trace.WithAttributes(attrs...))
}

// SetSuccess 设置成功状态
func (sh *SpanHelper) SetSuccess() {
	sh.span.SetStatus(codes.Ok, "success")
}

// End 结束 Span
func (sh *SpanHelper) End() {
	sh.span.End()
}

// TraceOperation 追踪操作的辅助函数
func TraceOperation(ctx context.Context, tracer trace.Tracer, operationName string, fn func(ctx context.Context, span *SpanHelper) error) error {
	ctx, span := tracer.Start(ctx, operationName)
	helper := NewSpanHelper(span)
	defer helper.End()
	
	startTime := time.Now()
	err := fn(ctx, helper)
	duration := time.Since(startTime)
	
	// 记录执行时间
	helper.SetAttribute("duration_ms", duration.Milliseconds())
	
	if err != nil {
		helper.RecordError(err)
		return err
	}
	
	helper.SetSuccess()
	return nil
}

// CommonAttributes 通用属性常量
const (
	// LLM 相关
	AttrLLMProvider    = "llm.provider"
	AttrLLMModel       = "llm.model"
	AttrLLMPrompt      = "llm.prompt"
	AttrLLMResponse    = "llm.response"
	AttrLLMTokensInput = "llm.tokens.input"
	AttrLLMTokensOutput = "llm.tokens.output"
	AttrLLMTokensTotal = "llm.tokens.total"
	
	// Agent 相关
	AttrAgentType      = "agent.type"
	AttrAgentAction    = "agent.action"
	AttrAgentTool      = "agent.tool"
	AttrAgentStep      = "agent.step"
	AttrAgentIteration = "agent.iteration"
	
	// RAG 相关
	AttrRAGQuery       = "rag.query"
	AttrRAGDocCount    = "rag.document_count"
	AttrRAGVectorStore = "rag.vector_store"
	AttrRAGEmbedding   = "rag.embedding"
	AttrRAGReranker    = "rag.reranker"
	
	// Chain 相关
	AttrChainName      = "chain.name"
	AttrChainType      = "chain.type"
	AttrChainInput     = "chain.input"
	AttrChainOutput    = "chain.output"
	
	// Memory 相关
	AttrMemoryType     = "memory.type"
	AttrMemoryMessages = "memory.messages"
	
	// Tool 相关
	AttrToolName       = "tool.name"
	AttrToolInput      = "tool.input"
	AttrToolOutput     = "tool.output"
)

// TraceLLMCall 追踪 LLM 调用
func TraceLLMCall(ctx context.Context, tracer trace.Tracer, provider, model string, fn func(ctx context.Context) (string, error)) (string, error) {
	var result string
	err := TraceOperation(ctx, tracer, "llm.call", func(ctx context.Context, span *SpanHelper) error {
		span.SetAttribute(AttrLLMProvider, provider)
		span.SetAttribute(AttrLLMModel, model)
		
		var err error
		result, err = fn(ctx)
		
		if err == nil && len(result) > 0 {
			// 只记录前1000个字符，避免 Span 过大
			truncated := result
			if len(truncated) > 1000 {
				truncated = truncated[:1000] + "..."
			}
			span.SetAttribute(AttrLLMResponse, truncated)
		}
		
		return err
	})
	
	return result, err
}

// TraceAgentStep 追踪 Agent 步骤
func TraceAgentStep(ctx context.Context, tracer trace.Tracer, agentType string, step int, fn func(ctx context.Context) error) error {
	return TraceOperation(ctx, tracer, fmt.Sprintf("agent.step.%d", step), func(ctx context.Context, span *SpanHelper) error {
		span.SetAttribute(AttrAgentType, agentType)
		span.SetAttribute(AttrAgentStep, step)
		return fn(ctx)
	})
}

// TraceToolCall 追踪工具调用
func TraceToolCall(ctx context.Context, tracer trace.Tracer, toolName string, fn func(ctx context.Context) (any, error)) (any, error) {
	var result any
	err := TraceOperation(ctx, tracer, "tool.call", func(ctx context.Context, span *SpanHelper) error {
		span.SetAttribute(AttrToolName, toolName)
		
		var err error
		result, err = fn(ctx)
		
		if err == nil && result != nil {
			span.SetAttribute(AttrToolOutput, fmt.Sprintf("%v", result))
		}
		
		return err
	})
	
	return result, err
}

// TraceRAGQuery 追踪 RAG 查询
func TraceRAGQuery(ctx context.Context, tracer trace.Tracer, query string, fn func(ctx context.Context) (int, error)) (int, error) {
	var docCount int
	err := TraceOperation(ctx, tracer, "rag.query", func(ctx context.Context, span *SpanHelper) error {
		span.SetAttribute(AttrRAGQuery, query)
		
		var err error
		docCount, err = fn(ctx)
		
		if err == nil {
			span.SetAttribute(AttrRAGDocCount, docCount)
		}
		
		return err
	})
	
	return docCount, err
}
