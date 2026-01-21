package optimization

import (
	"context"
	"testing"
	"time"

	"github.com/zhucl121/langchain-go/pkg/types"
	"github.com/zhucl121/langchain-go/retrieval/learning/evaluation"
	"github.com/zhucl121/langchain-go/retrieval/learning/feedback"
)

func setupTestOptimizer(t *testing.T) (Optimizer, string) {
	storage := feedback.NewMemoryStorage()
	collector := feedback.NewCollector(storage)
	evaluator := evaluation.NewEvaluator(collector)
	
	ctx := context.Background()
	
	// 创建测试数据
	strategyID := "test-strategy"
	for i := 0; i < 10; i++ {
		queryID := "query-" + string(rune('0'+i))
		query := &feedback.Query{
			ID:        queryID,
			Text:      "test query",
			UserID:    "user1",
			Strategy:  strategyID,
			Timestamp: time.Now(),
		}
		collector.RecordQuery(ctx, query)
		
		results := []types.Document{
			{ID: "doc1", Content: "content1"},
			{ID: "doc2", Content: "content2"},
		}
		collector.RecordResults(ctx, queryID, results)
		
		collector.CollectExplicitFeedback(ctx, &feedback.ExplicitFeedback{
			QueryID:   queryID,
			Type:      feedback.FeedbackTypeRating,
			Rating:    4 + (i % 2),
			Timestamp: time.Now(),
		})
	}
	
	optimizer := NewOptimizer(evaluator, collector, DefaultConfig())
	return optimizer, strategyID
}

func TestOptimizer_Optimize(t *testing.T) {
	optimizer, strategyID := setupTestOptimizer(t)
	ctx := context.Background()
	
	// 定义参数空间
	paramSpace := ParameterSpace{
		Params: []Parameter{
			{
				Name:    "top_k",
				Type:    ParamTypeInt,
				Min:     5,
				Max:     20,
				Default: 10,
			},
			{
				Name:    "temperature",
				Type:    ParamTypeFloat,
				Min:     0.1,
				Max:     1.0,
				Default: 0.7,
			},
		},
	}
	
	// 优化参数
	result, err := optimizer.Optimize(ctx, strategyID, paramSpace, OptimizeOptions{
		MaxIterations: 10,
		TargetMetric:  "overall_score",
		MinSampleSize: 5,
	})
	
	if err != nil {
		t.Fatalf("Optimize() error = %v", err)
	}
	
	// 验证结果
	if result.StrategyID != strategyID {
		t.Errorf("expected strategy ID %s, got %s", strategyID, result.StrategyID)
	}
	
	if result.Iterations != 10 {
		t.Errorf("expected 10 iterations, got %d", result.Iterations)
	}
	
	if len(result.BestParams) != 2 {
		t.Errorf("expected 2 parameters, got %d", len(result.BestParams))
	}
	
	if result.BestScore < 0 || result.BestScore > 1 {
		t.Errorf("best score should be between 0 and 1, got %f", result.BestScore)
	}
	
	if len(result.History) != 10 {
		t.Errorf("expected 10 history entries, got %d", len(result.History))
	}
	
	t.Logf("Optimization result: BestScore=%.3f, Improvement=%.2f%%", 
		result.BestScore, result.Improvement)
}

