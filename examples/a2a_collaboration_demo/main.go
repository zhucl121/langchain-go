// A2A Collaboration Demo
//
// This example demonstrates multi-agent collaboration using A2A protocol:
// - Smart task routing
// - Multi-agent coordination
// - Complex task decomposition
// - Result aggregation
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
	
	fmt.Println("=== A2A Collaboration Demo ===\n")
	
	// Step 1: Setup registry and router
	fmt.Println("Step 1: Setting up agent registry and router...")
	
	registry := a2a.NewLocalRegistry()
	router := a2a.NewSmartTaskRouter(registry, a2a.RouterConfig{
		Strategy: a2a.StrategyHybrid,
	})
	
	fmt.Println("✓ Registry and router created\n")
	
	// Step 2: Register specialized agents
	fmt.Println("Step 2: Registering specialized agents...")
	
	// Researcher Agent
	researcherAgent := &SpecializedAgent{
		name:        "Researcher",
		specialty:   "research",
		description: "Expert in information gathering and research",
	}
	
	researcher := a2a.NewA2AAgentBridge(researcherAgent, &a2a.BridgeConfig{
		Info: &a2a.AgentInfo{
			ID:          "agent-researcher",
			Name:        "Researcher Agent",
			Description: "Expert in research and information gathering",
			Type:        a2a.AgentTypeSpecialist,
			Status:      a2a.AgentStatusOnline,
		},
		Capabilities: &a2a.AgentCapabilities{
			Capabilities:       []string{"research", "search", "information_gathering"},
			Tools:              []string{"search", "web_scraper", "academic_db"},
			MaxConcurrentTasks: 3,
		},
	})
	registry.Register(ctx, researcher)
	fmt.Println("✓ Registered: Researcher Agent")
	
	// Analyst Agent
	analystAgent := &SpecializedAgent{
		name:        "Analyst",
		specialty:   "analysis",
		description: "Expert in data analysis and insights",
	}
	
	analyst := a2a.NewA2AAgentBridge(analystAgent, &a2a.BridgeConfig{
		Info: &a2a.AgentInfo{
			ID:          "agent-analyst",
			Name:        "Analyst Agent",
			Description: "Expert in data analysis and insights",
			Type:        a2a.AgentTypeSpecialist,
			Status:      a2a.AgentStatusOnline,
		},
		Capabilities: &a2a.AgentCapabilities{
			Capabilities:       []string{"analysis", "statistics", "data_science"},
			Tools:              []string{"data_analyzer", "statistics_engine"},
			MaxConcurrentTasks: 2,
		},
	})
	registry.Register(ctx, analyst)
	fmt.Println("✓ Registered: Analyst Agent")
	
	// Writer Agent
	writerAgent := &SpecializedAgent{
		name:        "Writer",
		specialty:   "writing",
		description: "Expert in content creation and writing",
	}
	
	writer := a2a.NewA2AAgentBridge(writerAgent, &a2a.BridgeConfig{
		Info: &a2a.AgentInfo{
			ID:          "agent-writer",
			Name:        "Writer Agent",
			Description: "Expert in content creation and writing",
			Type:        a2a.AgentTypeSpecialist,
			Status:      a2a.AgentStatusOnline,
		},
		Capabilities: &a2a.AgentCapabilities{
			Capabilities:       []string{"writing", "editing", "content_creation"},
			Tools:              []string{"grammar_checker", "style_guide"},
			MaxConcurrentTasks: 5,
		},
	})
	registry.Register(ctx, writer)
	fmt.Println("✓ Registered: Writer Agent\n")
	
	// Step 3: Simple task routing
	fmt.Println("Step 3: Testing simple task routing...")
	
	simpleTask := &a2a.Task{
		ID:       uuid.New().String(),
		Type:     a2a.TaskTypeQuery,
		Priority: a2a.PriorityMedium,
		Input: &a2a.TaskInput{
			Type:    "text",
			Content: "Find information about AI developments",
		},
		Requirements: &a2a.TaskRequirements{
			RequiredTools: []string{"search"},
		},
	}
	
	agent, err := router.Route(ctx, simpleTask)
	if err != nil {
		log.Fatal(err)
	}
	
	info, _ := agent.GetInfo(ctx)
	fmt.Printf("Task routed to: %s\n", info.Name)
	
	response, err := agent.SendTask(ctx, simpleTask)
	if err != nil {
		log.Fatal(err)
	}
	
	fmt.Printf("Status: %s\n", response.Status)
	fmt.Printf("Result: %s\n\n", response.Result.Content[:100]+"...")
	
	// Step 4: Complex task with coordinator
	fmt.Println("Step 4: Testing complex task coordination...")
	
	coordinator := a2a.NewCollaborationCoordinator(registry, router)
	
	complexTask := &a2a.Task{
		ID:       uuid.New().String(),
		Type:     a2a.TaskTypeComplex,
		Priority: a2a.PriorityHigh,
		Input: &a2a.TaskInput{
			Type:    "text",
			Content: "Research AI trends in 2026, analyze the data, and write a comprehensive report",
		},
	}
	
	fmt.Printf("Complex task: %s\n", complexTask.Input.Content)
	fmt.Println("Decomposing and coordinating...")
	
	startTime := time.Now()
	finalResult, err := coordinator.Coordinate(ctx, complexTask)
	if err != nil {
		log.Fatal(err)
	}
	duration := time.Since(startTime)
	
	fmt.Printf("\n✓ Coordination completed in %v\n\n", duration)
	fmt.Println("=== Final Result ===")
	fmt.Println(finalResult.Content)
	
	// Step 5: Show session details
	fmt.Println("\n=== Collaboration Session Details ===")
	sessions := coordinator.ListSessions()
	for _, session := range sessions {
		fmt.Printf("Session ID: %s\n", session.ID)
		fmt.Printf("Status: %s\n", session.Status)
		fmt.Printf("Participants: %d agent(s)\n", len(session.Participants))
		fmt.Printf("Subtasks: %d\n", len(session.SubTasks))
		fmt.Printf("Created: %v\n", session.CreatedAt.Format(time.RFC3339))
		fmt.Printf("Duration: %v\n", session.UpdatedAt.Sub(session.CreatedAt))
		
		fmt.Println("\nParticipating Agents:")
		for agentID := range session.Participants {
			fmt.Printf("  - %s\n", agentID)
		}
	}
	
	fmt.Println("\n=== Demo Completed ===")
}

