// Package durability 提供 LangGraph Durability 模式实现。
//
// durability 包负责确保图执行的持久性和可恢复性。
//
// # Durability 概念
//
// Durability 模式确保图执行在面对故障时能够恢复：
//   - 自动保存检查点
//   - 故障检测和恢复
//   - 任务包装和重试
//   - 幂等性保证
//
// # 核心组件
//
// 1. **DurabilityMode** - 持久性模式
//   - AtMostOnce: 最多执行一次
//   - AtLeastOnce: 至少执行一次
//   - ExactlyOnce: 恰好执行一次（最强保证）
//
// 2. **DurableTask** - 持久化任务
//   - 任务包装
//   - 状态追踪
//   - 重试逻辑
//
// 3. **RecoveryManager** - 恢复管理器
//   - 故障检测
//   - 状态恢复
//   - 续传执行
//
// # 基本使用
//
// 配置 Durability 模式：
//
//	graph := state.NewStateGraph[MyState]("my-graph")
//	graph.WithDurability(durability.ExactlyOnce)
//
// 包装持久化任务：
//
//	task := durability.NewDurableTask("task-1", func(ctx context.Context, state S) (S, error) {
//	    // 任务逻辑
//	    return state, nil
//	})
//
// 恢复执行：
//
//	manager := durability.NewRecoveryManager[MyState](checkpointer)
//	state, err := manager.Recover(ctx, threadID)
//
// # 工作原理
//
// Durability 模式通过以下机制保证可靠性：
//   - 执行前保存检查点
//   - 记录任务执行状态
//   - 失败时自动重试或恢复
//   - 幂等性检查避免重复执行
//
package durability
