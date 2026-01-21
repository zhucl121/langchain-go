package optimization

import (
	"context"
)

// Optimizer 参数优化器接口
type Optimizer interface {
	// Optimize 优化参数
	Optimize(ctx context.Context, strategyID string, paramSpace ParameterSpace, opts OptimizeOptions) (*OptimizeResult, error)

	// SuggestParams 建议参数
	SuggestParams(ctx context.Context, strategyID string, paramSpace ParameterSpace) (map[string]interface{}, error)

	// AutoTune 自动调优（持续运行）
	AutoTune(ctx context.Context, strategyID string, paramSpace ParameterSpace, config AutoTuneConfig) error

	// ValidateParams 验证参数
	ValidateParams(params map[string]interface{}, paramSpace ParameterSpace) error

	// GetHistory 获取优化历史
	GetHistory(ctx context.Context, strategyID string) ([]OptimizeResult, error)
}
