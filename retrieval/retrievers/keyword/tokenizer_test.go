package keyword

import (
	"reflect"
	"testing"
)

func TestWhitespaceTokenizer(t *testing.T) {
	tokenizer := NewWhitespaceTokenizer()

	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "simple english",
			input:    "Hello World",
			expected: []string{"hello", "world"},
		},
		{
			name:     "with punctuation",
			input:    "Hello, World!",
			expected: []string{"hello", "world"},
		},
		{
			name:     "multiple spaces",
			input:    "Go   is    great",
			expected: []string{"go", "is", "great"},
		},
		{
			name:     "numbers",
			input:    "Python 3.11 is here",
			expected: []string{"python", "3", "11", "is", "here"},
		},
		{
			name:     "empty string",
			input:    "",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tokenizer.Tokenize(tt.input)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestSimpleChineseTokenizer(t *testing.T) {
	tokenizer := NewSimpleChineseTokenizer()

	tests := []struct {
		name     string
		input    string
		expected int // 期望的 token 数量
	}{
		{
			name:     "simple chinese",
			input:    "你好世界",
			expected: 4,
		},
		{
			name:     "mixed chinese and english",
			input:    "Go语言很好",
			expected: 6, // Go, 语, 言, 很, 好
		},
		{
			name:     "with punctuation",
			input:    "你好，世界！",
			expected: 4, // 标点被过滤
		},
		{
			name:     "with spaces",
			input:    "你好 世界",
			expected: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tokenizer.Tokenize(tt.input)
			if len(result) != tt.expected {
				t.Errorf("Expected %d tokens, got %d: %v", tt.expected, len(result), result)
			}
		})
	}
}

func TestUnicodeTokenizer(t *testing.T) {
	tokenizer := NewUnicodeTokenizer()

	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "english",
			input:    "Hello World",
			expected: []string{"hello", "world"},
		},
		{
			name:     "chinese",
			input:    "你好世界",
			expected: []string{"你好世界"}, // 作为一个 token
		},
		{
			name:     "mixed",
			input:    "Hello世界",
			expected: []string{"hello世界"},
		},
		{
			name:     "with punctuation",
			input:    "Hello, 世界!",
			expected: []string{"hello", "世界"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tokenizer.Tokenize(tt.input)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestNGramTokenizer(t *testing.T) {
	tests := []struct {
		name     string
		n        int
		input    string
		expected []string
	}{
		{
			name:     "bigram",
			n:        2,
			input:    "hello",
			expected: []string{"he", "el", "ll", "lo"},
		},
		{
			name:     "trigram",
			n:        3,
			input:    "hello",
			expected: []string{"hel", "ell", "llo"},
		},
		{
			name:     "chinese bigram",
			n:        2,
			input:    "你好",
			expected: []string{"你好"},
		},
		{
			name:     "chinese trigram",
			n:        3,
			input:    "你好世界",
			expected: []string{"你好世", "好世界"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenizer := NewNGramTokenizer(tt.n)
			result := tokenizer.Tokenize(tt.input)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestCustomTokenizer(t *testing.T) {
	// 自定义分词函数：按逗号分割
	tokenizer := NewCustomTokenizer(func(text string) []string {
		return []string{"custom", "tokenizer", "test"}
	})

	result := tokenizer.Tokenize("anything")
	expected := []string{"custom", "tokenizer", "test"}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Expected %v, got %v", expected, result)
	}
}

func TestStopWordsFilter(t *testing.T) {
	baseTokenizer := NewWhitespaceTokenizer()
	tokenizer := NewStopWordsFilter(baseTokenizer, DefaultEnglishStopWords)

	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "with stop words",
			input:    "the quick brown fox",
			expected: []string{"quick", "brown", "fox"}, // "the" 被过滤
		},
		{
			name:     "all stop words",
			input:    "the and or",
			expected: []string{}, // 全部被过滤
		},
		{
			name:     "no stop words",
			input:    "quick brown fox",
			expected: []string{"quick", "brown", "fox"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tokenizer.Tokenize(tt.input)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestStopWordsFilter_Chinese(t *testing.T) {
	baseTokenizer := NewSimpleChineseTokenizer()
	tokenizer := NewStopWordsFilter(baseTokenizer, DefaultChineseStopWords)

	input := "我的世界很好"
	result := tokenizer.Tokenize(input)

	// "的" 和 "很" 应该被过滤
	for _, token := range result {
		if token == "的" || token == "很" {
			t.Errorf("Stop word %s should be filtered", token)
		}
	}

	t.Logf("Filtered result: %v", result)
}

func BenchmarkWhitespaceTokenizer(b *testing.B) {
	tokenizer := NewWhitespaceTokenizer()
	text := "This is a sample text for benchmarking the whitespace tokenizer performance"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = tokenizer.Tokenize(text)
	}
}

func BenchmarkSimpleChineseTokenizer(b *testing.B) {
	tokenizer := NewSimpleChineseTokenizer()
	text := "这是一个用于测试中文分词器性能的示例文本"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = tokenizer.Tokenize(text)
	}
}

func BenchmarkUnicodeTokenizer(b *testing.B) {
	tokenizer := NewUnicodeTokenizer()
	text := "This is mixed 这是混合 text 文本 for testing"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = tokenizer.Tokenize(text)
	}
}
