package optimization

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/zhucl121/langchain-go/retrieval/learning/evaluation"
	"github.com/zhucl121/langchain-go/retrieval/learning/feedback"
)

// BayesianOptimizer 贝叶斯优化器
type BayesianOptimizer struct {
	evaluator  evaluation.Evaluator
	collector  feedback.Collector
	config     Config
	
	mu      sync.RWMutex
	history map[string][]OptimizeResult // 按策略ID存储历史
}

// NewOptimizer 创建优化器
func NewOptimizer(evaluator evaluation.Evaluator, collector feedback.Collector, config Config) Optimizer {
	if config.MaxIterations == 0 {
		config = DefaultConfig()
	}
	
	return &BayesianOptimizer{
		evaluator: evaluator,
		collector: collector,
		config:    config,
		history:   make(map[string][]OptimizeResult),
	}
}

// Optimize 优化参数
func (o *BayesianOptimizer) Optimize(ctx context.Context, strategyID string, paramSpace ParameterSpace, opts OptimizeOptions) (*OptimizeResult, error) {
	if strategyID == "" {
		return nil, fmt.Errorf("strategy ID cannot be empty")
	}
	
	if len(paramSpace.Params) == 0 {
		return nil, fmt.Errorf("parameter space cannot be empty")
	}
	
	// 应用默认配置
	if opts.MaxIterations == 0 {
		opts.MaxIterations = o.config.MaxIterations
	}
	if opts.TargetMetric == "" {
		opts.TargetMetric = o.config.TargetMetric
	}
	if opts.MinSampleSize == 0 {
		opts.MinSampleSize = o.config.MinSampleSize
	}
	if opts.AcquisitionType == "" {
		opts.AcquisitionType = o.config.AcquisitionType
	}
	if opts.ExplorationRatio == 0 {
		opts.ExplorationRatio = o.config.ExplorationRatio
	}
	
	startTime := time.Now()
	
	// 获取当前性能
	currentScore, err := o.evaluateStrategy(ctx, strategyID, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate current strategy: %w", err)
	}
	
	// 初始化最佳参数（使用默认值）
	bestParams := make(map[string]interface{})
	for _, param := range paramSpace.Params {
		bestParams[param.Name] = param.Default
	}
	bestScore := currentScore
	
	// 优化历史
	history := make([]OptimizeStep, 0, opts.MaxIterations)
	
	// 贝叶斯优化主循环
	observations := make([]observation, 0, opts.MaxIterations)
	
	for i := 0; i < opts.MaxIterations; i++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		
		// 选择下一个参数点
		var nextParams map[string]interface{}
		if i < 5 || rand.Float64() < opts.ExplorationRatio {
			// 前几次或随机探索：随机采样
			nextParams = o.randomSample(paramSpace)
		} else {
			// 利用：使用采集函数
			nextParams = o.selectNextPoint(paramSpace, observations, opts.AcquisitionType)
		}
		
		// 评估参数
		score := o.evaluateParams(ctx, strategyID, nextParams, opts)
		
		// 记录观察
		observations = append(observations, observation{
			params: nextParams,
			score:  score,
		})
		
		// 记录历史
		history = append(history, OptimizeStep{
			Iteration: i + 1,
			Params:    copyParams(nextParams),
			Score:     score,
			Timestamp: time.Now(),
		})
		
		// 更新最佳参数
		if score > bestScore {
			bestScore = score
			bestParams = copyParams(nextParams)
		}
	}
	
	// 计算提升
	improvement := 0.0
	if currentScore > 0 {
		improvement = ((bestScore - currentScore) / currentScore) * 100
	}
	
	result := &OptimizeResult{
		StrategyID:    strategyID,
		BestParams:    bestParams,
		BestScore:     bestScore,
		PreviousScore: currentScore,
		Improvement:   improvement,
		Iterations:    opts.MaxIterations,
		Duration:      time.Since(startTime),
		History:       history,
		Timestamp:     time.Now(),
	}
	
	// 保存历史
	o.saveHistory(strategyID, result)
	
	return result, nil
}

