package loaders

import (
	"context"
	"fmt"
	"strings"

	"github.com/xuri/excelize/v2"
)

// ExcelLoader loads and parses Microsoft Excel files (.xlsx, .xls)
type ExcelLoader struct {
	*BaseLoader
	sheetName        string
	includeHeaders   bool
	includeSheetName bool
	headerRow        int
	skipRows         int
	maxRows          int
}

// ExcelLoaderOptions contains configuration options for Excel loader
type ExcelLoaderOptions struct {
	// Path is the file path to load
	Path string

	// SheetName specifies which sheet to load (empty = all sheets)
	SheetName string

	// IncludeHeaders includes column headers in the output
	IncludeHeaders bool

	// IncludeSheetName includes sheet name in metadata
	IncludeSheetName bool

	// HeaderRow specifies which row contains headers (default: 0)
	HeaderRow int

	// SkipRows skips the first N rows
	SkipRows int

	// MaxRows limits the number of rows to read (0 = unlimited)
	MaxRows int

	// Metadata to include with the document
	Metadata map[string]any
}

// NewExcelLoader creates a new Excel loader with the given options
func NewExcelLoader(options ExcelLoaderOptions) *ExcelLoader {
	base := &BaseLoader{
		path:     options.Path,
		metadata: options.Metadata,
	}

	if base.metadata == nil {
		base.metadata = make(map[string]any)
	}

	return &ExcelLoader{
		BaseLoader:       base,
		sheetName:        options.SheetName,
		includeHeaders:   options.IncludeHeaders,
		includeSheetName: options.IncludeSheetName,
		headerRow:        options.HeaderRow,
		skipRows:         options.SkipRows,
		maxRows:          options.MaxRows,
	}
}

// Load loads the Excel document and returns parsed documents
func (loader *ExcelLoader) Load(ctx context.Context) ([]*Document, error) {
	// Open Excel file
	file, err := excelize.OpenFile(loader.path)
	if err != nil {
		return nil, fmt.Errorf("failed to open Excel file: %w", err)
	}
	defer file.Close()

	var docs []*Document

	// Get sheet list
	sheets := file.GetSheetList()
	if len(sheets) == 0 {
		return nil, fmt.Errorf("no sheets found in Excel file")
	}

	// Determine which sheets to load
	var sheetsToLoad []string
	if loader.sheetName != "" {
		// Load specific sheet
		sheetsToLoad = []string{loader.sheetName}
	} else {
		// Load all sheets
		sheetsToLoad = sheets
	}

	// Load each sheet
	for _, sheetName := range sheetsToLoad {
		doc, err := loader.loadSheet(file, sheetName)
		if err != nil {
			// Skip sheets that can't be loaded
			continue
		}
		if doc != nil {
			docs = append(docs, doc)
		}
	}

	if len(docs) == 0 {
		return nil, fmt.Errorf("no data loaded from Excel file")
	}

	return docs, nil
}

// loadSheet loads a single sheet from the Excel file
func (loader *ExcelLoader) loadSheet(file *excelize.File, sheetName string) (*Document, error) {
	// Get all rows
	rows, err := file.GetRows(sheetName)
	if err != nil {
		return nil, fmt.Errorf("failed to get rows from sheet %s: %w", sheetName, err)
	}

	if len(rows) == 0 {
		return nil, nil
	}

	// Apply skip rows
	startRow := loader.skipRows
	if startRow >= len(rows) {
		return nil, nil
	}

	// Determine end row
	endRow := len(rows)
	if loader.maxRows > 0 {
		maxEndRow := startRow + loader.maxRows
		if maxEndRow < endRow {
			endRow = maxEndRow
		}
	}

	// Extract headers if needed
	var headers []string
	if loader.includeHeaders && loader.headerRow < len(rows) {
		headers = rows[loader.headerRow]
	}

	// Build text content
	var textBuilder strings.Builder

	// Add sheet name if requested
	if loader.includeSheetName {
		textBuilder.WriteString("Sheet: ")
		textBuilder.WriteString(sheetName)
		textBuilder.WriteString("\n\n")
	}

	// Add headers if included
	if len(headers) > 0 {
		textBuilder.WriteString(strings.Join(headers, " | "))
		textBuilder.WriteString("\n")
		textBuilder.WriteString(strings.Repeat("-", len(textBuilder.String())-1))
		textBuilder.WriteString("\n")
	}

	// Add data rows
	for i := startRow; i < endRow; i++ {
		// Skip header row if it's within data range
		if loader.includeHeaders && i == loader.headerRow {
			continue
		}

		row := rows[i]
		if len(row) == 0 {
			continue
		}

		// Clean empty cells at the end
		lastNonEmpty := len(row) - 1
		for lastNonEmpty >= 0 && strings.TrimSpace(row[lastNonEmpty]) == "" {
			lastNonEmpty--
		}

		if lastNonEmpty >= 0 {
			row = row[:lastNonEmpty+1]
			textBuilder.WriteString(strings.Join(row, " | "))
			textBuilder.WriteString("\n")
		}
	}

	content := strings.TrimSpace(textBuilder.String())
	if content == "" {
		return nil, nil
	}

	// Create metadata
	metadata := make(map[string]any)
	for k, v := range loader.metadata {
		metadata[k] = v
	}
	metadata["source"] = loader.path
	metadata["file_type"] = "excel"
	metadata["sheet_name"] = sheetName
	metadata["row_count"] = endRow - startRow

	if len(headers) > 0 {
		metadata["headers"] = headers
		metadata["column_count"] = len(headers)
	}

	return NewDocument(content, metadata), nil
}

