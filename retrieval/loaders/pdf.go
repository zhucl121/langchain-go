package loaders

import (
	"context"
	"fmt"
	"strings"

	"github.com/ledongthuc/pdf"
)

// PDFLoader 是 PDF 文档加载器。
//
// 支持加载 PDF 文件并提取文本内容。
//
type PDFLoader struct {
	*BaseLoader
	password      string
	extractImages bool
	pageRange     *PageRange
}

// PageRange 定义要加载的页面范围。
type PageRange struct {
	Start int // 起始页（从 1 开始）
	End   int // 结束页（包含，0 表示到最后一页）
}

// NewPDFLoader 创建 PDF 加载器。
//
// 参数：
//   - filePath: PDF 文件路径
//
// 返回：
//   - *PDFLoader: PDF 加载器实例
//
func NewPDFLoader(filePath string) *PDFLoader {
	return &PDFLoader{
		BaseLoader:    NewBaseLoader(filePath),
		password:      "",
		extractImages: false,
		pageRange:     nil,
	}
}

// WithPassword 设置 PDF 密码。
//
// 用于打开加密的 PDF 文件。
//
func (pl *PDFLoader) WithPassword(password string) *PDFLoader {
	pl.password = password
	return pl
}

// WithExtractImages 设置是否提取图片信息。
//
// 注意：当前版本不支持图片提取，此选项保留用于未来扩展。
//
func (pl *PDFLoader) WithExtractImages(extract bool) *PDFLoader {
	pl.extractImages = extract
	return pl
}

// WithPageRange 设置要加载的页面范围。
//
// 参数：
//   - start: 起始页（从 1 开始）
//   - end: 结束页（包含，0 表示到最后一页）
//
func (pl *PDFLoader) WithPageRange(start, end int) *PDFLoader {
	pl.pageRange = &PageRange{
		Start: start,
		End:   end,
	}
	return pl
}

// Load 实现 DocumentLoader 接口。
//
// 返回整个 PDF 文档作为单个 Document，包含所有页面的文本。
//
func (pl *PDFLoader) Load(ctx context.Context) ([]*Document, error) {
	// 打开 PDF 文件
	file, reader, err := pdf.Open(pl.source)
	if file != nil {
		defer file.Close()
	}
	if err != nil {
		return nil, fmt.Errorf("pdf loader: failed to open file: %w", err)
	}

	// 获取总页数
	totalPages := reader.NumPage()

	// 确定要加载的页面范围
	startPage, endPage := pl.getPageRange(totalPages)

	// 提取所有页面的文本
	var allText strings.Builder
	pageTexts := make([]string, 0, endPage-startPage+1)

	for pageNum := startPage; pageNum <= endPage; pageNum++ {
		page := reader.Page(pageNum)
		if page.V.IsNull() {
			continue
		}

		// 提取页面文本
		text, err := page.GetPlainText(nil)
		if err != nil {
			// 记录错误但继续处理
			fmt.Printf("Warning: failed to extract text from page %d: %v\n", pageNum, err)
			continue
		}

		pageTexts = append(pageTexts, text)
		allText.WriteString(text)
		allText.WriteString("\n\n") // 页面之间添加分隔
	}

	// 创建文档
	doc := NewDocument(strings.TrimSpace(allText.String()), map[string]any{
		"source":      pl.source,
		"type":        "pdf",
		"total_pages": totalPages,
		"loaded_pages": map[string]int{
			"start": startPage,
			"end":   endPage,
		},
	})
	doc.Source = pl.source

	return []*Document{doc}, nil
}

