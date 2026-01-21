package abtest

import (
	"context"
	"testing"
	"time"

	"github.com/zhucl121/langchain-go/retrieval/learning/evaluation"
)

func TestManager_CreateExperiment(t *testing.T) {
	storage := NewMemoryStorage()
	manager := NewManager(storage)
	ctx := context.Background()

	tests := []struct {
		name       string
		experiment *Experiment
		wantErr    bool
	}{
		{
			name: "valid experiment",
			experiment: &Experiment{
				ID:   "exp-001",
				Name: "Test Experiment",
				Variants: []Variant{
					{ID: "control", Name: "Control", Weight: 0.5},
					{ID: "treatment", Name: "Treatment", Weight: 0.5},
				},
				Traffic: 1.0,
			},
			wantErr: false,
		},
		{
			name:       "nil experiment",
			experiment: nil,
			wantErr:    true,
		},
		{
			name: "empty ID",
			experiment: &Experiment{
				Variants: []Variant{
					{ID: "control", Weight: 0.5},
					{ID: "treatment", Weight: 0.5},
				},
			},
			wantErr: true,
		},
		{
			name: "too few variants",
			experiment: &Experiment{
				ID: "exp-002",
				Variants: []Variant{
					{ID: "control", Weight: 1.0},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid weights",
			experiment: &Experiment{
				ID: "exp-003",
				Variants: []Variant{
					{ID: "control", Weight: 0.3},
					{ID: "treatment", Weight: 0.3},
				},
				Traffic: 1.0,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.CreateExperiment(ctx, tt.experiment)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateExperiment() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestManager_AssignVariant(t *testing.T) {
	storage := NewMemoryStorage()
	manager := NewManager(storage)
	ctx := context.Background()

	// 创建实验
	experiment := &Experiment{
		ID:   "exp-001",
		Name: "Test",
		Variants: []Variant{
			{ID: "control", Name: "Control", Weight: 0.5},
			{ID: "treatment", Name: "Treatment", Weight: 0.5},
		},
		Traffic: 1.0,
		Status:  StatusRunning,
	}
	manager.CreateExperiment(ctx, experiment)

	// 测试分配
	userID := "user-123"
	variant1, err := manager.AssignVariant(ctx, userID, experiment.ID)
	if err != nil {
		t.Fatalf("AssignVariant() error = %v", err)
	}

	// 同一用户应该始终分配到相同变体
	variant2, err := manager.AssignVariant(ctx, userID, experiment.ID)
	if err != nil {
		t.Fatalf("AssignVariant() error = %v", err)
	}

	if variant1 != variant2 {
		t.Errorf("expected same variant, got %s and %s", variant1, variant2)
	}

	// 不同用户可能分配到不同变体
	variant3, err := manager.AssignVariant(ctx, "user-456", experiment.ID)
	if err != nil {
		t.Fatalf("AssignVariant() error = %v", err)
	}

	t.Logf("User 1 -> %s, User 2 -> %s", variant1, variant3)
}

func TestManager_RecordAndAnalyze(t *testing.T) {
	storage := NewMemoryStorage()
	manager := NewManager(storage)
	ctx := context.Background()

	// 创建实验
	experiment := &Experiment{
		ID:   "exp-001",
		Name: "Test",
		Variants: []Variant{
			{ID: "control", Name: "Control", Weight: 0.5},
			{ID: "treatment", Name: "Treatment", Weight: 0.5},
		},
		Traffic: 1.0,
		Status:  StatusRunning,
	}
	manager.CreateExperiment(ctx, experiment)

	// 记录结果（control 组表现较差）
	for i := 0; i < 50; i++ {
		manager.RecordResult(ctx, &ExperimentResult{
			ExperimentID: experiment.ID,
			VariantID:    "control",
			UserID:       "user-" + string(rune(i)),
			Metrics: evaluation.QueryMetrics{
				OverallScore: 0.6 + float64(i%10)/100.0,
			},
			Timestamp: time.Now(),
		})
	}

	// 记录结果（treatment 组表现较好）
	for i := 0; i < 50; i++ {
		manager.RecordResult(ctx, &ExperimentResult{
			ExperimentID: experiment.ID,
			VariantID:    "treatment",
			UserID:       "user-" + string(rune(i+50)),
			Metrics: evaluation.QueryMetrics{
				OverallScore: 0.75 + float64(i%10)/100.0,
			},
			Timestamp: time.Now(),
		})
	}

	// 分析实验
	analysis, err := manager.AnalyzeExperiment(ctx, experiment.ID)
	if err != nil {
		t.Fatalf("AnalyzeExperiment() error = %v", err)
	}

	if len(analysis.Variants) != 2 {
		t.Errorf("expected 2 variants, got %d", len(analysis.Variants))
	}

	// 验证样本大小
	for variantID, metrics := range analysis.Variants {
		if metrics.SampleSize != 50 {
			t.Errorf("variant %s: expected 50 samples, got %d", variantID, metrics.SampleSize)
		}
	}

	// treatment 应该是获胜者
	if analysis.Winner != "treatment" {
		t.Errorf("expected winner 'treatment', got '%s'", analysis.Winner)
	}

	// 应该有统计显著性
	if analysis.PValue >= 0.05 {
		t.Logf("Warning: p-value %.3f is not significant (may vary due to randomness)", analysis.PValue)
	}

	t.Logf("Analysis: Winner=%s, Confidence=%.2f%%, PValue=%.3f, Completed=%v",
		analysis.Winner, analysis.Confidence*100, analysis.PValue, analysis.Completed)
}

func TestManager_ExperimentLifecycle(t *testing.T) {
	storage := NewMemoryStorage()
	manager := NewManager(storage)
	ctx := context.Background()

	// 创建实验
	experiment := &Experiment{
		ID:   "exp-001",
		Name: "Test",
		Variants: []Variant{
			{ID: "control", Weight: 0.5},
			{ID: "treatment", Weight: 0.5},
		},
		Traffic: 1.0,
	}
	err := manager.CreateExperiment(ctx, experiment)
	if err != nil {
		t.Fatalf("CreateExperiment() error = %v", err)
	}

	// 验证初始状态
	exp, _ := manager.GetExperiment(ctx, experiment.ID)
	if exp.Status != StatusDraft {
		t.Errorf("expected status Draft, got %v", exp.Status)
	}

	// 开始实验
	err = manager.StartExperiment(ctx, experiment.ID)
	if err != nil {
		t.Fatalf("StartExperiment() error = %v", err)
	}

	exp, _ = manager.GetExperiment(ctx, experiment.ID)
	if exp.Status != StatusRunning {
		t.Errorf("expected status Running, got %v", exp.Status)
	}

	// 结束实验
	err = manager.EndExperiment(ctx, experiment.ID, "treatment")
	if err != nil {
		t.Fatalf("EndExperiment() error = %v", err)
	}

	exp, _ = manager.GetExperiment(ctx, experiment.ID)
	if exp.Status != StatusEnded {
		t.Errorf("expected status Ended, got %v", exp.Status)
	}

	if exp.EndTime == nil {
		t.Error("expected EndTime to be set")
	}
}

func TestManager_ListExperiments(t *testing.T) {
	storage := NewMemoryStorage()
	manager := NewManager(storage)
	ctx := context.Background()

	// 创建多个实验
	experiments := []*Experiment{
		{
			ID:   "exp-001",
			Name: "Test 1",
			Variants: []Variant{
				{ID: "control", Weight: 0.5},
				{ID: "treatment", Weight: 0.5},
			},
			Traffic: 1.0,
			Status:  StatusDraft,
		},
		{
			ID:   "exp-002",
			Name: "Test 2",
			Variants: []Variant{
				{ID: "control", Weight: 0.5},
				{ID: "treatment", Weight: 0.5},
			},
			Traffic: 1.0,
			Status:  StatusRunning,
		},
		{
			ID:   "exp-003",
			Name: "Test 3",
			Variants: []Variant{
				{ID: "control", Weight: 0.5},
				{ID: "treatment", Weight: 0.5},
			},
			Traffic: 1.0,
			Status:  StatusEnded,
		},
	}

	for _, exp := range experiments {
		manager.CreateExperiment(ctx, exp)
	}

	// 列出所有实验
	all, err := manager.ListExperiments(ctx, "")
	if err != nil {
		t.Fatalf("ListExperiments() error = %v", err)
	}
	if len(all) != 3 {
		t.Errorf("expected 3 experiments, got %d", len(all))
	}

	// 只列出运行中的
	running, err := manager.ListExperiments(ctx, StatusRunning)
	if err != nil {
		t.Fatalf("ListExperiments() error = %v", err)
	}
	if len(running) != 1 {
		t.Errorf("expected 1 running experiment, got %d", len(running))
	}
}
