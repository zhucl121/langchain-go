# LLM-based Reranking 使用指南

**创建日期**: 2026-01-15  
**版本**: v1.0  
**状态**: ✅ 已完成

---

## 📋 简介

LLM-based Reranking 是一种使用大语言模型（LLM）对检索结果进行重排序的高级技术。相比基于向量相似度的排序，LLM 可以更好地理解语义和上下文，从而提供更准确的排序结果。

### 核心优势

- **语义理解更深** - LLM 能理解复杂的语义关系
- **上下文感知** - 考虑查询和文档的整体语境
- **高准确度** - 显著提升检索精度
- **灵活性强** - 可通过提示词定制评分标准

---

## 🚀 快速开始

### 基础使用

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/zhucl121/langchain-go/core/chat/providers/openai"
    "github.com/zhucl121/langchain-go/retrieval/embeddings"
    "github.com/zhucl121/langchain-go/retrieval/loaders"
    "github.com/zhucl121/langchain-go/retrieval/vectorstores"
)

func main() {
    ctx := context.Background()

    // 1. 创建 LLM 模型
    llm, err := openai.NewChatOpenAI(openai.Config{
        APIKey: "your-api-key",
        Model:  "gpt-4",
    })
    if err != nil {
        log.Fatal(err)
    }

    // 2. 创建 LLM 重排序器
    reranker, err := vectorstores.NewLLMReranker(vectorstores.LLMRerankerConfig{
        LLM:  llm,
        TopK: 20, // 只对前 20 个结果重排序
    })
    if err != nil {
        log.Fatal(err)
    }

    // 3. 创建向量存储并添加文档
    emb := embeddings.NewOpenAIEmbeddings("your-api-key")
    store := vectorstores.NewInMemoryVectorStore(emb)

    docs := []*loaders.Document{
        loaders.NewDocument("AI is transforming technology", nil),
        loaders.NewDocument("Machine learning is powerful", nil),
        loaders.NewDocument("The weather is nice today", nil),
    }
    store.AddDocuments(ctx, docs)

    // 4. 使用 LLM 重排序搜索
    results, err := store.SimilaritySearchWithRerank(
        ctx,
        "artificial intelligence",
        5,       // 返回 5 个结果
        reranker, // 使用 LLM 重排序
    )
    
    if err != nil {
        log.Fatal(err)
    }

    // 5. 打印结果
    for i, doc := range results {
        fmt.Printf("%d. %s\n", i+1, doc.Content)
    }
}
```

---

## ⚙️ 配置选项

### LLMRerankerConfig

```go
type LLMRerankerConfig struct {
    // LLM 模型（必需）
    LLM chat.ChatModel

    // 提示词模板（可选）
    // 默认会要求 LLM 评分 0-10
    PromptTemplate string

    // 只对前 TopK 个结果重排序（可选）
    // 默认为 20
    TopK int
}
```

### 示例：自定义配置

```go
config := vectorstores.LLMRerankerConfig{
    LLM: llm,
    PromptTemplate: `评估文档与查询的相关性。

查询: {{.Query}}
文档: {{.Document}}

请给出 0-10 的分数（10 表示最相关）：`,
    TopK: 30, // 对前 30 个结果重排序
}

reranker, err := vectorstores.NewLLMReranker(config)
```

---

## 💡 使用场景

### 1. 精准搜索

当需要极高的搜索准确度时使用 LLM reranking：

```go
// 场景：法律文档搜索，需要精确匹配
reranker, _ := vectorstores.NewLLMReranker(vectorstores.LLMRerankerConfig{
    LLM:  gpt4,
    TopK: 15,
})

// 搜索并重排序
results, _ := store.SimilaritySearchWithRerank(
    ctx,
    "合同违约的法律责任",
    5,
    reranker,
)
```

### 2. 复杂查询

对于复杂的、多层次的查询：

```go
// 复杂查询示例
query := `
找到关于以下主题的文档：
1. 人工智能在医疗领域的应用
2. 特别关注诊断准确性
3. 必须包含真实案例
`

results, _ := store.SimilaritySearchWithRerank(ctx, query, 10, reranker)
```

### 3. 领域特定搜索

通过自定义提示词进行领域特定的重排序：

```go
// 医疗领域特定重排序
config := vectorstores.LLMRerankerConfig{
    LLM: llm,
    PromptTemplate: `作为医疗专家，评估文档的相关性。

查询: {{.Query}}
文档: {{.Document}}

考虑因素：
- 医学准确性
- 证据等级
- 临床应用价值

评分 (0-10):`,
    TopK: 20,
}

