package quantization

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
)

// ScalarQuantizationConfig 标量量化配置
type ScalarQuantizationConfig struct {
	// Bits 量化位数 (1, 2, 4, 8)
	Bits int
	
	// Dimension 向量维度
	Dimension int
	
	// ClipRange 剪切范围 (用于限制极端值)
	// 如果为 0，则自动从数据中计算
	ClipRange float32
	
	// UseSymmetric 是否使用对称量化
	// true: [-range, range]
	// false: [min, max]
	UseSymmetric bool
}

// ScalarQuantizer 标量量化器
//
// 将浮点向量量化为低精度整数（8-bit, 4-bit, 2-bit, 1-bit）。
// 内存节省：8-bit (75%), 4-bit (87.5%), 2-bit (93.75%), 1-bit (96.875%)
type ScalarQuantizer struct {
	config ScalarQuantizationConfig
	
	// 量化参数（从训练数据中学习）
	scale  float32  // 缩放因子
	offset float32  // 偏移量（非对称量化）
	min    float32  // 最小值
	max    float32  // 最大值
	
	trained bool
}

// NewScalarQuantizer 创建标量量化器
func NewScalarQuantizer(config ScalarQuantizationConfig) *ScalarQuantizer {
	return &ScalarQuantizer{
		config:  config,
		trained: false,
	}
}

func (q *ScalarQuantizer) Type() QuantizationType {
	return QuantizationScalar
}

func (q *ScalarQuantizer) Dimension() int {
	return q.config.Dimension
}

func (q *ScalarQuantizer) IsTrained() bool {
	return q.trained
}

func (q *ScalarQuantizer) CompressionRatio() float64 {
	// 原始: float32 = 32 bits
	// 量化: config.Bits bits
	return 32.0 / float64(q.config.Bits)
}

// Train 训练量化器（计算缩放因子和偏移量）
func (q *ScalarQuantizer) Train(ctx context.Context, vectors [][]float32) error {
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
	
	// 计算全局最小值和最大值
	q.min = math.MaxFloat32
	q.max = -math.MaxFloat32
	
	for _, vec := range vectors {
		for _, val := range vec {
			if val < q.min {
				q.min = val
			}
			if val > q.max {
				q.max = val
			}
		}
	}
	
	// 应用剪切范围
	if q.config.ClipRange > 0 {
		clipMin := -q.config.ClipRange
		clipMax := q.config.ClipRange
		if q.min < clipMin {
			q.min = clipMin
		}
		if q.max > clipMax {
			q.max = clipMax
		}
	}
	
	// 计算量化参数
	q.computeQuantizationParams()
	q.trained = true
	
	return nil
}

// computeQuantizationParams 计算量化参数
func (q *ScalarQuantizer) computeQuantizationParams() {
	qmax := float32(uint32(1<<q.config.Bits) - 1) // 量化后的最大值
	
	if q.config.UseSymmetric {
		// 对称量化: [-range, range] -> [0, qmax]
		absMax := float32(math.Max(math.Abs(float64(q.min)), math.Abs(float64(q.max))))
		q.scale = qmax / (2.0 * absMax)
		q.offset = qmax / 2.0
	} else {
		// 非对称量化: [min, max] -> [0, qmax]
		q.scale = qmax / (q.max - q.min)
		q.offset = -q.min * q.scale
	}
}

// Encode 编码向量
func (q *ScalarQuantizer) Encode(vectors [][]float32) (QuantizedVectors, error) {
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
	
	var encoded []byte
	
	switch q.config.Bits {
	case 8:
		encoded = q.encode8bit(vectors)
	case 4:
		encoded = q.encode4bit(vectors)
	case 2:
		encoded = q.encode2bit(vectors)
	case 1:
		encoded = q.encode1bit(vectors)
	default:
		return nil, fmt.Errorf("%w: %d", ErrInvalidBits, q.config.Bits)
	}
	
	return &scalarQuantizedVectors{
		bits:      q.config.Bits,
		dimension: q.config.Dimension,
		count:     len(vectors),
		data:      encoded,
		scale:     q.scale,
		offset:    q.offset,
	}, nil
}

