// Package quantization 提供向量量化功能，用于压缩和优化向量存储。
//
// 向量量化可以大幅降低内存占用和提高检索速度，支持以下量化方法：
//   - Scalar Quantization: 标量量化（8-bit, 4-bit, 2-bit）
//   - Binary Quantization: 二值量化（1-bit）
//   - Product Quantization: 乘积量化（PQ）
//
// 使用示例：
//
//	// Scalar Quantization (8-bit)
//	config := quantization.ScalarQuantizationConfig{
//	    Bits: 8,
//	}
//	quantizer := quantization.NewScalarQuantizer(config)
//	quantized := quantizer.Encode(vectors)
//	reconstructed := quantizer.Decode(quantized)
//
//	// Binary Quantization
//	config := quantization.BinaryQuantizationConfig{}
//	quantizer := quantization.NewBinaryQuantizer(config)
//	quantized := quantizer.Encode(vectors)
//	distance := quantizer.ComputeDistance(quantized1, quantized2)
//
//	// Product Quantization
//	config := quantization.ProductQuantizationConfig{
//	    M: 8,        // 子向量数量
//	    NBits: 8,    // 每个子向量的编码位数
//	}
//	quantizer := quantization.NewProductQuantizer(config)
//	quantizer.Train(trainingData)
//	quantized := quantizer.Encode(vectors)
//
package quantization

import (
	"context"
	"errors"
	"fmt"
)

// 错误定义
var (
	ErrInvalidDimension  = errors.New("quantization: invalid dimension")
	ErrInvalidBits       = errors.New("quantization: invalid bits parameter")
	ErrNotTrained        = errors.New("quantization: quantizer not trained")
	ErrInvalidM          = errors.New("quantization: invalid M parameter (must divide dimension)")
	ErrInvalidNBits      = errors.New("quantization: invalid NBits parameter (must be 1-16)")
	ErrInsufficientData  = errors.New("quantization: insufficient training data")
)

// QuantizationType 量化类型
type QuantizationType string

const (
	// QuantizationNone 无量化
	QuantizationNone QuantizationType = "none"
	
	// QuantizationScalar 标量量化
	QuantizationScalar QuantizationType = "scalar"
	
	// QuantizationBinary 二值量化
	QuantizationBinary QuantizationType = "binary"
	
	// QuantizationProduct 乘积量化
	QuantizationProduct QuantizationType = "product"
)

// Config 量化配置
type Config struct {
	// Type 量化类型
	Type QuantizationType
	
	// Bits 量化位数（用于 Scalar Quantization）
	// 支持: 8, 4, 2, 1
	Bits int
	
	// M 子向量数量（用于 Product Quantization）
	// 必须能整除向量维度
	M int
	
	// NBits 每个子向量的编码位数（用于 Product Quantization）
	// 取值范围: 1-16
	NBits int
	
	// TrainingSize 训练样本数量（用于 Product Quantization）
	// 建议: 至少 1000 个样本
	TrainingSize int
}

// DefaultConfig 返回默认配置
func DefaultConfig() Config {
	return Config{
		Type:         QuantizationNone,
		Bits:         8,
		M:            8,
		NBits:        8,
		TrainingSize: 1000,
	}
}

// Validate 验证配置
func (c Config) Validate() error {
	switch c.Type {
	case QuantizationNone:
		return nil
		
	case QuantizationScalar:
		if c.Bits != 8 && c.Bits != 4 && c.Bits != 2 && c.Bits != 1 {
			return fmt.Errorf("%w: must be 1, 2, 4, or 8", ErrInvalidBits)
		}
		
	case QuantizationBinary:
		// Binary quantization doesn't need additional validation
		
	case QuantizationProduct:
		if c.M <= 0 {
			return fmt.Errorf("%w: M must be positive", ErrInvalidM)
		}
		if c.NBits < 1 || c.NBits > 16 {
			return fmt.Errorf("%w: NBits must be between 1 and 16", ErrInvalidNBits)
		}
		if c.TrainingSize < 100 {
			return fmt.Errorf("%w: TrainingSize must be at least 100", ErrInsufficientData)
		}
		
	default:
		return fmt.Errorf("quantization: unsupported type: %s", c.Type)
	}
	
	return nil
}

// Quantizer 量化器接口
//
// 量化器负责将浮点向量压缩为低精度表示，以降低内存占用和提高检索速度。
type Quantizer interface {
	// Type 返回量化类型
	Type() QuantizationType
	
	// Dimension 返回向量维度
	Dimension() int
	
	// Train 训练量化器（仅 Product Quantization 需要）
	//
	// 参数：
	//   - ctx: 上下文
	//   - vectors: 训练数据（多个向量）
	//
	// 返回：
	//   - error: 错误
	Train(ctx context.Context, vectors [][]float32) error
	
	// Encode 编码向量
	//
	// 参数：
	//   - vectors: 原始向量
	//
	// 返回：
	//   - QuantizedVectors: 量化后的向量
	//   - error: 错误
	Encode(vectors [][]float32) (QuantizedVectors, error)
	
	// Decode 解码向量（近似重构）
	//
	// 参数：
	//   - quantized: 量化后的向量
	//
	// 返回：
	//   - [][]float32: 重构的向量
	//   - error: 错误
	Decode(quantized QuantizedVectors) ([][]float32, error)
	
	// ComputeDistance 计算量化向量之间的距离
	//
	// 参数：
	//   - query: 查询向量（量化后）
	//   - vectors: 候选向量列表（量化后）
	//
	// 返回：
	//   - []float32: 距离列表
	//   - error: 错误
	ComputeDistance(query QuantizedVector, vectors []QuantizedVector) ([]float32, error)
	
	// CompressionRatio 返回压缩比
	//
	// 返回：
	//   - float64: 压缩比 (原始大小 / 压缩大小)
	CompressionRatio() float64
	
	// IsTrained 返回量化器是否已训练
	//
	// 返回：
	//   - bool: 是否已训练
	IsTrained() bool
}

