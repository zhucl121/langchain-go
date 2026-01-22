package a2a

import (
	"context"
	"fmt"
	"math"
	"sync"
)

// TaskRouter routes tasks to appropriate agents.
type TaskRouter interface {
	// Route routes a task to a single agent
	Route(ctx context.Context, task *Task) (A2AAgent, error)
	
	// RouteMultiple routes a task to multiple agents
	RouteMultiple(ctx context.Context, task *Task, count int) ([]A2AAgent, error)
	
	// RouteWithLoadBalancing routes considering load balancing
	RouteWithLoadBalancing(ctx context.Context, task *Task) (A2AAgent, error)
}

// RoutingStrategy represents routing strategy.
type RoutingStrategy string

const (
	StrategyCapability  RoutingStrategy = "capability"   // Based on capability matching
	StrategyLoad        RoutingStrategy = "load"         // Based on load
	StrategyPerformance RoutingStrategy = "performance"  // Based on performance
	StrategyHybrid      RoutingStrategy = "hybrid"       // Hybrid strategy (recommended)
)

// RouterConfig configures a task router.
type RouterConfig struct {
	Strategy RoutingStrategy
	Scorer   *AgentScorer
}

// SmartTaskRouter is an intelligent task router.
type SmartTaskRouter struct {
	registry AgentRegistry
	config   RouterConfig
	
	// Agent metrics
	metrics   map[string]*AgentMetrics
	metricsMu sync.RWMutex
}

// AgentMetrics tracks agent performance metrics.
type AgentMetrics struct {
	TotalTasks     int
	CompletedTasks int
	FailedTasks    int
	AvgResponseTime float64
	SuccessRate     float64
	CurrentLoad     int
}

// AgentScorer scores agents for task matching.
type AgentScorer struct {
	Weights *ScoringWeights
}

// ScoringWeights defines scoring weights for different factors.
type ScoringWeights struct {
	CapabilityMatch float64 // Weight for capability matching (0-1)
	Load            float64 // Weight for load (0-1)
	Performance     float64 // Weight for performance (0-1)
	Reputation      float64 // Weight for reputation (0-1)
}

// DefaultScoringWeights returns default scoring weights.
func DefaultScoringWeights() *ScoringWeights {
	return &ScoringWeights{
		CapabilityMatch: 0.4,
		Load:            0.3,
		Performance:     0.2,
		Reputation:      0.1,
	}
}

// NewSmartTaskRouter creates a new smart task router.
func NewSmartTaskRouter(registry AgentRegistry, config RouterConfig) *SmartTaskRouter {
	// Set defaults
	if config.Strategy == "" {
		config.Strategy = StrategyHybrid
	}
	
	if config.Scorer == nil {
		config.Scorer = &AgentScorer{
			Weights: DefaultScoringWeights(),
		}
	}
	
	return &SmartTaskRouter{
		registry: registry,
		config:   config,
		metrics:  make(map[string]*AgentMetrics),
	}
}

// Route routes a task to the best matching agent.
func (r *SmartTaskRouter) Route(ctx context.Context, task *Task) (A2AAgent, error) {
	// Get all agents
	agents, err := r.registry.ListAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("list agents: %w", err)
	}
	
	if len(agents) == 0 {
		return nil, fmt.Errorf("no agents available")
	}
	
	// Score each agent
	type scoredAgent struct {
		agent A2AAgent
		score float64
	}
	
	scored := make([]scoredAgent, 0, len(agents))
	for _, agent := range agents {
		score, err := r.scoreAgent(ctx, agent, task)
		if err != nil {
			continue // Skip agents that can't be scored
		}
		
		scored = append(scored, scoredAgent{
			agent: agent,
			score: score,
		})
	}
	
	if len(scored) == 0 {
		return nil, fmt.Errorf("no suitable agents found")
	}
	
	// Find agent with highest score
	best := scored[0]
	for _, sa := range scored[1:] {
		if sa.score > best.score {
			best = sa
		}
	}
	
	return best.agent, nil
}

