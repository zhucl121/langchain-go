package observability

import (
	"net/http"
	"sync"
	"time"
	
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// MetricsConfig Prometheus 指标配置
type MetricsConfig struct {
	// Namespace 指标命名空间
	Namespace string
	
	// Subsystem 指标子系统
	Subsystem string
	
	// EnableDefaultMetrics 是否启用默认指标（Go runtime 等）
	EnableDefaultMetrics bool
	
	// HTTPPath HTTP 端点路径
	HTTPPath string
	
	// HTTPPort HTTP 端点端口
	HTTPPort string
}

// MetricsCollector 指标收集器
type MetricsCollector struct {
	config MetricsConfig
	
	// LLM 指标
	llmCallsTotal        *prometheus.CounterVec
	llmCallDuration      *prometheus.HistogramVec
	llmTokensTotal       *prometheus.CounterVec
	llmErrorsTotal       *prometheus.CounterVec
	
	// Agent 指标
	agentStepsTotal      *prometheus.CounterVec
	agentStepDuration    *prometheus.HistogramVec
	agentIterationsTotal *prometheus.CounterVec
	agentErrorsTotal     *prometheus.CounterVec
	
	// Tool 指标
	toolCallsTotal       *prometheus.CounterVec
	toolCallDuration     *prometheus.HistogramVec
	toolErrorsTotal      *prometheus.CounterVec
	
	// RAG 指标
	ragQueriesTotal      *prometheus.CounterVec
	ragQueryDuration     *prometheus.HistogramVec
	ragDocumentsRetrieved *prometheus.HistogramVec
	ragErrorsTotal       *prometheus.CounterVec
	
	// Chain 指标
	chainExecutionsTotal *prometheus.CounterVec
	chainDuration        *prometheus.HistogramVec
	chainErrorsTotal     *prometheus.CounterVec
	
	// Memory 指标
	memoryOperationsTotal *prometheus.CounterVec
	memorySize           *prometheus.GaugeVec
	
	registry *prometheus.Registry
	mu       sync.RWMutex
}

// NewMetricsCollector 创建指标收集器
func NewMetricsCollector(config MetricsConfig) *MetricsCollector {
	// 设置默认值
	if config.Namespace == "" {
		config.Namespace = "langchain"
	}
	if config.Subsystem == "" {
		config.Subsystem = "go"
	}
	if config.HTTPPath == "" {
		config.HTTPPath = "/metrics"
	}
	if config.HTTPPort == "" {
		config.HTTPPort = "9090"
	}
	
	registry := prometheus.NewRegistry()
	
	// 如果启用默认指标
	if config.EnableDefaultMetrics {
		registry.MustRegister(prometheus.NewGoCollector())
		registry.MustRegister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))
	}
	
	mc := &MetricsCollector{
		config:   config,
		registry: registry,
	}
	
	mc.initMetrics()
	return mc
}

