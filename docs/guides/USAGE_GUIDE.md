# LangChain-Go 扩展功能使用指南

## 🎉 欢迎使用 LangChain-Go 扩展功能!

本指南介绍如何使用新实现的高层 API,让您用 **3 行代码** 完成原本需要 **150 行** 的 RAG 应用!

---

## 📦 新增功能概览

### 1. RAG Chain - 检索增强生成

**之前** (150+ 行手动代码):
```go
func (r *RAGService) Query(ctx context.Context, req QueryRequest) (*QueryResponse, error) {
    // 手动检索文档
    retrieved, err := r.vectorStore.SimilaritySearch(ctx, req.Question, req.TopK)
    
    // 手动过滤低分文档
    var relevantDocs []*Document
    for _, doc := range retrieved {
        if doc.Score >= req.MinScore {
            relevantDocs = append(relevantDocs, doc)
        }
    }
    
    // 手动构建上下文
    var context strings.Builder
    for i, doc := range relevantDocs {
        context.WriteString(fmt.Sprintf("[文档 %d]\n%s\n", i+1, doc.Content))
    }
    
    // 手动构建 prompt
    prompt := fmt.Sprintf(`基于以下上下文回答问题...
上下文: %s
问题: %s`, context.String(), req.Question)
    
    // 手动调用 LLM
    messages := []types.Message{types.NewUserMessage(prompt)}
    response, err := r.chatModel.Invoke(ctx, messages)
    
    // ... 更多手动处理
    return &QueryResponse{...}, nil
}
```

**现在** (3 行代码):
```go
retriever := retrievers.NewVectorStoreRetriever(vectorStore)
ragChain := chains.NewRAGChain(retriever, llm)
result, _ := ragChain.Run(ctx, "What is LangChain?")
```

**效率提升**: **50x** 🚀

---

## 🚀 快速开始

### 安装

```bash
go get github.com/zhuchenglong/langchain-go/retrieval/chains
go get github.com/zhuchenglong/langchain-go/retrieval/retrievers
```

### 基础 RAG 应用

```go
package main

import (
    "context"
    "fmt"
    
    "github.com/zhuchenglong/langchain-go/core/chat/ollama"
    "github.com/zhuchenglong/langchain-go/retrieval/chains"
    "github.com/zhuchenglong/langchain-go/retrieval/embeddings"
    "github.com/zhuchenglong/langchain-go/retrieval/loaders"
    "github.com/zhuchenglong/langchain-go/retrieval/retrievers"
    "github.com/zhuchenglong/langchain-go/retrieval/vectorstores"
)

func main() {
    ctx := context.Background()
    
    // 步骤 1: 准备文档
    docs := []*loaders.Document{
        {Content: "LangChain 是一个用于构建 LLM 应用的框架"},
        {Content: "RAG 结合了检索和生成两个步骤"},
    }
    
    // 步骤 2: 创建向量存储并添加文档
    embedder := embeddings.NewOllamaEmbeddings("nomic-embed-text")
    vectorStore := vectorstores.NewInMemoryVectorStore(embedder)
    vectorStore.AddDocuments(ctx, docs)
    
    // 步骤 3: 创建检索器
    retriever := retrievers.NewVectorStoreRetriever(vectorStore)
    
    // 步骤 4: 创建 RAG Chain
    llm := ollama.NewChatOllama("qwen2.5:7b")
    ragChain := chains.NewRAGChain(retriever, llm)
    
    // 步骤 5: 执行查询
    result, err := ragChain.Run(ctx, "什么是 RAG?")
    if err != nil {
        panic(err)
    }
    
    // 步骤 6: 输出结果
    fmt.Println("答案:", result.Answer)
    fmt.Printf("置信度: %.2f\n", result.Confidence)
    fmt.Printf("耗时: %v\n", result.TimeElapsed)
}
```

---

## 💡 高级功能

### 1. 配置选项

```go
ragChain := chains.NewRAGChain(retriever, llm,
    chains.WithScoreThreshold(0.7),        // 设置相似度阈值
    chains.WithMaxContextLen(2000),        // 限制上下文长度
    chains.WithTopK(3),                    // 返回 top 3 文档
    chains.WithReturnSources(true),        // 返回来源文档
    chains.WithPrompt(customPrompt),       // 自定义 prompt
)
```

