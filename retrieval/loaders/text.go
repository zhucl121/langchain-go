package loaders

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// TextLoader 是文本文件加载器。
//
// 支持加载纯文本文件，包括 .txt、.md 等格式。
//
type TextLoader struct {
	*BaseLoader
	encoding string
}

// NewTextLoader 创建文本加载器。
//
// 参数：
//   - filePath: 文件路径
//
// 返回：
//   - *TextLoader: 文本加载器实例
//
func NewTextLoader(filePath string) *TextLoader {
	return &TextLoader{
		BaseLoader: NewBaseLoader(filePath),
		encoding:   "utf-8",
	}
}

// WithEncoding 设置编码。
func (tl *TextLoader) WithEncoding(encoding string) *TextLoader {
	tl.encoding = encoding
	return tl
}

// Load 实现 DocumentLoader 接口。
func (tl *TextLoader) Load(ctx context.Context) ([]*Document, error) {
	// 读取文件
	content, err := os.ReadFile(tl.source)
	if err != nil {
		return nil, fmt.Errorf("text loader: read file failed: %w", err)
	}
	
	// 创建文档
	doc := NewDocument(string(content), map[string]any{
		"source":   tl.source,
		"encoding": tl.encoding,
		"type":     "text",
	})
	doc.Source = tl.source
	
	return []*Document{doc}, nil
}

// LoadAndSplit 实现 DocumentLoader 接口。
func (tl *TextLoader) LoadAndSplit(ctx context.Context) ([]*Document, error) {
	// 简单实现：按段落分割
	docs, err := tl.Load(ctx)
	if err != nil {
		return nil, err
	}
	
	if len(docs) == 0 {
		return docs, nil
	}
	
	// 按双换行符分割段落
	paragraphs := strings.Split(docs[0].Content, "\n\n")
	result := make([]*Document, 0, len(paragraphs))
	
	for i, para := range paragraphs {
		para = strings.TrimSpace(para)
		if para == "" {
			continue
		}
		
		doc := NewDocument(para, map[string]any{
			"source":    tl.source,
			"paragraph": i,
			"type":      "text",
		})
		doc.Source = tl.source
		result = append(result, doc)
	}
	
	return result, nil
}

// MarkdownLoader 是 Markdown 文件加载器。
type MarkdownLoader struct {
	*TextLoader
}

// NewMarkdownLoader 创建 Markdown 加载器。
func NewMarkdownLoader(filePath string) *MarkdownLoader {
	return &MarkdownLoader{
		TextLoader: NewTextLoader(filePath),
	}
}

// Load 实现 DocumentLoader 接口。
func (ml *MarkdownLoader) Load(ctx context.Context) ([]*Document, error) {
	docs, err := ml.TextLoader.Load(ctx)
	if err != nil {
		return nil, err
	}
	
	// 更新元数据
	for _, doc := range docs {
		doc.Metadata["type"] = "markdown"
	}
	
	return docs, nil
}

// DirectoryLoader 是目录加载器。
//
// 批量加载目录下的所有文件。
//
type DirectoryLoader struct {
	*BaseLoader
	glob       string
	recursive  bool
	loaderFunc func(string) DocumentLoader
}

// NewDirectoryLoader 创建目录加载器。
//
// 参数：
//   - dirPath: 目录路径
//
// 返回：
//   - *DirectoryLoader: 目录加载器实例
//
func NewDirectoryLoader(dirPath string) *DirectoryLoader {
	return &DirectoryLoader{
		BaseLoader: NewBaseLoader(dirPath),
		glob:       "*",
		recursive:  false,
		loaderFunc: func(path string) DocumentLoader {
			return NewTextLoader(path)
		},
	}
}

// WithGlob 设置文件匹配模式。
func (dl *DirectoryLoader) WithGlob(pattern string) *DirectoryLoader {
	dl.glob = pattern
	return dl
}

// WithRecursive 设置是否递归加载子目录。
func (dl *DirectoryLoader) WithRecursive(recursive bool) *DirectoryLoader {
	dl.recursive = recursive
	return dl
}

// WithLoaderFunc 设置加载器函数。
//
// 用于为不同文件类型指定不同的加载器。
//
func (dl *DirectoryLoader) WithLoaderFunc(fn func(string) DocumentLoader) *DirectoryLoader {
	dl.loaderFunc = fn
	return dl
}

// Load 实现 DocumentLoader 接口。
func (dl *DirectoryLoader) Load(ctx context.Context) ([]*Document, error) {
	var allDocs []*Document
	
	// 遍历目录
	err := filepath.Walk(dl.source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		// 跳过目录
		if info.IsDir() {
			if !dl.recursive && path != dl.source {
				return filepath.SkipDir
			}
			return nil
		}
		
		// 检查文件是否匹配模式
		matched, err := filepath.Match(dl.glob, info.Name())
		if err != nil {
			return err
		}
		if !matched {
			return nil
		}
		
		// 使用加载器函数加载文件
		loader := dl.loaderFunc(path)
		docs, err := loader.Load(ctx)
		if err != nil {
			// 记录错误但继续处理其他文件
			fmt.Printf("Warning: failed to load %s: %v\n", path, err)
			return nil
		}
		
		allDocs = append(allDocs, docs...)
		return nil
	})
	
	if err != nil {
		return nil, fmt.Errorf("directory loader: walk failed: %w", err)
	}
	
	return allDocs, nil
}

// LoadAndSplit 实现 DocumentLoader 接口。
func (dl *DirectoryLoader) LoadAndSplit(ctx context.Context) ([]*Document, error) {
	var allDocs []*Document
	
	err := filepath.Walk(dl.source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		if info.IsDir() {
			if !dl.recursive && path != dl.source {
				return filepath.SkipDir
			}
			return nil
		}
		
		matched, err := filepath.Match(dl.glob, info.Name())
		if err != nil {
			return err
		}
		if !matched {
			return nil
		}
		
		loader := dl.loaderFunc(path)
		docs, err := loader.LoadAndSplit(ctx)
		if err != nil {
			fmt.Printf("Warning: failed to load and split %s: %v\n", path, err)
			return nil
		}
		
		allDocs = append(allDocs, docs...)
		return nil
	})
	
	if err != nil {
		return nil, fmt.Errorf("directory loader: walk failed: %w", err)
	}
	
	return allDocs, nil
}