// RouteMultiple routes a task to multiple agents.
func (r *SmartTaskRouter) RouteMultiple(ctx context.Context, task *Task, count int) ([]A2AAgent, error) {
	agents, err := r.registry.ListAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("list agents: %w", err)
	}
	
	if len(agents) == 0 {
		return nil, fmt.Errorf("no agents available")
	}
	
	if count > len(agents) {
		count = len(agents)
	}
	
	// Score and sort agents
	type scoredAgent struct {
		agent A2AAgent
		score float64
	}
	
	scored := make([]scoredAgent, 0, len(agents))
	for _, agent := range agents {
		score, err := r.scoreAgent(ctx, agent, task)
		if err != nil {
			continue
		}
		scored = append(scored, scoredAgent{agent, score})
	}
	
	// Sort by score (descending)
	for i := 0; i < len(scored)-1; i++ {
		for j := i + 1; j < len(scored); j++ {
			if scored[j].score > scored[i].score {
				scored[i], scored[j] = scored[j], scored[i]
			}
		}
	}
	
	// Return top N agents
	result := make([]A2AAgent, 0, count)
	for i := 0; i < count && i < len(scored); i++ {
		result = append(result, scored[i].agent)
	}
	
	return result, nil
}

// RouteWithLoadBalancing routes considering load balancing.
func (r *SmartTaskRouter) RouteWithLoadBalancing(ctx context.Context, task *Task) (A2AAgent, error) {
	// Use load-focused strategy
	oldStrategy := r.config.Strategy
	r.config.Strategy = StrategyLoad
	defer func() { r.config.Strategy = oldStrategy }()
	
	return r.Route(ctx, task)
}

// scoreAgent scores an agent for a task.
func (r *SmartTaskRouter) scoreAgent(ctx context.Context, agent A2AAgent, task *Task) (float64, error) {
	info, err := agent.GetInfo(ctx)
	if err != nil {
		return 0, err
	}
	
	caps, err := agent.GetCapabilities(ctx)
	if err != nil {
		return 0, err
	}
	
	// Check if agent is online
	if info.Status != AgentStatusOnline {
		return 0, nil
	}
	
	var totalScore float64
	
	switch r.config.Strategy {
	case StrategyCapability:
		totalScore = r.scoreCapability(caps, task)
		
	case StrategyLoad:
		totalScore = r.scoreLoad(info.ID)
		
	case StrategyPerformance:
		totalScore = r.scorePerformance(info.ID)
		
	case StrategyHybrid:
		capScore := r.scoreCapability(caps, task)
		loadScore := r.scoreLoad(info.ID)
		perfScore := r.scorePerformance(info.ID)
		repScore := r.scoreReputation(info.ID)
		
		w := r.config.Scorer.Weights
		totalScore = capScore*w.CapabilityMatch +
			loadScore*w.Load +
			perfScore*w.Performance +
			repScore*w.Reputation
	}
	
	return totalScore, nil
}

// scoreCapability scores based on capability matching.
func (r *SmartTaskRouter) scoreCapability(caps *AgentCapabilities, task *Task) float64 {
	if task.Requirements == nil || len(task.Requirements.RequiredTools) == 0 {
		// No specific requirements, any agent is suitable
		return 0.8
	}
	
	// Count matched tools
	requiredTools := task.Requirements.RequiredTools
	availableTools := caps.Tools
	
	matched := 0
	for _, required := range requiredTools {
		for _, available := range availableTools {
			if required == available {
				matched++
				break
			}
		}
	}
	
	if len(requiredTools) == 0 {
		return 1.0
	}
	
	return float64(matched) / float64(len(requiredTools))
}

// scoreLoad scores based on current load.
func (r *SmartTaskRouter) scoreLoad(agentID string) float64 {
	r.metricsMu.RLock()
	defer r.metricsMu.RUnlock()
	
	metrics, ok := r.metrics[agentID]
	if !ok {
		return 1.0 // No load data, assume available
	}
	
	// Calculate load score (lower load = higher score)
	if metrics.CurrentLoad == 0 {
		return 1.0
	}
	
	// Assume max load is 10
	loadRatio := float64(metrics.CurrentLoad) / 10.0
	if loadRatio > 1.0 {
		loadRatio = 1.0
	}
	
	return 1.0 - loadRatio
}

