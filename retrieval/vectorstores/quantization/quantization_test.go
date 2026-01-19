package quantization

import (
	"context"
	"fmt"
	"math"
	"testing"
)

// TestConfig 测试配置验证
func TestConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid scalar 8-bit",
			config: Config{
				Type: QuantizationScalar,
				Bits: 8,
			},
			wantErr: false,
		},
		{
			name: "valid scalar 4-bit",
			config: Config{
				Type: QuantizationScalar,
				Bits: 4,
			},
			wantErr: false,
		},
		{
			name: "invalid scalar bits",
			config: Config{
				Type: QuantizationScalar,
				Bits: 3, // 不支持
			},
			wantErr: true,
		},
		{
			name: "valid binary",
			config: Config{
				Type: QuantizationBinary,
			},
			wantErr: false,
		},
		{
			name: "valid product",
			config: Config{
				Type:         QuantizationProduct,
				M:            8,
				NBits:        8,
				TrainingSize: 1000,
			},
			wantErr: false,
		},
		{
			name: "invalid product - negative M",
			config: Config{
				Type:         QuantizationProduct,
				M:            -1,
				NBits:        8,
				TrainingSize: 1000,
			},
			wantErr: true,
		},
		{
			name: "invalid product - NBits too large",
			config: Config{
				Type:         QuantizationProduct,
				M:            8,
				NBits:        17, // 最大16
				TrainingSize: 1000,
			},
			wantErr: true,
		},
		{
			name: "invalid product - insufficient training size",
			config: Config{
				Type:         QuantizationProduct,
				M:            8,
				NBits:        8,
				TrainingSize: 50, // 最小100
			},
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestNewQuantizer 测试量化器创建
func TestNewQuantizer(t *testing.T) {
	tests := []struct {
		name      string
		config    Config
		dimension int
		wantType  QuantizationType
		wantErr   bool
	}{
		{
			name: "scalar quantizer",
			config: Config{
				Type: QuantizationScalar,
				Bits: 8,
			},
			dimension: 128,
			wantType:  QuantizationScalar,
			wantErr:   false,
		},
		{
			name: "binary quantizer",
			config: Config{
				Type: QuantizationBinary,
			},
			dimension: 128,
			wantType:  QuantizationBinary,
			wantErr:   false,
		},
		{
			name: "product quantizer",
			config: Config{
				Type:         QuantizationProduct,
				M:            8,
				NBits:        8,
				TrainingSize: 1000,
			},
			dimension: 128, // 128 % 8 == 0
			wantType:  QuantizationProduct,
			wantErr:   false,
		},
		{
			name: "product quantizer - invalid dimension",
			config: Config{
				Type:         QuantizationProduct,
				M:            8,
				NBits:        8,
				TrainingSize: 1000,
			},
			dimension: 127, // 127 % 8 != 0
			wantErr:   true,
		},
		{
			name:      "invalid dimension",
			config:    Config{Type: QuantizationNone},
			dimension: -1,
			wantErr:   true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q, err := NewQuantizer(tt.config, tt.dimension)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewQuantizer() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil {
				if q.Type() != tt.wantType {
					t.Errorf("NewQuantizer() type = %v, want %v", q.Type(), tt.wantType)
				}
				if q.Dimension() != tt.dimension {
					t.Errorf("NewQuantizer() dimension = %v, want %v", q.Dimension(), tt.dimension)
				}
			}
		})
	}
}

// TestNoOpQuantizer 测试无量化器
func TestNoOpQuantizer(t *testing.T) {
	dimension := 128
	q := NewNoOpQuantizer(dimension)
	
	// 测试基本属性
	if q.Type() != QuantizationNone {
		t.Errorf("Type() = %v, want %v", q.Type(), QuantizationNone)
	}
	if q.Dimension() != dimension {
		t.Errorf("Dimension() = %v, want %v", q.Dimension(), dimension)
	}
	if !q.IsTrained() {
		t.Error("IsTrained() = false, want true")
	}
	if q.CompressionRatio() != 1.0 {
		t.Errorf("CompressionRatio() = %v, want 1.0", q.CompressionRatio())
	}
	
	// 测试编码/解码
	vectors := generateRandomVectors(10, dimension)
	
	quantized, err := q.Encode(vectors)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}
	
	if quantized.Type() != QuantizationNone {
		t.Errorf("quantized.Type() = %v, want %v", quantized.Type(), QuantizationNone)
	}
	if quantized.Count() != len(vectors) {
		t.Errorf("quantized.Count() = %v, want %v", quantized.Count(), len(vectors))
	}
	
	decoded, err := q.Decode(quantized)
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	
	// 验证解码后与原始向量相同
	if len(decoded) != len(vectors) {
		t.Fatalf("decoded length = %v, want %v", len(decoded), len(vectors))
	}
	
	for i := range vectors {
		for j := range vectors[i] {
			if decoded[i][j] != vectors[i][j] {
				t.Errorf("decoded[%d][%d] = %v, want %v", i, j, decoded[i][j], vectors[i][j])
			}
		}
	}
}

