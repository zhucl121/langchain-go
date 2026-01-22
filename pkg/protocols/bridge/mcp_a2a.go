// Package bridge provides protocol bridging between MCP and A2A.
package bridge

import (
	"context"
	"fmt"
	
	"github.com/google/uuid"
	"github.com/zhucl121/langchain-go/pkg/protocols/a2a"
	"github.com/zhucl121/langchain-go/pkg/protocols/mcp"
)

// MCPToA2ABridge bridges MCP tool calls to A2A tasks.
//
// This allows MCP clients (like Claude Desktop) to invoke A2A agents.
//
// Example:
//
//	bridge := bridge.NewMCPToA2ABridge(mcpServer, router)
//	// MCP tool calls are automatically routed to A2A agents
type MCPToA2ABridge struct {
	mcpServer mcp.MCPServer
	router    a2a.TaskRouter
	registry  a2a.AgentRegistry
}

// NewMCPToA2ABridge creates a new MCP to A2A bridge.
func NewMCPToA2ABridge(mcpServer mcp.MCPServer, router a2a.TaskRouter, registry a2a.AgentRegistry) *MCPToA2ABridge {
	return &MCPToA2ABridge{
		mcpServer: mcpServer,
		router:    router,
		registry:  registry,
	}
}

// ToolCallToTask converts an MCP tool call to an A2A task.
func (b *MCPToA2ABridge) ToolCallToTask(name string, args map[string]any) *a2a.Task {
	return &a2a.Task{
		ID:       uuid.New().String(),
		Type:     a2a.TaskTypeExecute,
		Priority: a2a.PriorityMedium,
		Input: &a2a.TaskInput{
			Type: "tool",
			Data: map[string]any{
				"tool": name,
				"args": args,
			},
		},
		Metadata: map[string]any{
			"source": "mcp",
		},
	}
}

// TaskResponseToToolResult converts an A2A task response to MCP tool result.
func (b *MCPToA2ABridge) TaskResponseToToolResult(response *a2a.TaskResponse) *mcp.ToolResult {
	if response.Status == a2a.TaskStatusFailed {
		return &mcp.ToolResult{
			Content: []mcp.ContentBlock{
				{
					Type: "text",
					Text: response.Error.Message,
				},
			},
			IsError: true,
		}
	}
	
	return &mcp.ToolResult{
		Content: []mcp.ContentBlock{
			{
				Type: "text",
				Text: response.Result.Content,
			},
		},
		IsError: false,
	}
}

// ExecuteToolViaA2A executes an MCP tool call via A2A routing.
func (b *MCPToA2ABridge) ExecuteToolViaA2A(ctx context.Context, name string, args map[string]any) (*mcp.ToolResult, error) {
	// Convert to A2A task
	task := b.ToolCallToTask(name, args)
	
	// Route to appropriate agent
	agent, err := b.router.Route(ctx, task)
	if err != nil {
		return &mcp.ToolResult{
			Content: []mcp.ContentBlock{
				{
					Type: "text",
					Text: fmt.Sprintf("Routing failed: %v", err),
				},
			},
			IsError: true,
		}, nil
	}
	
	// Execute task
	response, err := agent.SendTask(ctx, task)
	if err != nil {
		return &mcp.ToolResult{
			Content: []mcp.ContentBlock{
				{
					Type: "text",
					Text: fmt.Sprintf("Execution failed: %v", err),
				},
			},
			IsError: true,
		}, nil
	}
	
	// Convert response back to MCP tool result
	return b.TaskResponseToToolResult(response), nil
}

// A2AToMCPBridge bridges A2A agents to MCP resources.
//
// This allows A2A agents to be exposed as MCP resources that can be
// accessed by MCP clients.
//
// Example:
//
//	bridge := bridge.NewA2AToMCPBridge(registry, mcpServer)
//	resources := bridge.ExposeAgentsAsResources()
type A2AToMCPBridge struct {
	registry  a2a.AgentRegistry
	mcpServer *mcp.DefaultMCPServer
}

// NewA2AToMCPBridge creates a new A2A to MCP bridge.
func NewA2AToMCPBridge(registry a2a.AgentRegistry, mcpServer *mcp.DefaultMCPServer) *A2AToMCPBridge {
	return &A2AToMCPBridge{
		registry:  registry,
		mcpServer: mcpServer,
	}
}

