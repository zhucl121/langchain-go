package loaders

import (
	"archive/zip"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDOCXLoader tests DOCX document loader
func TestDOCXLoader(t *testing.T) {
	ctx := context.Background()

	t.Run("NewDOCXLoader", func(t *testing.T) {
		loader := NewDOCXLoader(DOCXLoaderOptions{
			Path:          "test.docx",
			ExtractStyles: true,
			ExtractTables: true,
			Metadata:      map[string]any{"custom": "value"},
		})

		assert.NotNil(t, loader)
		assert.Equal(t, "test.docx", loader.path)
		assert.True(t, loader.extractStyles)
		assert.True(t, loader.extractTables)
		assert.Equal(t, "value", loader.metadata["custom"])
	})

	t.Run("NewDOCXLoader_DefaultOptions", func(t *testing.T) {
		loader := NewDOCXLoader(DOCXLoaderOptions{
			Path: "test.docx",
		})

		assert.NotNil(t, loader)
		assert.False(t, loader.extractStyles)
		assert.False(t, loader.extractTables)
		assert.NotNil(t, loader.metadata)
	})

	t.Run("Load_SimpleDOCX", func(t *testing.T) {
		// Create a test DOCX file
		testFile := filepath.Join(t.TempDir(), "test.docx")
		err := createTestDOCX(testFile, "Hello World\nThis is a test document.")
		require.NoError(t, err)

		loader := NewDOCXLoader(DOCXLoaderOptions{
			Path: testFile,
		})

		docs, err := loader.Load(ctx)
		require.NoError(t, err)
		require.Len(t, docs, 1)

		assert.Contains(t, docs[0].Content, "Hello World")
		assert.Contains(t, docs[0].Content, "test document")
		assert.Equal(t, testFile, docs[0].Metadata["source"])
		assert.Equal(t, "docx", docs[0].Metadata["file_type"])
	})

	t.Run("Load_WithTables", func(t *testing.T) {
		// Create a test DOCX file with table
		testFile := filepath.Join(t.TempDir(), "test_table.docx")
		err := createTestDOCXWithTable(testFile)
		require.NoError(t, err)

		loader := NewDOCXLoader(DOCXLoaderOptions{
			Path:          testFile,
			ExtractTables: true,
		})

		docs, err := loader.Load(ctx)
		require.NoError(t, err)
		require.Len(t, docs, 1)

		// Should contain table data
		assert.NotEmpty(t, docs[0].Content)
	})

	t.Run("Load_FileNotFound", func(t *testing.T) {
		loader := NewDOCXLoader(DOCXLoaderOptions{
			Path: "nonexistent.docx",
		})

		docs, err := loader.Load(ctx)
		assert.Error(t, err)
		assert.Nil(t, docs)
	})

	t.Run("Load_InvalidDOCX", func(t *testing.T) {
		// Create an invalid DOCX file
		testFile := filepath.Join(t.TempDir(), "invalid.docx")
		err := os.WriteFile(testFile, []byte("not a valid docx"), 0644)
		require.NoError(t, err)

		loader := NewDOCXLoader(DOCXLoaderOptions{
			Path: testFile,
		})

		docs, err := loader.Load(ctx)
		assert.Error(t, err)
		assert.Nil(t, docs)
	})

	t.Run("extractParagraphText", func(t *testing.T) {
		para := paragraph{
			Runs: []run{
				{
					Texts: []text{
						{Value: "Hello "},
						{Value: "World"},
					},
				},
				{
					Texts: []text{
						{Value: "!"},
					},
				},
			},
		}

		result := extractParagraphText(para)
		assert.Equal(t, "Hello World!", result)
	})

	t.Run("extractParagraphText_Empty", func(t *testing.T) {
		para := paragraph{
			Runs: []run{},
		}

		result := extractParagraphText(para)
		assert.Empty(t, result)
	})

	t.Run("extractTableText", func(t *testing.T) {
		table := table{
			Rows: []tableRow{
				{
					Cells: []tableCell{
						{
							Paragraphs: []paragraph{
								{
									Runs: []run{
										{Texts: []text{{Value: "Cell 1"}}},
									},
								},
							},
						},
						{
							Paragraphs: []paragraph{
								{
									Runs: []run{
										{Texts: []text{{Value: "Cell 2"}}},
									},
								},
							},
						},
					},
				},
			},
		}

		result := extractTableText(table)
		assert.Contains(t, result, "Cell 1")
		assert.Contains(t, result, "Cell 2")
		assert.Contains(t, result, "|")
	})

	t.Run("LoadAndSplit", func(t *testing.T) {
		// Create a test DOCX file with long content
		testFile := filepath.Join(t.TempDir(), "test_split.docx")
		longText := strings.Repeat("This is a sentence. ", 100)
		err := createTestDOCX(testFile, longText)
		require.NoError(t, err)

		loader := NewDOCXLoader(DOCXLoaderOptions{
			Path: testFile,
		})

		splitter := NewCharacterTextSplitter(CharacterTextSplitterOptions{
			ChunkSize:    100,
			ChunkOverlap: 20,
		})

		docs, err := loader.LoadAndSplit(ctx, splitter)
		require.NoError(t, err)
		assert.Greater(t, len(docs), 1)

		// All chunks should have metadata
		for _, doc := range docs {
			assert.Equal(t, testFile, doc.Metadata["source"])
			assert.Equal(t, "docx", doc.Metadata["file_type"])
		}
	})
}

// TestDOCLoader tests DOC document loader
func TestDOCLoader(t *testing.T) {
	ctx := context.Background()

	t.Run("NewDOCLoader", func(t *testing.T) {
		loader := NewDOCLoader("test.doc", map[string]any{"custom": "value"})

		assert.NotNil(t, loader)
		assert.Equal(t, "test.doc", loader.path)
		assert.Equal(t, "value", loader.metadata["custom"])
	})

	t.Run("Load_SimpleDOC", func(t *testing.T) {
		// Create a test DOC file (simplified)
		testFile := filepath.Join(t.TempDir(), "test.doc")
		content := "Hello World\nThis is a test document."
		err := os.WriteFile(testFile, []byte(content), 0644)
		require.NoError(t, err)

		loader := NewDOCLoader(testFile, nil)

		docs, err := loader.Load(ctx)
		require.NoError(t, err)
		require.Len(t, docs, 1)

		assert.NotEmpty(t, docs[0].Content)
		assert.Equal(t, testFile, docs[0].Metadata["source"])
		assert.Equal(t, "doc", docs[0].Metadata["file_type"])
	})

	t.Run("Load_FileNotFound", func(t *testing.T) {
		loader := NewDOCLoader("nonexistent.doc", nil)

		docs, err := loader.Load(ctx)
		assert.Error(t, err)
		assert.Nil(t, docs)
	})

	t.Run("extractTextFromDOC", func(t *testing.T) {
		data := []byte("Hello World\n\nThis is a test.")
		result := extractTextFromDOC(data)

		assert.Contains(t, result, "Hello World")
		assert.Contains(t, result, "This is a test")
	})

	t.Run("extractTextFromDOC_WithNonPrintable", func(t *testing.T) {
		data := []byte("Hello\x00\x01\x02World")
		result := extractTextFromDOC(data)

		// Non-printable characters should be filtered
		assert.NotContains(t, result, "\x00")
		assert.NotContains(t, result, "\x01")
	})

	t.Run("LoadAndSplit", func(t *testing.T) {
		testFile := filepath.Join(t.TempDir(), "test_split.doc")
		longText := strings.Repeat("This is a sentence. ", 100)
		err := os.WriteFile(testFile, []byte(longText), 0644)
		require.NoError(t, err)

		loader := NewDOCLoader(testFile, nil)

		splitter := NewCharacterTextSplitter(CharacterTextSplitterOptions{
			ChunkSize:    100,
			ChunkOverlap: 20,
		})

		docs, err := loader.LoadAndSplit(ctx, splitter)
		require.NoError(t, err)
		assert.Greater(t, len(docs), 1)
	})
}

// Helper functions for creating test DOCX files

func createTestDOCX(path string, content string) error {
	// Create a minimal DOCX file
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	zipWriter := zip.NewWriter(file)
	defer zipWriter.Close()

	// Create document.xml
	documentXML := createDocumentXML(content)
	w, err := zipWriter.Create("word/document.xml")
	if err != nil {
		return err
	}
	_, err = w.Write([]byte(documentXML))
	if err != nil {
		return err
	}

	// Create core.xml
	coreXML := createCoreXML()
	w, err = zipWriter.Create("docProps/core.xml")
	if err != nil {
		return err
	}
	_, err = w.Write([]byte(coreXML))
	if err != nil {
		return err
	}

	return nil
}

func createTestDOCXWithTable(path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	zipWriter := zip.NewWriter(file)
	defer zipWriter.Close()

	// Create document.xml with table
	documentXML := createDocumentXMLWithTable()
	w, err := zipWriter.Create("word/document.xml")
	if err != nil {
		return err
	}
	_, err = w.Write([]byte(documentXML))
	if err != nil {
		return err
	}

	return nil
}

func createDocumentXML(content string) string {
	paragraphs := strings.Split(content, "\n")
	var paraXML strings.Builder

	for _, para := range paragraphs {
		if para != "" {
			paraXML.WriteString("<w:p><w:r><w:t>")
			paraXML.WriteString(para)
			paraXML.WriteString("</w:t></w:r></w:p>")
		}
	}

	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>` + paraXML.String() + `</w:body>
</w:document>`
}

func createDocumentXMLWithTable() string {
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>
    <w:tbl>
      <w:tr>
        <w:tc>
          <w:p><w:r><w:t>Row 1 Cell 1</w:t></w:r></w:p>
        </w:tc>
        <w:tc>
          <w:p><w:r><w:t>Row 1 Cell 2</w:t></w:r></w:p>
        </w:tc>
      </w:tr>
      <w:tr>
        <w:tc>
          <w:p><w:r><w:t>Row 2 Cell 1</w:t></w:r></w:p>
        </w:tc>
        <w:tc>
          <w:p><w:r><w:t>Row 2 Cell 2</w:t></w:r></w:p>
        </w:tc>
      </w:tr>
    </w:tbl>
  </w:body>
</w:document>`
}

func createCoreXML() string {
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<cp:coreProperties xmlns:cp="http://schemas.openxmlformats.org/package/2006/metadata/core-properties"
                   xmlns:dc="http://purl.org/dc/elements/1.1/"
                   xmlns:dcterms="http://purl.org/dc/terms/">
  <dc:title>Test Document</dc:title>
  <dc:creator>Test Author</dc:creator>
  <dc:subject>Testing</dc:subject>
  <dc:description>A test document</dc:description>
</cp:coreProperties>`
}
