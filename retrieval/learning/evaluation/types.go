package evaluation

import (
	"time"
)

// QueryMetrics 查询指标
type QueryMetrics struct {
	QueryID string `json:"query_id"`

	// 相关性指标
	Precision float64 `json:"precision"` // 精确率
	Recall    float64 `json:"recall"`    // 召回率
	F1Score   float64 `json:"f1_score"`  // F1 分数
	NDCG      float64 `json:"ndcg"`      // 归一化折损累计增益
	MRR       float64 `json:"mrr"`       // 平均倒数排名

	// 用户满意度指标
	AvgRating float64 `json:"avg_rating"` // 平均评分
	CTR       float64 `json:"ctr"`        // 点击率
	ReadRate  float64 `json:"read_rate"`  // 阅读率

	// 效率指标
	ResponseTime time.Duration `json:"response_time"` // 响应时间

	// 综合得分
	OverallScore float64 `json:"overall_score"` // 综合得分 (0-1)
}

// StrategyMetrics 策略指标
type StrategyMetrics struct {
	StrategyID   string       `json:"strategy_id"`
	TotalQueries int          `json:"total_queries"`
	AvgMetrics   QueryMetrics `json:"avg_metrics"`
	P95Metrics   QueryMetrics `json:"p95_metrics"`
	Timestamp    time.Time    `json:"timestamp"`
}

// ComparisonResult 对比结果
type ComparisonResult struct {
	StrategyA     StrategyMetrics `json:"strategy_a"`
	StrategyB     StrategyMetrics `json:"strategy_b"`
	Winner        string          `json:"winner"`
	Confidence    float64         `json:"confidence"`     // 0-1
	Improvement   float64         `json:"improvement"`    // A vs B 提升百分比
	SignificantAt float64         `json:"significant_at"` // p-value
}

// EvaluateOptions 评估选项
type EvaluateOptions struct {
	TimeRange      time.Duration // 时间范围
	MinSampleSize  int           // 最小样本数
	RelevanceModel RelevanceModel // 相关性模型
}

// RelevanceModel 相关性模型
type RelevanceModel interface {
	// IsRelevant 判断文档是否相关
	IsRelevant(docID string, queryFeedback interface{}) bool

	// GetRelevance 获取文档相关度得分 (0-1)
	GetRelevance(docID string, queryFeedback interface{}) float64
}

// DefaultRelevanceModel 默认相关性模型
type DefaultRelevanceModel struct{}

// ImplicitRelevanceModel 基于隐式反馈的相关性模型
type ImplicitRelevanceModel struct {
	ClickWeight    float64 // 点击权重
	ReadWeight     float64 // 阅读权重
	DurationWeight float64 // 时长权重
}
