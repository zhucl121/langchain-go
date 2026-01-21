package nebula

import (
	"context"
	"fmt"
	"strings"
	"sync"

	nebula "github.com/vesoft-inc/nebula-go/v3"
	"github.com/zhucl121/langchain-go/retrieval/graphdb"
)

// NebulaDriver NebulaGraph 驱动器
//
// NebulaDriver 实现了 graphdb.GraphDB 接口，提供 NebulaGraph 图数据库支持。
type NebulaDriver struct {
	config    Config
	pool      *nebula.ConnectionPool
	session   *nebula.Session
	spaceName string
	mu        sync.RWMutex
	connected bool
	qb        *QueryBuilder
	converter *Converter
}

// NewNebulaDriver 创建 NebulaGraph 驱动器
//
// 参数：
//   - config: 驱动器配置
//
// 返回：
//   - *NebulaDriver: 驱动器实例
//   - error: 错误
//
// 示例：
//
//	config := nebula.DefaultConfig()
//	config.Space = "my_space"
//	driver, err := nebula.NewNebulaDriver(config)
//
func NewNebulaDriver(config Config) (*NebulaDriver, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("nebula: invalid config: %w", err)
	}

	return &NebulaDriver{
		config:    config,
		spaceName: config.Space,
		qb:        NewQueryBuilder(config.Space),
		converter: NewConverter(),
	}, nil
}

// Connect 连接到 NebulaGraph
func (d *NebulaDriver) Connect(ctx context.Context) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.connected {
		return nil
	}

	// 创建连接池配置
	poolConfig := nebula.PoolConfig{
		TimeOut:         d.config.Timeout,
		IdleTime:        d.config.IdleTime,
		MaxConnPoolSize: d.config.MaxConnPoolSize,
		MinConnPoolSize: d.config.MinConnPoolSize,
	}

	// 转换地址格式
	addresses := make([]nebula.HostAddress, len(d.config.Addresses))
	for i, addr := range d.config.Addresses {
		// 解析 "host:port" 格式
		parts := strings.Split(addr, ":")
		if len(parts) != 2 {
			return fmt.Errorf("nebula: invalid address format: %s", addr)
		}
		port := 9669
		fmt.Sscanf(parts[1], "%d", &port)
		addresses[i] = nebula.HostAddress{
			Host: parts[0],
			Port: port,
		}
	}

	// 创建连接池
	pool, err := nebula.NewConnectionPool(
		addresses,
		poolConfig,
		nebula.DefaultLogger{},
	)
	if err != nil {
		return fmt.Errorf("nebula: failed to create connection pool: %w", err)
	}

	d.pool = pool

	// 创建 session
	session, err := d.pool.GetSession(d.config.Username, d.config.Password)
	if err != nil {
		d.pool.Close()
		return fmt.Errorf("nebula: failed to create session: %w", err)
	}

	d.session = session

	// 使用图空间
	if d.spaceName != "" {
		query := fmt.Sprintf("USE %s", d.spaceName)
		result, err := d.session.Execute(query)
		if err != nil {
			d.session.Release()
			d.pool.Close()
			return fmt.Errorf("nebula: failed to use space: %w", err)
		}
		if !result.IsSucceed() {
			d.session.Release()
			d.pool.Close()
			return fmt.Errorf("nebula: failed to use space: %s", result.GetErrorMsg())
		}
	}

	d.connected = true
	return nil
}

// Close 关闭连接
func (d *NebulaDriver) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if !d.connected {
		return nil
	}

	if d.session != nil {
		d.session.Release()
		d.session = nil
	}

	if d.pool != nil {
		d.pool.Close()
		d.pool = nil
	}

	d.connected = false
	return nil
}

// IsConnected 检查连接状态
func (d *NebulaDriver) IsConnected() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.connected
}

