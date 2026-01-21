package feedback

import (
	"context"

	"github.com/zhucl121/langchain-go/pkg/types"
)

// Collector 反馈收集器接口
type Collector interface {
	// RecordQuery 记录查询
	RecordQuery(ctx context.Context, query *Query) error

	// RecordResults 记录检索结果
	RecordResults(ctx context.Context, queryID string, results []types.Document) error

	// CollectExplicitFeedback 收集显式反馈
	CollectExplicitFeedback(ctx context.Context, feedback *ExplicitFeedback) error

	// CollectImplicitFeedback 收集隐式反馈
	CollectImplicitFeedback(ctx context.Context, feedback *ImplicitFeedback) error

	// GetQueryFeedback 获取查询反馈
	GetQueryFeedback(ctx context.Context, queryID string) (*QueryFeedback, error)

	// AggregateStats 聚合反馈统计
	AggregateStats(ctx context.Context, opts AggregateOptions) (*FeedbackStats, error)

	// GetStorage 获取底层存储（用于高级功能）
	GetStorage() Storage
}

// Storage 反馈存储接口
type Storage interface {
	// SaveQuery 保存查询
	SaveQuery(ctx context.Context, query *Query) error

	// SaveResults 保存检索结果
	SaveResults(ctx context.Context, queryID string, results []types.Document) error

	// SaveExplicitFeedback 保存显式反馈
	SaveExplicitFeedback(ctx context.Context, feedback *ExplicitFeedback) error

	// SaveImplicitFeedback 保存隐式反馈
	SaveImplicitFeedback(ctx context.Context, feedback *ImplicitFeedback) error

	// GetQueryFeedback 获取查询反馈
	GetQueryFeedback(ctx context.Context, queryID string) (*QueryFeedback, error)

	// ListQueries 列出查询
	ListQueries(ctx context.Context, opts ListOptions) ([]Query, error)

	// Aggregate 聚合统计
	Aggregate(ctx context.Context, opts AggregateOptions) (*FeedbackStats, error)
}
