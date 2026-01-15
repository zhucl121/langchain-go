package loaders

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// 创建测试用的简单 PDF 文件
// 注意：这里我们需要一个真实的 PDF 文件来测试
// 在实际测试中，你应该准备一些测试 PDF 文件

// getTestPDFFile 获取测试 PDF 文件路径
func getTestPDFFile(t *testing.T) string {
	testFiles := []string{
		filepath.Join("testdata", "sample.pdf"),
		filepath.Join("testdata", "数据治理全景解析与实践-蚂蚁数科.pdf"),
	}
	
	for _, file := range testFiles {
		if _, err := os.Stat(file); err == nil {
			return file
		}
	}
	
	t.Skip("No test PDF file found, skipping test")
	return ""
}

// TestPDFLoaderBasic 测试基础 PDF 加载。
func TestPDFLoaderBasic(t *testing.T) {
	testFile := getTestPDFFile(t)
	ctx := context.Background()

	t.Run("Load", func(t *testing.T) {
		loader := NewPDFLoader(testFile)
		docs, err := loader.Load(ctx)

		if err != nil {
			t.Fatalf("Failed to load PDF: %v", err)
		}

		if len(docs) != 1 {
			t.Errorf("Expected 1 document, got %d", len(docs))
		}

		if docs[0].Content == "" {
			t.Error("Document content is empty")
		}

		if docs[0].Metadata["type"] != "pdf" {
			t.Errorf("Expected type 'pdf', got %v", docs[0].Metadata["type"])
		}
	})

	t.Run("LoadAndSplit", func(t *testing.T) {
		loader := NewPDFLoader(testFile)
		docs, err := loader.LoadAndSplit(ctx)

		if err != nil {
			t.Fatalf("Failed to load and split PDF: %v", err)
		}

		if len(docs) == 0 {
			t.Error("No documents loaded")
		}

		// 每个文档应该代表一页
		for i, doc := range docs {
			if doc.Content == "" {
				t.Errorf("Document %d has empty content", i)
			}

			if _, ok := doc.Metadata["page"]; !ok {
				t.Errorf("Document %d missing 'page' metadata", i)
			}
		}
	})
}

// TestPDFLoaderWithPageRange 测试页面范围加载。
func TestPDFLoaderWithPageRange(t *testing.T) {
	testFile := getTestPDFFile(t)
	ctx := context.Background()

	t.Run("LoadFirstPage", func(t *testing.T) {
		loader := NewPDFLoader(testFile).WithPageRange(1, 1)
		docs, err := loader.LoadAndSplit(ctx)

		if err != nil {
			t.Fatalf("Failed to load first page: %v", err)
		}

		if len(docs) > 1 {
			t.Errorf("Expected at most 1 document, got %d", len(docs))
		}

		if len(docs) == 1 && docs[0].Metadata["page"] != 1 {
			t.Errorf("Expected page 1, got %v", docs[0].Metadata["page"])
		}
	})

	t.Run("LoadPageRange", func(t *testing.T) {
		loader := NewPDFLoader(testFile)
		docs, err := loader.LoadPageRange(ctx, 1, 2)

		if err != nil {
			t.Fatalf("Failed to load page range: %v", err)
		}

		if len(docs) > 2 {
			t.Errorf("Expected at most 2 documents, got %d", len(docs))
		}
	})
}

// TestPDFLoaderGetPageCount 测试获取页数。
func TestPDFLoaderGetPageCount(t *testing.T) {
	testFile := getTestPDFFile(t)
	loader := NewPDFLoader(testFile)
	pageCount, err := loader.GetPageCount()

	if err != nil {
		t.Fatalf("Failed to get page count: %v", err)
	}

	if pageCount <= 0 {
		t.Errorf("Invalid page count: %d", pageCount)
	}
}

// TestPDFLoaderExtractMetadata 测试元数据提取。
func TestPDFLoaderExtractMetadata(t *testing.T) {
	testFile := getTestPDFFile(t)
	loader := NewPDFLoader(testFile)
	metadata, err := loader.ExtractMetadata()

	if err != nil {
		t.Fatalf("Failed to extract metadata: %v", err)
	}

	if metadata["total_pages"] == nil {
		t.Error("Metadata missing 'total_pages'")
	}

	if metadata["file_path"] != testFile {
		t.Errorf("Expected file_path %s, got %v", testFile, metadata["file_path"])
	}
}

// TestPDFLoaderLoadByPages 测试按页加载。
func TestPDFLoaderLoadByPages(t *testing.T) {
	testFile := getTestPDFFile(t)
	ctx := context.Background()
	loader := NewPDFLoader(testFile)

	docs, err := loader.LoadByPages(ctx)
	if err != nil {
		t.Fatalf("Failed to load by pages: %v", err)
	}

	if len(docs) == 0 {
		t.Error("No documents loaded")
	}
}

