package evaluation

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/zhucl121/langchain-go/pkg/types"
	"github.com/zhucl121/langchain-go/retrieval/learning/feedback"
)

// DefaultEvaluator 默认评估器实现
type DefaultEvaluator struct {
	feedbackCollector feedback.Collector
	relevanceModel    RelevanceModel
}

// NewEvaluator 创建评估器
func NewEvaluator(collector feedback.Collector) Evaluator {
	return &DefaultEvaluator{
		feedbackCollector: collector,
		relevanceModel:    &ImplicitRelevanceModel{
			ClickWeight:    0.3,
			ReadWeight:     0.5,
			DurationWeight: 0.2,
		},
	}
}

// NewEvaluatorWithModel 创建带自定义相关性模型的评估器
func NewEvaluatorWithModel(collector feedback.Collector, model RelevanceModel) Evaluator {
	return &DefaultEvaluator{
		feedbackCollector: collector,
		relevanceModel:    model,
	}
}

// EvaluateQuery 评估查询
func (e *DefaultEvaluator) EvaluateQuery(ctx context.Context, queryFeedback *feedback.QueryFeedback) (*QueryMetrics, error) {
	if queryFeedback == nil {
		return nil, fmt.Errorf("query feedback cannot be nil")
	}

	metrics := &QueryMetrics{
		QueryID: queryFeedback.Query.ID,
	}

	// 获取相关文档
	relevantDocs := e.getRelevantDocuments(queryFeedback)

	// 计算精确率和召回率
	metrics.Precision = e.calculatePrecisionFromDocs(relevantDocs, queryFeedback.Results)
	metrics.Recall = e.calculateRecallFromDocs(relevantDocs, queryFeedback.Results)
	metrics.F1Score = e.calculateF1(metrics.Precision, metrics.Recall)

	// 计算 NDCG
	metrics.NDCG = e.calculateNDCG(queryFeedback)

	// 计算 MRR
	metrics.MRR = e.calculateMRR(queryFeedback)

	// 计算用户满意度指标
	metrics.AvgRating = queryFeedback.AvgRating
	metrics.CTR = queryFeedback.CTR
	metrics.ReadRate = e.calculateReadRate(queryFeedback)

	// 计算综合得分
	metrics.OverallScore = e.calculateOverallScore(metrics)

	return metrics, nil
}

