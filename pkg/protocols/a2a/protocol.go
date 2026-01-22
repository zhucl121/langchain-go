// Package a2a implements the Agent-to-Agent (A2A) protocol for standardized agent communication.
//
// A2A protocol enables distributed multi-agent systems with:
//   - Agent registration and discovery
//   - Task routing and load balancing
//   - Collaboration coordination
//   - Cross-platform agent communication
//
// Example usage:
//
//	// Bridge existing agent to A2A
//	a2aAgent := a2a.NewA2AAgentBridge(myAgent, &a2a.BridgeConfig{
//	    Info: &a2a.AgentInfo{
//	        ID:   "agent-1",
//	        Name: "Research Agent",
//	    },
//	    Capabilities: &a2a.AgentCapabilities{
//	        Capabilities: []string{"research", "search"},
//	    },
//	})
//	
//	// Register to registry
//	registry := a2a.NewConsulRegistry(consulConfig)
//	registry.Register(ctx, a2aAgent)
package a2a

import (
	"context"
	"time"
)

// A2AAgent represents an agent that implements the A2A protocol.
type A2AAgent interface {
	// Agent Information
	GetInfo(ctx context.Context) (*AgentInfo, error)
	GetCapabilities(ctx context.Context) (*AgentCapabilities, error)
	
	// Task Processing
	SendTask(ctx context.Context, task *Task) (*TaskResponse, error)
	GetTaskStatus(ctx context.Context, taskID string) (*TaskStatus, error)
	CancelTask(ctx context.Context, taskID string) error
	
	// Streaming
	StreamTask(ctx context.Context, task *Task) (<-chan *TaskEvent, error)
	
	// Messaging
	SendMessage(ctx context.Context, msg *Message) error
	ReceiveMessages(ctx context.Context) (<-chan *Message, error)
	
	// Collaboration
	RequestHelp(ctx context.Context, req *HelpRequest) (*HelpResponse, error)
	OfferHelp(ctx context.Context, offer *HelpOffer) error
}

// AgentInfo represents agent information.
type AgentInfo struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Version     string         `json:"version"`
	Vendor      string         `json:"vendor"`
	Type        AgentType      `json:"type"`
	Status      AgentStatus    `json:"status"`
	Metadata    map[string]any `json:"metadata"`
}

// AgentType represents the type of agent.
type AgentType string

const (
	AgentTypeAssistant   AgentType = "assistant"   // General assistant
	AgentTypeSpecialist  AgentType = "specialist"  // Specialist agent
	AgentTypeCoordinator AgentType = "coordinator" // Coordinator agent
)

// AgentStatus represents the status of an agent.
type AgentStatus string

const (
	AgentStatusOnline      AgentStatus = "online"      // Online and available
	AgentStatusBusy        AgentStatus = "busy"        // Busy processing tasks
	AgentStatusOffline     AgentStatus = "offline"     // Offline
	AgentStatusMaintenance AgentStatus = "maintenance" // Under maintenance
)

// AgentCapabilities represents agent capabilities.
type AgentCapabilities struct {
	Capabilities       []string      `json:"capabilities"`       // Capability tags
	Tools              []string      `json:"tools"`              // Available tools
	MessageFormats     []string      `json:"messageFormats"`     // Supported message formats
	Languages          []string      `json:"languages"`          // Supported languages
	MaxConcurrentTasks int           `json:"maxConcurrentTasks"` // Max concurrent tasks
	AvgResponseTime    time.Duration `json:"avgResponseTime"`    // Average response time
}

// Task represents a task to be executed by an agent.
type Task struct {
	ID           string            `json:"id"`
	Type         TaskType          `json:"type"`
	Priority     Priority          `json:"priority"`
	Input        *TaskInput        `json:"input"`
	Context      map[string]any    `json:"context"`
	Requirements *TaskRequirements `json:"requirements,omitempty"`
	Deadline     *time.Time        `json:"deadline,omitempty"`
	Metadata     map[string]any    `json:"metadata"`
}

// TaskType represents the type of task.
type TaskType string

const (
	TaskTypeQuery    TaskType = "query"    // Query task
	TaskTypeAnalyze  TaskType = "analyze"  // Analysis task
	TaskTypeGenerate TaskType = "generate" // Generation task
	TaskTypeExecute  TaskType = "execute"  // Execution task
	TaskTypeComplex  TaskType = "complex"  // Complex task (needs decomposition)
)

// Priority represents task priority.
type Priority string

const (
	PriorityLow    Priority = "low"    // Low priority
	PriorityMedium Priority = "medium" // Medium priority
	PriorityHigh   Priority = "high"   // High priority
	PriorityUrgent Priority = "urgent" // Urgent priority
)

// TaskInput represents task input.
type TaskInput struct {
	Type    string         `json:"type"`              // Input type
	Content string         `json:"content,omitempty"` // Content (for text type)
	Data    map[string]any `json:"data,omitempty"`    // Data (for data type)
	Files   []FileRef      `json:"files,omitempty"`   // Files (for file type)
}

