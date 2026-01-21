package feedback

import (
	"time"

	"github.com/zhucl121/langchain-go/pkg/types"
)

// Query 查询信息
type Query struct {
	ID        string                 `json:"id"`
	Text      string                 `json:"text"`
	UserID    string                 `json:"user_id"`
	Strategy  string                 `json:"strategy"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// FeedbackType 反馈类型
type FeedbackType string

const (
	FeedbackTypePositive FeedbackType = "positive" // 👍
	FeedbackTypeNegative FeedbackType = "negative" // 👎
	FeedbackTypeRating   FeedbackType = "rating"   // ⭐
	FeedbackTypeComment  FeedbackType = "comment"  // 💬
)

// ExplicitFeedback 显式反馈
type ExplicitFeedback struct {
	QueryID   string       `json:"query_id"`
	UserID    string       `json:"user_id"`
	Type      FeedbackType `json:"type"`
	Rating    int          `json:"rating"`         // 1-5 星
	Comment   string       `json:"comment,omitempty"`
	Timestamp time.Time    `json:"timestamp"`
}

// UserAction 用户行为
type UserAction string

const (
	ActionClick    UserAction = "click"    // 点击
	ActionRead     UserAction = "read"     // 阅读
	ActionCopy     UserAction = "copy"     // 复制
	ActionDownload UserAction = "download" // 下载
	ActionIgnore   UserAction = "ignore"   // 忽略
	ActionSkip     UserAction = "skip"     // 跳过
)

// ImplicitFeedback 隐式反馈
type ImplicitFeedback struct {
	QueryID    string        `json:"query_id"`
	UserID     string        `json:"user_id"`
	DocumentID string        `json:"document_id"`
	Action     UserAction    `json:"action"`
	Duration   time.Duration `json:"duration"` // 行为持续时间
	Timestamp  time.Time     `json:"timestamp"`
}

// QueryFeedback 查询反馈汇总
type QueryFeedback struct {
	Query            Query              `json:"query"`
	Results          []types.Document   `json:"results"`
	ExplicitFeedback []ExplicitFeedback `json:"explicit_feedback"`
	ImplicitFeedback []ImplicitFeedback `json:"implicit_feedback"`
	AvgRating        float64            `json:"avg_rating"`
	CTR              float64            `json:"ctr"`              // Click-Through Rate
	AvgReadDuration  time.Duration      `json:"avg_read_duration"`
}

// FeedbackStats 反馈统计
type FeedbackStats struct {
	TotalQueries     int           `json:"total_queries"`
	AvgRating        float64       `json:"avg_rating"`
	PositiveRate     float64       `json:"positive_rate"`
	NegativeRate     float64       `json:"negative_rate"`
	AvgCTR           float64       `json:"avg_ctr"`
	AvgReadDuration  time.Duration `json:"avg_read_duration"`
	TopQueries       []string      `json:"top_queries"`
	LowRatingQueries []string      `json:"low_rating_queries"`
}

// AggregateOptions 聚合选项
type AggregateOptions struct {
	TimeRange time.Duration // 时间范围
	Strategy  string        // 过滤特定策略
	MinRating int           // 最低评分
}

// ListOptions 列表选项
type ListOptions struct {
	Limit     int       // 限制数量
	Offset    int       // 偏移量
	StartTime time.Time // 开始时间
	EndTime   time.Time // 结束时间
	UserID    string    // 过滤用户
	Strategy  string    // 过滤策略
}