// TestPDFLoaderConvenienceFunctions 测试便捷函数。
func TestPDFLoaderConvenienceFunctions(t *testing.T) {
	testFile := getTestPDFFile(t)

	t.Run("LoadPDF", func(t *testing.T) {
		doc, err := LoadPDF(testFile)
		if err != nil {
			t.Fatalf("LoadPDF failed: %v", err)
		}

		if doc == nil {
			t.Error("Document is nil")
		}

		if doc.Content == "" {
			t.Error("Document content is empty")
		}
	})

	t.Run("SplitPDFByPages", func(t *testing.T) {
		docs, err := SplitPDFByPages(testFile)
		if err != nil {
			t.Fatalf("SplitPDFByPages failed: %v", err)
		}

		if len(docs) == 0 {
			t.Error("No documents loaded")
		}
	})
}

// TestPDFLoaderInvalidFile 测试无效文件处理。
func TestPDFLoaderInvalidFile(t *testing.T) {
	ctx := context.Background()

	t.Run("NonExistentFile", func(t *testing.T) {
		loader := NewPDFLoader("non_existent_file.pdf")
		_, err := loader.Load(ctx)

		if err == nil {
			t.Error("Expected error for non-existent file, got nil")
		}
	})

	t.Run("InvalidPDF", func(t *testing.T) {
		// 创建一个不是 PDF 的文件
		tmpFile := filepath.Join(os.TempDir(), "invalid.pdf")
		err := os.WriteFile(tmpFile, []byte("This is not a PDF"), 0644)
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		defer os.Remove(tmpFile)

		loader := NewPDFLoader(tmpFile)
		_, err = loader.Load(ctx)

		if err == nil {
			t.Error("Expected error for invalid PDF, got nil")
		}
	})
}

// TestPDFLoaderGetPageRange 测试页面范围计算。
func TestPDFLoaderGetPageRange(t *testing.T) {
	tests := []struct {
		name        string
		pageRange   *PageRange
		totalPages  int
		expectStart int
		expectEnd   int
	}{
		{
			name:        "NoRange",
			pageRange:   nil,
			totalPages:  10,
			expectStart: 1,
			expectEnd:   10,
		},
		{
			name:        "ValidRange",
			pageRange:   &PageRange{Start: 2, End: 5},
			totalPages:  10,
			expectStart: 2,
			expectEnd:   5,
		},
		{
			name:        "EndZero",
			pageRange:   &PageRange{Start: 3, End: 0},
			totalPages:  10,
			expectStart: 3,
			expectEnd:   10,
		},
		{
			name:        "StartTooLarge",
			pageRange:   &PageRange{Start: 15, End: 20},
			totalPages:  10,
			expectStart: 10,
			expectEnd:   10,
		},
		{
			name:        "EndTooLarge",
			pageRange:   &PageRange{Start: 5, End: 20},
			totalPages:  10,
			expectStart: 5,
			expectEnd:   10,
		},
		{
			name:        "StartLessThanOne",
			pageRange:   &PageRange{Start: 0, End: 5},
			totalPages:  10,
			expectStart: 1,
			expectEnd:   5,
		},
		{
			name:        "EndLessThanStart",
			pageRange:   &PageRange{Start: 8, End: 5},
			totalPages:  10,
			expectStart: 8,
			expectEnd:   8,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loader := &PDFLoader{
				BaseLoader: NewBaseLoader("test.pdf"),
				pageRange:  tt.pageRange,
			}

			start, end := loader.getPageRange(tt.totalPages)

			if start != tt.expectStart {
				t.Errorf("Expected start %d, got %d", tt.expectStart, start)
			}

			if end != tt.expectEnd {
				t.Errorf("Expected end %d, got %d", tt.expectEnd, end)
			}
		})
	}
}

// TestPDFLoaderChaining 测试链式配置。
func TestPDFLoaderChaining(t *testing.T) {
	loader := NewPDFLoader("test.pdf").
		WithPassword("secret").
		WithExtractImages(true).
		WithPageRange(1, 5)

	if loader.password != "secret" {
		t.Errorf("Expected password 'secret', got '%s'", loader.password)
	}

	if !loader.extractImages {
		t.Error("Expected extractImages to be true")
	}

	if loader.pageRange == nil {
		t.Fatal("Expected pageRange to be set")
	}

	if loader.pageRange.Start != 1 || loader.pageRange.End != 5 {
		t.Errorf("Expected page range 1-5, got %d-%d", loader.pageRange.Start, loader.pageRange.End)
	}
}
