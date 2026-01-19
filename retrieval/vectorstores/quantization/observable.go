package quantization

import (
	"context"
	"time"
	
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// ObservableQuantizer 可观测的量化器包装器
//
// 为量化器添加 tracing 和 metrics 支持
type ObservableQuantizer struct {
	quantizer Quantizer
	tracer    trace.Tracer
	
	// 指标收集器（可选）
	metricsCollector MetricsCollector
}

// MetricsCollector 指标收集器接口
type MetricsCollector interface {
	// RecordTraining 记录训练指标
	RecordTraining(qType QuantizationType, duration time.Duration, vectorCount int, err error)
	
	// RecordEncoding 记录编码指标
	RecordEncoding(qType QuantizationType, duration time.Duration, vectorCount int, err error)
	
	// RecordDecoding 记录解码指标
	RecordDecoding(qType QuantizationType, duration time.Duration, vectorCount int, err error)
	
	// RecordDistance 记录距离计算指标
	RecordDistance(qType QuantizationType, duration time.Duration, vectorCount int, err error)
	
	// RecordCompressionRatio 记录压缩比
	RecordCompressionRatio(qType QuantizationType, ratio float64)
}

// NewObservableQuantizer 创建可观测的量化器
func NewObservableQuantizer(quantizer Quantizer, tracer trace.Tracer, metrics MetricsCollector) *ObservableQuantizer {
	return &ObservableQuantizer{
		quantizer:        quantizer,
		tracer:           tracer,
		metricsCollector: metrics,
	}
}

func (q *ObservableQuantizer) Type() QuantizationType {
	return q.quantizer.Type()
}

func (q *ObservableQuantizer) Dimension() int {
	return q.quantizer.Dimension()
}

func (q *ObservableQuantizer) IsTrained() bool {
	return q.quantizer.IsTrained()
}

func (q *ObservableQuantizer) CompressionRatio() float64 {
	ratio := q.quantizer.CompressionRatio()
	
	// 记录压缩比指标
	if q.metricsCollector != nil {
		q.metricsCollector.RecordCompressionRatio(q.quantizer.Type(), ratio)
	}
	
	return ratio
}

// Train 训练量化器（带追踪）
func (q *ObservableQuantizer) Train(ctx context.Context, vectors [][]float32) error {
	if q.tracer == nil {
		return q.quantizer.Train(ctx, vectors)
	}
	
	// 开始 span
	ctx, span := q.tracer.Start(ctx, "quantization.train",
		trace.WithAttributes(
			attribute.String("quantization.type", string(q.quantizer.Type())),
			attribute.Int("quantization.dimension", q.quantizer.Dimension()),
			attribute.Int("quantization.vector_count", len(vectors)),
		),
	)
	defer span.End()
	
	// 执行训练
	start := time.Now()
	err := q.quantizer.Train(ctx, vectors)
	duration := time.Since(start)
	
	// 记录结果
	if err != nil {
		span.RecordError(err)
	} else {
		span.SetAttributes(
			attribute.Float64("quantization.compression_ratio", q.quantizer.CompressionRatio()),
			attribute.Int64("quantization.training_duration_ms", duration.Milliseconds()),
		)
	}
	
	// 记录指标
	if q.metricsCollector != nil {
		q.metricsCollector.RecordTraining(q.quantizer.Type(), duration, len(vectors), err)
	}
	
	return err
}

// Encode 编码向量（带追踪）
func (q *ObservableQuantizer) Encode(vectors [][]float32) (QuantizedVectors, error) {
	if q.tracer == nil {
		return q.quantizer.Encode(vectors)
	}
	
	// 开始 span
	ctx, span := q.tracer.Start(context.Background(), "quantization.encode",
		trace.WithAttributes(
			attribute.String("quantization.type", string(q.quantizer.Type())),
			attribute.Int("quantization.dimension", q.quantizer.Dimension()),
			attribute.Int("quantization.vector_count", len(vectors)),
		),
	)
	defer span.End()
	
	// 执行编码
	start := time.Now()
	quantized, err := q.quantizer.Encode(vectors)
	duration := time.Since(start)
	
	// 记录结果
	if err != nil {
		span.RecordError(err)
	} else {
		originalSize := len(vectors) * q.quantizer.Dimension() * 4 // float32 = 4 bytes
		compressedSize := quantized.TotalSize()
		ratio := float64(originalSize) / float64(compressedSize)
		
		span.SetAttributes(
			attribute.Int("quantization.original_size_bytes", originalSize),
			attribute.Int("quantization.compressed_size_bytes", compressedSize),
			attribute.Float64("quantization.compression_ratio", ratio),
			attribute.Int64("quantization.encoding_duration_ms", duration.Milliseconds()),
		)
	}
	
	// 记录指标
	if q.metricsCollector != nil {
		q.metricsCollector.RecordEncoding(q.quantizer.Type(), duration, len(vectors), err)
	}
	
	_ = ctx
	return quantized, err
}

// Decode 解码向量（带追踪）
func (q *ObservableQuantizer) Decode(quantized QuantizedVectors) ([][]float32, error) {
	if q.tracer == nil {
		return q.quantizer.Decode(quantized)
	}
	
	// 开始 span
	ctx, span := q.tracer.Start(context.Background(), "quantization.decode",
		trace.WithAttributes(
			attribute.String("quantization.type", string(q.quantizer.Type())),
			attribute.Int("quantization.vector_count", quantized.Count()),
		),
	)
	defer span.End()
	
	// 执行解码
	start := time.Now()
	decoded, err := q.quantizer.Decode(quantized)
	duration := time.Since(start)
	
	// 记录结果
	if err != nil {
		span.RecordError(err)
	} else {
		span.SetAttributes(
			attribute.Int64("quantization.decoding_duration_ms", duration.Milliseconds()),
		)
	}
	
	// 记录指标
	if q.metricsCollector != nil {
		q.metricsCollector.RecordDecoding(q.quantizer.Type(), duration, quantized.Count(), err)
	}
	
	_ = ctx
	return decoded, err
}

// ComputeDistance 计算距离（带追踪）
func (q *ObservableQuantizer) ComputeDistance(query QuantizedVector, vectors []QuantizedVector) ([]float32, error) {
	if q.tracer == nil {
		return q.quantizer.ComputeDistance(query, vectors)
	}
	
	// 开始 span
	ctx, span := q.tracer.Start(context.Background(), "quantization.distance",
		trace.WithAttributes(
			attribute.String("quantization.type", string(q.quantizer.Type())),
			attribute.Int("quantization.vector_count", len(vectors)),
		),
	)
	defer span.End()
	
	// 执行距离计算
	start := time.Now()
	distances, err := q.quantizer.ComputeDistance(query, vectors)
	duration := time.Since(start)
	
	// 记录结果
	if err != nil {
		span.RecordError(err)
	} else {
		span.SetAttributes(
			attribute.Int64("quantization.distance_duration_us", duration.Microseconds()),
		)
	}
	
	// 记录指标
	if q.metricsCollector != nil {
		q.metricsCollector.RecordDistance(q.quantizer.Type(), duration, len(vectors), err)
	}
	
	_ = ctx
	return distances, err
}

// SimpleMetricsCollector 简单的指标收集器实现
type SimpleMetricsCollector struct {
	// 统计数据
	TrainingCount   int
	EncodingCount   int
	DecodingCount   int
	DistanceCount   int
	TotalTrainTime  time.Duration
	TotalEncodeTime time.Duration
	TotalDecodeTime time.Duration
	TotalDistTime   time.Duration
	ErrorCount      int
}

func NewSimpleMetricsCollector() *SimpleMetricsCollector {
	return &SimpleMetricsCollector{}
}

func (m *SimpleMetricsCollector) RecordTraining(qType QuantizationType, duration time.Duration, vectorCount int, err error) {
	m.TrainingCount++
	m.TotalTrainTime += duration
	if err != nil {
		m.ErrorCount++
	}
}

func (m *SimpleMetricsCollector) RecordEncoding(qType QuantizationType, duration time.Duration, vectorCount int, err error) {
	m.EncodingCount++
	m.TotalEncodeTime += duration
	if err != nil {
		m.ErrorCount++
	}
}

func (m *SimpleMetricsCollector) RecordDecoding(qType QuantizationType, duration time.Duration, vectorCount int, err error) {
	m.DecodingCount++
	m.TotalDecodeTime += duration
	if err != nil {
		m.ErrorCount++
	}
}

func (m *SimpleMetricsCollector) RecordDistance(qType QuantizationType, duration time.Duration, vectorCount int, err error) {
	m.DistanceCount++
	m.TotalDistTime += duration
	if err != nil {
		m.ErrorCount++
	}
}

func (m *SimpleMetricsCollector) RecordCompressionRatio(qType QuantizationType, ratio float64) {
	// 简单实现，不记录
}

// Stats 返回统计摘要
func (m *SimpleMetricsCollector) Stats() map[string]interface{} {
	return map[string]interface{}{
		"training_count":      m.TrainingCount,
		"encoding_count":      m.EncodingCount,
		"decoding_count":      m.DecodingCount,
		"distance_count":      m.DistanceCount,
		"total_train_time_ms": m.TotalTrainTime.Milliseconds(),
		"total_encode_time_ms": m.TotalEncodeTime.Milliseconds(),
		"total_decode_time_ms": m.TotalDecodeTime.Milliseconds(),
		"total_dist_time_us":  m.TotalDistTime.Microseconds(),
		"error_count":         m.ErrorCount,
	}
}
