package graphrag

import (
	"fmt"
	"math"
	"sort"

	"github.com/zhucl121/langchain-go/pkg/types"
	"github.com/zhucl121/langchain-go/retrieval/graphdb"
)

// fuseResults 融合向量和图检索结果。
func (r *GraphRAGRetriever) fuseResults(vectorDocs []*types.Document, graphNodes []*graphdb.Node, options SearchOptions) []FusedResult {
	switch options.FusionStrategy {
	case FusionStrategyWeighted:
		return r.fusedWeighted(vectorDocs, graphNodes, options)
	case FusionStrategyRRF:
		return r.fuseRRF(vectorDocs, graphNodes, options)
	case FusionStrategyMax:
		return r.fuseMax(vectorDocs, graphNodes, options)
	case FusionStrategyMin:
		return r.fuseMin(vectorDocs, graphNodes, options)
	default:
		return r.fusedWeighted(vectorDocs, graphNodes, options)
	}
}

// fusedWeighted 加权融合。
func (r *GraphRAGRetriever) fusedWeighted(vectorDocs []*types.Document, graphNodes []*graphdb.Node, options SearchOptions) []FusedResult {
	// 创建结果映射
	resultMap := make(map[string]*FusedResult)

	// 处理向量检索结果
	for i, doc := range vectorDocs {
		// 计算向量分数（基于排名）
		vectorScore := 1.0 - float64(i)/float64(len(vectorDocs))

		key := r.getDocumentKey(doc)
		resultMap[key] = &FusedResult{
			Document:     doc,
			VectorScore:  vectorScore,
			GraphScore:   0.0,
			RelatedNodes: []*graphdb.Node{},
			Metadata:     make(map[string]interface{}),
		}
	}

	// 处理图检索结果
	for j, node := range graphNodes {
		// 计算图分数（基于排名）
		graphScore := 1.0 - float64(j)/float64(len(graphNodes))

		doc := r.nodeToDocument(node)
		key := r.getDocumentKey(doc)

		if existing, ok := resultMap[key]; ok {
			// 已存在，更新图分数
			existing.GraphScore = graphScore
			existing.RelatedNodes = append(existing.RelatedNodes, node)
		} else {
			// 新文档
			resultMap[key] = &FusedResult{
				Document:     doc,
				VectorScore:  0.0,
				GraphScore:   graphScore,
				RelatedNodes: []*graphdb.Node{node},
				Metadata:     make(map[string]interface{}),
			}
		}
	}

	// 计算融合分数
	results := make([]FusedResult, 0, len(resultMap))
	for _, result := range resultMap {
		// 加权融合
		result.FusedScore = options.VectorWeight*result.VectorScore + options.GraphWeight*result.GraphScore

		// 归一化权重
		totalWeight := options.VectorWeight + options.GraphWeight
		if totalWeight > 0 {
			result.FusedScore /= totalWeight
		}

		results = append(results, *result)
	}

	// 按融合分数排序
	sort.Slice(results, func(i, j int) bool {
		return results[i].FusedScore > results[j].FusedScore
	})

	// 设置排名
	for i := range results {
		results[i].Rank = i + 1
	}

	return results
}

// fuseRRF Reciprocal Rank Fusion。
//
// RRF 公式: score = sum(1 / (k + rank_i))
// 其中 k 是常数（默认60），rank_i 是在第 i 个列表中的排名
func (r *GraphRAGRetriever) fuseRRF(vectorDocs []*types.Document, graphNodes []*graphdb.Node, options SearchOptions) []FusedResult {
	// 创建结果映射
	resultMap := make(map[string]*FusedResult)
	k := r.config.RRFConstant

	// 处理向量检索结果
	for i, doc := range vectorDocs {
		rank := float64(i + 1)
		score := 1.0 / (k + rank)

		key := r.getDocumentKey(doc)
		resultMap[key] = &FusedResult{
			Document:     doc,
			VectorScore:  score,
			GraphScore:   0.0,
			RelatedNodes: []*graphdb.Node{},
			Metadata:     make(map[string]interface{}),
		}
	}

	// 处理图检索结果
	for j, node := range graphNodes {
		rank := float64(j + 1)
		score := 1.0 / (k + rank)

		doc := r.nodeToDocument(node)
		key := r.getDocumentKey(doc)

		if existing, ok := resultMap[key]; ok {
			// 已存在，累加图分数
			existing.GraphScore = score
			existing.RelatedNodes = append(existing.RelatedNodes, node)
		} else {
			// 新文档
			resultMap[key] = &FusedResult{
				Document:     doc,
				VectorScore:  0.0,
				GraphScore:   score,
				RelatedNodes: []*graphdb.Node{node},
				Metadata:     make(map[string]interface{}),
			}
		}
	}

	// 计算 RRF 融合分数
	results := make([]FusedResult, 0, len(resultMap))
	for _, result := range resultMap {
		// RRF: 直接求和
		result.FusedScore = result.VectorScore + result.GraphScore
		results = append(results, *result)
	}

	// 按融合分数排序
	sort.Slice(results, func(i, j int) bool {
		return results[i].FusedScore > results[j].FusedScore
	})

	// 设置排名
	for i := range results {
		results[i].Rank = i + 1
	}

	return results
}

