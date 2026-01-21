package evaluation

import (
	"context"
	"testing"
	"time"

	"github.com/zhucl121/langchain-go/pkg/types"
	"github.com/zhucl121/langchain-go/retrieval/learning/feedback"
)

func setupTestData(t *testing.T) (feedback.Collector, string) {
	storage := feedback.NewMemoryStorage()
	collector := feedback.NewCollector(storage)
	ctx := context.Background()

	// 创建测试查询
	queryID := "test-query-1"
	query := &feedback.Query{
		ID:        queryID,
		Text:      "测试查询",
		UserID:    "user1",
		Strategy:  "hybrid",
		Timestamp: time.Now(),
	}

	if err := collector.RecordQuery(ctx, query); err != nil {
		t.Fatal(err)
	}

	// 添加检索结果
	results := []types.Document{
		{ID: "doc1", Content: "内容1"},
		{ID: "doc2", Content: "内容2"},
		{ID: "doc3", Content: "内容3"},
	}

	if err := collector.RecordResults(ctx, queryID, results); err != nil {
		t.Fatal(err)
	}

	// 添加显式反馈
	if err := collector.CollectExplicitFeedback(ctx, &feedback.ExplicitFeedback{
		QueryID:   queryID,
		UserID:    "user1",
		Type:      feedback.FeedbackTypeRating,
		Rating:    5,
		Timestamp: time.Now(),
	}); err != nil {
		t.Fatal(err)
	}

	// 添加隐式反馈
	if err := collector.CollectImplicitFeedback(ctx, &feedback.ImplicitFeedback{
		QueryID:    queryID,
		UserID:     "user1",
		DocumentID: "doc1",
		Action:     feedback.ActionClick,
		Timestamp:  time.Now(),
	}); err != nil {
		t.Fatal(err)
	}

	if err := collector.CollectImplicitFeedback(ctx, &feedback.ImplicitFeedback{
		QueryID:    queryID,
		UserID:     "user1",
		DocumentID: "doc1",
		Action:     feedback.ActionRead,
		Duration:   60 * time.Second,
		Timestamp:  time.Now(),
	}); err != nil {
		t.Fatal(err)
	}

	return collector, queryID
}

func TestEvaluator_EvaluateQuery(t *testing.T) {
	collector, queryID := setupTestData(t)
	ctx := context.Background()

	evaluator := NewEvaluator(collector)

	// 获取查询反馈
	qf, err := collector.GetQueryFeedback(ctx, queryID)
	if err != nil {
		t.Fatalf("GetQueryFeedback() error = %v", err)
	}

	// 评估查询
	metrics, err := evaluator.EvaluateQuery(ctx, qf)
	if err != nil {
		t.Fatalf("EvaluateQuery() error = %v", err)
	}

	// 验证指标
	if metrics.QueryID != queryID {
		t.Errorf("expected query ID %s, got %s", queryID, metrics.QueryID)
	}

	if metrics.AvgRating != 5.0 {
		t.Errorf("expected avg rating 5.0, got %f", metrics.AvgRating)
	}

	if metrics.CTR <= 0 {
		t.Errorf("expected positive CTR, got %f", metrics.CTR)
	}

	if metrics.NDCG < 0 || metrics.NDCG > 1 {
		t.Errorf("NDCG should be between 0 and 1, got %f", metrics.NDCG)
	}

	if metrics.OverallScore <= 0 || metrics.OverallScore > 1 {
		t.Errorf("OverallScore should be between 0 and 1, got %f", metrics.OverallScore)
	}

	t.Logf("Metrics: Precision=%.3f, Recall=%.3f, F1=%.3f, NDCG=%.3f, MRR=%.3f, Overall=%.3f",
		metrics.Precision, metrics.Recall, metrics.F1Score, metrics.NDCG, metrics.MRR, metrics.OverallScore)
}

func TestEvaluator_EvaluateStrategy(t *testing.T) {
	storage := feedback.NewMemoryStorage()
	collector := feedback.NewCollector(storage)
	ctx := context.Background()

	// 创建多个查询
	for i := 0; i < 5; i++ {
		queryID := "query-" + string(rune('1'+i))
		query := &feedback.Query{
			ID:        queryID,
			Text:      "测试查询",
			UserID:    "user1",
			Strategy:  "hybrid",
			Timestamp: time.Now(),
		}
		collector.RecordQuery(ctx, query)

		results := []types.Document{
			{ID: "doc1", Content: "内容1"},
			{ID: "doc2", Content: "内容2"},
		}
		collector.RecordResults(ctx, queryID, results)

		collector.CollectExplicitFeedback(ctx, &feedback.ExplicitFeedback{
			QueryID:   queryID,
			UserID:    "user1",
			Type:      feedback.FeedbackTypeRating,
			Rating:    4 + (i % 2),
			Timestamp: time.Now(),
		})
	}

	evaluator := NewEvaluator(collector)

	// 评估策略
	strategyMetrics, err := evaluator.EvaluateStrategy(ctx, "hybrid", EvaluateOptions{
		TimeRange:     1 * time.Hour,
		MinSampleSize: 3,
	})
	if err != nil {
		t.Fatalf("EvaluateStrategy() error = %v", err)
	}

	if strategyMetrics.StrategyID != "hybrid" {
		t.Errorf("expected strategy ID 'hybrid', got %s", strategyMetrics.StrategyID)
	}

	if strategyMetrics.TotalQueries != 5 {
		t.Errorf("expected 5 queries, got %d", strategyMetrics.TotalQueries)
	}

	if strategyMetrics.AvgMetrics.AvgRating < 4 || strategyMetrics.AvgMetrics.AvgRating > 5 {
		t.Errorf("expected avg rating between 4 and 5, got %f", strategyMetrics.AvgMetrics.AvgRating)
	}

	t.Logf("Strategy Metrics: TotalQueries=%d, AvgRating=%.2f, OverallScore=%.3f",
		strategyMetrics.TotalQueries,
		strategyMetrics.AvgMetrics.AvgRating,
		strategyMetrics.AvgMetrics.OverallScore)
}

