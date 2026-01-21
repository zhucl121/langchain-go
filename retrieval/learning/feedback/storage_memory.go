package feedback

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/zhucl121/langchain-go/pkg/types"
)

// MemoryStorage 内存存储实现
type MemoryStorage struct {
	mu               sync.RWMutex
	queries          map[string]*Query
	results          map[string][]types.Document
	explicitFeedback map[string][]*ExplicitFeedback // query_id -> feedbacks
	implicitFeedback map[string][]*ImplicitFeedback // query_id -> feedbacks
}

// NewMemoryStorage 创建内存存储
func NewMemoryStorage() Storage {
	return &MemoryStorage{
		queries:          make(map[string]*Query),
		results:          make(map[string][]types.Document),
		explicitFeedback: make(map[string][]*ExplicitFeedback),
		implicitFeedback: make(map[string][]*ImplicitFeedback),
	}
}

// SaveQuery 保存查询
func (s *MemoryStorage) SaveQuery(ctx context.Context, query *Query) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.queries[query.ID] = query
	return nil
}

// SaveResults 保存检索结果
func (s *MemoryStorage) SaveResults(ctx context.Context, queryID string, results []types.Document) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.results[queryID] = results
	return nil
}

// SaveExplicitFeedback 保存显式反馈
func (s *MemoryStorage) SaveExplicitFeedback(ctx context.Context, feedback *ExplicitFeedback) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.explicitFeedback[feedback.QueryID] = append(
		s.explicitFeedback[feedback.QueryID],
		feedback,
	)
	return nil
}

// SaveImplicitFeedback 保存隐式反馈
func (s *MemoryStorage) SaveImplicitFeedback(ctx context.Context, feedback *ImplicitFeedback) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.implicitFeedback[feedback.QueryID] = append(
		s.implicitFeedback[feedback.QueryID],
		feedback,
	)
	return nil
}

// GetQueryFeedback 获取查询反馈
func (s *MemoryStorage) GetQueryFeedback(ctx context.Context, queryID string) (*QueryFeedback, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query, ok := s.queries[queryID]
	if !ok {
		return nil, fmt.Errorf("query not found: %s", queryID)
	}

	qf := &QueryFeedback{
		Query:            *query,
		Results:          s.results[queryID],
		ExplicitFeedback: make([]ExplicitFeedback, 0),
		ImplicitFeedback: make([]ImplicitFeedback, 0),
	}

	// 复制显式反馈
	for _, fb := range s.explicitFeedback[queryID] {
		qf.ExplicitFeedback = append(qf.ExplicitFeedback, *fb)
	}

	// 复制隐式反馈
	for _, fb := range s.implicitFeedback[queryID] {
		qf.ImplicitFeedback = append(qf.ImplicitFeedback, *fb)
	}

	// 计算统计指标
	qf.AvgRating = s.calculateAvgRating(qf.ExplicitFeedback)
	qf.CTR = s.calculateCTR(qf.Results, qf.ImplicitFeedback)
	qf.AvgReadDuration = s.calculateAvgReadDuration(qf.ImplicitFeedback)

	return qf, nil
}

// ListQueries 列出查询
func (s *MemoryStorage) ListQueries(ctx context.Context, opts ListOptions) ([]Query, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	queries := make([]Query, 0)
	for _, q := range s.queries {
		// 应用过滤条件
		if opts.UserID != "" && q.UserID != opts.UserID {
			continue
		}
		if opts.Strategy != "" && q.Strategy != opts.Strategy {
			continue
		}
		if !opts.StartTime.IsZero() && q.Timestamp.Before(opts.StartTime) {
			continue
		}
		if !opts.EndTime.IsZero() && q.Timestamp.After(opts.EndTime) {
			continue
		}

		queries = append(queries, *q)
	}

	// 按时间排序
	sort.Slice(queries, func(i, j int) bool {
		return queries[i].Timestamp.After(queries[j].Timestamp)
	})

	// 应用分页
	start := opts.Offset
	if start >= len(queries) {
		return []Query{}, nil
	}

	end := start + opts.Limit
	if opts.Limit == 0 || end > len(queries) {
		end = len(queries)
	}

	return queries[start:end], nil
}

