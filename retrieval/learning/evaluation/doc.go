// Package evaluation 提供检索质量评估功能。
//
// 该包实现了学习型检索系统的质量评估组件，支持：
//
// 1. 相关性指标（Precision, Recall, F1, NDCG, MRR）
// 2. 用户满意度指标（评分、CTR、阅读率）
// 3. 效率指标（响应时间）
// 4. 策略对比分析
//
// 基本用法：
//
//	// 创建评估器
//	evaluator := evaluation.NewEvaluator(feedbackCollector)
//
//	// 评估单个查询
//	metrics, _ := evaluator.EvaluateQuery(ctx, queryFeedback)
//	fmt.Printf("NDCG: %.3f, MRR: %.3f\n", metrics.NDCG, metrics.MRR)
//
//	// 评估策略
//	strategyMetrics, _ := evaluator.EvaluateStrategy(ctx, "hybrid", evaluation.EvaluateOptions{
//	    TimeRange: 24 * time.Hour,
//	})
//
//	// 对比两个策略
//	comparison, _ := evaluator.CompareStrategies(ctx, "hybrid", "vector")
//	fmt.Printf("Winner: %s, Improvement: %.2f%%\n",
//	    comparison.Winner, comparison.Improvement)
//
// 评估指标：
//
// 相关性指标：
//   - Precision: 检索结果中相关文档的比例
//   - Recall: 相关文档中被检索到的比例
//   - F1 Score: Precision 和 Recall 的调和平均
//   - NDCG: 归一化折损累计增益，考虑排序位置
//   - MRR: 平均倒数排名，首个相关文档的排名
//
// 用户满意度指标：
//   - AvgRating: 平均用户评分
//   - CTR: 点击率
//   - ReadRate: 阅读率
//
// 综合评分：
//   - OverallScore: 综合所有指标的得分 (0-1)
//
package evaluation
