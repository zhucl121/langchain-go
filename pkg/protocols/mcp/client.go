package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
)

// DefaultMCPClient is the default implementation of MCPClient.
type DefaultMCPClient struct {
	config    ClientConfig
	transport Transport
	
	requestID atomic.Uint64
	pending   sync.Map // requestID -> chan *JSONRPCMessage
	
	// Event handlers
	resourceHandlers []ResourceUpdateHandler
	progressHandlers []ProgressUpdateHandler
	
	// Server info
	serverInfo   *ServerInfo
	initialized  bool
	
	mu sync.RWMutex
}

// ClientConfig configures an MCP client.
type ClientConfig struct {
	Name    string
	Version string
	Timeout int // Timeout in seconds (0 = no timeout)
	Debug   bool
}

// NewClient creates a new MCP client.
func NewClient(config ClientConfig) *DefaultMCPClient {
	return &DefaultMCPClient{
		config:           config,
		resourceHandlers: make([]ResourceUpdateHandler, 0),
		progressHandlers: make([]ProgressUpdateHandler, 0),
	}
}

// Connect connects to an MCP server.
// Supported URI schemes:
//   - stdio://path/to/server
//   - http://host:port/path (SSE)
//   - ws://host:port/path (WebSocket)
func (c *DefaultMCPClient) Connect(ctx context.Context, uri string) error {
	// Create transport based on URI scheme
	transport, err := c.createTransport(uri)
	if err != nil {
		return fmt.Errorf("create transport: %w", err)
	}
	
	c.mu.Lock()
	c.transport = transport
	c.mu.Unlock()
	
	// Initialize connection
	if err := c.initialize(ctx); err != nil {
		return fmt.Errorf("initialize: %w", err)
	}
	
	// Start receive loop
	go c.receiveLoop(ctx)
	
	if c.config.Debug {
		log.Printf("[MCP Client] Connected to server: %s %s", c.serverInfo.Name, c.serverInfo.Version)
	}
	
	return nil
}

// Disconnect disconnects from the server.
func (c *DefaultMCPClient) Disconnect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if c.transport != nil {
		return c.transport.Close()
	}
	
	return nil
}

// ListResources lists all resources from the server.
func (c *DefaultMCPClient) ListResources(ctx context.Context) ([]*Resource, error) {
	req, err := NewRequest(c.nextRequestID(), "resources/list", nil)
	if err != nil {
		return nil, err
	}
	
	resp, err := c.sendRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	
	var result struct {
		Resources []*Resource `json:"resources"`
	}
	if err := resp.ParseResult(&result); err != nil {
		return nil, fmt.Errorf("parse result: %w", err)
	}
	
	return result.Resources, nil
}

// ReadResource reads a resource from the server.
func (c *DefaultMCPClient) ReadResource(ctx context.Context, uri string) (*ResourceContent, error) {
	req, err := NewRequest(c.nextRequestID(), "resources/read", map[string]any{
		"uri": uri,
	})
	if err != nil {
		return nil, err
	}
	
	resp, err := c.sendRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	
	var result struct {
		Contents []*ResourceContent `json:"contents"`
	}
	if err := resp.ParseResult(&result); err != nil {
		return nil, fmt.Errorf("parse result: %w", err)
	}
	
	if len(result.Contents) == 0 {
		return nil, fmt.Errorf("no content returned")
	}
	
	return result.Contents[0], nil
}

// SubscribeResource subscribes to resource updates.
func (c *DefaultMCPClient) SubscribeResource(ctx context.Context, uri string) error {
	req, err := NewRequest(c.nextRequestID(), "resources/subscribe", map[string]any{
		"uri": uri,
	})
	if err != nil {
		return err
	}
	
	_, err = c.sendRequest(ctx, req)
	return err
}

// UnsubscribeResource unsubscribes from resource updates.
func (c *DefaultMCPClient) UnsubscribeResource(ctx context.Context, uri string) error {
	req, err := NewRequest(c.nextRequestID(), "resources/unsubscribe", map[string]any{
		"uri": uri,
	})
	if err != nil {
		return err
	}
	
	_, err = c.sendRequest(ctx, req)
	return err
}

