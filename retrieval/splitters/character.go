package splitters

import (
	"strings"
)

// CharacterTextSplitter 是基于字符的文本分割器。
//
// 按照指定的分隔符分割文本，然后合并成指定大小的块。
//
type CharacterTextSplitter struct {
	*BaseTextSplitter
	separator string
}

// NewCharacterTextSplitter 创建字符分割器。
//
// 参数：
//   - chunkSize: 每个块的大小（字符数）
//   - chunkOverlap: 块之间的重叠大小
//
// 返回：
//   - *CharacterTextSplitter: 字符分割器实例
//
func NewCharacterTextSplitter(chunkSize, chunkOverlap int) *CharacterTextSplitter {
	return &CharacterTextSplitter{
		BaseTextSplitter: NewBaseTextSplitter(chunkSize, chunkOverlap),
		separator:        "\n\n",
	}
}

// WithSeparator 设置分隔符。
func (cts *CharacterTextSplitter) WithSeparator(sep string) *CharacterTextSplitter {
	cts.separator = sep
	cts.Separator = sep
	return cts
}

// SplitText 实现 TextSplitter 接口。
func (cts *CharacterTextSplitter) SplitText(text string) []string {
	if text == "" {
		return []string{}
	}
	
	// 按分隔符分割
	splits := strings.Split(text, cts.separator)
	
	// 过滤空字符串
	var nonEmpty []string
	for _, s := range splits {
		if strings.TrimSpace(s) != "" {
			nonEmpty = append(nonEmpty, s)
		}
	}
	
	// 合并成块
	return cts.mergeSplits(nonEmpty)
}

// RecursiveCharacterTextSplitter 是递归字符分割器。
//
// 尝试使用多个分隔符递归分割文本，直到每个块都小于指定大小。
//
type RecursiveCharacterTextSplitter struct {
	*BaseTextSplitter
	separators []string
}

// NewRecursiveCharacterTextSplitter 创建递归字符分割器。
//
// 参数：
//   - chunkSize: 每个块的大小
//   - chunkOverlap: 重叠大小
//
// 返回：
//   - *RecursiveCharacterTextSplitter: 递归分割器实例
//
func NewRecursiveCharacterTextSplitter(chunkSize, chunkOverlap int) *RecursiveCharacterTextSplitter {
	return &RecursiveCharacterTextSplitter{
		BaseTextSplitter: NewBaseTextSplitter(chunkSize, chunkOverlap),
		separators:       []string{"\n\n", "\n", " ", ""},
	}
}

// WithSeparators 设置分隔符列表。
func (rcts *RecursiveCharacterTextSplitter) WithSeparators(seps []string) *RecursiveCharacterTextSplitter {
	rcts.separators = seps
	return rcts
}

// SplitText 实现 TextSplitter 接口。
func (rcts *RecursiveCharacterTextSplitter) SplitText(text string) []string {
	return rcts.splitTextRecursive(text, rcts.separators)
}

// splitTextRecursive 递归分割文本。
func (rcts *RecursiveCharacterTextSplitter) splitTextRecursive(text string, separators []string) []string {
	if text == "" {
		return []string{}
	}
	
	// 如果文本足够小，直接返回
	if len(text) <= rcts.ChunkSize {
		return []string{text}
	}
	
	// 没有更多分隔符，强制分割
	if len(separators) == 0 {
		return rcts.forceSplit(text)
	}
	
	// 使用当前分隔符分割
	separator := separators[0]
	remainingSeparators := separators[1:]
	
	var splits []string
	if separator == "" {
		// 空分隔符，按字符分割
		splits = rcts.forceSplit(text)
	} else {
		splits = strings.Split(text, separator)
	}
	
	// 递归处理每个分割
	var result []string
	for _, split := range splits {
		if strings.TrimSpace(split) == "" {
			continue
		}
		
		if len(split) > rcts.ChunkSize {
			// 如果分割仍然太大，使用下一个分隔符递归分割
			subSplits := rcts.splitTextRecursive(split, remainingSeparators)
			result = append(result, subSplits...)
		} else {
			result = append(result, split)
		}
	}
	
	// 合并小块
	rcts.Separator = separator
	return rcts.mergeSplits(result)
}

