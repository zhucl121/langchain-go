package mcp

import (
	"context"
	"fmt"
	"log"
	"sync"
)

// DefaultMCPServer is the default implementation of MCPServer.
type DefaultMCPServer struct {
	config ServerConfig
	
	// Resources
	resources   map[string]*Resource
	providers   map[string]ResourceProvider
	subscribers map[string][]ResourceUpdateHandler
	resourcesMu sync.RWMutex
	
	// Tools
	tools   map[string]*ToolRegistration
	toolsMu sync.RWMutex
	
	// Prompts
	prompts   map[string]*PromptRegistration
	promptsMu sync.RWMutex
	
	// State
	initialized bool
	clientInfo  *ClientInfo
	transport   Transport
	logLevel    LogLevel
	
	mu sync.RWMutex
}

// ToolRegistration represents a registered tool.
type ToolRegistration struct {
	Tool    *Tool
	Handler ToolHandler
}

// PromptRegistration represents a registered prompt.
type PromptRegistration struct {
	Prompt  *Prompt
	Handler PromptHandler
}

// ServerConfig configures an MCP server.
type ServerConfig struct {
	Name        string
	Version     string
	Vendor      string
	Description string
	
	Capabilities ServerCapabilities
	
	// Debug mode
	Debug bool
}

// NewServer creates a new MCP server.
func NewServer(config ServerConfig) *DefaultMCPServer {
	// Set default capabilities
	if config.Capabilities.Resources == nil {
		config.Capabilities.Resources = &ResourceCapability{
			Subscribe: true,
		}
	}
	if config.Capabilities.Tools == nil {
		config.Capabilities.Tools = &ToolCapability{}
	}
	if config.Capabilities.Prompts == nil {
		config.Capabilities.Prompts = &PromptCapability{}
	}
	
	return &DefaultMCPServer{
		config:      config,
		resources:   make(map[string]*Resource),
		providers:   make(map[string]ResourceProvider),
		subscribers: make(map[string][]ResourceUpdateHandler),
		tools:       make(map[string]*ToolRegistration),
		prompts:     make(map[string]*PromptRegistration),
		logLevel:    LogLevelInfo,
	}
}

// RegisterResource registers a resource with its provider.
func (s *DefaultMCPServer) RegisterResource(resource *Resource, provider ResourceProvider) error {
	s.resourcesMu.Lock()
	defer s.resourcesMu.Unlock()
	
	if resource == nil {
		return fmt.Errorf("resource cannot be nil")
	}
	if resource.URI == "" {
		return fmt.Errorf("resource URI cannot be empty")
	}
	if provider == nil {
		return fmt.Errorf("provider cannot be nil")
	}
	
	s.resources[resource.URI] = resource
	s.providers[resource.URI] = provider
	
	if s.config.Debug {
		log.Printf("[MCP Server] Registered resource: %s (%s)", resource.Name, resource.URI)
	}
	
	return nil
}

// RegisterTool registers a tool with its handler.
func (s *DefaultMCPServer) RegisterTool(tool *Tool, handler ToolHandler) error {
	s.toolsMu.Lock()
	defer s.toolsMu.Unlock()
	
	if tool == nil {
		return fmt.Errorf("tool cannot be nil")
	}
	if tool.Name == "" {
		return fmt.Errorf("tool name cannot be empty")
	}
	if handler == nil {
		return fmt.Errorf("handler cannot be nil")
	}
	
	s.tools[tool.Name] = &ToolRegistration{
		Tool:    tool,
		Handler: handler,
	}
	
	if s.config.Debug {
		log.Printf("[MCP Server] Registered tool: %s", tool.Name)
	}
	
	return nil
}

// RegisterPrompt registers a prompt with its handler.
func (s *DefaultMCPServer) RegisterPrompt(prompt *Prompt, handler PromptHandler) error {
	s.promptsMu.Lock()
	defer s.promptsMu.Unlock()
	
	if prompt == nil {
		return fmt.Errorf("prompt cannot be nil")
	}
	if prompt.Name == "" {
		return fmt.Errorf("prompt name cannot be empty")
	}
	if handler == nil {
		return fmt.Errorf("handler cannot be nil")
	}
	
	s.prompts[prompt.Name] = &PromptRegistration{
		Prompt:  prompt,
		Handler: handler,
	}
	
	if s.config.Debug {
		log.Printf("[MCP Server] Registered prompt: %s", prompt.Name)
	}
	
	return nil
}

