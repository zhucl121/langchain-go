package quantization

import (
	"time"
	
	"github.com/prometheus/client_golang/prometheus"
)

// PrometheusMetricsCollector Prometheus 指标收集器
type PrometheusMetricsCollector struct {
	// 训练指标
	trainingTotal    *prometheus.CounterVec
	trainingDuration *prometheus.HistogramVec
	trainingErrors   *prometheus.CounterVec
	
	// 编码指标
	encodingTotal    *prometheus.CounterVec
	encodingDuration *prometheus.HistogramVec
	encodingErrors   *prometheus.CounterVec
	
	// 解码指标
	decodingTotal    *prometheus.CounterVec
	decodingDuration *prometheus.HistogramVec
	decodingErrors   *prometheus.CounterVec
	
	// 距离计算指标
	distanceTotal    *prometheus.CounterVec
	distanceDuration *prometheus.HistogramVec
	distanceErrors   *prometheus.CounterVec
	
	// 压缩比指标
	compressionRatio *prometheus.GaugeVec
	
	registry *prometheus.Registry
}

// NewPrometheusMetricsCollector 创建 Prometheus 指标收集器
func NewPrometheusMetricsCollector(namespace, subsystem string, registry *prometheus.Registry) *PrometheusMetricsCollector {
	if namespace == "" {
		namespace = "langchain"
	}
	if subsystem == "" {
		subsystem = "quantization"
	}
	if registry == nil {
		registry = prometheus.NewRegistry()
	}
	
	collector := &PrometheusMetricsCollector{
		registry: registry,
	}
	
	// 训练指标
	collector.trainingTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "training_total",
			Help:      "Total number of quantizer training operations",
		},
		[]string{"type", "status"},
	)
	
	collector.trainingDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "training_duration_seconds",
			Help:      "Duration of quantizer training operations",
			Buckets:   []float64{0.001, 0.01, 0.1, 0.5, 1, 5, 10, 30},
		},
		[]string{"type"},
	)
	
	collector.trainingErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "training_errors_total",
			Help:      "Total number of quantizer training errors",
		},
		[]string{"type"},
	)
	
	// 编码指标
	collector.encodingTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "encoding_total",
			Help:      "Total number of encoding operations",
		},
		[]string{"type", "status"},
	)
	
	collector.encodingDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "encoding_duration_seconds",
			Help:      "Duration of encoding operations",
			Buckets:   []float64{0.0001, 0.001, 0.01, 0.1, 0.5, 1},
		},
		[]string{"type"},
	)
	
	collector.encodingErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "encoding_errors_total",
			Help:      "Total number of encoding errors",
		},
		[]string{"type"},
	)
	
	// 解码指标
	collector.decodingTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "decoding_total",
			Help:      "Total number of decoding operations",
		},
		[]string{"type", "status"},
	)
	
	collector.decodingDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "decoding_duration_seconds",
			Help:      "Duration of decoding operations",
			Buckets:   []float64{0.0001, 0.001, 0.01, 0.1, 0.5, 1},
		},
		[]string{"type"},
	)
	
	collector.decodingErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "decoding_errors_total",
			Help:      "Total number of decoding errors",
		},
		[]string{"type"},
	)
	
	// 距离计算指标
	collector.distanceTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "distance_computation_total",
			Help:      "Total number of distance computations",
		},
		[]string{"type", "status"},
	)
	
	collector.distanceDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "distance_computation_duration_seconds",
			Help:      "Duration of distance computations",
			Buckets:   []float64{0.00001, 0.0001, 0.001, 0.01, 0.1},
		},
		[]string{"type"},
	)
	
	collector.distanceErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "distance_computation_errors_total",
			Help:      "Total number of distance computation errors",
		},
		[]string{"type"},
	)
	
	// 压缩比指标
	collector.compressionRatio = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "compression_ratio",
			Help:      "Current compression ratio",
		},
		[]string{"type"},
	)
	
	// 注册所有指标
	registry.MustRegister(
		collector.trainingTotal,
		collector.trainingDuration,
		collector.trainingErrors,
		collector.encodingTotal,
		collector.encodingDuration,
		collector.encodingErrors,
		collector.decodingTotal,
		collector.decodingDuration,
		collector.decodingErrors,
		collector.distanceTotal,
		collector.distanceDuration,
		collector.distanceErrors,
		collector.compressionRatio,
	)
	
	return collector
}

func (c *PrometheusMetricsCollector) RecordTraining(qType QuantizationType, duration time.Duration, vectorCount int, err error) {
	status := "success"
	if err != nil {
		status = "error"
		c.trainingErrors.WithLabelValues(string(qType)).Inc()
	}
	
	c.trainingTotal.WithLabelValues(string(qType), status).Inc()
	c.trainingDuration.WithLabelValues(string(qType)).Observe(duration.Seconds())
}

func (c *PrometheusMetricsCollector) RecordEncoding(qType QuantizationType, duration time.Duration, vectorCount int, err error) {
	status := "success"
	if err != nil {
		status = "error"
		c.encodingErrors.WithLabelValues(string(qType)).Inc()
	}
	
	c.encodingTotal.WithLabelValues(string(qType), status).Inc()
	c.encodingDuration.WithLabelValues(string(qType)).Observe(duration.Seconds())
}

func (c *PrometheusMetricsCollector) RecordDecoding(qType QuantizationType, duration time.Duration, vectorCount int, err error) {
	status := "success"
	if err != nil {
		status = "error"
		c.decodingErrors.WithLabelValues(string(qType)).Inc()
	}
	
	c.decodingTotal.WithLabelValues(string(qType), status).Inc()
	c.decodingDuration.WithLabelValues(string(qType)).Observe(duration.Seconds())
}

func (c *PrometheusMetricsCollector) RecordDistance(qType QuantizationType, duration time.Duration, vectorCount int, err error) {
	status := "success"
	if err != nil {
		status = "error"
		c.distanceErrors.WithLabelValues(string(qType)).Inc()
	}
	
	c.distanceTotal.WithLabelValues(string(qType), status).Inc()
	c.distanceDuration.WithLabelValues(string(qType)).Observe(duration.Seconds())
}

func (c *PrometheusMetricsCollector) RecordCompressionRatio(qType QuantizationType, ratio float64) {
	c.compressionRatio.WithLabelValues(string(qType)).Set(ratio)
}

// GetRegistry 返回 Prometheus Registry
func (c *PrometheusMetricsCollector) GetRegistry() *prometheus.Registry {
	return c.registry
}
