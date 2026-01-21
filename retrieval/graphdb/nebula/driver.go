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

	// 需要知道节点类型，先尝试查询所有 tag
	// 简化实现：假设节点类型存储在 properties 中
	query := fmt.Sprintf("FETCH PROP ON * \"%s\" YIELD vertex AS v", id)
	result, err := d.Execute(ctx, query)
	if err != nil {
		return nil, err
	}

	if result.GetRowSize() == 0 {
		return nil, graphdb.ErrNodeNotFound
	}

	// 转换结果
	node := &graphdb.Node{
		ID:         id,
		Properties: make(map[string]interface{}),
	}

	// TODO: 从 result 中提取节点属性
	// 这需要解析 NebulaGraph 的返回结果

	return node, nil
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
	// NebulaGraph 的边没有独立 ID，需要通过 source + target + type 查询
	return nil, fmt.Errorf("nebula: GetEdge by ID not supported, use source/target instead")
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

	// 转换结果
	traverseResult := &graphdb.TraverseResult{
		Nodes: []*graphdb.Node{},
		Edges: []*graphdb.Edge{},
		Paths: []*graphdb.Path{},
	}

	// TODO: 解析遍历结果
	_ = result

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

	// 转换结果
	path := &graphdb.Path{
		Nodes:  []*graphdb.Node{},
		Edges:  []*graphdb.Edge{},
		Length: 0,
		Cost:   0,
	}

	// TODO: 解析路径结果
	_ = result

	return path, nil
}

// ExecuteQuery 执行原生查询
func (d *NebulaDriver) ExecuteQuery(ctx context.Context, query string) (interface{}, error) {
	return d.Execute(ctx, query)
}
