package retrievers

import (
	"context"
	"fmt"
	"strings"

	"github.com/zhucl121/langchain-go/core/chat"
	"github.com/zhucl121/langchain-go/core/prompts"
	"github.com/zhucl121/langchain-go/pkg/types"
	"github.com/zhucl121/langchain-go/retrieval/loaders"
)

// MultiQueryRetriever 多查询检索器
//
// 使用 LLM 生成多个查询变体，从不同角度检索文档，提高召回率。
//
// 工作原理：
//  1. 使用 LLM 为原始查询生成多个变体
//  2. 对每个查询变体执行检索
//  3. 合并和去重所有结果
//
type MultiQueryRetriever struct {
	*BaseRetriever
	baseRetriever   Retriever
	llm             chat.ChatModel
	prompt          *prompts.PromptTemplate
	includeOriginal bool
	numQueries      int
}

// NewMultiQueryRetriever 创建多查询检索器
//
// 参数：
//   - baseRetriever: 基础检索器
//   - llm: 聊天模型，用于生成查询变体
//   - opts: 可选配置项
//
// 返回：
//   - *MultiQueryRetriever: 检索器实例
//
// 使用示例：
//
//	retriever := retrievers.NewMultiQueryRetriever(
//	    baseRetriever,
//	    llm,
//	    retrievers.WithNumQueries(3),
//	    retrievers.WithIncludeOriginal(true),
//	)
//
func NewMultiQueryRetriever(
	baseRetriever Retriever,
	llm chat.ChatModel,
	opts ...MultiQueryOption,
) *MultiQueryRetriever {
	r := &MultiQueryRetriever{
		BaseRetriever:   NewBaseRetriever(),
		baseRetriever:   baseRetriever,
		llm:             llm,
		prompt:          DefaultMultiQueryPrompt,
		includeOriginal: true,
		numQueries:      3,
	}

	for _, opt := range opts {
		opt(r)
	}

	return r
}

// 配置选项函数

// WithIncludeOriginal 设置是否包含原始查询
func WithIncludeOriginal(include bool) MultiQueryOption {
	return func(r *MultiQueryRetriever) {
		r.includeOriginal = include
	}
}

// WithNumQueries 设置生成的查询数量
func WithNumQueries(num int) MultiQueryOption {
	return func(r *MultiQueryRetriever) {
		r.numQueries = num
	}
}

// WithMultiQueryPrompt 设置自定义 prompt
func WithMultiQueryPrompt(prompt *prompts.PromptTemplate) MultiQueryOption {
	return func(r *MultiQueryRetriever) {
		r.prompt = prompt
	}
}

// GetRelevantDocuments 实现 Retriever 接口
func (r *MultiQueryRetriever) GetRelevantDocuments(ctx context.Context, query string) ([]*loaders.Document, error) {
	// 触发开始回调
	r.triggerStart(ctx, query)

	// 1. 生成多个查询变体
	queries, err := r.generateQueries(ctx, query)
	if err != nil {
		r.triggerError(ctx, err)
		return nil, fmt.Errorf("failed to generate queries: %w", err)
	}

	// 包含原始查询
	if r.includeOriginal {
		queries = append([]string{query}, queries...)
	}

	// 2. 对每个查询检索，并去重
	allDocs := make(map[string]*loaders.Document) // 使用内容哈希去重

	for _, q := range queries {
		docs, err := r.baseRetriever.GetRelevantDocuments(ctx, q)
		if err != nil {
			// 忽略单个查询的错误，继续处理
			continue
		}

		for _, doc := range docs {
			// 使用内容哈希作为 key 去重
			key := hashContent(doc.Content)
			if _, exists := allDocs[key]; !exists {
				allDocs[key] = doc
			}
		}
	}

	// 3. 返回去重后的文档
	result := make([]*loaders.Document, 0, len(allDocs))
	for _, doc := range allDocs {
		result = append(result, doc)
	}

	// 触发结束回调
	r.triggerEnd(ctx, result)

	return result, nil
}

