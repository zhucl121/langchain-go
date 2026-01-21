package main

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/zhucl121/langchain-go/pkg/types"
	"github.com/zhucl121/langchain-go/retrieval/learning/evaluation"
	"github.com/zhucl121/langchain-go/retrieval/learning/feedback"
)

func main() {
	fmt.Println("=== LangChain-Go Learning Retrieval - 质量评估示例 ===\n")

	// 创建反馈收集器
	storage := feedback.NewMemoryStorage()
	collector := feedback.NewCollector(storage)
	ctx := context.Background()

	// 创建评估器
	evaluator := evaluation.NewEvaluator(collector)

	// 示例 1: 评估单个查询
	fmt.Println("1. 评估单个查询")
	queryID := createSampleQuery(ctx, collector, "什么是深度学习？", "hybrid", 5)
	
	qf, _ := collector.GetQueryFeedback(ctx, queryID)
	metrics, err := evaluator.EvaluateQuery(ctx, qf)
	if err != nil {
		panic(err)
	}

	fmt.Printf("查询: %s\n", qf.Query.Text)
	fmt.Printf("策略: %s\n", qf.Query.Strategy)
	printMetrics(metrics)
	fmt.Println()

	// 示例 2: 评估策略
	fmt.Println("2. 评估检索策略")
	
	// 创建多个查询用于评估
	fmt.Println("创建测试数据...")
	for i := 0; i < 10; i++ {
		rating := 3 + (i % 3)
		createSampleQuery(ctx, collector, fmt.Sprintf("测试查询 %d", i+1), "hybrid", rating)
	}

	strategyMetrics, err := evaluator.EvaluateStrategy(ctx, "hybrid", evaluation.EvaluateOptions{
		TimeRange:     1 * time.Hour,
		MinSampleSize: 5,
	})
	if err != nil {
		panic(err)
	}

	fmt.Printf("\n策略 ID: %s\n", strategyMetrics.StrategyID)
	fmt.Printf("总查询数: %d\n", strategyMetrics.TotalQueries)
	fmt.Println("\n平均指标:")
	printMetrics(&strategyMetrics.AvgMetrics)
	fmt.Println()

	// 示例 3: 对比两个策略
	fmt.Println("3. 对比两个检索策略")
	
	// 创建第二个策略的数据
	fmt.Println("创建策略 B 的测试数据...")
	for i := 0; i < 10; i++ {
		rating := 2 + (i % 2)
		createSampleQuery(ctx, collector, fmt.Sprintf("策略B查询 %d", i+1), "vector", rating)
	}

	comparison, err := evaluator.CompareStrategies(ctx, "hybrid", "vector")
	if err != nil {
		panic(err)
	}

	fmt.Printf("\n策略对比结果:\n")
	fmt.Printf("策略 A: %s (%.2f 分)\n", comparison.StrategyA.StrategyID, comparison.StrategyA.AvgMetrics.OverallScore)
	fmt.Printf("策略 B: %s (%.2f 分)\n", comparison.StrategyB.StrategyID, comparison.StrategyB.AvgMetrics.OverallScore)
	fmt.Printf("\n🏆 获胜者: %s\n", comparison.Winner)
	fmt.Printf("📈 提升: %.2f%%\n", comparison.Improvement)
	fmt.Printf("✅ 置信度: %.2f%%\n", comparison.Confidence*100)
	fmt.Printf("📊 显著性 (p-value): %.3f\n", comparison.SignificantAt)
	
	if comparison.SignificantAt < 0.05 {
		fmt.Println("✨ 结果具有统计显著性 (p < 0.05)")
	}
	fmt.Println()

	// 示例 4: 相关性模型演示
	fmt.Println("4. 相关性模型演示")
	demonstrateRelevanceModel(ctx, collector)

	fmt.Println("\n=== 示例完成 ===")
}

func createSampleQuery(ctx context.Context, collector feedback.Collector, text, strategy string, rating int) string {
	queryID := uuid.New().String()

	// 记录查询
	query := &feedback.Query{
		ID:        queryID,
		Text:      text,
		UserID:    "demo-user",
		Strategy:  strategy,
		Timestamp: time.Now(),
	}
	collector.RecordQuery(ctx, query)

	// 记录结果
	results := []types.Document{
		{ID: "doc-1", Content: "相关文档 1"},
		{ID: "doc-2", Content: "相关文档 2"},
		{ID: "doc-3", Content: "相关文档 3"},
	}
	collector.RecordResults(ctx, queryID, results)

	// 添加显式反馈
	collector.CollectExplicitFeedback(ctx, &feedback.ExplicitFeedback{
		QueryID:   queryID,
		UserID:    "demo-user",
		Type:      feedback.FeedbackTypeRating,
		Rating:    rating,
		Timestamp: time.Now(),
	})

	// 添加隐式反馈
	if rating >= 4 {
		collector.CollectImplicitFeedback(ctx, &feedback.ImplicitFeedback{
			QueryID:    queryID,
			UserID:     "demo-user",
			DocumentID: "doc-1",
			Action:     feedback.ActionClick,
			Timestamp:  time.Now(),
		})

		collector.CollectImplicitFeedback(ctx, &feedback.ImplicitFeedback{
			QueryID:    queryID,
			UserID:     "demo-user",
			DocumentID: "doc-1",
			Action:     feedback.ActionRead,
			Duration:   time.Duration(rating*20) * time.Second,
			Timestamp:  time.Now(),
		})
	}

	return queryID
}

