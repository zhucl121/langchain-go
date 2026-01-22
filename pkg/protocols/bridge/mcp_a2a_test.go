package bridge

import (
	"context"
	"testing"
	
	"github.com/google/uuid"
	"github.com/zhucl121/langchain-go/pkg/protocols/a2a"
	"github.com/zhucl121/langchain-go/pkg/protocols/mcp"
)

func TestNewMCPToA2ABridge(t *testing.T) {
	mcpServer := mcp.NewServer(mcp.ServerConfig{Name: "test"})
	registry := a2a.NewLocalRegistry()
	router := a2a.NewSmartTaskRouter(registry, a2a.RouterConfig{})
	
	bridge := NewMCPToA2ABridge(mcpServer, router, registry)
	
	if bridge == nil {
		t.Fatal("NewMCPToA2ABridge() returned nil")
	}
}

func TestMCPToA2ABridge_ToolCallToTask(t *testing.T) {
	mcpServer := mcp.NewServer(mcp.ServerConfig{Name: "test"})
	registry := a2a.NewLocalRegistry()
	router := a2a.NewSmartTaskRouter(registry, a2a.RouterConfig{})
	
	bridge := NewMCPToA2ABridge(mcpServer, router, registry)
	
	toolName := "calculator"
	args := map[string]any{
		"expression": "2 + 2",
	}
	
	task := bridge.ToolCallToTask(toolName, args)
	
	if task == nil {
		t.Fatal("ToolCallToTask() returned nil")
	}
	
	if task.Type != a2a.TaskTypeExecute {
		t.Errorf("Task.Type = %v, want %v", task.Type, a2a.TaskTypeExecute)
	}
	
	if task.Input.Type != "tool" {
		t.Errorf("Task.Input.Type = %v, want tool", task.Input.Type)
	}
	
	if task.Input.Data["tool"] != toolName {
		t.Errorf("Task tool name = %v, want %v", task.Input.Data["tool"], toolName)
	}
}

func TestMCPToA2ABridge_TaskResponseToToolResult(t *testing.T) {
	mcpServer := mcp.NewServer(mcp.ServerConfig{Name: "test"})
	registry := a2a.NewLocalRegistry()
	router := a2a.NewSmartTaskRouter(registry, a2a.RouterConfig{})
	
	bridge := NewMCPToA2ABridge(mcpServer, router, registry)
	
	tests := []struct {
		name     string
		response *a2a.TaskResponse
		wantErr  bool
	}{
		{
			name: "Successful response",
			response: &a2a.TaskResponse{
				TaskID: "task-1",
				Status: a2a.TaskStatusCompleted,
				Result: &a2a.TaskResult{
					Type:    "text",
					Content: "Success result",
				},
			},
			wantErr: false,
		},
		{
			name: "Failed response",
			response: &a2a.TaskResponse{
				TaskID: "task-2",
				Status: a2a.TaskStatusFailed,
				Error: &a2a.TaskError{
					Code:    "ERROR",
					Message: "Task failed",
				},
			},
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := bridge.TaskResponseToToolResult(tt.response)
			
			if result == nil {
				t.Fatal("TaskResponseToToolResult() returned nil")
			}
			
			if result.IsError != tt.wantErr {
				t.Errorf("Result.IsError = %v, want %v", result.IsError, tt.wantErr)
			}
			
			if len(result.Content) == 0 {
				t.Error("Expected result content to be non-empty")
			}
		})
	}
}

func TestNewA2AToMCPBridge(t *testing.T) {
	registry := a2a.NewLocalRegistry()
	mcpServer := mcp.NewServer(mcp.ServerConfig{Name: "test"})
	
	bridge := NewA2AToMCPBridge(registry, mcpServer)
	
	if bridge == nil {
		t.Fatal("NewA2AToMCPBridge() returned nil")
	}
}

func TestA2AToMCPBridge_ExposeAgentsAsResources(t *testing.T) {
	registry := a2a.NewLocalRegistry()
	mcpServer := mcp.NewServer(mcp.ServerConfig{Name: "test"})
	ctx := context.Background()
	
	// Register A2A agents
	agent1 := &mockA2AAgent{
		info: &a2a.AgentInfo{
			ID:          "agent-1",
			Name:        "Test Agent 1",
			Description: "A test agent",
			Type:        a2a.AgentTypeSpecialist,
			Status:      a2a.AgentStatusOnline,
		},
		capabilities: &a2a.AgentCapabilities{
			Capabilities: []string{"test", "demo"},
		},
	}
	
	agent2 := &mockA2AAgent{
		info: &a2a.AgentInfo{
			ID:   "agent-2",
			Name: "Test Agent 2",
		},
		capabilities: &a2a.AgentCapabilities{
			Capabilities: []string{"analysis"},
		},
	}
	
	registry.Register(ctx, agent1)
	registry.Register(ctx, agent2)
	
	bridge := NewA2AToMCPBridge(registry, mcpServer)
	
	resources, err := bridge.ExposeAgentsAsResources(ctx)
	if err != nil {
		t.Fatalf("ExposeAgentsAsResources() error = %v", err)
	}
	
	if len(resources) != 2 {
		t.Errorf("Expected 2 resources, got %d", len(resources))
	}
	
	// Verify resource structure
	for _, res := range resources {
		if res.URI == "" {
			t.Error("Resource URI should not be empty")
		}
		
		if res.Name == "" {
			t.Error("Resource Name should not be empty")
		}
		
		if res.MimeType != "application/json" {
			t.Errorf("Resource MimeType = %v, want application/json", res.MimeType)
		}
		
		if res.Metadata["agentId"] == nil {
			t.Error("Resource metadata should include agentId")
		}
	}
}