// LoadAndSplit loads the Excel document and splits it into chunks
func (loader *ExcelLoader) LoadAndSplit(
	ctx context.Context,
	splitter TextSplitter,
) ([]*Document, error) {
	docs, err := loader.Load(ctx)
	if err != nil {
		return nil, err
	}

	return SplitDocuments(docs, splitter)
}

// CSVLoader loads CSV files (similar to Excel but simpler)
// Note: This is a convenience wrapper around ExcelLoader
type CSVLoader struct {
	*ExcelLoader
}

// CSVLoaderOptions contains configuration options for CSV loader
type CSVLoaderOptions struct {
	// Path is the file path to load
	Path string

	// IncludeHeaders includes column headers in the output
	IncludeHeaders bool

	// SkipRows skips the first N rows
	SkipRows int

	// MaxRows limits the number of rows to read (0 = unlimited)
	MaxRows int

	// Metadata to include with the document
	Metadata map[string]any
}

// NewCSVLoader creates a new CSV loader
func NewCSVLoader(options CSVLoaderOptions) *CSVLoader {
	excelOptions := ExcelLoaderOptions{
		Path:           options.Path,
		IncludeHeaders: options.IncludeHeaders,
		SkipRows:       options.SkipRows,
		MaxRows:        options.MaxRows,
		Metadata:       options.Metadata,
	}

	return &CSVLoader{
		ExcelLoader: NewExcelLoader(excelOptions),
	}
}

// ExcelMetadataExtractor extracts additional metadata from Excel files
type ExcelMetadataExtractor struct {
	path string
}

// NewExcelMetadataExtractor creates a new metadata extractor
func NewExcelMetadataExtractor(path string) *ExcelMetadataExtractor {
	return &ExcelMetadataExtractor{
		path: path,
	}
}

// ExtractMetadata extracts metadata from Excel file
func (extractor *ExcelMetadataExtractor) ExtractMetadata() (map[string]any, error) {
	file, err := excelize.OpenFile(extractor.path)
	if err != nil {
		return nil, fmt.Errorf("failed to open Excel file: %w", err)
	}
	defer file.Close()

	metadata := make(map[string]any)

	// Get basic info
	sheets := file.GetSheetList()
	metadata["sheet_count"] = len(sheets)
	metadata["sheet_names"] = sheets

	// Get properties
	props, err := file.GetDocProps()
	if err == nil {
		if props.Title != "" {
			metadata["title"] = props.Title
		}
		if props.Creator != "" {
			metadata["creator"] = props.Creator
		}
		if props.Subject != "" {
			metadata["subject"] = props.Subject
		}
		if props.Description != "" {
			metadata["description"] = props.Description
		}
		if props.Keywords != "" {
			metadata["keywords"] = props.Keywords
		}
		if props.Category != "" {
			metadata["category"] = props.Category
		}
		if props.Created != "" {
			metadata["created"] = props.Created
		}
		if props.Modified != "" {
			metadata["modified"] = props.Modified
		}
	}

	// Get row/column counts for each sheet
	sheetInfo := make(map[string]map[string]int)
	for _, sheetName := range sheets {
		rows, err := file.GetRows(sheetName)
		if err == nil {
			info := make(map[string]int)
			info["row_count"] = len(rows)

			// Get max column count
			maxCols := 0
			for _, row := range rows {
				if len(row) > maxCols {
					maxCols = len(row)
				}
			}
			info["column_count"] = maxCols

			sheetInfo[sheetName] = info
		}
	}
	metadata["sheet_info"] = sheetInfo

	return metadata, nil
}

// ExcelTableExtractor extracts tables with structured data
type ExcelTableExtractor struct {
	loader *ExcelLoader
}

// NewExcelTableExtractor creates a new table extractor
func NewExcelTableExtractor(loader *ExcelLoader) *ExcelTableExtractor {
	return &ExcelTableExtractor{
		loader: loader,
	}
}

// ExtractTable extracts data as a structured table
func (extractor *ExcelTableExtractor) ExtractTable(ctx context.Context, sheetName string) ([]map[string]any, error) {
	file, err := excelize.OpenFile(extractor.loader.path)
	if err != nil {
		return nil, fmt.Errorf("failed to open Excel file: %w", err)
	}
	defer file.Close()

	// Get all rows
	rows, err := file.GetRows(sheetName)
	if err != nil {
		return nil, fmt.Errorf("failed to get rows: %w", err)
	}

	if len(rows) == 0 {
		return nil, nil
	}

	// First row as headers
	headers := rows[0]
	if len(headers) == 0 {
		return nil, fmt.Errorf("no headers found")
	}

	// Extract data rows
	var result []map[string]any
	for i := 1; i < len(rows); i++ {
		row := rows[i]
		if len(row) == 0 {
			continue
		}

		rowData := make(map[string]any)
		for j := 0; j < len(headers) && j < len(row); j++ {
			if headers[j] != "" && row[j] != "" {
				rowData[headers[j]] = row[j]
			}
		}

		if len(rowData) > 0 {
			result = append(result, rowData)
		}
	}

	return result, nil
}