// encode8bit 8-bit 量化
func (q *ScalarQuantizer) encode8bit(vectors [][]float32) []byte {
	data := make([]byte, len(vectors)*q.config.Dimension)
	idx := 0
	
	for _, vec := range vectors {
		for _, val := range vec {
			// 量化: val -> [0, 255]
			quantized := val*q.scale + q.offset
			quantized = float32(math.Max(0, math.Min(255, float64(quantized))))
			data[idx] = uint8(quantized)
			idx++
		}
	}
	
	return data
}

// encode4bit 4-bit 量化
func (q *ScalarQuantizer) encode4bit(vectors [][]float32) []byte {
	// 每个字节存储2个4-bit值
	totalElements := len(vectors) * q.config.Dimension
	data := make([]byte, (totalElements+1)/2)
	idx := 0
	
	for _, vec := range vectors {
		for _, val := range vec {
			// 量化: val -> [0, 15]
			quantized := val*q.scale + q.offset
			quantized = float32(math.Max(0, math.Min(15, float64(quantized))))
			
			nibble := uint8(quantized)
			if idx%2 == 0 {
				data[idx/2] = nibble << 4
			} else {
				data[idx/2] |= nibble
			}
			idx++
		}
	}
	
	return data
}

// encode2bit 2-bit 量化
func (q *ScalarQuantizer) encode2bit(vectors [][]float32) []byte {
	// 每个字节存储4个2-bit值
	totalElements := len(vectors) * q.config.Dimension
	data := make([]byte, (totalElements+3)/4)
	idx := 0
	
	for _, vec := range vectors {
		for _, val := range vec {
			// 量化: val -> [0, 3]
			quantized := val*q.scale + q.offset
			quantized = float32(math.Max(0, math.Min(3, float64(quantized))))
			
			bits := uint8(quantized)
			shift := uint(6 - (idx%4)*2)
			data[idx/4] |= bits << shift
			idx++
		}
	}
	
	return data
}

// encode1bit 1-bit 量化 (Binary Quantization的特殊情况)
func (q *ScalarQuantizer) encode1bit(vectors [][]float32) []byte {
	// 每个字节存储8个1-bit值
	totalElements := len(vectors) * q.config.Dimension
	data := make([]byte, (totalElements+7)/8)
	idx := 0
	
	for _, vec := range vectors {
		for _, val := range vec {
			// 量化: val -> 0 or 1
			quantized := val*q.scale + q.offset
			bit := uint8(0)
			if quantized > 0.5 {
				bit = 1
			}
			
			if bit == 1 {
				data[idx/8] |= 1 << uint(7-idx%8)
			}
			idx++
		}
	}
	
	return data
}

// Decode 解码向量
func (q *ScalarQuantizer) Decode(quantized QuantizedVectors) ([][]float32, error) {
	sq, ok := quantized.(*scalarQuantizedVectors)
	if !ok {
		return nil, errors.New("quantization: invalid quantized vectors type")
	}
	
	if sq.bits != q.config.Bits {
		return nil, fmt.Errorf("quantization: bits mismatch: expected %d, got %d",
			q.config.Bits, sq.bits)
	}
	
	vectors := make([][]float32, sq.count)
	
	switch q.config.Bits {
	case 8:
		q.decode8bit(sq, vectors)
	case 4:
		q.decode4bit(sq, vectors)
	case 2:
		q.decode2bit(sq, vectors)
	case 1:
		q.decode1bit(sq, vectors)
	default:
		return nil, fmt.Errorf("%w: %d", ErrInvalidBits, q.config.Bits)
	}
	
	return vectors, nil
}

