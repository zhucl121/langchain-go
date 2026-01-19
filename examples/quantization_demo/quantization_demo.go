// Package main 演示向量量化功能的使用
//
// 此示例展示了如何使用不同的量化方法来压缩向量，
// 以及它们之间的性能和精度权衡。
package main

import (
	"context"
	"fmt"
	"math"
	"time"
	
	"github.com/zhucl121/langchain-go/retrieval/vectorstores/quantization"
)

func main() {
	fmt.Println("=== 向量量化示例 ===\n")
	
	// 生成测试向量
	dimension := 768
	vectorCount := 1000
	vectors := generateTestVectors(vectorCount, dimension)
	
	fmt.Printf("测试数据: %d 个向量，每个 %d 维度\n", vectorCount, dimension)
	fmt.Printf("原始大小: %d bytes (%.2f MB)\n\n",
		vectorCount*dimension*4, float64(vectorCount*dimension*4)/(1024*1024))
	
	// 1. Scalar Quantization (8-bit, 4-bit, 2-bit)
	testScalarQuantization(vectors, dimension)
	
	// 2. Binary Quantization
	testBinaryQuantization(vectors, dimension)
	
	// 3. Product Quantization
	testProductQuantization(vectors, dimension)
	
	// 4. 比较总结
	printComparison()
}

// generateTestVectors 生成测试向量
func generateTestVectors(count, dimension int) [][]float32 {
	vectors := make([][]float32, count)
	for i := 0; i < count; i++ {
		vectors[i] = make([]float32, dimension)
		for j := 0; j < dimension; j++ {
			// 生成归一化的随机向量
			vectors[i][j] = float32(math.Sin(float64(i*dimension+j) * 0.01))
		}
	}
	return vectors
}

// testScalarQuantization 测试标量量化
func testScalarQuantization(vectors [][]float32, dimension int) {
	fmt.Println("━━━ 1. Scalar Quantization ━━━")
	
	bits := []int{8, 4, 2, 1}
	
	for _, bit := range bits {
		fmt.Printf("\n▸ %d-bit Scalar Quantization:\n", bit)
		
		config := quantization.Config{
			Type: quantization.QuantizationScalar,
			Bits: bit,
		}
		
		q, _ := quantization.NewQuantizer(config, dimension)
		
		// 训练
		ctx := context.Background()
		startTrain := time.Now()
		q.Train(ctx, vectors[:100]) // 用前100个向量训练
		trainTime := time.Since(startTrain)
		
		// 编码
		startEncode := time.Now()
		quantized, _ := q.Encode(vectors)
		encodeTime := time.Since(startEncode)
		
		// 解码
		startDecode := time.Now()
		decoded, _ := q.Decode(quantized)
		decodeTime := time.Since(startDecode)
		
		// 统计
		originalSize := len(vectors) * dimension * 4
		compressedSize := quantized.TotalSize()
		ratio := float64(originalSize) / float64(compressedSize)
		mse := computeMSE(vectors, decoded)
		
		fmt.Printf("  压缩比: %.2fx\n", ratio)
		fmt.Printf("  压缩后大小: %.2f MB\n", float64(compressedSize)/(1024*1024))
		fmt.Printf("  量化误差 (MSE): %.6f\n", mse)
		fmt.Printf("  训练时间: %v\n", trainTime)
		fmt.Printf("  编码时间: %v (%.2f μs/vec)\n",
			encodeTime, float64(encodeTime.Microseconds())/float64(len(vectors)))
		fmt.Printf("  解码时间: %v (%.2f μs/vec)\n",
			decodeTime, float64(decodeTime.Microseconds())/float64(len(vectors)))
	}
	
	fmt.Println()
}

// testBinaryQuantization 测试二值量化
func testBinaryQuantization(vectors [][]float32, dimension int) {
	fmt.Println("━━━ 2. Binary Quantization ━━━\n")
	
	config := quantization.Config{
		Type: quantization.QuantizationBinary,
	}
	
	q, _ := quantization.NewQuantizer(config, dimension)
	
	// 训练
	ctx := context.Background()
	startTrain := time.Now()
	q.Train(ctx, vectors[:100])
	trainTime := time.Since(startTrain)
	
	// 编码
	startEncode := time.Now()
	quantized, _ := q.Encode(vectors)
	encodeTime := time.Since(startEncode)
	
	// 解码
	startDecode := time.Now()
	decoded, _ := q.Decode(quantized)
	decodeTime := time.Since(startDecode)
	
	// Hamming 距离计算
	query, _ := quantized.Get(0)
	quantizedVecs := make([]quantization.QuantizedVector, len(vectors))
	for i := range vectors {
		quantizedVecs[i], _ = quantized.Get(i)
	}
	
	startDistance := time.Now()
	_, _ = q.ComputeDistance(query, quantizedVecs)
	distanceTime := time.Since(startDistance)
	
	// 统计
	originalSize := len(vectors) * dimension * 4
	compressedSize := quantized.TotalSize()
	ratio := float64(originalSize) / float64(compressedSize)
	_ = decoded // 标记使用（在二值量化中我们不计算 MSE，因为总是很高）
	
	fmt.Printf("  压缩比: %.2fx (最大可能)\n", ratio)
	fmt.Printf("  压缩后大小: %.2f MB\n", float64(compressedSize)/(1024*1024))
	fmt.Printf("  训练时间: %v\n", trainTime)
	fmt.Printf("  编码时间: %v (%.2f μs/vec)\n",
		encodeTime, float64(encodeTime.Microseconds())/float64(len(vectors)))
	fmt.Printf("  解码时间: %v (%.2f μs/vec)\n",
		decodeTime, float64(decodeTime.Microseconds())/float64(len(vectors)))
	fmt.Printf("  Hamming 距离计算: %v (%.2f μs/query)\n",
		distanceTime, float64(distanceTime.Microseconds()))
	fmt.Println("  特点: 极致压缩，超快的 Hamming 距离计算")
	fmt.Println()
}

