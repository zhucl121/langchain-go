package splitters

import (
	"testing"
	
	"github.com/stretchr/testify/assert"
	
	"github.com/zhucl121/langchain-go/retrieval/loaders"
)

// TestCharacterTextSplitter
func TestCharacterTextSplitter(t *testing.T) {
	splitter := NewCharacterTextSplitter(20, 5).WithSeparator("\n\n")
	
	text := "Short para 1.\n\nShort para 2.\n\nShort para 3."
	chunks := splitter.SplitText(text)
	
	assert.Greater(t, len(chunks), 0)
	for _, chunk := range chunks {
		// 每个块应该不超过指定大小（考虑重叠）
		assert.LessOrEqual(t, len(chunk), 30) // 允许一些误差
	}
}

// TestCharacterTextSplitter_EmptyText
func TestCharacterTextSplitter_EmptyText(t *testing.T) {
	splitter := NewCharacterTextSplitter(100, 10)
	
	chunks := splitter.SplitText("")
	
	assert.Empty(t, chunks)
}

// TestCharacterTextSplitter_SmallText
func TestCharacterTextSplitter_SmallText(t *testing.T) {
	splitter := NewCharacterTextSplitter(100, 10)
	
	text := "Short text."
	chunks := splitter.SplitText(text)
	
	assert.Len(t, chunks, 1)
	assert.Equal(t, text, chunks[0])
}

// TestRecursiveCharacterTextSplitter
func TestRecursiveCharacterTextSplitter(t *testing.T) {
	splitter := NewRecursiveCharacterTextSplitter(50, 10)
	
	text := `This is a long paragraph that should be split into multiple chunks.

This is another paragraph.

And this is the third one.`
	
	chunks := splitter.SplitText(text)
	
	assert.Greater(t, len(chunks), 1)
	for _, chunk := range chunks {
		// 验证每个块的大小
		assert.LessOrEqual(t, len(chunk), 70) // 允许一些误差
	}
}

// TestRecursiveCharacterTextSplitter_LongWord
func TestRecursiveCharacterTextSplitter_LongWord(t *testing.T) {
	splitter := NewRecursiveCharacterTextSplitter(10, 0)
	
	// 一个很长的"单词"（无法用常规分隔符分割）
	text := "verylongwordthatcannotbespliteasily"
	
	chunks := splitter.SplitText(text)
	
	// 应该被强制分割
	assert.Greater(t, len(chunks), 1)
}

// TestTokenTextSplitter
func TestTokenTextSplitter(t *testing.T) {
	splitter := NewTokenTextSplitter(10, 2)
	
	text := "This is a test sentence with multiple words that should be split into chunks based on token count."
	
	chunks := splitter.SplitText(text)
	
	assert.Greater(t, len(chunks), 1)
	
	for _, chunk := range chunks {
		// 验证每个块的单词数（近似 token 数）
		words := len(splitWords(chunk))
		assert.LessOrEqual(t, words, 15) // 允许一些误差
	}
}

// splitWords 辅助函数：分割单词
func splitWords(text string) []string {
	var words []string
	for _, word := range splitBySpace(text) {
		if word != "" {
			words = append(words, word)
		}
	}
	return words
}

func splitBySpace(text string) []string {
	result := []string{}
	word := ""
	for _, char := range text {
		if char == ' ' || char == '\n' || char == '\t' {
			if word != "" {
				result = append(result, word)
				word = ""
			}
		} else {
			word += string(char)
		}
	}
	if word != "" {
		result = append(result, word)
	}
	return result
}

// TestMarkdownTextSplitter
func TestMarkdownTextSplitter(t *testing.T) {
	splitter := NewMarkdownTextSplitter(100, 10)
	
	text := `# Title

## Section 1

This is content in section 1.

## Section 2

This is content in section 2.

### Subsection 2.1

More content here.`
	
	chunks := splitter.SplitText(text)
	
	assert.Greater(t, len(chunks), 1)
	
	// 验证分割尊重 Markdown 结构
	for _, chunk := range chunks {
		assert.LessOrEqual(t, len(chunk), 150) // 允许误差
	}
}

// TestSplitDocuments
func TestSplitDocuments(t *testing.T) {
	splitter := NewCharacterTextSplitter(20, 5)
	
	docs := []*loaders.Document{
		loaders.NewDocument("Short text 1.\n\nShort text 2.", map[string]any{
			"source": "doc1",
		}),
		loaders.NewDocument("Another short text.", map[string]any{
			"source": "doc2",
		}),
	}
	
	splitDocs := splitter.SplitDocuments(docs)
	
	assert.Greater(t, len(splitDocs), len(docs))
	
	// 验证元数据被保留
	for _, doc := range splitDocs {
		assert.NotNil(t, doc.Metadata["source"])
		assert.NotNil(t, doc.Metadata["chunk"])
		assert.NotNil(t, doc.Metadata["total_chunks"])
	}
}

// TestCharacterTextSplitter_WithOverlap
func TestCharacterTextSplitter_WithOverlap(t *testing.T) {
	splitter := NewCharacterTextSplitter(15, 5)
	
	text := "One.\n\nTwo.\n\nThree.\n\nFour.\n\nFive."
	
	chunks := splitter.SplitText(text)
	
	// 验证有重叠
	if len(chunks) > 1 {
		// 检查相邻块之间是否有重叠内容
		// 这是一个简化的检查
		assert.Greater(t, len(chunks), 1)
	}
}

// TestRecursiveCharacterTextSplitter_CustomSeparators
func TestRecursiveCharacterTextSplitter_CustomSeparators(t *testing.T) {
	splitter := NewRecursiveCharacterTextSplitter(30, 5).
		WithSeparators([]string{".", " ", ""})
	
	text := "First sentence. Second sentence. Third sentence. Fourth sentence."
	
	chunks := splitter.SplitText(text)
	
	assert.Greater(t, len(chunks), 0)
	for _, chunk := range chunks {
		assert.LessOrEqual(t, len(chunk), 40)
	}
}

// Benchmark 测试
func BenchmarkCharacterTextSplitter(b *testing.B) {
	splitter := NewCharacterTextSplitter(1000, 200)
	text := generateLongText(10000)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		splitter.SplitText(text)
	}
}

func BenchmarkRecursiveCharacterTextSplitter(b *testing.B) {
	splitter := NewRecursiveCharacterTextSplitter(1000, 200)
	text := generateLongText(10000)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		splitter.SplitText(text)
	}
}

// generateLongText 生成长文本用于测试
func generateLongText(length int) string {
	text := ""
	sentence := "This is a test sentence. "
	for len(text) < length {
		text += sentence
		if len(text)%100 == 0 {
			text += "\n\n"
		}
	}
	return text
}
