package vectorstores

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/zhucl121/langchain-go/core/chat"
	"github.com/zhucl121/langchain-go/pkg/types"
	"github.com/zhucl121/langchain-go/retrieval/loaders"
)

// LLMReranker 是基于 LLM 的重排序器。
//
// LLMReranker 使用大语言模型来评估文档与查询的相关性，
// 相比基于向量相似度的排序，可以提供更准确的结果。
//
type LLMReranker struct {
	llm            chat.ChatModel
	promptTemplate string
	topK           int // 只对前 topK 个结果进行重排序
}

// LLMRerankerConfig 是 LLM 重排序器的配置。
type LLMRerankerConfig struct {
	// LLM 模型
	LLM chat.ChatModel

	// 提示词模板（可选）
	// 默认模板会要求 LLM 评分 0-10
	PromptTemplate string

	// 只对前 TopK 个结果重排序（可选）
	// 默认为 20，设置为 0 表示对所有结果重排序
	TopK int
}

// DefaultRerankPromptTemplate 是默认的重排序提示词模板。
const DefaultRerankPromptTemplate = `给定一个查询和一个文档，请评估文档与查询的相关性。

查询: {{.Query}}

文档: {{.Document}}

请给出一个 0-10 之间的相关性分数，其中：
- 0 表示完全不相关
- 5 表示部分相关
- 10 表示完全相关

只需要输出数字分数，不需要任何解释。`

// NewLLMReranker 创建 LLM 重排序器。
//
// 参数：
//   - config: 重排序器配置
//
// 返回：
//   - *LLMReranker: 重排序器实例
//   - error: 错误
//
func NewLLMReranker(config LLMRerankerConfig) (*LLMReranker, error) {
	if config.LLM == nil {
		return nil, fmt.Errorf("LLM is required")
	}

	promptTemplate := config.PromptTemplate
	if promptTemplate == "" {
		promptTemplate = DefaultRerankPromptTemplate
	}

	topK := config.TopK
	if topK == 0 {
		topK = 20 // 默认值
	}

	return &LLMReranker{
		llm:            config.LLM,
		promptTemplate: promptTemplate,
		topK:           topK,
	}, nil
}

// Rerank 对搜索结果进行重排序。
//
// 参数：
//   - ctx: 上下文
//   - query: 查询文本
//   - documents: 待重排序的文档列表
//
// 返回：
//   - []DocumentWithScore: 重排序后的文档列表
//   - error: 错误
//
func (r *LLMReranker) Rerank(
	ctx context.Context,
	query string,
	documents []DocumentWithScore,
) ([]DocumentWithScore, error) {
	if len(documents) == 0 {
		return documents, nil
	}

	// 只对前 topK 个文档进行重排序
	docsToRerank := documents
	if len(documents) > r.topK {
		docsToRerank = documents[:r.topK]
	}

	// 为每个文档获取 LLM 评分
	type scoredDoc struct {
		doc      DocumentWithScore
		llmScore float32
	}

	scoredDocs := make([]scoredDoc, len(docsToRerank))

	for i, docWithScore := range docsToRerank {
		// 生成提示词
		prompt := r.generatePrompt(query, docWithScore.Document.Content)

		// 调用 LLM
		messages := []types.Message{
			types.NewUserMessage(prompt),
		}

		response, err := r.llm.Invoke(ctx, messages)
		if err != nil {
			// 如果 LLM 调用失败，使用原始分数
			scoredDocs[i] = scoredDoc{
				doc:      docWithScore,
				llmScore: docWithScore.Score,
			}
			continue
		}

		// 解析 LLM 响应获取分数
		score, err := r.parseScore(response.Content)
		if err != nil {
			// 如果解析失败，使用原始分数
			scoredDocs[i] = scoredDoc{
				doc:      docWithScore,
				llmScore: docWithScore.Score,
			}
			continue
		}

		// 归一化分数到 0-1
		normalizedScore := float32(score) / 10.0

		scoredDocs[i] = scoredDoc{
			doc:      docWithScore,
			llmScore: normalizedScore,
		}
	}

	// 按 LLM 分数降序排序
	sort.Slice(scoredDocs, func(i, j int) bool {
		return scoredDocs[i].llmScore > scoredDocs[j].llmScore
	})

	// 构建结果
	result := make([]DocumentWithScore, len(docsToRerank))
	for i, scored := range scoredDocs {
		result[i] = DocumentWithScore{
			Document: scored.doc.Document,
			Score:    scored.llmScore,
		}
	}

	// 如果原始文档数量大于 topK，追加剩余文档
	if len(documents) > r.topK {
		result = append(result, documents[r.topK:]...)
	}

	return result, nil
}

