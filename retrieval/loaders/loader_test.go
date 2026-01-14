package loaders

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewDocument
func TestNewDocument(t *testing.T) {
	doc := NewDocument("test content", map[string]any{
		"key": "value",
	})
	
	assert.Equal(t, "test content", doc.Content)
	assert.Equal(t, "value", doc.Metadata["key"])
	assert.NotNil(t, doc.Metadata)
}

// TestNewDocument_NilMetadata
func TestNewDocument_NilMetadata(t *testing.T) {
	doc := NewDocument("test", nil)
	
	assert.NotNil(t, doc.Metadata)
	assert.Len(t, doc.Metadata, 0)
}

// TestTextLoader_Load
func TestTextLoader_Load(t *testing.T) {
	// 创建临时文件
	tmpFile, err := os.CreateTemp("", "test-*.txt")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	
	content := "This is a test document.\nWith multiple lines."
	_, err = tmpFile.WriteString(content)
	require.NoError(t, err)
	tmpFile.Close()
	
	// 加载文档
	loader := NewTextLoader(tmpFile.Name())
	docs, err := loader.Load(context.Background())
	
	assert.NoError(t, err)
	assert.Len(t, docs, 1)
	assert.Equal(t, content, docs[0].Content)
	assert.Equal(t, tmpFile.Name(), docs[0].Source)
	assert.Equal(t, "text", docs[0].Metadata["type"])
}

// TestTextLoader_LoadAndSplit
func TestTextLoader_LoadAndSplit(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-*.txt")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	
	content := "Paragraph 1.\nLine 2 of para 1.\n\nParagraph 2.\n\nParagraph 3."
	_, err = tmpFile.WriteString(content)
	require.NoError(t, err)
	tmpFile.Close()
	
	loader := NewTextLoader(tmpFile.Name())
	docs, err := loader.LoadAndSplit(context.Background())
	
	assert.NoError(t, err)
	assert.Equal(t, 3, len(docs))
	assert.Contains(t, docs[0].Content, "Paragraph 1")
	assert.Contains(t, docs[1].Content, "Paragraph 2")
	assert.Contains(t, docs[2].Content, "Paragraph 3")
}

// TestMarkdownLoader
func TestMarkdownLoader(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-*.md")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	
	content := "# Title\n\nThis is markdown content."
	_, err = tmpFile.WriteString(content)
	require.NoError(t, err)
	tmpFile.Close()
	
	loader := NewMarkdownLoader(tmpFile.Name())
	docs, err := loader.Load(context.Background())
	
	assert.NoError(t, err)
	assert.Len(t, docs, 1)
	assert.Equal(t, content, docs[0].Content)
	assert.Equal(t, "markdown", docs[0].Metadata["type"])
}

// TestDirectoryLoader
func TestDirectoryLoader(t *testing.T) {
	// 创建临时目录
	tmpDir, err := os.MkdirTemp("", "test-dir-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)
	
	// 创建多个文件
	file1 := filepath.Join(tmpDir, "doc1.txt")
	file2 := filepath.Join(tmpDir, "doc2.txt")
	
	err = os.WriteFile(file1, []byte("Content 1"), 0644)
	require.NoError(t, err)
	
	err = os.WriteFile(file2, []byte("Content 2"), 0644)
	require.NoError(t, err)
	
	// 加载目录
	loader := NewDirectoryLoader(tmpDir).WithGlob("*.txt")
	docs, err := loader.Load(context.Background())
	
	assert.NoError(t, err)
	assert.Len(t, docs, 2)
}

// TestDirectoryLoader_Recursive
func TestDirectoryLoader_Recursive(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test-dir-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)
	
	// 创建子目录
	subDir := filepath.Join(tmpDir, "subdir")
	err = os.Mkdir(subDir, 0755)
	require.NoError(t, err)
	
	// 在主目录和子目录创建文件
	err = os.WriteFile(filepath.Join(tmpDir, "doc1.txt"), []byte("Content 1"), 0644)
	require.NoError(t, err)
	
	err = os.WriteFile(filepath.Join(subDir, "doc2.txt"), []byte("Content 2"), 0644)
	require.NoError(t, err)
	
	// 递归加载
	loader := NewDirectoryLoader(tmpDir).
		WithGlob("*.txt").
		WithRecursive(true)
	docs, err := loader.Load(context.Background())
	
	assert.NoError(t, err)
	assert.Len(t, docs, 2)
}

// TestJSONLoader_SingleObject
func TestJSONLoader_SingleObject(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-*.json")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	
	jsonContent := `{"content": "This is JSON content", "author": "Test"}`
	_, err = tmpFile.WriteString(jsonContent)
	require.NoError(t, err)
	tmpFile.Close()
	
	loader := NewJSONLoader(tmpFile.Name())
	docs, err := loader.Load(context.Background())
	
	assert.NoError(t, err)
	assert.Len(t, docs, 1)
	assert.Equal(t, "This is JSON content", docs[0].Content)
	assert.Equal(t, "Test", docs[0].Metadata["author"])
	assert.Equal(t, "json", docs[0].Metadata["type"])
}

// TestJSONLoader_Array
func TestJSONLoader_Array(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-*.json")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	
	jsonContent := `[
		{"content": "Doc 1", "id": 1},
		{"content": "Doc 2", "id": 2}
	]`
	_, err = tmpFile.WriteString(jsonContent)
	require.NoError(t, err)
	tmpFile.Close()
	
	loader := NewJSONLoader(tmpFile.Name())
	docs, err := loader.Load(context.Background())
	
	assert.NoError(t, err)
	assert.Len(t, docs, 2)
	assert.Equal(t, "Doc 1", docs[0].Content)
	assert.Equal(t, float64(1), docs[0].Metadata["id"])
	assert.Equal(t, "Doc 2", docs[1].Content)
}

// TestCSVLoader
func TestCSVLoader(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-*.csv")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	
	csvContent := `name,age,description
Alice,30,Engineer
Bob,25,Designer`
	_, err = tmpFile.WriteString(csvContent)
	require.NoError(t, err)
	tmpFile.Close()
	
	loader := NewCSVLoader(tmpFile.Name())
	docs, err := loader.Load(context.Background())
	
	assert.NoError(t, err)
	assert.Len(t, docs, 2)
	assert.Contains(t, docs[0].Content, "Alice")
	assert.Equal(t, "Alice", docs[0].Metadata["name"])
	assert.Equal(t, "30", docs[0].Metadata["age"])
}

// TestCSVLoader_WithContentColumns
func TestCSVLoader_WithContentColumns(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-*.csv")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	
	csvContent := `name,age,description
Alice,30,Senior Engineer
Bob,25,Junior Designer`
	_, err = tmpFile.WriteString(csvContent)
	require.NoError(t, err)
	tmpFile.Close()
	
	loader := NewCSVLoader(tmpFile.Name()).
		WithContentColumns("name", "description")
	docs, err := loader.Load(context.Background())
	
	assert.NoError(t, err)
	assert.Len(t, docs, 2)
	// 内容应该只包含指定的列
	assert.Equal(t, "Alice\nSenior Engineer", docs[0].Content)
	assert.Equal(t, "Bob\nJunior Designer", docs[1].Content)
	// 但元数据应该包含所有列
	assert.Equal(t, "30", docs[0].Metadata["age"])
}
