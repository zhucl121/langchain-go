package quantization

import (
	"context"
	"errors"
	"fmt"
	"math"
)

// BinaryQuantizationConfig 二值量化配置
type BinaryQuantizationConfig struct {
	// Dimension 向量维度
	Dimension int
	
	// Threshold 二值化阈值
	// 如果为 0，则使用向量均值作为阈值
	Threshold float32
	
	// UseMedian 是否使用中位数作为阈值
	UseMedian bool
}

// BinaryQuantizer 二值量化器
//
// 将浮点向量量化为二值（0/1），实现最大压缩比（96.875%）。
// 适合高维向量的快速检索，使用 Hamming 距离计算。
type BinaryQuantizer struct {
	config    BinaryQuantizationConfig
	threshold float32 // 全局阈值（从训练数据中学习）
	trained   bool
}

// NewBinaryQuantizer 创建二值量化器
func NewBinaryQuantizer(config BinaryQuantizationConfig) *BinaryQuantizer {
	return &BinaryQuantizer{
		config:  config,
		trained: false,
	}
}

func (q *BinaryQuantizer) Type() QuantizationType {
	return QuantizationBinary
}

func (q *BinaryQuantizer) Dimension() int {
	return q.config.Dimension
}

func (q *BinaryQuantizer) IsTrained() bool {
	return q.trained
}

func (q *BinaryQuantizer) CompressionRatio() float64 {
	// 原始: float32 = 32 bits
	// 量化: 1 bit
	return 32.0
}

// Train 训练量化器（计算全局阈值）
func (q *BinaryQuantizer) Train(ctx context.Context, vectors [][]float32) error {
	if len(vectors) == 0 {
		return ErrInsufficientData
	}
	
	// 检查维度
	for _, vec := range vectors {
		if len(vec) != q.config.Dimension {
			return fmt.Errorf("%w: expected %d, got %d",
				ErrInvalidDimension, q.config.Dimension, len(vec))
		}
	}
	
	// 如果配置了固定阈值，直接使用
	if q.config.Threshold != 0 {
		q.threshold = q.config.Threshold
		q.trained = true
		return nil
	}
	
	// 收集所有值
	totalElements := len(vectors) * q.config.Dimension
	allValues := make([]float32, 0, totalElements)
	
	for _, vec := range vectors {
		allValues = append(allValues, vec...)
	}
	
	// 计算阈值
	if q.config.UseMedian {
		q.threshold = median(allValues)
	} else {
		q.threshold = mean(allValues)
	}
	
	q.trained = true
	return nil
}

// Encode 编码向量
func (q *BinaryQuantizer) Encode(vectors [][]float32) (QuantizedVectors, error) {
	if !q.trained {
		// 如果未训练，使用当前数据进行训练
		if err := q.Train(context.Background(), vectors); err != nil {
			return nil, err
		}
	}
	
	// 验证维度
	for i, vec := range vectors {
		if len(vec) != q.config.Dimension {
			return nil, fmt.Errorf("%w: vector %d has dimension %d, expected %d",
				ErrInvalidDimension, i, len(vec), q.config.Dimension)
		}
	}
	
	// 每个向量占用 (dimension + 7) / 8 字节
	bytesPerVector := (q.config.Dimension + 7) / 8
	data := make([]byte, len(vectors)*bytesPerVector)
	
	idx := 0
	for _, vec := range vectors {
		for i, val := range vec {
			if val >= q.threshold {
				// 设置对应的 bit 为 1
				byteIdx := idx + i/8
				bitIdx := uint(7 - i%8)
				data[byteIdx] |= 1 << bitIdx
			}
		}
		idx += bytesPerVector
	}
	
	return &binaryQuantizedVectors{
		dimension: q.config.Dimension,
		count:     len(vectors),
		data:      data,
		threshold: q.threshold,
	}, nil
}

// Decode 解码向量（重构为 0/1 向量）
func (q *BinaryQuantizer) Decode(quantized QuantizedVectors) ([][]float32, error) {
	bq, ok := quantized.(*binaryQuantizedVectors)
	if !ok {
		return nil, errors.New("quantization: invalid quantized vectors type")
	}
	
	bytesPerVector := (q.config.Dimension + 7) / 8
	vectors := make([][]float32, bq.count)
	
	for i := 0; i < bq.count; i++ {
		vectors[i] = make([]float32, q.config.Dimension)
		start := i * bytesPerVector
		
		for j := 0; j < q.config.Dimension; j++ {
			byteIdx := start + j/8
			bitIdx := uint(7 - j%8)
			bit := (bq.data[byteIdx] >> bitIdx) & 0x01
			
			if bit == 1 {
				vectors[i][j] = 1.0
			} else {
				vectors[i][j] = 0.0
			}
		}
	}
	
	return vectors, nil
}

// ComputeDistance 计算 Hamming 距离
func (q *BinaryQuantizer) ComputeDistance(query QuantizedVector, vectors []QuantizedVector) ([]float32, error) {
	bq, ok := query.(*binaryQuantizedVector)
	if !ok {
		return nil, errors.New("quantization: invalid query vector type")
	}
	
	distances := make([]float32, len(vectors))
	
	for i, vec := range vectors {
		bv, ok := vec.(*binaryQuantizedVector)
		if !ok {
			return nil, fmt.Errorf("quantization: invalid vector type at index %d", i)
		}
		
		distances[i] = q.computeHammingDistance(bq, bv)
	}
	
	return distances, nil
}

