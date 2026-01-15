package loaders

import (
	"archive/zip"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"strings"
)

// DOCXLoader loads and parses Microsoft Word (.docx) documents
type DOCXLoader struct {
	*BaseLoader
	extractStyles bool
	extractTables bool
}

// DOCXLoaderOptions contains configuration options for DOCX loader
type DOCXLoaderOptions struct {
	// Path is the file path to load
	Path string

	// ExtractStyles determines whether to extract text formatting information
	ExtractStyles bool

	// ExtractTables determines whether to parse tables separately
	ExtractTables bool

	// Metadata to include with the document
	Metadata map[string]any
}

// NewDOCXLoader creates a new DOCX loader with the given options
func NewDOCXLoader(options DOCXLoaderOptions) *DOCXLoader {
	base := &BaseLoader{
		path:     options.Path,
		metadata: options.Metadata,
	}

	if base.metadata == nil {
		base.metadata = make(map[string]any)
	}

	return &DOCXLoader{
		BaseLoader:    base,
		extractStyles: options.ExtractStyles,
		extractTables: options.ExtractTables,
	}
}

// Load loads the DOCX document and returns parsed documents
func (loader *DOCXLoader) Load(ctx context.Context) ([]*Document, error) {
	// Open the DOCX file as a ZIP archive
	reader, err := zip.OpenReader(loader.path)
	if err != nil {
		return nil, fmt.Errorf("failed to open DOCX file: %w", err)
	}
	defer reader.Close()

	// Extract text from document.xml
	text, err := loader.extractText(&reader.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to extract text: %w", err)
	}

	// Extract core properties
	coreProps, err := loader.extractCoreProperties(&reader.Reader)
	if err == nil {
		// Add core properties to metadata
		for k, v := range coreProps {
			loader.metadata[k] = v
		}
	}

	// Create document metadata
	metadata := make(map[string]any)
	for k, v := range loader.metadata {
		metadata[k] = v
	}
	metadata["source"] = loader.path
	metadata["file_type"] = "docx"

	// Create document
	doc := NewDocument(text, metadata)

	return []*Document{doc}, nil
}

// extractText extracts text content from document.xml
func (loader *DOCXLoader) extractText(zipReader *zip.Reader) (string, error) {
	// Find document.xml
	var documentXML *zip.File
	for _, file := range zipReader.File {
		if file.Name == "word/document.xml" {
			documentXML = file
			break
		}
	}

	if documentXML == nil {
		return "", fmt.Errorf("document.xml not found in DOCX file")
	}

	// Open document.xml
	rc, err := documentXML.Open()
	if err != nil {
		return "", fmt.Errorf("failed to open document.xml: %w", err)
	}
	defer rc.Close()

	// Read content
	content, err := io.ReadAll(rc)
	if err != nil {
		return "", fmt.Errorf("failed to read document.xml: %w", err)
	}

	// Parse XML and extract text
	text, err := loader.parseDocumentXML(content)
	if err != nil {
		return "", fmt.Errorf("failed to parse document.xml: %w", err)
	}

	return text, nil
}

// parseDocumentXML parses document.xml and extracts text
func (loader *DOCXLoader) parseDocumentXML(xmlContent []byte) (string, error) {
	var doc documentXML

	err := xml.Unmarshal(xmlContent, &doc)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal XML: %w", err)
	}

	var textBuilder strings.Builder

	// Extract text from paragraphs
	for _, para := range doc.Body.Paragraphs {
		paraText := extractParagraphText(para)
		if paraText != "" {
			textBuilder.WriteString(paraText)
			textBuilder.WriteString("\n")
		}
	}

	// Extract text from tables if enabled
	if loader.extractTables {
		for _, table := range doc.Body.Tables {
			tableText := extractTableText(table)
			if tableText != "" {
				textBuilder.WriteString(tableText)
				textBuilder.WriteString("\n")
			}
		}
	}

	return strings.TrimSpace(textBuilder.String()), nil
}

// extractParagraphText extracts text from a paragraph
func extractParagraphText(para paragraph) string {
	var textBuilder strings.Builder

	for _, run := range para.Runs {
		for _, text := range run.Texts {
			textBuilder.WriteString(text.Value)
		}
	}

	return textBuilder.String()
}

// extractTableText extracts text from a table
func extractTableText(table table) string {
	var textBuilder strings.Builder

	for _, row := range table.Rows {
		var rowTexts []string
		for _, cell := range row.Cells {
			var cellText string
			for _, para := range cell.Paragraphs {
				paraText := extractParagraphText(para)
				if paraText != "" {
					cellText += paraText + " "
				}
			}
			rowTexts = append(rowTexts, strings.TrimSpace(cellText))
		}
		textBuilder.WriteString(strings.Join(rowTexts, " | "))
		textBuilder.WriteString("\n")
	}

	return textBuilder.String()
}