// scorePerformance scores based on historical performance.
func (r *SmartTaskRouter) scorePerformance(agentID string) float64 {
	r.metricsMu.RLock()
	defer r.metricsMu.RUnlock()
	
	metrics, ok := r.metrics[agentID]
	if !ok {
		return 0.5 // No performance data, neutral score
	}
	
	// Performance score based on success rate and response time
	successScore := metrics.SuccessRate
	
	// Normalize response time (assume 5s is baseline, faster is better)
	timeScore := 1.0
	if metrics.AvgResponseTime > 0 {
		timeScore = math.Max(0, 1.0-(metrics.AvgResponseTime/5.0))
	}
	
	// Weighted average
	return successScore*0.7 + timeScore*0.3
}

// scoreReputation scores based on reputation.
func (r *SmartTaskRouter) scoreReputation(agentID string) float64 {
	r.metricsMu.RLock()
	defer r.metricsMu.RUnlock()
	
	metrics, ok := r.metrics[agentID]
	if !ok {
		return 0.5 // No reputation data
	}
	
	// Simple reputation based on total completed tasks
	if metrics.TotalTasks == 0 {
		return 0.5
	}
	
	// More tasks completed = higher reputation
	// Normalize to 0-1 range (assume 100 tasks is excellent)
	return math.Min(1.0, float64(metrics.CompletedTasks)/100.0)
}

// UpdateMetrics updates agent metrics after task completion.
func (r *SmartTaskRouter) UpdateMetrics(agentID string, success bool, responseTime float64) {
	r.metricsMu.Lock()
	defer r.metricsMu.Unlock()
	
	metrics, ok := r.metrics[agentID]
	if !ok {
		metrics = &AgentMetrics{}
		r.metrics[agentID] = metrics
	}
	
	metrics.TotalTasks++
	if success {
		metrics.CompletedTasks++
	} else {
		metrics.FailedTasks++
	}
	
	// Update success rate
	metrics.SuccessRate = float64(metrics.CompletedTasks) / float64(metrics.TotalTasks)
	
	// Update average response time (exponential moving average)
	alpha := 0.3
	if metrics.AvgResponseTime == 0 {
		metrics.AvgResponseTime = responseTime
	} else {
		metrics.AvgResponseTime = alpha*responseTime + (1-alpha)*metrics.AvgResponseTime
	}
}

// IncrementLoad increments the current load of an agent.
func (r *SmartTaskRouter) IncrementLoad(agentID string) {
	r.metricsMu.Lock()
	defer r.metricsMu.Unlock()
	
	metrics, ok := r.metrics[agentID]
	if !ok {
		metrics = &AgentMetrics{}
		r.metrics[agentID] = metrics
	}
	
	metrics.CurrentLoad++
}

// DecrementLoad decrements the current load of an agent.
func (r *SmartTaskRouter) DecrementLoad(agentID string) {
	r.metricsMu.Lock()
	defer r.metricsMu.Unlock()
	
	metrics, ok := r.metrics[agentID]
	if !ok {
		return
	}
	
	if metrics.CurrentLoad > 0 {
		metrics.CurrentLoad--
	}
}

// GetMetrics returns the metrics for an agent.
func (r *SmartTaskRouter) GetMetrics(agentID string) *AgentMetrics {
	r.metricsMu.RLock()
	defer r.metricsMu.RUnlock()
	
	metrics, ok := r.metrics[agentID]
	if !ok {
		return &AgentMetrics{}
	}
	
	// Return a copy
	return &AgentMetrics{
		TotalTasks:      metrics.TotalTasks,
		CompletedTasks:  metrics.CompletedTasks,
		FailedTasks:     metrics.FailedTasks,
		AvgResponseTime: metrics.AvgResponseTime,
		SuccessRate:     metrics.SuccessRate,
		CurrentLoad:     metrics.CurrentLoad,
	}
}
