package abtest

import (
	"context"
	"fmt"
	"hash/fnv"
	"math"
	"sort"
	"time"
)

// DefaultManager 默认 A/B 测试管理器
type DefaultManager struct {
	storage Storage
}

// NewManager 创建 A/B 测试管理器
func NewManager(storage Storage) Manager {
	return &DefaultManager{
		storage: storage,
	}
}

// CreateExperiment 创建实验
func (m *DefaultManager) CreateExperiment(ctx context.Context, experiment *Experiment) error {
	if experiment == nil {
		return fmt.Errorf("experiment cannot be nil")
	}
	if experiment.ID == "" {
		return fmt.Errorf("experiment ID cannot be empty")
	}
	if len(experiment.Variants) < 2 {
		return fmt.Errorf("experiment must have at least 2 variants")
	}

	// 验证权重总和
	totalWeight := 0.0
	for _, v := range experiment.Variants {
		totalWeight += v.Weight
	}
	if math.Abs(totalWeight-1.0) > 0.01 {
		return fmt.Errorf("variant weights must sum to 1.0, got %.2f", totalWeight)
	}

	// 验证流量比例
	if experiment.Traffic < 0 || experiment.Traffic > 1 {
		return fmt.Errorf("traffic must be between 0 and 1, got %.2f", experiment.Traffic)
	}

	// 设置默认状态
	if experiment.Status == "" {
		experiment.Status = StatusDraft
	}

	return m.storage.SaveExperiment(ctx, experiment)
}

// GetExperiment 获取实验
func (m *DefaultManager) GetExperiment(ctx context.Context, experimentID string) (*Experiment, error) {
	if experimentID == "" {
		return nil, fmt.Errorf("experiment ID cannot be empty")
	}
	return m.storage.GetExperiment(ctx, experimentID)
}

// UpdateExperiment 更新实验
func (m *DefaultManager) UpdateExperiment(ctx context.Context, experiment *Experiment) error {
	if experiment == nil {
		return fmt.Errorf("experiment cannot be nil")
	}
	if experiment.ID == "" {
		return fmt.Errorf("experiment ID cannot be empty")
	}

	// 检查实验是否存在
	existing, err := m.storage.GetExperiment(ctx, experiment.ID)
	if err != nil {
		return fmt.Errorf("experiment not found: %w", err)
	}

	// 运行中的实验不能修改关键配置
	if existing.Status == StatusRunning {
		if len(experiment.Variants) != len(existing.Variants) {
			return fmt.Errorf("cannot change variants of running experiment")
		}
	}

	return m.storage.SaveExperiment(ctx, experiment)
}

// StartExperiment 开始实验
func (m *DefaultManager) StartExperiment(ctx context.Context, experimentID string) error {
	experiment, err := m.storage.GetExperiment(ctx, experimentID)
	if err != nil {
		return err
	}

	if experiment.Status == StatusRunning {
		return fmt.Errorf("experiment already running")
	}

	experiment.Status = StatusRunning
	experiment.StartTime = time.Now()

	return m.storage.SaveExperiment(ctx, experiment)
}

