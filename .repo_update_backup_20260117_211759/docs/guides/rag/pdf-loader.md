# PDF 文档加载器使用指南

**创建日期**: 2026-01-15  
**版本**: v1.0  
**状态**: ✅ 已完成

---

## 📋 简介

PDF 文档加载器是 LangChain-Go 的文档加载器系列的一部分，专门用于加载和处理 PDF 文件。它支持从 PDF 文件中提取文本内容，并将其转换为 RAG 系统可用的 Document 格式。

### 核心特性

- ✅ **文本提取** - 提取 PDF 中的所有文本内容
- ✅ **分页加载** - 支持按页分割或整体加载
- ✅ **页面范围** - 可指定加载特定页面范围
- ✅ **元数据提取** - 自动提取页码、总页数等信息
- ✅ **链式配置** - 支持流畅的配置API

---

## 🚀 快速开始

### 基础使用

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/zhucl121/langchain-go/retrieval/loaders"
)

func main() {
    ctx := context.Background()

    // 1. 创建 PDF 加载器
    loader := loaders.NewPDFLoader("document.pdf")

    // 2. 加载整个 PDF
    docs, err := loader.Load(ctx)
    if err != nil {
        log.Fatal(err)
    }

    // 3. 打印内容
    for i, doc := range docs {
        fmt.Printf("Document %d:\n", i+1)
        fmt.Printf("Content: %s\n", doc.Content)
        fmt.Printf("Pages: %v\n\n", doc.Metadata["total_pages"])
    }
}
```

### 按页加载

```go
// 将 PDF 的每一页作为独立的 Document
loader := loaders.NewPDFLoader("document.pdf")
pages, err := loader.LoadAndSplit(ctx)

for _, page := range pages {
    fmt.Printf("Page %v: %s\n", page.Metadata["page"], page.Content[:100])
}
```

---

## ⚙️ 配置选项

### 1. 页面范围

```go
// 只加载前 5 页
loader := loaders.NewPDFLoader("document.pdf").
    WithPageRange(1, 5)

docs, err := loader.Load(ctx)
```

```go
// 加载第 10 页到最后一页
loader := loaders.NewPDFLoader("document.pdf").
    WithPageRange(10, 0) // 0 表示到最后一页

docs, err := loader.Load(ctx)
```

### 2. 密码保护的 PDF

```go
// 加载加密的 PDF
loader := loaders.NewPDFLoader("encrypted.pdf").
    WithPassword("secret123")

docs, err := loader.Load(ctx)
```

### 3. 链式配置

```go
// 组合多个配置
loader := loaders.NewPDFLoader("document.pdf").
    WithPassword("secret").
    WithPageRange(1, 10).
    WithExtractImages(true) // 未来功能

docs, err := loader.Load(ctx)
```

---

## 💡 使用场景

### 1. 学术论文处理

```go
func processPaper(paperPath string) error {
    ctx := context.Background()
    loader := loaders.NewPDFLoader(paperPath)

    // 按页加载，便于引用特定页面
    pages, err := loader.LoadAndSplit(ctx)
    if err != nil {
        return err
    }

    for _, page := range pages {
        pageNum := page.Metadata["page"]
        fmt.Printf("Processing page %v\n", pageNum)

        // 这里可以进行文本分析、向量化等操作
        // ...
    }

    return nil
}
```

### 2. 法律文档分析

```go
func analyzeLegalDocument(docPath string) ([]*loaders.Document, error) {
    ctx := context.Background()

    // 提取文档元数据
    loader := loaders.NewPDFLoader(docPath)
    metadata, err := loader.ExtractMetadata()
    if err != nil {
        return nil, err
    }

    fmt.Printf("Document has %v pages\n", metadata["total_pages"])

    // 加载整个文档
    docs, err := loader.Load(ctx)
    if err != nil {
        return nil, err
    }

    return docs, nil
}
```

### 3. 批量处理合同

```go
func processContracts(contractDir string) error {
    // 使用目录加载器批量处理
    dirLoader := loaders.NewDirectoryLoader(contractDir).
        WithGlob("*.pdf").
        WithRecursive(false).
        WithLoaderFunc(func(path string) loaders.DocumentLoader {
            // 为每个 PDF 使用自定义加载配置
            return loaders.NewPDFLoader(path).
                WithPageRange(1, 0) // 加载所有页
        })

    docs, err := dirLoader.Load(context.Background())
    if err != nil {
        return err
    }

    fmt.Printf("Loaded %d documents from contracts\n", len(docs))
    return nil
}
```

### 4. 提取特定章节

```go
// 提取第 5-10 页（假设是某个章节）
func extractChapter(bookPath string, startPage, endPage int) (*loaders.Document, error) {
    loader := loaders.NewPDFLoader(bookPath)

    pages, err := loader.LoadPageRange(context.Background(), startPage, endPage)
    if err != nil {
        return nil, err
    }

    // 合并所有页面
    var content strings.Builder
    for _, page := range pages {
        content.WriteString(page.Content)
        content.WriteString("\n\n")
    }

    // 创建章节文档
    chapter := loaders.NewDocument(content.String(), map[string]any{
        "source":      bookPath,
        "type":        "pdf_chapter",
        "start_page":  startPage,
        "end_page":    endPage,
    })

    return chapter, nil
}
```

---

## 🎓 高级用法

### 1. 与向量存储结合

```go
import (
    "github.com/zhucl121/langchain-go/retrieval/embeddings"
    "github.com/zhucl121/langchain-go/retrieval/loaders"
    "github.com/zhucl121/langchain-go/retrieval/vectorstores"
)

