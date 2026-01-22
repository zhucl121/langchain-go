// Package mcp implements the Model Context Protocol (MCP) specification.
//
// MCP is an open standard protocol proposed by Anthropic for standardized
// communication between AI models and external tools/data sources.
//
// Core concepts:
//   - Server: Provides resources, tools, and prompts
//   - Client: Uses these capabilities (e.g., Claude Desktop)
//   - Resources: Data sources that AI can access
//   - Tools: Functions that AI can invoke
//   - Prompts: Pre-defined prompt templates
//   - Sampling: Server requests client to use LLM for generation
//
// Example usage:
//
//	server := mcp.NewServer(mcp.ServerConfig{
//	    Name:    "my-server",
//	    Version: "1.0.0",
//	})
//	
//	server.RegisterResource(&mcp.Resource{
//	    URI:  "file:///docs",
//	    Name: "Documentation",
//	}, provider)
//	
//	server.Serve(ctx, mcp.NewStdioTransport())
package mcp

import (
	"context"
	"io"
	"time"
)

// MCPServer represents a Model Context Protocol server.
// It provides resources, tools, prompts, and sampling capabilities to clients.
type MCPServer interface {
	// Resource Management
	ListResources(ctx context.Context) ([]*Resource, error)
	ReadResource(ctx context.Context, uri string) (*ResourceContent, error)
	SubscribeResource(ctx context.Context, uri string, handler ResourceUpdateHandler) error
	UnsubscribeResource(ctx context.Context, uri string) error
	
	// Tool Management
	ListTools(ctx context.Context) ([]*Tool, error)
	CallTool(ctx context.Context, name string, args map[string]any) (*ToolResult, error)
	
	// Prompt Management
	ListPrompts(ctx context.Context) ([]*Prompt, error)
	GetPrompt(ctx context.Context, name string, args map[string]any) (*PromptResult, error)
	
	// Sampling
	CreateMessage(ctx context.Context, req *MessageRequest) (*MessageResponse, error)
	
	// Lifecycle
	Initialize(ctx context.Context, info *ClientInfo) (*ServerInfo, error)
	Ping(ctx context.Context) error
	SetLogLevel(ctx context.Context, level LogLevel) error
	
	// Serve starts the server with the given transport
	Serve(ctx context.Context, transport Transport) error
}

// MCPClient represents a Model Context Protocol client.
// It connects to an MCP server and uses its capabilities.
type MCPClient interface {
	// Connection Management
	Connect(ctx context.Context, uri string) error
	Disconnect(ctx context.Context) error
	
	// Resource Operations
	ListResources(ctx context.Context) ([]*Resource, error)
	ReadResource(ctx context.Context, uri string) (*ResourceContent, error)
	SubscribeResource(ctx context.Context, uri string) error
	UnsubscribeResource(ctx context.Context, uri string) error
	
	// Tool Operations
	ListTools(ctx context.Context) ([]*Tool, error)
	CallTool(ctx context.Context, name string, args map[string]any) (*ToolResult, error)
	
	// Prompt Operations
	ListPrompts(ctx context.Context) ([]*Prompt, error)
	GetPrompt(ctx context.Context, name string, args map[string]any) (*PromptResult, error)
	
	// Event Handlers
	OnResourceUpdated(handler ResourceUpdateHandler)
	OnProgressUpdate(handler ProgressUpdateHandler)
}

// Resource represents an MCP resource.
// Resources are data sources that can be accessed by AI models.
type Resource struct {
	URI         string         `json:"uri"`         // Resource URI (e.g., file:///, db://)
	Name        string         `json:"name"`        // Resource name
	Description string         `json:"description"` // Resource description
	MimeType    string         `json:"mimeType"`    // MIME type
	Metadata    map[string]any `json:"metadata"`    // Additional metadata
}

// ResourceContent represents the content of a resource.
type ResourceContent struct {
	URI      string `json:"uri"`                // Resource URI
	MimeType string `json:"mimeType"`           // MIME type
	Text     string `json:"text,omitempty"`     // Text content
	Blob     []byte `json:"blob,omitempty"`     // Binary content (base64 encoded in JSON)
}