// initMetrics 初始化指标
func (mc *MetricsCollector) initMetrics() {
	// LLM 指标
	mc.llmCallsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: mc.config.Namespace,
			Subsystem: mc.config.Subsystem,
			Name:      "llm_calls_total",
			Help:      "Total number of LLM calls",
		},
		[]string{"provider", "model", "status"},
	)
	
	mc.llmCallDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: mc.config.Namespace,
			Subsystem: mc.config.Subsystem,
			Name:      "llm_call_duration_seconds",
			Help:      "Duration of LLM calls in seconds",
			Buckets:   []float64{0.1, 0.5, 1, 2, 5, 10, 30, 60},
		},
		[]string{"provider", "model"},
	)
	
	mc.llmTokensTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: mc.config.Namespace,
			Subsystem: mc.config.Subsystem,
			Name:      "llm_tokens_total",
			Help:      "Total number of tokens processed",
		},
		[]string{"provider", "model", "type"}, // type: input, output
	)
	
	mc.llmErrorsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: mc.config.Namespace,
			Subsystem: mc.config.Subsystem,
			Name:      "llm_errors_total",
			Help:      "Total number of LLM errors",
		},
		[]string{"provider", "model", "error_type"},
	)
	
	// Agent 指标
	mc.agentStepsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: mc.config.Namespace,
			Subsystem: mc.config.Subsystem,
			Name:      "agent_steps_total",
			Help:      "Total number of agent steps",
		},
		[]string{"agent_type", "status"},
	)
	
	mc.agentStepDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: mc.config.Namespace,
			Subsystem: mc.config.Subsystem,
			Name:      "agent_step_duration_seconds",
			Help:      "Duration of agent steps in seconds",
			Buckets:   []float64{0.1, 0.5, 1, 2, 5, 10, 30},
		},
		[]string{"agent_type"},
	)
	
	mc.agentIterationsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: mc.config.Namespace,
			Subsystem: mc.config.Subsystem,
			Name:      "agent_iterations_total",
			Help:      "Total number of agent iterations",
		},
		[]string{"agent_type"},
	)
	
	mc.agentErrorsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: mc.config.Namespace,
			Subsystem: mc.config.Subsystem,
			Name:      "agent_errors_total",
			Help:      "Total number of agent errors",
		},
		[]string{"agent_type", "error_type"},
	)
	
	// Tool 指标
	mc.toolCallsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: mc.config.Namespace,
			Subsystem: mc.config.Subsystem,
			Name:      "tool_calls_total",
			Help:      "Total number of tool calls",
		},
		[]string{"tool_name", "status"},
	)
	
	mc.toolCallDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: mc.config.Namespace,
			Subsystem: mc.config.Subsystem,
			Name:      "tool_call_duration_seconds",
			Help:      "Duration of tool calls in seconds",
			Buckets:   []float64{0.01, 0.05, 0.1, 0.5, 1, 2, 5},
		},
		[]string{"tool_name"},
	)
	
	mc.toolErrorsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: mc.config.Namespace,
			Subsystem: mc.config.Subsystem,
			Name:      "tool_errors_total",
			Help:      "Total number of tool errors",
		},
		[]string{"tool_name", "error_type"},
	)
	
	// RAG 指标
	mc.ragQueriesTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: mc.config.Namespace,
			Subsystem: mc.config.Subsystem,
			Name:      "rag_queries_total",
			Help:      "Total number of RAG queries",
		},
		[]string{"vector_store", "status"},
	)
	
	mc.ragQueryDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: mc.config.Namespace,
			Subsystem: mc.config.Subsystem,
			Name:      "rag_query_duration_seconds",
			Help:      "Duration of RAG queries in seconds",
			Buckets:   []float64{0.01, 0.05, 0.1, 0.5, 1, 2},
		},
		[]string{"vector_store"},
	)
	
	mc.ragDocumentsRetrieved = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: mc.config.Namespace,
			Subsystem: mc.config.Subsystem,
			Name:      "rag_documents_retrieved",
			Help:      "Number of documents retrieved by RAG queries",
			Buckets:   []float64{1, 5, 10, 20, 50, 100},
		},
		[]string{"vector_store"},
	)
	
	mc.ragErrorsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: mc.config.Namespace,
			Subsystem: mc.config.Subsystem,
			Name:      "rag_errors_total",
			Help:      "Total number of RAG errors",
		},
		[]string{"vector_store", "error_type"},
	)
	
	// Chain 指标
	mc.chainExecutionsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: mc.config.Namespace,
			Subsystem: mc.config.Subsystem,
			Name:      "chain_executions_total",
			Help:      "Total number of chain executions",
		},
		[]string{"chain_name", "status"},
	)
	
	mc.chainDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: mc.config.Namespace,
			Subsystem: mc.config.Subsystem,
			Name:      "chain_duration_seconds",
			Help:      "Duration of chain executions in seconds",
			Buckets:   []float64{0.1, 0.5, 1, 2, 5, 10, 30},
		},
		[]string{"chain_name"},
	)
	
	mc.chainErrorsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: mc.config.Namespace,
			Subsystem: mc.config.Subsystem,
			Name:      "chain_errors_total",
			Help:      "Total number of chain errors",
		},
		[]string{"chain_name", "error_type"},
	)
	
	// Memory 指标
	mc.memoryOperationsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: mc.config.Namespace,
			Subsystem: mc.config.Subsystem,
			Name:      "memory_operations_total",
			Help:      "Total number of memory operations",
		},
		[]string{"memory_type", "operation"}, // operation: load, save, clear
	)
	
	mc.memorySize = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: mc.config.Namespace,
			Subsystem: mc.config.Subsystem,
			Name:      "memory_size_messages",
			Help:      "Current size of memory in messages",
		},
		[]string{"memory_type"},
	)
	
	// 注册所有指标
	mc.registry.MustRegister(
		mc.llmCallsTotal,
		mc.llmCallDuration,
		mc.llmTokensTotal,
		mc.llmErrorsTotal,
		mc.agentStepsTotal,
		mc.agentStepDuration,
		mc.agentIterationsTotal,
		mc.agentErrorsTotal,
		mc.toolCallsTotal,
		mc.toolCallDuration,
		mc.toolErrorsTotal,
		mc.ragQueriesTotal,
		mc.ragQueryDuration,
		mc.ragDocumentsRetrieved,
		mc.ragErrorsTotal,
		mc.chainExecutionsTotal,
		mc.chainDuration,
		mc.chainErrorsTotal,
		mc.memoryOperationsTotal,
		mc.memorySize,
	)
}

