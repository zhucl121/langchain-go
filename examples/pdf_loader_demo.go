package main

import (
	"context"
	"fmt"
	"log"

	"langchain-go/retrieval/loaders"
)

func main() {
	ctx := context.Background()

	// 测试 PDF 文件路径
	pdfPath := "retrieval/loaders/testdata/数据治理全景解析与实践-蚂蚁数科.pdf"

	fmt.Println("=== PDF 加载器演示 ===\n")

	// 1. 获取 PDF 信息
	fmt.Println("1. 获取 PDF 基本信息")
	loader := loaders.NewPDFLoader(pdfPath)
	
	pageCount, err := loader.GetPageCount()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   总页数: %d\n", pageCount)

	metadata, err := loader.ExtractMetadata()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   元数据: %+v\n\n", metadata)

	// 2. 加载第一页
	fmt.Println("2. 加载第一页内容")
	pages, err := loader.WithPageRange(1, 1).LoadAndSplit(ctx)
	if err != nil {
		log.Fatal(err)
	}
	
	if len(pages) > 0 {
		content := pages[0].Content
		if len(content) > 200 {
			content = content[:200] + "..."
		}
		fmt.Printf("   第一页内容预览: %s\n\n", content)
	}

	// 3. 按页加载（前3页）
	fmt.Println("3. 按页加载（前3页）")
	loader2 := loaders.NewPDFLoader(pdfPath).WithPageRange(1, 3)
	pages, err = loader2.LoadAndSplit(ctx)
	if err != nil {
		log.Fatal(err)
	}
	
	fmt.Printf("   加载了 %d 页\n", len(pages))
	for i, page := range pages {
		preview := page.Content
		if len(preview) > 100 {
			preview = preview[:100] + "..."
		}
		fmt.Printf("   页 %d: %s\n", i+1, preview)
	}
	fmt.Println()

	// 4. 加载整个文档
	fmt.Println("4. 加载整个文档")
	loader3 := loaders.NewPDFLoader(pdfPath)
	docs, err := loader3.Load(ctx)
	if err != nil {
		log.Fatal(err)
	}
	
	if len(docs) > 0 {
		fmt.Printf("   文档总长度: %d 字符\n", len(docs[0].Content))
		fmt.Printf("   元数据: %+v\n", docs[0].Metadata)
	}

	fmt.Println("\n=== 演示完成 ===")
}
