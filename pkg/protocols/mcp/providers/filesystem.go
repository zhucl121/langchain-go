// Package providers implements resource providers for MCP.
package providers

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	
	"github.com/zhucl121/langchain-go/pkg/protocols/mcp"
)

// FileSystemProvider provides access to filesystem resources.
type FileSystemProvider struct {
	root string // Root directory
}

// NewFileSystemProvider creates a new filesystem provider.
// root is the root directory that this provider can access.
func NewFileSystemProvider(root string) *FileSystemProvider {
	return &FileSystemProvider{
		root: root,
	}
}

// Read reads a file from the filesystem.
func (p *FileSystemProvider) Read(ctx context.Context, uri string) (*mcp.ResourceContent, error) {
	// Parse URI: file:///path/to/file
	path := p.uriToPath(uri)
	
	// Security check: ensure path is within root
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("abs path: %w", err)
	}
	
	absRoot, err := filepath.Abs(p.root)
	if err != nil {
		return nil, fmt.Errorf("abs root: %w", err)
	}
	
	if !strings.HasPrefix(absPath, absRoot) {
		return nil, fmt.Errorf("access denied: path outside root directory")
	}
	
	// Read file
	file, err := os.Open(absPath)
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}
	defer file.Close()
	
	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}
	
	// Detect MIME type
	mimeType := p.detectMimeType(absPath)
	
	return &mcp.ResourceContent{
		URI:      uri,
		MimeType: mimeType,
		Text:     string(data),
	}, nil
}

// Subscribe is not implemented for filesystem provider.
func (p *FileSystemProvider) Subscribe(ctx context.Context, uri string) (<-chan *mcp.ResourceContent, error) {
	return nil, fmt.Errorf("subscribe not supported for filesystem provider")
}

// Unsubscribe is not implemented for filesystem provider.
func (p *FileSystemProvider) Unsubscribe(ctx context.Context, uri string) error {
	return fmt.Errorf("unsubscribe not supported for filesystem provider")
}

// uriToPath converts a file:// URI to a filesystem path.
func (p *FileSystemProvider) uriToPath(uri string) string {
	// Remove file:// prefix
	path := strings.TrimPrefix(uri, "file://")
	
	// If path is relative, make it relative to root
	if !filepath.IsAbs(path) {
		path = filepath.Join(p.root, path)
	}
	
	return path
}

// detectMimeType detects the MIME type of a file based on extension.
func (p *FileSystemProvider) detectMimeType(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	
	switch ext {
	case ".txt":
		return "text/plain"
	case ".md", ".markdown":
		return "text/markdown"
	case ".json":
		return "application/json"
	case ".xml":
		return "application/xml"
	case ".html", ".htm":
		return "text/html"
	case ".css":
		return "text/css"
	case ".js":
		return "application/javascript"
	case ".go":
		return "text/x-go"
	case ".py":
		return "text/x-python"
	case ".java":
		return "text/x-java"
	case ".c", ".h":
		return "text/x-c"
	case ".cpp", ".hpp", ".cc":
		return "text/x-c++"
	default:
		return "text/plain"
	}
}
