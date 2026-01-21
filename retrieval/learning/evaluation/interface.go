package evaluation

import (
	"context"

	"github.com/zhucl121/langchain-go/retrieval/learning/feedback"
)

// Evaluator 质量评估器接口
type Evaluator interface {
	// EvaluateQuery 评估单个查询
	EvaluateQuery(ctx context.Context, queryFeedback *feedback.QueryFeedback) (*QueryMetrics, error)

	// EvaluateStrategy 评估策略
	EvaluateStrategy(ctx context.Context, strategyID string, opts EvaluateOptions) (*StrategyMetrics, error)

	// CompareStrategies 对比策略
	CompareStrategies(ctx context.Context, strategyA, strategyB string) (*ComparisonResult, error)
}
