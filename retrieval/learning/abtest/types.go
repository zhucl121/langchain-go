package abtest

import (
	"time"

	"github.com/zhucl121/langchain-go/retrieval/learning/evaluation"
)

// ExperimentStatus 实验状态
type ExperimentStatus string

const (
	StatusDraft   ExperimentStatus = "draft"   // 草稿
	StatusRunning ExperimentStatus = "running" // 运行中
	StatusPaused  ExperimentStatus = "paused"  // 暂停
	StatusEnded   ExperimentStatus = "ended"   // 已结束
)

// Experiment A/B 测试实验
type Experiment struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Variants    []Variant              `json:"variants"`
	StartTime   time.Time              `json:"start_time"`
	EndTime     *time.Time             `json:"end_time,omitempty"`
	Status      ExperimentStatus       `json:"status"`
	Traffic     float64                `json:"traffic"`  // 0-1, 参与实验的流量比例
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// Variant 实验变体
type Variant struct {
	ID       string                 `json:"id"`
	Name     string                 `json:"name"`
	Strategy string                 `json:"strategy"`  // 检索策略 ID
	Params   map[string]interface{} `json:"params"`    // 参数配置
	Weight   float64                `json:"weight"`    // 流量权重 (0-1)
}

// ExperimentResult 实验结果
type ExperimentResult struct {
	ExperimentID string                 `json:"experiment_id"`
	VariantID    string                 `json:"variant_id"`
	UserID       string                 `json:"user_id"`
	QueryID      string                 `json:"query_id"`
	Metrics      evaluation.QueryMetrics `json:"metrics"`
	Timestamp    time.Time              `json:"timestamp"`
}

// Assignment 用户分配记录
type Assignment struct {
	ExperimentID string    `json:"experiment_id"`
	UserID       string    `json:"user_id"`
	VariantID    string    `json:"variant_id"`
	Timestamp    time.Time `json:"timestamp"`
}

// ExperimentAnalysis 实验分析结果
type ExperimentAnalysis struct {
	ExperimentID string                    `json:"experiment_id"`
	Variants     map[string]VariantMetrics `json:"variants"`
	Winner       string                    `json:"winner,omitempty"`
	Confidence   float64                   `json:"confidence"`     // 0-1
	PValue       float64                   `json:"p_value"`        // 统计显著性
	Completed    bool                      `json:"completed"`      // 是否有明确结论
	Timestamp    time.Time                 `json:"timestamp"`
}

// VariantMetrics 变体指标
type VariantMetrics struct {
	VariantID       string                 `json:"variant_id"`
	SampleSize      int                    `json:"sample_size"`
	AvgScore        float64                `json:"avg_score"`
	StdDev          float64                `json:"std_dev"`
	ConfInterval    [2]float64             `json:"conf_interval"` // 95% 置信区间
	DetailedMetrics evaluation.QueryMetrics `json:"detailed_metrics"`
}
