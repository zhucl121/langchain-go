package builder

import (
	"context"
	"fmt"
	"sync"

	"github.com/zhucl121/langchain-go/retrieval/graphdb"
)

// StandardKGBuilder 标准知识图谱构建器实现。
type StandardKGBuilder struct {
	config KGBuilderConfig
	mu     sync.RWMutex
}

// NewKGBuilder 创建知识图谱构建器。
//
// 参数：
//   - config: 构建器配置
//
// 返回：
//   - *StandardKGBuilder: 构建器实例
//   - error: 错误
//
func NewKGBuilder(config KGBuilderConfig) (*StandardKGBuilder, error) {
	// 验证必需的配置
	if config.GraphDB == nil {
		return nil, fmt.Errorf("GraphDB is required")
	}
	if config.EntityExtractor == nil {
		return nil, fmt.Errorf("EntityExtractor is required")
	}
	if config.RelationExtractor == nil {
		return nil, fmt.Errorf("RelationExtractor is required")
	}

	// 设置默认值
	if config.BatchSize == 0 {
		config.BatchSize = 10
	}
	if config.MaxConcurrency == 0 {
		config.MaxConcurrency = 5
	}

	return &StandardKGBuilder{
		config: config,
	}, nil
}

// Build 从文本构建知识图谱。
func (b *StandardKGBuilder) Build(ctx context.Context, text string) (*KnowledgeGraph, error) {
	// 1. 提取实体
	entities, err := b.config.EntityExtractor.Extract(ctx, text)
	if err != nil {
		return nil, fmt.Errorf("entity extraction failed: %w", err)
	}

	if len(entities) == 0 {
		return &KnowledgeGraph{
			Entities:  []Entity{},
			Relations: []Relation{},
			Metadata:  make(map[string]interface{}),
		}, nil
	}

	// 2. 实体消歧（如果启用）
	if b.config.EnableDisambiguation && b.config.Disambiguator != nil {
		entities, _, err = b.config.Disambiguator.Disambiguate(ctx, entities)
		if err != nil {
			return nil, fmt.Errorf("entity disambiguation failed: %w", err)
		}
	}

	// 3. 向量化实体（如果启用）
	if b.config.EnableEmbedding && b.config.Embedder != nil {
		if err := b.embedEntities(ctx, entities); err != nil {
			return nil, fmt.Errorf("entity embedding failed: %w", err)
		}
	}

	// 4. 提取关系
	relations, err := b.config.RelationExtractor.Extract(ctx, text, entities)
	if err != nil {
		return nil, fmt.Errorf("relation extraction failed: %w", err)
	}

	// 5. 构建知识图谱
	kg := &KnowledgeGraph{
		Entities:  entities,
		Relations: relations,
		Metadata: map[string]interface{}{
			"source_text":       text,
			"entity_count":      len(entities),
			"relation_count":    len(relations),
			"embedding_enabled": b.config.EnableEmbedding,
		},
	}

	// 6. 验证（如果启用）
	if b.config.EnableValidation && b.config.Validator != nil {
		validationErrors, err := b.config.Validator.ValidateGraph(ctx, kg)
		if err != nil {
			return nil, fmt.Errorf("validation failed: %w", err)
		}

		if len(validationErrors) > 0 {
			// 记录验证错误但不阻止构建
			kg.Metadata["validation_errors"] = validationErrors
		}
	}

	return kg, nil
}

// BuildAndStore 构建知识图谱并存储到图数据库。
func (b *StandardKGBuilder) BuildAndStore(ctx context.Context, text string) (*KnowledgeGraph, error) {
	// 构建知识图谱
	kg, err := b.Build(ctx, text)
	if err != nil {
		return nil, err
	}

	// 存储到图数据库
	if err := b.storeGraph(ctx, kg); err != nil {
		return nil, fmt.Errorf("failed to store graph: %w", err)
	}

	return kg, nil
}

// BuildBatch 批量构建知识图谱。
func (b *StandardKGBuilder) BuildBatch(ctx context.Context, texts []string) ([]*KnowledgeGraph, error) {
	if len(texts) == 0 {
		return []*KnowledgeGraph{}, nil
	}

	results := make([]*KnowledgeGraph, len(texts))
	errors := make([]error, len(texts))

	// 使用信号量限制并发
	sem := make(chan struct{}, b.config.MaxConcurrency)
	var wg sync.WaitGroup

	for i, text := range texts {
		wg.Add(1)
		go func(idx int, txt string) {
			defer wg.Done()

			// 获取信号量
			sem <- struct{}{}
			defer func() { <-sem }()

			// 构建知识图谱
			kg, err := b.Build(ctx, txt)
			if err != nil {
				errors[idx] = err
				return
			}

			results[idx] = kg
		}(i, text)
	}

	wg.Wait()

	// 检查是否有错误
	for i, err := range errors {
		if err != nil {
			return nil, fmt.Errorf("build failed for text[%d]: %w", i, err)
		}
	}

	return results, nil
}

