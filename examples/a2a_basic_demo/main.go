// A2A Basic Demo
//
// This example demonstrates basic A2A (Agent-to-Agent) protocol usage:
// - Register agents to a local registry
// - Discover agents by capability
// - Send tasks to agents
// - Track task status
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
)

func main() {
	ctx := context.Background()
	
	fmt.Println("=== A2A Basic Demo ===\n")
	
	// Step 1: Create local registry
	fmt.Println("Step 1: Creating local agent registry...")
	registry := a2a.NewLocalRegistry()
	
	// Step 2: Create and register agents
	fmt.Println("Step 2: Creating and registering agents...\n")
	
	// Research Agent
	researchAgent := &SimpleAgent{
		name:        "Research Agent",
		description: "Specializes in research and information gathering",
		capability:  "research",
	}
	
	researchA2A := a2a.NewA2AAgentBridge(researchAgent, &a2a.BridgeConfig{
		Info: &a2a.AgentInfo{
			ID:          "agent-research",
			Name:        "Research Agent",
			Description: "Specializes in research and information gathering",
			Version:     "1.0.0",
			Type:        a2a.AgentTypeSpecialist,
			Status:      a2a.AgentStatusOnline,
		},
		Capabilities: &a2a.AgentCapabilities{
			Capabilities:       []string{"research", "search", "analysis"},
			Tools:              []string{"search", "web_scraper"},
			MessageFormats:     []string{"text", "json"},
			Languages:          []string{"zh", "en"},
			MaxConcurrentTasks: 3,
			AvgResponseTime:    2 * time.Second,
		},
	})
	
	if err := registry.Register(ctx, researchA2A); err != nil {
		log.Fatal(err)
	}
	fmt.Println("✓ Registered: Research Agent (ID: agent-research)")
	
	// Analysis Agent
	analysisAgent := &SimpleAgent{
		name:        "Analysis Agent",
		description: "Specializes in data analysis",
		capability:  "analysis",
	}
	
	analysisA2A := a2a.NewA2AAgentBridge(analysisAgent, &a2a.BridgeConfig{
		Info: &a2a.AgentInfo{
			ID:          "agent-analysis",
			Name:        "Analysis Agent",
			Description: "Specializes in data analysis",
			Version:     "1.0.0",
			Type:        a2a.AgentTypeSpecialist,
			Status:      a2a.AgentStatusOnline,
		},
		Capabilities: &a2a.AgentCapabilities{
			Capabilities:       []string{"analysis", "statistics", "visualization"},
			Tools:              []string{"data_analyzer", "plotter"},
			MessageFormats:     []string{"text", "json"},
			Languages:          []string{"zh", "en"},
			MaxConcurrentTasks: 2,
			AvgResponseTime:    3 * time.Second,
		},
	})
	
	if err := registry.Register(ctx, analysisA2A); err != nil {
		log.Fatal(err)
	}
	fmt.Println("✓ Registered: Analysis Agent (ID: agent-analysis)")
	
	// Writing Agent
	writingAgent := &SimpleAgent{
		name:        "Writing Agent",
		description: "Specializes in content writing",
		capability:  "writing",
	}
	
	writingA2A := a2a.NewA2AAgentBridge(writingAgent, &a2a.BridgeConfig{
		Info: &a2a.AgentInfo{
			ID:          "agent-writing",
			Name:        "Writing Agent",
			Description: "Specializes in content writing",
			Version:     "1.0.0",
			Type:        a2a.AgentTypeSpecialist,
			Status:      a2a.AgentStatusOnline,
		},
		Capabilities: &a2a.AgentCapabilities{
			Capabilities:       []string{"writing", "editing", "summarization"},
			Tools:              []string{"grammar_checker", "style_analyzer"},
			MessageFormats:     []string{"text", "markdown"},
			Languages:          []string{"zh", "en"},
			MaxConcurrentTasks: 5,
			AvgResponseTime:    1 * time.Second,
		},
	})
	
	if err := registry.Register(ctx, writingA2A); err != nil {
		log.Fatal(err)
	}
	fmt.Println("✓ Registered: Writing Agent (ID: agent-writing)")
	
	fmt.Println()
	
	// Step 3: List all agents
	fmt.Println("Step 3: Listing all registered agents...")
	allAgents, err := registry.ListAll(ctx)
	if err != nil {
		log.Fatal(err)
	}
	
	fmt.Printf("Total agents: %d\n\n", len(allAgents))
	for _, agent := range allAgents {
		info, _ := agent.GetInfo(ctx)
		caps, _ := agent.GetCapabilities(ctx)
		fmt.Printf("- %s (ID: %s)\n", info.Name, info.ID)
		fmt.Printf("  Type: %s, Status: %s\n", info.Type, info.Status)
		fmt.Printf("  Capabilities: %v\n", caps.Capabilities)
		fmt.Println()
	}
	
	// Step 4: Discover agents by capability
	fmt.Println("Step 4: Discovering agents by capability...")
	
	researchAgents, err := registry.FindByCapability(ctx, "research")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Found %d agent(s) with 'research' capability\n", len(researchAgents))
	
	analysisAgents, err := registry.FindByCapability(ctx, "analysis")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Found %d agent(s) with 'analysis' capability\n", len(analysisAgents))
	
	fmt.Println()
	
	// Step 5: Send task to specific agent
	fmt.Println("Step 5: Sending task to Research Agent...")
	
	task := &a2a.Task{
		ID:       uuid.New().String(),
		Type:     a2a.TaskTypeQuery,
		Priority: a2a.PriorityHigh,
		Input: &a2a.TaskInput{
			Type:    "text",
			Content: "Research the latest AI developments in 2026",
		},
		Metadata: make(map[string]any),
	}
	
	fmt.Printf("Task ID: %s\n", task.ID)
	fmt.Printf("Task Type: %s\n", task.Type)
	fmt.Printf("Task Content: %s\n\n", task.Input.Content)
	
	response, err := researchAgents[0].SendTask(ctx, task)
	if err != nil {
		log.Fatal(err)
	}
	
	fmt.Printf("Task Status: %s\n", response.Status)
	fmt.Printf("Task Progress: %.0f%%\n", response.Progress*100)
	if response.Result != nil {
		fmt.Printf("Result: %s\n", response.Result.Content)
	}
	
	fmt.Println()
	
	// Step 6: Send task to Analysis Agent
	fmt.Println("Step 6: Sending task to Analysis Agent...")
	
	task2 := &a2a.Task{
		ID:       uuid.New().String(),
		Type:     a2a.TaskTypeAnalyze,
		Priority: a2a.PriorityMedium,
		Input: &a2a.TaskInput{
			Type:    "data",
			Content: "Analyze user behavior data",
			Data: map[string]any{
				"dataset": "user_behavior",
				"metrics": []string{"engagement", "retention"},
			},
		},
	}
	
	response2, err := analysisAgents[0].SendTask(ctx, task2)
	if err != nil {
		log.Fatal(err)
	}
	
	fmt.Printf("Task Status: %s\n", response2.Status)
	if response2.Result != nil {
		fmt.Printf("Result: %s\n", response2.Result.Content)
	}
	
	fmt.Println()
	
	// Step 7: Check agent health
	fmt.Println("Step 7: Checking agent health...")
	
	health, err := registry.CheckHealth(ctx, "agent-research")
	if err != nil {
		log.Fatal(err)
	}
	
	fmt.Printf("Agent ID: %s\n", health.AgentID)
	fmt.Printf("Status: %s\n", health.Status)
	fmt.Printf("Last Heartbeat: %v\n", health.LastHeartbeat.Format(time.RFC3339))
	fmt.Printf("Uptime: %v\n", health.Uptime)
	
	fmt.Println("\n=== Demo Completed ===")
}

// SimpleAgent is a simple agent implementation for demo purposes.
type SimpleAgent struct {
	name        string
	description string
	capability  string
}

// Execute executes a task.
func (a *SimpleAgent) Execute(ctx context.Context, input string) (string, error) {
	// Simulate some processing time
	time.Sleep(100 * time.Millisecond)
	
	// Generate response based on capability
	var response string
	switch a.capability {
	case "research":
		response = fmt.Sprintf("[%s] Researched: '%s'. Found relevant information about AI developments, including advances in LLMs, agent systems, and more.", a.name, input)
	case "analysis":
		response = fmt.Sprintf("[%s] Analyzed: '%s'. Key insights: High user engagement, positive retention trends, actionable recommendations provided.", a.name, input)
	case "writing":
		response = fmt.Sprintf("[%s] Written content for: '%s'. Generated well-structured content with proper formatting.", a.name, input)
	default:
		response = fmt.Sprintf("[%s] Processed: '%s'", a.name, input)
	}
	
	return response, nil
}

// GetName returns the agent name.
func (a *SimpleAgent) GetName() string {
	return a.name
}

// GetDescription returns the agent description.
func (a *SimpleAgent) GetDescription() string {
	return a.description
}