// forceSplit 强制分割文本。
func (rcts *RecursiveCharacterTextSplitter) forceSplit(text string) []string {
	var chunks []string
	
	for i := 0; i < len(text); i += rcts.ChunkSize {
		end := i + rcts.ChunkSize
		if end > len(text) {
			end = len(text)
		}
		chunks = append(chunks, text[i:end])
	}
	
	return chunks
}

// TokenTextSplitter 是基于 Token 的分割器。
//
// 注意：这是简化实现，实际的 Token 计数应该使用
// tiktoken 或其他 tokenizer。
//
type TokenTextSplitter struct {
	*BaseTextSplitter
	tokensPerChunk int
}

// NewTokenTextSplitter 创建 Token 分割器。
//
// 参数：
//   - tokensPerChunk: 每个块的 Token 数
//   - overlapTokens: 重叠的 Token 数
//
// 返回：
//   - *TokenTextSplitter: Token 分割器实例
//
func NewTokenTextSplitter(tokensPerChunk, overlapTokens int) *TokenTextSplitter {
	return &TokenTextSplitter{
		BaseTextSplitter: NewBaseTextSplitter(tokensPerChunk*4, overlapTokens*4), // 估算字符数
		tokensPerChunk:   tokensPerChunk,
	}
}

// SplitText 实现 TextSplitter 接口。
func (tts *TokenTextSplitter) SplitText(text string) []string {
	// 简化实现：使用空格分割作为 Token 的近似
	words := strings.Fields(text)
	
	var chunks []string
	var currentChunk []string
	currentTokens := 0
	
	for _, word := range words {
		wordTokens := 1 // 简化：每个单词算作 1 个 token
		
		if currentTokens+wordTokens > tts.tokensPerChunk && len(currentChunk) > 0 {
			// 保存当前块
			chunks = append(chunks, strings.Join(currentChunk, " "))
			
			// 开始新块，保留重叠
			overlapWords := 0
			if tts.ChunkOverlap > 0 {
				overlapWords = tts.ChunkOverlap / 4 // 估算单词数
				if overlapWords > len(currentChunk) {
					overlapWords = len(currentChunk)
				}
			}
			
			if overlapWords > 0 {
				currentChunk = currentChunk[len(currentChunk)-overlapWords:]
				currentTokens = overlapWords
			} else {
				currentChunk = []string{}
				currentTokens = 0
			}
		}
		
		currentChunk = append(currentChunk, word)
		currentTokens += wordTokens
	}
	
	// 添加最后一个块
	if len(currentChunk) > 0 {
		chunks = append(chunks, strings.Join(currentChunk, " "))
	}
	
	return chunks
}

// MarkdownTextSplitter 是 Markdown 分割器。
//
// 尝试按照 Markdown 的结构（标题、段落等）智能分割。
//
type MarkdownTextSplitter struct {
	*RecursiveCharacterTextSplitter
}

// NewMarkdownTextSplitter 创建 Markdown 分割器。
func NewMarkdownTextSplitter(chunkSize, chunkOverlap int) *MarkdownTextSplitter {
	base := NewRecursiveCharacterTextSplitter(chunkSize, chunkOverlap)
	// Markdown 特定的分隔符
	base.separators = []string{
		"\n## ",   // H2 标题
		"\n### ",  // H3 标题
		"\n#### ", // H4 标题
		"\n\n",    // 段落
		"\n",      // 行
		" ",       // 单词
		"",        // 字符
	}
	
	return &MarkdownTextSplitter{
		RecursiveCharacterTextSplitter: base,
	}
}
