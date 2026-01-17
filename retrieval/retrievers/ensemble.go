package retrievers

import (
	"context"
	"sort"

	"github.com/zhucl121/langchain-go/retrieval/loaders"
)

// EnsembleRetriever 集成检索器
//
// 融合多个检索器的结果，使用 RRF (Reciprocal Rank Fusion) 算法。
//
// RRF 算法：
//   - 对每个检索器的结果，根据排名计算分数
//   - 分数公式: score = weight / (k + rank)
//   - 对相同文档的分数求和
//   - 按最终分数排序
//
// 适用场景：
//   - 混合检索 (向量搜索 + BM25)
//   - 多策略融合
//   - 提高检索鲁棒性
//
type EnsembleRetriever struct {
	*BaseRetriever
	retrievers []Retriever
	weights    []float64
	rrfK       int
}

// NewEnsembleRetriever 创建集成检索器
//
// 参数：
//   - retrievers: 检索器列表
//   - opts: 可选配置项
//
// 返回：
//   - *EnsembleRetriever: 检索器实例
//
// 使用示例：
//
//	ensemble := retrievers.NewEnsembleRetriever(
//	    []retrievers.Retriever{vectorRetriever, bm25Retriever},
//	    retrievers.WithWeights([]float64{0.5, 0.5}),
//	    retrievers.WithRRFK(60),
//	)
//
func NewEnsembleRetriever(retrievers []Retriever, opts ...EnsembleOption) *EnsembleRetriever {
	// 默认等权重
	weights := make([]float64, len(retrievers))
	for i := range weights {
		weights[i] = 1.0 / float64(len(retrievers))
	}

	r := &EnsembleRetriever{
		BaseRetriever: NewBaseRetriever(),
		retrievers:    retrievers,
		weights:       weights,
		rrfK:          60, // RRF 默认 k=60 (来自论文)
	}

	for _, opt := range opts {
		opt(r)
	}

	return r
}

// 配置选项函数

// WithWeights 设置检索器权重
//
// 权重数量必须与检索器数量一致。
//
func WithWeights(weights []float64) EnsembleOption {
	return func(r *EnsembleRetriever) {
		if len(weights) == len(r.retrievers) {
			r.weights = weights
		}
	}
}

// WithRRFK 设置 RRF 常数 k
//
// k 值越大，排名对分数的影响越小。
// 推荐值: 60 (来自原始论文)
//
func WithRRFK(k int) EnsembleOption {
	return func(r *EnsembleRetriever) {
		r.rrfK = k
	}
}

// GetRelevantDocuments 实现 Retriever 接口
func (r *EnsembleRetriever) GetRelevantDocuments(ctx context.Context, query string) ([]*loaders.Document, error) {
	// 触发开始回调
	r.triggerStart(ctx, query)

	// 1. 从所有检索器获取带分数的结果
	var allResults [][]DocumentWithScore

	for _, retriever := range r.retrievers {
		docs, err := retriever.GetRelevantDocumentsWithScore(ctx, query)
		if err != nil {
			// 忽略单个检索器的错误，继续处理
			continue
		}
		allResults = append(allResults, docs)
	}

	if len(allResults) == 0 {
		r.triggerEnd(ctx, []*loaders.Document{})
		return []*loaders.Document{}, nil
	}

	// 2. 使用 RRF 融合
	fusedResults := r.applyRRF(allResults)

	// 3. 转换为文档列表
	docs := make([]*loaders.Document, len(fusedResults))
	for i, result := range fusedResults {
		docs[i] = result.Document
	}

	// 触发结束回调
	r.triggerEnd(ctx, docs)

	return docs, nil
}