// extractCoreProperties extracts document properties from docProps/core.xml
func (loader *DOCXLoader) extractCoreProperties(zipReader *zip.Reader) (map[string]any, error) {
	// Find core.xml
	var coreXML *zip.File
	for _, file := range zipReader.File {
		if file.Name == "docProps/core.xml" {
			coreXML = file
			break
		}
	}

	if coreXML == nil {
		return nil, fmt.Errorf("core.xml not found")
	}

	// Open core.xml
	rc, err := coreXML.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open core.xml: %w", err)
	}
	defer rc.Close()

	// Read content
	content, err := io.ReadAll(rc)
	if err != nil {
		return nil, fmt.Errorf("failed to read core.xml: %w", err)
	}

	// Parse core properties
	var props coreProperties
	err = xml.Unmarshal(content, &props)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal core.xml: %w", err)
	}

	// Build properties map
	result := make(map[string]any)

	if props.Title != "" {
		result["title"] = props.Title
	}
	if props.Creator != "" {
		result["creator"] = props.Creator
	}
	if props.Subject != "" {
		result["subject"] = props.Subject
	}
	if props.Description != "" {
		result["description"] = props.Description
	}
	if props.Keywords != "" {
		result["keywords"] = props.Keywords
	}
	if props.Created != "" {
		result["created"] = props.Created
	}
	if props.Modified != "" {
		result["modified"] = props.Modified
	}

	return result, nil
}

// LoadAndSplit loads the DOCX document and splits it into chunks
func (loader *DOCXLoader) LoadAndSplit(
	ctx context.Context,
	splitter TextSplitter,
) ([]*Document, error) {
	docs, err := loader.Load(ctx)
	if err != nil {
		return nil, err
	}

	return SplitDocuments(docs, splitter)
}

// XML structure definitions for DOCX

type documentXML struct {
	XMLName xml.Name `xml:"document"`
	Body    body     `xml:"body"`
}

type body struct {
	Paragraphs []paragraph `xml:"p"`
	Tables     []table     `xml:"tbl"`
}

type paragraph struct {
	Runs []run `xml:"r"`
}

type run struct {
	Texts []text `xml:"t"`
}

type text struct {
	Value string `xml:",chardata"`
}

type table struct {
	Rows []tableRow `xml:"tr"`
}

type tableRow struct {
	Cells []tableCell `xml:"tc"`
}

type tableCell struct {
	Paragraphs []paragraph `xml:"p"`
}

type coreProperties struct {
	XMLName     xml.Name `xml:"coreProperties"`
	Title       string   `xml:"title"`
	Creator     string   `xml:"creator"`
	Subject     string   `xml:"subject"`
	Description string   `xml:"description"`
	Keywords    string   `xml:"keywords"`
	Created     string   `xml:"created"`
	Modified    string   `xml:"modified"`
}

// DOCLoader loads legacy Microsoft Word (.doc) documents
// Note: This is a basic implementation that attempts to extract text
// from .doc files. For production use, consider using a more robust
// library or converting .doc to .docx first.
type DOCLoader struct {
	*BaseLoader
}

// NewDOCLoader creates a new DOC loader
func NewDOCLoader(path string, metadata map[string]any) *DOCLoader {
	return &DOCLoader{
		BaseLoader: &BaseLoader{
			path:     path,
			metadata: metadata,
		},
	}
}

// Load loads the DOC document
func (loader *DOCLoader) Load(ctx context.Context) ([]*Document, error) {
	// Read file
	data, err := os.ReadFile(loader.path)
	if err != nil {
		return nil, fmt.Errorf("failed to read DOC file: %w", err)
	}

	// Basic text extraction (this is a simplified approach)
	// For production use, consider using antiword or converting to DOCX
	text := extractTextFromDOC(data)

	// Create metadata
	metadata := make(map[string]any)
	if loader.metadata != nil {
		for k, v := range loader.metadata {
			metadata[k] = v
		}
	}
	metadata["source"] = loader.path
	metadata["file_type"] = "doc"

	doc := NewDocument(text, metadata)

	return []*Document{doc}, nil
}

// extractTextFromDOC performs basic text extraction from .doc files
// Note: This is a simplified implementation and may not work for all .doc files
func extractTextFromDOC(data []byte) string {
	// Filter out non-printable characters and extract visible text
	var textBuilder strings.Builder

	for i := 0; i < len(data); i++ {
		b := data[i]
		// Keep printable ASCII characters and common whitespace
		if (b >= 32 && b <= 126) || b == 10 || b == 13 || b == 9 {
			textBuilder.WriteByte(b)
		} else if b > 126 {
			// May be part of UTF-8 sequence
			textBuilder.WriteByte(b)
		}
	}

	// Clean up the text
	text := textBuilder.String()
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")

	// Remove excessive whitespace
	lines := strings.Split(text, "\n")
	var cleanLines []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			cleanLines = append(cleanLines, line)
		}
	}

	return strings.Join(cleanLines, "\n")
}

// LoadAndSplit loads the DOC document and splits it into chunks
func (loader *DOCLoader) LoadAndSplit(
	ctx context.Context,
	splitter TextSplitter,
) ([]*Document, error) {
	docs, err := loader.Load(ctx)
	if err != nil {
		return nil, err
	}

	return SplitDocuments(docs, splitter)
}
