package chains

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"langchain-go/core/chat"
	"langchain-go/core/prompts"
	"langchain-go/pkg/types"
	"langchain-go/retrieval/loaders"
)

// NewRAGChain 创建 RAG Chain
//
// 参数：
//   - retriever: 检索器，用于检索相关文档
//   - llm: 聊天模型，用于生成答案
//   - opts: 可选配置项
//
// 返回：
//   - *RAGChain: RAG Chain 实例
//
// 使用示例：
//
//	retriever := retrievers.NewVectorStoreRetriever(vectorStore)
//	llm := ollama.NewChatOllama("qwen2.5:7b")
//	ragChain := chains.NewRAGChain(retriever, llm,
//	    chains.WithScoreThreshold(0.7),
//	    chains.WithMaxContextLen(2000),
//	)
//
func NewRAGChain(
	retriever Retriever,
	llm chat.ChatModel,
	opts ...Option,
) *RAGChain {
	chain := &RAGChain{
		retriever: retriever,
		llm:       llm,
		prompt:    DefaultRAGPrompt,
		config:    DefaultRAGConfig(),
	}

	for _, opt := range opts {
		opt(chain)
	}

	return chain
}

// DefaultRAGConfig 默认配置
func DefaultRAGConfig() RAGConfig {
	return RAGConfig{
		ReturnSources:    true,
		ScoreThreshold:   0.0,
		MaxContextLen:    4000,
		TopK:             5,
		ContextFormatter: DefaultContextFormatter,
	}
}

// 配置选项函数

// WithPrompt 设置自定义 prompt 模板
func WithPrompt(prompt *prompts.PromptTemplate) Option {
	return func(c *RAGChain) { c.prompt = prompt }
}

// WithScoreThreshold 设置相似度阈值
func WithScoreThreshold(threshold float32) Option {
	return func(c *RAGChain) { c.config.ScoreThreshold = threshold }
}

// WithMaxContextLen 设置最大上下文长度
func WithMaxContextLen(maxLen int) Option {
	return func(c *RAGChain) { c.config.MaxContextLen = maxLen }
}

// WithReturnSources 设置是否返回来源文档
func WithReturnSources(returnSources bool) Option {
	return func(c *RAGChain) { c.config.ReturnSources = returnSources }
}

// WithTopK 设置返回文档数量
func WithTopK(topK int) Option {
	return func(c *RAGChain) { c.config.TopK = topK }
}

// WithContextFormatter 设置自定义上下文格式化器
func WithContextFormatter(formatter ContextFormatter) Option {
	return func(c *RAGChain) { c.config.ContextFormatter = formatter }
}

// Run 执行 RAG (同步)
//
// 参数：
//   - ctx: 上下文
//   - question: 用户问题
//
// 返回：
//   - RAGResult: 执行结果
//   - error: 错误
//
// 执行流程：
//  1. 检索相关文档
//  2. 过滤低分文档
//  3. 构建上下文
//  4. 格式化 prompt
//  5. 调用 LLM
//  6. 返回结果
//
func (c *RAGChain) Run(ctx context.Context, question string) (RAGResult, error) {
	start := time.Now()

	// 1. 检索相关文档
	docs, err := c.retriever.GetRelevantDocuments(ctx, question)
	if err != nil {
		return RAGResult{}, fmt.Errorf("retrieval failed: %w", err)
	}

	// 2. 过滤低分文档
	relevantDocs := c.filterDocsByScore(docs)
	if len(relevantDocs) == 0 {
		return RAGResult{
			Question:    question,
			Answer:      "抱歉，我没有找到相关信息来回答这个问题。",
			Context:     relevantDocs,
			Confidence:  0.0,
			TimeElapsed: time.Since(start),
			Metadata: map[string]interface{}{
				"total_docs_retrieved": len(docs),
				"filtered_docs":        0,
			},
		}, nil
	}

	// 限制文档数量
	if len(relevantDocs) > c.config.TopK {
		relevantDocs = relevantDocs[:c.config.TopK]
	}

	// 3. 构建上下文
	contextStr := c.config.ContextFormatter(relevantDocs)

	// 4. 限制上下文长度
	if len(contextStr) > c.config.MaxContextLen {
		contextStr = contextStr[:c.config.MaxContextLen] + "...\n[上下文已截断]"
	}

	// 5. 格式化 prompt
	promptStr, err := c.prompt.Format(map[string]interface{}{
		"context":  contextStr,
		"question": question,
	})
	if err != nil {
		return RAGResult{}, fmt.Errorf("prompt formatting failed: %w", err)
	}

	// 6. 调用 LLM
	messages := []types.Message{
		types.NewUserMessage(promptStr),
	}
	response, err := c.llm.Invoke(ctx, messages)
	if err != nil {
		return RAGResult{}, fmt.Errorf("LLM invocation failed: %w", err)
	}

	// 7. 构建结果
	result := RAGResult{
		Question:    question,
		Answer:      response.Content,
		Confidence:  c.calculateConfidence(relevantDocs),
		TimeElapsed: time.Since(start),
		Metadata: map[string]interface{}{
			"total_docs_retrieved": len(docs),
			"filtered_docs":        len(relevantDocs),
			"context_length":       len(contextStr),
		},
	}

	if c.config.ReturnSources {
		result.Context = relevantDocs
	}

	return result, nil
}

