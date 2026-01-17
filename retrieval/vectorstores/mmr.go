package vectorstores

import (
	"context"
	"fmt"

	"github.com/zhucl121/langchain-go/retrieval/loaders"
)

// MMROptions 是 MMR 搜索的配置选项。
//
// MMR (Maximum Marginal Relevance) 算法在保持相关性的同时增加结果的多样性。
//
type MMROptions struct {
	// Lambda 控制相关性和多样性的平衡
	// - 1.0 = 最大相关性（与普通搜索相同）
	// - 0.0 = 最大多样性
	// - 0.5 = 平衡相关性和多样性（推荐默认值）
	Lambda float32

	// FetchK 是初始获取的候选文档数量
	// 应该大于最终返回的 K 个结果
	// 默认值是 K 的 4 倍
	FetchK int
}

// DefaultMMROptions 返回默认的 MMR 选项。
func DefaultMMROptions(k int) *MMROptions {
	return &MMROptions{
		Lambda: 0.5,
		FetchK: k * 4,
	}
}

// Validate 验证 MMR 选项的有效性。
func (opts *MMROptions) Validate(k int) error {
	if opts.Lambda < 0 || opts.Lambda > 1 {
		return fmt.Errorf("lambda must be between 0 and 1, got %f", opts.Lambda)
	}
	if opts.FetchK < k {
		return fmt.Errorf("fetchK (%d) must be >= k (%d)", opts.FetchK, k)
	}
	return nil
}

// MMRVectorStore 是支持 MMR 搜索的向量存储接口。
//
// 实现该接口的向量存储可以使用 MMR 算法进行多样性搜索。
//
type MMRVectorStore interface {
	VectorStore

	// SimilaritySearchWithMMR 使用 MMR 算法进行相似度搜索。
	//
	// 参数：
	//   - ctx: 上下文
	//   - query: 查询文本
	//   - k: 返回结果数量
	//   - options: MMR 选项（可选，使用默认值）
	//
	// 返回：
	//   - []*loaders.Document: 多样性文档列表
	//   - error: 错误
	//
	SimilaritySearchWithMMR(ctx context.Context, query string, k int, options *MMROptions) ([]*loaders.Document, error)
}

// maxMarginalRelevance 实现 MMR 算法。
//
// 算法描述：
//  1. 从候选文档中选择与查询最相似的文档
//  2. 迭代选择剩余文档中满足以下条件的文档：
//     - 与查询相似（相关性）
//     - 与已选文档不相似（多样性）
//  3. 使用 lambda 参数平衡相关性和多样性
//
// 参数：
//   - queryVector: 查询向量
//   - candidateVectors: 候选文档向量列表
//   - k: 返回结果数量
//   - lambda: 相关性权重（0-1）
//
// 返回：
//   - []int: 选中的候选文档索引列表
//
func maxMarginalRelevance(queryVector []float32, candidateVectors [][]float32, k int, lambda float32) []int {
	if len(candidateVectors) == 0 {
		return []int{}
	}

	if k >= len(candidateVectors) {
		// 如果 k 大于等于候选数量，返回所有候选
		indices := make([]int, len(candidateVectors))
		for i := range indices {
			indices[i] = i
		}
		return indices
	}

	// 计算查询与所有候选的相似度
	querySimilarities := make([]float32, len(candidateVectors))
	for i, vec := range candidateVectors {
		querySimilarities[i] = cosineSimilarity(queryVector, vec)
	}

	// 存储已选择的文档索引
	selected := make([]int, 0, k)
	remaining := make(map[int]bool)
	for i := range candidateVectors {
		remaining[i] = true
	}

	// 第一步：选择与查询最相似的文档
	maxIdx := 0
	maxScore := querySimilarities[0]
	for i := 1; i < len(querySimilarities); i++ {
		if querySimilarities[i] > maxScore {
			maxScore = querySimilarities[i]
			maxIdx = i
		}
	}
	selected = append(selected, maxIdx)
	delete(remaining, maxIdx)

	// 第二步：迭代选择剩余文档
	for len(selected) < k && len(remaining) > 0 {
		maxMMRScore := float32(-1)
		maxMMRIdx := -1

		// 对每个剩余文档计算 MMR 分数
		for idx := range remaining {
			// 相关性分数：与查询的相似度
			relevanceScore := querySimilarities[idx]

			// 多样性分数：与已选文档的最大相似度（越小越好）
			maxSimToSelected := float32(-1)
			for _, selectedIdx := range selected {
				sim := cosineSimilarity(candidateVectors[idx], candidateVectors[selectedIdx])
				if sim > maxSimToSelected {
					maxSimToSelected = sim
				}
			}

			// MMR 分数：lambda * 相关性 - (1 - lambda) * 多样性
			mmrScore := lambda*relevanceScore - (1-lambda)*maxSimToSelected

			// 选择 MMR 分数最高的文档
			if mmrScore > maxMMRScore {
				maxMMRScore = mmrScore
				maxMMRIdx = idx
			}
		}

		if maxMMRIdx >= 0 {
			selected = append(selected, maxMMRIdx)
			delete(remaining, maxMMRIdx)
		} else {
			// 异常情况：无法选择更多文档
			break
		}
	}

	return selected
}

// SimilaritySearchWithMMR 为 InMemoryVectorStore 实现 MMR 搜索。
func (store *InMemoryVectorStore) SimilaritySearchWithMMR(
	ctx context.Context,
	query string,
	k int,
	options *MMROptions,
) ([]*loaders.Document, error) {
	// 使用默认选项
	if options == nil {
		options = DefaultMMROptions(k)
	}

	// 验证选项
	if err := options.Validate(k); err != nil {
		return nil, fmt.Errorf("invalid MMR options: %w", err)
	}

	// 生成查询向量
	queryVector, err := store.embeddings.EmbedQuery(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to embed query: %w", err)
	}

	// 获取候选文档（fetchK 个）
	candidates, err := store.SimilaritySearchWithScore(ctx, query, options.FetchK)
	if err != nil {
		return nil, fmt.Errorf("failed to get candidates: %w", err)
	}

	if len(candidates) == 0 {
		return []*loaders.Document{}, nil
	}

	// 如果候选数量小于等于 k，直接返回所有候选
	if len(candidates) <= k {
		docs := make([]*loaders.Document, len(candidates))
		for i, candidate := range candidates {
			docs[i] = candidate.Document
		}
		return docs, nil
	}

	// 提取候选向量
	store.mu.RLock()
	candidateVectors := make([][]float32, len(candidates))
	for i, candidate := range candidates {
		// 根据文档内容查找对应的向量
		for id, doc := range store.documents {
			if doc == candidate.Document {
				candidateVectors[i] = store.vectors[id]
				break
			}
		}
	}
	store.mu.RUnlock()

	// 应用 MMR 算法
	selectedIndices := maxMarginalRelevance(queryVector, candidateVectors, k, options.Lambda)

	// 构建结果
	results := make([]*loaders.Document, len(selectedIndices))
	for i, idx := range selectedIndices {
		results[i] = candidates[idx].Document
	}

	return results, nil
}
