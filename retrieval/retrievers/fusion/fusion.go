// Package fusion 提供多种检索结果融合策略。
//
// 融合策略用于合并来自不同检索器（如向量检索、关键词检索）的结果。
// 常用的融合策略包括 RRF (Reciprocal Rank Fusion) 和加权融合。
//
// 使用示例：
//
//	// RRF 融合
//	strategy := fusion.NewRRFStrategy(60)
//	results := strategy.Fuse([]fusion.RankedList{vectorResults, keywordResults})
//
//	// 加权融合
//	strategy := fusion.NewWeightedStrategy(map[string]float64{
//	    "vector": 0.7,
//	    "keyword": 0.3,
//	})
//	results := strategy.Fuse([]fusion.RankedList{vectorResults, keywordResults})
//
package fusion

import (
	"sort"

	"github.com/zhucl121/langchain-go/pkg/types"
)

// FusionStrategy 融合策略接口
//
// 定义如何合并多个检索结果列表。
type FusionStrategy interface {
	// Fuse 融合多个排序列表
	//
	// 参数：
	//   - rankedLists: 多个排序列表，每个列表来自不同的检索器
	//
	// 返回：
	//   - []FusedDocument: 融合并重新排序后的文档列表
	Fuse(rankedLists []RankedList) []FusedDocument
}

// RankedList 排序列表
//
// 表示单个检索器返回的排序结果。
type RankedList struct {
	// Source 来源标识（如 "vector", "keyword"）
	Source string

	// Documents 排序后的文档列表
	Documents []RankedDocument
}

// RankedDocument 带排名的文档
type RankedDocument struct {
	Document types.Document
	Score    float64
	Rank     int // 从 1 开始
}

// FusedDocument 融合后的文档
type FusedDocument struct {
	Document types.Document
	Score    float64

	// SourceScores 记录各来源的原始分数（用于调试）
	SourceScores map[string]float64

	// SourceRanks 记录各来源的原始排名
	SourceRanks map[string]int
}

// RRFStrategy Reciprocal Rank Fusion 策略
//
// RRF 是一种基于排名的融合算法，对每个文档计算：
// score = Σ 1 / (k + rank_i)
//
// 其中 k 是一个常量（通常为 60），rank_i 是文档在第 i 个列表中的排名。
// RRF 的优点是不依赖原始分数的尺度，只依赖排名。
//
// 参考：
// Cormack, G. V., Clarke, C. L., & Buettcher, S. (2009).
// "Reciprocal rank fusion outperforms condorcet and individual rank learning methods."
type RRFStrategy struct {
	// K 常量，通常取值 60
	K float64
}

// NewRRFStrategy 创建 RRF 策略
func NewRRFStrategy(k float64) *RRFStrategy {
	if k <= 0 {
		k = 60 // 默认值
	}
	return &RRFStrategy{K: k}
}

