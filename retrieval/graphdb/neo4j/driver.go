package neo4j

import (
	"context"
	"fmt"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/zhucl121/langchain-go/retrieval/graphdb"
)

// Neo4jDriver Neo4j 图数据库驱动器
type Neo4jDriver struct {
	config    Config
	driver    neo4j.DriverWithContext
	connected bool
}

// NewNeo4jDriver 创建 Neo4j 驱动器
func NewNeo4jDriver(config Config) (*Neo4jDriver, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &Neo4jDriver{
		config:    config,
		connected: false,
	}, nil
}

// Connect 连接到 Neo4j 数据库
func (d *Neo4jDriver) Connect(ctx context.Context) error {
	if d.connected {
		return nil
	}

	// 创建驱动器配置
	auth := neo4j.BasicAuth(d.config.Username, d.config.Password, "")

	driverConfig := func(config *neo4j.Config) {
		config.MaxConnectionPoolSize = d.config.MaxConnectionPoolSize
		config.ConnectionAcquisitionTimeout = d.config.ConnectionAcquisitionTimeout
		config.MaxConnectionLifetime = d.config.MaxConnectionLifetime
		config.MaxTransactionRetryTime = d.config.MaxTransactionRetryTime
	}

	// 创建驱动器
	driver, err := neo4j.NewDriverWithContext(d.config.URI, auth, driverConfig)
	if err != nil {
		return fmt.Errorf("failed to create neo4j driver: %w", err)
	}

	d.driver = driver

	// 验证连接
	if err := d.driver.VerifyConnectivity(ctx); err != nil {
		d.driver.Close(ctx)
		return fmt.Errorf("failed to verify connectivity: %w", err)
	}

	d.connected = true
	return nil
}

// Close 关闭连接
func (d *Neo4jDriver) Close() error {
	if !d.connected {
		return nil
	}

	if d.driver != nil {
		if err := d.driver.Close(context.Background()); err != nil {
			return fmt.Errorf("failed to close driver: %w", err)
		}
	}

	d.connected = false
	return nil
}

// Ping 健康检查
func (d *Neo4jDriver) Ping(ctx context.Context) error {
	if !d.connected {
		return graphdb.ErrNotConnected
	}

	return d.driver.VerifyConnectivity(ctx)
}

// AddNode 添加节点
func (d *Neo4jDriver) AddNode(ctx context.Context, node *graphdb.Node) error {
	if !d.connected {
		return graphdb.ErrNotConnected
	}

	if node == nil || node.ID == "" {
		return graphdb.ErrInvalidNode
	}

	session := d.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: d.config.Database,
	})
	defer session.Close(ctx)

	// 构建 Cypher 查询
	query := fmt.Sprintf(`
		MERGE (n:%s {id: $id})
		SET n += $properties
		SET n.label = $label
		RETURN n
	`, node.Type)

	params := map[string]interface{}{
		"id":         node.ID,
		"properties": node.Properties,
		"label":      node.Label,
	}

	// 添加 embedding（如果存在）
	if len(node.Embedding) > 0 {
		params["properties"].(map[string]interface{})["embedding"] = node.Embedding
	}

	_, err := session.Run(ctx, query, params)
	if err != nil {
		return fmt.Errorf("failed to add node: %w", err)
	}

	return nil
}

// GetNode 获取节点
func (d *Neo4jDriver) GetNode(ctx context.Context, id string) (*graphdb.Node, error) {
	if !d.connected {
		return nil, graphdb.ErrNotConnected
	}

	session := d.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: d.config.Database,
	})
	defer session.Close(ctx)

	query := `
		MATCH (n {id: $id})
		RETURN n, labels(n) as types
	`

	result, err := session.Run(ctx, query, map[string]interface{}{
		"id": id,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get node: %w", err)
	}

	if !result.Next(ctx) {
		return nil, graphdb.ErrNodeNotFound
	}

	record := result.Record()
	return d.recordToNode(record)
}

