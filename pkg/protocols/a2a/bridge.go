package a2a

import (
	"context"
	"fmt"
	"sync"
	"time"
	
	"github.com/google/uuid"
)

// A2AAgentBridge bridges existing agents to the A2A protocol.
// This allows existing LangChain-Go agents to participate in A2A communication.
type A2AAgentBridge struct {
	// Agent interface adapter (to be defined based on actual agent interface)
	agent AgentAdapter
	
	// A2A information
	info *AgentInfo
	caps *AgentCapabilities
	
	// Task tracking
	tasks   map[string]*TaskState
	tasksMu sync.RWMutex
}

// AgentAdapter is an interface that existing agents should implement
// or be wrapped to implement.
type AgentAdapter interface {
	// Execute processes input and returns output
	Execute(ctx context.Context, input string) (string, error)
	
	// GetName returns the agent name
	GetName() string
	
	// GetDescription returns the agent description (optional)
	GetDescription() string
}

// TaskState represents the state of a task being processed.
type TaskState struct {
	Task      *Task
	Status    TaskStatus
	Result    *TaskResult
	Error     *TaskError
	Progress  float64
	StartedAt time.Time
	UpdatedAt time.Time
}

// BridgeConfig configures an A2A agent bridge.
type BridgeConfig struct {
	Info         *AgentInfo
	Capabilities *AgentCapabilities
}

// NewA2AAgentBridge creates a new A2A agent bridge.
func NewA2AAgentBridge(agent AgentAdapter, config *BridgeConfig) *A2AAgentBridge {
	// Set defaults
	if config.Info == nil {
		config.Info = &AgentInfo{
			ID:     uuid.New().String(),
			Name:   agent.GetName(),
			Type:   AgentTypeAssistant,
			Status: AgentStatusOnline,
		}
	}
	
	if config.Capabilities == nil {
		config.Capabilities = &AgentCapabilities{
			Capabilities:       []string{"general"},
			MessageFormats:     []string{"text"},
			Languages:          []string{"zh", "en"},
			MaxConcurrentTasks: 5,
			AvgResponseTime:    2 * time.Second,
		}
	}
	
	return &A2AAgentBridge{
		agent: agent,
		info:  config.Info,
		caps:  config.Capabilities,
		tasks: make(map[string]*TaskState),
	}
}

// GetInfo returns agent information.
func (b *A2AAgentBridge) GetInfo(ctx context.Context) (*AgentInfo, error) {
	return b.info, nil
}

// GetCapabilities returns agent capabilities.
func (b *A2AAgentBridge) GetCapabilities(ctx context.Context) (*AgentCapabilities, error) {
	return b.caps, nil
}

// SendTask sends a task to the agent.
func (b *A2AAgentBridge) SendTask(ctx context.Context, task *Task) (*TaskResponse, error) {
	// Create task state
	b.tasksMu.Lock()
	b.tasks[task.ID] = &TaskState{
		Task:      task,
		Status:    TaskStatusRunning,
		StartedAt: time.Now(),
		UpdatedAt: time.Now(),
		Progress:  0.0,
	}
	b.tasksMu.Unlock()
	
	// Convert task to agent input
	input := b.taskToAgentInput(task)
	
	// Execute task
	output, err := b.agent.Execute(ctx, input)
	
	// Update task state
	b.tasksMu.Lock()
	defer b.tasksMu.Unlock()
	
	state := b.tasks[task.ID]
	state.UpdatedAt = time.Now()
	
	if err != nil {
		state.Status = TaskStatusFailed
		state.Error = &TaskError{
			Code:    "EXECUTION_ERROR",
			Message: err.Error(),
		}
		state.Progress = 0.0
		
		return &TaskResponse{
			TaskID:   task.ID,
			Status:   TaskStatusFailed,
			Error:    state.Error,
			Progress: 0.0,
			Metadata: make(map[string]any),
		}, nil
	}
	
	// Task completed successfully
	state.Status = TaskStatusCompleted
	state.Result = &TaskResult{
		Type:       "text",
		Content:    output,
		Confidence: 0.9,
	}
	state.Progress = 1.0
	
	return &TaskResponse{
		TaskID:   task.ID,
		Status:   TaskStatusCompleted,
		Result:   state.Result,
		Progress: 1.0,
		Metadata: make(map[string]any),
	}, nil
}

// GetTaskStatus returns the status of a task.
func (b *A2AAgentBridge) GetTaskStatus(ctx context.Context, taskID string) (*TaskStatus, error) {
	b.tasksMu.RLock()
	defer b.tasksMu.RUnlock()
	
	state, ok := b.tasks[taskID]
	if !ok {
		return nil, fmt.Errorf("task not found: %s", taskID)
	}
	
	status := TaskStatus(state.Status)
	return &status, nil
}

// CancelTask cancels a task.
func (b *A2AAgentBridge) CancelTask(ctx context.Context, taskID string) error {
	b.tasksMu.Lock()
	defer b.tasksMu.Unlock()
	
	state, ok := b.tasks[taskID]
	if !ok {
		return fmt.Errorf("task not found: %s", taskID)
	}
	
	if state.Status != TaskStatusRunning {
		return fmt.Errorf("task not running: %s", taskID)
	}
	
	state.Status = TaskStatusCancelled
	state.UpdatedAt = time.Now()
	
	return nil
}

// StreamTask streams task execution events (not implemented in basic version).
func (b *A2AAgentBridge) StreamTask(ctx context.Context, task *Task) (<-chan *TaskEvent, error) {
	return nil, fmt.Errorf("streaming not supported")
}

// SendMessage sends a message (not implemented in basic version).
func (b *A2AAgentBridge) SendMessage(ctx context.Context, msg *Message) error {
	return fmt.Errorf("messaging not implemented")
}

// ReceiveMessages receives messages (not implemented in basic version).
func (b *A2AAgentBridge) ReceiveMessages(ctx context.Context) (<-chan *Message, error) {
	return nil, fmt.Errorf("messaging not implemented")
}

// RequestHelp requests help from other agents (not implemented in basic version).
func (b *A2AAgentBridge) RequestHelp(ctx context.Context, req *HelpRequest) (*HelpResponse, error) {
	return nil, fmt.Errorf("help request not implemented")
}

// OfferHelp offers help to other agents (not implemented in basic version).
func (b *A2AAgentBridge) OfferHelp(ctx context.Context, offer *HelpOffer) error {
	return fmt.Errorf("help offer not implemented")
}

// taskToAgentInput converts an A2A task to agent input.
func (b *A2AAgentBridge) taskToAgentInput(task *Task) string {
	if task.Input == nil {
		return ""
	}
	
	switch task.Input.Type {
	case "text":
		return task.Input.Content
	case "data":
		// Convert data to string representation
		return fmt.Sprintf("%v", task.Input.Data)
	default:
		return task.Input.Content
	}
}