// decode8bit 8-bit 解码
func (q *ScalarQuantizer) decode8bit(sq *scalarQuantizedVectors, vectors [][]float32) {
	idx := 0
	for i := 0; i < sq.count; i++ {
		vectors[i] = make([]float32, sq.dimension)
		for j := 0; j < sq.dimension; j++ {
			quantized := float32(sq.data[idx])
			vectors[i][j] = (quantized - sq.offset) / sq.scale
			idx++
		}
	}
}

// decode4bit 4-bit 解码
func (q *ScalarQuantizer) decode4bit(sq *scalarQuantizedVectors, vectors [][]float32) {
	idx := 0
	for i := 0; i < sq.count; i++ {
		vectors[i] = make([]float32, sq.dimension)
		for j := 0; j < sq.dimension; j++ {
			var nibble uint8
			if idx%2 == 0 {
				nibble = sq.data[idx/2] >> 4
			} else {
				nibble = sq.data[idx/2] & 0x0F
			}
			quantized := float32(nibble)
			vectors[i][j] = (quantized - sq.offset) / sq.scale
			idx++
		}
	}
}

// decode2bit 2-bit 解码
func (q *ScalarQuantizer) decode2bit(sq *scalarQuantizedVectors, vectors [][]float32) {
	idx := 0
	for i := 0; i < sq.count; i++ {
		vectors[i] = make([]float32, sq.dimension)
		for j := 0; j < sq.dimension; j++ {
			shift := uint(6 - (idx%4)*2)
			bits := (sq.data[idx/4] >> shift) & 0x03
			quantized := float32(bits)
			vectors[i][j] = (quantized - sq.offset) / sq.scale
			idx++
		}
	}
}

// decode1bit 1-bit 解码
func (q *ScalarQuantizer) decode1bit(sq *scalarQuantizedVectors, vectors [][]float32) {
	idx := 0
	for i := 0; i < sq.count; i++ {
		vectors[i] = make([]float32, sq.dimension)
		for j := 0; j < sq.dimension; j++ {
			bit := (sq.data[idx/8] >> uint(7-idx%8)) & 0x01
			quantized := float32(bit)
			vectors[i][j] = (quantized - sq.offset) / sq.scale
			idx++
		}
	}
}

// ComputeDistance 计算量化向量之间的距离
func (q *ScalarQuantizer) ComputeDistance(query QuantizedVector, vectors []QuantizedVector) ([]float32, error) {
	sq, ok := query.(*scalarQuantizedVector)
	if !ok {
		return nil, errors.New("quantization: invalid query vector type")
	}
	
	distances := make([]float32, len(vectors))
	
	for i, vec := range vectors {
		sv, ok := vec.(*scalarQuantizedVector)
		if !ok {
			return nil, fmt.Errorf("quantization: invalid vector type at index %d", i)
		}
		
		distances[i] = q.computeL2Distance(sq, sv)
	}
	
	return distances, nil
}

// computeL2Distance 计算 L2 距离（在量化空间中）
func (q *ScalarQuantizer) computeL2Distance(a, b *scalarQuantizedVector) float32 {
	var sum float32
	
	switch q.config.Bits {
	case 8:
		for i := 0; i < q.config.Dimension; i++ {
			diff := float32(a.data[i]) - float32(b.data[i])
			sum += diff * diff
		}
	case 4:
		for i := 0; i < q.config.Dimension; i++ {
			var aNibble, bNibble uint8
			if i%2 == 0 {
				aNibble = a.data[i/2] >> 4
				bNibble = b.data[i/2] >> 4
			} else {
				aNibble = a.data[i/2] & 0x0F
				bNibble = b.data[i/2] & 0x0F
			}
			diff := float32(aNibble) - float32(bNibble)
			sum += diff * diff
		}
	case 2:
		for i := 0; i < q.config.Dimension; i++ {
			shift := uint(6 - (i%4)*2)
			aBits := (a.data[i/4] >> shift) & 0x03
			bBits := (b.data[i/4] >> shift) & 0x03
			diff := float32(aBits) - float32(bBits)
			sum += diff * diff
		}
	case 1:
		// Hamming distance for 1-bit
		for i := 0; i < (q.config.Dimension+7)/8; i++ {
			xor := a.data[i] ^ b.data[i]
			// Count set bits (Hamming weight)
			sum += float32(popcount(xor))
		}
		return sum
	}
	
	return float32(math.Sqrt(float64(sum)))
}

