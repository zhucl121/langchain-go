package graphdb

import "errors"

var (
	// ErrNodeNotFound 节点未找到
	ErrNodeNotFound = errors.New("graphdb: node not found")

	// ErrEdgeNotFound 边未找到
	ErrEdgeNotFound = errors.New("graphdb: edge not found")

	// ErrNodeExists 节点已存在
	ErrNodeExists = errors.New("graphdb: node already exists")

	// ErrEdgeExists 边已存在
	ErrEdgeExists = errors.New("graphdb: edge already exists")

	// ErrInvalidNode 无效节点
	ErrInvalidNode = errors.New("graphdb: invalid node")

	// ErrInvalidEdge 无效边
	ErrInvalidEdge = errors.New("graphdb: invalid edge")

	// ErrNotConnected 未连接
	ErrNotConnected = errors.New("graphdb: not connected to database")

	// ErrConnectionFailed 连接失败
	ErrConnectionFailed = errors.New("graphdb: failed to connect to database")

	// ErrQueryFailed 查询失败
	ErrQueryFailed = errors.New("graphdb: query execution failed")

	// ErrTransactionFailed 事务失败
	ErrTransactionFailed = errors.New("graphdb: transaction failed")

	// ErrInvalidTraverseOptions 无效遍历选项
	ErrInvalidTraverseOptions = errors.New("graphdb: invalid traverse options")

	// ErrInvalidPathOptions 无效路径选项
	ErrInvalidPathOptions = errors.New("graphdb: invalid path options")

	// ErrMaxDepthExceeded 超过最大深度
	ErrMaxDepthExceeded = errors.New("graphdb: max depth exceeded")

	// ErrNoPathFound 未找到路径
	ErrNoPathFound = errors.New("graphdb: no path found")
)
