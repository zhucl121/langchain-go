package observability

import (
	"context"
	"fmt"
	"time"
	
	"go.opentelemetry.io/otel/trace"
)

// ObservabilityContext 可观测性上下文
type ObservabilityContext struct {
	Tracer  trace.Tracer
	Logger  Logger
	Metrics *MetricsCollector
}

// NewObservabilityContext 创建可观测性上下文
func NewObservabilityContext(tracer trace.Tracer, logger Logger, metrics *MetricsCollector) *ObservabilityContext {
	return &ObservabilityContext{
		Tracer:  tracer,
		Logger:  logger,
		Metrics: metrics,
	}
}

// 使用字符串作为上下文键
const (
	obsContextKey = "observability_context"
)

// WithObservability 将可观测性上下文添加到 context
func WithObservability(ctx context.Context, obs *ObservabilityContext) context.Context {
	return context.WithValue(ctx, obsContextKey, obs)
}

// FromContext 从 context 获取可观测性上下文
func FromContext(ctx context.Context) (*ObservabilityContext, bool) {
	obs, ok := ctx.Value(obsContextKey).(*ObservabilityContext)
	return obs, ok
}

// MustFromContext 从 context 获取可观测性上下文（失败则 panic）
func MustFromContext(ctx context.Context) *ObservabilityContext {
	obs, ok := FromContext(ctx)
	if !ok {
		panic("observability context not found in context")
	}
	return obs
}

// StartSpan 从 context 启动一个新的 Span
func StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	obs, ok := FromContext(ctx)
	if !ok || obs.Tracer == nil {
		// 如果没有 Tracer，返回无操作的 span
		return ctx, trace.SpanFromContext(ctx)
	}
	
	return obs.Tracer.Start(ctx, name, opts...)
}

// RecordMetric 记录指标
func RecordMetric(ctx context.Context, metricType string, labels map[string]string, value float64) {
	obs, ok := FromContext(ctx)
	if !ok || obs.Metrics == nil {
		return
	}
	
	// 根据指标类型记录
	// 这是一个简化版本，实际使用时应该根据具体需求扩展
}

// LogFromContext 从 context 获取 Logger
func LogFromContext(ctx context.Context) Logger {
	obs, ok := FromContext(ctx)
	if !ok || obs.Logger == nil {
		return GetGlobalLogger().WithContext(ctx)
	}
	
	return obs.Logger.WithContext(ctx)
}

// OperationTracker 操作追踪器
type OperationTracker struct {
	ctx        context.Context
	span       trace.Span
	spanHelper *SpanHelper
	logger     Logger
	metrics    *MetricsCollector
	
	operationName string
	startTime     time.Time
	
	// 标签
	labels map[string]string
}

// StartOperation 开始追踪一个操作
func StartOperation(ctx context.Context, operationName string, labels map[string]string) *OperationTracker {
	obs, _ := FromContext(ctx)
	
	// 创建 span
	newCtx, span := StartSpan(ctx, operationName)
	spanHelper := NewSpanHelper(span)
	
	// 获取 logger
	logger := LogFromContext(ctx)
	
	// 设置标签
	if len(labels) > 0 {
		spanHelper.SetAttributes(convertLabelsToAny(labels))
	}
	
	// 记录开始日志
	fields := make([]Field, 0, len(labels)+1)
	fields = append(fields, F("operation", operationName))
	for k, v := range labels {
		fields = append(fields, F(k, v))
	}
	logger.Debug("operation started", fields...)
	
	tracker := &OperationTracker{
		ctx:           newCtx,
		span:          span,
		spanHelper:    spanHelper,
		logger:        logger,
		operationName: operationName,
		startTime:     time.Now(),
		labels:        labels,
	}
	
	if obs != nil {
		tracker.metrics = obs.Metrics
	}
	
	return tracker
}

// Context 返回操作的 context
func (ot *OperationTracker) Context() context.Context {
	return ot.ctx
}

// SetAttribute 设置属性
func (ot *OperationTracker) SetAttribute(key string, value any) {
	if ot.spanHelper != nil {
		ot.spanHelper.SetAttribute(key, value)
	}
}

// SetAttributes 批量设置属性
func (ot *OperationTracker) SetAttributes(attrs map[string]any) {
	if ot.spanHelper != nil {
		ot.spanHelper.SetAttributes(attrs)
	}
}

// RecordError 记录错误
func (ot *OperationTracker) RecordError(err error) {
	if err == nil {
		return
	}
	
	if ot.spanHelper != nil {
		ot.spanHelper.RecordError(err)
	}
	
	if ot.logger != nil {
		fields := []Field{
			F("operation", ot.operationName),
			Err(err),
		}
		for k, v := range ot.labels {
			fields = append(fields, F(k, v))
		}
		ot.logger.Error("operation failed", fields...)
	}
}