// popcount 计算字节中1的个数
func popcount(x uint8) int {
	count := 0
	for x != 0 {
		count++
		x &= x - 1
	}
	return count
}

// scalarQuantizedVectors 标量量化的向量集合
type scalarQuantizedVectors struct {
	bits      int
	dimension int
	count     int
	data      []byte
	scale     float32
	offset    float32
}

func (v *scalarQuantizedVectors) Type() QuantizationType {
	return QuantizationScalar
}

func (v *scalarQuantizedVectors) Count() int {
	return v.count
}

func (v *scalarQuantizedVectors) Data() []byte {
	return v.data
}

func (v *scalarQuantizedVectors) TotalSize() int {
	return len(v.data) + 8 + 8 // data + scale + offset
}

func (v *scalarQuantizedVectors) Get(index int) (QuantizedVector, error) {
	if index < 0 || index >= v.count {
		return nil, errors.New("quantization: index out of range")
	}
	
	// 计算单个向量的大小
	bytesPerVector := 0
	switch v.bits {
	case 8:
		bytesPerVector = v.dimension
	case 4:
		bytesPerVector = (v.dimension + 1) / 2
	case 2:
		bytesPerVector = (v.dimension + 3) / 4
	case 1:
		bytesPerVector = (v.dimension + 7) / 8
	}
	
	start := index * bytesPerVector
	end := start + bytesPerVector
	
	vecData := make([]byte, bytesPerVector)
	copy(vecData, v.data[start:end])
	
	return &scalarQuantizedVector{
		bits:      v.bits,
		dimension: v.dimension,
		data:      vecData,
		scale:     v.scale,
		offset:    v.offset,
	}, nil
}

// scalarQuantizedVector 标量量化的单个向量
type scalarQuantizedVector struct {
	bits      int
	dimension int
	data      []byte
	scale     float32
	offset    float32
}

func (v *scalarQuantizedVector) Type() QuantizationType {
	return QuantizationScalar
}

func (v *scalarQuantizedVector) Data() []byte {
	return v.data
}

func (v *scalarQuantizedVector) Size() int {
	return len(v.data) + 8 + 8 // data + scale + offset
}

// Serialize 序列化量化参数
func (v *scalarQuantizedVector) Serialize() []byte {
	// Format: [scale(4)][offset(4)][data...]
	buf := make([]byte, 8+len(v.data))
	binary.LittleEndian.PutUint32(buf[0:4], math.Float32bits(v.scale))
	binary.LittleEndian.PutUint32(buf[4:8], math.Float32bits(v.offset))
	copy(buf[8:], v.data)
	return buf
}

// DeserializeScalarQuantizedVector 反序列化
func DeserializeScalarQuantizedVector(data []byte, bits, dimension int) (*scalarQuantizedVector, error) {
	if len(data) < 8 {
		return nil, errors.New("quantization: insufficient data for deserialization")
	}
	
	scale := math.Float32frombits(binary.LittleEndian.Uint32(data[0:4]))
	offset := math.Float32frombits(binary.LittleEndian.Uint32(data[4:8]))
	vecData := make([]byte, len(data)-8)
	copy(vecData, data[8:])
	
	return &scalarQuantizedVector{
		bits:      bits,
		dimension: dimension,
		data:      vecData,
		scale:     scale,
		offset:    offset,
	}, nil
}