// generatePrompt 生成提示词。
func (r *LLMReranker) generatePrompt(query, document string) string {
	// 简单的模板替换
	prompt := r.promptTemplate
	prompt = strings.ReplaceAll(prompt, "{{.Query}}", query)
	prompt = strings.ReplaceAll(prompt, "{{.Document}}", document)
	return prompt
}

// parseScore 从 LLM 响应中解析分数。
func (r *LLMReranker) parseScore(response string) (float64, error) {
	// 去除空白字符
	response = strings.TrimSpace(response)

	// 尝试解析为数字
	score, err := strconv.ParseFloat(response, 64)
	if err != nil {
		// 如果不是纯数字，尝试提取第一个数字
		fields := strings.Fields(response)
		if len(fields) > 0 {
			score, err = strconv.ParseFloat(fields[0], 64)
			if err != nil {
				return 0, fmt.Errorf("failed to parse score from response: %s", response)
			}
		} else {
			return 0, fmt.Errorf("empty response")
		}
	}

	// 验证分数范围
	if score < 0 || score > 10 {
		return 0, fmt.Errorf("score out of range (0-10): %f", score)
	}

	return score, nil
}

// RerankDocuments 是一个便捷函数，直接对文档列表重排序。
//
// 参数：
//   - ctx: 上下文
//   - query: 查询文本
//   - documents: 文档列表（不带分数）
//
// 返回：
//   - []*loaders.Document: 重排序后的文档列表
//   - error: 错误
//
func (r *LLMReranker) RerankDocuments(
	ctx context.Context,
	query string,
	documents []*loaders.Document,
) ([]*loaders.Document, error) {
	// 转换为带分数的文档
	docsWithScore := make([]DocumentWithScore, len(documents))
	for i, doc := range documents {
		docsWithScore[i] = DocumentWithScore{
			Document: doc,
			Score:    1.0, // 初始分数
		}
	}

	// 重排序
	reranked, err := r.Rerank(ctx, query, docsWithScore)
	if err != nil {
		return nil, err
	}

	// 转换回文档列表
	result := make([]*loaders.Document, len(reranked))
	for i, docWithScore := range reranked {
		result[i] = docWithScore.Document
	}

	return result, nil
}

// RerankerVectorStore 是支持 LLM 重排序的向量存储接口。
type RerankerVectorStore interface {
	VectorStore

	// SimilaritySearchWithRerank 使用 LLM 重排序的相似度搜索。
	//
	// 参数：
	//   - ctx: 上下文
	//   - query: 查询文本
	//   - k: 返回结果数量
	//   - reranker: LLM 重排序器
	//
	// 返回：
	//   - []*loaders.Document: 重排序后的文档列表
	//   - error: 错误
	//
	SimilaritySearchWithRerank(
		ctx context.Context,
		query string,
		k int,
		reranker *LLMReranker,
	) ([]*loaders.Document, error)
}

// SimilaritySearchWithRerank 为 InMemoryVectorStore 实现 LLM 重排序搜索。
func (store *InMemoryVectorStore) SimilaritySearchWithRerank(
	ctx context.Context,
	query string,
	k int,
	reranker *LLMReranker,
) ([]*loaders.Document, error) {
	if reranker == nil {
		return nil, fmt.Errorf("reranker is required")
	}

	// 先获取较多的候选文档（用于重排序）
	// 通常获取 k * 3 到 k * 5 个候选
	candidateK := k * 4
	if candidateK > store.GetDocumentCount() {
		candidateK = store.GetDocumentCount()
	}

	// 执行初始搜索
	candidates, err := store.SimilaritySearchWithScore(ctx, query, candidateK)
	if err != nil {
		return nil, fmt.Errorf("initial search failed: %w", err)
	}

	// 使用 LLM 重排序
	reranked, err := reranker.Rerank(ctx, query, candidates)
	if err != nil {
		return nil, fmt.Errorf("reranking failed: %w", err)
	}

	// 取前 k 个结果
	if len(reranked) > k {
		reranked = reranked[:k]
	}

	// 转换为文档列表
	result := make([]*loaders.Document, len(reranked))
	for i, docWithScore := range reranked {
		result[i] = docWithScore.Document
	}

	return result, nil
}
