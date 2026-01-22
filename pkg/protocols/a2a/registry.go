package a2a

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// AgentRegistry manages agent registration and discovery.
type AgentRegistry interface {
	// Registration
	Register(ctx context.Context, agent A2AAgent) error
	Unregister(ctx context.Context, agentID string) error
	UpdateStatus(ctx context.Context, agentID string, status AgentStatus) error
	
	// Discovery
	FindByID(ctx context.Context, agentID string) (A2AAgent, error)
	FindByCapability(ctx context.Context, capability string) ([]A2AAgent, error)
	FindByType(ctx context.Context, agentType AgentType) ([]A2AAgent, error)
	ListAll(ctx context.Context) ([]A2AAgent, error)
	
	// Health
	Heartbeat(ctx context.Context, agentID string) error
	CheckHealth(ctx context.Context, agentID string) (*HealthStatus, error)
}

// HealthStatus represents agent health status.
type HealthStatus struct {
	AgentID       string        `json:"agentId"`
	Status        string        `json:"status"` // "healthy", "unhealthy"
	Latency       time.Duration `json:"latency"`
	LastHeartbeat time.Time     `json:"lastHeartbeat"`
	Uptime        time.Duration `json:"uptime"`
}

// RegisteredAgent represents a registered agent.
type RegisteredAgent struct {
	Agent        A2AAgent
	Info         *AgentInfo
	Capabilities *AgentCapabilities
	RegisteredAt time.Time
	LastSeen     time.Time
	Uptime       time.Duration
}

// LocalAgentRegistry is an in-memory agent registry.
// It's suitable for development, testing, and single-node deployments.
type LocalAgentRegistry struct {
	agents map[string]*RegisteredAgent
	mu     sync.RWMutex
}

// NewLocalRegistry creates a new local agent registry.
func NewLocalRegistry() *LocalAgentRegistry {
	return &LocalAgentRegistry{
		agents: make(map[string]*RegisteredAgent),
	}
}

// Register registers an agent.
func (r *LocalAgentRegistry) Register(ctx context.Context, agent A2AAgent) error {
	info, err := agent.GetInfo(ctx)
	if err != nil {
		return fmt.Errorf("get agent info: %w", err)
	}
	
	capabilities, err := agent.GetCapabilities(ctx)
	if err != nil {
		return fmt.Errorf("get agent capabilities: %w", err)
	}
	
	r.mu.Lock()
	defer r.mu.Unlock()
	
	now := time.Now()
	r.agents[info.ID] = &RegisteredAgent{
		Agent:        agent,
		Info:         info,
		Capabilities: capabilities,
		RegisteredAt: now,
		LastSeen:     now,
	}
	
	return nil
}

// Unregister unregisters an agent.
func (r *LocalAgentRegistry) Unregister(ctx context.Context, agentID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	delete(r.agents, agentID)
	return nil
}

// UpdateStatus updates an agent's status.
func (r *LocalAgentRegistry) UpdateStatus(ctx context.Context, agentID string, status AgentStatus) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	agent, ok := r.agents[agentID]
	if !ok {
		return fmt.Errorf("agent not found: %s", agentID)
	}
	
	agent.Info.Status = status
	agent.LastSeen = time.Now()
	
	return nil
}

// FindByID finds an agent by ID.
func (r *LocalAgentRegistry) FindByID(ctx context.Context, agentID string) (A2AAgent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	agent, ok := r.agents[agentID]
	if !ok {
		return nil, fmt.Errorf("agent not found: %s", agentID)
	}
	
	return agent.Agent, nil
}

// FindByCapability finds agents by capability.
func (r *LocalAgentRegistry) FindByCapability(ctx context.Context, capability string) ([]A2AAgent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	var result []A2AAgent
	for _, agent := range r.agents {
		// Check if agent has the capability
		for _, cap := range agent.Capabilities.Capabilities {
			if cap == capability {
				result = append(result, agent.Agent)
				break
			}
		}
	}
	
	return result, nil
}

// FindByType finds agents by type.
func (r *LocalAgentRegistry) FindByType(ctx context.Context, agentType AgentType) ([]A2AAgent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	var result []A2AAgent
	for _, agent := range r.agents {
		if agent.Info.Type == agentType {
			result = append(result, agent.Agent)
		}
	}
	
	return result, nil
}

// ListAll lists all registered agents.
func (r *LocalAgentRegistry) ListAll(ctx context.Context) ([]A2AAgent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	result := make([]A2AAgent, 0, len(r.agents))
	for _, agent := range r.agents {
		result = append(result, agent.Agent)
	}
	
	return result, nil
}

// Heartbeat updates the last seen time of an agent.
func (r *LocalAgentRegistry) Heartbeat(ctx context.Context, agentID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	agent, ok := r.agents[agentID]
	if !ok {
		return fmt.Errorf("agent not found: %s", agentID)
	}
	
	agent.LastSeen = time.Now()
	agent.Uptime = time.Since(agent.RegisteredAt)
	
	return nil
}

// CheckHealth checks the health of an agent.
func (r *LocalAgentRegistry) CheckHealth(ctx context.Context, agentID string) (*HealthStatus, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	agent, ok := r.agents[agentID]
	if !ok {
		return nil, fmt.Errorf("agent not found: %s", agentID)
	}
	
	// Calculate health status
	timeSinceLastSeen := time.Since(agent.LastSeen)
	status := "healthy"
	if timeSinceLastSeen > 30*time.Second {
		status = "unhealthy"
	}
	
	return &HealthStatus{
		AgentID:       agentID,
		Status:        status,
		Latency:       0, // Not applicable for local registry
		LastHeartbeat: agent.LastSeen,
		Uptime:        agent.Uptime,
	}, nil
}
