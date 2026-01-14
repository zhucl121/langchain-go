// Package executor 提供 LangGraph 图执行引擎。
//
// executor 包负责执行已编译的状态图，管理节点调度和状态传递。
//
// # 执行模型
//
// 执行引擎采用基于状态的执行模型：
//   - 从入口点开始
//   - 按边路由执行节点
//   - 传递和更新状态
//   - 支持中断和恢复
//
// # 核心组件
//
// 1. **Executor** - 执行器
//   - 管理图执行
//   - 协调节点调度
//   - 处理错误和中断
//
// 2. **ExecutionContext** - 执行上下文
//   - 维护运行时状态
//   - 记录执行历史
//   - 支持 Checkpoint
//
// 3. **Scheduler** - 调度器
//   - 节点调度策略
//   - 并发控制
//   - 资源管理
//
// # 基本使用
//
// 执行图：
//
//	executor := executor.NewExecutor[MyState]()
//	result, err := executor.Execute(ctx, compiled, initialState)
//	if err != nil {
//	    // 处理执行错误
//	}
//
// 带上下文的执行：
//
//	execCtx := executor.NewExecutionContext[MyState](initialState)
//	execCtx.WithCheckpointer(checkpointer)
//	execCtx.WithCallback(func(event Event) {
//	    // 处理事件
//	})
//
//	result, err := executor.ExecuteWithContext(ctx, compiled, execCtx)
//
// # 执行控制
//
// 执行引擎支持多种控制机制：
//   - Context 取消
//   - 中断点（Interrupt）
//   - 超时控制
//   - 最大步数限制
//
// # 错误处理
//
// 执行器提供详细的错误信息：
//   - 节点执行错误
//   - 路由错误
//   - 超时错误
//   - 中断错误
//
package executor