// LoadAndSplit 实现 DocumentLoader 接口。
//
// 返回 PDF 的每一页作为单独的 Document。
//
func (pl *PDFLoader) LoadAndSplit(ctx context.Context) ([]*Document, error) {
	// 打开 PDF 文件
	file, reader, err := pdf.Open(pl.source)
	if file != nil {
		defer file.Close()
	}
	if err != nil {
		return nil, fmt.Errorf("pdf loader: failed to open file: %w", err)
	}

	// 获取总页数
	totalPages := reader.NumPage()

	// 确定要加载的页面范围
	startPage, endPage := pl.getPageRange(totalPages)

	// 为每一页创建一个 Document
	documents := make([]*Document, 0, endPage-startPage+1)

	for pageNum := startPage; pageNum <= endPage; pageNum++ {
		page := reader.Page(pageNum)
		if page.V.IsNull() {
			continue
		}

		// 提取页面文本
		text, err := page.GetPlainText(nil)
		if err != nil {
			fmt.Printf("Warning: failed to extract text from page %d: %v\n", pageNum, err)
			continue
		}

		// 跳过空白页
		trimmedText := strings.TrimSpace(text)
		if trimmedText == "" {
			continue
		}

		// 创建文档
		doc := NewDocument(trimmedText, map[string]any{
			"source":      pl.source,
			"type":        "pdf",
			"page":        pageNum,
			"total_pages": totalPages,
		})
		doc.Source = pl.source

		documents = append(documents, doc)
	}

	return documents, nil
}

// LoadByPages 按页加载 PDF，返回每页的 Document。
//
// 这是 LoadAndSplit 的别名，提供更明确的语义。
//
func (pl *PDFLoader) LoadByPages(ctx context.Context) ([]*Document, error) {
	return pl.LoadAndSplit(ctx)
}

// LoadPageRange 加载指定范围的页面。
//
// 参数：
//   - ctx: 上下文
//   - start: 起始页（从 1 开始）
//   - end: 结束页（包含，0 表示到最后一页）
//
// 返回：
//   - []*Document: 指定范围内的文档列表
//   - error: 错误
//
func (pl *PDFLoader) LoadPageRange(ctx context.Context, start, end int) ([]*Document, error) {
	// 临时设置页面范围
	originalRange := pl.pageRange
	pl.pageRange = &PageRange{Start: start, End: end}

	// 加载文档
	docs, err := pl.LoadAndSplit(ctx)

	// 恢复原始范围
	pl.pageRange = originalRange

	return docs, err
}

// GetPageCount 获取 PDF 的总页数。
//
// 不加载内容，只返回页数。
//
func (pl *PDFLoader) GetPageCount() (int, error) {
	file, reader, err := pdf.Open(pl.source)
	if file != nil {
		defer file.Close()
	}
	if err != nil {
		return 0, fmt.Errorf("pdf loader: failed to open file: %w", err)
	}

	return reader.NumPage(), nil
}

// ExtractMetadata 提取 PDF 元数据。
//
// 返回 PDF 的元数据信息，如标题、作者、创建日期等。
//
func (pl *PDFLoader) ExtractMetadata() (map[string]any, error) {
	file, reader, err := pdf.Open(pl.source)
	if file != nil {
		defer file.Close()
	}
	if err != nil {
		return nil, fmt.Errorf("pdf loader: failed to open file: %w", err)
	}

	metadata := make(map[string]any)
	metadata["total_pages"] = reader.NumPage()
	metadata["file_path"] = pl.source

	// 尝试提取文档信息
	// 注意：ledongthuc/pdf 库对元数据的支持有限
	// 这里只提供基础信息

	return metadata, nil
}

// getPageRange 获取要加载的页面范围。
//
// 返回实际的起始页和结束页（从 1 开始）。
//
func (pl *PDFLoader) getPageRange(totalPages int) (int, int) {
	if pl.pageRange == nil {
		return 1, totalPages
	}

	start := pl.pageRange.Start
	if start < 1 {
		start = 1
	}
	if start > totalPages {
		start = totalPages
	}

	end := pl.pageRange.End
	if end == 0 || end > totalPages {
		end = totalPages
	}
	if end < start {
		end = start
	}

	return start, end
}

// SplitByPages 将 PDF 按页分割。
//
// 这是一个便捷函数，等同于 LoadAndSplit。
//
func SplitPDFByPages(filePath string) ([]*Document, error) {
	loader := NewPDFLoader(filePath)
	return loader.LoadAndSplit(context.Background())
}

// LoadPDF 加载整个 PDF 文档。
//
// 这是一个便捷函数，等同于 Load。
//
func LoadPDF(filePath string) (*Document, error) {
	loader := NewPDFLoader(filePath)
	docs, err := loader.Load(context.Background())
	if err != nil {
		return nil, err
	}
	if len(docs) == 0 {
		return nil, fmt.Errorf("no documents loaded from PDF")
	}
	return docs[0], nil
}