func indexPDF(pdfPath string) error {
    ctx := context.Background()

    // 1. 加载 PDF
    loader := loaders.NewPDFLoader(pdfPath)
    docs, err := loader.LoadAndSplit(ctx)
    if err != nil {
        return err
    }

    // 2. 创建向量存储
    emb := embeddings.NewOpenAIEmbeddings("your-api-key")
    store := vectorstores.NewInMemoryVectorStore(emb)

    // 3. 添加文档
    _, err = store.AddDocuments(ctx, docs)
    if err != nil {
        return err
    }

    // 4. 搜索
    results, err := store.SimilaritySearch(ctx, "your query", 5)
    if err != nil {
        return err
    }

    for _, doc := range results {
        fmt.Printf("Page %v: %s\n", doc.Metadata["page"], doc.Content[:100])
    }

    return nil
}
```

### 2. 与文本分割器结合

```go
import "github.com/zhucl121/langchain-go/retrieval/splitters"

func splitPDFByChunks(pdfPath string) ([]*loaders.Document, error) {
    ctx := context.Background()

    // 1. 加载 PDF
    loader := loaders.NewPDFLoader(pdfPath)
    docs, err := loader.Load(ctx) // 加载为单个文档
    if err != nil {
        return nil, err
    }

    // 2. 使用文本分割器
    splitter := splitters.NewRecursiveCharacterTextSplitter().
        WithChunkSize(1000).
        WithChunkOverlap(200)

    // 3. 分割文档
    chunks, err := splitter.SplitDocuments(docs)
    if err != nil {
        return nil, err
    }

    fmt.Printf("Split into %d chunks\n", len(chunks))
    return chunks, nil
}
```

### 3. 获取PDF信息

```go
func getPDFInfo(pdfPath string) error {
    loader := loaders.NewPDFLoader(pdfPath)

    // 获取页数（无需加载内容）
    pageCount, err := loader.GetPageCount()
    if err != nil {
        return err
    }
    fmt.Printf("Total pages: %d\n", pageCount)

    // 获取元数据
    metadata, err := loader.ExtractMetadata()
    if err != nil {
        return err
    }
    fmt.Printf("Metadata: %+v\n", metadata)

    return nil
}
```

### 4. 便捷函数

```go
// 快速加载整个 PDF
doc, err := loaders.LoadPDF("document.pdf")
if err != nil {
    log.Fatal(err)
}
fmt.Println(doc.Content)

// 快速按页分割
pages, err := loaders.SplitPDFByPages("document.pdf")
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Loaded %d pages\n", len(pages))
```

---

## 📊 API 参考

### PDFLoader 方法

| 方法 | 描述 | 返回值 |
|------|------|--------|
| `NewPDFLoader(filePath)` | 创建 PDF 加载器 | `*PDFLoader` |
| `Load(ctx)` | 加载整个 PDF | `[]*Document, error` |
| `LoadAndSplit(ctx)` | 按页加载 | `[]*Document, error` |
| `LoadByPages(ctx)` | 按页加载（别名） | `[]*Document, error` |
| `LoadPageRange(ctx, start, end)` | 加载指定页面范围 | `[]*Document, error` |
| `GetPageCount()` | 获取总页数 | `int, error` |
| `ExtractMetadata()` | 提取元数据 | `map[string]any, error` |

### 配置方法

| 方法 | 描述 | 返回值 |
|------|------|--------|
| `WithPassword(password)` | 设置 PDF 密码 | `*PDFLoader` |
| `WithPageRange(start, end)` | 设置页面范围 | `*PDFLoader` |
| `WithExtractImages(extract)` | 设置是否提取图片 | `*PDFLoader` |

### 便捷函数

| 函数 | 描述 | 返回值 |
|------|------|--------|
| `LoadPDF(filePath)` | 快速加载 PDF | `*Document, error` |
| `SplitPDFByPages(filePath)` | 快速按页分割 | `[]*Document, error` |

---

## 🔧 Document 元数据

### Load() 返回的元数据

```go
{
    "source":       "document.pdf",
    "type":         "pdf",
    "total_pages":  10,
    "loaded_pages": {
        "start": 1,
        "end":   10,
    },
}
```

### LoadAndSplit() 返回的元数据

```go
{
    "source":      "document.pdf",
    "type":        "pdf",
    "page":        3,          // 当前页码
    "total_pages": 10,
}
```

---

## ⚠️ 注意事项

### 1. PDF 格式支持

- ✅ 支持标准 PDF 格式
- ✅ 支持文本型 PDF
- ⚠️ 扫描型 PDF（图片）需要 OCR 处理
- ⚠️ 加密 PDF 需要提供正确密码

### 2. 文本提取限制

```go
// 扫描版 PDF 无法直接提取文本
loader := loaders.NewPDFLoader("scanned.pdf")
docs, err := loader.Load(ctx)
// docs 可能为空或内容很少