// FileRef represents a file reference.
type FileRef struct {
	URI      string `json:"uri"`      // File URI
	Name     string `json:"name"`     // File name
	MimeType string `json:"mimeType"` // MIME type
	Size     int64  `json:"size"`     // File size in bytes
}

// TaskRequirements represents task requirements.
type TaskRequirements struct {
	Quality       string         `json:"quality,omitempty"`       // Quality level
	MaxDuration   int            `json:"maxDuration,omitempty"`   // Max duration in seconds
	RequiredTools []string       `json:"requiredTools,omitempty"` // Required tools
	Constraints   map[string]any `json:"constraints,omitempty"`   // Additional constraints
}

// TaskResponse represents the response to a task.
type TaskResponse struct {
	TaskID        string         `json:"taskId"`
	Status        TaskStatus     `json:"status"`
	Result        *TaskResult    `json:"result,omitempty"`
	Error         *TaskError     `json:"error,omitempty"`
	Progress      float64        `json:"progress"` // 0-1
	EstimatedTime *time.Duration `json:"estimatedTime,omitempty"`
	Metadata      map[string]any `json:"metadata"`
}

// TaskStatus represents task status.
type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"   // Pending
	TaskStatusRunning   TaskStatus = "running"   // Running
	TaskStatusCompleted TaskStatus = "completed" // Completed
	TaskStatusFailed    TaskStatus = "failed"    // Failed
	TaskStatusCancelled TaskStatus = "cancelled" // Cancelled
)

// TaskResult represents task result.
type TaskResult struct {
	Type       string         `json:"type"`                 // Result type
	Content    string         `json:"content,omitempty"`    // Content
	Data       map[string]any `json:"data,omitempty"`       // Data
	Files      []FileRef      `json:"files,omitempty"`      // Files
	Artifacts  []Artifact     `json:"artifacts,omitempty"`  // Artifacts
	Confidence float64        `json:"confidence,omitempty"` // Confidence score (0-1)
}

// Artifact represents a task artifact.
type Artifact struct {
	ID       string         `json:"id"`
	Type     string         `json:"type"`
	Name     string         `json:"name"`
	Content  string         `json:"content,omitempty"`
	URI      string         `json:"uri,omitempty"`
	Metadata map[string]any `json:"metadata"`
}

// TaskError represents a task error.
type TaskError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// TaskEvent represents a task event (for streaming).
type TaskEvent struct {
	TaskID   string         `json:"taskId"`
	Type     string         `json:"type"` // "progress", "result", "error"
	Data     map[string]any `json:"data"`
	Timestamp time.Time     `json:"timestamp"`
}

// Message represents an agent-to-agent message.
type Message struct {
	ID        string         `json:"id"`
	From      string         `json:"from"`     // Sender agent ID
	To        string         `json:"to"`       // Recipient agent ID
	Type      MessageType    `json:"type"`     // Message type
	Content   string         `json:"content"`  // Message content
	Data      map[string]any `json:"data,omitempty"`
	ReplyTo   string         `json:"replyTo,omitempty"` // Reply to message ID
	Timestamp time.Time      `json:"timestamp"`
	Metadata  map[string]any `json:"metadata"`
}

// MessageType represents message type.
type MessageType string

const (
	MessageTypeRequest      MessageType = "request"      // Request
	MessageTypeResponse     MessageType = "response"     // Response
	MessageTypeNotification MessageType = "notification" // Notification
	MessageTypeBroadcast    MessageType = "broadcast"    // Broadcast
)

// HelpRequest represents a request for help from other agents.
type HelpRequest struct {
	RequestID    string            `json:"requestId"`
	RequesterID  string            `json:"requesterId"`
	Task         *Task             `json:"task"`
	Reason       string            `json:"reason"`
	Requirements *HelpRequirements `json:"requirements,omitempty"`
}

// HelpRequirements represents requirements for help.
type HelpRequirements struct {
	RequiredCapabilities []string      `json:"requiredCapabilities"`
	PreferredAgents      []string      `json:"preferredAgents,omitempty"`
	MaxWaitTime          time.Duration `json:"maxWaitTime,omitempty"`
}

// HelpResponse represents a response to a help request.
type HelpResponse struct {
	RequestID string `json:"requestId"`
	HelperID  string `json:"helperId"`
	Success   bool   `json:"success"`
	Message   string `json:"message"`
}

// HelpOffer represents an offer to help.
type HelpOffer struct {
	OfferID      string   `json:"offerId"`
	HelperID     string   `json:"helperId"`
	Capabilities []string `json:"capabilities"`
	Availability string   `json:"availability"`
}

// TaskStatus is re-defined to include more status information.
type TaskStatusInfo struct {
	TaskID        string         `json:"taskId"`
	Status        TaskStatus     `json:"status"`
	Progress      float64        `json:"progress"`
	StartedAt     time.Time      `json:"startedAt"`
	UpdatedAt     time.Time      `json:"updatedAt"`
	EstimatedTime *time.Duration `json:"estimatedTime,omitempty"`
}
