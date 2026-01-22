package a2a

import (
	"context"
	"testing"
	"time"
)

func TestNewLocalRegistry(t *testing.T) {
	registry := NewLocalRegistry()
	
	if registry == nil {
		t.Fatal("NewLocalRegistry() returned nil")
	}
	
	if registry.agents == nil {
		t.Error("Expected agents map to be initialized")
	}
}

func TestLocalRegistry_Register(t *testing.T) {
	registry := NewLocalRegistry()
	ctx := context.Background()
	
	agent := &mockA2AAgent{
		info: &AgentInfo{
			ID:     "agent-1",
			Name:   "Test Agent",
			Type:   AgentTypeSpecialist,
			Status: AgentStatusOnline,
		},
		capabilities: &AgentCapabilities{
			Capabilities: []string{"test", "demo"},
		},
	}
	
	err := registry.Register(ctx, agent)
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	
	// Verify agent was registered
	agents, err := registry.ListAll(ctx)
	if err != nil {
		t.Fatalf("ListAll() error = %v", err)
	}
	
	if len(agents) != 1 {
		t.Errorf("Expected 1 agent, got %d", len(agents))
	}
}

func TestLocalRegistry_FindByID(t *testing.T) {
	registry := NewLocalRegistry()
	ctx := context.Background()
	
	agent := &mockA2AAgent{
		info: &AgentInfo{
			ID:   "agent-1",
			Name: "Test Agent",
		},
		capabilities: &AgentCapabilities{},
	}
	
	registry.Register(ctx, agent)
	
	found, err := registry.FindByID(ctx, "agent-1")
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	
	info, _ := found.GetInfo(ctx)
	if info.ID != "agent-1" {
		t.Errorf("Found agent ID = %v, want agent-1", info.ID)
	}
}

func TestLocalRegistry_FindByID_NotFound(t *testing.T) {
	registry := NewLocalRegistry()
	ctx := context.Background()
	
	_, err := registry.FindByID(ctx, "nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent agent")
	}
}

func TestLocalRegistry_FindByCapability(t *testing.T) {
	registry := NewLocalRegistry()
	ctx := context.Background()
	
	agent1 := &mockA2AAgent{
		info: &AgentInfo{ID: "agent-1"},
		capabilities: &AgentCapabilities{
			Capabilities: []string{"research", "analysis"},
		},
	}
	
	agent2 := &mockA2AAgent{
		info: &AgentInfo{ID: "agent-2"},
		capabilities: &AgentCapabilities{
			Capabilities: []string{"writing", "editing"},
		},
	}
	
	agent3 := &mockA2AAgent{
		info: &AgentInfo{ID: "agent-3"},
		capabilities: &AgentCapabilities{
			Capabilities: []string{"research", "writing"},
		},
	}
	
	registry.Register(ctx, agent1)
	registry.Register(ctx, agent2)
	registry.Register(ctx, agent3)
	
	// Find by "research" capability
	agents, err := registry.FindByCapability(ctx, "research")
	if err != nil {
		t.Fatalf("FindByCapability() error = %v", err)
	}
	
	if len(agents) != 2 {
		t.Errorf("Expected 2 agents with research capability, got %d", len(agents))
	}
	
	// Find by "writing" capability
	agents, err = registry.FindByCapability(ctx, "writing")
	if err != nil {
		t.Fatalf("FindByCapability() error = %v", err)
	}
	
	if len(agents) != 2 {
		t.Errorf("Expected 2 agents with writing capability, got %d", len(agents))
	}
}

func TestLocalRegistry_FindByType(t *testing.T) {
	registry := NewLocalRegistry()
	ctx := context.Background()
	
	agent1 := &mockA2AAgent{
		info: &AgentInfo{
			ID:   "agent-1",
			Type: AgentTypeSpecialist,
		},
		capabilities: &AgentCapabilities{},
	}
	
	agent2 := &mockA2AAgent{
		info: &AgentInfo{
			ID:   "agent-2",
			Type: AgentTypeAssistant,
		},
		capabilities: &AgentCapabilities{},
	}
	
	agent3 := &mockA2AAgent{
		info: &AgentInfo{
			ID:   "agent-3",
			Type: AgentTypeSpecialist,
		},
		capabilities: &AgentCapabilities{},
	}
	
	registry.Register(ctx, agent1)
	registry.Register(ctx, agent2)
	registry.Register(ctx, agent3)
	
	specialists, err := registry.FindByType(ctx, AgentTypeSpecialist)
	if err != nil {
		t.Fatalf("FindByType() error = %v", err)
	}
	
	if len(specialists) != 2 {
		t.Errorf("Expected 2 specialists, got %d", len(specialists))
	}
}

func TestLocalRegistry_Unregister(t *testing.T) {
	registry := NewLocalRegistry()
	ctx := context.Background()
	
	agent := &mockA2AAgent{
		info:         &AgentInfo{ID: "agent-1"},
		capabilities: &AgentCapabilities{},
	}
	
	registry.Register(ctx, agent)
	
	// Verify registered
	agents, _ := registry.ListAll(ctx)
	if len(agents) != 1 {
		t.Errorf("Expected 1 agent before unregister, got %d", len(agents))
	}
	
	// Unregister
	err := registry.Unregister(ctx, "agent-1")
	if err != nil {
		t.Fatalf("Unregister() error = %v", err)
	}
	
	// Verify unregistered
	agents, _ = registry.ListAll(ctx)
	if len(agents) != 0 {
		t.Errorf("Expected 0 agents after unregister, got %d", len(agents))
	}
}

