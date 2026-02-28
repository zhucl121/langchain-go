package main

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/zhucl121/langchain-go/pkg/types"
	"github.com/zhucl121/langchain-go/retrieval/learning/evaluation"
	"github.com/zhucl121/langchain-go/retrieval/learning/feedback"
	"github.com/zhucl121/langchain-go/retrieval/learning/optimization"
)

func main() {
	fmt.Println("=== LangChain-Go Learning Retrieval - 参数优化示例 ===")

	// 创建反馈收集器和评估器
	storage := feedback.NewMemoryStorage()
	collector := feedback.NewCollector(storage)
	evaluator := evaluation.NewEvaluator(collector)
	ctx := context.Background()

	// 创建测试数据
	fmt.Println("1. 准备测试数据...")
	strategyID := "hybrid-search"
	createTestData(ctx, collector, strategyID, 20)
	fmt.Printf("✓ 已创建 20 个测试查询 (策略: %s)\n\n", strategyID)

	// 评估当前性能
	fmt.Println("2. 评估当前性能...")
	metrics, err := evaluator.EvaluateStrategy(ctx, strategyID, evaluation.EvaluateOptions{
		TimeRange:     1 * time.Hour,
		MinSampleSize: 10,
	})
	if err != nil {
		panic(err)
	}
	
	fmt.Printf("当前性能:\n")
	fmt.Printf("  📊 综合得分: %.3f\n", metrics.AvgMetrics.OverallScore)
	fmt.Printf("  🎯 NDCG: %.3f\n", metrics.AvgMetrics.NDCG)
	fmt.Printf("  ⭐ 平均评分: %.1f/5.0\n", metrics.AvgMetrics.AvgRating)
	fmt.Printf("  📈 点击率: %.1f%%\n\n", metrics.AvgMetrics.CTR*100)

	// 创建优化器
	fmt.Println("3. 创建参数优化器...")
	optimizer := optimization.NewOptimizer(evaluator, collector, optimization.DefaultConfig())
	fmt.Println("✓ 优化器已创建")

	// 定义参数空间
	fmt.Println("4. 定义参数空间...")
	paramSpace := optimization.ParameterSpace{
		Params: []optimization.Parameter{
			{
				Name:    "top_k",
				Type:    optimization.ParamTypeInt,
				Min:     5,
				Max:     30,
				Default: 10,
			},
			{
				Name:    "temperature",
				Type:    optimization.ParamTypeFloat,
				Min:     0.1,
				Max:     1.0,
				Default: 0.7,
			},
			{
				Name:    "rerank_strategy",
				Type:    optimization.ParamTypeChoice,
				Values:  []string{"score", "diversity", "mmr"},
				Default: "score",
			},
		},
	}
	
	for _, param := range paramSpace.Params {
		fmt.Printf("  • %s (%s): ", param.Name, param.Type)
		switch param.Type {
		case optimization.ParamTypeInt, optimization.ParamTypeFloat:
			fmt.Printf("%.1f - %.1f (默认: %v)\n", param.Min, param.Max, param.Default)
		case optimization.ParamTypeChoice:
			fmt.Printf("%v (默认: %v)\n", param.Values, param.Default)
		}
	}
	fmt.Println()

	// 运行优化
	fmt.Println("5. 运行贝叶斯优化...")
	fmt.Println("这将尝试 20 次不同的参数组合...")
	
	result, err := optimizer.Optimize(ctx, strategyID, paramSpace, optimization.OptimizeOptions{
		MaxIterations:    20,
		TargetMetric:     "overall_score",
		MinSampleSize:    10,
		AcquisitionType:  "EI",
		ExplorationRatio: 0.15,
	})
	if err != nil {
		panic(err)
	}
	
	fmt.Println("✓ 优化完成！")

	// 显示优化结果
	fmt.Println("6. 优化结果:")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	
	fmt.Printf("\n🎯 最佳参数:\n")
	for name, value := range result.BestParams {
		fmt.Printf("  • %s: %v\n", name, value)
	}
	
	fmt.Printf("\n📊 性能提升:\n")
	fmt.Printf("  优化前得分: %.3f\n", result.PreviousScore)
	fmt.Printf("  优化后得分: %.3f\n", result.BestScore)
	fmt.Printf("  提升幅度:   %.2f%%\n", result.Improvement)
	
	fmt.Printf("\n⏱️  优化统计:\n")
	fmt.Printf("  迭代次数: %d\n", result.Iterations)
	fmt.Printf("  耗时:     %v\n", result.Duration)
	
	if result.Improvement > 10 {
		fmt.Println("\n✨ 显著提升！建议应用优化后的参数")
	} else if result.Improvement > 0 {
		fmt.Println("\n📈 有所提升，可以考虑应用")
	} else {
		fmt.Println("\n🤔 当前参数已经不错，暂无需调整")
	}
	
	fmt.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// 显示优化历史
	fmt.Println("\n7. 优化历史（前 5 次迭代）:")
	for i, step := range result.History {
		if i >= 5 {
			break
		}
		fmt.Printf("  迭代 %2d: 得分 %.3f, 参数 %v\n", 
			step.Iteration, step.Score, formatParams(step.Params))
	}
	if len(result.History) > 5 {
		fmt.Printf("  ... 还有 %d 次迭代\n", len(result.History)-5)
	}

	// 参数验证示例
	fmt.Println("\n8. 参数验证示例:")
	
	validParams := map[string]interface{}{
		"top_k":           15,
		"temperature":     0.8,
		"rerank_strategy": "mmr",
	}
	
	if err := optimizer.ValidateParams(validParams, paramSpace); err != nil {
		fmt.Printf("  ❌ 验证失败: %v\n", err)
	} else {
		fmt.Printf("  ✅ 参数有效: %v\n", validParams)
	}
	
	invalidParams := map[string]interface{}{
		"top_k":           100, // 超出范围
		"temperature":     0.8,
		"rerank_strategy": "mmr",
	}
	
	if err := optimizer.ValidateParams(invalidParams, paramSpace); err != nil {
		fmt.Printf("  ❌ 参数无效: %v\n", err)
	}

	// 建议参数
	fmt.Println("\n9. 获取参数建议:")
	suggested, err := optimizer.SuggestParams(ctx, strategyID, paramSpace)
	if err != nil {
		panic(err)
	}
	fmt.Printf("  💡 建议参数: %v\n", suggested)

	fmt.Println("\n=== 示例完成 ===")
	
	// 展示自动调优的说明
	fmt.Println("\n💡 提示: 自动调优功能")
	fmt.Println("可以使用 AutoTune() 持续监控和优化参数:")
	fmt.Println()
	fmt.Println("```go")
	fmt.Println("go optimizer.AutoTune(ctx, strategyID, paramSpace, optimization.AutoTuneConfig{")
	fmt.Println("    CheckInterval:  1 * time.Hour,  // 每小时检查一次")
	fmt.Println("    ScoreThreshold: 0.7,            // 低于 0.7 时触发优化")
	fmt.Println("})")
	fmt.Println("```")
}

