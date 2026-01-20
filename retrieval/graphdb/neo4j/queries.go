package neo4j

import (
	"fmt"
	"strings"

	"github.com/zhucl121/langchain-go/retrieval/graphdb"
)

// buildNodeQuery 构建节点查询
func (d *Neo4jDriver) buildNodeQuery(filter graphdb.NodeFilter) (string, map[string]interface{}) {
	var conditions []string
	params := make(map[string]interface{})

	// 类型过滤
	if len(filter.Types) > 0 {
		var typeConditions []string
		for _, t := range filter.Types {
			typeConditions = append(typeConditions, fmt.Sprintf("'%s' IN labels(n)", t))
		}
		conditions = append(conditions, "("+strings.Join(typeConditions, " OR ")+")")
	}

	// 属性过滤
	i := 0
	for key, value := range filter.Properties {
		paramKey := fmt.Sprintf("prop%d", i)
		conditions = append(conditions, fmt.Sprintf("n.%s = $%s", key, paramKey))
		params[paramKey] = value
		i++
	}

	// 标签过滤
	if len(filter.Labels) > 0 {
		var labelConditions []string
		for i, label := range filter.Labels {
			paramKey := fmt.Sprintf("label%d", i)
			labelConditions = append(labelConditions, fmt.Sprintf("n.label = $%s", paramKey))
			params[paramKey] = label
		}
		conditions = append(conditions, "("+strings.Join(labelConditions, " OR ")+")")
	}

	// 构建查询
	query := "MATCH (n) "
	if len(conditions) > 0 {
		query += "WHERE " + strings.Join(conditions, " AND ") + " "
	}
	query += "RETURN n, labels(n) as types "

	// 分页
	if filter.Offset > 0 {
		query += fmt.Sprintf("SKIP %d ", filter.Offset)
	}
	if filter.Limit > 0 {
		query += fmt.Sprintf("LIMIT %d", filter.Limit)
	}

	return query, params
}

// buildEdgeQuery 构建边查询
func (d *Neo4jDriver) buildEdgeQuery(filter graphdb.EdgeFilter) (string, map[string]interface{}) {
	var conditions []string
	params := make(map[string]interface{})
	i := 0

	// 类型过滤
	var typePattern string
	if len(filter.Types) > 0 {
		typePattern = fmt.Sprintf("[r:%s]", strings.Join(filter.Types, "|"))
	} else {
		typePattern = "[r]"
	}

	// 源节点过滤
	if len(filter.SourceIDs) > 0 {
		conditions = append(conditions, "a.id IN $sourceIDs")
		params["sourceIDs"] = filter.SourceIDs
	}

	// 目标节点过滤
	if len(filter.TargetIDs) > 0 {
		conditions = append(conditions, "b.id IN $targetIDs")
		params["targetIDs"] = filter.TargetIDs
	}

	// 属性过滤
	for key, value := range filter.Properties {
		paramKey := fmt.Sprintf("prop%d", i)
		conditions = append(conditions, fmt.Sprintf("r.%s = $%s", key, paramKey))
		params[paramKey] = value
		i++
	}

	// 构建查询
	query := fmt.Sprintf("MATCH (a)-%s->(b) ", typePattern)
	if len(conditions) > 0 {
		query += "WHERE " + strings.Join(conditions, " AND ") + " "
	}
	query += "RETURN r, a.id as source, b.id as target, type(r) as edgeType "

	// 分页
	if filter.Offset > 0 {
		query += fmt.Sprintf("SKIP %d ", filter.Offset)
	}
	if filter.Limit > 0 {
		query += fmt.Sprintf("LIMIT %d", filter.Limit)
	}

	return query, params
}

// buildTraverseQuery 构建遍历查询
func (d *Neo4jDriver) buildTraverseQuery(startID string, opts graphdb.TraverseOptions) string {
	// 确定方向
	var relationshipPattern string
	switch opts.Direction {
	case graphdb.DirectionOutbound:
		relationshipPattern = "-[r*1..%d]->"
	case graphdb.DirectionInbound:
		relationshipPattern = "<-[r*1..%d]-"
	case graphdb.DirectionBoth:
		relationshipPattern = "-[r*1..%d]-"
	default:
		relationshipPattern = "-[r*1..%d]-"
	}

	// 添加边类型过滤
	if len(opts.EdgeTypes) > 0 {
		relationshipPattern = fmt.Sprintf("-[r:%s*1..%%d]-", strings.Join(opts.EdgeTypes, "|"))
		if opts.Direction == graphdb.DirectionInbound {
			relationshipPattern = "<" + relationshipPattern
		} else if opts.Direction == graphdb.DirectionOutbound {
			relationshipPattern += ">"
		}
	}

	relationshipPattern = fmt.Sprintf(relationshipPattern, opts.MaxDepth)

	// 构建查询
	query := fmt.Sprintf(`
		MATCH path = (start {id: $startID})%s(end)
		WHERE start.id <> end.id
	`, relationshipPattern)

	// 节点类型过滤
	if len(opts.NodeTypes) > 0 {
		query += fmt.Sprintf(" AND any(label IN labels(end) WHERE label IN %v)", opts.NodeTypes)
	}

	// 返回节点和边
	if opts.IncludePath {
		query += " RETURN path, nodes(path) as nodes, relationships(path) as rels"
	} else {
		query += " RETURN DISTINCT end as n, relationships(path) as rels"
	}

	// 限制
	if opts.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", opts.Limit)
	}

	return query
}

// buildPathQuery 构建路径查询
func (d *Neo4jDriver) buildPathQuery(opts graphdb.PathOptions) string {
	var query string

	switch opts.Algorithm {
	case graphdb.AlgorithmDijkstra:
		// 使用 Dijkstra 算法（考虑权重）
		query = fmt.Sprintf(`
			MATCH (start {id: $startID}), (end {id: $endID})
			CALL apoc.algo.dijkstra(start, end, 'RELATES_TO>', 'weight') YIELD path, weight
			RETURN path
			LIMIT %d
		`, maxInt(opts.Limit, 1))
	case graphdb.AlgorithmBFS:
		fallthrough
	default:
		// 使用 BFS（不考虑权重）
		query = fmt.Sprintf(`
			MATCH path = shortestPath((start {id: $startID})-[*1..%d]-(end {id: $endID}))
			RETURN path
			LIMIT %d
		`, opts.MaxDepth, maxInt(opts.Limit, 1))
	}

	return query
}

// maxInt 返回两个整数中的最大值
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
