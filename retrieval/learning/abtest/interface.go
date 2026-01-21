package abtest

import (
	"context"
)

// Manager A/B 测试管理器接口
type Manager interface {
	// CreateExperiment 创建实验
	CreateExperiment(ctx context.Context, experiment *Experiment) error

	// GetExperiment 获取实验
	GetExperiment(ctx context.Context, experimentID string) (*Experiment, error)

	// UpdateExperiment 更新实验
	UpdateExperiment(ctx context.Context, experiment *Experiment) error

	// StartExperiment 开始实验
	StartExperiment(ctx context.Context, experimentID string) error

	// EndExperiment 结束实验
	EndExperiment(ctx context.Context, experimentID string, winner string) error

	// AssignVariant 分配变体（用户分流）
	AssignVariant(ctx context.Context, userID string, experimentID string) (string, error)

	// RecordResult 记录实验结果
	RecordResult(ctx context.Context, result *ExperimentResult) error

	// AnalyzeExperiment 分析实验
	AnalyzeExperiment(ctx context.Context, experimentID string) (*ExperimentAnalysis, error)

	// ListExperiments 列出实验
	ListExperiments(ctx context.Context, status ExperimentStatus) ([]*Experiment, error)
}

// Storage A/B 测试存储接口
type Storage interface {
	// SaveExperiment 保存实验
	SaveExperiment(ctx context.Context, experiment *Experiment) error

	// GetExperiment 获取实验
	GetExperiment(ctx context.Context, experimentID string) (*Experiment, error)

	// ListExperiments 列出实验
	ListExperiments(ctx context.Context, status ExperimentStatus) ([]*Experiment, error)

	// SaveAssignment 保存分配
	SaveAssignment(ctx context.Context, assignment *Assignment) error

	// GetAssignment 获取分配
	GetAssignment(ctx context.Context, experimentID string, userID string) (*Assignment, error)

	// SaveResult 保存结果
	SaveResult(ctx context.Context, result *ExperimentResult) error

	// GetResults 获取结果
	GetResults(ctx context.Context, experimentID string) ([]*ExperimentResult, error)
}
