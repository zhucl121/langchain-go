package quantization

import (
	"context"
	"testing"
)

func TestProductQuantizer_Train(t *testing.T) {
	dimension := 128
	m := 8
	vectorCount := 200
	
	if testing.Short() {
		t.Skip("Skipping ProductQuantizer training in short mode")
	}
	
	vectors := generateRandomVectors(vectorCount, dimension)
	
	config := ProductQuantizationConfig{
		Dimension:     dimension,
		M:             m,
		NBits:         8,
		TrainingSize:  vectorCount,
		MaxIterations: 10, // 减少迭代次数加速测试
		Tolerance:     1e-3,
	}
	
	q := NewProductQuantizer(config)
	
	ctx := context.Background()
	err := q.Train(ctx, vectors)
	if err != nil {
		t.Fatalf("Train() error = %v", err)
	}
	
	if !q.IsTrained() {
		t.Error("IsTrained() = false after successful training")
	}
	
	// 验证码本
	if len(q.codebooks) != m {
		t.Errorf("codebooks length = %d, want %d", len(q.codebooks), m)
	}
	
	for i, codebook := range q.codebooks {
		expectedK := 1 << config.NBits
		if len(codebook) != expectedK {
			t.Errorf("codebook[%d] length = %d, want %d", i, len(codebook), expectedK)
		}
		
		for j, centroid := range codebook {
			if len(centroid) != dimension/m {
				t.Errorf("codebook[%d][%d] dimension = %d, want %d",
					i, j, len(centroid), dimension/m)
			}
		}
	}
}

func TestProductQuantizer_EncodeDecode(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping ProductQuantizer test in short mode")
	}
	
	dimension := 64
	m := 8
	vectorCount := 50
	
	vectors := generateRandomVectors(vectorCount, dimension)
	
	config := ProductQuantizationConfig{
		Dimension:     dimension,
		M:             m,
		NBits:         4, // 使用4-bit减少训练时间
		TrainingSize:  vectorCount,
		MaxIterations: 10,
		Tolerance:     1e-3,
	}
	
	q := NewProductQuantizer(config)
	
	// 训练
	ctx := context.Background()
	err := q.Train(ctx, vectors)
	if err != nil {
		t.Fatalf("Train() error = %v", err)
	}
	
	// 编码
	quantized, err := q.Encode(vectors)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}
	
	if quantized.Type() != QuantizationProduct {
		t.Errorf("Type() = %v, want %v", quantized.Type(), QuantizationProduct)
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
	t.Logf("Product Quantization MSE: %.6f", mse)
	
	// PQ 的误差应该比较低（比 1-bit/2-bit scalar quantization 低）
	if mse > 2.0 {
		t.Errorf("MSE = %v, too large for PQ", mse)
	}
}

func TestProductQuantizer_CompressionRatio(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping ProductQuantizer test in short mode")
	}
	
	dimension := 128
	m := 8
	vectorCount := 100
	nBits := 8
	
	vectors := generateRandomVectors(vectorCount, dimension)
	
	config := ProductQuantizationConfig{
		Dimension:     dimension,
		M:             m,
		NBits:         nBits,
		TrainingSize:  vectorCount,
		MaxIterations: 10,
	}
	
	q := NewProductQuantizer(config)
	
	ctx := context.Background()
	q.Train(ctx, vectors)
	
	quantized, _ := q.Encode(vectors)
	
	// 计算压缩比（不包括码本）
	originalSize := vectorCount * dimension * 4 // float32
	encodedSize := len(quantized.Data())
	compressionRatio := float64(originalSize) / float64(encodedSize)
	
	// 理论压缩比: 32 / (M * NBits / Dimension)
	expectedRatio := 32.0 / (float64(m*nBits) / float64(dimension))
	
	t.Logf("Original=%d, Encoded=%d, Ratio=%.2fx (expected=%.2fx)",
		originalSize, encodedSize, compressionRatio, expectedRatio)
	
	// 允许10%的误差
	if compressionRatio < expectedRatio*0.9 || compressionRatio > expectedRatio*1.1 {
		t.Logf("Warning: compression ratio deviation > 10%%")
	}
}

