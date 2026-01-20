package graphrag

import (
	"fmt"
	"strings"

	"github.com/zhucl121/langchain-go/pkg/types"
	"github.com/zhucl121/langchain-go/retrieval/graphdb"
)

// augmentContext 增强上下文。
//
// 为每个文档添加相关的图信息
func (r *GraphRAGRetriever) augmentContext(results []FusedResult, graphNodes []*graphdb.Node) []*types.Document {
	docs := make([]*types.Document, len(results))

	for i, result := range results {
		doc := result.Document.Clone()

		// 添加融合分数
		doc.AddMetadata("fused_score", result.FusedScore)
		doc.AddMetadata("vector_score", result.VectorScore)
		doc.AddMetadata("graph_score", result.GraphScore)
		doc.AddMetadata("rank", result.Rank)

		// 如果有相关节点，添加图上下文
		if len(result.RelatedNodes) > 0 {
			contextInfo := r.buildContextInfo(result, graphNodes)

			// 添加相关实体
			if len(contextInfo.RelatedEntities) > 0 {
				doc.AddMetadata("related_entities", contextInfo.RelatedEntities)
			}

			// 添加关系路径
			if len(contextInfo.RelationshipPaths) > 0 {
				doc.AddMetadata("relationship_paths", contextInfo.RelationshipPaths)
			}

			// 添加邻居数量
			doc.AddMetadata("neighbor_count", contextInfo.NeighborCount)

			// 添加图深度
			doc.AddMetadata("graph_depth", contextInfo.GraphDepth)

			// 如果有额外上下文，追加到内容中
			if contextInfo.AdditionalContext != "" {
				doc.Content = doc.Content + "\n\n" + contextInfo.AdditionalContext
			}
		}

		docs[i] = doc
	}

	return docs
}

// buildContextInfo 构建上下文信息。
func (r *GraphRAGRetriever) buildContextInfo(result FusedResult, allNodes []*graphdb.Node) ContextInfo {
	info := ContextInfo{
		RelatedEntities:   []string{},
		RelationshipPaths: []string{},
		NeighborCount:     len(result.RelatedNodes),
		GraphDepth:        0,
	}

	if len(result.RelatedNodes) == 0 {
		return info
	}

	// 提取相关实体名称
	entityNames := make([]string, 0, len(result.RelatedNodes))
	for _, node := range result.RelatedNodes {
		if node.Label != "" {
			entityNames = append(entityNames, fmt.Sprintf("%s (%s)", node.Label, node.Type))
		}
	}
	info.RelatedEntities = entityNames

	// 构建额外上下文文本
	if len(entityNames) > 0 {
		info.AdditionalContext = fmt.Sprintf("Related Entities: %s",
			strings.Join(entityNames, ", "))
	}

	// 计算平均深度（简化）
	info.GraphDepth = 1 // 简化处理，实际应该从遍历结果获取

	return info
}

// EnhanceWithGraphStructure 使用图结构增强文档。
//
// 参数：
//   - doc: 原始文档
//   - node: 对应的图节点
//   - neighbors: 邻居节点列表
//
// 返回：
//   - *types.Document: 增强后的文档
//
func EnhanceWithGraphStructure(doc *types.Document, node *graphdb.Node, neighbors []*graphdb.Node) *types.Document {
	enhanced := doc.Clone()

	// 添加节点信息
	if node != nil {
		enhanced.AddMetadata("node_id", node.ID)
		enhanced.AddMetadata("node_type", node.Type)
		enhanced.AddMetadata("node_label", node.Label)

		// 添加节点属性
		for k, v := range node.Properties {
			enhanced.AddMetadata(fmt.Sprintf("node_%s", k), v)
		}
	}

	// 添加邻居信息
	if len(neighbors) > 0 {
		neighborLabels := make([]string, len(neighbors))
		for i, n := range neighbors {
			neighborLabels[i] = n.Label
		}

		enhanced.AddMetadata("neighbors", neighborLabels)
		enhanced.AddMetadata("neighbor_count", len(neighbors))

		// 追加邻居上下文到内容
		contextText := fmt.Sprintf("\n\nConnected to: %s", strings.Join(neighborLabels, ", "))
		enhanced.Content = enhanced.Content + contextText
	}

	return enhanced
}

// ExtractGraphContext 从图中提取上下文。
//
// 参数：
//   - entityID: 实体 ID
//   - graphDB: 图数据库
//   - maxDepth: 最大深度
//
// 返回：
//   - string: 上下文文本
//   - error: 错误
//
func ExtractGraphContext(entityID string, graphDB graphdb.GraphDB, maxDepth int) (string, error) {
	// 这个函数可以在外部使用，提供更灵活的上下文提取
	// TODO: 实现完整的图上下文提取逻辑

	return "", nil
}

// FormatContextForLLM 格式化上下文用于 LLM。
//
// 参数：
//   - docs: 文档列表
//   - includeMetadata: 是否包含元数据
//
// 返回：
//   - string: 格式化后的上下文
//
func FormatContextForLLM(docs []*types.Document, includeMetadata bool) string {
	var builder strings.Builder

	builder.WriteString("Context:\n\n")

	for i, doc := range docs {
		builder.WriteString(fmt.Sprintf("Document %d:\n", i+1))
		builder.WriteString(doc.Content)
		builder.WriteString("\n")

		if includeMetadata && len(doc.Metadata) > 0 {
			builder.WriteString("Metadata:\n")

			// 打印关键元数据
			if entities, ok := doc.Metadata["related_entities"].([]string); ok {
				builder.WriteString(fmt.Sprintf("  Related Entities: %s\n", strings.Join(entities, ", ")))
			}

			if score, ok := doc.Metadata["fused_score"].(float64); ok {
				builder.WriteString(fmt.Sprintf("  Relevance Score: %.3f\n", score))
			}
		}

		builder.WriteString("\n---\n\n")
	}

	return builder.String()
}
