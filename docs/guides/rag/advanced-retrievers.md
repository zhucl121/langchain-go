# 高级 RAG 检索技术

本指南介绍 LangChain-Go 中的高级 RAG 检索技术。这些技术可以显著提升检索质量和相关性。

---

## 📚 概述

LangChain-Go 提供了4种高级 RAG 检索技术：

1. **Multi-Query Generation** - 生成多个查询变体提高召回率
2. **HyDE (假设文档嵌入)** - 克服查询-文档语义鸿沟
3. **Parent Document Retriever** - 平衡检索精度和上下文完整性
4. **Self-Query Retriever** - 自动提取结构化查询条件

---

## 🔍 Multi-Query Generation RAG

### 原理

为单个查询生成多个语义等价的变体，并行检索后合并结果，提高召回率和结果多样性。

### 使用示例

```go
import (
    "github.com/zhucl121/langchain-go/retrieval/retrievers"
)

// 创建 Multi-Query Retriever
multiQuery := retrievers.NewMultiQueryRetriever(
    baseRetriever,
    llm,
    retrievers.WithNumQueries(3),
    retrievers.WithCombineStrategy("ranked"),
)

// 检索
docs, err := multiQuery.GetRelevantDocuments(ctx, "什么是机器学习？")
```

### 合并策略

1. **union** - 合并所有结果（默认）
2. **intersection** - 返回所有查询都匹配的结果
3. **ranked** - 按照匹配次数排序

### 配置选项

```go
// 设置查询数量
WithNumQueries(5)

// 设置合并策略
WithCombineStrategy("ranked")

// 自定义提示词
WithQueryPrompt("生成3个不同的查询变体：{query}")

// 设置每个查询的返回数量
WithTopK(4)
```

---

## 🎯 HyDE (假设文档嵌入)

### 原理

让 LLM 生成假设性的答案文档，使用这些假设文档的嵌入进行检索。这克服了"查询短，文档长"的语义不匹配问题。

### 使用示例

```go
import (
    "github.com/zhucl121/langchain-go/retrieval/retrievers"
)

// 创建 HyDE Retriever
hyde := retrievers.NewHyDERetriever(
    llm,
    embedder,
    vectorStore,
    retrievers.WithNumHypothetical(2),
    retrievers.WithCombineStrategy("average"),
)

// 检索
docs, err := hyde.GetRelevantDocuments(ctx, "深度学习的应用场景有哪些？")
```

### 组合策略

1. **average** - 平均所有假设文档的嵌入（默认）
2. **first** - 只使用第一个假设文档
3. **separate** - 分别检索后合并

### 配置选项

```go
// 设置假设文档数量
WithNumHypothetical(3)

// 设置组合策略
WithCombineStrategy("average")

// 包含原始查询嵌入
WithQueryEmbedding(true, 0.3) // 权重 0.3

// 设置返回数量
WithTopK(5)
```

---

## 📚 Parent Document Retriever

### 原理

索引小文档块（提高检索精度），但返回完整的父文档（保持上下文完整性）。这是"精确检索 + 完整上下文"的最佳平衡。

### 使用示例

```go
import (
    "github.com/zhucl121/langchain-go/retrieval/retrievers"
    "github.com/zhucl121/langchain-go/retrieval/splitters"
)

// 创建分割器
childSplitter := splitters.NewRecursiveCharacterTextSplitter(
    splitters.WithChunkSize(200),
    splitters.WithChunkOverlap(50),
)

parentSplitter := splitters.NewRecursiveCharacterTextSplitter(
    splitters.WithChunkSize(2000),
    splitters.WithChunkOverlap(200),
)

// 创建文档存储
docStore := retrievers.NewMemoryDocumentStore()

// 创建 Parent Document Retriever
parentDoc := retrievers.NewParentDocumentRetriever(
    vectorStore,
    docStore,
    childSplitter,
    retrievers.WithParentSplitter(parentSplitter),
    retrievers.WithParentTopK(4),
)

// 添加文档
err := parentDoc.AddDocuments(ctx, documents)

// 检索
docs, err := parentDoc.GetRelevantDocuments(ctx, "查询")
```

### 工作流程

1. **索引阶段**:
   - 将文档分割成父文档（大块）
   - 将父文档分割成子文档（小块）
   - 子文档添加到向量存储
   - 父文档添加到文档存储

2. **检索阶段**:
   - 用查询搜索子文档
   - 提取子文档的父文档 ID
   - 从文档存储获取完整父文档

### 配置选项

```go
// 设置父文档分割器
WithParentSplitter(splitter)

// 设置检索数量
WithParentTopK(5)

// 设置 ID 键名
WithIDKey("doc_id")
WithParentIDKey("parent_id")

// 只返回子文档
WithReturnFullDocument(false)
```

---

## 🔍 Self-Query Retriever

