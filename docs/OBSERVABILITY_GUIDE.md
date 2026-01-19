# 可观测性指南

**版本**: v0.2.0  
**创建日期**: 2026-01-20  
**作者**: LangChain-Go Team

---

## 📋 目录

- [概述](#概述)
- [快速开始](#快速开始)
- [OpenTelemetry Tracing](#opentelemetry-tracing)
- [Prometheus Metrics](#prometheus-metrics)
- [Grafana Dashboard](#grafana-dashboard)
- [最佳实践](#最佳实践)
- [故障排查](#故障排查)

---

## 概述

LangChain-Go 提供完整的可观测性支持，帮助您监控和调试应用程序。

### 核心功能

- **OpenTelemetry Tracing**: 分布式追踪
- **Prometheus Metrics**: 指标收集
- **Grafana Dashboard**: 可视化监控

### 支持的组件

- ✅ 向量量化器 (Quantization)
- ✅ 向量存储 (Vector Stores) - 即将支持
- ✅ LLM 调用 - 即将支持
- ✅ Agent 执行 - 即将支持

---

## 快速开始

### 1. 启动监控基础设施

使用 Docker Compose 一键启动 Prometheus 和 Grafana：

```bash
cd config
docker-compose -f docker-compose.observability.yml up -d
```

服务地址：
- **Prometheus**: http://localhost:9091
- **Grafana**: http://localhost:3000 (admin/admin)
- **Jaeger UI**: http://localhost:16686

### 2. 运行示例程序

```bash
# 运行可观测性示例
go run examples/observability_demo/observability_demo.go
```

### 3. 查看监控数据

1. **访问 Grafana**: http://localhost:3000
2. **默认登录**: admin/admin
3. **打开 Dashboard**: "LangChain-Go 向量量化监控"

---

## OpenTelemetry Tracing

### 基本用法

```go
package main

import (
    "context"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
    "go.opentelemetry.io/otel/sdk/trace"
    "github.com/zhucl121/langchain-go/retrieval/vectorstores/quantization"
)

func main() {
    // 1. 设置 Tracer Provider
    tp := setupTracing()
    defer tp.Shutdown(context.Background())
    
    tracer := tp.Tracer("my-app")
    
    // 2. 创建可观测的量化器
    baseQ, _ := quantization.NewQuantizer(config, dimension)
    observableQ := quantization.NewObservableQuantizer(baseQ, tracer, nil)
    
    // 3. 正常使用，自动追踪
    ctx := context.Background()
    observableQ.Train(ctx, vectors)
    quantized, _ := observableQ.Encode(vectors)
}

func setupTracing() *trace.TracerProvider {
    exporter, _ := otlptracegrpc.New(
        context.Background(),
        otlptracegrpc.WithEndpoint("localhost:4317"),
        otlptracegrpc.WithInsecure(),
    )
    
    tp := trace.NewTracerProvider(
        trace.WithBatcher(exporter),
        trace.WithSampler(trace.AlwaysSample()),
    )
    
    otel.SetTracerProvider(tp)
    return tp
}
```

### Span 属性

量化操作会自动记录以下属性：

**训练 (Train)**:
- `quantization.type`: 量化类型
- `quantization.dimension`: 向量维度
- `quantization.vector_count`: 训练样本数
- `quantization.training_duration_ms`: 训练时长
- `quantization.compression_ratio`: 压缩比

**编码 (Encode)**:
- `quantization.type`: 量化类型
- `quantization.vector_count`: 向量数量
- `quantization.original_size_bytes`: 原始大小
- `quantization.compressed_size_bytes`: 压缩后大小
- `quantization.compression_ratio`: 压缩比
- `quantization.encoding_duration_ms`: 编码时长

**解码 (Decode)**:
- `quantization.vector_count`: 向量数量
- `quantization.decoding_duration_ms`: 解码时长

**距离计算 (Distance)**:
- `quantization.type`: 量化类型
- `quantization.vector_count`: 向量数量
- `quantization.distance_duration_us`: 计算时长

---

## Prometheus Metrics

### 指标列表

#### 训练指标

```
# 训练次数
langchain_quantization_training_total{type="scalar|binary|product", status="success|error"}

# 训练时长
langchain_quantization_training_duration_seconds{type="..."}

# 训练错误
langchain_quantization_training_errors_total{type="..."}
```

#### 编码指标

```
# 编码次数
langchain_quantization_encoding_total{type="...", status="success|error"}

# 编码时长
langchain_quantization_encoding_duration_seconds{type="..."}

# 编码错误
langchain_quantization_encoding_errors_total{type="..."}
```

#### 解码指标

```
# 解码次数
langchain_quantization_decoding_total{type="...", status="success|error"}

# 解码时长
langchain_quantization_decoding_duration_seconds{type="..."}

# 解码错误
langchain_quantization_decoding_errors_total{type="..."}
```

#### 距离计算指标

```
# 距离计算次数
langchain_quantization_distance_computation_total{type="...", status="success|error"}

# 距离计算时长
langchain_quantization_distance_computation_duration_seconds{type="..."}

# 距离计算错误
langchain_quantization_distance_computation_errors_total{type="..."}
```

#### 压缩比指标

```
# 当前压缩比
langchain_quantization_compression_ratio{type="..."}
```

### 使用示例

```go
import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promhttp"
    "github.com/zhucl121/langchain-go/retrieval/vectorstores/quantization"
)

func main() {
    // 1. 创建 Prometheus Registry
    registry := prometheus.NewRegistry()
    
    // 2. 创建指标收集器
    metricsCollector := quantization.NewPrometheusMetricsCollector(
        "myapp",         // namespace
        "quantization",  // subsystem
        registry,
    )
    
    // 3. 创建可观测量化器
    observableQ := quantization.NewObservableQuantizer(
        baseQuantizer,
        tracer,
        metricsCollector,
    )
    
    // 4. 启动 HTTP 服务器
    http.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
    go http.ListenAndServe(":9090", nil)
    
    // 5. 正常使用
    observableQ.Train(ctx, vectors)
    observableQ.Encode(vectors)
}
```

---

## Grafana Dashboard

### 导入 Dashboard

1. **登录 Grafana**: http://localhost:3000
2. **导航到**: Dashboards → Import
3. **上传文件**: `config/grafana/quantization-dashboard.json`
4. **选择数据源**: Prometheus
5. **导入**

### Dashboard 面板

#### 1. 量化操作 QPS
- 显示每秒执行的操作数
- 按操作类型（编码/解码/距离）分组

#### 2. 操作延迟 (P99)
- 显示操作的 99 分位延迟
- 帮助识别性能瓶颈

#### 3. 错误率
- 显示各种操作的错误率
- 按类型分组

#### 4. 压缩比
- 显示当前的压缩比
- 按量化类型展示

#### 5. 训练操作
- 显示训练操作的频率

#### 6. 训练时长 (P95)
- 显示训练操作的 95 分位延迟

#### 7. 操作分布
- 饼图展示操作类型分布

#### 8. 成功率
- 显示操作的成功率
- 颜色编码：红(<95%), 黄(95-99%), 绿(>99%)

#### 9. 量化类型使用量
- 柱状图展示不同量化类型的使用频率

### 自定义面板

添加新面板示例：

```json
{
  "title": "平均编码速度",
  "targets": [{
    "expr": "rate(langchain_quantization_encoding_total[5m]) / rate(langchain_quantization_encoding_duration_seconds_sum[5m])",
    "legendFormat": "{{type}}"
  }],
  "yaxes": [
    {"format": "ops", "label": "向量/秒"}
  ]
}
```

---

## 最佳实践

### 1. 采样策略

生产环境建议使用基于概率的采样：

```go
import "go.opentelemetry.io/otel/sdk/trace"

tp := trace.NewTracerProvider(
    trace.WithSampler(trace.TraceIDRatioBased(0.1)), // 10% 采样
)
```

### 2. 资源限制

限制 Prometheus 指标的基数：

```go
// 使用固定的标签集
labels := []string{"type", "status"}

// 避免使用高基数标签（如 vector_id, user_id）
```

### 3. 告警规则

在 Prometheus 中配置告警：

```yaml
# alert_rules.yml
groups:
  - name: quantization_alerts
    rules:
      # 错误率告警
      - alert: HighErrorRate
        expr: rate(langchain_quantization_encoding_errors_total[5m]) > 0.05
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "量化编码错误率过高"
          description: "错误率: {{ $value }}"
      
      # 延迟告警
      - alert: HighLatency
        expr: histogram_quantile(0.99, rate(langchain_quantization_encoding_duration_seconds_bucket[5m])) > 1
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "量化编码延迟过高"
          description: "P99 延迟: {{ $value }}s"
```

### 4. 日志关联

结合日志和追踪：

```go
import (
    "go.opentelemetry.io/otel/trace"
    "log/slog"
)

func logWithTrace(ctx context.Context, msg string) {
    spanCtx := trace.SpanContextFromContext(ctx)
    if spanCtx.IsValid() {
        slog.InfoContext(ctx, msg,
            "trace_id", spanCtx.TraceID().String(),
            "span_id", spanCtx.SpanID().String(),
        )
    }
}
```

### 5. 性能优化

减少可观测性开销：

```go
// 批量操作时只追踪一次
func batchEncode(vectors [][]float32) {
    ctx, span := tracer.Start(context.Background(), "batch_encode")
    defer span.End()
    
    for _, batch := range chunk(vectors, 100) {
        // 不为每个批次创建 span
        quantizer.Encode(batch)
    }
}
```

---

## 故障排查

### 问题 1: Prometheus 无法抓取指标

**症状**: Prometheus targets 页面显示 "Context deadline exceeded"

**解决方案**:
1. 检查应用是否在运行
2. 确认端口正确: `curl http://localhost:9090/metrics`
3. 检查防火墙规则

### 问题 2: Grafana 无数据

**症状**: Dashboard 显示 "No data"

**解决方案**:
1. 检查 Prometheus 数据源配置
2. 确认时间范围正确
3. 验证 PromQL 查询: `rate(langchain_quantization_encoding_total[5m])`

### 问题 3: Trace 数据未显示

**症状**: Jaeger UI 无追踪数据

**解决方案**:
1. 检查 OTLP Exporter 配置
2. 确认 Jaeger 端口开放
3. 验证采样率不是 0

### 问题 4: 高内存占用

**症状**: Prometheus 内存占用过高

**解决方案**:
1. 减少保留时间: `--storage.tsdb.retention.time=15d`
2. 降低采集频率: `scrape_interval: 30s`
3. 限制指标基数

---

## 高级配置

### 远程写入

将指标发送到远程存储（如 Thanos、Cortex）：

```yaml
# prometheus.yml
remote_write:
  - url: "http://thanos-receive:19291/api/v1/receive"
    queue_config:
      capacity: 10000
      max_shards: 50
```

### 多实例聚合

使用 Prometheus Federation：

```yaml
# 中心 Prometheus
scrape_configs:
  - job_name: 'federate'
    honor_labels: true
    metrics_path: '/federate'
    params:
      'match[]':
        - '{job="langchain-go"}'
    static_configs:
      - targets:
          - 'prometheus-1:9090'
          - 'prometheus-2:9090'
```

---

## 参考资源

- [OpenTelemetry Go 文档](https://opentelemetry.io/docs/instrumentation/go/)
- [Prometheus 查询语言](https://prometheus.io/docs/prometheus/latest/querying/basics/)
- [Grafana Dashboard 最佳实践](https://grafana.com/docs/grafana/latest/dashboards/build-dashboards/best-practices/)

---

**文档版本**: v1.0  
**最后更新**: 2026-01-20  
**维护者**: LangChain-Go Team