// EndExperiment 结束实验
func (m *DefaultManager) EndExperiment(ctx context.Context, experimentID string, winner string) error {
	experiment, err := m.storage.GetExperiment(ctx, experimentID)
	if err != nil {
		return err
	}

	if experiment.Status == StatusEnded {
		return fmt.Errorf("experiment already ended")
	}

	// 验证 winner
	if winner != "" {
		found := false
		for _, v := range experiment.Variants {
			if v.ID == winner {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("invalid winner variant ID: %s", winner)
		}
	}

	experiment.Status = StatusEnded
	now := time.Now()
	experiment.EndTime = &now
	
	if experiment.Metadata == nil {
		experiment.Metadata = make(map[string]interface{})
	}
	experiment.Metadata["winner"] = winner

	return m.storage.SaveExperiment(ctx, experiment)
}

// AssignVariant 分配变体（用户分流）
func (m *DefaultManager) AssignVariant(ctx context.Context, userID string, experimentID string) (string, error) {
	if userID == "" {
		return "", fmt.Errorf("user ID cannot be empty")
	}
	if experimentID == "" {
		return "", fmt.Errorf("experiment ID cannot be empty")
	}

	// 检查是否已分配
	assignment, err := m.storage.GetAssignment(ctx, experimentID, userID)
	if err == nil && assignment != nil {
		return assignment.VariantID, nil
	}

	// 获取实验配置
	experiment, err := m.storage.GetExperiment(ctx, experimentID)
	if err != nil {
		return "", err
	}

	if experiment.Status != StatusRunning {
		return "", fmt.Errorf("experiment not running")
	}

	// 流量控制：判断用户是否参与实验
	if !m.shouldParticipate(userID, experimentID, experiment.Traffic) {
		return "", fmt.Errorf("user not in experiment traffic")
	}

	// 使用一致性哈希分配变体
	variantID := m.assignVariantByHash(userID, experimentID, experiment.Variants)

	// 保存分配记录
	assignment = &Assignment{
		ExperimentID: experimentID,
		UserID:       userID,
		VariantID:    variantID,
		Timestamp:    time.Now(),
	}
	if err := m.storage.SaveAssignment(ctx, assignment); err != nil {
		return "", err
	}

	return variantID, nil
}

// RecordResult 记录实验结果
func (m *DefaultManager) RecordResult(ctx context.Context, result *ExperimentResult) error {
	if result == nil {
		return fmt.Errorf("result cannot be nil")
	}
	if result.ExperimentID == "" {
		return fmt.Errorf("experiment ID cannot be empty")
	}
	if result.VariantID == "" {
		return fmt.Errorf("variant ID cannot be empty")
	}

	return m.storage.SaveResult(ctx, result)
}

// AnalyzeExperiment 分析实验
func (m *DefaultManager) AnalyzeExperiment(ctx context.Context, experimentID string) (*ExperimentAnalysis, error) {
	// 获取实验配置
	experiment, err := m.storage.GetExperiment(ctx, experimentID)
	if err != nil {
		return nil, err
	}

	// 获取所有结果
	results, err := m.storage.GetResults(ctx, experimentID)
	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return &ExperimentAnalysis{
			ExperimentID: experimentID,
			Variants:     make(map[string]VariantMetrics),
			Completed:    false,
			Timestamp:    time.Now(),
		}, nil
	}

	// 按变体聚合结果
	variantScores := make(map[string][]float64)
	variantMetrics := make(map[string][]float64)
	
	for _, result := range results {
		score := result.Metrics.OverallScore
		variantScores[result.VariantID] = append(variantScores[result.VariantID], score)
	}

	// 计算每个变体的指标
	analysis := &ExperimentAnalysis{
		ExperimentID: experimentID,
		Variants:     make(map[string]VariantMetrics),
		Timestamp:    time.Now(),
	}

	for variantID, scores := range variantScores {
		metrics := VariantMetrics{
			VariantID:    variantID,
			SampleSize:   len(scores),
			AvgScore:     mean(scores),
			StdDev:       stdDev(scores),
			ConfInterval: confidenceInterval(scores, 0.95),
		}
		analysis.Variants[variantID] = metrics
		variantMetrics[variantID] = scores
	}

	// 如果有2个变体，进行 t-test
	if len(experiment.Variants) == 2 {
		variantIDs := make([]string, 0, 2)
		for id := range variantScores {
			variantIDs = append(variantIDs, id)
		}
		
		scoresA := variantScores[variantIDs[0]]
		scoresB := variantScores[variantIDs[1]]
		
		// t-test
		pValue := tTest(scoresA, scoresB)
		analysis.PValue = pValue
		
		// 确定获胜者
		avgA := mean(scoresA)
		avgB := mean(scoresB)
		
		if avgA > avgB {
			analysis.Winner = variantIDs[0]
		} else {
			analysis.Winner = variantIDs[1]
		}
		
		// 置信度
		analysis.Confidence = 1.0 - pValue
		
		// 判断是否有明确结论（p < 0.05 且样本充足）
		minSampleSize := 30
		if pValue < 0.05 && len(scoresA) >= minSampleSize && len(scoresB) >= minSampleSize {
			analysis.Completed = true
		}
	}

	return analysis, nil
}

// ListExperiments 列出实验
func (m *DefaultManager) ListExperiments(ctx context.Context, status ExperimentStatus) ([]*Experiment, error) {
	return m.storage.ListExperiments(ctx, status)
}

// 内部辅助方法

func (m *DefaultManager) shouldParticipate(userID, experimentID string, traffic float64) bool {
	if traffic >= 1.0 {
		return true
	}
	
	// 使用哈希确定用户是否在流量范围内
	hash := hashString(userID + experimentID + "traffic")
	return hash < traffic
}

func (m *DefaultManager) assignVariantByHash(userID, experimentID string, variants []Variant) string {
	// 使用一致性哈希确保用户始终分配到相同变体
	hash := hashString(userID + experimentID)
	
	// 根据权重分配
	cumulative := 0.0
	for _, variant := range variants {
		cumulative += variant.Weight
		if hash < cumulative {
			return variant.ID
		}
	}
	
	// 默认返回第一个变体
	return variants[0].ID
}

func hashString(s string) float64 {
	h := fnv.New64a()
	h.Write([]byte(s))
	return float64(h.Sum64()) / float64(^uint64(0))
}

// 统计函数

func mean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func stdDev(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	m := mean(values)
	sum := 0.0
	for _, v := range values {
		diff := v - m
		sum += diff * diff
	}
	return math.Sqrt(sum / float64(len(values)))
}

func confidenceInterval(values []float64, confidence float64) [2]float64 {
	if len(values) < 2 {
		return [2]float64{0, 0}
	}
	
	m := mean(values)
	sd := stdDev(values)
	n := float64(len(values))
	
	// 使用 t 分布的近似（实际应该查表）
	// 这里简化为使用 1.96 (95% 置信度)
	tValue := 1.96
	margin := tValue * sd / math.Sqrt(n)
	
	return [2]float64{m - margin, m + margin}
}

func tTest(samplesA, samplesB []float64) float64 {
	if len(samplesA) < 2 || len(samplesB) < 2 {
		return 1.0 // 样本不足，返回最大 p-value
	}
	
	meanA := mean(samplesA)
	meanB := mean(samplesB)
	sdA := stdDev(samplesA)
	sdB := stdDev(samplesB)
	nA := float64(len(samplesA))
	nB := float64(len(samplesB))
	
	// 合并标准差
	pooledSD := math.Sqrt((sdA*sdA)/nA + (sdB*sdB)/nB)
	
	if pooledSD == 0 {
		return 1.0
	}
	
	// t 统计量
	t := math.Abs((meanA - meanB) / pooledSD)
	
	// 简化的 p-value 估计（实际应该查 t 分布表）
	// 这里使用启发式方法
	if t < 1.0 {
		return 0.5
	} else if t < 2.0 {
		return 0.1
	} else if t < 3.0 {
		return 0.05
	} else {
		return 0.01
	}
}

func sortFloat64(values []float64) []float64 {
	sorted := make([]float64, len(values))
	copy(sorted, values)
	sort.Float64s(sorted)
	return sorted
}