// ListResources lists all available resources.
func (s *DefaultMCPServer) ListResources(ctx context.Context) ([]*Resource, error) {
	s.resourcesMu.RLock()
	defer s.resourcesMu.RUnlock()
	
	resources := make([]*Resource, 0, len(s.resources))
	for _, res := range s.resources {
		resources = append(resources, res)
	}
	
	return resources, nil
}

// ReadResource reads a resource's content.
func (s *DefaultMCPServer) ReadResource(ctx context.Context, uri string) (*ResourceContent, error) {
	s.resourcesMu.RLock()
	provider, ok := s.providers[uri]
	s.resourcesMu.RUnlock()
	
	if !ok {
		return nil, ErrResourceNotFound.WithData("uri", uri)
	}
	
	content, err := provider.Read(ctx, uri)
	if err != nil {
		return nil, fmt.Errorf("read resource: %w", err)
	}
	
	return content, nil
}

// SubscribeResource subscribes to resource updates.
func (s *DefaultMCPServer) SubscribeResource(ctx context.Context, uri string, handler ResourceUpdateHandler) error {
	s.resourcesMu.Lock()
	defer s.resourcesMu.Unlock()
	
	if _, ok := s.resources[uri]; !ok {
		return ErrResourceNotFound.WithData("uri", uri)
	}
	
	s.subscribers[uri] = append(s.subscribers[uri], handler)
	
	if s.config.Debug {
		log.Printf("[MCP Server] Subscribed to resource: %s", uri)
	}
	
	return nil
}

// UnsubscribeResource unsubscribes from resource updates.
func (s *DefaultMCPServer) UnsubscribeResource(ctx context.Context, uri string) error {
	s.resourcesMu.Lock()
	defer s.resourcesMu.Unlock()
	
	delete(s.subscribers, uri)
	
	if s.config.Debug {
		log.Printf("[MCP Server] Unsubscribed from resource: %s", uri)
	}
	
	return nil
}

// ListTools lists all available tools.
func (s *DefaultMCPServer) ListTools(ctx context.Context) ([]*Tool, error) {
	s.toolsMu.RLock()
	defer s.toolsMu.RUnlock()
	
	tools := make([]*Tool, 0, len(s.tools))
	for _, reg := range s.tools {
		tools = append(tools, reg.Tool)
	}
	
	return tools, nil
}

// CallTool calls a tool with the given arguments.
func (s *DefaultMCPServer) CallTool(ctx context.Context, name string, args map[string]any) (*ToolResult, error) {
	s.toolsMu.RLock()
	reg, ok := s.tools[name]
	s.toolsMu.RUnlock()
	
	if !ok {
		return nil, ErrToolNotFound.WithData("name", name)
	}
	
	result, err := reg.Handler(ctx, args)
	if err != nil {
		return &ToolResult{
			Content: []ContentBlock{
				{
					Type: "text",
					Text: err.Error(),
				},
			},
			IsError: true,
		}, nil
	}
	
	return result, nil
}

// ListPrompts lists all available prompts.
func (s *DefaultMCPServer) ListPrompts(ctx context.Context) ([]*Prompt, error) {
	s.promptsMu.RLock()
	defer s.promptsMu.RUnlock()
	
	prompts := make([]*Prompt, 0, len(s.prompts))
	for _, reg := range s.prompts {
		prompts = append(prompts, reg.Prompt)
	}
	
	return prompts, nil
}

// GetPrompt gets a prompt with the given arguments.
func (s *DefaultMCPServer) GetPrompt(ctx context.Context, name string, args map[string]any) (*PromptResult, error) {
	s.promptsMu.RLock()
	reg, ok := s.prompts[name]
	s.promptsMu.RUnlock()
	
	if !ok {
		return nil, ErrPromptNotFound.WithData("name", name)
	}
	
	result, err := reg.Handler(ctx, args)
	if err != nil {
		return nil, fmt.Errorf("get prompt: %w", err)
	}
	
	return result, nil
}

