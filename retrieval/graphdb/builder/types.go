package builder

import (
	"github.com/zhucl121/langchain-go/retrieval/graphdb"
)

// Entity 表示一个实体。
//
// Entity 是知识图谱的基本单元，代表现实世界中的对象、概念或事物。
type Entity struct {
	// ID 实体唯一标识符
	ID string

	// Type 实体类型（如 Person, Organization, Location, Concept）
	Type string

	// Name 实体名称
	Name string

	// Description 实体描述
	Description string

	// Properties 实体属性
	Properties map[string]interface{}

	// Metadata 元数据
	Metadata map[string]interface{}

	// Embedding 实体的向量表示（用于相似度搜索）
	Embedding []float32

	// SourceText 提取实体的原始文本
	SourceText string

	// SourceSpan 实体在原文中的位置 [start, end)
	SourceSpan [2]int

	// Confidence 提取置信度 (0-1)
	Confidence float64
}

// Relation 表示两个实体之间的关系。
//
// Relation 连接知识图谱中的实体，表达它们之间的语义关系。
type Relation struct {
	// ID 关系唯一标识符
	ID string

	// Type 关系类型（如 WORKS_FOR, LOCATED_IN, KNOWS）
	Type string

	// Description 关系描述
	Description string

	// Source 源实体 ID
	Source string

	// Target 目标实体 ID
	Target string

	// Properties 关系属性
	Properties map[string]interface{}

	// Metadata 元数据
	Metadata map[string]interface{}

	// Weight 关系权重 (0-1)
	Weight float64

	// Directed 是否为有向关系
	Directed bool

	// SourceText 提取关系的原始文本
	SourceText string

	// Confidence 提取置信度 (0-1)
	Confidence float64
}

// EntityType 预定义的实体类型。
const (
	EntityTypePerson       = "Person"
	EntityTypeOrganization = "Organization"
	EntityTypeLocation     = "Location"
	EntityTypeConcept      = "Concept"
	EntityTypeEvent        = "Event"
	EntityTypeProduct      = "Product"
	EntityTypeTechnology   = "Technology"
	EntityTypeOther        = "Other"
)

// RelationType 预定义的关系类型。
const (
	RelationTypeWorksFor    = "WORKS_FOR"
	RelationTypeLocatedIn   = "LOCATED_IN"
	RelationTypeKnows       = "KNOWS"
	RelationTypeFounded     = "FOUNDED"
	RelationTypeOwns        = "OWNS"
	RelationTypePartOf      = "PART_OF"
	RelationTypeRelatedTo   = "RELATED_TO"
	RelationTypeCreated     = "CREATED"
	RelationTypeInfluenced  = "INFLUENCED"
	RelationTypeOther       = "OTHER"
)

// EntitySchema 实体类型的 Schema 定义。
//
// EntitySchema 定义了特定类型实体的结构和约束。
type EntitySchema struct {
	// Type 实体类型
	Type string

	// Description 类型描述
	Description string

	// Properties 属性定义
	Properties map[string]PropertySchema

	// Required 必需的属性列表
	Required []string

	// Examples 示例实体
	Examples []Entity
}

// RelationSchema 关系类型的 Schema 定义。
//
// RelationSchema 定义了特定类型关系的结构和约束。
type RelationSchema struct {
	// Type 关系类型
	Type string

	// Description 类型描述
	Description string

	// SourceTypes 允许的源实体类型列表
	SourceTypes []string

	// TargetTypes 允许的目标实体类型列表
	TargetTypes []string

	// Properties 属性定义
	Properties map[string]PropertySchema

	// Directed 是否为有向关系
	Directed bool

	// Examples 示例关系
	Examples []Relation
}

// PropertySchema 属性的 Schema 定义。
type PropertySchema struct {
	// Type 属性类型（string, number, boolean, array, object）
	Type string

	// Description 属性描述
	Description string

	// Required 是否必需
	Required bool

	// DefaultValue 默认值
	DefaultValue interface{}

	// Enum 枚举值（如果适用）
	Enum []interface{}
}

// KnowledgeGraph 表示一个完整的知识图谱。
type KnowledgeGraph struct {
	// Entities 实体列表
	Entities []Entity

	// Relations 关系列表
	Relations []Relation

	// Metadata 图谱元数据
	Metadata map[string]interface{}
}