// UpdateNode 更新节点
func (d *Neo4jDriver) UpdateNode(ctx context.Context, node *graphdb.Node) error {
	if !d.connected {
		return graphdb.ErrNotConnected
	}

	if node == nil || node.ID == "" {
		return graphdb.ErrInvalidNode
	}

	session := d.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: d.config.Database,
	})
	defer session.Close(ctx)

	query := `
		MATCH (n {id: $id})
		SET n += $properties
		RETURN n
	`

	params := map[string]interface{}{
		"id":         node.ID,
		"properties": node.Properties,
	}

	if node.Label != "" {
		params["properties"].(map[string]interface{})["label"] = node.Label
	}

	result, err := session.Run(ctx, query, params)
	if err != nil {
		return fmt.Errorf("failed to update node: %w", err)
	}

	if !result.Next(ctx) {
		return graphdb.ErrNodeNotFound
	}

	return nil
}

// DeleteNode 删除节点
func (d *Neo4jDriver) DeleteNode(ctx context.Context, id string) error {
	if !d.connected {
		return graphdb.ErrNotConnected
	}

	session := d.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: d.config.Database,
	})
	defer session.Close(ctx)

	query := `
		MATCH (n {id: $id})
		DETACH DELETE n
	`

	_, err := session.Run(ctx, query, map[string]interface{}{
		"id": id,
	})
	if err != nil {
		return fmt.Errorf("failed to delete node: %w", err)
	}

	return nil
}

// BatchAddNodes 批量添加节点
func (d *Neo4jDriver) BatchAddNodes(ctx context.Context, nodes []*graphdb.Node) error {
	if !d.connected {
		return graphdb.ErrNotConnected
	}

	session := d.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: d.config.Database,
	})
	defer session.Close(ctx)

	// 使用事务
	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		for _, node := range nodes {
			if node == nil || node.ID == "" {
				return nil, graphdb.ErrInvalidNode
			}

			query := fmt.Sprintf(`
				MERGE (n:%s {id: $id})
				SET n += $properties
				SET n.label = $label
			`, node.Type)

			params := map[string]interface{}{
				"id":         node.ID,
				"properties": node.Properties,
				"label":      node.Label,
			}

			if _, err := tx.Run(ctx, query, params); err != nil {
				return nil, err
			}
		}
		return nil, nil
	})

	if err != nil {
		return fmt.Errorf("failed to batch add nodes: %w", err)
	}

	return nil
}

// AddEdge 添加边
func (d *Neo4jDriver) AddEdge(ctx context.Context, edge *graphdb.Edge) error {
	if !d.connected {
		return graphdb.ErrNotConnected
	}

	if edge == nil || edge.ID == "" {
		return graphdb.ErrInvalidEdge
	}

	session := d.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: d.config.Database,
	})
	defer session.Close(ctx)

	query := fmt.Sprintf(`
		MATCH (a {id: $source})
		MATCH (b {id: $target})
		MERGE (a)-[r:%s {id: $id}]->(b)
		SET r += $properties
		SET r.label = $label
		SET r.weight = $weight
		RETURN r
	`, edge.Type)

	params := map[string]interface{}{
		"id":         edge.ID,
		"source":     edge.Source,
		"target":     edge.Target,
		"properties": edge.Properties,
		"label":      edge.Label,
		"weight":     edge.Weight,
	}

	_, err := session.Run(ctx, query, params)
	if err != nil {
		return fmt.Errorf("failed to add edge: %w", err)
	}

	return nil
}

// GetEdge 获取边
func (d *Neo4jDriver) GetEdge(ctx context.Context, id string) (*graphdb.Edge, error) {
	if !d.connected {
		return nil, graphdb.ErrNotConnected
	}

	session := d.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: d.config.Database,
	})
	defer session.Close(ctx)

	query := `
		MATCH (a)-[r {id: $id}]->(b)
		RETURN r, a.id as source, b.id as target, type(r) as edgeType
	`

	result, err := session.Run(ctx, query, map[string]interface{}{
		"id": id,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get edge: %w", err)
	}

	if !result.Next(ctx) {
		return nil, graphdb.ErrEdgeNotFound
	}

	record := result.Record()
	return d.recordToEdge(record)
}

// DeleteEdge 删除边
func (d *Neo4jDriver) DeleteEdge(ctx context.Context, id string) error {
	if !d.connected {
		return graphdb.ErrNotConnected
	}

	session := d.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: d.config.Database,
	})
	defer session.Close(ctx)

	query := `
		MATCH ()-[r {id: $id}]->()
		DELETE r
	`

	_, err := session.Run(ctx, query, map[string]interface{}{
		"id": id,
	})
	if err != nil {
		return fmt.Errorf("failed to delete edge: %w", err)
	}

	return nil
}

