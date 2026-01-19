package quantization

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand"
)

// ProductQuantizationConfig 乘积量化配置
type ProductQuantizationConfig struct {
	// Dimension 向量维度
	Dimension int
	
	// M 子向量数量（dimension 必须能被 M 整除）
	M int
	
	// NBits 每个子向量的编码位数 (1-16)
	// 码本大小 = 2^NBits
	NBits int
	
	// TrainingSize 训练样本数量
	TrainingSize int
	
	// MaxIterations K-means 最大迭代次数
	MaxIterations int
	
	// Tolerance K-means 收敛阈值
	Tolerance float32
}

// ProductQuantizer 乘积量化器
//
// 将向量分割为 M 个子向量，分别使用 K-means 聚类，
// 实现高压缩比的同时保持较好的精度。
type ProductQuantizer struct {
	config ProductQuantizationConfig
	
	// 码本（M 个子空间，每个子空间有 K 个聚类中心）
	codebooks [][][]float32 // [M][K][D/M]
	
	// 子向量维度
	subDim int
	
	// 码本大小
	K int
	
	trained bool
}

// NewProductQuantizer 创建乘积量化器
func NewProductQuantizer(config ProductQuantizationConfig) *ProductQuantizer {
	// 设置默认值
	if config.MaxIterations == 0 {
		config.MaxIterations = 100
	}
	if config.Tolerance == 0 {
		config.Tolerance = 1e-4
	}
	
	subDim := config.Dimension / config.M
	K := 1 << config.NBits // 2^NBits
	
	return &ProductQuantizer{
		config:    config,
		subDim:    subDim,
		K:         K,
		codebooks: make([][][]float32, config.M),
		trained:   false,
	}
}

func (q *ProductQuantizer) Type() QuantizationType {
	return QuantizationProduct
}

func (q *ProductQuantizer) Dimension() int {
	return q.config.Dimension
}

func (q *ProductQuantizer) IsTrained() bool {
	return q.trained
}

func (q *ProductQuantizer) CompressionRatio() float64 {
	// 原始: float32 = 32 bits per dimension
	// 量化: NBits per subvector
	// 平均每维度使用: M * NBits / Dimension bits
	bitsPerDim := float64(q.config.M*q.config.NBits) / float64(q.config.Dimension)
	return 32.0 / bitsPerDim
}

// Train 训练量化器（使用 K-means 训练码本）
func (q *ProductQuantizer) Train(ctx context.Context, vectors [][]float32) error {
	if len(vectors) < q.config.TrainingSize {
		return fmt.Errorf("%w: need at least %d vectors, got %d",
			ErrInsufficientData, q.config.TrainingSize, len(vectors))
	}
	
	// 检查维度
	for _, vec := range vectors {
		if len(vec) != q.config.Dimension {
			return fmt.Errorf("%w: expected %d, got %d",
				ErrInvalidDimension, q.config.Dimension, len(vec))
		}
	}
	
	// 为每个子空间训练码本
	for m := 0; m < q.config.M; m++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		
		// 提取子向量
		subVectors := q.extractSubVectors(vectors, m)
		
		// K-means 聚类
		codebook, err := q.kMeans(subVectors, q.K)
		if err != nil {
			return fmt.Errorf("failed to train codebook %d: %w", m, err)
		}
		
		q.codebooks[m] = codebook
	}
	
	q.trained = true
	return nil
}

// extractSubVectors 提取子向量
func (q *ProductQuantizer) extractSubVectors(vectors [][]float32, m int) [][]float32 {
	start := m * q.subDim
	end := start + q.subDim
	
	subVectors := make([][]float32, len(vectors))
	for i, vec := range vectors {
		subVectors[i] = make([]float32, q.subDim)
		copy(subVectors[i], vec[start:end])
	}
	
	return subVectors
}