// Execute 执行 nGQL 查询
func (d *NebulaDriver) Execute(ctx context.Context, query string) (*nebula.ResultSet, error) {
	d.mu.RLock()
	if !d.connected {
		d.mu.RUnlock()
		return nil, graphdb.ErrNotConnected
	}
	session := d.session
	d.mu.RUnlock()

	result, err := session.Execute(query)
	if err != nil {
		return nil, fmt.Errorf("nebula: execute query failed: %w", err)
	}

	if !result.IsSucceed() {
		return nil, fmt.Errorf("nebula: query failed: %s", result.GetErrorMsg())
	}

	return result, nil
}

// AddNode 添加节点
func (d *NebulaDriver) AddNode(ctx context.Context, node *graphdb.Node) error {
	if node == nil {
		return fmt.Errorf("nebula: node is nil")
	}

	if node.ID == "" {
		return fmt.Errorf("nebula: node ID is required")
	}

	query := d.qb.InsertVertex(node.ID, node.Type, node.Properties)
	_, err := d.Execute(ctx, query)
	return err
}

// GetNode 获取节点
func (d *NebulaDriver) GetNode(ctx context.Context, id string) (*graphdb.Node, error) {
	if id == "" {
		return nil, fmt.Errorf("nebula: node ID is required")
	}

	// 使用 FETCH PROP 查询节点的所有属性
	// YIELD vertex AS v 返回完整的节点对象
	query := fmt.Sprintf("FETCH PROP ON * \"%s\" YIELD vertex AS v", id)
	result, err := d.Execute(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("nebula: failed to fetch node: %w", err)
	}

	if result.GetRowSize() == 0 {
		return nil, graphdb.ErrNodeNotFound
	}

	// 使用 converter 提取节点
	nodes, _, _, err := d.converter.ExtractFromResultSet(result)
	if err != nil {
		return nil, fmt.Errorf("nebula: failed to extract node from result: %w", err)
	}

	if len(nodes) == 0 {
		return nil, graphdb.ErrNodeNotFound
	}

	// 返回第一个节点（应该只有一个）
	return nodes[0], nil
}

// UpdateNode 更新节点
func (d *NebulaDriver) UpdateNode(ctx context.Context, node *graphdb.Node) error {
	if node == nil {
		return fmt.Errorf("nebula: node is nil")
	}

	if node.ID == "" {
		return fmt.Errorf("nebula: node ID is required")
	}

	query := d.qb.UpdateVertex(node.ID, node.Type, node.Properties)
	_, err := d.Execute(ctx, query)
	return err
}

// DeleteNode 删除节点
func (d *NebulaDriver) DeleteNode(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("nebula: node ID is required")
	}

	query := d.qb.DeleteVertex(id)
	_, err := d.Execute(ctx, query)
	return err
}

// AddEdge 添加边
func (d *NebulaDriver) AddEdge(ctx context.Context, edge *graphdb.Edge) error {
	if edge == nil {
		return fmt.Errorf("nebula: edge is nil")
	}

	if edge.Source == "" || edge.Target == "" {
		return fmt.Errorf("nebula: source and target are required")
	}

	query := d.qb.InsertEdge(edge.Source, edge.Target, edge.Type, edge.Properties)
	_, err := d.Execute(ctx, query)
	return err
}

// GetEdge 获取边
func (d *NebulaDriver) GetEdge(ctx context.Context, id string) (*graphdb.Edge, error) {
	if id == "" {
		return nil, fmt.Errorf("nebula: edge ID is required")
	}

	// NebulaGraph 的边 ID 格式: source_id-edge_type-target_id
	// 需要解析 ID
	parts := strings.Split(id, "-")
	if len(parts) < 3 {
		return nil, fmt.Errorf("nebula: invalid edge ID format, expected source-type-target")
	}

	srcID := parts[0]
	edgeType := strings.Join(parts[1:len(parts)-1], "-")
	dstID := parts[len(parts)-1]

	// 使用 FETCH PROP 查询边的所有属性
	// YIELD edge AS e 返回完整的边对象
	query := fmt.Sprintf("FETCH PROP ON %s \"%s\" -> \"%s\" YIELD edge AS e", edgeType, srcID, dstID)
	result, err := d.Execute(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("nebula: failed to fetch edge: %w", err)
	}

	if result.GetRowSize() == 0 {
		return nil, graphdb.ErrEdgeNotFound
	}

	// 使用 converter 提取边
	_, edges, _, err := d.converter.ExtractFromResultSet(result)
	if err != nil {
		return nil, fmt.Errorf("nebula: failed to extract edge from result: %w", err)
	}

	if len(edges) == 0 {
		return nil, graphdb.ErrEdgeNotFound
	}

	// 返回第一条边（应该只有一条）
	return edges[0], nil
}

