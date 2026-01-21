// Package health 提供节点健康检查功能。
//
// 本包实现了多种健康检查机制，支持：
// - HTTP 健康检查
// - TCP 连接检查
// - 自定义健康检查
// - 健康状态聚合
//
// 示例:
//
//	// 创建 HTTP 健康检查器
//	httpChecker := health.NewHTTPChecker(health.HTTPConfig{
//	    Endpoint: "/health",
//	    Timeout:  5 * time.Second,
//	})
//
//	// 创建 TCP 健康检查器
//	tcpChecker := health.NewTCPChecker(health.TCPConfig{
//	    Timeout: 3 * time.Second,
//	})
//
//	// 检查节点健康
//	result, err := httpChecker.Check(ctx, node)
//	if err != nil {
//	    log.Printf("Health check failed: %v", err)
//	}
//	fmt.Printf("Node is healthy: %v\n", result.Healthy)
package health