// Tool represents an MCP tool definition.
// Tools are functions that AI models can invoke.
type Tool struct {
	Name        string         `json:"name"`        // Tool name
	Description string         `json:"description"` // Tool description
	InputSchema map[string]any `json:"inputSchema"` // JSON Schema for input
}

// ToolResult represents the result of a tool execution.
type ToolResult struct {
	Content []ContentBlock `json:"content"`           // Result content blocks
	IsError bool           `json:"isError,omitempty"` // Whether this is an error result
}

// ContentBlock represents a block of content in various formats.
type ContentBlock struct {
	Type     string         `json:"type"`                // "text", "image", "resource"
	Text     string         `json:"text,omitempty"`      // Text content
	Data     string         `json:"data,omitempty"`      // Base64 encoded data
	MimeType string         `json:"mimeType,omitempty"`  // MIME type
	Resource *Resource      `json:"resource,omitempty"`  // Resource reference
}

// Prompt represents a prompt template.
type Prompt struct {
	Name        string            `json:"name"`                  // Prompt name
	Description string            `json:"description,omitempty"` // Prompt description
	Arguments   []PromptArgument  `json:"arguments,omitempty"`   // Prompt arguments
}

// PromptArgument represents a prompt argument.
type PromptArgument struct {
	Name        string `json:"name"`                  // Argument name
	Description string `json:"description,omitempty"` // Argument description
	Required    bool   `json:"required,omitempty"`    // Whether required
}

// PromptResult represents the result of getting a prompt.
type PromptResult struct {
	Description string    `json:"description,omitempty"` // Prompt description
	Messages    []Message `json:"messages"`              // Prompt messages
}

// Message represents a message in a conversation.
type Message struct {
	Role    string       `json:"role"`    // "user" or "assistant"
	Content ContentBlock `json:"content"` // Message content
}

// MessageRequest represents a sampling request.
// The server asks the client to use an LLM to generate content.
type MessageRequest struct {
	Messages         []Message       `json:"messages"`                   // Conversation messages
	ModelPrefs       *ModelPrefs     `json:"modelPreferences,omitempty"` // Model preferences
	SystemPrompt     string          `json:"systemPrompt,omitempty"`     // System prompt
	MaxTokens        int             `json:"maxTokens,omitempty"`        // Maximum tokens
	Temperature      float64         `json:"temperature,omitempty"`      // Temperature
	StopSequences    []string        `json:"stopSequences,omitempty"`    // Stop sequences
	Metadata         map[string]any  `json:"metadata,omitempty"`         // Additional metadata
}

// ModelPrefs represents model preferences for sampling.
type ModelPrefs struct {
	Hints                 []ModelHint `json:"hints,omitempty"`                 // Model hints
	CostPriority          float64     `json:"costPriority,omitempty"`          // Cost priority (0-1)
	SpeedPriority         float64     `json:"speedPriority,omitempty"`         // Speed priority (0-1)
	IntelligencePriority  float64     `json:"intelligencePriority,omitempty"`  // Intelligence priority (0-1)
}

// ModelHint represents a model hint.
type ModelHint struct {
	Name string `json:"name"` // Model name (e.g., "gpt-4", "claude-3-opus")
}

// MessageResponse represents the response from sampling.
type MessageResponse struct {
	Role       string       `json:"role"`       // "assistant"
	Content    ContentBlock `json:"content"`    // Generated content
	Model      string       `json:"model"`      // Model used
	StopReason string       `json:"stopReason"` // Stop reason
}

// ServerInfo represents MCP server information.
type ServerInfo struct {
	Name         string             `json:"name"`                   // Server name
	Version      string             `json:"version"`                // Server version
	Vendor       string             `json:"vendor,omitempty"`       // Vendor name
	Capabilities ServerCapabilities `json:"capabilities,omitempty"` // Server capabilities
}

// ServerCapabilities represents server capabilities.
type ServerCapabilities struct {
	Resources *ResourceCapability `json:"resources,omitempty"` // Resource capabilities
	Tools     *ToolCapability     `json:"tools,omitempty"`     // Tool capabilities
	Prompts   *PromptCapability   `json:"prompts,omitempty"`   // Prompt capabilities
	Sampling  *SamplingCapability `json:"sampling,omitempty"`  // Sampling capabilities
	Logging   *LoggingCapability  `json:"logging,omitempty"`   // Logging capabilities
}