// ExposeAgentsAsResources exposes A2A agents as MCP resources.
func (b *A2AToMCPBridge) ExposeAgentsAsResources(ctx context.Context) ([]*mcp.Resource, error) {
	agents, err := b.registry.ListAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("list agents: %w", err)
	}
	
	resources := make([]*mcp.Resource, 0, len(agents))
	for _, agent := range agents {
		info, err := agent.GetInfo(ctx)
		if err != nil {
			continue
		}
		
		caps, err := agent.GetCapabilities(ctx)
		if err != nil {
			continue
		}
		
		// Create resource for each agent
		resource := &mcp.Resource{
			URI:         fmt.Sprintf("a2a://agent/%s", info.ID),
			Name:        info.Name,
			Description: info.Description,
			MimeType:    "application/json",
			Metadata: map[string]any{
				"agentId":      info.ID,
				"type":         info.Type,
				"status":       info.Status,
				"capabilities": caps.Capabilities,
			},
		}
		
		resources = append(resources, resource)
	}
	
	return resources, nil
}

// CreateAgentResourceProvider creates a resource provider for an A2A agent.
func (b *A2AToMCPBridge) CreateAgentResourceProvider(agent a2a.A2AAgent) mcp.ResourceProvider {
	return &AgentResourceProvider{
		agent: agent,
	}
}

// AgentResourceProvider provides access to A2A agent as MCP resource.
type AgentResourceProvider struct {
	agent a2a.A2AAgent
}

// Read reads agent information as resource content.
func (p *AgentResourceProvider) Read(ctx context.Context, uri string) (*mcp.ResourceContent, error) {
	info, err := p.agent.GetInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("get agent info: %w", err)
	}
	
	caps, err := p.agent.GetCapabilities(ctx)
	if err != nil {
		return nil, fmt.Errorf("get agent capabilities: %w", err)
	}
	
	// Format agent info as JSON
	content := fmt.Sprintf(`{
  "id": "%s",
  "name": "%s",
  "description": "%s",
  "type": "%s",
  "status": "%s",
  "capabilities": %v
}`, info.ID, info.Name, info.Description, info.Type, info.Status, caps.Capabilities)
	
	return &mcp.ResourceContent{
		URI:      uri,
		MimeType: "application/json",
		Text:     content,
	}, nil
}

// Subscribe is not supported for agent resources.
func (p *AgentResourceProvider) Subscribe(ctx context.Context, uri string) (<-chan *mcp.ResourceContent, error) {
	return nil, fmt.Errorf("subscribe not supported for agent resources")
}

// Unsubscribe is not supported for agent resources.
func (p *AgentResourceProvider) Unsubscribe(ctx context.Context, uri string) error {
	return fmt.Errorf("unsubscribe not supported for agent resources")
}

// BidirectionalBridge provides bidirectional bridging between MCP and A2A.
type BidirectionalBridge struct {
	mcpToA2A  *MCPToA2ABridge
	a2aToMCP  *A2AToMCPBridge
	mcpServer *mcp.DefaultMCPServer // Use concrete type to access RegisterResource
}

// NewBidirectionalBridge creates a new bidirectional bridge.
func NewBidirectionalBridge(
	mcpServer *mcp.DefaultMCPServer,
	router a2a.TaskRouter,
	registry a2a.AgentRegistry,
) *BidirectionalBridge {
	return &BidirectionalBridge{
		mcpToA2A:  NewMCPToA2ABridge(mcpServer, router, registry),
		a2aToMCP:  NewA2AToMCPBridge(registry, mcpServer),
		mcpServer: mcpServer,
	}
}

// Setup sets up the bidirectional bridge.
func (b *BidirectionalBridge) Setup(ctx context.Context) error {
	// Expose A2A agents as MCP resources
	resources, err := b.a2aToMCP.ExposeAgentsAsResources(ctx)
	if err != nil {
		return fmt.Errorf("expose agents as resources: %w", err)
	}
	
	// Register each agent as a resource
	for _, resource := range resources {
		// Find agent by ID
		agentID := resource.Metadata["agentId"].(string)
		agent, err := b.a2aToMCP.registry.FindByID(ctx, agentID)
		if err != nil {
			continue
		}
		
		provider := b.a2aToMCP.CreateAgentResourceProvider(agent)
		if err := b.mcpServer.RegisterResource(resource, provider); err != nil {
			return fmt.Errorf("register resource: %w", err)
		}
	}
	
	return nil
}

// MCPToolToA2ATask converts MCP tool to A2A task.
func (b *BidirectionalBridge) MCPToolToA2ATask(name string, args map[string]any) *a2a.Task {
	return b.mcpToA2A.ToolCallToTask(name, args)
}

// A2AResponseToMCPResult converts A2A response to MCP result.
func (b *BidirectionalBridge) A2AResponseToMCPResult(response *a2a.TaskResponse) *mcp.ToolResult {
	return b.mcpToA2A.TaskResponseToToolResult(response)
}