// Stream 流式执行
//
// 参数：
//   - ctx: 上下文
//   - question: 用户问题
//
// 返回：
//   - <-chan RAGChunk: 事件流
//   - error: 错误
//
// 事件类型：
//   - "retrieval": 检索完成，Data 为 map[string]interface{}
//   - "llm_token": LLM token 流，Data 为 string
//   - "done": 执行完成，Data 为 RAGResult
//   - "error": 发生错误，Data 为 error
//
func (c *RAGChain) Stream(ctx context.Context, question string) (<-chan RAGChunk, error) {
	resultChan := make(chan RAGChunk, 10)

	go func() {
		defer close(resultChan)
		start := time.Now()

		// 发送开始事件
		resultChan <- RAGChunk{
			Type:      "start",
			Data:      map[string]interface{}{"question": question},
			Timestamp: time.Now(),
		}

		// 1. 检索文档
		docs, err := c.retriever.GetRelevantDocuments(ctx, question)
		if err != nil {
			resultChan <- RAGChunk{
				Type:      "error",
				Data:      err,
				Timestamp: time.Now(),
			}
			return
		}

		// 过滤和限制文档
		relevantDocs := c.filterDocsByScore(docs)
		if len(relevantDocs) > c.config.TopK {
			relevantDocs = relevantDocs[:c.config.TopK]
		}

		// 发送检索事件
		resultChan <- RAGChunk{
			Type: "retrieval",
			Data: map[string]interface{}{
				"documents": relevantDocs,
				"count":     len(relevantDocs),
			},
			Timestamp: time.Now(),
		}

		if len(relevantDocs) == 0 {
			resultChan <- RAGChunk{
				Type: "done",
				Data: RAGResult{
					Question:    question,
					Answer:      "抱歉，我没有找到相关信息来回答这个问题。",
					Context:     relevantDocs,
					Confidence:  0.0,
					TimeElapsed: time.Since(start),
				},
				Timestamp: time.Now(),
			}
			return
		}

		// 2. 构建 prompt
		contextStr := c.config.ContextFormatter(relevantDocs)
		if len(contextStr) > c.config.MaxContextLen {
			contextStr = contextStr[:c.config.MaxContextLen] + "..."
		}

		promptStr, err := c.prompt.Format(map[string]interface{}{
			"context":  contextStr,
			"question": question,
		})
		if err != nil {
			resultChan <- RAGChunk{
				Type:      "error",
				Data:      err,
				Timestamp: time.Now(),
			}
			return
		}

		// 3. 流式 LLM 调用
		messages := []types.Message{types.NewUserMessage(promptStr)}
		streamChan, err := c.llm.Stream(ctx, messages)
		if err != nil {
			resultChan <- RAGChunk{
				Type:      "error",
				Data:      err,
				Timestamp: time.Now(),
			}
			return
		}

		// 收集完整答案
		var fullAnswer strings.Builder

		// 转发 LLM tokens
		for event := range streamChan {
			if event.Error != nil {
				resultChan <- RAGChunk{
					Type:      "error",
					Data:      event.Error,
					Timestamp: time.Now(),
				}
				return
			}

			fullAnswer.WriteString(event.Data.Content)

			resultChan <- RAGChunk{
				Type:      "llm_token",
				Data:      event.Data.Content,
				Timestamp: time.Now(),
			}
		}

		// 完成信号
		resultChan <- RAGChunk{
			Type: "done",
			Data: RAGResult{
				Question:    question,
				Answer:      fullAnswer.String(),
				Context:     relevantDocs,
				Confidence:  c.calculateConfidence(relevantDocs),
				TimeElapsed: time.Since(start),
			},
			Timestamp: time.Now(),
		}
	}()

	return resultChan, nil
}

// Batch 批量执行
//
// 参数：
//   - ctx: 上下文
//   - questions: 问题列表
//
// 返回：
//   - []RAGResult: 结果列表
//   - error: 错误
//
// 注意：批量执行会并行处理所有问题
//
func (c *RAGChain) Batch(ctx context.Context, questions []string) ([]RAGResult, error) {
	if len(questions) == 0 {
		return []RAGResult{}, nil
	}

	results := make([]RAGResult, len(questions))
	errors := make([]error, len(questions))

	// 并行处理
	type result struct {
		idx int
		res RAGResult
		err error
	}

	resultChan := make(chan result, len(questions))
	var wg sync.WaitGroup

	for i, q := range questions {
		wg.Add(1)
		go func(idx int, question string) {
			defer wg.Done()
			res, err := c.Run(ctx, question)
			resultChan <- result{idx: idx, res: res, err: err}
		}(i, q)
	}

	// 等待所有完成
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// 收集结果
	for r := range resultChan {
		results[r.idx] = r.res
		errors[r.idx] = r.err
	}

	// 检查错误
	var errs []string
	for i, err := range errors {
		if err != nil {
			errs = append(errs, fmt.Sprintf("question %d: %v", i, err))
		}
	}

	if len(errs) > 0 {
		return results, fmt.Errorf("batch execution had errors: %s", strings.Join(errs, "; "))
	}

	return results, nil
}