// QuantizedVector 量化后的单个向量
type QuantizedVector interface {
	// Type 返回量化类型
	Type() QuantizationType
	
	// Data 返回量化数据
	Data() []byte
	
	// Size 返回数据大小（字节）
	Size() int
}

// QuantizedVectors 量化后的向量集合
type QuantizedVectors interface {
	// Type 返回量化类型
	Type() QuantizationType
	
	// Count 返回向量数量
	Count() int
	
	// Get 获取指定索引的向量
	Get(index int) (QuantizedVector, error)
	
	// Data 返回所有量化数据
	Data() []byte
	
	// TotalSize 返回总大小（字节）
	TotalSize() int
}

// Statistics 量化统计信息
type Statistics struct {
	// Type 量化类型
	Type QuantizationType
	
	// OriginalSize 原始大小（字节）
	OriginalSize int64
	
	// CompressedSize 压缩后大小（字节）
	CompressedSize int64
	
	// CompressionRatio 压缩比
	CompressionRatio float64
	
	// VectorCount 向量数量
	VectorCount int
	
	// Dimension 向量维度
	Dimension int
	
	// QuantizationError 量化误差（MSE）
	QuantizationError float64
	
	// TrainingTime 训练时间（秒）
	TrainingTime float64
	
	// EncodingTime 编码时间（秒）
	EncodingTime float64
}

// NewQuantizer 创建量化器
//
// 参数：
//   - config: 量化配置
//   - dimension: 向量维度
//
// 返回：
//   - Quantizer: 量化器实例
//   - error: 错误
func NewQuantizer(config Config, dimension int) (Quantizer, error) {
	if dimension <= 0 {
		return nil, fmt.Errorf("%w: dimension must be positive", ErrInvalidDimension)
	}
	
	if err := config.Validate(); err != nil {
		return nil, err
	}
	
	switch config.Type {
	case QuantizationNone:
		return NewNoOpQuantizer(dimension), nil
		
	case QuantizationScalar:
		return NewScalarQuantizer(ScalarQuantizationConfig{
			Bits:      config.Bits,
			Dimension: dimension,
		}), nil
		
	case QuantizationBinary:
		return NewBinaryQuantizer(BinaryQuantizationConfig{
			Dimension: dimension,
		}), nil
		
	case QuantizationProduct:
		if dimension%config.M != 0 {
			return nil, fmt.Errorf("%w: dimension %d must be divisible by M %d", 
				ErrInvalidM, dimension, config.M)
		}
		return NewProductQuantizer(ProductQuantizationConfig{
			Dimension:    dimension,
			M:            config.M,
			NBits:        config.NBits,
			TrainingSize: config.TrainingSize,
		}), nil
		
	default:
		return nil, fmt.Errorf("quantization: unsupported type: %s", config.Type)
	}
}

// NoOpQuantizer 无量化器（用于测试和对比）
type NoOpQuantizer struct {
	dimension int
}

// NewNoOpQuantizer 创建无量化器
func NewNoOpQuantizer(dimension int) *NoOpQuantizer {
	return &NoOpQuantizer{dimension: dimension}
}

func (q *NoOpQuantizer) Type() QuantizationType { return QuantizationNone }
func (q *NoOpQuantizer) Dimension() int         { return q.dimension }
func (q *NoOpQuantizer) IsTrained() bool        { return true }
func (q *NoOpQuantizer) CompressionRatio() float64 { return 1.0 }

func (q *NoOpQuantizer) Train(ctx context.Context, vectors [][]float32) error {
	return nil
}

func (q *NoOpQuantizer) Encode(vectors [][]float32) (QuantizedVectors, error) {
	// 直接返回原始向量（不压缩）
	return &noOpQuantizedVectors{vectors: vectors}, nil
}

func (q *NoOpQuantizer) Decode(quantized QuantizedVectors) ([][]float32, error) {
	if noop, ok := quantized.(*noOpQuantizedVectors); ok {
		return noop.vectors, nil
	}
	return nil, errors.New("quantization: invalid quantized vectors type")
}

func (q *NoOpQuantizer) ComputeDistance(query QuantizedVector, vectors []QuantizedVector) ([]float32, error) {
	return nil, errors.New("quantization: NoOpQuantizer does not support distance computation")
}

// noOpQuantizedVectors 无量化的向量集合
type noOpQuantizedVectors struct {
	vectors [][]float32
}

func (v *noOpQuantizedVectors) Type() QuantizationType { return QuantizationNone }
func (v *noOpQuantizedVectors) Count() int             { return len(v.vectors) }
func (v *noOpQuantizedVectors) Data() []byte           { return nil }

func (v *noOpQuantizedVectors) TotalSize() int {
	if len(v.vectors) == 0 {
		return 0
	}
	return len(v.vectors) * len(v.vectors[0]) * 4 // float32 = 4 bytes
}

func (v *noOpQuantizedVectors) Get(index int) (QuantizedVector, error) {
	if index < 0 || index >= len(v.vectors) {
		return nil, errors.New("quantization: index out of range")
	}
	return &noOpQuantizedVector{vector: v.vectors[index]}, nil
}

// noOpQuantizedVector 无量化的单个向量
type noOpQuantizedVector struct {
	vector []float32
}

func (v *noOpQuantizedVector) Type() QuantizationType { return QuantizationNone }
func (v *noOpQuantizedVector) Data() []byte           { return nil }
func (v *noOpQuantizedVector) Size() int              { return len(v.vector) * 4 }