// computeHammingDistance 计算 Hamming 距离
func (q *BinaryQuantizer) computeHammingDistance(a, b *binaryQuantizedVector) float32 {
	if len(a.data) != len(b.data) {
		return float32(math.MaxFloat32)
	}
	
	distance := 0
	for i := 0; i < len(a.data); i++ {
		xor := a.data[i] ^ b.data[i]
		// 计算 XOR 结果中 1 的个数（即不同的 bit 数）
		distance += popcount(xor)
	}
	
	return float32(distance)
}

// mean 计算均值
func mean(values []float32) float32 {
	if len(values) == 0 {
		return 0
	}
	
	var sum float32
	for _, v := range values {
		sum += v
	}
	return sum / float32(len(values))
}

// median 计算中位数
func median(values []float32) float32 {
	if len(values) == 0 {
		return 0
	}
	
	// 复制并排序
	sorted := make([]float32, len(values))
	copy(sorted, values)
	
	// 简单排序（对于大数据集，应该使用更高效的算法）
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i] > sorted[j] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}
	
	mid := len(sorted) / 2
	if len(sorted)%2 == 0 {
		return (sorted[mid-1] + sorted[mid]) / 2
	}
	return sorted[mid]
}

// binaryQuantizedVectors 二值量化的向量集合
type binaryQuantizedVectors struct {
	dimension int
	count     int
	data      []byte
	threshold float32
}

func (v *binaryQuantizedVectors) Type() QuantizationType {
	return QuantizationBinary
}

func (v *binaryQuantizedVectors) Count() int {
	return v.count
}

func (v *binaryQuantizedVectors) Data() []byte {
	return v.data
}

func (v *binaryQuantizedVectors) TotalSize() int {
	return len(v.data) + 4 // data + threshold
}

func (v *binaryQuantizedVectors) Get(index int) (QuantizedVector, error) {
	if index < 0 || index >= v.count {
		return nil, errors.New("quantization: index out of range")
	}
	
	bytesPerVector := (v.dimension + 7) / 8
	start := index * bytesPerVector
	end := start + bytesPerVector
	
	vecData := make([]byte, bytesPerVector)
	copy(vecData, v.data[start:end])
	
	return &binaryQuantizedVector{
		dimension: v.dimension,
		data:      vecData,
		threshold: v.threshold,
	}, nil
}

// binaryQuantizedVector 二值量化的单个向量
type binaryQuantizedVector struct {
	dimension int
	data      []byte
	threshold float32
}

func (v *binaryQuantizedVector) Type() QuantizationType {
	return QuantizationBinary
}

func (v *binaryQuantizedVector) Data() []byte {
	return v.data
}

func (v *binaryQuantizedVector) Size() int {
	return len(v.data) + 4 // data + threshold
}

// HammingWeight 返回向量的 Hamming 权重（1 的个数）
func (v *binaryQuantizedVector) HammingWeight() int {
	weight := 0
	for _, b := range v.data {
		weight += popcount(b)
	}
	return weight
}

// BitwiseAND 按位与操作
func (v *binaryQuantizedVector) BitwiseAND(other *binaryQuantizedVector) *binaryQuantizedVector {
	if len(v.data) != len(other.data) {
		return nil
	}
	
	result := &binaryQuantizedVector{
		dimension: v.dimension,
		data:      make([]byte, len(v.data)),
		threshold: v.threshold,
	}
	
	for i := 0; i < len(v.data); i++ {
		result.data[i] = v.data[i] & other.data[i]
	}
	
	return result
}

// BitwiseOR 按位或操作
func (v *binaryQuantizedVector) BitwiseOR(other *binaryQuantizedVector) *binaryQuantizedVector {
	if len(v.data) != len(other.data) {
		return nil
	}
	
	result := &binaryQuantizedVector{
		dimension: v.dimension,
		data:      make([]byte, len(v.data)),
		threshold: v.threshold,
	}
	
	for i := 0; i < len(v.data); i++ {
		result.data[i] = v.data[i] | other.data[i]
	}
	
	return result
}

// BitwiseXOR 按位异或操作
func (v *binaryQuantizedVector) BitwiseXOR(other *binaryQuantizedVector) *binaryQuantizedVector {
	if len(v.data) != len(other.data) {
		return nil
	}
	
	result := &binaryQuantizedVector{
		dimension: v.dimension,
		data:      make([]byte, len(v.data)),
		threshold: v.threshold,
	}
	
	for i := 0; i < len(v.data); i++ {
		result.data[i] = v.data[i] ^ other.data[i]
	}
	
	return result
}

// JaccardSimilarity 计算 Jaccard 相似度
func (v *binaryQuantizedVector) JaccardSimilarity(other *binaryQuantizedVector) float32 {
	and := v.BitwiseAND(other)
	or := v.BitwiseOR(other)
	
	if and == nil || or == nil {
		return 0
	}
	
	andWeight := and.HammingWeight()
	orWeight := or.HammingWeight()
	
	if orWeight == 0 {
		return 1.0 // 两个向量都是零向量
	}
	
	return float32(andWeight) / float32(orWeight)
}

// CosineSimilarity 计算余弦相似度（二值向量）
func (v *binaryQuantizedVector) CosineSimilarity(other *binaryQuantizedVector) float32 {
	and := v.BitwiseAND(other)
	if and == nil {
		return 0
	}
	
	dotProduct := float32(and.HammingWeight())
	normA := float32(v.HammingWeight())
	normB := float32(other.HammingWeight())
	
	if normA == 0 || normB == 0 {
		return 0
	}
	
	return dotProduct / float32(math.Sqrt(float64(normA*normB)))
}