// SpecializedAgent is a specialized agent implementation for demo.
type SpecializedAgent struct {
	name        string
	specialty   string
	description string
}

// Execute executes a task based on specialty.
func (a *SpecializedAgent) Execute(ctx context.Context, input string) (string, error) {
	// Simulate processing time based on specialty
	switch a.specialty {
	case "research":
		time.Sleep(200 * time.Millisecond)
		return fmt.Sprintf("[%s] Completed research on: '%s'. Key findings: AI developments in 2026 show significant progress in multi-agent systems, improved reasoning capabilities, and better efficiency. Notable trends include increased adoption of agent protocols like MCP and A2A.", a.name, input), nil
		
	case "analysis":
		time.Sleep(300 * time.Millisecond)
		return fmt.Sprintf("[%s] Analyzed: '%s'. Statistical analysis reveals: 45%% growth in AI agent deployments, 78%% improvement in task completion rates, strong correlation between agent collaboration and output quality. Recommendation: Focus on standardized protocols for interoperability.", a.name, input), nil
		
	case "writing":
		time.Sleep(150 * time.Millisecond)
		return fmt.Sprintf("[%s] Written report for: '%s'.\n\n# AI Trends Report 2026\n\n## Executive Summary\nBased on comprehensive research and data analysis, AI development in 2026 shows remarkable progress in agent systems and collaborative AI.\n\n## Key Findings\n1. Multi-agent systems adoption increased by 45%%\n2. Standardized protocols (MCP, A2A) gaining traction\n3. Improved reasoning and efficiency across all models\n\n## Recommendations\nOrganizations should invest in agent protocol standardization to ensure future interoperability.", a.name, input), nil
		
	default:
		time.Sleep(100 * time.Millisecond)
		return fmt.Sprintf("[%s] Processed: '%s'", a.name, input), nil
	}
}

// GetName returns the agent name.
func (a *SpecializedAgent) GetName() string {
	return a.name
}

// GetDescription returns the agent description.
func (a *SpecializedAgent) GetDescription() string {
	return a.description
}