// BatchAddEdges 批量添加边
func (d *Neo4jDriver) BatchAddEdges(ctx context.Context, edges []*graphdb.Edge) error {
	if !d.connected {
		return graphdb.ErrNotConnected
	}

	session := d.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: d.config.Database,
	})
	defer session.Close(ctx)

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		for _, edge := range edges {
			if edge == nil || edge.ID == "" {
				return nil, graphdb.ErrInvalidEdge
			}

			query := fmt.Sprintf(`
				MATCH (a {id: $source})
				MATCH (b {id: $target})
				MERGE (a)-[r:%s {id: $id}]->(b)
				SET r += $properties
				SET r.label = $label
				SET r.weight = $weight
			`, edge.Type)

			params := map[string]interface{}{
				"id":         edge.ID,
				"source":     edge.Source,
				"target":     edge.Target,
				"properties": edge.Properties,
				"label":      edge.Label,
				"weight":     edge.Weight,
			}

			if _, err := tx.Run(ctx, query, params); err != nil {
				return nil, err
			}
		}
		return nil, nil
	})

	if err != nil {
		return fmt.Errorf("failed to batch add edges: %w", err)
	}

	return nil
}

// FindNodes 查找节点
func (d *Neo4jDriver) FindNodes(ctx context.Context, filter graphdb.NodeFilter) ([]*graphdb.Node, error) {
	if !d.connected {
		return nil, graphdb.ErrNotConnected
	}

	session := d.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: d.config.Database,
	})
	defer session.Close(ctx)

	query, params := d.buildNodeQuery(filter)

	result, err := session.Run(ctx, query, params)
	if err != nil {
		return nil, fmt.Errorf("failed to find nodes: %w", err)
	}

	var nodes []*graphdb.Node
	for result.Next(ctx) {
		record := result.Record()
		node, err := d.recordToNode(record)
		if err != nil {
			continue
		}
		nodes = append(nodes, node)
	}

	return nodes, nil
}

// FindEdges 查找边
func (d *Neo4jDriver) FindEdges(ctx context.Context, filter graphdb.EdgeFilter) ([]*graphdb.Edge, error) {
	if !d.connected {
		return nil, graphdb.ErrNotConnected
	}

	session := d.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: d.config.Database,
	})
	defer session.Close(ctx)

	query, params := d.buildEdgeQuery(filter)

	result, err := session.Run(ctx, query, params)
	if err != nil {
		return nil, fmt.Errorf("failed to find edges: %w", err)
	}

	var edges []*graphdb.Edge
	for result.Next(ctx) {
		record := result.Record()
		edge, err := d.recordToEdge(record)
		if err != nil {
			continue
		}
		edges = append(edges, edge)
	}

	return edges, nil
}

// Traverse 图遍历
func (d *Neo4jDriver) Traverse(ctx context.Context, startID string, opts graphdb.TraverseOptions) (*graphdb.TraverseResult, error) {
	if !d.connected {
		return nil, graphdb.ErrNotConnected
	}

	session := d.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: d.config.Database,
	})
	defer session.Close(ctx)

	query := d.buildTraverseQuery(startID, opts)

	result, err := session.Run(ctx, query, map[string]interface{}{
		"startID":  startID,
		"maxDepth": opts.MaxDepth,
		"limit":    opts.Limit,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to traverse: %w", err)
	}

	return d.parseTraverseResult(ctx, result)
}

// ShortestPath 最短路径
func (d *Neo4jDriver) ShortestPath(ctx context.Context, startID, endID string, opts graphdb.PathOptions) (*graphdb.Path, error) {
	if !d.connected {
		return nil, graphdb.ErrNotConnected
	}

	session := d.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: d.config.Database,
	})
	defer session.Close(ctx)

	query := d.buildPathQuery(opts)

	result, err := session.Run(ctx, query, map[string]interface{}{
		"startID":  startID,
		"endID":    endID,
		"maxDepth": opts.MaxDepth,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to find shortest path: %w", err)
	}

	if !result.Next(ctx) {
		return nil, graphdb.ErrNoPathFound
	}

	record := result.Record()
	return d.recordToPath(record)
}