// kMeans K-means 聚类
func (q *ProductQuantizer) kMeans(vectors [][]float32, k int) ([][]float32, error) {
	if len(vectors) < k {
		return nil, fmt.Errorf("%w: need at least %d vectors for %d clusters",
			ErrInsufficientData, k, k)
	}
	
	// 初始化聚类中心（随机选择 k 个向量）
	centroids := q.initializeCentroids(vectors, k)
	
	// 迭代优化
	for iter := 0; iter < q.config.MaxIterations; iter++ {
		// 分配每个向量到最近的聚类中心
		assignments := make([]int, len(vectors))
		for i, vec := range vectors {
			assignments[i] = q.findNearestCentroid(vec, centroids)
		}
		
		// 更新聚类中心
		newCentroids := q.updateCentroids(vectors, assignments, k)
		
		// 检查收敛
		if q.hasConverged(centroids, newCentroids) {
			break
		}
		
		centroids = newCentroids
	}
	
	return centroids, nil
}

// initializeCentroids 初始化聚类中心（K-means++）
func (q *ProductQuantizer) initializeCentroids(vectors [][]float32, k int) [][]float32 {
	centroids := make([][]float32, k)
	
	// 第一个中心随机选择
	centroids[0] = make([]float32, len(vectors[0]))
	copy(centroids[0], vectors[rand.Intn(len(vectors))])
	
	// 后续中心使用 K-means++ 策略
	for i := 1; i < k; i++ {
		distances := make([]float32, len(vectors))
		totalDist := float32(0)
		
		// 计算每个点到最近中心的距离
		for j, vec := range vectors {
			minDist := float32(math.MaxFloat32)
			for c := 0; c < i; c++ {
				dist := q.l2Distance(vec, centroids[c])
				if dist < minDist {
					minDist = dist
				}
			}
			distances[j] = minDist * minDist // 平方距离
			totalDist += distances[j]
		}
		
		// 按概率选择下一个中心
		r := rand.Float32() * totalDist
		cumSum := float32(0)
		for j, dist := range distances {
			cumSum += dist
			if cumSum >= r {
				centroids[i] = make([]float32, len(vectors[j]))
				copy(centroids[i], vectors[j])
				break
			}
		}
	}
	
	return centroids
}

// findNearestCentroid 找到最近的聚类中心
func (q *ProductQuantizer) findNearestCentroid(vec []float32, centroids [][]float32) int {
	minDist := float32(math.MaxFloat32)
	nearest := 0
	
	for i, centroid := range centroids {
		dist := q.l2Distance(vec, centroid)
		if dist < minDist {
			minDist = dist
			nearest = i
		}
	}
	
	return nearest
}

// updateCentroids 更新聚类中心
func (q *ProductQuantizer) updateCentroids(vectors [][]float32, assignments []int, k int) [][]float32 {
	newCentroids := make([][]float32, k)
	counts := make([]int, k)
	
	// 初始化
	for i := 0; i < k; i++ {
		newCentroids[i] = make([]float32, len(vectors[0]))
	}
	
	// 累加
	for i, vec := range vectors {
		cluster := assignments[i]
		counts[cluster]++
		for j, val := range vec {
			newCentroids[cluster][j] += val
		}
	}
	
	// 平均
	for i := 0; i < k; i++ {
		if counts[i] > 0 {
			for j := range newCentroids[i] {
				newCentroids[i][j] /= float32(counts[i])
			}
		}
	}
	
	return newCentroids
}

// hasConverged 检查是否收敛
func (q *ProductQuantizer) hasConverged(old, new [][]float32) bool {
	totalChange := float32(0)
	
	for i := range old {
		totalChange += q.l2Distance(old[i], new[i])
	}
	
	return totalChange < q.config.Tolerance
}

// l2Distance 计算 L2 距离
func (q *ProductQuantizer) l2Distance(a, b []float32) float32 {
	sum := float32(0)
	for i := range a {
		diff := a[i] - b[i]
		sum += diff * diff
	}
	return float32(math.Sqrt(float64(sum)))
}