// End 结束操作
func (ot *OperationTracker) End(err error) {
	duration := time.Since(ot.startTime)
	
	// 设置执行时间
	if ot.spanHelper != nil {
		ot.spanHelper.SetAttribute("duration_ms", duration.Milliseconds())
		
		if err != nil {
			ot.spanHelper.RecordError(err)
		} else {
			ot.spanHelper.SetSuccess()
		}
		
		ot.spanHelper.End()
	}
	
	// 记录日志
	if ot.logger != nil {
		fields := []Field{
			F("operation", ot.operationName),
			F("duration_ms", duration.Milliseconds()),
		}
		for k, v := range ot.labels {
			fields = append(fields, F(k, v))
		}
		
		if err != nil {
			fields = append(fields, Err(err))
			ot.logger.Error("operation failed", fields...)
		} else {
			ot.logger.Info("operation completed", fields...)
		}
	}
	
	// 记录指标
	if ot.metrics != nil {
		// 根据操作类型记录不同的指标
		// 这是一个简化版本，实际使用时应该根据具体需求扩展
	}
}

// TrackOperation 追踪一个操作
func TrackOperation(ctx context.Context, operationName string, labels map[string]string, fn func(ctx context.Context) error) error {
	tracker := StartOperation(ctx, operationName, labels)
	defer func() {
		// 从 panic 中恢复并记录
		if r := recover(); r != nil {
			err := fmt.Errorf("panic in operation %s: %v", operationName, r)
			tracker.RecordError(err)
			tracker.End(err)
			panic(r) // 重新 panic
		}
	}()
	
	err := fn(tracker.Context())
	tracker.End(err)
	return err
}

// convertLabelsToAny 转换标签为 any map
func convertLabelsToAny(labels map[string]string) map[string]any {
	result := make(map[string]any, len(labels))
	for k, v := range labels {
		result[k] = v
	}
	return result
}

// LLMOperationTracker LLM 操作追踪器
type LLMOperationTracker struct {
	*OperationTracker
	provider      string
	model         string
	inputTokens   int
	outputTokens  int
	totalTokens   int
}

// StartLLMOperation 开始追踪 LLM 操作
func StartLLMOperation(ctx context.Context, provider, model string) *LLMOperationTracker {
	labels := map[string]string{
		"provider": provider,
		"model":    model,
	}
	
	tracker := StartOperation(ctx, "llm.call", labels)
	
	return &LLMOperationTracker{
		OperationTracker: tracker,
		provider:         provider,
		model:            model,
	}
}

// SetTokens 设置 token 数量
func (lt *LLMOperationTracker) SetTokens(inputTokens, outputTokens int) {
	lt.inputTokens = inputTokens
	lt.outputTokens = outputTokens
	lt.totalTokens = inputTokens + outputTokens
	
	if lt.spanHelper != nil {
		lt.spanHelper.SetAttribute(AttrLLMTokensInput, inputTokens)
		lt.spanHelper.SetAttribute(AttrLLMTokensOutput, outputTokens)
		lt.spanHelper.SetAttribute(AttrLLMTokensTotal, lt.totalTokens)
	}
}

// End 结束 LLM 操作
func (lt *LLMOperationTracker) End(err error) {
	duration := time.Since(lt.startTime)
	
	// 调用父类的 End
	lt.OperationTracker.End(err)
	
	// 记录 LLM 特定的指标
	if lt.metrics != nil {
		lt.metrics.RecordLLMCall(lt.provider, lt.model, duration, err)
		
		if lt.totalTokens > 0 {
			lt.metrics.RecordLLMTokens(lt.provider, lt.model, lt.inputTokens, lt.outputTokens)
		}
	}
}

// RAGOperationTracker RAG 操作追踪器
type RAGOperationTracker struct {
	*OperationTracker
	vectorStore  string
	query        string
	docCount     int
}

// StartRAGOperation 开始追踪 RAG 操作
func StartRAGOperation(ctx context.Context, vectorStore, query string) *RAGOperationTracker {
	labels := map[string]string{
		"vector_store": vectorStore,
	}
	
	tracker := StartOperation(ctx, "rag.query", labels)
	tracker.SetAttribute(AttrRAGQuery, query)
	
	return &RAGOperationTracker{
		OperationTracker: tracker,
		vectorStore:      vectorStore,
		query:            query,
	}
}

// SetDocumentCount 设置检索的文档数量
func (rt *RAGOperationTracker) SetDocumentCount(count int) {
	rt.docCount = count
	
	if rt.spanHelper != nil {
		rt.spanHelper.SetAttribute(AttrRAGDocCount, count)
	}
}