### 2. 流式输出

实时显示 LLM 生成的内容:

```go
stream, _ := ragChain.Stream(ctx, "Explain LangChain")

for chunk := range stream {
    switch chunk.Type {
    case "retrieval":
        fmt.Println("✓ 检索完成")
    case "llm_token":
        fmt.Print(chunk.Data) // 实时打印 token
    case "done":
        fmt.Println("\n✓ 完成")
    }
}
```

### 3. 批量处理

并行处理多个问题:

```go
questions := []string{
    "什么是 LangChain?",
    "什么是 RAG?",
    "如何使用向量数据库?",
}

results, _ := ragChain.Batch(ctx, questions)

for i, result := range results {
    fmt.Printf("Q%d: %s\nA%d: %s\n\n", i+1, questions[i], i+1, result.Answer)
}
```

### 4. 自定义 Prompt

使用预定义模板:

```go
import "github.com/zhuchenglong/langchain-go/core/prompts/templates"

ragChain := chains.NewRAGChain(retriever, llm,
    chains.WithPrompt(templates.DetailedRAGPrompt),
)
```

或创建自定义 prompt:

```go
customPrompt, _ := prompts.NewPromptTemplate(prompts.PromptTemplateConfig{
    Template: `你是一个专业的技术顾问。

参考资料:
{{.context}}

用户问题: {{.question}}

请提供详细的技术解答:`,
    InputVariables: []string{"context", "question"},
})

ragChain := chains.NewRAGChain(retriever, llm,
    chains.WithPrompt(customPrompt),
)
```

---

## 🔍 高级检索器

### 1. 多查询检索器

使用 LLM 生成多个查询变体,提高召回率:

```go
import "github.com/zhuchenglong/langchain-go/retrieval/retrievers"

// 基础检索器
baseRetriever := retrievers.NewVectorStoreRetriever(vectorStore)

// 多查询检索器
multiRetriever := retrievers.NewMultiQueryRetriever(
    baseRetriever,
    llm,
    retrievers.WithNumQueries(3),          // 生成 3 个查询变体
    retrievers.WithIncludeOriginal(true),  // 包含原始查询
)

// 自动生成多个查询并检索
docs, _ := multiRetriever.GetRelevantDocuments(ctx, "如何使用 LangChain?")
```

**工作原理**:
1. LLM 为原始查询生成 3 个不同措辞的变体
2. 对每个变体分别检索
3. 合并去重结果

### 2. 集成检索器 (混合检索)

融合多个检索器的结果,使用 RRF 算法:

```go
// 向量检索器
vectorRetriever := retrievers.NewVectorStoreRetriever(vectorStore)

// BM25 检索器 (如果已实现)
// bm25Retriever := retrievers.NewBM25Retriever(documents)

// 集成检索器
ensemble := retrievers.NewEnsembleRetriever(
    []retrievers.Retriever{vectorRetriever /*, bm25Retriever*/},
    retrievers.WithWeights([]float64{0.5, 0.5}), // 等权重
    retrievers.WithRRFK(60),                     // RRF 常数
)

// 自动融合结果
docs, _ := ensemble.GetRelevantDocuments(ctx, "question")
```

**RRF (Reciprocal Rank Fusion)**:
- 对每个检索器的结果按排名计算分数
- 分数公式: `score = weight / (k + rank)`
- 合并相同文档的分数
- 按最终分数排序

---

## 📚 Prompt 模板库

### 预定义模板

