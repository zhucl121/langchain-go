// Package keyword 提供基于关键词的检索功能，包括 BM25 算法实现。
//
// BM25 (Best Matching 25) 是一种基于概率的排序函数，用于信息检索。
//
// 使用示例：
//
//	docs := []types.Document{
//	    {Content: "Go is a programming language"},
//	    {Content: "Python is also a programming language"},
//	}
//
//	retriever := keyword.NewBM25Retriever(docs, keyword.DefaultBM25Config())
//	results, _ := retriever.Search(ctx, "programming", 5)
//
package keyword

import (
	"context"
	"math"
	"sort"
	"strings"

	"github.com/zhucl121/langchain-go/pkg/types"
)

// BM25Config BM25 算法配置
type BM25Config struct {
	// K1 控制词频饱和度，通常取值 1.2-2.0
	K1 float64

	// B 控制文档长度归一化，通常取值 0.75
	B float64

	// Tokenizer 分词器
	Tokenizer Tokenizer
}

// DefaultBM25Config 返回默认配置
func DefaultBM25Config() BM25Config {
	return BM25Config{
		K1:        1.5,
		B:         0.75,
		Tokenizer: NewWhitespaceTokenizer(),
	}
}

// BM25Retriever 基于 BM25 算法的检索器
type BM25Retriever struct {
	documents []types.Document
	index     *BM25Index
	config    BM25Config
}

// BM25Index BM25 索引结构
type BM25Index struct {
	// 文档频率 (Document Frequency): 包含某个词的文档数
	docFreq map[string]int

	// 文档长度
	docLengths []int

	// 平均文档长度
	avgDocLength float64

	// 总文档数
	totalDocs int

	// 倒排索引: term -> [doc_id1, doc_id2, ...]
	invertedIndex map[string][]int

	// 词频索引: doc_id -> term -> count
	termFreq []map[string]int
}

// ScoredDocument 带分数的文档
type ScoredDocument struct {
	Document types.Document
	Score    float64
	TermInfo map[string]float64 // 每个词的贡献分数（用于调试）
}

// NewBM25Retriever 创建新的 BM25 检索器
func NewBM25Retriever(documents []types.Document, config BM25Config) *BM25Retriever {
	retriever := &BM25Retriever{
		documents: documents,
		config:    config,
	}

	retriever.buildIndex()
	return retriever
}

// buildIndex 构建 BM25 索引
func (r *BM25Retriever) buildIndex() {
	r.index = &BM25Index{
		docFreq:       make(map[string]int),
		docLengths:    make([]int, len(r.documents)),
		totalDocs:     len(r.documents),
		invertedIndex: make(map[string][]int),
		termFreq:      make([]map[string]int, len(r.documents)),
	}

	totalLength := 0

	// 遍历文档，构建索引
	for docID, doc := range r.documents {
		// 分词
		tokens := r.config.Tokenizer.Tokenize(doc.Content)

		// 记录文档长度
		r.index.docLengths[docID] = len(tokens)
		totalLength += len(tokens)

		// 统计词频
		termFreqMap := make(map[string]int)
		uniqueTerms := make(map[string]bool)

		for _, term := range tokens {
			term = strings.ToLower(term) // 转小写
			termFreqMap[term]++
			uniqueTerms[term] = true
		}

		r.index.termFreq[docID] = termFreqMap

		// 更新倒排索引和文档频率
		for term := range uniqueTerms {
			r.index.invertedIndex[term] = append(r.index.invertedIndex[term], docID)
			r.index.docFreq[term]++
		}
	}

	// 计算平均文档长度
	if r.index.totalDocs > 0 {
		r.index.avgDocLength = float64(totalLength) / float64(r.index.totalDocs)
	}
}

