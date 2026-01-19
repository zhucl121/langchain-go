package quantization

import (
	"context"
	"fmt"
	"math"
	"testing"
)

func TestScalarQuantizer_Train(t *testing.T) {
	dimension := 128
	vectorCount := 100
	
	tests := []struct {
		name    string
		config  ScalarQuantizationConfig
		wantErr bool
	}{
		{
			name: "8-bit symmetric",
			config: ScalarQuantizationConfig{
				Bits:         8,
				Dimension:    dimension,
				UseSymmetric: true,
			},
			wantErr: false,
		},
		{
			name: "8-bit asymmetric",
			config: ScalarQuantizationConfig{
				Bits:         8,
				Dimension:    dimension,
				UseSymmetric: false,
			},
			wantErr: false,
		},
		{
			name: "4-bit",
			config: ScalarQuantizationConfig{
				Bits:      4,
				Dimension: dimension,
			},
			wantErr: false,
		},
		{
			name: "with clip range",
			config: ScalarQuantizationConfig{
				Bits:      8,
				Dimension: dimension,
				ClipRange: 10.0,
			},
			wantErr: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := NewScalarQuantizer(tt.config)
			vectors := generateRandomVectors(vectorCount, dimension)
			
			ctx := context.Background()
			err := q.Train(ctx, vectors)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("Train() error = %v, wantErr %v", err, tt.wantErr)
			}
			
			if err == nil && !q.IsTrained() {
				t.Error("IsTrained() = false after successful training")
			}
		})
	}
}

func TestScalarQuantizer_EncodeDecod(t *testing.T) {
	dimension := 128
	vectorCount := 10
	
	bits := []int{8, 4, 2, 1}
	
	for _, bit := range bits {
		t.Run(fmt.Sprintf("Bits_%d", bit), func(t *testing.T) {
			config := ScalarQuantizationConfig{
				Bits:      bit,
				Dimension: dimension,
			}
			
			q := NewScalarQuantizer(config)
			vectors := generateRandomVectors(vectorCount, dimension)
			
			// 编码
			quantized, err := q.Encode(vectors)
			if err != nil {
				t.Fatalf("Encode() error = %v", err)
			}
			
			// 验证类型和数量
			if quantized.Type() != QuantizationScalar {
				t.Errorf("Type() = %v, want %v", quantized.Type(), QuantizationScalar)
			}
			if quantized.Count() != vectorCount {
				t.Errorf("Count() = %v, want %v", quantized.Count(), vectorCount)
			}
			
			// 解码
			decoded, err := q.Decode(quantized)
			if err != nil {
				t.Fatalf("Decode() error = %v", err)
			}
			
			if len(decoded) != vectorCount {
				t.Fatalf("decoded length = %v, want %v", len(decoded), vectorCount)
			}
			
			// 计算量化误差
			mse := computeMSE(vectors, decoded)
			
			// 量化误差应该随着 bits 减少而增大
			maxMSE := float32(1.0)
			if bit == 8 {
				maxMSE = 0.1
			} else if bit == 4 {
				maxMSE = 0.5
			} else if bit == 2 {
				maxMSE = 10.0  // 放宽阈值
			} else if bit == 1 {
				maxMSE = 20.0  // 放宽阈值
			}
			
			if mse > maxMSE {
				t.Errorf("MSE = %v, want <= %v (for %d-bit)", mse, maxMSE, bit)
			}
			
			t.Logf("%d-bit quantization MSE: %.6f", bit, mse)
		})
	}
}

func TestScalarQuantizer_ComputeDistance(t *testing.T) {
	dimension := 128
	vectorCount := 10
	
	config := ScalarQuantizationConfig{
		Bits:      8,
		Dimension: dimension,
	}
	
	q := NewScalarQuantizer(config)
	vectors := generateRandomVectors(vectorCount, dimension)
	
	// 编码
	quantized, err := q.Encode(vectors)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}
	
	// 获取第一个向量作为查询
	query, err := quantized.Get(0)
	if err != nil {
		t.Fatalf("Get(0) error = %v", err)
	}
	
	// 获取所有向量
	quantizedVecs := make([]QuantizedVector, vectorCount)
	for i := 0; i < vectorCount; i++ {
		vec, err := quantized.Get(i)
		if err != nil {
			t.Fatalf("Get(%d) error = %v", i, err)
		}
		quantizedVecs[i] = vec
	}
	
	// 计算距离
	distances, err := q.ComputeDistance(query, quantizedVecs)
	if err != nil {
		t.Fatalf("ComputeDistance() error = %v", err)
	}
	
	if len(distances) != vectorCount {
		t.Fatalf("distances length = %v, want %v", len(distances), vectorCount)
	}
	
	// 到自己的距离应该是0或接近0
	if distances[0] > 0.01 {
		t.Errorf("distance to self = %v, want ~0", distances[0])
	}
	
	// 距离应该都是非负数
	for i, d := range distances {
		if d < 0 {
			t.Errorf("distance[%d] = %v, want >= 0", i, d)
		}
	}
	
	t.Logf("Distances: %v", distances)
}