reranker, _ := vectorstores.NewLLMReranker(config)
```

---

## 🎯 最佳实践

### 1. 选择合适的 TopK

```go
// 文档库规模与 TopK 的关系
var topK int
switch {
case docCount < 100:
    topK = 20  // 小型库：重排序前 20 个
case docCount < 1000:
    topK = 50  // 中型库：重排序前 50 个
default:
    topK = 100 // 大型库：重排序前 100 个
}

config := vectorstores.LLMRerankerConfig{
    LLM:  llm,
    TopK: topK,
}
```

### 2. 两阶段检索策略

```go
// 阶段 1：向量搜索快速筛选
candidateK := 50
candidates, _ := store.SimilaritySearch(ctx, query, candidateK)

// 转换为 DocumentWithScore
docsWithScore := make([]vectorstores.DocumentWithScore, len(candidates))
for i, doc := range candidates {
    docsWithScore[i] = vectorstores.DocumentWithScore{
        Document: doc,
        Score:    float32(candidateK-i) / float32(candidateK),
    }
}

// 阶段 2：LLM 精确重排序
reranked, _ := reranker.Rerank(ctx, query, docsWithScore)

// 取前 k 个
finalResults := reranked[:5]
```

### 3. 成本优化

LLM Reranking 会调用 LLM API，需要考虑成本：

```go
// 策略 1：仅对高价值查询使用 LLM 重排序
func adaptiveSearch(ctx context.Context, query string, isImportant bool) ([]*loaders.Document, error) {
    if isImportant {
        // 重要查询：使用 LLM 重排序
        return store.SimilaritySearchWithRerank(ctx, query, 5, reranker)
    } else {
        // 普通查询：仅使用向量搜索
        return store.SimilaritySearch(ctx, query, 5)
    }
}

// 策略 2：使用更便宜的模型
cheapLLM, _ := openai.NewChatOpenAI(openai.Config{
    APIKey: apiKey,
    Model:  "gpt-3.5-turbo", // 更便宜
})

reranker, _ := vectorstores.NewLLMReranker(vectorstores.LLMRerankerConfig{
    LLM:  cheapLLM,
    TopK: 10, // 减少 LLM 调用次数
})
```

### 4. 批量重排序

如果需要为多个查询重排序，可以批量处理：

```go
func batchRerank(
    ctx context.Context,
    queries []string,
    documents []vectorstores.DocumentWithScore,
    reranker *vectorstores.LLMReranker,
) ([][]vectorstores.DocumentWithScore, error) {
    results := make([][]vectorstores.DocumentWithScore, len(queries))
    
    for i, query := range queries {
        reranked, err := reranker.Rerank(ctx, query, documents)
        if err != nil {
            return nil, err
        }
        results[i] = reranked
    }
    
    return results, nil
}
```

---

## 📊 性能对比

### 向量搜索 vs LLM Reranking

| 指标 | 向量搜索 | LLM Reranking |
|------|---------|---------------|
| 速度 | ⚡️ 极快 (< 100ms) | 🐢 较慢 (1-5s) |
| 准确度 | ⭐⭐⭐ | ⭐⭐⭐⭐⭐ |
| 成本 | 💰 低 | 💰💰💰 高 |
| 复杂查询支持 | ⭐⭐ | ⭐⭐⭐⭐⭐ |

### 何时使用

#### ✅ 应该使用 LLM Reranking

- 搜索准确度至关重要
- 查询较为复杂
- 文档内容专业性强
- 有预算支持 LLM API 调用

#### ❌ 不需要 LLM Reranking

- 实时性要求极高（< 200ms）
- 简单的关键词匹配
- 预算有限
- 向量搜索已经足够准确

---

## 🔧 高级用法

### 1. 自定义评分标准

```go
// 多维度评分
config := vectorstores.LLMRerankerConfig{
    LLM: llm,
    PromptTemplate: `评估文档质量，考虑以下维度：

查询: {{.Query}}
文档: {{.Document}}

评分维度（每项 0-2 分）：
1. 相关性: [0-2]
2. 完整性: [0-2]
3. 可信度: [0-2]
4. 时效性: [0-2]
5. 可读性: [0-2]

总分 (0-10):`,
}
```

### 2. 结合 MMR 使用

可以先用 MMR 保证多样性，再用 LLM 精排：

```go
// 步骤 1：MMR 搜索（保证多样性）
mmrOptions := &vectorstores.MMROptions{
    Lambda: 0.5,
    FetchK: 30,
}
mmrResults, _ := store.SimilaritySearchWithMMR(ctx, query, 15, mmrOptions)

// 步骤 2：转换为带分数的文档
docsWithScore := make([]vectorstores.DocumentWithScore, len(mmrResults))
for i, doc := range mmrResults {
    docsWithScore[i] = vectorstores.DocumentWithScore{
        Document: doc,
        Score:    1.0,
    }
}