// 建议：先用 OCR 工具处理
```

### 3. 内存使用

```go
// 大文件建议按页加载
loader := loaders.NewPDFLoader("large.pdf")

// 方式1：按页加载（推荐）
pages, err := loader.LoadAndSplit(ctx)

// 方式2：分批处理
for i := 1; i <= 100; i += 10 {
    batch, err := loader.LoadPageRange(ctx, i, i+9)
    // 处理 batch...
}
```

### 4. 错误处理

```go
loader := loaders.NewPDFLoader("document.pdf")
docs, err := loader.Load(ctx)
if err != nil {
    // 检查具体错误类型
    if os.IsNotExist(err) {
        fmt.Println("File not found")
    } else if strings.Contains(err.Error(), "encrypted") {
        fmt.Println("PDF is encrypted, password required")
    } else {
        fmt.Printf("Failed to load PDF: %v\n", err)
    }
    return
}
```

---

## 🎯 最佳实践

### 1. 按需选择加载方式

```go
// 场景1：需要完整文档内容
func needFullContent(pdfPath string) {
    loader := loaders.NewPDFLoader(pdfPath)
    docs, _ := loader.Load(ctx) // 使用 Load
    // 处理完整内容...
}

// 场景2：需要引用特定页面
func needPageReference(pdfPath string) {
    loader := loaders.NewPDFLoader(pdfPath)
    pages, _ := loader.LoadAndSplit(ctx) // 使用 LoadAndSplit
    // 可以精确引用到页码...
}
```

### 2. 大文件处理

```go
func processLargePDF(pdfPath string) error {
    loader := loaders.NewPDFLoader(pdfPath)

    // 先获取总页数
    totalPages, err := loader.GetPageCount()
    if err != nil {
        return err
    }

    // 分批处理
    batchSize := 10
    for start := 1; start <= totalPages; start += batchSize {
        end := start + batchSize - 1
        if end > totalPages {
            end = totalPages
        }

        batch, err := loader.LoadPageRange(context.Background(), start, end)
        if err != nil {
            log.Printf("Failed to load pages %d-%d: %v\n", start, end, err)
            continue
        }

        // 处理这批页面
        processBatch(batch)
    }

    return nil
}
```

### 3. 与 RAG 系统集成

```go
func buildPDFRAGSystem(pdfPaths []string) error {
    ctx := context.Background()

    // 1. 创建向量存储
    emb := embeddings.NewOpenAIEmbeddings("api-key")
    store := vectorstores.NewInMemoryVectorStore(emb)

    // 2. 加载所有 PDF
    for _, path := range pdfPaths {
        loader := loaders.NewPDFLoader(path)
        docs, err := loader.LoadAndSplit(ctx)
        if err != nil {
            log.Printf("Failed to load %s: %v\n", path, err)
            continue
        }

        // 3. 添加到向量存储
        _, err = store.AddDocuments(ctx, docs)
        if err != nil {
            log.Printf("Failed to index %s: %v\n", path, err)
            continue
        }
    }

    // 4. 查询
    results, _ := store.SimilaritySearch(ctx, "your question", 5)
    for _, doc := range results {
        fmt.Printf("Source: %s, Page: %v\n", 
            doc.Metadata["source"], 
            doc.Metadata["page"])
    }

    return nil
}
```

---

## 📚 完整示例

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/zhucl121/langchain-go/retrieval/loaders"
    "github.com/zhucl121/langchain-go/retrieval/embeddings"
    "github.com/zhucl121/langchain-go/retrieval/vectorstores"
)

func main() {
    ctx := context.Background()

    // 1. 加载 PDF
    loader := loaders.NewPDFLoader("research_paper.pdf").
        WithPageRange(1, 20) // 只加载前 20 页

    docs, err := loader.LoadAndSplit(ctx)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Loaded %d pages\n", len(docs))

    // 2. 创建向量存储
    emb := embeddings.NewOpenAIEmbeddings("your-api-key")
    store := vectorstores.NewInMemoryVectorStore(emb)

    // 3. 索引文档
    _, err = store.AddDocuments(ctx, docs)
    if err != nil {
        log.Fatal(err)
    }

    // 4. 搜索
    query := "What is the main contribution of this paper?"
    results, err := store.SimilaritySearch(ctx, query, 3)
    if err != nil {
        log.Fatal(err)
    }

    // 5. 显示结果
    for i, doc := range results {
        fmt.Printf("\n--- Result %d ---\n", i+1)
        fmt.Printf("Page: %v\n", doc.Metadata["page"])
        fmt.Printf("Content: %s...\n", doc.Content[:200])
    }
}
```

---

## 🔗 相关文档

- [TextLoader 使用指南](./text-loader-guide.md)
- [向量存储使用指南](../vectorstores/README.md)
- [文本分割器使用指南](../splitters/README.md)

---

**文档维护者**: AI Assistant  
**反馈渠道**: GitHub Issues