// GetRelevantDocumentsWithScore 实现 Retriever 接口
func (r *MultiQueryRetriever) GetRelevantDocumentsWithScore(ctx context.Context, query string) ([]DocumentWithScore, error) {
	// 触发开始回调
	r.triggerStart(ctx, query)

	// 1. 生成多个查询变体
	queries, err := r.generateQueries(ctx, query)
	if err != nil {
		r.triggerError(ctx, err)
		return nil, fmt.Errorf("failed to generate queries: %w", err)
	}

	if r.includeOriginal {
		queries = append([]string{query}, queries...)
	}

	// 2. 对每个查询检索带分数的结果
	type docWithMaxScore struct {
		doc      *loaders.Document
		maxScore float32
	}
	allDocs := make(map[string]*docWithMaxScore) // 内容哈希 -> 文档和最高分数

	for _, q := range queries {
		docs, err := r.baseRetriever.GetRelevantDocumentsWithScore(ctx, q)
		if err != nil {
			continue
		}

		for _, docScore := range docs {
			key := hashContent(docScore.Document.Content)

			if existing, exists := allDocs[key]; exists {
				// 保留最高分数
				if docScore.Score > existing.maxScore {
					existing.maxScore = docScore.Score
				}
			} else {
				allDocs[key] = &docWithMaxScore{
					doc:      docScore.Document,
					maxScore: docScore.Score,
				}
			}
		}
	}

	// 3. 转换为结果
	result := make([]DocumentWithScore, 0, len(allDocs))
	for _, item := range allDocs {
		// 确保分数在元数据中
		if item.doc.Metadata == nil {
			item.doc.Metadata = make(map[string]interface{})
		}
		item.doc.Metadata["score"] = item.maxScore

		result = append(result, DocumentWithScore{
			Document: item.doc,
			Score:    item.maxScore,
		})
	}

	// 触发结束回调
	plainDocs := make([]*loaders.Document, len(result))
	for i, d := range result {
		plainDocs[i] = d.Document
	}
	r.triggerEnd(ctx, plainDocs)

	return result, nil
}

// generateQueries 使用 LLM 生成查询变体
func (r *MultiQueryRetriever) generateQueries(ctx context.Context, query string) ([]string, error) {
	// 格式化 prompt
	promptStr, err := r.prompt.Format(map[string]interface{}{
		"question":    query,
		"num_queries": r.numQueries,
	})
	if err != nil {
		return nil, fmt.Errorf("prompt formatting failed: %w", err)
	}

	// 调用 LLM
	messages := []types.Message{types.NewUserMessage(promptStr)}
	response, err := r.llm.Invoke(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("LLM invocation failed: %w", err)
	}

	// 解析生成的查询列表
	queries := parseQueries(response.Content)

	// 限制数量
	if len(queries) > r.numQueries {
		queries = queries[:r.numQueries]
	}

	return queries, nil
}

// parseQueries 解析 LLM 生成的查询列表
//
// 支持多种格式：
//   - 编号列表: "1. 查询1\n2. 查询2"
//   - 短横线列表: "- 查询1\n- 查询2"
//   - 纯文本: "查询1\n查询2"
//
func parseQueries(content string) []string {
	lines := strings.Split(content, "\n")
	var queries []string

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// 跳过空行和标题
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "查询") {
			continue
		}

		// 移除编号 (1. 2. 3. 或 1) 2) 3))
		line = strings.TrimPrefix(line, "- ")
		line = strings.TrimPrefix(line, "* ")

		// 移除数字编号
		for i := 1; i <= 20; i++ {
			line = strings.TrimPrefix(line, fmt.Sprintf("%d. ", i))
			line = strings.TrimPrefix(line, fmt.Sprintf("%d) ", i))
			line = strings.TrimPrefix(line, fmt.Sprintf("%d）", i))
		}

		line = strings.TrimSpace(line)

		// 添加非空查询
		if line != "" && !strings.HasPrefix(line, "```") {
			queries = append(queries, line)
		}
	}

	return queries
}

// DefaultMultiQueryPrompt 默认多查询 prompt
var DefaultMultiQueryPrompt, _ = prompts.NewPromptTemplate(prompts.PromptTemplateConfig{
	Template: `你是一个 AI 助手，帮助生成多个搜索查询。

用户问题: {{.question}}

请生成 {{.num_queries}} 个相关但措辞不同的搜索查询，以便从不同角度检索相关信息。
这些查询应该:
1. 表达相同的核心意图
2. 使用不同的措辞和表达方式
3. 从不同角度提问

每个查询一行，不需要编号或其他格式。

查询列表:
`,
	InputVariables: []string{"question", "num_queries"},
})

// ChineseMultiQueryPrompt 中文优化的多查询 prompt
var ChineseMultiQueryPrompt, _ = prompts.NewPromptTemplate(prompts.PromptTemplateConfig{
	Template: `请为以下问题生成 {{.num_queries}} 个相似但表达不同的搜索查询。

原问题: {{.question}}

要求:
- 保持核心意图不变
- 使用不同的词汇和句式
- 从不同角度思考

请直接列出查询，每行一个:
`,
	InputVariables: []string{"question", "num_queries"},
})

// EnglishMultiQueryPrompt 英文多查询 prompt
var EnglishMultiQueryPrompt, _ = prompts.NewPromptTemplate(prompts.PromptTemplateConfig{
	Template: `You are an AI assistant that helps generate multiple search queries.

Original question: {{.question}}

Please generate {{.num_queries}} alternative search queries that:
1. Express the same core intent
2. Use different wording and phrasing
3. Approach the topic from different angles

List the queries, one per line:
`,
	InputVariables: []string{"question", "num_queries"},
})