// Merge 合并多个知识图谱。
func (b *StandardKGBuilder) Merge(ctx context.Context, graphs []*KnowledgeGraph) (*KnowledgeGraph, error) {
	if len(graphs) == 0 {
		return &KnowledgeGraph{
			Entities:  []Entity{},
			Relations: []Relation{},
			Metadata:  make(map[string]interface{}),
		}, nil
	}

	if len(graphs) == 1 {
		return graphs[0], nil
	}

	// 合并所有实体和关系
	allEntities := make([]Entity, 0)
	allRelations := make([]Relation, 0)

	for _, kg := range graphs {
		allEntities = append(allEntities, kg.Entities...)
		allRelations = append(allRelations, kg.Relations...)
	}

	// 实体去重（基于 ID）
	entityMap := make(map[string]Entity)
	for _, entity := range allEntities {
		if existing, ok := entityMap[entity.ID]; ok {
			// 合并属性（简单策略：保留置信度更高的）
			if entity.Confidence > existing.Confidence {
				entityMap[entity.ID] = entity
			}
		} else {
			entityMap[entity.ID] = entity
		}
	}

	// 关系去重（基于 Source-Type-Target）
	relationKey := func(r Relation) string {
		return fmt.Sprintf("%s-%s-%s", r.Source, r.Type, r.Target)
	}

	relationMap := make(map[string]Relation)
	for _, relation := range allRelations {
		key := relationKey(relation)
		if existing, ok := relationMap[key]; ok {
			// 合并权重（取平均）
			if relation.Confidence > existing.Confidence {
				relationMap[key] = relation
			}
		} else {
			relationMap[key] = relation
		}
	}

	// 转换为切片
	mergedEntities := make([]Entity, 0, len(entityMap))
	for _, entity := range entityMap {
		mergedEntities = append(mergedEntities, entity)
	}

	mergedRelations := make([]Relation, 0, len(relationMap))
	for _, relation := range relationMap {
		mergedRelations = append(mergedRelations, relation)
	}

	mergedKG := &KnowledgeGraph{
		Entities:  mergedEntities,
		Relations: mergedRelations,
		Metadata: map[string]interface{}{
			"merged_from":    len(graphs),
			"entity_count":   len(mergedEntities),
			"relation_count": len(mergedRelations),
		},
	}

	return mergedKG, nil
}

// UpdateIncremental 增量更新知识图谱。
func (b *StandardKGBuilder) UpdateIncremental(ctx context.Context, text string) (*ExtractionResult, error) {
	// 提取新的实体和关系
	kg, err := b.Build(ctx, text)
	if err != nil {
		return nil, err
	}

	// 存储到图数据库
	if err := b.storeGraph(ctx, kg); err != nil {
		return nil, fmt.Errorf("failed to store graph: %w", err)
	}

	result := &ExtractionResult{
		Entities:   kg.Entities,
		Relations:  kg.Relations,
		SourceText: text,
		Metadata:   kg.Metadata,
	}

	return result, nil
}

// embedEntities 为实体生成向量。
func (b *StandardKGBuilder) embedEntities(ctx context.Context, entities []Entity) error {
	if len(entities) == 0 {
		return nil
	}

	// 准备文本
	texts := make([]string, len(entities))
	for i, entity := range entities {
		// 使用实体名称和描述组合作为嵌入文本
		text := entity.Name
		if entity.Description != "" {
			text = text + ": " + entity.Description
		}
		texts[i] = text
	}

	// 批量生成向量
	embeddings, err := b.config.Embedder.EmbedBatch(ctx, texts)
	if err != nil {
		return err
	}

	// 分配向量
	for i := range entities {
		if i < len(embeddings) {
			entities[i].Embedding = embeddings[i]
		}
	}

	return nil
}

// storeGraph 将知识图谱存储到图数据库。
func (b *StandardKGBuilder) storeGraph(ctx context.Context, kg *KnowledgeGraph) error {
	// 1. 批量添加节点
	nodes := make([]*graphdb.Node, len(kg.Entities))
	for i, entity := range kg.Entities {
		nodes[i] = entity.ToNode()
	}

	if len(nodes) > 0 {
		if err := b.config.GraphDB.BatchAddNodes(ctx, nodes); err != nil {
			return fmt.Errorf("failed to add nodes: %w", err)
		}
	}

	// 2. 批量添加边
	edges := make([]*graphdb.Edge, len(kg.Relations))
	for i, relation := range kg.Relations {
		edges[i] = relation.ToEdge()
	}

	if len(edges) > 0 {
		if err := b.config.GraphDB.BatchAddEdges(ctx, edges); err != nil {
			return fmt.Errorf("failed to add edges: %w", err)
		}
	}

	return nil
}
