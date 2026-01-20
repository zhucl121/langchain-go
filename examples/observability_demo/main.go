package main

import (
	"context"
	"fmt"
	"time"
	
	"github.com/zhucl121/langchain-go/pkg/observability"
	"github.com/zhucl121/langchain-go/pkg/observability/profiling"
	
	"go.opentelemetry.io/otel/trace"
)

func main() {
	fmt.Println("LangChain-Go v0.4.0 Observability Demo")
	fmt.Println("======================================\n")
	
	// 初始化可观测性
	obs, cleanup := initObservability()
	defer cleanup()
	
	ctx := observability.WithObservability(context.Background(), obs)
	
	// 示例 1: 基础日志
	fmt.Println("示例 1: 基础日志")
	fmt.Println("--------------")
	demoLogging(ctx)
	fmt.Println()
	
	// 示例 2: 分布式追踪
	fmt.Println("示例 2: 分布式追踪")
	fmt.Println("-----------------")
	demoTracing(ctx)
	fmt.Println()
	
	// 示例 3: 指标收集
	fmt.Println("示例 3: 指标收集")
	fmt.Println("---------------")
	demoMetrics(obs.Metrics)
	fmt.Println()
	
	// 示例 4: 性能分析
	fmt.Println("示例 4: 性能分析")
	fmt.Println("---------------")
	demoProfiling()
	fmt.Println()
	
	// 示例 5: 操作追踪
	fmt.Println("示例 5: 操作追踪")
	fmt.Println("---------------")
	demoOperationTracking(ctx)
	fmt.Println()
	
	fmt.Println("Demo 完成！")
	fmt.Println("\n提示:")
	fmt.Println("  - 日志输出到 stdout")
	fmt.Println("  - Metrics 可通过 http://localhost:9090/metrics 访问")
	fmt.Println("  - Profile 文件保存在 ./profiles/ 目录")
}

// initObservability 初始化可观测性组件
func initObservability() (*observability.ObservabilityContext, func()) {
	// 1. 初始化日志
	logConfig := observability.LoggerConfig{
		Level:  observability.LogLevelInfo,
		Format: "json",
		Output: "stdout",
		Attributes: map[string]any{
			"service": "observability-demo",
			"version": "1.0.0",
		},
	}
	
	if err := observability.InitGlobalLogger(logConfig); err != nil {
		panic(fmt.Sprintf("Failed to initialize logger: %v", err))
	}
	
	// 2. 初始化追踪（使用内存导出器，不需要外部服务）
	tracerConfig := observability.TracerConfig{
		ServiceName:    "observability-demo",
		ServiceVersion: "1.0.0",
		Environment:    "development",
		ExporterType:   "otlp-grpc",
		Endpoint:       "localhost:4317",
		SampleRate:     1.0,
	}
	
	tracerProvider, err := observability.NewTracerProvider(tracerConfig)
	if err != nil {
		// 如果无法连接 OTLP，使用默认的 tracer
		observability.Warn("Failed to initialize tracer, using default",
			observability.Err(err))
	}
	
	// 3. 初始化指标
	metricsConfig := observability.MetricsConfig{
		Namespace:            "langchain",
		Subsystem:            "demo",
		EnableDefaultMetrics: true,
		HTTPPath:             "/metrics",
		HTTPPort:             "9090",
	}
	
	metrics := observability.NewMetricsCollector(metricsConfig)
	
	// 启动 metrics 服务器
	go func() {
		observability.Info("Starting metrics server",
			observability.String("address", "http://localhost:9090/metrics"))
		
		if err := metrics.StartServer(); err != nil {
			observability.Error("Failed to start metrics server",
				observability.Err(err))
		}
	}()
	
	// 创建可观测性上下文
	var tracer trace.Tracer
	if tracerProvider != nil {
		tracer = tracerProvider.GetTracer()
	}
	
	obs := observability.NewObservabilityContext(
		tracer,
		observability.GetGlobalLogger(),
		metrics,
	)
	
	// 清理函数
	cleanup := func() {
		if tracerProvider != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			tracerProvider.Shutdown(ctx)
		}
	}
	
	return obs, cleanup
}

// demoLogging 演示日志功能
func demoLogging(ctx context.Context) {
	logger := observability.LogFromContext(ctx)
	
	// 不同级别的日志
	logger.Debug("This is a debug message",
		observability.String("component", "demo"),
	)
	
	logger.Info("Processing user request",
		observability.String("user_id", "12345"),
		observability.String("action", "login"),
		observability.Int("attempt", 1),
	)
	
	logger.Warn("API rate limit approaching",
		observability.Int("current", 95),
		observability.Int("limit", 100),
		observability.Float64("percentage", 95.0),
	)
	
	// 模拟错误
	err := fmt.Errorf("connection timeout")
	logger.Error("Failed to connect to database",
		observability.Err(err),
		observability.String("host", "db.example.com"),
		observability.Duration("timeout", 30*time.Second),
	)
	
	// 使用子 logger
	requestLogger := logger.With(
		observability.String("request_id", "req-123"),
		observability.String("method", "GET"),
		observability.String("path", "/api/users"),
	)
	
	requestLogger.Info("Request started")
	time.Sleep(10 * time.Millisecond)
	requestLogger.Info("Request completed",
		observability.Int("status_code", 200),
		observability.Duration("latency", 10*time.Millisecond),
	)
	
	fmt.Println("✓ 日志已输出到 stdout")
}

