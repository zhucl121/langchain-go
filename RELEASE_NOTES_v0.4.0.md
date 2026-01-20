# LangChain-Go v0.4.0 发布说明

**发布日期**: 2026-01-20  
**版本**: v0.4.0  
**主题**: 完整的监控与可观测性

---

## 🎉 概述

LangChain-Go v0.4.0 正式发布！本版本带来了**生产级的监控与可观测性**能力，为 AI 应用提供完整的监控、追踪、日志和性能分析支持。

---

## ✨ 新功能

### 1. 结构化日志系统

基于 Go 标准库 `log/slog` 实现的高性能日志系统：

```go
// 初始化全局日志
observability.InitGlobalLogger(observability.DefaultLoggerConfig())

// 使用日志
observability.Info("User logged in",
    observability.String("user_id", "12345"),
    observability.Int("status", 200),
)

// 创建子 Logger
logger := observability.GetGlobalLogger().With(
    observability.String("service", "api"),
    observability.String("version", "1.0.0"),
)
```

**特性**:
- ✅ 多级别日志 (Debug/Info/Warn/Error)
- ✅ 多种输出格式 (JSON/Text)
- ✅ 多种输出目标 (stdout/stderr/file)
- ✅ 自动提取 TraceID 和 SpanID
- ✅ 子 Logger 支持
- ✅ 类型安全的字段系统

### 2. 统一可观测性上下文

统一管理 Tracer、Logger、Metrics 的上下文系统：

```go
// 创建统一上下文
obs := observability.NewObservabilityContext(tracer, logger, metrics)
ctx := observability.WithObservability(context.Background(), obs)

// 自动追踪 LLM 调用
tracker := observability.StartLLMOperation(ctx, "openai", "gpt-4")
defer tracker.End(err)

result, err := llm.Invoke(ctx, messages)
tracker.SetTokens(100, 50)
```

**特性**:
- ✅ 自动 Span 创建和管理
- ✅ 6 种专用操作追踪器 (LLM/RAG/Tool/Agent/Chain/通用)
- ✅ 自动日志记录
- ✅ 自动指标收集
- ✅ Context 自动传播

### 3. 性能分析工具

完整的性能分析和监控工具集：

```go
// Profiler - 性能分析
config := profiling.DefaultProfilerConfig()
profiler, _ := profiling.NewProfiler(config)
profiler.Start()
// ... 执行代码 ...
profiler.Stop()

// Analyzer - 性能监控
analyzer := profiling.NewAnalyzer()
analyzer.SetBaseline()
// ... 执行操作 ...
report := analyzer.Analyze()
fmt.Println(report)

// Benchmark - 基准测试
report := profiling.RunBenchmark("operation", func() {
    // 执行需要测试的代码
})
```

**特性**:
- ✅ CPU/内存/Goroutine/阻塞/互斥锁分析
- ✅ 执行追踪
- ✅ 实时性能指标收集
- ✅ 基准对比
- ✅ 自动问题检测（内存泄漏、Goroutine 泄漏、高 GC）
- ✅ 详细性能报告

---

## 📊 技术指标

### 性能

- **性能开销**: < 5% CPU
- **内存开销**: < 10MB
- **日志性能**: < 1% CPU，< 1MB 内存
- **追踪性能**: < 3% CPU（100% 采样）

### 测试

- **测试数量**: 59 个测试
- **通过率**: 100%
- **测试覆盖率**: 87%+

### 代码质量

- **核心代码**: 2,300 行
- **测试代码**: 1,250 行
- **文档**: 2,100 行
- **总计**: 5,650 行

---

## 🚀 快速开始

### 安装

```bash
go get -u github.com/zhucl121/langchain-go
```

### 最小示例

```go
package main

import (
    "context"
    "github.com/zhucl121/langchain-go/pkg/observability"
)

func main() {
    // 初始化日志
    observability.InitGlobalLogger(observability.DefaultLoggerConfig())
    
    // 创建 Tracer
    tracerConfig := observability.TracerConfig{
        ServiceName: "my-app",
        Endpoint:    "localhost:4317",
    }
    tracerProvider, _ := observability.NewTracerProvider(tracerConfig)
    defer tracerProvider.Shutdown(context.Background())
    
    // 创建 Metrics
    metrics := observability.NewMetricsCollector(observability.MetricsConfig{
        Namespace: "myapp",
    })
    
    // 创建可观测性上下文
    obs := observability.NewObservabilityContext(
        tracerProvider.GetTracer(),
        observability.GetGlobalLogger(),
        metrics,
    )
    
    ctx := observability.WithObservability(context.Background(), obs)
    
    // 使用
    observability.Info("Application started")
    
    err := observability.TrackOperation(ctx, "my-operation", nil, 
        func(ctx context.Context) error {
            // 执行业务逻辑
            return nil
        })
    
    if err != nil {
        observability.Error("Operation failed", observability.Err(err))
    }
}
```

