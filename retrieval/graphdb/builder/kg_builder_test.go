package builder_test

import (
	"context"
	"testing"

	"github.com/zhucl121/langchain-go/retrieval/graphdb/builder"
	"github.com/zhucl121/langchain-go/retrieval/graphdb/mock"
)

// MockEntityExtractor Mock 实体提取器（用于测试）
type MockEntityExtractor struct {
	entities []builder.Entity
	err      error
}

func (m *MockEntityExtractor) Extract(ctx context.Context, text string) ([]builder.Entity, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.entities, nil
}

func (m *MockEntityExtractor) ExtractWithSchema(ctx context.Context, text string, schema *builder.EntitySchema) ([]builder.Entity, error) {
	return m.Extract(ctx, text)
}

// MockRelationExtractor Mock 关系提取器（用于测试）
type MockRelationExtractor struct {
	relations []builder.Relation
	err       error
}

func (m *MockRelationExtractor) Extract(ctx context.Context, text string, entities []builder.Entity) ([]builder.Relation, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.relations, nil
}

func (m *MockRelationExtractor) ExtractWithSchema(ctx context.Context, text string, entities []builder.Entity, schema *builder.RelationSchema) ([]builder.Relation, error) {
	return m.Extract(ctx, text, entities)
}

func TestKGBuilder_Build(t *testing.T) {
	ctx := context.Background()

	// 准备 Mock 数据
	mockEntities := []builder.Entity{
		{
			ID:          "person-1",
			Type:        builder.EntityTypePerson,
			Name:        "John Smith",
			Description: "CEO",
			Confidence:  0.95,
		},
		{
			ID:          "org-1",
			Type:        builder.EntityTypeOrganization,
			Name:        "TechCorp",
			Description: "Technology company",
			Confidence:  0.9,
		},
	}

	mockRelations := []builder.Relation{
		{
			ID:          "rel-1",
			Type:        builder.RelationTypeWorksFor,
			Source:      "person-1",
			Target:      "org-1",
			Description: "works for",
			Directed:    true,
			Weight:      1.0,
			Confidence:  0.9,
		},
	}

	// 创建组件
	graphDB := mock.NewMockGraphDB()
	entityExtractor := &MockEntityExtractor{entities: mockEntities}
	relationExtractor := &MockRelationExtractor{relations: mockRelations}

	config := builder.KGBuilderConfig{
		GraphDB:           graphDB,
		EntityExtractor:   entityExtractor,
		RelationExtractor: relationExtractor,
		EnableEmbedding:   false,
	}

	kgBuilder, err := builder.NewKGBuilder(config)
	if err != nil {
		t.Fatalf("NewKGBuilder failed: %v", err)
	}

	// 测试 Build
	text := "John Smith is the CEO of TechCorp."
	kg, err := kgBuilder.Build(ctx, text)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	// 验证结果
	if len(kg.Entities) != 2 {
		t.Errorf("Expected 2 entities, got %d", len(kg.Entities))
	}

	if len(kg.Relations) != 1 {
		t.Errorf("Expected 1 relation, got %d", len(kg.Relations))
	}

	if kg.Metadata["entity_count"] != 2 {
		t.Errorf("Metadata entity_count mismatch")
	}

	if kg.Metadata["relation_count"] != 1 {
		t.Errorf("Metadata relation_count mismatch")
	}
}

