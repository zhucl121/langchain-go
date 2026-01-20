package graphrag

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/zhucl121/langchain-go/pkg/types"
	"github.com/zhucl121/langchain-go/retrieval/graphdb"
	"github.com/zhucl121/langchain-go/retrieval/graphdb/builder"
)

// GraphRAGRetriever GraphRAG 检索器。
//
// GraphRAGRetriever 结合向量检索和图遍历，实现混合检索。
type GraphRAGRetriever struct {
	config Config
	mu     sync.RWMutex
	stats  Statistics
}

// NewGraphRAGRetriever 创建 GraphRAG 检索器。
//
// 参数：
//   - config: 配置
//
// 返回：
//   - *GraphRAGRetriever: 检索器实例
//   - error: 错误
//
func NewGraphRAGRetriever(config Config) (*GraphRAGRetriever, error) {
	// 验证配置
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// 设置默认值
	if config.VectorWeight == 0 && config.GraphWeight == 0 {
		config.VectorWeight = 0.6
		config.GraphWeight = 0.4
	}

	if config.MaxTraverseDepth == 0 {
		config.MaxTraverseDepth = 2
	}

	if config.TopK == 0 {
		config.TopK = 10
	}

	if config.FusionStrategy == "" {
		config.FusionStrategy = FusionStrategyWeighted
	}

	if config.RerankStrategy == "" {
		config.RerankStrategy = RerankStrategyScore
	}

	if config.RRFConstant == 0 {
		config.RRFConstant = 60.0
	}

	if config.MMRLambda == 0 {
		config.MMRLambda = 0.5
	}

	return &GraphRAGRetriever{
		config: config,
		stats:  Statistics{},
	}, nil
}

// Search 执行检索。
//
// 参数：
//   - ctx: 上下文
//   - query: 查询文本
//   - opts: 检索选项（可选）
//
// 返回：
//   - []*types.Document: 检索结果
//   - error: 错误
//
func (r *GraphRAGRetriever) Search(ctx context.Context, query string, opts ...SearchOptions) ([]*types.Document, error) {
	startTime := time.Now()

	// 合并选项
	options := r.mergeOptions(opts...)

	// 重置统计
	r.resetStats()

	var docs []*types.Document
	var err error

	switch options.Mode {
	case SearchModeHybrid:
		docs, err = r.hybridSearch(ctx, query, options)
	case SearchModeVector:
		docs, err = r.vectorSearch(ctx, query, options.K)
	case SearchModeGraph:
		docs, err = r.graphSearch(ctx, query, options)
	default:
		return nil, fmt.Errorf("unsupported search mode: %s", options.Mode)
	}

	if err != nil {
		return nil, err
	}

	// 更新总耗时
	r.mu.Lock()
	r.stats.TotalTime = time.Since(startTime).Milliseconds()
	r.mu.Unlock()

	return docs, nil
}

// HybridSearch 混合检索（向量+图）。
func (r *GraphRAGRetriever) hybridSearch(ctx context.Context, query string, options SearchOptions) ([]*types.Document, error) {
	// 1. 向量检索
	vectorStart := time.Now()
	vectorDocs, err := r.vectorSearch(ctx, query, options.K*2) // 获取更多候选
	if err != nil {
		return nil, fmt.Errorf("vector search failed: %w", err)
	}
	r.mu.Lock()
	r.stats.VectorResultsCount = len(vectorDocs)
	r.stats.VectorSearchTime = time.Since(vectorStart).Milliseconds()
	r.mu.Unlock()

	// 2. 提取查询实体
	entities, err := r.extractQueryEntities(ctx, query)
	if err != nil {
		// 不阻塞流程，只记录错误
		fmt.Printf("Warning: entity extraction failed: %v\n", err)
		entities = []builder.Entity{}
	}

	r.mu.Lock()
	r.stats.EntitiesExtracted = len(entities)
	r.mu.Unlock()

	// 3. 图遍历检索
	graphStart := time.Now()
	graphNodes, err := r.graphTraverse(ctx, entities, options)
	if err != nil {
		// 不阻塞流程
		fmt.Printf("Warning: graph traverse failed: %v\n", err)
		graphNodes = []*graphdb.Node{}
	}

	r.mu.Lock()
	r.stats.GraphResultsCount = len(graphNodes)
	r.stats.NodesTraversed = len(graphNodes)
	r.stats.GraphSearchTime = time.Since(graphStart).Milliseconds()
	r.mu.Unlock()

	// 4. 融合结果
	fusionStart := time.Now()
	fusedResults := r.fuseResults(vectorDocs, graphNodes, options)
	r.mu.Lock()
	r.stats.FusedResultsCount = len(fusedResults)
	r.stats.FusionTime = time.Since(fusionStart).Milliseconds()
	r.mu.Unlock()

	// 5. 重排序
	rerankStart := time.Now()
	rankedResults := r.rerank(fusedResults, query, options)
	r.mu.Lock()
	r.stats.RerankTime = time.Since(rerankStart).Milliseconds()
	r.mu.Unlock()

	// 6. 上下文增强
	var finalDocs []*types.Document
	if options.EnableContextAugmentation {
		finalDocs = r.augmentContext(rankedResults, graphNodes)
	} else {
		finalDocs = make([]*types.Document, len(rankedResults))
		for i, result := range rankedResults {
			finalDocs[i] = result.Document
		}
	}

	// 7. 过滤和截断
	filteredDocs := r.filterAndTruncate(finalDocs, options)

	return filteredDocs, nil
}