// CreateMessage creates a message (sampling - not implemented in basic version).
func (s *DefaultMCPServer) CreateMessage(ctx context.Context, req *MessageRequest) (*MessageResponse, error) {
	return nil, fmt.Errorf("sampling not implemented")
}

// Initialize initializes the server with client info.
func (s *DefaultMCPServer) Initialize(ctx context.Context, info *ClientInfo) (*ServerInfo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.initialized = true
	s.clientInfo = info
	
	if s.config.Debug {
		log.Printf("[MCP Server] Initialized with client: %s %s", info.Name, info.Version)
	}
	
	return &ServerInfo{
		Name:         s.config.Name,
		Version:      s.config.Version,
		Vendor:       s.config.Vendor,
		Capabilities: s.config.Capabilities,
	}, nil
}

// Ping responds to ping requests.
func (s *DefaultMCPServer) Ping(ctx context.Context) error {
	return nil
}

// SetLogLevel sets the log level.
func (s *DefaultMCPServer) SetLogLevel(ctx context.Context, level LogLevel) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.logLevel = level
	
	if s.config.Debug {
		log.Printf("[MCP Server] Log level set to: %s", level)
	}
	
	return nil
}

// Serve starts the server with the given transport.
func (s *DefaultMCPServer) Serve(ctx context.Context, transport Transport) error {
	s.mu.Lock()
	s.transport = transport
	s.mu.Unlock()
	
	if s.config.Debug {
		log.Printf("[MCP Server] Starting server: %s %s", s.config.Name, s.config.Version)
	}
	
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		
		// Receive message
		data, err := transport.Receive(ctx)
		if err != nil {
			if err == context.Canceled {
				return nil
			}
			return fmt.Errorf("receive message: %w", err)
		}
		
		// Parse JSON-RPC message
		msg, err := FromJSON(data)
		if err != nil {
			if s.config.Debug {
				log.Printf("[MCP Server] Parse error: %v", err)
			}
			continue
		}
		
		// Handle message
		response := s.handleMessage(ctx, msg)
		if response != nil {
			responseData, err := response.ToJSON()
			if err != nil {
				if s.config.Debug {
					log.Printf("[MCP Server] Failed to marshal response: %v", err)
				}
				continue
			}
			
			if err := transport.Send(ctx, responseData); err != nil {
				return fmt.Errorf("send response: %w", err)
			}
		}
	}
}

// handleMessage handles a JSON-RPC message and returns a response.
func (s *DefaultMCPServer) handleMessage(ctx context.Context, msg *JSONRPCMessage) *JSONRPCMessage {
	if s.config.Debug {
		log.Printf("[MCP Server] Received: %s (ID: %v)", msg.Method, msg.ID)
	}
	
	// Handle request
	if msg.IsRequest() {
		return s.handleRequest(ctx, msg)
	}
	
	// Handle notification
	if msg.IsNotification() {
		s.handleNotification(ctx, msg)
		return nil
	}
	
	return nil
}

