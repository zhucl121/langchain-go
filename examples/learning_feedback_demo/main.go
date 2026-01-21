package main

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/zhucl121/langchain-go/pkg/types"
	"github.com/zhucl121/langchain-go/retrieval/learning/feedback"
)

func main() {
	fmt.Println("=== LangChain-Go Learning Retrieval - 反馈收集示例 ===\n")

	// 创建内存存储
	storage := feedback.NewMemoryStorage()
	collector := feedback.NewCollector(storage)
	ctx := context.Background()

	// 示例 1: 记录查询和结果
	fmt.Println("1. 记录查询和检索结果")
	query := &feedback.Query{
		ID:        uuid.New().String(),
		Text:      "什么是机器学习？",
		UserID:    "user-123",
		Strategy:  "hybrid",
		Timestamp: time.Now(),
		Metadata: map[string]interface{}{
			"source": "web",
			"lang":   "zh",
		},
	}

	if err := collector.RecordQuery(ctx, query); err != nil {
		panic(err)
	}
	fmt.Printf("✓ 已记录查询: %s\n", query.Text)

	// 模拟检索结果
	results := []types.Document{
		{
			ID:      "doc-1",
			Content: "机器学习是人工智能的一个分支...",
			Metadata: map[string]interface{}{
				"score": 0.95,
				"rank":  1,
			},
		},
		{
			ID:      "doc-2",
			Content: "机器学习包括监督学习、无监督学习...",
			Metadata: map[string]interface{}{
				"score": 0.88,
				"rank":  2,
			},
		},
	}

	if err := collector.RecordResults(ctx, query.ID, results); err != nil {
		panic(err)
	}
	fmt.Printf("✓ 已记录 %d 个检索结果\n\n", len(results))

	// 示例 2: 收集显式反馈
	fmt.Println("2. 收集显式反馈")

	// 用户点赞
	explicitFB := &feedback.ExplicitFeedback{
		QueryID:   query.ID,
		UserID:    "user-123",
		Type:      feedback.FeedbackTypePositive,
		Timestamp: time.Now(),
	}
	if err := collector.CollectExplicitFeedback(ctx, explicitFB); err != nil {
		panic(err)
	}
	fmt.Println("✓ 用户点赞")

	// 用户评分
	ratingFB := &feedback.ExplicitFeedback{
		QueryID:   query.ID,
		UserID:    "user-123",
		Type:      feedback.FeedbackTypeRating,
		Rating:    5,
		Comment:   "结果很准确，帮助很大！",
		Timestamp: time.Now(),
	}
	if err := collector.CollectExplicitFeedback(ctx, ratingFB); err != nil {
		panic(err)
	}
	fmt.Printf("✓ 用户评分: %d 星\n\n", ratingFB.Rating)

	// 示例 3: 收集隐式反馈
	fmt.Println("3. 收集隐式反馈")

	// 用户点击第一个文档
	clickFB := &feedback.ImplicitFeedback{
		QueryID:    query.ID,
		UserID:     "user-123",
		DocumentID: "doc-1",
		Action:     feedback.ActionClick,
		Duration:   0,
		Timestamp:  time.Now(),
	}
	if err := collector.CollectImplicitFeedback(ctx, clickFB); err != nil {
		panic(err)
	}
	fmt.Println("✓ 用户点击了第一个文档")

	// 用户阅读文档
	readFB := &feedback.ImplicitFeedback{
		QueryID:    query.ID,
		UserID:     "user-123",
		DocumentID: "doc-1",
		Action:     feedback.ActionRead,
		Duration:   120 * time.Second, // 阅读了 2 分钟
		Timestamp:  time.Now(),
	}
	if err := collector.CollectImplicitFeedback(ctx, readFB); err != nil {
		panic(err)
	}
	fmt.Println("✓ 用户阅读文档 120 秒\n")

	// 示例 4: 获取查询反馈
	fmt.Println("4. 获取查询反馈汇总")
	qf, err := collector.GetQueryFeedback(ctx, query.ID)
	if err != nil {
		panic(err)
	}

	fmt.Printf("查询: %s\n", qf.Query.Text)
	fmt.Printf("策略: %s\n", qf.Query.Strategy)
	fmt.Printf("结果数: %d\n", len(qf.Results))
	fmt.Printf("平均评分: %.1f/5\n", qf.AvgRating)
	fmt.Printf("点击率: %.1f%%\n", qf.CTR*100)
	fmt.Printf("平均阅读时长: %v\n\n", qf.AvgReadDuration)

	// 示例 5: 模拟更多查询
	fmt.Println("5. 模拟更多查询...")
	for i := 0; i < 5; i++ {
		q := &feedback.Query{
			ID:        uuid.New().String(),
			Text:      fmt.Sprintf("测试查询 %d", i+1),
			UserID:    fmt.Sprintf("user-%d", i),
			Strategy:  "hybrid",
			Timestamp: time.Now(),
		}
		collector.RecordQuery(ctx, q)

		// 模拟不同的评分
		rating := 3 + (i % 3)
		fb := &feedback.ExplicitFeedback{
			QueryID:   q.ID,
			UserID:    q.UserID,
			Type:      feedback.FeedbackTypeRating,
			Rating:    rating,
			Timestamp: time.Now(),
		}
		collector.CollectExplicitFeedback(ctx, fb)
	}
	fmt.Println("✓ 已添加 5 个测试查询\n")

	// 示例 6: 聚合统计
	fmt.Println("6. 聚合统计")
	stats, err := collector.AggregateStats(ctx, feedback.AggregateOptions{
		TimeRange: 1 * time.Hour,
		MinRating: 3,
	})
	if err != nil {
		panic(err)
	}

	fmt.Printf("总查询数: %d\n", stats.TotalQueries)
	fmt.Printf("平均评分: %.2f/5\n", stats.AvgRating)
	fmt.Printf("正面反馈率: %.1f%%\n", stats.PositiveRate*100)
	fmt.Printf("负面反馈率: %.1f%%\n", stats.NegativeRate*100)
	fmt.Printf("平均点击率: %.1f%%\n", stats.AvgCTR*100)
	fmt.Printf("平均阅读时长: %v\n", stats.AvgReadDuration)

	if len(stats.LowRatingQueries) > 0 {
		fmt.Printf("\n低评分查询:\n")
		for _, q := range stats.LowRatingQueries {
			fmt.Printf("  - %s\n", q)
		}
	}

	fmt.Println("\n=== 示例完成 ===")
}
