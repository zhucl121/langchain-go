package loaders

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"
)

// TestExcelLoader tests Excel document loader
func TestExcelLoader(t *testing.T) {
	ctx := context.Background()

	t.Run("NewExcelLoader", func(t *testing.T) {
		loader := NewExcelLoader(ExcelLoaderOptions{
			Path:             "test.xlsx",
			SheetName:        "Sheet1",
			IncludeHeaders:   true,
			IncludeSheetName: true,
			HeaderRow:        0,
			SkipRows:         1,
			MaxRows:          100,
			Metadata:         map[string]any{"custom": "value"},
		})

		assert.NotNil(t, loader)
		assert.Equal(t, "test.xlsx", loader.path)
		assert.Equal(t, "Sheet1", loader.sheetName)
		assert.True(t, loader.includeHeaders)
		assert.True(t, loader.includeSheetName)
		assert.Equal(t, 0, loader.headerRow)
		assert.Equal(t, 1, loader.skipRows)
		assert.Equal(t, 100, loader.maxRows)
	})

	t.Run("Load_SimpleExcel", func(t *testing.T) {
		// Create test Excel file
		testFile := filepath.Join(t.TempDir(), "test.xlsx")
		err := createTestExcel(testFile, map[string][][]string{
			"Sheet1": {
				{"Name", "Age", "City"},
				{"Alice", "30", "New York"},
				{"Bob", "25", "Los Angeles"},
			},
		})
		require.NoError(t, err)

		loader := NewExcelLoader(ExcelLoaderOptions{
			Path:           testFile,
			IncludeHeaders: true,
		})

		docs, err := loader.Load(ctx)
		require.NoError(t, err)
		require.Len(t, docs, 1)

		assert.Contains(t, docs[0].Content, "Name")
		assert.Contains(t, docs[0].Content, "Alice")
		assert.Contains(t, docs[0].Content, "Bob")
		assert.Equal(t, testFile, docs[0].Metadata["source"])
		assert.Equal(t, "excel", docs[0].Metadata["file_type"])
	})

	t.Run("Load_MultipleSheets", func(t *testing.T) {
		testFile := filepath.Join(t.TempDir(), "test_multi.xlsx")
		err := createTestExcel(testFile, map[string][][]string{
			"Sheet1": {
				{"Data1"},
				{"Value1"},
			},
			"Sheet2": {
				{"Data2"},
				{"Value2"},
			},
		})
		require.NoError(t, err)

		loader := NewExcelLoader(ExcelLoaderOptions{
			Path: testFile,
		})

		docs, err := loader.Load(ctx)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(docs), 2)

		// Check that different sheets are loaded
		sheetNames := make(map[string]bool)
		for _, doc := range docs {
			sheetName := doc.Metadata["sheet_name"].(string)
			sheetNames[sheetName] = true
		}
		assert.True(t, sheetNames["Sheet1"] || sheetNames["Sheet2"])
	})

	t.Run("Load_SpecificSheet", func(t *testing.T) {
		testFile := filepath.Join(t.TempDir(), "test_specific.xlsx")
		err := createTestExcel(testFile, map[string][][]string{
			"Sheet1": {
				{"Data1"},
			},
			"Sheet2": {
				{"Data2"},
			},
		})
		require.NoError(t, err)

		loader := NewExcelLoader(ExcelLoaderOptions{
			Path:      testFile,
			SheetName: "Sheet2",
		})

		docs, err := loader.Load(ctx)
		require.NoError(t, err)
		require.Len(t, docs, 1)

		assert.Equal(t, "Sheet2", docs[0].Metadata["sheet_name"])
		assert.Contains(t, docs[0].Content, "Data2")
	})

	t.Run("Load_WithSkipRows", func(t *testing.T) {
		testFile := filepath.Join(t.TempDir(), "test_skip.xlsx")
		err := createTestExcel(testFile, map[string][][]string{
			"Sheet1": {
				{"Skip this row"},
				{"Header"},
				{"Data"},
			},
		})
		require.NoError(t, err)

		loader := NewExcelLoader(ExcelLoaderOptions{
			Path:     testFile,
			SkipRows: 1,
		})

		docs, err := loader.Load(ctx)
		require.NoError(t, err)
		require.Len(t, docs, 1)

		assert.NotContains(t, docs[0].Content, "Skip this row")
		assert.Contains(t, docs[0].Content, "Header")
	})

	t.Run("Load_WithMaxRows", func(t *testing.T) {
		testFile := filepath.Join(t.TempDir(), "test_max.xlsx")
		rows := [][]string{
			{"Row1"},
			{"Row2"},
			{"Row3"},
			{"Row4"},
			{"Row5"},
		}
		err := createTestExcel(testFile, map[string][][]string{
			"Sheet1": rows,
		})
		require.NoError(t, err)

		loader := NewExcelLoader(ExcelLoaderOptions{
			Path:    testFile,
			MaxRows: 2,
		})

		docs, err := loader.Load(ctx)
		require.NoError(t, err)
		require.Len(t, docs, 1)

		// Should only contain first 2 rows
		assert.Contains(t, docs[0].Content, "Row1")
		assert.Contains(t, docs[0].Content, "Row2")
		assert.NotContains(t, docs[0].Content, "Row3")
	})

	t.Run("Load_WithSheetName", func(t *testing.T) {
		testFile := filepath.Join(t.TempDir(), "test_sheetname.xlsx")
		err := createTestExcel(testFile, map[string][][]string{
			"MySheet": {
				{"Data"},
			},
		})
		require.NoError(t, err)

		loader := NewExcelLoader(ExcelLoaderOptions{
			Path:             testFile,
			IncludeSheetName: true,
		})

		docs, err := loader.Load(ctx)
		require.NoError(t, err)
		require.Len(t, docs, 1)

		assert.Contains(t, docs[0].Content, "Sheet: MySheet")
	})

	t.Run("Load_FileNotFound", func(t *testing.T) {
		loader := NewExcelLoader(ExcelLoaderOptions{
			Path: "nonexistent.xlsx",
		})

		docs, err := loader.Load(ctx)
		assert.Error(t, err)
		assert.Nil(t, docs)
	})

	t.Run("Load_EmptySheet", func(t *testing.T) {
		testFile := filepath.Join(t.TempDir(), "test_empty.xlsx")
		err := createTestExcel(testFile, map[string][][]string{
			"Sheet1": {},
		})
		require.NoError(t, err)

		loader := NewExcelLoader(ExcelLoaderOptions{
			Path: testFile,
		})

		docs, err := loader.Load(ctx)
		// Should handle empty sheets gracefully
		if err == nil {
			assert.Empty(t, docs)
		}
	})

	t.Run("LoadAndSplit", func(t *testing.T) {
		testFile := filepath.Join(t.TempDir(), "test_split.xlsx")
		// Create file with many rows
		var rows [][]string
		for i := 0; i < 100; i++ {
			rows = append(rows, []string{"Data", "More data", "Even more data"})
		}
		err := createTestExcel(testFile, map[string][][]string{
			"Sheet1": rows,
		})
		require.NoError(t, err)

		loader := NewExcelLoader(ExcelLoaderOptions{
			Path: testFile,
		})

		splitter := NewCharacterTextSplitter(CharacterTextSplitterOptions{
			ChunkSize:    200,
			ChunkOverlap: 20,
		})

		docs, err := loader.LoadAndSplit(ctx, splitter)
		require.NoError(t, err)
		assert.Greater(t, len(docs), 1)
	})
}