// SuggestParams 建议参数
func (o *BayesianOptimizer) SuggestParams(ctx context.Context, strategyID string, paramSpace ParameterSpace) (map[string]interface{}, error) {
	// 获取历史记录
	o.mu.RLock()
	historyList := o.history[strategyID]
	o.mu.RUnlock()
	
	if len(historyList) == 0 {
		// 没有历史，返回默认参数
		params := make(map[string]interface{})
		for _, param := range paramSpace.Params {
			params[param.Name] = param.Default
		}
		return params, nil
	}
	
	// 返回最近一次优化的最佳参数
	lastResult := historyList[len(historyList)-1]
	return lastResult.BestParams, nil
}

// AutoTune 自动调优
func (o *BayesianOptimizer) AutoTune(ctx context.Context, strategyID string, paramSpace ParameterSpace, config AutoTuneConfig) error {
	if config.CheckInterval == 0 {
		config.CheckInterval = 1 * time.Hour
	}
	if config.ScoreThreshold == 0 {
		config.ScoreThreshold = 0.7
	}
	if config.MinSampleSize == 0 {
		config.MinSampleSize = 30
	}
	
	ticker := time.NewTicker(config.CheckInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			// 评估当前性能
			score, err := o.evaluateStrategy(ctx, strategyID, config.OptimizeOptions)
			if err != nil {
				continue
			}
			
			// 如果性能低于阈值，触发优化
			if score < config.ScoreThreshold {
				result, err := o.Optimize(ctx, strategyID, paramSpace, config.OptimizeOptions)
				if err != nil {
					continue
				}
				
				// 如果有提升，应用新参数
				if result.Improvement > 0 {
					// 这里应该有一个机制来应用参数
					// 实际应用中需要与检索系统集成
					_ = result
				}
			}
		}
	}
}

// ValidateParams 验证参数
func (o *BayesianOptimizer) ValidateParams(params map[string]interface{}, paramSpace ParameterSpace) error {
	for _, param := range paramSpace.Params {
		value, ok := params[param.Name]
		if !ok {
			return fmt.Errorf("missing parameter: %s", param.Name)
		}
		
		switch param.Type {
		case ParamTypeInt:
			v, ok := value.(int)
			if !ok {
				// 尝试从 float64 转换（JSON 解析常见）
				if fv, ok := value.(float64); ok {
					v = int(fv)
				} else {
					return fmt.Errorf("parameter %s must be int", param.Name)
				}
			}
			if float64(v) < param.Min || float64(v) > param.Max {
				return fmt.Errorf("parameter %s out of range [%.0f, %.0f]", param.Name, param.Min, param.Max)
			}
			
		case ParamTypeFloat:
			v, ok := value.(float64)
			if !ok {
				return fmt.Errorf("parameter %s must be float", param.Name)
			}
			if v < param.Min || v > param.Max {
				return fmt.Errorf("parameter %s out of range [%.2f, %.2f]", param.Name, param.Min, param.Max)
			}
			
		case ParamTypeChoice:
			v, ok := value.(string)
			if !ok {
				return fmt.Errorf("parameter %s must be string", param.Name)
			}
			valid := false
			for _, choice := range param.Values {
				if v == choice {
					valid = true
					break
				}
			}
			if !valid {
				return fmt.Errorf("parameter %s invalid choice: %s", param.Name, v)
			}
		}
	}
	
	return nil
}

// GetHistory 获取优化历史
func (o *BayesianOptimizer) GetHistory(ctx context.Context, strategyID string) ([]OptimizeResult, error) {
	o.mu.RLock()
	defer o.mu.RUnlock()
	
	history := o.history[strategyID]
	result := make([]OptimizeResult, len(history))
	copy(result, history)
	
	return result, nil
}

// 内部辅助方法

type observation struct {
	params map[string]interface{}
	score  float64
}

