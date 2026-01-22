// Package transport provides transport layer implementations for MCP.
package transport

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
)

// StdioTransport implements MCP transport over standard input/output.
// This is the primary transport method used by Claude Desktop and other
// local MCP clients.
//
// Example usage:
//
//	transport := transport.NewStdioTransport()
//	server.Serve(ctx, transport)
type StdioTransport struct {
	cmd    *exec.Cmd         // External command (if any)
	stdin  io.WriteCloser    // Standard input
	stdout io.ReadCloser     // Standard output
	stderr io.ReadCloser     // Standard error
	
	scanner *bufio.Scanner
	mu      sync.Mutex
	closed  bool
}

// NewStdioTransport creates a new Stdio transport using os.Stdin and os.Stdout.
func NewStdioTransport() *StdioTransport {
	return &StdioTransport{
		stdin:   os.Stdin,
		stdout:  os.Stdout,
		stderr:  os.Stderr,
		scanner: bufio.NewScanner(os.Stdin),
	}
}

// NewStdioTransportWithCommand creates a new Stdio transport by launching an external command.
// This is useful for MCP clients connecting to external servers.
//
// Example:
//
//	transport, err := transport.NewStdioTransportWithCommand("python", "mcp_server.py")
func NewStdioTransportWithCommand(name string, args ...string) (*StdioTransport, error) {
	cmd := exec.Command(name, args...)
	
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("create stdin pipe: %w", err)
	}
	
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		stdin.Close()
		return nil, fmt.Errorf("create stdout pipe: %w", err)
	}
	
	stderr, err := cmd.StderrPipe()
	if err != nil {
		stdin.Close()
		stdout.Close()
		return nil, fmt.Errorf("create stderr pipe: %w", err)
	}
	
	if err := cmd.Start(); err != nil {
		stdin.Close()
		stdout.Close()
		stderr.Close()
		return nil, fmt.Errorf("start command: %w", err)
	}
	
	return &StdioTransport{
		cmd:     cmd,
		stdin:   stdin,
		stdout:  stdout,
		stderr:  stderr,
		scanner: bufio.NewScanner(stdout),
	}, nil
}

// Send sends a message over the transport.
// Messages are sent as JSON followed by a newline.
func (t *StdioTransport) Send(ctx context.Context, data []byte) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	
	if t.closed {
		return fmt.Errorf("transport closed")
	}
	
	// Check context cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	
	// Write message followed by newline
	if _, err := t.stdin.Write(data); err != nil {
		return fmt.Errorf("write message: %w", err)
	}
	
	if _, err := t.stdin.Write([]byte("\n")); err != nil {
		return fmt.Errorf("write newline: %w", err)
	}
	
	return nil
}

// Receive receives a message from the transport.
// Messages are received as JSON lines (one JSON object per line).
func (t *StdioTransport) Receive(ctx context.Context) ([]byte, error) {
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return nil, fmt.Errorf("transport closed")
	}
	scanner := t.scanner
	t.mu.Unlock()
	
	// Check context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	
	// Read next line
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return nil, fmt.Errorf("scan error: %w", err)
		}
		return nil, io.EOF
	}
	
	return scanner.Bytes(), nil
}

// Close closes the transport and any associated resources.
func (t *StdioTransport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	
	if t.closed {
		return nil
	}
	
	t.closed = true
	
	var errs []error
	
	// Close pipes
	if t.stdin != os.Stdin && t.stdin != nil {
		if err := t.stdin.Close(); err != nil {
			errs = append(errs, fmt.Errorf("close stdin: %w", err))
		}
	}
	
	if t.stdout != os.Stdout && t.stdout != nil {
		if err := t.stdout.Close(); err != nil {
			errs = append(errs, fmt.Errorf("close stdout: %w", err))
		}
	}
	
	if t.stderr != os.Stderr && t.stderr != nil {
		if err := t.stderr.Close(); err != nil {
			errs = append(errs, fmt.Errorf("close stderr: %w", err))
		}
	}
	
	// Wait for command to finish
	if t.cmd != nil {
		if err := t.cmd.Wait(); err != nil {
			errs = append(errs, fmt.Errorf("wait for command: %w", err))
		}
	}
	
	if len(errs) > 0 {
		return fmt.Errorf("close errors: %v", errs)
	}
	
	return nil
}

// IsClosed returns whether the transport is closed.
func (t *StdioTransport) IsClosed() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.closed
}

// GetStderr returns the stderr reader (useful for debugging).
func (t *StdioTransport) GetStderr() io.Reader {
	return t.stderr
}