// TestCSVLoader tests CSV loader
func TestCSVLoader(t *testing.T) {
	ctx := context.Background()

	t.Run("NewCSVLoader", func(t *testing.T) {
		loader := NewCSVLoader(CSVLoaderOptions{
			Path:           "test.csv",
			IncludeHeaders: true,
			SkipRows:       1,
			MaxRows:        100,
		})

		assert.NotNil(t, loader)
		assert.NotNil(t, loader.ExcelLoader)
	})

	t.Run("Load_CSV", func(t *testing.T) {
		// Create test CSV file (Excel format)
		testFile := filepath.Join(t.TempDir(), "test.csv")
		err := createTestExcel(testFile, map[string][][]string{
			"Sheet1": {
				{"Name", "Value"},
				{"Item1", "100"},
				{"Item2", "200"},
			},
		})
		require.NoError(t, err)

		loader := NewCSVLoader(CSVLoaderOptions{
			Path:           testFile,
			IncludeHeaders: true,
		})

		docs, err := loader.Load(ctx)
		require.NoError(t, err)
		assert.NotEmpty(t, docs)
	})
}

// TestExcelMetadataExtractor tests metadata extraction
func TestExcelMetadataExtractor(t *testing.T) {
	t.Run("ExtractMetadata", func(t *testing.T) {
		testFile := filepath.Join(t.TempDir(), "test_metadata.xlsx")
		err := createTestExcelWithMetadata(testFile)
		require.NoError(t, err)

		extractor := NewExcelMetadataExtractor(testFile)
		metadata, err := extractor.ExtractMetadata()

		require.NoError(t, err)
		assert.NotNil(t, metadata)
		assert.Contains(t, metadata, "sheet_count")
		assert.Contains(t, metadata, "sheet_names")
	})

	t.Run("ExtractMetadata_FileNotFound", func(t *testing.T) {
		extractor := NewExcelMetadataExtractor("nonexistent.xlsx")
		metadata, err := extractor.ExtractMetadata()

		assert.Error(t, err)
		assert.Nil(t, metadata)
	})
}