// End 结束 RAG 操作
func (rt *RAGOperationTracker) End(err error) {
	duration := time.Since(rt.startTime)
	
	// 调用父类的 End
	rt.OperationTracker.End(err)
	
	// 记录 RAG 特定的指标
	if rt.metrics != nil {
		rt.metrics.RecordRAGQuery(rt.vectorStore, duration, rt.docCount, err)
	}
}

// ToolOperationTracker Tool 操作追踪器
type ToolOperationTracker struct {
	*OperationTracker
	toolName string
	input    any
	output   any
}

// StartToolOperation 开始追踪 Tool 操作
func StartToolOperation(ctx context.Context, toolName string, input any) *ToolOperationTracker {
	labels := map[string]string{
		"tool_name": toolName,
	}
	
	tracker := StartOperation(ctx, "tool.call", labels)
	tracker.SetAttribute(AttrToolName, toolName)
	tracker.SetAttribute(AttrToolInput, fmt.Sprintf("%v", input))
	
	return &ToolOperationTracker{
		OperationTracker: tracker,
		toolName:         toolName,
		input:            input,
	}
}

// SetOutput 设置输出
func (tt *ToolOperationTracker) SetOutput(output any) {
	tt.output = output
	
	if tt.spanHelper != nil {
		tt.spanHelper.SetAttribute(AttrToolOutput, fmt.Sprintf("%v", output))
	}
}

// End 结束 Tool 操作
func (tt *ToolOperationTracker) End(err error) {
	duration := time.Since(tt.startTime)
	
	// 调用父类的 End
	tt.OperationTracker.End(err)
	
	// 记录 Tool 特定的指标
	if tt.metrics != nil {
		tt.metrics.RecordToolCall(tt.toolName, duration, err)
	}
	
	// 记录 Tool 调用日志
	if tt.logger != nil {
		LogToolCall(tt.ctx, tt.toolName, tt.input, tt.output, err)
	}
}

// AgentOperationTracker Agent 操作追踪器
type AgentOperationTracker struct {
	*OperationTracker
	agentType string
	step      int
	iteration int
}

// StartAgentOperation 开始追踪 Agent 操作
func StartAgentOperation(ctx context.Context, agentType string, step int) *AgentOperationTracker {
	labels := map[string]string{
		"agent_type": agentType,
	}
	
	tracker := StartOperation(ctx, fmt.Sprintf("agent.step.%d", step), labels)
	tracker.SetAttribute(AttrAgentType, agentType)
	tracker.SetAttribute(AttrAgentStep, step)
	
	return &AgentOperationTracker{
		OperationTracker: tracker,
		agentType:        agentType,
		step:             step,
	}
}

// SetIteration 设置迭代次数
func (at *AgentOperationTracker) SetIteration(iteration int) {
	at.iteration = iteration
	
	if at.spanHelper != nil {
		at.spanHelper.SetAttribute(AttrAgentIteration, iteration)
	}
}

// SetAction 设置动作
func (at *AgentOperationTracker) SetAction(action string) {
	if at.spanHelper != nil {
		at.spanHelper.SetAttribute(AttrAgentAction, action)
	}
}

// End 结束 Agent 操作
func (at *AgentOperationTracker) End(err error) {
	duration := time.Since(at.startTime)
	
	// 调用父类的 End
	at.OperationTracker.End(err)
	
	// 记录 Agent 特定的指标
	if at.metrics != nil {
		at.metrics.RecordAgentStep(at.agentType, duration, err)
		
		if at.iteration > 0 {
			at.metrics.RecordAgentIteration(at.agentType)
		}
	}
}

// ChainOperationTracker Chain 操作追踪器
type ChainOperationTracker struct {
	*OperationTracker
	chainName string
}

// StartChainOperation 开始追踪 Chain 操作
func StartChainOperation(ctx context.Context, chainName string) *ChainOperationTracker {
	labels := map[string]string{
		"chain_name": chainName,
	}
	
	tracker := StartOperation(ctx, "chain.execute", labels)
	tracker.SetAttribute(AttrChainName, chainName)
	
	return &ChainOperationTracker{
		OperationTracker: tracker,
		chainName:        chainName,
	}
}

// End 结束 Chain 操作
func (ct *ChainOperationTracker) End(err error) {
	duration := time.Since(ct.startTime)
	
	// 调用父类的 End
	ct.OperationTracker.End(err)
	
	// 记录 Chain 特定的指标
	if ct.metrics != nil {
		ct.metrics.RecordChainExecution(ct.chainName, duration, err)
	}
}
