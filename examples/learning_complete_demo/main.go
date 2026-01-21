package main

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/zhucl121/langchain-go/pkg/types"
	"github.com/zhucl121/langchain-go/retrieval/learning/abtest"
	"github.com/zhucl121/langchain-go/retrieval/learning/evaluation"
	"github.com/zhucl121/langchain-go/retrieval/learning/feedback"
	"github.com/zhucl121/langchain-go/retrieval/learning/optimization"
)

func main() {
	fmt.Println("╔═══════════════════════════════════════════════════════╗")
	fmt.Println("║  LangChain-Go Learning Retrieval - 完整工作流示例      ║")
	fmt.Println("╚═══════════════════════════════════════════════════════╝")
	fmt.Println()

	ctx := context.Background()

	// ============================================
	// 步骤 1: 初始化系统
	// ============================================
	fmt.Println("📦 步骤 1: 初始化学习型检索系统")
	fmt.Println("─────────────────────────────────────────")

	// 创建反馈收集器
	feedbackStorage := feedback.NewMemoryStorage()
	collector := feedback.NewCollector(feedbackStorage)
	fmt.Println("✓ 反馈收集器已创建")

	// 创建质量评估器
	evaluator := evaluation.NewEvaluator(collector)
	fmt.Println("✓ 质量评估器已创建")

	// 创建参数优化器
	optimizer := optimization.NewOptimizer(evaluator, collector, optimization.DefaultConfig())
	fmt.Println("✓ 参数优化器已创建")

	// 创建 A/B 测试管理器
	abtestStorage := abtest.NewMemoryStorage()
	abtestManager := abtest.NewManager(abtestStorage)
	fmt.Println("✓ A/B 测试管理器已创建")
	fmt.Println()

	// ============================================
	// 步骤 2: 收集用户反馈
	// ============================================
	fmt.Println("📊 步骤 2: 收集用户反馈数据")
	fmt.Println("─────────────────────────────────────────")

	strategyID := "hybrid-search"
	fmt.Printf("收集策略 '%s' 的用户反馈...\n", strategyID)

	// 模拟 30 个用户查询
	for i := 0; i < 30; i++ {
		queryID := uuid.New().String()
		query := &feedback.Query{
			ID:        queryID,
			Text:      fmt.Sprintf("用户查询 %d", i+1),
			UserID:    fmt.Sprintf("user-%d", i),
			Strategy:  strategyID,
			Timestamp: time.Now(),
		}
		collector.RecordQuery(ctx, query)

		// 模拟检索结果
		results := []types.Document{
			{ID: fmt.Sprintf("doc-%d-1", i), Content: "文档内容 1"},
			{ID: fmt.Sprintf("doc-%d-2", i), Content: "文档内容 2"},
			{ID: fmt.Sprintf("doc-%d-3", i), Content: "文档内容 3"},
		}
		collector.RecordResults(ctx, queryID, results)

		// 模拟用户反馈（质量参差不齐）
		rating := 3 + (i % 3)
		collector.CollectExplicitFeedback(ctx, &feedback.ExplicitFeedback{
			QueryID:   queryID,
			Type:      feedback.FeedbackTypeRating,
			Rating:    rating,
			Timestamp: time.Now(),
		})

		// 高分查询模拟点击行为
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
					Duration:   time.Duration(30+i) * time.Second,
					Timestamp:  time.Now(),
				})
			}
		}
	}

	// 显示反馈统计
	stats, _ := collector.AggregateStats(ctx, feedback.AggregateOptions{
		TimeRange: 1 * time.Hour,
	})

	fmt.Printf("✓ 已收集 %d 个查询的反馈\n", stats.TotalQueries)
	fmt.Printf("  • 平均评分: %.1f/5.0\n", stats.AvgRating)
	fmt.Printf("  • 平均点击率: %.1f%%\n", stats.AvgCTR*100)
	fmt.Println()

	// ============================================
	// 步骤 3: 评估检索质量
	// ============================================
	fmt.Println("🎯 步骤 3: 评估检索质量")
	fmt.Println("─────────────────────────────────────────")

	strategyMetrics, err := evaluator.EvaluateStrategy(ctx, strategyID, evaluation.EvaluateOptions{
		TimeRange:     1 * time.Hour,
		MinSampleSize: 10,
	})
	if err != nil {
		panic(err)
	}

	fmt.Printf("策略性能评估:\n")
	fmt.Printf("  • 综合得分: %.3f\n", strategyMetrics.AvgMetrics.OverallScore)
	fmt.Printf("  • NDCG: %.3f\n", strategyMetrics.AvgMetrics.NDCG)
	fmt.Printf("  • MRR: %.3f\n", strategyMetrics.AvgMetrics.MRR)
	fmt.Printf("  • F1 Score: %.3f\n", strategyMetrics.AvgMetrics.F1Score)

	if strategyMetrics.AvgMetrics.OverallScore < 0.7 {
		fmt.Println("\n⚠️  性能低于预期，建议进行参数优化")
	}
	fmt.Println()

	// ============================================
	// 步骤 4: 自动参数优化
	// ============================================
	fmt.Println("⚙️  步骤 4: 自动参数优化")
	fmt.Println("─────────────────────────────────────────")

	// 定义参数空间
	paramSpace := optimization.ParameterSpace{
		Params: []optimization.Parameter{
			{Name: "top_k", Type: optimization.ParamTypeInt, Min: 5, Max: 30, Default: 10},
			{Name: "temperature", Type: optimization.ParamTypeFloat, Min: 0.1, Max: 1.0, Default: 0.7},
			{Name: "rerank", Type: optimization.ParamTypeChoice, Values: []string{"score", "diversity", "mmr"}, Default: "score"},
		},
	}

	fmt.Println("运行贝叶斯优化（15 次迭代）...")
	optimizeResult, err := optimizer.Optimize(ctx, strategyID, paramSpace, optimization.OptimizeOptions{
		MaxIterations:    15,
		TargetMetric:     "overall_score",
		MinSampleSize:    10,
		ExplorationRatio: 0.15,
	})
	if err != nil {
		panic(err)
	}

	fmt.Println("✓ 参数优化完成")
	fmt.Printf("  • 最佳参数: %v\n", formatParams(optimizeResult.BestParams))
	fmt.Printf("  • 优化前得分: %.3f\n", optimizeResult.PreviousScore)
	fmt.Printf("  • 优化后得分: %.3f\n", optimizeResult.BestScore)
	fmt.Printf("  • 性能提升: %.2f%%\n", optimizeResult.Improvement)
	fmt.Println()

	// ============================================
	// 步骤 5: A/B 测试验证
	// ============================================
	fmt.Println("🧪 步骤 5: A/B 测试验证优化效果")
	fmt.Println("─────────────────────────────────────────")

	// 创建 A/B 测试实验
	experiment := &abtest.Experiment{
		ID:          "exp-optimization-validation",
		Name:        "参数优化效果验证",
		Description: "对比优化前后的性能",
		Variants: []abtest.Variant{
			{
				ID:       "control",
				Name:     "优化前（当前参数）",
				Strategy: strategyID,
				Params:   map[string]interface{}{"top_k": 10, "temperature": 0.7},
				Weight:   0.5,
			},
			{
				ID:       "treatment",
				Name:     "优化后（最佳参数）",
				Strategy: strategyID,
				Params:   optimizeResult.BestParams,
				Weight:   0.5,
			},
		},
		Traffic: 1.0,
	}

	abtestManager.CreateExperiment(ctx, experiment)
	abtestManager.StartExperiment(ctx, experiment.ID)
	fmt.Println("✓ A/B 测试实验已创建并启动")

	// 模拟收集实验数据
	fmt.Println("收集实验数据...")

	// 对照组
	for i := 0; i < 50; i++ {
		userID := fmt.Sprintf("ab-control-%d", i)
		abtestManager.AssignVariant(ctx, userID, experiment.ID)

		score := 0.60 + float64(i%15)/100.0
		abtestManager.RecordResult(ctx, &abtest.ExperimentResult{
			ExperimentID: experiment.ID,
			VariantID:    "control",
			UserID:       userID,
			Metrics: evaluation.QueryMetrics{
				OverallScore: score,
			},
			Timestamp: time.Now(),
		})
	}

	// 实验组（优化后性能更好）
	for i := 0; i < 50; i++ {
		userID := fmt.Sprintf("ab-treatment-%d", i)
		abtestManager.AssignVariant(ctx, userID, experiment.ID)

		score := 0.68 + float64(i%15)/100.0 // 明显提升
		abtestManager.RecordResult(ctx, &abtest.ExperimentResult{
			ExperimentID: experiment.ID,
			VariantID:    "treatment",
			UserID:       userID,
			Metrics: evaluation.QueryMetrics{
				OverallScore: score,
			},
			Timestamp: time.Now(),
		})
	}

	// 分析实验结果
	analysis, err := abtestManager.AnalyzeExperiment(ctx, experiment.ID)
	if err != nil {
		panic(err)
	}

	fmt.Printf("✓ 实验数据收集完成 (每组 50 个样本)\n\n")

	fmt.Println("实验结果:")
	for variantID, metrics := range analysis.Variants {
		variantName := variantID
		for _, v := range experiment.Variants {
			if v.ID == variantID {
				variantName = v.Name
				break
			}
		}
		fmt.Printf("  %s:\n", variantName)
		fmt.Printf("    • 平均得分: %.3f\n", metrics.AvgScore)
		fmt.Printf("    • 置信区间: [%.3f, %.3f]\n", metrics.ConfInterval[0], metrics.ConfInterval[1])
	}

	fmt.Println()
	if analysis.Completed {
		fmt.Printf("🏆 获胜者: %s\n", getVariantName(experiment, analysis.Winner))
		fmt.Printf("📈 提升: %.2f%%\n", (analysis.Variants[analysis.Winner].AvgScore-analysis.Variants["control"].AvgScore)/analysis.Variants["control"].AvgScore*100)
		fmt.Printf("✅ 统计显著性: p = %.3f (p < 0.05)\n", analysis.PValue)
		fmt.Println()
		fmt.Println("💡 优化效果已通过 A/B 测试验证，可以推广到生产环境")
	}

	abtestManager.EndExperiment(ctx, experiment.ID, analysis.Winner)
	fmt.Println()

	// ============================================
	// 步骤 6: 完整工作流总结
	// ============================================
	fmt.Println("📋 步骤 6: 学习型检索完整工作流总结")
	fmt.Println("─────────────────────────────────────────")
	fmt.Println()

	fmt.Println("1️⃣  反馈收集:")
	fmt.Printf("   ✓ 收集了 %d 个查询的反馈数据\n", stats.TotalQueries)
	fmt.Printf("   ✓ 包含显式反馈（评分）和隐式反馈（点击、阅读）\n")
	fmt.Println()

	fmt.Println("2️⃣  质量评估:")
	fmt.Printf("   ✓ 计算了多维度评估指标\n")
	fmt.Printf("   ✓ 综合得分: %.3f\n", strategyMetrics.AvgMetrics.OverallScore)
	fmt.Println()

	fmt.Println("3️⃣  参数优化:")
	fmt.Printf("   ✓ 通过 %d 次迭代找到最佳参数\n", optimizeResult.Iterations)
	fmt.Printf("   ✓ 性能提升: %.2f%%\n", optimizeResult.Improvement)
	fmt.Println()

	fmt.Println("4️⃣  A/B 测试验证:")
	fmt.Printf("   ✓ 实验组相比对照组提升明显\n")
	fmt.Printf("   ✓ 统计显著性: p = %.3f\n", analysis.PValue)
	fmt.Println()

	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()
	fmt.Println("🎉 学习型检索系统工作流完成！")
	fmt.Println()
	fmt.Println("📈 整体效果:")
	fmt.Printf("   • 从用户反馈中学习\n")
	fmt.Printf("   • 自动发现性能问题\n")
	fmt.Printf("   • 智能优化参数配置\n")
	fmt.Printf("   • 科学验证优化效果\n")
	fmt.Printf("   • 持续提升检索质量\n")
	fmt.Println()

	fmt.Println("💡 生产环境建议:")
	fmt.Println("   1. 使用 PostgreSQL 存储持久化数据")
	fmt.Println("   2. 开启 AutoTune 持续监控和优化")
	fmt.Println("   3. 定期运行 A/B 测试验证效果")
	fmt.Println("   4. 关注用户满意度和业务指标")
	fmt.Println("   5. 建立完整的监控和告警体系")
	fmt.Println()

	fmt.Println("╔═══════════════════════════════════════════════════════╗")
	fmt.Println("║              感谢使用 LangChain-Go！                   ║")
	fmt.Println("╚═══════════════════════════════════════════════════════╝")
}

func getVariantName(experiment *abtest.Experiment, variantID string) string {
	for _, v := range experiment.Variants {
		if v.ID == variantID {
			return v.Name
		}
	}
	return variantID
}

func formatParams(params map[string]interface{}) string {
	result := ""
	first := true
	for k, v := range params {
		if !first {
			result += ", "
		}
		result += fmt.Sprintf("%s=%v", k, v)
		first = false
	}
	return result
}