func TestAgentResourceProvider_Read(t *testing.T) {
	agent := &mockA2AAgent{
		info: &a2a.AgentInfo{
			ID:          "agent-1",
			Name:        "Test Agent",
			Description: "A test agent",
			Type:        a2a.AgentTypeSpecialist,
			Status:      a2a.AgentStatusOnline,
		},
		capabilities: &a2a.AgentCapabilities{
			Capabilities: []string{"test"},
		},
	}
	
	provider := &AgentResourceProvider{agent: agent}
	
	ctx := context.Background()
	content, err := provider.Read(ctx, "a2a://agent/agent-1")
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	
	if content.MimeType != "application/json" {
		t.Errorf("MimeType = %v, want application/json", content.MimeType)
	}
	
	if content.Text == "" {
		t.Error("Expected content.Text to be non-empty")
	}
}

func TestNewBidirectionalBridge(t *testing.T) {
	mcpServer := mcp.NewServer(mcp.ServerConfig{Name: "test"})
	registry := a2a.NewLocalRegistry()
	router := a2a.NewSmartTaskRouter(registry, a2a.RouterConfig{})
	
	bridge := NewBidirectionalBridge(mcpServer, router, registry)
	
	if bridge == nil {
		t.Fatal("NewBidirectionalBridge() returned nil")
	}
	
	if bridge.mcpToA2A == nil {
		t.Error("Expected mcpToA2A to be initialized")
	}
	
	if bridge.a2aToMCP == nil {
		t.Error("Expected a2aToMCP to be initialized")
	}
}

func TestBidirectionalBridge_Setup(t *testing.T) {
	mcpServer := mcp.NewServer(mcp.ServerConfig{Name: "test"})
	registry := a2a.NewLocalRegistry()
	ctx := context.Background()
	
	// Register A2A agent
	agent := &mockA2AAgent{
		info: &a2a.AgentInfo{
			ID:     "agent-1",
			Name:   "Test Agent",
			Status: a2a.AgentStatusOnline,
		},
		capabilities: &a2a.AgentCapabilities{
			Capabilities: []string{"test"},
		},
	}
	registry.Register(ctx, agent)
	
	router := a2a.NewSmartTaskRouter(registry, a2a.RouterConfig{})
	bridge := NewBidirectionalBridge(mcpServer, router, registry)
	
	// Setup bridge
	err := bridge.Setup(ctx)
	if err != nil {
		t.Fatalf("Setup() error = %v", err)
	}
	
	// Verify A2A agents are exposed as MCP resources
	resources, err := mcpServer.ListResources(ctx)
	if err != nil {
		t.Fatalf("ListResources() error = %v", err)
	}
	
	if len(resources) != 1 {
		t.Errorf("Expected 1 MCP resource, got %d", len(resources))
	}
}

func TestBidirectionalBridge_Conversions(t *testing.T) {
	mcpServer := mcp.NewServer(mcp.ServerConfig{Name: "test"})
	registry := a2a.NewLocalRegistry()
	router := a2a.NewSmartTaskRouter(registry, a2a.RouterConfig{})
	
	bridge := NewBidirectionalBridge(mcpServer, router, registry)
	
	// Test MCP → A2A
	toolName := "test-tool"
	args := map[string]any{"param": "value"}
	task := bridge.MCPToolToA2ATask(toolName, args)
	
	if task == nil {
		t.Fatal("MCPToolToA2ATask() returned nil")
	}
	
	// Test A2A → MCP
	response := &a2a.TaskResponse{
		TaskID: uuid.New().String(),
		Status: a2a.TaskStatusCompleted,
		Result: &a2a.TaskResult{
			Type:    "text",
			Content: "test result",
		},
	}
	
	mcpResult := bridge.A2AResponseToMCPResult(response)
	if mcpResult == nil {
		t.Fatal("A2AResponseToMCPResult() returned nil")
	}
	
	if mcpResult.IsError {
		t.Error("Expected IsError to be false for completed task")
	}
}

// Mock implementations

type mockA2AAgent struct {
	info         *a2a.AgentInfo
	capabilities *a2a.AgentCapabilities
}

func (m *mockA2AAgent) GetInfo(ctx context.Context) (*a2a.AgentInfo, error) {
	return m.info, nil
}

func (m *mockA2AAgent) GetCapabilities(ctx context.Context) (*a2a.AgentCapabilities, error) {
	return m.capabilities, nil
}

func (m *mockA2AAgent) SendTask(ctx context.Context, task *a2a.Task) (*a2a.TaskResponse, error) {
	return &a2a.TaskResponse{
		TaskID:   task.ID,
		Status:   a2a.TaskStatusCompleted,
		Result:   &a2a.TaskResult{Type: "text", Content: "mock result"},
		Progress: 1.0,
	}, nil
}

func (m *mockA2AAgent) GetTaskStatus(ctx context.Context, taskID string) (*a2a.TaskStatus, error) {
	status := a2a.TaskStatusCompleted
	return &status, nil
}

func (m *mockA2AAgent) CancelTask(ctx context.Context, taskID string) error {
	return nil
}

func (m *mockA2AAgent) StreamTask(ctx context.Context, task *a2a.Task) (<-chan *a2a.TaskEvent, error) {
	return nil, nil
}

func (m *mockA2AAgent) SendMessage(ctx context.Context, msg *a2a.Message) error {
	return nil
}

func (m *mockA2AAgent) ReceiveMessages(ctx context.Context) (<-chan *a2a.Message, error) {
	return nil, nil
}

func (m *mockA2AAgent) RequestHelp(ctx context.Context, req *a2a.HelpRequest) (*a2a.HelpResponse, error) {
	return nil, nil
}

func (m *mockA2AAgent) OfferHelp(ctx context.Context, offer *a2a.HelpOffer) error {
	return nil
}