func TestEvaluator_CompareStrategies(t *testing.T) {
	storage := feedback.NewMemoryStorage()
	collector := feedback.NewCollector(storage)
	ctx := context.Background()

	// 创建策略 A 的查询（高分）
	for i := 0; i < 5; i++ {
		queryID := "queryA-" + string(rune('1'+i))
		query := &feedback.Query{
			ID:        queryID,
			Text:      "测试查询",
			UserID:    "user1",
			Strategy:  "strategyA",
			Timestamp: time.Now(),
		}
		collector.RecordQuery(ctx, query)

		results := []types.Document{{ID: "doc1", Content: "内容1"}}
		collector.RecordResults(ctx, queryID, results)

		collector.CollectExplicitFeedback(ctx, &feedback.ExplicitFeedback{
			QueryID:   queryID,
			Type:      feedback.FeedbackTypeRating,
			Rating:    5,
			Timestamp: time.Now(),
		})

		collector.CollectImplicitFeedback(ctx, &feedback.ImplicitFeedback{
			QueryID:    queryID,
			DocumentID: "doc1",
			Action:     feedback.ActionRead,
			Duration:   60 * time.Second,
			Timestamp:  time.Now(),
		})
	}

	// 创建策略 B 的查询（低分）
	for i := 0; i < 5; i++ {
		queryID := "queryB-" + string(rune('1'+i))
		query := &feedback.Query{
			ID:        queryID,
			Text:      "测试查询",
			UserID:    "user1",
			Strategy:  "strategyB",
			Timestamp: time.Now(),
		}
		collector.RecordQuery(ctx, query)

		results := []types.Document{{ID: "doc1", Content: "内容1"}}
		collector.RecordResults(ctx, queryID, results)

		collector.CollectExplicitFeedback(ctx, &feedback.ExplicitFeedback{
			QueryID:   queryID,
			Type:      feedback.FeedbackTypeRating,
			Rating:    3,
			Timestamp: time.Now(),
		})
	}

	evaluator := NewEvaluator(collector)

	// 对比策略
	comparison, err := evaluator.CompareStrategies(ctx, "strategyA", "strategyB")
	if err != nil {
		t.Fatalf("CompareStrategies() error = %v", err)
	}

	if comparison.Winner != "strategyA" {
		t.Errorf("expected winner 'strategyA', got %s", comparison.Winner)
	}

	if comparison.Improvement <= 0 {
		t.Errorf("expected positive improvement, got %f", comparison.Improvement)
	}

	if comparison.Confidence < 0 || comparison.Confidence > 1 {
		t.Errorf("confidence should be between 0 and 1, got %f", comparison.Confidence)
	}

	t.Logf("Comparison: Winner=%s, Improvement=%.2f%%, Confidence=%.2f, p-value=%.3f",
		comparison.Winner,
		comparison.Improvement,
		comparison.Confidence,
		comparison.SignificantAt)
}

func TestRelevanceModel_Default(t *testing.T) {
	model := &DefaultRelevanceModel{}

	qf := &feedback.QueryFeedback{
		Query: feedback.Query{ID: "q1"},
		ImplicitFeedback: []feedback.ImplicitFeedback{
			{DocumentID: "doc1", Action: feedback.ActionClick},
			{DocumentID: "doc2", Action: feedback.ActionRead, Duration: 60 * time.Second},
		},
	}

	// 测试 IsRelevant
	if !model.IsRelevant("doc1", qf) {
		t.Error("doc1 should be relevant (clicked)")
	}

	if !model.IsRelevant("doc2", qf) {
		t.Error("doc2 should be relevant (read)")
	}

	if model.IsRelevant("doc3", qf) {
		t.Error("doc3 should not be relevant")
	}

	// 测试 GetRelevance
	rel1 := model.GetRelevance("doc1", qf)
	rel2 := model.GetRelevance("doc2", qf)

	if rel1 <= 0 {
		t.Errorf("doc1 relevance should be positive, got %f", rel1)
	}

	if rel2 <= rel1 {
		t.Errorf("doc2 (read with duration) should have higher relevance than doc1 (click)")
	}

	t.Logf("Relevance: doc1=%.3f, doc2=%.3f", rel1, rel2)
}

func TestRelevanceModel_Implicit(t *testing.T) {
	model := &ImplicitRelevanceModel{
		ClickWeight:    0.3,
		ReadWeight:     0.5,
		DurationWeight: 0.2,
	}

	qf := &feedback.QueryFeedback{
		Query: feedback.Query{ID: "q1"},
		ImplicitFeedback: []feedback.ImplicitFeedback{
			{DocumentID: "doc1", Action: feedback.ActionClick},
			{DocumentID: "doc2", Action: feedback.ActionRead, Duration: 120 * time.Second},
		},
	}

	rel1 := model.GetRelevance("doc1", qf)
	rel2 := model.GetRelevance("doc2", qf)

	expectedRel1 := 0.3 // click weight
	if rel1 != expectedRel1 {
		t.Errorf("expected doc1 relevance %f, got %f", expectedRel1, rel1)
	}

	// doc2 should have higher relevance due to read + duration
	if rel2 <= rel1 {
		t.Errorf("doc2 should have higher relevance than doc1")
	}

	t.Logf("Implicit Relevance: doc1=%.3f, doc2=%.3f", rel1, rel2)
}
