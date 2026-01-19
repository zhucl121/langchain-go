package keyword

import (
	"regexp"
	"strings"
	"unicode"
)

// Tokenizer 分词器接口
//
// Tokenizer 负责将文本分割成词元（tokens）。
// 不同的语言和场景可能需要不同的分词策略。
type Tokenizer interface {
	// Tokenize 将文本分词
	Tokenize(text string) []string
}

// WhitespaceTokenizer 空格分词器
//
// 适用于英文等使用空格分隔的语言。
// 会移除标点符号和特殊字符。
type WhitespaceTokenizer struct {
	// LowerCase 是否转小写
	LowerCase bool

	// MinLength 最小词长（过滤短词）
	MinLength int
}

// NewWhitespaceTokenizer 创建空格分词器
func NewWhitespaceTokenizer() *WhitespaceTokenizer {
	return &WhitespaceTokenizer{
		LowerCase: true,
		MinLength: 1,
	}
}

// Tokenize 实现 Tokenizer 接口
func (t *WhitespaceTokenizer) Tokenize(text string) []string {
	// 移除标点符号
	reg := regexp.MustCompile(`[^\w\s]+`)
	text = reg.ReplaceAllString(text, " ")

	// 分词
	words := strings.Fields(text)

	// 过滤和处理
	tokens := make([]string, 0, len(words))
	for _, word := range words {
		if t.LowerCase {
			word = strings.ToLower(word)
		}

		// 过滤短词
		if len(word) >= t.MinLength {
			tokens = append(tokens, word)
		}
	}

	return tokens
}

// SimpleChineseTokenizer 简单中文分词器
//
// 使用单字分词策略。
// 注意：这是一个简化实现，对于生产环境建议使用 sego 或 jieba 等专业分词库。
type SimpleChineseTokenizer struct {
	// LowerCase 是否转小写（对中文无效，但保留选项）
	LowerCase bool

	// IncludePunctuation 是否包含标点符号
	IncludePunctuation bool
}

// NewSimpleChineseTokenizer 创建简单中文分词器
func NewSimpleChineseTokenizer() *SimpleChineseTokenizer {
	return &SimpleChineseTokenizer{
		LowerCase:          true,
		IncludePunctuation: false,
	}
}

// Tokenize 实现 Tokenizer 接口
func (t *SimpleChineseTokenizer) Tokenize(text string) []string {
	tokens := make([]string, 0)

	for _, r := range text {
		// 跳过空白字符
		if unicode.IsSpace(r) {
			continue
		}

		// 跳过标点符号（如果配置）
		if !t.IncludePunctuation && unicode.IsPunct(r) {
			continue
		}

		// 中文字符或字母数字
		if unicode.Is(unicode.Han, r) || unicode.IsLetter(r) || unicode.IsNumber(r) {
			token := string(r)
			if t.LowerCase && unicode.IsLetter(r) {
				token = strings.ToLower(token)
			}
			tokens = append(tokens, token)
		}
	}

	return tokens
}

// UnicodeTokenizer 通用 Unicode 分词器
//
// 使用 Unicode 单词边界进行分词，适用于多种语言。
type UnicodeTokenizer struct {
	// LowerCase 是否转小写
	LowerCase bool

	// MinLength 最小词长
	MinLength int
}

// NewUnicodeTokenizer 创建 Unicode 分词器
func NewUnicodeTokenizer() *UnicodeTokenizer {
	return &UnicodeTokenizer{
		LowerCase: true,
		MinLength: 1,
	}
}

// Tokenize 实现 Tokenizer 接口
func (t *UnicodeTokenizer) Tokenize(text string) []string {
	tokens := make([]string, 0)
	var currentToken strings.Builder

	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsNumber(r) || unicode.Is(unicode.Han, r) {
			// 字母、数字或中文字符
			currentToken.WriteRune(r)
		} else {
			// 遇到分隔符，保存当前token
			if currentToken.Len() > 0 {
				token := currentToken.String()
				if t.LowerCase {
					token = strings.ToLower(token)
				}
				if len(token) >= t.MinLength {
					tokens = append(tokens, token)
				}
				currentToken.Reset()
			}
		}
	}

	// 保存最后一个 token
	if currentToken.Len() > 0 {
		token := currentToken.String()
		if t.LowerCase {
			token = strings.ToLower(token)
		}
		if len(token) >= t.MinLength {
			tokens = append(tokens, token)
		}
	}

	return tokens
}

// NGramTokenizer N-gram 分词器
//
// 生成 n-gram tokens，适用于模糊匹配和短文本。
type NGramTokenizer struct {
	// N gram 大小
	N int

	// LowerCase 是否转小写
	LowerCase bool
}

// NewNGramTokenizer 创建 n-gram 分词器
func NewNGramTokenizer(n int) *NGramTokenizer {
	if n < 1 {
		n = 2 // 默认 bigram
	}
	return &NGramTokenizer{
		N:         n,
		LowerCase: true,
	}
}

// Tokenize 实现 Tokenizer 接口
func (t *NGramTokenizer) Tokenize(text string) []string {
	if t.LowerCase {
		text = strings.ToLower(text)
	}

	// 移除空格
	text = strings.ReplaceAll(text, " ", "")

	runes := []rune(text)
	tokens := make([]string, 0)

	// 生成 n-grams
	for i := 0; i <= len(runes)-t.N; i++ {
		ngram := string(runes[i : i+t.N])
		tokens = append(tokens, ngram)
	}

	return tokens
}

// CustomTokenizer 自定义分词器
//
// 允许用户提供自定义分词函数。
type CustomTokenizer struct {
	TokenizeFunc func(string) []string
}

// NewCustomTokenizer 创建自定义分词器
func NewCustomTokenizer(fn func(string) []string) *CustomTokenizer {
	return &CustomTokenizer{
		TokenizeFunc: fn,
	}
}

// Tokenize 实现 Tokenizer 接口
func (t *CustomTokenizer) Tokenize(text string) []string {
	return t.TokenizeFunc(text)
}

// StopWordsFilter 停用词过滤器
//
// 包装其他分词器，过滤停用词。
type StopWordsFilter struct {
	baseTokenizer Tokenizer
	stopWords     map[string]bool
}

// NewStopWordsFilter 创建停用词过滤器
func NewStopWordsFilter(base Tokenizer, stopWords []string) *StopWordsFilter {
	stopWordsMap := make(map[string]bool, len(stopWords))
	for _, word := range stopWords {
		stopWordsMap[strings.ToLower(word)] = true
	}

	return &StopWordsFilter{
		baseTokenizer: base,
		stopWords:     stopWordsMap,
	}
}

// Tokenize 实现 Tokenizer 接口
func (f *StopWordsFilter) Tokenize(text string) []string {
	tokens := f.baseTokenizer.Tokenize(text)

	// 过滤停用词
	filtered := make([]string, 0, len(tokens))
	for _, token := range tokens {
		if !f.stopWords[strings.ToLower(token)] {
			filtered = append(filtered, token)
		}
	}

	return filtered
}

// DefaultEnglishStopWords 默认英文停用词
var DefaultEnglishStopWords = []string{
	"a", "an", "and", "are", "as", "at", "be", "by", "for", "from",
	"has", "he", "in", "is", "it", "its", "of", "on", "or", "that", "the",
	"to", "was", "will", "with",
}

// DefaultChineseStopWords 默认中文停用词
var DefaultChineseStopWords = []string{
	"的", "了", "在", "是", "我", "有", "和", "就", "不", "人",
	"都", "一", "一个", "上", "也", "很", "到", "说", "要", "去",
}