// 步骤 3：LLM 重排序（提升准确度）
finalResults, _ := reranker.Rerank(ctx, query, docsWithScore)
```

### 3. 错误处理和降级

```go
func robustSearch(
    ctx context.Context,
    query string,
    k int,
) ([]*loaders.Document, error) {
    // 尝试使用 LLM 重排序
    results, err := store.SimilaritySearchWithRerank(ctx, query, k, reranker)
    if err != nil {
        // LLM 调用失败，降级到向量搜索
        log.Printf("LLM reranking failed, falling back to vector search: %v", err)
        return store.SimilaritySearch(ctx, query, k)
    }
    
    return results, nil
}
```

---

## 🎓 实际案例

### 案例 1：技术文档搜索

```go
// 技术文档库
docs := []*loaders.Document{
    loaders.NewDocument("Python 字典的基本用法", nil),
    loaders.NewDocument("Python 字典性能优化技巧", nil),
    loaders.NewDocument("Python 列表推导式", nil),
    loaders.NewDocument("Python 字典与 JSON 转换", nil),
}

store.AddDocuments(ctx, docs)

// 查询
query := "如何优化 Python 字典的性能？"

// 向量搜索可能返回：
// 1. Python 字典的基本用法
// 2. Python 字典性能优化技巧
// 3. Python 字典与 JSON 转换

// LLM 重排序后：
// 1. Python 字典性能优化技巧 ✅ (最相关)
// 2. Python 字典的基本用法
// 3. Python 字典与 JSON 转换
```

### 案例 2：问答系统

```go
// 创建专用的 QA 重排序器
qaReranker, _ := vectorstores.NewLLMReranker(vectorstores.LLMRerankerConfig{
    LLM: llm,
    PromptTemplate: `判断文档能否回答这个问题。

问题: {{.Query}}
文档: {{.Document}}

评分标准：
- 10: 完美回答，包含所有必要信息
- 7-9: 大部分回答，需要补充
- 4-6: 部分相关，但不完整
- 1-3: 略微相关
- 0: 无关

评分:`,
    TopK: 15,
})

// 使用
answer, _ := store.SimilaritySearchWithRerank(
    ctx,
    "什么是机器学习？",
    1, // 只要最相关的一个
    qaReranker,
)
```

---

## 🚨 注意事项

### 1. API 限流

LLM API 可能有速率限制：

```go
import "time"

// 添加速率限制
type RateLimitedReranker struct {
    reranker  *vectorstores.LLMReranker
    rateLimit time.Duration
    lastCall  time.Time
}

func (r *RateLimitedReranker) Rerank(
    ctx context.Context,
    query string,
    docs []vectorstores.DocumentWithScore,
) ([]vectorstores.DocumentWithScore, error) {
    // 等待直到可以调用
    elapsed := time.Since(r.lastCall)
    if elapsed < r.rateLimit {
        time.Sleep(r.rateLimit - elapsed)
    }
    
    result, err := r.reranker.Rerank(ctx, query, docs)
    r.lastCall = time.Now()
    
    return result, err
}
```

### 2. 超时控制

```go
// 设置超时
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

results, err := store.SimilaritySearchWithRerank(ctx, query, 5, reranker)
if err != nil {
    if ctx.Err() == context.DeadlineExceeded {
        // 超时处理
        return store.SimilaritySearch(context.Background(), query, 5)
    }
    return nil, err
}
```

### 3. 文档长度限制

LLM 有 token 限制，过长的文档可能被截断：

```go
// 截断长文档
func truncateDocument(doc *loaders.Document, maxLen int) *loaders.Document {
    if len(doc.Content) > maxLen {
        return loaders.NewDocument(
            doc.Content[:maxLen]+" ...",
            doc.Metadata,
        )
    }
    return doc
}

// 使用
docsWithScore := make([]vectorstores.DocumentWithScore, len(candidates))
for i, doc := range candidates {
    docsWithScore[i] = vectorstores.DocumentWithScore{
        Document: truncateDocument(doc, 2000), // 限制 2000 字符
        Score:    1.0,
    }
}
```

---

## 📚 总结

### 主要特点

✅ **高准确度** - 使用 LLM 理解语义  
✅ **灵活配置** - 自定义提示词和评分标准  
✅ **易于集成** - 与现有向量搜索无缝结合  
✅ **降级支持** - LLM 失败时自动降级

### 性能建议

- 🎯 TopK 设置为 10-30 平衡准确度和成本
- 💰 对重要查询使用，普通查询用向量搜索
- ⚡ 考虑缓存重排序结果
- 🔄 使用更便宜的 LLM 模型（如 gpt-3.5-turbo）

---

**文档维护者**: AI Assistant  
**反馈渠道**: GitHub Issues