// vectorSearch 向量检索。
func (r *GraphRAGRetriever) vectorSearch(ctx context.Context, query string, k int) ([]*types.Document, error) {
	docs, err := r.config.VectorStore.SimilaritySearch(ctx, query, k)
	if err != nil {
		return nil, err
	}

	// 确保 docs 中的 Document 是 types.Document 类型
	// loaders.Document 是 types.Document 的别名，所以可以直接使用
	return docs, nil
}

// graphSearch 纯图检索。
func (r *GraphRAGRetriever) graphSearch(ctx context.Context, query string, options SearchOptions) ([]*types.Document, error) {
	// 提取查询实体
	entities, err := r.extractQueryEntities(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("entity extraction failed: %w", err)
	}

	if len(entities) == 0 {
		return []*types.Document{}, nil
	}

	// 图遍历
	nodes, err := r.graphTraverse(ctx, entities, options)
	if err != nil {
		return nil, fmt.Errorf("graph traverse failed: %w", err)
	}

	// 转换为文档
	docs := make([]*types.Document, len(nodes))
	for i, node := range nodes {
		docs[i] = r.nodeToDocument(node)
	}

	return docs, nil
}

// extractQueryEntities 从查询中提取实体。
func (r *GraphRAGRetriever) extractQueryEntities(ctx context.Context, query string) ([]builder.Entity, error) {
	if r.config.EntityExtractor == nil {
		return []builder.Entity{}, nil
	}

	entities, err := r.config.EntityExtractor.Extract(ctx, query)
	if err != nil {
		return nil, err
	}

	return entities, nil
}

// graphTraverse 图遍历。
func (r *GraphRAGRetriever) graphTraverse(ctx context.Context, entities []builder.Entity, options SearchOptions) ([]*graphdb.Node, error) {
	if len(entities) == 0 {
		return []*graphdb.Node{}, nil
	}

	var allNodes []*graphdb.Node
	traversedIDs := make(map[string]bool)

	for _, entity := range entities {
		// 遍历每个实体
		result, err := r.config.GraphDB.Traverse(ctx, entity.ID, graphdb.TraverseOptions{
			MaxDepth:  options.MaxTraverseDepth,
			Direction: graphdb.DirectionBoth,
			Strategy:  graphdb.StrategyBFS,
			Limit:     options.K * 2,
		})
		if err != nil {
			// 继续处理其他实体
			continue
		}

		// 去重添加节点
		for _, node := range result.Nodes {
			if !traversedIDs[node.ID] {
				allNodes = append(allNodes, node)
				traversedIDs[node.ID] = true
			}
		}
	}

	return allNodes, nil
}

// mergeOptions 合并选项。
func (r *GraphRAGRetriever) mergeOptions(opts ...SearchOptions) SearchOptions {
	if len(opts) == 0 {
		return SearchOptions{
			Mode:                      SearchModeHybrid,
			K:                         r.config.TopK,
			VectorWeight:              r.config.VectorWeight,
			GraphWeight:               r.config.GraphWeight,
			MaxTraverseDepth:          r.config.MaxTraverseDepth,
			FusionStrategy:            r.config.FusionStrategy,
			RerankStrategy:            r.config.RerankStrategy,
			EnableContextAugmentation: r.config.EnableContextAugmentation,
			MinScore:                  r.config.MinScore,
			IncludeMetadata:           true,
		}
	}

	options := opts[0]

	// 使用配置中的默认值填充未设置的选项
	if options.K == 0 {
		options.K = r.config.TopK
	}
	if options.VectorWeight == 0 && options.GraphWeight == 0 {
		options.VectorWeight = r.config.VectorWeight
		options.GraphWeight = r.config.GraphWeight
	}
	if options.MaxTraverseDepth == 0 {
		options.MaxTraverseDepth = r.config.MaxTraverseDepth
	}
	if options.FusionStrategy == "" {
		options.FusionStrategy = r.config.FusionStrategy
	}
	if options.RerankStrategy == "" {
		options.RerankStrategy = r.config.RerankStrategy
	}

	return options
}

// resetStats 重置统计信息。
func (r *GraphRAGRetriever) resetStats() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.stats = Statistics{}
}

// GetStatistics 获取统计信息。
func (r *GraphRAGRetriever) GetStatistics() Statistics {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.stats
}

// nodeToDocument 将图节点转换为文档。
func (r *GraphRAGRetriever) nodeToDocument(node *graphdb.Node) *types.Document {
	// 构建内容
	content := fmt.Sprintf("%s: %s", node.Type, node.Label)

	// 添加描述
	if desc, ok := node.Properties["description"].(string); ok {
		content = content + "\n" + desc
	}

	// 创建文档
	doc := types.NewDocument(content, nil)
	doc.ID = node.ID

	// 添加元数据
	doc.AddMetadata("entity_id", node.ID)
	doc.AddMetadata("entity_type", node.Type)
	doc.AddMetadata("entity_label", node.Label)

	// 添加所有属性到元数据
	for k, v := range node.Properties {
		doc.AddMetadata(k, v)
	}

	return doc
}

// filterAndTruncate 过滤和截断结果。
func (r *GraphRAGRetriever) filterAndTruncate(docs []*types.Document, options SearchOptions) []*types.Document {
	// 应用最小分数过滤
	if options.MinScore > 0 {
		filtered := make([]*types.Document, 0, len(docs))
		for _, doc := range docs {
			if score, ok := doc.Metadata["fused_score"].(float64); ok && score >= options.MinScore {
				filtered = append(filtered, doc)
			} else if !ok {
				// 如果没有分数，保留文档
				filtered = append(filtered, doc)
			}
		}
		docs = filtered
	}

	// 截断到 K
	if len(docs) > options.K {
		docs = docs[:options.K]
	}

	return docs
}

// min 返回两个整数的最小值。
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// max 返回两个整数的最大值。
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