// handleRequest handles a JSON-RPC request.
func (s *DefaultMCPServer) handleRequest(ctx context.Context, msg *JSONRPCMessage) *JSONRPCMessage {
	switch msg.Method {
	case "initialize":
		var params struct {
			ProtocolVersion string      `json:"protocolVersion"`
			Capabilities    any         `json:"capabilities"`
			ClientInfo      *ClientInfo `json:"clientInfo"`
		}
		if err := msg.ParseParams(&params); err != nil {
			return NewErrorResponse(msg.ID, ErrCodeInvalidParams, "Invalid params", nil)
		}
		
		serverInfo, err := s.Initialize(ctx, params.ClientInfo)
		if err != nil {
			return NewErrorResponse(msg.ID, ErrCodeInternalError, err.Error(), nil)
		}
		
		response, _ := NewResponse(msg.ID, map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities":    serverInfo.Capabilities,
			"serverInfo":      serverInfo,
		})
		return response
		
	case "ping":
		if err := s.Ping(ctx); err != nil {
			return NewErrorResponse(msg.ID, ErrCodeInternalError, err.Error(), nil)
		}
		response, _ := NewResponse(msg.ID, map[string]any{})
		return response
		
	case "resources/list":
		resources, err := s.ListResources(ctx)
		if err != nil {
			return NewErrorResponse(msg.ID, ErrCodeInternalError, err.Error(), nil)
		}
		response, _ := NewResponse(msg.ID, map[string]any{"resources": resources})
		return response
		
	case "resources/read":
		var params struct {
			URI string `json:"uri"`
		}
		if err := msg.ParseParams(&params); err != nil {
			return NewErrorResponse(msg.ID, ErrCodeInvalidParams, "Invalid params", nil)
		}
		
		content, err := s.ReadResource(ctx, params.URI)
		if err != nil {
			if mcpErr, ok := err.(*MCPError); ok {
				return NewErrorResponse(msg.ID, mcpErr.Code, mcpErr.Message, mcpErr.Data)
			}
			return NewErrorResponse(msg.ID, ErrCodeInternalError, err.Error(), nil)
		}
		
		response, _ := NewResponse(msg.ID, map[string]any{
			"contents": []*ResourceContent{content},
		})
		return response
		
	case "tools/list":
		tools, err := s.ListTools(ctx)
		if err != nil {
			return NewErrorResponse(msg.ID, ErrCodeInternalError, err.Error(), nil)
		}
		response, _ := NewResponse(msg.ID, map[string]any{"tools": tools})
		return response
		
	case "tools/call":
		var params struct {
			Name      string         `json:"name"`
			Arguments map[string]any `json:"arguments"`
		}
		if err := msg.ParseParams(&params); err != nil {
			return NewErrorResponse(msg.ID, ErrCodeInvalidParams, "Invalid params", nil)
		}
		
		result, err := s.CallTool(ctx, params.Name, params.Arguments)
		if err != nil {
			if mcpErr, ok := err.(*MCPError); ok {
				return NewErrorResponse(msg.ID, mcpErr.Code, mcpErr.Message, mcpErr.Data)
			}
			return NewErrorResponse(msg.ID, ErrCodeInternalError, err.Error(), nil)
		}
		
		response, _ := NewResponse(msg.ID, result)
		return response
		
	case "prompts/list":
		prompts, err := s.ListPrompts(ctx)
		if err != nil {
			return NewErrorResponse(msg.ID, ErrCodeInternalError, err.Error(), nil)
		}
		response, _ := NewResponse(msg.ID, map[string]any{"prompts": prompts})
		return response
		
	case "prompts/get":
		var params struct {
			Name      string         `json:"name"`
			Arguments map[string]any `json:"arguments"`
		}
		if err := msg.ParseParams(&params); err != nil {
			return NewErrorResponse(msg.ID, ErrCodeInvalidParams, "Invalid params", nil)
		}
		
		result, err := s.GetPrompt(ctx, params.Name, params.Arguments)
		if err != nil {
			if mcpErr, ok := err.(*MCPError); ok {
				return NewErrorResponse(msg.ID, mcpErr.Code, mcpErr.Message, mcpErr.Data)
			}
			return NewErrorResponse(msg.ID, ErrCodeInternalError, err.Error(), nil)
		}
		
		response, _ := NewResponse(msg.ID, result)
		return response
		
	default:
		return NewErrorResponse(msg.ID, ErrCodeMethodNotFound, "Method not found", map[string]any{
			"method": msg.Method,
		})
	}
}

// handleNotification handles a JSON-RPC notification.
func (s *DefaultMCPServer) handleNotification(ctx context.Context, msg *JSONRPCMessage) {
	switch msg.Method {
	case "notifications/initialized":
		if s.config.Debug {
			log.Printf("[MCP Server] Client initialized")
		}
		
	default:
		if s.config.Debug {
			log.Printf("[MCP Server] Unknown notification: %s", msg.Method)
		}
	}
}