func TestKGBuilder_BuildAndStore(t *testing.T) {
	ctx := context.Background()

	mockEntities := []builder.Entity{
		{
			ID:   "person-1",
			Type: builder.EntityTypePerson,
			Name: "Alice",
		},
	}

	mockRelations := []builder.Relation{}

	graphDB := mock.NewMockGraphDB()
	
	// 连接 Mock 数据库
	if err := graphDB.Connect(ctx); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	defer graphDB.Close()
	
	entityExtractor := &MockEntityExtractor{entities: mockEntities}
	relationExtractor := &MockRelationExtractor{relations: mockRelations}

	config := builder.KGBuilderConfig{
		GraphDB:           graphDB,
		EntityExtractor:   entityExtractor,
		RelationExtractor: relationExtractor,
	}

	kgBuilder, err := builder.NewKGBuilder(config)
	if err != nil {
		t.Fatalf("NewKGBuilder failed: %v", err)
	}

	// 测试 BuildAndStore
	text := "Alice is a person."
	kg, err := kgBuilder.BuildAndStore(ctx, text)
	if err != nil {
		t.Fatalf("BuildAndStore failed: %v", err)
	}

	// 验证图数据库中的数据
	node, err := graphDB.GetNode(ctx, "person-1")
	if err != nil {
		t.Fatalf("GetNode failed: %v", err)
	}

	if node.Label != "Alice" {
		t.Errorf("Expected node label 'Alice', got '%s'", node.Label)
	}

	if len(kg.Entities) != 1 {
		t.Errorf("Expected 1 entity, got %d", len(kg.Entities))
	}
}

func TestKGBuilder_BuildBatch(t *testing.T) {
	ctx := context.Background()

	mockEntities := []builder.Entity{
		{ID: "person-1", Type: builder.EntityTypePerson, Name: "Alice"},
	}

	graphDB := mock.NewMockGraphDB()
	entityExtractor := &MockEntityExtractor{entities: mockEntities}
	relationExtractor := &MockRelationExtractor{relations: []builder.Relation{}}

	config := builder.KGBuilderConfig{
		GraphDB:           graphDB,
		EntityExtractor:   entityExtractor,
		RelationExtractor: relationExtractor,
		MaxConcurrency:    2,
	}

	kgBuilder, err := builder.NewKGBuilder(config)
	if err != nil {
		t.Fatalf("NewKGBuilder failed: %v", err)
	}

	// 测试 BuildBatch
	texts := []string{
		"Alice is a person.",
		"Bob is a person.",
		"Charlie is a person.",
	}

	graphs, err := kgBuilder.BuildBatch(ctx, texts)
	if err != nil {
		t.Fatalf("BuildBatch failed: %v", err)
	}

	if len(graphs) != 3 {
		t.Errorf("Expected 3 graphs, got %d", len(graphs))
	}

	for i, kg := range graphs {
		if len(kg.Entities) != 1 {
			t.Errorf("Graph[%d]: expected 1 entity, got %d", i, len(kg.Entities))
		}
	}
}

func TestKGBuilder_Merge(t *testing.T) {
	ctx := context.Background()

	graphDB := mock.NewMockGraphDB()
	entityExtractor := &MockEntityExtractor{}
	relationExtractor := &MockRelationExtractor{}

	config := builder.KGBuilderConfig{
		GraphDB:           graphDB,
		EntityExtractor:   entityExtractor,
		RelationExtractor: relationExtractor,
	}

	kgBuilder, err := builder.NewKGBuilder(config)
	if err != nil {
		t.Fatalf("NewKGBuilder failed: %v", err)
	}

	// 准备多个知识图谱
	graphs := []*builder.KnowledgeGraph{
		{
			Entities: []builder.Entity{
				{ID: "person-1", Name: "Alice", Confidence: 0.9},
				{ID: "person-2", Name: "Bob", Confidence: 0.8},
			},
			Relations: []builder.Relation{
				{ID: "rel-1", Source: "person-1", Target: "person-2", Type: "KNOWS"},
			},
		},
		{
			Entities: []builder.Entity{
				{ID: "person-1", Name: "Alice", Confidence: 0.95}, // 重复实体（置信度更高）
				{ID: "person-3", Name: "Charlie", Confidence: 0.85},
			},
			Relations: []builder.Relation{
				{ID: "rel-2", Source: "person-1", Target: "person-3", Type: "KNOWS"},
			},
		},
	}

	// 测试 Merge
	mergedKG, err := kgBuilder.Merge(ctx, graphs)
	if err != nil {
		t.Fatalf("Merge failed: %v", err)
	}

	// 验证结果：3个唯一实体（person-1 被去重）
	if len(mergedKG.Entities) != 3 {
		t.Errorf("Expected 3 unique entities after merge, got %d", len(mergedKG.Entities))
	}

	// 验证关系：2个唯一关系
	if len(mergedKG.Relations) != 2 {
		t.Errorf("Expected 2 unique relations after merge, got %d", len(mergedKG.Relations))
	}

	// 验证去重后保留了高置信度的实体
	for _, entity := range mergedKG.Entities {
		if entity.ID == "person-1" {
			if entity.Confidence != 0.95 {
				t.Errorf("Expected merged entity confidence 0.95, got %f", entity.Confidence)
			}
		}
	}
}

