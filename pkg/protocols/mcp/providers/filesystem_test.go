package providers

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestNewFileSystemProvider(t *testing.T) {
	root := "/tmp/test"
	provider := NewFileSystemProvider(root)
	
	if provider == nil {
		t.Fatal("NewFileSystemProvider() returned nil")
	}
	
	if provider.root != root {
		t.Errorf("root = %v, want %v", provider.root, root)
	}
}

func TestFileSystemProvider_Read(t *testing.T) {
	// Create temp directory and file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	content := "Hello, MCP!"
	
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	
	provider := NewFileSystemProvider(tmpDir)
	ctx := context.Background()
	
	// Read file using absolute path
	uri := "file://" + testFile
	result, err := provider.Read(ctx, uri)
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	
	if result.Text != content {
		t.Errorf("Content = %v, want %v", result.Text, content)
	}
	
	if result.MimeType != "text/plain" {
		t.Errorf("MimeType = %v, want text/plain", result.MimeType)
	}
}

func TestFileSystemProvider_Read_SecurityCheck(t *testing.T) {
	tmpDir := t.TempDir()
	provider := NewFileSystemProvider(tmpDir)
	ctx := context.Background()
	
	// Try to read file outside root (should fail)
	_, err := provider.Read(ctx, "file:///../../../etc/passwd")
	if err == nil {
		t.Error("Expected error for path outside root")
	}
}

func TestFileSystemProvider_DetectMimeType(t *testing.T) {
	provider := NewFileSystemProvider("/tmp")
	
	tests := []struct {
		path     string
		wantType string
	}{
		{"/test.txt", "text/plain"},
		{"/test.md", "text/markdown"},
		{"/test.json", "application/json"},
		{"/test.html", "text/html"},
		{"/test.go", "text/x-go"},
		{"/test.py", "text/x-python"},
		{"/test.unknown", "text/plain"}, // Default
	}
	
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := provider.detectMimeType(tt.path)
			if got != tt.wantType {
				t.Errorf("detectMimeType(%v) = %v, want %v", tt.path, got, tt.wantType)
			}
		})
	}
}

func TestFileSystemProvider_URIToPath(t *testing.T) {
	provider := NewFileSystemProvider("/root")
	
	tests := []struct {
		name string
		uri  string
		want string
	}{
		{
			name: "Absolute path",
			uri:  "file:///absolute/path",
			want: "/absolute/path",
		},
		{
			name: "Relative path",
			uri:  "file://relative/path",
			want: "/root/relative/path",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := provider.uriToPath(tt.uri)
			// Normalize paths for comparison
			gotAbs, _ := filepath.Abs(got)
			wantAbs, _ := filepath.Abs(tt.want)
			
			if gotAbs != wantAbs {
				t.Errorf("uriToPath(%v) = %v, want %v", tt.uri, got, tt.want)
			}
		})
	}
}

func TestFileSystemProvider_Subscribe(t *testing.T) {
	provider := NewFileSystemProvider("/tmp")
	ctx := context.Background()
	
	_, err := provider.Subscribe(ctx, "file:///test.txt")
	if err == nil {
		t.Error("Expected Subscribe to return error (not supported)")
	}
}

func TestFileSystemProvider_Unsubscribe(t *testing.T) {
	provider := NewFileSystemProvider("/tmp")
	ctx := context.Background()
	
	err := provider.Unsubscribe(ctx, "file:///test.txt")
	if err == nil {
		t.Error("Expected Unsubscribe to return error (not supported)")
	}
}