---

## 📚 文档

- **用户指南**: [V0.4.0_USER_GUIDE.md](docs/V0.4.0_USER_GUIDE.md)
- **实施计划**: [V0.4.0_IMPLEMENTATION_PLAN.md](docs/V0.4.0_IMPLEMENTATION_PLAN.md)
- **完成报告**: [V0.4.0_COMPLETION_REPORT.md](docs/V0.4.0_COMPLETION_REPORT.md)

### 示例程序

- **observability_demo**: 完整的可观测性示例
  ```bash
  cd examples/observability_demo
  go run main.go
  ```

---

## 🔄 迁移指南

从 v0.3.0 升级到 v0.4.0：

### 1. 添加日志

```go
// 之前
fmt.Println("Processing request")

// 现在
observability.Info("Processing request",
    observability.String("request_id", reqID),
)
```

### 2. 添加追踪

```go
// 之前
result, err := doSomething()

// 现在
tracker := observability.StartOperation(ctx, "do-something", nil)
defer tracker.End(err)

result, err := doSomething()
```

### 3. 添加指标

```go
// 之前
// 无指标

// 现在
start := time.Now()
result, err := llm.Invoke(ctx, messages)
duration := time.Since(start)

metrics.RecordLLMCall("openai", "gpt-4", duration, err)
```

---

## 🎯 应用场景

### 生产环境监控

```go
// 自动监控所有 LLM 调用
tracker := observability.StartLLMOperation(ctx, "openai", "gpt-4")
defer tracker.End(err)

result, err := chatModel.Invoke(ctx, messages)
tracker.SetTokens(inputTokens, outputTokens)

// 自动记录:
// - Span: llm.call with attributes
// - Metrics: llm_calls_total, llm_tokens_total, llm_call_duration_seconds
// - Logs: LLM call started/completed/failed
```

### 性能调优

```go
// 性能分析
config := profiling.DefaultProfilerConfig()
profiler, _ := profiling.NewProfiler(config)
profiler.Start()

// 执行需要优化的代码
processLargeDataset()

profiler.Stop()

// 分析结果
// 使用 go tool pprof 查看 CPU/内存分析
```

### 问题排查

```go
// 1. 从日志获取 trace_id
// 2. 在 Jaeger/Zipkin 中查看完整链路
// 3. 查看每个 Span 的详细信息
// 4. 定位慢查询或错误

// Prometheus + Grafana 实时监控
// - 访问 http://localhost:9090/metrics
// - 配置 Grafana Dashboard
// - 设置告警规则
```

---

## 🔍 与 v0.3.0 对比

| 功能 | v0.3.0 | v0.4.0 | 提升 |
|------|--------|--------|------|
| 结构化日志 | ❌ | ✅ 完整 | +100% |
| 分布式追踪 | ⚠️ 基础 | ✅ 完整 | +100% |
| Prometheus | ⚠️ 基础 | ✅ 完整 | +50% |
| 性能分析 | ❌ | ✅ 完整 | +100% |
| 统一上下文 | ❌ | ✅ 完整 | +100% |

---

## 🐛 已知问题

无重大已知问题。

---

## 🙏 致谢

感谢所有贡献者和用户的支持！

特别感谢：
- Go 标准库团队（log/slog）
- OpenTelemetry 团队
- Prometheus 团队

---

## 📅 路线图

### v0.4.1 (下一版本)

**主题**: GraphRAG - 知识图谱增强检索

- 🔜 图数据库抽象
- 🔜 Neo4j 集成
- 🔜 知识图谱构建
- 🔜 混合图向量检索

**预计发布**: 2-3 周

### 未来版本

- **v0.4.2**: 学习型检索（自适应优化）
- **v0.5.0**: 分布式部署（集群支持）
- **v0.6.0**: 加密检索（隐私计算）

---

## 📞 联系我们

- **GitHub**: https://github.com/zhucl121/langchain-go
- **Issues**: https://github.com/zhucl121/langchain-go/issues
- **Discussions**: https://github.com/zhucl121/langchain-go/discussions

---

## 📄 许可证

MIT License

---

**发布团队**: LangChain-Go Team  
**发布日期**: 2026-01-20  
**版本**: v0.4.0
