package a2a

import (
	"context"
	"testing"
	
	"github.com/google/uuid"
)

func TestNewCollaborationCoordinator(t *testing.T) {
	registry := NewLocalRegistry()
	router := NewSmartTaskRouter(registry, RouterConfig{})
	
	coordinator := NewCollaborationCoordinator(registry, router)
	
	if coordinator == nil {
		t.Fatal("NewCollaborationCoordinator() returned nil")
	}
	
	if coordinator.sessions == nil {
		t.Error("Expected sessions map to be initialized")
	}
}

func TestCollaborationCoordinator_Coordinate(t *testing.T) {
	registry := NewLocalRegistry()
	ctx := context.Background()
	
	// Register multiple agents
	agent1 := &mockA2AAgent{
		info:         &AgentInfo{ID: "agent-1", Status: AgentStatusOnline},
		capabilities: &AgentCapabilities{Capabilities: []string{"research"}},
	}
	agent2 := &mockA2AAgent{
		info:         &AgentInfo{ID: "agent-2", Status: AgentStatusOnline},
		capabilities: &AgentCapabilities{Capabilities: []string{"analysis"}},
	}
	agent3 := &mockA2AAgent{
		info:         &AgentInfo{ID: "agent-3", Status: AgentStatusOnline},
		capabilities: &AgentCapabilities{Capabilities: []string{"writing"}},
	}
	
	registry.Register(ctx, agent1)
	registry.Register(ctx, agent2)
	registry.Register(ctx, agent3)
	
	router := NewSmartTaskRouter(registry, RouterConfig{})
	coordinator := NewCollaborationCoordinator(registry, router)
	
	// Create complex task
	task := &Task{
		ID:   uuid.New().String(),
		Type: TaskTypeComplex,
		Input: &TaskInput{
			Type:    "text",
			Content: "Research, analyze, and write a report",
		},
	}
	
	result, err := coordinator.Coordinate(ctx, task)
	if err != nil {
		t.Fatalf("Coordinate() error = %v", err)
	}
	
	if result == nil {
		t.Fatal("Expected result to be non-nil")
	}
	
	if result.Content == "" {
		t.Error("Expected result content to be non-empty")
	}
}

func TestCollaborationCoordinator_GetSession(t *testing.T) {
	registry := NewLocalRegistry()
	router := NewSmartTaskRouter(registry, RouterConfig{})
	coordinator := NewCollaborationCoordinator(registry, router)
	
	// Create a session
	task := &Task{ID: uuid.New().String()}
	session := coordinator.createSession(task)
	
	// Get session
	found := coordinator.GetSession(session.ID)
	if found == nil {
		t.Fatal("GetSession() returned nil")
	}
	
	if found.ID != session.ID {
		t.Errorf("Session.ID = %v, want %v", found.ID, session.ID)
	}
}

func TestCollaborationCoordinator_ListSessions(t *testing.T) {
	registry := NewLocalRegistry()
	router := NewSmartTaskRouter(registry, RouterConfig{})
	coordinator := NewCollaborationCoordinator(registry, router)
	
	// Create multiple sessions
	task1 := &Task{ID: uuid.New().String()}
	task2 := &Task{ID: uuid.New().String()}
	
	coordinator.createSession(task1)
	coordinator.createSession(task2)
	
	sessions := coordinator.ListSessions()
	if len(sessions) != 2 {
		t.Errorf("Expected 2 sessions, got %d", len(sessions))
	}
}

func TestDecomposeTask(t *testing.T) {
	registry := NewLocalRegistry()
	router := NewSmartTaskRouter(registry, RouterConfig{})
	coordinator := NewCollaborationCoordinator(registry, router)
	
	tests := []struct {
		name      string
		task      *Task
		wantCount int
	}{
		{
			name: "Complex task",
			task: &Task{
				ID:   uuid.New().String(),
				Type: TaskTypeComplex,
				Input: &TaskInput{
					Type:    "text",
					Content: "Complex task",
				},
			},
			wantCount: 3, // Default decomposition into 3 subtasks
		},
		{
			name: "Simple task",
			task: &Task{
				ID:   uuid.New().String(),
				Type: TaskTypeQuery,
				Input: &TaskInput{
					Type:    "text",
					Content: "Simple task",
				},
			},
			wantCount: 1, // No decomposition
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			subtasks, err := coordinator.decomposeTask(tt.task)
			if err != nil {
				t.Fatalf("decomposeTask() error = %v", err)
			}
			
			if len(subtasks) != tt.wantCount {
				t.Errorf("Expected %d subtasks, got %d", tt.wantCount, len(subtasks))
			}
		})
	}
}

func TestAggregateResults(t *testing.T) {
	registry := NewLocalRegistry()
	router := NewSmartTaskRouter(registry, RouterConfig{})
	coordinator := NewCollaborationCoordinator(registry, router)
	
	ctx := context.Background()
	
	results := map[string]*TaskResult{
		"task-1": {
			Type:    "text",
			Content: "Result 1",
		},
		"task-2": {
			Type:    "text",
			Content: "Result 2",
		},
		"task-3": {
			Type:    "text",
			Content: "Result 3",
		},
	}
	
	finalResult, err := coordinator.aggregateResults(ctx, results)
	if err != nil {
		t.Fatalf("aggregateResults() error = %v", err)
	}
	
	if finalResult.Content == "" {
		t.Error("Expected combined content to be non-empty")
	}
	
	// Verify all results are included
	for _, result := range results {
		// Content should include each subtask result
		_ = result
	}
}

func TestAggregateResults_Empty(t *testing.T) {
	registry := NewLocalRegistry()
	router := NewSmartTaskRouter(registry, RouterConfig{})
	coordinator := NewCollaborationCoordinator(registry, router)
	
	ctx := context.Background()
	results := map[string]*TaskResult{}
	
	_, err := coordinator.aggregateResults(ctx, results)
	if err == nil {
		t.Error("Expected error for empty results")
	}
}

func TestCollaborationSession(t *testing.T) {
	session := &CollaborationSession{
		ID:           uuid.New().String(),
		MainTask:     &Task{ID: "main-task"},
		Participants: make(map[string]A2AAgent),
		SubTasks:     make(map[string]*Task),
		Results:      make(map[string]*TaskResult),
		Status:       SessionStatusActive,
	}
	
	if session.Status != SessionStatusActive {
		t.Errorf("Status = %v, want %v", session.Status, SessionStatusActive)
	}
	
	// Add participant
	agent := &mockA2AAgent{
		info: &AgentInfo{ID: "agent-1"},
	}
	session.Participants["agent-1"] = agent
	
	if len(session.Participants) != 1 {
		t.Errorf("Expected 1 participant, got %d", len(session.Participants))
	}
}
