package mcp

import (
	"context"
	"testing"
)

func TestNewServer(t *testing.T) {
	config := ServerConfig{
		Name:    "test-server",
		Version: "1.0.0",
		Vendor:  "test-vendor",
		Debug:   true,
	}
	
	server := NewServer(config)
	
	if server == nil {
		t.Fatal("NewServer() returned nil")
	}
	
	if server.config.Name != "test-server" {
		t.Errorf("Server name = %v, want test-server", server.config.Name)
	}
	
	// Check default capabilities
	if server.config.Capabilities.Resources == nil {
		t.Error("Expected Resources capability to be set")
	}
	
	if server.config.Capabilities.Tools == nil {
		t.Error("Expected Tools capability to be set")
	}
}

func TestServer_RegisterResource(t *testing.T) {
	server := NewServer(ServerConfig{Name: "test"})
	
	resource := &Resource{
		URI:  "test://resource",
		Name: "Test Resource",
	}
	
	provider := &mockResourceProvider{}
	
	err := server.RegisterResource(resource, provider)
	if err != nil {
		t.Fatalf("RegisterResource() error = %v", err)
	}
	
	// Try to list resources
	ctx := context.Background()
	resources, err := server.ListResources(ctx)
	if err != nil {
		t.Fatalf("ListResources() error = %v", err)
	}
	
	if len(resources) != 1 {
		t.Errorf("Expected 1 resource, got %d", len(resources))
	}
	
	if resources[0].URI != "test://resource" {
		t.Errorf("Resource URI = %v, want test://resource", resources[0].URI)
	}
}

