package skills

import (
	"testing"
	"time"
)

func TestSkillExample(t *testing.T) {
	example := SkillExample{
		Input:     "test input",
		Output:    "test output",
		Reasoning: "test reasoning",
		Metadata:  map[string]any{"key": "value"},
	}

	if example.Input != "test input" {
		t.Errorf("Expected input 'test input', got '%s'", example.Input)
	}
}

func TestSkillMetadata(t *testing.T) {
	metadata := &SkillMetadata{
		Version:    "1.0.0",
		Author:     "Test Author",
		License:    "MIT",
		Repository: "https://github.com/test/repo",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		Extra:      map[string]any{"custom": "data"},
	}

	if metadata.Version != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got '%s'", metadata.Version)
	}

	if metadata.Author != "Test Author" {
		t.Errorf("Expected author 'Test Author', got '%s'", metadata.Author)
	}
}

func TestLoadConfig(t *testing.T) {
	config := &LoadConfig{
		Lazy:                 true,
		AutoLoadDependencies: true,
		Context:              map[string]any{"key": "value"},
	}

	if !config.Lazy {
		t.Error("Expected Lazy to be true")
	}

	if !config.AutoLoadDependencies {
		t.Error("Expected AutoLoadDependencies to be true")
	}

	if config.Context["key"] != "value" {
		t.Error("Expected Context to contain key-value pair")
	}
}

func TestDefaultLoadConfig(t *testing.T) {
	config := DefaultLoadConfig()

	if config == nil {
		t.Fatal("Expected non-nil config")
	}

	if config.Lazy {
		t.Error("Expected Lazy to be false by default")
	}

	if !config.AutoLoadDependencies {
		t.Error("Expected AutoLoadDependencies to be true by default")
	}

	if config.Context == nil {
		t.Error("Expected Context to be initialized")
	}
}

func TestSkillCategory(t *testing.T) {
	categories := []SkillCategory{
		CategoryCoding,
		CategoryDataAnalysis,
		CategoryKnowledge,
		CategoryCreative,
		CategoryResearch,
		CategoryAutomation,
		CategoryCommunication,
		CategoryGeneral,
	}

	for _, cat := range categories {
		if cat == "" {
			t.Errorf("Category should not be empty")
		}
	}
}
