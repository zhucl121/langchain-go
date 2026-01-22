// Protocol Bridge Demo
//
// This example demonstrates protocol bridging between MCP and A2A:
// - MCP tool calls → A2A tasks
// - A2A agents → MCP resources
// - Bidirectional protocol translation
//
// Usage:
//   go run main.go
package main

import (
	"context"
	"fmt"
	"log"
	"time"
	
	"github.com/google/uuid"
	"github.com/zhucl121/langchain-go/pkg/protocols/a2a"
	"github.com/zhucl121/langchain-go/pkg/protocols/bridge"
	"github.com/zhucl121/langchain-go/pkg/protocols/mcp"
)

func main() {
	ctx := context.Background()
	
	fmt.Println("=== Protocol Bridge Demo ===\n")
	
	// Step 1: Setup A2A system
	fmt.Println("Step 1: Setting up A2A agent system...")
	
	registry := a2a.NewLocalRegistry()
	router := a2a.NewSmartTaskRouter(registry, a2a.RouterConfig{
		Strategy: a2a.StrategyHybrid,
	})
	
	// Register A2A agents
	researchAgent := &SimpleAgent{name: "Researcher", specialty: "research"}
	researcher := a2a.NewA2AAgentBridge(researchAgent, &a2a.BridgeConfig{
		Info: &a2a.AgentInfo{
			ID:     "agent-researcher",
			Name:   "Researcher Agent",
			Type:   a2a.AgentTypeSpecialist,
			Status: a2a.AgentStatusOnline,
		},
		Capabilities: &a2a.AgentCapabilities{
			Capabilities: []string{"research", "search"},
			Tools:        []string{"search", "web_scraper"},
		},
	})
	registry.Register(ctx, researcher)
	
	analystAgent := &SimpleAgent{name: "Analyst", specialty: "analysis"}
	analyst := a2a.NewA2AAgentBridge(analystAgent, &a2a.BridgeConfig{
		Info: &a2a.AgentInfo{
			ID:     "agent-analyst",
			Name:   "Analyst Agent",
			Type:   a2a.AgentTypeSpecialist,
			Status: a2a.AgentStatusOnline,
		},
		Capabilities: &a2a.AgentCapabilities{
			Capabilities: []string{"analysis", "statistics"},
			Tools:        []string{"data_analyzer"},
		},
	})
	registry.Register(ctx, analyst)
	
	fmt.Println("✓ A2A agents registered\n")
	
	// Step 2: Setup MCP server
	fmt.Println("Step 2: Setting up MCP server...")
	
	mcpServer := mcp.NewServer(mcp.ServerConfig{
		Name:    "bridge-server",
		Version: "1.0.0",
		Debug:   true,
	})
	
	fmt.Println("✓ MCP server created\n")
	
	// Step 3: Create bidirectional bridge
	fmt.Println("Step 3: Creating bidirectional bridge...")
	
	bridgeInstance := bridge.NewBidirectionalBridge(mcpServer, router, registry)
	
	// Setup bridge (expose A2A agents as MCP resources)
	if err := bridgeInstance.Setup(ctx); err != nil {
		log.Fatal(err)
	}
	
	fmt.Println("✓ Bridge setup completed\n")
	
	// Step 4: Test MCP → A2A bridging
	fmt.Println("Step 4: Testing MCP → A2A bridging...")
	
	fmt.Println("Simulating MCP tool call...")
	task := bridgeInstance.MCPToolToA2ATask("research", map[string]any{
		"query": "AI developments in 2026",
	})
	
	fmt.Printf("✓ MCP tool call converted to A2A task\n")
	fmt.Printf("  Task ID: %s\n", task.ID)
	fmt.Printf("  Task Type: %s\n", task.Type)
	
	// Route and execute via A2A
	agent, err := router.Route(ctx, task)
	if err != nil {
		log.Fatal(err)
	}
	
	info, _ := agent.GetInfo(ctx)
	fmt.Printf("  Routed to: %s\n", info.Name)
	
	response, err := agent.SendTask(ctx, task)
	if err != nil {
		log.Fatal(err)
	}
	
	// Convert back to MCP result
	mcpResult := bridgeInstance.A2AResponseToMCPResult(response)
	fmt.Printf("  Result: %s\n\n", mcpResult.Content[0].Text[:80]+"...")
	
	// Step 5: Test A2A → MCP bridging
	fmt.Println("Step 5: Testing A2A → MCP bridging...")
	
	// List MCP resources (should include A2A agents)
	resources, err := mcpServer.ListResources(ctx)
	if err != nil {
		log.Fatal(err)
	}
	
	fmt.Printf("✓ MCP resources (including A2A agents): %d\n", len(resources))
	for _, res := range resources {
		if res.Metadata["agentId"] != nil {
			fmt.Printf("  - %s (URI: %s)\n", res.Name, res.URI)
			fmt.Printf("    Agent ID: %s\n", res.Metadata["agentId"])
			fmt.Printf("    Capabilities: %v\n", res.Metadata["capabilities"])
		}
	}
	
	fmt.Println()
	
	// Step 6: Read A2A agent as MCP resource
	fmt.Println("Step 6: Reading A2A agent as MCP resource...")
	
	if len(resources) > 0 {
		content, err := mcpServer.ReadResource(ctx, resources[0].URI)
		if err != nil {
			log.Fatal(err)
		}
		
		fmt.Printf("✓ Resource content:\n%s\n\n", content.Text)
	}
	
	// Step 7: Performance test
	fmt.Println("Step 7: Performance test...")
	
	startTime := time.Now()
	for i := 0; i < 10; i++ {
		task := bridgeInstance.MCPToolToA2ATask("analyze", map[string]any{
			"data": fmt.Sprintf("dataset_%d", i),
		})
		
		agent, _ := router.Route(ctx, task)
		agent.SendTask(ctx, task)
	}
	duration := time.Since(startTime)
	
	fmt.Printf("✓ Processed 10 tasks in %v\n", duration)
	fmt.Printf("  Average: %.2fms per task\n", float64(duration.Milliseconds())/10.0)
	
	fmt.Println("\n=== Demo Completed ===")
}

// SimpleAgent is a simple agent implementation.
type SimpleAgent struct {
	name      string
	specialty string
}

// Execute executes a task.
func (a *SimpleAgent) Execute(ctx context.Context, input string) (string, error) {
	time.Sleep(100 * time.Millisecond)
	
	switch a.specialty {
	case "research":
		return fmt.Sprintf("[%s] Research completed: %s. Found comprehensive information.", a.name, input), nil
	case "analysis":
		return fmt.Sprintf("[%s] Analysis completed: %s. Generated insights and recommendations.", a.name, input), nil
	default:
		return fmt.Sprintf("[%s] Processed: %s", a.name, input), nil
	}
}

// GetName returns the agent name.
func (a *SimpleAgent) GetName() string {
	return a.name
}

// GetDescription returns the agent description.
func (a *SimpleAgent) GetDescription() string {
	return fmt.Sprintf("%s agent specializing in %s", a.name, a.specialty)
}