// ListTools lists all tools from the server.
func (c *DefaultMCPClient) ListTools(ctx context.Context) ([]*Tool, error) {
	req, err := NewRequest(c.nextRequestID(), "tools/list", nil)
	if err != nil {
		return nil, err
	}
	
	resp, err := c.sendRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	
	var result struct {
		Tools []*Tool `json:"tools"`
	}
	if err := resp.ParseResult(&result); err != nil {
		return nil, fmt.Errorf("parse result: %w", err)
	}
	
	return result.Tools, nil
}

// CallTool calls a tool on the server.
func (c *DefaultMCPClient) CallTool(ctx context.Context, name string, args map[string]any) (*ToolResult, error) {
	req, err := NewRequest(c.nextRequestID(), "tools/call", map[string]any{
		"name":      name,
		"arguments": args,
	})
	if err != nil {
		return nil, err
	}
	
	resp, err := c.sendRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	
	var result ToolResult
	if err := resp.ParseResult(&result); err != nil {
		return nil, fmt.Errorf("parse result: %w", err)
	}
	
	return &result, nil
}

// ListPrompts lists all prompts from the server.
func (c *DefaultMCPClient) ListPrompts(ctx context.Context) ([]*Prompt, error) {
	req, err := NewRequest(c.nextRequestID(), "prompts/list", nil)
	if err != nil {
		return nil, err
	}
	
	resp, err := c.sendRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	
	var result struct {
		Prompts []*Prompt `json:"prompts"`
	}
	if err := resp.ParseResult(&result); err != nil {
		return nil, fmt.Errorf("parse result: %w", err)
	}
	
	return result.Prompts, nil
}

// GetPrompt gets a prompt from the server.
func (c *DefaultMCPClient) GetPrompt(ctx context.Context, name string, args map[string]any) (*PromptResult, error) {
	req, err := NewRequest(c.nextRequestID(), "prompts/get", map[string]any{
		"name":      name,
		"arguments": args,
	})
	if err != nil {
		return nil, err
	}
	
	resp, err := c.sendRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	
	var result PromptResult
	if err := resp.ParseResult(&result); err != nil {
		return nil, fmt.Errorf("parse result: %w", err)
	}
	
	return &result, nil
}

// OnResourceUpdated registers a handler for resource update notifications.
func (c *DefaultMCPClient) OnResourceUpdated(handler ResourceUpdateHandler) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.resourceHandlers = append(c.resourceHandlers, handler)
}

// OnProgressUpdate registers a handler for progress update notifications.
func (c *DefaultMCPClient) OnProgressUpdate(handler ProgressUpdateHandler) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.progressHandlers = append(c.progressHandlers, handler)
}

// initialize initializes the connection with the server.
func (c *DefaultMCPClient) initialize(ctx context.Context) error {
	req, err := NewRequest(c.nextRequestID(), "initialize", map[string]any{
		"protocolVersion": "2024-11-05",
		"capabilities": map[string]any{
			"sampling": map[string]any{},
		},
		"clientInfo": map[string]any{
			"name":    c.config.Name,
			"version": c.config.Version,
		},
	})
	if err != nil {
		return err
	}
	
	resp, err := c.sendRequest(ctx, req)
	if err != nil {
		return err
	}
	
	var result struct {
		ProtocolVersion string      `json:"protocolVersion"`
		ServerInfo      *ServerInfo `json:"serverInfo"`
		Capabilities    any         `json:"capabilities"`
	}
	if err := resp.ParseResult(&result); err != nil {
		return fmt.Errorf("parse result: %w", err)
	}
	
	c.mu.Lock()
	c.serverInfo = result.ServerInfo
	c.initialized = true
	c.mu.Unlock()
	
	// Send initialized notification
	notification, _ := NewNotification("notifications/initialized", nil)
	notificationData, _ := notification.ToJSON()
	c.transport.Send(ctx, notificationData)
	
	return nil
}