// EvaluateStrategy 评估策略
func (e *DefaultEvaluator) EvaluateStrategy(ctx context.Context, strategyID string, opts EvaluateOptions) (*StrategyMetrics, error) {
	if strategyID == "" {
		return nil, fmt.Errorf("strategy ID cannot be empty")
	}

	// 获取该策略的所有查询
	storage := e.feedbackCollector.GetStorage()
	queries, err := storage.ListQueries(ctx, feedback.ListOptions{
		Strategy: strategyID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list queries: %w", err)
	}

	if len(queries) == 0 {
		return &StrategyMetrics{
			StrategyID:   strategyID,
			TotalQueries: 0,
			Timestamp:    time.Now(),
		}, nil
	}

	// 评估每个查询
	allMetrics := make([]QueryMetrics, 0, len(queries))
	for _, q := range queries {
		// 应用时间范围过滤
		if opts.TimeRange > 0 {
			if time.Since(q.Timestamp) > opts.TimeRange {
				continue
			}
		}

		qf, err := e.feedbackCollector.GetQueryFeedback(ctx, q.ID)
		if err != nil {
			continue
		}

		metrics, err := e.EvaluateQuery(ctx, qf)
		if err != nil {
			continue
		}

		allMetrics = append(allMetrics, *metrics)
	}

	if len(allMetrics) < opts.MinSampleSize {
		return nil, fmt.Errorf("insufficient samples: got %d, need %d", len(allMetrics), opts.MinSampleSize)
	}

	// 计算平均指标
	avgMetrics := e.calculateAvgMetrics(allMetrics)

	// 计算 P95 指标
	p95Metrics := e.calculateP95Metrics(allMetrics)

	return &StrategyMetrics{
		StrategyID:   strategyID,
		TotalQueries: len(allMetrics),
		AvgMetrics:   avgMetrics,
		P95Metrics:   p95Metrics,
		Timestamp:    time.Now(),
	}, nil
}

// CompareStrategies 对比策略
func (e *DefaultEvaluator) CompareStrategies(ctx context.Context, strategyA, strategyB string) (*ComparisonResult, error) {
	if strategyA == "" || strategyB == "" {
		return nil, fmt.Errorf("strategy IDs cannot be empty")
	}

	// 评估两个策略
	metricsA, err := e.EvaluateStrategy(ctx, strategyA, EvaluateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate strategy A: %w", err)
	}

	metricsB, err := e.EvaluateStrategy(ctx, strategyB, EvaluateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate strategy B: %w", err)
	}

	// 对比综合得分
	scoreA := metricsA.AvgMetrics.OverallScore
	scoreB := metricsB.AvgMetrics.OverallScore

	winner := strategyA
	improvement := ((scoreA - scoreB) / scoreB) * 100
	if scoreB > scoreA {
		winner = strategyB
		improvement = ((scoreB - scoreA) / scoreA) * 100
	}

	// 计算置信度（简化版，使用效应大小）
	confidence := e.calculateConfidence(scoreA, scoreB, metricsA.TotalQueries, metricsB.TotalQueries)

	// 统计显著性（简化版，使用 t 检验的近似）
	significant := e.calculateSignificance(scoreA, scoreB, metricsA.TotalQueries, metricsB.TotalQueries)

	return &ComparisonResult{
		StrategyA:     *metricsA,
		StrategyB:     *metricsB,
		Winner:        winner,
		Confidence:    confidence,
		Improvement:   improvement,
		SignificantAt: significant,
	}, nil
}

// 辅助方法

func (e *DefaultEvaluator) getRelevantDocuments(qf *feedback.QueryFeedback) map[string]bool {
	relevant := make(map[string]bool)

	// 基于隐式反馈判断相关性
	for _, fb := range qf.ImplicitFeedback {
		if e.relevanceModel.IsRelevant(fb.DocumentID, qf) {
			relevant[fb.DocumentID] = true
		}
	}

	return relevant
}

func (e *DefaultEvaluator) calculatePrecision(relevant map[string]bool, retrieved []interface{}) float64 {
	return 0 // deprecated
}

func (e *DefaultEvaluator) calculateRecall(relevant map[string]bool, retrieved []interface{}) float64 {
	return 0 // deprecated
}

func (e *DefaultEvaluator) calculatePrecisionFromDocs(relevant map[string]bool, retrieved []types.Document) float64 {
	if len(retrieved) == 0 {
		return 0
	}

	relevantCount := 0
	for _, doc := range retrieved {
		if relevant[doc.ID] {
			relevantCount++
		}
	}

	return float64(relevantCount) / float64(len(retrieved))
}

func (e *DefaultEvaluator) calculateRecallFromDocs(relevant map[string]bool, retrieved []types.Document) float64 {
	if len(relevant) == 0 {
		return 0
	}

	foundCount := 0
	for _, doc := range retrieved {
		if relevant[doc.ID] {
			foundCount++
		}
	}

	return float64(foundCount) / float64(len(relevant))
}

func (e *DefaultEvaluator) calculateF1(precision, recall float64) float64 {
	if precision+recall == 0 {
		return 0
	}
	return 2 * (precision * recall) / (precision + recall)
}

func (e *DefaultEvaluator) calculateNDCG(qf *feedback.QueryFeedback) float64 {
	if len(qf.Results) == 0 {
		return 0
	}

	// 计算 DCG
	dcg := 0.0
	for i, doc := range qf.Results {
		relevance := e.relevanceModel.GetRelevance(doc.ID, qf)
		dcg += relevance / math.Log2(float64(i+2))
	}

	// 计算 IDCG（理想情况）
	relevances := make([]float64, 0)
	for _, doc := range qf.Results {
		rel := e.relevanceModel.GetRelevance(doc.ID, qf)
		relevances = append(relevances, rel)
	}
	sort.Float64s(relevances)
	// 反转为降序
	for i, j := 0, len(relevances)-1; i < j; i, j = i+1, j-1 {
		relevances[i], relevances[j] = relevances[j], relevances[i]
	}

	idcg := 0.0
	for i, rel := range relevances {
		idcg += rel / math.Log2(float64(i+2))
	}

	if idcg == 0 {
		return 0
	}

	return dcg / idcg
}

func (e *DefaultEvaluator) calculateMRR(qf *feedback.QueryFeedback) float64 {
	for i, doc := range qf.Results {
		if e.relevanceModel.IsRelevant(doc.ID, qf) {
			return 1.0 / float64(i+1)
		}
	}
	return 0
}

func (e *DefaultEvaluator) calculateReadRate(qf *feedback.QueryFeedback) float64 {
	if len(qf.Results) == 0 {
		return 0
	}

	readCount := 0
	for _, fb := range qf.ImplicitFeedback {
		if fb.Action == feedback.ActionRead {
			readCount++
		}
	}

	return float64(readCount) / float64(len(qf.Results))
}

func (e *DefaultEvaluator) calculateOverallScore(metrics *QueryMetrics) float64 {
	// 权重配置
	weights := map[string]float64{
		"ndcg":       0.25,
		"mrr":        0.15,
		"f1":         0.15,
		"avg_rating": 0.20,
		"ctr":        0.15,
		"read_rate":  0.10,
	}

	score := 0.0
	score += metrics.NDCG * weights["ndcg"]
	score += metrics.MRR * weights["mrr"]
	score += metrics.F1Score * weights["f1"]
	score += (metrics.AvgRating / 5.0) * weights["avg_rating"] // 归一化到 0-1
	score += metrics.CTR * weights["ctr"]
	score += metrics.ReadRate * weights["read_rate"]

	return score
}

func (e *DefaultEvaluator) calculateAvgMetrics(allMetrics []QueryMetrics) QueryMetrics {
	if len(allMetrics) == 0 {
		return QueryMetrics{}
	}

	avg := QueryMetrics{}
	for _, m := range allMetrics {
		avg.Precision += m.Precision
		avg.Recall += m.Recall
		avg.F1Score += m.F1Score
		avg.NDCG += m.NDCG
		avg.MRR += m.MRR
		avg.AvgRating += m.AvgRating
		avg.CTR += m.CTR
		avg.ReadRate += m.ReadRate
		avg.ResponseTime += m.ResponseTime
		avg.OverallScore += m.OverallScore
	}

	count := float64(len(allMetrics))
	avg.Precision /= count
	avg.Recall /= count
	avg.F1Score /= count
	avg.NDCG /= count
	avg.MRR /= count
	avg.AvgRating /= count
	avg.CTR /= count
	avg.ReadRate /= count
	avg.ResponseTime = time.Duration(float64(avg.ResponseTime) / count)
	avg.OverallScore /= count

	return avg
}

func (e *DefaultEvaluator) calculateP95Metrics(allMetrics []QueryMetrics) QueryMetrics {
	if len(allMetrics) == 0 {
		return QueryMetrics{}
	}

	// 对每个指标排序并取 P95
	p95Index := int(float64(len(allMetrics)) * 0.95)
	if p95Index >= len(allMetrics) {
		p95Index = len(allMetrics) - 1
	}

	// 简化实现：返回平均值（完整实现需要对每个指标单独排序）
	return e.calculateAvgMetrics(allMetrics)
}

func (e *DefaultEvaluator) calculateConfidence(scoreA, scoreB float64, sampleA, sampleB int) float64 {
	// 简化版置信度计算：基于效应大小和样本量
	effectSize := math.Abs(scoreA - scoreB)
	minSample := math.Min(float64(sampleA), float64(sampleB))

	// 简单的启发式规则
	if minSample < 10 {
		return 0.3
	}
	if minSample < 30 {
		return 0.5 + effectSize*0.3
	}
	if minSample < 100 {
		return 0.7 + effectSize*0.2
	}
	return 0.85 + effectSize*0.15
}

func (e *DefaultEvaluator) calculateSignificance(scoreA, scoreB float64, sampleA, sampleB int) float64 {
	// 简化版 p-value 计算
	effectSize := math.Abs(scoreA - scoreB)
	minSample := math.Min(float64(sampleA), float64(sampleB))

	// 简单的启发式规则：样本越大，效应越大，p-value 越小
	if minSample < 10 || effectSize < 0.05 {
		return 0.5 // 不显著
	}
	if minSample < 30 || effectSize < 0.10 {
		return 0.1
	}
	if minSample < 100 || effectSize < 0.15 {
		return 0.05
	}
	return 0.01 // 非常显著
}