// Search 执行 BM25 搜索
//
// 参数：
//   - ctx: 上下文
//   - query: 查询字符串
//   - k: 返回结果数量
//
// 返回：
//   - []ScoredDocument: 排序后的结果
//   - error: 错误
func (r *BM25Retriever) Search(ctx context.Context, query string, k int) ([]ScoredDocument, error) {
	// 分词
	queryTerms := r.config.Tokenizer.Tokenize(query)

	// 计算每个文档的 BM25 分数
	scores := make([]ScoredDocument, 0, len(r.documents))

	for docID, doc := range r.documents {
		score, termInfo := r.calculateBM25Score(queryTerms, docID)

		if score > 0 {
			scores = append(scores, ScoredDocument{
				Document: doc,
				Score:    score,
				TermInfo: termInfo,
			})
		}
	}

	// 按分数降序排序
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].Score > scores[j].Score
	})

	// 返回 top-k
	if len(scores) > k {
		scores = scores[:k]
	}

	return scores, nil
}

// calculateBM25Score 计算文档的 BM25 分数
//
// BM25 公式:
// score(D,Q) = Σ IDF(qi) · (f(qi,D) · (k1 + 1)) / (f(qi,D) + k1 · (1 - b + b · |D| / avgdl))
//
// 其中:
//   - IDF(qi) = log((N - df(qi) + 0.5) / (df(qi) + 0.5) + 1)
//   - f(qi,D) = qi 在文档 D 中的词频
//   - |D| = 文档 D 的长度
//   - avgdl = 平均文档长度
//   - k1, b = 调节参数
func (r *BM25Retriever) calculateBM25Score(queryTerms []string, docID int) (float64, map[string]float64) {
	score := 0.0
	termInfo := make(map[string]float64)

	docLength := float64(r.index.docLengths[docID])
	termFreqMap := r.index.termFreq[docID]

	// 遍历查询词
	for _, term := range queryTerms {
		term = strings.ToLower(term)

		// 获取词频
		tf := float64(termFreqMap[term])
		if tf == 0 {
			continue // 文档不包含该词
		}

		// 计算 IDF
		df := float64(r.index.docFreq[term])
		idf := r.calculateIDF(df)

		// 计算 BM25 分数
		numerator := tf * (r.config.K1 + 1)
		denominator := tf + r.config.K1*(1-r.config.B+r.config.B*(docLength/r.index.avgDocLength))

		termScore := idf * (numerator / denominator)
		score += termScore
		termInfo[term] = termScore
	}

	return score, termInfo
}

// calculateIDF 计算 IDF (Inverse Document Frequency)
//
// IDF(qi) = log((N - df(qi) + 0.5) / (df(qi) + 0.5) + 1)
func (r *BM25Retriever) calculateIDF(df float64) float64 {
	n := float64(r.index.totalDocs)
	idf := math.Log((n-df+0.5)/(df+0.5) + 1)
	return idf
}

// AddDocuments 添加新文档并更新索引
func (r *BM25Retriever) AddDocuments(documents []types.Document) {
	r.documents = append(r.documents, documents...)
	r.buildIndex() // 重建索引
}

// GetDocumentCount 获取文档数量
func (r *BM25Retriever) GetDocumentCount() int {
	return len(r.documents)
}

// GetIndexStats 获取索引统计信息
func (r *BM25Retriever) GetIndexStats() map[string]any {
	return map[string]any{
		"total_docs":       r.index.totalDocs,
		"avg_doc_length":   r.index.avgDocLength,
		"unique_terms":     len(r.index.docFreq),
		"total_term_freq":  r.getTotalTermFreq(),
		"max_doc_length":   r.getMaxDocLength(),
		"min_doc_length":   r.getMinDocLength(),
	}
}

func (r *BM25Retriever) getTotalTermFreq() int {
	total := 0
	for _, freq := range r.index.termFreq {
		for _, count := range freq {
			total += count
		}
	}
	return total
}

func (r *BM25Retriever) getMaxDocLength() int {
	max := 0
	for _, length := range r.index.docLengths {
		if length > max {
			max = length
		}
	}
	return max
}

func (r *BM25Retriever) getMinDocLength() int {
	if len(r.index.docLengths) == 0 {
		return 0
	}
	min := r.index.docLengths[0]
	for _, length := range r.index.docLengths {
		if length < min {
			min = length
		}
	}
	return min
}