// TestExcelTableExtractor tests table extraction
func TestExcelTableExtractor(t *testing.T) {
	ctx := context.Background()

	t.Run("ExtractTable", func(t *testing.T) {
		testFile := filepath.Join(t.TempDir(), "test_table.xlsx")
		err := createTestExcel(testFile, map[string][][]string{
			"Sheet1": {
				{"Name", "Age", "City"},
				{"Alice", "30", "New York"},
				{"Bob", "25", "Los Angeles"},
			},
		})
		require.NoError(t, err)

		loader := NewExcelLoader(ExcelLoaderOptions{
			Path: testFile,
		})

		extractor := NewExcelTableExtractor(loader)
		table, err := extractor.ExtractTable(ctx, "Sheet1")

		require.NoError(t, err)
		assert.Len(t, table, 2)

		// Check first row
		assert.Equal(t, "Alice", table[0]["Name"])
		assert.Equal(t, "30", table[0]["Age"])
		assert.Equal(t, "New York", table[0]["City"])

		// Check second row
		assert.Equal(t, "Bob", table[1]["Name"])
		assert.Equal(t, "25", table[1]["Age"])
	})

	t.Run("ExtractTable_EmptySheet", func(t *testing.T) {
		testFile := filepath.Join(t.TempDir(), "test_empty_table.xlsx")
		err := createTestExcel(testFile, map[string][][]string{
			"Sheet1": {},
		})
		require.NoError(t, err)

		loader := NewExcelLoader(ExcelLoaderOptions{
			Path: testFile,
		})

		extractor := NewExcelTableExtractor(loader)
		table, err := extractor.ExtractTable(ctx, "Sheet1")

		// Should handle empty sheet gracefully
		if err == nil {
			assert.Empty(t, table)
		}
	})
}

// Helper function to create test Excel files

func createTestExcel(path string, sheets map[string][][]string) error {
	file := excelize.NewFile()
	defer file.Close()

	// Delete default sheet
	file.DeleteSheet("Sheet1")

	// Create sheets
	for sheetName, rows := range sheets {
		index, err := file.NewSheet(sheetName)
		if err != nil {
			return err
		}

		// Write rows
		for i, row := range rows {
			for j, cell := range row {
				cellName, _ := excelize.CoordinatesToCellName(j+1, i+1)
				file.SetCellValue(sheetName, cellName, cell)
			}
		}

		// Set as active sheet (first one)
		if index == 1 {
			file.SetActiveSheet(index)
		}
	}

	return file.SaveAs(path)
}

func createTestExcelWithMetadata(path string) error {
	file := excelize.NewFile()
	defer file.Close()

	// Set document properties
	err := file.SetDocProps(&excelize.DocProperties{
		Title:       "Test Document",
		Creator:     "Test Author",
		Subject:     "Testing",
		Description: "A test document",
		Keywords:    "test, excel, metadata",
		Category:    "Test Category",
	})
	if err != nil {
		return err
	}

	// Create some sheets
	file.NewSheet("Sheet1")
	file.NewSheet("Sheet2")

	// Add some data
	file.SetCellValue("Sheet1", "A1", "Data")

	return file.SaveAs(path)
}