// demoTracing 演示追踪功能
func demoTracing(ctx context.Context) {
	// 开始操作追踪
	err := observability.TrackOperation(ctx, "demo-workflow",
		map[string]string{
			"workflow": "user-registration",
		},
		func(ctx context.Context) error {
			// 步骤 1: 验证用户
			time.Sleep(10 * time.Millisecond)
			observability.Info("Step 1: Validate user")
			
			// 步骤 2: 创建账户
			time.Sleep(20 * time.Millisecond)
			observability.Info("Step 2: Create account")
			
			// 步骤 3: 发送欢迎邮件
			time.Sleep(15 * time.Millisecond)
			observability.Info("Step 3: Send welcome email")
			
			return nil
		})
	
	if err != nil {
		observability.Error("Workflow failed", observability.Err(err))
	} else {
		fmt.Println("✓ 工作流追踪完成")
	}
}

// demoMetrics 演示指标收集
func demoMetrics(metrics *observability.MetricsCollector) {
	// 模拟 LLM 调用
	for i := 0; i < 5; i++ {
		duration := time.Duration(100+i*50) * time.Millisecond
		metrics.RecordLLMCall("openai", "gpt-4", duration, nil)
		metrics.RecordLLMTokens("openai", "gpt-4", 100, 50)
	}
	
	// 模拟 RAG 查询
	for i := 0; i < 3; i++ {
		duration := time.Duration(50+i*20) * time.Millisecond
		metrics.RecordRAGQuery("milvus", duration, 10, nil)
	}
	
	// 模拟 Tool 调用
	for i := 0; i < 10; i++ {
		duration := time.Duration(10+i*5) * time.Millisecond
		metrics.RecordToolCall("calculator", duration, nil)
	}
	
	fmt.Println("✓ 指标已记录")
	fmt.Println("  访问 http://localhost:9090/metrics 查看")
}

// demoProfiling 演示性能分析
func demoProfiling() {
	// 示例 1: 使用 Analyzer
	fmt.Println("\n性能分析器:")
	
	analyzer := profiling.NewAnalyzer()
	analyzer.SetBaseline()
	
	// 模拟一些工作
	data := make([][]byte, 1000)
	for i := 0; i < 1000; i++ {
		data[i] = make([]byte, 1024)
	}
	time.Sleep(50 * time.Millisecond)
	
	report := analyzer.Analyze()
	fmt.Println(report)
	
	// 示例 2: 基准测试
	fmt.Println("\n基准测试:")
	
	benchReport := profiling.RunBenchmark("string-concatenation", func() {
		result := ""
		for i := 0; i < 10000; i++ {
			result += fmt.Sprintf("item-%d,", i)
		}
		_ = result
	})
	
	fmt.Printf("  操作耗时: %s\n", benchReport.Delta.Duration)
	fmt.Printf("  内存分配: +%d bytes\n", benchReport.Delta.MemoryAllocDelta)
	fmt.Printf("  GC 次数: +%d\n", benchReport.Delta.NumGCDelta)
}

// demoOperationTracking 演示操作追踪
func demoOperationTracking(ctx context.Context) {
	// LLM 操作
	fmt.Println("追踪 LLM 操作...")
	llmTracker := observability.StartLLMOperation(ctx, "openai", "gpt-4")
	time.Sleep(100 * time.Millisecond)
	llmTracker.SetTokens(120, 80)
	llmTracker.End(nil)
	fmt.Println("✓ LLM 操作追踪完成")
	
	// RAG 操作
	fmt.Println("追踪 RAG 操作...")
	ragTracker := observability.StartRAGOperation(ctx, "milvus", "What is LangChain?")
	time.Sleep(50 * time.Millisecond)
	ragTracker.SetDocumentCount(10)
	ragTracker.End(nil)
	fmt.Println("✓ RAG 操作追踪完成")
	
	// Tool 操作
	fmt.Println("追踪 Tool 操作...")
	toolTracker := observability.StartToolOperation(ctx, "calculator", "2+2")
	time.Sleep(10 * time.Millisecond)
	toolTracker.SetOutput("4")
	toolTracker.End(nil)
	fmt.Println("✓ Tool 操作追踪完成")
	
	// Agent 操作
	fmt.Println("追踪 Agent 操作...")
	agentTracker := observability.StartAgentOperation(ctx, "react", 1)
	agentTracker.SetIteration(1)
	agentTracker.SetAction("think")
	time.Sleep(30 * time.Millisecond)
	agentTracker.End(nil)
	fmt.Println("✓ Agent 操作追踪完成")
}