// sendRequest sends a request and waits for response.
func (c *DefaultMCPClient) sendRequest(ctx context.Context, req *JSONRPCMessage) (*JSONRPCMessage, error) {
	// Create response channel
	respChan := make(chan *JSONRPCMessage, 1)
	c.pending.Store(req.ID, respChan)
	defer c.pending.Delete(req.ID)
	
	// Send request
	data, err := req.ToJSON()
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}
	
	if c.config.Debug {
		log.Printf("[MCP Client] Sending: %s (ID: %v)", req.Method, req.ID)
	}
	
	if err := c.transport.Send(ctx, data); err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	
	// Wait for response
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case resp := <-respChan:
		if resp.IsError() {
			return nil, resp.Error.ToMCPError()
		}
		return resp, nil
	}
}

// receiveLoop receives messages from the transport.
func (c *DefaultMCPClient) receiveLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		
		// Receive message
		data, err := c.transport.Receive(ctx)
		if err != nil {
			if err == context.Canceled {
				return
			}
			if c.config.Debug {
				log.Printf("[MCP Client] Receive error: %v", err)
			}
			continue
		}
		
		// Parse message
		msg, err := FromJSON(data)
		if err != nil {
			if c.config.Debug {
				log.Printf("[MCP Client] Parse error: %v", err)
			}
			continue
		}
		
		// Handle message
		c.handleMessage(ctx, msg)
	}
}

// handleMessage handles received messages.
func (c *DefaultMCPClient) handleMessage(ctx context.Context, msg *JSONRPCMessage) {
	if c.config.Debug {
		if msg.IsResponse() {
			log.Printf("[MCP Client] Received response (ID: %v)", msg.ID)
		} else if msg.IsNotification() {
			log.Printf("[MCP Client] Received notification: %s", msg.Method)
		}
	}
	
	// Handle response
	if msg.IsResponse() {
		if ch, ok := c.pending.Load(msg.ID); ok {
			respChan := ch.(chan *JSONRPCMessage)
			select {
			case respChan <- msg:
			default:
			}
		}
		return
	}
	
	// Handle notification
	if msg.IsNotification() {
		c.handleNotification(ctx, msg)
	}
}

// handleNotification handles notification messages.
func (c *DefaultMCPClient) handleNotification(ctx context.Context, msg *JSONRPCMessage) {
	switch msg.Method {
	case "notifications/resources/updated":
		var params struct {
			URI string `json:"uri"`
		}
		if err := msg.ParseParams(&params); err != nil {
			return
		}
		
		c.mu.RLock()
		handlers := c.resourceHandlers
		c.mu.RUnlock()
		
		for _, handler := range handlers {
			handler(params.URI)
		}
		
	case "notifications/progress":
		var params struct {
			Progress float64 `json:"progress"`
		}
		if err := msg.ParseParams(&params); err != nil {
			return
		}
		
		c.mu.RLock()
		handlers := c.progressHandlers
		c.mu.RUnlock()
		
		for _, handler := range handlers {
			handler(params.Progress)
		}
	}
}

// createTransport creates a transport based on URI scheme.
func (c *DefaultMCPClient) createTransport(uri string) (Transport, error) {
	if len(uri) < 8 {
		return nil, fmt.Errorf("invalid URI: %s", uri)
	}
	
	scheme := uri[:7]
	
	switch scheme {
	case "stdio:/":
		// stdio://path/to/server
		path := uri[8:]
		return NewStdioTransportWithCommand(path)
		
	default:
		return nil, fmt.Errorf("unsupported URI scheme: %s", scheme)
	}
}

// nextRequestID generates the next request ID.
func (c *DefaultMCPClient) nextRequestID() string {
	id := c.requestID.Add(1)
	return fmt.Sprintf("req-%d", id)
}

// NewStdioTransportWithCommand creates a Stdio transport by launching a command.
// This is defined here temporarily until we move it to the transport package.
func NewStdioTransportWithCommand(path string) (Transport, error) {
	// Import from transport package
	return nil, fmt.Errorf("not implemented - use transport.NewStdioTransportWithCommand")
}