```go
import "github.com/zhuchenglong/langchain-go/core/prompts/templates"

// RAG 模板
templates.DefaultRAGPrompt        // 默认 RAG prompt
templates.DetailedRAGPrompt       // 详细的 RAG prompt
templates.ConversationalRAGPrompt // 对话式 RAG prompt
templates.MultilingualRAGPrompt   // 多语言 RAG prompt
templates.StructuredRAGPrompt     // 结构化 RAG (返回 JSON)
templates.ConciseRAGPrompt        // 简洁的 RAG prompt

// Agent 模板
templates.ReActPrompt             // ReAct Agent prompt
templates.ChineseReActPrompt      // 中文 ReAct prompt
templates.PlanExecutePrompt       // Plan-Execute prompt
templates.ToolCallingPrompt       // Tool Calling prompt

// 其他模板
templates.SummarizationPrompt     // 摘要
templates.TranslationPrompt       // 翻译
templates.CodeExplanationPrompt   // 代码解释
templates.SentimentAnalysisPrompt // 情感分析
```

### 使用模板

```go
// 方式 1: 直接使用
ragChain := chains.NewRAGChain(retriever, llm,
    chains.WithPrompt(templates.DetailedRAGPrompt),
)

// 方式 2: 通过名称获取
prompt := templates.GetRAGTemplate("detailed")
ragChain := chains.NewRAGChain(retriever, llm,
    chains.WithPrompt(prompt),
)
```

---

## 🎯 实际应用示例

### 示例 1: 技术文档问答系统

```go
func main() {
    ctx := context.Background()
    
    // 加载技术文档
    loader := loaders.NewDirectoryLoader("./docs", "*.md")
    docs, _ := loader.Load()
    
    // 创建向量存储
    embedder := embeddings.NewOllamaEmbeddings("nomic-embed-text")
    vectorStore := vectorstores.NewInMemoryVectorStore(embedder)
    vectorStore.AddDocuments(ctx, docs)
    
    // 创建 RAG Chain
    retriever := retrievers.NewVectorStoreRetriever(vectorStore,
        retrievers.WithTopK(3),
        retrievers.WithScoreThreshold(0.7),
    )
    
    llm := ollama.NewChatOllama("qwen2.5:7b")
    ragChain := chains.NewRAGChain(retriever, llm,
        chains.WithPrompt(templates.DetailedRAGPrompt),
    )
    
    // 交互式问答
    for {
        fmt.Print("\n问题: ")
        var question string
        fmt.Scanln(&question)
        
        if question == "exit" {
            break
        }
        
        result, err := ragChain.Run(ctx, question)
        if err != nil {
            fmt.Println("错误:", err)
            continue
        }
        
        fmt.Println("\n答案:", result.Answer)
        fmt.Printf("置信度: %.2f\n", result.Confidence)
        
        if len(result.Context) > 0 {
            fmt.Println("\n来源:")
            for i, doc := range result.Context {
                source := doc.Metadata["source"].(string)
                fmt.Printf("  [%d] %s\n", i+1, source)
            }
        }
    }
}
```

### 示例 2: 多语言智能客服

```go
func main() {
    // ... 初始化代码 ...
    
    // 使用多语言 prompt
    ragChain := chains.NewRAGChain(retriever, llm,
        chains.WithPrompt(templates.MultilingualRAGPrompt),
    )
    
    // 自动适应用户语言
    questions := []string{
        "What is your return policy?",      // 英文
        "退货政策是什么?",                    // 中文
        "Quelle est votre politique?",      // 法文
    }
    
    results, _ := ragChain.Batch(ctx, questions)
    
    for i, result := range results {
        fmt.Printf("Q: %s\nA: %s\n\n", questions[i], result.Answer)
    }
}
```

### 示例 3: 流式实时问答

```go
func streamingQA(ragChain *chains.RAGChain, question string) {
    ctx := context.Background()
    
    fmt.Printf("问题: %s\n\n", question)
    fmt.Print("回答: ")
    
    stream, err := ragChain.Stream(ctx, question)
    if err != nil {
        panic(err)
    }
    
    for chunk := range stream {
        switch chunk.Type {
        case "start":
            fmt.Print("🤔 思考中...")
            
        case "retrieval":
            data := chunk.Data.(map[string]interface{})
            count := data["count"].(int)
            fmt.Printf("\r✓ 找到 %d 个相关文档\n\n", count)
            
        case "llm_token":
            fmt.Print(chunk.Data.(string))
            
        case "done":
            result := chunk.Data.(chains.RAGResult)
            fmt.Printf("\n\n✓ 完成 (耗时: %v, 置信度: %.2f)\n",
                result.TimeElapsed, result.Confidence)
                
        case "error":
            fmt.Printf("\n❌ 错误: %v\n", chunk.Data)
        }
    }
}
```