// Encode 编码向量
func (q *ProductQuantizer) Encode(vectors [][]float32) (QuantizedVectors, error) {
	if !q.trained {
		return nil, ErrNotTrained
	}
	
	// 验证维度
	for i, vec := range vectors {
		if len(vec) != q.config.Dimension {
			return nil, fmt.Errorf("%w: vector %d has dimension %d, expected %d",
				ErrInvalidDimension, i, len(vec), q.config.Dimension)
		}
	}
	
	// 编码每个向量
	codes := make([][]uint16, len(vectors))
	for i, vec := range vectors {
		codes[i] = q.encodeVector(vec)
	}
	
	// 打包成字节数组
	data := q.packCodes(codes)
	
	return &productQuantizedVectors{
		m:         q.config.M,
		nBits:     q.config.NBits,
		dimension: q.config.Dimension,
		count:     len(vectors),
		data:      data,
		codebooks: q.codebooks,
	}, nil
}

// encodeVector 编码单个向量
func (q *ProductQuantizer) encodeVector(vec []float32) []uint16 {
	codes := make([]uint16, q.config.M)
	
	for m := 0; m < q.config.M; m++ {
		start := m * q.subDim
		end := start + q.subDim
		subVec := vec[start:end]
		
		// 找到最近的码本条目
		codes[m] = uint16(q.findNearestCentroid(subVec, q.codebooks[m]))
	}
	
	return codes
}

// packCodes 打包编码
func (q *ProductQuantizer) packCodes(codes [][]uint16) []byte {
	// 每个向量需要 M * NBits bits
	bitsPerVector := q.config.M * q.config.NBits
	bytesPerVector := (bitsPerVector + 7) / 8
	
	data := make([]byte, len(codes)*bytesPerVector)
	
	for i, code := range codes {
		offset := i * bytesPerVector
		q.packCode(code, data[offset:offset+bytesPerVector])
	}
	
	return data
}

// packCode 打包单个编码
func (q *ProductQuantizer) packCode(code []uint16, dst []byte) {
	bitPos := 0
	
	for _, c := range code {
		// 写入 NBits bits
		for bit := q.config.NBits - 1; bit >= 0; bit-- {
			if (c & (1 << bit)) != 0 {
				byteIdx := bitPos / 8
				bitIdx := 7 - (bitPos % 8)
				dst[byteIdx] |= 1 << bitIdx
			}
			bitPos++
		}
	}
}

// Decode 解码向量
func (q *ProductQuantizer) Decode(quantized QuantizedVectors) ([][]float32, error) {
	pq, ok := quantized.(*productQuantizedVectors)
	if !ok {
		return nil, errors.New("quantization: invalid quantized vectors type")
	}
	
	vectors := make([][]float32, pq.count)
	
	bitsPerVector := q.config.M * q.config.NBits
	bytesPerVector := (bitsPerVector + 7) / 8
	
	for i := 0; i < pq.count; i++ {
		offset := i * bytesPerVector
		codes := q.unpackCode(pq.data[offset:offset+bytesPerVector], q.config.M, q.config.NBits)
		vectors[i] = q.reconstructVector(codes)
	}
	
	return vectors, nil
}

// unpackCode 解包编码
func (q *ProductQuantizer) unpackCode(data []byte, m, nBits int) []uint16 {
	codes := make([]uint16, m)
	bitPos := 0
	
	for i := 0; i < m; i++ {
		code := uint16(0)
		for bit := nBits - 1; bit >= 0; bit-- {
			byteIdx := bitPos / 8
			bitIdx := 7 - (bitPos % 8)
			if (data[byteIdx] & (1 << bitIdx)) != 0 {
				code |= 1 << bit
			}
			bitPos++
		}
		codes[i] = code
	}
	
	return codes
}

// reconstructVector 重构向量
func (q *ProductQuantizer) reconstructVector(codes []uint16) []float32 {
	vec := make([]float32, q.config.Dimension)
	
	for m := 0; m < q.config.M; m++ {
		start := m * q.subDim
		centroid := q.codebooks[m][codes[m]]
		copy(vec[start:start+q.subDim], centroid)
	}
	
	return vec
}

