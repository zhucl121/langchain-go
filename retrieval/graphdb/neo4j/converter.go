package neo4j

import (
	"context"
	"fmt"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/zhucl121/langchain-go/retrieval/graphdb"
)

// recordToNode 将 Neo4j 记录转换为节点
func (d *Neo4jDriver) recordToNode(record *neo4j.Record) (*graphdb.Node, error) {
	nodeValue, ok := record.Get("n")
	if !ok {
		return nil, fmt.Errorf("node not found in record")
	}

	neo4jNode, ok := nodeValue.(neo4j.Node)
	if !ok {
		return nil, fmt.Errorf("invalid node type")
	}

	// 获取节点类型
	typesValue, _ := record.Get("types")
	types, _ := typesValue.([]interface{})
	var nodeType string
	if len(types) > 0 {
		nodeType = fmt.Sprint(types[0])
	}

	// 转换属性
	properties := make(map[string]interface{})
	for key, value := range neo4jNode.Props {
		if key != "id" && key != "label" && key != "embedding" {
			properties[key] = value
		}
	}

	node := &graphdb.Node{
		ID:         fmt.Sprint(neo4jNode.Props["id"]),
		Type:       nodeType,
		Label:      fmt.Sprint(neo4jNode.Props["label"]),
		Properties: properties,
	}

	// 处理 embedding
	if embedding, ok := neo4jNode.Props["embedding"].([]interface{}); ok {
		node.Embedding = make([]float32, len(embedding))
		for i, v := range embedding {
			if f, ok := v.(float64); ok {
				node.Embedding[i] = float32(f)
			}
		}
	}

	return node, nil
}

// recordToEdge 将 Neo4j 记录转换为边
func (d *Neo4jDriver) recordToEdge(record *neo4j.Record) (*graphdb.Edge, error) {
	relValue, ok := record.Get("r")
	if !ok {
		return nil, fmt.Errorf("edge not found in record")
	}

	neo4jRel, ok := relValue.(neo4j.Relationship)
	if !ok {
		return nil, fmt.Errorf("invalid edge type")
	}

	source, _ := record.Get("source")
	target, _ := record.Get("target")
	edgeType, _ := record.Get("edgeType")

	// 转换属性
	properties := make(map[string]interface{})
	for key, value := range neo4jRel.Props {
		if key != "id" && key != "label" && key != "weight" {
			properties[key] = value
		}
	}

	edge := &graphdb.Edge{
		ID:         fmt.Sprint(neo4jRel.Props["id"]),
		Source:     fmt.Sprint(source),
		Target:     fmt.Sprint(target),
		Type:       fmt.Sprint(edgeType),
		Label:      fmt.Sprint(neo4jRel.Props["label"]),
		Properties: properties,
		Weight:     getFloat64(neo4jRel.Props["weight"]),
		Directed:   true, // Neo4j 关系总是有向的
	}

	return edge, nil
}

// recordToPath 将 Neo4j 记录转换为路径
func (d *Neo4jDriver) recordToPath(record *neo4j.Record) (*graphdb.Path, error) {
	pathValue, ok := record.Get("path")
	if !ok {
		return nil, fmt.Errorf("path not found in record")
	}

	neo4jPath, ok := pathValue.(neo4j.Path)
	if !ok {
		return nil, fmt.Errorf("invalid path type")
	}

	path := &graphdb.Path{
		Nodes:  []*graphdb.Node{},
		Edges:  []*graphdb.Edge{},
		Length: len(neo4jPath.Relationships),
		Cost:   0,
	}

	// 转换节点
	for _, neo4jNode := range neo4jPath.Nodes {
		node := d.neo4jNodeToNode(neo4jNode)
		path.Nodes = append(path.Nodes, node)
	}

	// 转换边
	for _, neo4jRel := range neo4jPath.Relationships {
		edge := d.neo4jRelToEdge(neo4jRel)
		path.Edges = append(path.Edges, edge)
		path.Cost += edge.Weight
	}

	return path, nil
}

// neo4jNodeToNode 将 Neo4j 节点转换为图节点
func (d *Neo4jDriver) neo4jNodeToNode(neo4jNode neo4j.Node) *graphdb.Node {
	properties := make(map[string]interface{})
	for key, value := range neo4jNode.Props {
		if key != "id" && key != "label" {
			properties[key] = value
		}
	}

	var nodeType string
	if len(neo4jNode.Labels) > 0 {
		nodeType = neo4jNode.Labels[0]
	}

	return &graphdb.Node{
		ID:         fmt.Sprint(neo4jNode.Props["id"]),
		Type:       nodeType,
		Label:      fmt.Sprint(neo4jNode.Props["label"]),
		Properties: properties,
	}
}

// neo4jRelToEdge 将 Neo4j 关系转换为边
func (d *Neo4jDriver) neo4jRelToEdge(neo4jRel neo4j.Relationship) *graphdb.Edge {
	properties := make(map[string]interface{})
	for key, value := range neo4jRel.Props {
		if key != "id" && key != "label" && key != "weight" {
			properties[key] = value
		}
	}

	return &graphdb.Edge{
		ID:         fmt.Sprint(neo4jRel.Props["id"]),
		Source:     fmt.Sprint(neo4jRel.StartId),
		Target:     fmt.Sprint(neo4jRel.EndId),
		Type:       neo4jRel.Type,
		Label:      fmt.Sprint(neo4jRel.Props["label"]),
		Properties: properties,
		Weight:     getFloat64(neo4jRel.Props["weight"]),
		Directed:   true,
	}
}

// parseTraverseResult 解析遍历结果
func (d *Neo4jDriver) parseTraverseResult(ctx context.Context, result neo4j.ResultWithContext) (*graphdb.TraverseResult, error) {
	traverseResult := &graphdb.TraverseResult{
		Nodes: []*graphdb.Node{},
		Edges: []*graphdb.Edge{},
		Paths: []*graphdb.Path{},
	}

	for result.Next(ctx) {
		record := result.Record()

		// 获取节点
		if nodeValue, ok := record.Get("n"); ok {
			if neo4jNode, ok := nodeValue.(neo4j.Node); ok {
				node := d.neo4jNodeToNode(neo4jNode)
				traverseResult.Nodes = append(traverseResult.Nodes, node)
			}
		}

		// 获取边
		if relValue, ok := record.Get("r"); ok {
			if neo4jRel, ok := relValue.(neo4j.Relationship); ok {
				edge := d.neo4jRelToEdge(neo4jRel)
				traverseResult.Edges = append(traverseResult.Edges, edge)
			}
		}

		// 获取路径
		if pathValue, ok := record.Get("path"); ok {
			if neo4jPath, ok := pathValue.(neo4j.Path); ok {
				path := &graphdb.Path{
					Nodes: []*graphdb.Node{},
					Edges: []*graphdb.Edge{},
				}
				for _, n := range neo4jPath.Nodes {
					path.Nodes = append(path.Nodes, d.neo4jNodeToNode(n))
				}
				for _, r := range neo4jPath.Relationships {
					edge := d.neo4jRelToEdge(r)
					path.Edges = append(path.Edges, edge)
					path.Cost += edge.Weight
				}
				path.Length = len(path.Edges)
				traverseResult.Paths = append(traverseResult.Paths, path)
			}
		}
	}

	return traverseResult, nil
}

// getFloat64 安全地获取 float64 值
func getFloat64(value interface{}) float64 {
	switch v := value.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	default:
		return 0
	}
}