// fuseMax 取最大值融合。
func (r *GraphRAGRetriever) fuseMax(vectorDocs []*types.Document, graphNodes []*graphdb.Node, options SearchOptions) []FusedResult {
	resultMap := make(map[string]*FusedResult)

	// 处理向量检索结果
	for i, doc := range vectorDocs {
		vectorScore := 1.0 - float64(i)/float64(len(vectorDocs))
		key := r.getDocumentKey(doc)
		resultMap[key] = &FusedResult{
			Document:     doc,
			VectorScore:  vectorScore,
			GraphScore:   0.0,
			RelatedNodes: []*graphdb.Node{},
			Metadata:     make(map[string]interface{}),
		}
	}

	// 处理图检索结果
	for j, node := range graphNodes {
		graphScore := 1.0 - float64(j)/float64(len(graphNodes))
		doc := r.nodeToDocument(node)
		key := r.getDocumentKey(doc)

		if existing, ok := resultMap[key]; ok {
			existing.GraphScore = graphScore
			existing.RelatedNodes = append(existing.RelatedNodes, node)
		} else {
			resultMap[key] = &FusedResult{
				Document:     doc,
				VectorScore:  0.0,
				GraphScore:   graphScore,
				RelatedNodes: []*graphdb.Node{node},
				Metadata:     make(map[string]interface{}),
			}
		}
	}

	// 取最大值
	results := make([]FusedResult, 0, len(resultMap))
	for _, result := range resultMap {
		result.FusedScore = math.Max(result.VectorScore, result.GraphScore)
		results = append(results, *result)
	}

	// 排序
	sort.Slice(results, func(i, j int) bool {
		return results[i].FusedScore > results[j].FusedScore
	})

	for i := range results {
		results[i].Rank = i + 1
	}

	return results
}

// fuseMin 取最小值融合。
func (r *GraphRAGRetriever) fuseMin(vectorDocs []*types.Document, graphNodes []*graphdb.Node, options SearchOptions) []FusedResult {
	resultMap := make(map[string]*FusedResult)

	// 处理向量检索结果
	for i, doc := range vectorDocs {
		vectorScore := 1.0 - float64(i)/float64(len(vectorDocs))
		key := r.getDocumentKey(doc)
		resultMap[key] = &FusedResult{
			Document:     doc,
			VectorScore:  vectorScore,
			GraphScore:   0.0,
			RelatedNodes: []*graphdb.Node{},
			Metadata:     make(map[string]interface{}),
		}
	}

	// 处理图检索结果
	for j, node := range graphNodes {
		graphScore := 1.0 - float64(j)/float64(len(graphNodes))
		doc := r.nodeToDocument(node)
		key := r.getDocumentKey(doc)

		if existing, ok := resultMap[key]; ok {
			existing.GraphScore = graphScore
			existing.RelatedNodes = append(existing.RelatedNodes, node)
		} else {
			resultMap[key] = &FusedResult{
				Document:     doc,
				VectorScore:  0.0,
				GraphScore:   graphScore,
				RelatedNodes: []*graphdb.Node{node},
				Metadata:     make(map[string]interface{}),
			}
		}
	}

	// 取最小值（只对同时存在于两个结果中的文档）
	results := make([]FusedResult, 0, len(resultMap))
	for _, result := range resultMap {
		if result.VectorScore > 0 && result.GraphScore > 0 {
			result.FusedScore = math.Min(result.VectorScore, result.GraphScore)
		} else {
			// 只在一个结果中，使用该分数
			result.FusedScore = math.Max(result.VectorScore, result.GraphScore)
		}
		results = append(results, *result)
	}

	// 排序
	sort.Slice(results, func(i, j int) bool {
		return results[i].FusedScore > results[j].FusedScore
	})

	for i := range results {
		results[i].Rank = i + 1
	}

	return results
}

// getDocumentKey 获取文档唯一键。
func (r *GraphRAGRetriever) getDocumentKey(doc *types.Document) string {
	if doc.ID != "" {
		return doc.ID
	}

	// 使用内容的前100个字符作为键
	if len(doc.Content) > 100 {
		return doc.Content[:100]
	}

	return doc.Content
}

// ExplainScore 解释分数计算。
func ExplainScore(result FusedResult, strategy FusionStrategy) string {
	switch strategy {
	case FusionStrategyWeighted:
		return fmt.Sprintf("Weighted: %.3f (vector: %.3f, graph: %.3f)",
			result.FusedScore, result.VectorScore, result.GraphScore)
	case FusionStrategyRRF:
		return fmt.Sprintf("RRF: %.3f (vector: %.3f, graph: %.3f)",
			result.FusedScore, result.VectorScore, result.GraphScore)
	case FusionStrategyMax:
		return fmt.Sprintf("Max: %.3f (vector: %.3f, graph: %.3f)",
			result.FusedScore, result.VectorScore, result.GraphScore)
	case FusionStrategyMin:
		return fmt.Sprintf("Min: %.3f (vector: %.3f, graph: %.3f)",
			result.FusedScore, result.VectorScore, result.GraphScore)
	default:
		return fmt.Sprintf("Score: %.3f", result.FusedScore)
	}
}
