package optimization

import (
	"time"
)

// ParamType 参数类型
type ParamType string

const (
	ParamTypeInt    ParamType = "int"    // 整数参数
	ParamTypeFloat  ParamType = "float"  // 浮点参数
	ParamTypeChoice ParamType = "choice" // 离散选择
)

// Parameter 参数定义
type Parameter struct {
	Name    string      `json:"name"`              // 参数名
	Type    ParamType   `json:"type"`              // 参数类型
	Min     float64     `json:"min,omitempty"`     // 最小值（数值型）
	Max     float64     `json:"max,omitempty"`     // 最大值（数值型）
	Values  []string    `json:"values,omitempty"`  // 可选值（选择型）
	Default interface{} `json:"default"`           // 默认值
}

// ParameterSpace 参数空间
type ParameterSpace struct {
	Params []Parameter `json:"params"`
}

// OptimizeOptions 优化选项
type OptimizeOptions struct {
	MaxIterations    int           // 最大迭代次数
	TargetMetric     string        // 目标指标（如 "overall_score"）
	MinSampleSize    int           // 最小样本数
	TimeRange        time.Duration // 评估时间范围
	AcquisitionType  string        // 采集函数类型（EI/UCB/PI）
	ExplorationRatio float64       // 探索-利用比例（0-1）
}

// OptimizeResult 优化结果
type OptimizeResult struct {
	StrategyID    string                 `json:"strategy_id"`
	BestParams    map[string]interface{} `json:"best_params"`
	BestScore     float64                `json:"best_score"`
	PreviousScore float64                `json:"previous_score"`
	Improvement   float64                `json:"improvement"`    // 提升百分比
	Iterations    int                    `json:"iterations"`     // 实际迭代次数
	Duration      time.Duration          `json:"duration"`       // 优化耗时
	History       []OptimizeStep         `json:"history"`        // 优化历史
	Timestamp     time.Time              `json:"timestamp"`
}

// OptimizeStep 优化步骤
type OptimizeStep struct {
	Iteration int                    `json:"iteration"`
	Params    map[string]interface{} `json:"params"`
	Score     float64                `json:"score"`
	Timestamp time.Time              `json:"timestamp"`
}

// ParamChange 参数变化记录
type ParamChange struct {
	ParamName  string      `json:"param_name"`
	OldValue   interface{} `json:"old_value"`
	NewValue   interface{} `json:"new_value"`
	Reason     string      `json:"reason"`
	Score      float64     `json:"score"`       // 该参数的得分
	Timestamp  time.Time   `json:"timestamp"`
}

// AutoTuneConfig 自动调优配置
type AutoTuneConfig struct {
	CheckInterval   time.Duration // 检查间隔
	ScoreThreshold  float64       // 得分阈值，低于此值触发优化
	MinSampleSize   int           // 最小样本数
	OptimizeOptions OptimizeOptions // 优化选项
}

// Config 优化器配置
type Config struct {
	MaxIterations    int           // 默认最大迭代次数
	TargetMetric     string        // 默认目标指标
	MinSampleSize    int           // 默认最小样本数
	AcquisitionType  string        // 默认采集函数
	ExplorationRatio float64       // 默认探索比例
}

// DefaultConfig 返回默认配置
func DefaultConfig() Config {
	return Config{
		MaxIterations:    50,
		TargetMetric:     "overall_score",
		MinSampleSize:    30,
		AcquisitionType:  "EI", // Expected Improvement
		ExplorationRatio: 0.1,
	}
}