func TestOptimizer_ValidateParams(t *testing.T) {
	optimizer, _ := setupTestOptimizer(t)
	
	paramSpace := ParameterSpace{
		Params: []Parameter{
			{Name: "top_k", Type: ParamTypeInt, Min: 5, Max: 20, Default: 10},
			{Name: "temperature", Type: ParamTypeFloat, Min: 0.1, Max: 1.0, Default: 0.7},
			{Name: "strategy", Type: ParamTypeChoice, Values: []string{"hybrid", "vector", "graph"}, Default: "hybrid"},
		},
	}
	
	tests := []struct {
		name    string
		params  map[string]interface{}
		wantErr bool
	}{
		{
			name: "valid params",
			params: map[string]interface{}{
				"top_k":       10,
				"temperature": 0.7,
				"strategy":    "hybrid",
			},
			wantErr: false,
		},
		{
			name: "missing param",
			params: map[string]interface{}{
				"top_k":       10,
				"temperature": 0.7,
			},
			wantErr: true,
		},
		{
			name: "out of range int",
			params: map[string]interface{}{
				"top_k":       25,
				"temperature": 0.7,
				"strategy":    "hybrid",
			},
			wantErr: true,
		},
		{
			name: "out of range float",
			params: map[string]interface{}{
				"top_k":       10,
				"temperature": 1.5,
				"strategy":    "hybrid",
			},
			wantErr: true,
		},
		{
			name: "invalid choice",
			params: map[string]interface{}{
				"top_k":       10,
				"temperature": 0.7,
				"strategy":    "invalid",
			},
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := optimizer.ValidateParams(tt.params, paramSpace)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateParams() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestOptimizer_SuggestParams(t *testing.T) {
	optimizer, strategyID := setupTestOptimizer(t)
	ctx := context.Background()
	
	paramSpace := ParameterSpace{
		Params: []Parameter{
			{Name: "top_k", Type: ParamTypeInt, Min: 5, Max: 20, Default: 10},
			{Name: "temperature", Type: ParamTypeFloat, Min: 0.1, Max: 1.0, Default: 0.7},
		},
	}
	
	// 第一次建议（没有历史）
	params1, err := optimizer.SuggestParams(ctx, strategyID, paramSpace)
	if err != nil {
		t.Fatalf("SuggestParams() error = %v", err)
	}
	
	if len(params1) != 2 {
		t.Errorf("expected 2 parameters, got %d", len(params1))
	}
	
	// 运行优化
	optimizer.Optimize(ctx, strategyID, paramSpace, OptimizeOptions{
		MaxIterations: 5,
	})
	
	// 第二次建议（有历史）
	params2, err := optimizer.SuggestParams(ctx, strategyID, paramSpace)
	if err != nil {
		t.Fatalf("SuggestParams() error = %v", err)
	}
	
	if len(params2) != 2 {
		t.Errorf("expected 2 parameters, got %d", len(params2))
	}
}

func TestOptimizer_GetHistory(t *testing.T) {
	// 使用一个共享的优化器实例
	storage := feedback.NewMemoryStorage()
	collector := feedback.NewCollector(storage)
	evaluator := evaluation.NewEvaluator(collector)
	optimizer := NewOptimizer(evaluator, collector, DefaultConfig())
	
	ctx := context.Background()
	strategyID := "test-strategy"
	
	// 创建测试数据
	for i := 0; i < 10; i++ {
		queryID := "query-" + string(rune('0'+i))
		query := &feedback.Query{
			ID:        queryID,
			Text:      "test query",
			UserID:    "user1",
			Strategy:  strategyID,
			Timestamp: time.Now(),
		}
		collector.RecordQuery(ctx, query)
		
		results := []types.Document{
			{ID: "doc1", Content: "content1"},
		}
		collector.RecordResults(ctx, queryID, results)
		
		collector.CollectExplicitFeedback(ctx, &feedback.ExplicitFeedback{
			QueryID:   queryID,
			Type:      feedback.FeedbackTypeRating,
			Rating:    4,
			Timestamp: time.Now(),
		})
	}
	
	paramSpace := ParameterSpace{
		Params: []Parameter{
			{Name: "top_k", Type: ParamTypeInt, Min: 5, Max: 20, Default: 10},
		},
	}
	
	// 初始没有历史
	history1, err := optimizer.GetHistory(ctx, strategyID)
	if err != nil {
		t.Fatalf("GetHistory() error = %v", err)
	}
	if len(history1) != 0 {
		t.Errorf("expected 0 history entries, got %d", len(history1))
	}
	
	// 运行优化
	_, err = optimizer.Optimize(ctx, strategyID, paramSpace, OptimizeOptions{
		MaxIterations: 5,
		MinSampleSize: 5,
	})
	if err != nil {
		t.Fatalf("Optimize() error = %v", err)
	}
	
	// 应该有历史记录
	history2, err := optimizer.GetHistory(ctx, strategyID)
	if err != nil {
		t.Fatalf("GetHistory() error = %v", err)
	}
	if len(history2) != 1 {
		t.Errorf("expected 1 history entry, got %d", len(history2))
	}
	
	// 再次优化
	_, err = optimizer.Optimize(ctx, strategyID, paramSpace, OptimizeOptions{
		MaxIterations: 5,
		MinSampleSize: 5,
	})
	if err != nil {
		t.Fatalf("Optimize() error = %v", err)
	}
	
	// 应该有2条历史记录
	history3, err := optimizer.GetHistory(ctx, strategyID)
	if err != nil {
		t.Fatalf("GetHistory() error = %v", err)
	}
	if len(history3) != 2 {
		t.Errorf("expected 2 history entries, got %d", len(history3))
	}
}

func TestParameterTypes(t *testing.T) {
	optimizer, _ := setupTestOptimizer(t)
	
	tests := []struct {
		name       string
		paramSpace ParameterSpace
		wantParams int
	}{
		{
			name: "int parameter",
			paramSpace: ParameterSpace{
				Params: []Parameter{
					{Name: "top_k", Type: ParamTypeInt, Min: 5, Max: 20, Default: 10},
				},
			},
			wantParams: 1,
		},
		{
			name: "float parameter",
			paramSpace: ParameterSpace{
				Params: []Parameter{
					{Name: "temperature", Type: ParamTypeFloat, Min: 0.0, Max: 1.0, Default: 0.5},
				},
			},
			wantParams: 1,
		},
		{
			name: "choice parameter",
			paramSpace: ParameterSpace{
				Params: []Parameter{
					{Name: "mode", Type: ParamTypeChoice, Values: []string{"a", "b", "c"}, Default: "a"},
				},
			},
			wantParams: 1,
		},
		{
			name: "mixed parameters",
			paramSpace: ParameterSpace{
				Params: []Parameter{
					{Name: "top_k", Type: ParamTypeInt, Min: 5, Max: 20, Default: 10},
					{Name: "temperature", Type: ParamTypeFloat, Min: 0.0, Max: 1.0, Default: 0.5},
					{Name: "mode", Type: ParamTypeChoice, Values: []string{"a", "b"}, Default: "a"},
				},
			},
			wantParams: 3,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := make(map[string]interface{})
			for _, param := range tt.paramSpace.Params {
				params[param.Name] = param.Default
			}
			
			err := optimizer.ValidateParams(params, tt.paramSpace)
			if err != nil {
				t.Errorf("ValidateParams() error = %v", err)
			}
			
			if len(params) != tt.wantParams {
				t.Errorf("expected %d parameters, got %d", tt.wantParams, len(params))
			}
		})
	}
}
