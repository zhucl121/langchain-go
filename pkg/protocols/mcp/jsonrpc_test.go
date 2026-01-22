package mcp

import (
	"encoding/json"
	"testing"
)

func TestNewRequest(t *testing.T) {
	tests := []struct {
		name   string
		id     any
		method string
		params any
		wantID any
	}{
		{
			name:   "String ID",
			id:     "req-1",
			method: "test/method",
			params: map[string]any{"key": "value"},
			wantID: "req-1",
		},
		{
			name:   "Numeric ID",
			id:     123,
			method: "test/method",
			params: nil,
			wantID: 123,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := NewRequest(tt.id, tt.method, tt.params)
			if err != nil {
				t.Fatalf("NewRequest() error = %v", err)
			}
			
			if req.JSONRPC != "2.0" {
				t.Errorf("JSONRPC = %v, want 2.0", req.JSONRPC)
			}
			
			if req.ID != tt.wantID {
				t.Errorf("ID = %v, want %v", req.ID, tt.wantID)
			}
			
			if req.Method != tt.method {
				t.Errorf("Method = %v, want %v", req.Method, tt.method)
			}
		})
	}
}

func TestNewResponse(t *testing.T) {
	result := map[string]any{
		"status": "success",
		"data":   "test data",
	}
	
	resp, err := NewResponse("req-1", result)
	if err != nil {
		t.Fatalf("NewResponse() error = %v", err)
	}
	
	if resp.JSONRPC != "2.0" {
		t.Errorf("JSONRPC = %v, want 2.0", resp.JSONRPC)
	}
	
	if resp.ID != "req-1" {
		t.Errorf("ID = %v, want req-1", resp.ID)
	}
	
	// Parse result
	var parsedResult map[string]any
	if err := json.Unmarshal(resp.Result, &parsedResult); err != nil {
		t.Fatalf("Failed to parse result: %v", err)
	}
	
	if parsedResult["status"] != "success" {
		t.Errorf("Result status = %v, want success", parsedResult["status"])
	}
}

func TestNewErrorResponse(t *testing.T) {
	resp := NewErrorResponse("req-1", ErrCodeInvalidParams, "Invalid params", map[string]any{
		"param": "test",
	})
	
	if resp.JSONRPC != "2.0" {
		t.Errorf("JSONRPC = %v, want 2.0", resp.JSONRPC)
	}
	
	if resp.Error == nil {
		t.Fatal("Expected error to be non-nil")
	}
	
	if resp.Error.Code != ErrCodeInvalidParams {
		t.Errorf("Error.Code = %v, want %v", resp.Error.Code, ErrCodeInvalidParams)
	}
	
	if resp.Error.Message != "Invalid params" {
		t.Errorf("Error.Message = %v, want Invalid params", resp.Error.Message)
	}
}

func TestNewNotification(t *testing.T) {
	params := map[string]any{"event": "test"}
	
	notif, err := NewNotification("notifications/test", params)
	if err != nil {
		t.Fatalf("NewNotification() error = %v", err)
	}
	
	if notif.JSONRPC != "2.0" {
		t.Errorf("JSONRPC = %v, want 2.0", notif.JSONRPC)
	}
	
	if notif.ID != nil {
		t.Errorf("ID = %v, want nil for notification", notif.ID)
	}
	
	if notif.Method != "notifications/test" {
		t.Errorf("Method = %v, want notifications/test", notif.Method)
	}
}