// ResourceCapability represents resource capabilities.
type ResourceCapability struct {
	Subscribe   bool `json:"subscribe,omitempty"`   // Supports resource subscription
	ListChanged bool `json:"listChanged,omitempty"` // Resource list can change
}

// ToolCapability represents tool capabilities.
type ToolCapability struct {
	ListChanged bool `json:"listChanged,omitempty"` // Tool list can change
}

// PromptCapability represents prompt capabilities.
type PromptCapability struct {
	ListChanged bool `json:"listChanged,omitempty"` // Prompt list can change
}

// SamplingCapability represents sampling capabilities.
type SamplingCapability struct{}

// LoggingCapability represents logging capabilities.
type LoggingCapability struct{}

// ClientInfo represents MCP client information.
type ClientInfo struct {
	Name         string             `json:"name"`                   // Client name
	Version      string             `json:"version"`                // Client version
	Capabilities ClientCapabilities `json:"capabilities,omitempty"` // Client capabilities
}

// ClientCapabilities represents client capabilities.
type ClientCapabilities struct {
	Sampling *SamplingCapability `json:"sampling,omitempty"` // Can perform sampling
	Roots    *RootsCapability    `json:"roots,omitempty"`    // Roots capability
}

// RootsCapability represents roots capability.
type RootsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"` // Roots list can change
}

// LogLevel represents log level.
type LogLevel string

const (
	LogLevelDebug     LogLevel = "debug"
	LogLevelInfo      LogLevel = "info"
	LogLevelNotice    LogLevel = "notice"
	LogLevelWarning   LogLevel = "warning"
	LogLevelError     LogLevel = "error"
	LogLevelCritical  LogLevel = "critical"
	LogLevelAlert     LogLevel = "alert"
	LogLevelEmergency LogLevel = "emergency"
)

// ResourceUpdateHandler handles resource update notifications.
type ResourceUpdateHandler func(uri string)

// ProgressUpdateHandler handles progress update notifications.
type ProgressUpdateHandler func(progress float64)

// ResourceProvider provides access to resources.
// Implementations handle different resource types (filesystem, database, etc.).
type ResourceProvider interface {
	// Read reads the resource content
	Read(ctx context.Context, uri string) (*ResourceContent, error)
	
	// Subscribe subscribes to resource updates (optional)
	Subscribe(ctx context.Context, uri string) (<-chan *ResourceContent, error)
	
	// Unsubscribe unsubscribes from resource updates (optional)
	Unsubscribe(ctx context.Context, uri string) error
}

// Transport represents a transport layer for MCP communication.
// Implementations include Stdio, SSE, and WebSocket.
type Transport interface {
	// Send sends a message
	Send(ctx context.Context, msg []byte) error
	
	// Receive receives a message
	Receive(ctx context.Context) ([]byte, error)
	
	// Close closes the transport
	Close() error
}

// ToolHandler handles tool execution.
type ToolHandler func(ctx context.Context, args map[string]any) (*ToolResult, error)

// PromptHandler handles prompt generation.
type PromptHandler func(ctx context.Context, args map[string]any) (*PromptResult, error)

// Config error types
var (
	ErrResourceNotFound   = NewMCPError(-32000, "Resource not found")
	ErrResourceAccessDenied = NewMCPError(-32001, "Resource access denied")
	ErrToolNotFound       = NewMCPError(-32002, "Tool not found")
	ErrToolExecutionFailed = NewMCPError(-32003, "Tool execution failed")
	ErrPromptNotFound     = NewMCPError(-32004, "Prompt not found")
	ErrSamplingFailed     = NewMCPError(-32005, "Sampling failed")
)

// MCPError represents an MCP-specific error.
type MCPError struct {
	Code    int            `json:"code"`
	Message string         `json:"message"`
	Data    map[string]any `json:"data,omitempty"`
}

// NewMCPError creates a new MCP error.
func NewMCPError(code int, message string) *MCPError {
	return &MCPError{
		Code:    code,
		Message: message,
		Data:    make(map[string]any),
	}
}

// Error implements the error interface.
func (e *MCPError) Error() string {
	return e.Message
}

// WithData adds data to the error.
func (e *MCPError) WithData(key string, value any) *MCPError {
	e.Data[key] = value
	return e
}