// RecordLLMCall 记录 LLM 调用
func (mc *MetricsCollector) RecordLLMCall(provider, model string, duration time.Duration, err error) {
	status := "success"
	if err != nil {
		status = "error"
		mc.llmErrorsTotal.WithLabelValues(provider, model, "unknown").Inc()
	}
	
	mc.llmCallsTotal.WithLabelValues(provider, model, status).Inc()
	mc.llmCallDuration.WithLabelValues(provider, model).Observe(duration.Seconds())
}

// RecordLLMTokens 记录 LLM token 使用
func (mc *MetricsCollector) RecordLLMTokens(provider, model string, inputTokens, outputTokens int) {
	mc.llmTokensTotal.WithLabelValues(provider, model, "input").Add(float64(inputTokens))
	mc.llmTokensTotal.WithLabelValues(provider, model, "output").Add(float64(outputTokens))
}

// RecordAgentStep 记录 Agent 步骤
func (mc *MetricsCollector) RecordAgentStep(agentType string, duration time.Duration, err error) {
	status := "success"
	if err != nil {
		status = "error"
		mc.agentErrorsTotal.WithLabelValues(agentType, "unknown").Inc()
	}
	
	mc.agentStepsTotal.WithLabelValues(agentType, status).Inc()
	mc.agentStepDuration.WithLabelValues(agentType).Observe(duration.Seconds())
}

// RecordAgentIteration 记录 Agent 迭代
func (mc *MetricsCollector) RecordAgentIteration(agentType string) {
	mc.agentIterationsTotal.WithLabelValues(agentType).Inc()
}

// RecordToolCall 记录工具调用
func (mc *MetricsCollector) RecordToolCall(toolName string, duration time.Duration, err error) {
	status := "success"
	if err != nil {
		status = "error"
		mc.toolErrorsTotal.WithLabelValues(toolName, "unknown").Inc()
	}
	
	mc.toolCallsTotal.WithLabelValues(toolName, status).Inc()
	mc.toolCallDuration.WithLabelValues(toolName).Observe(duration.Seconds())
}

// RecordRAGQuery 记录 RAG 查询
func (mc *MetricsCollector) RecordRAGQuery(vectorStore string, duration time.Duration, docCount int, err error) {
	status := "success"
	if err != nil {
		status = "error"
		mc.ragErrorsTotal.WithLabelValues(vectorStore, "unknown").Inc()
	}
	
	mc.ragQueriesTotal.WithLabelValues(vectorStore, status).Inc()
	mc.ragQueryDuration.WithLabelValues(vectorStore).Observe(duration.Seconds())
	mc.ragDocumentsRetrieved.WithLabelValues(vectorStore).Observe(float64(docCount))
}

// RecordChainExecution 记录 Chain 执行
func (mc *MetricsCollector) RecordChainExecution(chainName string, duration time.Duration, err error) {
	status := "success"
	if err != nil {
		status = "error"
		mc.chainErrorsTotal.WithLabelValues(chainName, "unknown").Inc()
	}
	
	mc.chainExecutionsTotal.WithLabelValues(chainName, status).Inc()
	mc.chainDuration.WithLabelValues(chainName).Observe(duration.Seconds())
}

// RecordMemoryOperation 记录 Memory 操作
func (mc *MetricsCollector) RecordMemoryOperation(memoryType, operation string) {
	mc.memoryOperationsTotal.WithLabelValues(memoryType, operation).Inc()
}

// SetMemorySize 设置 Memory 大小
func (mc *MetricsCollector) SetMemorySize(memoryType string, size int) {
	mc.memorySize.WithLabelValues(memoryType).Set(float64(size))
}

// GetRegistry 获取 Prometheus Registry
func (mc *MetricsCollector) GetRegistry() *prometheus.Registry {
	return mc.registry
}

// Handler 返回 HTTP 处理器
func (mc *MetricsCollector) Handler() http.Handler {
	return promhttp.HandlerFor(mc.registry, promhttp.HandlerOpts{
		EnableOpenMetrics: true,
	})
}

// StartServer 启动指标服务器
func (mc *MetricsCollector) StartServer() error {
	http.Handle(mc.config.HTTPPath, mc.Handler())
	return http.ListenAndServe(":"+mc.config.HTTPPort, nil)
}
