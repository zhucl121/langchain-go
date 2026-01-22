package mcp

import (
	"encoding/json"
	"fmt"
)

// JSON-RPC 2.0 implementation for MCP

// JSONRPCMessage represents a JSON-RPC 2.0 message.
type JSONRPCMessage struct {
	JSONRPC string          `json:"jsonrpc"`         // Must be "2.0"
	ID      any             `json:"id,omitempty"`    // Request ID (string or number)
	Method  string          `json:"method,omitempty"` // Method name
	Params  json.RawMessage `json:"params,omitempty"` // Method parameters
	Result  json.RawMessage `json:"result,omitempty"` // Method result
	Error   *JSONRPCError   `json:"error,omitempty"`  // Error object
}

// JSONRPCError represents a JSON-RPC 2.0 error.
type JSONRPCError struct {
	Code    int            `json:"code"`              // Error code
	Message string         `json:"message"`           // Error message
	Data    map[string]any `json:"data,omitempty"`    // Additional error data
}

// JSON-RPC 2.0 error codes
const (
	ErrCodeParseError     = -32700 // Parse error
	ErrCodeInvalidRequest = -32600 // Invalid Request
	ErrCodeMethodNotFound = -32601 // Method not found
	ErrCodeInvalidParams  = -32602 // Invalid params
	ErrCodeInternalError  = -32603 // Internal error
	ErrCodeServerError    = -32000 // Server error (start of range)
)

// NewRequest creates a new JSON-RPC request.
func NewRequest(id any, method string, params any) (*JSONRPCMessage, error) {
	var paramsBytes json.RawMessage
	if params != nil {
		bytes, err := json.Marshal(params)
		if err != nil {
			return nil, fmt.Errorf("marshal params: %w", err)
		}
		paramsBytes = bytes
	}
	
	return &JSONRPCMessage{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  paramsBytes,
	}, nil
}

// NewResponse creates a new JSON-RPC response.
func NewResponse(id any, result any) (*JSONRPCMessage, error) {
	var resultBytes json.RawMessage
	if result != nil {
		bytes, err := json.Marshal(result)
		if err != nil {
			return nil, fmt.Errorf("marshal result: %w", err)
		}
		resultBytes = bytes
	}
	
	return &JSONRPCMessage{
		JSONRPC: "2.0",
		ID:      id,
		Result:  resultBytes,
	}, nil
}

// NewErrorResponse creates a new JSON-RPC error response.
func NewErrorResponse(id any, code int, message string, data map[string]any) *JSONRPCMessage {
	return &JSONRPCMessage{
		JSONRPC: "2.0",
		ID:      id,
		Error: &JSONRPCError{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}
}

// NewNotification creates a new JSON-RPC notification (no ID).
func NewNotification(method string, params any) (*JSONRPCMessage, error) {
	var paramsBytes json.RawMessage
	if params != nil {
		bytes, err := json.Marshal(params)
		if err != nil {
			return nil, fmt.Errorf("marshal params: %w", err)
		}
		paramsBytes = bytes
	}
	
	return &JSONRPCMessage{
		JSONRPC: "2.0",
		Method:  method,
		Params:  paramsBytes,
	}, nil
}

// IsRequest checks if the message is a request.
func (m *JSONRPCMessage) IsRequest() bool {
	return m.Method != "" && m.ID != nil
}

// IsResponse checks if the message is a response.
func (m *JSONRPCMessage) IsResponse() bool {
	return m.ID != nil && (m.Result != nil || m.Error != nil)
}

// IsNotification checks if the message is a notification.
func (m *JSONRPCMessage) IsNotification() bool {
	return m.Method != "" && m.ID == nil
}

// IsError checks if the message is an error response.
func (m *JSONRPCMessage) IsError() bool {
	return m.Error != nil
}

// ParseParams parses the request parameters into the given struct.
func (m *JSONRPCMessage) ParseParams(v any) error {
	if m.Params == nil {
		return nil
	}
	
	return json.Unmarshal(m.Params, v)
}

// ParseResult parses the response result into the given struct.
func (m *JSONRPCMessage) ParseResult(v any) error {
	if m.Result == nil {
		return nil
	}
	
	return json.Unmarshal(m.Result, v)
}

// ToJSON converts the message to JSON bytes.
func (m *JSONRPCMessage) ToJSON() ([]byte, error) {
	return json.Marshal(m)
}

// FromJSON parses JSON bytes into a message.
func FromJSON(data []byte) (*JSONRPCMessage, error) {
	var msg JSONRPCMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, &JSONRPCError{
			Code:    ErrCodeParseError,
			Message: "Parse error",
			Data:    map[string]any{"details": err.Error()},
		}
	}
	
	// Validate JSON-RPC version
	if msg.JSONRPC != "2.0" {
		return nil, &JSONRPCError{
			Code:    ErrCodeInvalidRequest,
			Message: "Invalid Request",
			Data:    map[string]any{"details": "jsonrpc must be '2.0'"},
		}
	}
	
	return &msg, nil
}

// Error implements the error interface for JSONRPCError.
func (e *JSONRPCError) Error() string {
	return fmt.Sprintf("JSON-RPC error %d: %s", e.Code, e.Message)
}

// ToMCPError converts a JSON-RPC error to an MCP error.
func (e *JSONRPCError) ToMCPError() *MCPError {
	return &MCPError{
		Code:    e.Code,
		Message: e.Message,
		Data:    e.Data,
	}
}

// FromMCPError converts an MCP error to a JSON-RPC error.
func FromMCPError(err *MCPError) *JSONRPCError {
	return &JSONRPCError{
		Code:    err.Code,
		Message: err.Message,
		Data:    err.Data,
	}
}

// BatchMessage represents a batch of JSON-RPC messages.
type BatchMessage []JSONRPCMessage

// ToJSON converts the batch to JSON bytes.
func (b BatchMessage) ToJSON() ([]byte, error) {
	return json.Marshal(b)
}

// FromJSONBatch parses JSON bytes into a batch of messages.
func FromJSONBatch(data []byte) (BatchMessage, error) {
	var batch BatchMessage
	if err := json.Unmarshal(data, &batch); err != nil {
		return nil, &JSONRPCError{
			Code:    ErrCodeParseError,
			Message: "Parse error",
			Data:    map[string]any{"details": err.Error()},
		}
	}
	
	// Validate each message
	for i, msg := range batch {
		if msg.JSONRPC != "2.0" {
			return nil, &JSONRPCError{
				Code:    ErrCodeInvalidRequest,
				Message: "Invalid Request",
				Data:    map[string]any{"details": fmt.Sprintf("message %d: jsonrpc must be '2.0'", i)},
			}
		}
	}
	
	return batch, nil
}