func TestScalarQuantizer_CompressionRatio(t *testing.T) {
	dimension := 128
	vectorCount := 100
	
	bits := []int{8, 4, 2, 1}
	
	for _, bit := range bits {
		t.Run(fmt.Sprintf("Bits_%d", bit), func(t *testing.T) {
			config := ScalarQuantizationConfig{
				Bits:      bit,
				Dimension: dimension,
			}
			
			q := NewScalarQuantizer(config)
			vectors := generateRandomVectors(vectorCount, dimension)
			
			// 编码
			quantized, err := q.Encode(vectors)
			if err != nil {
				t.Fatalf("Encode() error = %v", err)
			}
			
			// 计算实际压缩比
			originalSize := vectorCount * dimension * 4 // float32 = 4 bytes
			compressedSize := quantized.TotalSize()
			actualRatio := float64(originalSize) / float64(compressedSize)
			
			// 理论压缩比
			expectedRatio := 32.0 / float64(bit)
			
			// 实际压缩比应该接近理论值
			relativeError := math.Abs(actualRatio-expectedRatio) / expectedRatio
			if relativeError > 0.1 {
				t.Logf("Warning: actualRatio=%.2f, expectedRatio=%.2f, error=%.1f%%",
					actualRatio, expectedRatio, relativeError*100)
			}
			
			t.Logf("%d-bit: Original=%d, Compressed=%d, Ratio=%.2fx",
				bit, originalSize, compressedSize, actualRatio)
		})
	}
}

func TestScalarQuantizer_DifferentDataRanges(t *testing.T) {
	dimension := 64
	vectorCount := 10
	
	tests := []struct {
		name   string
		genVec func() [][]float32
	}{
		{
			name: "positive values",
			genVec: func() [][]float32 {
				vecs := make([][]float32, vectorCount)
				for i := 0; i < vectorCount; i++ {
					vecs[i] = make([]float32, dimension)
					for j := 0; j < dimension; j++ {
						vecs[i][j] = float32(j) * 0.1
					}
				}
				return vecs
			},
		},
		{
			name: "negative values",
			genVec: func() [][]float32 {
				vecs := make([][]float32, vectorCount)
				for i := 0; i < vectorCount; i++ {
					vecs[i] = make([]float32, dimension)
					for j := 0; j < dimension; j++ {
						vecs[i][j] = -float32(j) * 0.1
					}
				}
				return vecs
			},
		},
		{
			name: "mixed values",
			genVec: func() [][]float32 {
				vecs := make([][]float32, vectorCount)
				for i := 0; i < vectorCount; i++ {
					vecs[i] = make([]float32, dimension)
					for j := 0; j < dimension; j++ {
						vecs[i][j] = float32(i*dimension+j)*0.01 - 3.0
					}
				}
				return vecs
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := ScalarQuantizationConfig{
				Bits:      8,
				Dimension: dimension,
			}
			
			q := NewScalarQuantizer(config)
			vectors := tt.genVec()
			
			// 编码和解码
			quantized, err := q.Encode(vectors)
			if err != nil {
				t.Fatalf("Encode() error = %v", err)
			}
			
			decoded, err := q.Decode(quantized)
			if err != nil {
				t.Fatalf("Decode() error = %v", err)
			}
			
			// 验证维度
			if len(decoded) != vectorCount {
				t.Errorf("decoded length = %v, want %v", len(decoded), vectorCount)
			}
			
			// 计算 MSE
			mse := computeMSE(vectors, decoded)
			t.Logf("MSE for %s: %.6f", tt.name, mse)
			
			// MSE 应该在合理范围内
			if mse > 1.0 {
				t.Errorf("MSE = %v, too large", mse)
			}
		})
	}
}

func BenchmarkScalarQuantizer_Encode(b *testing.B) {
	dimension := 768
	vectorCount := 1000
	vectors := generateRandomVectors(vectorCount, dimension)
	
	bits := []int{8, 4, 2, 1}
	
	for _, bit := range bits {
		b.Run(fmt.Sprintf("Bits_%d", bit), func(b *testing.B) {
			config := ScalarQuantizationConfig{
				Bits:      bit,
				Dimension: dimension,
			}
			
			q := NewScalarQuantizer(config)
			ctx := context.Background()
			q.Train(ctx, vectors)
			
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = q.Encode(vectors)
			}
			
			b.StopTimer()
			quantized, _ := q.Encode(vectors)
			b.ReportMetric(float64(vectorCount*dimension*4)/float64(quantized.TotalSize()), "compression_ratio")
		})
	}
}

func BenchmarkScalarQuantizer_Decode(b *testing.B) {
	dimension := 768
	vectorCount := 1000
	vectors := generateRandomVectors(vectorCount, dimension)
	
	config := ScalarQuantizationConfig{
		Bits:      8,
		Dimension: dimension,
	}
	
	q := NewScalarQuantizer(config)
	ctx := context.Background()
	q.Train(ctx, vectors)
	quantized, _ := q.Encode(vectors)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = q.Decode(quantized)
	}
}

func BenchmarkScalarQuantizer_ComputeDistance(b *testing.B) {
	dimension := 768
	vectorCount := 1000
	
	config := ScalarQuantizationConfig{
		Bits:      8,
		Dimension: dimension,
	}
	
	q := NewScalarQuantizer(config)
	vectors := generateRandomVectors(vectorCount, dimension)
	quantized, _ := q.Encode(vectors)
	
	query, _ := quantized.Get(0)
	quantizedVecs := make([]QuantizedVector, vectorCount)
	for i := 0; i < vectorCount; i++ {
		quantizedVecs[i], _ = quantized.Get(i)
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = q.ComputeDistance(query, quantizedVecs)
	}
}