// filterDocsByScore 按分数过滤文档
func (c *RAGChain) filterDocsByScore(docs []*loaders.Document) []*loaders.Document {
	if c.config.ScoreThreshold <= 0 {
		return docs
	}

	var filtered []*loaders.Document
	for _, doc := range docs {
		// 检查 doc.Metadata 中的 "score" 字段
		if score, ok := doc.Metadata["score"].(float32); ok {
			if score >= c.config.ScoreThreshold {
				filtered = append(filtered, doc)
			}
		} else if score, ok := doc.Metadata["score"].(float64); ok {
			if float32(score) >= c.config.ScoreThreshold {
				filtered = append(filtered, doc)
			}
		} else {
			// 没有分数信息，保留
			filtered = append(filtered, doc)
		}
	}
	return filtered
}

// calculateConfidence 计算置信度
//
// 基于检索到的文档分数计算整体置信度。
// 使用平均分数作为置信度指标。
//
func (c *RAGChain) calculateConfidence(docs []*loaders.Document) float64 {
	if len(docs) == 0 {
		return 0.0
	}

	var totalScore float64
	count := 0

	for _, doc := range docs {
		if score, ok := doc.Metadata["score"].(float32); ok {
			totalScore += float64(score)
			count++
		} else if score, ok := doc.Metadata["score"].(float64); ok {
			totalScore += score
			count++
		}
	}

	if count == 0 {
		return 0.5 // 默认置信度
	}

	avgScore := totalScore / float64(count)

	// 归一化到 [0, 1]
	if avgScore > 1.0 {
		avgScore = 1.0
	}
	if avgScore < 0.0 {
		avgScore = 0.0
	}

	return avgScore
}

// DefaultContextFormatter 默认上下文格式化器
//
// 将文档列表格式化为带编号的上下文文本。
//
func DefaultContextFormatter(docs []*loaders.Document) string {
	var builder strings.Builder

	for i, doc := range docs {
		builder.WriteString(fmt.Sprintf("\n[文档 %d]\n", i+1))
		builder.WriteString(doc.Content)
		builder.WriteString("\n")

		// 添加来源信息
		if source, ok := doc.Metadata["source"].(string); ok {
			builder.WriteString(fmt.Sprintf("来源: %s\n", source))
		}

		// 添加分数信息
		if score, ok := doc.Metadata["score"].(float32); ok {
			builder.WriteString(fmt.Sprintf("相关度: %.2f\n", score))
		} else if score, ok := doc.Metadata["score"].(float64); ok {
			builder.WriteString(fmt.Sprintf("相关度: %.2f\n", score))
		}
	}

	return builder.String()
}

// SimpleContextFormatter 简洁的上下文格式化器
//
// 只包含文档内容，不包含来源和分数。
//
func SimpleContextFormatter(docs []*loaders.Document) string {
	var parts []string
	for _, doc := range docs {
		parts = append(parts, doc.Content)
	}
	return strings.Join(parts, "\n\n---\n\n")
}

// StructuredContextFormatter 结构化上下文格式化器
//
// 以 JSON 格式输出文档信息。
//
func StructuredContextFormatter(docs []*loaders.Document) string {
	var builder strings.Builder
	builder.WriteString("[\n")

	for i, doc := range docs {
		if i > 0 {
			builder.WriteString(",\n")
		}
		builder.WriteString(fmt.Sprintf("  {\n    \"content\": %q,\n", doc.Content))

		if source, ok := doc.Metadata["source"].(string); ok {
			builder.WriteString(fmt.Sprintf("    \"source\": %q,\n", source))
		}

		if score, ok := doc.Metadata["score"].(float32); ok {
			builder.WriteString(fmt.Sprintf("    \"score\": %.2f\n", score))
		} else if score, ok := doc.Metadata["score"].(float64); ok {
			builder.WriteString(fmt.Sprintf("    \"score\": %.2f\n", score))
		}

		builder.WriteString("  }")
	}

	builder.WriteString("\n]")
	return builder.String()
}

// DefaultRAGPrompt 从模板库导入
//
// 使用预定义的 RAG prompt 模板
//
var DefaultRAGPrompt = &prompts.PromptTemplate{
	Template: `基于以下上下文回答问题。如果上下文中没有相关信息，请明确说明无法回答。

上下文:
{{.context}}

问题: {{.question}}

回答:`,
	InputVariables: []string{"context", "question"},
}

// 注意: 可以使用 core/prompts/templates.DefaultRAGPrompt 替代
// import "langchain-go/core/prompts/templates"
// var DefaultRAGPrompt = templates.DefaultRAGPrompt
