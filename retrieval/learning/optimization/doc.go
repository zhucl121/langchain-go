// Package optimization 提供自适应参数优化功能。
//
// 该包实现了学习型检索系统的参数优化组件，支持：
//
// 1. 参数空间定义
// 2. 贝叶斯优化
// 3. 自动调优
// 4. 参数验证和回滚
//
// 基本用法：
//
//	// 创建优化器
//	optimizer := optimization.NewOptimizer(evaluator, collector, optimization.Config{
//	    MaxIterations: 50,
//	    TargetMetric:  "overall_score",
//	})
//
//	// 定义参数空间
//	paramSpace := optimization.ParameterSpace{
//	    Params: []optimization.Parameter{
//	        {Name: "top_k", Type: optimization.ParamTypeInt, Min: 5, Max: 20},
//	        {Name: "temperature", Type: optimization.ParamTypeFloat, Min: 0.1, Max: 1.0},
//	    },
//	}
//
//	// 优化参数
//	result, _ := optimizer.Optimize(ctx, "hybrid", paramSpace)
//	fmt.Printf("最佳参数: %v, 提升: %.2f%%\n",
//	    result.BestParams, result.Improvement)
//
//	// 自动调优（持续运行）
//	go optimizer.AutoTune(ctx, "hybrid", paramSpace)
//
// 优化算法：
//
// 使用贝叶斯优化进行智能参数搜索：
//   - 高斯过程回归建模参数-性能关系
//   - 采集函数（EI/UCB/PI）选择下一个参数点
//   - 自适应学习率和探索-利用平衡
//
// 参数类型：
//   - Int: 整数参数（如 top_k, max_length）
//   - Float: 浮点参数（如 temperature, threshold）
//   - Choice: 离散选择（如 strategy, model）
//
// 最佳实践：
//   - 定义合理的参数范围
//   - 使用充足的样本数（建议 > 30）
//   - 设置合适的迭代次数（建议 50-100）
//   - 定期验证优化效果
//
package optimization