func TestKGBuilder_WithEmbedding(t *testing.T) {
	ctx := context.Background()

	mockEntities := []builder.Entity{
		{ID: "person-1", Name: "Alice", Description: "A person"},
	}

	graphDB := mock.NewMockGraphDB()
	entityExtractor := &MockEntityExtractor{entities: mockEntities}
	relationExtractor := &MockRelationExtractor{relations: []builder.Relation{}}
	embedder := builder.NewMockEmbedder(128) // 128维向量

	config := builder.KGBuilderConfig{
		GraphDB:           graphDB,
		EntityExtractor:   entityExtractor,
		RelationExtractor: relationExtractor,
		Embedder:          embedder,
		EnableEmbedding:   true,
	}

	kgBuilder, err := builder.NewKGBuilder(config)
	if err != nil {
		t.Fatalf("NewKGBuilder failed: %v", err)
	}

	// 测试 Build with Embedding
	kg, err := kgBuilder.Build(ctx, "Alice is a person.")
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	// 验证实体有向量
	if len(kg.Entities) != 1 {
		t.Fatalf("Expected 1 entity, got %d", len(kg.Entities))
	}

	entity := kg.Entities[0]
	if len(entity.Embedding) != 128 {
		t.Errorf("Expected embedding dimension 128, got %d", len(entity.Embedding))
	}

	// 验证向量不全为零
	hasNonZero := false
	for _, v := range entity.Embedding {
		if v != 0 {
			hasNonZero = true
			break
		}
	}

	if !hasNonZero {
		t.Errorf("Embedding should not be all zeros")
	}
}

func TestEntity_ToNode(t *testing.T) {
	entity := builder.Entity{
		ID:          "person-1",
		Type:        builder.EntityTypePerson,
		Name:        "John",
		Description: "A person",
		Properties: map[string]interface{}{
			"age": 30,
		},
		Metadata: map[string]interface{}{
			"source": "test",
		},
		Embedding:  []float32{0.1, 0.2, 0.3},
		Confidence: 0.95,
	}

	node := entity.ToNode()

	if node.ID != "person-1" {
		t.Errorf("Node ID mismatch")
	}
	if node.Type != builder.EntityTypePerson {
		t.Errorf("Node Type mismatch")
	}
	if node.Label != "John" {
		t.Errorf("Node Label mismatch")
	}
	if node.Properties["age"] != 30 {
		t.Errorf("Node property 'age' mismatch")
	}
	if node.Properties["description"] != "A person" {
		t.Errorf("Node property 'description' mismatch")
	}
	if node.Properties["confidence"] != 0.95 {
		t.Errorf("Node property 'confidence' mismatch")
	}
}

func TestRelation_ToEdge(t *testing.T) {
	relation := builder.Relation{
		ID:          "rel-1",
		Type:        builder.RelationTypeWorksFor,
		Source:      "person-1",
		Target:      "org-1",
		Description: "works for",
		Properties: map[string]interface{}{
			"since": 2020,
		},
		Metadata: map[string]interface{}{
			"source": "test",
		},
		Directed:   true,
		Weight:     1.0,
		Confidence: 0.9,
	}

	edge := relation.ToEdge()

	if edge.ID != "rel-1" {
		t.Errorf("Edge ID mismatch")
	}
	if edge.Type != builder.RelationTypeWorksFor {
		t.Errorf("Edge Type mismatch")
	}
	if edge.Source != "person-1" {
		t.Errorf("Edge Source mismatch")
	}
	if edge.Target != "org-1" {
		t.Errorf("Edge Target mismatch")
	}
	if edge.Directed != true {
		t.Errorf("Edge Directed mismatch")
	}
	if edge.Weight != 1.0 {
		t.Errorf("Edge Weight mismatch")
	}
}
