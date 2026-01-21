package abtest

import (
	"context"
	"fmt"
	"sync"
)

// MemoryStorage 内存存储实现
type MemoryStorage struct {
	mu          sync.RWMutex
	experiments map[string]*Experiment
	assignments map[string]map[string]*Assignment // experimentID -> userID -> Assignment
	results     map[string][]*ExperimentResult    // experimentID -> []Result
}

// NewMemoryStorage 创建内存存储
func NewMemoryStorage() Storage {
	return &MemoryStorage{
		experiments: make(map[string]*Experiment),
		assignments: make(map[string]map[string]*Assignment),
		results:     make(map[string][]*ExperimentResult),
	}
}

// SaveExperiment 保存实验
func (s *MemoryStorage) SaveExperiment(ctx context.Context, experiment *Experiment) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 深拷贝
	exp := *experiment
	exp.Variants = make([]Variant, len(experiment.Variants))
	copy(exp.Variants, experiment.Variants)
	
	s.experiments[experiment.ID] = &exp
	return nil
}

// GetExperiment 获取实验
func (s *MemoryStorage) GetExperiment(ctx context.Context, experimentID string) (*Experiment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	exp, ok := s.experiments[experimentID]
	if !ok {
		return nil, fmt.Errorf("experiment not found: %s", experimentID)
	}

	// 深拷贝
	result := *exp
	result.Variants = make([]Variant, len(exp.Variants))
	copy(result.Variants, exp.Variants)

	return &result, nil
}

// ListExperiments 列出实验
func (s *MemoryStorage) ListExperiments(ctx context.Context, status ExperimentStatus) ([]*Experiment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var experiments []*Experiment
	for _, exp := range s.experiments {
		if status == "" || exp.Status == status {
			// 深拷贝
			expCopy := *exp
			expCopy.Variants = make([]Variant, len(exp.Variants))
			copy(expCopy.Variants, exp.Variants)
			experiments = append(experiments, &expCopy)
		}
	}

	return experiments, nil
}

// SaveAssignment 保存分配
func (s *MemoryStorage) SaveAssignment(ctx context.Context, assignment *Assignment) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.assignments[assignment.ExperimentID] == nil {
		s.assignments[assignment.ExperimentID] = make(map[string]*Assignment)
	}

	s.assignments[assignment.ExperimentID][assignment.UserID] = assignment
	return nil
}

// GetAssignment 获取分配
func (s *MemoryStorage) GetAssignment(ctx context.Context, experimentID string, userID string) (*Assignment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.assignments[experimentID] == nil {
		return nil, fmt.Errorf("no assignments for experiment: %s", experimentID)
	}

	assignment, ok := s.assignments[experimentID][userID]
	if !ok {
		return nil, fmt.Errorf("assignment not found for user: %s", userID)
	}

	return assignment, nil
}

// SaveResult 保存结果
func (s *MemoryStorage) SaveResult(ctx context.Context, result *ExperimentResult) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.results[result.ExperimentID] = append(s.results[result.ExperimentID], result)
	return nil
}

// GetResults 获取结果
func (s *MemoryStorage) GetResults(ctx context.Context, experimentID string) ([]*ExperimentResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	results := s.results[experimentID]
	if results == nil {
		return []*ExperimentResult{}, nil
	}

	// 返回拷贝
	resultsCopy := make([]*ExperimentResult, len(results))
	copy(resultsCopy, results)

	return resultsCopy, nil
}