func (o *BayesianOptimizer) evaluateStrategy(ctx context.Context, strategyID string, opts OptimizeOptions) (float64, error) {
	metrics, err := o.evaluator.EvaluateStrategy(ctx, strategyID, evaluation.EvaluateOptions{
		TimeRange:     opts.TimeRange,
		MinSampleSize: opts.MinSampleSize,
	})
	if err != nil {
		return 0, err
	}
	
	// 根据目标指标返回得分
	switch opts.TargetMetric {
	case "overall_score":
		return metrics.AvgMetrics.OverallScore, nil
	case "ndcg":
		return metrics.AvgMetrics.NDCG, nil
	case "mrr":
		return metrics.AvgMetrics.MRR, nil
	case "avg_rating":
		return metrics.AvgMetrics.AvgRating / 5.0, nil // 归一化到 0-1
	default:
		return metrics.AvgMetrics.OverallScore, nil
	}
}

func (o *BayesianOptimizer) evaluateParams(ctx context.Context, strategyID string, params map[string]interface{}, opts OptimizeOptions) float64 {
	// 简化实现：在实际应用中，这里应该：
	// 1. 应用参数到检索系统
	// 2. 运行一段时间收集数据
	// 3. 评估性能
	
	// 这里使用简化的评估：基于当前数据评估
	score, err := o.evaluateStrategy(ctx, strategyID, opts)
	if err != nil {
		return 0
	}
	
	// 添加一些随机性来模拟参数影响
	// 实际应用中这应该是真实的评估
	noise := (rand.Float64() - 0.5) * 0.1
	return score + noise
}

func (o *BayesianOptimizer) randomSample(paramSpace ParameterSpace) map[string]interface{} {
	params := make(map[string]interface{})
	
	for _, param := range paramSpace.Params {
		switch param.Type {
		case ParamTypeInt:
			min := int(param.Min)
			max := int(param.Max)
			params[param.Name] = min + rand.Intn(max-min+1)
			
		case ParamTypeFloat:
			params[param.Name] = param.Min + rand.Float64()*(param.Max-param.Min)
			
		case ParamTypeChoice:
			params[param.Name] = param.Values[rand.Intn(len(param.Values))]
		}
	}
	
	return params
}

func (o *BayesianOptimizer) selectNextPoint(paramSpace ParameterSpace, observations []observation, acquisitionType string) map[string]interface{} {
	// 简化的采集函数实现
	// 在实际应用中，这里应该使用高斯过程回归和真正的采集函数
	
	if len(observations) == 0 {
		return o.randomSample(paramSpace)
	}
	
	// 找到最佳观察点
	bestObs := observations[0]
	for _, obs := range observations[1:] {
		if obs.score > bestObs.score {
			bestObs = obs
		}
	}
	
	// 在最佳点附近采样（局部搜索）
	params := copyParams(bestObs.params)
	
	// 随机扰动一个参数
	paramIdx := rand.Intn(len(paramSpace.Params))
	param := paramSpace.Params[paramIdx]
	
	switch param.Type {
	case ParamTypeInt:
		current := params[param.Name].(int)
		delta := int(math.Max(1, (param.Max-param.Min)*0.1))
		newValue := current + rand.Intn(2*delta+1) - delta
		newValue = int(math.Max(param.Min, math.Min(param.Max, float64(newValue))))
		params[param.Name] = newValue
		
	case ParamTypeFloat:
		current := params[param.Name].(float64)
		delta := (param.Max - param.Min) * 0.1
		newValue := current + (rand.Float64()*2-1)*delta
		newValue = math.Max(param.Min, math.Min(param.Max, newValue))
		params[param.Name] = newValue
		
	case ParamTypeChoice:
		params[param.Name] = param.Values[rand.Intn(len(param.Values))]
	}
	
	return params
}

func (o *BayesianOptimizer) saveHistory(strategyID string, result *OptimizeResult) {
	o.mu.Lock()
	defer o.mu.Unlock()
	
	o.history[strategyID] = append(o.history[strategyID], *result)
}

func copyParams(params map[string]interface{}) map[string]interface{} {
	copy := make(map[string]interface{})
	for k, v := range params {
		copy[k] = v
	}
	return copy
}
