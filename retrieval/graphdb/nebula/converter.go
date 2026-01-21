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
// 自动遍历结果集的所有列，提取所有节点类型的值。
// 适用于包含节点数据的任何查询结果。
func (c *Converter) ResultSetToNodes(result *nebula.ResultSet) ([]*graphdb.Node, error) {
	if result == nil || !result.IsSucceed() {
		return nil, fmt.Errorf("invalid result set")
	}

	nodes := []*graphdb.Node{}
	nodeMap := make(map[string]*graphdb.Node) // 去重

	// 获取列名
	colNames := result.GetColNames()

	// 遍历每一行
	for i := 0; i < result.GetRowSize(); i++ {
		row, err := result.GetRowValuesByIndex(i)
		if err != nil {
			continue
		}

		// 遍历每一列，查找节点
		for j := 0; j < len(colNames); j++ {
			val, err := row.GetValueByIndex(j)
			if err != nil {
				continue
			}

			// 检查是否是节点
			if val.IsVertex() {
				node, err := val.AsNode()
				if err == nil {
					graphNode, err := c.VertexToNode(node)
					if err == nil {
						// 使用 ID 去重
						if _, exists := nodeMap[graphNode.ID]; !exists {
							nodeMap[graphNode.ID] = graphNode
							nodes = append(nodes, graphNode)
						}
					}
				}
			}
		}
	}

	return nodes, nil
}

// ResultSetToEdges 从 ResultSet 提取边列表
//
// 自动遍历结果集的所有列，提取所有边类型的值。
// 适用于包含边数据的任何查询结果。
func (c *Converter) ResultSetToEdges(result *nebula.ResultSet) ([]*graphdb.Edge, error) {
	if result == nil || !result.IsSucceed() {
		return nil, fmt.Errorf("invalid result set")
	}

	edges := []*graphdb.Edge{}
	edgeMap := make(map[string]*graphdb.Edge) // 去重

	// 获取列名
	colNames := result.GetColNames()

	// 遍历每一行
	for i := 0; i < result.GetRowSize(); i++ {
		row, err := result.GetRowValuesByIndex(i)
		if err != nil {
			continue
		}

		// 遍历每一列，查找边
		for j := 0; j < len(colNames); j++ {
			val, err := row.GetValueByIndex(j)
			if err != nil {
				continue
			}

			// 检查是否是边
			if val.IsEdge() {
				rel, err := val.AsRelationship()
				if err == nil {
					graphEdge, err := c.EdgeToGraphEdge(rel)
					if err == nil {
						// 使用 ID 去重
						if _, exists := edgeMap[graphEdge.ID]; !exists {
							edgeMap[graphEdge.ID] = graphEdge
							edges = append(edges, graphEdge)
						}
					}
				}
			}
		}
	}

	return edges, nil
}

// ResultSetToPaths 从 ResultSet 提取路径列表
//
// 自动遍历结果集的所有列，提取所有路径类型的值。
// 适用于 FIND SHORTEST PATH 等路径查询结果。
func (c *Converter) ResultSetToPaths(result *nebula.ResultSet) ([]*graphdb.Path, error) {
	if result == nil || !result.IsSucceed() {
		return nil, fmt.Errorf("invalid result set")
	}

	paths := []*graphdb.Path{}

	// 获取列名
	colNames := result.GetColNames()

	// 遍历每一行
	for i := 0; i < result.GetRowSize(); i++ {
		row, err := result.GetRowValuesByIndex(i)
		if err != nil {
			continue
		}

		// 遍历每一列，查找路径
		for j := 0; j < len(colNames); j++ {
			val, err := row.GetValueByIndex(j)
			if err != nil {
				continue
			}

			// 检查是否是路径
			if val.IsPath() {
				pathWrapper, err := val.AsPath()
				if err == nil {
					path, err := c.PathToGraphPath(pathWrapper)
					if err == nil {
						paths = append(paths, path)
					}
				}
			}
		}
	}

	return paths, nil
}

// ExtractFromResultSet 从 ResultSet 提取所有图元素
//
// 自动识别并提取节点、边和路径，返回统一的结果。
// 这是一个通用方法，适用于任何查询结果。
func (c *Converter) ExtractFromResultSet(result *nebula.ResultSet) (nodes []*graphdb.Node, edges []*graphdb.Edge, paths []*graphdb.Path, err error) {
	if result == nil || !result.IsSucceed() {
		return nil, nil, nil, fmt.Errorf("invalid result set")
	}

	nodeMap := make(map[string]*graphdb.Node)
	edgeMap := make(map[string]*graphdb.Edge)
	nodes = []*graphdb.Node{}
	edges = []*graphdb.Edge{}
	paths = []*graphdb.Path{}

	// 获取列名
	colNames := result.GetColNames()

	// 遍历每一行
	for i := 0; i < result.GetRowSize(); i++ {
		row, err := result.GetRowValuesByIndex(i)
		if err != nil {
			continue
		}

		// 遍历每一列
		for j := 0; j < len(colNames); j++ {
			val, err := row.GetValueByIndex(j)
			if err != nil {
				continue
			}

			// 提取节点
			if val.IsVertex() {
				node, err := val.AsNode()
				if err == nil {
					graphNode, err := c.VertexToNode(node)
					if err == nil {
						if _, exists := nodeMap[graphNode.ID]; !exists {
							nodeMap[graphNode.ID] = graphNode
							nodes = append(nodes, graphNode)
						}
					}
				}
			}

			// 提取边
			if val.IsEdge() {
				rel, err := val.AsRelationship()
				if err == nil {
					graphEdge, err := c.EdgeToGraphEdge(rel)
					if err == nil {
						if _, exists := edgeMap[graphEdge.ID]; !exists {
							edgeMap[graphEdge.ID] = graphEdge
							edges = append(edges, graphEdge)
						}
					}
				}
			}

			// 提取路径
			if val.IsPath() {
				pathWrapper, err := val.AsPath()
				if err == nil {
					path, err := c.PathToGraphPath(pathWrapper)
					if err == nil {
						paths = append(paths, path)
						// 同时提取路径中的节点和边
						for _, n := range path.Nodes {
							if _, exists := nodeMap[n.ID]; !exists {
								nodeMap[n.ID] = n
								nodes = append(nodes, n)
							}
						}
						for _, e := range path.Edges {
							if _, exists := edgeMap[e.ID]; !exists {
								edgeMap[e.ID] = e
								edges = append(edges, e)
							}
						}
					}
				}
			}
		}
	}

	return nodes, edges, paths, nil
}