---

## 🔧 配置最佳实践

### 1. 阈值设置

```go
// 严格模式 (高精度)
ragChain := chains.NewRAGChain(retriever, llm,
    chains.WithScoreThreshold(0.8),
    chains.WithTopK(2),
)

// 平衡模式 (推荐)
ragChain := chains.NewRAGChain(retriever, llm,
    chains.WithScoreThreshold(0.7),
    chains.WithTopK(3),
)

// 宽松模式 (高召回)
ragChain := chains.NewRAGChain(retriever, llm,
    chains.WithScoreThreshold(0.5),
    chains.WithTopK(5),
)
```

### 2. 上下文长度

根据 LLM 的上下文窗口设置:

```go
// 7B 模型 (上下文窗口 4096)
ragChain := chains.NewRAGChain(retriever, llm,
    chains.WithMaxContextLen(2000),
)

// 70B 模型 (上下文窗口 128K)
ragChain := chains.NewRAGChain(retriever, llm,
    chains.WithMaxContextLen(10000),
)
```

### 3. 错误处理

```go
result, err := ragChain.Run(ctx, question)
if err != nil {
    log.Printf("RAG 执行失败: %v", err)
    return
}

// 检查置信度
if result.Confidence < 0.5 {
    log.Println("警告: 低置信度回答")
}

// 检查是否有文档
if len(result.Context) == 0 {
    log.Println("警告: 未找到相关文档")
}
```

---

## 📖 API 参考

### RAGChain

```go
type RAGChain struct {
    // 私有字段
}

// 创建
func NewRAGChain(retriever Retriever, llm ChatModel, opts ...Option) *RAGChain

// 执行
func (c *RAGChain) Run(ctx context.Context, question string) (RAGResult, error)
func (c *RAGChain) Stream(ctx context.Context, question string) (<-chan RAGChunk, error)
func (c *RAGChain) Batch(ctx context.Context, questions []string) ([]RAGResult, error)

// 配置选项
func WithPrompt(prompt *PromptTemplate) Option
func WithScoreThreshold(threshold float32) Option
func WithMaxContextLen(maxLen int) Option
func WithReturnSources(returnSources bool) Option
func WithTopK(topK int) Option
func WithContextFormatter(formatter ContextFormatter) Option
```

### Retriever

```go
type Retriever interface {
    GetRelevantDocuments(ctx context.Context, query string) ([]*Document, error)
    GetRelevantDocumentsWithScore(ctx context.Context, query string) ([]DocumentWithScore, error)
}

// VectorStoreRetriever
func NewVectorStoreRetriever(store VectorStore, opts ...VectorStoreOption) *VectorStoreRetriever
func WithSearchType(searchType SearchType) VectorStoreOption
func WithTopK(k int) VectorStoreOption
func WithScoreThreshold(threshold float32) VectorStoreOption

// MultiQueryRetriever
func NewMultiQueryRetriever(baseRetriever Retriever, llm ChatModel, opts ...MultiQueryOption) *MultiQueryRetriever
func WithNumQueries(num int) MultiQueryOption
func WithIncludeOriginal(include bool) MultiQueryOption

// EnsembleRetriever
func NewEnsembleRetriever(retrievers []Retriever, opts ...EnsembleOption) *EnsembleRetriever
func WithWeights(weights []float64) EnsembleOption
func WithRRFK(k int) EnsembleOption
```

---

## 🎓 学习资源

- **实施计划**: `EXTENSION_IMPLEMENTATION_PLAN.md`
- **实施总结**: `IMPLEMENTATION_SUMMARY.md`
- **Python API 参考**: `PYTHON_API_REFERENCE.md`
- **功能对比**: `PYTHON_VS_GO_COMPARISON.md`

---

## 🤝 贡献

欢迎贡献代码、报告问题或提出建议!

---

## 📄 许可证

MIT License

---

**Happy Coding!** 🚀
