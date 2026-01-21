package nebula

import (
	"fmt"

	nebula "github.com/vesoft-inc/nebula-go/v3"
	"github.com/zhucl121/langchain-go/retrieval/graphdb"
)

// Converter NebulaGraph 类型转换器
//
// 负责在 NebulaGraph 类型和 graphdb 通用类型之间转换。
// 注意：NebulaGraph Go 客户端的某些类型转换比较复杂，这里提供基础实现。
type Converter struct{}

// NewConverter 创建转换器
func NewConverter() *Converter {
	return &Converter{}
}

// VertexToNode 将 NebulaGraph Node 转换为 graphdb.Node
func (c *Converter) VertexToNode(vertex *nebula.Node) (*graphdb.Node, error) {
	if vertex == nil {
		return nil, fmt.Errorf("vertex is nil")
	}

	id, err := vertex.GetID().AsString()
	if err != nil {
		return nil, fmt.Errorf("failed to get vertex ID: %w", err)
	}

	node := &graphdb.Node{
		ID:         id,
		Properties: make(map[string]interface{}),
	}

	// 获取 tags（节点类型）
	tags := vertex.GetTags()
	if len(tags) > 0 {
		node.Type = tags[0]

		// 获取第一个 tag 的属性
		props, err := vertex.Properties(tags[0])
		if err == nil && props != nil {
			node.Properties = c.convertProperties(props)

			// 尝试从属性中获取 label
			if name, ok := node.Properties["name"].(string); ok {
				node.Label = name
			} else if label, ok := node.Properties["label"].(string); ok {
				node.Label = label
			}
		}
	}

	return node, nil
}

// EdgeToGraphEdge 将 NebulaGraph Relationship 转换为 graphdb.Edge
func (c *Converter) EdgeToGraphEdge(edge *nebula.Relationship) (*graphdb.Edge, error) {
	if edge == nil {
		return nil, fmt.Errorf("edge is nil")
	}

	srcID, err := edge.GetSrcVertexID().AsString()
	if err != nil {
		return nil, fmt.Errorf("failed to get source ID: %w", err)
	}

	dstID, err := edge.GetDstVertexID().AsString()
	if err != nil {
		return nil, fmt.Errorf("failed to get target ID: %w", err)
	}

	graphEdge := &graphdb.Edge{
		ID:         fmt.Sprintf("%s-%s-%s", srcID, edge.GetEdgeName(), dstID),
		Source:     srcID,
		Target:     dstID,
		Type:       edge.GetEdgeName(),
		Directed:   true, // NebulaGraph 的边默认是有向的
		Properties: c.convertProperties(edge.Properties()),
	}

	// 尝试从属性中获取权重
	if weight, ok := graphEdge.Properties["weight"].(float64); ok {
		graphEdge.Weight = weight
	} else if weight, ok := graphEdge.Properties["weight"].(int64); ok {
		graphEdge.Weight = float64(weight)
	}

	return graphEdge, nil
}

// PathToGraphPath 将 NebulaGraph PathWrapper 转换为 graphdb.Path
func (c *Converter) PathToGraphPath(pathWrapper *nebula.PathWrapper) (*graphdb.Path, error) {
	if pathWrapper == nil {
		return nil, fmt.Errorf("path is nil")
	}

	path := &graphdb.Path{
		Nodes:  []*graphdb.Node{},
		Edges:  []*graphdb.Edge{},
		Length: 0,
		Cost:   0,
	}

	// 获取节点
	nodes := pathWrapper.GetNodes()
	for _, node := range nodes {
		graphNode, err := c.VertexToNode(node)
		if err != nil {
			continue
		}
		path.Nodes = append(path.Nodes, graphNode)
	}

	// 获取边
	relationships := pathWrapper.GetRelationships()
	for _, rel := range relationships {
		graphEdge, err := c.EdgeToGraphEdge(rel)
		if err != nil {
			continue
		}
		path.Edges = append(path.Edges, graphEdge)
		path.Cost += graphEdge.Weight
	}

	path.Length = len(path.Edges)

	return path, nil
}