func TestJSONRPCMessage_IsRequest(t *testing.T) {
	tests := []struct {
		name string
		msg  *JSONRPCMessage
		want bool
	}{
		{
			name: "Valid request",
			msg: &JSONRPCMessage{
				JSONRPC: "2.0",
				ID:      "req-1",
				Method:  "test/method",
			},
			want: true,
		},
		{
			name: "Response (not request)",
			msg: &JSONRPCMessage{
				JSONRPC: "2.0",
				ID:      "req-1",
				Result:  json.RawMessage(`{"status": "ok"}`),
			},
			want: false,
		},
		{
			name: "Notification (not request)",
			msg: &JSONRPCMessage{
				JSONRPC: "2.0",
				Method:  "notifications/test",
			},
			want: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.msg.IsRequest(); got != tt.want {
				t.Errorf("IsRequest() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestJSONRPCMessage_IsResponse(t *testing.T) {
	tests := []struct {
		name string
		msg  *JSONRPCMessage
		want bool
	}{
		{
			name: "Valid response with result",
			msg: &JSONRPCMessage{
				JSONRPC: "2.0",
				ID:      "req-1",
				Result:  json.RawMessage(`{"status": "ok"}`),
			},
			want: true,
		},
		{
			name: "Valid response with error",
			msg: &JSONRPCMessage{
				JSONRPC: "2.0",
				ID:      "req-1",
				Error: &JSONRPCError{
					Code:    -32000,
					Message: "Test error",
				},
			},
			want: true,
		},
		{
			name: "Request (not response)",
			msg: &JSONRPCMessage{
				JSONRPC: "2.0",
				ID:      "req-1",
				Method:  "test/method",
			},
			want: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.msg.IsResponse(); got != tt.want {
				t.Errorf("IsResponse() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestJSONRPCMessage_IsNotification(t *testing.T) {
	tests := []struct {
		name string
		msg  *JSONRPCMessage
		want bool
	}{
		{
			name: "Valid notification",
			msg: &JSONRPCMessage{
				JSONRPC: "2.0",
				Method:  "notifications/test",
			},
			want: true,
		},
		{
			name: "Request (not notification)",
			msg: &JSONRPCMessage{
				JSONRPC: "2.0",
				ID:      "req-1",
				Method:  "test/method",
			},
			want: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.msg.IsNotification(); got != tt.want {
				t.Errorf("IsNotification() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestJSONRPCMessage_ParseParams(t *testing.T) {
	params := map[string]any{
		"name": "test",
		"value": 123,
	}
	
	msg, err := NewRequest("req-1", "test/method", params)
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}
	
	var parsed struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}
	
	if err := msg.ParseParams(&parsed); err != nil {
		t.Fatalf("ParseParams() error = %v", err)
	}
	
	if parsed.Name != "test" {
		t.Errorf("Parsed name = %v, want test", parsed.Name)
	}
	
	if parsed.Value != 123 {
		t.Errorf("Parsed value = %v, want 123", parsed.Value)
	}
}

func TestJSONRPCMessage_ToJSON(t *testing.T) {
	msg := &JSONRPCMessage{
		JSONRPC: "2.0",
		ID:      "req-1",
		Method:  "test/method",
	}
	
	data, err := msg.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON() error = %v", err)
	}
	
	// Parse back
	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}
	
	if parsed["jsonrpc"] != "2.0" {
		t.Errorf("jsonrpc = %v, want 2.0", parsed["jsonrpc"])
	}
	
	if parsed["id"] != "req-1" {
		t.Errorf("id = %v, want req-1", parsed["id"])
	}
}

func TestFromJSON(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		wantErr bool
		checkFn func(*testing.T, *JSONRPCMessage)
	}{
		{
			name:    "Valid request",
			data:    `{"jsonrpc":"2.0","id":"req-1","method":"test"}`,
			wantErr: false,
			checkFn: func(t *testing.T, msg *JSONRPCMessage) {
				if msg.Method != "test" {
					t.Errorf("Method = %v, want test", msg.Method)
				}
			},
		},
		{
			name:    "Invalid JSON",
			data:    `{"jsonrpc":"2.0","id":`,
			wantErr: true,
		},
		{
			name:    "Invalid JSON-RPC version",
			data:    `{"jsonrpc":"1.0","id":"req-1"}`,
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg, err := FromJSON([]byte(tt.data))
			if (err != nil) != tt.wantErr {
				t.Errorf("FromJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if !tt.wantErr && tt.checkFn != nil {
				tt.checkFn(t, msg)
			}
		})
	}
}

func TestJSONRPCError_Error(t *testing.T) {
	err := &JSONRPCError{
		Code:    -32000,
		Message: "Test error",
	}
	
	expected := "JSON-RPC error -32000: Test error"
	if err.Error() != expected {
		t.Errorf("Error() = %v, want %v", err.Error(), expected)
	}
}

func TestBatchMessage(t *testing.T) {
	batch := BatchMessage{
		{
			JSONRPC: "2.0",
			ID:      "req-1",
			Method:  "test1",
		},
		{
			JSONRPC: "2.0",
			ID:      "req-2",
			Method:  "test2",
		},
	}
	
	data, err := batch.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON() error = %v", err)
	}
	
	// Parse back
	parsed, err := FromJSONBatch(data)
	if err != nil {
		t.Fatalf("FromJSONBatch() error = %v", err)
	}
	
	if len(parsed) != 2 {
		t.Errorf("Parsed batch length = %v, want 2", len(parsed))
	}
}