// Fuse 实现 FusionStrategy 接口
func (s *RRFStrategy) Fuse(rankedLists []RankedList) []FusedDocument {
	if len(rankedLists) == 0 {
		return []FusedDocument{}
	}

	// 收集所有文档并计算 RRF 分数
	docScores := make(map[string]*FusedDocument)

	for _, rankedList := range rankedLists {
		for _, rankedDoc := range rankedList.Documents {
			// 使用文档内容作为唯一标识（实际应用可能需要更好的 ID）
			docKey := getDocumentKey(rankedDoc.Document)

			if _, exists := docScores[docKey]; !exists {
				docScores[docKey] = &FusedDocument{
					Document:     rankedDoc.Document,
					Score:        0,
					SourceScores: make(map[string]float64),
					SourceRanks:  make(map[string]int),
				}
			}

			// 计算 RRF 分数: 1 / (k + rank)
			rrfScore := 1.0 / (s.K + float64(rankedDoc.Rank))
			docScores[docKey].Score += rrfScore

			// 记录来源信息
			docScores[docKey].SourceScores[rankedList.Source] = rankedDoc.Score
			docScores[docKey].SourceRanks[rankedList.Source] = rankedDoc.Rank
		}
	}

	// 转换为列表并排序
	results := make([]FusedDocument, 0, len(docScores))
	for _, doc := range docScores {
		results = append(results, *doc)
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	return results
}

// WeightedStrategy 加权融合策略
//
// 对每个来源的分数乘以权重，然后求和：
// score = Σ weight_i * score_i
//
// 注意：要求各来源的分数在相同的尺度上，或者经过归一化。
type WeightedStrategy struct {
	// Weights 每个来源的权重，总和应为 1.0
	Weights map[string]float64

	// Normalize 是否对分数进行归一化（MinMax 归一化）
	Normalize bool
}

// NewWeightedStrategy 创建加权融合策略
func NewWeightedStrategy(weights map[string]float64) *WeightedStrategy {
	return &WeightedStrategy{
		Weights:   weights,
		Normalize: true, // 默认开启归一化
	}
}

// Fuse 实现 FusionStrategy 接口
func (s *WeightedStrategy) Fuse(rankedLists []RankedList) []FusedDocument {
	if len(rankedLists) == 0 {
		return []FusedDocument{}
	}

	// 如果需要归一化，先对每个列表归一化
	normalizedLists := rankedLists
	if s.Normalize {
		normalizedLists = s.normalizeLists(rankedLists)
	}

	// 收集所有文档并计算加权分数
	docScores := make(map[string]*FusedDocument)

	for _, rankedList := range normalizedLists {
		weight := s.Weights[rankedList.Source]
		if weight == 0 {
			weight = 1.0 / float64(len(normalizedLists)) // 默认平均权重
		}

		for _, rankedDoc := range rankedList.Documents {
			docKey := getDocumentKey(rankedDoc.Document)

			if _, exists := docScores[docKey]; !exists {
				docScores[docKey] = &FusedDocument{
					Document:     rankedDoc.Document,
					Score:        0,
					SourceScores: make(map[string]float64),
					SourceRanks:  make(map[string]int),
				}
			}

			// 加权分数
			weightedScore := weight * rankedDoc.Score
			docScores[docKey].Score += weightedScore

			// 记录来源信息
			docScores[docKey].SourceScores[rankedList.Source] = rankedDoc.Score
			docScores[docKey].SourceRanks[rankedList.Source] = rankedDoc.Rank
		}
	}

	// 转换为列表并排序
	results := make([]FusedDocument, 0, len(docScores))
	for _, doc := range docScores {
		results = append(results, *doc)
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	return results
}

// normalizeLists 对每个列表的分数进行 MinMax 归一化
func (s *WeightedStrategy) normalizeLists(lists []RankedList) []RankedList {
	normalized := make([]RankedList, len(lists))

	for i, list := range lists {
		normalized[i] = RankedList{
			Source:    list.Source,
			Documents: make([]RankedDocument, len(list.Documents)),
		}

		// 找出最大最小分数
		if len(list.Documents) == 0 {
			continue
		}

		minScore := list.Documents[0].Score
		maxScore := list.Documents[0].Score

		for _, doc := range list.Documents {
			if doc.Score < minScore {
				minScore = doc.Score
			}
			if doc.Score > maxScore {
				maxScore = doc.Score
			}
		}

		// MinMax 归一化
		scoreRange := maxScore - minScore
		
		for j, doc := range list.Documents {
			var normalizedScore float64
			if scoreRange == 0 {
				// 所有分数相同，归一化为 1.0
				normalizedScore = 1.0
			} else {
				normalizedScore = (doc.Score - minScore) / scoreRange
			}
			
			normalized[i].Documents[j] = RankedDocument{
				Document: doc.Document,
				Score:    normalizedScore,
				Rank:     doc.Rank,
			}
		}
	}

	return normalized
}

// LinearCombinationStrategy 线性组合策略
//
// 简单的线性组合，不进行归一化：
// score = Σ weight_i * score_i
//
// 适用于分数已经在相同尺度的场景。
type LinearCombinationStrategy struct {
	Weights map[string]float64
}

// NewLinearCombinationStrategy 创建线性组合策略
func NewLinearCombinationStrategy(weights map[string]float64) *LinearCombinationStrategy {
	return &LinearCombinationStrategy{
		Weights: weights,
	}
}

// Fuse 实现 FusionStrategy 接口
func (s *LinearCombinationStrategy) Fuse(rankedLists []RankedList) []FusedDocument {
	if len(rankedLists) == 0 {
		return []FusedDocument{}
	}

	docScores := make(map[string]*FusedDocument)

	for _, rankedList := range rankedLists {
		weight := s.Weights[rankedList.Source]
		if weight == 0 {
			weight = 1.0
		}

		for _, rankedDoc := range rankedList.Documents {
			docKey := getDocumentKey(rankedDoc.Document)

			if _, exists := docScores[docKey]; !exists {
				docScores[docKey] = &FusedDocument{
					Document:     rankedDoc.Document,
					Score:        0,
					SourceScores: make(map[string]float64),
					SourceRanks:  make(map[string]int),
				}
			}

			docScores[docKey].Score += weight * rankedDoc.Score
			docScores[docKey].SourceScores[rankedList.Source] = rankedDoc.Score
			docScores[docKey].SourceRanks[rankedList.Source] = rankedDoc.Rank
		}
	}

	results := make([]FusedDocument, 0, len(docScores))
	for _, doc := range docScores {
		results = append(results, *doc)
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	return results
}

// getDocumentKey 生成文档的唯一标识
//
// 注意：这是一个简化实现，实际应用应该使用更可靠的 ID 系统。
func getDocumentKey(doc types.Document) string {
	// 优先使用 Metadata 中的 ID
	if id, ok := doc.Metadata["id"].(string); ok && id != "" {
		return id
	}

	// 使用内容的前 100 个字符作为 key
	content := doc.Content
	if len(content) > 100 {
		content = content[:100]
	}

	return content
}

// ConvertToRankedList 辅助函数：将带分数的文档列表转换为 RankedList
func ConvertToRankedList(source string, docs []types.Document, scores []float64) RankedList {
	rankedDocs := make([]RankedDocument, len(docs))

	for i := range docs {
		score := 0.0
		if i < len(scores) {
			score = scores[i]
		}

		rankedDocs[i] = RankedDocument{
			Document: docs[i],
			Score:    score,
			Rank:     i + 1, // 排名从 1 开始
		}
	}

	return RankedList{
		Source:    source,
		Documents: rankedDocs,
	}
}