func printMetrics(metrics *evaluation.QueryMetrics) {
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	
	fmt.Println("📊 相关性指标:")
	fmt.Printf("  Precision:  %.3f\n", metrics.Precision)
	fmt.Printf("  Recall:     %.3f\n", metrics.Recall)
	fmt.Printf("  F1 Score:   %.3f\n", metrics.F1Score)
	fmt.Printf("  NDCG:       %.3f\n", metrics.NDCG)
	fmt.Printf("  MRR:        %.3f\n", metrics.MRR)

	fmt.Println("\n😊 用户满意度:")
	fmt.Printf("  评分:       %.1f/5.0\n", metrics.AvgRating)
	fmt.Printf("  点击率:     %.1f%%\n", metrics.CTR*100)
	fmt.Printf("  阅读率:     %.1f%%\n", metrics.ReadRate*100)

	fmt.Println("\n⭐ 综合得分:")
	fmt.Printf("  Overall:    %.3f\n", metrics.OverallScore)
	
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
}

func demonstrateRelevanceModel(ctx context.Context, collector feedback.Collector) {
	// 创建一个测试查询
	queryID := uuid.New().String()
	query := &feedback.Query{
		ID:        queryID,
		Text:      "相关性模型测试",
		UserID:    "demo-user",
		Strategy:  "test",
		Timestamp: time.Now(),
	}
	collector.RecordQuery(ctx, query)

	results := []types.Document{
		{ID: "doc-A", Content: "文档 A"},
		{ID: "doc-B", Content: "文档 B"},
		{ID: "doc-C", Content: "文档 C"},
	}
	collector.RecordResults(ctx, queryID, results)

	// doc-A: 只点击
	collector.CollectImplicitFeedback(ctx, &feedback.ImplicitFeedback{
		QueryID:    queryID,
		DocumentID: "doc-A",
		Action:     feedback.ActionClick,
		Timestamp:  time.Now(),
	})

	// doc-B: 点击 + 短时间阅读
	collector.CollectImplicitFeedback(ctx, &feedback.ImplicitFeedback{
		QueryID:    queryID,
		DocumentID: "doc-B",
		Action:     feedback.ActionClick,
		Timestamp:  time.Now(),
	})
	collector.CollectImplicitFeedback(ctx, &feedback.ImplicitFeedback{
		QueryID:    queryID,
		DocumentID: "doc-B",
		Action:     feedback.ActionRead,
		Duration:   30 * time.Second,
		Timestamp:  time.Now(),
	})

	// doc-C: 长时间阅读 + 复制
	collector.CollectImplicitFeedback(ctx, &feedback.ImplicitFeedback{
		QueryID:    queryID,
		DocumentID: "doc-C",
		Action:     feedback.ActionRead,
		Duration:   120 * time.Second,
		Timestamp:  time.Now(),
	})
	collector.CollectImplicitFeedback(ctx, &feedback.ImplicitFeedback{
		QueryID:    queryID,
		DocumentID: "doc-C",
		Action:     feedback.ActionCopy,
		Timestamp:  time.Now(),
	})

	// 评估相关性
	qf, _ := collector.GetQueryFeedback(ctx, queryID)
	
	model := &evaluation.DefaultRelevanceModel{}
	
	fmt.Println("基于用户行为的相关性评分:")
	for _, doc := range results {
		relevance := model.GetRelevance(doc.ID, qf)
		isRelevant := model.IsRelevant(doc.ID, qf)
		
		status := "❌"
		if isRelevant {
			status = "✅"
		}
		
		fmt.Printf("  %s %s: %.3f ", status, doc.ID, relevance)
		
		// 显示用户行为
		actions := []string{}
		for _, fb := range qf.ImplicitFeedback {
			if fb.DocumentID == doc.ID {
				actionStr := string(fb.Action)
				if fb.Action == feedback.ActionRead && fb.Duration > 0 {
					actionStr += fmt.Sprintf("(%ds)", int(fb.Duration.Seconds()))
				}
				actions = append(actions, actionStr)
			}
		}
		if len(actions) > 0 {
			fmt.Printf("(%s)", actions[0])
			if len(actions) > 1 {
				for _, a := range actions[1:] {
					fmt.Printf(" + %s", a)
				}
			}
		}
		fmt.Println()
	}
}
