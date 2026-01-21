// Package failover 提供故障转移和高可用功能。
//
// 本包实现了自动故障检测、转移和恢复机制：
// - 故障转移管理器
// - 熔断器模式
// - 自动健康监控
// - 自动重新平衡
// - 告警通知
//
// 示例:
//
//	// 创建故障转移管理器
//	manager := failover.NewFailoverManager(failover.Config{
//	    NodeManager:         nodeManager,
//	    HealthCheckInterval: 10 * time.Second,
//	    FailureThreshold:    3,
//	    RecoveryThreshold:   2,
//	    AutoRebalance:       true,
//	})
//
//	// 启动健康监控
//	ctx := context.Background()
//	go manager.MonitorHealth(ctx)
//
//	// 手动触发故障转移
//	err := manager.HandleFailure(ctx, "node-123")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// 使用熔断器
//	cb := failover.NewCircuitBreaker(failover.CircuitBreakerConfig{
//	    FailureThreshold: 5,
//	    Timeout:         30 * time.Second,
//	})
//
//	err := cb.Execute(func() error {
//	    // 执行可能失败的操作
//	    return callRemoteService()
//	})
//
// 支持的功能：
// - 自动故障检测
// - 故障转移
// - 节点恢复
// - 自动重新平衡
// - 熔断器保护
// - 告警通知
//
package failover