// Aggregate 聚合统计
func (s *MemoryStorage) Aggregate(ctx context.Context, opts AggregateOptions) (*FeedbackStats, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := &FeedbackStats{
		TopQueries:       make([]string, 0),
		LowRatingQueries: make([]string, 0),
	}

	// 时间范围过滤
	cutoffTime := time.Now()
	if opts.TimeRange > 0 {
		cutoffTime = cutoffTime.Add(-opts.TimeRange)
	}

	totalRating := 0.0
	ratingCount := 0
	positiveCount := 0
	negativeCount := 0
	totalCTR := 0.0
	ctrCount := 0
	totalDuration := time.Duration(0)
	durationCount := 0

	queryRatings := make(map[string][]float64)

	// 遍历所有查询
	for queryID, query := range s.queries {
		// 时间过滤
		if query.Timestamp.Before(cutoffTime) {
			continue
		}

		// 策略过滤
		if opts.Strategy != "" && query.Strategy != opts.Strategy {
			continue
		}

		stats.TotalQueries++

		// 处理显式反馈
		for _, fb := range s.explicitFeedback[queryID] {
			if fb.Type == FeedbackTypePositive {
				positiveCount++
			} else if fb.Type == FeedbackTypeNegative {
				negativeCount++
			}

			if fb.Rating > 0 {
				totalRating += float64(fb.Rating)
				ratingCount++
				queryRatings[queryID] = append(queryRatings[queryID], float64(fb.Rating))
			}
		}

		// 处理隐式反馈
		if implicitFB := s.implicitFeedback[queryID]; len(implicitFB) > 0 {
			clickCount := 0
			for _, fb := range implicitFB {
				if fb.Action == ActionClick || fb.Action == ActionRead {
					clickCount++
					if fb.Duration > 0 {
						totalDuration += fb.Duration
						durationCount++
					}
				}
			}

			results := s.results[queryID]
			if len(results) > 0 {
				ctr := float64(clickCount) / float64(len(results))
				totalCTR += ctr
				ctrCount++
			}
		}
	}

	// 计算平均值
	if ratingCount > 0 {
		stats.AvgRating = totalRating / float64(ratingCount)
	}

	totalFeedback := positiveCount + negativeCount
	if totalFeedback > 0 {
		stats.PositiveRate = float64(positiveCount) / float64(totalFeedback)
		stats.NegativeRate = float64(negativeCount) / float64(totalFeedback)
	}

	if ctrCount > 0 {
		stats.AvgCTR = totalCTR / float64(ctrCount)
	}

	if durationCount > 0 {
		stats.AvgReadDuration = totalDuration / time.Duration(durationCount)
	}

	// 找出低评分查询
	for queryID, ratings := range queryRatings {
		avgRating := 0.0
		for _, r := range ratings {
			avgRating += r
		}
		avgRating /= float64(len(ratings))

		if avgRating < float64(opts.MinRating) {
			if query, ok := s.queries[queryID]; ok {
				stats.LowRatingQueries = append(stats.LowRatingQueries, query.Text)
			}
		}
	}

	return stats, nil
}

// 辅助方法

func (s *MemoryStorage) calculateAvgRating(feedbacks []ExplicitFeedback) float64 {
	if len(feedbacks) == 0 {
		return 0
	}

	total := 0.0
	count := 0
	for _, fb := range feedbacks {
		if fb.Rating > 0 {
			total += float64(fb.Rating)
			count++
		}
	}

	if count == 0 {
		return 0
	}
	return total / float64(count)
}

func (s *MemoryStorage) calculateCTR(results []types.Document, feedbacks []ImplicitFeedback) float64 {
	if len(results) == 0 {
		return 0
	}

	clickCount := 0
	for _, fb := range feedbacks {
		if fb.Action == ActionClick || fb.Action == ActionRead {
			clickCount++
		}
	}

	return float64(clickCount) / float64(len(results))
}

func (s *MemoryStorage) calculateAvgReadDuration(feedbacks []ImplicitFeedback) time.Duration {
	if len(feedbacks) == 0 {
		return 0
	}

	total := time.Duration(0)
	count := 0
	for _, fb := range feedbacks {
		if fb.Action == ActionRead && fb.Duration > 0 {
			total += fb.Duration
			count++
		}
	}

	if count == 0 {
		return 0
	}
	return total / time.Duration(count)
}
