package loaders

import (
	"testing"
)

func TestNewGitHubLoader(t *testing.T) {
	tests := []struct {
		name      string
		config    GitHubLoaderConfig
		wantError bool
	}{
		{
			name: "valid config",
			config: GitHubLoaderConfig{
				Owner: "test-owner",
				Repo:  "test-repo",
			},
			wantError: false,
		},
		{
			name: "missing owner",
			config: GitHubLoaderConfig{
				Repo: "test-repo",
			},
			wantError: true,
		},
		{
			name: "missing repo",
			config: GitHubLoaderConfig{
				Owner: "test-owner",
			},
			wantError: true,
		},
		{
			name: "with custom branch",
			config: GitHubLoaderConfig{
				Owner:  "test-owner",
				Repo:   "test-repo",
				Branch: "develop",
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loader, err := NewGitHubLoader(tt.config)
			if tt.wantError {
				if err == nil {
					t.Error("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if loader == nil {
					t.Error("expected loader, got nil")
				}
				// 验证默认值
				if loader != nil && loader.config.Branch == "" {
					t.Error("expected default branch to be set")
				}
			}
		})
	}
}

func TestShouldExclude(t *testing.T) {
	loader := &GitHubLoader{
		config: GitHubLoaderConfig{
			ExcludePatterns: []string{"test", "vendor", ".git"},
		},
	}

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"normal file", "src/main.go", false},
		{"test file", "src/test/main_test.go", true},
		{"vendor", "vendor/pkg/lib.go", true},
		{"git", ".git/config", true},
		{"docs", "docs/README.md", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := loader.shouldExclude(tt.path)
			if result != tt.expected {
				t.Errorf("shouldExclude(%s) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestHasAllowedExtension(t *testing.T) {
	loader := &GitHubLoader{
		config: GitHubLoaderConfig{
			FileExtensions: []string{".md", ".txt", ".go"},
		},
	}

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"markdown", "README.md", true},
		{"text", "notes.txt", true},
		{"go", "main.go", true},
		{"python", "script.py", false},
		{"json", "config.json", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := loader.hasAllowedExtension(tt.path)
			if result != tt.expected {
				t.Errorf("hasAllowedExtension(%s) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestExtractLabels(t *testing.T) {
	loader := &GitHubLoader{}

	labels := []GitHubLabel{
		{Name: "bug", Color: "ff0000"},
		{Name: "enhancement", Color: "00ff00"},
		{Name: "documentation", Color: "0000ff"},
	}

	result := loader.extractLabels(labels)

	if len(result) != 3 {
		t.Errorf("expected 3 labels, got %d", len(result))
	}

	expected := []string{"bug", "enhancement", "documentation"}
	for i, name := range expected {
		if result[i] != name {
			t.Errorf("expected label %d to be %s, got %s", i, name, result[i])
		}
	}
}

// 集成测试（需要网络访问）
func TestGitHubLoaderIntegration(t *testing.T) {
	t.Skip("Integration test - requires network access")

	// 取消注释以运行集成测试
	// config := GitHubLoaderConfig{
	// 	Owner:  "langchain-ai",
	// 	Repo:   "langchain",
	// 	Branch: "main",
	// 	FileExtensions: []string{".md"},
	// 	ExcludePatterns: []string{"test"},
	// }
	//
	// loader, err := NewGitHubLoader(config)
	// if err != nil {
	// 	t.Fatalf("Failed to create loader: %v", err)
	// }
	//
	// ctx := context.Background()
	//
	// // 测试加载单个文件
	// doc, err := loader.LoadFile(ctx, "README.md")
	// if err != nil {
	// 	t.Fatalf("Failed to load file: %v", err)
	// }
	//
	// if doc.PageContent == "" {
	// 	t.Error("Expected non-empty content")
	// }
	//
	// t.Logf("Loaded file: %s (length: %d)", doc.Metadata["path"], len(doc.PageContent))
}
