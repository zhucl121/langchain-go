// Package main 演示向量量化的可观测性功能
package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"net/http"
	"time"
	
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/zhucl121/langchain-go/retrieval/vectorstores/quantization"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/trace"
)

func main() {
	fmt.Println("=== 向量量化可观测性示例 ===\n")
	
	// 1. 设置 OpenTelemetry Tracing
	tp, err := setupTracing()
	if err != nil {
		log.Fatalf("Failed to setup tracing: %v", err)
	}
	defer func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			log.Printf("Error shutting down tracer provider: %v", err)
		}
	}()
	
	tracer := tp.Tracer("quantization-demo")
	
	// 2. 设置 Prometheus Metrics
	registry := prometheus.NewRegistry()
	metricsCollector := quantization.NewPrometheusMetricsCollector("demo", "quantization", registry)
	
	// 3. 启动 Prometheus HTTP 服务器
	go func() {
		http.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
		fmt.Println("Prometheus metrics server started at http://localhost:9090/metrics")
		if err := http.ListenAndServe(":9090", nil); err != nil {
			log.Printf("Metrics server error: %v", err)
		}
	}()
	
	// 等待服务器启动
	time.Sleep(100 * time.Millisecond)
	
	// 4. 创建测试数据
	dimension := 128
	vectorCount := 100
	vectors := generateTestVectors(vectorCount, dimension)
	
	// 5. 测试不同的量化方法
	fmt.Println("━━━ 测试 Scalar 8-bit 量化 ━━━")
	testQuantizationWithObservability(
		quantization.QuantizationScalar,
		8,
		vectors,
		dimension,
		tracer,
		metricsCollector,
	)
	
	fmt.Println("\n━━━ 测试 Binary 量化 ━━━")
	testQuantizationWithObservability(
		quantization.QuantizationBinary,
		1,
		vectors,
		dimension,
		tracer,
		metricsCollector,
	)
	
	// 6. 显示指标摘要
	fmt.Println("\n━━━ 指标摘要 ━━━")
	fmt.Println("访问 http://localhost:9090/metrics 查看详细指标")
	
	// 保持程序运行以便查看指标
	fmt.Println("\n按 Ctrl+C 退出...")
	select {}
}

// setupTracing 设置 OpenTelemetry Tracing
func setupTracing() (*trace.TracerProvider, error) {
	// 使用 stdout exporter（生产环境应使用 OTLP）
	exporter, err := stdouttrace.New(
		stdouttrace.WithPrettyPrint(),
	)
	if err != nil {
		return nil, err
	}
	
	tp := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
		trace.WithSampler(trace.AlwaysSample()),
	)
	
	otel.SetTracerProvider(tp)
	
	return tp, nil
}

// testQuantizationWithObservability 测试带可观测性的量化
func testQuantizationWithObservability(
	qType quantization.QuantizationType,
	bits int,
	vectors [][]float32,
	dimension int,
	tracer trace.Tracer,
	metrics quantization.MetricsCollector,
) {
	// 创建配置
	config := quantization.Config{
		Type: qType,
		Bits: bits,
	}
	
	// 创建基础量化器
	baseQuantizer, err := quantization.NewQuantizer(config, dimension)
	if err != nil {
		log.Printf("Error creating quantizer: %v", err)
		return
	}
	
	// 包装为可观测量化器
	observableQ := quantization.NewObservableQuantizer(baseQuantizer, tracer, metrics)
	
	ctx := context.Background()
	
	// 1. 训练
	fmt.Print("  训练中...")
	start := time.Now()
	if err := observableQ.Train(ctx, vectors[:50]); err != nil {
		log.Printf("Training error: %v", err)
		return
	}
	trainTime := time.Since(start)
	fmt.Printf(" 完成 (%v)\n", trainTime)
	
	// 2. 编码
	fmt.Print("  编码中...")
	start = time.Now()
	quantized, err := observableQ.Encode(vectors)
	if err != nil {
		log.Printf("Encoding error: %v", err)
		return
	}
	encodeTime := time.Since(start)
	fmt.Printf(" 完成 (%v)\n", encodeTime)
	
	// 3. 解码
	fmt.Print("  解码中...")
	start = time.Now()
	decoded, err := observableQ.Decode(quantized)
	if err != nil {
		log.Printf("Decoding error: %v", err)
		return
	}
	decodeTime := time.Since(start)
	fmt.Printf(" 完成 (%v)\n", decodeTime)
	
	// 4. 距离计算
	fmt.Print("  距离计算中...")
	query, _ := quantized.Get(0)
	candidates := make([]quantization.QuantizedVector, 10)
	for i := 0; i < 10; i++ {
		candidates[i], _ = quantized.Get(i)
	}
	
	start = time.Now()
	distances, err := observableQ.ComputeDistance(query, candidates)
	if err != nil {
		log.Printf("Distance computation error: %v", err)
		return
	}
	distTime := time.Since(start)
	fmt.Printf(" 完成 (%v)\n", distTime)
	
	// 5. 计算压缩比和误差
	ratio := observableQ.CompressionRatio()
	mse := computeMSE(vectors, decoded)
	
	// 6. 输出统计
	fmt.Printf("  压缩比: %.2fx\n", ratio)
	fmt.Printf("  MSE: %.6f\n", mse)
	fmt.Printf("  距离示例: %v\n", distances[:min(5, len(distances))])
	
	// 给 Prometheus 一点时间收集指标
	time.Sleep(100 * time.Millisecond)
}

// generateTestVectors 生成测试向量
func generateTestVectors(count, dimension int) [][]float32 {
	vectors := make([][]float32, count)
	for i := 0; i < count; i++ {
		vectors[i] = make([]float32, dimension)
		for j := 0; j < dimension; j++ {
			vectors[i][j] = float32(math.Sin(float64(i*dimension+j) * 0.01))
		}
	}
	return vectors
}

// computeMSE 计算均方误差
func computeMSE(original, reconstructed [][]float32) float32 {
	if len(original) != len(reconstructed) {
		return float32(math.MaxFloat32)
	}
	
	totalError := float32(0)
	count := 0
	
	for i := range original {
		if len(original[i]) != len(reconstructed[i]) {
			continue
		}
		for j := range original[i] {
			diff := original[i][j] - reconstructed[i][j]
			totalError += diff * diff
			count++
		}
	}
	
	if count == 0 {
		return 0
	}
	
	return totalError / float32(count)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
