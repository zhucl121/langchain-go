# MMR (Maximum Marginal Relevance) 使用指南

**创建日期**: 2026-01-15  
**版本**: v1.0  
**状态**: ✅ 已完成

---

## 📋 简介

MMR (Maximum Marginal Relevance，最大边际相关性) 是一种智能搜索算法，能够在保持搜索结果相关性的同时，增加结果的多样性。这对于避免返回大量相似的重复文档非常有用。

### 核心思想

传统的相似度搜索只关注与查询的相关性，可能返回很多内容相似的文档。MMR 通过以下方式解决这个问题：

1. **相关性** - 文档与查询的相似度要高
2. **多样性** - 已选文档之间的相似度要低
3. **平衡参数 λ (Lambda)** - 控制相关性和多样性的权重

### 算法公式

```
MMR = λ × Sim(query, doc) - (1-λ) × max(Sim(doc, selected_docs))
```

- `λ = 1.0`: 最大相关性（类似普通搜索）
- `λ = 0.0`: 最大多样性
- `λ = 0.5`: 平衡相关性和多样性（推荐）

---

## 🚀 快速开始

### 基础使用

```go
package main

import (
    "context"
    "fmt"
    "log"

    "langchain-go/retrieval/embeddings"
    "langchain-go/retrieval/loaders"
    "langchain-go/retrieval/vectorstores"
)

func main() {
    ctx := context.Background()

    // 1. 创建嵌入模型
    emb := embeddings.NewOpenAIEmbeddings("your-api-key")

    // 2. 创建向量存储
    store := vectorstores.NewInMemoryVectorStore(emb)

    // 3. 添加文档
    docs := []*loaders.Document{
        loaders.NewDocument("AI is transforming technology", nil),
        loaders.NewDocument("Artificial intelligence is the future", nil),
        loaders.NewDocument("Machine learning is a subset of AI", nil),
        loaders.NewDocument("The weather is nice today", nil),
        loaders.NewDocument("Deep learning uses neural networks", nil),
        loaders.NewDocument("I love eating pizza", nil),
    }
    
    _, err := store.AddDocuments(ctx, docs)
    if err != nil {
        log.Fatal(err)
    }

    // 4. 使用 MMR 搜索（使用默认选项）
    results, err := store.SimilaritySearchWithMMR(
        ctx,
        "artificial intelligence",
        3,    // 返回3个结果
        nil,  // 使用默认选项
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

### 自定义 Lambda 参数

```go
// 更偏向相关性（λ = 0.7）
options := &vectorstores.MMROptions{
    Lambda: 0.7,  // 70% 相关性，30% 多样性
    FetchK: 20,   // 先获取 20 个候选文档
}

results, err := store.SimilaritySearchWithMMR(ctx, query, 5, options)
```

```go
// 更偏向多样性（λ = 0.3）
options := &vectorstores.MMROptions{
    Lambda: 0.3,  // 30% 相关性，70% 多样性
    FetchK: 20,
}

results, err := store.SimilaritySearchWithMMR(ctx, query, 5, options)
```

---

## 🎯 使用场景

### 1. 新闻文章检索

问题：搜索"人工智能"可能返回10篇内容相似的文章

```go
// ❌ 普通搜索 - 返回大量相似文章
results, _ := store.SimilaritySearch(ctx, "人工智能", 10)
// 结果: AI技术1, AI技术2, AI技术3, ... (内容重复)

// ✅ MMR 搜索 - 返回多样化的相关文章
results, _ := store.SimilaritySearchWithMMR(ctx, "人工智能", 10, nil)
// 结果: AI技术, AI伦理, AI应用, AI历史, ... (内容多样)
```

### 2. 产品推荐

```go
// 用户搜索 "笔记本电脑"
// MMR 会返回不同品牌、价格区间、配置的产品
// 而不是10个配置几乎相同的产品

options := &vectorstores.MMROptions{
    Lambda: 0.5,  // 平衡相关性和多样性
    FetchK: 50,   // 从50个候选中选择
}

products, _ := store.SimilaritySearchWithMMR(
    ctx,
    "适合编程的笔记本电脑",
    10,
    options,
)
```

### 3. 文档问答系统

```go
// 在大型文档库中查找相关段落
// MMR 确保返回的段落来自不同章节，提供更全面的信息

