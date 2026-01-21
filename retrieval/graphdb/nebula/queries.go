package nebula

import (
	"fmt"
	"strings"
)

// QueryBuilder nGQL 查询构建器
//
// QueryBuilder 用于构建 NebulaGraph 的 nGQL 查询语句。
type QueryBuilder struct {
	space string
}

// NewQueryBuilder 创建查询构建器
func NewQueryBuilder(space string) *QueryBuilder {
	return &QueryBuilder{space: space}
}

// InsertVertex 构建插入节点查询
//
// nGQL 示例:
//
//	INSERT VERTEX Person(name, age) VALUES "person-1":("John", 30)
//
func (qb *QueryBuilder) InsertVertex(id string, tag string, properties map[string]interface{}) string {
	if len(properties) == 0 {
		return fmt.Sprintf("INSERT VERTEX %s() VALUES \"%s\":()", tag, id)
	}

	// 属性名称和值
	propNames := make([]string, 0, len(properties))
	propValues := make([]string, 0, len(properties))

	for key, value := range properties {
		propNames = append(propNames, key)
		propValues = append(propValues, formatValue(value))
	}

	return fmt.Sprintf("INSERT VERTEX %s(%s) VALUES \"%s\":(%s)",
		tag,
		strings.Join(propNames, ", "),
		id,
		strings.Join(propValues, ", "))
}

// FetchVertex 构建获取节点查询
//
// nGQL 示例:
//
//	FETCH PROP ON Person "person-1" YIELD properties(vertex)
//
func (qb *QueryBuilder) FetchVertex(id string, tag string) string {
	if tag == "" {
		// 获取所有 tag 的属性
		return fmt.Sprintf("FETCH PROP ON * \"%s\" YIELD vertex AS v", id)
	}
	return fmt.Sprintf("FETCH PROP ON %s \"%s\" YIELD properties(vertex)", tag, id)
}

// UpdateVertex 构建更新节点查询
//
// nGQL 示例:
//
//	UPDATE VERTEX "person-1" SET Person.age = 31, Person.name = "Jane"
//
func (qb *QueryBuilder) UpdateVertex(id string, tag string, properties map[string]interface{}) string {
	if len(properties) == 0 {
		return ""
	}

	sets := make([]string, 0, len(properties))
	for key, value := range properties {
		sets = append(sets, fmt.Sprintf("%s.%s = %s", tag, key, formatValue(value)))
	}

	return fmt.Sprintf("UPDATE VERTEX \"%s\" SET %s", id, strings.Join(sets, ", "))
}

// DeleteVertex 构建删除节点查询
//
// nGQL 示例:
//
//	DELETE VERTEX "person-1"
//
func (qb *QueryBuilder) DeleteVertex(id string) string {
	return fmt.Sprintf("DELETE VERTEX \"%s\"", id)
}

// InsertEdge 构建插入边查询
//
// nGQL 示例:
//
//	INSERT EDGE WORKS_FOR(since) VALUES "person-1"->"org-1":(2020)
//
func (qb *QueryBuilder) InsertEdge(source, target, edgeType string, properties map[string]interface{}) string {
	if len(properties) == 0 {
		return fmt.Sprintf("INSERT EDGE %s() VALUES \"%s\"->\"%s\":()", edgeType, source, target)
	}

	propNames := make([]string, 0, len(properties))
	propValues := make([]string, 0, len(properties))

	for key, value := range properties {
		propNames = append(propNames, key)
		propValues = append(propValues, formatValue(value))
	}

	return fmt.Sprintf("INSERT EDGE %s(%s) VALUES \"%s\"->\"%s\":(%s)",
		edgeType,
		strings.Join(propNames, ", "),
		source,
		target,
		strings.Join(propValues, ", "))
}

// FetchEdge 构建获取边查询
//
// nGQL 示例:
//
//	FETCH PROP ON WORKS_FOR "person-1"->"org-1" YIELD properties(edge)
//
func (qb *QueryBuilder) FetchEdge(source, target, edgeType string) string {
	return fmt.Sprintf("FETCH PROP ON %s \"%s\"->\"%s\" YIELD properties(edge)", edgeType, source, target)
}

// UpdateEdge 构建更新边查询
//
// nGQL 示例:
//
//	UPDATE EDGE ON WORKS_FOR "person-1"->"org-1" SET since = 2021
//
func (qb *QueryBuilder) UpdateEdge(source, target, edgeType string, properties map[string]interface{}) string {
	if len(properties) == 0 {
		return ""
	}

	sets := make([]string, 0, len(properties))
	for key, value := range properties {
		sets = append(sets, fmt.Sprintf("%s = %s", key, formatValue(value)))
	}

	return fmt.Sprintf("UPDATE EDGE ON %s \"%s\"->\"%s\" SET %s",
		edgeType, source, target, strings.Join(sets, ", "))
}

