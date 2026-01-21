package feedback

import (
	"context"
	"testing"
	"time"

	"github.com/zhucl121/langchain-go/pkg/types"
)

func TestCollector_RecordQuery(t *testing.T) {
	storage := NewMemoryStorage()
	collector := NewCollector(storage)
	ctx := context.Background()

	tests := []struct {
		name    string
		query   *Query
		wantErr bool
	}{
		{
			name: "valid query",
			query: &Query{
				ID:        "q1",
				Text:      "什么是机器学习？",
				UserID:    "user1",
				Strategy:  "hybrid",
				Timestamp: time.Now(),
			},
			wantErr: false,
		},
		{
			name:    "nil query",
			query:   nil,
			wantErr: true,
		},
		{
			name: "empty ID",
			query: &Query{
				Text: "test",
			},
			wantErr: true,
		},
		{
			name: "empty text",
			query: &Query{
				ID: "q2",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := collector.RecordQuery(ctx, tt.query)
			if (err != nil) != tt.wantErr {
				t.Errorf("RecordQuery() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCollector_RecordResults(t *testing.T) {
	storage := NewMemoryStorage()
	collector := NewCollector(storage)
	ctx := context.Background()

	// 先记录查询
	query := &Query{
		ID:        "q1",
		Text:      "test",
		UserID:    "user1",
		Timestamp: time.Now(),
	}
	collector.RecordQuery(ctx, query)

	results := []types.Document{
		{ID: "doc1", Content: "content1"},
		{ID: "doc2", Content: "content2"},
	}

	err := collector.RecordResults(ctx, query.ID, results)
	if err != nil {
		t.Errorf("RecordResults() error = %v", err)
	}

	// 测试空 queryID
	err = collector.RecordResults(ctx, "", results)
	if err == nil {
		t.Error("RecordResults() should return error for empty queryID")
	}
}

func TestCollector_CollectExplicitFeedback(t *testing.T) {
	storage := NewMemoryStorage()
	collector := NewCollector(storage)
	ctx := context.Background()

	tests := []struct {
		name     string
		feedback *ExplicitFeedback
		wantErr  bool
	}{
		{
			name: "valid positive feedback",
			feedback: &ExplicitFeedback{
				QueryID:   "q1",
				UserID:    "user1",
				Type:      FeedbackTypePositive,
				Timestamp: time.Now(),
			},
			wantErr: false,
		},
		{
			name: "valid rating feedback",
			feedback: &ExplicitFeedback{
				QueryID:   "q1",
				UserID:    "user1",
				Type:      FeedbackTypeRating,
				Rating:    5,
				Timestamp: time.Now(),
			},
			wantErr: false,
		},
		{
			name: "invalid rating - too low",
			feedback: &ExplicitFeedback{
				QueryID:   "q1",
				UserID:    "user1",
				Type:      FeedbackTypeRating,
				Rating:    0,
				Timestamp: time.Now(),
			},
			wantErr: true,
		},
		{
			name: "invalid rating - too high",
			feedback: &ExplicitFeedback{
				QueryID:   "q1",
				UserID:    "user1",
				Type:      FeedbackTypeRating,
				Rating:    6,
				Timestamp: time.Now(),
			},
			wantErr: true,
		},
		{
			name:     "nil feedback",
			feedback: nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := collector.CollectExplicitFeedback(ctx, tt.feedback)
			if (err != nil) != tt.wantErr {
				t.Errorf("CollectExplicitFeedback() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCollector_CollectImplicitFeedback(t *testing.T) {
	storage := NewMemoryStorage()
	collector := NewCollector(storage)
	ctx := context.Background()

	tests := []struct {
		name     string
		feedback *ImplicitFeedback
		wantErr  bool
	}{
		{
			name: "valid click feedback",
			feedback: &ImplicitFeedback{
				QueryID:    "q1",
				UserID:     "user1",
				DocumentID: "doc1",
				Action:     ActionClick,
				Duration:   10 * time.Second,
				Timestamp:  time.Now(),
			},
			wantErr: false,
		},
		{
			name:     "nil feedback",
			feedback: nil,
			wantErr:  true,
		},
		{
			name: "empty queryID",
			feedback: &ImplicitFeedback{
				DocumentID: "doc1",
				Action:     ActionClick,
			},
			wantErr: true,
		},
		{
			name: "empty documentID",
			feedback: &ImplicitFeedback{
				QueryID: "q1",
				Action:  ActionClick,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := collector.CollectImplicitFeedback(ctx, tt.feedback)
			if (err != nil) != tt.wantErr {
				t.Errorf("CollectImplicitFeedback() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCollector_GetQueryFeedback(t *testing.T) {
	storage := NewMemoryStorage()
	collector := NewCollector(storage)
	ctx := context.Background()

	// 准备测试数据
	queryID := "q1"
	query := &Query{
		ID:        queryID,
		Text:      "什么是人工智能？",
		UserID:    "user1",
		Strategy:  "hybrid",
		Timestamp: time.Now(),
	}

	results := []types.Document{
		{ID: "doc1", Content: "AI is..."},
		{ID: "doc2", Content: "Machine learning..."},
	}

	collector.RecordQuery(ctx, query)
	collector.RecordResults(ctx, queryID, results)

	// 添加显式反馈
	collector.CollectExplicitFeedback(ctx, &ExplicitFeedback{
		QueryID:   queryID,
		UserID:    "user1",
		Type:      FeedbackTypeRating,
		Rating:    5,
		Timestamp: time.Now(),
	})

	// 添加隐式反馈
	collector.CollectImplicitFeedback(ctx, &ImplicitFeedback{
		QueryID:    queryID,
		UserID:     "user1",
		DocumentID: "doc1",
		Action:     ActionClick,
		Duration:   30 * time.Second,
		Timestamp:  time.Now(),
	})

	// 获取反馈
	qf, err := collector.GetQueryFeedback(ctx, queryID)
	if err != nil {
		t.Fatalf("GetQueryFeedback() error = %v", err)
	}

	// 验证结果
	if qf.Query.ID != queryID {
		t.Errorf("expected query ID %s, got %s", queryID, qf.Query.ID)
	}

	if len(qf.Results) != 2 {
		t.Errorf("expected 2 results, got %d", len(qf.Results))
	}

	if len(qf.ExplicitFeedback) != 1 {
		t.Errorf("expected 1 explicit feedback, got %d", len(qf.ExplicitFeedback))
	}

	if len(qf.ImplicitFeedback) != 1 {
		t.Errorf("expected 1 implicit feedback, got %d", len(qf.ImplicitFeedback))
	}

	if qf.AvgRating != 5.0 {
		t.Errorf("expected avg rating 5.0, got %f", qf.AvgRating)
	}

	if qf.CTR != 0.5 { // 1 click out of 2 results
		t.Errorf("expected CTR 0.5, got %f", qf.CTR)
	}
}

func TestCollector_AggregateStats(t *testing.T) {
	storage := NewMemoryStorage()
	collector := NewCollector(storage)
	ctx := context.Background()

	// 准备测试数据
	now := time.Now()

	// Query 1 - 高评分
	q1 := &Query{
		ID:        "q1",
		Text:      "query1",
		UserID:    "user1",
		Strategy:  "hybrid",
		Timestamp: now,
	}
	collector.RecordQuery(ctx, q1)
	collector.RecordResults(ctx, q1.ID, []types.Document{{ID: "doc1"}})
	collector.CollectExplicitFeedback(ctx, &ExplicitFeedback{
		QueryID:   q1.ID,
		Type:      FeedbackTypePositive,
		Rating:    5,
		Timestamp: now,
	})

	// Query 2 - 低评分
	q2 := &Query{
		ID:        "q2",
		Text:      "query2",
		UserID:    "user2",
		Strategy:  "hybrid",
		Timestamp: now,
	}
	collector.RecordQuery(ctx, q2)
	collector.CollectExplicitFeedback(ctx, &ExplicitFeedback{
		QueryID:   q2.ID,
		Type:      FeedbackTypeNegative,
		Rating:    2,
		Timestamp: now,
	})

	// 聚合统计
	stats, err := collector.AggregateStats(ctx, AggregateOptions{
		TimeRange: 1 * time.Hour,
		MinRating: 3,
	})
	if err != nil {
		t.Fatalf("AggregateStats() error = %v", err)
	}

	// 验证结果
	if stats.TotalQueries != 2 {
		t.Errorf("expected 2 total queries, got %d", stats.TotalQueries)
	}

	if stats.AvgRating != 3.5 {
		t.Errorf("expected avg rating 3.5, got %f", stats.AvgRating)
	}

	if stats.PositiveRate != 0.5 {
		t.Errorf("expected positive rate 0.5, got %f", stats.PositiveRate)
	}

	if stats.NegativeRate != 0.5 {
		t.Errorf("expected negative rate 0.5, got %f", stats.NegativeRate)
	}
}