// TestCompressionRatio 测试压缩比计算
func TestCompressionRatio(t *testing.T) {
	dimension := 128
	
	tests := []struct {
		name     string
		config   Config
		expected float64
	}{
		{
			name: "scalar 8-bit",
			config: Config{
				Type: QuantizationScalar,
				Bits: 8,
			},
			expected: 4.0, // 32 / 8
		},
		{
			name: "scalar 4-bit",
			config: Config{
				Type: QuantizationScalar,
				Bits: 4,
			},
			expected: 8.0, // 32 / 4
		},
		{
			name: "scalar 2-bit",
			config: Config{
				Type: QuantizationScalar,
				Bits: 2,
			},
			expected: 16.0, // 32 / 2
		},
		{
			name: "scalar 1-bit",
			config: Config{
				Type: QuantizationScalar,
				Bits: 1,
			},
			expected: 32.0, // 32 / 1
		},
		{
			name: "binary",
			config: Config{
				Type: QuantizationBinary,
			},
			expected: 32.0, // 32 / 1
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q, err := NewQuantizer(tt.config, dimension)
			if err != nil {
				t.Fatalf("NewQuantizer() error = %v", err)
			}
			
			ratio := q.CompressionRatio()
			if math.Abs(ratio-tt.expected) > 0.01 {
				t.Errorf("CompressionRatio() = %v, want %v", ratio, tt.expected)
			}
		})
	}
}

// generateRandomVectors 生成随机向量
func generateRandomVectors(count, dimension int) [][]float32 {
	vectors := make([][]float32, count)
	for i := 0; i < count; i++ {
		vectors[i] = make([]float32, dimension)
		for j := 0; j < dimension; j++ {
			vectors[i][j] = float32(i*dimension+j) * 0.01 // 简单的确定性值
		}
	}
	return vectors
}

// generateNormalizedVectors 生成归一化向量
func generateNormalizedVectors(count, dimension int) [][]float32 {
	vectors := generateRandomVectors(count, dimension)
	
	// 归一化
	for i := range vectors {
		norm := float32(0)
		for j := range vectors[i] {
			norm += vectors[i][j] * vectors[i][j]
		}
		norm = float32(math.Sqrt(float64(norm)))
		if norm > 0 {
			for j := range vectors[i] {
				vectors[i][j] /= norm
			}
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
			return float32(math.MaxFloat32)
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

// TestStatistics 测试统计信息
func TestStatistics(t *testing.T) {
	dimension := 128
	vectorCount := 100
	vectors := generateRandomVectors(vectorCount, dimension)
	
	config := Config{
		Type: QuantizationScalar,
		Bits: 8,
	}
	
	q, err := NewQuantizer(config, dimension)
	if err != nil {
		t.Fatalf("NewQuantizer() error = %v", err)
	}
	
	// 训练
	ctx := context.Background()
	if err := q.Train(ctx, vectors); err != nil {
		t.Fatalf("Train() error = %v", err)
	}
	
	// 编码
	quantized, err := q.Encode(vectors)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}
	
	// 验证统计信息
	originalSize := vectorCount * dimension * 4 // float32 = 4 bytes
	compressedSize := quantized.TotalSize()
	
	if compressedSize >= originalSize {
		t.Errorf("compressedSize (%d) >= originalSize (%d)", compressedSize, originalSize)
	}
	
	ratio := float64(originalSize) / float64(compressedSize)
	expectedRatio := q.CompressionRatio()
	
	// 允许一定误差（因为有元数据）
	if math.Abs(ratio-expectedRatio)/expectedRatio > 0.2 {
		t.Logf("Warning: ratio = %v, expected = %v (difference > 20%%)", ratio, expectedRatio)
	}
	
	t.Logf("Original size: %d bytes", originalSize)
	t.Logf("Compressed size: %d bytes", compressedSize)
	t.Logf("Compression ratio: %.2fx", ratio)
}

// BenchmarkScalarQuantization 基准测试标量量化
func BenchmarkScalarQuantization(b *testing.B) {
	dimension := 768
	vectorCount := 1000
	vectors := generateRandomVectors(vectorCount, dimension)
	
	bits := []int{8, 4, 2, 1}
	
	for _, bit := range bits {
		b.Run(fmt.Sprintf("Bits_%d", bit), func(b *testing.B) {
			config := Config{
				Type: QuantizationScalar,
				Bits: bit,
			}
			
			q, _ := NewQuantizer(config, dimension)
			ctx := context.Background()
			q.Train(ctx, vectors)
			
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = q.Encode(vectors)
			}
		})
	}
}

// BenchmarkBinaryQuantization 基准测试二值量化
func BenchmarkBinaryQuantization(b *testing.B) {
	dimension := 768
	vectorCount := 1000
	vectors := generateRandomVectors(vectorCount, dimension)
	
	config := Config{
		Type: QuantizationBinary,
	}
	
	q, _ := NewQuantizer(config, dimension)
	ctx := context.Background()
	q.Train(ctx, vectors)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = q.Encode(vectors)
	}
}