// convertProperties 转换属性
func (c *Converter) convertProperties(props map[string]*nebula.ValueWrapper) map[string]interface{} {
	result := make(map[string]interface{})

	for key, valWrapper := range props {
		if valWrapper == nil {
			continue
		}

		val, err := c.convertValue(valWrapper)
		if err == nil {
			result[key] = val
		}
	}

	return result
}

// convertValue 转换单个值
func (c *Converter) convertValue(valWrapper *nebula.ValueWrapper) (interface{}, error) {
	if valWrapper == nil {
		return nil, fmt.Errorf("value wrapper is nil")
	}

	// 使用 ValueWrapper 的方法
	if valWrapper.IsNull() {
		return nil, nil
	}

	if valWrapper.IsBool() {
		return valWrapper.AsBool()
	}

	if valWrapper.IsInt() {
		return valWrapper.AsInt()
	}

	if valWrapper.IsFloat() {
		return valWrapper.AsFloat()
	}

	if valWrapper.IsString() {
		return valWrapper.AsString()
	}

	if valWrapper.IsList() {
		list, err := valWrapper.AsList()
		if err != nil {
			return nil, err
		}
		result := make([]interface{}, len(list))
		for i, item := range list {
			v, _ := c.convertValue(&item)
			result[i] = v
		}
		return result, nil
	}

	if valWrapper.IsMap() {
		m, err := valWrapper.AsMap()
		if err != nil {
			return nil, err
		}
		result := make(map[string]interface{})
		for key, item := range m {
			v, _ := c.convertValue(&item)
			result[key] = v
		}
		return result, nil
	}

	// 其他类型转为字符串
	return valWrapper.String(), nil
}

// ResultSetToNodes 从 ResultSet 提取节点列表
//
// 注意：这是一个简化实现，假设结果集的列包含节点数据。
// 实际使用时可能需要根据查询结果的具体结构调整。
func (c *Converter) ResultSetToNodes(result *nebula.ResultSet) ([]*graphdb.Node, error) {
	if result == nil || !result.IsSucceed() {
		return nil, fmt.Errorf("invalid result set")
	}

	nodes := []*graphdb.Node{}

	// 遍历每一行
	for i := 0; i < result.GetRowSize(); i++ {
		row, err := result.GetRowValuesByIndex(i)
		if err != nil {
			continue
		}

		// 尝试提取节点（这里假设结果集的第一列是节点）
		// 实际实现可能需要根据查询的 YIELD 语句调整
		val, err := row.GetValueByIndex(0)
		if err == nil {
			// TODO: 根据实际 NebulaGraph 客户端 API 实现节点提取
			// 目前 ValueWrapper 的 API 不太清晰，这里返回一个简化实现
			_ = val
		}
	}

	return nodes, nil
}

// ResultSetToEdges 从 ResultSet 提取边列表
func (c *Converter) ResultSetToEdges(result *nebula.ResultSet) ([]*graphdb.Edge, error) {
	if result == nil || !result.IsSucceed() {
		return nil, fmt.Errorf("invalid result set")
	}

	edges := []*graphdb.Edge{}

	// 遍历每一行
	for i := 0; i < result.GetRowSize(); i++ {
		row, err := result.GetRowValuesByIndex(i)
		if err != nil {
			continue
		}

		// TODO: 实现边提取
		_ = row
	}

	return edges, nil
}

// ResultSetToPaths 从 ResultSet 提取路径列表
func (c *Converter) ResultSetToPaths(result *nebula.ResultSet) ([]*graphdb.Path, error) {
	if result == nil || !result.IsSucceed() {
		return nil, fmt.Errorf("invalid result set")
	}

	paths := []*graphdb.Path{}

	// 遍历每一行
	for i := 0; i < result.GetRowSize(); i++ {
		row, err := result.GetRowValuesByIndex(i)
		if err != nil {
			continue
		}

		// TODO: 实现路径提取
		_ = row
	}

	return paths, nil
}
