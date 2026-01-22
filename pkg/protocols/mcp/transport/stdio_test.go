package transport

import (
	"context"
	"testing"
)

func TestNewStdioTransport(t *testing.T) {
	transport := NewStdioTransport()
	
	if transport == nil {
		t.Fatal("NewStdioTransport() returned nil")
	}
	
	if transport.stdin == nil {
		t.Error("Expected stdin to be set")
	}
	
	if transport.stdout == nil {
		t.Error("Expected stdout to be set")
	}
}

func TestStdioTransport_IsClosed(t *testing.T) {
	transport := NewStdioTransport()
	
	if transport.IsClosed() {
		t.Error("Expected transport to not be closed initially")
	}
	
	transport.Close()
	
	if !transport.IsClosed() {
		t.Error("Expected transport to be closed after Close()")
	}
}

func TestStdioTransport_Send_AfterClose(t *testing.T) {
	transport := NewStdioTransport()
	transport.Close()
	
	ctx := context.Background()
	err := transport.Send(ctx, []byte("test"))
	
	if err == nil {
		t.Error("Expected error when sending after close")
	}
}

func TestStdioTransport_Receive_AfterClose(t *testing.T) {
	transport := NewStdioTransport()
	transport.Close()
	
	ctx := context.Background()
	_, err := transport.Receive(ctx)
	
	if err == nil {
		t.Error("Expected error when receiving after close")
	}
}

func TestStdioTransport_Close_Multiple(t *testing.T) {
	transport := NewStdioTransport()
	
	err := transport.Close()
	if err != nil {
		t.Errorf("First Close() error = %v", err)
	}
	
	// Second close should not error
	err = transport.Close()
	if err != nil {
		t.Errorf("Second Close() error = %v", err)
	}
}

func TestStdioTransport_GetStderr(t *testing.T) {
	transport := NewStdioTransport()
	
	stderr := transport.GetStderr()
	if stderr == nil {
		t.Error("Expected GetStderr() to return non-nil reader")
	}
}