func createTestData(ctx context.Context, collector feedback.Collector, strategyID string, count int) {
	for i := 0; i < count; i++ {
		queryID := uuid.New().String()
		
		query := &feedback.Query{
			ID:        queryID,
			Text:      fmt.Sprintf("测试查询 %d", i+1),
			UserID:    fmt.Sprintf("user-%d", i%5),
			Strategy:  strategyID,
			Timestamp: time.Now().Add(-time.Duration(count-i) * time.Minute),
		}
		collector.RecordQuery(ctx, query)
		
		// 模拟检索结果
		numResults := 3 + (i % 3)
		results := make([]types.Document, numResults)
		for j := 0; j < numResults; j++ {
			results[j] = types.Document{
				ID:      fmt.Sprintf("doc-%d-%d", i, j),
				Content: fmt.Sprintf("文档内容 %d-%d", i, j),
			}
		}
		collector.RecordResults(ctx, queryID, results)
		
		// 模拟用户反馈（随机质量）
		rating := 3 + (i % 3)
		collector.CollectExplicitFeedback(ctx, &feedback.ExplicitFeedback{
			QueryID:   queryID,
			Type:      feedback.FeedbackTypeRating,
			Rating:    rating,
			Timestamp: time.Now(),
		})
		
		// 模拟用户点击行为
		if rating >= 4 {
			collector.CollectImplicitFeedback(ctx, &feedback.ImplicitFeedback{
				QueryID:    queryID,
				DocumentID: results[0].ID,
				Action:     feedback.ActionClick,
				Timestamp:  time.Now(),
			})
			
			if rating == 5 {
				collector.CollectImplicitFeedback(ctx, &feedback.ImplicitFeedback{
					QueryID:    queryID,
					DocumentID: results[0].ID,
					Action:     feedback.ActionRead,
					Duration:   time.Duration(30+i*5) * time.Second,
					Timestamp:  time.Now(),
				})
			}
		}
	}
}

func formatParams(params map[string]interface{}) string {
	result := "{"
	first := true
	for k, v := range params {
		if !first {
			result += ", "
		}
		result += fmt.Sprintf("%s:%v", k, v)
		first = false
	}
	result += "}"
	return result
}
