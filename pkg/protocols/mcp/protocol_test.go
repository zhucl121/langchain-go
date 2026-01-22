package mcp

import (
	"testing"
)

func TestMCPError(t *testing.T) {
	tests := []struct {
		name     string
		code     int
		message  string
		wantErr  string
		wantCode int
	}{
		{
			name:     "Resource not found",
			code:     -32000,
			message:  "Resource not found",
			wantErr:  "Resource not found",
			wantCode: -32000,
		},
		{
			name:     "Tool execution failed",
			code:     -32003,
			message:  "Tool execution failed",
			wantErr:  "Tool execution failed",
			wantCode: -32003,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewMCPError(tt.code, tt.message)
			
			if err.Error() != tt.wantErr {
				t.Errorf("MCPError.Error() = %v, want %v", err.Error(), tt.wantErr)
			}
			
			if err.Code != tt.wantCode {
				t.Errorf("MCPError.Code = %v, want %v", err.Code, tt.wantCode)
			}
		})
	}
}

func TestMCPError_WithData(t *testing.T) {
	err := NewMCPError(-32000, "Test error")
	err.WithData("key1", "value1")
	err.WithData("key2", 123)
	
	if len(err.Data) != 2 {
		t.Errorf("Expected 2 data entries, got %d", len(err.Data))
	}
	
	if err.Data["key1"] != "value1" {
		t.Errorf("Expected key1 = value1, got %v", err.Data["key1"])
	}
	
	if err.Data["key2"] != 123 {
		t.Errorf("Expected key2 = 123, got %v", err.Data["key2"])
	}
}

func TestResource(t *testing.T) {
	resource := &Resource{
		URI:         "file:///test.txt",
		Name:        "Test Resource",
		Description: "A test resource",
		MimeType:    "text/plain",
		Metadata: map[string]any{
			"size": 1024,
		},
	}
	
	if resource.URI != "file:///test.txt" {
		t.Errorf("Expected URI file:///test.txt, got %s", resource.URI)
	}
	
	if resource.Name != "Test Resource" {
		t.Errorf("Expected Name Test Resource, got %s", resource.Name)
	}
	
	if resource.Metadata["size"] != 1024 {
		t.Errorf("Expected size 1024, got %v", resource.Metadata["size"])
	}
}

func TestTool(t *testing.T) {
	tool := &Tool{
		Name:        "calculator",
		Description: "A calculator tool",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"expression": map[string]any{
					"type": "string",
				},
			},
		},
	}
	
	if tool.Name != "calculator" {
		t.Errorf("Expected Name calculator, got %s", tool.Name)
	}
	
	schema, ok := tool.InputSchema["type"]
	if !ok || schema != "object" {
		t.Errorf("Expected InputSchema type object, got %v", schema)
	}
}

func TestContentBlock(t *testing.T) {
	tests := []struct {
		name     string
		block    ContentBlock
		wantType string
	}{
		{
			name: "Text content",
			block: ContentBlock{
				Type: "text",
				Text: "Hello world",
			},
			wantType: "text",
		},
		{
			name: "Image content",
			block: ContentBlock{
				Type:     "image",
				Data:     "base64data...",
				MimeType: "image/png",
			},
			wantType: "image",
		},
		{
			name: "Resource reference",
			block: ContentBlock{
				Type: "resource",
				Resource: &Resource{
					URI:  "file:///doc.txt",
					Name: "Document",
				},
			},
			wantType: "resource",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.block.Type != tt.wantType {
				t.Errorf("ContentBlock.Type = %v, want %v", tt.block.Type, tt.wantType)
			}
		})
	}
}

func TestToolResult(t *testing.T) {
	t.Run("Success result", func(t *testing.T) {
		result := &ToolResult{
			Content: []ContentBlock{
				{
					Type: "text",
					Text: "Result: 42",
				},
			},
			IsError: false,
		}
		
		if result.IsError {
			t.Error("Expected IsError to be false")
		}
		
		if len(result.Content) != 1 {
			t.Errorf("Expected 1 content block, got %d", len(result.Content))
		}
	})
	
	t.Run("Error result", func(t *testing.T) {
		result := &ToolResult{
			Content: []ContentBlock{
				{
					Type: "text",
					Text: "Error: Division by zero",
				},
			},
			IsError: true,
		}
		
		if !result.IsError {
			t.Error("Expected IsError to be true")
		}
	})
}

func TestServerCapabilities(t *testing.T) {
	caps := ServerCapabilities{
		Resources: &ResourceCapability{
			Subscribe:   true,
			ListChanged: true,
		},
		Tools: &ToolCapability{
			ListChanged: false,
		},
		Prompts: &PromptCapability{
			ListChanged: true,
		},
	}
	
	if !caps.Resources.Subscribe {
		t.Error("Expected Resources.Subscribe to be true")
	}
	
	if caps.Tools.ListChanged {
		t.Error("Expected Tools.ListChanged to be false")
	}
	
	if !caps.Prompts.ListChanged {
		t.Error("Expected Prompts.ListChanged to be true")
	}
}

func TestLogLevel(t *testing.T) {
	tests := []struct {
		name  string
		level LogLevel
		want  string
	}{
		{"Debug", LogLevelDebug, "debug"},
		{"Info", LogLevelInfo, "info"},
		{"Warning", LogLevelWarning, "warning"},
		{"Error", LogLevelError, "error"},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.level) != tt.want {
				t.Errorf("LogLevel = %v, want %v", tt.level, tt.want)
			}
		})
	}
}

func TestPromptArgument(t *testing.T) {
	arg := PromptArgument{
		Name:        "username",
		Description: "The username to greet",
		Required:    true,
	}
	
	if arg.Name != "username" {
		t.Errorf("Expected Name username, got %s", arg.Name)
	}
	
	if !arg.Required {
		t.Error("Expected Required to be true")
	}
}

func TestMessageRequest(t *testing.T) {
	req := &MessageRequest{
		Messages: []Message{
			{
				Role: "user",
				Content: ContentBlock{
					Type: "text",
					Text: "Hello",
				},
			},
		},
		SystemPrompt: "You are a helpful assistant",
		MaxTokens:    1000,
		Temperature:  0.7,
	}
	
	if len(req.Messages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(req.Messages))
	}
	
	if req.Temperature != 0.7 {
		t.Errorf("Expected Temperature 0.7, got %f", req.Temperature)
	}
	
	if req.MaxTokens != 1000 {
		t.Errorf("Expected MaxTokens 1000, got %d", req.MaxTokens)
	}
}
