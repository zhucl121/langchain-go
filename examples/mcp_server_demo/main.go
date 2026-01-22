// MCP Server Demo
//
// This example demonstrates how to create a simple MCP server that:
// - Provides filesystem resources
// - Exposes calculator tool
// - Uses Stdio transport (compatible with Claude Desktop)
//
// Usage:
//   go run main.go
//
// To use with Claude Desktop, add to claude_desktop_config.json:
//   {
//     "mcpServers": {
//       "demo": {
//         "command": "/path/to/mcp_server_demo"
//       }
//     }
//   }
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	
	"github.com/zhucl121/langchain-go/pkg/protocols/mcp"
	"github.com/zhucl121/langchain-go/pkg/protocols/mcp/providers"
	"github.com/zhucl121/langchain-go/pkg/protocols/mcp/transport"
)

func main() {
	// Create MCP server
	server := mcp.NewServer(mcp.ServerConfig{
		Name:        "demo-server",
		Version:     "0.1.0",
		Vendor:      "LangChain-Go",
		Description: "Demo MCP Server",
		Debug:       true, // Enable debug logging
	})
	
	// Register filesystem resource
	homeDir, _ := os.UserHomeDir()
	docsPath := filepath.Join(homeDir, "Documents")
	
	fsProvider := providers.NewFileSystemProvider(docsPath)
	if err := server.RegisterResource(&mcp.Resource{
		URI:         "file:///documents",
		Name:        "Documents",
		Description: "User documents directory",
		MimeType:    "text/plain",
	}, fsProvider); err != nil {
		log.Fatalf("Failed to register resource: %v", err)
	}
	
	// Register calculator tool
	calculatorTool := &mcp.Tool{
		Name:        "calculator",
		Description: "Perform basic mathematical calculations",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"expression": map[string]any{
					"type":        "string",
					"description": "Mathematical expression (e.g., '2 + 2', '10 * 5')",
				},
			},
			"required": []string{"expression"},
		},
	}
	
	if err := server.RegisterTool(calculatorTool, calculatorHandler); err != nil {
		log.Fatalf("Failed to register tool: %v", err)
	}
	
	// Register simple prompt
	greetPrompt := &mcp.Prompt{
		Name:        "greet",
		Description: "Generate a greeting message",
		Arguments: []mcp.PromptArgument{
			{
				Name:        "name",
				Description: "Name of the person to greet",
				Required:    true,
			},
		},
	}
	
	if err := server.RegisterPrompt(greetPrompt, greetPromptHandler); err != nil {
		log.Fatalf("Failed to register prompt: %v", err)
	}
	
	log.Println("MCP Server starting...")
	log.Printf("Server: %s v%s", "demo-server", "0.1.0")
	log.Printf("Resources: file:///documents -> %s", docsPath)
	log.Println("Tools: calculator")
	log.Println("Prompts: greet")
	log.Println("Listening on stdin/stdout...")
	
	// Create Stdio transport
	mcpTransport := transport.NewStdioTransport()
	
	// Handle graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	
	go func() {
		<-sigChan
		log.Println("\nShutting down...")
		cancel()
	}()
	
	// Start server
	if err := server.Serve(ctx, mcpTransport); err != nil && err != context.Canceled {
		log.Fatalf("Server error: %v", err)
	}
	
	log.Println("Server stopped")
}

// calculatorHandler handles calculator tool calls.
func calculatorHandler(ctx context.Context, args map[string]any) (*mcp.ToolResult, error) {
	expression, ok := args["expression"].(string)
	if !ok {
		return &mcp.ToolResult{
			Content: []mcp.ContentBlock{
				{
					Type: "text",
					Text: "Error: 'expression' must be a string",
				},
			},
			IsError: true,
		}, nil
	}
	
	// Simple calculator (just for demo)
	result, err := calculateExpression(expression)
	if err != nil {
		return &mcp.ToolResult{
			Content: []mcp.ContentBlock{
				{
					Type: "text",
					Text: fmt.Sprintf("Error: %v", err),
				},
			},
			IsError: true,
		}, nil
	}
	
	return &mcp.ToolResult{
		Content: []mcp.ContentBlock{
			{
				Type: "text",
				Text: fmt.Sprintf("Result: %s = %s", expression, result),
			},
		},
	}, nil
}

// greetPromptHandler handles greet prompt requests.
func greetPromptHandler(ctx context.Context, args map[string]any) (*mcp.PromptResult, error) {
	name, ok := args["name"].(string)
	if !ok {
		return nil, fmt.Errorf("'name' argument is required")
	}
	
	promptText := fmt.Sprintf("Please generate a friendly greeting message for %s. Be warm and welcoming.", name)
	
	return &mcp.PromptResult{
		Description: "Greeting prompt",
		Messages: []mcp.Message{
			{
				Role: "user",
				Content: mcp.ContentBlock{
					Type: "text",
					Text: promptText,
				},
			},
		},
	}, nil
}

// calculateExpression performs simple calculation (demo only).
func calculateExpression(expr string) (string, error) {
	// This is a very simple calculator for demo purposes
	// In production, use a proper expression parser
	
	// Try to parse as float
	if result, err := strconv.ParseFloat(expr, 64); err == nil {
		return fmt.Sprintf("%.2f", result), nil
	}
	
	return "", fmt.Errorf("unsupported expression (demo calculator only supports simple numbers)")
}