func TestServer_RegisterResource_Validation(t *testing.T) {
	server := NewServer(ServerConfig{Name: "test"})
	provider := &mockResourceProvider{}
	
	tests := []struct {
		name     string
		resource *Resource
		provider ResourceProvider
		wantErr  bool
	}{
		{
			name:     "Nil resource",
			resource: nil,
			provider: provider,
			wantErr:  true,
		},
		{
			name: "Empty URI",
			resource: &Resource{
				URI:  "",
				Name: "Test",
			},
			provider: provider,
			wantErr:  true,
		},
		{
			name: "Nil provider",
			resource: &Resource{
				URI:  "test://resource",
				Name: "Test",
			},
			provider: nil,
			wantErr:  true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := server.RegisterResource(tt.resource, tt.provider)
			if (err != nil) != tt.wantErr {
				t.Errorf("RegisterResource() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestServer_RegisterTool(t *testing.T) {
	server := NewServer(ServerConfig{Name: "test"})
	
	tool := &Tool{
		Name:        "calculator",
		Description: "A calculator",
		InputSchema: map[string]any{"type": "object"},
	}
	
	handler := func(ctx context.Context, args map[string]any) (*ToolResult, error) {
		return &ToolResult{
			Content: []ContentBlock{{Type: "text", Text: "42"}},
		}, nil
	}
	
	err := server.RegisterTool(tool, handler)
	if err != nil {
		t.Fatalf("RegisterTool() error = %v", err)
	}
	
	// List tools
	ctx := context.Background()
	tools, err := server.ListTools(ctx)
	if err != nil {
		t.Fatalf("ListTools() error = %v", err)
	}
	
	if len(tools) != 1 {
		t.Errorf("Expected 1 tool, got %d", len(tools))
	}
	
	if tools[0].Name != "calculator" {
		t.Errorf("Tool name = %v, want calculator", tools[0].Name)
	}
}

func TestServer_CallTool(t *testing.T) {
	server := NewServer(ServerConfig{Name: "test"})
	
	tool := &Tool{
		Name:        "test-tool",
		Description: "A test tool",
	}
	
	handler := func(ctx context.Context, args map[string]any) (*ToolResult, error) {
		input := args["input"].(string)
		return &ToolResult{
			Content: []ContentBlock{
				{Type: "text", Text: "Processed: " + input},
			},
		}, nil
	}
	
	server.RegisterTool(tool, handler)
	
	ctx := context.Background()
	result, err := server.CallTool(ctx, "test-tool", map[string]any{
		"input": "test data",
	})
	
	if err != nil {
		t.Fatalf("CallTool() error = %v", err)
	}
	
	if len(result.Content) != 1 {
		t.Fatalf("Expected 1 content block, got %d", len(result.Content))
	}
	
	if result.Content[0].Text != "Processed: test data" {
		t.Errorf("Result text = %v, want Processed: test data", result.Content[0].Text)
	}
}

func TestServer_CallTool_NotFound(t *testing.T) {
	server := NewServer(ServerConfig{Name: "test"})
	ctx := context.Background()
	
	_, err := server.CallTool(ctx, "nonexistent", nil)
	if err == nil {
		t.Error("Expected error for nonexistent tool")
	}
	
	mcpErr, ok := err.(*MCPError)
	if !ok {
		t.Fatalf("Expected MCPError, got %T", err)
	}
	
	if mcpErr.Code != ErrToolNotFound.Code {
		t.Errorf("Error code = %v, want %v", mcpErr.Code, ErrToolNotFound.Code)
	}
}

func TestServer_RegisterPrompt(t *testing.T) {
	server := NewServer(ServerConfig{Name: "test"})
	
	prompt := &Prompt{
		Name:        "greet",
		Description: "Greeting prompt",
		Arguments: []PromptArgument{
			{Name: "name", Required: true},
		},
	}
	
	handler := func(ctx context.Context, args map[string]any) (*PromptResult, error) {
		name := args["name"].(string)
		return &PromptResult{
			Messages: []Message{
				{
					Role: "user",
					Content: ContentBlock{
						Type: "text",
						Text: "Greet " + name,
					},
				},
			},
		}, nil
	}
	
	err := server.RegisterPrompt(prompt, handler)
	if err != nil {
		t.Fatalf("RegisterPrompt() error = %v", err)
	}
	
	// List prompts
	ctx := context.Background()
	prompts, err := server.ListPrompts(ctx)
	if err != nil {
		t.Fatalf("ListPrompts() error = %v", err)
	}
	
	if len(prompts) != 1 {
		t.Errorf("Expected 1 prompt, got %d", len(prompts))
	}
}

func TestServer_Initialize(t *testing.T) {
	server := NewServer(ServerConfig{
		Name:    "test-server",
		Version: "1.0.0",
	})
	
	clientInfo := &ClientInfo{
		Name:    "test-client",
		Version: "0.1.0",
	}
	
	ctx := context.Background()
	serverInfo, err := server.Initialize(ctx, clientInfo)
	if err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}
	
	if serverInfo.Name != "test-server" {
		t.Errorf("ServerInfo.Name = %v, want test-server", serverInfo.Name)
	}
	
	if serverInfo.Version != "1.0.0" {
		t.Errorf("ServerInfo.Version = %v, want 1.0.0", serverInfo.Version)
	}
}

func TestServer_ReadResource(t *testing.T) {
	server := NewServer(ServerConfig{Name: "test"})
	
	provider := &mockResourceProvider{
		readFunc: func(ctx context.Context, uri string) (*ResourceContent, error) {
			return &ResourceContent{
				URI:      uri,
				MimeType: "text/plain",
				Text:     "test content",
			}, nil
		},
	}
	
	resource := &Resource{
		URI:  "test://file",
		Name: "Test File",
	}
	
	server.RegisterResource(resource, provider)
	
	ctx := context.Background()
	content, err := server.ReadResource(ctx, "test://file")
	if err != nil {
		t.Fatalf("ReadResource() error = %v", err)
	}
	
	if content.Text != "test content" {
		t.Errorf("Content.Text = %v, want test content", content.Text)
	}
}

func TestServer_Ping(t *testing.T) {
	server := NewServer(ServerConfig{Name: "test"})
	ctx := context.Background()
	
	err := server.Ping(ctx)
	if err != nil {
		t.Errorf("Ping() error = %v", err)
	}
}

func TestServer_SetLogLevel(t *testing.T) {
	server := NewServer(ServerConfig{Name: "test"})
	ctx := context.Background()
	
	err := server.SetLogLevel(ctx, LogLevelDebug)
	if err != nil {
		t.Errorf("SetLogLevel() error = %v", err)
	}
	
	// Verify log level was set
	if server.logLevel != LogLevelDebug {
		t.Errorf("logLevel = %v, want %v", server.logLevel, LogLevelDebug)
	}
}

// Mock implementations

type mockResourceProvider struct {
	readFunc func(ctx context.Context, uri string) (*ResourceContent, error)
}

func (m *mockResourceProvider) Read(ctx context.Context, uri string) (*ResourceContent, error) {
	if m.readFunc != nil {
		return m.readFunc(ctx, uri)
	}
	return &ResourceContent{
		URI:      uri,
		MimeType: "text/plain",
		Text:     "mock content",
	}, nil
}

func (m *mockResourceProvider) Subscribe(ctx context.Context, uri string) (<-chan *ResourceContent, error) {
	return nil, nil
}

func (m *mockResourceProvider) Unsubscribe(ctx context.Context, uri string) error {
	return nil
}