// ExtractionResult 表示一次提取操作的结果。
type ExtractionResult struct {
	// Entities 提取到的实体
	Entities []Entity

	// Relations 提取到的关系
	Relations []Relation

	// SourceText 原始文本
	SourceText string

	// Metadata 元数据
	Metadata map[string]interface{}
}

// ToNode 将 Entity 转换为 graphdb.Node。
func (e *Entity) ToNode() *graphdb.Node {
	node := &graphdb.Node{
		ID:    e.ID,
		Type:  e.Type,
		Label: e.Name,
		Properties: make(map[string]interface{}),
	}

	// 复制属性
	for k, v := range e.Properties {
		node.Properties[k] = v
	}

	// 添加额外字段
	if e.Description != "" {
		node.Properties["description"] = e.Description
	}
	if e.SourceText != "" {
		node.Properties["source_text"] = e.SourceText
	}
	if e.Confidence > 0 {
		node.Properties["confidence"] = e.Confidence
	}
	if len(e.Embedding) > 0 {
		node.Properties["embedding"] = e.Embedding
	}

	// 添加元数据
	for k, v := range e.Metadata {
		node.Properties["meta_"+k] = v
	}

	return node
}

// ToEdge 将 Relation 转换为 graphdb.Edge。
func (r *Relation) ToEdge() *graphdb.Edge {
	edge := &graphdb.Edge{
		ID:       r.ID,
		Source:   r.Source,
		Target:   r.Target,
		Type:     r.Type,
		Label:    r.Description,
		Directed: r.Directed,
		Weight:   r.Weight,
		Properties: make(map[string]interface{}),
	}

	// 复制属性
	for k, v := range r.Properties {
		edge.Properties[k] = v
	}

	// 添加额外字段
	if r.SourceText != "" {
		edge.Properties["source_text"] = r.SourceText
	}
	if r.Confidence > 0 {
		edge.Properties["confidence"] = r.Confidence
	}

	// 添加元数据
	for k, v := range r.Metadata {
		edge.Properties["meta_"+k] = v
	}

	return edge
}

// FromNode 从 graphdb.Node 创建 Entity。
func EntityFromNode(node *graphdb.Node) *Entity {
	entity := &Entity{
		ID:         node.ID,
		Type:       node.Type,
		Name:       node.Label,
		Properties: make(map[string]interface{}),
		Metadata:   make(map[string]interface{}),
	}

	// 提取标准字段
	for k, v := range node.Properties {
		switch k {
		case "description":
			if desc, ok := v.(string); ok {
				entity.Description = desc
			}
		case "source_text":
			if src, ok := v.(string); ok {
				entity.SourceText = src
			}
		case "confidence":
			if conf, ok := v.(float64); ok {
				entity.Confidence = conf
			}
		case "embedding":
			if emb, ok := v.([]float32); ok {
				entity.Embedding = emb
			}
		default:
			// 元数据
			if len(k) > 5 && k[:5] == "meta_" {
				entity.Metadata[k[5:]] = v
			} else {
				entity.Properties[k] = v
			}
		}
	}

	return entity
}

// FromEdge 从 graphdb.Edge 创建 Relation。
func RelationFromEdge(edge *graphdb.Edge) *Relation {
	relation := &Relation{
		ID:         edge.ID,
		Type:       edge.Type,
		Description: edge.Label,
		Source:     edge.Source,
		Target:     edge.Target,
		Directed:   edge.Directed,
		Weight:     edge.Weight,
		Properties: make(map[string]interface{}),
		Metadata:   make(map[string]interface{}),
	}

	// 提取标准字段
	for k, v := range edge.Properties {
		switch k {
		case "source_text":
			if src, ok := v.(string); ok {
				relation.SourceText = src
			}
		case "confidence":
			if conf, ok := v.(float64); ok {
				relation.Confidence = conf
			}
		default:
			// 元数据
			if len(k) > 5 && k[:5] == "meta_" {
				relation.Metadata[k[5:]] = v
			} else {
				relation.Properties[k] = v
			}
		}
	}

	return relation
}
