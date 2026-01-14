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
)

// Document 表示一个文档。
//
// Document 是 RAG 系统中的基本单元，包含内容和元数据。
//
type Document struct {
	// Content 文档内容
	Content string
	
	// Metadata 文档元数据
	Metadata map[string]any
	
	// Source 文档来源
	Source string
}

// NewDocument 创建文档。
func NewDocument(content string, metadata map[string]any) *Document {
	if metadata == nil {
		metadata = make(map[string]any)
	}
	return &Document{
		Content:  content,
		Metadata: metadata,
	}
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
	source string
}

// NewBaseLoader 创建基础加载器。
func NewBaseLoader(source string) *BaseLoader {
	return &BaseLoader{
		source: source,
	}
}

// GetSource 获取来源。
func (bl *BaseLoader) GetSource() string {
	return bl.source
}
