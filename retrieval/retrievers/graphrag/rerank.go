package graphrag

import (
	"math"
	"sort"
	"strings"
)

// rerank 重排序结果。
func (r *GraphRAGRetriever) rerank(results []FusedResult, query string, options SearchOptions) []FusedResult {
	switch options.RerankStrategy {
	case RerankStrategyScore:
		return r.rerankByScore(results)
	case RerankStrategyDiversity:
		return r.rerankByDiversity(results)
	case RerankStrategyMMR:
		return r.rerankByMMR(results, query, options.K)
	default:
		return r.rerankByScore(results)
	}
}

// rerankByScore 基于分数重排（已经排序，直接返回）。
func (r *GraphRAGRetriever) rerankByScore(results []FusedResult) []FusedResult {
	// 已经按分数排序，直接返回
	return results
}

// rerankByDiversity 基于多样性重排。
//
// 策略：在保持高分的同时，增加结果的多样性
func (r *GraphRAGRetriever) rerankByDiversity(results []FusedResult) []FusedResult {
	if len(results) <= 1 {
		return results
	}

	// 选择第一个（最高分）
	selected := []FusedResult{results[0]}
	remaining := results[1:]

	// 贪心选择：每次选择与已选择结果最不相似的
	for len(remaining) > 0 && len(selected) < len(results) {
		maxMinSim := -1.0
		maxIdx := 0

		for i, candidate := range remaining {
			// 计算与已选择结果的最小相似度
			minSim := 1.0
			for _, sel := range selected {
				sim := r.calculateSimilarity(candidate, sel)
				if sim < minSim {
					minSim = sim
				}
			}

			// 选择最小相似度最大的（即最不相似的）
			if minSim > maxMinSim {
				maxMinSim = minSim
				maxIdx = i
			}
		}

		// 添加选中的结果
		selected = append(selected, remaining[maxIdx])

		// 从剩余中移除
		remaining = append(remaining[:maxIdx], remaining[maxIdx+1:]...)
	}

	return selected
}

// rerankByMMR Maximal Marginal Relevance 重排。
//
// MMR 公式: MMR = λ * Sim(D, Q) - (1-λ) * max Sim(D, D_i)
// 其中 λ 控制相关性和多样性的平衡
func (r *GraphRAGRetriever) rerankByMMR(results []FusedResult, query string, k int) []FusedResult {
	if len(results) <= 1 {
		return results
	}

	lambda := r.config.MMRLambda
	selected := []FusedResult{}
	remaining := make([]FusedResult, len(results))
	copy(remaining, results)

	// 选择第一个（最高分）
	if len(remaining) > 0 {
		selected = append(selected, remaining[0])
		remaining = remaining[1:]
	}

	// MMR 迭代选择
	for len(selected) < k && len(remaining) > 0 {
		maxMMR := -math.MaxFloat64
		maxIdx := 0

		for i, candidate := range remaining {
			// 相关性分数（使用融合分数）
			relevance := candidate.FusedScore

			// 计算与已选择文档的最大相似度
			maxSim := 0.0
			for _, sel := range selected {
				sim := r.calculateSimilarity(candidate, sel)
				if sim > maxSim {
					maxSim = sim
				}
			}

			// 计算 MMR 分数
			mmr := lambda*relevance - (1-lambda)*maxSim

			if mmr > maxMMR {
				maxMMR = mmr
				maxIdx = i
			}
		}

		// 添加选中的结果
		selected = append(selected, remaining[maxIdx])

		// 从剩余中移除
		remaining = append(remaining[:maxIdx], remaining[maxIdx+1:]...)
	}

	return selected
}

// calculateSimilarity 计算两个结果的相似度。
//
// 使用简单的内容相似度（基于词重叠）
func (r *GraphRAGRetriever) calculateSimilarity(a, b FusedResult) float64 {
	// 提取词
	wordsA := r.extractWords(a.Document.Content)
	wordsB := r.extractWords(b.Document.Content)

	if len(wordsA) == 0 || len(wordsB) == 0 {
		return 0.0
	}

	// 计算 Jaccard 相似度
	intersection := 0
	wordsASet := make(map[string]bool)
	for _, word := range wordsA {
		wordsASet[word] = true
	}

	for _, word := range wordsB {
		if wordsASet[word] {
			intersection++
		}
	}

	union := len(wordsA) + len(wordsB) - intersection

	if union == 0 {
		return 0.0
	}

	return float64(intersection) / float64(union)
}

// extractWords 提取词（简化版本）。
func (r *GraphRAGRetriever) extractWords(text string) []string {
	// 转小写
	text = strings.ToLower(text)

	// 分割（简单空格分割）
	words := strings.Fields(text)

	// 去除标点（简化处理）
	cleaned := make([]string, 0, len(words))
	for _, word := range words {
		word = strings.Trim(word, ".,!?;:\"'()[]{}")
		if len(word) > 0 {
			cleaned = append(cleaned, word)
		}
	}

	return cleaned
}

// RerankWithCustomScorer 使用自定义评分函数重排序。
//
// 参数：
//   - results: 结果列表
//   - scorer: 评分函数 (result) -> score
//
// 返回：
//   - []FusedResult: 重排序后的结果
//
func RerankWithCustomScorer(results []FusedResult, scorer func(FusedResult) float64) []FusedResult {
	// 计算自定义分数
	scored := make([]struct {
		result FusedResult
		score  float64
	}, len(results))

	for i, result := range results {
		scored[i] = struct {
			result FusedResult
			score  float64
		}{
			result: result,
			score:  scorer(result),
		}
	}

	// 排序
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	// 提取结果
	reranked := make([]FusedResult, len(results))
	for i, item := range scored {
		reranked[i] = item.result
		reranked[i].Rank = i + 1
		reranked[i].FusedScore = item.score
	}

	return reranked
}