// GetRelevantDocumentsWithScore 实现 Retriever 接口
func (r *EnsembleRetriever) GetRelevantDocumentsWithScore(ctx context.Context, query string) ([]DocumentWithScore, error) {
	// 触发开始回调
	r.triggerStart(ctx, query)

	// 1. 从所有检索器获取结果
	var allResults [][]DocumentWithScore

	for _, retriever := range r.retrievers {
		docs, err := retriever.GetRelevantDocumentsWithScore(ctx, query)
		if err != nil {
			continue
		}
		allResults = append(allResults, docs)
	}

	if len(allResults) == 0 {
		r.triggerEnd(ctx, []*loaders.Document{})
		return []DocumentWithScore{}, nil
	}

	// 2. 使用 RRF 融合
	fusedResults := r.applyRRF(allResults)

	// 触发结束回调
	plainDocs := make([]*loaders.Document, len(fusedResults))
	for i, d := range fusedResults {
		plainDocs[i] = d.Document
	}
	r.triggerEnd(ctx, plainDocs)

	return fusedResults, nil
}

// scoredDoc 临时结构，用于 RRF 计算
type scoredDoc struct {
	doc   *loaders.Document
	score float32
}

// applyRRF 应用 Reciprocal Rank Fusion 算法
//
// RRF 算法步骤:
//  1. 对每个结果集，根据排名计算 RRF 分数
//  2. 对相同文档（基于内容哈希）的分数求和
//  3. 按最终分数降序排序
//
func (r *EnsembleRetriever) applyRRF(resultSets [][]DocumentWithScore) []DocumentWithScore {
	docScores := make(map[string]*scoredDoc)

	// 遍历每个结果集
	for setIdx, results := range resultSets {
		weight := r.weights[setIdx]

		// 遍历结果集中的每个文档
		for rank, docWithScore := range results {
			// 使用内容哈希作为唯一标识
			key := hashContent(docWithScore.Document.Content)

			if _, exists := docScores[key]; !exists {
				docScores[key] = &scoredDoc{
					doc:   docWithScore.Document,
					score: 0,
				}
			}

			// RRF 公式: weight / (k + rank + 1)
			// rank 从 0 开始，所以加 1
			rrfScore := weight / float64(r.rrfK+rank+1)
			docScores[key].score += float32(rrfScore)
		}
	}

	// 转换为切片
	var results []DocumentWithScore
	for _, sd := range docScores {
		// 更新分数到元数据
		if sd.doc.Metadata == nil {
			sd.doc.Metadata = make(map[string]interface{})
		}
		sd.doc.Metadata["score"] = sd.score

		results = append(results, DocumentWithScore{
			Document: sd.doc,
			Score:    sd.score,
		})
	}

	// 按分数降序排序
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	return results
}

// AddRetriever 动态添加检索器
//
// 添加后，权重会被重新均分。
//
func (r *EnsembleRetriever) AddRetriever(retriever Retriever, weight float64) {
	r.retrievers = append(r.retrievers, retriever)
	r.weights = append(r.weights, weight)

	// 归一化权重
	r.normalizeWeights()
}

// normalizeWeights 归一化权重，使总和为 1
func (r *EnsembleRetriever) normalizeWeights() {
	var sum float64
	for _, w := range r.weights {
		sum += w
	}

	if sum > 0 {
		for i := range r.weights {
			r.weights[i] /= sum
		}
	}
}

// GetRetrievers 获取所有检索器
func (r *EnsembleRetriever) GetRetrievers() []Retriever {
	return r.retrievers
}

// GetWeights 获取权重
func (r *EnsembleRetriever) GetWeights() []float64 {
	return r.weights
}

// SetWeights 设置权重
func (r *EnsembleRetriever) SetWeights(weights []float64) {
	if len(weights) == len(r.retrievers) {
		r.weights = weights
	}
}

// GetRRFK 获取 RRF k 值
func (r *EnsembleRetriever) GetRRFK() int {
	return r.rrfK
}

// SetRRFK 设置 RRF k 值
func (r *EnsembleRetriever) SetRRFK(k int) {
	r.rrfK = k
}
