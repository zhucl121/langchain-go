package a2a

import (
	"context"
	"testing"
	"time"
	
	"github.com/google/uuid"
)

func TestNewA2AAgentBridge(t *testing.T) {
	adapter := &mockAgentAdapter{
		name: "Test Agent",
	}
	
	config := &BridgeConfig{
		Info: &AgentInfo{
			ID:   "agent-1",
			Name: "Test Agent",
		},
		Capabilities: &AgentCapabilities{
			Capabilities: []string{"test"},
		},
	}
	
	bridge := NewA2AAgentBridge(adapter, config)
	
	if bridge == nil {
		t.Fatal("NewA2AAgentBridge() returned nil")
	}
	
	if bridge.info.ID != "agent-1" {
		t.Errorf("Info.ID = %v, want agent-1", bridge.info.ID)
	}
}

func TestA2AAgentBridge_GetInfo(t *testing.T) {
	adapter := &mockAgentAdapter{name: "Test"}
	config := &BridgeConfig{
		Info: &AgentInfo{
			ID:   "agent-1",
			Name: "Test Agent",
		},
	}
	
	bridge := NewA2AAgentBridge(adapter, config)
	ctx := context.Background()
	
	info, err := bridge.GetInfo(ctx)
	if err != nil {
		t.Fatalf("GetInfo() error = %v", err)
	}
	
	if info.ID != "agent-1" {
		t.Errorf("Info.ID = %v, want agent-1", info.ID)
	}
}

func TestA2AAgentBridge_SendTask(t *testing.T) {
	adapter := &mockAgentAdapter{
		name: "Test Agent",
		executeFunc: func(ctx context.Context, input string) (string, error) {
			return "Result: " + input, nil
		},
	}
	
	bridge := NewA2AAgentBridge(adapter, &BridgeConfig{
		Info: &AgentInfo{ID: "agent-1"},
	})
	
	ctx := context.Background()
	task := &Task{
		ID:   uuid.New().String(),
		Type: TaskTypeQuery,
		Input: &TaskInput{
			Type:    "text",
			Content: "test input",
		},
	}
	
	response, err := bridge.SendTask(ctx, task)
	if err != nil {
		t.Fatalf("SendTask() error = %v", err)
	}
	
	if response.Status != TaskStatusCompleted {
		t.Errorf("Status = %v, want %v", response.Status, TaskStatusCompleted)
	}
	
	if response.Result.Content != "Result: test input" {
		t.Errorf("Result.Content = %v, want Result: test input", response.Result.Content)
	}
	
	if response.Progress != 1.0 {
		t.Errorf("Progress = %v, want 1.0", response.Progress)
	}
}

func TestA2AAgentBridge_SendTask_Error(t *testing.T) {
	adapter := &mockAgentAdapter{
		name: "Test Agent",
		executeFunc: func(ctx context.Context, input string) (string, error) {
			return "", context.DeadlineExceeded
		},
	}
	
	bridge := NewA2AAgentBridge(adapter, &BridgeConfig{
		Info: &AgentInfo{ID: "agent-1"},
	})
	
	ctx := context.Background()
	task := &Task{
		ID:    uuid.New().String(),
		Type:  TaskTypeQuery,
		Input: &TaskInput{Type: "text", Content: "test"},
	}
	
	response, err := bridge.SendTask(ctx, task)
	if err != nil {
		t.Fatalf("SendTask() should not return error, got %v", err)
	}
	
	if response.Status != TaskStatusFailed {
		t.Errorf("Status = %v, want %v", response.Status, TaskStatusFailed)
	}
	
	if response.Error == nil {
		t.Error("Expected Error to be set")
	}
}

func TestA2AAgentBridge_GetTaskStatus(t *testing.T) {
	adapter := &mockAgentAdapter{name: "Test"}
	bridge := NewA2AAgentBridge(adapter, &BridgeConfig{
		Info: &AgentInfo{ID: "agent-1"},
	})
	
	ctx := context.Background()
	taskID := uuid.New().String()
	
	// Add task state
	bridge.tasksMu.Lock()
	bridge.tasks[taskID] = &TaskState{
		Task:   &Task{ID: taskID},
		Status: TaskStatusRunning,
	}
	bridge.tasksMu.Unlock()
	
	status, err := bridge.GetTaskStatus(ctx, taskID)
	if err != nil {
		t.Fatalf("GetTaskStatus() error = %v", err)
	}
	
	if *status != TaskStatusRunning {
		t.Errorf("Status = %v, want %v", *status, TaskStatusRunning)
	}
}

func TestA2AAgentBridge_CancelTask(t *testing.T) {
	adapter := &mockAgentAdapter{name: "Test"}
	bridge := NewA2AAgentBridge(adapter, &BridgeConfig{
		Info: &AgentInfo{ID: "agent-1"},
	})
	
	ctx := context.Background()
	taskID := uuid.New().String()
	
	// Add running task
	bridge.tasksMu.Lock()
	bridge.tasks[taskID] = &TaskState{
		Task:      &Task{ID: taskID},
		Status:    TaskStatusRunning,
		StartedAt: time.Now(),
	}
	bridge.tasksMu.Unlock()
	
	// Cancel task
	err := bridge.CancelTask(ctx, taskID)
	if err != nil {
		t.Fatalf("CancelTask() error = %v", err)
	}
	
	// Verify cancelled
	bridge.tasksMu.RLock()
	state := bridge.tasks[taskID]
	bridge.tasksMu.RUnlock()
	
	if state.Status != TaskStatusCancelled {
		t.Errorf("Status = %v, want %v", state.Status, TaskStatusCancelled)
	}
}

func TestA2AAgentBridge_DefaultConfig(t *testing.T) {
	adapter := &mockAgentAdapter{name: "Test Agent"}
	
	// Create bridge without config
	bridge := NewA2AAgentBridge(adapter, &BridgeConfig{})
	
	if bridge.info == nil {
		t.Fatal("Expected info to be set with defaults")
	}
	
	if bridge.info.Name != "Test Agent" {
		t.Errorf("Name = %v, want Test Agent", bridge.info.Name)
	}
	
	if bridge.caps == nil {
		t.Fatal("Expected capabilities to be set with defaults")
	}
	
	if len(bridge.caps.Capabilities) == 0 {
		t.Error("Expected default capabilities to be set")
	}
}

func TestTaskToAgentInput(t *testing.T) {
	adapter := &mockAgentAdapter{name: "Test"}
	bridge := NewA2AAgentBridge(adapter, &BridgeConfig{
		Info: &AgentInfo{ID: "agent-1"},
	})
	
	tests := []struct {
		name  string
		task  *Task
		want  string
	}{
		{
			name: "Text input",
			task: &Task{
				Input: &TaskInput{
					Type:    "text",
					Content: "Hello world",
				},
			},
			want: "Hello world",
		},
		{
			name: "Data input",
			task: &Task{
				Input: &TaskInput{
					Type: "data",
					Data: map[string]any{"key": "value"},
				},
			},
			want: "map[key:value]",
		},
		{
			name: "Nil input",
			task: &Task{},
			want: "",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := bridge.taskToAgentInput(tt.task)
			if got != tt.want {
				t.Errorf("taskToAgentInput() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Mock Agent Adapter

type mockAgentAdapter struct {
	name        string
	executeFunc func(ctx context.Context, input string) (string, error)
}

func (m *mockAgentAdapter) Execute(ctx context.Context, input string) (string, error) {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, input)
	}
	return "mock result", nil
}

func (m *mockAgentAdapter) GetName() string {
	return m.name
}

func (m *mockAgentAdapter) GetDescription() string {
	return "Mock agent adapter"
}