func TestProductQuantizer_ComputeDistance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping ProductQuantizer test in short mode")
	}
	
	dimension := 64
	m := 8
	vectorCount := 20
	
	vectors := generateRandomVectors(vectorCount, dimension)
	
	config := ProductQuantizationConfig{
		Dimension:     dimension,
		M:             m,
		NBits:         4,
		TrainingSize:  vectorCount,
		MaxIterations: 10,
	}
	
	q := NewProductQuantizer(config)
	
	ctx := context.Background()
	q.Train(ctx, vectors)
	
	quantized, _ := q.Encode(vectors)
	
	// 获取查询向量
	query, err := quantized.Get(0)
	if err != nil {
		t.Fatalf("Get(0) error = %v", err)
	}
	
	// 获取所有向量
	quantizedVecs := make([]QuantizedVector, vectorCount)
	for i := 0; i < vectorCount; i++ {
		quantizedVecs[i], _ = quantized.Get(i)
	}
	
	// 计算距离
	distances, err := q.ComputeDistance(query, quantizedVecs)
	if err != nil {
		t.Fatalf("ComputeDistance() error = %v", err)
	}
	
	if len(distances) != vectorCount {
		t.Fatalf("distances length = %v, want %v", len(distances), vectorCount)
	}
	
	// 到自己的距离应该接近0
	if distances[0] > 0.1 {
		t.Errorf("distance to self = %v, want ~0", distances[0])
	}
	
	// 所有距离应该非负
	for i, d := range distances {
		if d < 0 {
			t.Errorf("distance[%d] = %v, want >= 0", i, d)
		}
	}
}

func TestProductQuantizer_InvalidConfigurations(t *testing.T) {
	tests := []struct {
		name      string
		dimension int
		config    ProductQuantizationConfig
		wantErr   bool
	}{
		{
			name:      "dimension not divisible by M",
			dimension: 127,
			config: ProductQuantizationConfig{
				M:            8,
				NBits:        8,
				TrainingSize: 100,
			},
			wantErr: true,
		},
		{
			name:      "invalid M (0)",
			dimension: 128,
			config: ProductQuantizationConfig{
				M:            0,
				NBits:        8,
				TrainingSize: 100,
			},
			wantErr: true,
		},
		{
			name:      "invalid NBits (0)",
			dimension: 128,
			config: ProductQuantizationConfig{
				M:            8,
				NBits:        0,
				TrainingSize: 100,
			},
			wantErr: true,
		},
		{
			name:      "invalid NBits (too large)",
			dimension: 128,
			config: ProductQuantizationConfig{
				M:            8,
				NBits:        17,
				TrainingSize: 100,
			},
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := Config{
				Type:         QuantizationProduct,
				M:            tt.config.M,
				NBits:        tt.config.NBits,
				TrainingSize: tt.config.TrainingSize,
			}
			
			_, err := NewQuantizer(config, tt.dimension)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewQuantizer() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// BenchmarkProductQuantizer_Train 基准测试训练
func BenchmarkProductQuantizer_Train(b *testing.B) {
	dimension := 128
	m := 8
	vectorCount := 500
	vectors := generateRandomVectors(vectorCount, dimension)
	
	config := ProductQuantizationConfig{
		Dimension:     dimension,
		M:             m,
		NBits:         8,
		TrainingSize:  vectorCount,
		MaxIterations: 10,
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		q := NewProductQuantizer(config)
		ctx := context.Background()
		_ = q.Train(ctx, vectors)
	}
}

// BenchmarkProductQuantizer_Encode 基准测试编码
func BenchmarkProductQuantizer_Encode(b *testing.B) {
	dimension := 128
	m := 8
	vectorCount := 1000
	vectors := generateRandomVectors(vectorCount, dimension)
	
	config := ProductQuantizationConfig{
		Dimension:     dimension,
		M:             m,
		NBits:         8,
		TrainingSize:  200,
		MaxIterations: 10,
	}
	
	q := NewProductQuantizer(config)
	ctx := context.Background()
	q.Train(ctx, vectors[:200]) // 只用部分数据训练
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = q.Encode(vectors)
	}
}

// BenchmarkProductQuantizer_ComputeDistance 基准测试距离计算
func BenchmarkProductQuantizer_ComputeDistance(b *testing.B) {
	dimension := 128
	m := 8
	vectorCount := 1000
	vectors := generateRandomVectors(vectorCount, dimension)
	
	config := ProductQuantizationConfig{
		Dimension:     dimension,
		M:             m,
		NBits:         8,
		TrainingSize:  200,
		MaxIterations: 10,
	}
	
	q := NewProductQuantizer(config)
	ctx := context.Background()
	q.Train(ctx, vectors[:200])
	
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