options := &vectorstores.MMROptions{
    Lambda: 0.6,  // 稍微偏向相关性
    FetchK: 30,
}

passages, _ := store.SimilaritySearchWithMMR(
    ctx,
    "如何优化数据库性能？",
    5,
    options,
)
```

---

## ⚙️ 参数调优指南

### Lambda (λ) 选择

| Lambda 值 | 相关性 | 多样性 | 适用场景 |
|-----------|--------|--------|----------|
| 1.0 | ⭐⭐⭐⭐⭐ | ⭐ | 精确匹配，对结果质量要求极高 |
| 0.7-0.9 | ⭐⭐⭐⭐ | ⭐⭐ | 技术文档搜索，需要高相关性 |
| 0.5 | ⭐⭐⭐ | ⭐⭐⭐ | **推荐默认值**，平衡场景 |
| 0.2-0.4 | ⭐⭐ | ⭐⭐⭐⭐ | 探索性搜索，需要广泛了解 |
| 0.0 | ⭐ | ⭐⭐⭐⭐⭐ | 极端多样性（很少使用） |

### FetchK 选择

`FetchK` 是先获取的候选文档数量，然后从中选择最终的 K 个结果。

```go
// 规则：FetchK >= K × 2 到 K × 5
k := 5

// ❌ 太小 - 多样性不足
options := &vectorstores.MMROptions{
    Lambda: 0.5,
    FetchK: 6,  // 仅比 k 多 1
}

// ✅ 合适 - 有足够的候选空间
options := &vectorstores.MMROptions{
    Lambda: 0.5,
    FetchK: 20,  // k 的 4 倍（推荐）
}

// ⚠️ 过大 - 可能影响性能
options := &vectorstores.MMROptions{
    Lambda: 0.5,
    FetchK: 100,  // k 的 20 倍（适用于大规模文档库）
}
```

---

## 📊 性能考虑

### 时间复杂度

- **普通相似度搜索**: O(N × D) - N 文档数，D 向量维度
- **MMR 搜索**: O(FetchK × D + K² × D)

MMR 的额外开销来自：
1. 获取 FetchK 个候选（而不是 K 个）
2. 计算候选之间的相似度

### 优化建议

```go
// 1. 合理设置 FetchK
options := &vectorstores.MMROptions{
    Lambda: 0.5,
    FetchK: k * 4,  // 不要设置过大
}

// 2. 对于大规模文档库，先筛选后 MMR
// 方式1: 先用向量搜索获取候选，再 MMR
candidates, _ := store.SimilaritySearch(ctx, query, 100)
// ... 对 candidates 应用 MMR

// 方式2: 使用元数据过滤
// (需要向量存储支持)
```

---

## 🔍 对比示例

### 示例：搜索"机器学习"

#### 普通相似度搜索

```go
results, _ := store.SimilaritySearch(ctx, "机器学习", 5)

// 结果：
// 1. 机器学习是AI的一个分支 (相似度: 0.95)
// 2. 机器学习算法可以从数据中学习 (相似度: 0.93)
// 3. 机器学习应用广泛 (相似度: 0.91)
// 4. 机器学习需要大量数据 (相似度: 0.90)
// 5. 机器学习模型训练需要算力 (相似度: 0.88)
// 
// 问题：5个结果都是关于"机器学习"的基本定义和特点，内容重复
```

#### MMR 搜索 (λ = 0.5)

```go
options := &vectorstores.MMROptions{Lambda: 0.5, FetchK: 20}
results, _ := store.SimilaritySearchWithMMR(ctx, "机器学习", 5, options)

// 结果：
// 1. 机器学习是AI的一个分支 (相关且首选)
// 2. 深度学习使用神经网络 (相关但角度不同)
// 3. 监督学习vs无监督学习 (相关且提供对比)
// 4. 机器学习在医疗领域的应用 (相关且聚焦应用)
// 5. 机器学习模型评估指标 (相关但聚焦技术细节)
//
// ✅ 优势：5个结果覆盖了机器学习的不同方面，提供更全面的信息
```

---

## 🎓 最佳实践

### 1. 选择合适的 Lambda

```go
// 场景1: 技术文档精确搜索
techOptions := &vectorstores.MMROptions{
    Lambda: 0.8,  // 高相关性
    FetchK: 30,
}

