// Package splitters 提供文本分割器。
//
// Text Splitters 负责将长文档分割成更小的块（chunks），
// 这对于向量数据库存储和语义搜索至关重要。
//
// 支持的分割器：
//   - CharacterTextSplitter: 基于字符数分割
//   - RecursiveCharacterTextSplitter: 递归分割
//   - TokenTextSplitter: 基于 Token 数分割
//   - MarkdownTextSplitter: 基于 Markdown 结构分割
//
// 使用示例：
//
//	splitter := splitters.NewCharacterTextSplitter(1000, 200)
//	chunks := splitter.SplitText(longText)
//
package splitters

import (
	"strings"
	
	"langchain-go/retrieval/loaders"
)

// TextSplitter 是文本分割器接口。
type TextSplitter interface {
	// SplitText 分割文本
	//
	// 参数：
	//   - text: 要分割的文本
	//
	// 返回：
	//   - []string: 分割后的文本块
	//
	SplitText(text string) []string
	
	// SplitDocuments 分割文档
	//
	// 参数：
	//   - docs: 文档列表
	//
	// 返回：
	//   - []*loaders.Document: 分割后的文档列表
	//
	SplitDocuments(docs []*loaders.Document) []*loaders.Document
}

// BaseTextSplitter 提供分割器的基础实现。
type BaseTextSplitter struct {
	ChunkSize    int // 每个块的大小
	ChunkOverlap int // 块之间的重叠大小
	Separator    string
}

// NewBaseTextSplitter 创建基础分割器。
func NewBaseTextSplitter(chunkSize, chunkOverlap int) *BaseTextSplitter {
	return &BaseTextSplitter{
		ChunkSize:    chunkSize,
		ChunkOverlap: chunkOverlap,
		Separator:    "\n\n",
	}
}

// SplitDocuments 实现 TextSplitter 接口。
func (bts *BaseTextSplitter) SplitDocuments(docs []*loaders.Document) []*loaders.Document {
	var result []*loaders.Document
	
	for _, doc := range docs {
		// BaseTextSplitter 不直接实现 SplitText
		// 子类应该实现具体的分割逻辑
		// 这里提供默认实现：按 Separator 简单分割
		chunks := bts.simpleSplit(doc.Content)
		
		for i, chunk := range chunks {
			// 复制元数据
			metadata := make(map[string]any)
			for k, v := range doc.Metadata {
				metadata[k] = v
			}
			metadata["chunk"] = i
			metadata["total_chunks"] = len(chunks)
			
			newDoc := loaders.NewDocument(chunk, metadata)
			newDoc.Source = doc.Source
			result = append(result, newDoc)
		}
	}
	
	return result
}

// simpleSplit 简单分割（默认实现）。
func (bts *BaseTextSplitter) simpleSplit(text string) []string {
	if text == "" {
		return []string{}
	}
	splits := strings.Split(text, bts.Separator)
	return bts.mergeSplits(splits)
}

// mergeSplits 合并分割的文本块。
func (bts *BaseTextSplitter) mergeSplits(splits []string) []string {
	if len(splits) == 0 {
		return []string{}
	}
	
	var chunks []string
	var currentChunk strings.Builder
	currentLength := 0
	
	for _, split := range splits {
		splitLen := len(split)
		
		// 如果当前块加上新分割超过大小限制
		if currentLength+splitLen > bts.ChunkSize && currentLength > 0 {
			// 保存当前块
			chunks = append(chunks, currentChunk.String())
			
			// 开始新块，保留重叠部分
			currentChunk.Reset()
			if bts.ChunkOverlap > 0 && len(chunks) > 0 {
				// 从上一个块的末尾获取重叠内容
				lastChunk := chunks[len(chunks)-1]
				if len(lastChunk) > bts.ChunkOverlap {
					overlapText := lastChunk[len(lastChunk)-bts.ChunkOverlap:]
					currentChunk.WriteString(overlapText)
					currentLength = len(overlapText)
				}
			} else {
				currentLength = 0
			}
		}
		
		// 添加当前分割
		if currentLength > 0 {
			currentChunk.WriteString(bts.Separator)
			currentLength += len(bts.Separator)
		}
		currentChunk.WriteString(split)
		currentLength += splitLen
	}
	
	// 添加最后一个块
	if currentLength > 0 {
		chunks = append(chunks, currentChunk.String())
	}
	
	return chunks
}