// DeleteEdge 构建删除边查询
//
// nGQL 示例:
//
//	DELETE EDGE WORKS_FOR "person-1"->"org-1"
//
func (qb *QueryBuilder) DeleteEdge(source, target, edgeType string) string {
	return fmt.Sprintf("DELETE EDGE %s \"%s\"->\"%s\"", edgeType, source, target)
}

// Traverse 构建遍历查询
//
// nGQL 示例:
//
//	GO 1 TO 3 STEPS FROM "person-1" OVER * BIDIRECT YIELD $$ AS dst, edge AS e
//
func (qb *QueryBuilder) Traverse(startID string, maxDepth int, direction string) string {
	dir := ""
	if direction == "BIDIRECT" {
		dir = "BIDIRECT"
	}

	// GO 语句返回目标节点（$$）和边
	// $$ 在 GO 语句中表示目标点，会返回完整的 vertex 对象
	if maxDepth == 1 {
		return fmt.Sprintf("GO FROM \"%s\" OVER * %s YIELD $$ AS dst, edge AS e",
			startID, dir)
	}

	return fmt.Sprintf("GO 1 TO %d STEPS FROM \"%s\" OVER * %s YIELD $$ AS dst, edge AS e",
		maxDepth, startID, dir)
}

// ShortestPath 构建最短路径查询
//
// nGQL 示例:
//
//	FIND SHORTEST PATH WITH PROP FROM "person-1" TO "org-1" OVER * UPTO 5 STEPS YIELD path AS p
//
func (qb *QueryBuilder) ShortestPath(fromID, toID string, maxDepth int) string {
	// 使用 WITH PROP 来获取完整的节点和边属性
	return fmt.Sprintf("FIND SHORTEST PATH WITH PROP FROM \"%s\" TO \"%s\" OVER * UPTO %d STEPS YIELD path AS p",
		fromID, toID, maxDepth)
}

// AllPaths 构建所有路径查询
//
// nGQL 示例:
//
//	FIND ALL PATH FROM "person-1" TO "org-1" OVER * UPTO 5 STEPS
//
func (qb *QueryBuilder) AllPaths(fromID, toID string, maxDepth int) string {
	return fmt.Sprintf("FIND ALL PATH FROM \"%s\" TO \"%s\" OVER * UPTO %d STEPS",
		fromID, toID, maxDepth)
}

// Match 构建 MATCH 查询
//
// nGQL 示例:
//
//	MATCH (v:Person) WHERE v.Person.name == "John" RETURN v
//
func (qb *QueryBuilder) Match(pattern string, where string, returnClause string) string {
	query := fmt.Sprintf("MATCH %s", pattern)
	if where != "" {
		query += fmt.Sprintf(" WHERE %s", where)
	}
	if returnClause != "" {
		query += fmt.Sprintf(" RETURN %s", returnClause)
	}
	return query
}

// Lookup 构建 LOOKUP 查询
//
// nGQL 示例:
//
//	LOOKUP ON Person WHERE Person.age > 30 YIELD properties(vertex)
//
func (qb *QueryBuilder) Lookup(tag string, where string) string {
	return fmt.Sprintf("LOOKUP ON %s WHERE %s YIELD properties(vertex)", tag, where)
}

// formatValue 格式化值为 nGQL 格式
func formatValue(value interface{}) string {
	switch v := value.(type) {
	case string:
		// 转义字符串中的引号
		escaped := strings.ReplaceAll(v, "\"", "\\\"")
		return fmt.Sprintf("\"%s\"", escaped)
	case int, int8, int16, int32, int64:
		return fmt.Sprintf("%d", v)
	case uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%d", v)
	case float32:
		return fmt.Sprintf("%f", v)
	case float64:
		return fmt.Sprintf("%f", v)
	case bool:
		if v {
			return "true"
		}
		return "false"
	case []interface{}:
		// 数组
		items := make([]string, len(v))
		for i, item := range v {
			items[i] = formatValue(item)
		}
		return fmt.Sprintf("[%s]", strings.Join(items, ", "))
	case map[string]interface{}:
		// Map
		pairs := make([]string, 0, len(v))
		for k, val := range v {
			pairs = append(pairs, fmt.Sprintf("%s: %s", k, formatValue(val)))
		}
		return fmt.Sprintf("{%s}", strings.Join(pairs, ", "))
	case nil:
		return "NULL"
	default:
		// 默认转为字符串
		return fmt.Sprintf("\"%v\"", v)
	}
}

// EscapeString 转义字符串
func EscapeString(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\r", "\\r")
	s = strings.ReplaceAll(s, "\t", "\\t")
	return s
}