// 场景2: 探索性研究
exploreOptions := &vectorstores.MMROptions{
    Lambda: 0.3,  // 高多样性
    FetchK: 50,
}

// 场景3: 通用问答
qaOptions := &vectorstores.MMROptions{
    Lambda: 0.5,  // 平衡
    FetchK: 20,
}
```

### 2. 动态调整参数

```go
func adaptiveMMRSearch(
    store vectorstores.MMRVectorStore,
    query string,
    k int,
    userIntent string,
) ([]*loaders.Document, error) {
    var lambda float32
    var fetchK int
    
    switch userIntent {
    case "precise":
        lambda = 0.8
        fetchK = k * 3
    case "explore":
        lambda = 0.3
        fetchK = k * 5
    default:
        lambda = 0.5
        fetchK = k * 4
    }
    
    options := &vectorstores.MMROptions{
        Lambda: lambda,
        FetchK: fetchK,
    }
    
    return store.SimilaritySearchWithMMR(ctx, query, k, options)
}
```

### 3. 结合其他技术

```go
// MMR + 元数据过滤
func searchWithFilters(
    store vectorstores.MMRVectorStore,
    query string,
    category string,
    k int,
) ([]*loaders.Document, error) {
    // 1. 先用相似度搜索获取同类别的候选
    allResults, _ := store.SimilaritySearch(ctx, query, 100)
    
    // 2. 过滤元数据
    var candidates []*loaders.Document
    for _, doc := range allResults {
        if doc.Metadata["category"] == category {
            candidates = append(candidates, doc)
        }
    }
    
    // 3. 在过滤后的结果中应用 MMR
    // (这需要自定义实现，或等待未来版本支持)
    
    return candidates[:k], nil
}
```

---

## 🧪 测试和验证

### 验证 MMR 效果

```go
func TestMMRDiversity(t *testing.T) {
    ctx := context.Background()
    
    // 创建测试数据
    docs := []*loaders.Document{
        loaders.NewDocument("Python is a programming language", nil),
        loaders.NewDocument("Python is used for data science", nil),
        loaders.NewDocument("Python has clean syntax", nil),
        loaders.NewDocument("Go is a compiled language", nil),
        loaders.NewDocument("JavaScript runs in browsers", nil),
    }
    
    store := vectorstores.NewInMemoryVectorStore(embeddings)
    store.AddDocuments(ctx, docs)
    
    // 普通搜索
    normalResults, _ := store.SimilaritySearch(ctx, "Python", 3)
    
    // MMR 搜索
    mmrResults, _ := store.SimilaritySearchWithMMR(
        ctx,
        "Python",
        3,
        &vectorstores.MMROptions{Lambda: 0.5, FetchK: 5},
    )
    
    // 验证：MMR 结果应该包含其他编程语言
    // 而不是全部关于 Python
    assert.Contains(t, mmrResults, "Go is a compiled language")
}
```

---

## 📚 参考资料

- **论文**: "The Use of MMR, Diversity-Based Reranking for Reordering Documents and Producing Summaries" (Carbonell & Goldstein, 1998)
- **LangChain Python**: [MMR Documentation](https://python.langchain.com/docs/modules/retrieval/vectorstores/mmr)

---

## ✅ 总结

### MMR 的优势

✅ **避免结果重复** - 自动去除相似内容  
✅ **信息全面** - 覆盖查询的多个方面  
✅ **用户体验好** - 提供更有价值的搜索结果  
✅ **简单易用** - 只需一个 Lambda 参数

### 何时使用 MMR

- ✅ 文档内容有大量重复或相似的情况
- ✅ 需要从多角度了解一个主题
- ✅ 产品推荐、新闻聚合等场景
- ✅ 用户可能对多样性有需求的场景

### 何时不用 MMR

- ❌ 需要绝对精确匹配的场景
- ❌ 文档库本身就很多样化
- ❌ 性能极度敏感的实时系统
- ❌ 结果数量很少（k < 3）

---

**文档维护者**: AI Assistant  
**反馈渠道**: GitHub Issues