// UpdateEdge 更新边
func (d *NebulaDriver) UpdateEdge(ctx context.Context, edge *graphdb.Edge) error {
	if edge == nil {
		return fmt.Errorf("nebula: edge is nil")
	}

	// NebulaGraph 不支持直接更新边，需要先删除再添加
	// 或使用 UPDATE EDGE 语句
	return fmt.Errorf("nebula: UpdateEdge not yet implemented")
}

// DeleteEdge 删除边
func (d *NebulaDriver) DeleteEdge(ctx context.Context, id string) error {
	// NebulaGraph 的边没有独立 ID
	return fmt.Errorf("nebula: DeleteEdge by ID not supported, use source/target/type instead")
}

// DeleteEdgeByEndpoints 通过端点删除边
func (d *NebulaDriver) DeleteEdgeByEndpoints(ctx context.Context, source, target, edgeType string) error {
	query := d.qb.DeleteEdge(source, target, edgeType)
	_, err := d.Execute(ctx, query)
	return err
}

// Traverse 图遍历
func (d *NebulaDriver) Traverse(ctx context.Context, startID string, opts graphdb.TraverseOptions) (*graphdb.TraverseResult, error) {
	if startID == "" {
		return nil, fmt.Errorf("nebula: start node ID is required")
	}

	// 设置默认值
	if opts.MaxDepth <= 0 {
		opts.MaxDepth = 5
	}

	direction := ""
	if opts.Direction == graphdb.DirectionBoth {
		direction = "BIDIRECT"
	}

	query := d.qb.Traverse(startID, opts.MaxDepth, direction)
	result, err := d.Execute(ctx, query)
	if err != nil {
		return nil, err
	}

	// 使用转换器提取结果
	nodes, edges, paths, err := d.converter.ExtractFromResultSet(result)
	if err != nil {
		return nil, fmt.Errorf("nebula: failed to extract traverse result: %w", err)
	}

	traverseResult := &graphdb.TraverseResult{
		Nodes: nodes,
		Edges: edges,
		Paths: paths,
	}

	return traverseResult, nil
}

// ShortestPath 最短路径
func (d *NebulaDriver) ShortestPath(ctx context.Context, fromID, toID string, opts graphdb.PathOptions) (*graphdb.Path, error) {
	if fromID == "" || toID == "" {
		return nil, fmt.Errorf("nebula: from and to IDs are required")
	}

	// 设置默认值
	if opts.MaxDepth <= 0 {
		opts.MaxDepth = 10
	}

	query := d.qb.ShortestPath(fromID, toID, opts.MaxDepth)
	result, err := d.Execute(ctx, query)
	if err != nil {
		return nil, err
	}

	// 使用转换器提取路径
	_, _, paths, err := d.converter.ExtractFromResultSet(result)
	if err != nil {
		return nil, fmt.Errorf("nebula: failed to extract path result: %w", err)
	}

	// 返回第一条路径（最短路径）
	if len(paths) > 0 {
		return paths[0], nil
	}

	// 没有找到路径
	return &graphdb.Path{
		Nodes:  []*graphdb.Node{},
		Edges:  []*graphdb.Edge{},
		Length: 0,
		Cost:   0,
	}, nil
}

// ExecuteQuery 执行原生查询
func (d *NebulaDriver) ExecuteQuery(ctx context.Context, query string) (interface{}, error) {
	return d.Execute(ctx, query)
}
