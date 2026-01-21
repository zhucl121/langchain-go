package feedback

import (
	"context"
	"fmt"

	"github.com/zhucl121/langchain-go/pkg/types"
)

// DefaultCollector 默认反馈收集器实现
type DefaultCollector struct {
	storage Storage
}

// NewCollector 创建反馈收集器
func NewCollector(storage Storage) Collector {
	return &DefaultCollector{
		storage: storage,
	}
}

// GetStorage 获取存储（用于高级功能）
func (c *DefaultCollector) GetStorage() Storage {
	return c.storage
}

// RecordQuery 记录查询
func (c *DefaultCollector) RecordQuery(ctx context.Context, query *Query) error {
	if query == nil {
		return fmt.Errorf("query cannot be nil")
	}
	if query.ID == "" {
		return fmt.Errorf("query ID cannot be empty")
	}
	if query.Text == "" {
		return fmt.Errorf("query text cannot be empty")
	}

	return c.storage.SaveQuery(ctx, query)
}

// RecordResults 记录检索结果
func (c *DefaultCollector) RecordResults(ctx context.Context, queryID string, results []types.Document) error {
	if queryID == "" {
		return fmt.Errorf("query ID cannot be empty")
	}

	return c.storage.SaveResults(ctx, queryID, results)
}

// CollectExplicitFeedback 收集显式反馈
func (c *DefaultCollector) CollectExplicitFeedback(ctx context.Context, feedback *ExplicitFeedback) error {
	if feedback == nil {
		return fmt.Errorf("feedback cannot be nil")
	}
	if feedback.QueryID == "" {
		return fmt.Errorf("query ID cannot be empty")
	}

	// 验证评分范围
	if feedback.Type == FeedbackTypeRating {
		if feedback.Rating < 1 || feedback.Rating > 5 {
			return fmt.Errorf("rating must be between 1 and 5")
		}
	}

	return c.storage.SaveExplicitFeedback(ctx, feedback)
}

// CollectImplicitFeedback 收集隐式反馈
func (c *DefaultCollector) CollectImplicitFeedback(ctx context.Context, feedback *ImplicitFeedback) error {
	if feedback == nil {
		return fmt.Errorf("feedback cannot be nil")
	}
	if feedback.QueryID == "" {
		return fmt.Errorf("query ID cannot be empty")
	}
	if feedback.DocumentID == "" {
		return fmt.Errorf("document ID cannot be empty")
	}

	return c.storage.SaveImplicitFeedback(ctx, feedback)
}

// GetQueryFeedback 获取查询反馈
func (c *DefaultCollector) GetQueryFeedback(ctx context.Context, queryID string) (*QueryFeedback, error) {
	if queryID == "" {
		return nil, fmt.Errorf("query ID cannot be empty")
	}

	return c.storage.GetQueryFeedback(ctx, queryID)
}

// AggregateStats 聚合反馈统计
func (c *DefaultCollector) AggregateStats(ctx context.Context, opts AggregateOptions) (*FeedbackStats, error) {
	return c.storage.Aggregate(ctx, opts)
}