// ComputeDistance 计算距离（使用查表法优化）
func (q *ProductQuantizer) ComputeDistance(query QuantizedVector, vectors []QuantizedVector) ([]float32, error) {
	pq, ok := query.(*productQuantizedVector)
	if !ok {
		return nil, errors.New("quantization: invalid query vector type")
	}
	
	// 预计算距离表
	distTable := q.computeDistanceTable(pq.reconstructed)
	
	distances := make([]float32, len(vectors))
	
	for i, vec := range vectors {
		pv, ok := vec.(*productQuantizedVector)
		if !ok {
			return nil, fmt.Errorf("quantization: invalid vector type at index %d", i)
		}
		
		distances[i] = q.asymmetricDistance(pv.codes, distTable)
	}
	
	return distances, nil
}

// computeDistanceTable 计算距离表
func (q *ProductQuantizer) computeDistanceTable(query []float32) [][]float32 {
	distTable := make([][]float32, q.config.M)
	
	for m := 0; m < q.config.M; m++ {
		distTable[m] = make([]float32, q.K)
		start := m * q.subDim
		end := start + q.subDim
		subQuery := query[start:end]
		
		for k := 0; k < q.K; k++ {
			distTable[m][k] = q.l2Distance(subQuery, q.codebooks[m][k])
		}
	}
	
	return distTable
}

// asymmetricDistance 非对称距离计算
func (q *ProductQuantizer) asymmetricDistance(codes []uint16, distTable [][]float32) float32 {
	sum := float32(0)
	
	for m := 0; m < q.config.M; m++ {
		dist := distTable[m][codes[m]]
		sum += dist * dist
	}
	
	return float32(math.Sqrt(float64(sum)))
}

// productQuantizedVectors 乘积量化的向量集合
type productQuantizedVectors struct {
	m         int
	nBits     int
	dimension int
	count     int
	data      []byte
	codebooks [][][]float32
}

func (v *productQuantizedVectors) Type() QuantizationType {
	return QuantizationProduct
}

func (v *productQuantizedVectors) Count() int {
	return v.count
}

func (v *productQuantizedVectors) Data() []byte {
	return v.data
}

func (v *productQuantizedVectors) TotalSize() int {
	// data + codebooks size estimation
	codebookSize := v.m * (1 << v.nBits) * (v.dimension / v.m) * 4
	return len(v.data) + codebookSize
}

func (v *productQuantizedVectors) Get(index int) (QuantizedVector, error) {
	if index < 0 || index >= v.count {
		return nil, errors.New("quantization: index out of range")
	}
	
	bitsPerVector := v.m * v.nBits
	bytesPerVector := (bitsPerVector + 7) / 8
	start := index * bytesPerVector
	end := start + bytesPerVector
	
	vecData := make([]byte, bytesPerVector)
	copy(vecData, v.data[start:end])
	
	// 解包编码
	pq := &ProductQuantizer{
		config: ProductQuantizationConfig{
			M:         v.m,
			NBits:     v.nBits,
			Dimension: v.dimension,
		},
		codebooks: v.codebooks,
	}
	codes := pq.unpackCode(vecData, v.m, v.nBits)
	reconstructed := pq.reconstructVector(codes)
	
	return &productQuantizedVector{
		m:             v.m,
		nBits:         v.nBits,
		dimension:     v.dimension,
		data:          vecData,
		codes:         codes,
		reconstructed: reconstructed,
	}, nil
}

// productQuantizedVector 乘积量化的单个向量
type productQuantizedVector struct {
	m             int
	nBits         int
	dimension     int
	data          []byte
	codes         []uint16
	reconstructed []float32
}

func (v *productQuantizedVector) Type() QuantizationType {
	return QuantizationProduct
}

func (v *productQuantizedVector) Data() []byte {
	return v.data
}

func (v *productQuantizedVector) Size() int {
	return len(v.data)
}