### 原理

从自然语言查询中自动提取：
1. 语义查询部分（用于向量搜索）
2. 元数据过滤条件（用于结构化过滤）

### 使用示例

```go
import (
    "github.com/zhucl121/langchain-go/retrieval/retrievers"
)

// 定义元数据字段
metadataFields := []retrievers.MetadataField{
    retrievers.NewMetadataField(
        "category",
        "string",
        "文档类别",
        "技术", "科学", "艺术",
    ),
    retrievers.NewMetadataField(
        "year",
        "number",
        "发布年份",
    ),
    retrievers.NewMetadataField(
        "language",
        "string",
        "语言",
        "中文", "英文",
    ),
}

// 创建 Self-Query Retriever
selfQuery := retrievers.NewSelfQueryRetriever(
    llm,
    vectorStore,
    "技术文档集合",
    metadataFields,
    retrievers.WithSelfQueryTopK(5),
)

// 检索 - 会自动提取过滤条件
docs, err := selfQuery.GetRelevantDocuments(
    ctx,
    "找一些2023年的中文技术文章",
)
```

### 查询解析示例

**输入**: "找一些2023年的中文技术文章"

**解析结果**:
```json
{
  "query": "技术文章",
  "filter": {
    "year": 2023,
    "language": "中文",
    "category": "技术"
  }
}
```

### 配置选项

```go
// 设置返回数量
WithSelfQueryTopK(10)

// 自定义提示词
WithSelfQueryPrompt(customPrompt)

// 允许空查询（只过滤）
WithAllowEmptyQuery(true)

// 允许空过滤（只查询）
WithAllowEmptyFilter(true)
```

---

## 🔄 组合使用

这些技术可以组合使用以获得更好的效果：

### 示例：HyDE + Multi-Query

```go
// 先用 HyDE 生成假设文档
hyde := retrievers.NewHyDERetriever(llm, embedder, vectorStore)

// 再用 Multi-Query 增加多样性
multiQuery := retrievers.NewMultiQueryRetriever(
    hyde,
    llm,
    retrievers.WithNumQueries(3),
)

docs, err := multiQuery.GetRelevantDocuments(ctx, "复杂查询")
```

### 示例：Self-Query + Parent Document

```go
// 先用 Self-Query 提取过滤条件
selfQuery := retrievers.NewSelfQueryRetriever(
    llm,
    vectorStore,
    "docs",
    metadataFields,
)

// 封装到 Parent Document 中
parentDoc := retrievers.NewParentDocumentRetriever(
    selfQuery,
    docStore,
    childSplitter,
)
```

---

## 📊 性能对比

| 技术 | 召回率 | 精确度 | 延迟 | 适用场景 |
|------|--------|--------|------|----------|
| **Multi-Query** | ⬆️⬆️ 高 | ⬆️ 中 | ⬇️ 高 | 需要高召回率 |
| **HyDE** | ⬆️ 中 | ⬆️⬆️ 高 | ⬇️ 高 | 语义鸿沟大 |
| **Parent Doc** | ⬆️ 中 | ⬆️⬆️ 高 | ➡️ 中 | 需要完整上下文 |
| **Self-Query** | ➡️ 中 | ⬆️⬆️ 高 | ➡️ 中 | 结构化数据 |

---

## 💡 最佳实践

### 1. 选择合适的技术

- **查询模糊** → Multi-Query
- **查询-文档语义差异大** → HyDE
- **需要完整上下文** → Parent Document
- **有结构化元数据** → Self-Query

### 2. 参数调优

```go
// Multi-Query：查询数量 3-5 个
WithNumQueries(3)

// HyDE：假设文档 1-3 个
WithNumHypothetical(2)

// Parent Document：合适的块大小
childSplitter: 100-300 chars
parentSplitter: 1000-3000 chars

// Self-Query：完整的元数据定义
提供详细的字段描述
```

### 3. 监控和调试

```go
// 启用日志
retriever.WithLogging(true)

// 收集指标
metrics := retriever.GetMetrics()
fmt.Printf("检索时间: %v\n", metrics.LatencyMs)
fmt.Printf("结果数量: %d\n", metrics.NumResults)
```

---

## 🔗 相关资源

- [RAG 概述](./overview.md)
- [Milvus 向量存储](./milvus.md)
- [MMR 搜索](./mmr.md)
- [LLM Reranking](./reranking.md)

---

## 📚 参考论文

1. **HyDE**: [Precise Zero-Shot Dense Retrieval](https://arxiv.org/abs/2212.10496)
2. **Multi-Query**: Query Expansion 相关研究
3. **Parent Document**: Chunk 策略相关研究
4. **Self-Query**: Semantic Parsing 相关研究

---

<div align="center">

**[⬆ 回到顶部](#高级-rag-检索技术)**

</div>