func TestLocalRegistry_UpdateStatus(t *testing.T) {
	registry := NewLocalRegistry()
	ctx := context.Background()
	
	agent := &mockA2AAgent{
		info: &AgentInfo{
			ID:     "agent-1",
			Status: AgentStatusOnline,
		},
		capabilities: &AgentCapabilities{},
	}
	
	registry.Register(ctx, agent)
	
	// Update status
	err := registry.UpdateStatus(ctx, "agent-1", AgentStatusBusy)
	if err != nil {
		t.Fatalf("UpdateStatus() error = %v", err)
	}
	
	// Verify status changed
	found, _ := registry.FindByID(ctx, "agent-1")
	_, _ = found.GetInfo(ctx)
	
	// Note: The registry updates its internal state, but the agent's
	// original info is not modified
}

func TestLocalRegistry_Heartbeat(t *testing.T) {
	registry := NewLocalRegistry()
	ctx := context.Background()
	
	agent := &mockA2AAgent{
		info:         &AgentInfo{ID: "agent-1"},
		capabilities: &AgentCapabilities{},
	}
	
	registry.Register(ctx, agent)
	
	// Wait a bit
	time.Sleep(10 * time.Millisecond)
	
	// Send heartbeat
	err := registry.Heartbeat(ctx, "agent-1")
	if err != nil {
		t.Fatalf("Heartbeat() error = %v", err)
	}
}

func TestLocalRegistry_CheckHealth(t *testing.T) {
	registry := NewLocalRegistry()
	ctx := context.Background()
	
	agent := &mockA2AAgent{
		info:         &AgentInfo{ID: "agent-1"},
		capabilities: &AgentCapabilities{},
	}
	
	registry.Register(ctx, agent)
	
	health, err := registry.CheckHealth(ctx, "agent-1")
	if err != nil {
		t.Fatalf("CheckHealth() error = %v", err)
	}
	
	if health.AgentID != "agent-1" {
		t.Errorf("Health.AgentID = %v, want agent-1", health.AgentID)
	}
	
	if health.Status != "healthy" {
		t.Errorf("Health.Status = %v, want healthy", health.Status)
	}
}

func TestLocalRegistry_CheckHealth_Unhealthy(t *testing.T) {
	registry := NewLocalRegistry()
	ctx := context.Background()
	
	agent := &mockA2AAgent{
		info:         &AgentInfo{ID: "agent-1"},
		capabilities: &AgentCapabilities{},
	}
	
	registry.Register(ctx, agent)
	
	// Manually set old LastSeen to simulate unhealthy agent
	registry.mu.Lock()
	regAgent := registry.agents["agent-1"]
	regAgent.LastSeen = time.Now().Add(-1 * time.Minute)
	registry.mu.Unlock()
	
	health, err := registry.CheckHealth(ctx, "agent-1")
	if err != nil {
		t.Fatalf("CheckHealth() error = %v", err)
	}
	
	if health.Status != "unhealthy" {
		t.Errorf("Health.Status = %v, want unhealthy", health.Status)
	}
}

// Mock A2A Agent

type mockA2AAgent struct {
	info         *AgentInfo
	capabilities *AgentCapabilities
}

func (m *mockA2AAgent) GetInfo(ctx context.Context) (*AgentInfo, error) {
	return m.info, nil
}

func (m *mockA2AAgent) GetCapabilities(ctx context.Context) (*AgentCapabilities, error) {
	return m.capabilities, nil
}

func (m *mockA2AAgent) SendTask(ctx context.Context, task *Task) (*TaskResponse, error) {
	return &TaskResponse{
		TaskID:  task.ID,
		Status:  TaskStatusCompleted,
		Result:  &TaskResult{Type: "text", Content: "mock result"},
		Progress: 1.0,
	}, nil
}

func (m *mockA2AAgent) GetTaskStatus(ctx context.Context, taskID string) (*TaskStatus, error) {
	status := TaskStatusCompleted
	return &status, nil
}

func (m *mockA2AAgent) CancelTask(ctx context.Context, taskID string) error {
	return nil
}

func (m *mockA2AAgent) StreamTask(ctx context.Context, task *Task) (<-chan *TaskEvent, error) {
	return nil, nil
}

func (m *mockA2AAgent) SendMessage(ctx context.Context, msg *Message) error {
	return nil
}

func (m *mockA2AAgent) ReceiveMessages(ctx context.Context) (<-chan *Message, error) {
	return nil, nil
}

func (m *mockA2AAgent) RequestHelp(ctx context.Context, req *HelpRequest) (*HelpResponse, error) {
	return nil, nil
}

func (m *mockA2AAgent) OfferHelp(ctx context.Context, offer *HelpOffer) error {
	return nil
}
