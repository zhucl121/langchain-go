# 向量量化技术文档

**版本**: v0.2.0  
**创建日期**: 2026-01-20  
**作者**: LangChain-Go Team

---

## 📋 目录

- [概述](#概述)
- [为什么需要向量量化](#为什么需要向量量化)
- [支持的量化方法](#支持的量化方法)
- [快速开始](#快速开始)
- [详细用法](#详细用法)
- [性能对比](#性能对比)
- [最佳实践](#最佳实践)
- [API 参考](#api-参考)

---

## 概述

向量量化是一种压缩技术，将高精度浮点向量转换为低精度表示，以降低内存占用和提高检索速度。

### 主要优势

- **节省内存**: 压缩比 4x ~ 60x
- **加速检索**: 距离计算速度提升 2-10x
- **可控精度**: 根据需求选择合适的量化方法
- **易于集成**: 统一的 API 接口

---

## 为什么需要向量量化

### 场景 1: 大规模向量存储

**问题**: 存储 1亿 个 768 维的 float32 向量需要 ~288 GB 内存

```
1亿 × 768 × 4字节 = 307,200,000,000 字节 ≈ 288 GB
```

**解决方案**: 使用 8-bit 标量量化

```
1亿 × 768 × 1字节 = 76,800,000,000 字节 ≈ 72 GB  (节省 75%)
```

### 场景 2: 实时检索

**问题**: 浮点距离计算耗时长，影响 QPS

**解决方案**: 使用 Binary Quantization + Hamming 距离

- 距离计算速度提升 10x+
- 支持位运算优化

### 场景 3: 边缘设备部署

**问题**: 移动设备内存有限 (< 4GB)

**解决方案**: 使用 Product Quantization

- 高压缩比（10-60x）
- 低精度损失（MSE < 0.01）

---

## 支持的量化方法

### 1. Scalar Quantization (标量量化)

将每个浮点数独立量化为低位整数。

#### 特点

- ✅ 实现简单，训练快速
- ✅ 支持 8-bit, 4-bit, 2-bit, 1-bit
- ✅ 压缩比: 4x ~ 32x
- ⚠️ 低 bit 时精度损失较大

#### 使用场景

- 通用场景的首选方案
- 需要快速部署
- 对精度要求不严格

---

### 2. Binary Quantization (二值量化)

将每个浮点数量化为 0 或 1。

#### 特点

- ✅ 最大压缩比 (32x)
- ✅ 极快的 Hamming 距离计算
- ✅ 支持位运算优化
- ⚠️ 精度损失最大

#### 使用场景

- 极致内存优化
- 实时检索场景
- 粗排阶段的快速过滤

---

### 3. Product Quantization (乘积量化)

将向量分割为子向量，分别量化。

#### 特点

- ✅ 高压缩比 + 低精度损失
- ✅ 查表法优化距离计算
- ✅ 可调节 M 和 NBits
- ⚠️ 训练成本高

#### 使用场景

- 高精度要求
- 大规模向量库
- 生产环境

---

## 快速开始

### 安装

```bash
go get github.com/zhucl121/langchain-go
```

### 示例 1: Scalar Quantization (8-bit)

```go
package main

import (
    "context"
    "fmt"
    "github.com/zhucl121/langchain-go/retrieval/vectorstores/quantization"
)

func main() {
    // 1. 创建量化器
    config := quantization.Config{
        Type: quantization.QuantizationScalar,
        Bits: 8,
    }
    
    dimension := 768
    q, err := quantization.NewQuantizer(config, dimension)
    if err != nil {
        panic(err)
    }
    
    // 2. 准备向量数据
    vectors := loadVectors() // [][]float32
    
    // 3. 训练量化器（可选，Scalar Quantization 可自动训练）
    ctx := context.Background()
    if err := q.Train(ctx, vectors[:100]); err != nil {
        panic(err)
    }
    
    // 4. 编码向量
    quantized, err := q.Encode(vectors)
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("压缩比: %.2fx\n", q.CompressionRatio())
    fmt.Printf("原始大小: %d bytes\n", len(vectors)*dimension*4)
    fmt.Printf("压缩后: %d bytes\n", quantized.TotalSize())
    
    // 5. 解码（如果需要）
    decoded, err := q.Decode(quantized)
    if err != nil {
        panic(err)
    }
    
    _ = decoded
}
```

### 示例 2: Binary Quantization

```go
// 1. 创建 Binary 量化器
config := quantization.Config{
    Type: quantization.QuantizationBinary,
}

q, _ := quantization.NewQuantizer(config, dimension)

// 2. 训练 + 编码
ctx := context.Background()
q.Train(ctx, vectors[:100])
quantized, _ := q.Encode(vectors)

// 3. 计算 Hamming 距离
query, _ := quantized.Get(0)
candidates := []quantization.QuantizedVector{...}

distances, _ := q.ComputeDistance(query, candidates)
fmt.Printf("Hamming 距离: %v\n", distances)
```

### 示例 3: Product Quantization

```go
// 1. 创建 PQ 量化器
config := quantization.Config{
    Type:         quantization.QuantizationProduct,
    M:            8,          // 分割为 8 个子向量
    NBits:        8,          // 每个子向量 8-bit 编码
    TrainingSize: 1000,       // 训练样本数
}

q, _ := quantization.NewQuantizer(config, 768) // dimension 必须能被 M 整除

// 2. 训练（必须）
ctx := context.Background()
if err := q.Train(ctx, vectors); err != nil {
    panic(err)
}

// 3. 编码
quantized, _ := q.Encode(vectors)

// 4. 快速距离计算（ADC）
query, _ := quantized.Get(0)
distances, _ := q.ComputeDistance(query, candidates)
```

---

## 详细用法

### 配置选项

```go
type Config struct {
    // 量化类型
    Type QuantizationType
    
    // Scalar Quantization 参数
    Bits int  // 1, 2, 4, 8
    
    // Product Quantization 参数
    M            int  // 子向量数量
    NBits        int  // 每个子向量的编码位数 (1-16)
    TrainingSize int  // 训练样本数量
}
```

### 创建量化器

```go
// 方式 1: 使用 Config
config := quantization.Config{
    Type: quantization.QuantizationScalar,
    Bits: 8,
}
q, err := quantization.NewQuantizer(config, dimension)

// 方式 2: 直接创建特定类型
scalarQ := quantization.NewScalarQuantizer(quantization.ScalarQuantizationConfig{
    Bits:      8,
    Dimension: 768,
    UseSymmetric: true,
})
```

### 训练量化器

```go
// Scalar Quantization: 可选（自动从编码数据中学习）
ctx := context.Background()
if err := q.Train(ctx, vectors); err != nil {
    // 处理错误
}

// Binary Quantization: 可选（学习阈值）
q.Train(ctx, vectors)

// Product Quantization: 必须（K-means 聚类）
if err := q.Train(ctx, vectors); err != nil {
    panic(err) // 训练失败
}
```

### 编码和解码

```go
// 编码
quantized, err := q.Encode(vectors)
if err != nil {
    return err
}

// 获取单个量化向量
vec, err := quantized.Get(0)
if err != nil {
    return err
}

// 解码（近似重构）
decoded, err := q.Decode(quantized)
if err != nil {
    return err
}

// 计算重构误差
mse := computeMSE(vectors, decoded)
fmt.Printf("MSE: %.6f\n", mse)
```

### 距离计算

```go
// 获取查询向量和候选向量
query, _ := quantized.Get(0)

candidates := make([]quantization.QuantizedVector, 100)
for i := 0; i < 100; i++ {
    candidates[i], _ = quantized.Get(i)
}

// 计算距离
distances, err := q.ComputeDistance(query, candidates)
if err != nil {
    return err
}

// 找到最近的 K 个向量
k := 10
topK := findTopK(distances, k)
```

---

## 性能对比

### 测试环境

- 硬件: M1 Mac (8核 CPU)
- 向量: 1000 个 768 维 float32
- 原始大小: 3 MB

### 压缩比

| 方法              | 压缩比   | 压缩后大小 |
|-------------------|----------|------------|
| Scalar 8-bit      | 4.00x    | 0.73 MB    |
| Scalar 4-bit      | 8.00x    | 0.37 MB    |
| Scalar 2-bit      | 16.00x   | 0.18 MB    |
| Scalar 1-bit      | 32.00x   | 0.09 MB    |
| Binary            | 32.00x   | 0.09 MB    |
| Product (M=8, 4b) | 57.80x   | 0.05 MB    |

### 精度损失 (MSE)

| 方法              | MSE       | 精度等级 |
|-------------------|-----------|----------|
| Scalar 8-bit      | 0.000021  | 极低 ⭐⭐⭐⭐⭐ |
| Scalar 4-bit      | 0.006120  | 低 ⭐⭐⭐⭐ |
| Scalar 2-bit      | 0.159050  | 中 ⭐⭐⭐ |
| Scalar 1-bit      | 0.226636  | 高 ⭐⭐ |
| Binary            | ~0.5      | 高 ⭐⭐ |
| Product (M=8, 4b) | 0.011436  | 低 ⭐⭐⭐⭐ |

### 速度性能

| 操作             | Scalar 8-bit | Binary      | Product     |
|------------------|--------------|-------------|-------------|
| 训练时间         | 172 μs       | 265 μs      | 97 ms       |
| 编码速度         | 8.24 μs/vec  | 1.33 μs/vec | 10.80 μs/vec|
| 解码速度         | 1.71 μs/vec  | 1.30 μs/vec | 0.46 μs/vec |
| 距离计算 (1000)  | ~1 ms        | 337 μs      | 19 μs ⚡    |

**关键发现**:
- Binary Quantization 编码最快
- Product Quantization 距离计算最快（查表法）
- Scalar Quantization 最平衡

---

## 最佳实践

### 1. 选择合适的量化方法

```go
// 决策树
func chooseQuantization(scenario string) quantization.QuantizationType {
    switch scenario {
    case "general":
        // 通用场景: Scalar 8-bit
        return quantization.QuantizationScalar // Bits: 8
        
    case "extreme_compression":
        // 极致压缩: Binary
        return quantization.QuantizationBinary
        
    case "high_accuracy":
        // 高精度: Product Quantization
        return quantization.QuantizationProduct
        
    case "real_time":
        // 实时检索: Binary or Product
        return quantization.QuantizationBinary
        
    default:
        return quantization.QuantizationScalar
    }
}
```

### 2. 训练数据准备

```go
// PQ 需要足够的训练数据
func prepareTrainingData(vectors [][]float32, m, nBits int) [][]float32 {
    minRequired := 1 << nBits // 2^nBits
    
    if len(vectors) < minRequired {
        // 数据增强或降低 nBits
        fmt.Printf("Warning: need at least %d vectors, got %d\n", 
            minRequired, len(vectors))
    }
    
    // 随机采样训练数据
    trainingSize := min(len(vectors), minRequired*10)
    return sampleVectors(vectors, trainingSize)
}
```

### 3. 渐进式压缩

```go
// 先用 Scalar 8-bit，再根据需求降低
func progressiveCompression(vectors [][]float32, dimension int) {
    bits := []int{8, 4, 2}
    
    for _, bit := range bits {
        config := quantization.Config{
            Type: quantization.QuantizationScalar,
            Bits: bit,
        }
        
        q, _ := quantization.NewQuantizer(config, dimension)
        quantized, _ := q.Encode(vectors)
        decoded, _ := q.Decode(quantized)
        
        mse := computeMSE(vectors, decoded)
        
        fmt.Printf("%d-bit: MSE=%.6f, Size=%d\n", 
            bit, mse, quantized.TotalSize())
        
        // 如果 MSE 可接受，停止
        if mse < 0.01 {
            fmt.Printf("使用 %d-bit 量化\n", bit)
            break
        }
    }
}
```

### 4. 混合量化策略

```go
// 不同向量使用不同量化方法
type HybridQuantizer struct {
    importantVectors map[int]bool
    scalarQ          quantization.Quantizer
    binaryQ          quantization.Quantizer
}

func (h *HybridQuantizer) Encode(vectors [][]float32) {
    for i, vec := range vectors {
        if h.importantVectors[i] {
            // 重要向量使用高精度
            h.scalarQ.Encode([][]float32{vec})
        } else {
            // 普通向量使用高压缩
            h.binaryQ.Encode([][]float32{vec})
        }
    }
}
```

### 5. 性能监控

```go
// 监控量化效果
type QuantizationMetrics struct {
    CompressionRatio float64
    MSE              float64
    EncodeTime       time.Duration
    DecodeTime       time.Duration
}

func monitorQuantization(q quantization.Quantizer, vectors [][]float32) QuantizationMetrics {
    start := time.Now()
    quantized, _ := q.Encode(vectors)
    encodeTime := time.Since(start)
    
    start = time.Now()
    decoded, _ := q.Decode(quantized)
    decodeTime := time.Since(start)
    
    return QuantizationMetrics{
        CompressionRatio: q.CompressionRatio(),
        MSE:              computeMSE(vectors, decoded),
        EncodeTime:       encodeTime,
        DecodeTime:       decodeTime,
    }
}
```

---

## API 参考

### Quantizer 接口

```go
type Quantizer interface {
    // Type 返回量化类型
    Type() QuantizationType
    
    // Dimension 返回向量维度
    Dimension() int
    
    // Train 训练量化器
    Train(ctx context.Context, vectors [][]float32) error
    
    // Encode 编码向量
    Encode(vectors [][]float32) (QuantizedVectors, error)
    
    // Decode 解码向量
    Decode(quantized QuantizedVectors) ([][]float32, error)
    
    // ComputeDistance 计算量化向量之间的距离
    ComputeDistance(query QuantizedVector, vectors []QuantizedVector) ([]float32, error)
    
    // CompressionRatio 返回压缩比
    CompressionRatio() float64
    
    // IsTrained 返回量化器是否已训练
    IsTrained() bool
}
```

### QuantizedVectors 接口

```go
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
```

### 错误定义

```go
var (
    ErrInvalidDimension  = errors.New("quantization: invalid dimension")
    ErrInvalidBits       = errors.New("quantization: invalid bits parameter")
    ErrNotTrained        = errors.New("quantization: quantizer not trained")
    ErrInvalidM          = errors.New("quantization: invalid M parameter")
    ErrInvalidNBits      = errors.New("quantization: invalid NBits parameter")
    ErrInsufficientData  = errors.New("quantization: insufficient training data")
)
```

---

## 附录

### A. 完整示例

完整示例代码: `examples/quantization_demo/quantization_demo.go`

### B. 性能测试

```bash
# 运行基准测试
go test -bench=. -benchmem ./retrieval/vectorstores/quantization/...

# 运行完整测试
go test -v ./retrieval/vectorstores/quantization/...

# 跳过耗时的 PQ 测试
go test -v -short ./retrieval/vectorstores/quantization/...
```

### C. 参考文献

1. Product Quantization for Nearest Neighbor Search (Jégou et al., 2011)
2. Billion-scale similarity search with GPUs (Johnson et al., 2017)
3. Binary Quantization for Search (Gong et al., 2013)

---

**文档版本**: v1.0  
**最后更新**: 2026-01-20  
**维护者**: LangChain-Go Team
