// Package loaders 提供各种格式的文档加载器。
//
// Document Loaders 负责从不同来源和格式加载文档，并转换为统一的 Document 结构。
//
// 支持的加载器类型：
//   - TextLoader: 纯文本文件
//   - MarkdownLoader: Markdown 文件
//   - JSONLoader: JSON 文件
//   - CSVLoader: CSV 文件
//   - DirectoryLoader: 批量加载目录
//
// 使用示例：
//
//	loader := loaders.NewTextLoader("document.txt")
//	docs, err := loader.Load(ctx)
//	for _, doc := range docs {
//	    fmt.Println(doc.Content)
//	}
//
package loaders

import (
	"context"

	"github.com/zhucl121/langchain-go/pkg/types"
)

// Document 是 types.Document 的别名，用于向后兼容
type Document = types.Document

// NewDocument 创建文档（委托给 types.NewDocument）
func NewDocument(content string, metadata map[string]any) *Document {
	return types.NewDocument(content, metadata)
}

// TextSplitter 是文本分割器接口(从 splitters 包引用)
// 为了避免循环依赖,这里定义接口
type TextSplitter interface {
	SplitText(text string) []string
	SplitDocuments(docs []*Document) []*Document
}

// DocumentLoader 是文档加载器接口。
//
// 所有加载器都必须实现此接口。
//
type DocumentLoader interface {
	// Load 加载文档
	//
	// 参数：
	//   - ctx: 上下文
	//
	// 返回：
	//   - []*Document: 文档列表
	//   - error: 错误
	//
	Load(ctx context.Context) ([]*Document, error)
	
	// LoadAndSplit 加载并分割文档
	//
	// 使用默认的分割器分割文档。
	//
	// 参数：
	//   - ctx: 上下文
	//
	// 返回：
	//   - []*Document: 分割后的文档列表
	//   - error: 错误
	//
	LoadAndSplit(ctx context.Context) ([]*Document, error)
}

// BaseLoader 提供加载器的基础实现。
type BaseLoader struct {
	source   string
	path     string
	metadata map[string]any
}

// NewBaseLoader 创建基础加载器。
func NewBaseLoader(source string) *BaseLoader {
	return &BaseLoader{
		source:   source,
		path:     source,
		metadata: make(map[string]any),
	}
}

// GetSource 获取来源。
func (bl *BaseLoader) GetSource() string {
	return bl.source
}

// GetPath 获取路径。
func (bl *BaseLoader) GetPath() string {
	return bl.path
}

// GetMetadata 获取元数据。
func (bl *BaseLoader) GetMetadata() map[string]any {
	return bl.metadata
}

// SplitDocuments 使用分割器分割文档列表
func SplitDocuments(docs []*Document, splitter TextSplitter) ([]*Document, error) {
	if splitter == nil {
		return docs, nil
	}
	return splitter.SplitDocuments(docs), nil
}
