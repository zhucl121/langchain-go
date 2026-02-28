package main

import (
	"context"
	"fmt"
	"time"

	"github.com/zhucl121/langchain-go/retrieval/learning/abtest"
	"github.com/zhucl121/langchain-go/retrieval/learning/evaluation"
)

func main() {
	fmt.Println("=== LangChain-Go Learning Retrieval - A/B 测试示例 ===")

	// 创建 A/B 测试管理器
	storage := abtest.NewMemoryStorage()
	manager := abtest.NewManager(storage)
	ctx := context.Background()

	// 1. 创建实验
	fmt.Println("1. 创建 A/B 测试实验")
	experiment := &abtest.Experiment{
		ID:          "exp-search-strategy",
		Name:        "检索策略对比实验",
		Description: "对比 Hybrid Search vs Vector Search 的效果",
		Variants: []abtest.Variant{
			{
				ID:       "control",
				Name:     "对照组 - Hybrid Search",
				Strategy: "hybrid",
				Params: map[string]interface{}{
					"top_k":       10,
					"temperature": 0.7,
				},
				Weight: 0.5,
			},
			{
				ID:       "treatment",
				Name:     "实验组 - Vector Search",
				Strategy: "vector",
				Params: map[string]interface{}{
					"top_k":       15,
					"temperature": 0.8,
				},
				Weight: 0.5,
			},
		},
		Traffic: 1.0, // 100% 流量参与实验
	}

	if err := manager.CreateExperiment(ctx, experiment); err != nil {
		panic(err)
	}

	fmt.Printf("✓ 实验创建成功\n")
	fmt.Printf("  实验 ID: %s\n", experiment.ID)
	fmt.Printf("  实验名称: %s\n", experiment.Name)
	fmt.Printf("  变体数: %d\n", len(experiment.Variants))
	for _, v := range experiment.Variants {
		fmt.Printf("    • %s (%s) - 权重: %.0f%%\n", v.Name, v.ID, v.Weight*100)
	}
	fmt.Println()

	// 2. 开始实验
	fmt.Println("2. 开始实验")
	if err := manager.StartExperiment(ctx, experiment.ID); err != nil {
		panic(err)
	}
	fmt.Println("✓ 实验已开始运行")

	// 3. 用户分流演示
	fmt.Println("3. 用户分流演示")
	users := []string{"alice", "bob", "charlie", "david", "eve"}
	
	assignments := make(map[string]string)
	for _, userID := range users {
		variantID, err := manager.AssignVariant(ctx, userID, experiment.ID)
		if err != nil {
			panic(err)
		}
		assignments[userID] = variantID
		
		variantName := "未知"
		for _, v := range experiment.Variants {
			if v.ID == variantID {
				variantName = v.Name
				break
			}
		}
		fmt.Printf("  用户 %-8s -> %s\n", userID, variantName)
	}
	fmt.Println()

	// 4. 模拟收集实验数据
	fmt.Println("4. 模拟收集实验数据...")
	
	// 对照组（hybrid）- 基准性能
	fmt.Println("  • 对照组收集中...")
	for i := 0; i < 100; i++ {
		userID := fmt.Sprintf("user-control-%d", i)
		manager.AssignVariant(ctx, userID, experiment.ID) // 确保分配
		
		// 模拟性能：平均 0.65
		score := 0.60 + float64(i%20)/100.0
		
		manager.RecordResult(ctx, &abtest.ExperimentResult{
			ExperimentID: experiment.ID,
			VariantID:    "control",
			UserID:       userID,
			QueryID:      fmt.Sprintf("query-%d", i),
			Metrics: evaluation.QueryMetrics{
				OverallScore: score,
				NDCG:         score * 0.9,
				MRR:          score * 0.85,
				AvgRating:    3.5 + score,
				CTR:          0.3 + score*0.2,
			},
			Timestamp: time.Now(),
		})
	}

	// 实验组（vector）- 改进性能
	fmt.Println("  • 实验组收集中...")
	for i := 0; i < 100; i++ {
		userID := fmt.Sprintf("user-treatment-%d", i)
		manager.AssignVariant(ctx, userID, experiment.ID)
		
		// 模拟性能：平均 0.75（明显提升）
		score := 0.70 + float64(i%20)/100.0
		
		manager.RecordResult(ctx, &abtest.ExperimentResult{
			ExperimentID: experiment.ID,
			VariantID:    "treatment",
			UserID:       userID,
			QueryID:      fmt.Sprintf("query-%d", i+100),
			Metrics: evaluation.QueryMetrics{
				OverallScore: score,
				NDCG:         score * 0.95,
				MRR:          score * 0.9,
				AvgRating:    4.0 + score*0.5,
				CTR:          0.4 + score*0.3,
			},
			Timestamp: time.Now(),
		})
	}
	fmt.Println("✓ 数据收集完成 (每组 100 个样本)")

	// 5. 分析实验结果
	fmt.Println("5. 分析实验结果")
	analysis, err := manager.AnalyzeExperiment(ctx, experiment.ID)
	if err != nil {
		panic(err)
	}

	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	// 显示各变体的指标
	fmt.Println("📊 各变体性能:")
	for variantID, metrics := range analysis.Variants {
		variantName := variantID
		for _, v := range experiment.Variants {
			if v.ID == variantID {
				variantName = v.Name
				break
			}
		}
		
		fmt.Printf("\n  %s:\n", variantName)
		fmt.Printf("    样本数:     %d\n", metrics.SampleSize)
		fmt.Printf("    平均得分:   %.3f\n", metrics.AvgScore)
		fmt.Printf("    标准差:     %.3f\n", metrics.StdDev)
		fmt.Printf("    置信区间:   [%.3f, %.3f]\n", 
			metrics.ConfInterval[0], metrics.ConfInterval[1])
	}

	fmt.Println()
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// 显示统计分析结果
	fmt.Println()
	fmt.Println("🏆 实验结论:")
	
	if analysis.Winner != "" {
		winnerName := analysis.Winner
		for _, v := range experiment.Variants {
			if v.ID == analysis.Winner {
				winnerName = v.Name
				break
			}
		}
		fmt.Printf("  获胜者: %s\n", winnerName)
	}
	
	fmt.Printf("  置信度: %.2f%%\n", analysis.Confidence*100)
	fmt.Printf("  P-Value: %.3f\n", analysis.PValue)
	
	if analysis.Completed {
		fmt.Println("  ✅ 结果具有统计显著性 (p < 0.05)")
		
		// 计算提升幅度
		controlScore := analysis.Variants["control"].AvgScore
		treatmentScore := analysis.Variants["treatment"].AvgScore
		improvement := ((treatmentScore - controlScore) / controlScore) * 100
		
		fmt.Printf("  📈 性能提升: %.2f%%\n", improvement)
		fmt.Println()
		fmt.Println("💡 建议: 可以将实验组策略推广到全量用户")
	} else {
		fmt.Println("  ⚠️  样本不足或差异不显著，建议继续收集数据")
	}

	fmt.Println()
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// 6. 结束实验
	fmt.Println()
	fmt.Println("6. 结束实验")
	if err := manager.EndExperiment(ctx, experiment.ID, analysis.Winner); err != nil {
		panic(err)
	}
	fmt.Printf("✓ 实验已结束，获胜者: %s\n", analysis.Winner)

	// 7. 查看实验列表
	fmt.Println()
	fmt.Println("7. 查看实验列表")
	
	// 创建更多示例实验
	manager.CreateExperiment(ctx, &abtest.Experiment{
		ID:   "exp-002",
		Name: "参数调优实验",
		Variants: []abtest.Variant{
			{ID: "v1", Weight: 0.5},
			{ID: "v2", Weight: 0.5},
		},
		Traffic: 1.0,
		Status:  abtest.StatusDraft,
	})
	
	manager.CreateExperiment(ctx, &abtest.Experiment{
		ID:   "exp-003",
		Name: "模型对比实验",
		Variants: []abtest.Variant{
			{ID: "v1", Weight: 0.5},
			{ID: "v2", Weight: 0.5},
		},
		Traffic: 1.0,
		Status:  abtest.StatusRunning,
	})

	// 列出所有实验
	allExps, _ := manager.ListExperiments(ctx, "")
	fmt.Printf("  总实验数: %d\n", len(allExps))
	
	// 按状态统计
	statusCount := make(map[abtest.ExperimentStatus]int)
	for _, exp := range allExps {
		statusCount[exp.Status]++
	}
	
	for status, count := range statusCount {
		fmt.Printf("    %s: %d\n", status, count)
	}

	fmt.Println()
	fmt.Println("=== 示例完成 ===")
	
	// 展示最佳实践
	fmt.Println()
	fmt.Println("💡 A/B 测试最佳实践:")
	fmt.Println("  1. 确保样本量充足（每组至少 30 个样本）")
	fmt.Println("  2. 控制变量，只改变一个因素")
	fmt.Println("  3. 注意统计显著性（p-value < 0.05）")
	fmt.Println("  4. 考虑实际业务价值，不只看统计结果")
	fmt.Println("  5. 长期监控，避免新奇效应")
}
