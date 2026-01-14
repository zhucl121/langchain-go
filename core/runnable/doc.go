// Package runnable 提供 LangChain 的核心抽象 - Runnable 接口。
//
// Runnable 是 LangChain 中所有可执行组件的基础接口，类似于 Python
// LangChain 中的 LCEL (LangChain Expression Language)。
//
// 主要特性：
//   - 统一的执行接口（Invoke, Batch, Stream）
//   - 支持泛型，类型安全
//   - 链式组合（Pipe）
//   - 并行执行（Batch 自动并行）
//   - 流式输出（Stream）
//
// 使用示例：
//
//	// 创建一个简单的 Runnable
//	doubler := runnable.Lambda(func(ctx context.Context, x int) (int, error) {
//	    return x * 2, nil
//	})
//
//	// 执行
//	result, _ := doubler.Invoke(ctx, 5) // 返回 10
//
//	// 批量执行（自动并行）
//	results, _ := doubler.Batch(ctx, []int{1, 2, 3}) // 返回 [2, 4, 6]
//
//	// 链式组合
//	adder := runnable.Lambda(func(ctx context.Context, x int) (int, error) {
//	    return x + 1, nil
//	})
//	chain := doubler.Pipe(adder) // 先乘2，再加1
//	result, _ := chain.Invoke(ctx, 5) // 返回 11
//
package runnable