// testProductQuantization 测试乘积量化
func testProductQuantization(vectors [][]float32, dimension int) {
	fmt.Println("━━━ 3. Product Quantization ━━━\n")
	
	// 使用较小的参数以加快演示速度
	m := 8
	nBits := 4 // 使用 4-bit，码本大小为 16，更容易训练
	trainingSize := 500 // 增加训练样本数
	
	if dimension%m != 0 {
		fmt.Printf("  ⚠️  跳过: 维度 %d 不能被 M=%d 整除\n\n", dimension, m)
		return
	}
	
	// 确保有足够的训练数据
	if len(vectors) < trainingSize {
		trainingSize = len(vectors)
	}
	
	fmt.Printf("  配置: M=%d 个子向量, %d-bit 编码，训练样本=%d\n", m, nBits, trainingSize)
	
	config := quantization.Config{
		Type:         quantization.QuantizationProduct,
		M:            m,
		NBits:        nBits,
		TrainingSize: trainingSize,
	}
	
	q, _ := quantization.NewQuantizer(config, dimension)
	
	// 训练 (最耗时的部分)
	fmt.Print("  训练中...")
	ctx := context.Background()
	startTrain := time.Now()
	err := q.Train(ctx, vectors[:trainingSize])
	trainTime := time.Since(startTrain)
	if err != nil {
		fmt.Printf(" 失败: %v\n\n", err)
		return
	}
	fmt.Printf(" 完成 (%v)\n", trainTime)
	
	// 编码
	startEncode := time.Now()
	quantized, _ := q.Encode(vectors)
	encodeTime := time.Since(startEncode)
	
	// 解码
	startDecode := time.Now()
	decoded, _ := q.Decode(quantized)
	decodeTime := time.Since(startDecode)
	
	// 距离计算（使用查表法优化）
	query, _ := quantized.Get(0)
	quantizedVecs := make([]quantization.QuantizedVector, len(vectors))
	for i := range vectors {
		quantizedVecs[i], _ = quantized.Get(i)
	}
	
	startDistance := time.Now()
	_, _ = q.ComputeDistance(query, quantizedVecs)
	distanceTime := time.Since(startDistance)
	
	// 统计
	originalSize := len(vectors) * dimension * 4
	compressedSize := quantized.TotalSize()
	ratio := float64(originalSize) / float64(compressedSize)
	mse := computeMSE(vectors, decoded)
	
	fmt.Printf("  压缩比: %.2fx\n", ratio)
	fmt.Printf("  压缩后大小: %.2f MB\n", float64(compressedSize)/(1024*1024))
	fmt.Printf("  量化误差 (MSE): %.6f\n", mse)
	fmt.Printf("  训练时间: %v\n", trainTime)
	fmt.Printf("  编码时间: %v (%.2f μs/vec)\n",
		encodeTime, float64(encodeTime.Microseconds())/float64(len(vectors)))
	fmt.Printf("  解码时间: %v (%.2f μs/vec)\n",
		decodeTime, float64(decodeTime.Microseconds())/float64(len(vectors)))
	fmt.Printf("  距离计算 (ADC): %v (%.2f μs/query)\n",
		distanceTime, float64(distanceTime.Microseconds()))
	fmt.Println("  特点: 高压缩比 + 低精度损失 + 快速 ADC 查询")
	fmt.Println()
}

// printComparison 打印比较总结
func printComparison() {
	fmt.Println("━━━ 比较总结 ━━━\n")
	
	fmt.Println("┌─────────────────────┬──────────┬──────────┬────────────┬──────────┐")
	fmt.Println("│ 方法                │ 压缩比   │ 精度损失 │ 训练成本   │ 查询速度 │")
	fmt.Println("├─────────────────────┼──────────┼──────────┼────────────┼──────────┤")
	fmt.Println("│ Scalar 8-bit        │ 4x       │ 很低     │ 极低       │ 快       │")
	fmt.Println("│ Scalar 4-bit        │ 8x       │ 低       │ 极低       │ 快       │")
	fmt.Println("│ Scalar 2-bit        │ 16x      │ 中       │ 极低       │ 快       │")
	fmt.Println("│ Scalar 1-bit        │ 32x      │ 高       │ 极低       │ 很快     │")
	fmt.Println("│ Binary              │ 32x      │ 高       │ 低         │ 极快     │")
	fmt.Println("│ Product (M=8, 8bit) │ 4-8x     │ 低       │ 高         │ 很快*    │")
	fmt.Println("└─────────────────────┴──────────┴──────────┴────────────┴──────────┘")
	fmt.Println("\n* Product Quantization 使用 ADC (Asymmetric Distance Computation) 查表法优化")
	
	fmt.Println("\n使用建议:")
	fmt.Println("  • 通用场景: Scalar 8-bit (平衡压缩比和精度)")
	fmt.Println("  • 极致压缩: Binary Quantization (32x 压缩)")
	fmt.Println("  • 高精度需求: Product Quantization (最佳精度)")
	fmt.Println("  • 实时检索: Binary or Product Quantization (极快查询)")
	fmt.Println("  • 内存受限: 根据预算选择合适的 bits")
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
